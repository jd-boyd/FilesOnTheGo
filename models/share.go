package models

import (
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/bcrypt"
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
	core.BaseModel
	User           string         `db:"user" json:"user"` // Relation ID
	ResourceType   ResourceType   `db:"resource_type" json:"resource_type"`
	File           string         `db:"file" json:"file"`               // Relation ID (optional)
	Directory      string         `db:"directory" json:"directory"`     // Relation ID (optional)
	ShareToken     string         `db:"share_token" json:"share_token"` // Unique token for sharing
	PermissionType PermissionType `db:"permission_type" json:"permission_type"`
	PasswordHash   string         `db:"password_hash" json:"-"` // Never expose in JSON
	ExpiresAt      time.Time      `db:"expires_at" json:"expires_at"`
	AccessCount    int64          `db:"access_count" json:"access_count"`
}

// TableName returns the table name for the Share model
func (s *Share) TableName() string {
	return "shares"
}

// IsExpired checks if the share link has expired
// Returns false if no expiration is set
func (s *Share) IsExpired() bool {
	// If ExpiresAt is zero value (not set), share never expires
	if s.ExpiresAt.IsZero() {
		return false
	}
	// Check if current time is after expiration
	return time.Now().After(s.ExpiresAt)
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
func (s *Share) IncrementAccessCount(app *pocketbase.PocketBase) error {
	// Increment the counter
	s.AccessCount++

	// Update the record in the database
	record, err := app.FindRecordById("shares", s.Id)
	if err != nil {
		return err
	}

	record.Set("access_count", s.AccessCount)
	if err := app.Save(record); err != nil {
		return err
	}

	return nil
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
