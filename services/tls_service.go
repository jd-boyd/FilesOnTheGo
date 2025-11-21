package services

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jd-boyd/filesonthego/config"
	"golang.org/x/crypto/acme/autocert"
)

// TLSService manages TLS configuration and Let's Encrypt certificate automation
type TLSService struct {
	cfg         *config.Config
	certManager *autocert.Manager
	mu          sync.RWMutex
}

// NewTLSService creates a new TLS service with the given configuration
func NewTLSService(cfg *config.Config) *TLSService {
	return &TLSService{
		cfg: cfg,
	}
}

// GetTLSConfig returns the TLS configuration based on the config settings
// For Let's Encrypt, it sets up autocert; for manual certs, it loads them from files
func (s *TLSService) GetTLSConfig() (*tls.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.TLSEnabled {
		return nil, errors.New("TLS is not enabled")
	}

	if s.cfg.UsesLetsEncrypt() {
		return s.getLetsEncryptTLSConfig()
	}

	if s.cfg.UsesCertificateFiles() {
		return s.getCertificateFileTLSConfig()
	}

	return nil, errors.New("invalid TLS configuration: no certificate source specified")
}

// getLetsEncryptTLSConfig creates a TLS config using Let's Encrypt autocert
func (s *TLSService) getLetsEncryptTLSConfig() (*tls.Config, error) {
	// Ensure cache directory exists
	cacheDir := s.cfg.LetsEncryptCache
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create certificate cache directory: %w", err)
	}

	// Parse domains - support comma-separated list
	domains := parseDomains(s.cfg.LetsEncryptDomain)
	if len(domains) == 0 {
		return nil, errors.New("no valid domains specified for Let's Encrypt")
	}

	// Create the autocert manager
	s.certManager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(cacheDir),
		Email:      s.cfg.LetsEncryptEmail,
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		GetCertificate: s.certManager.GetCertificate,
		MinVersion:     tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	return tlsConfig, nil
}

// getCertificateFileTLSConfig creates a TLS config using certificate files
func (s *TLSService) getCertificateFileTLSConfig() (*tls.Config, error) {
	// Validate certificate files exist
	if err := s.validateCertificateFiles(); err != nil {
		return nil, err
	}

	// Load the certificate
	cert, err := tls.LoadX509KeyPair(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	return tlsConfig, nil
}

// validateCertificateFiles checks if the certificate and key files exist and are readable
func (s *TLSService) validateCertificateFiles() error {
	// Check certificate file
	if _, err := os.Stat(s.cfg.TLSCertFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS certificate file not found: %s", s.cfg.TLSCertFile)
	}

	// Check key file
	if _, err := os.Stat(s.cfg.TLSKeyFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS key file not found: %s", s.cfg.TLSKeyFile)
	}

	return nil
}

// GetHTTPChallengeHandler returns an HTTP handler for Let's Encrypt HTTP-01 challenges
// This should be used on port 80 when Let's Encrypt is enabled
func (s *TLSService) GetHTTPChallengeHandler(fallback http.Handler) http.Handler {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.certManager == nil {
		return fallback
	}

	return s.certManager.HTTPHandler(fallback)
}

// CreateTLSListener creates a TLS listener on the specified address
func (s *TLSService) CreateTLSListener(addr string) (net.Listener, error) {
	tlsConfig, err := s.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS listener on %s: %w", addr, err)
	}

	return listener, nil
}

// HTTPRedirectHandler returns an HTTP handler that redirects all requests to HTTPS
func (s *TLSService) HTTPRedirectHandler(httpsPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		// Remove port if present
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		// Build HTTPS URL
		target := "https://" + host
		if httpsPort != "443" {
			target += ":" + httpsPort
		}
		target += r.URL.RequestURI()

		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

// GetRedirectAndChallengeHandler returns a handler that handles both ACME challenges
// and HTTP to HTTPS redirects
func (s *TLSService) GetRedirectAndChallengeHandler() http.Handler {
	redirectHandler := s.HTTPRedirectHandler(s.cfg.TLSPort)

	if s.cfg.UsesLetsEncrypt() {
		return s.GetHTTPChallengeHandler(redirectHandler)
	}

	return redirectHandler
}

// GetCertManager returns the autocert manager for Let's Encrypt
// Returns nil if Let's Encrypt is not enabled
func (s *TLSService) GetCertManager() *autocert.Manager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.certManager
}

// ValidateDomain checks if a domain name is valid
func ValidateDomain(domain string) bool {
	if domain == "" {
		return false
	}

	// Basic validation - check for invalid characters
	domain = strings.TrimSpace(domain)
	if strings.ContainsAny(domain, " \t\n\r") {
		return false
	}

	// Must not start or end with a dot
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// Must contain at least one dot for a valid domain
	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}

// parseDomains parses a comma-separated list of domains
func parseDomains(domainStr string) []string {
	var domains []string
	for _, d := range strings.Split(domainStr, ",") {
		d = strings.TrimSpace(d)
		if ValidateDomain(d) {
			domains = append(domains, d)
		}
	}
	return domains
}

// GetCertificateCachePath returns the absolute path to the certificate cache directory
func (s *TLSService) GetCertificateCachePath() (string, error) {
	if !s.cfg.UsesLetsEncrypt() {
		return "", errors.New("Let's Encrypt is not enabled")
	}

	return filepath.Abs(s.cfg.LetsEncryptCache)
}
