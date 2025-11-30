// Package main implements the FilesOnTheGo application entry point.
// FilesOnTheGo is a self-hosted file storage and sharing service built with Gin and GORM.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/assets"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/database"
	handlers "github.com/jd-boyd/filesonthego/handlers_gin"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
	gormlogger "gorm.io/gorm/logger"
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

	// Initialize database
	dbConfig := database.Config{
		DSN:             cfg.DBPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		LogLevel:        gormlogger.Silent,
	}
	if cfg.IsDevelopment() {
		dbConfig.LogLevel = gormlogger.Info
	}

	if err := database.Initialize(dbConfig); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer database.Close()

	// Run database migrations
	if err := database.AutoMigrate(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run database migrations")
	}
	logger.Info().Msg("Database initialized successfully")

	// Initialize JWT manager
	jwtConfig := auth.JWTConfig{
		SecretKey:        []byte(getEnvOrDefault("JWT_SECRET", "change-this-in-production-"+cfg.AppEnvironment)),
		AccessExpiration: 24 * time.Hour, // 24 hours
		Issuer:           "filesonthego",
	}
	jwtManager := auth.NewJWTManager(jwtConfig)

	// Initialize session manager
	sessionConfig := auth.SessionConfig{
		CookieName:     "filesonthego_session",
		CookieDomain:   "",
		CookiePath:     "/",
		CookieSecure:   cfg.TLSEnabled, // Only send over HTTPS in production
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		MaxAge:         24 * time.Hour,
	}
	sessionManager := auth.NewSessionManager(jwtManager, sessionConfig)

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

	// Initialize services
	db := database.GetDB()
	metricsService := services.NewMetricsService()
	userService := services.NewUserService(db, logger)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, templateRenderer, logger, cfg, jwtManager, sessionManager)
	settingsHandler := handlers.NewSettingsHandler(userService, templateRenderer, logger)
	adminHandler := handlers.NewAdminHandler(userService, templateRenderer, logger)

	// Ensure admin user exists with proper permissions
	ensureAdminUser(userService, logger)

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(ginLogger(logger))
	router.Use(metricsMiddleware(metricsService))

	// Metrics endpoint for Prometheus scraping
	router.GET("/metrics", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, metricsService.GetMetrics())
	})

	// Health check endpoint
	router.GET("/api/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"environment": cfg.AppEnvironment,
			"version":     "0.2.0",
		})
	})

	// Serve static files from assets filesystem
	router.StaticFS("/static", http.FS(staticFS))

	// Authentication routes (public)
	router.GET("/login", authHandler.ShowLoginPage)
	router.GET("/register", authHandler.ShowRegisterPage)
	router.POST("/api/auth/login", authHandler.HandleLogin)
	router.POST("/api/auth/register", authHandler.HandleRegister)
	router.POST("/logout", authHandler.HandleLogout)

	// Protected routes (require authentication)
	protected := router.Group("/")
	protected.Use(sessionManager.RequireAuth())
	{
		// Dashboard
		protected.GET("/dashboard", authHandler.ShowDashboard)

		// Settings routes (personal user settings)
		protected.GET("/settings", settingsHandler.ShowSettingsPage)

		// Profile routes (user profile management)
		protected.GET("/profile", settingsHandler.ShowProfilePage)
		protected.GET("/api/profile", settingsHandler.GetProfile)
		protected.GET("/api/profile/stats", settingsHandler.GetProfileStats)
		protected.POST("/api/profile/update", settingsHandler.UpdateProfile)
		protected.POST("/api/profile/password", settingsHandler.UpdatePassword)
	}

	// Admin routes (require admin privileges)
	admin := router.Group("/admin")
	admin.Use(sessionManager.RequireAdmin())
	{
		// Admin dashboard
		admin.GET("", adminHandler.ShowAdminDashboard)

		// User management
		admin.GET("/api/users", adminHandler.ListUsers)
		admin.GET("/api/users/:id", adminHandler.GetUser)
		admin.GET("/api/users/:id/stats", adminHandler.GetUserStats)
		admin.POST("/api/users", adminHandler.CreateUser)
		admin.PUT("/api/users/:id", adminHandler.UpdateUser)
		admin.POST("/api/users/:id/password", adminHandler.ResetUserPassword)
		admin.DELETE("/api/users/:id", adminHandler.DeleteUser)
		admin.GET("/api/users/search", adminHandler.SearchUsers)
	}

	// Root redirect to dashboard or login
	router.GET("/", func(c *gin.Context) {
		// Check if user is authenticated
		if sessionManager.IsAuthenticated(c) {
			c.Redirect(http.StatusFound, "/dashboard")
			return
		}

		// Not authenticated, redirect to login
		c.Redirect(http.StatusFound, "/login")
	})

	logger.Info().Msg("Routes configured successfully")

	// Configure HTTP server
	httpAddr := ":" + cfg.AppPort
	srv := &http.Server{
		Addr:           httpAddr,
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		logger.Info().
			Str("address", httpAddr).
			Bool("tls_enabled", cfg.TLSEnabled).
			Msg("Starting HTTP server")

		var err error
		if cfg.TLSEnabled && cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			// Start with TLS
			err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			// Start without TLS
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	logger.Info().
		Str("address", httpAddr).
		Str("environment", cfg.AppEnvironment).
		Msg("FilesOnTheGo server started successfully")

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-quit
	logger.Info().
		Str("signal", sig.String()).
		Msg("Received shutdown signal, initiating graceful shutdown")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Close database connection
	if err := database.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
	}

	logger.Info().Msg("Server shutdown completed successfully")
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

// ginLogger creates a Gin middleware for zerolog
func ginLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		logEvent := logger.Info()
		if statusCode >= 500 {
			logEvent = logger.Error()
		} else if statusCode >= 400 {
			logEvent = logger.Warn()
		}

		logEvent.
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("HTTP request")
	}
}

// metricsMiddleware wraps the metrics service for Gin
func metricsMiddleware(metricsService *services.MetricsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		metricsService.RecordHTTPRequest(
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}

// ensureAdminUser ensures the admin user exists with proper is_admin flag
func ensureAdminUser(userService *services.UserService, logger zerolog.Logger) {
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		logger.Debug().Msg("No ADMIN_EMAIL environment variable set, skipping admin user setup")
		return
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123" // Default password - should be changed
		logger.Warn().Msg("No ADMIN_PASSWORD set, using default password 'admin123'")
	}

	logger.Info().
		Str("admin_email", adminEmail).
		Msg("Ensuring admin user exists with proper permissions")

	// Try to find user by email
	user, err := userService.GetUserByEmail(adminEmail)
	if err != nil {
		// User doesn't exist, create them
		user, err = userService.CreateUser(adminEmail, "admin", adminPassword, true)
		if err != nil {
			logger.Error().
				Str("admin_email", adminEmail).
				Err(err).
				Msg("Failed to create admin user")
			return
		}

		logger.Info().
			Str("admin_email", adminEmail).
			Str("user_id", user.ID).
			Msg("Admin user created successfully")
		return
	}

	// Check if user already has admin flag
	if user.IsAdmin {
		logger.Info().
			Str("admin_email", adminEmail).
			Msg("Admin user already has is_admin flag set")
		return
	}

	// Set the admin flag
	updates := map[string]interface{}{
		"is_admin": true,
	}
	_, err = userService.UpdateUser(user.ID, updates)
	if err != nil {
		logger.Error().
			Str("admin_email", adminEmail).
			Err(err).
			Msg("Failed to set is_admin flag on existing admin user")
		return
	}

	logger.Info().
		Str("admin_email", adminEmail).
		Msg("Successfully set is_admin flag on existing admin user")
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
