// Package assets provides access to embedded templates and static files,
// with optional fallback to filesystem files for development.
package assets

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates static
var embeddedFS embed.FS

// UseEmbedded controls whether to use embedded files or filesystem files.
// When true (default), uses embedded files compiled into the binary.
// When false, reads files from the filesystem (useful for development).
var UseEmbedded = true

// baseDir is the base directory for filesystem access when UseEmbedded is false
var baseDir = "."

// SetBaseDir sets the base directory for filesystem file access
func SetBaseDir(dir string) {
	baseDir = dir
}

// FS returns an fs.FS for accessing templates and static files.
// Returns embedded files if UseEmbedded is true, otherwise filesystem files.
func FS() fs.FS {
	if UseEmbedded {
		return embeddedFS
	}
	return &filesystemFS{baseDir: baseDir}
}

// TemplatesFS returns an fs.FS rooted at the templates directory
func TemplatesFS() (fs.FS, error) {
	if UseEmbedded {
		return fs.Sub(embeddedFS, "templates")
	}
	return &filesystemFS{baseDir: filepath.Join(baseDir, "templates")}, nil
}

// StaticFS returns an fs.FS rooted at the static directory
func StaticFS() (fs.FS, error) {
	if UseEmbedded {
		return fs.Sub(embeddedFS, "static")
	}
	return &filesystemFS{baseDir: filepath.Join(baseDir, "static")}, nil
}

// filesystemFS implements fs.FS using the local filesystem
type filesystemFS struct {
	baseDir string
}

func (f *filesystemFS) Open(name string) (fs.File, error) {
	// Ensure the path is clean and doesn't escape the base directory
	cleanPath := filepath.Clean(name)
	if filepath.IsAbs(cleanPath) || cleanPath == ".." || len(cleanPath) > 2 && cleanPath[:3] == "../" {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	fullPath := filepath.Join(f.baseDir, cleanPath)
	return os.Open(fullPath)
}

// ReadFile reads a file from the assets filesystem
func ReadFile(name string) ([]byte, error) {
	if UseEmbedded {
		return embeddedFS.ReadFile(name)
	}
	return os.ReadFile(filepath.Join(baseDir, name))
}

// ReadDir reads a directory from the assets filesystem
func ReadDir(name string) ([]fs.DirEntry, error) {
	if UseEmbedded {
		return embeddedFS.ReadDir(name)
	}
	return os.ReadDir(filepath.Join(baseDir, name))
}
