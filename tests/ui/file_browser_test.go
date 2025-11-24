package ui_test

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TemplateRenderer helps render templates for testing
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(t *testing.T) *TemplateRenderer {
	t.Helper()

	// Find template directory
	templateDir := findTemplateDir(t)

	// Custom template functions
	funcMap := template.FuncMap{
		"hasPrefix": strings.HasPrefix,
		"eq":        func(a, b string) bool { return a == b },
	}

	// Parse all templates
	tmpl := template.New("").Funcs(funcMap)

	// Parse layouts
	layoutFiles, err := filepath.Glob(filepath.Join(templateDir, "layouts", "*.html"))
	require.NoError(t, err, "Failed to find layout files")
	for _, f := range layoutFiles {
		_, err := tmpl.ParseFiles(f)
		require.NoError(t, err, "Failed to parse layout: %s", f)
	}

	// Parse components
	componentFiles, err := filepath.Glob(filepath.Join(templateDir, "components", "*.html"))
	require.NoError(t, err, "Failed to find component files")
	for _, f := range componentFiles {
		_, err := tmpl.ParseFiles(f)
		require.NoError(t, err, "Failed to parse component: %s", f)
	}

	// Parse pages
	pageFiles, err := filepath.Glob(filepath.Join(templateDir, "pages", "*.html"))
	require.NoError(t, err, "Failed to find page files")
	for _, f := range pageFiles {
		_, err := tmpl.ParseFiles(f)
		require.NoError(t, err, "Failed to parse page: %s", f)
	}

	return &TemplateRenderer{templates: tmpl}
}

func findTemplateDir(t *testing.T) string {
	t.Helper()

	// Try relative paths
	paths := []string{
		"../../assets/templates",
		"../assets/templates",
		"assets/templates",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	t.Fatal("Could not find templates directory")
	return ""
}

// RenderComponent renders a specific component template
func (r *TemplateRenderer) RenderComponent(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	err := r.templates.ExecuteTemplate(&buf, name, data)
	return buf.String(), err
}

// ItemInfo represents a file or directory item for testing
type ItemInfo struct {
	ID               string
	Name             string
	Type             string
	Size             int64
	SizeFormatted    string
	MimeType         string
	Extension        string
	Created          string
	Updated          string
	UpdatedFormatted string
}

// FileListData represents data for file list template
type FileListData struct {
	CurrentDirectoryID string
	ViewMode           string
	SortBy             string
	SortOrder          string
	IsLoading          bool
	Items              []ItemInfo
}

// TestFileListTemplateExists verifies the file-list.html template exists
func TestFileListTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-list.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "file-list.html should exist")
}

// TestFileItemTemplateExists verifies the file-item.html template exists
func TestFileItemTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-item.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "file-item.html should exist")
}

// TestFileActionsTemplateExists verifies the file-actions.html template exists
func TestFileActionsTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-actions.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "file-actions.html should exist")
}

// TestContextMenuTemplateExists verifies the context-menu.html template exists
func TestContextMenuTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "context-menu.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "context-menu.html should exist")
}

// TestFileDetailsModalExists verifies the file-details-modal.html template exists
func TestFileDetailsModalExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-details-modal.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "file-details-modal.html should exist")
}

// TestFilesPageTemplateExists verifies the files.html page template exists
func TestFilesPageTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "pages", "files.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "files.html page should exist")
}

// TestFileBrowserJSExists verifies the file-browser.js script exists
func TestFileBrowserJSExists(t *testing.T) {
	// Find static directory
	paths := []string{
		"../../assets/static/js/file-browser.js",
		"../assets/static/js/file-browser.js",
		"assets/static/js/file-browser.js",
	}

	found := false
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			found = true
			break
		}
	}

	assert.True(t, found, "file-browser.js should exist")
}

// TestFileListTemplateContainsRequiredElements tests file-list.html has required elements
func TestFileListTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-list.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file-list.html")

	contentStr := string(content)

	// Check for required elements
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"file list container", "file-list-container"},
		{"empty state", "empty-state"},
		{"loading state", "animate-spin"},
		{"list view", "file-list"},
		{"grid view", "file-grid"},
		{"select all checkbox", "select-all"},
		{"HTMX attributes", "hx-"},
		{"accessibility role", `role="`},
		{"aria label", "aria-label"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in file-list.html", elem.name)
	}
}

// TestFileItemTemplateContainsRequiredElements tests file-item.html has required elements
func TestFileItemTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-item.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file-item.html")

	contentStr := string(content)

	// Check for required elements
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"file item class", "file-item"},
		{"checkbox for selection", "file-checkbox"},
		{"context menu function", "showContextMenu"},
		{"data-id attribute", "data-id"},
		{"data-type attribute", "data-type"},
		{"download action", "download"},
		{"mobile view", "file-item-mobile"},
		{"file icon template", "file-icon"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in file-item.html", elem.name)
	}
}

// TestFileActionsTemplateContainsRequiredElements tests file-actions.html has required elements
func TestFileActionsTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-actions.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file-actions.html")

	contentStr := string(content)

	// Check for required elements
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"upload button", "openUploadModal"},
		{"new folder button", "openNewFolderModal"},
		{"batch actions", "batch-actions"},
		{"search box", "file-search"},
		{"sort dropdown", "sort"},
		{"view toggle", "view-list-btn"},
		{"view toggle grid", "view-grid-btn"},
		{"new folder modal", "new-folder-modal"},
		{"upload modal include", "upload-modal.html"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in file-actions.html", elem.name)
	}
}

// TestContextMenuTemplateContainsRequiredElements tests context-menu.html has required elements
func TestContextMenuTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "context-menu.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read context-menu.html")

	contentStr := string(content)

	// Check for required elements
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"context menu container", "context-menu"},
		{"download option", "context-menu-download"},
		{"rename option", "context-menu-rename"},
		{"move option", "context-menu-move"},
		{"delete option", "context-menu-delete"},
		{"share option", "context-menu-share"},
		{"properties option", "context-menu-details"},
		{"rename modal", "rename-modal"},
		{"delete modal", "delete-modal"},
		{"move modal", "move-modal"},
		{"share modal", "share-modal"},
		{"menu role", `role="menu"`},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in context-menu.html", elem.name)
	}
}

// TestFileDetailsModalContainsRequiredElements tests file-details-modal.html has required elements
func TestFileDetailsModalContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-details-modal.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file-details-modal.html")

	contentStr := string(content)

	// Check for required elements
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"modal container", "file-details-modal"},
		{"file name", "details-name"},
		{"file size", "details-size"},
		{"file type", "details-mime-type"},
		{"file path", "details-path"},
		{"created date", "details-created"},
		{"modified date", "details-modified"},
		{"download button", "details-download-btn"},
		{"share button", "shareFromDetails"},
		{"close function", "closeFileDetailsModal"},
		{"dialog role", `role="dialog"`},
		{"aria modal", `aria-modal="true"`},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in file-details-modal.html", elem.name)
	}
}

// TestFilesPageContainsRequiredElements tests files.html page has required elements
func TestFilesPageContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "pages", "files.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read files.html")

	contentStr := string(content)

	// Check for required elements
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"extends app layout", `template "app.html"`},
		{"page title", `define "title"`},
		{"file browser script", "file-browser.js"},
		{"main content", "main-content"},
		{"storage usage", "Storage"},
		{"file actions template", `template "file-actions.html"`},
		{"context menu template", `template "context-menu.html"`},
		{"file details modal", `template "file-details-modal.html"`},
		{"keyboard shortcuts help", "keyboard-shortcuts-help"},
		{"drag drop overlay", "drag-drop-overlay"},
		{"HTMX trigger", "hx-trigger"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in files.html", elem.name)
	}
}

// TestFileBrowserJSContainsRequiredFunctions tests file-browser.js has required functions
func TestFileBrowserJSContainsRequiredFunctions(t *testing.T) {
	// Find the JS file
	paths := []string{
		"../../assets/static/js/file-browser.js",
		"../assets/static/js/file-browser.js",
		"assets/static/js/file-browser.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read file-browser.js")

	contentStr := string(content)

	// Check for required functions
	requiredFunctions := []string{
		"initFileBrowser",
		"toggleFileSelection",
		"toggleSelectAll",
		"clearSelection",
		"showContextMenu",
		"closeContextMenu",
		"contextMenuAction",
		"downloadFile",
		"navigateToDirectory",
		"downloadSelected",
		"deleteSelected",
		"openNewFolderModal",
		"closeNewFolderModal",
		"openUploadModal",
		"closeUploadModal",
		"openRenameModal",
		"closeRenameModal",
		"openDeleteModal",
		"closeDeleteModal",
		"confirmDelete",
		"openMoveModal",
		"closeMoveModal",
		"confirmMove",
		"openShareModal",
		"closeShareModal",
		"openFileDetailsModal",
		"closeFileDetailsModal",
		"setViewMode",
		"clearSearch",
		"handleDragOver",
		"handleDragLeave",
		"handleDrop",
		"handleFileSelect",
		"startUpload",
		"initKeyboardShortcuts",
	}

	for _, fn := range requiredFunctions {
		assert.Contains(t, contentStr, "function "+fn,
			"function %s should be present in file-browser.js", fn)
	}
}

// TestFileBrowserJSKeyboardShortcuts tests that keyboard shortcuts are implemented
func TestFileBrowserJSKeyboardShortcuts(t *testing.T) {
	paths := []string{
		"../../assets/static/js/file-browser.js",
		"../assets/static/js/file-browser.js",
		"assets/static/js/file-browser.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read file-browser.js")

	contentStr := string(content)

	// Check for keyboard shortcut implementations
	shortcuts := []struct {
		key     string
		pattern string
	}{
		{"Delete", "Delete"},
		{"Ctrl+A", "ctrlKey"},
		{"Escape", "Escape"},
	}

	for _, shortcut := range shortcuts {
		assert.Contains(t, contentStr, shortcut.pattern,
			"keyboard shortcut for %s should be implemented", shortcut.key)
	}
}

// TestFileBrowserJSStateManagement tests state management is implemented
func TestFileBrowserJSStateManagement(t *testing.T) {
	paths := []string{
		"../../assets/static/js/file-browser.js",
		"../assets/static/js/file-browser.js",
		"assets/static/js/file-browser.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read file-browser.js")

	contentStr := string(content)

	// Check for state management
	stateElements := []string{
		"fileBrowserState",
		"selectedFiles",
		"currentDirectory",
		"viewMode",
		"contextMenuTarget",
		"pendingUploadFiles",
	}

	for _, elem := range stateElements {
		assert.Contains(t, contentStr, elem,
			"state element %s should be present in file-browser.js", elem)
	}
}

// TestFileBrowserJSHTMXIntegration tests HTMX integration
func TestFileBrowserJSHTMXIntegration(t *testing.T) {
	paths := []string{
		"../../assets/static/js/file-browser.js",
		"../assets/static/js/file-browser.js",
		"assets/static/js/file-browser.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read file-browser.js")

	contentStr := string(content)

	// Check for HTMX integration
	htmxPatterns := []string{
		"htmx:afterSwap",
		"htmx:responseError",
		"htmx.ajax",
		"htmx.trigger",
	}

	for _, pattern := range htmxPatterns {
		assert.Contains(t, contentStr, pattern,
			"HTMX integration pattern %s should be present", pattern)
	}
}

// TestFileBrowserJSAccessibility tests accessibility features
func TestFileBrowserJSAccessibility(t *testing.T) {
	paths := []string{
		"../../assets/static/js/file-browser.js",
		"../assets/static/js/file-browser.js",
		"assets/static/js/file-browser.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read file-browser.js")

	contentStr := string(content)

	// Check for accessibility features
	accessibilityPatterns := []string{
		"focus", // Focus management
	}

	for _, pattern := range accessibilityPatterns {
		assert.Contains(t, contentStr, pattern,
			"accessibility feature %s should be present", pattern)
	}
}

// TestFileIconTemplateCoversCommonTypes tests that file icons cover common file types
func TestFileIconTemplateCoversCommonTypes(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-item.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file-item.html")

	contentStr := string(content)

	// Check for common file type icons
	fileTypes := []struct {
		description string
		pattern     string
	}{
		{"image files", "image/"},
		{"PDF files", ".pdf"},
		{"document files", ".doc"},
		{"spreadsheet files", ".xls"},
		{"archive files", ".zip"},
		{"video files", ".mp4"},
		{"audio files", ".mp3"},
		{"code files", ".js"},
		{"default file icon", "{{else}}"},
	}

	for _, ft := range fileTypes {
		assert.Contains(t, contentStr, ft.pattern,
			"icon for %s should be present", ft.description)
	}
}
