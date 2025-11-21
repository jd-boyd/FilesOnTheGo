package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jd-boyd/filesonthego/config"
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

	// Create PocketBase instance with custom data directory
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
		Msg("Configuration loaded")

	// Set up health check endpoint
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/health", func(c *core.RequestEvent) error {
			return c.JSON(200, map[string]interface{}{
				"status":      "ok",
				"environment": cfg.AppEnvironment,
				"version":     "0.1.0",
			})
		})
		return nil
	})

	// Log when the server is starting
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		logger.Info().
			Str("address", ":"+cfg.AppPort).
			Msg("PocketBase HTTP server starting")
		return nil
	})

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

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
