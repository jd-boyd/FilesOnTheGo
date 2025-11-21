package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFilename_PreventPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
			wantErr:  false,
		},
		{
			name:     "filename with spaces",
			input:    "my document.pdf",
			expected: "my document.pdf",
			wantErr:  false,
		},
		{
			name:     "parent directory traversal",
			input:    "../../../etc/passwd",
			expected: "passwd",
			wantErr:  false,
		},
		{
			name:     "windows path traversal",
			input:    "..\\..\\..\\windows\\system32\\config",
			expected: "config",
			wantErr:  false,
		},
		{
			name:     "null byte injection",
			input:    "file\x00.txt",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "too long filename",
			input:    strings.Repeat("a", 300),
			expected: "",
			wantErr:  true,
		},
		{
			name:     "control characters",
			input:    "file\n\r\t.txt",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "only control characters",
			input:    "\n\r\t",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "dot file",
			input:    ".gitignore",
			expected: ".gitignore",
			wantErr:  false,
		},
		{
			name:     "current directory reference",
			input:    ".",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "parent directory reference",
			input:    "..",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty filename",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "unicode filename",
			input:    "文档.pdf",
			expected: "文档.pdf",
			wantErr:  false,
		},
		{
			name:     "filename with multiple dots",
			input:    "archive.tar.gz",
			expected: "archive.tar.gz",
			wantErr:  false,
		},
		{
			name:     "path with filename",
			input:    "/path/to/file.txt",
			expected: "file.txt",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeFilename(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizePath_SecurityValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "normal path",
			input:    "/documents/work",
			expected: "/documents/work",
			wantErr:  false,
		},
		{
			name:     "empty path",
			input:    "",
			expected: "/",
			wantErr:  false,
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
			wantErr:  false,
		},
		{
			name:     "path without leading slash",
			input:    "documents/work",
			expected: "/documents/work",
			wantErr:  false,
		},
		{
			name:     "path with trailing slash",
			input:    "/documents/work/",
			expected: "/documents/work",
			wantErr:  false,
		},
		{
			name:     "null byte in path",
			input:    "/documents\x00/work",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "path too long",
			input:    "/" + strings.Repeat("a", 1025),
			expected: "",
			wantErr:  true,
		},
		{
			name:     "traversal attempt",
			input:    "/documents/../../../etc",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "relative path with dots",
			input:    "/documents/./work",
			expected: "/documents/work",
			wantErr:  false,
		},
		{
			name:     "double slashes",
			input:    "/documents//work",
			expected: "/documents/work",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizePath(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidatePathTraversal_SecurityChecks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "normal path",
			input:   "/documents/work/file.txt",
			wantErr: false,
		},
		{
			name:    "empty path",
			input:   "",
			wantErr: false,
		},
		{
			name:    "root path",
			input:   "/",
			wantErr: false,
		},
		{
			name:    "traversal with ..",
			input:   "../etc/passwd",
			wantErr: true,
		},
		{
			name:    "traversal in middle",
			input:   "/documents/../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "encoded traversal",
			input:   "/documents%2f%2e%2e%2fetc",
			wantErr: true,
		},
		{
			name:    "windows backslash traversal",
			input:   "..\\windows\\system32",
			wantErr: true,
		},
		{
			name:    "url encoded dots",
			input:   "/%2e%2e/etc",
			wantErr: true,
		},
		{
			name:    "null byte",
			input:   "/documents\x00/file",
			wantErr: true,
		},
		{
			name:    "current directory reference",
			input:   "/documents/./work",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePathTraversal(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		maxSize int64
		wantErr bool
	}{
		{
			name:    "valid size within limit",
			size:    1024,
			maxSize: 2048,
			wantErr: false,
		},
		{
			name:    "size at limit",
			size:    2048,
			maxSize: 2048,
			wantErr: false,
		},
		{
			name:    "size exceeds limit",
			size:    3000,
			maxSize: 2048,
			wantErr: true,
		},
		{
			name:    "negative size",
			size:    -1,
			maxSize: 2048,
			wantErr: true,
		},
		{
			name:    "zero size",
			size:    0,
			maxSize: 2048,
			wantErr: true,
		},
		{
			name:    "no max size limit",
			size:    999999,
			maxSize: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSize(tt.size, tt.maxSize)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMimeType(t *testing.T) {
	tests := []struct {
		name         string
		mimeType     string
		allowedTypes []string
		wantErr      bool
	}{
		{
			name:         "exact match",
			mimeType:     "image/png",
			allowedTypes: []string{"image/png", "image/jpeg"},
			wantErr:      false,
		},
		{
			name:         "wildcard match",
			mimeType:     "image/png",
			allowedTypes: []string{"image/*"},
			wantErr:      false,
		},
		{
			name:         "not in allowed list",
			mimeType:     "application/exe",
			allowedTypes: []string{"image/*", "application/pdf"},
			wantErr:      true,
		},
		{
			name:         "empty allowed list",
			mimeType:     "anything/goes",
			allowedTypes: []string{},
			wantErr:      false,
		},
		{
			name:         "case insensitive match",
			mimeType:     "IMAGE/PNG",
			allowedTypes: []string{"image/png"},
			wantErr:      false,
		},
		{
			name:         "wildcard with case difference",
			mimeType:     "IMAGE/PNG",
			allowedTypes: []string{"image/*"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMimeType(tt.mimeType, tt.allowedTypes)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "valid filename",
			filename: "document.pdf",
			expected: true,
		},
		{
			name:     "invalid - has path",
			filename: "/path/to/file.txt",
			expected: false,
		},
		{
			name:     "invalid - null byte",
			filename: "file\x00.txt",
			expected: false,
		},
		{
			name:     "invalid - too long",
			filename: strings.Repeat("a", 300),
			expected: false,
		},
		{
			name:     "valid - unicode",
			filename: "文档.pdf",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidFilename(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "valid path",
			path:     "/documents/work",
			expected: true,
		},
		{
			name:     "valid - empty",
			path:     "",
			expected: true,
		},
		{
			name:     "invalid - traversal",
			path:     "../../../etc",
			expected: false,
		},
		{
			name:     "invalid - null byte",
			path:     "/documents\x00/work",
			expected: false,
		},
		{
			name:     "invalid - too long",
			path:     "/" + strings.Repeat("a", 1025),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "no control characters",
			input:    "normal text",
			expected: false,
		},
		{
			name:     "newline",
			input:    "text\nwith newline",
			expected: true,
		},
		{
			name:     "tab",
			input:    "text\twith tab",
			expected: true,
		},
		{
			name:     "carriage return",
			input:    "text\rwith CR",
			expected: true,
		},
		{
			name:     "null byte",
			input:    "text\x00with null",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsControlCharacters(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasDangerousExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "safe extension - pdf",
			filename: "document.pdf",
			expected: false,
		},
		{
			name:     "safe extension - jpg",
			filename: "image.jpg",
			expected: false,
		},
		{
			name:     "dangerous - exe",
			filename: "malware.exe",
			expected: true,
		},
		{
			name:     "dangerous - bat",
			filename: "script.bat",
			expected: true,
		},
		{
			name:     "dangerous - sh",
			filename: "script.sh",
			expected: true,
		},
		{
			name:     "dangerous - ps1",
			filename: "script.ps1",
			expected: true,
		},
		{
			name:     "case insensitive - EXE",
			filename: "malware.EXE",
			expected: true,
		},
		{
			name:     "no extension",
			filename: "file",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasDangerousExtension(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal path",
			input:    "/documents/work",
			expected: "/documents/work",
		},
		{
			name:     "backslashes to forward slashes",
			input:    "\\documents\\work",
			expected: "/documents/work",
		},
		{
			name:     "mixed slashes",
			input:    "/documents\\work/files",
			expected: "/documents/work/files",
		},
		{
			name:     "double slashes",
			input:    "/documents//work",
			expected: "/documents/work",
		},
		{
			name:     "no leading slash",
			input:    "documents/work",
			expected: "/documents/work",
		},
		{
			name:     "trailing slash",
			input:    "/documents/work/",
			expected: "/documents/work",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
