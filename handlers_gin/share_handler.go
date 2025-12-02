package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// ShareHandler handles share link operations
type ShareHandler struct {
	db                *gorm.DB
	shareService      *services.ShareService
	permissionService *services.PermissionService
	logger            zerolog.Logger
	renderer          *TemplateRenderer
}

// NewShareHandler creates a new share handler
func NewShareHandler(
	db *gorm.DB,
	shareService *services.ShareService,
	permissionService *services.PermissionService,
	logger zerolog.Logger,
	renderer *TemplateRenderer,
) *ShareHandler {
	return &ShareHandler{
		db:                db,
		shareService:      shareService,
		permissionService: permissionService,
		logger:            logger,
		renderer:          renderer,
	}
}

// CreateShare creates a new share link
func (h *ShareHandler) CreateShare(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var req struct {
		ResourceID     string `json:"resource_id" binding:"required"`
		ResourceType   string `json:"resource_type" binding:"required"`
		PermissionType string `json:"permission_type" binding:"required"`
		Password       string `json:"password"`
		ExpiresAt      *time.Time `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate resource type
	resourceType := models.ResourceType(req.ResourceType)
	if resourceType != models.ResourceTypeFile && resourceType != models.ResourceTypeDirectory {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource type"})
		return
	}

	// Validate permission type
	permType := models.PermissionType(req.PermissionType)
	if permType != models.PermissionRead && permType != models.PermissionReadUpload && permType != models.PermissionUploadOnly {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission type"})
		return
	}

	// Check if user can create share
	canCreate, err := h.permissionService.CanCreateShare(userID, req.ResourceID, req.ResourceType)
	if err != nil || !canCreate {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Create share
	share, err := h.shareService.CreateShare(userID, req.ResourceID, resourceType, permType, req.Password, req.ExpiresAt)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create share")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create share"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"share": share})
}

// ListShares lists all shares created by the user
func (h *ShareHandler) ListShares(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	shares, err := h.shareService.ListUserShares(userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list shares")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list shares"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"shares": shares})
}

// GetShare retrieves a share by ID
func (h *ShareHandler) GetShare(c *gin.Context) {
	shareID := c.Param("id")
	userID, _ := auth.GetUserID(c)

	share, err := h.shareService.GetShare(shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share not found"})
		return
	}

	// Check if user owns the share
	if share.User != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"share": share})
}

// RevokeShare deletes a share
func (h *ShareHandler) RevokeShare(c *gin.Context) {
	shareID := c.Param("id")
	userID, _ := auth.GetUserID(c)

	// Check permission
	canRevoke, err := h.permissionService.CanRevokeShare(userID, shareID)
	if err != nil || !canRevoke {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	if err := h.shareService.RevokeShare(shareID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to revoke share")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Share revoked successfully"})
}

// AccessShare handles accessing a shared resource
func (h *ShareHandler) AccessShare(c *gin.Context) {
	shareToken := c.Query("token")
	password := c.PostForm("password")

	if shareToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Share token required"})
		return
	}

	// Validate share token
	sharePerms, err := h.permissionService.ValidateShareToken(shareToken, password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if sharePerms.IsExpired {
		c.JSON(http.StatusGone, gin.H{"error": "Share link has expired"})
		return
	}

	// Log access
	h.shareService.LogShareAccess(
		sharePerms.ShareID,
		c.ClientIP(),
		c.Request.UserAgent(),
		"view",
		"",
	)

	// Increment access count
	h.shareService.IncrementAccessCount(sharePerms.ShareID)

	// Return resource based on type
	if sharePerms.ResourceType == string(models.ResourceTypeFile) {
		var file models.File
		if err := h.db.First(&file, "id = ?", sharePerms.ResourceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"type":        "file",
			"resource":    file,
			"permissions": sharePerms,
		})
	} else {
		var dir models.Directory
		if err := h.db.First(&dir, "id = ?", sharePerms.ResourceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Directory not found"})
			return
		}

		// Get files in directory if user can view
		var files []*models.File
		if sharePerms.PermissionType != string(models.PermissionUploadOnly) {
			h.db.Where("parent_directory = ?", sharePerms.ResourceID).Find(&files)
		}

		c.JSON(http.StatusOK, gin.H{
			"type":        "directory",
			"resource":    dir,
			"files":       files,
			"permissions": sharePerms,
		})
	}
}
