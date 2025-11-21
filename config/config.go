package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// S3 Configuration
	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool

	// Application Configuration
	AppPort        string
	AppEnvironment string
	AppURL         string

	// Database Configuration
	DBPath string

	// Upload Configuration
	MaxUploadSize int64 // in bytes

	// Security Configuration
	JWTSecret string

	// Feature Flags
	PublicRegistration  bool
	EmailVerification   bool
	RequireEmailAuth    bool

	// User Quota Configuration
	DefaultUserQuota int64 // in bytes, 0 means unlimited
}

// Load reads configuration from environment variables and returns a Config struct
func Load() (*Config, error) {
	cfg := &Config{
		// S3 Configuration
		S3Endpoint:  getEnv("S3_ENDPOINT", ""),
		S3Region:    getEnv("S3_REGION", "us-east-1"),
		S3Bucket:    getEnv("S3_BUCKET", ""),
		S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey: getEnv("S3_SECRET_KEY", ""),
		S3UseSSL:    getEnvBool("S3_USE_SSL", true),

		// Application Configuration
		AppPort:        getEnv("APP_PORT", "8090"),
		AppEnvironment: getEnv("APP_ENVIRONMENT", "development"),
		AppURL:         getEnv("APP_URL", "http://localhost:8090"),

		// Database Configuration
		DBPath: getEnv("DB_PATH", "./pb_data"),

		// Upload Configuration
		MaxUploadSize: getEnvInt64("MAX_UPLOAD_SIZE", 100*1024*1024), // Default: 100MB

		// Security Configuration
		JWTSecret: getEnv("JWT_SECRET", ""),

		// Feature Flags
		PublicRegistration:  getEnvBool("PUBLIC_REGISTRATION", true),
		EmailVerification:   getEnvBool("EMAIL_VERIFICATION", false),
		RequireEmailAuth:    getEnvBool("REQUIRE_EMAIL_AUTH", true),

		// User Quota Configuration
		DefaultUserQuota: getEnvInt64("DEFAULT_USER_QUOTA", 10*1024*1024*1024), // Default: 10GB
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errs []error

	// Required S3 Configuration
	if c.S3Endpoint == "" {
		errs = append(errs, errors.New("S3_ENDPOINT is required"))
	}
	if c.S3Bucket == "" {
		errs = append(errs, errors.New("S3_BUCKET is required"))
	}
	if c.S3AccessKey == "" {
		errs = append(errs, errors.New("S3_ACCESS_KEY is required"))
	}
	if c.S3SecretKey == "" {
		errs = append(errs, errors.New("S3_SECRET_KEY is required"))
	}

	// Validate JWT Secret in production
	if c.AppEnvironment == "production" && c.JWTSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET is required in production"))
	}

	// Validate port number
	if c.AppPort == "" {
		errs = append(errs, errors.New("APP_PORT cannot be empty"))
	}

	// Validate upload size
	if c.MaxUploadSize <= 0 {
		errs = append(errs, errors.New("MAX_UPLOAD_SIZE must be greater than 0"))
	}

	// Validate app URL
	if c.AppURL == "" {
		errs = append(errs, errors.New("APP_URL is required"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors: %v", errs)
	}

	return nil
}

// IsDevelopment returns true if the app is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.AppEnvironment == "development"
}

// IsProduction returns true if the app is running in production mode
func (c *Config) IsProduction() bool {
	return c.AppEnvironment == "production"
}

// Helper functions for reading environment variables

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool reads a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return boolValue
	}
	return defaultValue
}

// getEnvInt64 reads an int64 environment variable or returns a default value
func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}
