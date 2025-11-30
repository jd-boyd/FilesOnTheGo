package handlers

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// FileUploadHandler handles file upload requests with streaming support
type FileUploadHandler struct {
	app               *pocketbase.PocketBase
	s3Service         services.S3Service
	permissionService services.PermissionService
	config            *config.Config
}

// NewFileUploadHandler creates a new file upload handler
func NewFileUploadHandler(
	app *pocketbase.PocketBase,
	s3Service services.S3Service,
	permissionService services.PermissionService,
	cfg *config.Config,
) *FileUploadHandler {
	return &FileUploadHandler{
		app:               app,
		s3Service:         s3Service,
		permissionService: permissionService,
		config:            cfg,
	}
}

// UploadResponse represents the response for a successful file upload
type UploadResponse struct {
	Success bool       `json:"success"`
	File    *FileInfo  `json:"file,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

// FileInfo represents information about an uploaded file
type FileInfo struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	MimeType string    `json:"mime_type"`
	Path     string    `json:"path"`
	S3Key    string    `json:"s3_key,omitempty"`
	Created  time.Time `json:"created"`
}

// ErrorInfo represents detailed error information
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HandleUpload processes file upload requests
// POST /api/files/upload
func (h *FileUploadHandler) HandleUpload(c *core.RequestEvent) error {
	startTime := time.Now()
	isHTMX := IsHTMXRequest(c)

	// Get user ID from auth context
	userID := h.getUserID(c)
	shareToken := c.Request.Header.Get("X-Share-Token")
	if shareToken == "" {
		shareToken = c.Request.FormValue("share_token")
	}

	// Require either user authentication or share token
	if userID == "" && shareToken == "" {
		log.Warn().Msg("Upload attempt without authentication or share token")
		return h.errorResponse(c, isHTMX, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
	}

	// Parse multipart form with max memory
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		log.Error().Err(err).Msg("Failed to parse multipart form")
		return h.errorResponse(c, isHTMX, http.StatusBadRequest, "INVALID_FORM", "Invalid multipart form data", nil)
	}

	// Get directory ID from form
	directoryID := c.Request.FormValue("directory_id")
	if directoryID == "" {
		directoryID = c.Request.FormValue("path")
	}
	// Empty directory ID means root
	if directoryID == "root" {
		directoryID = ""
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Warn().Err(err).Msg("No file in upload request")
		return h.errorResponse(c, isHTMX, http.StatusBadRequest, "NO_FILE", "No file provided", nil)
	}
	defer file.Close()

	// Get file size
	fileSize := header.Size

	// Validate file size
	if err := h.validateFileSize(fileSize); err != nil {
		log.Warn().
			Int64("size", fileSize).
			Err(err).
			Msg("File size validation failed")
		return h.errorResponse(c, isHTMX, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error(), map[string]interface{}{
			"size":     fileSize,
			"max_size": h.config.MaxUploadSize,
		})
	}

	// Sanitize filename
	sanitizedFilename, err := models.SanitizeFilename(header.Filename)
	if err != nil {
		log.Warn().
			Str("filename", header.Filename).
			Err(err).
			Msg("Filename sanitization failed")
		return h.errorResponse(c, isHTMX, http.StatusBadRequest, "INVALID_FILENAME", "Invalid filename", map[string]interface{}{
			"original_filename": header.Filename,
			"error":             err.Error(),
		})
	}

	// Detect MIME type
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Check permissions
	canUpload, err := h.permissionService.CanUploadFile(userID, directoryID, shareToken)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("directory_id", directoryID).
			Msg("Permission check failed")
		return h.errorResponse(c, isHTMX, http.StatusInternalServerError, "PERMISSION_CHECK_FAILED", "Permission check failed", nil)
	}
	if !canUpload {
		log.Warn().
			Str("user_id", userID).
			Str("directory_id", directoryID).
			Str("share_token", shareToken).
			Msg("Upload permission denied")
		return h.errorResponse(c, isHTMX, http.StatusForbidden, "PERMISSION_DENIED", "You do not have permission to upload to this location", nil)
	}

	// Check quota (only for authenticated users, not share uploads)
	if userID != "" {
		canUploadSize, err := h.permissionService.CanUploadSize(userID, fileSize)
		if err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Int64("file_size", fileSize).
				Msg("Quota check failed")
			return h.errorResponse(c, isHTMX, http.StatusInternalServerError, "QUOTA_CHECK_FAILED", "Quota check failed", nil)
		}
		if !canUploadSize {
			quota, _ := h.permissionService.GetUserQuota(userID)
			log.Warn().
				Str("user_id", userID).
				Int64("file_size", fileSize).
				Int64("available", quota.Available).
				Msg("Quota exceeded")
			return h.errorResponse(c, isHTMX, http.StatusForbidden, "QUOTA_EXCEEDED", "Upload would exceed storage quota", map[string]interface{}{
				"quota":     quota.TotalQuota,
				"used":      quota.UsedQuota,
				"available": quota.Available,
				"requested": fileSize,
			})
		}
	}

	// If upload via share token, get owner from share
	ownerID := userID
	if userID == "" && shareToken != "" {
		sharePerms, err := h.permissionService.ValidateShareToken(shareToken, "")
		if err == nil && !sharePerms.IsExpired {
			// Get the directory to find owner
			if directoryID != "" {
				dir, err := h.app.FindRecordById("directories", directoryID)
				if err == nil {
					ownerID = dir.GetString("user")
				}
			}
		}
		if ownerID == "" {
			return h.errorResponse(c, isHTMX, http.StatusBadRequest, "INVALID_SHARE", "Unable to determine file owner", nil)
		}
	}

	// Generate unique file ID
	fileID := uuid.New().String()

	// Generate S3 key
	s3Key := services.GenerateS3Key(ownerID, fileID, sanitizedFilename)

	// Upload to S3 using streaming
	log.Debug().
		Str("file_id", fileID).
		Str("filename", sanitizedFilename).
		Str("s3_key", s3Key).
		Int64("size", fileSize).
		Msg("Starting S3 upload")

	// Use streaming upload for efficiency
	if err := h.s3Service.UploadFile(s3Key, file, fileSize, mimeType); err != nil {
		log.Error().
			Err(err).
			Str("file_id", fileID).
			Str("s3_key", s3Key).
			Msg("S3 upload failed")
		return h.errorResponse(c, isHTMX, http.StatusInternalServerError, "UPLOAD_FAILED", "Failed to upload file to storage", nil)
	}

	log.Info().
		Str("file_id", fileID).
		Str("s3_key", s3Key).
		Int64("size", fileSize).
		Msg("S3 upload successful")

	// Determine parent directory path
	filePath := "/"
	if directoryID != "" {
		dir, err := h.app.FindRecordById("directories", directoryID)
		if err == nil {
			filePath = dir.GetString("path")
			if filePath == "" {
				filePath = "/"
			}
		}
	}

	// Create database record
	collection, err := h.app.FindCollectionByNameOrId("files")
	if err != nil {
		// Cleanup: delete S3 file
		_ = h.s3Service.DeleteFile(s3Key)
		log.Error().Err(err).Msg("Failed to find files collection")
		return h.errorResponse(c, isHTMX, http.StatusInternalServerError, "DATABASE_ERROR", "Database error", nil)
	}

	record := core.NewRecord(collection)
	record.Set("id", fileID)
	record.Set("name", sanitizedFilename)
	record.Set("path", filePath)
	record.Set("user", ownerID)
	record.Set("size", fileSize)
	record.Set("mime_type", mimeType)
	record.Set("s3_key", s3Key)
	record.Set("s3_bucket", h.config.S3Bucket)

	if directoryID != "" {
		record.Set("parent_directory", directoryID)
	}

	// Save record
	if err := h.app.Save(record); err != nil {
		// Cleanup: delete S3 file
		_ = h.s3Service.DeleteFile(s3Key)
		log.Error().
			Err(err).
			Str("file_id", fileID).
			Msg("Failed to save file record to database")
		return h.errorResponse(c, isHTMX, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save file metadata", nil)
	}

	duration := time.Since(startTime)
	log.Info().
		Str("user_id", ownerID).
		Str("file_id", fileID).
		Str("filename", sanitizedFilename).
		Int64("size", fileSize).
		Dur("duration", duration).
		Msg("File uploaded successfully")

	// Prepare response
	fileInfo := &FileInfo{
		ID:       fileID,
		Name:     sanitizedFilename,
		Size:     fileSize,
		MimeType: mimeType,
		Path:     filePath,
		S3Key:    s3Key,
		Created:  record.GetDateTime("created").Time(),
	}

	return h.successResponse(c, isHTMX, fileInfo)
}

// validateFileSize validates that the file size is within limits
func (h *FileUploadHandler) validateFileSize(size int64) error {
	return models.ValidateFileSize(size, h.config.MaxUploadSize)
}

// getUserID extracts the user ID from the request context
func (h *FileUploadHandler) getUserID(c *core.RequestEvent) string {
	auth := c.Get("authRecord")
	if auth == nil {
		return ""
	}
	if record, ok := auth.(*core.Record); ok {
		return record.Id
	}
	return ""
}

// successResponse sends a success response
func (h *FileUploadHandler) successResponse(c *core.RequestEvent, isHTMX bool, file *FileInfo) error {
	if isHTMX {
		// Return HTML fragment for file list item
		html := fmt.Sprintf(`
			<div class="file-item bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow" data-file-id="%s">
				<div class="flex items-center justify-between">
					<div class="flex items-center space-x-3">
						<div class="flex-shrink-0">
							<svg class="h-8 w-8 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
								<path d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z"/>
							</svg>
						</div>
						<div>
							<p class="text-sm font-medium text-gray-900">%s</p>
							<p class="text-xs text-gray-500">%s • %s</p>
						</div>
					</div>
					<div class="flex items-center space-x-2">
						<button class="text-gray-400 hover:text-gray-600">
							<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
							</svg>
						</button>
						<button class="text-gray-400 hover:text-red-600">
							<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
							</svg>
						</button>
					</div>
				</div>
			</div>
		`, file.ID, file.Name, formatFileSize(file.Size), file.Created.Format("Jan 2, 2006"))

		c.Response.Header().Set("HX-Trigger", "fileUploaded")
		return c.HTML(http.StatusOK, html)
	}

	// Return JSON response
	return c.JSON(http.StatusOK, UploadResponse{
		Success: true,
		File:    file,
	})
}

// errorResponse sends an error response
func (h *FileUploadHandler) errorResponse(c *core.RequestEvent, isHTMX bool, statusCode int, code, message string, details map[string]interface{}) error {
	if isHTMX {
		// Return HTML error message
		html := fmt.Sprintf(`
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm font-medium">%s</p>
				<p class="text-xs mt-1">%s</p>
			</div>
		`, message, code)

		return c.HTML(statusCode, html)
	}

	// Return JSON error
	return c.JSON(statusCode, UploadResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ValidateUploadRequest validates the upload request parameters
func ValidateUploadRequest(file multipart.File, header *multipart.FileHeader, maxSize int64) error {
	if file == nil || header == nil {
		return errors.New("no file provided")
	}

	// Validate file size
	if err := models.ValidateFileSize(header.Size, maxSize); err != nil {
		return err
	}

	// Validate filename
	if _, err := models.SanitizeFilename(header.Filename); err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}

	return nil
}

// GetFileReader returns a reader for the uploaded file
func GetFileReader(file multipart.File) (io.Reader, error) {
	// Reset file pointer to beginning
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek to start of file: %w", err)
		}
	}
	return file, nil
}
