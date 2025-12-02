package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
)

// SettingsHandler handles user settings and profile management
type SettingsHandler struct {
	userService *services.UserService
	renderer    *TemplateRenderer
	logger      zerolog.Logger
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(
	userService *services.UserService,
	renderer *TemplateRenderer,
	logger zerolog.Logger,
) *SettingsHandler {
	return &SettingsHandler{
		userService: userService,
		renderer:    renderer,
		logger:      logger,
	}
}

// ShowSettingsPage renders the settings page
func (h *SettingsHandler) ShowSettingsPage(c *gin.Context) {
	data := PrepareTemplateData(c)
	data.Title = "Settings - FilesOnTheGo"

	// Get current user info
	userID, _ := auth.GetUserID(c)
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		c.String(http.StatusInternalServerError, "Failed to load settings")
		return
	}

	data.User = user

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(c.Writer, "settings", data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render settings page")
		c.String(http.StatusInternalServerError, "Internal server error")
	}
}

// ShowProfilePage renders the profile page
func (h *SettingsHandler) ShowProfilePage(c *gin.Context) {
	data := PrepareTemplateData(c)
	data.Title = "Profile - FilesOnTheGo"

	// Get current user info
	userID, _ := auth.GetUserID(c)
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		c.String(http.StatusInternalServerError, "Failed to load profile")
		return
	}

	// Get user stats
	stats, err := h.userService.GetUserStats(userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user stats")
		// Continue without stats
		stats = map[string]interface{}{}
	}

	data.User = user
	data.Settings = stats

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(c.Writer, "profile", data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render profile page")
		c.String(http.StatusInternalServerError, "Internal server error")
	}
}

// GetProfile returns the current user's profile information
func (h *SettingsHandler) GetProfile(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// GetProfileStats returns the current user's statistics
func (h *SettingsHandler) GetProfileStats(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	stats, err := h.userService.GetUserStats(userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// UpdateProfile handles updating the current user's profile
func (h *SettingsHandler) UpdateProfile(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	isHTMX := IsHTMXRequest(c)

	var req struct {
		Email           string `json:"email" form:"email"`
		Username        string `json:"username" form:"username"`
		EmailVisibility bool   `json:"email_visibility" form:"email_visibility"`
	}

	// Handle both JSON and form data
	if isHTMX {
		if err := c.ShouldBind(&req); err != nil {
			h.handleUpdateError(c, isHTMX, "Invalid input")
			return
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Build updates map (only include non-empty fields)
	updates := make(map[string]interface{})
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	updates["email_visibility"] = req.EmailVisibility

	// Update user
	user, err := h.userService.UpdateUser(userID, updates)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update profile")
		h.handleUpdateError(c, isHTMX, err.Error())
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Interface("updates", updates).
		Msg("Profile updated successfully")

	if isHTMX {
		c.Data(http.StatusOK, "text/html", []byte(`
			<div class="bg-green-50 border border-green-200 text-green-800 rounded-md p-4">
				<p class="text-sm">Profile updated successfully</p>
			</div>
		`))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    user,
	})
}

// UpdatePassword handles updating the current user's password
func (h *SettingsHandler) UpdatePassword(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	isHTMX := IsHTMXRequest(c)

	var req struct {
		CurrentPassword string `json:"current_password" form:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" form:"new_password" binding:"required,min=8"`
		ConfirmPassword string `json:"confirm_password" form:"confirm_password" binding:"required"`
	}

	// Handle both JSON and form data
	if isHTMX {
		if err := c.ShouldBind(&req); err != nil {
			h.handlePasswordError(c, isHTMX, "All fields are required")
			return
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Validate password confirmation
	if req.NewPassword != req.ConfirmPassword {
		h.handlePasswordError(c, isHTMX, "New passwords do not match")
		return
	}

	// Update password
	if err := h.userService.UpdatePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update password")
		h.handlePasswordError(c, isHTMX, err.Error())
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Msg("Password updated successfully")

	if isHTMX {
		c.Data(http.StatusOK, "text/html", []byte(`
			<div class="bg-green-50 border border-green-200 text-green-800 rounded-md p-4">
				<p class="text-sm">Password updated successfully</p>
			</div>
		`))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password updated successfully",
	})
}

// Helper methods for error handling

func (h *SettingsHandler) handleUpdateError(c *gin.Context, isHTMX bool, message string) {
	if isHTMX {
		c.Data(http.StatusBadRequest, "text/html", []byte(`
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm">`+message+`</p>
			</div>
		`))
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}

func (h *SettingsHandler) handlePasswordError(c *gin.Context, isHTMX bool, message string) {
	if isHTMX {
		c.Data(http.StatusBadRequest, "text/html", []byte(`
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm">`+message+`</p>
			</div>
		`))
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}
