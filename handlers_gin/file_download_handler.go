package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// FileDownloadHandler handles file download requests
type FileDownloadHandler struct {
	db                *gorm.DB
	s3Service         services.S3Service
	permissionService *services.PermissionService
	logger            zerolog.Logger
}

// NewFileDownloadHandler creates a new file download handler
func NewFileDownloadHandler(
	db *gorm.DB,
	s3Service services.S3Service,
	permissionService *services.PermissionService,
	logger zerolog.Logger,
) *FileDownloadHandler {
	return &FileDownloadHandler{
		db:                db,
		s3Service:         s3Service,
		permissionService: permissionService,
		logger:            logger,
	}
}

// HandleDownload handles file download
func (h *FileDownloadHandler) HandleDownload(c *gin.Context) {
	fileID := c.Param("id")
	userID, _ := auth.GetUserID(c)
	shareToken := c.Query("share_token")

	// Check read permission
	canRead, err := h.permissionService.CanReadFile(userID, fileID, shareToken)
	if err != nil || !canRead {
		h.logger.Warn().
			Str("user_id", userID).
			Str("file_id", fileID).
			Msg("Download permission denied")
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Get file record
	var file models.File
	if err := h.db.First(&file, "id = ?", fileID).Error; err != nil {
		h.logger.Error().Err(err).Str("file_id", fileID).Msg("File not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Download from S3
	reader, err := h.s3Service.DownloadFile(file.S3Key)
	if err != nil {
		h.logger.Error().Err(err).Str("s3_key", file.S3Key).Msg("Failed to download file from S3")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download file"})
		return
	}
	defer reader.Close()

	// Set headers
	c.Header("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", string(rune(file.Size)))

	// Stream file to client
	if _, err := io.Copy(c.Writer, reader); err != nil {
		h.logger.Error().Err(err).Msg("Failed to stream file to client")
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("file_id", fileID).
		Str("filename", file.Name).
		Msg("File downloaded successfully")
}

// HandleDelete handles file deletion
func (h *FileDownloadHandler) HandleDelete(c *gin.Context) {
	fileID := c.Param("id")
	userID, _ := auth.GetUserID(c)

	// Check delete permission
	canDelete, err := h.permissionService.CanDeleteFile(userID, fileID)
	if err != nil || !canDelete {
		h.logger.Warn().
			Str("user_id", userID).
			Str("file_id", fileID).
			Msg("Delete permission denied")
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Get file record
	var file models.File
	if err := h.db.First(&file, "id = ?", fileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Delete from S3
	if err := h.s3Service.DeleteFile(file.S3Key); err != nil {
		h.logger.Error().Err(err).Str("s3_key", file.S3Key).Msg("Failed to delete file from S3")
		// Continue with database deletion even if S3 delete fails
	}

	// Delete from database
	if err := h.db.Delete(&file).Error; err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete file record")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("file_id", fileID).
		Str("filename", file.Name).
		Msg("File deleted successfully")

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
