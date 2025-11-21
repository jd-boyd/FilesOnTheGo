// Package main implements the FilesOnTheGo application entry point.
// FilesOnTheGo is a self-hosted file storage and sharing service built with PocketBase.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/handlers"
	_ "github.com/jd-boyd/filesonthego/migrations" // Import migrations for side effects
	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize structured logging with zerolog
	logger := initLogger(cfg)
	logger.Info().Msg("Starting FilesOnTheGo application")

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

	// Initialize template renderer
	templateRenderer := handlers.NewTemplateRenderer(".")
	if err := templateRenderer.LoadTemplates(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to load templates")
	}
	logger.Info().Msg("Templates loaded successfully")

	// Set up health check endpoint and routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		e := se
		// Health check endpoint
		e.Router.GET("/api/health", func(c *core.RequestEvent) error {
			return c.JSON(200, map[string]interface{}{
				"status":      "ok",
				"environment": cfg.AppEnvironment,
				"version":     "0.1.0",
			})
		})

		// Serve static files - using the built-in static file handler
		// PocketBase's static file serving will be configured separately

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

	// Log when the server is starting
	app.OnServe().BindFunc(func(_ *core.ServeEvent) error {
		logger.Info().
			Str("address", ":"+cfg.AppPort).
			Bool("tls_enabled", cfg.TLSEnabled).
			Msg("PocketBase HTTP server starting")
		return nil
	})

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize TLS service if TLS is enabled
	var tlsService *services.TLSService
	var httpRedirectServer *http.Server
	var httpsServer *http.Server

	if cfg.TLSEnabled {
		tlsService = services.NewTLSService(cfg)

		// Get TLS configuration
		tlsConfig, err := tlsService.GetTLSConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to initialize TLS configuration")
		}

		logger.Info().
			Str("tls_port", cfg.TLSPort).
			Bool("letsencrypt", cfg.LetsEncryptEnabled).
			Msg("TLS initialized successfully")

		// Set up HTTPS server using PocketBase's router
		// We need to start PocketBase first to get the router, then attach it to HTTPS
		app.OnServe().BindFunc(func(e *core.ServeEvent) error {
			httpsServer = &http.Server{
				Addr:      ":" + cfg.TLSPort,
				Handler:   e.Router,
				TLSConfig: tlsConfig,
			}

			go func() {
				logger.Info().
					Str("address", ":"+cfg.TLSPort).
					Msg("Starting HTTPS server")
				if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					logger.Error().Err(err).Msg("HTTPS server error")
				}
			}()

			return nil
		})

		// Start HTTP redirect server if TLS redirect is enabled
		if cfg.TLSRedirect {
			httpRedirectServer = &http.Server{
				Addr:    ":" + cfg.AppPort,
				Handler: tlsService.GetRedirectAndChallengeHandler(),
			}

			go func() {
				logger.Info().
					Str("address", ":"+cfg.AppPort).
					Msg("Starting HTTP redirect server")
				if err := httpRedirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error().Err(err).Msg("HTTP redirect server error")
				}
			}()
		}
	}

	// Start PocketBase in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := app.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logger.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal, initiating graceful shutdown")

		// Create a context with timeout for graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
		defer shutdownCancel()

		// Perform graceful shutdown operations here
		logger.Info().Msg("Graceful shutdown initiated")

		// Shutdown HTTPS server if running
		if httpsServer != nil {
			logger.Info().Msg("Shutting down HTTPS server")
			if err := httpsServer.Shutdown(shutdownCtx); err != nil {
				logger.Error().Err(err).Msg("Error shutting down HTTPS server")
			}
		}

		// Shutdown HTTP redirect server if running
		if httpRedirectServer != nil {
			logger.Info().Msg("Shutting down HTTP redirect server")
			if err := httpRedirectServer.Shutdown(shutdownCtx); err != nil {
				logger.Error().Err(err).Msg("Error shutting down HTTP redirect server")
			}
		}

		// Wait for shutdown context to complete or timeout
		<-shutdownCtx.Done()

		if shutdownCtx.Err() == context.DeadlineExceeded {
			logger.Warn().Msg("Shutdown timeout exceeded, forcing exit")
		} else {
			logger.Info().Msg("Shutdown completed successfully")
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
