# Step 07: File Download Handler

## Overview
Implement the file download handler with permission validation, pre-signed URL generation, streaming support, and share link access.

## Dependencies
- Step 02: S3 service (requires S3 operations)
- Step 03: Database models (requires File model)
- Step 04: Permission service (requires permission checks)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 07 of the FilesOnTheGo project. Your task is to create a secure file download handler with pre-signed URLs and streaming support.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
feat: implement secure file download handler with pre-signed URLs
```

**Full commit message:**
```
feat: implement secure file download handler with pre-signed URLs

Add file download endpoint with S3 pre-signed URLs, streaming support,
batch downloads, and comprehensive access logging.

Includes:
- File download handler with GET /api/files/{file_id}/download
- Pre-signed S3 URL generation with configurable expiration
- Direct streaming option for bandwidth tracking
- Batch download with ZIP file creation
- Directory download with recursive file collection
- Share token access support with permission validation
- Upload-only share download prevention
- Access logging for audit trails
- Proper HTTP headers (Content-Type, Content-Disposition, etc.)
- Range request support for video streaming (optional)
- Thumbnail/preview generation (optional)
- Unit tests with mocking
- Integration tests with end-to-end flows
- Security tests for unauthorized access
- Performance tests for various file sizes

Test coverage: 80%+
All tests passing
Security: Permission validation and upload-only enforcement validated
```

Use this exact format when committing your work.

### Tasks

1. **Create handlers/file_download_handler.go**

   Define handler struct:
   ```go
   type FileDownloadHandler struct {
       app              *pocketbase.PocketBase
       s3Service        services.S3Service
       permissionService services.PermissionService
   }
   ```

2. **Implement Download Endpoint**

   `GET /api/files/{file_id}/download`

   **Request:**
   - Path params:
     - file_id: File identifier (required)
   - Query params:
     - share_token: Share token (optional)
     - inline: Display inline vs download (optional, boolean)
   - Headers:
     - Authorization: Bearer {token} (required if not using share)

   **Process:**
   1. Get file record from database
   2. Authenticate user OR validate share token
   3. Check permissions (CanReadFile)
   4. Validate share is not upload-only
   5. Generate pre-signed S3 URL (15 min expiry)
   6. Log download access
   7. Redirect to pre-signed URL OR stream file

3. **Implement Direct Streaming (Alternative to Redirect)**

   Option to stream through backend:
   - Useful for additional processing
   - Bandwidth tracking
   - Access logging
   - Watermarking (future)

   **Process:**
   1. Same permission checks
   2. Get file from S3
   3. Set proper headers (Content-Type, Content-Disposition)
   4. Stream to client
   5. Close S3 stream on completion

4. **Implement Batch Download (ZIP)**

   `POST /api/files/download/batch`

   **Request:**
   ```json
   {
     "file_ids": ["id1", "id2", "id3"],
     "directory_id": "optional"
   }
   ```

   **Process:**
   1. Validate all file access
   2. Create temporary ZIP file
   3. Stream files from S3 into ZIP
   4. Stream ZIP to client
   5. Clean up temporary file

5. **Implement Directory Download**

   `GET /api/directories/{directory_id}/download`

   **Process:**
   1. Check directory access
   2. Recursively get all files in directory
   3. Create ZIP with directory structure
   4. Stream to client

6. **Implement Thumbnail/Preview Generation (Optional)**

   `GET /api/files/{file_id}/preview`

   For images/PDFs:
   - Generate thumbnail on first access
   - Cache in S3 with different key
   - Return cached version on subsequent requests

7. **Create Response Headers**

   Set appropriate headers:
   - `Content-Type`: File MIME type
   - `Content-Disposition`: attachment or inline
   - `Content-Length`: File size
   - `ETag`: File checksum
   - `Cache-Control`: Caching policy
   - `X-Content-Type-Options: nosniff`

8. **Implement Access Logging**

   Log all downloads:
   - File ID and name
   - User ID or share token
   - IP address
   - User agent
   - Timestamp
   - Success/failure

   For share links:
   - Increment access_count
   - Create share_access_log entry

9. **Implement Range Requests (Optional)**

   Support HTTP Range header:
   - Enable video streaming
   - Resume interrupted downloads
   - Partial file downloads

10. **Write Comprehensive Tests**

    **Unit Tests (handlers/file_download_handler_test.go):**
    - Test successful download
    - Test permission checks
    - Test share token access
    - Test upload-only share denial
    - Test pre-signed URL generation
    - Test file not found
    - Test expired share
    - Test streaming vs redirect modes

    **Integration Tests (tests/integration/file_download_test.go):**
    - Test end-to-end download flow
    - Test download with authentication
    - Test download with share token
    - Test batch download creates valid ZIP
    - Test directory download
    - Test access logging works
    - Test pre-signed URLs are valid

    **Security Tests (tests/security/file_download_test.go):**
    - Test unauthorized access blocked
    - Test path traversal attempts
    - Test expired share blocked
    - Test upload-only share cannot download
    - Test file enumeration prevention
    - Test proper Content-Type prevents XSS

    **Test Coverage:** 80%+ required

### Success Criteria

- [ ] File download endpoint works
- [ ] Pre-signed URLs generated correctly
- [ ] Permissions enforced properly
- [ ] Share links work for read/read_upload
- [ ] Upload-only shares blocked from download
- [ ] Streaming works for large files
- [ ] Batch/directory download creates valid ZIP
- [ ] Access logging implemented
- [ ] Proper HTTP headers set
- [ ] All tests pass
- [ ] Test coverage >= 80%
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run handler tests
go test ./handlers/... -run TestFileDownload -v

# Run integration tests
go test ./tests/integration/... -run TestFileDownload -v

# Run security tests
go test ./tests/security/... -run TestFileDownload -v

# Test with curl
curl -L -o downloaded.pdf \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8090/api/files/file123/download

# Test with share token
curl -L -o shared.pdf \
  "http://localhost:8090/api/files/file123/download?share_token=abc123"
```

### Example Test Structure

```go
func TestFileDownloadHandler_Success(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileDownloadHandler(app.PB(), app.S3(), app.Permissions())

    // Create and upload test file
    fileID := createAndUploadTestFile(t, app, "test.pdf", []byte("PDF content"))

    // Create request
    req := httptest.NewRequest("GET", "/api/files/"+fileID+"/download", nil)
    req.Header.Set("Authorization", "Bearer "+testToken)

    // Execute
    rec := httptest.NewRecorder()
    handler.HandleDownload(rec, req)

    // Assert redirect to pre-signed URL
    assert.Equal(t, http.StatusFound, rec.Code)
    location := rec.Header().Get("Location")
    assert.Contains(t, location, "s3.amazonaws.com")
    assert.Contains(t, location, "X-Amz-Signature")
}

func TestFileDownloadHandler_ShareToken(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileDownloadHandler(app.PB(), app.S3(), app.Permissions())

    // Create file and share
    ownerID := "owner123"
    fileID := createTestFile(t, app, ownerID)
    shareToken := createTestShare(t, app, ownerID, fileID, "read")

    // Download with share token (no auth)
    req := httptest.NewRequest("GET", "/api/files/"+fileID+"/download?share_token="+shareToken, nil)

    rec := httptest.NewRecorder()
    handler.HandleDownload(rec, req)

    // Should succeed
    assert.Equal(t, http.StatusFound, rec.Code)

    // Verify access logged
    logs := getShareAccessLogs(t, app, shareToken)
    assert.Equal(t, 1, len(logs))
    assert.Equal(t, "download", logs[0].Action)
}

func TestFileDownloadHandler_UploadOnlyShare_Denied(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileDownloadHandler(app.PB(), app.S3(), app.Permissions())

    // Create file and upload-only share
    ownerID := "owner123"
    fileID := createTestFile(t, app, ownerID)
    shareToken := createTestShare(t, app, ownerID, fileID, "upload_only")

    // Try to download with upload-only share
    req := httptest.NewRequest("GET", "/api/files/"+fileID+"/download?share_token="+shareToken, nil)

    rec := httptest.NewRecorder()
    handler.HandleDownload(rec, req)

    // Should be forbidden
    assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestFileDownloadHandler_UnauthorizedAccess(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileDownloadHandler(app.PB(), app.S3(), app.Permissions())

    // Create file owned by user1
    fileID := createTestFile(t, app, "user1")

    // Try to access as user2
    user2Token := createTestUser(t, app, "user2@example.com")

    req := httptest.NewRequest("GET", "/api/files/"+fileID+"/download", nil)
    req.Header.Set("Authorization", "Bearer "+user2Token)

    rec := httptest.NewRecorder()
    handler.HandleDownload(rec, req)

    assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestFileDownloadHandler_BatchDownload_ZIP(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileDownloadHandler(app.PB(), app.S3(), app.Permissions())

    // Create multiple files
    file1 := createAndUploadTestFile(t, app, "file1.txt", []byte("content1"))
    file2 := createAndUploadTestFile(t, app, "file2.txt", []byte("content2"))

    // Request batch download
    reqBody := `{"file_ids": ["` + file1 + `", "` + file2 + `"]}`
    req := httptest.NewRequest("POST", "/api/files/download/batch", strings.NewReader(reqBody))
    req.Header.Set("Authorization", "Bearer "+testToken)
    req.Header.Set("Content-Type", "application/json")

    rec := httptest.NewRecorder()
    handler.HandleBatchDownload(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Equal(t, "application/zip", rec.Header().Get("Content-Type"))

    // Verify ZIP contents
    zipReader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
    assert.NoError(t, err)
    assert.Equal(t, 2, len(zipReader.File))
}

func TestFileDownloadHandler_PreventPathTraversal(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewFileDownloadHandler(app.PB(), app.S3(), app.Permissions())

    // Try various path traversal attempts
    attacks := []string{
        "../../../etc/passwd",
        "..%2F..%2Fetc%2Fpasswd",
        "....//....//etc/passwd",
    }

    for _, attack := range attacks {
        req := httptest.NewRequest("GET", "/api/files/"+attack+"/download", nil)
        req.Header.Set("Authorization", "Bearer "+testToken)

        rec := httptest.NewRecorder()
        handler.HandleDownload(rec, req)

        // Should return 404 or 400, not access actual files
        assert.NotEqual(t, http.StatusOK, rec.Code)
        assert.NotEqual(t, http.StatusFound, rec.Code)
    }
}
```

### References

- DESIGN.md: File Management Endpoints section
- CLAUDE.md: Security Guidelines and Testing Requirements
- HTTP Range Requests: RFC 7233
- Pre-signed URLs: AWS S3 documentation

### Notes

- Pre-signed URLs are more efficient than proxying
- Set appropriate expiration times (15 minutes default)
- Log all access for audit trails
- Implement rate limiting to prevent abuse
- Use proper Content-Disposition to prevent XSS
- Consider adding download quotas per share link
- Cache pre-signed URLs briefly to reduce S3 API calls
- Implement virus scanning check before allowing download
- For ZIP downloads, stream to avoid memory issues
