package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findStaticDir finds the static directory
func findStaticDir(t *testing.T) string {
	t.Helper()

	paths := []string{
		"../../static",
		"../static",
		"static",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	t.Fatal("Could not find static directory")
	return ""
}

// ============================================
// Upload Component Template Tests
// ============================================

// TestUploadButtonTemplateExists verifies the upload-button.html template exists
func TestUploadButtonTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-button.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "upload-button.html should exist")
}

// TestUploadModalTemplateExists verifies the upload-modal.html template exists
func TestUploadModalTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-modal.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "upload-modal.html should exist")
}

// TestUploadProgressTemplateExists verifies the upload-progress.html template exists
func TestUploadProgressTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-progress.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "upload-progress.html should exist")
}

// TestDropZoneTemplateExists verifies the drop-zone.html template exists
func TestDropZoneTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "drop-zone.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "drop-zone.html should exist")
}

// TestUploadFormTemplateExists verifies the upload-form.html template exists
func TestUploadFormTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-form.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "upload-form.html should exist")
}

// TestUploadJSExists verifies the upload.js script exists
func TestUploadJSExists(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	_, err := os.Stat(path)
	assert.NoError(t, err, "upload.js should exist")
}

// ============================================
// Upload Button Template Content Tests
// ============================================

func TestUploadButtonTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-button.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload-button.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"upload button id", "upload-button"},
		{"click handler", "openUploadModal"},
		{"upload icon", "svg"},
		{"keyboard shortcut display", "Ctrl+U"},
		{"aria label", "aria-label"},
		{"primary button styling", "bg-primary"},
		{"responsive text", "hidden sm:inline"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload-button.html", elem.name)
	}
}

// ============================================
// Upload Modal Template Content Tests
// ============================================

func TestUploadModalTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-modal.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload-modal.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"modal container", "upload-modal"},
		{"modal title", "Upload Files"},
		{"drop zone", "modal-drop-zone"},
		{"file input", "modal-file-input"},
		{"file list", "upload-file-list"},
		{"progress section", "upload-progress"},
		{"progress bar", "upload-progress-bar"},
		{"upload button", "upload-btn"},
		{"cancel button", "closeUploadModal"},
		{"cancel all button", "cancelAllUploads"},
		{"drag handlers", "handleDrop"},
		{"dialog role", `role="dialog"`},
		{"aria modal", `aria-modal="true"`},
		{"close button", "Close upload dialog"},
		{"directory input", "modal-directory-id"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload-modal.html", elem.name)
	}
}

// ============================================
// Upload Progress Template Content Tests
// ============================================

func TestUploadProgressTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-progress.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload-progress.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"overall progress section", "upload-overall-progress"},
		{"upload queue", "upload-queue"},
		{"file item template", "upload-file-item-template"},
		{"progress bar", "progress-bar"},
		{"progress text", "progress-text"},
		{"cancel button", "cancel-btn"},
		{"success status", "status-success"},
		{"error status", "status-error"},
		{"error template", "upload-error-template"},
		{"success template", "upload-success-template"},
		{"file name", "file-name"},
		{"file size", "file-size"},
		{"file preview", "file-preview"},
		{"remove button", "remove-btn"},
		{"aria progressbar", `role="progressbar"`},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload-progress.html", elem.name)
	}
}

// ============================================
// Drop Zone Template Content Tests
// ============================================

func TestDropZoneTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "drop-zone.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read drop-zone.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"drop zone id", "drop-zone"},
		{"dashed border", "border-dashed"},
		{"drag handlers", "ondrop"},
		{"drag over handler", "ondragover"},
		{"drag leave handler", "ondragleave"},
		{"click to browse", "onclick"},
		{"file input", "file-input"},
		{"upload icon", "svg"},
		{"browse text", "browse"},
		{"max file size info", "Max file size"},
		{"file types info", "All file types"},
		{"multiple files info", "Multiple files"},
		{"keyboard accessibility", "tabindex"},
		{"aria label", "aria-label"},
		{"keyboard handler", "onkeydown"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in drop-zone.html", elem.name)
	}
}

// ============================================
// Upload Form Template Content Tests
// ============================================

func TestUploadFormTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-form.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload-form.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"form id", "upload-form"},
		{"htmx post", "hx-post"},
		{"multipart encoding", "multipart/form-data"},
		{"directory id input", "upload-directory-id"},
		{"share token input", "upload-share-token"},
		{"upload results container", "upload-results"},
		{"upload controls", "upload-controls"},
		{"clear queue button", "clear-queue-btn"},
		{"upload all button", "upload-all-btn"},
		{"file count display", "upload-file-count"},
		{"total size display", "upload-total-size"},
		{"loading indicator", "upload-indicator"},
		{"drop zone template", `template "drop-zone.html"`},
		{"progress template", `template "upload-progress.html"`},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload-form.html", elem.name)
	}
}

// ============================================
// Upload JavaScript Tests
// ============================================

func TestUploadJSContainsRequiredFunctions(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	requiredFunctions := []string{
		"initUploadModule",
		"openUploadModal",
		"closeUploadModal",
		"resetUploadState",
		"handleDragOver",
		"handleDragLeave",
		"handleDrop",
		"handleFileSelect",
		"addFilesToQueue",
		"validateFile",
		"updateFileListUI",
		"createFileListItem",
		"removeUploadFile",
		"clearUploadQueue",
		"startUpload",
		"uploadFile",
		"updateOverallProgress",
		"cancelUpload",
		"cancelAllUploads",
		"formatFileSize",
		"escapeHtml",
	}

	for _, fn := range requiredFunctions {
		assert.Contains(t, contentStr, "function "+fn,
			"function %s should be present in upload.js", fn)
	}
}

func TestUploadJSContainsUploadConfig(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	configElements := []struct {
		name    string
		pattern string
	}{
		{"upload config object", "uploadConfig"},
		{"max file size setting", "maxFileSize"},
		{"max concurrent uploads", "maxConcurrentUploads"},
		{"upload endpoint", "uploadEndpoint"},
		{"retry attempts", "retryAttempts"},
	}

	for _, elem := range configElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload.js", elem.name)
	}
}

func TestUploadJSContainsStateManagement(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	stateElements := []struct {
		name    string
		pattern string
	}{
		{"upload state object", "uploadState"},
		{"files array", "files:"},
		{"uploads map", "uploads:"},
		{"is uploading flag", "isUploading"},
		{"uploaded count", "uploadedCount"},
		{"total size", "totalSize"},
		{"uploaded size", "uploadedSize"},
	}

	for _, elem := range stateElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload.js", elem.name)
	}
}

func TestUploadJSContainsValidation(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	validationPatterns := []struct {
		name    string
		pattern string
	}{
		{"file size validation", "file.size"},
		{"duplicate check", "isDuplicate"},
		{"validation error messages", "showValidationErrors"},
		{"file type mime", "mimeType"},
	}

	for _, elem := range validationPatterns {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload.js", elem.name)
	}
}

func TestUploadJSContainsProgressTracking(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	progressPatterns := []struct {
		name    string
		pattern string
	}{
		{"XMLHttpRequest usage", "XMLHttpRequest"},
		{"upload progress event", "upload.addEventListener"},
		{"progress event", "progress"},
		{"loaded/total calculation", "lengthComputable"},
		{"progress bar update", "style.width"},
	}

	for _, elem := range progressPatterns {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload.js", elem.name)
	}
}

func TestUploadJSContainsErrorHandling(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	errorPatterns := []struct {
		name    string
		pattern string
	}{
		{"XHR load event", "addEventListener"},
		{"status check", "xhr.status"},
		{"error status assignment", "status = 'error'"},
		{"error message", "errorMessage"},
		{"showToast error", "showToast"},
	}

	for _, elem := range errorPatterns {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload.js", elem.name)
	}
}

func TestUploadJSContainsImagePreview(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	previewPatterns := []struct {
		name    string
		pattern string
	}{
		{"FileReader usage", "FileReader"},
		{"image type check", "image/"},
		{"preview property", "preview"},
		{"readAsDataURL", "readAsDataURL"},
	}

	for _, elem := range previewPatterns {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload.js", elem.name)
	}
}

func TestUploadJSContainsFileTypeIcons(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	// Check for function that provides file type icons
	assert.Contains(t, contentStr, "getFileTypeIcon",
		"getFileTypeIcon function should be present")

	// Check for various file type handling
	fileTypes := []string{
		"pdf",
		"word",
		"image/",
		"video/",
		"audio/",
		"zip",
		"text",
	}

	for _, ft := range fileTypes {
		assert.Contains(t, contentStr, ft,
			"file type %s should be handled in upload.js", ft)
	}
}

func TestUploadJSContainsKeyboardShortcuts(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	keyboardPatterns := []struct {
		name    string
		pattern string
	}{
		{"keyboard shortcut init", "initUploadKeyboardShortcuts"},
		{"keydown event", "keydown"},
		{"Ctrl key check", "ctrlKey"},
		{"Escape key", "Escape"},
		{"u key for upload", "'u'"},
	}

	for _, elem := range keyboardPatterns {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in upload.js", elem.name)
	}
}

func TestUploadJSExportsGlobalFunctions(t *testing.T) {
	staticDir := findStaticDir(t)
	path := filepath.Join(staticDir, "js", "upload.js")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload.js")

	contentStr := string(content)

	exports := []string{
		"window.openUploadModal",
		"window.closeUploadModal",
		"window.handleDragOver",
		"window.handleDragLeave",
		"window.handleDrop",
		"window.handleFileSelect",
		"window.removeUploadFile",
		"window.clearUploadQueue",
		"window.startUpload",
		"window.cancelUpload",
		"window.cancelAllUploads",
	}

	for _, export := range exports {
		assert.Contains(t, contentStr, export,
			"%s should be exported globally", export)
	}
}

// ============================================
// Integration Tests for Upload Flow
// ============================================

func TestFilesPageIncludesUploadJS(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "pages", "files.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read files.html")

	contentStr := string(content)

	assert.Contains(t, contentStr, "upload.js",
		"files.html should include upload.js")
}

func TestFileActionsIncludesUploadModal(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "file-actions.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file-actions.html")

	contentStr := string(content)

	assert.Contains(t, contentStr, "upload-modal",
		"file-actions.html should include upload modal")
}

// ============================================
// Accessibility Tests
// ============================================

func TestUploadModalAccessibility(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "upload-modal.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read upload-modal.html")

	contentStr := string(content)

	accessibilityPatterns := []struct {
		name    string
		pattern string
	}{
		{"dialog role", `role="dialog"`},
		{"aria modal", `aria-modal="true"`},
		{"aria labelledby", "aria-labelledby"},
		{"sr-only close label", "sr-only"},
		{"focusable close button", "button"},
	}

	for _, elem := range accessibilityPatterns {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present for accessibility", elem.name)
	}
}

func TestDropZoneAccessibility(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "drop-zone.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read drop-zone.html")

	contentStr := string(content)

	accessibilityPatterns := []struct {
		name    string
		pattern string
	}{
		{"button role", `role="button"`},
		{"tabindex", "tabindex"},
		{"aria label", "aria-label"},
		{"keyboard handler", "onkeydown"},
	}

	for _, elem := range accessibilityPatterns {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present for accessibility", elem.name)
	}
}
