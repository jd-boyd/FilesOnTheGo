package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// DirectoryHandler handles directory operations
type DirectoryHandler struct {
	db                *gorm.DB
	permissionService *services.PermissionService
	logger            zerolog.Logger
	renderer          *TemplateRenderer
}

// NewDirectoryHandler creates a new directory handler
func NewDirectoryHandler(
	db *gorm.DB,
	permissionService *services.PermissionService,
	logger zerolog.Logger,
	renderer *TemplateRenderer,
) *DirectoryHandler {
	return &DirectoryHandler{
		db:                db,
		permissionService: permissionService,
		logger:            logger,
		renderer:          renderer,
	}
}

// ListDirectory lists files and directories in a directory
func (h *DirectoryHandler) ListDirectory(c *gin.Context) {
	directoryID := c.Query("directory_id")
	userID, _ := auth.GetUserID(c)
	shareToken := c.Query("share_token")

	// Check read permission
	canRead, err := h.permissionService.CanReadDirectory(userID, directoryID, shareToken)
	if err != nil || !canRead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Get subdirectories
	var directories []*models.Directory
	query := h.db.Where("user = ?", userID)
	if directoryID != "" {
		query = query.Where("parent_directory = ?", directoryID)
	} else {
		query = query.Where("parent_directory IS NULL OR parent_directory = ''")
	}
	query.Order("name ASC").Find(&directories)

	// Get files
	var files []*models.File
	fileQuery := h.db.Where("user = ?", userID)
	if directoryID != "" {
		fileQuery = fileQuery.Where("parent_directory = ?", directoryID)
	} else {
		fileQuery = fileQuery.Where("parent_directory IS NULL OR parent_directory = ''")
	}
	fileQuery.Order("name ASC").Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"directories": directories,
		"files":       files,
	})
}

// CreateDirectory creates a new directory
func (h *DirectoryHandler) CreateDirectory(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var req struct {
		Name      string `json:"name" binding:"required"`
		ParentID  string `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check permission
	canCreate, err := h.permissionService.CanCreateDirectory(userID, req.ParentID)
	if err != nil || !canCreate {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Sanitize name
	name, err := models.SanitizeFilename(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid directory name"})
		return
	}

	// Get parent path
	path := "/"
	if req.ParentID != "" {
		var parent models.Directory
		if err := h.db.First(&parent, "id = ?", req.ParentID).Error; err == nil {
			path = parent.GetFullPath()
		}
	}

	// Create directory
	dir := &models.Directory{
		Name:            name,
		Path:            path,
		User:            userID,
		ParentDirectory: req.ParentID,
	}

	if err := h.db.Create(dir).Error; err != nil {
		h.logger.Error().Err(err).Msg("Failed to create directory")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("directory_id", dir.ID).
		Str("name", name).
		Msg("Directory created successfully")

	c.JSON(http.StatusCreated, gin.H{"directory": dir})
}

// DeleteDirectory deletes a directory
func (h *DirectoryHandler) DeleteDirectory(c *gin.Context) {
	directoryID := c.Param("id")
	userID, _ := auth.GetUserID(c)

	// Check permission
	canDelete, err := h.permissionService.CanDeleteDirectory(userID, directoryID)
	if err != nil || !canDelete {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Check if directory is empty
	var fileCount, dirCount int64
	h.db.Model(&models.File{}).Where("parent_directory = ?", directoryID).Count(&fileCount)
	h.db.Model(&models.Directory{}).Where("parent_directory = ?", directoryID).Count(&dirCount)

	if fileCount > 0 || dirCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Directory is not empty"})
		return
	}

	// Delete directory
	if err := h.db.Delete(&models.Directory{}, "id = ?", directoryID).Error; err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete directory")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete directory"})
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("directory_id", directoryID).
		Msg("Directory deleted successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Directory deleted successfully"})
}
