package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShare_IsExpired(t *testing.T) {
	expiredTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)
	justExpiredTime := time.Now().Add(-1 * time.Second)

	tests := []struct {
		name     string
		share    *Share
		expected bool
	}{
		{
			name: "expired share",
			share: &Share{
				ExpiresAt: &expiredTime,
			},
			expected: true,
		},
		{
			name: "not expired",
			share: &Share{
				ExpiresAt: &futureTime,
			},
			expected: false,
		},
		{
			name: "no expiration",
			share: &Share{
				ExpiresAt: nil,
			},
			expected: false,
		},
		{
			name: "expires exactly now (edge case)",
			share: &Share{
				ExpiresAt: &justExpiredTime,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.share.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_IsPasswordProtected(t *testing.T) {
	tests := []struct {
		name     string
		share    *Share
		expected bool
	}{
		{
			name: "has password",
			share: &Share{
				PasswordHash: "somehash",
			},
			expected: true,
		},
		{
			name: "no password",
			share: &Share{
				PasswordHash: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.share.IsPasswordProtected()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_SetPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "securePassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: "averylongpasswordthatshouldbehashed",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share := &Share{}
			err := share.SetPassword(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.password == "" {
					assert.Empty(t, share.PasswordHash)
				} else {
					assert.NotEmpty(t, share.PasswordHash)
					assert.NotEqual(t, tt.password, share.PasswordHash, "Password should be hashed, not stored in plaintext")
				}
			}
		})
	}
}

func TestShare_ValidatePassword(t *testing.T) {
	share := &Share{}
	correctPassword := "mySecurePassword"
	err := share.SetPassword(correctPassword)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{
			name:     "correct password",
			password: correctPassword,
			expected: true,
		},
		{
			name:     "incorrect password",
			password: "wrongPassword",
			expected: false,
		},
		{
			name:     "empty password",
			password: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := share.ValidatePassword(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_ValidatePassword_NoPasswordSet(t *testing.T) {
	share := &Share{
		PasswordHash: "",
	}

	// When no password is set, any password should validate to true
	result := share.ValidatePassword("anyPassword")
	assert.True(t, result)
}

func TestShare_CanPerformAction(t *testing.T) {
	tests := []struct {
		name       string
		permission PermissionType
		action     string
		expected   bool
	}{
		// Read permission tests
		{
			name:       "read permission - view",
			permission: PermissionRead,
			action:     "view",
			expected:   true,
		},
		{
			name:       "read permission - download",
			permission: PermissionRead,
			action:     "download",
			expected:   true,
		},
		{
			name:       "read permission - upload",
			permission: PermissionRead,
			action:     "upload",
			expected:   false,
		},

		// Read+Upload permission tests
		{
			name:       "read_upload permission - view",
			permission: PermissionReadUpload,
			action:     "view",
			expected:   true,
		},
		{
			name:       "read_upload permission - download",
			permission: PermissionReadUpload,
			action:     "download",
			expected:   true,
		},
		{
			name:       "read_upload permission - upload",
			permission: PermissionReadUpload,
			action:     "upload",
			expected:   true,
		},

		// Upload-only permission tests
		{
			name:       "upload_only permission - view",
			permission: PermissionUploadOnly,
			action:     "view",
			expected:   false,
		},
		{
			name:       "upload_only permission - download",
			permission: PermissionUploadOnly,
			action:     "download",
			expected:   false,
		},
		{
			name:       "upload_only permission - upload",
			permission: PermissionUploadOnly,
			action:     "upload",
			expected:   true,
		},

		// Unknown actions
		{
			name:       "unknown action",
			permission: PermissionRead,
			action:     "delete",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share := &Share{
				PermissionType: tt.permission,
			}
			result := share.CanPerformAction(tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_CanView(t *testing.T) {
	tests := []struct {
		name       string
		permission PermissionType
		expected   bool
	}{
		{
			name:       "read permission",
			permission: PermissionRead,
			expected:   true,
		},
		{
			name:       "read_upload permission",
			permission: PermissionReadUpload,
			expected:   true,
		},
		{
			name:       "upload_only permission",
			permission: PermissionUploadOnly,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share := &Share{
				PermissionType: tt.permission,
			}
			result := share.CanView()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_CanDownload(t *testing.T) {
	tests := []struct {
		name       string
		permission PermissionType
		expected   bool
	}{
		{
			name:       "read permission",
			permission: PermissionRead,
			expected:   true,
		},
		{
			name:       "read_upload permission",
			permission: PermissionReadUpload,
			expected:   true,
		},
		{
			name:       "upload_only permission",
			permission: PermissionUploadOnly,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share := &Share{
				PermissionType: tt.permission,
			}
			result := share.CanDownload()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_CanUpload(t *testing.T) {
	tests := []struct {
		name       string
		permission PermissionType
		expected   bool
	}{
		{
			name:       "read permission",
			permission: PermissionRead,
			expected:   false,
		},
		{
			name:       "read_upload permission",
			permission: PermissionReadUpload,
			expected:   true,
		},
		{
			name:       "upload_only permission",
			permission: PermissionUploadOnly,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share := &Share{
				PermissionType: tt.permission,
			}
			result := share.CanUpload()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_IsValid(t *testing.T) {
	futureTime := time.Now().Add(1 * time.Hour)
	expiredTime := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name     string
		share    *Share
		expected bool
	}{
		{
			name: "valid - not expired",
			share: &Share{
				ExpiresAt: &futureTime,
			},
			expected: true,
		},
		{
			name: "invalid - expired",
			share: &Share{
				ExpiresAt: &expiredTime,
			},
			expected: false,
		},
		{
			name: "valid - no expiration",
			share: &Share{
				ExpiresAt: nil,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.share.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShare_TableName(t *testing.T) {
	share := &Share{}
	assert.Equal(t, "shares", share.TableName())
}

func TestPermissionType_Constants(t *testing.T) {
	assert.Equal(t, PermissionType("read"), PermissionRead)
	assert.Equal(t, PermissionType("read_upload"), PermissionReadUpload)
	assert.Equal(t, PermissionType("upload_only"), PermissionUploadOnly)
}

func TestResourceType_Constants(t *testing.T) {
	assert.Equal(t, ResourceType("file"), ResourceTypeFile)
	assert.Equal(t, ResourceType("directory"), ResourceTypeDirectory)
}

// Note: IncrementAccessCount tests would require a real database instance
// or mocking, so we'll rely on integration tests for that functionality.
