# FilesOnTheGo Design

This doc covers architecture, data models, and API design for FilesOnTheGo.

## Overview

FilesOnTheGo is a self-hosted file storage service. Files go to S3-compatible storage, metadata lives in a SQLite database managed by GORM. The goal is Nextcloud-like functionality with less complexity.

Key features:
- Upload/download files organized in directories
- Share links with different permission levels (read, read/upload, upload-only)
- Optional password protection and expiration on shares
- User quotas and multi-user support

## Architecture

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ HTTPS
       ▼
┌─────────────────────────────────────────┐
│         FilesOnTheGo Backend            │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │         Gin Web Framework        │  │
│  │  - HTTP Routing & Middleware     │  │
│  │  - Request/Response Handling     │  │
│  └──────────────────────────────────┘  │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │         GORM ORM                 │  │
│  │  - Database Models               │  │
│  │  - Migrations                    │  │
│  │  - Query Builder                 │  │
│  └──────────────────────────────────┘  │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │     Custom Services              │  │
│  │  - File Upload Handler           │  │
│  │  - Share Link Generator          │  │
│  │  - Permission Validator          │  │
│  │  - S3 Client Integration         │  │
│  │  - JWT Authentication            │  │
│  └──────────────────────────────────┘  │
└────────────┬────────────────────────────┘
             │
             ▼
    ┌────────────────┐
    │  S3 Storage    │
    │  (MinIO/AWS)   │
    └────────────────┘
```

## Tech Stack

**Backend:** Gin web framework + GORM ORM (Go). Provides HTTP routing, middleware, database models, and migrations.

**Database:** SQLite with GORM for migrations and queries.

**Storage:** S3-compatible (MinIO for dev, AWS/Backblaze/Wasabi for prod).

**Frontend:** HTMX + Tailwind. Server-rendered pages with dynamic updates. Minimal JS—just drag-and-drop and clipboard stuff.

**Libraries:** AWS SDK for Go (S3 ops), golang-jwt/jwt (JWT authentication), bcrypt (password hashing), go-uuid (share tokens).

## Data Models

### Users
GORM model for user authentication and management.

```go
type User struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Email     string    `gorm:"uniqueIndex;not null" json:"email"`
    Username  string    `gorm:"uniqueIndex;not null" json:"username"`
    Password  string    `gorm:"not null" json:"-"` // bcrypt hash, excluded from JSON
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Quota     int64     `json:"quota"`        // bytes
    Used      int64     `json:"used"`         // bytes used
    IsActive  bool      `gorm:"default:true" json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### Files
```go
type File struct {
    ID             uint      `gorm:"primaryKey" json:"id"`
    UserID         uint      `gorm:"not null;index" json:"user_id"`
    User           User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Name           string    `gorm:"not null" json:"name"`
    Path           string    `gorm:"not null" json:"path"` // e.g., "/documents/work/report.pdf"
    ParentID       *uint     `gorm:"index" json:"parent_id"` // null for root
    Parent         *Directory `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
    Size           int64     `gorm:"not null" json:"size"`
    MimeType       string    `json:"mime_type"`
    S3Key          string    `gorm:"uniqueIndex;not null" json:"s3_key"` // unique key in S3
    S3Bucket       string    `gorm:"not null" json:"s3_bucket"`
    Checksum       string    `json:"checksum"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

### Directories
```go
type Directory struct {
    ID         uint        `gorm:"primaryKey" json:"id"`
    UserID     uint        `gorm:"not null;index" json:"user_id"`
    User       User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Name       string      `gorm:"not null" json:"name"`
    Path       string      `gorm:"uniqueIndex;not null" json:"path"` // e.g., "/documents/work"
    ParentID   *uint       `gorm:"index" json:"parent_id"` // null for root
    Parent     *Directory  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
    CreatedAt  time.Time   `json:"created_at"`
    UpdatedAt  time.Time   `json:"updated_at"`

    // Associations
    Files      []File      `gorm:"foreignKey:ParentID" json:"files,omitempty"`
    Subdirs    []Directory `gorm:"foreignKey:ParentID" json:"subdirs,omitempty"`
}
```

### Shares
```go
type Share struct {
    ID             uint       `gorm:"primaryKey" json:"id"`
    UserID         uint       `gorm:"not null;index" json:"user_id"`
    User           User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
    ResourceType   string     `gorm:"not null;check:resource_type IN ('file','directory')" json:"resource_type"`
    FileID         *uint      `gorm:"index" json:"file_id,omitempty"`
    File           *File      `gorm:"foreignKey:FileID" json:"file,omitempty"`
    DirectoryID    *uint      `gorm:"index" json:"directory_id,omitempty"`
    Directory      *Directory `gorm:"foreignKey:DirectoryID" json:"directory,omitempty"`
    ShareToken     string     `gorm:"uniqueIndex;not null" json:"share_token"` // UUID for the link
    PermissionType string     `gorm:"not null;check:permission_type IN ('read','read_upload','upload_only')" json:"permission_type"`
    PasswordHash   *string    `json:"password_hash,omitempty"` // optional, bcrypt
    ExpiresAt      *time.Time `json:"expires_at,omitempty"` // optional
    AccessCount    int64      `gorm:"default:0" json:"access_count"`
    IsActive       bool       `gorm:"default:true" json:"is_active"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}
```

### S3 Key Structure

Files are stored as: `users/{user_id}/{file_id}/{filename}`

This keeps users isolated, makes files uniquely identifiable, and preserves original filenames.

## API

### Auth (Custom JWT)
- `POST /api/auth/login` - Login (email + password)
- `POST /api/auth/register` - Register user
- `POST /api/auth/refresh` - Refresh JWT token
- `POST /api/auth/logout` - Logout (revoke token)

### Files

**Upload**
```
POST /api/files/upload
Authorization: Bearer {token}
Body: multipart/form-data (file, directory_id)

Response: { file: { id, name, path, size, mime_type, created } }
```

**List**
```
GET /api/files?directory_id={id}
Authorization: Bearer {token}

Response: {
  items: [{ id, name, path, is_directory, size, ... }],
  current_path: string,
  parent_directory: string
}
```

**Download**
```
GET /api/files/{file_id}/download
Authorization: Bearer {token}

Response: Redirect to pre-signed S3 URL
```

**Delete**
```
DELETE /api/files/{file_id}
Authorization: Bearer {token}
```

**Move/Rename**
```
PATCH /api/files/{file_id}
Body: { name, directory_id }
```

### Directories

**Create**
```
POST /api/directories
Body: { name, parent_directory_id }
```

**Delete**
```
DELETE /api/directories/{id}?recursive=true
```

### Shares

**Create share link**
```
POST /api/shares
Body: {
  resource_type: "file" | "directory",
  resource_id: string,
  permission_type: "read" | "read_upload" | "upload_only",
  password: string,       // optional
  expires_at: datetime    // optional
}

Response: { share: { id, share_token, share_url, ... } }
```

**Access shared resource**
```
GET /api/public/share/{share_token}?password={password}

Response: { resource_type, name, permission_type, items }
```

**Upload to share**
```
POST /api/public/share/{share_token}/upload
Body: multipart/form-data (file, password)
```

**Download from share**
```
GET /api/public/share/{share_token}/download/{file_id}?password={password}
```

**List user's shares**
```
GET /api/shares
Authorization: Bearer {token}
```

**Revoke share**
```
DELETE /api/shares/{share_id}
Authorization: Bearer {token}
```

## Permissions

| Action | Private | Read | Read/Upload | Upload-Only | Owner |
|--------|---------|------|-------------|-------------|-------|
| View metadata | ❌ | ✅ | ✅ | ✅ (names only) | ✅ |
| Download | ❌ | ✅ | ✅ | ❌ | ✅ |
| Upload | ❌ | ❌ | ✅ | ✅ | ✅ |
| Delete | ❌ | ❌ | ❌ | ❌ | ✅ |
| Create directory | ❌ | ❌ | ✅ | ✅ | ✅ |
| Manage shares | ❌ | ❌ | ❌ | ❌ | ✅ |

Permission checks happen in this order:
1. Is user authenticated? (for non-public endpoints)
2. Is user the owner? → full access
3. Is there a valid share token? → check permission level
4. Deny by default

## S3 Integration

**Upload flow:**
1. Client uploads to API
2. Backend validates auth + permissions
3. Generate S3 key: `users/{user_id}/{file_id}/{filename}`
4. Stream to S3
5. Save metadata to database
6. Return success

**Download flow:**
1. Client requests download
2. Validate permissions
3. Generate pre-signed URL (5-15 min expiry)
4. Redirect client to S3

Pre-signed URLs mean downloads go directly from S3 to the client—no proxying through the backend.

## Security Notes

**Auth:** Use custom JWT implementation with golang-jwt/jwt. Validate on every request. Rate limit sensitive endpoints.

**Uploads:** Validate file types and sizes. Sanitize filenames (strip path components, special chars). Enforce per-user quotas.

**Shares:** Generate tokens with UUID v4. Hash passwords with bcrypt. Log access attempts. Support expiration and revocation.

**S3:** Use IAM with minimal permissions. Enable bucket versioning. Block public access. Use server-side encryption.

## UI Pages

1. **File browser** - List files/folders, breadcrumb nav, upload button, drag-and-drop
2. **Context menu** - Download, rename, move, delete, share
3. **Share dialog** - Permission picker, password toggle, expiration date, copy link
4. **Public share page** - File list (if directory), download/upload buttons, password prompt
5. **Settings** - Account info, storage usage

HTMX handles dynamic updates. Server returns HTML fragments. Minimal JS for drag-and-drop and clipboard.

## Deployment

Single server setup:
```
Nginx (HTTPS termination)
    ↓
FilesOnTheGo (standalone Go binary, port 8090)
    ↓
SQLite (embedded, managed by GORM)
    ↓
S3 Storage (over network)
```

Container deployments are supported—see deployment docs.

## Configuration

Configuration can be set via YAML file, environment variables, or both. Environment variables always override YAML values.

**Config file locations** (searched in order):
1. `./config.yaml`
2. `./config.yml`
3. `/etc/filesonthego/config.yaml`
4. `/etc/filesonthego/config.yml`

**YAML example** (`config.yaml`):
```yaml
s3_endpoint: http://localhost:9000
s3_bucket: filesonthego
s3_access_key: minioadmin
s3_secret_key: minioadmin
s3_region: us-east-1
s3_use_ssl: false

app_port: "8090"
app_environment: development
app_url: http://localhost:8090

database_url: ./filesonthego.db
max_upload_size: 104857600  # 100MB
jwt_secret: change-me-in-production

public_registration: true
default_user_quota: 10737418240  # 10GB
```

**Environment variables** (override YAML):
```bash
# Required
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=filesonthego
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
APP_URL=https://files.example.com

# Optional
APP_ENVIRONMENT=production
APP_PORT=8090
S3_REGION=us-east-1
S3_USE_SSL=true
MAX_UPLOAD_SIZE=5368709120         # 5GB
DEFAULT_USER_QUOTA=107374182400    # 100GB
PUBLIC_REGISTRATION=false
JWT_SECRET=your-secret-here        # Required in production
```

See `config.yaml.example` and `.env.example` for all options.

## Implementation Roadmap

The project is organized into **16 detailed implementation steps** optimized for 4 parallel agents, with comprehensive testing and security requirements.

### Progress Summary
- **Completed:** 11/16 steps (69%)
- **Remaining:** 5 steps (31%)
- **Total Estimated Time:** ~10.5 hours with 4 parallel agents

### Dependency Groups

**Group 1: Foundation (30 min)**
- ✅ Step 01: Project scaffolding and PocketBase setup

**Group 2: Core Services (45 min each, run in parallel)**
- ✅ Step 02: S3 service implementation
- ✅ Step 03: Database models and collections setup
- ✅ Step 04: Permission service implementation
- ✅ Step 05: Basic HTMX UI layout

**Group 3: Business Logic (60 min each, run in parallel)**
- ✅ Step 06: File upload handler
- ✅ Step 07: File download handler
- ✅ Step 08: Directory management
- ✅ Step 09: Share service implementation

**Group 4: Frontend Components (45 min each, run in parallel)**
- ✅ Step 10: File browser UI component
- ⏳ Step 11: Upload UI component
- ✅ Step 12: Share creation UI
- ⏳ Step 13: Public share page

**Group 5: Quality Assurance (90 min total)**
- ⏳ Step 14: Integration tests (60 min)
- ⏳ Step 15: Security tests (60 min) - can run parallel with Step 14
- ⏳ Step 16: Documentation and deployment (30 min)

### Critical Path
**Step 01 → Step 02 → Step 06 → Step 10 → Step 14 → Step 16**

## Enhanced Security Architecture

### Comprehensive Permission System
The implementation includes a robust permission validation service with:

**Security Features:**
- **Rate limiting protection** against share token brute force attacks
- **Constant-time password comparison** to prevent timing attacks (bcrypt cost 12)
- **Comprehensive audit logging** for all security events
- **Input sanitization** preventing path traversal and injection attacks
- **Permission matrix enforcement** with granular access controls

**Permission Types:**
- `read`: View metadata and download files
- `read_upload`: View, download, and upload files
- `upload_only`: Upload files without viewing existing content

**Testing Requirements:**
- **100% test coverage** for security-critical permission code
- **OWASP Top 10 security tests** covering access control, injection, and authentication attacks
- **Timing attack validation** ensuring constant-time password comparisons
- **Edge case testing** for deleted users, expired tokens, circular references

### File Upload Security
- **MIME type validation** with extension whitelisting
- **File size limits** enforced per user and globally
- **Filename sanitization** preventing path traversal attacks
- **Virus scanning hooks** for production deployments
- **Quota enforcement** preventing storage abuse
- **Chunked upload support** for large files with integrity verification

## Testing Strategy

### Four Types of Testing

**1. Unit Tests (80%+ coverage required, 100% for security code)**
- Business logic validation in services layer
- Permission and authorization testing
- S3 client mocking for reliable tests
- Error handling edge cases

**2. Integration Tests**
- End-to-end API workflows
- Database operations with PocketBase
- S3 upload/download flows
- Authentication and authorization flows
- HTMX UI interactions

**3. Security Tests**
- Path traversal prevention
- Permission escalation attempts
- Share token brute force protection
- Input sanitization validation
- Timing attack resistance
- Rate limiting effectiveness

**4. Performance Tests**
- Large file upload benchmarks (1KB, 10MB, 100MB)
- Concurrent operation handling
- Database query optimization
- Memory usage profiling

### Test Organization
```
tests/
├── integration/          # End-to-end API tests
├── security/            # OWASP Top 10 security tests
├── unit/               # Business logic unit tests
└── ui/                 # Frontend component tests
```

### Test Execution Commands
```bash
# Run all tests with coverage
make test

# Run specific test types
make test-unit          # Unit tests only
make test-integration    # Integration tests only
make test-coverage       # Generate coverage report

# Security testing
go test ./tests/security/... -v

# Performance benchmarking
make benchmark
```

## Advanced Configuration System

### Configuration Sources (priority order)
1. Environment variables (highest priority)
2. `./config.yaml`
3. `./config.yml`
4. `/etc/filesonthego/config.yaml`
5. `/etc/filesonthego/config.yml` (lowest priority)

### Complete Configuration Schema
```yaml
# S3 Configuration
s3_endpoint: "https://s3.amazonaws.com"      # Required
s3_bucket: "filesonthego"                     # Required
s3_access_key: "..."                         # Required
s3_secret_key: "..."                         # Required
s3_region: "us-east-1"                       # Optional
s3_use_ssl: true                              # Optional

# Application Configuration
app_port: 8090                                # Optional (default: 8090)
app_environment: "production"                   # Optional (default: development)
app_url: "https://files.example.com"           # Required
jwt_secret: "your-secret-here"                # Required in production

# Database Configuration
db_path: "./pb_data"                           # Optional (default: ./pb_data)

# File Upload Limits
max_upload_size: 5368709120                   # Optional (default: 5GB)
default_user_quota: 107374182400                # Optional (default: 100GB)

# Feature Flags
public_registration: false                       # Optional (default: true)
email_verification: true                        # Optional (default: false)
share_password_required: false                   # Optional (default: false)

# Security Settings
rate_limit_requests: 100                        # Optional (default: 100)
rate_limit_window: 3600                         # Optional (default: 1 hour)
session_timeout: 86400                          # Optional (default: 24 hours)
```

## HTMX-Based Frontend Architecture

### Component System
The UI uses a modular component approach with server-rendered HTML fragments:

**Template Structure:**
```
templates/
├── layouts/
│   ├── base.html           # Base layout with HTMX setup
│   ├── auth.html          # Authentication pages layout
│   └── app.html           # Main application layout
├── components/            # Reusable UI components
│   ├── file-list.html     # File/folder listing
│   ├── file-item.html     # Individual file display
│   ├── file-actions.html  # Upload, sort, search toolbar
│   ├── context-menu.html  # Right-click action menu
│   ├── file-details-modal.html  # File properties dialog
│   ├── share-button.html  # Share creation button
│   ├── share-modal.html   # Share creation dialog
│   ├── share-link-display.html  # Share link with copy/QR
│   ├── breadcrumb.html    # Navigation breadcrumbs
│   ├── toast.html        # Notification system
│   └── loading.html      # Loading indicators
└── pages/
    ├── files.html         # Main file browser
    ├── shares.html        # Share management
    ├── login.html         # Login page
    └── public-share.html # Public share access
```

### HTMX Interaction Patterns

**Dynamic Content Loading:**
```html
<!-- Directory navigation -->
<a hx-get="/api/directories/{id}"
   hx-target="#file-list"
   hx-push-url="true"
   hx-indicator="#loading">
  Folder Name
</a>

<!-- File download with progress -->
<button hx-get="/api/files/{id}/download"
        hx-indicator="#loading"
        class="download-btn">
  Download
</button>

<!-- Inline operations -->
<form hx-patch="/api/files/{id}"
      hx-target="closest .file-item"
      hx-swap="outerHTML">
  <input name="name" value="current name">
</form>
```

**Smart Response Handling:**
- **HTMX requests** → Return HTML fragments
- **Direct navigation** → Return full pages
- **JavaScript clients** → Return JSON responses
- **Error conditions** → Appropriate HTTP status codes with structured error messages

### JavaScript Minimalism
Only essential JavaScript for UX enhancements:
- **Context menu positioning** and event handling
- **Keyboard shortcuts** (Delete, Ctrl+A, Escape, arrow keys)
- **File drag-and-drop** with progress indicators
- **Clipboard operations** for copying share links
- **Toast notifications** system
- **Modal management** (open/close, focus management)
- **File size formatting** and date localization

### Accessibility Features
- **ARIA labels** on all interactive elements
- **Keyboard navigation** support throughout interface
- **Screen reader compatibility** with semantic HTML
- **High contrast mode** support
- **Focus management** in modals and dynamic content
- **Error announcement** for screen readers

## GitHub Hooks Integration

### Development Workflow with GitHub Hooks

#### Pre-commit Hooks
```bash
#!/bin/sh
# .git/hooks/pre-commit

echo "Running pre-commit checks..."

# 1. Code formatting
echo "Checking Go code formatting..."
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    echo "Files need formatting:"
    echo "$unformatted"
    exit 1
fi

# 2. Go vet analysis
echo "Running go vet..."
go vet ./...
if [ $? -ne 0 ]; then
    echo "go vet failed"
    exit 1
fi

# 3. Security linting
echo "Running gosec security scan..."
gosec ./...
if [ $? -ne 0 ]; then
    echo "Security issues found"
    exit 1
fi

# 4. Run unit tests
echo "Running unit tests..."
go test ./... -short
if [ $? -ne 0 ]; then
    echo "Unit tests failed"
    exit 1
fi

# 5. Check test coverage
echo "Checking test coverage..."
coverage=$(go test -cover ./... | grep -o '[0-9.]*%' | sed 's/%//')
if (( $(echo "$coverage < 80" | bc -l) )); then
    echo "Test coverage below 80%: $coverage%"
    exit 1
fi

echo "Pre-commit checks passed!"
```

#### Pre-push Hooks
```bash
#!/bin/sh
# .git/hooks/pre-push

echo "Running pre-push validation..."

# 1. Full test suite
echo "Running comprehensive test suite..."
make test
if [ $? -ne 0 ]; then
    echo "Test suite failed"
    exit 1
fi

# 2. Security tests
echo "Running security tests..."
go test ./tests/security/... -v
if [ $? -ne 0 ]; then
    echo "Security tests failed"
    exit 1
fi

# 3. Integration tests
echo "Running integration tests..."
go test ./tests/integration/... -v
if [ $? -ne 0 ]; then
    echo "Integration tests failed"
    exit 1
fi

# 4. Build verification
echo "Verifying build..."
go build -o /tmp/filesonthego main.go
if [ $? -ne 0 ]; then
    echo "Build failed"
    exit 1
fi

# 5. Performance benchmarks
echo "Running performance benchmarks..."
go test -bench=. ./...
if [ $? -ne 0 ]; then
    echo "Benchmarks failed"
    exit 1
fi

rm -f /tmp/filesonthego
echo "Pre-push validation passed!"
```

#### Commit Message Validation
```bash
#!/bin/sh
# .git/hooks/commit-msg

commit_regex='^(feat|fix|test|refactor|docs|perf|security|chore)(\(.+\))?: .{1,50}'

if ! grep -qE "$commit_regex" "$1"; then
    echo "Invalid commit message format!"
    echo "Format: type(scope): subject (max 50 chars)"
    echo "Types: feat, fix, test, refactor, docs, perf, security, chore"
    echo "Example: feat(upload): add chunked upload support"
    exit 1
fi

# Check for issue reference
if grep -qE "(fixes|closes|resolves) #[0-9]+" "$1"; then
    echo "Good: Issue reference found"
else
    echo "Consider adding issue reference: (fixes #123)"
fi
```

### Continuous Integration GitHub Actions

#### Main CI Workflow (`.github/workflows/ci.yml`)
```yaml
name: Continuous Integration

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

env:
  GO_VERSION: '1.21'

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.20', '1.21', '1.22']

    services:
      minio:
        image: minio/minio:latest
        ports:
          - 9000:9000
        env:
          MINIO_ROOT_USER: minioadmin
          MINIO_ROOT_PASSWORD: minioadmin
        options: >-
          --health-cmd "curl -f http://localhost:9000/minio/health/live"
          --health-interval 30s
          --health-timeout 10s
          --health-retries 3

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: |
          ~/.cache/go-build
          ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}

    - name: Install dependencies
      run: go mod download

    - name: Run go vet
      run: go vet ./...

    - name: Run gosec security scan
      uses: securecodewarrior/github-action-gosec@master
      with:
        args: '-no-fail -fmt sarif -out results.sarif ./...'

    - name: Upload gosec results
      uses: github/codeql-action/upload-sarif@v2
      if: always()
      with:
        sarif_file: results.sarif

    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

    - name: Run integration tests
      run: go test -v -tags=integration ./tests/integration/...
      env:
        S3_ENDPOINT: http://localhost:9000
        S3_ACCESS_KEY: minioadmin
        S3_SECRET_KEY: minioadmin
        S3_BUCKET: test-bucket

    - name: Run security tests
      run: go test -v ./tests/security/...

    - name: Build application
      run: |
        go build -v -o filesonthego main.go
        ./filesonthego version

    - name: Performance benchmarks
      run: go test -bench=. -benchmem ./...

  security-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'

    - name: Upload Trivy scan results
      uses: github/codeql-action/upload-sarif@v2
      if: always()
      with:
        sarif_file: 'trivy-results.sarif'

  docker-build:
    runs-on: ubuntu-latest
    needs: test
    steps:
    - uses: actions/checkout@v4

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Build Docker image
      uses: docker/build-push-action@v5
      with:
        context: .
        push: false
        tags: filesonthego:test
        cache-from: type=gha
        cache-to: type=gha,mode=max
```

#### Release Workflow (`.github/workflows/release.yml`)
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  create-release:
    runs-on: ubuntu-latest
    outputs:
      upload_url: ${{ steps.create_release.outputs.upload_url }}
    steps:
    - uses: actions/checkout@v4

    - name: Generate changelog
      run: |
        # Generate changelog from git commits since last tag
        git fetch --prune --unshallow
        PREV_TAG=$(git describe --tags --abbrev=0 HEAD^)
        echo "## Changes since $PREV_TAG" > CHANGELOG.md
        git log $PREV_TAG..HEAD --pretty=format:"- %s" >> CHANGELOG.md

    - name: Create Release
      id: create_release
      uses: actions/create-release@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      with:
        tag_name: ${{ github.ref }}
        release_name: Release ${{ github.ref }}
        body_path: CHANGELOG.md
        draft: false
        prerelease: false

  build-and-upload:
    needs: create-release
    strategy:
      matrix:
        goos: [linux, windows, darwin]
        goarch: [amd64, arm64]
        exclude:
          - goos: windows
            goarch: arm64

    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Build binary
      run: |
        GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
        go build -v -o filesonthego-${{ matrix.goos }}-${{ matrix.goarch }} main.go

    - name: Package binary
      run: |
        tar czf filesonthego-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz \
          filesonthego-${{ matrix.goos }}-${{ matrix.goarch }}

    - name: Upload Release Asset
      uses: actions/upload-release-asset@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      with:
        upload_url: ${{ needs.create-release.outputs.upload_url }}
        asset_path: filesonthego-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz
        asset_name: filesonthego-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz
        asset_content_type: application/gzip

  docker-release:
    needs: create-release
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Login to Docker Hub
      uses: docker/login-action@v3
      with:
        username: ${{ secrets.DOCKER_USERNAME }}
        password: ${{ secrets.DOCKER_PASSWORD }}

    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v5
      with:
        images: jd-boyd/filesonthego
        tags: |
          type=ref,event=tag
          type=semver,pattern={{version}}
          type=semver,pattern={{major}}.{{minor}}

    - name: Build and push Docker image
      uses: docker/build-push-action@v5
      with:
        context: .
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        cache-from: type=gha
        cache-to: type=gha,mode=max
```

### Pull Request Template (`.github/pull_request_template.md`)
```markdown
## Description
Brief description of changes and their purpose.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Security fix
- [ ] Refactoring (non-functional improvement)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass: `go test ./...`
- [ ] Integration tests pass: `go test ./tests/integration/...`
- [ ] Security tests pass: `go test ./tests/security/...`
- [ ] Test coverage meets requirements: `go test -cover ./...`
- [ ] Manual testing completed

## Security Considerations
- [ ] Input validation implemented
- [ ] Permission checks added
- [ ] Potential attack vectors considered
- [ ] Sensitive data properly handled

## Performance Impact
- [ ] No performance impact expected
- [ ] Performance improvements included
- [ ] Performance testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review of the code completed
- [ ] Documentation updated if necessary
- [ ] CHANGELOG.md updated with changes
- [ ] Ready for production deployment
```

## Legacy Roadmap

**Phase 1 (MVP):**
- [x] Gin + GORM project setup
- [x] S3 client integration
- [x] Data models (GORM)
- [ ] File upload/download
- [ ] Directory navigation
- [ ] Basic JWT auth
- [ ] HTMX file browser
- [ ] Read-only sharing
- [ ] Password-protected shares

**Phase 2 (Sharing):**
- [x] All permission types
- [x] Share expiration
- [x] Share management UI
- [x] Access logging

**Phase 3 (UX):**
- [x] Drag-and-drop upload
- [x] Chunked uploads for large files
- [ ] ZIP downloads for folders
- [x] Mobile-responsive design

**Phase 4 (Advanced):**
- [ ] Search
- [ ] User quotas
- [ ] Admin panel
- [ ] Webhooks

**Maybe later:**
- Team/org support
- LDAP/SSO
- File versioning
- Malware scanning
