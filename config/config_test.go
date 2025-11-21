package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestGetEnvBool_ParsesTrue(t *testing.T) {
	// Arrange
	os.Setenv("TEST_BOOL_TRUE", "true")
	defer os.Unsetenv("TEST_BOOL_TRUE")

	// Act
	result := getEnvBool("TEST_BOOL_TRUE", false)

	// Assert
	assert.True(t, result)
}

func TestGetEnvBool_ParsesFalse(t *testing.T) {
	// Arrange
	os.Setenv("TEST_BOOL_FALSE", "false")
	defer os.Unsetenv("TEST_BOOL_FALSE")

	// Act
	result := getEnvBool("TEST_BOOL_FALSE", true)

	// Assert
	assert.False(t, result)
}

func TestGetEnvBool_UsesDefaultOnInvalid(t *testing.T) {
	// Arrange
	os.Setenv("TEST_BOOL_INVALID", "not-a-bool")
	defer os.Unsetenv("TEST_BOOL_INVALID")

	// Act
	result := getEnvBool("TEST_BOOL_INVALID", true)

	// Assert
	assert.True(t, result)
}

func TestGetEnvBool_UsesDefaultWhenUnset(t *testing.T) {
	// Act
	result := getEnvBool("UNSET_BOOL_VAR", true)

	// Assert
	assert.True(t, result)
}

func TestGetEnvInt64_ParsesValue(t *testing.T) {
	// Arrange
	os.Setenv("TEST_INT64", "12345")
	defer os.Unsetenv("TEST_INT64")

	// Act
	result := getEnvInt64("TEST_INT64", 0)

	// Assert
	assert.Equal(t, int64(12345), result)
}

func TestGetEnvInt64_UsesDefaultOnInvalid(t *testing.T) {
	// Arrange
	os.Setenv("TEST_INT64_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_INT64_INVALID")

	// Act
	result := getEnvInt64("TEST_INT64_INVALID", 999)

	// Assert
	assert.Equal(t, int64(999), result)
}

func TestGetEnvInt64_UsesDefaultWhenUnset(t *testing.T) {
	// Act
	result := getEnvInt64("UNSET_INT64_VAR", 999)

	// Assert
	assert.Equal(t, int64(999), result)
}

func TestGetEnv_ReturnsValue(t *testing.T) {
	// Arrange
	os.Setenv("TEST_STRING", "test-value")
	defer os.Unsetenv("TEST_STRING")

	// Act
	result := getEnv("TEST_STRING", "default")

	// Assert
	assert.Equal(t, "test-value", result)
}

func TestGetEnv_UsesDefaultWhenUnset(t *testing.T) {
	// Act
	result := getEnv("UNSET_STRING_VAR", "default")

	// Assert
	assert.Equal(t, "default", result)
}

func TestLoad_DefaultValues(t *testing.T) {
	// Arrange - minimal config
	os.Setenv("S3_ENDPOINT", "http://minio:9000")
	os.Setenv("S3_BUCKET", "test")
	os.Setenv("S3_ACCESS_KEY", "key")
	os.Setenv("S3_SECRET_KEY", "secret")
	defer cleanTestEnv(t)

	// Act
	cfg, err := Load()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "us-east-1", cfg.S3Region) // Default
	assert.Equal(t, true, cfg.S3UseSSL) // Default
	assert.Equal(t, "8090", cfg.AppPort) // Default
	assert.Equal(t, "development", cfg.AppEnvironment) // Default
	assert.Equal(t, "./pb_data", cfg.DBPath) // Default
	assert.Equal(t, int64(100*1024*1024), cfg.MaxUploadSize) // Default 100MB
	assert.Equal(t, true, cfg.PublicRegistration) // Default
	assert.Equal(t, false, cfg.EmailVerification) // Default
	assert.Equal(t, int64(10*1024*1024*1024), cfg.DefaultUserQuota) // Default 10GB
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
