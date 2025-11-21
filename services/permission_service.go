package services

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// PermissionService defines the interface for permission validation
type PermissionService interface {
	// File permissions
	CanReadFile(userID, fileID, shareToken string) (bool, error)
	CanUploadFile(userID, directoryID, shareToken string) (bool, error)
	CanDeleteFile(userID, fileID string) (bool, error)
	CanMoveFile(userID, fileID, targetDirID string) (bool, error)

	// Directory permissions
	CanReadDirectory(userID, directoryID, shareToken string) (bool, error)
	CanCreateDirectory(userID, parentDirID string) (bool, error)
	CanDeleteDirectory(userID, directoryID string) (bool, error)

	// Share permissions
	CanCreateShare(userID, resourceID, resourceType string) (bool, error)
	CanRevokeShare(userID, shareID string) (bool, error)

	// Share token validation
	ValidateShareToken(shareToken, password string) (*SharePermissions, error)

	// Quota checks
	CanUploadSize(userID string, fileSize int64) (bool, error)
	GetUserQuota(userID string) (*QuotaInfo, error)
}

// SharePermissions represents the permissions granted by a share token
type SharePermissions struct {
	ShareID          string
	ResourceType     string // "file" or "directory"
	ResourceID       string
	PermissionType   string // "read", "read_upload", "upload_only"
	IsExpired        bool
	RequiresPassword bool
}

// QuotaInfo represents user storage quota information
type QuotaInfo struct {
	TotalQuota int64
	UsedQuota  int64
	Available  int64
	Percentage float64
}

// PermissionServiceImpl implements the PermissionService interface
type PermissionServiceImpl struct {
	app *pocketbase.PocketBase

	// Rate limiting for share token validation
	rateLimiter     map[string]*rateLimitEntry
	rateLimiterLock sync.RWMutex
}

// rateLimitEntry tracks rate limit information for share tokens
type rateLimitEntry struct {
	attempts  int
	firstSeen time.Time
	blocked   bool
}

// NewPermissionService creates a new permission service instance
func NewPermissionService(app *pocketbase.PocketBase) *PermissionServiceImpl {
	service := &PermissionServiceImpl{
		app:         app,
		rateLimiter: make(map[string]*rateLimitEntry),
	}

	// Start background goroutine to clean up old rate limit entries
	go service.cleanupRateLimiter()

	return service
}

// cleanupRateLimiter periodically removes old rate limit entries
func (s *PermissionServiceImpl) cleanupRateLimiter() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.rateLimiterLock.Lock()
		now := time.Now()
		for key, entry := range s.rateLimiter {
			// Remove entries older than 15 minutes
			if now.Sub(entry.firstSeen) > 15*time.Minute {
				delete(s.rateLimiter, key)
			}
		}
		s.rateLimiterLock.Unlock()
	}
}

// checkRateLimit checks if the given identifier has exceeded rate limits
func (s *PermissionServiceImpl) checkRateLimit(identifier string) bool {
	s.rateLimiterLock.Lock()
	defer s.rateLimiterLock.Unlock()

	now := time.Now()
	entry, exists := s.rateLimiter[identifier]

	if !exists {
		s.rateLimiter[identifier] = &rateLimitEntry{
			attempts:  1,
			firstSeen: now,
			blocked:   false,
		}
		return true
	}

	// If blocked, check if block period has expired (15 minutes)
	if entry.blocked {
		if now.Sub(entry.firstSeen) > 15*time.Minute {
			// Reset the entry
			entry.attempts = 1
			entry.firstSeen = now
			entry.blocked = false
			return true
		}
		return false
	}

	// Reset if window has expired (1 minute)
	if now.Sub(entry.firstSeen) > 1*time.Minute {
		entry.attempts = 1
		entry.firstSeen = now
		return true
	}

	// Increment attempts
	entry.attempts++

	// Block if too many attempts (10 per minute)
	if entry.attempts > 10 {
		entry.blocked = true
		log.Warn().
			Str("identifier", identifier).
			Int("attempts", entry.attempts).
			Msg("Rate limit exceeded for share token validation")
		return false
	}

	return true
}

// CanReadFile checks if a user or share token can read a file
func (s *PermissionServiceImpl) CanReadFile(userID, fileID, shareToken string) (bool, error) {
	// Check if user owns the file
	if userID != "" {
		file, err := s.app.FindRecordById("files", fileID)
		if err != nil {
			log.Debug().Err(err).Str("file_id", fileID).Msg("File not found")
			return false, errors.New("file not found")
		}

		if file.GetString("user") == userID {
			return true, nil
		}
	}

	// Check share permissions
	if shareToken != "" {
		sharePerms, err := s.ValidateShareToken(shareToken, "")
		if err != nil {
			s.logPermissionDenial(userID, "read_file", fileID, "invalid_share_token")
			return false, nil
		}

		if sharePerms.IsExpired {
			s.logPermissionDenial(userID, "read_file", fileID, "expired_share")
			return false, nil
		}

		// Check if share is for this file or its parent directory
		if sharePerms.ResourceType == "file" && sharePerms.ResourceID == fileID {
			return s.validateShareAction(sharePerms.PermissionType, "read"), nil
		}

		if sharePerms.ResourceType == "directory" {
			// Check if file is in shared directory
			file, err := s.app.FindRecordById("files", fileID)
			if err != nil {
				return false, errors.New("file not found")
			}

			if s.isFileInDirectory(file, sharePerms.ResourceID) {
				return s.validateShareAction(sharePerms.PermissionType, "read"), nil
			}
		}
	}

	s.logPermissionDenial(userID, "read_file", fileID, "no_permission")
	return false, nil
}

// CanUploadFile checks if a user or share token can upload a file to a directory
func (s *PermissionServiceImpl) CanUploadFile(userID, directoryID, shareToken string) (bool, error) {
	// Check if user owns the directory
	if userID != "" {
		if directoryID == "" || directoryID == "root" {
			// Uploading to root - check if user is authenticated
			return userID != "", nil
		}

		directory, err := s.app.FindRecordById("directories", directoryID)
		if err != nil {
			log.Debug().Err(err).Str("directory_id", directoryID).Msg("Directory not found")
			return false, errors.New("directory not found")
		}

		if directory.GetString("user") == userID {
			return true, nil
		}
	}

	// Check share permissions
	if shareToken != "" {
		sharePerms, err := s.ValidateShareToken(shareToken, "")
		if err != nil {
			s.logPermissionDenial(userID, "upload_file", directoryID, "invalid_share_token")
			return false, nil
		}

		if sharePerms.IsExpired {
			s.logPermissionDenial(userID, "upload_file", directoryID, "expired_share")
			return false, nil
		}

		// Check if share is for this directory
		if sharePerms.ResourceType == "directory" && sharePerms.ResourceID == directoryID {
			return s.validateShareAction(sharePerms.PermissionType, "upload"), nil
		}
	}

	s.logPermissionDenial(userID, "upload_file", directoryID, "no_permission")
	return false, nil
}

// CanDeleteFile checks if a user can delete a file (shares cannot delete)
func (s *PermissionServiceImpl) CanDeleteFile(userID, fileID string) (bool, error) {
	if userID == "" {
		s.logPermissionDenial(userID, "delete_file", fileID, "no_user")
		return false, nil
	}

	file, err := s.app.FindRecordById("files", fileID)
	if err != nil {
		log.Debug().Err(err).Str("file_id", fileID).Msg("File not found")
		return false, errors.New("file not found")
	}

	if file.GetString("user") == userID {
		return true, nil
	}

	s.logPermissionDenial(userID, "delete_file", fileID, "not_owner")
	return false, nil
}

// CanMoveFile checks if a user can move a file to a target directory
func (s *PermissionServiceImpl) CanMoveFile(userID, fileID, targetDirID string) (bool, error) {
	if userID == "" {
		s.logPermissionDenial(userID, "move_file", fileID, "no_user")
		return false, nil
	}

	// Check if user owns the file
	file, err := s.app.FindRecordById("files", fileID)
	if err != nil {
		log.Debug().Err(err).Str("file_id", fileID).Msg("File not found")
		return false, errors.New("file not found")
	}

	if file.GetString("user") != userID {
		s.logPermissionDenial(userID, "move_file", fileID, "not_owner")
		return false, nil
	}

	// Check if user owns the target directory
	if targetDirID != "" && targetDirID != "root" {
		targetDir, err := s.app.FindRecordById("directories", targetDirID)
		if err != nil {
			log.Debug().Err(err).Str("directory_id", targetDirID).Msg("Target directory not found")
			return false, errors.New("target directory not found")
		}

		if targetDir.GetString("user") != userID {
			s.logPermissionDenial(userID, "move_file", fileID, "target_not_owned")
			return false, nil
		}
	}

	return true, nil
}

// CanReadDirectory checks if a user or share token can read a directory
func (s *PermissionServiceImpl) CanReadDirectory(userID, directoryID, shareToken string) (bool, error) {
	// Root directory
	if directoryID == "" || directoryID == "root" {
		return userID != "", nil
	}

	// Check if user owns the directory
	if userID != "" {
		directory, err := s.app.FindRecordById("directories", directoryID)
		if err != nil {
			log.Debug().Err(err).Str("directory_id", directoryID).Msg("Directory not found")
			return false, errors.New("directory not found")
		}

		if directory.GetString("user") == userID {
			return true, nil
		}
	}

	// Check share permissions
	if shareToken != "" {
		sharePerms, err := s.ValidateShareToken(shareToken, "")
		if err != nil {
			s.logPermissionDenial(userID, "read_directory", directoryID, "invalid_share_token")
			return false, nil
		}

		if sharePerms.IsExpired {
			s.logPermissionDenial(userID, "read_directory", directoryID, "expired_share")
			return false, nil
		}

		// Check if share is for this directory
		if sharePerms.ResourceType == "directory" && sharePerms.ResourceID == directoryID {
			return s.validateShareAction(sharePerms.PermissionType, "read"), nil
		}
	}

	s.logPermissionDenial(userID, "read_directory", directoryID, "no_permission")
	return false, nil
}

// CanCreateDirectory checks if a user can create a directory
func (s *PermissionServiceImpl) CanCreateDirectory(userID, parentDirID string) (bool, error) {
	if userID == "" {
		s.logPermissionDenial(userID, "create_directory", parentDirID, "no_user")
		return false, nil
	}

	// Root directory
	if parentDirID == "" || parentDirID == "root" {
		return true, nil
	}

	// Check if user owns the parent directory
	parentDir, err := s.app.FindRecordById("directories", parentDirID)
	if err != nil {
		log.Debug().Err(err).Str("directory_id", parentDirID).Msg("Parent directory not found")
		return false, errors.New("parent directory not found")
	}

	if parentDir.GetString("user") == userID {
		return true, nil
	}

	s.logPermissionDenial(userID, "create_directory", parentDirID, "not_owner")
	return false, nil
}

// CanDeleteDirectory checks if a user can delete a directory (shares cannot delete)
func (s *PermissionServiceImpl) CanDeleteDirectory(userID, directoryID string) (bool, error) {
	if userID == "" {
		s.logPermissionDenial(userID, "delete_directory", directoryID, "no_user")
		return false, nil
	}

	directory, err := s.app.FindRecordById("directories", directoryID)
	if err != nil {
		log.Debug().Err(err).Str("directory_id", directoryID).Msg("Directory not found")
		return false, errors.New("directory not found")
	}

	if directory.GetString("user") == userID {
		return true, nil
	}

	s.logPermissionDenial(userID, "delete_directory", directoryID, "not_owner")
	return false, nil
}

// CanCreateShare checks if a user can create a share for a resource
func (s *PermissionServiceImpl) CanCreateShare(userID, resourceID, resourceType string) (bool, error) {
	if userID == "" {
		s.logPermissionDenial(userID, "create_share", resourceID, "no_user")
		return false, nil
	}

	// Check if user owns the resource
	collection := "files"
	if resourceType == "directory" {
		collection = "directories"
	}

	resource, err := s.app.FindRecordById(collection, resourceID)
	if err != nil {
		log.Debug().Err(err).Str("resource_id", resourceID).Msg("Resource not found")
		return false, fmt.Errorf("%s not found", resourceType)
	}

	if resource.GetString("user") == userID {
		return true, nil
	}

	s.logPermissionDenial(userID, "create_share", resourceID, "not_owner")
	return false, nil
}

// CanRevokeShare checks if a user can revoke a share
func (s *PermissionServiceImpl) CanRevokeShare(userID, shareID string) (bool, error) {
	if userID == "" {
		s.logPermissionDenial(userID, "revoke_share", shareID, "no_user")
		return false, nil
	}

	share, err := s.app.FindRecordById("shares", shareID)
	if err != nil {
		log.Debug().Err(err).Str("share_id", shareID).Msg("Share not found")
		return false, errors.New("share not found")
	}

	if share.GetString("user") == userID {
		return true, nil
	}

	s.logPermissionDenial(userID, "revoke_share", shareID, "not_creator")
	return false, nil
}

// ValidateShareToken validates a share token and returns its permissions
func (s *PermissionServiceImpl) ValidateShareToken(shareToken, password string) (*SharePermissions, error) {
	if shareToken == "" {
		return nil, errors.New("share token is required")
	}

	// Check rate limit
	if !s.checkRateLimit(shareToken) {
		return nil, errors.New("rate limit exceeded")
	}

	// Find share by token
	share, err := s.app.FindFirstRecordByData("shares", "share_token", shareToken)
	if err != nil {
		log.Debug().Err(err).Str("share_token", shareToken).Msg("Share not found")
		return nil, errors.New("invalid share token")
	}

	// Check if expired
	expiresAt := share.GetDateTime("expires_at")
	isExpired := !expiresAt.IsZero() && expiresAt.Time().Before(time.Now())

	// Check password if required
	passwordHash := share.GetString("password_hash")
	requiresPassword := passwordHash != ""

	if requiresPassword {
		if password == "" {
			return &SharePermissions{
				ShareID:          share.Id,
				ResourceType:     share.GetString("resource_type"),
				ResourceID:       s.getResourceID(share),
				PermissionType:   share.GetString("permission_type"),
				IsExpired:        isExpired,
				RequiresPassword: true,
			}, errors.New("password required")
		}

		// Use constant-time comparison to prevent timing attacks
		err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
		if err != nil {
			s.logPermissionDenial("", "validate_share_token", shareToken, "invalid_password")
			return nil, errors.New("invalid password")
		}
	}

	resourceID := s.getResourceID(share)

	return &SharePermissions{
		ShareID:          share.Id,
		ResourceType:     share.GetString("resource_type"),
		ResourceID:       resourceID,
		PermissionType:   share.GetString("permission_type"),
		IsExpired:        isExpired,
		RequiresPassword: requiresPassword,
	}, nil
}

// CanUploadSize checks if a user has enough quota to upload a file
func (s *PermissionServiceImpl) CanUploadSize(userID string, fileSize int64) (bool, error) {
	if userID == "" {
		return false, errors.New("user ID is required")
	}

	quota, err := s.GetUserQuota(userID)
	if err != nil {
		return false, err
	}

	if quota.Available < fileSize {
		s.logPermissionDenial(userID, "upload_size", "", "quota_exceeded")
		return false, nil
	}

	return true, nil
}

// GetUserQuota returns the current quota information for a user
func (s *PermissionServiceImpl) GetUserQuota(userID string) (*QuotaInfo, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	// Get user's total quota (default 100GB if not set)
	user, err := s.app.FindRecordById("users", userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	totalQuota := int64(user.GetInt("quota"))
	if totalQuota == 0 {
		totalQuota = 107374182400 // 100GB default
	}

	// Calculate used quota by summing file sizes
	usedQuota := int64(0)
	files, err := s.app.FindRecordsByFilter(
		"files",
		"user = {:user}",
		"",
		0,
		0,
		map[string]any{
			"user": userID,
		},
	)
	if err == nil {
		for _, file := range files {
			usedQuota += int64(file.GetInt("size"))
		}
	}

	available := totalQuota - usedQuota
	if available < 0 {
		available = 0
	}

	percentage := float64(usedQuota) / float64(totalQuota) * 100
	if totalQuota == 0 {
		percentage = 0
	}

	return &QuotaInfo{
		TotalQuota: totalQuota,
		UsedQuota:  usedQuota,
		Available:  available,
		Percentage: percentage,
	}, nil
}

// validateShareAction checks if an action is allowed for a given permission type
func (s *PermissionServiceImpl) validateShareAction(permissionType, action string) bool {
	switch permissionType {
	case "read":
		// Read permission allows view and download
		return action == "read" || action == "download" || action == "view"
	case "read_upload":
		// Read/Upload permission allows view, download, and upload
		return action == "read" || action == "download" || action == "view" || action == "upload"
	case "upload_only":
		// Upload-only allows view (names only) and upload, but not download
		return action == "view" || action == "upload"
	default:
		return false
	}
}

// isFileInDirectory checks if a file is in a given directory
func (s *PermissionServiceImpl) isFileInDirectory(file *core.Record, directoryID string) bool {
	parentDirID := file.GetString("parent_directory")
	if parentDirID == directoryID {
		return true
	}

	// Check parent directories recursively (with cycle detection)
	visited := make(map[string]bool)
	for parentDirID != "" && !visited[parentDirID] {
		visited[parentDirID] = true

		if parentDirID == directoryID {
			return true
		}

		parentDir, err := s.app.FindRecordById("directories", parentDirID)
		if err != nil {
			break
		}

		parentDirID = parentDir.GetString("parent_directory")
	}

	return false
}

// getResourceID extracts the resource ID from a share record
func (s *PermissionServiceImpl) getResourceID(share *core.Record) string {
	resourceType := share.GetString("resource_type")
	if resourceType == "file" {
		return share.GetString("file")
	}
	return share.GetString("directory")
}

// logPermissionDenial logs when permission is denied for auditing purposes
func (s *PermissionServiceImpl) logPermissionDenial(userID, action, resourceID, reason string) {
	log.Warn().
		Str("user_id", userID).
		Str("action", action).
		Str("resource_id", resourceID).
		Str("reason", reason).
		Msg("Permission denied")
}
