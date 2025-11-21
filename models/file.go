package models

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// File represents a file record in the database
type File struct {
	core.BaseModel
	Name            string `db:"name" json:"name"`
	Path            string `db:"path" json:"path"`
	User            string `db:"user" json:"user"`                         // Relation ID
	ParentDirectory string `db:"parent_directory" json:"parent_directory"` // Relation ID (optional)
	Size            int64  `db:"size" json:"size"`
	MimeType        string `db:"mime_type" json:"mime_type"`
	S3Key           string `db:"s3_key" json:"s3_key"`
	S3Bucket        string `db:"s3_bucket" json:"s3_bucket"`
	Checksum        string `db:"checksum" json:"checksum"`
}

// TableName returns the table name for the File model
func (f *File) TableName() string {
	return "files"
}

// IsOwnedBy checks if the file is owned by the specified user
func (f *File) IsOwnedBy(userID string) bool {
	return f.User == userID
}

// GetFullPath returns the full path to the file including the filename
func (f *File) GetFullPath() string {
	if f.Path == "" || f.Path == "/" {
		return f.Name
	}
	// Ensure path doesn't end with slash
	path := strings.TrimSuffix(f.Path, "/")
	return filepath.Join(path, f.Name)
}

// Validate performs validation on the File model
func (f *File) Validate() error {
	// Validate required fields
	if f.Name == "" {
		return errors.New("file name is required")
	}

	if f.User == "" {
		return errors.New("user is required")
	}

	if f.S3Key == "" {
		return errors.New("s3_key is required")
	}

	if f.S3Bucket == "" {
		return errors.New("s3_bucket is required")
	}

	if f.Size < 0 {
		return errors.New("file size cannot be negative")
	}

	// Validate name length
	if len(f.Name) > 255 {
		return errors.New("file name exceeds maximum length of 255 characters")
	}

	// Validate path length
	if len(f.Path) > 1024 {
		return errors.New("file path exceeds maximum length of 1024 characters")
	}

	// Validate s3_key length
	if len(f.S3Key) > 512 {
		return errors.New("s3_key exceeds maximum length of 512 characters")
	}

	// Validate filename for dangerous characters
	sanitized, err := SanitizeFilename(f.Name)
	if err != nil {
		return err
	}
	if sanitized != f.Name {
		return errors.New("file name contains invalid characters")
	}

	// Validate path for traversal attempts
	if err := ValidatePathTraversal(f.Path); err != nil {
		return err
	}

	return nil
}

// GetExtension returns the file extension (including the dot)
func (f *File) GetExtension() string {
	return filepath.Ext(f.Name)
}

// GetNameWithoutExtension returns the filename without extension
func (f *File) GetNameWithoutExtension() string {
	ext := f.GetExtension()
	if ext == "" {
		return f.Name
	}
	return strings.TrimSuffix(f.Name, ext)
}
