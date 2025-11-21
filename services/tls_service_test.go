package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTLSService(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled: true,
		TLSPort:    "443",
	}

	service := NewTLSService(cfg)

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.cfg)
}

func TestGetTLSConfig_TLSDisabled(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled: false,
	}

	service := NewTLSService(cfg)
	tlsConfig, err := service.GetTLSConfig()

	assert.Error(t, err)
	assert.Nil(t, tlsConfig)
	assert.Contains(t, err.Error(), "TLS is not enabled")
}

func TestGetTLSConfig_NoConfigurationSource(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		LetsEncryptEnabled: false,
		TLSCertFile:        "",
		TLSKeyFile:         "",
	}

	service := NewTLSService(cfg)
	tlsConfig, err := service.GetTLSConfig()

	assert.Error(t, err)
	assert.Nil(t, tlsConfig)
	assert.Contains(t, err.Error(), "no certificate source specified")
}

func TestGetTLSConfig_WithCertificateFiles(t *testing.T) {
	// Create temporary certificate files
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	err := createTestCertificateFiles(certFile, keyFile)
	require.NoError(t, err)

	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		TLSCertFile:        certFile,
		TLSKeyFile:         keyFile,
		LetsEncryptEnabled: false,
	}

	service := NewTLSService(cfg)
	tlsConfig, err := service.GetTLSConfig()

	assert.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.Len(t, tlsConfig.Certificates, 1)
}

func TestGetTLSConfig_CertificateFileNotFound(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		TLSCertFile:        "/nonexistent/cert.pem",
		TLSKeyFile:         "/nonexistent/key.pem",
		LetsEncryptEnabled: false,
	}

	service := NewTLSService(cfg)
	tlsConfig, err := service.GetTLSConfig()

	assert.Error(t, err)
	assert.Nil(t, tlsConfig)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetTLSConfig_LetsEncrypt(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		LetsEncryptEnabled: true,
		LetsEncryptDomain:  "example.com",
		LetsEncryptEmail:   "admin@example.com",
		LetsEncryptCache:   tempDir,
	}

	service := NewTLSService(cfg)
	tlsConfig, err := service.GetTLSConfig()

	assert.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.NotNil(t, tlsConfig.GetCertificate)
	assert.NotNil(t, service.certManager)
}

func TestGetTLSConfig_LetsEncrypt_MultipleDomains(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		LetsEncryptEnabled: true,
		LetsEncryptDomain:  "example.com, www.example.com, api.example.com",
		LetsEncryptEmail:   "admin@example.com",
		LetsEncryptCache:   tempDir,
	}

	service := NewTLSService(cfg)
	tlsConfig, err := service.GetTLSConfig()

	assert.NoError(t, err)
	assert.NotNil(t, tlsConfig)
}

func TestGetTLSConfig_LetsEncrypt_NoDomain(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		LetsEncryptEnabled: true,
		LetsEncryptDomain:  "",
		LetsEncryptEmail:   "admin@example.com",
		LetsEncryptCache:   tempDir,
	}

	service := NewTLSService(cfg)
	tlsConfig, err := service.GetTLSConfig()

	assert.Error(t, err)
	assert.Nil(t, tlsConfig)
	assert.Contains(t, err.Error(), "no valid domains")
}

func TestHTTPRedirectHandler(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled: true,
		TLSPort:    "443",
	}

	service := NewTLSService(cfg)
	handler := service.HTTPRedirectHandler("443")

	testCases := []struct {
		name           string
		requestHost    string
		requestPath    string
		expectedTarget string
	}{
		{
			name:           "Simple redirect",
			requestHost:    "example.com",
			requestPath:    "/",
			expectedTarget: "https://example.com/",
		},
		{
			name:           "Redirect with path",
			requestHost:    "example.com",
			requestPath:    "/files/upload",
			expectedTarget: "https://example.com/files/upload",
		},
		{
			name:           "Redirect with query string",
			requestHost:    "example.com",
			requestPath:    "/search?q=test",
			expectedTarget: "https://example.com/search?q=test",
		},
		{
			name:           "Redirect with port in host",
			requestHost:    "example.com:80",
			requestPath:    "/",
			expectedTarget: "https://example.com/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.requestPath, nil)
			req.Host = tc.requestHost
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusMovedPermanently, rec.Code)
			assert.Equal(t, tc.expectedTarget, rec.Header().Get("Location"))
		})
	}
}

func TestHTTPRedirectHandler_NonStandardPort(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled: true,
		TLSPort:    "8443",
	}

	service := NewTLSService(cfg)
	handler := service.HTTPRedirectHandler("8443")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "https://example.com:8443/", rec.Header().Get("Location"))
}

func TestValidateDomain(t *testing.T) {
	testCases := []struct {
		domain   string
		expected bool
	}{
		{"example.com", true},
		{"www.example.com", true},
		{"sub.domain.example.com", true},
		{"example-test.com", true},
		{"", false},
		{"example", false},
		{".example.com", false},
		{"example.com.", false},
		{"example .com", false},
		{"example\t.com", false},
		{"example\n.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			result := ValidateDomain(tc.domain)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseDomains(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single domain",
			input:    "example.com",
			expected: []string{"example.com"},
		},
		{
			name:     "Multiple domains",
			input:    "example.com,www.example.com,api.example.com",
			expected: []string{"example.com", "www.example.com", "api.example.com"},
		},
		{
			name:     "Domains with spaces",
			input:    "example.com, www.example.com , api.example.com",
			expected: []string{"example.com", "www.example.com", "api.example.com"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Invalid domains filtered out",
			input:    "example.com,invalid,www.example.com",
			expected: []string{"example.com", "www.example.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseDomains(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetHTTPChallengeHandler_NoCertManager(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		LetsEncryptEnabled: false,
	}

	service := NewTLSService(cfg)
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fallback"))
	})

	handler := service.GetHTTPChallengeHandler(fallback)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "fallback", rec.Body.String())
}

func TestGetCertificateCachePath(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		TLSEnabled:         true,
		LetsEncryptEnabled: true,
		LetsEncryptDomain:  "example.com",
		LetsEncryptEmail:   "admin@example.com",
		LetsEncryptCache:   tempDir,
	}

	service := NewTLSService(cfg)
	path, err := service.GetCertificateCachePath()

	assert.NoError(t, err)
	assert.Contains(t, path, tempDir)
}

func TestGetCertificateCachePath_NotLetsEncrypt(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled:         true,
		LetsEncryptEnabled: false,
	}

	service := NewTLSService(cfg)
	path, err := service.GetCertificateCachePath()

	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "Let's Encrypt is not enabled")
}

func TestValidateCertificateFiles(t *testing.T) {
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	// Create empty files
	err := os.WriteFile(certFile, []byte(""), 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, []byte(""), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		TLSEnabled:  true,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}

	service := NewTLSService(cfg)
	err = service.validateCertificateFiles()

	assert.NoError(t, err)
}

func TestValidateCertificateFiles_MissingCert(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled:  true,
		TLSCertFile: "/nonexistent/cert.pem",
		TLSKeyFile:  "/nonexistent/key.pem",
	}

	service := NewTLSService(cfg)
	err := service.validateCertificateFiles()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate file not found")
}

func TestGetRedirectAndChallengeHandler_LetsEncrypt(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		LetsEncryptEnabled: true,
		LetsEncryptDomain:  "example.com",
		LetsEncryptEmail:   "admin@example.com",
		LetsEncryptCache:   tempDir,
	}

	service := NewTLSService(cfg)
	// Initialize the cert manager by calling GetTLSConfig
	_, err := service.GetTLSConfig()
	require.NoError(t, err)

	handler := service.GetRedirectAndChallengeHandler()
	assert.NotNil(t, handler)
}

func TestGetRedirectAndChallengeHandler_NoLetsEncrypt(t *testing.T) {
	cfg := &config.Config{
		TLSEnabled:         true,
		TLSPort:            "443",
		LetsEncryptEnabled: false,
	}

	service := NewTLSService(cfg)
	handler := service.GetRedirectAndChallengeHandler()

	// Test that it acts as a redirect handler
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
}

// createTestCertificateFiles creates a self-signed certificate for testing
func createTestCertificateFiles(certFile, keyFile string) error {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Organization"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "example.com"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	// Write key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		return err
	}

	return nil
}
