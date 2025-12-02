// Package handlers provides HTTP request handlers for FilesOnTheGo.
// It includes handlers for file operations, authentication, and API endpoints.
package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	db             *gorm.DB
	renderer       *TemplateRenderer
	logger         zerolog.Logger
	config         *config.Config
	jwtManager     *auth.JWTManager
	sessionManager *auth.SessionManager
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(
	db *gorm.DB,
	renderer *TemplateRenderer,
	logger zerolog.Logger,
	cfg *config.Config,
	jwtManager *auth.JWTManager,
	sessionManager *auth.SessionManager,
) *AuthHandler {
	return &AuthHandler{
		db:             db,
		renderer:       renderer,
		logger:         logger,
		config:         cfg,
		jwtManager:     jwtManager,
		sessionManager: sessionManager,
	}
}

// ShowLoginPage renders the login page
func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	data := &TemplateData{
		Title:              "Login - FilesOnTheGo",
		PublicRegistration: h.config.PublicRegistration,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(c.Writer, "login", data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render login page")
		c.String(http.StatusInternalServerError, "Internal server error")
	}
}

// ShowRegisterPage renders the registration page
func (h *AuthHandler) ShowRegisterPage(c *gin.Context) {
	// Check if public registration is enabled
	if !h.config.PublicRegistration {
		h.logger.Warn().Msg("Registration page accessed when public registration is disabled")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	data := &TemplateData{
		Title: "Register - FilesOnTheGo",
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(c.Writer, "register", data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render register page")
		c.String(http.StatusInternalServerError, "Internal server error")
	}
}

// HandleLogin processes login requests
func (h *AuthHandler) HandleLogin(c *gin.Context) {
	isHTMX := IsHTMXRequest(c)

	// Get form data
	email := c.PostForm("email")
	password := c.PostForm("password")

	// Validate input
	if email == "" || password == "" {
		h.logger.Warn().
			Str("email", email).
			Msg("Login attempt with missing credentials")

		if isHTMX {
			c.Data(http.StatusBadRequest, "text/html", []byte(`
				<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
					<p class="text-sm">Email and password are required</p>
				</div>
			`))
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email and password are required",
		})
		return
	}

	// Find user by email
	var user models.User
	if err := h.db.Where("email = ?", strings.ToLower(email)).First(&user).Error; err != nil {
		h.logger.Warn().
			Str("email", email).
			Msg("Login attempt with non-existent email")
		h.handleLoginError(c, isHTMX, "Invalid email or password")
		return
	}

	// Validate password
	if !user.ValidatePassword(password) {
		h.logger.Warn().
			Str("email", email).
			Str("user_id", user.ID).
			Msg("Login attempt with invalid password")
		h.handleLoginError(c, isHTMX, "Invalid email or password")
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.GenerateToken(&user)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate JWT token")
		h.handleLoginError(c, isHTMX, "Authentication failed")
		return
	}

	// Set session cookie
	h.sessionManager.SetSession(c, token)

	h.logger.Info().
		Str("email", email).
		Str("user_id", user.ID).
		Msg("User logged in successfully")

	// Redirect to dashboard
	if isHTMX {
		c.Header("HX-Redirect", "/dashboard")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/dashboard")
}

// HandleRegister processes registration requests
func (h *AuthHandler) HandleRegister(c *gin.Context) {
	isHTMX := IsHTMXRequest(c)

	// Check if public registration is enabled
	if !h.config.PublicRegistration {
		h.logger.Warn().Msg("Registration attempt when public registration is disabled")
		h.handleRegisterError(c, isHTMX, "Public registration is currently disabled")
		return
	}

	// Get form data
	email := c.PostForm("email")
	username := c.PostForm("username")
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("passwordConfirm")

	// Validate input
	if email == "" || username == "" || password == "" {
		h.handleRegisterError(c, isHTMX, "All fields are required")
		return
	}

	if password != passwordConfirm {
		h.handleRegisterError(c, isHTMX, "Passwords do not match")
		return
	}

	// Create new user
	user := &models.User{
		Email:           strings.ToLower(email),
		Username:        username,
		EmailVisibility: true,
	}

	// Set password
	if err := user.SetPassword(password); err != nil {
		h.logger.Error().Err(err).Msg("Failed to hash password")
		h.handleRegisterError(c, isHTMX, "Registration failed")
		return
	}

	// Save user to database
	if err := h.db.Create(user).Error; err != nil {
		h.logger.Warn().
			Err(err).
			Str("email", email).
			Str("username", username).
			Msg("Registration failed")

		errMsg := "Registration failed"
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			errMsg = "Email or username already exists"
		}
		h.handleRegisterError(c, isHTMX, errMsg)
		return
	}

	h.logger.Info().
		Str("email", email).
		Str("username", username).
		Str("user_id", user.ID).
		Msg("User registered successfully")

	// Generate JWT token for auto-login
	token, err := h.jwtManager.GenerateToken(user)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate JWT token after registration")
		// Don't fail registration, just redirect to login
		if isHTMX {
			c.Header("HX-Redirect", "/login")
			c.Status(http.StatusOK)
		} else {
			c.Redirect(http.StatusFound, "/login")
		}
		return
	}

	// Set session cookie
	h.sessionManager.SetSession(c, token)

	// Redirect to dashboard
	if isHTMX {
		c.Header("HX-Redirect", "/dashboard")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/dashboard")
}

// HandleLogout logs the user out
func (h *AuthHandler) HandleLogout(c *gin.Context) {
	// Clear session cookie
	h.sessionManager.ClearSession(c)

	h.logger.Info().Msg("User logged out")

	// Redirect to login
	if IsHTMXRequest(c) {
		c.Header("HX-Redirect", "/login")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

// ShowDashboard renders the dashboard page
func (h *AuthHandler) ShowDashboard(c *gin.Context) {
	data := PrepareTemplateData(c)
	data.Title = "Dashboard - FilesOnTheGo"

	// Get user from context
	userID, _ := auth.GetUserID(c)

	// Fetch user from database to get storage info
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err == nil {
		// Calculate storage display
		storageUsedMB := float64(user.StorageUsed) / (1024 * 1024)
		storageQuotaGB := float64(user.StorageQuota) / (1024 * 1024 * 1024)

		data.StorageUsed = formatBytes(user.StorageUsed)
		data.StorageQuota = formatBytes(user.StorageQuota)
		data.StoragePercent = int(user.GetQuotaUsagePercent())

		h.logger.Info().
			Float64("storage_used_mb", storageUsedMB).
			Float64("storage_quota_gb", storageQuotaGB).
			Int("storage_percent", data.StoragePercent).
			Msg("Dashboard storage info")

		// Check if user has files
		var fileCount int64
		h.db.Model(&models.File{}).Where("user = ?", userID).Count(&fileCount)
		data.HasFiles = fileCount > 0

		// Get admin status
		data.Settings = map[string]interface{}{
			"IsAdmin": user.IsAdmin,
		}
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(c.Writer, "dashboard", data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render dashboard")
		c.String(http.StatusInternalServerError, "Internal server error")
	}
}

// Helper methods for error handling

func (h *AuthHandler) handleLoginError(c *gin.Context, isHTMX bool, message string) {
	if isHTMX {
		c.Data(http.StatusUnauthorized, "text/html", []byte(`
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm">`+message+`</p>
			</div>
		`))
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{
		"error": message,
	})
}

func (h *AuthHandler) handleRegisterError(c *gin.Context, isHTMX bool, message string) {
	if isHTMX {
		c.Data(http.StatusBadRequest, "text/html", []byte(`
			<div class="bg-red-50 border border-red-200 text-red-800 rounded-md p-4">
				<p class="text-sm">`+message+`</p>
			</div>
		`))
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": message,
	})
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return "0 MB"
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"B", "KB", "MB", "GB", "TB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}
