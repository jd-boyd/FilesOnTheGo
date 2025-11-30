// Package handlers provides HTTP request handlers for FilesOnTheGo.
package handlers

import (
	"fmt"
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
		email := record.GetString("email")
		isAdminField := record.GetBool("is_admin")
		h.logger.Debug().
			Str("user_email", email).
			Bool("is_admin_field", isAdminField).
			Msg("Settings page: Checking admin status for user")

		isAdmin = h.isAdmin(record)
		h.logger.Info().
			Str("user_email", email).
			Bool("is_admin_result", isAdmin).
			Msg("Settings page: Admin status determined")
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

	h.logger.Debug().
		Bool("settings_is_admin", isAdmin).
		Msg("Settings page: Setting IsAdmin flag in template data")

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

// ShowProfilePage renders the profile edit page
func (h *SettingsHandler) ShowProfilePage(c *core.RequestEvent) error {
	// Get authenticated user (middleware ensures this exists)
	authRecord := c.Get("authRecord")

	// Check if authRecord exists and is valid
	if authRecord == nil {
		h.logger.Warn().Msg("Profile page: No authRecord found in request context")
		return c.String(http.StatusInternalServerError, "Authentication error: Please log in again")
	}

	// Handle both cases: when authRecord is a real *core.Record or our placeholder map
	var email string = "Unknown User"
	var isAdmin bool = false

	switch auth := authRecord.(type) {
	case *core.Record:
		// Real PocketBase record
		func() {
			defer func() {
				if r := recover(); r != nil {
					h.logger.Warn().Interface("panic", r).Msg("Profile page: Panic when accessing user fields, database may not be initialized")
				}
			}()

			email = auth.GetString("email")
			isAdminField := auth.GetBool("is_admin")
			h.logger.Debug().
				Str("user_email", email).
				Bool("is_admin_field", isAdminField).
				Msg("Profile page: Checking admin status for user")

			isAdmin = h.isAdmin(auth)
			h.logger.Info().
				Str("user_email", email).
				Bool("is_admin_result", isAdmin).
				Msg("Profile page: Admin status determined")
		}()
	case map[string]interface{}:
		// Placeholder auth from middleware
		var ok bool
		if emailInterface, exists := auth["email"]; exists {
			if email, ok = emailInterface.(string); ok {
				h.logger.Debug().
					Str("user_email", email).
					Msg("Profile page: Using placeholder auth record")

				// For placeholder auth, check if it's the admin email
				adminEmail := os.Getenv("ADMIN_EMAIL")
				if adminEmail == "" {
					adminEmail = "admin@filesonthego.local" // fallback
				}
				isAdmin = (email == adminEmail)

				h.logger.Info().
					Str("user_email", email).
					Bool("is_admin_result", isAdmin).
					Msg("Profile page: Admin status determined from placeholder")
			} else {
				h.logger.Warn().Msg("Profile page: Placeholder auth email is not a string")
			}
		} else {
			h.logger.Warn().Msg("Profile page: Placeholder auth has no email field")
		}
	default:
		h.logger.Error().
			Interface("authRecord_type", fmt.Sprintf("%T", authRecord)).
			Msg("Profile page: Unexpected authRecord type")
		return c.String(http.StatusInternalServerError, "Authentication error: Invalid user session")
	}

	data := PrepareTemplateData(c)
	data.Title = "Edit Profile - FilesOnTheGo"
	data.PublicRegistration = h.config.PublicRegistration

	// Add profile-specific data
	data.Settings = map[string]interface{}{
		"IsAdmin": isAdmin,
	}

	h.logger.Debug().
		Bool("profile_is_admin", isAdmin).
		Msg("Profile page: Setting IsAdmin flag in template data")

	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Try to render template, catch any rendering errors
	err := h.renderer.Render(c.Response, "profile", data)
	if err != nil {
		h.logger.Error().Err(err).Msg("Profile page: Failed to render template")
		return c.String(http.StatusInternalServerError, "Error loading profile page: "+err.Error())
	}

	return nil
}

// HandleUpdateProfile processes profile update requests
func (h *SettingsHandler) HandleUpdateProfile(c *core.RequestEvent) error {
	isHTMX := IsHTMXRequest(c)

	// Check if user is authenticated
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Handle both cases: when authRecord is a real *core.Record or our placeholder map
	var record *core.Record
	var userID string

	switch auth := authRecord.(type) {
	case *core.Record:
		// Real PocketBase record - we can update it directly
		record = auth
		userID = auth.Id
	case map[string]interface{}:
		// Placeholder auth - we can't update records without proper database access
		h.logger.Warn().Msg("Profile update attempted with placeholder auth - database not properly initialized")
		message := "Profile editing is currently unavailable due to database initialization issues. Please try again later."
		if isHTMX {
			return c.HTML(http.StatusServiceUnavailable, `
				<div class="bg-yellow-50 border border-yellow-200 text-yellow-800 rounded-md p-4">
					<p class="text-sm">`+message+`</p>
				</div>
			`)
		}
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": message,
		})
	default:
		h.logger.Error().Msg("Failed to cast authRecord to *core.Record")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// Get form values
	displayName := c.Request.FormValue("displayName")
	if displayName == "" {
		displayName = "User" // Default display name
	}

	// Validate display name length
	if len(displayName) < 2 || len(displayName) > 100 {
		message := "Display name must be between 2 and 100 characters"
		if isHTMX {
			return c.HTML(http.StatusOK, `
				<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
					<p class="text-sm">`+message+`</p>
				</div>
			`)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": message,
		})
	}

	// Update the user record
	record.Set("name", displayName)

	// Save the changes
	if err := h.app.Save(record); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update user profile")
		message := "Failed to update profile"
		if isHTMX {
			return c.HTML(http.StatusInternalServerError, `
				<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
					<p class="text-sm">`+message+`</p>
				</div>
			`)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": message,
		})
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("display_name", displayName).
		Msg("User profile updated successfully")

	// Return success message
	message := "Profile updated successfully!"
	if isHTMX {
		// Reload the form with updated data
		return c.HTML(http.StatusOK, `
			<div class="bg-green-50 border border-green-200 text-green-800 rounded-md p-4">
				<p class="text-sm">`+message+`</p>
			</div>
			<script>
				// Update the display name in the navigation menu
				setTimeout(() => {
					const userMenuButton = document.querySelector('button:has(div[class*="rounded-full"])');
					if (userMenuButton) {
						// This would need more specific implementation based on the actual DOM structure
						location.reload();
					}
				}, 500);
			</script>
		`)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": message,
	})
}

// isAdmin checks if a user is an admin
// For now, we check if the user's email matches the ADMIN_EMAIL environment variable
// or if they're the first user in the system
func (h *SettingsHandler) isAdmin(record *core.Record) bool {
	email := record.GetString("email")
	isAdminField := record.GetBool("is_admin")

	// Check if email matches admin email from environment
	adminEmail := os.Getenv("ADMIN_EMAIL")
	emailMatch := adminEmail != "" && email == adminEmail

	h.logger.Debug().
		Str("user_email", email).
		Str("admin_email_env", adminEmail).
		Bool("email_match", emailMatch).
		Bool("is_admin_field", isAdminField).
		Msg("Settings handler: Admin check details")

	if emailMatch {
		h.logger.Info().
			Str("user_email", email).
			Msg("Settings handler: User is admin by email match")
		return true
	}

	// Check if there's an admin field on the user record
	// PocketBase collections can have custom fields
	if isAdminField {
		h.logger.Info().
			Str("user_email", email).
			Msg("Settings handler: User is admin by field flag")
		return true
	}

	h.logger.Info().
		Str("user_email", email).
		Msg("Settings handler: User is not admin")
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

