# FilesOnTheGo Design

This doc covers architecture, data models, and API design for FilesOnTheGo.

## Overview

FilesOnTheGo is a self-hosted file storage service. Files go to S3-compatible storage, metadata lives in PocketBase's SQLite database. The goal is Nextcloud-like functionality with less complexity.

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
│  │       PocketBase Core            │  │
│  │  - Authentication                │  │
│  │  - API Routes                    │  │
│  │  - Database (SQLite)            │  │
│  └──────────────────────────────────┘  │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │     Custom Extensions            │  │
│  │  - File Upload Handler           │  │
│  │  - Share Link Generator          │  │
│  │  - Permission Validator          │  │
│  │  - S3 Client Integration         │  │
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

**Backend:** PocketBase (Go). Gives us auth, REST API, realtime subscriptions, and SQLite out of the box.

**Storage:** S3-compatible (MinIO for dev, AWS/Backblaze/Wasabi for prod).

**Frontend:** HTMX + Tailwind. Server-rendered pages with dynamic updates. Minimal JS—just drag-and-drop and clipboard stuff.

**Libraries:** AWS SDK for Go (S3 ops), go-uuid (share tokens).

## Data Models

### Users (PocketBase built-in)
Standard PocketBase user collection with email, username, password.

### Files
```javascript
{
  id: "string (auto)",
  user: "relation(users)",
  name: "string",
  path: "string",                    // e.g., "/documents/work/report.pdf"
  parent_directory: "relation(directories)",
  size: "number",
  mime_type: "string",
  s3_key: "string",                  // unique key in S3
  s3_bucket: "string",
  checksum: "string",
  created: "datetime",
  updated: "datetime"
}
```

### Directories
```javascript
{
  id: "string (auto)",
  user: "relation(users)",
  name: "string",
  path: "string",                    // e.g., "/documents/work"
  parent_directory: "relation(directories)",  // null for root
  created: "datetime",
  updated: "datetime"
}
```

### Shares
```javascript
{
  id: "string (auto)",
  user: "relation(users)",           // owner
  resource_type: "file | directory",
  file: "relation(files)",
  directory: "relation(directories)",
  share_token: "string (unique)",    // UUID for the link
  permission_type: "read | read_upload | upload_only",
  password_hash: "string",           // optional, bcrypt
  expires_at: "datetime",            // optional
  access_count: "number",
  created: "datetime",
  updated: "datetime"
}
```

### S3 Key Structure

Files are stored as: `users/{user_id}/{file_id}/{filename}`

This keeps users isolated, makes files uniquely identifiable, and preserves original filenames.

## API

### Auth (PocketBase built-in)
- `POST /api/collections/users/auth-with-password` - Login
- `POST /api/collections/users/auth-refresh` - Refresh token
- `POST /api/collections/users/records` - Register

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
5. Save metadata to PocketBase
6. Return success

**Download flow:**
1. Client requests download
2. Validate permissions
3. Generate pre-signed URL (5-15 min expiry)
4. Redirect client to S3

Pre-signed URLs mean downloads go directly from S3 to the client—no proxying through the backend.

## Security Notes

**Auth:** Use PocketBase's JWT. Validate on every request. Rate limit sensitive endpoints.

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
FilesOnTheGo (port 8090)
    ↓
SQLite (embedded)
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

db_path: ./pb_data
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

## Roadmap

**Phase 1 (MVP):**
- [x] PocketBase project setup
- [x] S3 client integration
- [x] Data models
- [ ] File upload/download
- [ ] Directory navigation
- [ ] Basic auth
- [ ] HTMX file browser
- [ ] Read-only sharing
- [ ] Password-protected shares

**Phase 2 (Sharing):**
- [ ] All permission types
- [ ] Share expiration
- [ ] Share management UI
- [ ] Access logging

**Phase 3 (UX):**
- [ ] Drag-and-drop upload
- [ ] Chunked uploads for large files
- [ ] ZIP downloads for folders
- [ ] Mobile-responsive design

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
