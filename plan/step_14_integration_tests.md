# Step 14: Integration Tests

## Overview
Create comprehensive end-to-end integration tests that verify the entire application works correctly with all components integrated.

## Dependencies
- All steps 01-13 (requires complete application)

## Duration Estimate
60 minutes

## Agent Prompt

You are implementing Step 14 of the FilesOnTheGo project. Your task is to create comprehensive integration tests that verify the entire system works correctly.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
test: add comprehensive integration test suite
```

**Full commit message:**
```
test: add comprehensive integration test suite

Create end-to-end integration tests verifying complete application
workflows with all components integrated.

Includes:
- Test infrastructure with setup and cleanup helpers
- File upload/download flow tests with S3 verification
- Large file and multi-file upload tests
- Directory management flow tests with path updates
- Share flow tests (read, read_upload, upload_only)
- Password-protected share tests
- Share expiration and revocation tests
- Authentication and authorization flow tests
- Quota management and enforcement tests
- Concurrent operations tests with race detection
- Error recovery and failure handling tests
- Performance benchmarks for operations
- User deletion cascade tests
- Orphan cleanup tests
- Test data fixtures for common scenarios
- Integration test documentation
- MinIO/S3 test bucket configuration
- Database consistency verification
- Test coverage >= 80%

All tests passing
Race conditions: None detected with -race flag
```

Use this exact format when committing your work.

### Tasks

1. **Create Test Infrastructure (tests/integration/setup_test.go)**

   **Test Helper Functions:**
   ```go
   - setupTestApp() - Initialize test PocketBase instance
   - setupTestS3() - Initialize test S3 bucket (MinIO)
   - createTestUser() - Create and authenticate test user
   - createTestFile() - Upload a test file
   - createTestDirectory() - Create a test directory
   - createTestShare() - Create a test share link
   - cleanupTestData() - Remove all test data
   ```

   **Test App Wrapper:**
   ```go
   type TestApp struct {
       PB *pocketbase.PocketBase
       S3 *S3ServiceImpl
       BaseURL string
       Cleanup func()
   }
   ```

2. **Create File Upload/Download Flow Tests (tests/integration/file_flow_test.go)**

   **Test Complete Upload Flow:**
   1. Authenticate user
   2. Upload file
   3. Verify file in database
   4. Verify file in S3
   5. Verify quota updated
   6. Download file
   7. Verify content matches

   **Test Large File Upload:**
   1. Upload 100MB file
   2. Verify streaming works
   3. Verify no memory issues
   4. Verify quota accurate

   **Test Multi-File Upload:**
   1. Upload 10 files concurrently
   2. Verify all succeed
   3. Verify quota updated correctly

   **Test Upload to Directory:**
   1. Create directory structure
   2. Upload to specific directory
   3. Verify path correct
   4. Verify breadcrumb navigation

3. **Create Directory Management Flow Tests (tests/integration/directory_flow_test.go)**

   **Test Directory CRUD:**
   1. Create root directory
   2. Create nested directories
   3. List directory contents
   4. Rename directory
   5. Move directory
   6. Verify paths updated
   7. Delete directory (recursive)
   8. Verify S3 files deleted

   **Test Complex Directory Operations:**
   1. Create: /A/B/C/D structure
   2. Upload files to each level
   3. Move B to /X
   4. Verify all paths updated
   5. Delete A (recursive)
   6. Verify all files removed from S3
   7. Verify quota updated

4. **Create Share Flow Tests (tests/integration/share_flow_test.go)**

   **Test Read-Only Share:**
   1. Create file
   2. Create read-only share
   3. Access share (no auth)
   4. Verify can view file
   5. Verify can download
   6. Verify cannot upload
   7. Verify access logged

   **Test Read/Upload Share:**
   1. Create directory
   2. Create read_upload share
   3. Access share
   4. Download existing file
   5. Upload new file
   6. Verify new file in directory
   7. Verify quota updated

   **Test Upload-Only Share:**
   1. Create directory
   2. Create upload_only share
   3. Access share
   4. Verify cannot download
   5. Upload file
   6. Verify upload succeeds
   7. Verify still cannot download

   **Test Password-Protected Share:**
   1. Create file
   2. Create share with password
   3. Access share
   4. Verify password prompt
   5. Submit wrong password
   6. Verify denied
   7. Submit correct password
   8. Verify access granted

   **Test Share Expiration:**
   1. Create share expiring in 1 second
   2. Access immediately (succeeds)
   3. Wait 2 seconds
   4. Access again (denied)

   **Test Share Revocation:**
   1. Create share
   2. Access share (succeeds)
   3. Revoke share
   4. Access share (denied)

5. **Create Authentication Flow Tests (tests/integration/auth_flow_test.go)**

   **Test Registration:**
   1. Register new user
   2. Verify user created
   3. Verify default quota set
   4. Login with new user
   5. Verify token received

   **Test Login/Logout:**
   1. Login with credentials
   2. Verify token received
   3. Make authenticated request
   4. Logout
   5. Verify token invalidated

   **Test Permission Enforcement:**
   1. User A creates file
   2. User B tries to access
   3. Verify denied
   4. User B tries to delete
   5. Verify denied

6. **Create Quota Management Tests (tests/integration/quota_test.go)**

   **Test Quota Enforcement:**
   1. Create user with 1MB quota
   2. Upload 500KB file
   3. Verify quota 50% used
   4. Upload another 500KB
   5. Verify quota 100% used
   6. Try to upload 100KB
   7. Verify denied (quota exceeded)

   **Test Quota Updates on Delete:**
   1. Upload file
   2. Note quota usage
   3. Delete file
   4. Verify quota decremented
   5. Verify matches original

   **Test Quota Accuracy:**
   1. Upload multiple files
   2. Calculate total size
   3. Verify quota matches exactly
   4. Delete some files
   5. Verify quota still accurate

7. **Create Concurrent Operations Tests (tests/integration/concurrent_test.go)**

   **Test Concurrent Uploads:**
   1. Spawn 10 goroutines
   2. Each uploads different file
   3. Wait for all to complete
   4. Verify all files exist
   5. Verify quota accurate

   **Test Concurrent Share Access:**
   1. Create share
   2. Spawn 20 goroutines
   3. Each accesses share
   4. Verify all succeed
   5. Verify access count = 20

   **Test Race Conditions:**
   1. Test concurrent directory moves
   2. Test concurrent quota updates
   3. Test concurrent share creation
   4. Run with -race flag

8. **Create Error Recovery Tests (tests/integration/error_recovery_test.go)**

   **Test S3 Failure Handling:**
   1. Mock S3 failure during upload
   2. Verify no database record created
   3. Verify quota not updated
   4. Verify graceful error

   **Test Database Failure Handling:**
   1. Mock database failure
   2. Verify S3 file not uploaded
   3. Verify quota not updated

   **Test Partial Upload Cleanup:**
   1. Start upload
   2. Simulate connection failure
   3. Verify partial files cleaned up

9. **Create Performance Tests (tests/integration/performance_test.go)**

   **Benchmark Operations:**
   ```go
   func BenchmarkFileUpload(b *testing.B)
   func BenchmarkFileDownload(b *testing.B)
   func BenchmarkDirectoryListing(b *testing.B)
   func BenchmarkShareAccess(b *testing.B)
   ```

   **Load Tests:**
   - 100 concurrent file uploads
   - 1000 concurrent share accesses
   - Measure response times
   - Identify bottlenecks

10. **Create Cleanup and Migration Tests (tests/integration/cleanup_test.go)**

    **Test User Deletion Cascade:**
    1. Create user
    2. Upload files
    3. Create directories
    4. Create shares
    5. Delete user
    6. Verify all files deleted from S3
    7. Verify all database records deleted

    **Test Orphan Cleanup:**
    1. Simulate orphaned S3 files
    2. Run cleanup job
    3. Verify orphans removed

11. **Write Test Documentation**

    Create `tests/integration/README.md`:
    - How to run integration tests
    - Required setup (MinIO, etc.)
    - Test coverage goals
    - How to add new tests

12. **Create Test Data Fixtures (tests/fixtures/)**

    - Sample files of various types
    - Sample user data
    - Sample directory structures
    - Sample share configurations

### Success Criteria

- [ ] All integration tests pass
- [ ] Tests cover all major user flows
- [ ] Tests verify database and S3 consistency
- [ ] Concurrent operation tests pass
- [ ] Performance benchmarks established
- [ ] Error recovery tests pass
- [ ] Tests run in CI/CD pipeline
- [ ] Test coverage >= 80% overall
- [ ] No race conditions detected
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run all integration tests
go test ./tests/integration/... -v

# Run specific test file
go test ./tests/integration/file_flow_test.go -v

# Run with race detector
go test ./tests/integration/... -race

# Run benchmarks
go test ./tests/integration/... -bench=. -benchmem

# Run with coverage
go test ./tests/integration/... -cover -coverprofile=integration_coverage.out

# Generate coverage report
go tool cover -html=integration_coverage.out

# Run in parallel
go test ./tests/integration/... -parallel 4

# Run with timeout
go test ./tests/integration/... -timeout 10m
```

### Example Test Structure

```go
func TestCompleteFileUploadDownloadFlow(t *testing.T) {
    // Setup
    app := setupTestApp(t)
    defer app.Cleanup()

    user := createTestUser(t, app, "test@example.com")
    token := authenticateUser(t, app, "test@example.com", "password")

    // Upload file
    fileContent := []byte("test file content for integration test")
    fileName := "integration-test.txt"

    uploadResp := uploadFile(t, app, token, fileName, fileContent, nil)

    assert.Equal(t, http.StatusOK, uploadResp.StatusCode)

    var uploadResult UploadResponse
    json.Unmarshal(uploadResp.Body, &uploadResult)

    fileID := uploadResult.File.ID

    // Verify in database
    fileRecord, err := app.PB.Dao().FindFirstRecordByData("files", "id", fileID)
    assert.NoError(t, err)
    assert.Equal(t, fileName, fileRecord.GetString("name"))
    assert.Equal(t, int64(len(fileContent)), fileRecord.GetInt("size"))

    // Verify in S3
    s3Key := fileRecord.GetString("s3_key")
    exists, err := app.S3.FileExists(s3Key)
    assert.NoError(t, err)
    assert.True(t, exists)

    // Verify quota updated
    updatedUser := getUser(t, app, user.ID)
    assert.Equal(t, int64(len(fileContent)), updatedUser.StorageUsed)

    // Download file
    downloadResp := downloadFile(t, app, token, fileID)
    assert.Equal(t, http.StatusOK, downloadResp.StatusCode)

    // Verify content
    downloadedContent, _ := ioutil.ReadAll(downloadResp.Body)
    assert.Equal(t, fileContent, downloadedContent)
}

func TestShareFlow_UploadOnly_CannotDownload(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    // Setup owner
    owner := createTestUser(t, app, "owner@example.com")
    ownerToken := authenticateUser(t, app, "owner@example.com", "password")

    // Create directory
    dirID := createDirectory(t, app, ownerToken, "Shared Folder")

    // Upload file to directory
    fileContent := []byte("secret file")
    fileID := uploadFile(t, app, ownerToken, "secret.txt", fileContent, &dirID)

    // Create upload-only share
    shareToken := createShare(t, app, ownerToken, "directory", dirID, "upload_only")

    // Try to download file with share token (should fail)
    downloadURL := fmt.Sprintf("/api/public/share/%s/download/%s", shareToken, fileID)
    req := httptest.NewRequest("GET", downloadURL, nil)
    rec := httptest.NewRecorder()
    app.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusForbidden, rec.Code)

    // Upload new file with share token (should succeed)
    newFileContent := []byte("uploaded via share")
    uploadURL := fmt.Sprintf("/api/public/share/%s/upload", shareToken)

    body, contentType := createMultipartUpload("new-file.txt", newFileContent)
    req = httptest.NewRequest("POST", uploadURL, body)
    req.Header.Set("Content-Type", contentType)

    rec = httptest.NewRecorder()
    app.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)

    // Verify file uploaded
    files := listDirectoryFiles(t, app, ownerToken, dirID)
    assert.Equal(t, 2, len(files))
}

func TestConcurrentUploads_QuotaAccuracy(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    user := createTestUser(t, app, "test@example.com")
    token := authenticateUser(t, app, "test@example.com", "password")

    numFiles := 10
    fileSize := 1000 // bytes

    var wg sync.WaitGroup
    errors := make(chan error, numFiles)

    for i := 0; i < numFiles; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()

            fileName := fmt.Sprintf("file-%d.txt", index)
            content := make([]byte, fileSize)
            rand.Read(content)

            resp := uploadFile(t, app, token, fileName, content, nil)
            if resp.StatusCode != http.StatusOK {
                errors <- fmt.Errorf("upload %d failed with status %d", index, resp.StatusCode)
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // Check for errors
    for err := range errors {
        t.Error(err)
    }

    // Verify quota
    updatedUser := getUser(t, app, user.ID)
    expectedQuota := int64(numFiles * fileSize)
    assert.Equal(t, expectedQuota, updatedUser.StorageUsed)
}
```

### References

- CLAUDE.md: Testing Requirements
- Go testing package: https://pkg.go.dev/testing
- Testify: https://github.com/stretchr/testify

### Notes

- Use test database (separate from development)
- Use test S3 bucket (MinIO recommended)
- Clean up all test data after each test
- Use table-driven tests where applicable
- Mock external dependencies when appropriate
- Test both success and failure paths
- Verify database consistency
- Verify S3 consistency
- Test edge cases and boundary conditions
- Run tests in isolation (no dependencies between tests)
