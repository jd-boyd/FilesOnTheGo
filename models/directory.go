// Package models provides data structures and business logic for FilesOnTheGo.
// It includes directory management, permissions, and other core domain models.
package models

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Directory represents a directory record in the database
type Directory struct {
	ID        string    `gorm:"primaryKey;size:15" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated"`

	Name            string `gorm:"size:255;not null;index" json:"name"`
	Path            string `gorm:"size:1024;not null;index" json:"path"`
	User            string `gorm:"size:15;not null;index" json:"user"` // Foreign key to users
	ParentDirectory string `gorm:"size:15;index" json:"parent_directory"` // Foreign key to directories (optional)
}

// TableName returns the table name for the Directory model
func (d *Directory) TableName() string {
	return "directories"
}

// BeforeCreate hook to generate ID if not set
func (d *Directory) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = GenerateID()
	}
	return nil
}

// IsOwnedBy checks if the directory is owned by the specified user
func (d *Directory) IsOwnedBy(userID string) bool {
	return d.User == userID
}

// GetFullPath returns the full path to the directory
func (d *Directory) GetFullPath() string {
	if d.Path == "" || d.Path == "/" {
		return d.Name
	}
	// Ensure path doesn't end with slash
	path := strings.TrimSuffix(d.Path, "/")
	return filepath.Join(path, d.Name)
}

// Validate performs validation on the Directory model
func (d *Directory) Validate() error {
	// Validate required fields
	if d.Name == "" {
		return errors.New("directory name is required")
	}

	if d.User == "" {
		return errors.New("user is required")
	}

	// Validate name length
	if len(d.Name) > 255 {
		return errors.New("directory name exceeds maximum length of 255 characters")
	}

	// Validate path length
	if len(d.Path) > 1024 {
		return errors.New("directory path exceeds maximum length of 1024 characters")
	}

	// Validate directory name for dangerous characters
	sanitized, err := SanitizeFilename(d.Name)
	if err != nil {
		return err
	}
	if sanitized != d.Name {
		return errors.New("directory name contains invalid characters")
	}

	// Validate path for traversal attempts
	if err := ValidatePathTraversal(d.Path); err != nil {
		return err
	}

	return nil
}

// Breadcrumb represents a single item in a directory breadcrumb trail
type Breadcrumb struct {
	ID   string
	Name string
	Path string
}

// GetBreadcrumbs returns the breadcrumb trail for the directory
// Returns an ordered list from root to current directory
func (d *Directory) GetBreadcrumbs(db *gorm.DB) ([]*Breadcrumb, error) {
	breadcrumbs := []*Breadcrumb{}

	// Start with the current directory
	current := d
	maxDepth := 100 // Prevent infinite loops

	for i := 0; i < maxDepth; i++ {
		// Add current directory to the front of breadcrumbs
		breadcrumbs = append([]*Breadcrumb{
			{
				ID:   current.ID,
				Name: current.Name,
				Path: current.GetFullPath(),
			},
		}, breadcrumbs...)

		// If no parent directory, we've reached the root
		if current.ParentDirectory == "" {
			break
		}

		// Fetch parent directory
		parent := &Directory{}
		if err := db.First(parent, "id = ?", current.ParentDirectory).Error; err != nil {
			// Parent not found, break the chain
			break
		}

		current = parent
	}

	return breadcrumbs, nil
}

// IsRoot checks if the directory is a root directory (has no parent)
func (d *Directory) IsRoot() bool {
	return d.ParentDirectory == ""
}

// GetDepth returns the depth of the directory in the hierarchy
// Root directories have depth 0
func (d *Directory) GetDepth() int {
	if d.Path == "" || d.Path == "/" {
		return 0
	}
	// Count the number of path separators
	return strings.Count(strings.TrimPrefix(d.Path, "/"), "/") + 1
}
