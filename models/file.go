package models

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// File represents a file record in the database
type File struct {
	ID        string    `gorm:"primaryKey;size:15" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated"`

	Name            string `gorm:"size:255;not null;index" json:"name"`
	Path            string `gorm:"size:1024;not null;index" json:"path"`
	User            string `gorm:"size:15;not null;index" json:"user"` // Foreign key to users
	ParentDirectory string `gorm:"size:15;index" json:"parent_directory"` // Foreign key to directories (optional)
	Size            int64  `gorm:"not null;default:0" json:"size"`
	MimeType        string `gorm:"size:255" json:"mime_type"`
	S3Key           string `gorm:"size:512;not null" json:"s3_key"`
	S3Bucket        string `gorm:"size:255;not null" json:"s3_bucket"`
	Checksum        string `gorm:"size:64" json:"checksum"` // SHA256 checksum
}

// TableName returns the table name for the File model
func (f *File) TableName() string {
	return "files"
}

// BeforeCreate hook to generate ID if not set
func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = GenerateID()
	}
	return nil
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
// Dotfiles (e.g., .gitignore, .bashrc) that have no additional dots
// are treated as having no extension
func (f *File) GetExtension() string {
	// Handle dotfiles: if the name starts with . and has no other dots,
	// it's a dotfile with no extension
	if strings.HasPrefix(f.Name, ".") {
		// Count dots in the filename
		dotCount := strings.Count(f.Name, ".")
		if dotCount == 1 {
			// Only one dot at the start, so no extension
			return ""
		}
	}
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
