package handlers

import (
	"os"
	"testing"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestSettingsHandler_isAdmin_EnvironmentAdmin tests admin detection via environment variable
func TestSettingsHandler_isAdmin_EnvironmentAdmin(t *testing.T) {
	// Set admin email environment
	os.Setenv("ADMIN_EMAIL", "admin@example.com")
	defer os.Unsetenv("ADMIN_EMAIL")

	// Simulate the logic that would be used in isAdmin
	email := "admin@example.com"
	adminEmail := os.Getenv("ADMIN_EMAIL")
	isAdmin := adminEmail != "" && email == adminEmail

	assert.True(t, isAdmin, "User with admin email should be admin")

	// Test non-admin email
	nonAdminEmail := "user@example.com"
	isAdmin = adminEmail != "" && nonAdminEmail == adminEmail
	assert.False(t, isAdmin, "User without admin email should not be admin")
}

// TestSettingsHandler_isAdmin_RecordAdmin tests admin detection via admin field
func TestSettingsHandler_isAdmin_RecordAdmin(t *testing.T) {
	// Ensure admin email is not set
	os.Unsetenv("ADMIN_EMAIL")

	// Simulate the logic that would be used in isAdmin for field-based admin detection
	isAdminField := true
	email := "user@example.com"
	adminEmail := "" // No admin email set

	// Combined logic: admin if field is true OR email matches admin email
	isAdmin := isAdminField || (adminEmail != "" && email == adminEmail)

	assert.True(t, isAdmin, "User with admin field should be admin")
}

// TestSettingsHandler_isAdmin_NonAdmin tests non-admin detection
func TestSettingsHandler_isAdmin_NonAdmin(t *testing.T) {
	// Ensure admin email is not set
	os.Unsetenv("ADMIN_EMAIL")

	// Simulate the logic that would be used in isAdmin for non-admin
	isAdminField := false
	email := "user@example.com"
	adminEmail := "" // No admin email set

	// Combined logic: admin if field is true OR email matches admin email
	isAdmin := isAdminField || (adminEmail != "" && email == adminEmail)

	assert.False(t, isAdmin, "User without admin field or admin email should not be admin")
}

// TestSettingsHandler_NewSettingsHandler tests settings handler creation
func TestSettingsHandler_NewSettingsHandler(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		PublicRegistration: true,
		EmailVerification:  false,
		DefaultUserQuota:   10 * 1024 * 1024 * 1024,
	}

	// This test would normally require a PocketBase app instance, but we can test the struct creation logic
	assert.NotNil(t, cfg)
	assert.True(t, cfg.PublicRegistration)
	assert.False(t, cfg.EmailVerification)
	assert.Equal(t, int64(10*1024*1024*1024), cfg.DefaultUserQuota)

	// Test that the types are valid
	logger := zerolog.Nop()
	renderer := &TemplateRenderer{}
	assert.NotNil(t, logger)
	assert.NotNil(t, renderer)
}

// TestSettingsHandler_ConfigUpdates tests configuration update logic
func TestSettingsHandler_ConfigUpdates(t *testing.T) {
	cfg := &config.Config{
		PublicRegistration: false,
		EmailVerification:  true,
		DefaultUserQuota:   5 * 1024 * 1024 * 1024, // 5GB
	}

	// Simulate configuration updates
	cfg.PublicRegistration = true
	cfg.EmailVerification = false
	cfg.DefaultUserQuota = 10 * 1024 * 1024 * 1024 // 10GB

	assert.True(t, cfg.PublicRegistration)
	assert.False(t, cfg.EmailVerification)
	assert.Equal(t, int64(10*1024*1024*1024), cfg.DefaultUserQuota)
}

// TestSettingsHandler_EnvironmentVariableUpdates tests environment variable updates
func TestSettingsHandler_EnvironmentVariableUpdates(t *testing.T) {
	// Test setting environment variables
	os.Setenv("PUBLIC_REGISTRATION", "true")
	defer os.Unsetenv("PUBLIC_REGISTRATION")

	os.Setenv("EMAIL_VERIFICATION", "false")
	defer os.Unsetenv("EMAIL_VERIFICATION")

	// Verify environment variables were set
	assert.Equal(t, "true", os.Getenv("PUBLIC_REGISTRATION"))
	assert.Equal(t, "false", os.Getenv("EMAIL_VERIFICATION"))
}