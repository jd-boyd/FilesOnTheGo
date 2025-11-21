# Step 09: Share Service Implementation

## Overview
Implement the share link service for creating, managing, and accessing shared files and directories with various permission levels, expiration, and password protection.

## Dependencies
- Step 03: Database models (requires Share model)
- Step 04: Permission service (requires permission checks)

## Duration Estimate
60 minutes

## Agent Prompt

You are implementing Step 09 of the FilesOnTheGo project. Your task is to create comprehensive share link functionality with security features.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
security: implement share service with password protection and expiration
```

**Full commit message:**
```
security: implement share service with password protection and expiration

Add comprehensive share link system with cryptographic tokens, password
protection, expiration, and rate limiting.

Includes:
- ShareService interface with complete share management
- Share creation with cryptographically secure UUID v4 tokens
- Password protection with bcrypt hashing (cost 12)
- Configurable expiration dates
- Share token validation with expiration checks
- Constant-time password comparison to prevent timing attacks
- Rate limiting (5 attempts/minute) against brute force
- Share revocation functionality
- User share listing and management
- Share expiration updates
- Access logging with IP, user agent, and action tracking
- Access count tracking per share
- Share handler with REST endpoints
- Public share access page
- Password validation endpoint
- Share URL generation
- Unit tests for all service methods
- Integration tests for complete flows
- Security tests for timing attacks and rate limiting
- Performance tests for high-load scenarios

Test coverage: 90%+
All tests passing
Security: Token generation, password hashing, rate limiting validated
```

Use this exact format when committing your work.

### Tasks

1. **Create services/share_service.go**

   Define `ShareService` interface:
   ```go
   type ShareService interface {
       CreateShare(params CreateShareParams) (*Share, error)
       GetShareByToken(token string) (*Share, error)
       ValidateShareAccess(token, password string) (*ShareAccessInfo, error)
       RevokeShare(shareID, userID string) error
       ListUserShares(userID, resourceType string) ([]*Share, error)
       UpdateShareExpiration(shareID, userID string, expiresAt time.Time) error
       GetShareAccessLogs(shareID, userID string) ([]*ShareAccessLog, error)
       LogShareAccess(shareID, action, fileName, ipAddress, userAgent string) error
   }
   ```

2. **Define Share Structs**

   ```go
   type CreateShareParams struct {
       UserID         string
       ResourceType   string // "file" or "directory"
       ResourceID     string
       PermissionType string // "read", "read_upload", "upload_only"
       Password       string // optional
       ExpiresAt      *time.Time // optional
   }

   type ShareAccessInfo struct {
       ShareID        string
       ResourceType   string
       ResourceID     string
       PermissionType string
       ExpiresAt      *time.Time
       IsValid        bool
       ErrorMessage   string
   }
   ```

3. **Implement CreateShare**

   **Process:**
   1. Validate user owns resource
   2. Validate permission type
   3. Generate unique share token (UUID v4)
   4. Hash password if provided (bcrypt)
   5. Set expiration if provided
   6. Create share record in database
   7. Return share with full URL

   **Security:**
   - Use cryptographically secure token generation
   - Hash passwords with bcrypt (cost 12)
   - Validate expiration is in future
   - Limit number of shares per resource (optional)

4. **Implement GetShareByToken**

   **Process:**
   1. Look up share by token
   2. Load related resource info
   3. Return share or error if not found

5. **Implement ValidateShareAccess**

   **Process:**
   1. Get share by token
   2. Check if share exists
   3. Check if expired
   4. Validate password if required
   5. Return access info

   **Security:**
   - Use constant-time password comparison
   - Implement rate limiting (max 5 attempts/minute)
   - Log failed access attempts
   - Return generic errors (don't reveal if token exists)

6. **Implement RevokeShare**

   **Process:**
   1. Verify user owns share
   2. Delete share record
   3. Optionally: soft-delete for audit trail
   4. Return success

7. **Implement ListUserShares**

   **Process:**
   1. Query shares by user
   2. Filter by resource type if specified
   3. Include resource details
   4. Order by created date (newest first)
   5. Return list

8. **Implement UpdateShareExpiration**

   **Process:**
   1. Verify user owns share
   2. Validate new expiration
   3. Update share record
   4. Return updated share

9. **Implement Share Access Logging**

   **LogShareAccess:**
   1. Create access log entry
   2. Increment share access_count
   3. Store IP, user agent, action, timestamp
   4. Use transaction for consistency

   **GetShareAccessLogs:**
   1. Verify user owns share
   2. Query access logs
   3. Order by accessed_at DESC
   4. Paginate results
   5. Return logs

10. **Create handlers/share_handler.go**

    Implement HTTP endpoints:
    - `POST /api/shares` - Create share
    - `GET /api/shares` - List user's shares
    - `GET /api/shares/{share_id}` - Get share details
    - `PATCH /api/shares/{share_id}` - Update share
    - `DELETE /api/shares/{share_id}` - Revoke share
    - `GET /api/shares/{share_id}/logs` - Get access logs
    - `GET /api/public/share/{share_token}` - Access shared resource
    - `POST /api/public/share/{share_token}/validate` - Validate password

11. **Implement Public Share Access**

    `GET /api/public/share/{share_token}`

    **Process:**
    1. Validate share token
    2. Check expiration
    3. If password-protected, show password prompt
    4. If valid, show resource:
       - File: Show file details with download button
       - Directory: Show file listing
    5. Respect permission type (hide download for upload-only)

12. **Implement Share URL Generation**

    ```go
    func GenerateShareURL(baseURL, token string) string {
        return fmt.Sprintf("%s/share/%s", baseURL, token)
    }
    ```

13. **Write Comprehensive Tests**

    **Unit Tests (services/share_service_test.go):**
    - Test create share generates unique token
    - Test password hashing
    - Test expiration validation
    - Test share token validation
    - Test password verification
    - Test revoke share
    - Test access logging
    - Test rate limiting

    **Integration Tests (tests/integration/share_test.go):**
    - Test full share creation flow
    - Test accessing shared file
    - Test accessing shared directory
    - Test password-protected share
    - Test expired share access denied
    - Test share revocation
    - Test access count increment
    - Test access logs created

    **Security Tests (tests/security/share_test.go):**
    - Test share token brute force protection
    - Test password timing attack protection
    - Test expired share blocked
    - Test permission enforcement (upload-only can't download)
    - Test unauthorized share modification
    - Test token enumeration prevention
    - Test share access without authentication

    **Test Coverage:** 90%+ required (security-critical)

### Success Criteria

- [ ] All share service methods implemented
- [ ] Share creation works with all options
- [ ] Token generation is cryptographically secure
- [ ] Password protection works
- [ ] Expiration enforcement works
- [ ] Access logging implemented
- [ ] Rate limiting prevents brute force
- [ ] All endpoints functional
- [ ] All tests pass
- [ ] Test coverage >= 90%
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run share service tests
go test ./services/... -run TestShare -v

# Run integration tests
go test ./tests/integration/... -run TestShare -v

# Run security tests
go test ./tests/security/... -run TestShare -v

# Create a share
curl -X POST http://localhost:8090/api/shares \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_type": "file",
    "resource_id": "file123",
    "permission_type": "read",
    "password": "optional_password",
    "expires_at": "2025-12-31T23:59:59Z"
  }'

# Access share
curl http://localhost:8090/api/public/share/abc-123-token
```

### Example Test Structure

```go
func TestShareService_CreateShare_GeneratesUniqueToken(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    service := NewShareService(app.PB())

    params := CreateShareParams{
        UserID:         testUserID,
        ResourceType:   "file",
        ResourceID:     "file123",
        PermissionType: "read",
    }

    share1, err := service.CreateShare(params)
    assert.NoError(t, err)

    share2, err := service.CreateShare(params)
    assert.NoError(t, err)

    assert.NotEqual(t, share1.ShareToken, share2.ShareToken)
    assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, share1.ShareToken)
}

func TestShareService_ValidateShareAccess_PasswordProtected(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    service := NewShareService(app.PB())

    // Create password-protected share
    params := CreateShareParams{
        UserID:         testUserID,
        ResourceType:   "file",
        ResourceID:     "file123",
        PermissionType: "read",
        Password:       "secret123",
    }

    share, _ := service.CreateShare(params)

    // Test with correct password
    info, err := service.ValidateShareAccess(share.ShareToken, "secret123")
    assert.NoError(t, err)
    assert.True(t, info.IsValid)

    // Test with wrong password
    info, err = service.ValidateShareAccess(share.ShareToken, "wrongpassword")
    assert.NoError(t, err)
    assert.False(t, info.IsValid)
    assert.Contains(t, info.ErrorMessage, "password")
}

func TestShareService_ValidateShareAccess_Expired(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    service := NewShareService(app.PB())

    // Create share that expired 1 hour ago
    expiresAt := time.Now().Add(-1 * time.Hour)
    params := CreateShareParams{
        UserID:         testUserID,
        ResourceType:   "file",
        ResourceID:     "file123",
        PermissionType: "read",
        ExpiresAt:      &expiresAt,
    }

    share, _ := service.CreateShare(params)

    info, err := service.ValidateShareAccess(share.ShareToken, "")
    assert.NoError(t, err)
    assert.False(t, info.IsValid)
    assert.Contains(t, info.ErrorMessage, "expired")
}

func TestShareService_ValidateShareAccess_RateLimiting(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    service := NewShareService(app.PB())

    share := createPasswordProtectedShare(t, app, "password123")

    // Make 5 attempts with wrong password
    for i := 0; i < 5; i++ {
        service.ValidateShareAccess(share.ShareToken, "wrongpassword")
    }

    // 6th attempt should be rate limited
    info, err := service.ValidateShareAccess(share.ShareToken, "wrongpassword")

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "rate limit")
}

func TestShareService_LogShareAccess_IncrementsCounter(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    service := NewShareService(app.PB())

    share := createTestShare(t, app)

    // Log access
    err := service.LogShareAccess(share.ID, "download", "file.txt", "192.168.1.1", "Mozilla/5.0")
    assert.NoError(t, err)

    // Get updated share
    updatedShare, _ := service.GetShareByToken(share.ShareToken)
    assert.Equal(t, 1, updatedShare.AccessCount)

    // Log another access
    service.LogShareAccess(share.ID, "view", "", "192.168.1.2", "Chrome/90")

    updatedShare, _ = service.GetShareByToken(share.ShareToken)
    assert.Equal(t, 2, updatedShare.AccessCount)
}

func TestShareService_RevokeShare_PreventsAccess(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    service := NewShareService(app.PB())

    share := createTestShare(t, app)

    // Verify share works before revocation
    info, _ := service.ValidateShareAccess(share.ShareToken, "")
    assert.True(t, info.IsValid)

    // Revoke share
    err := service.RevokeShare(share.ID, testUserID)
    assert.NoError(t, err)

    // Verify share no longer works
    info, _ = service.ValidateShareAccess(share.ShareToken, "")
    assert.False(t, info.IsValid)
}

// Security test: timing attack protection
func TestShareService_PasswordVerification_ConstantTime(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    service := NewShareService(app.PB())

    share := createPasswordProtectedShare(t, app, "correctpassword123")

    passwords := []string{"a", "co", "correc", "correctpasswor", "wrongpassword123"}
    times := []time.Duration{}

    for _, pwd := range passwords {
        start := time.Now()
        service.ValidateShareAccess(share.ShareToken, pwd)
        elapsed := time.Since(start)
        times = append(times, elapsed)
    }

    // Verify timing is consistent (prevents timing attacks)
    avgTime := averageDuration(times)
    for _, duration := range times {
        variance := math.Abs(float64(duration-avgTime)) / float64(avgTime)
        assert.Less(t, variance, 0.3, "Timing variance too high, vulnerable to timing attacks")
    }
}
```

### References

- DESIGN.md: Sharing & Permissions section
- CLAUDE.md: Security Guidelines
- OWASP: Broken Access Control
- UUID specification: RFC 4122

### Notes

- Use UUIDv4 for unpredictable tokens
- Implement rate limiting at application and database level
- Consider short-lived tokens for sensitive resources
- Log all share access for audit trails
- Implement share usage limits (max downloads per share)
- Consider adding email notifications for share access
- Support share expiration by date AND access count
- Implement share analytics (views, downloads, unique IPs)
