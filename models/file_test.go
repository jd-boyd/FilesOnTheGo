package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile_IsOwnedBy(t *testing.T) {
	file := &File{
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
			result := file.IsOwnedBy(tt.userID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFile_GetFullPath(t *testing.T) {
	tests := []struct {
		name     string
		file     *File
		expected string
	}{
		{
			name: "file in directory",
			file: &File{
				Name: "document.pdf",
				Path: "/documents/work",
			},
			expected: "/documents/work/document.pdf",
		},
		{
			name: "file in root",
			file: &File{
				Name: "readme.txt",
				Path: "/",
			},
			expected: "readme.txt",
		},
		{
			name: "file with empty path",
			file: &File{
				Name: "file.txt",
				Path: "",
			},
			expected: "file.txt",
		},
		{
			name: "path with trailing slash",
			file: &File{
				Name: "image.jpg",
				Path: "/photos/",
			},
			expected: "/photos/image.jpg",
		},
		{
			name: "nested path",
			file: &File{
				Name: "report.docx",
				Path: "/documents/2024/reports",
			},
			expected: "/documents/2024/reports/report.docx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.file.GetFullPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		file    *File
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid file",
			file: &File{
				Name:     "document.pdf",
				Path:     "/documents",
				User:     "user123",
				Size:     1024,
				S3Key:    "abc123/document.pdf",
				S3Bucket: "filesonthego",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			file: &File{
				Name:     "",
				Path:     "/documents",
				User:     "user123",
				Size:     1024,
				S3Key:    "abc123/file.pdf",
				S3Bucket: "bucket",
			},
			wantErr: true,
			errMsg:  "file name is required",
		},
		{
			name: "missing user",
			file: &File{
				Name:     "document.pdf",
				Path:     "/documents",
				User:     "",
				Size:     1024,
				S3Key:    "abc123/document.pdf",
				S3Bucket: "bucket",
			},
			wantErr: true,
			errMsg:  "user is required",
		},
		{
			name: "missing s3_key",
			file: &File{
				Name:     "document.pdf",
				Path:     "/documents",
				User:     "user123",
				Size:     1024,
				S3Key:    "",
				S3Bucket: "bucket",
			},
			wantErr: true,
			errMsg:  "s3_key is required",
		},
		{
			name: "missing s3_bucket",
			file: &File{
				Name:     "document.pdf",
				Path:     "/documents",
				User:     "user123",
				Size:     1024,
				S3Key:    "abc123/document.pdf",
				S3Bucket: "",
			},
			wantErr: true,
			errMsg:  "s3_bucket is required",
		},
		{
			name: "negative size",
			file: &File{
				Name:     "document.pdf",
				Path:     "/documents",
				User:     "user123",
				Size:     -100,
				S3Key:    "abc123/document.pdf",
				S3Bucket: "bucket",
			},
			wantErr: true,
			errMsg:  "file size cannot be negative",
		},
		{
			name: "name too long",
			file: &File{
				Name:     strings.Repeat("a", 300),
				Path:     "/documents",
				User:     "user123",
				Size:     1024,
				S3Key:    "abc123/file.pdf",
				S3Bucket: "bucket",
			},
			wantErr: true,
			errMsg:  "file name exceeds maximum length",
		},
		{
			name: "path too long",
			file: &File{
				Name:     "document.pdf",
				Path:     "/" + strings.Repeat("a", 1025),
				User:     "user123",
				Size:     1024,
				S3Key:    "abc123/document.pdf",
				S3Bucket: "bucket",
			},
			wantErr: true,
			errMsg:  "file path exceeds maximum length",
		},
		{
			name: "s3_key too long",
			file: &File{
				Name:     "document.pdf",
				Path:     "/documents",
				User:     "user123",
				Size:     1024,
				S3Key:    strings.Repeat("a", 600),
				S3Bucket: "bucket",
			},
			wantErr: true,
			errMsg:  "s3_key exceeds maximum length",
		},
		{
			name: "invalid characters in name",
			file: &File{
				Name:     "file\x00.pdf",
				Path:     "/documents",
				User:     "user123",
				Size:     1024,
				S3Key:    "abc123/file.pdf",
				S3Bucket: "bucket",
			},
			wantErr: true,
		},
		{
			name: "path traversal in path",
			file: &File{
				Name:     "document.pdf",
				Path:     "/documents/../../etc",
				User:     "user123",
				Size:     1024,
				S3Key:    "abc123/document.pdf",
				S3Bucket: "bucket",
			},
			wantErr: true,
		},
		{
			name: "zero size is valid",
			file: &File{
				Name:     "empty.txt",
				Path:     "/documents",
				User:     "user123",
				Size:     0,
				S3Key:    "abc123/empty.txt",
				S3Bucket: "bucket",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.Validate()
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

func TestFile_GetExtension(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "pdf extension",
			fileName: "document.pdf",
			expected: ".pdf",
		},
		{
			name:     "multiple dots",
			fileName: "archive.tar.gz",
			expected: ".gz",
		},
		{
			name:     "no extension",
			fileName: "README",
			expected: "",
		},
		{
			name:     "dot file",
			fileName: ".gitignore",
			expected: "",
		},
		{
			name:     "uppercase extension",
			fileName: "PHOTO.JPG",
			expected: ".JPG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &File{Name: tt.fileName}
			result := file.GetExtension()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFile_GetNameWithoutExtension(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "pdf file",
			fileName: "document.pdf",
			expected: "document",
		},
		{
			name:     "multiple dots",
			fileName: "archive.tar.gz",
			expected: "archive.tar",
		},
		{
			name:     "no extension",
			fileName: "README",
			expected: "README",
		},
		{
			name:     "dot file",
			fileName: ".gitignore",
			expected: ".gitignore",
		},
		{
			name:     "only extension",
			fileName: ".txt",
			expected: ".txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &File{Name: tt.fileName}
			result := file.GetNameWithoutExtension()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFile_TableName(t *testing.T) {
	file := &File{}
	assert.Equal(t, "files", file.TableName())
}
