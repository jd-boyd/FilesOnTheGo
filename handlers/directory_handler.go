package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
)

// DirectoryHandler handles directory-related requests
type DirectoryHandler struct {
	app               *pocketbase.PocketBase
	permissionService services.PermissionService
	s3Service         services.S3Service
	logger            zerolog.Logger
}

// NewDirectoryHandler creates a new directory handler
func NewDirectoryHandler(
	app *pocketbase.PocketBase,
	permissionService services.PermissionService,
	s3Service services.S3Service,
	logger zerolog.Logger,
) *DirectoryHandler {
	return &DirectoryHandler{
		app:               app,
		permissionService: permissionService,
		s3Service:         s3Service,
		logger:            logger,
	}
}

// DirectoryResponse represents the response for directory operations
type DirectoryResponse struct {
	Directory   *DirectoryInfo       `json:"directory,omitempty"`
	Breadcrumbs []*models.Breadcrumb `json:"breadcrumbs,omitempty"`
	Items       []ItemInfo           `json:"items,omitempty"`
}

// DirectoryInfo represents directory information
type DirectoryInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Path            string `json:"path"`
	ParentDirectory string `json:"parent_directory,omitempty"`
	Created         string `json:"created"`
	Updated         string `json:"updated"`
}

// ItemInfo represents a file or directory item in a listing
type ItemInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // "file" or "directory"
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// CreateDirectoryRequest represents a request to create a directory
type CreateDirectoryRequest struct {
	Name            string `json:"name"`
	ParentDirectory string `json:"parent_directory,omitempty"`
	Path            string `json:"path,omitempty"`
}

// UpdateDirectoryRequest represents a request to rename a directory
type UpdateDirectoryRequest struct {
	Name string `json:"name"`
}

// MoveDirectoryRequest represents a request to move a directory
type MoveDirectoryRequest struct {
	TargetDirectory string `json:"target_directory_id"`
}

// HandleCreate creates a new directory
// POST /api/directories
func (h *DirectoryHandler) HandleCreate(c *core.RequestEvent) error {
	// Get authenticated user
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		h.logger.Warn().Msg("Unauthorized directory creation attempt")
		return h.handleError(c, errors.New("unauthorized"), http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	user := authRecord.(*core.Record)
	userID := user.Id

	// Parse request body
	var req CreateDirectoryRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid request body")
		return h.handleError(c, err, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Validate directory name
	if req.Name == "" {
		return h.handleError(c, errors.New("name required"), http.StatusBadRequest, "INVALID_NAME", "Directory name is required")
	}

	// Sanitize directory name
	sanitized, err := models.SanitizeFilename(req.Name)
	if err != nil {
		h.logger.Warn().Err(err).Str("name", req.Name).Msg("Invalid directory name")
		return h.handleError(c, err, http.StatusBadRequest, "INVALID_NAME", "Directory name contains invalid characters")
	}
	req.Name = sanitized

	// Determine parent directory ID
	parentDirID := req.ParentDirectory
	if parentDirID == "root" || parentDirID == "" {
		parentDirID = ""
	}

	// Check permissions
	canCreate, err := h.permissionService.CanCreateDirectory(userID, parentDirID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Str("parent_dir_id", parentDirID).Msg("Permission check failed")
		return h.handleError(c, err, http.StatusInternalServerError, "PERMISSION_ERROR", "Failed to check permissions")
	}
	if !canCreate {
		h.logger.Warn().Str("user_id", userID).Str("parent_dir_id", parentDirID).Msg("Permission denied for directory creation")
		return h.handleError(c, errors.New("permission denied"), http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to create a directory here")
	}

	// Check for duplicate name in the same parent
	filter := "user = {:user} && name = {:name}"
	params := map[string]any{
		"user": userID,
		"name": req.Name,
	}

	if parentDirID != "" {
		filter += " && parent_directory = {:parent}"
		params["parent"] = parentDirID
	} else {
		filter += " && parent_directory = ''"
	}

	existingDirs, err := h.app.FindRecordsByFilter("directories", filter, "", 1, 0, params)
	if err == nil && len(existingDirs) > 0 {
		return h.handleError(c, errors.New("duplicate name"), http.StatusConflict, "DUPLICATE_NAME", "A directory with this name already exists in this location")
	}

	// Calculate full path
	fullPath := h.calculateFullPath(parentDirID, req.Name)

	// Create directory record
	collection, err := h.app.FindCollectionByNameOrId("directories")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find directories collection")
		return h.handleError(c, err, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}

	record := core.NewRecord(collection)
	record.Set("name", req.Name)
	record.Set("path", fullPath)
	record.Set("user", userID)
	if parentDirID != "" {
		record.Set("parent_directory", parentDirID)
	}

	if err := h.app.Save(record); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create directory")
		return h.handleError(c, err, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create directory")
	}

	h.logger.Info().
		Str("directory_id", record.Id).
		Str("name", req.Name).
		Str("user_id", userID).
		Msg("Directory created successfully")

	// Prepare response
	response := DirectoryResponse{
		Directory: &DirectoryInfo{
			ID:              record.Id,
			Name:            record.GetString("name"),
			Path:            record.GetString("path"),
			ParentDirectory: record.GetString("parent_directory"),
			Created:         record.GetDateTime("created").String(),
			Updated:         record.GetDateTime("updated").String(),
		},
	}

	if IsHTMXRequest(c) {
		// Return HTML fragment for HTMX request
		return c.HTML(http.StatusOK, fmt.Sprintf(`
			<div class="directory-item" data-id="%s">
				<span class="icon">📁</span>
				<span class="name">%s</span>
			</div>
		`, record.Id, record.GetString("name")))
	}

	return c.JSON(http.StatusOK, response)
}

// HandleList lists directory contents
// GET /api/directories/{id} or GET /api/directories/root
func (h *DirectoryHandler) HandleList(c *core.RequestEvent) error {
	// Get directory ID from URL parameter
	directoryID := c.Request.PathValue("id")
	if directoryID == "root" {
		directoryID = ""
	}

	// Get authenticated user or share token
	authRecord := c.Get("authRecord")
	shareToken := c.Request.URL.Query().Get("share_token")

	var userID string
	if authRecord != nil {
		user := authRecord.(*core.Record)
		userID = user.Id
	}

	// Check permissions
	canRead, err := h.permissionService.CanReadDirectory(userID, directoryID, shareToken)
	if err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Permission check failed")
		return h.handleError(c, err, http.StatusInternalServerError, "PERMISSION_ERROR", "Failed to check permissions")
	}
	if !canRead {
		h.logger.Warn().Str("user_id", userID).Str("directory_id", directoryID).Msg("Permission denied for directory listing")
		return h.handleError(c, errors.New("permission denied"), http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to access this directory")
	}

	var response DirectoryResponse

	// If not root, get directory information and breadcrumbs
	if directoryID != "" {
		dirRecord, err := h.app.FindRecordById("directories", directoryID)
		if err != nil {
			h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Directory not found")
			return h.handleError(c, err, http.StatusNotFound, "NOT_FOUND", "Directory not found")
		}

		response.Directory = &DirectoryInfo{
			ID:              dirRecord.Id,
			Name:            dirRecord.GetString("name"),
			Path:            dirRecord.GetString("path"),
			ParentDirectory: dirRecord.GetString("parent_directory"),
			Created:         dirRecord.GetDateTime("created").String(),
			Updated:         dirRecord.GetDateTime("updated").String(),
		}

		// Get breadcrumbs
		directory := &models.Directory{
			Name:            dirRecord.GetString("name"),
			Path:            dirRecord.GetString("path"),
			User:            dirRecord.GetString("user"),
			ParentDirectory: dirRecord.GetString("parent_directory"),
		}
		directory.Id = dirRecord.Id

		breadcrumbs, err := directory.GetBreadcrumbs(h.app)
		if err != nil {
			h.logger.Warn().Err(err).Str("directory_id", directoryID).Msg("Failed to get breadcrumbs")
		} else {
			response.Breadcrumbs = breadcrumbs
		}
	}

	// List subdirectories
	dirFilter := "user = {:user}"
	dirParams := map[string]any{"user": userID}

	if directoryID != "" {
		dirFilter += " && parent_directory = {:parent}"
		dirParams["parent"] = directoryID
	} else {
		dirFilter += " && parent_directory = ''"
	}

	subdirs, err := h.app.FindRecordsByFilter("directories", dirFilter, "+name", 0, 0, dirParams)
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to list subdirectories")
	}

	// List files
	fileFilter := "user = {:user}"
	fileParams := map[string]any{"user": userID}

	if directoryID != "" {
		fileFilter += " && parent_directory = {:parent}"
		fileParams["parent"] = directoryID
	} else {
		fileFilter += " && parent_directory = ''"
	}

	files, err := h.app.FindRecordsByFilter("files", fileFilter, "+name", 0, 0, fileParams)
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to list files")
	}

	// Combine items
	items := []ItemInfo{}

	// Add directories first
	for _, dir := range subdirs {
		items = append(items, ItemInfo{
			ID:      dir.Id,
			Name:    dir.GetString("name"),
			Type:    "directory",
			Created: dir.GetDateTime("created").String(),
			Updated: dir.GetDateTime("updated").String(),
		})
	}

	// Add files
	for _, file := range files {
		items = append(items, ItemInfo{
			ID:       file.Id,
			Name:     file.GetString("name"),
			Type:     "file",
			Size:     int64(file.GetInt("size")),
			MimeType: file.GetString("mime_type"),
			Created:  file.GetDateTime("created").String(),
			Updated:  file.GetDateTime("updated").String(),
		})
	}

	response.Items = items

	if IsHTMXRequest(c) {
		// Return HTML fragment for HTMX request
		html := "<div class=\"directory-contents\">"
		for _, item := range items {
			icon := "📄"
			if item.Type == "directory" {
				icon = "📁"
			}
			html += fmt.Sprintf(`
				<div class="item" data-id="%s" data-type="%s">
					<span class="icon">%s</span>
					<span class="name">%s</span>
				</div>
			`, item.ID, item.Type, icon, item.Name)
		}
		html += "</div>"
		return c.HTML(http.StatusOK, html)
	}

	return c.JSON(http.StatusOK, response)
}

// HandleDelete deletes a directory
// DELETE /api/directories/{id}
func (h *DirectoryHandler) HandleDelete(c *core.RequestEvent) error {
	// Get directory ID from URL parameter
	directoryID := c.Request.PathValue("id")

	// Get authenticated user (shares cannot delete)
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		h.logger.Warn().Msg("Unauthorized directory deletion attempt")
		return h.handleError(c, errors.New("unauthorized"), http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	user := authRecord.(*core.Record)
	userID := user.Id

	// Check if recursive delete is requested
	recursive := c.Request.URL.Query().Get("recursive") == "true"

	// Check permissions
	canDelete, err := h.permissionService.CanDeleteDirectory(userID, directoryID)
	if err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Permission check failed")
		return h.handleError(c, err, http.StatusInternalServerError, "PERMISSION_ERROR", "Failed to check permissions")
	}
	if !canDelete {
		h.logger.Warn().Str("user_id", userID).Str("directory_id", directoryID).Msg("Permission denied for directory deletion")
		return h.handleError(c, errors.New("permission denied"), http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to delete this directory")
	}

	// Get directory record
	dirRecord, err := h.app.FindRecordById("directories", directoryID)
	if err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Directory not found")
		return h.handleError(c, err, http.StatusNotFound, "NOT_FOUND", "Directory not found")
	}

	// Check if directory is empty (if not recursive)
	if !recursive {
		// Check for subdirectories
		subdirs, err := h.app.FindRecordsByFilter(
			"directories",
			"parent_directory = {:parent}",
			"",
			1,
			0,
			map[string]any{"parent": directoryID},
		)
		if err == nil && len(subdirs) > 0 {
			return h.handleError(c, errors.New("directory not empty"), http.StatusBadRequest, "NOT_EMPTY", "Directory is not empty. Use recursive=true to delete all contents")
		}

		// Check for files
		files, err := h.app.FindRecordsByFilter(
			"files",
			"parent_directory = {:parent}",
			"",
			1,
			0,
			map[string]any{"parent": directoryID},
		)
		if err == nil && len(files) > 0 {
			return h.handleError(c, errors.New("directory not empty"), http.StatusBadRequest, "NOT_EMPTY", "Directory is not empty. Use recursive=true to delete all contents")
		}
	}

	// If recursive, delete all contents
	if recursive {
		if err := h.deleteDirectoryRecursive(directoryID, userID); err != nil {
			h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Failed to delete directory recursively")
			return h.handleError(c, err, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete directory contents")
		}
	}

	// Delete the directory itself
	if err := h.app.Delete(dirRecord); err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Failed to delete directory")
		return h.handleError(c, err, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete directory")
	}

	h.logger.Info().
		Str("directory_id", directoryID).
		Str("user_id", userID).
		Bool("recursive", recursive).
		Msg("Directory deleted successfully")

	if IsHTMXRequest(c) {
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// HandleUpdate renames a directory
// PATCH /api/directories/{id}
func (h *DirectoryHandler) HandleUpdate(c *core.RequestEvent) error {
	// Get directory ID from URL parameter
	directoryID := c.Request.PathValue("id")

	// Get authenticated user
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		h.logger.Warn().Msg("Unauthorized directory update attempt")
		return h.handleError(c, errors.New("unauthorized"), http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	user := authRecord.(*core.Record)
	userID := user.Id

	// Check permissions
	canDelete, err := h.permissionService.CanDeleteDirectory(userID, directoryID)
	if err != nil || !canDelete {
		h.logger.Warn().Str("user_id", userID).Str("directory_id", directoryID).Msg("Permission denied for directory update")
		return h.handleError(c, errors.New("permission denied"), http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to update this directory")
	}

	// Parse request body
	var req UpdateDirectoryRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid request body")
		return h.handleError(c, err, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Validate new name
	if req.Name == "" {
		return h.handleError(c, errors.New("name required"), http.StatusBadRequest, "INVALID_NAME", "Directory name is required")
	}

	// Sanitize new name
	sanitized, err := models.SanitizeFilename(req.Name)
	if err != nil {
		h.logger.Warn().Err(err).Str("name", req.Name).Msg("Invalid directory name")
		return h.handleError(c, err, http.StatusBadRequest, "INVALID_NAME", "Directory name contains invalid characters")
	}
	req.Name = sanitized

	// Get directory record
	dirRecord, err := h.app.FindRecordById("directories", directoryID)
	if err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Directory not found")
		return h.handleError(c, err, http.StatusNotFound, "NOT_FOUND", "Directory not found")
	}

	oldName := dirRecord.GetString("name")
	parentDirID := dirRecord.GetString("parent_directory")

	// Check for duplicate name in parent
	filter := "user = {:user} && name = {:name} && id != {:id}"
	params := map[string]any{
		"user": userID,
		"name": req.Name,
		"id":   directoryID,
	}

	if parentDirID != "" {
		filter += " && parent_directory = {:parent}"
		params["parent"] = parentDirID
	} else {
		filter += " && parent_directory = ''"
	}

	existingDirs, err := h.app.FindRecordsByFilter("directories", filter, "", 1, 0, params)
	if err == nil && len(existingDirs) > 0 {
		return h.handleError(c, errors.New("duplicate name"), http.StatusConflict, "DUPLICATE_NAME", "A directory with this name already exists in this location")
	}

	// Update directory name
	dirRecord.Set("name", req.Name)

	// Recalculate path
	oldPath := dirRecord.GetString("path")
	newPath := h.calculateFullPath(parentDirID, req.Name)
	dirRecord.Set("path", newPath)

	if err := h.app.Save(dirRecord); err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Failed to update directory")
		return h.handleError(c, err, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update directory")
	}

	// Update paths of all children
	if err := h.updateChildPaths(directoryID, oldPath, newPath); err != nil {
		h.logger.Warn().Err(err).Str("directory_id", directoryID).Msg("Failed to update child paths")
	}

	h.logger.Info().
		Str("directory_id", directoryID).
		Str("old_name", oldName).
		Str("new_name", req.Name).
		Str("user_id", userID).
		Msg("Directory renamed successfully")

	// Prepare response
	response := DirectoryResponse{
		Directory: &DirectoryInfo{
			ID:              dirRecord.Id,
			Name:            dirRecord.GetString("name"),
			Path:            dirRecord.GetString("path"),
			ParentDirectory: dirRecord.GetString("parent_directory"),
			Created:         dirRecord.GetDateTime("created").String(),
			Updated:         dirRecord.GetDateTime("updated").String(),
		},
	}

	if IsHTMXRequest(c) {
		return c.HTML(http.StatusOK, fmt.Sprintf(`
			<div class="directory-item" data-id="%s">
				<span class="icon">📁</span>
				<span class="name">%s</span>
			</div>
		`, dirRecord.Id, dirRecord.GetString("name")))
	}

	return c.JSON(http.StatusOK, response)
}

// HandleMove moves a directory to a new parent
// PATCH /api/directories/{id}/move
func (h *DirectoryHandler) HandleMove(c *core.RequestEvent) error {
	// Get directory ID from URL parameter
	directoryID := c.Request.PathValue("id")

	// Get authenticated user
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		h.logger.Warn().Msg("Unauthorized directory move attempt")
		return h.handleError(c, errors.New("unauthorized"), http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	user := authRecord.(*core.Record)
	userID := user.Id

	// Check permissions for source directory
	canDelete, err := h.permissionService.CanDeleteDirectory(userID, directoryID)
	if err != nil || !canDelete {
		h.logger.Warn().Str("user_id", userID).Str("directory_id", directoryID).Msg("Permission denied for directory move")
		return h.handleError(c, errors.New("permission denied"), http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to move this directory")
	}

	// Parse request body
	var req MoveDirectoryRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid request body")
		return h.handleError(c, err, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	targetDirID := req.TargetDirectory
	if targetDirID == "root" {
		targetDirID = ""
	}

	// Check permissions for target directory
	canCreate, err := h.permissionService.CanCreateDirectory(userID, targetDirID)
	if err != nil || !canCreate {
		h.logger.Warn().Str("user_id", userID).Str("target_dir_id", targetDirID).Msg("Permission denied for target directory")
		return h.handleError(c, errors.New("permission denied"), http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to move directories to this location")
	}

	// Prevent moving into self
	if directoryID == targetDirID {
		return h.handleError(c, errors.New("circular reference"), http.StatusBadRequest, "CIRCULAR_REFERENCE", "Cannot move a directory into itself")
	}

	// Prevent circular references (moving into own subtree)
	if targetDirID != "" {
		isSubtree, err := h.isSubtree(targetDirID, directoryID)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to check for circular reference")
			return h.handleError(c, err, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate move operation")
		}
		if isSubtree {
			return h.handleError(c, errors.New("circular reference"), http.StatusBadRequest, "CIRCULAR_REFERENCE", "Cannot move a directory into one of its subdirectories")
		}
	}

	// Get directory record
	dirRecord, err := h.app.FindRecordById("directories", directoryID)
	if err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Directory not found")
		return h.handleError(c, err, http.StatusNotFound, "NOT_FOUND", "Directory not found")
	}

	dirName := dirRecord.GetString("name")

	// Check for duplicate name in target
	filter := "user = {:user} && name = {:name} && id != {:id}"
	params := map[string]any{
		"user": userID,
		"name": dirName,
		"id":   directoryID,
	}

	if targetDirID != "" {
		filter += " && parent_directory = {:parent}"
		params["parent"] = targetDirID
	} else {
		filter += " && parent_directory = ''"
	}

	existingDirs, err := h.app.FindRecordsByFilter("directories", filter, "", 1, 0, params)
	if err == nil && len(existingDirs) > 0 {
		return h.handleError(c, errors.New("duplicate name"), http.StatusConflict, "DUPLICATE_NAME", "A directory with this name already exists in the target location")
	}

	// Update parent directory
	oldPath := dirRecord.GetString("path")
	if targetDirID != "" {
		dirRecord.Set("parent_directory", targetDirID)
	} else {
		dirRecord.Set("parent_directory", "")
	}

	// Recalculate path
	newPath := h.calculateFullPath(targetDirID, dirName)
	dirRecord.Set("path", newPath)

	if err := h.app.Save(dirRecord); err != nil {
		h.logger.Error().Err(err).Str("directory_id", directoryID).Msg("Failed to move directory")
		return h.handleError(c, err, http.StatusInternalServerError, "MOVE_FAILED", "Failed to move directory")
	}

	// Update paths of all children
	if err := h.updateChildPaths(directoryID, oldPath, newPath); err != nil {
		h.logger.Warn().Err(err).Str("directory_id", directoryID).Msg("Failed to update child paths")
	}

	h.logger.Info().
		Str("directory_id", directoryID).
		Str("target_dir_id", targetDirID).
		Str("user_id", userID).
		Msg("Directory moved successfully")

	// Prepare response
	response := DirectoryResponse{
		Directory: &DirectoryInfo{
			ID:              dirRecord.Id,
			Name:            dirRecord.GetString("name"),
			Path:            dirRecord.GetString("path"),
			ParentDirectory: dirRecord.GetString("parent_directory"),
			Created:         dirRecord.GetDateTime("created").String(),
			Updated:         dirRecord.GetDateTime("updated").String(),
		},
	}

	if IsHTMXRequest(c) {
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, response)
}

// Helper functions

// calculateFullPath calculates the full path for a directory
func (h *DirectoryHandler) calculateFullPath(parentDirID, name string) string {
	if parentDirID == "" {
		return "/" + name
	}

	parentDir, err := h.app.FindRecordById("directories", parentDirID)
	if err != nil {
		h.logger.Warn().Err(err).Str("parent_dir_id", parentDirID).Msg("Failed to find parent directory")
		return "/" + name
	}

	parentPath := parentDir.GetString("path")
	if parentPath == "" || parentPath == "/" {
		return "/" + name
	}

	return strings.TrimSuffix(parentPath, "/") + "/" + name
}

// updateChildPaths updates the paths of all child directories and files
func (h *DirectoryHandler) updateChildPaths(directoryID, _, newPath string) error {
	// Update subdirectories
	subdirs, err := h.app.FindRecordsByFilter(
		"directories",
		"parent_directory = {:parent}",
		"",
		0,
		0,
		map[string]any{"parent": directoryID},
	)
	if err != nil {
		return err
	}

	for _, subdir := range subdirs {
		subdirPath := subdir.GetString("path")
		subdirName := subdir.GetString("name")

		// Calculate new path
		newSubdirPath := strings.TrimSuffix(newPath, "/") + "/" + subdirName
		subdir.Set("path", newSubdirPath)

		if err := h.app.Save(subdir); err != nil {
			h.logger.Warn().Err(err).Str("subdir_id", subdir.Id).Msg("Failed to update subdirectory path")
			continue
		}

		// Recursively update children of this subdirectory
		if err := h.updateChildPaths(subdir.Id, subdirPath, newSubdirPath); err != nil {
			h.logger.Warn().Err(err).Str("subdir_id", subdir.Id).Msg("Failed to update children of subdirectory")
		}
	}

	// Update files
	files, err := h.app.FindRecordsByFilter(
		"files",
		"parent_directory = {:parent}",
		"",
		0,
		0,
		map[string]any{"parent": directoryID},
	)
	if err != nil {
		return err
	}

	for _, file := range files {
		fileName := file.GetString("name")
		newFilePath := strings.TrimSuffix(newPath, "/") + "/" + fileName
		file.Set("path", newFilePath)

		if err := h.app.Save(file); err != nil {
			h.logger.Warn().Err(err).Str("file_id", file.Id).Msg("Failed to update file path")
		}
	}

	return nil
}

// isSubtree checks if targetDirID is a subdirectory of directoryID
func (h *DirectoryHandler) isSubtree(targetDirID, directoryID string) (bool, error) {
	visited := make(map[string]bool)
	current := targetDirID

	for current != "" && !visited[current] {
		visited[current] = true

		if current == directoryID {
			return true, nil
		}

		// Get parent
		record, err := h.app.FindRecordById("directories", current)
		if err != nil {
			return false, err
		}

		current = record.GetString("parent_directory")
	}

	return false, nil
}

// deleteDirectoryRecursive deletes a directory and all its contents
func (h *DirectoryHandler) deleteDirectoryRecursive(directoryID, userID string) error {
	// Delete all subdirectories recursively
	subdirs, err := h.app.FindRecordsByFilter(
		"directories",
		"parent_directory = {:parent}",
		"",
		0,
		0,
		map[string]any{"parent": directoryID},
	)
	if err != nil {
		return err
	}

	for _, subdir := range subdirs {
		if err := h.deleteDirectoryRecursive(subdir.Id, userID); err != nil {
			return err
		}

		if err := h.app.Delete(subdir); err != nil {
			h.logger.Warn().Err(err).Str("subdir_id", subdir.Id).Msg("Failed to delete subdirectory")
		}
	}

	// Delete all files in this directory
	files, err := h.app.FindRecordsByFilter(
		"files",
		"parent_directory = {:parent}",
		"",
		0,
		0,
		map[string]any{"parent": directoryID},
	)
	if err != nil {
		return err
	}

	for _, file := range files {
		// Delete from S3
		s3Key := file.GetString("s3_key")
		if s3Key != "" && h.s3Service != nil {
			if err := h.s3Service.DeleteFile(s3Key); err != nil {
				h.logger.Warn().Err(err).Str("s3_key", s3Key).Msg("Failed to delete file from S3")
			}
		}

		// Delete database record
		if err := h.app.Delete(file); err != nil {
			h.logger.Warn().Err(err).Str("file_id", file.Id).Msg("Failed to delete file record")
		}
	}

	return nil
}

// handleError handles errors consistently
func (h *DirectoryHandler) handleError(c *core.RequestEvent, err error, statusCode int, code, message string) error {
	detail := ""
	if h.app != nil {
		// Only include error details in development
		detail = err.Error()
	}

	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
	}

	if IsHTMXRequest(c) {
		return c.HTML(statusCode, fmt.Sprintf(`
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm">%s</p>
			</div>
		`, message))
	}

	return c.JSON(statusCode, response)
}
