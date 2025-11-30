package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
)

// AdminHandler handles admin-related requests
type AdminHandler struct {
	userService *services.UserService
	renderer    *TemplateRenderer
	logger      zerolog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	userService *services.UserService,
	renderer *TemplateRenderer,
	logger zerolog.Logger,
) *AdminHandler {
	return &AdminHandler{
		userService: userService,
		renderer:    renderer,
		logger:      logger,
	}
}

// ShowAdminDashboard renders the admin dashboard
func (h *AdminHandler) ShowAdminDashboard(c *gin.Context) {
	data := PrepareTemplateData(c)
	data.Title = "Admin Dashboard - FilesOnTheGo"

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(c.Writer, "admin", data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render admin dashboard")
		c.String(http.StatusInternalServerError, "Internal server error")
	}
}

// ListUsers handles listing all users (admin only)
func (h *AdminHandler) ListUsers(c *gin.Context) {
	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	// Get users
	users, total, err := h.userService.ListUsers(perPage, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	// Calculate pagination info
	totalPages := (int(total) + perPage - 1) / perPage

	c.JSON(http.StatusOK, gin.H{
		"users":       users,
		"total":       total,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
	})
}

// GetUser handles retrieving a single user (admin only)
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// GetUserStats handles retrieving user statistics (admin only)
func (h *AdminHandler) GetUserStats(c *gin.Context) {
	userID := c.Param("id")

	stats, err := h.userService.GetUserStats(userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user stats")
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateUser handles creating a new user (admin only)
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
		IsAdmin  bool   `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.CreateUser(req.Email, req.Username, req.Password, req.IsAdmin)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create user")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := auth.GetUserID(c)
	h.logger.Info().
		Str("admin_id", adminID).
		Str("new_user_id", user.ID).
		Str("email", user.Email).
		Msg("User created by admin")

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// UpdateUser handles updating a user (admin only)
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.UpdateUser(userID, req)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update user")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := auth.GetUserID(c)
	h.logger.Info().
		Str("admin_id", adminID).
		Str("user_id", userID).
		Interface("updates", req).
		Msg("User updated by admin")

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// ResetUserPassword handles resetting a user's password (admin only)
func (h *AdminHandler) ResetUserPassword(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.ResetPassword(userID, req.NewPassword); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to reset password")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := auth.GetUserID(c)
	h.logger.Warn().
		Str("admin_id", adminID).
		Str("user_id", userID).
		Msg("Password reset by admin")

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// DeleteUser handles deleting a user (admin only)
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// Prevent admin from deleting themselves
	adminID, _ := auth.GetUserID(c)
	if adminID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	if err := h.userService.DeleteUser(userID); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to delete user")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Warn().
		Str("admin_id", adminID).
		Str("user_id", userID).
		Msg("User deleted by admin")

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// SearchUsers handles searching for users (admin only)
func (h *AdminHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	users, err := h.userService.SearchUsers(query, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("query", query).Msg("Failed to search users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}
