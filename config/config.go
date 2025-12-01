// Package config provides configuration management for FilesOnTheGo.
// It supports configuration via YAML files and environment variables using Viper.
// Environment variables take precedence over YAML file values.
package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
// Values can be set via YAML file or environment variables.
// Environment variables take precedence over YAML values.
type Config struct {
	// S3 Configuration
	S3Endpoint  string `mapstructure:"s3_endpoint"`
	S3Region    string `mapstructure:"s3_region"`
	S3Bucket    string `mapstructure:"s3_bucket"`
	S3AccessKey string `mapstructure:"s3_access_key"`
	S3SecretKey string `mapstructure:"s3_secret_key"`
	S3UseSSL    bool   `mapstructure:"s3_use_ssl"`

	// Application Configuration
	AppPort        string `mapstructure:"app_port"`
	AppEnvironment string `mapstructure:"app_environment"`
	AppURL         string `mapstructure:"app_url"`

	// Database Configuration
	DBPath string `mapstructure:"db_path"`

	// Upload Configuration
	MaxUploadSize int64 `mapstructure:"max_upload_size"` // in bytes

	// Security Configuration
	JWTSecret string `mapstructure:"jwt_secret"`

	// Feature Flags
	PublicRegistration bool `mapstructure:"public_registration"`
	EmailVerification  bool `mapstructure:"email_verification"`
	RequireEmailAuth   bool `mapstructure:"require_email_auth"`

	// User Quota Configuration
	DefaultUserQuota int64 `mapstructure:"default_user_quota"` // in bytes, 0 means unlimited

	// TLS Configuration
	TLSEnabled  bool   `mapstructure:"tls_enabled"`   // Enable HTTPS
	TLSPort     string `mapstructure:"tls_port"`      // HTTPS port (default: 443)
	TLSCertFile string `mapstructure:"tls_cert_file"` // Path to TLS certificate file
	TLSKeyFile  string `mapstructure:"tls_key_file"`  // Path to TLS private key file
	TLSRedirect bool   `mapstructure:"tls_redirect"`  // Redirect HTTP to HTTPS

	// Let's Encrypt Configuration
	LetsEncryptEnabled bool   `mapstructure:"letsencrypt_enabled"` // Enable automatic Let's Encrypt certificates
	LetsEncryptDomain  string `mapstructure:"letsencrypt_domain"`  // Domain for Let's Encrypt certificate
	LetsEncryptEmail   string `mapstructure:"letsencrypt_email"`   // Email for Let's Encrypt account notifications
	LetsEncryptCache   string `mapstructure:"letsencrypt_cache"`   // Cache directory for ACME certificates
}

// Load reads configuration from YAML file and environment variables.
// It searches for config.yaml in current directory and /etc/filesonthego/,
// then applies environment variable overrides. ENV vars always take precedence.
func Load() (*Config, error) {
	return LoadWithPath("")
}

// LoadWithPath loads configuration from a specific YAML file path.
// If path is empty, it searches default locations.
// Environment variables always override YAML values.
func LoadWithPath(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Bind environment variables explicitly (Viper needs this for mapstructure tags)
	bindEnvVars(v)

	// Configure Viper for YAML
	v.SetConfigType("yaml")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/filesonthego/")
	}

	// Read config file (not required to exist)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if file was found but couldn't be parsed
			if configPath != "" {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	// Unmarshal into Config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// bindEnvVars explicitly binds environment variables to config keys
func bindEnvVars(v *viper.Viper) {
	// S3 Configuration
	v.BindEnv("s3_endpoint", "S3_ENDPOINT")
	v.BindEnv("s3_region", "S3_REGION")
	v.BindEnv("s3_bucket", "S3_BUCKET")
	v.BindEnv("s3_access_key", "S3_ACCESS_KEY")
	v.BindEnv("s3_secret_key", "S3_SECRET_KEY")
	v.BindEnv("s3_use_ssl", "S3_USE_SSL")

	// Application Configuration
	v.BindEnv("app_port", "APP_PORT")
	v.BindEnv("app_environment", "APP_ENVIRONMENT")
	v.BindEnv("app_url", "APP_URL")

	// Database Configuration
	v.BindEnv("db_path", "DB_PATH")

	// Upload Configuration
	v.BindEnv("max_upload_size", "MAX_UPLOAD_SIZE")

	// Security Configuration
	v.BindEnv("jwt_secret", "JWT_SECRET")

	// Feature Flags
	v.BindEnv("public_registration", "PUBLIC_REGISTRATION")
	v.BindEnv("email_verification", "EMAIL_VERIFICATION")
	v.BindEnv("require_email_auth", "REQUIRE_EMAIL_AUTH")

	// User Quota Configuration
	v.BindEnv("default_user_quota", "DEFAULT_USER_QUOTA")

	// TLS Configuration
	v.BindEnv("tls_enabled", "TLS_ENABLED")
	v.BindEnv("tls_port", "TLS_PORT")
	v.BindEnv("tls_cert_file", "TLS_CERT_FILE")
	v.BindEnv("tls_key_file", "TLS_KEY_FILE")
	v.BindEnv("tls_redirect", "TLS_REDIRECT")

	// Let's Encrypt Configuration
	v.BindEnv("letsencrypt_enabled", "LETSENCRYPT_ENABLED")
	v.BindEnv("letsencrypt_domain", "LETSENCRYPT_DOMAIN")
	v.BindEnv("letsencrypt_email", "LETSENCRYPT_EMAIL")
	v.BindEnv("letsencrypt_cache", "LETSENCRYPT_CACHE")
}

// setDefaults sets default values for all configuration options
func setDefaults(v *viper.Viper) {
	// S3 Configuration
	v.SetDefault("s3_region", "us-east-1")
	v.SetDefault("s3_use_ssl", true)

	// Application Configuration
	v.SetDefault("app_port", "8090")
	v.SetDefault("app_environment", "development")
	v.SetDefault("app_url", "http://localhost:8090")

	// Database Configuration
	v.SetDefault("db_path", "./filesonthego.db")

	// Upload Configuration
	v.SetDefault("max_upload_size", 100*1024*1024) // 100MB

	// Feature Flags
	v.SetDefault("public_registration", true)
	v.SetDefault("email_verification", false)
	v.SetDefault("require_email_auth", true)

	// User Quota Configuration
	v.SetDefault("default_user_quota", 10*1024*1024*1024) // 10GB

	// TLS Configuration
	v.SetDefault("tls_enabled", false)
	v.SetDefault("tls_port", "443")
	v.SetDefault("tls_redirect", true)

	// Let's Encrypt Configuration
	v.SetDefault("letsencrypt_enabled", false)
	v.SetDefault("letsencrypt_cache", "./certs")
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

	// Validate TLS configuration
	if c.TLSEnabled {
		// If TLS is enabled with certificate files, both must be provided
		if !c.LetsEncryptEnabled {
			if c.TLSCertFile == "" {
				errs = append(errs, errors.New("TLS_CERT_FILE is required when TLS is enabled without Let's Encrypt"))
			}
			if c.TLSKeyFile == "" {
				errs = append(errs, errors.New("TLS_KEY_FILE is required when TLS is enabled without Let's Encrypt"))
			}
		}

		// Validate TLS port
		if c.TLSPort == "" {
			errs = append(errs, errors.New("TLS_PORT cannot be empty when TLS is enabled"))
		}
	}

	// Validate Let's Encrypt configuration
	if c.LetsEncryptEnabled {
		if !c.TLSEnabled {
			errs = append(errs, errors.New("TLS_ENABLED must be true when using Let's Encrypt"))
		}
		if c.LetsEncryptDomain == "" {
			errs = append(errs, errors.New("LETSENCRYPT_DOMAIN is required when Let's Encrypt is enabled"))
		}
		if c.LetsEncryptEmail == "" {
			errs = append(errs, errors.New("LETSENCRYPT_EMAIL is required when Let's Encrypt is enabled"))
		}
		if c.LetsEncryptCache == "" {
			errs = append(errs, errors.New("LETSENCRYPT_CACHE is required when Let's Encrypt is enabled"))
		}
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

// UsesLetsEncrypt returns true if Let's Encrypt is configured for automatic certificates
func (c *Config) UsesLetsEncrypt() bool {
	return c.TLSEnabled && c.LetsEncryptEnabled
}

// UsesCertificateFiles returns true if TLS is enabled with manual certificate files
func (c *Config) UsesCertificateFiles() bool {
	return c.TLSEnabled && !c.LetsEncryptEnabled && c.TLSCertFile != "" && c.TLSKeyFile != ""
}
