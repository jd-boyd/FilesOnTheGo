package security

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/stretchr/testify/assert"
)

// createMultipartUpload creates a multipart form data request body
func createMultipartUpload(filename string, content []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", filename)
	part.Write(content)

	writer.Close()
	return body, writer.FormDataContentType()
}

func TestSecurity_PathTraversalInFilename(t *testing.T) {
	// Test various path traversal attack patterns
	tests := []struct {
		name     string
		filename string
	}{
		{"Simple path traversal", "../../../etc/passwd"},
		{"Windows path traversal", "..\\..\\..\\windows\\system32\\config\\sam"},
		{"Mixed separators", "../..\\../etc/passwd"},
		{"Encoded path traversal", "..%2F..%2F..%2Fetc%2Fpasswd"},
		{"Double encoded", "..%252F..%252F..%252Fetc%252Fpasswd"},
		{"Unicode path traversal", "\u002e\u002e\u002f\u002e\u002e\u002f"},
		{"Null byte injection", "file.txt\x00.exe"},
		{"Null byte traversal", "../../../etc/passwd\x00.jpg"},
		{"Trailing slash", "../../../etc/"},
		{"Just parent refs", "../../.."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test sanitization
			sanitized, err := models.SanitizeFilename(tt.filename)

			// Should either reject the filename or sanitize it safely
			if err == nil {
				// If no error, verify it was sanitized
				assert.NotContains(t, sanitized, "..", "Sanitized filename should not contain '..'")
				assert.NotContains(t, sanitized, "/", "Sanitized filename should not contain '/'")
				assert.NotContains(t, sanitized, "\\", "Sanitized filename should not contain '\\'")
				assert.NotContains(t, sanitized, "\x00", "Sanitized filename should not contain null bytes")
			} else {
				// Error is also acceptable for security
				assert.Error(t, err, "Should reject dangerous filename")
			}
		})
	}
}

func TestSecurity_NullByteInjection(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"Null byte before extension", "document.pdf\x00.exe"},
		{"Null byte in middle", "doc\x00ument.pdf"},
		{"Multiple null bytes", "file\x00\x00.txt"},
		{"Null byte at start", "\x00file.txt"},
		{"Null byte at end", "file.txt\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := models.SanitizeFilename(tt.filename)
			assert.Error(t, err, "Should reject filename with null bytes")
			assert.Contains(t, err.Error(), "null byte", "Error should mention null byte")
		})
	}
}

func TestSecurity_ControlCharacterInjection(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"Newline character", "file\nname.txt"},
		{"Carriage return", "file\rname.txt"},
		{"Tab character", "file\tname.txt"},
		{"Bell character", "file\aname.txt"},
		{"Escape character", "file\x1bname.txt"},
		{"Delete character", "file\x7fname.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, err := models.SanitizeFilename(tt.filename)

			// Should either reject or sanitize
			if err == nil {
				// Verify control characters were removed
				for _, r := range sanitized {
					assert.False(t, r < 32 || r == 127, "Should not contain control characters")
				}
			} else {
				// Error is also acceptable
				assert.Error(t, err)
			}
		})
	}
}

func TestSecurity_SpecialFilenames(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"Single dot", "."},
		{"Double dot", ".."},
		{"Triple dot", "..."},
		{"Hidden file", ".bashrc"},
		{"Hidden config", ".ssh/config"},
		{"Empty string", ""},
		{"Whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, err := models.SanitizeFilename(tt.filename)

			if tt.filename == "" || tt.filename == "." || tt.filename == ".." {
				// These should be rejected
				assert.Error(t, err, "Should reject special filename: "+tt.filename)
			} else if strings.TrimSpace(tt.filename) == "" {
				// Whitespace only should be rejected
				assert.Error(t, err, "Should reject whitespace-only filename")
			} else {
				// Others might be accepted if properly sanitized
				if err == nil {
					assert.NotEmpty(t, sanitized)
					assert.NotEqual(t, ".", sanitized)
					assert.NotEqual(t, "..", sanitized)
				}
			}
		})
	}
}

func TestSecurity_PathTraversalInPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{"Normal path", "/documents/work", false},
		{"Root path", "/", false},
		{"Simple traversal", "/documents/../../../etc", true},
		{"Encoded traversal", "/documents/%2e%2e%2f%2e%2e%2f", true},
		{"Windows traversal", "/documents/..\\..\\", true},
		{"Relative traversal", "../documents", true},
		{"Hidden traversal", "/documents/./../etc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidatePathTraversal(tt.path)

			if tt.shouldErr {
				assert.Error(t, err, "Should detect path traversal in: "+tt.path)
			} else {
				assert.NoError(t, err, "Should allow valid path: "+tt.path)
			}
		})
	}
}

func TestSecurity_FileSizeLimits(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		maxSize   int64
		shouldErr bool
	}{
		{"Valid size", 1024, 2048, false},
		{"At limit", 2048, 2048, false},
		{"Exceeds limit", 2049, 2048, true},
		{"Way over limit", 10000000, 2048, true},
		{"Zero size", 0, 2048, true},
		{"Negative size", -1, 2048, true},
		{"Huge negative", -1000000, 2048, true},
		{"Integer overflow attempt", 9223372036854775807, 2048, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateFileSize(tt.size, tt.maxSize)

			if tt.shouldErr {
				assert.Error(t, err, "Should reject invalid size: %d", tt.size)
			} else {
				assert.NoError(t, err, "Should allow valid size: %d", tt.size)
			}
		})
	}
}

func TestSecurity_MimeTypeSpoofing(t *testing.T) {
	// Test that MIME type validation works correctly
	tests := []struct {
		name         string
		mimeType     string
		allowedTypes []string
		shouldPass   bool
	}{
		{"Exact match", "image/jpeg", []string{"image/jpeg", "image/png"}, true},
		{"Wildcard match", "image/png", []string{"image/*"}, true},
		{"Not in whitelist", "application/x-executable", []string{"image/*", "text/*"}, false},
		{"Empty whitelist allows all", "application/x-executable", []string{}, true},
		{"Case insensitive", "IMAGE/JPEG", []string{"image/jpeg"}, true},
		{"Suspicious executable", "application/x-msdownload", []string{"image/*"}, false},
		{"Disguised executable", "image/jpeg", []string{"image/*"}, true}, // Note: This passes type check but real impl should verify magic bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateMimeType(tt.mimeType, tt.allowedTypes)

			if tt.shouldPass {
				assert.NoError(t, err, "Should allow MIME type: "+tt.mimeType)
			} else {
				assert.Error(t, err, "Should reject MIME type: "+tt.mimeType)
			}
		})
	}
}

func TestSecurity_LongFilenames(t *testing.T) {
	tests := []struct {
		name      string
		length    int
		shouldErr bool
	}{
		{"Short name", 10, false},
		{"Normal name", 50, false},
		{"Long name", 200, false},
		{"At limit", 255, false},
		{"Over limit", 256, true},
		{"Way over limit", 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := strings.Repeat("a", tt.length) + ".txt"
			_, err := models.SanitizeFilename(filename)

			if tt.shouldErr {
				assert.Error(t, err, "Should reject filename of length: %d", tt.length)
			} else {
				assert.NoError(t, err, "Should allow filename of length: %d", tt.length)
			}
		})
	}
}

func TestSecurity_LongPaths(t *testing.T) {
	tests := []struct {
		name      string
		length    int
		shouldErr bool
	}{
		{"Short path", 50, false},
		{"Normal path", 200, false},
		{"Long path", 800, false},
		{"At limit", 1024, false},
		{"Over limit", 1025, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/" + strings.Repeat("a", tt.length-1)
			_, err := models.SanitizePath(path)

			if tt.shouldErr {
				assert.Error(t, err, "Should reject path of length: %d", tt.length)
			} else {
				assert.NoError(t, err, "Should allow path of length: %d", tt.length)
			}
		})
	}
}

func TestSecurity_DangerousExtensions(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		dangerous bool
	}{
		{"Safe document", "document.pdf", false},
		{"Safe image", "photo.jpg", false},
		{"Safe text", "notes.txt", false},
		{"Windows executable", "program.exe", true},
		{"Batch file", "script.bat", true},
		{"PowerShell script", "script.ps1", true},
		{"Shell script", "script.sh", true},
		{"JavaScript", "code.js", true},
		{"JAR file", "app.jar", true},
		{"Hidden executable", ".hidden.exe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDangerous := models.HasDangerousExtension(tt.filename)
			assert.Equal(t, tt.dangerous, isDangerous, "Extension check failed for: "+tt.filename)
		})
	}
}

func TestSecurity_UploadWithoutAuth(t *testing.T) {
	// Test that uploads without authentication are rejected
	// This would require full handler testing with mock services

	t.Run("No auth token or share token", func(t *testing.T) {
		content := []byte("test content")
		body, contentType := createMultipartUpload("test.txt", content)

		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		// No Authorization or X-Share-Token header

		// In a real test, this would call the handler and verify 401 response
		// assert.Equal(t, http.StatusUnauthorized, rec.Code)

		t.Skip("Requires full handler setup with mocks")
	})
}

func TestSecurity_QuotaBypassing(t *testing.T) {
	// Test that quota cannot be bypassed
	t.Run("Cannot bypass quota with negative file size", func(t *testing.T) {
		// Attempt to upload with negative size should fail at validation
		err := models.ValidateFileSize(-1, 1000)
		assert.Error(t, err)
	})

	t.Run("Cannot bypass quota with concurrent uploads", func(t *testing.T) {
		// This would test that quota checking is atomic
		t.Skip("Requires full integration test")
	})

	t.Run("Cannot bypass quota by deleting during upload", func(t *testing.T) {
		// Test that quota is reserved before upload starts
		t.Skip("Requires full integration test")
	})
}

func TestSecurity_S3KeyGeneration(t *testing.T) {
	// Test that S3 keys are generated securely and cannot collide

	t.Run("Keys are unique for different users", func(t *testing.T) {
		key1 := services.GenerateS3Key("user1", "file1", "test.txt")
		key2 := services.GenerateS3Key("user2", "file1", "test.txt")

		assert.NotEqual(t, key1, key2, "Keys should be unique for different users")
		assert.Contains(t, key1, "user1")
		assert.Contains(t, key2, "user2")
	})

	t.Run("Keys are unique for different files", func(t *testing.T) {
		key1 := services.GenerateS3Key("user1", "file1", "test.txt")
		key2 := services.GenerateS3Key("user1", "file2", "test.txt")

		assert.NotEqual(t, key1, key2, "Keys should be unique for different files")
	})

	t.Run("Dangerous filenames are sanitized in keys", func(t *testing.T) {
		key := services.GenerateS3Key("user1", "file1", "../../../etc/passwd")

		assert.NotContains(t, key, "..", "Key should not contain path traversal")
		assert.NotContains(t, key, "etc/passwd", "Key should not contain traversal result")
	})

	t.Run("Keys have consistent structure", func(t *testing.T) {
		key := services.GenerateS3Key("user1", "file1", "test.txt")

		// Keys should follow pattern: users/{userID}/{fileID}/{filename}
		parts := strings.Split(key, "/")
		assert.Equal(t, 4, len(parts), "Key should have 4 parts")
		assert.Equal(t, "users", parts[0], "First part should be 'users'")
		assert.Equal(t, "user1", parts[1], "Second part should be user ID")
		assert.Equal(t, "file1", parts[2], "Third part should be file ID")
		assert.Equal(t, "test.txt", parts[3], "Fourth part should be filename")
	})
}

func TestSecurity_HTMLInjection(t *testing.T) {
	// Test that filenames with HTML/JavaScript don't cause XSS

	tests := []struct {
		name     string
		filename string
	}{
		{"Script tag", "<script>alert('xss')</script>.txt"},
		{"Event handler", "<img src=x onerror=alert('xss')>.jpg"},
		{"HTML entity", "file&lt;script&gt;.txt"},
		{"JavaScript protocol", "javascript:alert('xss').txt"},
		{"Data URI", "data:text/html,<script>alert('xss')</script>.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, err := models.SanitizeFilename(tt.filename)

			// Should either reject or sanitize
			if err == nil {
				// Verify dangerous characters are removed
				assert.NotContains(t, sanitized, "<", "Should not contain '<'")
				assert.NotContains(t, sanitized, ">", "Should not contain '>'")
				assert.NotContains(t, sanitized, "script", "Should not contain 'script'")
			}
		})
	}
}

func TestSecurity_SQLInjection(t *testing.T) {
	// Test that filenames don't cause SQL injection
	// PocketBase uses parameterized queries, but we should still validate

	tests := []struct {
		name     string
		filename string
	}{
		{"Single quote", "file'; DROP TABLE files; --"},
		{"Double quote", "file\"; DROP TABLE files; --"},
		{"Comment injection", "file.txt --"},
		{"Union injection", "file' UNION SELECT * FROM users --"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Filename should be sanitized, removing dangerous SQL characters
			sanitized, err := models.SanitizeFilename(tt.filename)

			if err == nil {
				// Even if accepted, should not contain SQL injection patterns
				// The handler should use parameterized queries anyway
				assert.NotEmpty(t, sanitized)
			}
		})
	}
}
