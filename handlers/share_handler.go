package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
)

// ShareHandler handles share link related requests
type ShareHandler struct {
	app          *pocketbase.PocketBase
	shareService services.ShareService
	logger       zerolog.Logger
}

// NewShareHandler creates a new share handler
func NewShareHandler(app *pocketbase.PocketBase, shareService services.ShareService, logger zerolog.Logger) *ShareHandler {
	return &ShareHandler{
		app:          app,
		shareService: shareService,
		logger:       logger,
	}
}

// CreateShareRequest represents the request body for creating a share
type CreateShareRequest struct {
	ResourceType   string     `json:"resource_type"`
	ResourceID     string     `json:"resource_id"`
	PermissionType string     `json:"permission_type"`
	Password       string     `json:"password,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// UpdateShareRequest represents the request body for updating a share
type UpdateShareRequest struct {
	ExpiresAt *time.Time `json:"expires_at"`
}

// ValidateSharePasswordRequest represents the request body for password validation
type ValidateSharePasswordRequest struct {
	Password string `json:"password"`
}

// CreateShare handles POST /api/shares
func (h *ShareHandler) CreateShare(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		h.logger.Warn().Msg("Unauthenticated share creation attempt")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	user := authRecord.(*core.Record)

	// Parse request body
	var req CreateShareRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid request body for share creation")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Create share
	params := services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		PermissionType: req.PermissionType,
		Password:       req.Password,
		ExpiresAt:      req.ExpiresAt,
	}

	share, err := h.shareService.CreateShare(params)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.Id).Msg("Failed to create share")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	h.logger.Info().
		Str("share_id", share.ID).
		Str("user_id", user.Id).
		Str("resource_type", req.ResourceType).
		Msg("Share created")

	// Generate share URL
	baseURL := h.getBaseURL(c)
	shareURL := baseURL + "/share/" + share.ShareToken

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"share": share,
		"url":   shareURL,
	})
}

// ListShares handles GET /api/shares
func (h *ShareHandler) ListShares(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	user := authRecord.(*core.Record)

	// Get optional resource type filter
	resourceType := c.Request.URL.Query().Get("resource_type")

	// List shares
	shares, err := h.shareService.ListUserShares(user.Id, resourceType)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.Id).Msg("Failed to list shares")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list shares",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"shares": shares,
	})
}

// GetShare handles GET /api/shares/{share_id}
func (h *ShareHandler) GetShare(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	user := authRecord.(*core.Record)

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share ID is required",
		})
	}

	// Get share
	share, err := h.shareService.GetShareByID(shareID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Share not found",
		})
	}

	// Verify user owns the share
	if share.UserID != user.Id {
		h.logger.Warn().
			Str("user_id", user.Id).
			Str("share_id", shareID).
			Msg("Unauthorized share access attempt")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You do not have permission to access this share",
		})
	}

	// Generate share URL
	baseURL := h.getBaseURL(c)
	shareURL := baseURL + "/share/" + share.ShareToken

	return c.JSON(http.StatusOK, map[string]interface{}{
		"share": share,
		"url":   shareURL,
	})
}

// UpdateShare handles PATCH /api/shares/{share_id}
func (h *ShareHandler) UpdateShare(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	user := authRecord.(*core.Record)

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share ID is required",
		})
	}

	// Parse request body
	var req UpdateShareRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Update expiration
	err := h.shareService.UpdateShareExpiration(shareID, user.Id, req.ExpiresAt)
	if err != nil {
		h.logger.Error().Err(err).Str("share_id", shareID).Msg("Failed to update share")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Get updated share
	share, err := h.shareService.GetShareByID(shareID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Share not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"share": share,
	})
}

// RevokeShare handles DELETE /api/shares/{share_id}
func (h *ShareHandler) RevokeShare(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	user := authRecord.(*core.Record)

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share ID is required",
		})
	}

	// Revoke share
	err := h.shareService.RevokeShare(shareID, user.Id)
	if err != nil {
		h.logger.Error().Err(err).Str("share_id", shareID).Msg("Failed to revoke share")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	h.logger.Info().Str("share_id", shareID).Msg("Share revoked")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Share revoked successfully",
	})
}

// GetShareLogs handles GET /api/shares/{share_id}/logs
func (h *ShareHandler) GetShareLogs(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	user := authRecord.(*core.Record)

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share ID is required",
		})
	}

	// Get access logs
	logs, err := h.shareService.GetShareAccessLogs(shareID, user.Id)
	if err != nil {
		h.logger.Error().Err(err).Str("share_id", shareID).Msg("Failed to get share logs")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs": logs,
	})
}

// AccessPublicShare handles GET /api/public/share/{share_token}
func (h *ShareHandler) AccessPublicShare(c *core.RequestEvent) error {
	shareToken := c.Request.PathValue("share_token")
	if shareToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share token is required",
		})
	}

	// Get share info
	share, err := h.shareService.GetShareByToken(shareToken)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Share not found",
		})
	}

	// Check if password is provided in query
	password := c.Request.URL.Query().Get("password")

	// Validate share access
	accessInfo, err := h.shareService.ValidateShareAccess(shareToken, password)
	if err != nil {
		h.logger.Error().Err(err).Str("share_token", shareToken).Msg("Share validation error")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to validate share",
		})
	}

	if !accessInfo.IsValid {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"valid":             false,
			"error":             accessInfo.ErrorMessage,
			"requires_password": share.IsPasswordProtected,
		})
	}

	// Log access
	ipAddress := c.Request.RemoteAddr
	userAgent := c.Request.UserAgent()
	h.shareService.LogShareAccess(share.ID, "view", "", ipAddress, userAgent)

	// Increment access count
	h.shareService.IncrementAccessCount(share.ID)

	// Return share info (excluding sensitive data)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":           true,
		"share_id":        share.ID,
		"resource_type":   share.ResourceType,
		"resource_id":     share.ResourceID,
		"permission_type": share.PermissionType,
		"expires_at":      share.ExpiresAt,
		"access_count":    share.AccessCount,
	})
}

// ValidateSharePassword handles POST /api/public/share/{share_token}/validate
func (h *ShareHandler) ValidateSharePassword(c *core.RequestEvent) error {
	shareToken := c.Request.PathValue("share_token")
	if shareToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share token is required",
		})
	}

	// Parse request body
	var req ValidateSharePasswordRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate share access with password
	accessInfo, err := h.shareService.ValidateShareAccess(shareToken, req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to validate password",
		})
	}

	if !accessInfo.IsValid {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"valid": false,
			"error": accessInfo.ErrorMessage,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":           true,
		"share_id":        accessInfo.ShareID,
		"resource_type":   accessInfo.ResourceType,
		"resource_id":     accessInfo.ResourceID,
		"permission_type": accessInfo.PermissionType,
	})
}

// getBaseURL extracts the base URL from the request
func (h *ShareHandler) getBaseURL(c *core.RequestEvent) string {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}

	// Check for X-Forwarded-Proto header (for proxies)
	if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := c.Request.Host
	return scheme + "://" + host
}

// ShareHTMXData represents data for HTMX share templates
type ShareHTMXData struct {
	ID                  string
	ShareURL            string
	ShareURLTruncated   string
	PermissionType      string
	IsPasswordProtected bool
	IsExpired           bool
	ExpiresAt           *time.Time
	ExpiresAtFormatted  string
	AccessCount         int64
	CreatedFormatted    string
	ResourceType        string
	Error               string
}

// ShareLogsHTMXData represents data for share access logs template
type ShareLogsHTMXData struct {
	Logs []ShareLogHTMXItem
}

// ShareLogHTMXItem represents a single access log entry for the template
type ShareLogHTMXItem struct {
	AccessedAtFormatted string
	Action              string
	FileName            string
	IPAddress           string
}

// CreateShareHTMX handles POST /api/shares/create-htmx - returns HTML fragment
func (h *ShareHandler) CreateShareHTMX(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return h.renderShareError(c, "Authentication required")
	}

	user := authRecord.(*core.Record)

	// Parse form data
	resourceType := c.Request.FormValue("resource_type")
	resourceID := c.Request.FormValue("resource_id")
	permissionType := c.Request.FormValue("permission_type")
	password := c.Request.FormValue("password")
	expiresAtStr := c.Request.FormValue("expires_at")

	// Parse expiration date
	var expiresAt *time.Time
	if expiresAtStr != "" {
		parsed, err := time.Parse("2006-01-02T15:04", expiresAtStr)
		if err == nil {
			expiresAt = &parsed
		}
	}

	// Create share
	params := services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		PermissionType: permissionType,
		Password:       password,
		ExpiresAt:      expiresAt,
	}

	share, err := h.shareService.CreateShare(params)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.Id).Msg("Failed to create share via HTMX")
		return h.renderShareError(c, err.Error())
	}

	h.logger.Info().
		Str("share_id", share.ID).
		Str("user_id", user.Id).
		Str("resource_type", resourceType).
		Msg("Share created via HTMX")

	// Generate share URL
	baseURL := h.getBaseURL(c)
	shareURL := baseURL + "/share/" + share.ShareToken

	// Format expiration date
	expiresAtFormatted := ""
	if share.ExpiresAt != nil {
		expiresAtFormatted = share.ExpiresAt.Format("Jan 2, 2006 3:04 PM")
	}

	// Render success template
	data := ShareHTMXData{
		ID:                  share.ID,
		ShareURL:            shareURL,
		PermissionType:      string(share.PermissionType),
		IsPasswordProtected: share.IsPasswordProtected,
		ExpiresAt:           share.ExpiresAt,
		ExpiresAtFormatted:  expiresAtFormatted,
		ResourceType:        resourceType,
	}

	return h.renderTemplate(c, "share-link-display", data)
}

// ListSharesHTMX handles GET /api/shares/list-htmx - returns HTML fragment
func (h *ShareHandler) ListSharesHTMX(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.String(http.StatusUnauthorized, "Authentication required")
	}

	user := authRecord.(*core.Record)

	// Get optional filters
	resourceType := c.Request.URL.Query().Get("resource_type")

	// List shares
	shares, err := h.shareService.ListUserShares(user.Id, resourceType)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.Id).Msg("Failed to list shares via HTMX")
		return c.String(http.StatusInternalServerError, "Failed to load shares")
	}

	// Generate base URL for share links
	baseURL := h.getBaseURL(c)

	// Convert to template data
	shareDataList := make([]ShareHTMXData, 0, len(shares))
	for _, share := range shares {
		shareURL := baseURL + "/share/" + share.ShareToken

		// Truncate URL for display
		truncatedURL := shareURL
		if len(truncatedURL) > 50 {
			truncatedURL = truncatedURL[:47] + "..."
		}

		// Format dates
		expiresAtFormatted := ""
		if share.ExpiresAt != nil {
			expiresAtFormatted = share.ExpiresAt.Format("Jan 2, 2006")
		}
		createdFormatted := share.Created.Format("Jan 2, 2006")

		shareDataList = append(shareDataList, ShareHTMXData{
			ID:                  share.ID,
			ShareURL:            shareURL,
			ShareURLTruncated:   truncatedURL,
			PermissionType:      string(share.PermissionType),
			IsPasswordProtected: share.IsPasswordProtected,
			IsExpired:           share.IsExpired,
			ExpiresAt:           share.ExpiresAt,
			ExpiresAtFormatted:  expiresAtFormatted,
			AccessCount:         share.AccessCount,
			CreatedFormatted:    createdFormatted,
			ResourceType:        string(share.ResourceType),
		})
	}

	// Render template
	return h.renderTemplate(c, "shares-list", map[string]interface{}{
		"Shares": shareDataList,
	})
}

// GetResourceSharesHTMX handles GET /api/shares/resource/{resource_type}/{resource_id} - returns existing shares for a resource
func (h *ShareHandler) GetResourceSharesHTMX(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.String(http.StatusUnauthorized, "Authentication required")
	}

	user := authRecord.(*core.Record)

	resourceType := c.Request.PathValue("resource_type")
	resourceID := c.Request.PathValue("resource_id")

	if resourceType == "" || resourceID == "" {
		return c.String(http.StatusBadRequest, "Resource type and ID are required")
	}

	// List shares for this resource
	shares, err := h.shareService.ListUserShares(user.Id, resourceType)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.Id).Msg("Failed to list resource shares via HTMX")
		return c.String(http.StatusInternalServerError, "Failed to load shares")
	}

	// Filter to only shares for this resource
	baseURL := h.getBaseURL(c)
	shareDataList := make([]ShareHTMXData, 0)

	for _, share := range shares {
		if share.ResourceID != resourceID {
			continue
		}

		shareURL := baseURL + "/share/" + share.ShareToken
		truncatedURL := shareURL
		if len(truncatedURL) > 50 {
			truncatedURL = truncatedURL[:47] + "..."
		}

		expiresAtFormatted := ""
		if share.ExpiresAt != nil {
			expiresAtFormatted = share.ExpiresAt.Format("Jan 2, 2006")
		}
		createdFormatted := share.Created.Format("Jan 2, 2006")

		shareDataList = append(shareDataList, ShareHTMXData{
			ID:                  share.ID,
			ShareURL:            shareURL,
			ShareURLTruncated:   truncatedURL,
			PermissionType:      string(share.PermissionType),
			IsPasswordProtected: share.IsPasswordProtected,
			IsExpired:           share.IsExpired,
			ExpiresAt:           share.ExpiresAt,
			ExpiresAtFormatted:  expiresAtFormatted,
			AccessCount:         share.AccessCount,
			CreatedFormatted:    createdFormatted,
			ResourceType:        string(share.ResourceType),
		})
	}

	return h.renderTemplate(c, "shares-list", map[string]interface{}{
		"Shares": shareDataList,
	})
}

// GetShareLogsHTMX handles GET /api/shares/{share_id}/logs-htmx - returns access logs HTML
func (h *ShareHandler) GetShareLogsHTMX(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.String(http.StatusUnauthorized, "Authentication required")
	}

	user := authRecord.(*core.Record)

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.String(http.StatusBadRequest, "Share ID is required")
	}

	// Get access logs
	logs, err := h.shareService.GetShareAccessLogs(shareID, user.Id)
	if err != nil {
		h.logger.Error().Err(err).Str("share_id", shareID).Msg("Failed to get share logs via HTMX")
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Convert to template data
	logItems := make([]ShareLogHTMXItem, 0, len(logs))
	for _, log := range logs {
		logItems = append(logItems, ShareLogHTMXItem{
			AccessedAtFormatted: log.AccessedAt.Format("Jan 2, 3:04 PM"),
			Action:              log.Action,
			FileName:            log.FileName,
			IPAddress:           log.IPAddress,
		})
	}

	return h.renderTemplate(c, "share-access-logs", ShareLogsHTMXData{
		Logs: logItems,
	})
}

// RevokeShareHTMX handles DELETE /api/shares/{share_id} with HTMX - returns empty/revoked message
func (h *ShareHandler) RevokeShareHTMX(c *core.RequestEvent) error {
	// Get authenticated user ID
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.String(http.StatusUnauthorized, "Authentication required")
	}

	user := authRecord.(*core.Record)

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.String(http.StatusBadRequest, "Share ID is required")
	}

	// Revoke share
	err := h.shareService.RevokeShare(shareID, user.Id)
	if err != nil {
		h.logger.Error().Err(err).Str("share_id", shareID).Msg("Failed to revoke share")
		return c.String(http.StatusBadRequest, err.Error())
	}

	h.logger.Info().Str("share_id", shareID).Msg("Share revoked via HTMX")

	// Check if this is an HTMX request
	if c.Request.Header.Get("HX-Request") == "true" {
		// Return the revoked template or empty response
		return h.renderTemplate(c, "share-revoked", nil)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Share revoked successfully",
	})
}

// renderShareError renders an error message for share creation
func (h *ShareHandler) renderShareError(c *core.RequestEvent, errorMsg string) error {
	return h.renderTemplate(c, "share-error", ShareHTMXData{
		Error: errorMsg,
	})
}

// renderTemplate renders an HTML template with the given data
func (h *ShareHandler) renderTemplate(c *core.RequestEvent, templateName string, data interface{}) error {
	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Build the HTML based on template name
	var html string
	switch templateName {
	case "share-link-display":
		html = h.buildShareLinkDisplayHTML(data.(ShareHTMXData))
	case "share-error":
		html = h.buildShareErrorHTML(data.(ShareHTMXData))
	case "shares-list":
		dataMap := data.(map[string]interface{})
		shares := dataMap["Shares"].([]ShareHTMXData)
		html = h.buildSharesListHTML(shares)
	case "share-access-logs":
		logsData := data.(ShareLogsHTMXData)
		html = h.buildShareAccessLogsHTML(logsData)
	case "share-revoked":
		html = `<div class="share-item border border-gray-200 rounded-lg p-4 bg-gray-50 opacity-50" style="animation: fadeOut 0.3s forwards;"><div class="text-center text-sm text-gray-500">Share revoked</div></div>`
	default:
		html = "<div>Template not found</div>"
	}

	return c.String(http.StatusOK, html)
}

// buildShareLinkDisplayHTML builds the HTML for a newly created share link
func (h *ShareHandler) buildShareLinkDisplayHTML(data ShareHTMXData) string {
	expiresInfo := "Never expires"
	if data.ExpiresAtFormatted != "" {
		expiresInfo = "Expires " + data.ExpiresAtFormatted
	}

	permBadge := h.getPermissionBadgeHTML(data.PermissionType)
	passwordBadge := ""
	if data.IsPasswordProtected {
		passwordBadge = `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800"><svg class="h-3 w-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path></svg>PASSWORD</span>`
	}

	resourceType := "file"
	if data.ResourceType == "directory" {
		resourceType = "folder"
	}

	passwordNote := ""
	if data.IsPasswordProtected {
		passwordNote = " They will need the password you set to access it."
	}

	return `<div class="bg-green-50 border border-green-200 rounded-lg p-4" data-share-url="` + data.ShareURL + `">
    <div class="flex items-center mb-3">
        <svg class="h-5 w-5 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        <span class="font-medium text-green-800">Share link created!</span>
    </div>
    <div class="flex items-center space-x-2 mb-3">
        <div class="flex-1 relative">
            <input type="text" id="share-url-input" value="` + data.ShareURL + `" readonly
                   class="w-full bg-white border border-gray-300 rounded-md px-3 py-2 pr-10 text-sm font-mono text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
                   onclick="this.select()">
            <button type="button" onclick="copyShareLinkFromInput()"
                    class="absolute right-2 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-blue-600">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
                </svg>
            </button>
        </div>
        <button type="button" onclick="copyShareLinkFromInput()"
                class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700">
            <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
            </svg>
            Copy
        </button>
    </div>
    <div class="flex flex-wrap items-center gap-2 text-sm">
        ` + permBadge + `
        ` + passwordBadge + `
        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-700">
            <svg class="h-3 w-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            ` + expiresInfo + `
        </span>
    </div>
    <p class="mt-3 text-sm text-gray-600">Share this link with anyone you want to give access to this ` + resourceType + `.` + passwordNote + `</p>
</div>`
}

// buildShareErrorHTML builds the HTML for a share creation error
func (h *ShareHandler) buildShareErrorHTML(data ShareHTMXData) string {
	return `<div class="bg-red-50 border border-red-200 rounded-lg p-4">
    <div class="flex items-center">
        <svg class="h-5 w-5 text-red-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        <span class="font-medium text-red-800">Failed to create share</span>
    </div>
    <p class="mt-2 text-sm text-red-700">` + data.Error + `</p>
</div>`
}

// buildSharesListHTML builds the HTML for a list of shares
func (h *ShareHandler) buildSharesListHTML(shares []ShareHTMXData) string {
	if len(shares) == 0 {
		return `<div class="text-center py-8">
    <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"></path>
    </svg>
    <h3 class="mt-2 text-sm font-medium text-gray-900">No shares yet</h3>
    <p class="mt-1 text-sm text-gray-500">Create a share link to give others access to this item.</p>
</div>`
	}

	var html string
	html = `<div class="space-y-3">`
	for _, share := range shares {
		html += h.buildShareListItemHTML(share)
	}
	html += `</div>`
	return html
}

// buildShareListItemHTML builds the HTML for a single share list item
func (h *ShareHandler) buildShareListItemHTML(data ShareHTMXData) string {
	permBadge := h.getPermissionBadgeHTML(data.PermissionType)

	protectedBadge := ""
	if data.IsPasswordProtected {
		protectedBadge = `<span class="inline-flex items-center text-xs text-purple-600"><svg class="h-3.5 w-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path></svg>Protected</span>`
	}

	expiresBadge := ""
	if data.IsExpired {
		expiresBadge = `<span class="inline-flex items-center text-xs text-red-600"><svg class="h-3.5 w-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>Expired</span>`
	} else if data.ExpiresAtFormatted != "" {
		expiresBadge = `<span class="inline-flex items-center text-xs text-gray-500"><svg class="h-3.5 w-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>` + data.ExpiresAtFormatted + `</span>`
	}

	accessPlural := "es"
	if data.AccessCount == 1 {
		accessPlural = ""
	}

	return `<div class="share-item border border-gray-200 rounded-lg p-4 hover:bg-gray-50 transition-colors" id="share-item-` + data.ID + `" data-share-url="` + data.ShareURL + `">
    <div class="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-3">
        <div class="flex-1 min-w-0">
            <div class="flex items-center space-x-2 mb-2">
                <code class="text-sm font-mono bg-gray-100 px-2 py-1 rounded truncate max-w-xs sm:max-w-md" title="` + data.ShareURL + `">` + data.ShareURLTruncated + `</code>
                <button type="button" onclick="copyShareLinkById('share-item-` + data.ID + `')" class="text-blue-600 hover:text-blue-800" title="Copy link">
                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
                </button>
            </div>
            <div class="flex flex-wrap items-center gap-2 text-sm">
                ` + permBadge + `
                ` + protectedBadge + `
                ` + expiresBadge + `
                <span class="inline-flex items-center text-xs text-gray-500">
                    <svg class="h-3.5 w-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path></svg>
                    ` + fmt.Sprintf("%d", data.AccessCount) + ` access` + accessPlural + `
                </span>
                <span class="inline-flex items-center text-xs text-gray-400">Created ` + data.CreatedFormatted + `</span>
            </div>
        </div>
        <div class="flex items-center space-x-2 flex-shrink-0">
            <button type="button" hx-get="/api/shares/` + data.ID + `/logs-htmx" hx-target="#share-logs-` + data.ID + `" hx-swap="innerHTML"
                    class="inline-flex items-center px-2 py-1 text-xs font-medium text-gray-600 hover:text-gray-900 rounded" title="View access logs">
                <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"></path></svg>
                Logs
            </button>
            <button type="button" hx-delete="/api/shares/` + data.ID + `" hx-target="#share-item-` + data.ID + `" hx-swap="outerHTML swap:0.3s" hx-confirm="Revoke this share link? It will immediately stop working."
                    class="inline-flex items-center px-2 py-1 text-xs font-medium text-red-600 hover:text-red-800 hover:bg-red-50 rounded transition-colors" title="Revoke share">
                <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                Revoke
            </button>
        </div>
    </div>
    <div id="share-logs-` + data.ID + `" class="mt-3"></div>
</div>`
}

// buildShareAccessLogsHTML builds the HTML for share access logs
func (h *ShareHandler) buildShareAccessLogsHTML(data ShareLogsHTMXData) string {
	if len(data.Logs) == 0 {
		return `<div class="bg-gray-50 rounded-md p-3 border border-gray-200"><p class="text-xs text-gray-500 text-center">No access logs yet</p></div>`
	}

	html := `<div class="bg-gray-50 rounded-md p-3 border border-gray-200">
    <h5 class="text-xs font-medium text-gray-700 mb-2">Recent Access Log</h5>
    <div class="space-y-1 max-h-32 overflow-y-auto">`

	for _, log := range data.Logs {
		fileName := ""
		if log.FileName != "" {
			fileName = `<span class="text-gray-500 truncate max-w-32">` + log.FileName + `</span>`
		}
		html += `<div class="flex items-center justify-between text-xs">
            <div class="flex items-center space-x-2">
                <span class="text-gray-500">` + log.AccessedAtFormatted + `</span>
                <span class="font-medium text-gray-700">` + log.Action + `</span>
                ` + fileName + `
            </div>
            <span class="text-gray-400 truncate max-w-24" title="` + log.IPAddress + `">` + log.IPAddress + `</span>
        </div>`
	}

	html += `</div></div>`
	return html
}

// getPermissionBadgeHTML returns the HTML for a permission type badge
func (h *ShareHandler) getPermissionBadgeHTML(permissionType string) string {
	switch permissionType {
	case "read":
		return `<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">READ-ONLY</span>`
	case "read_upload":
		return `<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">READ & UPLOAD</span>`
	case "upload_only":
		return `<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">UPLOAD-ONLY</span>`
	default:
		return `<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">` + permissionType + `</span>`
	}
}
