package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jd-boyd/filesonthego/handlers"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShareUI_CreateShareHandler_InvalidMethod verifies non-POST requests are rejected
func TestShareUI_CreateShareHandler_InvalidMethod(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Test GET request
	req := httptest.NewRequest("GET", "/api/shares/create-htmx", nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusMethodNotAllowed, resp.Code)
}

// TestShareUI_CreateShareHandler_MissingAuth verifies unauthenticated requests are rejected
func TestShareUI_CreateShareHandler_MissingAuth(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {"test-id"},
		"permission_type": {"read"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

// TestShareUI_CreateShareHandler_MissingFields verifies validation of required fields
func TestShareUI_CreateShareHandler_MissingFields(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)

	// Test missing resource_type
	formData := url.Values{
		"resource_id":     {"test-id"},
		"permission_type": {"read"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid resource type")
}

// TestShareUI_CreateShareHandler_InvalidPermissionType verifies validation of permission types
func TestShareUI_CreateShareHandler_InvalidPermissionType(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"invalid"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid permission type")
}

// TestShareUI_CreateShareHandler_CreatesShare verifies successful share creation
func TestShareUI_CreateShareHandler_CreatesShare(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "share-link-display")
	assert.Contains(t, resp.Body.String(), "/s/")
}

// TestShareUI_CreateShareHandler_WithPassword verifies share creation with password protection
func TestShareUI_CreateShareHandler_WithPassword(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
		"password":        {"test123"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "PASSWORD")
}

// TestShareUI_CreateShareHandler_WithExpiration verifies share creation with expiration
func TestShareUI_CreateShareHandler_WithExpiration(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	futureTime := time.Now().Add(24 * time.Hour).Format("2006-01-02T15:04")
	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
		"expires_at":      {futureTime},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Expires")
}

// TestShareUI_CreateShareHandler_DirectoryShare verifies directory share creation
func TestShareUI_CreateShareHandler_DirectoryShare(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	dir := testApp.createTestDirectory(t, user.Id)

	formData := url.Values{
		"resource_type":   {"directory"},
		"resource_id":     {dir.Id},
		"permission_type": {"read_upload"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "READ & UPLOAD")
}

// TestShareUI_CreateShareHandler_NonExistentResource verifies handling of non-existent resources
func TestShareUI_CreateShareHandler_NonExistentResource(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {"non-existent-id"},
		"permission_type": {"read"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

// TestShareUI_CreateShareHandler_UnauthorizedResource verifies user cannot share others' resources
func TestShareUI_CreateShareHandler_UnauthorizedResource(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user1 := testApp.createTestUser(t)
	user2 := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user2.Id) // File belongs to user2

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user1.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
}

// TestShareUI_CreateShareHandler_ReadUploadPermission verifies read_upload permission share creation
func TestShareUI_CreateShareHandler_ReadUploadPermission(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	dir := testApp.createTestDirectory(t, user.Id)

	formData := url.Values{
		"resource_type":   {"directory"},
		"resource_id":     {dir.Id},
		"permission_type": {"read_upload"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "READ & UPLOAD")
}

// TestShareUI_CreateShareHandler_UploadOnlyPermission verifies upload_only permission share creation
func TestShareUI_CreateShareHandler_UploadOnlyPermission(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	dir := testApp.createTestDirectory(t, user.Id)

	formData := url.Values{
		"resource_type":   {"directory"},
		"resource_id":     {dir.Id},
		"permission_type": {"upload_only"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "UPLOAD-ONLY")
}

// TestShareUI_ListSharesHandler_ValidRequest verifies successful shares listing
func TestShareUI_ListSharesHandler_ValidRequest(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create a test share
	shareService := services.NewShareService(testApp.app)
	_, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares/list-htmx", nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "share-list-item")
}

// TestShareUI_ListSharesHandler_Unauthenticated verifies unauthenticated requests are rejected
func TestShareUI_ListSharesHandler_Unauthenticated(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	req := httptest.NewRequest("GET", "/api/shares/list-htmx", nil)
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

// TestShareUI_ListSharesHandler_Filters verifies filtering functionality
func TestShareUI_ListSharesHandler_Filters(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)
	dir := testApp.createTestDirectory(t, user.Id)

	shareService := services.NewShareService(testApp.app)

	// Create file share
	_, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Create directory share
	_, err = shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "directory",
		ResourceID:     dir.Id,
		PermissionType: "read_upload",
	})
	require.NoError(t, err)

	// Test filter by resource_type
	req := httptest.NewRequest("GET", "/api/shares/list-htmx?resource_type=file", nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "share-list-item")
}

// TestShareUI_ListSharesHandler_Search verifies search functionality
func TestShareUI_ListSharesHandler_Search(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)
	_, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares/list-htmx?search="+file.Name, nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

// TestShareUI_GetShareLogsHandler_ValidRequest verifies access logs retrieval
func TestShareUI_GetShareLogsHandler_ValidRequest(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares/"+share.ID+"/logs-htmx", nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

// TestShareUI_GetShareLogsHandler_UnauthorizedShare verifies access control
func TestShareUI_GetShareLogsHandler_UnauthorizedShare(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user1 := testApp.createTestUser(t)
	user2 := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user2.Id)

	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user2.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares/"+share.ID+"/logs-htmx", nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user1.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

// TestShareUI_DeleteShareHandler_ValidRequest verifies share revocation
func TestShareUI_DeleteShareHandler_ValidRequest(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "share-revoked")
}

// TestShareUI_DeleteShareHandler_Unauthorized verifies access control
func TestShareUI_DeleteShareHandler_Unauthorized(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user1 := testApp.createTestUser(t)
	user2 := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user2.Id)

	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user2.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user1.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

// TestShareUI_GetResourceSharesHandler_ValidRequest verifies retrieving shares for a resource
func TestShareUI_GetResourceSharesHandler_ValidRequest(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)
	_, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares/resource/file/"+file.Id, nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "shares-list")
}

// TestShareUI_CreateShareHandler_PastExpiration verifies rejection of past expiration dates
func TestShareUI_CreateShareHandler_PastExpiration(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	pastTime := time.Now().Add(-24 * time.Hour).Format("2006-01-02T15:04")
	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
		"expires_at":      {pastTime},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

// ============================================
// Test Setup and Utilities
// ============================================

type shareUITestApp struct {
	app *pocketbase.PocketBase
}

func setupShareUITest(t *testing.T) *shareUITestApp {
	t.Helper()

	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)

	app := &pocketbase.PocketBase{}

	// Create temporary directory for test data
	tempDir := t.TempDir()

	// Initialize the app with test configuration
	err := app.InitRoot(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize PocketBase root: %v", err)
	}

	// Create test configuration file
	os.MkdirAll(tempDir+"/pb_data", 0755)

	// Initialize the app
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap PocketBase: %v", err)
	}

	// Register the share handler
	shareHandler := handlers.NewShareHandler(app)
	app.OnServe().Bind(&core.ServeEvent{
		Route: func(e *core.ServeEvent) error {
			e.Router.POST("/api/shares/create-htmx", shareHandler.CreateShareHTMX)
			e.Router.GET("/api/shares/list-htmx", shareHandler.ListSharesHTMX)
			e.Router.GET("/api/shares/:id/logs-htmx", shareHandler.GetShareLogsHTMX)
			e.Router.DELETE("/api/shares/:id", shareHandler.DeleteShareHTMX)
			e.Router.GET("/api/shares/resource/:resourceType/:resourceId", shareHandler.GetResourceSharesHTMX)
			return nil
		},
	})

	testApp := &shareUITestApp{app: app}

	// Clean up after test
	t.Cleanup(func() {
		app.ResetBootstrapState()
	})

	return testApp
}

func (ta *shareUITestApp) createTestUser(t *testing.T) *core.Record {
	t.Helper()

	collection, err := ta.app.FindCollectionByNameOrId("users")
	require.NoError(t, err)

	user := core.NewRecord(collection)
	user.Set("email", "testuser-"+time.Now().Format("20060102150405")+"@example.com")
	user.Set("password", "password123")
	user.Set("verified", true)

	err = ta.app.Save(user)
	require.NoError(t, err)

	return user
}

func (ta *shareUITestApp) createTestFile(t *testing.T, userID string) *core.Record {
	t.Helper()

	collection, err := ta.app.FindCollectionByNameOrId("files")
	require.NoError(t, err)

	file := core.NewRecord(collection)
	file.Set("name", "test-file-"+time.Now().Format("20060102150405")+".txt")
	file.Set("type", "text/plain")
	file.Set("size", 1024)
	file.Set("path", "/test/path/"+file.Get("name"))
	file.Set("owner", userID)

	err = ta.app.Save(file)
	require.NoError(t, err)

	return file
}

func (ta *shareUITestApp) createTestDirectory(t *testing.T, userID string) *core.Record {
	t.Helper()

	collection, err := ta.app.FindCollectionByNameOrId("directories")
	require.NoError(t, err)

	dir := core.NewRecord(collection)
	dir.Set("name", "test-dir-"+time.Now().Format("20060102150405"))
	dir.Set("parent", "")
	dir.Set("owner", userID)

	err = ta.app.Save(dir)
	require.NoError(t, err)

	return dir
}

func (ta *shareUITestApp) authenticateUser(email, password string) string {
	// This would typically use the actual authentication mechanism
	// For testing purposes, return a mock token
	return "mock-jwt-token-for-" + email
}

// TestShareUI_CreateShareHandler_ReadPermission verifies read permission share creation
func TestShareUI_CreateShareHandler_ReadPermission(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "READ-ONLY")
}

func TestShareUI_CreateShareHandler_CustomShareURL(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
		"custom_url":      {"my-custom-share"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/s/my-custom-share")
}

func TestShareUI_CreateShareHandler_CustomShareURL_InvalidFormat(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	formData := url.Values{
		"resource_type":   {"file"},
		"resource_id":     {file.Id},
		"permission_type": {"read"},
		"custom_url":      {"Invalid URL with spaces!"},
	}

	req := httptest.NewRequest("POST", "/api/shares/create-htmx", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestShareUI_ListSharesHandler_CustomShareURL(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create a test share with custom URL
	shareService := services.NewShareService(testApp.app)
	_, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		CustomShareURL: "my-test-share",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares/list-htmx", nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "my-test-share")
}

func TestShareUI_DeleteShareHandler_CustomShareURL(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		CustomShareURL: "my-delete-test",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	req.Header.Set("Authorization", "Bearer "+testApp.authenticateUser(user.Email, "password123"))
	req.Header.Set("HX-Request", "true")

	resp := httptest.NewRecorder()
	testApp.app.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "share-revoked")
}

func TestShareUI_ShareAccessCount_Increments(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)

	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Initial count should be 0
	assert.Equal(t, int64(0), share.AccessCount)

	// Increment access count
	err = shareService.IncrementAccessCount(share.ID)
	assert.NoError(t, err)

	// Fetch updated share
	updatedShare, err := shareService.GetShareByID(share.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updatedShare.AccessCount)
}

func TestShareUI_ValidateShareAccess_WithPassword(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)
	password := "secret123"

	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		Password:       password,
	})
	require.NoError(t, err)

	// Access without password should fail
	accessInfo, err := shareService.ValidateShareAccess(share.ShareToken, "")
	assert.NoError(t, err)
	assert.False(t, accessInfo.IsValid)
	assert.Equal(t, "Password required", accessInfo.ErrorMessage)

	// Access with wrong password should fail
	accessInfo, err = shareService.ValidateShareAccess(share.ShareToken, "wrongpassword")
	assert.NoError(t, err)
	assert.False(t, accessInfo.IsValid)
	assert.Equal(t, "Invalid password", accessInfo.ErrorMessage)

	// Access with correct password should succeed
	accessInfo, err = shareService.ValidateShareAccess(share.ShareToken, password)
	assert.NoError(t, err)
	assert.True(t, accessInfo.IsValid)
}

func TestShareUI_UpdateShareExpiration_Works(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	shareService := services.NewShareService(testApp.app)

	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Should have no expiration initially
	assert.Nil(t, share.ExpiresAt)

	// Update with expiration
	newExpiration := time.Now().Add(7 * 24 * time.Hour)
	err = shareService.UpdateShareExpiration(share.ID, &newExpiration)
	assert.NoError(t, err)

	// Fetch updated share
	updatedShare, err := shareService.GetShareByID(share.ID)
	require.NoError(t, err)
	assert.NotNil(t, updatedShare.ExpiresAt)
	assert.WithinDuration(t, newExpiration, *updatedShare.ExpiresAt, time.Minute)
}

func TestShareUI_CreateShareHandler_PastExpiration_Rejected(t *testing.T) {
	testApp := setupShareUITest(t)

	if _, err := testApp.app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	pastTime := time.Now().Add(-1 * time.Hour)

	shareService := services.NewShareService(testApp.app)
	_, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		ExpiresAt:      &pastTime,
	})

	// Should fail because expiration is in the past
	assert.Error(t, err)
}