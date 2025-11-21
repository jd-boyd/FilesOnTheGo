# Step 08: Directory Management

## Overview
Implement directory creation, deletion, navigation, listing, and move operations with proper permission enforcement and hierarchical path management.

## Dependencies
- Step 03: Database models (requires Directory model)
- Step 04: Permission service (requires permission checks)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 08 of the FilesOnTheGo project. Your task is to create comprehensive directory management functionality.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
feat: implement directory management with path tracking
```

**Full commit message:**
```
feat: implement directory management with path tracking

Add comprehensive directory operations with hierarchical path management,
breadcrumb navigation, and recursive operations.

Includes:
- Directory handler with create, read, update, delete operations
- Directory listing with files and subdirectories
- Breadcrumb path generation for navigation
- Full path calculation and tracking
- Directory rename with child path updates
- Directory move operations with circular reference prevention
- Recursive directory deletion with S3 cleanup
- Quota tracking updates on directory operations
- Root directory listing
- Permission enforcement for all operations
- Share token support for directory access
- HTMX and JSON response support
- Comprehensive error handling
- Unit tests for all operations
- Integration tests with path updates
- Security tests for circular references
- Performance tests for large directories

Test coverage: 80%+
All tests passing
Security: Circular reference prevention and path validation
```

Use this exact format when committing your work.

### Tasks

1. **Create handlers/directory_handler.go**

   Define handler struct:
   ```go
   type DirectoryHandler struct {
       app              *pocketbase.PocketBase
       permissionService services.PermissionService
   }
   ```

2. **Implement Create Directory Endpoint**

   `POST /api/directories`

   **Request:**
   ```json
   {
     "name": "New Folder",
     "parent_directory_id": "optional",
     "path": "optional alternative to parent_directory_id"
   }
   ```

   **Process:**
   1. Authenticate user OR validate share token
   2. Validate directory name (sanitize, check length)
   3. Check permissions (CanCreateDirectory)
   4. Check for duplicate name in same parent
   5. Calculate full path
   6. Create directory record
   7. Return created directory

3. **Implement List Directory Contents**

   `GET /api/directories/{directory_id}` or `GET /api/files?directory_id={id}`

   **Process:**
   1. Authenticate user OR validate share token
   2. Check permissions (CanReadDirectory)
   3. Get directory record
   4. List all subdirectories
   5. List all files
   6. Return combined list with metadata
   7. Include breadcrumb path

   **Response:**
   ```json
   {
     "directory": {
       "id": "dir123",
       "name": "Documents",
       "path": "/Documents"
     },
     "breadcrumbs": [
       {"id": null, "name": "Home", "path": "/"},
       {"id": "dir123", "name": "Documents", "path": "/Documents"}
     ],
     "items": [
       {
         "id": "dir456",
         "name": "Work",
         "type": "directory",
         "created": "2025-11-21T10:00:00Z"
       },
       {
         "id": "file789",
         "name": "report.pdf",
         "type": "file",
         "size": 1024000,
         "mime_type": "application/pdf",
         "created": "2025-11-21T10:00:00Z"
       }
     ]
   }
   ```

4. **Implement Delete Directory**

   `DELETE /api/directories/{directory_id}`

   **Query Params:**
   - recursive: true/false (default: false)

   **Process:**
   1. Authenticate user (no share delete allowed)
   2. Check ownership (CanDeleteDirectory)
   3. Check if directory is empty (if not recursive)
   4. If recursive:
      - Get all subdirectories and files
      - Delete all files from S3
      - Delete all database records
      - Update user quota
   5. Delete directory record
   6. Return success

5. **Implement Rename Directory**

   `PATCH /api/directories/{directory_id}`

   **Request:**
   ```json
   {
     "name": "New Name"
   }
   ```

   **Process:**
   1. Authenticate user
   2. Check ownership
   3. Validate new name
   4. Check for duplicates in parent
   5. Update directory name
   6. Update paths of all children (files and subdirs)
   7. Return updated directory

6. **Implement Move Directory**

   `PATCH /api/directories/{directory_id}/move`

   **Request:**
   ```json
   {
     "target_directory_id": "new_parent_id"
   }
   ```

   **Process:**
   1. Authenticate user
   2. Check ownership of source and target
   3. Prevent moving into own subtree (circular reference)
   4. Update parent_directory
   5. Recalculate paths for directory and all children
   6. Return updated directory

7. **Implement Root Directory Listing**

   `GET /api/files` or `GET /api/directories/root`

   **Process:**
   1. Authenticate user
   2. List all directories with null parent
   3. List all files with null parent_directory
   4. Return combined list

8. **Implement Breadcrumb Generation**

   Create service method:
   ```go
   func GetBreadcrumbs(directoryID string) ([]*Breadcrumb, error)
   ```

   **Process:**
   1. Start with current directory
   2. Walk up parent chain
   3. Build ordered list from root to current
   4. Return breadcrumb array

9. **Implement Path Calculation**

   Helper functions:
   ```go
   func CalculateFullPath(directoryID string) (string, error)
   func UpdateChildPaths(directoryID string) error
   ```

   **Process:**
   - Walk from root to directory
   - Concatenate names with '/'
   - Update all child records when parent path changes

10. **Write Comprehensive Tests**

    **Unit Tests (handlers/directory_handler_test.go):**
    - Test create directory
    - Test list directory contents
    - Test delete empty directory
    - Test delete recursive
    - Test rename directory
    - Test move directory
    - Test prevent circular move
    - Test breadcrumb generation
    - Test path calculation

    **Integration Tests (tests/integration/directory_test.go):**
    - Test create nested directories
    - Test move directory updates all paths
    - Test delete directory deletes files in S3
    - Test quota updates on directory delete
    - Test concurrent directory operations
    - Test directory with share access

    **Security Tests (tests/security/directory_test.go):**
    - Test unauthorized access blocked
    - Test path traversal in directory names
    - Test circular directory prevention
    - Test share permissions enforced

    **Test Coverage:** 80%+ required

### Success Criteria

- [ ] All directory endpoints implemented
- [ ] Create, read, update, delete work correctly
- [ ] Path calculations accurate
- [ ] Breadcrumbs generated correctly
- [ ] Move operations update all child paths
- [ ] Circular references prevented
- [ ] Recursive delete removes files from S3
- [ ] Permissions enforced
- [ ] All tests pass
- [ ] Test coverage >= 80%
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run directory handler tests
go test ./handlers/... -run TestDirectory -v

# Run integration tests
go test ./tests/integration/... -run TestDirectory -v

# Run security tests
go test ./tests/security/... -run TestDirectory -v

# Test creating directory
curl -X POST http://localhost:8090/api/directories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "New Folder"}'

# Test listing directory
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8090/api/directories/dir123
```

### Example Test Structure

```go
func TestDirectoryHandler_Create_Success(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewDirectoryHandler(app.PB(), app.Permissions())

    reqBody := `{"name": "Test Folder"}`
    req := httptest.NewRequest("POST", "/api/directories", strings.NewReader(reqBody))
    req.Header.Set("Authorization", "Bearer "+testToken)
    req.Header.Set("Content-Type", "application/json")

    rec := httptest.NewRecorder()
    handler.HandleCreate(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)

    var response DirectoryResponse
    json.Unmarshal(rec.Body.Bytes(), &response)

    assert.Equal(t, "Test Folder", response.Directory.Name)
    assert.Equal(t, "/Test Folder", response.Directory.Path)
}

func TestDirectoryHandler_Move_UpdatesPaths(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewDirectoryHandler(app.PB(), app.Permissions())

    // Create: /A/B/C structure
    dirA := createTestDirectory(t, app, "A", nil)
    dirB := createTestDirectory(t, app, "B", &dirA)
    dirC := createTestDirectory(t, app, "C", &dirB)
    file := createTestFile(t, app, "test.txt", dirC)

    // Create: /X
    dirX := createTestDirectory(t, app, "X", nil)

    // Move B under X (B's path: /A/B -> /X/B)
    reqBody := `{"target_directory_id": "` + dirX + `"}`
    req := httptest.NewRequest("PATCH", "/api/directories/"+dirB+"/move", strings.NewReader(reqBody))
    req.Header.Set("Authorization", "Bearer "+testToken)
    req.Header.Set("Content-Type", "application/json")

    rec := httptest.NewRecorder()
    handler.HandleMove(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)

    // Verify paths updated
    dirBRecord := getDirectory(t, app, dirB)
    assert.Equal(t, "/X/B", dirBRecord.Path)

    dirCRecord := getDirectory(t, app, dirC)
    assert.Equal(t, "/X/B/C", dirCRecord.Path)

    fileRecord := getFile(t, app, file)
    assert.Equal(t, "/X/B/C/test.txt", fileRecord.Path)
}

func TestDirectoryHandler_Move_PreventCircular(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewDirectoryHandler(app.PB(), app.Permissions())

    // Create: /A/B/C
    dirA := createTestDirectory(t, app, "A", nil)
    dirB := createTestDirectory(t, app, "B", &dirA)
    dirC := createTestDirectory(t, app, "C", &dirB)

    // Try to move A under C (would create circular reference)
    reqBody := `{"target_directory_id": "` + dirC + `"}`
    req := httptest.NewRequest("PATCH", "/api/directories/"+dirA+"/move", strings.NewReader(reqBody))
    req.Header.Set("Authorization", "Bearer "+testToken)
    req.Header.Set("Content-Type", "application/json")

    rec := httptest.NewRecorder()
    handler.HandleMove(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)

    var response ErrorResponse
    json.Unmarshal(rec.Body.Bytes(), &response)
    assert.Contains(t, response.Error.Message, "circular")
}

func TestDirectoryHandler_DeleteRecursive_RemovesFilesFromS3(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    handler := NewDirectoryHandler(app.PB(), app.Permissions())

    // Create directory with files
    dir := createTestDirectory(t, app, "ToDelete", nil)
    file1 := createAndUploadTestFile(t, app, "file1.txt", []byte("content1"), &dir)
    file2 := createAndUploadTestFile(t, app, "file2.txt", []byte("content2"), &dir)

    // Get S3 keys
    file1Record := getFile(t, app, file1)
    file2Record := getFile(t, app, file2)

    // Delete directory recursively
    req := httptest.NewRequest("DELETE", "/api/directories/"+dir+"?recursive=true", nil)
    req.Header.Set("Authorization", "Bearer "+testToken)

    rec := httptest.NewRecorder()
    handler.HandleDelete(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)

    // Verify files removed from S3
    exists1, _ := app.S3().FileExists(file1Record.S3Key)
    assert.False(t, exists1)

    exists2, _ := app.S3().FileExists(file2Record.S3Key)
    assert.False(t, exists2)

    // Verify quota updated
    user := getUser(t, app, testUserID)
    assert.Equal(t, int64(0), user.StorageUsed)
}
```

### References

- DESIGN.md: Directory Management section
- CLAUDE.md: Testing Requirements
- File system best practices

### Notes

- Prevent infinite loops in directory traversal
- Use transactions for multi-record updates
- Index path field for efficient lookups
- Consider caching breadcrumbs
- Implement pagination for large directories
- Add sorting options (name, date, size, type)
- Optimize recursive operations (use batch queries)
- Consider soft-delete with trash functionality
