package handlers

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// FileDownloadHandler handles file download operations with pre-signed URLs and streaming
type FileDownloadHandler struct {
	app               *pocketbase.PocketBase
	s3Service         services.S3Service
	permissionService services.PermissionService
}

// NewFileDownloadHandler creates a new file download handler instance
func NewFileDownloadHandler(
	app *pocketbase.PocketBase,
	s3Service services.S3Service,
	permissionService services.PermissionService,
) *FileDownloadHandler {
	return &FileDownloadHandler{
		app:               app,
		s3Service:         s3Service,
		permissionService: permissionService,
	}
}

// HandleDownload processes file download requests
// GET /api/files/{file_id}/download?share_token=xxx&inline=false&stream=false
func (h *FileDownloadHandler) HandleDownload(c *core.RequestEvent) error {
	fileID := c.Request.PathValue("id")
	if fileID == "" {
		log.Warn().Msg("File ID is required for download")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "File ID is required",
		})
	}

	// Get query parameters
	shareToken := c.Request.URL.Query().Get("share_token")
	inline := c.Request.URL.Query().Get("inline") == "true"
	stream := c.Request.URL.Query().Get("stream") == "true"

	// Get authenticated user (may be empty if using share token)
	var userID string
	authRecord := c.Auth
	if authRecord != nil {
		userID = authRecord.Id
	}

	// Fetch file record
	file, err := h.app.FindRecordById("files", fileID)
	if err != nil {
		log.Debug().
			Err(err).
			Str("file_id", fileID).
			Msg("File not found")
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "File not found",
		})
	}

	// Validate permission to read file
	canRead, err := h.permissionService.CanReadFile(userID, fileID, shareToken)
	if err != nil || !canRead {
		log.Warn().
			Str("user_id", userID).
			Str("file_id", fileID).
			Str("share_token", shareToken).
			Msg("Permission denied for file download")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// If using share token, validate it's not upload-only
	if shareToken != "" {
		sharePerms, err := h.permissionService.ValidateShareToken(shareToken, "")
		if err != nil {
			log.Warn().
				Str("share_token", shareToken).
				Msg("Invalid share token")
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Invalid share token",
			})
		}

		// Check if share is expired
		if sharePerms.IsExpired {
			log.Warn().
				Str("share_token", shareToken).
				Msg("Share token expired")
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Share link expired",
			})
		}

		// Prevent download with upload-only permission
		if sharePerms.PermissionType == "upload_only" {
			log.Warn().
				Str("share_token", shareToken).
				Str("file_id", fileID).
				Msg("Upload-only share cannot download")
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "This share link does not allow downloads",
			})
		}
	}

	// Log the download access
	h.logDownloadAccess(userID, fileID, shareToken, c.Request)

	// Get S3 key
	s3Key := file.GetString("s3_key")
	fileName := file.GetString("name")
	mimeType := file.GetString("mime_type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Choose download method: stream or pre-signed URL redirect
	if stream {
		return h.streamFile(c, s3Key, fileName, mimeType, inline)
	}

	// Generate pre-signed URL (default: 15 minutes)
	presignedURL, err := h.s3Service.GetPresignedURL(s3Key, 15)
	if err != nil {
		log.Error().
			Err(err).
			Str("s3_key", s3Key).
			Msg("Failed to generate pre-signed URL")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate download URL",
		})
	}

	log.Info().
		Str("user_id", userID).
		Str("file_id", fileID).
		Str("file_name", fileName).
		Bool("share_access", shareToken != "").
		Msg("File download - redirecting to pre-signed URL")

	// Redirect to pre-signed URL
	return c.Redirect(http.StatusFound, presignedURL)
}

// streamFile streams a file directly through the backend
func (h *FileDownloadHandler) streamFile(c *core.RequestEvent, s3Key, fileName, mimeType string, inline bool) error {
	// Download file from S3
	reader, err := h.s3Service.DownloadFile(s3Key)
	if err != nil {
		log.Error().
			Err(err).
			Str("s3_key", s3Key).
			Msg("Failed to download file from S3")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to download file",
		})
	}
	defer reader.Close()

	// Get file metadata for Content-Length
	metadata, err := h.s3Service.GetFileMetadata(s3Key)
	if err != nil {
		log.Warn().
			Err(err).
			Str("s3_key", s3Key).
			Msg("Failed to get file metadata, streaming without Content-Length")
	}

	// Set response headers
	h.setDownloadHeaders(c, fileName, mimeType, inline, metadata)

	// Stream file to client
	_, err = io.Copy(c.Response, reader)
	if err != nil {
		log.Error().
			Err(err).
			Str("s3_key", s3Key).
			Msg("Error streaming file to client")
		return err
	}

	log.Info().
		Str("s3_key", s3Key).
		Str("file_name", fileName).
		Msg("File streamed successfully")

	return nil
}

// HandleBatchDownload creates a ZIP file of multiple files
// POST /api/files/download/batch
func (h *FileDownloadHandler) HandleBatchDownload(c *core.RequestEvent) error {
	// Get authenticated user
	authRecord := c.Auth
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}
	userID := authRecord.Id

	// Parse request body
	var request struct {
		FileIDs     []string `json:"file_ids"`
		DirectoryID string   `json:"directory_id"`
	}

	if err := c.BindBody(&request); err != nil {
		log.Warn().Err(err).Msg("Invalid batch download request")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(request.FileIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No files specified for download",
		})
	}

	// Validate access to all files
	var files []*core.Record
	for _, fileID := range request.FileIDs {
		file, err := h.app.FindRecordById("files", fileID)
		if err != nil {
			log.Warn().
				Str("file_id", fileID).
				Msg("File not found in batch download")
			continue
		}

		// Check permission
		canRead, err := h.permissionService.CanReadFile(userID, fileID, "")
		if err != nil || !canRead {
			log.Warn().
				Str("user_id", userID).
				Str("file_id", fileID).
				Msg("Permission denied for file in batch download")
			continue
		}

		files = append(files, file)
	}

	if len(files) == 0 {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "No accessible files found",
		})
	}

	// Create ZIP and stream to client
	return h.createAndStreamZip(c, files, "files.zip")
}

// HandleDirectoryDownload downloads all files in a directory as a ZIP
// GET /api/directories/{directory_id}/download?share_token=xxx
func (h *FileDownloadHandler) HandleDirectoryDownload(c *core.RequestEvent) error {
	directoryID := c.Request.PathValue("id")
	if directoryID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Directory ID is required",
		})
	}

	shareToken := c.Request.URL.Query().Get("share_token")

	// Get authenticated user (may be empty if using share token)
	var userID string
	authRecord := c.Auth
	if authRecord != nil {
		userID = authRecord.Id
	}

	// Validate directory access
	canRead, err := h.permissionService.CanReadDirectory(userID, directoryID, shareToken)
	if err != nil || !canRead {
		log.Warn().
			Str("user_id", userID).
			Str("directory_id", directoryID).
			Msg("Permission denied for directory download")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// Get directory record for name
	directory, err := h.app.FindRecordById("directories", directoryID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Directory not found",
		})
	}

	dirName := directory.GetString("name")

	// Recursively get all files in directory
	files, err := h.getFilesInDirectory(directoryID, userID, shareToken)
	if err != nil {
		log.Error().
			Err(err).
			Str("directory_id", directoryID).
			Msg("Failed to get files in directory")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve directory files",
		})
	}

	if len(files) == 0 {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Directory is empty",
		})
	}

	zipFileName := fmt.Sprintf("%s.zip", dirName)
	return h.createAndStreamZip(c, files, zipFileName)
}

// getFilesInDirectory recursively gets all files in a directory
func (h *FileDownloadHandler) getFilesInDirectory(directoryID, userID, shareToken string) ([]*core.Record, error) {
	var allFiles []*core.Record

	// Get files directly in this directory
	files, err := h.app.FindRecordsByFilter(
		"files",
		"parent_directory = {:dir}",
		"",
		0,
		0,
		map[string]any{"dir": directoryID},
	)
	if err == nil {
		// Validate access to each file
		for _, file := range files {
			canRead, err := h.permissionService.CanReadFile(userID, file.Id, shareToken)
			if err == nil && canRead {
				allFiles = append(allFiles, file)
			}
		}
	}

	// Get subdirectories and recurse
	subdirs, err := h.app.FindRecordsByFilter(
		"directories",
		"parent_directory = {:dir}",
		"",
		0,
		0,
		map[string]any{"dir": directoryID},
	)
	if err == nil {
		for _, subdir := range subdirs {
			// Check if user can access subdirectory
			canRead, err := h.permissionService.CanReadDirectory(userID, subdir.Id, shareToken)
			if err == nil && canRead {
				subFiles, err := h.getFilesInDirectory(subdir.Id, userID, shareToken)
				if err == nil {
					allFiles = append(allFiles, subFiles...)
				}
			}
		}
	}

	return allFiles, nil
}

// createAndStreamZip creates a ZIP file and streams it to the client
func (h *FileDownloadHandler) createAndStreamZip(c *core.RequestEvent, files []*core.Record, zipFileName string) error {
	// Set headers for ZIP download
	c.Response.Header().Set("Content-Type", "application/zip")
	c.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))
	c.Response.Header().Set("X-Content-Type-Options", "nosniff")
	c.Response.WriteHeader(http.StatusOK)

	// Create ZIP writer
	zipWriter := zip.NewWriter(c.Response)
	defer zipWriter.Close()

	// Add each file to ZIP
	for _, file := range files {
		s3Key := file.GetString("s3_key")
		fileName := file.GetString("name")
		filePath := file.GetString("path")

		// Use path if available, otherwise just filename
		zipEntryName := fileName
		if filePath != "" {
			zipEntryName = filePath
		}

		// Download file from S3
		reader, err := h.s3Service.DownloadFile(s3Key)
		if err != nil {
			log.Error().
				Err(err).
				Str("s3_key", s3Key).
				Str("file_name", fileName).
				Msg("Failed to download file for ZIP")
			continue
		}

		// Create ZIP entry
		writer, err := zipWriter.Create(zipEntryName)
		if err != nil {
			log.Error().
				Err(err).
				Str("file_name", fileName).
				Msg("Failed to create ZIP entry")
			reader.Close()
			continue
		}

		// Copy file to ZIP
		_, err = io.Copy(writer, reader)
		reader.Close()

		if err != nil {
			log.Error().
				Err(err).
				Str("file_name", fileName).
				Msg("Failed to copy file to ZIP")
			continue
		}

		log.Debug().
			Str("file_name", fileName).
			Msg("Added file to ZIP")
	}

	log.Info().
		Int("file_count", len(files)).
		Str("zip_name", zipFileName).
		Msg("ZIP download completed")

	return nil
}

// setDownloadHeaders sets appropriate HTTP headers for file downloads
func (h *FileDownloadHandler) setDownloadHeaders(c *core.RequestEvent, fileName, mimeType string, inline bool, metadata *services.FileMetadata) {
	// Content-Type
	c.Response.Header().Set("Content-Type", mimeType)

	// Content-Disposition
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	// Sanitize filename to prevent header injection
	safeFileName := strings.ReplaceAll(fileName, "\"", "\\\"")
	c.Response.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, safeFileName))

	// Content-Length (if available)
	if metadata != nil {
		c.Response.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size))
	}

	// ETag (if available)
	if metadata != nil && metadata.ETag != "" {
		c.Response.Header().Set("ETag", metadata.ETag)
	}

	// Security headers
	c.Response.Header().Set("X-Content-Type-Options", "nosniff")

	// Cache control (private, must revalidate)
	c.Response.Header().Set("Cache-Control", "private, must-revalidate")
}

// logDownloadAccess logs file download access for audit trails
func (h *FileDownloadHandler) logDownloadAccess(userID, fileID, shareToken string, req *http.Request) {
	// Log to application logs
	log.Info().
		Str("user_id", userID).
		Str("file_id", fileID).
		Str("share_token", shareToken).
		Str("ip_address", getClientIP(req)).
		Str("user_agent", req.UserAgent()).
		Msg("File download access")

	// If share token is used, increment access count and create access log
	if shareToken != "" {
		h.logShareAccess(shareToken, fileID, "download", req)
	}
}

// logShareAccess creates a share access log entry
func (h *FileDownloadHandler) logShareAccess(shareToken, fileID, action string, req *http.Request) {
	// Find share by token
	share, err := h.app.FindFirstRecordByData("shares", "share_token", shareToken)
	if err != nil {
		log.Error().
			Err(err).
			Str("share_token", shareToken).
			Msg("Failed to find share for access logging")
		return
	}

	// Increment access count
	accessCount := share.GetInt("access_count")
	share.Set("access_count", accessCount+1)
	if err := h.app.Save(share); err != nil {
		log.Error().
			Err(err).
			Str("share_id", share.Id).
			Msg("Failed to update share access count")
	}

	// Create access log entry
	logsCollection, err := h.app.FindCollectionByNameOrId("share_access_logs")
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to find share_access_logs collection")
		return
	}

	logRecord := core.NewRecord(logsCollection)
	logRecord.Set("share", share.Id)
	logRecord.Set("ip_address", getClientIP(req))
	logRecord.Set("user_agent", req.UserAgent())
	logRecord.Set("action", action)

	// Get file name
	if fileID != "" {
		if file, err := h.app.FindRecordById("files", fileID); err == nil {
			logRecord.Set("file_name", file.GetString("name"))
		}
	}

	logRecord.Set("accessed_at", time.Now())

	if err := h.app.Save(logRecord); err != nil {
		log.Error().
			Err(err).
			Str("share_id", share.Id).
			Msg("Failed to create share access log")
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(req *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := req.RemoteAddr
	// Remove port if present
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}

	return ip
}

// ValidateS3Key validates that an S3 key is safe and properly formatted
func ValidateS3Key(key string) error {
	if key == "" {
		return errors.New("S3 key cannot be empty")
	}

	if len(key) > 1024 {
		return errors.New("S3 key too long")
	}

	// Check for path traversal attempts
	if strings.Contains(key, "..") {
		return errors.New("invalid S3 key: path traversal detected")
	}

	// Check for null bytes
	if strings.Contains(key, "\x00") {
		return errors.New("invalid S3 key: null byte detected")
	}

	return nil
}

// SanitizeFileName sanitizes a filename for safe use in headers
func SanitizeFileName(filename string) string {
	// Extract base filename
	filename = filepath.Base(filename)

	// Remove or replace dangerous characters
	filename = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 { // Control characters
			return -1
		}
		// Remove characters that could break HTTP headers
		if r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, filename)

	// Limit length
	if len(filename) > 255 {
		filename = filename[:255]
	}

	if filename == "" {
		filename = "download"
	}

	return filename
}
