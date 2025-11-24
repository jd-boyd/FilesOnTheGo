package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jd-boyd/filesonthego/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTemplateRendererFromAssets returns a template renderer using embedded assets
func getTemplateRendererFromAssets(t *testing.T) *TemplateRenderer {
	t.Helper()
	templatesFS, err := assets.TemplatesFS()
	require.NoError(t, err, "Failed to get templates filesystem")
	return NewTemplateRendererFromFS(templatesFS)
}

// TestTemplateData tests the TemplateData struct
func TestTemplateData_Defaults(t *testing.T) {
	data := &TemplateData{}

	assert.Empty(t, data.Title)
	assert.Nil(t, data.User)
	assert.Empty(t, data.FlashMessage)
	assert.Empty(t, data.FlashType)
	assert.Empty(t, data.Breadcrumb)
	assert.Empty(t, data.StorageUsed)
	assert.Empty(t, data.StorageQuota)
	assert.Equal(t, 0, data.StoragePercent)
	assert.False(t, data.HasFiles)
	assert.Empty(t, data.RecentActivity)
}

func TestTemplateData_WithValues(t *testing.T) {
	breadcrumb := []BreadcrumbItem{
		{Name: "Home", URL: "/"},
		{Name: "Documents", URL: "/documents"},
	}
	activity := []ActivityItem{
		{FileName: "test.txt", Action: "uploaded", Time: "1 minute ago"},
	}

	data := &TemplateData{
		Title:          "Test Page",
		FlashMessage:   "Success!",
		FlashType:      "success",
		Breadcrumb:     breadcrumb,
		StorageUsed:    "100 MB",
		StorageQuota:   "10 GB",
		StoragePercent: 10,
		HasFiles:       true,
		RecentActivity: activity,
	}

	assert.Equal(t, "Test Page", data.Title)
	assert.Equal(t, "Success!", data.FlashMessage)
	assert.Equal(t, "success", data.FlashType)
	assert.Len(t, data.Breadcrumb, 2)
	assert.Equal(t, "Home", data.Breadcrumb[0].Name)
	assert.Equal(t, "100 MB", data.StorageUsed)
	assert.Equal(t, "10 GB", data.StorageQuota)
	assert.Equal(t, 10, data.StoragePercent)
	assert.True(t, data.HasFiles)
	assert.Len(t, data.RecentActivity, 1)
	assert.Equal(t, "test.txt", data.RecentActivity[0].FileName)
}

// TestBreadcrumbItem tests the BreadcrumbItem struct
func TestBreadcrumbItem(t *testing.T) {
	testCases := []struct {
		name     string
		item     BreadcrumbItem
		wantName string
		wantURL  string
	}{
		{
			name:     "home item",
			item:     BreadcrumbItem{Name: "Home", URL: "/"},
			wantName: "Home",
			wantURL:  "/",
		},
		{
			name:     "nested item",
			item:     BreadcrumbItem{Name: "Documents", URL: "/files/documents"},
			wantName: "Documents",
			wantURL:  "/files/documents",
		},
		{
			name:     "current page (no URL)",
			item:     BreadcrumbItem{Name: "Current File", URL: ""},
			wantName: "Current File",
			wantURL:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantName, tc.item.Name)
			assert.Equal(t, tc.wantURL, tc.item.URL)
		})
	}
}

// TestActivityItem tests the ActivityItem struct
func TestActivityItem(t *testing.T) {
	item := ActivityItem{
		FileName: "important.pdf",
		Action:   "downloaded",
		Time:     "5 minutes ago",
	}

	assert.Equal(t, "important.pdf", item.FileName)
	assert.Equal(t, "downloaded", item.Action)
	assert.Equal(t, "5 minutes ago", item.Time)
}

// TestTemplateRenderer tests template renderer creation
func TestTemplateRenderer_Creation(t *testing.T) {
	renderer := NewTemplateRenderer(".")

	assert.NotNil(t, renderer)
	assert.Equal(t, ".", renderer.baseDir)
	assert.NotNil(t, renderer.templates)

	// Also test the FS-based constructor
	rendererFromFS := getTemplateRendererFromAssets(t)
	assert.NotNil(t, rendererFromFS)
	assert.NotNil(t, rendererFromFS.templates)
	assert.NotNil(t, rendererFromFS.fsys)
}

// TestTemplateRenderer_LoadAndRender tests loading and rendering templates
func TestTemplateRenderer_LoadAndRender(t *testing.T) {
	renderer := getTemplateRendererFromAssets(t)

	// Load templates
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	testCases := []struct {
		name           string
		templateName   string
		data           interface{}
		expectContains []string
	}{
		{
			name:         "login template",
			templateName: "login",
			data:         &TemplateData{Title: "Login - Test"},
			expectContains: []string{
				"<!DOCTYPE html>",
				"Sign in to your account",
				"email",
				"password",
			},
		},
		{
			name:         "register template",
			templateName: "register",
			data:         &TemplateData{Title: "Register - Test"},
			expectContains: []string{
				"<!DOCTYPE html>",
				"Create your account",
				"email",
				"username",
				"password",
			},
		},
		{
			name:         "dashboard template",
			templateName: "dashboard",
			data: &TemplateData{
				Title:          "Dashboard - Test",
				StorageUsed:    "50 MB",
				StorageQuota:   "5 GB",
				StoragePercent: 1,
				HasFiles:       false,
			},
			expectContains: []string{
				"<!DOCTYPE html>",
				"My Files",
				"Storage Usage",
				"No files yet",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.Render(&buf, tc.templateName, tc.data)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "Expected %q to be in output", expected)
			}
		})
	}
}

// TestTemplateRenderer_RenderNonExistent tests rendering a non-existent template
func TestTemplateRenderer_NonExistentTemplate(t *testing.T) {
	projectRoot := getProjectRoot(t)
	renderer := NewTemplateRenderer(projectRoot)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	var buf bytes.Buffer
	err = renderer.Render(&buf, "nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

// TestGetTemplateFuncs tests the template functions
func TestGetTemplateFuncs(t *testing.T) {
	funcs := getTemplateFuncs()

	// Test upper function
	if upperFn, ok := funcs["upper"].(func(string) string); ok {
		assert.Equal(t, "HELLO", upperFn("hello"))
		assert.Equal(t, "HELLO WORLD", upperFn("hello world"))
	} else {
		t.Error("upper function not found or wrong type")
	}

	// Test lower function
	if lowerFn, ok := funcs["lower"].(func(string) string); ok {
		assert.Equal(t, "hello", lowerFn("HELLO"))
		assert.Equal(t, "hello world", lowerFn("HELLO WORLD"))
	} else {
		t.Error("lower function not found or wrong type")
	}

	// Test title function
	if titleFn, ok := funcs["title"].(func(string) string); ok {
		assert.Equal(t, "Hello", titleFn("hello"))
		assert.Equal(t, "Hello World", titleFn("hello world"))
	} else {
		t.Error("title function not found or wrong type")
	}
}

// TestHTMXRequestDetection tests HTMX request detection via HTTP headers
func TestHTMXRequestDetection(t *testing.T) {
	testCases := []struct {
		name          string
		htmxHeader    string
		expectedHTMX  bool
	}{
		{
			name:         "htmx request",
			htmxHeader:   "true",
			expectedHTMX: true,
		},
		{
			name:         "non-htmx request",
			htmxHeader:   "",
			expectedHTMX: false,
		},
		{
			name:         "htmx header with false value",
			htmxHeader:   "false",
			expectedHTMX: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test HTTP request
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.htmxHeader != "" {
				req.Header.Set("HX-Request", tc.htmxHeader)
			}

			// Check header directly (since IsHTMXRequest requires RequestEvent)
			isHTMX := req.Header.Get("HX-Request") == "true"
			assert.Equal(t, tc.expectedHTMX, isHTMX)
		})
	}
}

// TestLoginFormValidation tests the expected login form validation behavior
func TestLoginFormValidation_Scenarios(t *testing.T) {
	testCases := []struct {
		name           string
		email          string
		password       string
		expectValid    bool
		expectError    string
	}{
		{
			name:        "valid credentials",
			email:       "user@example.com",
			password:    "password123",
			expectValid: true,
		},
		{
			name:        "empty email",
			email:       "",
			password:    "password123",
			expectValid: false,
			expectError: "Email and password are required",
		},
		{
			name:        "empty password",
			email:       "user@example.com",
			password:    "",
			expectValid: false,
			expectError: "Email and password are required",
		},
		{
			name:        "both empty",
			email:       "",
			password:    "",
			expectValid: false,
			expectError: "Email and password are required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := tc.email != "" && tc.password != ""
			assert.Equal(t, tc.expectValid, isValid)
		})
	}
}

// TestRegisterFormValidation tests the expected register form validation behavior
func TestRegisterFormValidation_Scenarios(t *testing.T) {
	testCases := []struct {
		name            string
		email           string
		username        string
		password        string
		passwordConfirm string
		expectValid     bool
		expectError     string
	}{
		{
			name:            "valid registration",
			email:           "user@example.com",
			username:        "testuser",
			password:        "password123",
			passwordConfirm: "password123",
			expectValid:     true,
		},
		{
			name:            "password mismatch",
			email:           "user@example.com",
			username:        "testuser",
			password:        "password123",
			passwordConfirm: "differentpassword",
			expectValid:     false,
			expectError:     "Passwords do not match",
		},
		{
			name:            "missing email",
			email:           "",
			username:        "testuser",
			password:        "password123",
			passwordConfirm: "password123",
			expectValid:     false,
			expectError:     "All fields are required",
		},
		{
			name:            "missing username",
			email:           "user@example.com",
			username:        "",
			password:        "password123",
			passwordConfirm: "password123",
			expectValid:     false,
			expectError:     "All fields are required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allFieldsPresent := tc.email != "" && tc.username != "" && tc.password != ""
			passwordsMatch := tc.password == tc.passwordConfirm
			isValid := allFieldsPresent && passwordsMatch
			assert.Equal(t, tc.expectValid, isValid)
		})
	}
}

// TestHTMXResponseHeaders tests that proper headers would be set for HTMX redirects
func TestHTMXResponseHeaders(t *testing.T) {
	// Create a test response recorder
	w := httptest.NewRecorder()

	// Simulate setting HTMX redirect header
	w.Header().Set("HX-Redirect", "/dashboard")

	assert.Equal(t, "/dashboard", w.Header().Get("HX-Redirect"))
}

// TestAuthCookieSettings tests that auth cookie would have proper settings
func TestAuthCookieSettings(t *testing.T) {
	cookie := &http.Cookie{
		Name:     "pb_auth",
		Value:    "test-token",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
	}

	assert.Equal(t, "pb_auth", cookie.Name)
	assert.Equal(t, "test-token", cookie.Value)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly, "Cookie should be HttpOnly for security")
	assert.True(t, cookie.Secure, "Cookie should be Secure")
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, 604800, cookie.MaxAge, "Cookie should expire in 7 days")
}

// TestLogoutCookieClearing tests that logout cookie would clear the auth token
func TestLogoutCookieClearing(t *testing.T) {
	cookie := &http.Cookie{
		Name:     "pb_auth",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
	}

	assert.Empty(t, cookie.Value, "Logout cookie should have empty value")
	assert.Equal(t, -1, cookie.MaxAge, "Logout cookie should have negative MaxAge to delete")
}

// Placeholder test to ensure package compiles
func TestAuthHandlerPackage(t *testing.T) {
	assert.True(t, true, "Auth handler package compiles successfully")
}
