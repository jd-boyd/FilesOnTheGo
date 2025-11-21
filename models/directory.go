package models

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Directory represents a directory record in the database
type Directory struct {
	core.BaseModel
	Name            string `db:"name" json:"name"`
	Path            string `db:"path" json:"path"`
	User            string `db:"user" json:"user"`                         // Relation ID
	ParentDirectory string `db:"parent_directory" json:"parent_directory"` // Relation ID (optional)
}

// TableName returns the table name for the Directory model
func (d *Directory) TableName() string {
	return "directories"
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
func (d *Directory) GetBreadcrumbs(app *pocketbase.PocketBase) ([]*Breadcrumb, error) {
	breadcrumbs := []*Breadcrumb{}

	// Start with the current directory
	current := d
	maxDepth := 100 // Prevent infinite loops

	for i := 0; i < maxDepth; i++ {
		// Add current directory to the front of breadcrumbs
		breadcrumbs = append([]*Breadcrumb{
			{
				ID:   current.Id,
				Name: current.Name,
				Path: current.GetFullPath(),
			},
		}, breadcrumbs...)

		// If no parent directory, we've reached the root
		if current.ParentDirectory == "" {
			break
		}

		// Fetch parent directory
		record, err := app.FindRecordById("directories", current.ParentDirectory)
		if err != nil {
			// Parent not found, break the chain
			break
		}

		// Convert record to Directory struct
		parent := &Directory{}
		parent.Id = record.Id
		parent.Name = record.GetString("name")
		parent.Path = record.GetString("path")
		parent.User = record.GetString("user")
		parent.ParentDirectory = record.GetString("parent_directory")

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
