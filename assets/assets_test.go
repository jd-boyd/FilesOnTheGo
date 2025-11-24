package assets

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedFS_ContainsTemplates(t *testing.T) {
	// Ensure embedded FS contains expected template files
	UseEmbedded = true

	fsys := FS()
	require.NotNil(t, fsys)

	// Check that templates directory exists
	entries, err := fs.ReadDir(fsys, "templates")
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	// Check for specific template directories
	dirNames := make([]string, 0, len(entries))
	for _, e := range entries {
		dirNames = append(dirNames, e.Name())
	}
	assert.Contains(t, dirNames, "layouts")
	assert.Contains(t, dirNames, "pages")
	assert.Contains(t, dirNames, "components")
}

func TestEmbeddedFS_ContainsStatic(t *testing.T) {
	// Ensure embedded FS contains expected static files
	UseEmbedded = true

	fsys := FS()
	require.NotNil(t, fsys)

	// Check that static directory exists
	entries, err := fs.ReadDir(fsys, "static")
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	// Check for specific static directories
	dirNames := make([]string, 0, len(entries))
	for _, e := range entries {
		dirNames = append(dirNames, e.Name())
	}
	assert.Contains(t, dirNames, "css")
	assert.Contains(t, dirNames, "js")
}

func TestTemplatesFS_Embedded(t *testing.T) {
	UseEmbedded = true

	templatesFS, err := TemplatesFS()
	require.NoError(t, err)
	require.NotNil(t, templatesFS)

	// Should be able to open a template directly (without "templates/" prefix)
	f, err := templatesFS.Open("layouts/base.html")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "<!DOCTYPE html>")
}

func TestStaticFS_Embedded(t *testing.T) {
	UseEmbedded = true

	staticFS, err := StaticFS()
	require.NoError(t, err)
	require.NotNil(t, staticFS)

	// Should be able to open a static file directly (without "static/" prefix)
	f, err := staticFS.Open("css/output.css")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestFilesystemFS_External(t *testing.T) {
	// Create a temp directory with test files
	tmpDir := t.TempDir()

	// Create templates structure
	templatesDir := filepath.Join(tmpDir, "templates", "layouts")
	require.NoError(t, os.MkdirAll(templatesDir, 0755))

	testContent := "<!DOCTYPE html><html>test</html>"
	testFile := filepath.Join(templatesDir, "test.html")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	// Configure to use external assets
	UseEmbedded = false
	SetBaseDir(tmpDir)

	// Test FS()
	fsys := FS()
	require.NotNil(t, fsys)

	f, err := fsys.Open("templates/layouts/test.html")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Reset to embedded for other tests
	UseEmbedded = true
}

func TestFilesystemFS_TemplatesFS_External(t *testing.T) {
	// Create a temp directory with test files
	tmpDir := t.TempDir()

	// Create templates structure
	templatesDir := filepath.Join(tmpDir, "templates", "pages")
	require.NoError(t, os.MkdirAll(templatesDir, 0755))

	testContent := "<h1>Test Page</h1>"
	testFile := filepath.Join(templatesDir, "test.html")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	// Configure to use external assets
	UseEmbedded = false
	SetBaseDir(tmpDir)

	// Test TemplatesFS()
	templatesFS, err := TemplatesFS()
	require.NoError(t, err)
	require.NotNil(t, templatesFS)

	// Should open without "templates/" prefix
	f, err := templatesFS.Open("pages/test.html")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Reset to embedded for other tests
	UseEmbedded = true
}

func TestFilesystemFS_StaticFS_External(t *testing.T) {
	// Create a temp directory with test files
	tmpDir := t.TempDir()

	// Create static structure
	staticDir := filepath.Join(tmpDir, "static", "js")
	require.NoError(t, os.MkdirAll(staticDir, 0755))

	testContent := "console.log('test');"
	testFile := filepath.Join(staticDir, "test.js")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	// Configure to use external assets
	UseEmbedded = false
	SetBaseDir(tmpDir)

	// Test StaticFS()
	staticFS, err := StaticFS()
	require.NoError(t, err)
	require.NotNil(t, staticFS)

	// Should open without "static/" prefix
	f, err := staticFS.Open("js/test.js")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Reset to embedded for other tests
	UseEmbedded = true
}

func TestFilesystemFS_PathTraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()

	UseEmbedded = false
	SetBaseDir(tmpDir)

	fsys := FS()

	// Attempt path traversal
	_, err := fsys.Open("../../../etc/passwd")
	assert.Error(t, err)

	_, err = fsys.Open("..\\..\\..\\etc\\passwd")
	assert.Error(t, err)

	// Reset to embedded for other tests
	UseEmbedded = true
}

func TestReadFile_Embedded(t *testing.T) {
	UseEmbedded = true

	content, err := ReadFile("templates/layouts/base.html")
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "<!DOCTYPE html>")
}

func TestReadFile_External(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := "test file content"
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	UseEmbedded = false
	SetBaseDir(tmpDir)

	content, err := ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Reset to embedded for other tests
	UseEmbedded = true
}

func TestReadDir_Embedded(t *testing.T) {
	UseEmbedded = true

	entries, err := ReadDir("templates")
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	dirNames := make([]string, 0, len(entries))
	for _, e := range entries {
		dirNames = append(dirNames, e.Name())
	}
	assert.Contains(t, dirNames, "layouts")
}

func TestReadDir_External(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir1"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir2"), 0755))

	UseEmbedded = false
	SetBaseDir(tmpDir)

	entries, err := ReadDir(".")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Reset to embedded for other tests
	UseEmbedded = true
}

func TestSwitchingBetweenModes(t *testing.T) {
	// Test that we can switch between embedded and external modes
	tmpDir := t.TempDir()

	// Create external file with different content
	templatesDir := filepath.Join(tmpDir, "templates", "layouts")
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	externalContent := "EXTERNAL CONTENT"
	require.NoError(t, os.WriteFile(filepath.Join(templatesDir, "base.html"), []byte(externalContent), 0644))

	// Read embedded version
	UseEmbedded = true
	embeddedContent, err := ReadFile("templates/layouts/base.html")
	require.NoError(t, err)
	assert.Contains(t, string(embeddedContent), "<!DOCTYPE html>")

	// Switch to external
	UseEmbedded = false
	SetBaseDir(tmpDir)
	externalReadContent, err := ReadFile("templates/layouts/base.html")
	require.NoError(t, err)
	assert.Equal(t, externalContent, string(externalReadContent))

	// Switch back to embedded
	UseEmbedded = true
	embeddedContent2, err := ReadFile("templates/layouts/base.html")
	require.NoError(t, err)
	assert.Equal(t, embeddedContent, embeddedContent2)
}
