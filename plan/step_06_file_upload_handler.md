# Step 06: File Upload Handler

## Overview
Implement the file upload handler with multipart form processing, S3 upload integration, database metadata storage, permission validation, and quota enforcement.

## Dependencies
- Step 02: S3 service (requires S3 operations)
- Step 03: Database models (requires File model)
- Step 04: Permission service (requires permission checks)

## Duration Estimate
60 minutes

## Agent Prompt

You are implementing Step 06 of the FilesOnTheGo project. Your task is to create a comprehensive file upload handler with security, quota enforcement, and tests.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
feat: implement file upload handler with quota enforcement
```

**Full commit message:**
```
feat: implement file upload handler with quota enforcement

Add complete file upload endpoint with multipart processing, S3
integration, security validation, and quota management.

Includes:
- File upload handler with POST /api/files/upload endpoint
- Multipart form data parsing and validation
- Authentication and share token support
- Permission validation using PermissionService
- Quota checking and enforcement
- Filename sanitization with path traversal prevention
- File size and MIME type validation
- Unique S3 key generation
- Database metadata storage
- User quota tracking and updates
- HTMX and JSON response support
- Comprehensive error handling for all scenarios
- Structured logging for upload operations
- Unit tests with mocking
- Integration tests with end-to-end flows
- Security tests for path traversal and injection
- Performance benchmarks

Test coverage: 80%+
All tests passing
Security: Input validation and sanitization validated
```

Use this exact format when committing your work.

### Tasks

1. **Create handlers/file_upload_handler.go**

   Define handler struct:
   ```go
   type FileUploadHandler struct {
       app              *pocketbase.PocketBase
       s3Service        services.S3Service
       permissionService services.PermissionService
   }
   ```

2. **Implement Upload Endpoint**

   `POST /api/files/upload`

   **Request:**
   - Multipart form data
   - Fields:
     - file: File data (required)
     - directory_id: Target directory (optional, root if empty)
     - path: Alternative to directory_id (optional)
   - Headers:
     - Authorization: Bearer {token} (required for authenticated)
     - share_token: Share token (optional, for shared directory upload)

   **Process:**
   1. Authenticate user OR validate share token
   2. Parse multipart form
   3. Validate file (size, type, filename)
   4. Check permissions (CanUploadFile)
   5. Check quota (CanUploadSize)
   6. Sanitize filename
   7. Generate unique file ID
   8. Generate S3 key
   9. Upload to S3
   10. Create database record
   11. Update user's storage_used
   12. Return success response or error

3. **Implement Validation Functions**

   **ValidateUploadRequest:**
   - Check max file size (from config)
   - Validate MIME type (optional whitelist)
   - Check filename length and characters
   - Prevent path traversal in filename

   **ValidateDirectoryAccess:**
   - Verify directory exists
   - Check user has access to directory
   - Handle root directory (nil parent)

4. **Implement Quota Management**

   **CheckAndReserveQuota:**
   - Get user's current quota usage
   - Check if upload would exceed limit
   - Temporarily reserve space
   - Release reservation on failure

   **UpdateQuotaUsage:**
   - Increment user's storage_used field
   - Use transaction to ensure consistency

5. **Implement Chunked Upload Support (Optional for MVP)**

   `POST /api/files/upload/chunk`
   - Support for resumable uploads
   - Track upload sessions
   - Merge chunks when complete
   - Clean up failed uploads

6. **Create Response Functions**

   **Success Response:**
   ```json
   {
     "success": true,
     "file": {
       "id": "file_id",
       "name": "filename.pdf",
       "size": 1024000,
       "mime_type": "application/pdf",
       "path": "/documents/filename.pdf",
       "created": "2025-11-21T10:00:00Z"
     }
   }
   ```

   **Error Response:**
   ```json
   {
     "success": false,
     "error": {
       "code": "QUOTA_EXCEEDED",
       "message": "Upload would exceed storage quota",
       "details": {
         "quota": 10737418240,
         "used": 10500000000,
         "requested": 500000000
       }
     }
   }
   ```

7. **Implement HTMX Response Handling**

   Detect `HX-Request` header:
   - If HTMX: Return HTML fragment (file list item)
   - If standard: Return JSON response
   - Include `HX-Trigger` header for client-side events

8. **Add Error Handling**

   Handle all error scenarios:
   - File too large
   - Quota exceeded
   - Invalid file type
   - S3 upload failure
   - Database error
   - Permission denied
   - Invalid directory
   - Duplicate filename (optional: auto-rename)

9. **Implement Logging and Metrics**

   Log all uploads:
   - User ID
   - File name and size
   - Upload duration
   - Success/failure
   - S3 key
   - Quota usage

10. **Write Comprehensive Tests**

    **Unit Tests (handlers/file_upload_handler_test.go):**
    - Test successful upload
    - Test authentication required
    - Test permission checks
    - Test quota enforcement
    - Test file size limits
    - Test invalid filenames
    - Test invalid MIME types
    - Test S3 upload failures
    - Test database errors

    **Integration Tests (tests/integration/file_upload_test.go):**
    - Test end-to-end file upload
    - Test upload to subdirectory
    - Test upload with share token
    - Test concurrent uploads
    - Test quota updates correctly
    - Test file appears in database
    - Test file exists in S3
    - Test HTMX vs JSON responses

    **Security Tests (tests/security/file_upload_test.go):**
    - Test path traversal prevention
    - Test filename injection attempts
    - Test MIME type spoofing
    - Test oversized file rejection
    - Test unauthorized upload attempts
    - Test share permission enforcement
    - Test upload-only share cannot download

    **Performance Tests:**
    - Benchmark small file upload (1KB)
    - Benchmark medium file upload (10MB)
    - Benchmark large file upload (100MB)
    - Test concurrent upload handling

    **Test Coverage:** 80%+ required

### Success Criteria

- [ ] File upload endpoint works
- [ ] Files uploaded to S3 successfully
- [ ] Database records created correctly
- [ ] Permissions enforced properly
- [ ] Quota checked and updated
- [ ] Filename sanitization prevents attacks
- [ ] File size limits enforced
- [ ] HTMX responses work
- [ ] Error handling comprehensive
- [ ] All tests pass
- [ ] Test coverage >= 80%
- [ ] Logging implemented
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run handler tests
go test ./handlers/... -run TestFileUpload -v

# Run integration tests
go test ./tests/integration/... -run TestFileUpload -v

# Run security tests
go test ./tests/security/... -run TestFileUpload -v

# Run with coverage
go test ./handlers/... ./tests/... -cover

# Test with real file
curl -X POST http://localhost:8090/api/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@testfile.pdf" \
  -F "directory_id=dir123"
```

### Example Test Structure

```go
func TestFileUploadHandler_Success(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileUploadHandler(app.PB(), app.S3(), app.Permissions())

    // Create test file
    fileContent := []byte("test file content")
    body, contentType := createMultipartUpload("test.txt", fileContent)

    // Create request
    req := httptest.NewRequest("POST", "/api/files/upload", body)
    req.Header.Set("Content-Type", contentType)
    req.Header.Set("Authorization", "Bearer "+testToken)

    // Execute
    rec := httptest.NewRecorder()
    handler.HandleUpload(rec, req)

    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)

    var response UploadResponse
    json.Unmarshal(rec.Body.Bytes(), &response)

    assert.True(t, response.Success)
    assert.Equal(t, "test.txt", response.File.Name)
    assert.Equal(t, int64(len(fileContent)), response.File.Size)

    // Verify file in S3
    exists, _ := app.S3().FileExists(response.File.S3Key)
    assert.True(t, exists)

    // Verify database record
    file, _ := app.PB().Dao().FindFirstRecordByData("files", "id", response.File.ID)
    assert.NotNil(t, file)
}

func TestFileUploadHandler_QuotaExceeded(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    // Create user with 1KB quota
    user := createTestUserWithQuota(t, app, 1024)
    // User already has 900 bytes used
    updateUserQuota(t, app, user.ID, 900)

    handler := NewFileUploadHandler(app.PB(), app.S3(), app.Permissions())

    // Try to upload 200 byte file (would exceed quota)
    fileContent := make([]byte, 200)
    body, contentType := createMultipartUpload("test.txt", fileContent)

    req := httptest.NewRequest("POST", "/api/files/upload", body)
    req.Header.Set("Content-Type", contentType)
    req.Header.Set("Authorization", "Bearer "+getUserToken(user))

    rec := httptest.NewRecorder()
    handler.HandleUpload(rec, req)

    assert.Equal(t, http.StatusForbidden, rec.Code)

    var response ErrorResponse
    json.Unmarshal(rec.Body.Bytes(), &response)

    assert.False(t, response.Success)
    assert.Equal(t, "QUOTA_EXCEEDED", response.Error.Code)
}

func TestFileUploadHandler_PathTraversal(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileUploadHandler(app.PB(), app.S3(), app.Permissions())

    maliciousFilenames := []string{
        "../../../etc/passwd",
        "..\\..\\..\\windows\\system32\\config\\sam",
        "file\x00.txt",
        "file\n.txt",
    }

    for _, filename := range maliciousFilenames {
        fileContent := []byte("malicious")
        body, contentType := createMultipartUpload(filename, fileContent)

        req := httptest.NewRequest("POST", "/api/files/upload", body)
        req.Header.Set("Content-Type", contentType)
        req.Header.Set("Authorization", "Bearer "+testToken)

        rec := httptest.NewRecorder()
        handler.HandleUpload(rec, req)

        // Should either sanitize or reject
        var response UploadResponse
        json.Unmarshal(rec.Body.Bytes(), &response)

        if response.Success {
            // If accepted, filename must be sanitized
            assert.NotContains(t, response.File.Name, "..")
            assert.NotContains(t, response.File.Name, "\x00")
            assert.NotContains(t, response.File.S3Key, "..")
        } else {
            // Or should be rejected with error
            assert.Contains(t, response.Error.Code, "INVALID")
        }
    }
}
```

### References

- DESIGN.md: File Management Endpoints section
- CLAUDE.md: Security Guidelines and Testing Requirements
- OWASP File Upload Security: https://owasp.org/www-community/vulnerabilities/Unrestricted_File_Upload

### Notes

- Always validate file extensions AND MIME types
- Generate unique S3 keys (never trust user input)
- Use streaming for large files to avoid memory issues
- Implement virus scanning hook for production
- Consider adding file deduplication (hash-based)
- Add upload progress tracking for large files
- Implement rate limiting per user
- Clean up S3 files if database insert fails
