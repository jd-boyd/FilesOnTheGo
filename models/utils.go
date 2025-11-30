package models

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/google/uuid"
)

// GenerateID generates a unique ID similar to PocketBase's ID format
// Uses a 15-character base62-like encoding
func GenerateID() string {
	// Generate a UUID v4
	id := uuid.New()

	// Convert to base64 and clean up
	encoded := base64.RawURLEncoding.EncodeToString(id[:])

	// Take first 15 characters and make it URL-safe
	if len(encoded) > 15 {
		encoded = encoded[:15]
	}

	return encoded
}

// GenerateToken generates a cryptographically secure random token
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base64 URL-safe string
	token := base64.RawURLEncoding.EncodeToString(bytes)

	// Remove any padding and take desired length
	token = strings.ReplaceAll(token, "=", "")
	if len(token) > length {
		token = token[:length]
	}

	return token, nil
}

// GenerateShareToken generates a secure share token
func GenerateShareToken() string {
	token, err := GenerateToken(32)
	if err != nil {
		// Fallback to UUID if random generation fails
		return strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	return token
}
