# Step 03: Database Models and Collections Setup

## Overview
Create PocketBase collections (database schema) for files, directories, and shares. Implement migrations and model helper functions.

## Dependencies
- Step 01: Project scaffolding (requires PocketBase setup)

## Duration Estimate
30 minutes

## Agent Prompt

You are implementing Step 03 of the FilesOnTheGo project. Your task is to create the database schema and models for FilesOnTheGo.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
feat: implement database schema and models with validation
```

**Full commit message:**
```
feat: implement database schema and models with validation

Create PocketBase collections, migrations, and model structs with
comprehensive validation and security features.

Includes:
- PocketBase migrations for all collections (files, directories, shares, users)
- Collection indexes for optimized queries
- Collection API rules for access control
- Model structs with helper methods (File, Directory, Share)
- Path sanitization functions with traversal prevention
- Filename validation with null byte and control character checks
- Share expiration and permission validation logic
- Breadcrumb navigation helpers
- 100% test coverage for validation functions
- Security tests for path traversal and injection attacks
- Integration tests for cascade deletes

Test coverage: 80%+ overall, 100% for validation
All tests passing
Security: Path traversal and injection prevention validated
```

Use this exact format when committing your work.

### Tasks

1. **Create migrations/001_initial_schema.go**

   Create migration to set up all collections:

   **Users Collection (extends built-in):**
   - Add custom fields:
     - storage_quota (number, default: 107374182400 = 100GB)
     - storage_used (number, default: 0)
     - is_admin (bool, default: false)

   **Directories Collection:**
   ```go
   {
     name: "string (required, max 255)",
     path: "string (required, indexed, max 1024)",
     user: "relation(users, required, cascade delete)",
     parent_directory: "relation(directories, optional, cascade delete)",
     created: "datetime",
     updated: "datetime"
   }
   ```

   **Files Collection:**
   ```go
   {
     name: "string (required, max 255)",
     path: "string (required, indexed, max 1024)",
     user: "relation(users, required, cascade delete)",
     parent_directory: "relation(directories, optional, cascade delete)",
     size: "number (required)",
     mime_type: "string",
     s3_key: "string (required, unique, indexed)",
     s3_bucket: "string (required)",
     checksum: "string (optional)",
     created: "datetime",
     updated: "datetime"
   }
   ```

   **Shares Collection:**
   ```go
   {
     user: "relation(users, required, cascade delete)",
     resource_type: "string (enum: file, directory)",
     file: "relation(files, optional, cascade delete)",
     directory: "relation(directories, optional, cascade delete)",
     share_token: "string (required, unique, indexed)",
     permission_type: "string (enum: read, read_upload, upload_only)",
     password_hash: "string (optional)",
     expires_at: "datetime (optional)",
     access_count: "number (default: 0)",
     created: "datetime",
     updated: "datetime"
   }
   ```

   **Share Access Logs Collection (optional but recommended):**
   ```go
   {
     share: "relation(shares, required, cascade delete)",
     ip_address: "string",
     user_agent: "string",
     action: "string (enum: view, download, upload)",
     file_name: "string (optional)",
     accessed_at: "datetime"
   }
   ```

2. **Create Collection Indexes**
   - Files: index on (user, path), (user, parent_directory), (s3_key)
   - Directories: index on (user, path), (user, parent_directory)
   - Shares: index on (share_token), (user), (expires_at)
   - Share Access Logs: index on (share, accessed_at)

3. **Set Collection Rules**

   For each collection, define API rules:

   **Files Collection:**
   - List: `@request.auth.id = user.id`
   - View: `@request.auth.id = user.id`
   - Create: `@request.auth.id = user.id`
   - Update: `@request.auth.id = user.id`
   - Delete: `@request.auth.id = user.id`

   **Directories Collection:**
   - List: `@request.auth.id = user.id`
   - View: `@request.auth.id = user.id`
   - Create: `@request.auth.id = user.id`
   - Update: `@request.auth.id = user.id`
   - Delete: `@request.auth.id = user.id`

   **Shares Collection:**
   - List: `@request.auth.id = user.id`
   - View: `@request.auth.id = user.id`
   - Create: `@request.auth.id = user.id`
   - Update: `@request.auth.id = user.id`
   - Delete: `@request.auth.id = user.id`

   **Share Access Logs Collection:**
   - List: `@request.auth.id = share.user.id`
   - View: `@request.auth.id = share.user.id`
   - Create: `""` (allow any, but through backend only)
   - Update: `""` (deny)
   - Delete: `""` (deny)

4. **Create models/file.go**
   - Define `File` struct matching the schema
   - Add helper methods:
     - `IsOwnedBy(userID string) bool`
     - `GetFullPath() string`
     - `Validate() error`

5. **Create models/directory.go**
   - Define `Directory` struct matching the schema
   - Add helper methods:
     - `IsOwnedBy(userID string) bool`
     - `GetFullPath() string`
     - `Validate() error`
     - `GetBreadcrumbs(app *pocketbase.PocketBase) ([]*Directory, error)`

6. **Create models/share.go**
   - Define `Share` struct matching the schema
   - Add helper methods:
     - `IsExpired() bool`
     - `IsPasswordProtected() bool`
     - `ValidatePassword(password string) bool`
     - `IncrementAccessCount(app *pocketbase.PocketBase) error`
     - `CanPerformAction(action string) bool`

7. **Create models/validation.go**
   - Implement sanitization functions:
     - `SanitizeFilename(filename string) (string, error)`
     - `SanitizePath(path string) (string, error)`
     - `ValidatePathTraversal(path string) error`
   - Implement validation functions:
     - `ValidateFileSize(size int64, maxSize int64) error`
     - `ValidateMimeType(mimeType string, allowedTypes []string) error`

8. **Write Comprehensive Tests**

   **migrations_test.go:**
   - Test migration up/down
   - Verify all collections created
   - Verify indexes created
   - Test cascade deletes work

   **models/file_test.go:**
   - Test File struct validation
   - Test helper methods
   - Test edge cases

   **models/directory_test.go:**
   - Test Directory struct validation
   - Test path construction
   - Test breadcrumb generation

   **models/share_test.go:**
   - Test share expiration logic
   - Test password validation
   - Test permission checking

   **models/validation_test.go:**
   - Test path traversal prevention
   - Test filename sanitization
   - Test null byte injection prevention
   - Test length limits
   - Test special character handling

   **Test Coverage:** 80%+ required, 100% for validation functions

### Success Criteria

**IMPORTANT: When you complete this step, update plan/PROGRESS.md to mark this step as completed and update the overall progress statistics.**

- [ ] All collections created in PocketBase
- [ ] Migrations run successfully
- [ ] Indexes created properly
- [ ] Collection rules configured
- [ ] Model structs defined
- [ ] Helper methods implemented
- [ ] Path sanitization prevents traversal attacks
- [ ] All tests pass
- [ ] Test coverage >= 80% (100% for validation)
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Run migration
go run main.go migrate up

# Verify collections in PocketBase Admin UI
# Access http://localhost:8090/_/

# Run tests
go test ./models/... -v
go test ./migrations/... -v

# Run with coverage
go test ./models/... -cover

# Run security tests specifically
go test ./models/... -run TestValidation
```

### Example Test Structure

```go
func TestSanitizeFilename_PreventPathTraversal(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"normal", "document.pdf", "document.pdf", false},
        {"parent dir", "../../../etc/passwd", "passwd", false},
        {"null byte", "file\x00.txt", "", true},
        {"too long", strings.Repeat("a", 300), "", true},
        {"control chars", "file\n\r\t.txt", "file.txt", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := SanitizeFilename(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}

func TestShare_IsExpired(t *testing.T) {
    share := &Share{
        ExpiresAt: time.Now().Add(-1 * time.Hour),
    }
    assert.True(t, share.IsExpired())

    share.ExpiresAt = time.Now().Add(1 * time.Hour)
    assert.False(t, share.IsExpired())

    share.ExpiresAt = time.Time{} // No expiration
    assert.False(t, share.IsExpired())
}
```

### References

- DESIGN.md: Data Models section
- CLAUDE.md: Security Guidelines and Testing Requirements
- PocketBase Collection API: https://pocketbase.io/docs/collections/

### Notes

- Use PocketBase migration system for all schema changes
- Implement proper cascade deletes to prevent orphaned data
- Add database constraints where possible
- Use indexes for frequently queried fields
- Document all collection rules
- Test with actual PocketBase instance, not just mocks
