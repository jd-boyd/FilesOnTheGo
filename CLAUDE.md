# CLAUDE.md - AI Development Guidelines

## Project Overview

**FilesOnTheGo** is a self-hosted file storage and sharing service built with Gin (Go web framework), GORM (Go ORM), S3-compatible storage, HTMX for frontend interactions, and Tailwind CSS for styling. This document provides guidelines for AI-assisted development to ensure consistency, quality, and maintainability.

## Core Principles

1. **Test-Driven Development**: Write tests for ALL new functionality before or alongside implementation
2. **Security First**: Validate all inputs, check permissions, and follow OWASP best practices
3. **Simplicity**: Prefer simple, readable code over clever abstractions
4. **Documentation**: Document complex logic, APIs, and non-obvious design decisions
5. **Incremental Development**: Build features in small, testable increments

## Technology Stack

### Backend
- **Gin**: Go web framework for HTTP routing and middleware
- **GORM**: Go ORM for database operations with SQLite
- **Go**: Primary backend language (version 1.24+)
- **SQLite**: Embedded database with GORM driver
- **JWT**: Token-based authentication with session management
- **AWS SDK for Go**: S3 integration

### Frontend
- **HTMX**: Hypermedia-driven dynamic interactions
- **Tailwind CSS**: Utility-first styling
- **Minimal JavaScript**: Only for essential features (drag-and-drop, clipboard)

### Storage
- **S3-Compatible Storage**: MinIO, AWS S3, Backblaze B2, etc.

### Testing
- **Go testing**: Standard library `testing` package
- **testify**: Assertions and mocking (`github.com/stretchr/testify`)
- **httptest**: HTTP handler testing
- **miniotest**: Mock S3 operations or use MinIO test server

## Project Structure

```
FilesOnTheGo/
├── main.go                    # Application entry point
├── database/                  # Database setup and migrations
│   └── database.go           # GORM database initialization
├── models/                    # GORM data models
│   ├── user.go               # User model
│   ├── file.go               # File model
│   ├── directory.go          # Directory model
│   ├── share.go              # Share model
│   └── share_access_log.go   # Share access logging
├── handlers_gin/              # Gin HTTP request handlers
│   ├── auth_handler.go       # Authentication routes
│   ├── file_upload_handler.go
│   ├── file_download_handler.go
│   ├── share_handler.go
│   ├── directory_handler.go
│   ├── settings_handler.go
│   ├── admin_handler.go
│   └── template_handler.go   # Template rendering
├── services/                  # Business logic layer
│   ├── s3_service.go         # S3 operations
│   ├── user_service.go       # User management
│   ├── share_service.go      # Share link management
│   ├── permission_service.go # Permission validation
│   └── metrics_service.go    # Metrics collection
├── auth/                      # Authentication system
│   ├── jwt.go                # JWT token management
│   └── session.go            # Session management
├── config/                    # Configuration management
│   └── config.go             # Environment variable handling
├── assets/                    # Embedded assets
│   ├── templates/            # HTMX HTML templates
│   │   ├── layouts/
│   │   ├── components/
│   │   └── pages/
│   └── static/               # CSS, JS, icons
├── tests/                     # Test files organized by package
│   ├── integration/
│   ├── unit/
│   └── fixtures/
├── go.mod
├── go.sum
├── DESIGN.md
├── CLAUDE.md                  # This file
└── README.md
```

## Testing Requirements

### Critical Rule: Tests Must Accompany All Code

**Every new feature, bug fix, or significant change MUST include tests.** This is non-negotiable.

### Test Types and When to Use Them

#### 1. Unit Tests
**Required for:**
- Business logic in `services/` package
- Utility functions and helpers
- Permission validation logic
- Share token generation
- Path sanitization and validation

**Example locations:**
- `services/s3_service_test.go`
- `services/permission_service_test.go`
- `services/share_service_test.go`

**Example test structure:**
```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestValidateFileAccess_Owner(t *testing.T) {
    // Arrange
    service := NewPermissionService()
    fileID := "test_file_123"
    userID := "owner_user_456"

    // Act
    result := service.ValidateFileAccess(fileID, userID, "", "download")

    // Assert
    assert.True(t, result, "Owner should have download access")
}

func TestValidateFileAccess_SharedReadOnly(t *testing.T) {
    // Arrange
    service := NewPermissionService()
    shareToken := "valid_share_token"

    // Act
    result := service.ValidateFileAccess("", "", shareToken, "download")

    // Assert
    assert.True(t, result, "Read-only share should allow download")
}

func TestValidateFileAccess_SharedReadOnlyDeniesUpload(t *testing.T) {
    // Arrange
    service := NewPermissionService()
    shareToken := "readonly_share_token"

    // Act
    result := service.ValidateFileAccess("", "", shareToken, "upload")

    // Assert
    assert.False(t, result, "Read-only share should deny upload")
}
```

#### 2. Integration Tests
**Required for:**
- API endpoint handlers
- Database operations with GORM
- S3 upload/download workflows
- Authentication flows
- Share link creation and access

**Example locations:**
- `tests/integration/file_upload_test.go`
- `tests/integration/share_access_test.go`
- `tests/integration/auth_test.go`

**Example test structure:**
```go
package integration

import (
    "bytes"
    "mime/multipart"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestFileUpload_Success(t *testing.T) {
    // Arrange
    app := setupTestApp(t)
    defer app.cleanup()

    token := app.authenticateUser("test@example.com", "password")
    body, contentType := createMultipartUpload("test.txt", []byte("content"))

    req := httptest.NewRequest("POST", "/api/files/upload", body)
    req.Header.Set("Content-Type", contentType)
    req.Header.Set("Authorization", "Bearer "+token)

    // Act
    resp := httptest.NewRecorder()
    app.ServeHTTP(resp, req)

    // Assert
    assert.Equal(t, http.StatusOK, resp.Code)
    // Verify file exists in S3
    // Verify metadata in database
}

func TestFileDownload_UnauthorizedUser(t *testing.T) {
    // Arrange
    app := setupTestApp(t)
    defer app.cleanup()

    fileID := app.createTestFile("owner@example.com")
    token := app.authenticateUser("other@example.com", "password")

    req := httptest.NewRequest("GET", "/api/files/"+fileID+"/download", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    // Act
    resp := httptest.NewRecorder()
    app.ServeHTTP(resp, req)

    // Assert
    assert.Equal(t, http.StatusForbidden, resp.Code)
}
```

#### 3. Security Tests
**Required for:**
- Permission validation edge cases
- Path traversal prevention
- Authentication bypass attempts
- Share link token security
- Input sanitization

**Example locations:**
- `tests/security/permission_test.go`
- `tests/security/path_traversal_test.go`

**Example test structure:**
```go
package security

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestPathTraversal_Blocked(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected bool
    }{
        {"Normal path", "documents/file.txt", true},
        {"Parent directory", "../../../etc/passwd", false},
        {"Encoded traversal", "..%2F..%2Fetc%2Fpasswd", false},
        {"Null byte injection", "file.txt\x00.jpg", false},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := sanitizePath(tc.input)
            assert.Equal(t, tc.expected, isValidPath(result))
        })
    }
}

func TestShareToken_BruteForceProtection(t *testing.T) {
    // Test rate limiting on share token attempts
}
```

#### 4. Performance Tests
**Required for:**
- Large file uploads (benchmark tests)
- Concurrent operations
- Directory listing with many files

**Example:**
```go
func BenchmarkFileUpload_1MB(b *testing.B) {
    app := setupTestApp(b)
    defer app.cleanup()

    data := make([]byte, 1024*1024) // 1MB
    token := app.authenticateUser("test@example.com", "password")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        uploadFile(app, token, "file.bin", data)
    }
}
```

### Test Coverage Requirements

- **Minimum coverage**: 80% for all packages
- **Critical paths**: 100% coverage for permission validation, authentication, and security-related code
- **Run coverage**: `make test coverage`
- **Generate report**: `make test coverage-report`

### Test Naming Conventions

Follow the pattern: `Test<FunctionName>_<Scenario>`

Examples:
- `TestValidateFileAccess_OwnerAlwaysHasAccess`
- `TestCreateShareLink_WithExpiration`
- `TestUploadFile_ExceedsQuota`
- `TestSanitizePath_PreventTraversal`

### Running Tests

```bash
# Run all tests with detailed summary (recommended)
make test

# Run with verbose output (for debugging)
make test-verbose

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run with coverage
make test-coverage

# Run specific package (use go test directly)
go test -v ./services/...

# Run specific test (use go test directly)
go test -v -run TestValidateFileAccess ./services/...

# Run security tests only (use go test directly)
go test -v ./tests/security/...

# Run with race detection
make race
```

## Development Workflow

### 1. Feature Development Process

When implementing a new feature:

1. **Review DESIGN.md** - Understand the architecture and requirements
2. **Write tests first** (TDD approach) or alongside implementation
3. **Implement minimal code** to pass tests
4. **Refactor** while keeping tests green
5. **Add integration tests** for end-to-end flows
6. **Update documentation** if behavior or APIs change
7. **Run full test suite** with `make test` before committing

### 2. Bug Fix Process

When fixing a bug:

1. **Write a failing test** that reproduces the bug
2. **Fix the code** to make the test pass
3. **Add regression tests** to prevent reoccurrence
4. **Verify fix** doesn't break existing functionality
5. **Document** the issue in code comments if non-obvious

### 3. Code Review Checklist

Before submitting code:

- [ ] All tests pass
- [ ] New tests added for new functionality
- [ ] Test coverage meets requirements (80%+)
- [ ] No security vulnerabilities introduced
- [ ] Input validation implemented
- [ ] Error handling is comprehensive
- [ ] Logging added for important operations
- [ ] Documentation updated (if applicable)
- [ ] Code follows Go best practices
- [ ] No hardcoded credentials or secrets

## Security Guidelines

### Authentication & Authorization

1. **Always validate authentication** on non-public endpoints
2. **Check permissions** before any file operation
3. **Use JWT tokens and session management** - don't roll your own auth
4. **Implement rate limiting** on sensitive endpoints

### Input Validation

```go
// Example: Validate and sanitize filename
func sanitizeFilename(filename string) (string, error) {
    // Remove path separators
    filename = filepath.Base(filename)

    // Check for null bytes
    if strings.Contains(filename, "\x00") {
        return "", errors.New("invalid filename: null byte detected")
    }

    // Check length
    if len(filename) > 255 {
        return "", errors.New("filename too long")
    }

    // Remove or replace dangerous characters
    filename = strings.Map(func(r rune) rune {
        if r < 32 || r == 127 { // Control characters
            return -1
        }
        return r
    }, filename)

    return filename, nil
}

// TEST THIS FUNCTION:
func TestSanitizeFilename_RemovesPathSeparators(t *testing.T) {
    result, err := sanitizeFilename("../../etc/passwd")
    assert.NoError(t, err)
    assert.Equal(t, "passwd", result)
}
```

### Permission Validation Pattern

```go
// Always follow this pattern for file operations
func (h *FileDownloadHandler) HandleDownload(c *gin.Context) error {
    fileID := c.Param("id")
    userID := c.GetString("user_id") // Gin context getter
    shareToken := c.Query("share_token")

    // Validate access
    if !h.permissionService.ValidateFileAccess(fileID, userID, shareToken, "download") {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return nil
    }

    // Proceed with download
    // ...
}

// WRITE TESTS FOR MULTIPLE SCENARIOS:
// - Owner access
// - Valid share token with read permission
// - Invalid share token
// - Expired share token
// - Wrong permission type (upload-only trying to download)
// - Unauthorized user
```

### S3 Security

1. **Generate short-lived pre-signed URLs** (5-15 minutes)
2. **Validate file types** before upload
3. **Use unique S3 keys** - never trust user input for paths
4. **Implement upload size limits** per user and per file
5. **Log all S3 operations** for audit trails

### Share Link Security

1. **Use cryptographically secure tokens** (UUID v4 or better)
2. **Validate expiration** on every access
3. **Hash passwords** with bcrypt (if password-protected)
4. **Log access attempts** for audit trails
5. **Implement rate limiting** to prevent token brute force

## HTMX Development Guidelines

### Server-Side Rendering

- Return HTML fragments from API endpoints when `HX-Request` header is present
- Return full pages for direct navigation
- Use `hx-target` to specify where content should be inserted
- Use `hx-swap` to control how content is inserted

### Example HTMX Endpoint

```go
func (h *DirectoryHandler) ListDirectory(c *gin.Context) error {
    directoryID := c.Query("directory_id")
    isHTMX := c.GetHeader("HX-Request") == "true"

    files, err := h.directoryService.GetFiles(directoryID)
    if err != nil {
        return err
    }

    if isHTMX {
        // Return HTML fragment for HTMX request
        return h.templateRenderer.HTML(c, http.StatusOK, "components/file-list.html", files)
    }

    // Return full page for direct navigation
    return h.templateRenderer.HTML(c, http.StatusOK, "pages/files.html", gin.H{
        "files": files,
    })
}
```

### HTMX Testing

```go
func TestFileList_HTMXRequest(t *testing.T) {
    app := setupTestApp(t)
    defer app.cleanup()

    req := httptest.NewRequest("GET", "/api/files?directory_id=123", nil)
    req.Header.Set("HX-Request", "true")
    req.Header.Set("Authorization", "Bearer "+token)

    resp := httptest.NewRecorder()
    app.ServeHTTP(resp, req)

    assert.Equal(t, http.StatusOK, resp.Code)
    assert.Contains(t, resp.Body.String(), "<div class=\"file-item\"")
}
```

## Error Handling

### Error Response Pattern

```go
// Define error types
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
}

// Use consistent error responses
func (h *BaseHandler) HandleError(c *gin.Context, err error, statusCode int) error {
    appErr := &AppError{
        Code:    "ERROR_CODE",
        Message: "User-friendly message",
        Detail:  err.Error(), // Only in development
    }

    // Log the error
    h.logger.Error().Err(err).Msg("Operation failed")

    c.JSON(statusCode, appErr)
    return nil
}
```

### Test Error Handling

```go
func TestFileUpload_QuotaExceeded(t *testing.T) {
    // Setup user with quota limit
    // Attempt to upload file that exceeds quota
    // Assert appropriate error response
}

func TestFileDownload_FileNotFound(t *testing.T) {
    // Request non-existent file
    // Assert 404 response
}
```

## Logging Guidelines

Use structured logging with appropriate levels:

```go
import "github.com/rs/zerolog/log"

// Info: Normal operations
log.Info().
    Str("user_id", userID).
    Str("file_id", fileID).
    Msg("File uploaded successfully")

// Warn: Unexpected but handled situations
log.Warn().
    Str("share_token", token).
    Msg("Expired share token accessed")

// Error: Operations that failed
log.Error().
    Err(err).
    Str("file_id", fileID).
    Msg("Failed to upload to S3")

// Debug: Detailed information for debugging
log.Debug().
    Str("path", filePath).
    Int64("size", fileSize).
    Msg("Processing file upload")
```

## Performance Considerations

### Large File Handling

- Use streaming for uploads/downloads
- Implement chunked uploads for files > 100MB
- Generate pre-signed URLs for direct S3 access
- Avoid loading entire files into memory

### Database Queries

- Use indexes on frequently queried fields (user_id, path, share_token)
- Paginate large result sets
- Use database transactions for multi-step operations
- Cache frequently accessed data (user quotas, share permissions)

### Example Streaming Upload

```go
func (s *S3Service) UploadStream(key string, reader io.Reader, size int64) error {
    uploader := s3manager.NewUploader(s.s3Client)

    _, err := uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:   reader,
    })

    return err
}

// TEST WITH MOCK
func TestS3Service_UploadStream(t *testing.T) {
    mockS3 := new(MockS3Client)
    service := NewS3Service(mockS3)

    data := bytes.NewReader([]byte("test data"))
    err := service.UploadStream("test-key", data, int64(data.Len()))

    assert.NoError(t, err)
    mockS3.AssertCalled(t, "Upload", mock.Anything)
}
```

## Git Commit Guidelines

### Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `docs`: Documentation changes
- `perf`: Performance improvements
- `security`: Security-related changes

**Examples:**
```
feat: implement upload-only share permission

Add support for upload-only share links that allow users to upload files
to a shared directory without being able to view or download existing files.

Includes:
- Permission validation logic
- API endpoint updates
- HTMX UI components
- Unit and integration tests

Closes #42
```

```
fix: prevent path traversal in file downloads

Sanitize file paths to prevent directory traversal attacks.
Added comprehensive security tests.

CVE: None (caught in development)
```

```
test: add integration tests for share link expiration

Covers:
- Expired share access denial
- Valid share access before expiration
- Edge cases around expiration timestamp
```

## Documentation Requirements

### Code Comments

```go
// ValidateFileAccess checks if a user or share token has permission to perform
// an action on a file. Returns true if access is granted, false otherwise.
//
// Parameters:
//   - fileID: The unique identifier of the file
//   - userID: The authenticated user ID (empty string if using share token)
//   - shareToken: The share link token (empty string if using user auth)
//   - action: The requested action ("read", "upload", "download", "delete")
//
// Permission precedence:
//   1. Owner always has full access
//   2. Share token permissions are validated if provided
//   3. Denied by default if no valid access method
func (s *PermissionService) ValidateFileAccess(fileID, userID, shareToken, action string) bool {
    // Implementation
}
```

### API Documentation

Document all endpoints with:
- Description
- Authentication requirements
- Request parameters
- Response format
- Example requests/responses
- Error codes

## Common Pitfalls to Avoid

1. **Don't trust user input** - Always validate and sanitize
2. **Don't skip permission checks** - Every operation must validate access
3. **Don't hardcode paths or URLs** - Use configuration
4. **Don't log sensitive data** - No passwords, tokens, or file contents
5. **Don't forget error handling** - Handle all error paths
6. **Don't skip tests** - Tests are not optional
7. **Don't commit secrets** - Use environment variables
8. **Don't block on I/O** - Use goroutines for concurrent operations
9. **Don't forget to close resources** - Use defer for cleanup
10. **Don't optimize prematurely** - Make it work, then make it fast

## Testing Best Practices Summary

1. **Write tests for every feature** - No exceptions
2. **Test both success and failure paths**
3. **Use table-driven tests** for multiple scenarios
4. **Mock external dependencies** (S3, database) in unit tests
5. **Use real dependencies in integration tests** (or test containers)
6. **Test edge cases and boundary conditions**
7. **Test security implications** - unauthorized access, invalid input
8. **Keep tests fast** - Unit tests < 100ms, integration tests < 5s
9. **Make tests deterministic** - No random failures
10. **Test one thing at a time** - Single responsibility per test

## Development Commands Reference

```bash
# Run application in development mode
go run main.go

# Run with external assets (for development)
go run main.go -external-assets -assets-dir .

# Run all tests with detailed summary (recommended)
make test

# Run tests with verbose output (for debugging)
make test-verbose

# Run tests with coverage
make test-coverage

# Run tests with race detection
make race

# Run specific test (use go test directly)
go test -v -run TestValidateFileAccess ./services/...

# Run specific package tests
go test -v ./services/...
go test -v ./models/...
go test -v ./handlers_gin/...

# Run benchmarks
make benchmark

# Format code
make fmt

# Lint code (requires golangci-lint)
make lint

# Update dependencies
go get -u ./...
go mod tidy

# Build for production
go build -o filesonthego main.go
```

## Questions to Ask Before Implementation

When starting a new feature, consider:

1. **What needs to be tested?** List all test scenarios first
2. **What are the security implications?** Permission checks, input validation
3. **What are the error cases?** How should errors be handled and communicated?
4. **What are the performance implications?** Will this scale with large datasets?
5. **How does this interact with existing features?** Integration points and dependencies
6. **Is this documented?** API changes, behavior changes, new endpoints
7. **Are there edge cases?** Boundary conditions, race conditions, concurrent access

## Final Reminders

- **Tests are mandatory** - Every change must include tests
- **Security first** - Always validate permissions and input
- **Keep it simple** - Readable code over clever code
- **Document the "why"** - Code shows "how", comments explain "why"
- **Review DESIGN.md** - Ensure alignment with architecture
- **Run tests before committing** with `make test` - No broken tests in version control
- **Think about the user** - Both API consumers and end users

---

**Document Version:** 2.0
**Last Updated:** 2025-12-01
**Author:** Joshua D. Boyd
