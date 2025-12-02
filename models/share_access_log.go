package models

import (
	"time"

	"gorm.io/gorm"
)

// ShareAccessLog represents an access log entry for share links
type ShareAccessLog struct {
	ID         string    `gorm:"primaryKey;size:15" json:"id"`
	CreatedAt  time.Time `gorm:"autoCreateTime;index" json:"accessed_at"`

	Share      string `gorm:"size:15;not null;index" json:"share"` // Foreign key to shares
	IPAddress  string `gorm:"size:45" json:"ip_address"` // IPv4 or IPv6
	UserAgent  string `gorm:"size:500" json:"user_agent"`
	Action     string `gorm:"size:50" json:"action"` // view, download, upload
	FileName   string `gorm:"size:255" json:"file_name,omitempty"` // File accessed (if applicable)
}

// TableName returns the table name for the ShareAccessLog model
func (s *ShareAccessLog) TableName() string {
	return "share_access_logs"
}

// BeforeCreate hook to generate ID if not set
func (s *ShareAccessLog) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = GenerateID()
	}
	return nil
}
