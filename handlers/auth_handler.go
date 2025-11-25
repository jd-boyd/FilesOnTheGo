// Package handlers provides HTTP request handlers for FilesOnTheGo.
// It includes handlers for file operations, authentication, and API endpoints.
package handlers

import (
	"net/http"
	"os"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	app      *pocketbase.PocketBase
	renderer *TemplateRenderer
	logger   zerolog.Logger
	config   *config.Config
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(app *pocketbase.PocketBase, renderer *TemplateRenderer, logger zerolog.Logger, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		app:      app,
		renderer: renderer,
		logger:   logger,
		config:   cfg,
	}
}

// ShowLoginPage renders the login page
func (h *AuthHandler) ShowLoginPage(c *core.RequestEvent) error {
	data := &TemplateData{
		Title:              "Login - FilesOnTheGo",
		PublicRegistration: h.config.PublicRegistration,
	}

	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response, "login", data)
}

// ShowRegisterPage renders the registration page
func (h *AuthHandler) ShowRegisterPage(c *core.RequestEvent) error {
	// Check if public registration is enabled
	if !h.config.PublicRegistration {
		h.logger.Warn().Msg("Registration page accessed when public registration is disabled")
		return c.Redirect(http.StatusFound, "/login")
	}

	data := &TemplateData{
		Title: "Register - FilesOnTheGo",
	}

	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response, "register", data)
}

// HandleLogin processes login requests
func (h *AuthHandler) HandleLogin(c *core.RequestEvent) error {
	isHTMX := IsHTMXRequest(c)

	// Get form data
	email := c.Request.FormValue("email")
	password := c.Request.FormValue("password")

	// Validate input
	if email == "" || password == "" {
		h.logger.Warn().
			Str("email", email).
			Msg("Login attempt with missing credentials")

		if isHTMX {
			return c.HTML(http.StatusBadRequest, `
				<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
					<p class="text-sm">Email and password are required</p>
				</div>
			`)
		}

		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email and password are required",
		})
	}

	// Attempt authentication using PocketBase
	collection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find users collection")
		return h.handleLoginError(c, isHTMX, "Authentication failed")
	}

	record, err := h.app.FindAuthRecordByEmail(collection, email)
	if err != nil {
		h.logger.Warn().
			Str("email", email).
			Msg("Login attempt with non-existent email")
		return h.handleLoginError(c, isHTMX, "Invalid email or password")
	}

	// Validate password
	if !record.ValidatePassword(password) {
		h.logger.Warn().
			Str("email", email).
			Str("user_id", record.Id).
			Msg("Login attempt with invalid password")
		return h.handleLoginError(c, isHTMX, "Invalid email or password")
	}

	// Use PocketBase's built-in authentication by setting the auth record in the request context
	// This will make PocketBase automatically handle the authentication session
	c.Set("authRecord", record)

	h.logger.Info().
		Str("email", email).
		Str("user_id", record.Id).
		Msg("User logged in successfully")

	// Set PocketBase auth cookie using the proper method
	// This ensures PocketBase recognizes the authentication
	c.SetCookie(&http.Cookie{
		Name:     "pb_auth",
		Value:    record.TokenKey(),
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to false for development with HTTP
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
	})

	// Redirect to dashboard
	if isHTMX {
		c.Response.Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/dashboard")
}

// HandleRegister processes registration requests
func (h *AuthHandler) HandleRegister(c *core.RequestEvent) error {
	isHTMX := IsHTMXRequest(c)

	// Check if public registration is enabled
	if !h.config.PublicRegistration {
		h.logger.Warn().Msg("Registration attempt when public registration is disabled")
		return h.handleRegisterError(c, isHTMX, "Public registration is currently disabled")
	}

	// Get form data
	email := c.Request.FormValue("email")
	username := c.Request.FormValue("username")
	password := c.Request.FormValue("password")
	passwordConfirm := c.Request.FormValue("passwordConfirm")

	// Validate input
	if email == "" || username == "" || password == "" {
		return h.handleRegisterError(c, isHTMX, "All fields are required")
	}

	if password != passwordConfirm {
		return h.handleRegisterError(c, isHTMX, "Passwords do not match")
	}

	// Get users collection
	collection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find users collection")
		return h.handleRegisterError(c, isHTMX, "Registration failed")
	}

	// Create new user record
	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("username", username)
	record.Set("emailVisibility", true)

	// Set password (SetPassword doesn't return an error)
	record.SetPassword(password)

	// Validate password confirmation
	if !record.ValidatePassword(password) {
		return h.handleRegisterError(c, isHTMX, "Password validation failed")
	}

	// Save the record
	if err := h.app.Save(record); err != nil {
		h.logger.Warn().
			Err(err).
			Str("email", email).
			Str("username", username).
			Msg("Registration failed")
		return h.handleRegisterError(c, isHTMX, "Registration failed: "+err.Error())
	}

	h.logger.Info().
		Str("email", email).
		Str("username", username).
		Str("user_id", record.Id).
		Msg("User registered successfully")

	// Auto-login after registration (simplified - using record token directly)
	token := record.TokenKey()

	// Set auth cookie
	c.SetCookie(&http.Cookie{
		Name:     "pb_auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
	})

	// Redirect to dashboard
	if isHTMX {
		c.Response.Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/dashboard")
}

// HandleLogout logs the user out
func (h *AuthHandler) HandleLogout(c *core.RequestEvent) error {
	// Clear auth cookie
	c.SetCookie(&http.Cookie{
		Name:     "pb_auth",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
	})

	h.logger.Info().Msg("User logged out")

	// Redirect to login
	if IsHTMXRequest(c) {
		c.Response.Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/login")
}

// ShowDashboard renders the dashboard page
func (h *AuthHandler) ShowDashboard(c *core.RequestEvent) error {
	data := PrepareTemplateData(c)
	data.Title = "Dashboard - FilesOnTheGo"
	data.StorageUsed = "0 MB"
	data.StorageQuota = "10 GB"
	data.StoragePercent = 0
	data.HasFiles = false

	// Check if user is admin to show admin link in header
	authRecord := c.Get("authRecord")
	if authRecord != nil {
		if record, ok := authRecord.(*core.Record); ok {
			isAdmin := h.isAdmin(record)
			data.Settings = map[string]interface{}{
				"IsAdmin": isAdmin,
			}
		}
	}

	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response, "dashboard", data)
}

// isAdmin checks if a user is an admin (duplicated from settings_handler for now)
func (h *AuthHandler) isAdmin(record *core.Record) bool {
	email := record.GetString("email")

	// Check if email matches admin email from environment
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail != "" && email == adminEmail {
		return true
	}

	// Check if there's an admin field on the user record
	if record.GetBool("is_admin") {
		return true
	}

	return false
}

// Helper methods for error handling

func (h *AuthHandler) handleLoginError(c *core.RequestEvent, isHTMX bool, message string) error {
	if isHTMX {
		return c.HTML(http.StatusUnauthorized, `
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm">`+message+`</p>
			</div>
		`)
	}

	return c.JSON(http.StatusUnauthorized, map[string]string{
		"error": message,
	})
}

func (h *AuthHandler) handleRegisterError(c *core.RequestEvent, isHTMX bool, message string) error {
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
