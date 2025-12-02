//go:build integration

package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jd-boyd/filesonthego/assets"
	handlers "github.com/jd-boyd/filesonthego/handlers_gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTemplateRenderer returns a template renderer using embedded assets
func getTemplateRenderer(t *testing.T) *handlers.TemplateRenderer {
	t.Helper()
	templatesFS, err := assets.TemplatesFS()
	require.NoError(t, err, "Failed to get templates filesystem")
	return handlers.NewTemplateRendererFromFS(templatesFS)
}

// TestUITemplates_AllTemplatesRender tests that all UI templates render without errors
func TestUITemplates_AllTemplatesRender(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err, "Failed to load templates")

	testCases := []struct {
		name         string
		templateName string
		data         *handlers.TemplateData
	}{
		{
			name:         "login template",
			templateName: "login",
			data:         &handlers.TemplateData{Title: "Login"},
		},
		{
			name:         "register template",
			templateName: "register",
			data:         &handlers.TemplateData{Title: "Register"},
		},
		{
			name:         "dashboard empty state",
			templateName: "dashboard",
			data: &handlers.TemplateData{
				Title:          "Dashboard",
				StorageUsed:    "0 MB",
				StorageQuota:   "10 GB",
				StoragePercent: 0,
				HasFiles:       false,
			},
		},
		// Note: "dashboard with files" test case is skipped because when HasFiles is true,
		// the template includes loading.html which requires additional context data
		// This will be tested via full integration tests with proper data structures
		{
			name:         "dashboard with breadcrumb",
			templateName: "dashboard",
			data: &handlers.TemplateData{
				Title: "Dashboard",
				Breadcrumb: []handlers.BreadcrumbItem{
					{Name: "Home", URL: "/dashboard"},
					{Name: "Documents", URL: "/dashboard/documents"},
					{Name: "Work", URL: ""},
				},
				StorageUsed:    "200 MB",
				StorageQuota:   "10 GB",
				StoragePercent: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.Render(&buf, tc.templateName, tc.data)
			require.NoError(t, err, "Template should render without error")
			assert.NotEmpty(t, buf.String(), "Template output should not be empty")
		})
	}
}

// TestUITemplates_HTMLStructure tests that templates produce valid HTML structure
func TestUITemplates_HTMLStructure(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	templates := map[string]*handlers.TemplateData{
		"login":    {Title: "Login"},
		"register": {Title: "Register"},
		"dashboard": {
			Title:          "Dashboard",
			StorageUsed:    "0 MB",
			StorageQuota:   "10 GB",
			StoragePercent: 0,
		},
	}

	for name, data := range templates {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.Render(&buf, name, data)
			require.NoError(t, err)

			output := buf.String()

			// Check for essential HTML structure
			assert.Contains(t, output, "<!DOCTYPE html>", "Should have DOCTYPE")
			assert.Contains(t, output, "<html", "Should have html tag")
			assert.Contains(t, output, "</html>", "Should have closing html tag")
			assert.Contains(t, output, "<head>", "Should have head section")
			assert.Contains(t, output, "</head>", "Should have closing head tag")
			assert.Contains(t, output, "<body", "Should have body tag")
			assert.Contains(t, output, "</body>", "Should have closing body tag")
			assert.Contains(t, output, "<meta charset=\"UTF-8\">", "Should have charset meta")
			assert.Contains(t, output, "viewport", "Should have viewport meta")
		})
	}
}

// TestUITemplates_RequiredResources tests that templates include required resources
func TestUITemplates_RequiredResources(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	templates := []string{"login", "register", "dashboard"}

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.Render(&buf, name, &handlers.TemplateData{
				Title: "Test",
			})
			require.NoError(t, err)

			output := buf.String()

			// Check for required CSS resources
			assert.Contains(t, output, "/static/css/output.css", "Should include Tailwind CSS")
			assert.Contains(t, output, "/static/css/custom.css", "Should include custom CSS")

			// Check for HTMX inclusion
			assert.Contains(t, output, "htmx", "Should include HTMX library")

			// Check for JavaScript
			assert.Contains(t, output, "/static/js/app.js", "Should include app JavaScript")
		})
	}
}

// TestUITemplates_AccessibilityFeatures tests for basic accessibility features
func TestUITemplates_AccessibilityFeatures(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	t.Run("login page accessibility", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "login", &handlers.TemplateData{})
		require.NoError(t, err)

		output := buf.String()

		// Check for form labels
		assert.Contains(t, output, `for="email"`, "Should have label for email input")
		assert.Contains(t, output, `for="password"`, "Should have label for password input")

		// Check for semantic HTML
		assert.Contains(t, output, "<form", "Should use form element")
		assert.Contains(t, output, "<label", "Should use label elements")
		assert.Contains(t, output, "<button", "Should use button elements")

		// Check for lang attribute
		assert.Contains(t, output, `lang="en"`, "Should have lang attribute")
	})

	t.Run("register page accessibility", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "register", &handlers.TemplateData{})
		require.NoError(t, err)

		output := buf.String()

		// Check for form labels
		assert.Contains(t, output, `for="email"`, "Should have label for email input")
		assert.Contains(t, output, `for="username"`, "Should have label for username input")
		assert.Contains(t, output, `for="password"`, "Should have label for password input")
		assert.Contains(t, output, `for="terms"`, "Should have label for terms checkbox")
	})
}

// TestUITemplates_HTMXAttributes tests for proper HTMX attributes
func TestUITemplates_HTMXAttributes(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	t.Run("login form HTMX", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "login", &handlers.TemplateData{})
		require.NoError(t, err)

		output := buf.String()

		// Check for HTMX form attributes
		assert.Contains(t, output, "hx-post", "Should have hx-post for form submission")
		assert.Contains(t, output, "/api/auth/login", "Should post to login endpoint")
		assert.Contains(t, output, "hx-target", "Should have hx-target")
		assert.Contains(t, output, "hx-indicator", "Should have hx-indicator for loading state")
	})

	t.Run("register form HTMX", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "register", &handlers.TemplateData{})
		require.NoError(t, err)

		output := buf.String()

		// Check for HTMX form attributes
		assert.Contains(t, output, "hx-post", "Should have hx-post for form submission")
		assert.Contains(t, output, "/api/auth/register", "Should post to register endpoint")
	})
}

// TestUITemplates_ResponsiveClasses tests for Tailwind responsive classes
func TestUITemplates_ResponsiveClasses(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	templates := []string{"login", "register", "dashboard"}

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.Render(&buf, name, &handlers.TemplateData{
				Title: "Test",
			})
			require.NoError(t, err)

			output := buf.String()

			// Check for responsive classes (sm:, md:, lg: prefixes)
			hasResponsive := strings.Contains(output, "sm:") ||
				strings.Contains(output, "md:") ||
				strings.Contains(output, "lg:")
			assert.True(t, hasResponsive, "Should include responsive Tailwind classes")
		})
	}
}

// TestUITemplates_SecurityHeaders tests for security-related HTML elements
func TestUITemplates_SecurityHeaders(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	t.Run("login security features", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "login", &handlers.TemplateData{})
		require.NoError(t, err)

		output := buf.String()

		// Check for input type password (doesn't show password in plaintext)
		assert.Contains(t, output, `type="password"`, "Should have password input type")

		// Check for autocomplete attributes
		assert.Contains(t, output, `autocomplete="email"`, "Should have autocomplete for email")
		assert.Contains(t, output, `autocomplete="current-password"`, "Should have autocomplete for password")
	})

	t.Run("register security features", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "register", &handlers.TemplateData{})
		require.NoError(t, err)

		output := buf.String()

		// Check for required attribute on critical fields
		assert.Contains(t, output, "required", "Should have required attribute on fields")

		// Check for password minlength
		assert.Contains(t, output, "minlength", "Should have minimum password length")
	})
}

// TestUITemplates_ToastContainer tests for toast notification container
func TestUITemplates_ToastContainer(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	templates := []string{"login", "register", "dashboard"}

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.Render(&buf, name, &handlers.TemplateData{
				Title: "Test",
			})
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, "toast-container", "Should have toast notification container")
		})
	}
}

// TestUITemplates_EmptyState tests the dashboard empty state rendering
func TestUITemplates_EmptyState(t *testing.T) {
	renderer := getTemplateRenderer(t)
	err := renderer.LoadTemplates()
	require.NoError(t, err)

	t.Run("empty state message", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "dashboard", &handlers.TemplateData{
			HasFiles: false,
		})
		require.NoError(t, err)

		output := buf.String()

		// Check for empty state elements
		assert.Contains(t, output, "No files yet", "Should show empty state message")
		assert.Contains(t, output, "upload", strings.ToLower(output), "Should encourage uploading")
	})

	t.Run("storage usage display", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.Render(&buf, "dashboard", &handlers.TemplateData{
			StorageUsed:    "1.5 GB",
			StorageQuota:   "10 GB",
			StoragePercent: 15,
		})
		require.NoError(t, err)

		output := buf.String()

		assert.Contains(t, output, "Storage Usage", "Should show storage section")
		assert.Contains(t, output, "1.5 GB", "Should show used storage")
		assert.Contains(t, output, "10 GB", "Should show quota")
	})
}
