# Step 15: Security Tests

## Overview
Create comprehensive security tests to verify the application is resistant to common vulnerabilities and follows security best practices.

## Dependencies
- All steps 01-13 (requires complete application)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 15 of the FilesOnTheGo project. Your task is to create thorough security tests covering OWASP Top 10 and other security concerns.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
security: add comprehensive security test suite covering OWASP Top 10
```

**Full commit message:**
```
security: add comprehensive security test suite covering OWASP Top 10

Create thorough security tests verifying application resistance to
common vulnerabilities and attack vectors.

Includes:
- Authentication security tests (bypass, token theft, session)
- Authorization tests (access control, privilege escalation)
- Path traversal prevention tests
- Null byte injection tests
- SQL injection prevention tests
- Command injection tests
- XSS prevention tests
- SSRF protection tests
- File upload security tests (size, MIME, malicious names)
- Share token security tests (unpredictability, enumeration)
- Password brute force protection with rate limiting
- Timing attack resistance tests
- Share expiration enforcement tests
- Session security tests (fixation, hijacking)
- Information disclosure prevention tests
- Cryptography tests (bcrypt, secure random)
- Rate limiting tests for all endpoints
- S3 security tests (access control, pre-signed URLs)
- Dependency vulnerability checks
- Security headers validation
- Security test documentation
- OWASP Top 10 coverage mapping

All tests passing
Security: No vulnerabilities detected
```

Use this exact format when committing your work.

### Tasks

1. **Create Authentication Security Tests (tests/security/auth_security_test.go)**

   **Test Authentication Bypass Attempts:**
   - Access protected endpoints without token
   - Use expired token
   - Use token from different user
   - Use malformed token
   - SQL injection in login
   - All should be denied

   **Test Session Security:**
   - Verify tokens expire correctly
   - Test token refresh mechanism
   - Verify logout invalidates tokens
   - Test concurrent session limits

   **Test Password Security:**
   - Verify passwords are hashed (bcrypt)
   - Test password strength requirements
   - Verify password reset security
   - Test timing attack resistance

2. **Create Authorization Security Tests (tests/security/authz_security_test.go)**

   **Test File Access Control:**
   - User A cannot access User B's files
   - User A cannot delete User B's files
   - User A cannot modify User B's files
   - User A cannot create shares for User B's files

   **Test Directory Access Control:**
   - User A cannot access User B's directories
   - User A cannot move files to User B's directories
   - User A cannot delete User B's directories

   **Test Permission Escalation:**
   - Read-only share cannot upload
   - Upload-only share cannot download
   - Non-owner cannot create shares
   - Non-owner cannot revoke shares

   **Test Horizontal Privilege Escalation:**
   - User cannot modify other user's quota
   - User cannot view other user's shares
   - User cannot access other user's logs

3. **Create Input Validation Tests (tests/security/input_validation_test.go)**

   **Test Path Traversal:**
   ```go
   - Filename: "../../../etc/passwd"
   - Filename: "..\\..\\..\\windows\\system32"
   - Filename: "%2e%2e%2f%2e%2e%2f"
   - Directory path: "../../../"
   - All should be sanitized or rejected
   ```

   **Test Null Byte Injection:**
   ```go
   - Filename: "file.txt\x00.jpg"
   - Path: "/path\x00/../etc/passwd"
   - Should be rejected
   ```

   **Test SQL Injection:**
   ```go
   - Username: "admin' OR '1'='1"
   - Filename: "'; DROP TABLE files; --"
   - Search: "' UNION SELECT * FROM users --"
   - Should be escaped/rejected
   ```

   **Test Command Injection:**
   ```go
   - Filename: "; rm -rf /"
   - Filename: "| cat /etc/passwd"
   - Should be sanitized
   ```

   **Test XSS in Filenames:**
   ```go
   - Filename: "<script>alert('XSS')</script>"
   - Filename: "javascript:alert(1)"
   - Should be sanitized in display
   ```

   **Test SSRF:**
   - S3 endpoint: "http://localhost:9000"
   - S3 endpoint: "http://169.254.169.254" (AWS metadata)
   - Should be validated

4. **Create File Upload Security Tests (tests/security/upload_security_test.go)**

   **Test File Size Limits:**
   - Upload file exceeding max size
   - Upload with forged Content-Length
   - Verify rejection

   **Test MIME Type Validation:**
   - Upload .exe as .jpg
   - Upload .php disguised as image
   - Verify validation

   **Test Malicious Filenames:**
   - Test long filenames (> 255 chars)
   - Test unicode exploitation
   - Test control characters
   - Test reserved names (CON, PRN on Windows)

   **Test Upload Bombs:**
   - Zip bomb detection (optional)
   - Decompression attacks
   - Billion laughs attack (XML)

   **Test Concurrent Upload Limits:**
   - Exceed concurrent upload limit
   - Verify rate limiting

5. **Create Share Security Tests (tests/security/share_security_test.go)**

   **Test Share Token Security:**
   - Verify tokens are unpredictable (UUIDv4)
   - Test token enumeration resistance
   - Verify no sequential patterns

   **Test Password Protection:**
   - Test password brute force protection
   - Verify rate limiting (max 5 attempts)
   - Test timing attack resistance
   - Verify constant-time comparison

   **Test Share Expiration:**
   - Verify expired shares are blocked
   - Test edge cases (exactly at expiration)
   - Verify cannot extend expired share

   **Test Share Permission Enforcement:**
   - Upload-only cannot download
   - Read-only cannot upload
   - Revoked shares cannot be accessed
   - Test permission changes take effect immediately

6. **Create Session Security Tests (tests/security/session_security_test.go)**

   **Test Session Fixation:**
   - Verify new session after login
   - Test session regeneration

   **Test Session Hijacking:**
   - Test token theft prevention
   - Verify CSRF protection
   - Test secure cookie flags

   **Test Concurrent Sessions:**
   - Test session limits per user
   - Verify old sessions invalidated

7. **Create Information Disclosure Tests (tests/security/info_disclosure_test.go)**

   **Test Error Messages:**
   - Verify no stack traces in production
   - Verify no database errors exposed
   - Verify no path information leaked
   - Generic error messages for auth failures

   **Test Enumeration Prevention:**
   - Username enumeration (login)
   - File ID enumeration
   - Share token enumeration
   - Directory enumeration

   **Test Metadata Leakage:**
   - Verify no server version in headers
   - Verify no framework version disclosed
   - Check HTTP headers (X-Powered-By, etc.)

8. **Create Cryptography Tests (tests/security/crypto_test.go)**

   **Test Password Hashing:**
   - Verify bcrypt used (not MD5/SHA1)
   - Verify proper cost factor (12+)
   - Test salt uniqueness

   **Test Share Token Generation:**
   - Verify cryptographically secure random
   - Test uniqueness
   - Verify sufficient entropy

   **Test HTTPS Enforcement:**
   - Verify redirect to HTTPS
   - Test HSTS headers
   - Verify secure cookies

9. **Create Rate Limiting Tests (tests/security/rate_limiting_test.go)**

   **Test Login Rate Limiting:**
   - Make 10 failed login attempts
   - Verify rate limiting kicks in
   - Verify lockout duration

   **Test Share Access Rate Limiting:**
   - Make 10 password attempts
   - Verify rate limiting
   - Test per-IP and per-token limits

   **Test Upload Rate Limiting:**
   - Rapid upload attempts
   - Verify throttling
   - Test per-user limits

   **Test API Rate Limiting:**
   - Rapid API requests
   - Verify 429 Too Many Requests
   - Test rate limit headers

10. **Create S3 Security Tests (tests/security/s3_security_test.go)**

    **Test S3 Access Control:**
    - Verify files not publicly accessible
    - Test bucket policy enforcement
    - Verify IAM permissions

    **Test Pre-signed URL Security:**
    - Verify URLs expire correctly
    - Test URL tampering detection
    - Verify signature validation

    **Test S3 Key Security:**
    - Verify keys are unpredictable
    - Test user isolation
    - Verify no key enumeration

11. **Create Dependency Security Tests (tests/security/dependency_test.go)**

    **Test Dependency Vulnerabilities:**
    - Run `go list -m all`
    - Check for known vulnerabilities
    - Verify dependencies are up-to-date

    **Test Supply Chain:**
    - Verify dependency checksums
    - Test go.sum integrity

12. **Create Security Headers Tests (tests/security/headers_test.go)**

    **Test Required Headers:**
    - X-Content-Type-Options: nosniff
    - X-Frame-Options: DENY
    - Content-Security-Policy
    - Strict-Transport-Security
    - X-XSS-Protection

    **Test CORS Configuration:**
    - Verify allowed origins
    - Test preflight requests
    - Verify credentials handling

13. **Write Security Test Documentation**

    Create `tests/security/README.md`:
    - Security testing methodology
    - OWASP Top 10 coverage
    - How to add new security tests
    - Vulnerability reporting process

### Success Criteria

- [ ] All security tests pass
- [ ] OWASP Top 10 covered
- [ ] Authentication/authorization tested
- [ ] Input validation comprehensive
- [ ] Rate limiting verified
- [ ] Cryptography tested
- [ ] No information disclosure
- [ ] Session security verified
- [ ] File upload security tested
- [ ] Share security tested
- [ ] S3 security tested
- [ ] Security headers configured
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run all security tests
go test ./tests/security/... -v

# Run specific security test
go test ./tests/security/auth_security_test.go -v

# Run with race detector
go test ./tests/security/... -race

# Run with coverage
go test ./tests/security/... -cover

# Check dependencies for vulnerabilities
go list -json -m all | nancy sleuth

# Or use govulncheck
govulncheck ./...

# Static analysis
golangci-lint run --enable=gosec

# Check for hardcoded secrets
gitleaks detect --source=.
```

### Example Test Structure

```go
func TestPathTraversal_FileUpload(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    token := authenticateTestUser(t, app)

    maliciousFilenames := []string{
        "../../../etc/passwd",
        "..\\..\\..\\windows\\system32\\config\\sam",
        "%2e%2e%2fetc%2fpasswd",
        "....//....//etc/passwd",
        ".\\..\\..\\..\\etc\\passwd",
    }

    for _, filename := range maliciousFilenames {
        t.Run(filename, func(t *testing.T) {
            content := []byte("malicious content")
            resp := uploadFile(t, app, token, filename, content, nil)

            // Should either sanitize or reject
            if resp.StatusCode == http.StatusOK {
                // If accepted, verify filename is sanitized
                var result UploadResponse
                json.Unmarshal(resp.Body, &result)

                assert.NotContains(t, result.File.Name, "..")
                assert.NotContains(t, result.File.S3Key, "..")
                assert.NotContains(t, result.File.Path, "..")
            } else {
                // Or should be rejected
                assert.Contains(t, []int{http.StatusBadRequest, http.StatusForbidden}, resp.StatusCode)
            }
        })
    }
}

func TestAuthorizationBypass_FileAccess(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    // Create User A and upload file
    userA := createTestUser(t, app, "userA@example.com")
    tokenA := authenticateUser(t, app, "userA@example.com", "password")
    fileID := uploadFile(t, app, tokenA, "private.txt", []byte("secret"), nil)

    // Create User B
    userB := createTestUser(t, app, "userB@example.com")
    tokenB := authenticateUser(t, app, "userB@example.com", "password")

    // User B tries to access User A's file
    req := httptest.NewRequest("GET", "/api/files/"+fileID+"/download", nil)
    req.Header.Set("Authorization", "Bearer "+tokenB)

    rec := httptest.NewRecorder()
    app.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusForbidden, rec.Code)

    // User B tries to delete User A's file
    req = httptest.NewRequest("DELETE", "/api/files/"+fileID, nil)
    req.Header.Set("Authorization", "Bearer "+tokenB)

    rec = httptest.NewRecorder()
    app.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSharePasswordBruteForce_RateLimited(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    // Create password-protected share
    token := authenticateTestUser(t, app)
    fileID := createTestFile(t, app, token)
    shareToken := createPasswordProtectedShare(t, app, token, fileID, "secretpassword")

    // Make 10 failed password attempts
    for i := 0; i < 10; i++ {
        req := httptest.NewRequest("POST", "/share/"+shareToken+"/validate",
            strings.NewReader(`{"password":"wrongpassword"}`))
        req.Header.Set("Content-Type", "application/json")

        rec := httptest.NewRecorder()
        app.ServeHTTP(rec, req)
    }

    // 11th attempt should be rate limited
    req := httptest.NewRequest("POST", "/share/"+shareToken+"/validate",
        strings.NewReader(`{"password":"wrongpassword"}`))
    req.Header.Set("Content-Type", "application/json")

    rec := httptest.NewRecorder()
    app.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestTimingAttack_PasswordVerification(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    shareToken := createPasswordProtectedShare(t, app, "correctpassword123")

    passwords := []string{
        "a",
        "co",
        "correc",
        "correctpasswor",
        "wrongpassword123",
    }

    times := []time.Duration{}

    for _, pwd := range passwords {
        start := time.Now()

        req := httptest.NewRequest("POST", "/share/"+shareToken+"/validate",
            strings.NewReader(fmt.Sprintf(`{"password":"%s"}`, pwd)))
        req.Header.Set("Content-Type", "application/json")

        rec := httptest.NewRecorder()
        app.ServeHTTP(rec, req)

        elapsed := time.Since(start)
        times = append(times, elapsed)
    }

    // Verify timing is consistent (prevents timing attacks)
    avgTime := averageDuration(times)
    for _, duration := range times {
        variance := math.Abs(float64(duration-avgTime)) / float64(avgTime)
        assert.Less(t, variance, 0.3, "Timing variance too high - vulnerable to timing attacks")
    }
}

func TestSQLInjection_Search(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    token := authenticateTestUser(t, app)

    sqlInjectionAttempts := []string{
        "' OR '1'='1",
        "'; DROP TABLE files; --",
        "' UNION SELECT * FROM users --",
        "admin'--",
        "' OR 1=1#",
    }

    for _, injection := range sqlInjectionAttempts {
        req := httptest.NewRequest("GET", "/api/files/search?q="+url.QueryEscape(injection), nil)
        req.Header.Set("Authorization", "Bearer "+token)

        rec := httptest.NewRecorder()
        app.ServeHTTP(rec, req)

        // Should not cause error or expose data
        assert.NotEqual(t, http.StatusInternalServerError, rec.Code)

        // Check database still intact
        _, err := app.PB.Dao().FindRecordsByExpr("files", dbx.NewExp("id != ''"))
        assert.NoError(t, err, "Database corrupted by SQL injection")
    }
}

func TestXSS_FilenameSanitization(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    token := authenticateTestUser(t, app)

    xssPayloads := []string{
        "<script>alert('XSS')</script>",
        "<img src=x onerror=alert('XSS')>",
        "javascript:alert(1)",
        "<svg/onload=alert('XSS')>",
    }

    for _, payload := range xssPayloads {
        fileID := uploadFile(t, app, token, payload+".txt", []byte("content"), nil)

        // Get file listing (HTML response)
        req := httptest.NewRequest("GET", "/files", nil)
        req.Header.Set("Authorization", "Bearer "+token)

        rec := httptest.NewRecorder()
        app.ServeHTTP(rec, req)

        html := rec.Body.String()

        // Verify XSS payload is escaped
        assert.NotContains(t, html, "<script>")
        assert.NotContains(t, html, "onerror=")
        assert.NotContains(t, html, "javascript:")
    }
}
```

### References

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- CLAUDE.md: Security Guidelines
- Go security best practices
- CWE Top 25: https://cwe.mitre.org/top25/

### Notes

- Run security tests regularly in CI/CD
- Use static analysis tools (gosec, golangci-lint)
- Check dependencies for vulnerabilities
- Perform penetration testing before production
- Set up bug bounty program
- Implement security monitoring and alerting
- Regular security audits
- Keep dependencies updated
- Follow principle of least privilege
- Defense in depth approach
