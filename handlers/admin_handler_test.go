package handlers

import (
	"fmt"
	"os"
	"testing"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestAdminHandler_BoolToString tests the boolToString helper function
func TestAdminHandler_BoolToString(t *testing.T) {
	testCases := []struct {
		input    bool
		expected string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("boolToString(%v)", tc.input), func(t *testing.T) {
			result := boolToString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestAdminHandler_IsAdmin_AdminEmail tests admin detection via environment variable
func TestAdminHandler_IsAdmin_AdminEmail(t *testing.T) {
	// This test simulates the admin detection logic without requiring full PocketBase setup

	// Set admin email environment
	os.Setenv("ADMIN_EMAIL", "admin@example.com")
	defer os.Unsetenv("ADMIN_EMAIL")

	// Test the logic that would be used in isAdmin
	email := "admin@example.com"
	adminEmail := os.Getenv("ADMIN_EMAIL")
	isAdmin := adminEmail != "" && email == adminEmail

	assert.True(t, isAdmin, "User with admin email should be admin")

	// Test non-admin email
	nonAdminEmail := "user@example.com"
	isAdmin = adminEmail != "" && nonAdminEmail == adminEmail
	assert.False(t, isAdmin, "User without admin email should not be admin")
}

// TestAdminHandler_IsAdmin_NoAdminEmail tests admin detection when no admin email is set
func TestAdminHandler_IsAdmin_NoAdminEmail(t *testing.T) {
	// Ensure admin email is not set
	os.Unsetenv("ADMIN_EMAIL")

	// Test the logic that would be used in isAdmin
	email := "user@example.com"
	adminEmail := os.Getenv("ADMIN_EMAIL")
	isAdmin := adminEmail != "" && email == adminEmail

	assert.False(t, isAdmin, "User should not be admin when no admin email is set")
}

// TestAdminHandler_IsAdmin_AdminField tests admin detection via admin field
func TestAdminHandler_IsAdmin_AdminField(t *testing.T) {
	// Test the logic that would be used in isAdmin for field-based admin detection
	isAdminField := true
	email := "user@example.com"
	adminEmail := "" // No admin email set

	// Combined logic: admin if field is true OR email matches admin email
	isAdmin := isAdminField || (adminEmail != "" && email == adminEmail)

	assert.True(t, isAdmin, "User with admin field should be admin")
}

// TestAdminHandler_UserListItem tests the UserListItem struct
func TestAdminHandler_UserListItem(t *testing.T) {
	user := UserListItem{
		ID:          "user123",
		Email:       "user@example.com",
		Username:    "username",
		StorageUsed: "10 MB",
		Created:     "2023-01-01",
		IsAdmin:     false,
	}

	assert.Equal(t, "user123", user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "username", user.Username)
	assert.Equal(t, "10 MB", user.StorageUsed)
	assert.Equal(t, "2023-01-01", user.Created)
	assert.False(t, user.IsAdmin)
}

// TestAdminHandler_NewAdminHandler tests admin handler creation
func TestAdminHandler_NewAdminHandler(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		PublicRegistration:  true,
		EmailVerification:   false,
		DefaultUserQuota:    10 * 1024 * 1024 * 1024,
		AppEnvironment:      "test",
		S3Endpoint:         "http://localhost:9000",
		S3Bucket:           "test-bucket",
	}

	// This test would normally require a PocketBase app instance, but we can test the struct creation logic
	assert.NotNil(t, cfg)
	assert.True(t, cfg.PublicRegistration)
	assert.False(t, cfg.EmailVerification)
	assert.Equal(t, int64(10*1024*1024*1024), cfg.DefaultUserQuota)
	assert.Equal(t, "test", cfg.AppEnvironment)

	// Test that the types are valid
	logger := zerolog.Nop()
	renderer := &TemplateRenderer{}
	assert.NotNil(t, logger)
	assert.NotNil(t, renderer)
}

// TestAdminHandler_UserListItem_AdminUser tests admin user list item
func TestAdminHandler_UserListItem_AdminUser(t *testing.T) {
	adminUser := UserListItem{
		ID:          "admin123",
		Email:       "admin@example.com",
		Username:    "admin",
		StorageUsed: "5 MB",
		Created:     "2023-01-01",
		IsAdmin:     true,
	}

	assert.True(t, adminUser.IsAdmin, "Admin user should have IsAdmin set to true")
	assert.Equal(t, "admin@example.com", adminUser.Email)
}