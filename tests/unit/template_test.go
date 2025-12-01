//go:build unit

package unit

import (
	"bytes"
	"testing"

	"github.com/jd-boyd/filesonthego/assets"
	handlers_gin "github.com/jd-boyd/filesonthego/handlers_gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTemplateRenderer returns a template renderer using embedded assets
func getTemplateRenderer(t *testing.T) *handlers_gin.TemplateRenderer {
	t.Helper()
	templatesFS, err := assets.TemplatesFS()
	require.NoError(t, err, "Failed to get templates filesystem")
	return handlers_gin.NewTemplateRendererFromFS(templatesFS)
}

func TestTemplateRenderer_LoadTemplates(t *testing.T) {
	renderer := getTemplateRenderer(t)

	// Test loading templates
	err := renderer.LoadTemplates()
	require.NoError(t, err, "Failed to load templates")
}

func TestTemplateRenderer_RenderLogin(t *testing.T) {
	renderer := getTemplateRenderer(t)

	// Load templates
	err := renderer.LoadTemplates()
	require.NoError(t, err, "Failed to load templates")

	// Prepare test data
	data := &handlers_gin.TemplateData{
		Title: "Login - FilesOnTheGo",
	}

	// Render template
	var buf bytes.Buffer
	err = renderer.Render(&buf, "login", data)
	require.NoError(t, err, "Failed to render login template")

	// Assert output contains expected elements
	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>", "Output should contain DOCTYPE")
	assert.Contains(t, output, "Sign in to your account", "Output should contain login heading")
	assert.Contains(t, output, "email", "Output should contain email input")
	assert.Contains(t, output, "password", "Output should contain password input")
}

func TestTemplateRenderer_RenderRegister(t *testing.T) {
	renderer := getTemplateRenderer(t)

	// Load templates
	err := renderer.LoadTemplates()
	require.NoError(t, err, "Failed to load templates")

	// Prepare test data
	data := &handlers_gin.TemplateData{
		Title: "Register - FilesOnTheGo",
	}

	// Render template
	var buf bytes.Buffer
	err = renderer.Render(&buf, "register", data)
	require.NoError(t, err, "Failed to render register template")

	// Assert output contains expected elements
	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>", "Output should contain DOCTYPE")
	assert.Contains(t, output, "Create your account", "Output should contain register heading")
	assert.Contains(t, output, "email", "Output should contain email input")
	assert.Contains(t, output, "username", "Output should contain username input")
	assert.Contains(t, output, "password", "Output should contain password input")
}

func TestTemplateRenderer_RenderDashboard(t *testing.T) {
	renderer := getTemplateRenderer(t)

	// Load templates
	err := renderer.LoadTemplates()
	require.NoError(t, err, "Failed to load templates")

	// Prepare test data
	data := &handlers_gin.TemplateData{
		Title:          "Dashboard - FilesOnTheGo",
		StorageUsed:    "0 MB",
		StorageQuota:   "10 GB",
		StoragePercent: 0,
		HasFiles:       false,
	}

	// Render template
	var buf bytes.Buffer
	err = renderer.Render(&buf, "dashboard", data)
	require.NoError(t, err, "Failed to render dashboard template")

	// Assert output contains expected elements
	output := buf.String()
	assert.Contains(t, output, "<!DOCTYPE html>", "Output should contain DOCTYPE")
	assert.Contains(t, output, "My Files", "Output should contain dashboard heading")
	assert.Contains(t, output, "Storage Usage", "Output should contain storage info")
}

func TestTemplateRenderer_RenderNonExistent(t *testing.T) {
	renderer := getTemplateRenderer(t)

	// Load templates
	err := renderer.LoadTemplates()
	require.NoError(t, err, "Failed to load templates")

	// Try to render non-existent template
	var buf bytes.Buffer
	err = renderer.Render(&buf, "nonexistent", nil)
	assert.Error(t, err, "Should fail to render non-existent template")
}

func TestPrepareTemplateData(t *testing.T) {
	// This test would require a mock RequestEvent
	// For now, we just test that the function exists and can be called
	// Full integration tests will be in the integration test suite
	t.Skip("Skipping PrepareTemplateData test - requires mock RequestEvent")
}
