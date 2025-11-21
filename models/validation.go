package models

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"unicode"
)

// Common validation errors
var (
	ErrNullByte         = errors.New("null byte detected in input")
	ErrPathTraversal    = errors.New("path traversal attempt detected")
	ErrInvalidCharacter = errors.New("invalid character in input")
	ErrTooLong          = errors.New("input exceeds maximum length")
	ErrEmpty            = errors.New("input cannot be empty")
)

// SanitizeFilename sanitizes a filename to prevent security issues
// Removes path separators, control characters, and validates length
func SanitizeFilename(filename string) (string, error) {
	if filename == "" {
		return "", ErrEmpty
	}

	// Check for null bytes - major security issue
	if strings.Contains(filename, "\x00") {
		return "", ErrNullByte
	}

	// Check for control characters (0-31 and 127) - reject rather than silently remove
	for _, r := range filename {
		if r < 32 || r == 127 {
			return "", fmt.Errorf("%w: control character detected", ErrInvalidCharacter)
		}
	}

	// Normalize backslashes to forward slashes for cross-platform compatibility
	// This ensures Windows-style paths are handled correctly on Linux
	filename = strings.ReplaceAll(filename, "\\", "/")

	// Use filepath.Base to remove any path components
	// This prevents "../" attacks
	filename = filepath.Base(filename)

	// Check length after base extraction
	if len(filename) > 255 {
		return "", fmt.Errorf("%w: filename too long (max 255 characters)", ErrTooLong)
	}

	// Additional cleaning: remove any remaining control characters that may have
	// been introduced by filepath.Base or other processing
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1 // Remove character
		}
		return r
	}, filename)

	// Check if filename became empty after cleaning
	if cleaned == "" {
		return "", fmt.Errorf("%w: filename contains only invalid characters", ErrInvalidCharacter)
	}

	// Additional validation: ensure it's not "." or ".."
	if cleaned == "." || cleaned == ".." {
		return "", fmt.Errorf("%w: invalid filename", ErrInvalidCharacter)
	}

	return cleaned, nil
}

// SanitizePath sanitizes a file path for storage
// Ensures the path doesn't contain traversal attempts and is normalized
func SanitizePath(path string) (string, error) {
	// Empty path is valid (root)
	if path == "" || path == "/" {
		return "/", nil
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return "", ErrNullByte
	}

	// Check length
	if len(path) > 1024 {
		return "", fmt.Errorf("%w: path too long (max 1024 characters)", ErrTooLong)
	}

	// Validate for traversal BEFORE cleaning
	// This catches attacks like "/documents/../../../etc"
	if err := ValidatePathTraversal(path); err != nil {
		return "", err
	}

	// Clean the path to normalize it
	cleaned := filepath.Clean(path)

	// Ensure path starts with /
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}

	return cleaned, nil
}

// ValidatePathTraversal checks if a path contains directory traversal attempts
func ValidatePathTraversal(path string) error {
	// Empty path is valid
	if path == "" || path == "/" {
		return nil
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return ErrNullByte
	}

	// URL-decode the path to catch encoded traversal attempts
	// Decode multiple times to catch double-encoding
	decodedPath := path
	for i := 0; i < 3; i++ {
		decoded, err := url.QueryUnescape(decodedPath)
		if err == nil && decoded != decodedPath {
			decodedPath = decoded
		} else {
			break
		}
	}

	// Check for common traversal patterns in both original and decoded paths
	dangerousPatterns := []string{
		"../",
		"..\\",
		"..",
	}

	pathsToCheck := []string{path, decodedPath, strings.ToLower(path), strings.ToLower(decodedPath)}
	for _, p := range pathsToCheck {
		for _, pattern := range dangerousPatterns {
			if strings.Contains(p, pattern) {
				return ErrPathTraversal
			}
		}
	}

	// Clean the path and check for escaping root
	cleaned := filepath.Clean(path)

	// Additional check: if cleaned path starts with "..", it's traversal
	if strings.HasPrefix(cleaned, "..") {
		return ErrPathTraversal
	}

	return nil
}

// ValidateFileSize validates that a file size is within acceptable limits
func ValidateFileSize(size int64, maxSize int64) error {
	if size < 0 {
		return errors.New("file size cannot be negative")
	}

	if size == 0 {
		return errors.New("file size cannot be zero")
	}

	if maxSize > 0 && size > maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes", size, maxSize)
	}

	return nil
}

// ValidateMimeType validates that a MIME type is in the allowed list
// If allowedTypes is empty, any MIME type is accepted
func ValidateMimeType(mimeType string, allowedTypes []string) error {
	// Empty list means all types allowed
	if len(allowedTypes) == 0 {
		return nil
	}

	// Normalize mime type to lowercase
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))

	// Check if mime type is in allowed list
	for _, allowed := range allowedTypes {
		allowedLower := strings.ToLower(strings.TrimSpace(allowed))

		// Exact match
		if mimeType == allowedLower {
			return nil
		}

		// Wildcard match (e.g., "image/*")
		if strings.HasSuffix(allowedLower, "/*") {
			prefix := strings.TrimSuffix(allowedLower, "/*")
			if strings.HasPrefix(mimeType, prefix+"/") {
				return nil
			}
		}
	}

	return fmt.Errorf("mime type %s is not allowed", mimeType)
}

// IsValidFilename checks if a filename is valid without sanitizing
func IsValidFilename(filename string) bool {
	sanitized, err := SanitizeFilename(filename)
	return err == nil && sanitized == filename
}

// IsValidPath checks if a path is valid without sanitizing
func IsValidPath(path string) bool {
	_, err := SanitizePath(path)
	return err == nil
}

// ContainsControlCharacters checks if a string contains control characters
func ContainsControlCharacters(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// HasDangerousExtension checks if a filename has a potentially dangerous extension
func HasDangerousExtension(filename string) bool {
	// List of potentially dangerous extensions
	dangerousExtensions := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr",
		".vbs", ".js", ".jar", ".wsf", ".ps1", ".sh",
		".app", ".deb", ".rpm", ".dmg", ".pkg",
	}

	ext := strings.ToLower(filepath.Ext(filename))
	for _, dangerous := range dangerousExtensions {
		if ext == dangerous {
			return true
		}
	}

	return false
}

// NormalizePath normalizes a path to use forward slashes and removes redundant elements
func NormalizePath(path string) string {
	// Handle empty path as root
	if path == "" || path == "/" {
		return "/"
	}

	// Convert backslashes to forward slashes
	path = strings.ReplaceAll(path, "\\", "/")

	// Clean the path
	path = filepath.Clean(path)

	// Ensure it starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}
