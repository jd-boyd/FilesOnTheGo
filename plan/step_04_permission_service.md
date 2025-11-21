# Step 04: Permission Service Implementation

## Overview
Implement the permission validation service that checks user access rights for file operations, directory access, and share link permissions.

## Dependencies
- Step 01: Project scaffolding (requires project structure)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 04 of the FilesOnTheGo project. Your task is to create a comprehensive permission validation service with full security test coverage.

### Tasks

1. **Create services/permission_service.go**

   Define `PermissionService` interface:
   ```go
   type PermissionService interface {
       // File permissions
       CanReadFile(userID, fileID, shareToken string) (bool, error)
       CanUploadFile(userID, directoryID, shareToken string) (bool, error)
       CanDeleteFile(userID, fileID string) (bool, error)
       CanMoveFile(userID, fileID, targetDirID string) (bool, error)

       // Directory permissions
       CanReadDirectory(userID, directoryID, shareToken string) (bool, error)
       CanCreateDirectory(userID, parentDirID string) (bool, error)
       CanDeleteDirectory(userID, directoryID string) (bool, error)

       // Share permissions
       CanCreateShare(userID, resourceID, resourceType string) (bool, error)
       CanRevokeShare(userID, shareID string) (bool, error)

       // Share token validation
       ValidateShareToken(shareToken, password string) (*SharePermissions, error)

       // Quota checks
       CanUploadSize(userID string, fileSize int64) (bool, error)
       GetUserQuota(userID string) (*QuotaInfo, error)
   }
   ```

2. **Define Permission Structs**

   ```go
   type SharePermissions struct {
       ShareID        string
       ResourceType   string // "file" or "directory"
       ResourceID     string
       PermissionType string // "read", "read_upload", "upload_only"
       IsExpired      bool
       RequiresPassword bool
   }

   type QuotaInfo struct {
       TotalQuota int64
       UsedQuota  int64
       Available  int64
       Percentage float64
   }
   ```

3. **Implement PermissionServiceImpl**

   **Constructor:**
   ```go
   func NewPermissionService(app *pocketbase.PocketBase) *PermissionServiceImpl
   ```

   **File Permission Methods:**
   - `CanReadFile`: Check if user owns file OR has valid read/read_upload share
   - `CanUploadFile`: Check if user owns directory OR has read_upload/upload_only share
   - `CanDeleteFile`: Check if user owns file (shares cannot delete)
   - `CanMoveFile`: Check if user owns file AND target directory

   **Directory Permission Methods:**
   - `CanReadDirectory`: Check if user owns directory OR has valid share
   - `CanCreateDirectory`: Check if user owns parent directory OR has read_upload/upload_only share
   - `CanDeleteDirectory`: Check if user owns directory (shares cannot delete)

   **Share Permission Methods:**
   - `CanCreateShare`: Only resource owner can create shares
   - `CanRevokeShare`: Only share creator can revoke

   **Share Token Validation:**
   - `ValidateShareToken`:
     1. Look up share by token
     2. Check if share exists
     3. Check if expired
     4. Validate password if required
     5. Return SharePermissions struct

   **Quota Methods:**
   - `CanUploadSize`: Check if user has enough quota remaining
   - `GetUserQuota`: Return current quota usage and limits

4. **Implement Permission Matrix**

   Create helper function to check action permissions:
   ```go
   func (s *PermissionServiceImpl) validateShareAction(
       permissionType string,
       action string,
   ) bool {
       // Implement permission matrix from DESIGN.md
       // read: can view, download
       // read_upload: can view, download, upload
       // upload_only: can view names, upload (no download)
   }
   ```

5. **Implement Security Features**

   - **Rate Limiting**: Add protection against share token brute force
   - **Audit Logging**: Log all permission denials
   - **Password Verification**: Use constant-time comparison for passwords
   - **Token Validation**: Ensure tokens are valid UUIDs

6. **Create middleware/permission_middleware.go**

   Create middleware for common permission checks:
   ```go
   func RequireFileOwnership(ps PermissionService) echo.MiddlewareFunc
   func RequireDirectoryAccess(ps PermissionService) echo.MiddlewareFunc
   func RequireValidShare(ps PermissionService) echo.MiddlewareFunc
   ```

7. **Write Comprehensive Tests (services/permission_service_test.go)**

   **Unit Tests:**
   - Test each permission method with various scenarios
   - Test owner access (should always be granted)
   - Test unauthorized access (should be denied)
   - Test share access with each permission type
   - Test expired shares (should be denied)
   - Test password-protected shares
   - Test quota enforcement

   **Security Tests:**
   - Test path traversal attempts through permissions
   - Test permission escalation attempts
   - Test invalid share tokens
   - Test expired share access
   - Test concurrent share access
   - Test password timing attacks (ensure constant-time comparison)
   - Test rate limiting on share token validation

   **Edge Cases:**
   - Test deleted user scenarios
   - Test deleted resource scenarios
   - Test circular directory references
   - Test null/empty parameters
   - Test very long paths

   **Test Coverage:** 100% required (security-critical code)

### Success Criteria

- [ ] All permission methods implemented
- [ ] Permission matrix correctly enforced
- [ ] Share token validation works
- [ ] Password verification is secure (constant-time)
- [ ] Quota checking works
- [ ] Rate limiting implemented
- [ ] Audit logging added
- [ ] Middleware created
- [ ] All tests pass
- [ ] Test coverage = 100%
- [ ] Security tests comprehensive
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run permission service tests
go test ./services/... -run TestPermission -v

# Run with coverage
go test ./services/... -run TestPermission -cover

# Run security tests specifically
go test ./services/... -run TestPermission.*Security -v

# Run with race detector
go test ./services/... -race
```

### Example Test Structure

```go
func TestPermissionService_CanReadFile_Owner(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    ps := NewPermissionService(app.PB())

    userID := "user123"
    fileID := createTestFile(t, app, userID)

    canRead, err := ps.CanReadFile(userID, fileID, "")

    assert.NoError(t, err)
    assert.True(t, canRead, "Owner should be able to read their own file")
}

func TestPermissionService_CanReadFile_ValidShare(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    ps := NewPermissionService(app.PB())

    ownerID := "owner123"
    fileID := createTestFile(t, app, ownerID)
    shareToken := createTestShare(t, app, ownerID, fileID, "read")

    canRead, err := ps.CanReadFile("", fileID, shareToken)

    assert.NoError(t, err)
    assert.True(t, canRead, "Read share should allow file access")
}

func TestPermissionService_CanReadFile_ExpiredShare(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    ps := NewPermissionService(app.PB())

    ownerID := "owner123"
    fileID := createTestFile(t, app, ownerID)
    shareToken := createExpiredShare(t, app, ownerID, fileID)

    canRead, err := ps.CanReadFile("", fileID, shareToken)

    assert.NoError(t, err)
    assert.False(t, canRead, "Expired share should deny access")
}

func TestPermissionService_CanUploadFile_UploadOnlyShare(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    ps := NewPermissionService(app.PB())

    ownerID := "owner123"
    dirID := createTestDirectory(t, app, ownerID)
    shareToken := createTestShare(t, app, ownerID, dirID, "upload_only")

    canUpload, err := ps.CanUploadFile("", dirID, shareToken)

    assert.NoError(t, err)
    assert.True(t, canUpload, "Upload-only share should allow uploads")
}

func TestPermissionService_CanUploadSize_ExceedsQuota(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    ps := NewPermissionService(app.PB())

    userID := createTestUser(t, app, 1000) // 1KB quota

    canUpload, err := ps.CanUploadSize(userID, 2000) // Try to upload 2KB

    assert.NoError(t, err)
    assert.False(t, canUpload, "Should deny upload exceeding quota")
}

// Security test: constant-time password comparison
func TestPermissionService_ValidateShareToken_TimingAttack(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    ps := NewPermissionService(app.PB())

    shareToken := createPasswordProtectedShare(t, app, "correctpassword")

    // Measure time for incorrect passwords of varying similarity
    times := []time.Duration{}
    passwords := []string{"a", "co", "correc", "correctpasswor", "wrongpassword123"}

    for _, pwd := range passwords {
        start := time.Now()
        ps.ValidateShareToken(shareToken, pwd)
        elapsed := time.Since(start)
        times = append(times, elapsed)
    }

    // All timings should be similar (within reasonable variance)
    // This prevents timing attacks to guess passwords
    avgTime := averageDuration(times)
    for _, t := range times {
        variance := float64(t-avgTime) / float64(avgTime)
        assert.Less(t, math.Abs(variance), 0.5, "Timing variance too high")
    }
}
```

### References

- DESIGN.md: Permission System section
- CLAUDE.md: Security Guidelines and Testing Requirements
- OWASP Top 10: Access Control vulnerabilities

### Notes

- This is security-critical code - 100% test coverage required
- Use constant-time comparison for password checks
- Log all permission denials for security auditing
- Implement rate limiting to prevent brute force attacks
- Never expose detailed error messages that reveal system internals
- Cache permission checks when safe to do so
- Consider adding permission check metrics for monitoring
