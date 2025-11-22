package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================
// Share Template Tests
// ============================================

// TestShareButtonTemplateExists verifies the share-button.html template exists
func TestShareButtonTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-button.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "share-button.html should exist")
}

// TestShareModalTemplateExists verifies the share-modal.html template exists
func TestShareModalTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-modal.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "share-modal.html should exist")
}

// TestShareLinkDisplayTemplateExists verifies the share-link-display.html template exists
func TestShareLinkDisplayTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-link-display.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "share-link-display.html should exist")
}

// TestShareListItemTemplateExists verifies the share-list-item.html template exists
func TestShareListItemTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-list-item.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "share-list-item.html should exist")
}

// TestSharesPageTemplateExists verifies the shares.html page template exists
func TestSharesPageTemplateExists(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "pages", "shares.html")

	_, err := os.Stat(path)
	assert.NoError(t, err, "shares.html page should exist")
}

// TestShareJSExists verifies the share.js script exists
func TestShareJSExists(t *testing.T) {
	paths := []string{
		"../../static/js/share.js",
		"../static/js/share.js",
		"static/js/share.js",
	}

	found := false
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			found = true
			break
		}
	}

	assert.True(t, found, "share.js should exist")
}

// ============================================
// Share Modal Tests
// ============================================

// TestShareModalTemplateContainsRequiredElements tests share-modal.html has required elements
func TestShareModalTemplateContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-modal.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-modal.html")

	contentStr := string(content)

	// Check for required elements
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"modal container", "share-modal"},
		{"modal title", "share-modal-title"},
		{"item name", "share-item-name"},
		{"create share form", "create-share-form"},
		{"permission type read", `value="read"`},
		{"permission type read_upload", `value="read_upload"`},
		{"permission type upload_only", `value="upload_only"`},
		{"password toggle", "share-password-toggle"},
		{"password input", "share-password"},
		{"expiration toggle", "share-expiration-toggle"},
		{"expiration input", "share-expiration"},
		{"expiration presets", "setShareExpiration"},
		{"generate button", "create-share-btn"},
		{"share result container", "share-result"},
		{"close function", "closeShareModal"},
		{"tabs navigation", "share-tab-create"},
		{"existing shares tab", "share-tab-existing"},
		{"existing shares list", "existing-shares-list"},
		{"dialog role", `role="dialog"`},
		{"aria modal", `aria-modal="true"`},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in share-modal.html", elem.name)
	}
}

// TestShareModalPermissionOptions verifies all permission options are present
func TestShareModalPermissionOptions(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-modal.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-modal.html")

	contentStr := string(content)

	permissions := []struct {
		value       string
		description string
	}{
		{"read", "Read-only"},
		{"read_upload", "Read & Upload"},
		{"upload_only", "Upload-only"},
	}

	for _, perm := range permissions {
		assert.Contains(t, contentStr, perm.value,
			"permission value %s should be present", perm.value)
		assert.Contains(t, contentStr, perm.description,
			"permission description %s should be present", perm.description)
	}
}

// ============================================
// Share Link Display Tests
// ============================================

// TestShareLinkDisplayContainsRequiredElements tests share-link-display.html
func TestShareLinkDisplayContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-link-display.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-link-display.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"share link display template", "share-link-display"},
		{"URL input", "share-url-input"},
		{"copy button", "copyShareLinkFromInput"},
		{"permission badge", "permission-badge"},
		{"QR code button", "showShareQRCode"},
		{"success message", "success"},
		{"data share url", "data-share-url"},
		{"password protected indicator", "IsPasswordProtected"},
		{"expiration info", "ExpiresAt"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in share-link-display.html", elem.name)
	}
}

// ============================================
// Share List Item Tests
// ============================================

// TestShareListItemContainsRequiredElements tests share-list-item.html
func TestShareListItemContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-list-item.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-list-item.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"share item class", "share-item"},
		{"share ID data attr", "data-share-id"},
		{"share URL data attr", "data-share-url"},
		{"copy link function", "copyShareLink"},
		{"permission badge", "PermissionType"},
		{"password protected icon", "IsPasswordProtected"},
		{"expiration display", "ExpiresAt"},
		{"access count", "AccessCount"},
		{"created date", "FormattedCreated"},
		{"edit expiration", "editShareExpiration"},
		{"revoke button", "revokeShare"},
		{"access logs", "viewShareAccessLogs"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in share-list-item.html", elem.name)
	}
}

// TestShareListItemCompactVariant tests compact variant exists
func TestShareListItemCompactVariant(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-list-item.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-list-item.html")

	contentStr := string(content)

	assert.Contains(t, contentStr, "share-list-item-compact",
		"compact variant should be defined")
}

// ============================================
// Shares Page Tests
// ============================================

// TestSharesPageContainsRequiredElements tests shares.html page
func TestSharesPageContainsRequiredElements(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "pages", "shares.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read shares.html")

	contentStr := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"extends app layout", `template "app.html"`},
		{"page title", `define "title"`},
		{"share.js script", "share.js"},
		{"shares page container", "shares-page"},
		{"stats total shares", "stat-total-shares"},
		{"stats active shares", "stat-active-shares"},
		{"stats expired shares", "stat-expired-shares"},
		{"stats total accesses", "stat-total-accesses"},
		{"filter by type", "filter-type"},
		{"filter by status", "filter-status"},
		{"sort dropdown", "sort-by"},
		{"search input", "share-search"},
		{"bulk actions", "bulk-actions"},
		{"select all checkbox", "select-all-shares"},
		{"shares list container", "shares-list"},
		{"empty state", "shares-empty"},
		{"edit expiration modal", "edit-expiration-modal"},
		{"revoke confirm modal", "revoke-confirm-modal"},
		{"access logs modal", "access-logs-modal"},
		{"HTMX trigger", "hx-get"},
	}

	for _, elem := range requiredElements {
		assert.Contains(t, contentStr, elem.pattern,
			"%s should be present in shares.html", elem.name)
	}
}

// ============================================
// Share JavaScript Tests
// ============================================

// TestShareJSContainsRequiredFunctions tests share.js has required functions
func TestShareJSContainsRequiredFunctions(t *testing.T) {
	paths := []string{
		"../../static/js/share.js",
		"../static/js/share.js",
		"static/js/share.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read share.js")

	contentStr := string(content)

	requiredFunctions := []string{
		"openShareModal",
		"closeShareModal",
		"switchShareTab",
		"updatePermissionSelection",
		"toggleSharePassword",
		"toggleShareExpiration",
		"setShareExpiration",
		"submitShareForm",
		"loadExistingShares",
		"revokeShare",
		"revokeShareFromModal",
		"confirmRevokeShare",
		"closeRevokeConfirmModal",
		"copyShareLink",
		"copyShareLinkFromInput",
		"showShareQRCode",
		"closeQRCodeModal",
		"viewShareAccessLogs",
		"closeAccessLogsModal",
		"editShareExpiration",
		"submitExpirationUpdate",
		"closeEditExpirationModal",
		"formatRelativeDate",
		"truncateUrl",
		"escapeHtml",
	}

	for _, fn := range requiredFunctions {
		assert.Contains(t, contentStr, "function "+fn,
			"function %s should be present in share.js", fn)
	}
}

// TestShareJSStateManagement tests state management is implemented
func TestShareJSStateManagement(t *testing.T) {
	paths := []string{
		"../../static/js/share.js",
		"../static/js/share.js",
		"static/js/share.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read share.js")

	contentStr := string(content)

	stateElements := []string{
		"shareState",
		"currentResourceId",
		"currentResourceType",
		"currentResourceName",
		"selectedShares",
		"pendingRevokeShareId",
		"existingShares",
		"baseUrl",
	}

	for _, elem := range stateElements {
		assert.Contains(t, contentStr, elem,
			"state element %s should be present in share.js", elem)
	}
}

// TestShareJSAPIIntegration tests API integration points
func TestShareJSAPIIntegration(t *testing.T) {
	paths := []string{
		"../../static/js/share.js",
		"../static/js/share.js",
		"static/js/share.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read share.js")

	contentStr := string(content)

	apiEndpoints := []string{
		"/api/shares",
		"method: 'POST'",
		"method: 'DELETE'",
		"method: 'PATCH'",
		"/logs",
	}

	for _, endpoint := range apiEndpoints {
		assert.Contains(t, contentStr, endpoint,
			"API endpoint/method %s should be referenced in share.js", endpoint)
	}
}

// TestShareJSClipboardSupport tests clipboard functionality
func TestShareJSClipboardSupport(t *testing.T) {
	paths := []string{
		"../../static/js/share.js",
		"../static/js/share.js",
		"static/js/share.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read share.js")

	contentStr := string(content)

	assert.Contains(t, contentStr, "navigator.clipboard.writeText",
		"clipboard API should be used for copying")
}

// TestShareJSShowsToastNotifications tests toast notification integration
func TestShareJSShowsToastNotifications(t *testing.T) {
	paths := []string{
		"../../static/js/share.js",
		"../static/js/share.js",
		"static/js/share.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read share.js")

	contentStr := string(content)

	toastCalls := []string{
		"showToast('success'",
		"showToast('error'",
	}

	for _, call := range toastCalls {
		assert.Contains(t, contentStr, call,
			"toast call %s should be present in share.js", call)
	}
}

// TestShareJSExportsGlobalFunctions tests functions are exported globally
func TestShareJSExportsGlobalFunctions(t *testing.T) {
	paths := []string{
		"../../static/js/share.js",
		"../static/js/share.js",
		"static/js/share.js",
	}

	var content []byte
	var err error

	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "Failed to read share.js")

	contentStr := string(content)

	globalExports := []string{
		"window.openShareModal",
		"window.closeShareModal",
		"window.copyShareLink",
		"window.revokeShare",
		"window.submitShareForm",
	}

	for _, export := range globalExports {
		assert.Contains(t, contentStr, export,
			"global export %s should be present in share.js", export)
	}
}

// ============================================
// Files Page Share Integration Tests
// ============================================

// TestFilesPageIncludesShareModal tests files.html includes share modal
func TestFilesPageIncludesShareModal(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "pages", "files.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read files.html")

	contentStr := string(content)

	assert.Contains(t, contentStr, `template "share-modal.html"`,
		"files.html should include share-modal.html template")
}

// TestFilesPageIncludesShareJS tests files.html includes share.js
func TestFilesPageIncludesShareJS(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "pages", "files.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read files.html")

	contentStr := string(content)

	assert.Contains(t, contentStr, "share.js",
		"files.html should include share.js script")
}

// TestContextMenuHasShareOption tests context menu has share option
func TestContextMenuHasShareOption(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "context-menu.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read context-menu.html")

	contentStr := string(content)

	assert.Contains(t, contentStr, "context-menu-share",
		"context menu should have share option")
	assert.Contains(t, contentStr, "contextMenuAction('share')",
		"context menu should trigger share action")
}

// ============================================
// Accessibility Tests
// ============================================

// TestShareModalAccessibility tests share modal accessibility features
func TestShareModalAccessibility(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-modal.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-modal.html")

	contentStr := string(content)

	accessibilityElements := []struct {
		name    string
		pattern string
	}{
		{"dialog role", `role="dialog"`},
		{"aria modal", `aria-modal="true"`},
		{"aria labelledby", "aria-labelledby"},
		{"close button accessible", "Close"},
	}

	for _, elem := range accessibilityElements {
		assert.Contains(t, contentStr, elem.pattern,
			"accessibility feature %s should be present", elem.name)
	}
}

// TestShareButtonAccessibility tests share button accessibility
func TestShareButtonAccessibility(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-button.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-button.html")

	contentStr := string(content)

	assert.Contains(t, contentStr, "aria-label",
		"share button should have aria-label")
}

// ============================================
// Permission Badge Tests
// ============================================

// TestShareLinkDisplayHasPermissionBadges tests permission badge styles
func TestShareLinkDisplayHasPermissionBadges(t *testing.T) {
	templateDir := findTemplateDir(t)
	path := filepath.Join(templateDir, "components", "share-link-display.html")

	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read share-link-display.html")

	contentStr := string(content)

	badges := []string{
		"permission-badge-read",
		"permission-badge-read_upload",
		"permission-badge-upload_only",
	}

	for _, badge := range badges {
		assert.Contains(t, contentStr, badge,
			"permission badge %s should be defined", badge)
	}
}
