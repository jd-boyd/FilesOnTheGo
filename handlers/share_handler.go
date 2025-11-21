package handlers

import (
	"encoding/json"
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
	authRecord := c.Auth()
	if authRecord == nil {
		h.logger.Warn().Msg("Unauthenticated share creation attempt")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

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
		UserID:         authRecord.Id,
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		PermissionType: req.PermissionType,
		Password:       req.Password,
		ExpiresAt:      req.ExpiresAt,
	}

	share, err := h.shareService.CreateShare(params)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", authRecord.Id).Msg("Failed to create share")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	h.logger.Info().
		Str("share_id", share.ID).
		Str("user_id", authRecord.Id).
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
	authRecord := c.Auth()
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Get optional resource type filter
	resourceType := c.Request.URL.Query().Get("resource_type")

	// List shares
	shares, err := h.shareService.ListUserShares(authRecord.Id, resourceType)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", authRecord.Id).Msg("Failed to list shares")
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
	authRecord := c.Auth()
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

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
	if share.UserID != authRecord.Id {
		h.logger.Warn().
			Str("user_id", authRecord.Id).
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
	authRecord := c.Auth()
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

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
	err := h.shareService.UpdateShareExpiration(shareID, authRecord.Id, req.ExpiresAt)
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
	authRecord := c.Auth()
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share ID is required",
		})
	}

	// Revoke share
	err := h.shareService.RevokeShare(shareID, authRecord.Id)
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
	authRecord := c.Auth()
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	shareID := c.Request.PathValue("share_id")
	if shareID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Share ID is required",
		})
	}

	// Get access logs
	logs, err := h.shareService.GetShareAccessLogs(shareID, authRecord.Id)
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
