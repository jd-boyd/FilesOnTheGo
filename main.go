// Package main implements the FilesOnTheGo application entry point.
// FilesOnTheGo is a self-hosted file storage and sharing service built with PocketBase.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jd-boyd/filesonthego/assets"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/handlers"
	"github.com/jd-boyd/filesonthego/middleware"
	_ "github.com/jd-boyd/filesonthego/migrations" // Import migrations for side effects
	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
)

func main() {
	// Parse CLI flags
	useExternalAssets := flag.Bool("external-assets", false, "Use external filesystem for templates and static files instead of embedded assets")
	assetsDir := flag.String("assets-dir", ".", "Base directory for external assets (only used with -external-assets)")
	flag.Parse()

	// Configure assets based on CLI flags
	if *useExternalAssets {
		assets.UseEmbedded = false
		assets.SetBaseDir(*assetsDir)
	}

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize structured logging with zerolog
	logger := initLogger(cfg)
	logger.Info().Msg("Starting FilesOnTheGo application")

	// Log asset mode
	if *useExternalAssets {
		logger.Info().Str("assets_dir", *assetsDir).Msg("Using external assets from filesystem")
	} else {
		logger.Info().Msg("Using embedded assets")
	}

	// Create PocketBase instance with data directory
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: cfg.DBPath,
	})

	// Log configuration (without sensitive data)
	logger.Info().
		Str("environment", cfg.AppEnvironment).
		Str("port", cfg.AppPort).
		Str("s3_bucket", cfg.S3Bucket).
		Str("s3_region", cfg.S3Region).
		Int64("max_upload_size", cfg.MaxUploadSize).
		Bool("public_registration", cfg.PublicRegistration).
		Bool("tls_enabled", cfg.TLSEnabled).
		Bool("letsencrypt_enabled", cfg.LetsEncryptEnabled).
		Msg("Configuration loaded")

	// Initialize template renderer using assets filesystem
	templatesFS, err := assets.TemplatesFS()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get templates filesystem")
	}
	templateRenderer := handlers.NewTemplateRendererFromFS(templatesFS)
	if err := templateRenderer.LoadTemplates(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to load templates")
	}
	logger.Info().Msg("Templates loaded successfully")

	// Get static files filesystem for serving
	staticFS, err := assets.StaticFS()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get static filesystem")
	}

	// Initialize metrics service
	metricsService := services.NewMetricsService()

	// Set up health check endpoint and routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		e := se

		// Register metrics middleware to track all HTTP requests
		e.Router.BindFunc(middleware.MetricsMiddleware(metricsService))

		// Metrics endpoint for Prometheus scraping
		e.Router.GET("/metrics", func(c *core.RequestEvent) error {
			c.Response.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
			return c.String(200, metricsService.GetMetrics())
		})

		// Health check endpoint
		e.Router.GET("/api/health", func(c *core.RequestEvent) error {
			return c.JSON(200, map[string]interface{}{
				"status":      "ok",
				"environment": cfg.AppEnvironment,
				"version":     "0.1.0",
			})
		})

		// Serve static files from assets filesystem
		staticHandler := http.FileServer(http.FS(staticFS))
		e.Router.GET("/static/{path...}", func(c *core.RequestEvent) error {
			// Strip the /static prefix and serve from staticFS
			http.StripPrefix("/static", staticHandler).ServeHTTP(c.Response, c.Request)
			return nil
		})

		// Initialize auth handler
		authHandler := handlers.NewAuthHandler(app, templateRenderer, logger)

		// Authentication routes
		e.Router.GET("/login", authHandler.ShowLoginPage)
		e.Router.GET("/register", authHandler.ShowRegisterPage)
		e.Router.POST("/api/auth/login", authHandler.HandleLogin)
		e.Router.POST("/api/auth/register", authHandler.HandleRegister)
		e.Router.POST("/logout", authHandler.HandleLogout)

		// Dashboard route (auth will be checked in handler)
		e.Router.GET("/dashboard", authHandler.ShowDashboard)

		// Root redirect to dashboard or login
		e.Router.GET("/", func(c *core.RequestEvent) error {
			// Check if user is authenticated
			if c.Get("authRecord") != nil {
				return c.Redirect(302, "/dashboard")
			}
			return c.Redirect(302, "/login")
		})

		logger.Info().Msg("Routes configured successfully")
		return nil
	})

	// Set up graceful shutdown
	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start PocketBase server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic: %v", r)
			}
		}()

		// Configure server addresses based on TLS settings
		var httpAddr, httpsAddr string
		var certificateDomains []string

		if cfg.TLSEnabled {
			httpsAddr = ":" + cfg.TLSPort
			if cfg.TLSRedirect {
				// If TLS redirect is enabled, we'll run HTTP on the standard port for redirects
				httpAddr = ":" + cfg.AppPort
			}
			// Add custom domain for Let's Encrypt if configured
			if cfg.LetsEncryptEnabled && cfg.LetsEncryptDomain != "" {
				certificateDomains = append(certificateDomains, cfg.LetsEncryptDomain)
			}
		} else {
			httpAddr = ":" + cfg.AppPort
		}

		// Use PocketBase's built-in serve function which handles TLS properly
		if err := apis.Serve(app, apis.ServeConfig{
			ShowStartBanner:    false, // We have our own logging
			HttpAddr:           httpAddr,
			HttpsAddr:          httpsAddr,
			CertificateDomains: certificateDomains,
		}); err != nil {
			errChan <- fmt.Errorf("failed to start server: %w", err)
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logger.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal, initiating graceful shutdown")

		// Give some time for graceful shutdown
		shutdownTimer := time.NewTimer(30 * time.Second)
		defer shutdownTimer.Stop()

		select {
		case <-time.After(30 * time.Second):
			logger.Warn().Msg("Shutdown timeout exceeded")
		case <-errChan:
			logger.Info().Msg("Server shutdown completed")
		}

	case err := <-errChan:
		logger.Error().
			Err(err).
			Msg("Application error occurred")
		os.Exit(1)
	}
}

// initLogger initializes and configures the zerolog logger
func initLogger(cfg *config.Config) zerolog.Logger {
	// Set log level based on environment
	var logLevel zerolog.Level
	if cfg.IsDevelopment() {
		logLevel = zerolog.DebugLevel
		// Use pretty console output in development
		logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			Level(logLevel).
			With().
			Timestamp().
			Caller().
			Logger()
		return logger
	}

	// Production: JSON output, info level
	logLevel = zerolog.InfoLevel
	logger := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Str("service", "filesonthego").
		Logger()

	return logger
}
