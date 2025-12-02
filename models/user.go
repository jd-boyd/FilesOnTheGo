package models

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user account in the system
type User struct {
	ID        string    `gorm:"primaryKey;size:15" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated"`

	Email           string `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Username        string `gorm:"uniqueIndex;size:100;not null" json:"username"`
	PasswordHash    string `gorm:"size:255;not null" json:"-"` // Never expose in JSON
	EmailVisibility bool   `gorm:"default:false" json:"email_visibility"`

	// Custom fields
	StorageQuota int64 `gorm:"default:5368709120" json:"storage_quota"` // Default 5GB
	StorageUsed  int64 `gorm:"default:0" json:"storage_used"`
	IsAdmin      bool  `gorm:"default:false" json:"is_admin"`
	Verified     bool  `gorm:"default:false" json:"verified"`
}

// TableName returns the table name for the User model
func (u *User) TableName() string {
	return "users"
}

// BeforeCreate hook to generate ID if not set
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = GenerateID()
	}
	return nil
}

// SetPassword hashes and sets the password for the user
func (u *User) SetPassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}

	// Hash the password with bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hash)
	return nil
}

// ValidatePassword validates a password against the stored hash
func (u *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// HasQuotaAvailable checks if the user has enough quota for the specified size
func (u *User) HasQuotaAvailable(size int64) bool {
	if u.StorageQuota <= 0 {
		// Unlimited quota (0 or negative means unlimited)
		return true
	}
	return (u.StorageUsed + size) <= u.StorageQuota
}

// GetAvailableQuota returns the available storage quota in bytes
func (u *User) GetAvailableQuota() int64 {
	if u.StorageQuota <= 0 {
		return -1 // Unlimited
	}
	available := u.StorageQuota - u.StorageUsed
	if available < 0 {
		return 0
	}
	return available
}

// GetQuotaUsagePercent returns the quota usage as a percentage
func (u *User) GetQuotaUsagePercent() float64 {
	if u.StorageQuota <= 0 {
		return 0.0
	}
	return (float64(u.StorageUsed) / float64(u.StorageQuota)) * 100.0
}

// IncrementStorageUsed increments the storage used by the specified amount
func (u *User) IncrementStorageUsed(db *gorm.DB, size int64) error {
	return db.Model(u).Update("storage_used", gorm.Expr("storage_used + ?", size)).Error
}

// DecrementStorageUsed decrements the storage used by the specified amount
func (u *User) DecrementStorageUsed(db *gorm.DB, size int64) error {
	return db.Model(u).Update("storage_used", gorm.Expr("storage_used - ?", size)).Error
}

// Validate performs validation on the User model
func (u *User) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}

	if u.Username == "" {
		return errors.New("username is required")
	}

	if len(u.Email) > 255 {
		return errors.New("email exceeds maximum length of 255 characters")
	}

	if len(u.Username) > 100 {
		return errors.New("username exceeds maximum length of 100 characters")
	}

	// Basic email validation
	if !strings.Contains(u.Email, "@") {
		return errors.New("invalid email format")
	}

	return nil
}
