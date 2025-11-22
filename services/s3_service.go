package services

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// Custom error types for S3 operations
var (
	ErrFileNotFound     = errors.New("file not found in S3")
	ErrUploadFailed     = errors.New("failed to upload file to S3")
	ErrDeleteFailed     = errors.New("failed to delete file to S3")
	ErrInvalidKey       = errors.New("invalid S3 key")
	ErrInvalidConfig    = errors.New("invalid S3 configuration")
	ErrConnectionFailed = errors.New("failed to connect to S3")
)

// FileMetadata represents metadata about a file stored in S3
type FileMetadata struct {
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// S3Service defines the interface for S3 operations
type S3Service interface {
	// UploadFile uploads a file with known size to S3
	UploadFile(key string, reader io.Reader, size int64, contentType string) error

	// UploadStream uploads a file using streaming (multipart upload for large files)
	UploadStream(key string, reader io.Reader) error

	// DownloadFile downloads a file from S3 as a stream
	DownloadFile(key string) (io.ReadCloser, error)

	// DeleteFile deletes a single file from S3
	DeleteFile(key string) error

	// DeleteFiles deletes multiple files from S3 (batch operation)
	DeleteFiles(keys []string) error

	// GetPresignedURL generates a time-limited presigned URL for file access
	GetPresignedURL(key string, expirationMinutes int) (string, error)

	// FileExists checks if a file exists in S3
	FileExists(key string) (bool, error)

	// GetFileMetadata retrieves metadata about a file
	GetFileMetadata(key string) (*FileMetadata, error)
}

// GenerateS3Key generates a unique S3 key for storing a file.
// The key follows the pattern: users/{userID}/{fileID}/{filename}
// The filename is sanitized to prevent path traversal attacks.
func GenerateS3Key(userID, fileID, filename string) string {
	// Sanitize filename to prevent path traversal
	// Use only the base name to strip any directory components
	sanitizedFilename := filepath.Base(filename)

	// Remove any remaining path traversal attempts
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "..", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "/", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "\\", "")

	// If sanitization results in empty string, use a default
	if sanitizedFilename == "" || sanitizedFilename == "." {
		sanitizedFilename = "file"
	}

	// Generate key in format: users/{userID}/{fileID}/{filename}
	return "users/" + userID + "/" + fileID + "/" + sanitizedFilename
}