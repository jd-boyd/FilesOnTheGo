package services

import (
	"errors"
	"sync"
	"time"

	"github.com/jd-boyd/filesonthego/models"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// PermissionService handles all permission validation logic
type PermissionService struct {
	db              *gorm.DB
	logger          zerolog.Logger
	rateLimiter     map[string]*rateLimitEntry
	rateLimiterLock sync.RWMutex
}

// rateLimitEntry tracks rate limit information for share tokens
type rateLimitEntry struct {
	attempts  int
	firstSeen time.Time
	blocked   bool
}

// SharePermissions represents the permissions granted by a share token
type SharePermissions struct {
	ShareID          string
	ResourceType     string // "file" or "directory"
	ResourceID       string
	PermissionType   string // "read", "read_upload", "upload_only"
	IsExpired        bool
	RequiresPassword bool
	Share            *models.Share
}

// QuotaInfo represents user storage quota information
type QuotaInfo struct {
	TotalQuota int64
	UsedQuota  int64
	Available  int64
	Percentage float64
}

// NewPermissionService creates a new permission service instance
func NewPermissionService(db *gorm.DB, logger zerolog.Logger) *PermissionService {
	service := &PermissionService{
		db:          db,
		logger:      logger,
		rateLimiter: make(map[string]*rateLimitEntry),
	}

	// Start background goroutine to clean up old rate limit entries
	go service.cleanupRateLimiter()

	return service
}

// cleanupRateLimiter periodically removes old rate limit entries
func (s *PermissionService) cleanupRateLimiter() {
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
func (s *PermissionService) checkRateLimit(identifier string) bool {
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
		s.logger.Warn().
			Str("identifier", identifier).
			Int("attempts", entry.attempts).
			Msg("Rate limit exceeded for share token validation")
		return false
	}

	return true
}

// CanReadFile checks if a user or share token can read a file
func (s *PermissionService) CanReadFile(userID, fileID, shareToken string) (bool, error) {
	// Check if user owns the file
	if userID != "" {
		var file models.File
		if err := s.db.First(&file, "id = ?", fileID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, errors.New("file not found")
			}
			return false, err
		}

		if file.User == userID {
			return true, nil
		}
	}

	// Check share permissions
	if shareToken != "" {
		sharePerms, err := s.ValidateShareToken(shareToken, "")
		if err != nil || sharePerms.IsExpired {
			return false, nil
		}

		// Check if share is for this file
		if sharePerms.ResourceType == string(models.ResourceTypeFile) && sharePerms.ResourceID == fileID {
			return s.canPerformAction(sharePerms.PermissionType, "read"), nil
		}

		// Check if file is in shared directory
		if sharePerms.ResourceType == string(models.ResourceTypeDirectory) {
			var file models.File
			if err := s.db.First(&file, "id = ?", fileID).Error; err == nil {
				if s.isFileInDirectory(fileID, sharePerms.ResourceID) {
					return s.canPerformAction(sharePerms.PermissionType, "read"), nil
				}
			}
		}
	}

	return false, nil
}

// CanUploadFile checks if a user or share token can upload to a directory
func (s *PermissionService) CanUploadFile(userID, directoryID, shareToken string) (bool, error) {
	// Root directory upload (no parent)
	if directoryID == "" {
		return userID != "", nil
	}

	// Check if user owns the directory
	if userID != "" {
		var dir models.Directory
		if err := s.db.First(&dir, "id = ?", directoryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, errors.New("directory not found")
			}
			return false, err
		}

		if dir.User == userID {
			return true, nil
		}
	}

	// Check share permissions
	if shareToken != "" {
		sharePerms, err := s.ValidateShareToken(shareToken, "")
		if err != nil || sharePerms.IsExpired {
			return false, nil
		}

		if sharePerms.ResourceType == string(models.ResourceTypeDirectory) && sharePerms.ResourceID == directoryID {
			return s.canPerformAction(sharePerms.PermissionType, "upload"), nil
		}
	}

	return false, nil
}

// CanDeleteFile checks if a user can delete a file
func (s *PermissionService) CanDeleteFile(userID, fileID string) (bool, error) {
	if userID == "" {
		return false, nil
	}

	var file models.File
	if err := s.db.First(&file, "id = ?", fileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("file not found")
		}
		return false, err
	}

	return file.User == userID, nil
}

// CanReadDirectory checks if a user or share token can read a directory
func (s *PermissionService) CanReadDirectory(userID, directoryID, shareToken string) (bool, error) {
	// Root directory is always readable if authenticated
	if directoryID == "" {
		return userID != "", nil
	}

	// Check if user owns the directory
	if userID != "" {
		var dir models.Directory
		if err := s.db.First(&dir, "id = ?", directoryID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, errors.New("directory not found")
			}
			return false, err
		}

		if dir.User == userID {
			return true, nil
		}
	}

	// Check share permissions
	if shareToken != "" {
		sharePerms, err := s.ValidateShareToken(shareToken, "")
		if err != nil || sharePerms.IsExpired {
			return false, nil
		}

		if sharePerms.ResourceType == string(models.ResourceTypeDirectory) && sharePerms.ResourceID == directoryID {
			return s.canPerformAction(sharePerms.PermissionType, "view"), nil
		}
	}

	return false, nil
}

// CanCreateDirectory checks if a user can create a directory
func (s *PermissionService) CanCreateDirectory(userID, parentDirID string) (bool, error) {
	if userID == "" {
		return false, nil
	}

	// Can always create in root
	if parentDirID == "" {
		return true, nil
	}

	// Check if user owns parent directory
	var dir models.Directory
	if err := s.db.First(&dir, "id = ?", parentDirID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("parent directory not found")
		}
		return false, err
	}

	return dir.User == userID, nil
}

// CanDeleteDirectory checks if a user can delete a directory
func (s *PermissionService) CanDeleteDirectory(userID, directoryID string) (bool, error) {
	if userID == "" || directoryID == "" {
		return false, nil
	}

	var dir models.Directory
	if err := s.db.First(&dir, "id = ?", directoryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("directory not found")
		}
		return false, err
	}

	return dir.User == userID, nil
}

// CanCreateShare checks if a user can create a share for a resource
func (s *PermissionService) CanCreateShare(userID, resourceID, resourceType string) (bool, error) {
	if userID == "" {
		return false, nil
	}

	if resourceType == string(models.ResourceTypeFile) {
		var file models.File
		if err := s.db.First(&file, "id = ?", resourceID).Error; err != nil {
			return false, errors.New("file not found")
		}
		return file.User == userID, nil
	}

	if resourceType == string(models.ResourceTypeDirectory) {
		var dir models.Directory
		if err := s.db.First(&dir, "id = ?", resourceID).Error; err != nil {
			return false, errors.New("directory not found")
		}
		return dir.User == userID, nil
	}

	return false, errors.New("invalid resource type")
}

// CanRevokeShare checks if a user can revoke a share
func (s *PermissionService) CanRevokeShare(userID, shareID string) (bool, error) {
	if userID == "" {
		return false, nil
	}

	var share models.Share
	if err := s.db.First(&share, "id = ?", shareID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("share not found")
		}
		return false, err
	}

	return share.User == userID, nil
}

// ValidateShareToken validates a share token and returns permissions
func (s *PermissionService) ValidateShareToken(shareToken, password string) (*SharePermissions, error) {
	if !s.checkRateLimit(shareToken) {
		return nil, errors.New("rate limit exceeded")
	}

	var share models.Share
	if err := s.db.Where("share_token = ?", shareToken).First(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid share token")
		}
		return nil, err
	}

	// Check if share is expired
	isExpired := share.IsExpired()

	// Validate password if required
	requiresPassword := share.IsPasswordProtected()
	if requiresPassword && !share.ValidatePassword(password) {
		return nil, errors.New("invalid password")
	}

	// Determine resource ID
	resourceID := share.File
	if share.ResourceType == models.ResourceTypeDirectory {
		resourceID = share.Directory
	}

	return &SharePermissions{
		ShareID:          share.ID,
		ResourceType:     string(share.ResourceType),
		ResourceID:       resourceID,
		PermissionType:   string(share.PermissionType),
		IsExpired:        isExpired,
		RequiresPassword: requiresPassword,
		Share:            &share,
	}, nil
}

// CanUploadSize checks if user has enough quota for file size
func (s *PermissionService) CanUploadSize(userID string, fileSize int64) (bool, error) {
	if userID == "" {
		return false, nil
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}

	return user.HasQuotaAvailable(fileSize), nil
}

// GetUserQuota returns user quota information
func (s *PermissionService) GetUserQuota(userID string) (*QuotaInfo, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	available := user.GetAvailableQuota()
	if available < 0 {
		available = 0
	}

	return &QuotaInfo{
		TotalQuota: user.StorageQuota,
		UsedQuota:  user.StorageUsed,
		Available:  available,
		Percentage: user.GetQuotaUsagePercent(),
	}, nil
}

// Helper methods

func (s *PermissionService) canPerformAction(permissionType, action string) bool {
	pt := models.PermissionType(permissionType)
	switch pt {
	case models.PermissionRead:
		return action == "view" || action == "download" || action == "read"
	case models.PermissionReadUpload:
		return true // All actions allowed
	case models.PermissionUploadOnly:
		return action == "upload"
	default:
		return false
	}
}

func (s *PermissionService) isFileInDirectory(fileID, directoryID string) bool {
	var file models.File
	if err := s.db.First(&file, "id = ?", fileID).Error; err != nil {
		return false
	}

	// Check direct parent
	if file.ParentDirectory == directoryID {
		return true
	}

	// Check if in subdirectory (recursive check)
	if file.ParentDirectory != "" {
		var parentDir models.Directory
		if err := s.db.First(&parentDir, "id = ?", file.ParentDirectory).Error; err == nil {
			return s.isDirectoryInDirectory(parentDir.ID, directoryID, 0)
		}
	}

	return false
}

func (s *PermissionService) isDirectoryInDirectory(childDirID, parentDirID string, depth int) bool {
	// Prevent infinite recursion
	if depth > 100 {
		return false
	}

	if childDirID == parentDirID {
		return true
	}

	var dir models.Directory
	if err := s.db.First(&dir, "id = ?", childDirID).Error; err != nil {
		return false
	}

	if dir.ParentDirectory == "" {
		return false
	}

	if dir.ParentDirectory == parentDirID {
		return true
	}

	return s.isDirectoryInDirectory(dir.ParentDirectory, parentDirID, depth+1)
}
