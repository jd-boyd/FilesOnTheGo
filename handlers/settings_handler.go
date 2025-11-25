// Package handlers provides HTTP request handlers for FilesOnTheGo.
package handlers

import (
	"net/http"
	"os"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
)

// SettingsHandler handles settings-related requests
type SettingsHandler struct {
	app      *pocketbase.PocketBase
	renderer *TemplateRenderer
	logger   zerolog.Logger
	config   *config.Config
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(app *pocketbase.PocketBase, renderer *TemplateRenderer, logger zerolog.Logger, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{
		app:      app,
		renderer: renderer,
		logger:   logger,
		config:   cfg,
	}
}

// ShowSettingsPage renders the settings page
func (h *SettingsHandler) ShowSettingsPage(c *core.RequestEvent) error {
	// Get authenticated user (middleware ensures this exists)
	authRecord := c.Get("authRecord")
	isAdmin := false

	// Check if user is admin
	if record, ok := authRecord.(*core.Record); ok {
		isAdmin = h.isAdmin(record)
	}

	data := PrepareTemplateData(c)
	data.Title = "Settings - FilesOnTheGo"
	data.PublicRegistration = h.config.PublicRegistration
	data.StorageUsed = "0 MB"
	data.StorageQuota = "10 GB"
	data.StoragePercent = 0

	// Add settings-specific data (only IsAdmin flag for personal settings)
	data.Settings = map[string]interface{}{
		"IsAdmin": isAdmin,
	}

	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response, "settings", data)
}

// HandleUpdateSettings processes settings update requests
func (h *SettingsHandler) HandleUpdateSettings(c *core.RequestEvent) error {
	isHTMX := IsHTMXRequest(c)

	// Check if user is authenticated
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Check if user is admin
	record, ok := authRecord.(*core.Record)
	if !ok {
		h.logger.Error().Msg("Failed to cast authRecord to *core.Record")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	if !h.isAdmin(record) {
		h.logger.Warn().
			Str("user_id", record.Id).
			Msg("Non-admin user attempted to update settings")
		return h.handleSettingsError(c, isHTMX, "Only administrators can update settings")
	}

	// Get form values
	publicRegistration := c.Request.FormValue("public_registration") == "on"
	emailVerification := c.Request.FormValue("email_verification") == "on"

	// Update environment variables (for current session)
	// Note: This updates the runtime config, but won't persist across restarts
	// For production, you'd want to update a config file or database
	os.Setenv("PUBLIC_REGISTRATION", boolToString(publicRegistration))
	os.Setenv("EMAIL_VERIFICATION", boolToString(emailVerification))

	// Update the config in memory
	h.config.PublicRegistration = publicRegistration
	h.config.EmailVerification = emailVerification

	h.logger.Info().
		Str("user_id", record.Id).
		Bool("public_registration", publicRegistration).
		Bool("email_verification", emailVerification).
		Msg("Settings updated by admin")

	// Return success message
	if isHTMX {
		return c.HTML(http.StatusOK, `
			<div class="bg-green-50 border border-green-200 text-green-800 rounded-md p-4">
				<p class="text-sm">Settings updated successfully! Changes are effective immediately.</p>
			</div>
		`)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Settings updated successfully",
	})
}

// isAdmin checks if a user is an admin
// For now, we check if the user's email matches the ADMIN_EMAIL environment variable
// or if they're the first user in the system
func (h *SettingsHandler) isAdmin(record *core.Record) bool {
	email := record.GetString("email")

	// Check if email matches admin email from environment
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail != "" && email == adminEmail {
		return true
	}

	// Check if there's an admin field on the user record
	// PocketBase collections can have custom fields
	if record.GetBool("is_admin") {
		return true
	}

	return false
}

// Helper methods

func (h *SettingsHandler) handleSettingsError(c *core.RequestEvent, isHTMX bool, message string) error {
	if isHTMX {
		return c.HTML(http.StatusBadRequest, `
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm">`+message+`</p>
			</div>
		`)
	}

	return c.JSON(http.StatusBadRequest, map[string]string{
		"error": message,
	})
}

