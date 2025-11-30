package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// PermissionType defines the type of permission for a share
type PermissionType string

const (
	PermissionRead       PermissionType = "read"        // Can view and download
	PermissionReadUpload PermissionType = "read_upload" // Can view, download, and upload
	PermissionUploadOnly PermissionType = "upload_only" // Can only upload, no viewing
)

// ResourceType defines what type of resource is being shared
type ResourceType string

const (
	ResourceTypeFile      ResourceType = "file"
	ResourceTypeDirectory ResourceType = "directory"
)

// Share represents a share link record in the database
type Share struct {
	ID        string    `gorm:"primaryKey;size:15" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated"`

	User           string         `gorm:"size:15;not null;index" json:"user"` // Foreign key to users
	ResourceType   ResourceType   `gorm:"size:20;not null" json:"resource_type"`
	File           string         `gorm:"size:15;index" json:"file"`               // Foreign key to files (optional)
	Directory      string         `gorm:"size:15;index" json:"directory"`          // Foreign key to directories (optional)
	ShareToken     string         `gorm:"uniqueIndex;size:64;not null" json:"share_token"` // Unique token for sharing
	PermissionType PermissionType `gorm:"size:20;not null" json:"permission_type"`
	PasswordHash   string         `gorm:"size:255" json:"-"` // Never expose in JSON
	ExpiresAt      *time.Time     `gorm:"index" json:"expires_at,omitempty"`
	AccessCount    int64          `gorm:"default:0" json:"access_count"`
}

// TableName returns the table name for the Share model
func (s *Share) TableName() string {
	return "shares"
}

// BeforeCreate hook to generate ID and share token if not set
func (s *Share) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = GenerateID()
	}
	if s.ShareToken == "" {
		s.ShareToken = GenerateShareToken()
	}
	return nil
}

// IsExpired checks if the share link has expired
// Returns false if no expiration is set
func (s *Share) IsExpired() bool {
	// If ExpiresAt is nil (not set), share never expires
	if s.ExpiresAt == nil {
		return false
	}
	// Check if current time is after expiration
	return time.Now().After(*s.ExpiresAt)
}

// IsPasswordProtected checks if the share requires a password
func (s *Share) IsPasswordProtected() bool {
	return s.PasswordHash != ""
}

// ValidatePassword validates a password against the stored hash
// Returns true if the password matches, false otherwise
func (s *Share) ValidatePassword(password string) bool {
	if !s.IsPasswordProtected() {
		// No password required
		return true
	}

	// Compare provided password with stored hash
	err := bcrypt.CompareHashAndPassword([]byte(s.PasswordHash), []byte(password))
	return err == nil
}

// SetPassword hashes and sets the password for the share
func (s *Share) SetPassword(password string) error {
	if password == "" {
		s.PasswordHash = ""
		return nil
	}

	// Hash the password with bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	s.PasswordHash = string(hash)
	return nil
}

// IncrementAccessCount increments the access count for the share
func (s *Share) IncrementAccessCount(db *gorm.DB) error {
	// Use atomic increment to avoid race conditions
	return db.Model(s).Update("access_count", gorm.Expr("access_count + ?", 1)).Error
}

// CanPerformAction checks if the share permission allows the specified action
func (s *Share) CanPerformAction(action string) bool {
	switch s.PermissionType {
	case PermissionRead:
		// Read permission allows view and download
		return action == "view" || action == "download"

	case PermissionReadUpload:
		// Read+Upload allows all actions
		return action == "view" || action == "download" || action == "upload"

	case PermissionUploadOnly:
		// Upload-only allows only upload
		return action == "upload"

	default:
		return false
	}
}

// CanView checks if the share allows viewing/listing files
func (s *Share) CanView() bool {
	return s.CanPerformAction("view")
}

// CanDownload checks if the share allows downloading files
func (s *Share) CanDownload() bool {
	return s.CanPerformAction("download")
}

// CanUpload checks if the share allows uploading files
func (s *Share) CanUpload() bool {
	return s.CanPerformAction("upload")
}

// IsValid checks if the share is currently valid (not expired)
func (s *Share) IsValid() bool {
	return !s.IsExpired()
}
