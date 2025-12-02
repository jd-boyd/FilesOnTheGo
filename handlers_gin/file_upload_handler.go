package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// FileUploadHandler handles file upload requests
type FileUploadHandler struct {
	db                *gorm.DB
	s3Service         services.S3Service
	permissionService *services.PermissionService
	userService       *services.UserService
	logger            zerolog.Logger
	config            *config.Config
}

// NewFileUploadHandler creates a new file upload handler
func NewFileUploadHandler(
	db *gorm.DB,
	s3Service services.S3Service,
	permissionService *services.PermissionService,
	userService       *services.UserService,
	logger zerolog.Logger,
	cfg *config.Config,
) *FileUploadHandler {
	return &FileUploadHandler{
		db:                db,
		s3Service:         s3Service,
		permissionService: permissionService,
		userService:       userService,
		logger:            logger,
		config:            cfg,
	}
}

// HandleUpload handles file upload
func (h *FileUploadHandler) HandleUpload(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	shareToken := c.Query("share_token")
	directoryID := c.PostForm("directory_id")

	// Check upload permission
	canUpload, err := h.permissionService.CanUploadFile(userID, directoryID, shareToken)
	if err != nil || !canUpload {
		h.logger.Warn().
			Str("user_id", userID).
			Str("directory_id", directoryID).
			Msg("Upload permission denied")
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Get file from request
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Validate file size
	if fileHeader.Size > h.config.MaxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", h.config.MaxUploadSize),
		})
		return
	}

	// Check user quota
	if userID != "" {
		canUpload, err := h.permissionService.CanUploadSize(userID, fileHeader.Size)
		if err != nil || !canUpload {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient storage quota"})
			return
		}
	}

	// Sanitize filename
	filename, err := models.SanitizeFilename(fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}

	// Open uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to open uploaded file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}
	defer file.Close()

	// Generate S3 key
	s3Key := h.generateS3Key(userID, filename)

	// Upload to S3
	err = h.s3Service.UploadFile(s3Key, file, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		h.logger.Error().Err(err).Str("s3_key", s3Key).Msg("Failed to upload file to S3")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Get directory path
	directoryPath := "/"
	if directoryID != "" {
		var dir models.Directory
		if err := h.db.First(&dir, "id = ?", directoryID).Error; err == nil {
			directoryPath = dir.GetFullPath()
		}
	}

	// Create file record
	fileRecord := &models.File{
		Name:            filename,
		Path:            directoryPath,
		User:            userID,
		ParentDirectory: directoryID,
		Size:            fileHeader.Size,
		MimeType:        fileHeader.Header.Get("Content-Type"),
		S3Key:           s3Key,
		S3Bucket:        h.config.S3Bucket,
	}

	if err := h.db.Create(fileRecord).Error; err != nil {
		// Rollback S3 upload
		h.s3Service.DeleteFile(s3Key)
		h.logger.Error().Err(err).Msg("Failed to create file record")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
		return
	}

	// Update user storage
	if userID != "" {
		h.userService.UpdateStorageUsed(userID, fileHeader.Size)
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("file_id", fileRecord.ID).
		Str("filename", filename).
		Int64("size", fileHeader.Size).
		Msg("File uploaded successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"file":    fileRecord,
	})
}

func (h *FileUploadHandler) generateS3Key(userID, filename string) string {
	// Generate unique S3 key: users/{userID}/{timestamp}_{filename}
	timestamp := models.GenerateID()
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Sanitize for S3
	safeFilename := strings.ReplaceAll(nameWithoutExt, " ", "_")
	return fmt.Sprintf("users/%s/%s_%s%s", userID, timestamp, safeFilename, ext)
}
