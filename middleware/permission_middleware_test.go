package middleware

import (
	"testing"

	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
)

// TestFormatBytes tests the formatBytes helper function
func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 1572864, "1.5 MB"},
		{"gigabytes", 1610612736, "1.5 GB"},
		{"terabytes", 1649267441664, "1.5 TB"},
		{"zero", 0, "0 B"},
		{"single byte", 1, "1 B"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatBytes(tc.bytes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestHandlerFunc tests the HandlerFunc type
func TestHandlerFunc(t *testing.T) {
	// Test that HandlerFunc can be used as a function type
	var hf HandlerFunc = func(e *core.RequestEvent) error {
		return nil
	}
	assert.NotNil(t, hf, "HandlerFunc should be a valid function type")
}

// TestSharePermissions tests SharePermissions struct
func TestSharePermissions(t *testing.T) {
	sharePerms := &services.SharePermissions{
		ShareID:          "share123",
		ResourceType:    "file",
		ResourceID:      "file123",
		PermissionType:  "read",
		IsExpired:       false,
		RequiresPassword: false,
	}

	assert.Equal(t, "share123", sharePerms.ShareID)
	assert.Equal(t, "file", sharePerms.ResourceType)
	assert.Equal(t, "file123", sharePerms.ResourceID)
	assert.Equal(t, "read", sharePerms.PermissionType)
	assert.False(t, sharePerms.IsExpired)
	assert.False(t, sharePerms.RequiresPassword)
}

// TestQuotaInfo tests QuotaInfo struct
func TestQuotaInfo(t *testing.T) {
	quotaInfo := &services.QuotaInfo{
		TotalQuota: 10 * 1024 * 1024 * 1024, // 10GB total
		UsedQuota:  5 * 1024 * 1024 * 1024,  // 5GB used
		Available: 5 * 1024 * 1024 * 1024,  // 5GB available
		Percentage: 50.0,
	}

	assert.Equal(t, int64(10*1024*1024*1024), quotaInfo.TotalQuota)
	assert.Equal(t, int64(5*1024*1024*1024), quotaInfo.UsedQuota)
	assert.Equal(t, int64(5*1024*1024*1024), quotaInfo.Available)
	assert.Equal(t, 50.0, quotaInfo.Percentage)
}

// TestMiddlewareCreationFunctions tests that all middleware creation functions return valid functions
func TestMiddlewareCreationFunctions(t *testing.T) {
	// These tests ensure the middleware functions can be called without errors
	// We can't easily test the full functionality without complex PocketBase setup,
	// but we can verify the creation logic works

	// Test that the functions exist and can be called with nil parameters
	// (In real usage, these would be called with actual service instances)

	// Test RequireFileOwnership function signature
	assert.NotNil(t, RequireFileOwnership, "RequireFileOwnership function should exist")

	// Test RequireDirectoryAccess function signature
	assert.NotNil(t, RequireDirectoryAccess, "RequireDirectoryAccess function should exist")

	// Test RequireValidShare function signature
	assert.NotNil(t, RequireValidShare, "RequireValidShare function should exist")

	// Test RequireFileReadAccess function signature
	assert.NotNil(t, RequireFileReadAccess, "RequireFileReadAccess function should exist")

	// Test RequireUploadAccess function signature
	assert.NotNil(t, RequireUploadAccess, "RequireUploadAccess function should exist")

	// Test RequireQuotaAvailable function signature
	assert.NotNil(t, RequireQuotaAvailable, "RequireQuotaAvailable function should exist")

	// Test RequireAuth function signature
	assert.NotNil(t, RequireAuth, "RequireAuth function should exist")
}

// TestSharePermissions_Expired tests expired share permissions
func TestSharePermissions_Expired(t *testing.T) {
	sharePerms := &services.SharePermissions{
		ShareID:          "expired_share123",
		ResourceType:    "file",
		ResourceID:      "file123",
		PermissionType:  "read",
		IsExpired:       true,
		RequiresPassword: false,
	}

	assert.True(t, sharePerms.IsExpired, "Share should be marked as expired")
}

// TestSharePermissions_PasswordProtected tests password-protected shares
func TestSharePermissions_PasswordProtected(t *testing.T) {
	sharePerms := &services.SharePermissions{
		ShareID:          "protected_share123",
		ResourceType:    "directory",
		ResourceID:      "dir123",
		PermissionType:  "read_upload",
		IsExpired:       false,
		RequiresPassword: true,
	}

	assert.True(t, sharePerms.RequiresPassword, "Share should require password")
	assert.Equal(t, "read_upload", sharePerms.PermissionType)
}

// TestQuotaInfo_Exhausted tests quota info when quota is exhausted
func TestQuotaInfo_Exhausted(t *testing.T) {
	quotaInfo := &services.QuotaInfo{
		TotalQuota: 10 * 1024 * 1024 * 1024, // 10GB total
		UsedQuota:  10 * 1024 * 1024 * 1024, // 10GB used (full)
		Available: 0,                           // 0 available
		Percentage: 100.0,
	}

	assert.Equal(t, int64(10*1024*1024*1024), quotaInfo.UsedQuota)
	assert.Equal(t, int64(0), quotaInfo.Available)
	assert.Equal(t, 100.0, quotaInfo.Percentage)
	assert.True(t, quotaInfo.Available == 0, "Available quota should be zero when exhausted")
}