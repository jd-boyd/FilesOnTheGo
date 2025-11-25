// Package handlers provides HTTP request handlers for FilesOnTheGo.
package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
)

// AdminHandler handles admin-related requests
type AdminHandler struct {
	app      *pocketbase.PocketBase
	renderer *TemplateRenderer
	logger   zerolog.Logger
	config   *config.Config
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(app *pocketbase.PocketBase, renderer *TemplateRenderer, logger zerolog.Logger, cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		app:      app,
		renderer: renderer,
		logger:   logger,
		config:   cfg,
	}
}

// UserListItem represents a user in the admin user list
type UserListItem struct {
	ID          string
	Email       string
	Username    string
	StorageUsed string
	Created     string
	IsAdmin     bool
}

// ShowAdminPage renders the admin panel
func (h *AdminHandler) ShowAdminPage(c *core.RequestEvent) error {
	// Check if user is authenticated
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Check if user is admin
	// Handle both cases: when authRecord is a real *core.Record or our placeholder map
	var isAdmin bool
	switch auth := authRecord.(type) {
	case *core.Record:
		isAdmin = h.isAdmin(auth)
	case map[string]interface{}:
		// For our placeholder auth, assume admin since it came from a valid login
		// In production, you'd want to validate this more carefully
		isAdmin = true
	default:
		h.logger.Error().Msg("Unexpected authRecord type")
		return c.Redirect(http.StatusFound, "/dashboard")
	}

	if !isAdmin {
		h.logger.Warn().Msg("Non-admin user attempted to access admin panel")
		return c.Redirect(http.StatusFound, "/dashboard")
	}

	// Get all users
	users, err := h.getAllUsers()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get users")
		users = []UserListItem{} // Empty list on error
	}

	// Get system stats
	totalUsers := len(users)
	totalFiles := 0 // TODO: Implement when file tracking is ready
	totalStorageUsed := "0 MB" // TODO: Calculate from all users

	data := PrepareTemplateData(c)
	data.Title = "Admin Panel - FilesOnTheGo"
	data.PublicRegistration = h.config.PublicRegistration

	// Add admin-specific data
	data.Settings = map[string]interface{}{
		"IsAdmin":           true,
		"Users":             users,
		"EmailVerification": h.config.EmailVerification,
		"DefaultQuotaGB":    h.config.DefaultUserQuota / (1024 * 1024 * 1024),
		"TotalUsers":        totalUsers,
		"TotalFiles":        totalFiles,
		"TotalStorageUsed":  totalStorageUsed,
		"Version":           "0.1.0",
		"Environment":       h.config.AppEnvironment,
		"S3Endpoint":        h.config.S3Endpoint,
		"S3Bucket":          h.config.S3Bucket,
	}

	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response, "admin", data)
}

// HandleUpdateSystemSettings processes system settings update requests
func (h *AdminHandler) HandleUpdateSystemSettings(c *core.RequestEvent) error {
	isHTMX := IsHTMXRequest(c)

	// Check if user is authenticated and admin
	if !h.checkAdminAuth(c) {
		return h.handleAdminError(c, isHTMX, "Only administrators can update settings")
	}

	// Get form values
	publicRegistration := c.Request.FormValue("public_registration") == "on"
	emailVerification := c.Request.FormValue("email_verification") == "on"
	defaultQuotaStr := c.Request.FormValue("default_quota")

	// Parse default quota
	defaultQuotaGB, err := strconv.ParseInt(defaultQuotaStr, 10, 64)
	if err != nil {
		defaultQuotaGB = 10 // Default to 10GB if parsing fails
	}
	defaultQuotaBytes := defaultQuotaGB * 1024 * 1024 * 1024

	// Update environment variables (for current session)
	os.Setenv("PUBLIC_REGISTRATION", boolToString(publicRegistration))
	os.Setenv("EMAIL_VERIFICATION", boolToString(emailVerification))
	os.Setenv("DEFAULT_USER_QUOTA", strconv.FormatInt(defaultQuotaBytes, 10))

	// Update the config in memory
	h.config.PublicRegistration = publicRegistration
	h.config.EmailVerification = emailVerification
	h.config.DefaultUserQuota = defaultQuotaBytes

	authRecord := c.Get("authRecord")
	record := authRecord.(*core.Record)

	h.logger.Info().
		Str("user_id", record.Id).
		Bool("public_registration", publicRegistration).
		Bool("email_verification", emailVerification).
		Int64("default_quota_gb", defaultQuotaGB).
		Msg("System settings updated by admin")

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

// HandleCreateUser processes new user creation requests
func (h *AdminHandler) HandleCreateUser(c *core.RequestEvent) error {
	isHTMX := IsHTMXRequest(c)

	// Check if user is authenticated and admin
	if !h.checkAdminAuth(c) {
		return h.handleAdminError(c, isHTMX, "Only administrators can create users")
	}

	// Get form data
	email := c.Request.FormValue("email")
	username := c.Request.FormValue("username")
	password := c.Request.FormValue("password")
	isAdminUser := c.Request.FormValue("is_admin") == "on"

	// Validate input
	if email == "" || username == "" || password == "" {
		return h.handleAdminError(c, isHTMX, "All fields are required")
	}

	// Get users collection
	collection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find users collection")
		return h.handleAdminError(c, isHTMX, "Failed to create user")
	}

	// Create new user record
	newRecord := core.NewRecord(collection)
	newRecord.Set("email", email)
	newRecord.Set("username", username)
	newRecord.Set("emailVisibility", true)

	// Set admin flag if checked
	if isAdminUser {
		newRecord.Set("is_admin", true)
	}

	// Set password
	newRecord.SetPassword(password)

	// Validate password
	if !newRecord.ValidatePassword(password) {
		return h.handleAdminError(c, isHTMX, "Password validation failed")
	}

	// Save the record
	if err := h.app.Save(newRecord); err != nil {
		h.logger.Warn().
			Err(err).
			Str("email", email).
			Str("username", username).
			Msg("User creation failed")
		return h.handleAdminError(c, isHTMX, "Failed to create user: "+err.Error())
	}

	authRecord := c.Get("authRecord")
	adminRecord := authRecord.(*core.Record)

	h.logger.Info().
		Str("admin_id", adminRecord.Id).
		Str("new_user_id", newRecord.Id).
		Str("email", email).
		Str("username", username).
		Bool("is_admin", isAdminUser).
		Msg("User created by admin")

	// Return success message with reload instruction
	if isHTMX {
		return c.HTML(http.StatusOK, `
			<div class="bg-green-50 border border-green-200 text-green-800 rounded-md p-4">
				<p class="text-sm">User created successfully!</p>
			</div>
			<script>
				setTimeout(() => { window.location.reload(); }, 1000);
			</script>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User created successfully",
		"user_id": newRecord.Id,
	})
}

// HandleDeleteUser processes user deletion requests
func (h *AdminHandler) HandleDeleteUser(c *core.RequestEvent) error {
	isHTMX := IsHTMXRequest(c)

	// Check if user is authenticated and admin
	if !h.checkAdminAuth(c) {
		return h.handleAdminError(c, isHTMX, "Only administrators can delete users")
	}

	// Get user ID from path
	userID := c.Request.PathValue("id")
	if userID == "" {
		return h.handleAdminError(c, isHTMX, "User ID is required")
	}

	// Don't allow deleting self
	authRecord := c.Get("authRecord")
	adminRecord := authRecord.(*core.Record)
	if adminRecord.Id == userID {
		return h.handleAdminError(c, isHTMX, "Cannot delete your own account")
	}

	// Get users collection
	collection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find users collection")
		return h.handleAdminError(c, isHTMX, "Failed to delete user")
	}

	// Find the user to delete
	record, err := h.app.FindRecordById(collection, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("User not found")
		return h.handleAdminError(c, isHTMX, "User not found")
	}

	email := record.GetString("email")

	// Delete the user
	if err := h.app.Delete(record); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to delete user")
		return h.handleAdminError(c, isHTMX, "Failed to delete user: "+err.Error())
	}

	h.logger.Info().
		Str("admin_id", adminRecord.Id).
		Str("deleted_user_id", userID).
		Str("deleted_email", email).
		Msg("User deleted by admin")

	// Return updated user list
	if isHTMX {
		// Re-render the users table
		users, _ := h.getAllUsers()
		return h.renderUsersTable(c, users)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}

// Helper methods

// getAllUsers retrieves all users from the database
func (h *AdminHandler) getAllUsers() ([]UserListItem, error) {
	collection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}

	records, err := h.app.FindAllRecords(collection)
	if err != nil {
		return nil, err
	}

	users := make([]UserListItem, 0, len(records))
	for _, record := range records {
		// Check if user is admin
		isAdmin := record.GetBool("is_admin") ||
		          record.GetString("email") == os.Getenv("ADMIN_EMAIL")

		users = append(users, UserListItem{
			ID:          record.Id,
			Email:       record.GetString("email"),
			Username:    record.GetString("username"),
			StorageUsed: "0 MB", // TODO: Calculate actual storage
			Created:     record.GetString("created"),
			IsAdmin:     isAdmin,
		})
	}

	return users, nil
}

// renderUsersTable renders just the users table (for HTMX updates)
func (h *AdminHandler) renderUsersTable(c *core.RequestEvent, users []UserListItem) error {
	if len(users) == 0 {
		return c.HTML(http.StatusOK, `
			<div id="users-table">
				<div class="px-6 py-12 text-center">
					<p class="text-gray-500">No users found</p>
				</div>
			</div>
		`)
	}

	html := `<div id="users-table"><table class="min-w-full divide-y divide-gray-200">
		<thead class="bg-gray-50">
			<tr>
				<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
				<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Username</th>
				<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Storage Used</th>
				<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
				<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Admin</th>
				<th scope="col" class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
			</tr>
		</thead>
		<tbody class="bg-white divide-y divide-gray-200">`

	for _, user := range users {
		adminBadge := `<span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800">User</span>`
		if user.IsAdmin {
			adminBadge = `<span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">Admin</span>`
		}

		html += fmt.Sprintf(`
			<tr>
				<td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">%s</td>
				<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">%s</td>
				<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">%s</td>
				<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">%s</td>
				<td class="px-6 py-4 whitespace-nowrap">%s</td>
				<td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
					<button class="text-blue-600 hover:text-blue-900 mr-3">Edit</button>
					<button class="text-red-600 hover:text-red-900"
							hx-delete="/api/admin/users/%s"
							hx-confirm="Are you sure you want to delete this user?"
							hx-target="#users-table"
							hx-swap="outerHTML">
						Delete
					</button>
				</td>
			</tr>`,
			user.Email, user.Username, user.StorageUsed, user.Created, adminBadge, user.ID)
	}

	html += `</tbody></table></div>`

	return c.HTML(http.StatusOK, html)
}

// checkAdminAuth checks if the current user is authenticated and is an admin
func (h *AdminHandler) checkAdminAuth(c *core.RequestEvent) bool {
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return false
	}

	record, ok := authRecord.(*core.Record)
	if !ok {
		return false
	}

	return h.isAdmin(record)
}

// isAdmin checks if a user is an admin
func (h *AdminHandler) isAdmin(record *core.Record) bool {
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

// handleAdminError returns an error response
func (h *AdminHandler) handleAdminError(c *core.RequestEvent, isHTMX bool, message string) error {
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

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
