package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithValidConfig(t *testing.T) {
	// Arrange
	setTestEnv(t)

	// Act
	cfg, err := Load()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "http://minio:9000", cfg.S3Endpoint)
	assert.Equal(t, "us-east-1", cfg.S3Region)
	assert.Equal(t, "filesonthego", cfg.S3Bucket)
	assert.Equal(t, "minioadmin", cfg.S3AccessKey)
	assert.Equal(t, "minioadmin", cfg.S3SecretKey)
	assert.Equal(t, false, cfg.S3UseSSL)
	assert.Equal(t, "8090", cfg.AppPort)
	assert.Equal(t, "development", cfg.AppEnvironment)
}

func TestLoad_MissingRequiredS3Endpoint(t *testing.T) {
	// Arrange
	setTestEnv(t)
	os.Unsetenv("S3_ENDPOINT")

	// Act
	cfg, err := Load()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "S3_ENDPOINT")
}

func TestLoad_MissingRequiredS3Bucket(t *testing.T) {
	// Arrange
	setTestEnv(t)
	os.Unsetenv("S3_BUCKET")

	// Act
	cfg, err := Load()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "S3_BUCKET")
}

func TestLoad_MissingRequiredS3AccessKey(t *testing.T) {
	// Arrange
	setTestEnv(t)
	os.Unsetenv("S3_ACCESS_KEY")

	// Act
	cfg, err := Load()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "S3_ACCESS_KEY")
}

func TestLoad_MissingRequiredS3SecretKey(t *testing.T) {
	// Arrange
	setTestEnv(t)
	os.Unsetenv("S3_SECRET_KEY")

	// Act
	cfg, err := Load()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "S3_SECRET_KEY")
}

func TestLoad_ProductionRequiresJWTSecret(t *testing.T) {
	// Arrange
	setTestEnv(t)
	os.Setenv("APP_ENVIRONMENT", "production")
	os.Unsetenv("JWT_SECRET")

	// Act
	cfg, err := Load()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_DevelopmentAllowsEmptyJWTSecret(t *testing.T) {
	// Arrange
	setTestEnv(t)
	os.Setenv("APP_ENVIRONMENT", "development")
	os.Unsetenv("JWT_SECRET")

	// Act
	cfg, err := Load()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "", cfg.JWTSecret)
}

func TestValidate_InvalidMaxUploadSize(t *testing.T) {
	// Arrange
	cfg := &Config{
		S3Endpoint:    "http://minio:9000",
		S3Bucket:      "test",
		S3AccessKey:   "key",
		S3SecretKey:   "secret",
		AppPort:       "8090",
		AppURL:        "http://localhost:8090",
		MaxUploadSize: 0, // Invalid
	}

	// Act
	err := cfg.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MAX_UPLOAD_SIZE")
}

func TestValidate_EmptyAppPort(t *testing.T) {
	// Arrange
	cfg := &Config{
		S3Endpoint:    "http://minio:9000",
		S3Bucket:      "test",
		S3AccessKey:   "key",
		S3SecretKey:   "secret",
		AppPort:       "", // Invalid
		AppURL:        "http://localhost:8090",
		MaxUploadSize: 100,
	}

	// Act
	err := cfg.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "APP_PORT")
}

func TestValidate_EmptyAppURL(t *testing.T) {
	// Arrange
	cfg := &Config{
		S3Endpoint:    "http://minio:9000",
		S3Bucket:      "test",
		S3AccessKey:   "key",
		S3SecretKey:   "secret",
		AppPort:       "8090",
		AppURL:        "", // Invalid
		MaxUploadSize: 100,
	}

	// Act
	err := cfg.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "APP_URL")
}

func TestIsDevelopment_ReturnsTrueInDevelopment(t *testing.T) {
	// Arrange
	cfg := &Config{
		AppEnvironment: "development",
	}

	// Act & Assert
	assert.True(t, cfg.IsDevelopment())
	assert.False(t, cfg.IsProduction())
}

func TestIsProduction_ReturnsTrueInProduction(t *testing.T) {
	// Arrange
	cfg := &Config{
		AppEnvironment: "production",
	}

	// Act & Assert
	assert.True(t, cfg.IsProduction())
	assert.False(t, cfg.IsDevelopment())
}

func TestLoad_DefaultValues(t *testing.T) {
	// Arrange - clean environment first to avoid test pollution
	cleanTestEnv(t)
	defer cleanTestEnv(t)

	// Set minimal required config
	os.Setenv("S3_ENDPOINT", "http://minio:9000")
	os.Setenv("S3_BUCKET", "test")
	os.Setenv("S3_ACCESS_KEY", "key")
	os.Setenv("S3_SECRET_KEY", "secret")

	// Act
	cfg, err := Load()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "us-east-1", cfg.S3Region)                      // Default
	assert.Equal(t, true, cfg.S3UseSSL)                             // Default
	assert.Equal(t, "8090", cfg.AppPort)                            // Default
	assert.Equal(t, "development", cfg.AppEnvironment)              // Default
	assert.Equal(t, "./pb_data", cfg.DBPath)                        // Default
	assert.Equal(t, int64(100*1024*1024), cfg.MaxUploadSize)        // Default 100MB
	assert.Equal(t, true, cfg.PublicRegistration)                   // Default
	assert.Equal(t, false, cfg.EmailVerification)                   // Default
	assert.Equal(t, int64(10*1024*1024*1024), cfg.DefaultUserQuota) // Default 10GB
}

func TestLoadWithPath_LoadsYAMLFile(t *testing.T) {
	// Arrange
	cleanTestEnv(t)
	defer cleanTestEnv(t)

	// Create temp YAML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
s3_endpoint: http://yaml-minio:9000
s3_bucket: yaml-bucket
s3_access_key: yaml-key
s3_secret_key: yaml-secret
s3_region: eu-west-1
app_port: "9090"
app_url: http://localhost:9090
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Act
	cfg, err := LoadWithPath(configPath)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "http://yaml-minio:9000", cfg.S3Endpoint)
	assert.Equal(t, "yaml-bucket", cfg.S3Bucket)
	assert.Equal(t, "yaml-key", cfg.S3AccessKey)
	assert.Equal(t, "yaml-secret", cfg.S3SecretKey)
	assert.Equal(t, "eu-west-1", cfg.S3Region)
	assert.Equal(t, "9090", cfg.AppPort)
}

func TestLoadWithPath_EnvOverridesYAML(t *testing.T) {
	// Arrange
	cleanTestEnv(t)
	defer cleanTestEnv(t)

	// Create temp YAML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
s3_endpoint: http://yaml-minio:9000
s3_bucket: yaml-bucket
s3_access_key: yaml-key
s3_secret_key: yaml-secret
s3_region: eu-west-1
app_port: "9090"
app_url: http://localhost:9090
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set ENV override
	os.Setenv("S3_ENDPOINT", "http://env-minio:9000")
	os.Setenv("S3_REGION", "us-west-2")

	// Act
	cfg, err := LoadWithPath(configPath)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// ENV overrides YAML
	assert.Equal(t, "http://env-minio:9000", cfg.S3Endpoint)
	assert.Equal(t, "us-west-2", cfg.S3Region)
	// YAML values still used when not overridden
	assert.Equal(t, "yaml-bucket", cfg.S3Bucket)
	assert.Equal(t, "9090", cfg.AppPort)
}

func TestLoadWithPath_InvalidYAMLReturnsError(t *testing.T) {
	// Arrange
	cleanTestEnv(t)
	defer cleanTestEnv(t)

	// Create temp invalid YAML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	invalidYAML := `
s3_endpoint: [invalid yaml
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Act
	cfg, err := LoadWithPath(configPath)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoadWithPath_MissingFileReturnsError(t *testing.T) {
	// Arrange
	cleanTestEnv(t)
	defer cleanTestEnv(t)

	// Act
	cfg, err := LoadWithPath("/nonexistent/config.yaml")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_WorksWithoutConfigFile(t *testing.T) {
	// Arrange - only use ENV, no config file
	cleanTestEnv(t)
	defer cleanTestEnv(t)

	os.Setenv("S3_ENDPOINT", "http://minio:9000")
	os.Setenv("S3_BUCKET", "test")
	os.Setenv("S3_ACCESS_KEY", "key")
	os.Setenv("S3_SECRET_KEY", "secret")

	// Act
	cfg, err := Load()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "http://minio:9000", cfg.S3Endpoint)
}

// Helper function to set up test environment variables
func setTestEnv(t *testing.T) {
	t.Helper()
	cleanTestEnv(t)

	os.Setenv("S3_ENDPOINT", "http://minio:9000")
	os.Setenv("S3_REGION", "us-east-1")
	os.Setenv("S3_BUCKET", "filesonthego")
	os.Setenv("S3_ACCESS_KEY", "minioadmin")
	os.Setenv("S3_SECRET_KEY", "minioadmin")
	os.Setenv("S3_USE_SSL", "false")
	os.Setenv("APP_PORT", "8090")
	os.Setenv("APP_ENVIRONMENT", "development")
	os.Setenv("APP_URL", "http://localhost:8090")
	os.Setenv("DB_PATH", "./pb_data")
	os.Setenv("MAX_UPLOAD_SIZE", "104857600") // 100MB
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("PUBLIC_REGISTRATION", "true")
	os.Setenv("EMAIL_VERIFICATION", "false")
	os.Setenv("DEFAULT_USER_QUOTA", "10737418240") // 10GB
}

// Helper function to clean up test environment variables
func cleanTestEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_USE_SSL",
		"APP_PORT", "APP_ENVIRONMENT", "APP_URL", "DB_PATH", "MAX_UPLOAD_SIZE", "JWT_SECRET",
		"PUBLIC_REGISTRATION", "EMAIL_VERIFICATION", "DEFAULT_USER_QUOTA",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
