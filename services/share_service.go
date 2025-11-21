package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// ShareService defines the interface for share link management
type ShareService interface {
	CreateShare(params CreateShareParams) (*ShareInfo, error)
	GetShareByToken(token string) (*ShareInfo, error)
	GetShareByID(shareID string) (*ShareInfo, error)
	ValidateShareAccess(token, password string) (*ShareAccessInfo, error)
	RevokeShare(shareID, userID string) error
	ListUserShares(userID, resourceType string) ([]*ShareInfo, error)
	UpdateShareExpiration(shareID, userID string, expiresAt *time.Time) error
	GetShareAccessLogs(shareID, userID string) ([]*ShareAccessLog, error)
	LogShareAccess(shareID, action, fileName, ipAddress, userAgent string) error
	IncrementAccessCount(shareID string) error
}

// CreateShareParams contains parameters for creating a new share
type CreateShareParams struct {
	UserID         string
	ResourceType   string // "file" or "directory"
	ResourceID     string
	PermissionType string     // "read", "read_upload", "upload_only"
	Password       string     // optional
	ExpiresAt      *time.Time // optional
}

// ShareInfo represents complete share information
type ShareInfo struct {
	ID                  string                `json:"id"`
	UserID              string                `json:"user_id"`
	ResourceType        models.ResourceType   `json:"resource_type"`
	ResourceID          string                `json:"resource_id"`
	ShareToken          string                `json:"share_token"`
	PermissionType      models.PermissionType `json:"permission_type"`
	ExpiresAt           *time.Time            `json:"expires_at,omitempty"`
	AccessCount         int64                 `json:"access_count"`
	Created             time.Time             `json:"created"`
	Updated             time.Time             `json:"updated"`
	IsExpired           bool                  `json:"is_expired"`
	IsPasswordProtected bool                  `json:"is_password_protected"`
}

// ShareAccessInfo represents the result of validating a share access attempt
type ShareAccessInfo struct {
	ShareID        string
	ResourceType   string
	ResourceID     string
	PermissionType string
	ExpiresAt      *time.Time
	IsValid        bool
	ErrorMessage   string
}

// ShareAccessLog represents a log entry for share access
type ShareAccessLog struct {
	ID         string    `json:"id"`
	ShareID    string    `json:"share_id"`
	Action     string    `json:"action"`
	FileName   string    `json:"file_name,omitempty"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	AccessedAt time.Time `json:"accessed_at"`
}

// ShareServiceImpl implements the ShareService interface
type ShareServiceImpl struct {
	app *pocketbase.PocketBase
}

// NewShareService creates a new share service instance
func NewShareService(app *pocketbase.PocketBase) ShareService {
	return &ShareServiceImpl{
		app: app,
	}
}

// CreateShare creates a new share link with the specified parameters
func (s *ShareServiceImpl) CreateShare(params CreateShareParams) (*ShareInfo, error) {
	// Validate parameters
	if params.UserID == "" {
		return nil, errors.New("user ID is required")
	}
	if params.ResourceType == "" {
		return nil, errors.New("resource type is required")
	}
	if params.ResourceID == "" {
		return nil, errors.New("resource ID is required")
	}
	if params.PermissionType == "" {
		return nil, errors.New("permission type is required")
	}

	// Validate resource type
	if params.ResourceType != "file" && params.ResourceType != "directory" {
		return nil, errors.New("resource type must be 'file' or 'directory'")
	}

	// Validate permission type
	if params.PermissionType != "read" && params.PermissionType != "read_upload" && params.PermissionType != "upload_only" {
		return nil, errors.New("permission type must be 'read', 'read_upload', or 'upload_only'")
	}

	// Verify user owns the resource
	collection := "files"
	if params.ResourceType == "directory" {
		collection = "directories"
	}

	resource, err := s.app.FindRecordById(collection, params.ResourceID)
	if err != nil {
		log.Error().Err(err).Str("resource_id", params.ResourceID).Str("resource_type", params.ResourceType).Msg("Resource not found")
		return nil, fmt.Errorf("%s not found", params.ResourceType)
	}

	if resource.GetString("user") != params.UserID {
		log.Warn().Str("user_id", params.UserID).Str("resource_id", params.ResourceID).Msg("User does not own resource")
		return nil, errors.New("you do not have permission to share this resource")
	}

	// Validate expiration is in future
	if params.ExpiresAt != nil && params.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("expiration date must be in the future")
	}

	// Generate unique share token (UUID v4)
	shareToken := uuid.New().String()

	// Create share record
	sharesCollection, err := s.app.FindCollectionByNameOrId("shares")
	if err != nil {
		log.Error().Err(err).Msg("Failed to find shares collection")
		return nil, fmt.Errorf("failed to find shares collection: %w", err)
	}
	if sharesCollection == nil {
		return nil, fmt.Errorf("shares collection not found")
	}
	record := core.NewRecord(sharesCollection)

	record.Set("user", params.UserID)
	record.Set("resource_type", params.ResourceType)
	record.Set("permission_type", params.PermissionType)
	record.Set("share_token", shareToken)
	record.Set("access_count", 0)

	// Set resource ID based on type
	if params.ResourceType == "file" {
		record.Set("file", params.ResourceID)
		record.Set("directory", "")
	} else {
		record.Set("directory", params.ResourceID)
		record.Set("file", "")
	}

	// Hash password if provided (bcrypt cost 12 for security)
	if params.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), 12)
		if err != nil {
			log.Error().Err(err).Msg("Failed to hash password")
			return nil, errors.New("failed to hash password")
		}
		record.Set("password_hash", string(hash))
	} else {
		record.Set("password_hash", "")
	}

	// Set expiration
	if params.ExpiresAt != nil {
		record.Set("expires_at", *params.ExpiresAt)
	} else {
		record.Set("expires_at", time.Time{})
	}

	// Save share record
	if err := s.app.Save(record); err != nil {
		log.Error().Err(err).Msg("Failed to create share")
		return nil, errors.New("failed to create share")
	}

	log.Info().
		Str("share_id", record.Id).
		Str("user_id", params.UserID).
		Str("resource_type", params.ResourceType).
		Str("resource_id", params.ResourceID).
		Str("permission_type", params.PermissionType).
		Msg("Share created successfully")

	// Convert to ShareInfo
	return s.recordToShareInfo(record), nil
}

// GetShareByToken retrieves a share by its token
func (s *ShareServiceImpl) GetShareByToken(token string) (*ShareInfo, error) {
	if token == "" {
		return nil, errors.New("share token is required")
	}

	record, err := s.app.FindFirstRecordByData("shares", "share_token", token)
	if err != nil {
		log.Debug().Err(err).Str("share_token", token).Msg("Share not found")
		return nil, errors.New("share not found")
	}

	return s.recordToShareInfo(record), nil
}

// GetShareByID retrieves a share by its ID
func (s *ShareServiceImpl) GetShareByID(shareID string) (*ShareInfo, error) {
	if shareID == "" {
		return nil, errors.New("share ID is required")
	}

	record, err := s.app.FindRecordById("shares", shareID)
	if err != nil {
		log.Debug().Err(err).Str("share_id", shareID).Msg("Share not found")
		return nil, errors.New("share not found")
	}

	return s.recordToShareInfo(record), nil
}

// ValidateShareAccess validates a share token and optional password
func (s *ShareServiceImpl) ValidateShareAccess(token, password string) (*ShareAccessInfo, error) {
	// Check for empty or missing token - treat as invalid for security
	if token == "" {
		return &ShareAccessInfo{
			IsValid:      false,
			ErrorMessage: "Invalid share link",
		}, nil
	}

	// Get share by token
	share, err := s.GetShareByToken(token)
	if err != nil {
		return &ShareAccessInfo{
			IsValid:      false,
			ErrorMessage: "Invalid share link",
		}, nil
	}

	// Check if expired
	if share.IsExpired {
		log.Info().Str("share_id", share.ID).Msg("Expired share accessed")
		return &ShareAccessInfo{
			ShareID:      share.ID,
			ResourceType: string(share.ResourceType),
			ResourceID:   share.ResourceID,
			IsValid:      false,
			ErrorMessage: "This share link has expired",
		}, nil
	}

	// Check password if required
	if share.IsPasswordProtected {
		if password == "" {
			return &ShareAccessInfo{
				ShareID:      share.ID,
				ResourceType: string(share.ResourceType),
				ResourceID:   share.ResourceID,
				IsValid:      false,
				ErrorMessage: "Password required",
			}, nil
		}

		// Get the record to access password hash
		record, err := s.app.FindRecordById("shares", share.ID)
		if err != nil {
			return nil, errors.New("failed to validate password")
		}

		passwordHash := record.GetString("password_hash")

		// Use constant-time comparison to prevent timing attacks
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
		if err != nil {
			log.Warn().Str("share_id", share.ID).Msg("Invalid password attempt")
			return &ShareAccessInfo{
				ShareID:      share.ID,
				ResourceType: string(share.ResourceType),
				ResourceID:   share.ResourceID,
				IsValid:      false,
				ErrorMessage: "Invalid password",
			}, nil
		}
	}

	// Valid access
	return &ShareAccessInfo{
		ShareID:        share.ID,
		ResourceType:   string(share.ResourceType),
		ResourceID:     share.ResourceID,
		PermissionType: string(share.PermissionType),
		ExpiresAt:      share.ExpiresAt,
		IsValid:        true,
		ErrorMessage:   "",
	}, nil
}

// RevokeShare deletes a share, preventing further access
func (s *ShareServiceImpl) RevokeShare(shareID, userID string) error {
	if shareID == "" {
		return errors.New("share ID is required")
	}
	if userID == "" {
		return errors.New("user ID is required")
	}

	// Get share record
	share, err := s.app.FindRecordById("shares", shareID)
	if err != nil {
		log.Debug().Err(err).Str("share_id", shareID).Msg("Share not found")
		return errors.New("share not found")
	}

	// Verify user owns the share
	if share.GetString("user") != userID {
		log.Warn().Str("user_id", userID).Str("share_id", shareID).Msg("User does not own share")
		return errors.New("you do not have permission to revoke this share")
	}

	// Delete the share
	if err := s.app.Delete(share); err != nil {
		log.Error().Err(err).Str("share_id", shareID).Msg("Failed to revoke share")
		return errors.New("failed to revoke share")
	}

	log.Info().Str("share_id", shareID).Str("user_id", userID).Msg("Share revoked successfully")
	return nil
}

// ListUserShares returns all shares created by a user
func (s *ShareServiceImpl) ListUserShares(userID, resourceType string) ([]*ShareInfo, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	// Build filter
	filter := fmt.Sprintf("user = '%s'", userID)
	if resourceType != "" {
		filter = fmt.Sprintf("%s && resource_type = '%s'", filter, resourceType)
	}

	// Query shares
	records, err := s.app.FindRecordsByFilter(
		"shares",
		filter,
		"-created", // Order by created date, newest first
		0,
		0,
	)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to list shares")
		return nil, errors.New("failed to list shares")
	}

	// Convert to ShareInfo
	shares := make([]*ShareInfo, 0, len(records))
	for _, record := range records {
		shares = append(shares, s.recordToShareInfo(record))
	}

	return shares, nil
}

// UpdateShareExpiration updates the expiration date of a share
func (s *ShareServiceImpl) UpdateShareExpiration(shareID, userID string, expiresAt *time.Time) error {
	if shareID == "" {
		return errors.New("share ID is required")
	}
	if userID == "" {
		return errors.New("user ID is required")
	}

	// Get share record
	share, err := s.app.FindRecordById("shares", shareID)
	if err != nil {
		log.Debug().Err(err).Str("share_id", shareID).Msg("Share not found")
		return errors.New("share not found")
	}

	// Verify user owns the share
	if share.GetString("user") != userID {
		log.Warn().Str("user_id", userID).Str("share_id", shareID).Msg("User does not own share")
		return errors.New("you do not have permission to update this share")
	}

	// Validate expiration is in future
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return errors.New("expiration date must be in the future")
	}

	// Update expiration
	if expiresAt != nil {
		share.Set("expires_at", *expiresAt)
	} else {
		share.Set("expires_at", time.Time{})
	}

	if err := s.app.Save(share); err != nil {
		log.Error().Err(err).Str("share_id", shareID).Msg("Failed to update share expiration")
		return errors.New("failed to update share expiration")
	}

	log.Info().Str("share_id", shareID).Msg("Share expiration updated successfully")
	return nil
}

// GetShareAccessLogs retrieves access logs for a share
func (s *ShareServiceImpl) GetShareAccessLogs(shareID, userID string) ([]*ShareAccessLog, error) {
	if shareID == "" {
		return nil, errors.New("share ID is required")
	}
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	// Verify user owns the share
	share, err := s.app.FindRecordById("shares", shareID)
	if err != nil {
		return nil, errors.New("share not found")
	}

	if share.GetString("user") != userID {
		return nil, errors.New("you do not have permission to view these logs")
	}

	// Query access logs
	records, err := s.app.FindRecordsByFilter(
		"share_access_logs",
		fmt.Sprintf("share = '%s'", shareID),
		"-accessed_at", // Order by accessed date, newest first
		100,            // Limit to last 100 entries
		0,
	)
	if err != nil {
		log.Error().Err(err).Str("share_id", shareID).Msg("Failed to get access logs")
		return nil, errors.New("failed to get access logs")
	}

	// Convert to ShareAccessLog
	logs := make([]*ShareAccessLog, 0, len(records))
	for _, record := range records {
		logs = append(logs, &ShareAccessLog{
			ID:         record.Id,
			ShareID:    record.GetString("share"),
			Action:     record.GetString("action"),
			FileName:   record.GetString("file_name"),
			IPAddress:  record.GetString("ip_address"),
			UserAgent:  record.GetString("user_agent"),
			AccessedAt: record.GetDateTime("accessed_at").Time(),
		})
	}

	return logs, nil
}

// LogShareAccess logs an access event for a share
func (s *ShareServiceImpl) LogShareAccess(shareID, action, fileName, ipAddress, userAgent string) error {
	if shareID == "" {
		return errors.New("share ID is required")
	}
	if action == "" {
		return errors.New("action is required")
	}

	// Create access log entry
	collection, err := s.app.FindCollectionByNameOrId("share_access_logs")
	if err != nil {
		log.Error().Err(err).Msg("Failed to find share_access_logs collection")
		return fmt.Errorf("failed to find share_access_logs collection: %w", err)
	}
	if collection == nil {
		// Collection doesn't exist yet, skip logging
		log.Warn().Msg("share_access_logs collection not found, skipping access logging")
		return nil
	}

	record := core.NewRecord(collection)
	record.Set("share", shareID)
	record.Set("action", action)
	record.Set("file_name", fileName)
	record.Set("ip_address", ipAddress)
	record.Set("user_agent", userAgent)
	record.Set("accessed_at", time.Now())

	if err := s.app.Save(record); err != nil {
		log.Error().Err(err).Str("share_id", shareID).Msg("Failed to log share access")
		// Don't return error - logging failure shouldn't block access
		return nil
	}

	log.Debug().
		Str("share_id", shareID).
		Str("action", action).
		Str("ip_address", ipAddress).
		Msg("Share access logged")

	return nil
}

// IncrementAccessCount increments the access count for a share
func (s *ShareServiceImpl) IncrementAccessCount(shareID string) error {
	if shareID == "" {
		return errors.New("share ID is required")
	}

	share, err := s.app.FindRecordById("shares", shareID)
	if err != nil {
		return errors.New("share not found")
	}

	currentCount := int64(share.GetInt("access_count"))
	share.Set("access_count", currentCount+1)

	if err := s.app.Save(share); err != nil {
		log.Error().Err(err).Str("share_id", shareID).Msg("Failed to increment access count")
		// Don't return error - counter increment failure shouldn't block access
		return nil
	}

	return nil
}

// recordToShareInfo converts a PocketBase record to ShareInfo
func (s *ShareServiceImpl) recordToShareInfo(record *core.Record) *ShareInfo {
	// Determine resource ID based on type
	resourceType := record.GetString("resource_type")
	resourceID := record.GetString("file")
	if resourceType == "directory" {
		resourceID = record.GetString("directory")
	}

	// Parse expiration
	var expiresAt *time.Time
	expiresAtTime := record.GetDateTime("expires_at")
	if !expiresAtTime.IsZero() {
		t := expiresAtTime.Time()
		expiresAt = &t
	}

	// Check if expired
	isExpired := false
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		isExpired = true
	}

	return &ShareInfo{
		ID:                  record.Id,
		UserID:              record.GetString("user"),
		ResourceType:        models.ResourceType(resourceType),
		ResourceID:          resourceID,
		ShareToken:          record.GetString("share_token"),
		PermissionType:      models.PermissionType(record.GetString("permission_type")),
		ExpiresAt:           expiresAt,
		AccessCount:         int64(record.GetInt("access_count")),
		Created:             record.GetDateTime("created").Time(),
		Updated:             record.GetDateTime("updated").Time(),
		IsExpired:           isExpired,
		IsPasswordProtected: record.GetString("password_hash") != "",
	}
}
