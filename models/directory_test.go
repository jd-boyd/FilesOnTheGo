package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectory_IsOwnedBy(t *testing.T) {
	dir := &Directory{
		User: "user123",
	}

	tests := []struct {
		name     string
		userID   string
		expected bool
	}{
		{
			name:     "owner",
			userID:   "user123",
			expected: true,
		},
		{
			name:     "different user",
			userID:   "user456",
			expected: false,
		},
		{
			name:     "empty user",
			userID:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dir.IsOwnedBy(tt.userID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirectory_GetFullPath(t *testing.T) {
	tests := []struct {
		name     string
		dir      *Directory
		expected string
	}{
		{
			name: "nested directory",
			dir: &Directory{
				Name: "work",
				Path: "/documents",
			},
			expected: "/documents/work",
		},
		{
			name: "root directory",
			dir: &Directory{
				Name: "documents",
				Path: "/",
			},
			expected: "documents",
		},
		{
			name: "directory with empty path",
			dir: &Directory{
				Name: "photos",
				Path: "",
			},
			expected: "photos",
		},
		{
			name: "path with trailing slash",
			dir: &Directory{
				Name: "projects",
				Path: "/work/",
			},
			expected: "/work/projects",
		},
		{
			name: "deeply nested",
			dir: &Directory{
				Name: "2024",
				Path: "/documents/work/reports",
			},
			expected: "/documents/work/reports/2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dir.GetFullPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirectory_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dir     *Directory
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid directory",
			dir: &Directory{
				Name: "documents",
				Path: "/",
				User: "user123",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			dir: &Directory{
				Name: "",
				Path: "/",
				User: "user123",
			},
			wantErr: true,
			errMsg:  "directory name is required",
		},
		{
			name: "missing user",
			dir: &Directory{
				Name: "documents",
				Path: "/",
				User: "",
			},
			wantErr: true,
			errMsg:  "user is required",
		},
		{
			name: "name too long",
			dir: &Directory{
				Name: strings.Repeat("a", 300),
				Path: "/",
				User: "user123",
			},
			wantErr: true,
			errMsg:  "directory name exceeds maximum length",
		},
		{
			name: "path too long",
			dir: &Directory{
				Name: "documents",
				Path: "/" + strings.Repeat("a", 1025),
				User: "user123",
			},
			wantErr: true,
			errMsg:  "directory path exceeds maximum length",
		},
		{
			name: "invalid characters in name",
			dir: &Directory{
				Name: "dir\x00ectory",
				Path: "/",
				User: "user123",
			},
			wantErr: true,
		},
		{
			name: "path traversal",
			dir: &Directory{
				Name: "documents",
				Path: "/work/../../etc",
				User: "user123",
			},
			wantErr: true,
		},
		{
			name: "valid unicode name",
			dir: &Directory{
				Name: "文档",
				Path: "/",
				User: "user123",
			},
			wantErr: false,
		},
		{
			name: "valid nested path",
			dir: &Directory{
				Name: "reports",
				Path: "/documents/work",
				User: "user123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dir.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDirectory_IsRoot(t *testing.T) {
	tests := []struct {
		name     string
		dir      *Directory
		expected bool
	}{
		{
			name: "root directory - no parent",
			dir: &Directory{
				Name:            "documents",
				ParentDirectory: "",
			},
			expected: true,
		},
		{
			name: "nested directory - has parent",
			dir: &Directory{
				Name:            "work",
				ParentDirectory: "parent123",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dir.IsRoot()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirectory_GetDepth(t *testing.T) {
	tests := []struct {
		name     string
		dir      *Directory
		expected int
	}{
		{
			name: "root directory",
			dir: &Directory{
				Path: "/",
			},
			expected: 0,
		},
		{
			name: "empty path",
			dir: &Directory{
				Path: "",
			},
			expected: 0,
		},
		{
			name: "first level",
			dir: &Directory{
				Path: "/documents",
			},
			expected: 1,
		},
		{
			name: "second level",
			dir: &Directory{
				Path: "/documents/work",
			},
			expected: 2,
		},
		{
			name: "deeply nested",
			dir: &Directory{
				Path: "/documents/work/projects/2024",
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dir.GetDepth()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirectory_TableName(t *testing.T) {
	dir := &Directory{}
	assert.Equal(t, "directories", dir.TableName())
}

// Note: GetBreadcrumbs tests would require a real PocketBase instance
// or mocking, so we'll skip comprehensive testing here and rely on
// integration tests for that functionality.
func TestBreadcrumb_Structure(t *testing.T) {
	// Test that Breadcrumb struct has expected fields
	breadcrumb := &Breadcrumb{
		ID:   "123",
		Name: "work",
		Path: "/documents/work",
	}

	assert.Equal(t, "123", breadcrumb.ID)
	assert.Equal(t, "work", breadcrumb.Name)
	assert.Equal(t, "/documents/work", breadcrumb.Path)
}
