# FilesOnTheGo - Design Document

## Project Overview

**FilesOnTheGo** is a self-hosted file storage and sharing service that provides Nextcloud/Google Drive-like functionality with S3-compatible storage backends. Built on PocketBase for rapid development and ease of deployment, it focuses on flexible sharing permissions and user-friendly file management.

## Core Requirements

### Functional Requirements

1. **User Management**
   - User registration and authentication
   - User profiles and settings
   - Multi-user support with isolated storage

2. **File Storage**
   - Upload files to hierarchical directory structures
   - Store actual file data in S3-compatible storage (MinIO, AWS S3, Backblaze B2, etc.)
   - Store metadata in PocketBase (file names, paths, sizes, MIME types, ownership)
   - Support for large files with chunked uploads
   - Version history (optional for v1)

3. **Directory Management**
   - Create, rename, move, and delete directories
   - Hierarchical folder structures
   - Path-based navigation
   - Breadcrumb navigation support

4. **Sharing & Permissions**
   - Per-file and per-directory sharing links
   - Four permission levels:
     - **Private**: No external access
     - **Read-only**: View and download files
     - **Read/Upload**: View, download, and upload files
     - **Upload-only**: Upload files and see names, but no download access
   - Share links with optional expiration dates
   - Password-protected shares (optional for v1)

5. **File Operations**
   - Upload single or multiple files
   - Download individual files or folders (as ZIP)
   - Delete files and directories
   - Move/copy files between directories
   - Search functionality

### Non-Functional Requirements

1. **Performance**
   - Efficient handling of large files (streaming, chunked uploads)
   - Fast metadata queries through PocketBase
   - CDN-friendly architecture

2. **Security**
   - Secure authentication (JWT tokens via PocketBase)
   - Encrypted connections (HTTPS)
   - Signed S3 URLs for direct file access
   - Permission validation on all operations
   - Input sanitization and validation

3. **Scalability**
   - Horizontal scaling capability through S3 storage
   - Database-backed metadata for quick access

4. **Maintainability**
   - Clear separation of concerns
   - Well-documented API
   - Easy deployment process

## System Architecture

### High-Level Architecture

```
┌─────────────┐
│   Client    │ (Web Browser / Mobile App)
│   (React)   │
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
    │  S3-Compatible │
    │     Storage    │
    │  (MinIO/AWS)   │
    └────────────────┘
```

### Technology Stack

**Backend:**
- **PocketBase** (Go-based backend)
  - Built-in authentication
  - RESTful API
  - Real-time subscriptions
  - SQLite database
  - Easy extensibility with Go

**Storage:**
- **S3-Compatible Storage** (MinIO, AWS S3, Backblaze B2, Wasabi, etc.)
  - Object storage for file data
  - Scalable and cost-effective
  - Industry-standard API

**Frontend:**
- **HTMX**
  - Hypermedia-driven interactions
  - Server-side rendering with dynamic updates
  - File upload with progress indicators
  - Minimal JavaScript footprint
- **Tailwind CSS**
  - Utility-first styling
  - Responsive design
  - Consistent UI components

**Additional Libraries:**
- **AWS SDK for Go** (S3 operations)
- **go-uuid** (generating share tokens)
- PocketBase built-in routing or Go's standard library router

## Data Models

### Collections in PocketBase

#### 1. Users Collection (Built-in)
- id (auto-generated)
- email
- username
- password (hashed)
- avatar (optional)
- created
- updated

#### 2. Files Collection
```javascript
{
  id: "string (auto)",
  user: "relation(users, required)",
  name: "string (required)",
  path: "string (required)", // Full path from root, e.g., "/documents/work/report.pdf"
  parent_directory: "relation(directories, optional)",
  size: "number (bytes)",
  mime_type: "string",
  s3_key: "string (required)", // Unique key in S3 bucket
  s3_bucket: "string",
  checksum: "string (md5/sha256)",
  is_directory: "bool (false)",
  created: "datetime",
  updated: "datetime"
}
```

#### 3. Directories Collection
```javascript
{
  id: "string (auto)",
  user: "relation(users, required)",
  name: "string (required)",
  path: "string (required)", // Full path, e.g., "/documents/work"
  parent_directory: "relation(directories, optional)", // null for root
  created: "datetime",
  updated: "datetime"
}
```

#### 4. Shares Collection
```javascript
{
  id: "string (auto)",
  user: "relation(users, required)", // Owner
  resource_type: "string (enum: file, directory)",
  file: "relation(files, optional)",
  directory: "relation(directories, optional)",
  share_token: "string (unique, indexed)", // UUID for share link
  permission_type: "string (enum: read, read_upload, upload_only)",
  password_hash: "string (optional)", // For password-protected shares
  expires_at: "datetime (optional)",
  access_count: "number (default: 0)",
  created: "datetime",
  updated: "datetime"
}
```

#### 5. Share Access Logs Collection (Optional)
```javascript
{
  id: "string (auto)",
  share: "relation(shares, required)",
  ip_address: "string",
  user_agent: "string",
  action: "string (enum: view, download, upload)",
  accessed_at: "datetime"
}
```

### S3 Storage Structure

Files stored in S3 will use a structured key format:

```
users/{user_id}/{file_id}/{filename}
```

Example:
```
users/abc123/file_xyz789/document.pdf
```

This structure provides:
- User isolation
- Unique file identification
- Original filename preservation
- Easy cleanup when users are deleted

## API Design

### Authentication Endpoints (Built-in PocketBase)

- `POST /api/collections/users/auth-with-password` - Login
- `POST /api/collections/users/auth-refresh` - Refresh token
- `POST /api/collections/users/records` - Register

### File Management Endpoints

#### Upload File
```
POST /api/files/upload
Headers: Authorization: Bearer {token}
Body: multipart/form-data
  - file: File
  - directory_id: string (optional, root if not provided)
  - path: string (optional alternative to directory_id)

Response: {
  file: {
    id, name, path, size, mime_type, created
  }
}
```

#### List Files/Directories
```
GET /api/files?directory_id={id}&path={path}
Headers: Authorization: Bearer {token}

Response: {
  items: [
    { id, name, path, is_directory, size, mime_type, created, updated }
  ],
  current_path: string,
  parent_directory: string
}
```

#### Download File
```
GET /api/files/{file_id}/download
Headers: Authorization: Bearer {token}

Response: Redirect to signed S3 URL or stream file
```

#### Delete File
```
DELETE /api/files/{file_id}
Headers: Authorization: Bearer {token}
```

#### Move/Rename File
```
PATCH /api/files/{file_id}
Headers: Authorization: Bearer {token}
Body: {
  name: string (optional),
  directory_id: string (optional),
  path: string (optional)
}
```

### Directory Management Endpoints

#### Create Directory
```
POST /api/directories
Headers: Authorization: Bearer {token}
Body: {
  name: string,
  parent_directory_id: string (optional),
  path: string (optional)
}
```

#### Delete Directory
```
DELETE /api/directories/{directory_id}
Headers: Authorization: Bearer {token}
Query: ?recursive=true (optional)
```

### Sharing Endpoints

#### Create Share Link
```
POST /api/shares
Headers: Authorization: Bearer {token}
Body: {
  resource_type: "file" | "directory",
  resource_id: string,
  permission_type: "read" | "read_upload" | "upload_only",
  password: string (optional),
  expires_at: datetime (optional)
}

Response: {
  share: {
    id, share_token, permission_type, expires_at, share_url
  }
}
```

#### Access Shared Resource
```
GET /api/public/share/{share_token}
Query: ?password={password} (if protected)

Response: {
  resource_type, name, permission_type, items (if directory)
}
```

#### Upload to Shared Directory
```
POST /api/public/share/{share_token}/upload
Body: multipart/form-data
  - file: File
  - password: string (if protected)
```

#### Download from Share
```
GET /api/public/share/{share_token}/download/{file_id}
Query: ?password={password} (if protected)

Response: Redirect to signed S3 URL or stream file
```

#### List Share Links
```
GET /api/shares?resource_type={type}&resource_id={id}
Headers: Authorization: Bearer {token}

Response: {
  shares: [
    { id, share_token, permission_type, expires_at, access_count, created }
  ]
}
```

#### Revoke Share
```
DELETE /api/shares/{share_id}
Headers: Authorization: Bearer {token}
```

## Permission System

### Permission Matrix

| Action | Private | Read | Read/Upload | Upload-Only | Owner |
|--------|---------|------|-------------|-------------|-------|
| View metadata | ❌ | ✅ | ✅ | ✅ (names only) | ✅ |
| Download file | ❌ | ✅ | ✅ | ❌ | ✅ |
| Upload file | ❌ | ❌ | ✅ | ✅ | ✅ |
| Delete file | ❌ | ❌ | ❌ | ❌ | ✅ |
| Create directory | ❌ | ❌ | ✅ | ✅ | ✅ |
| Share resource | ❌ | ❌ | ❌ | ❌ | ✅ |

### Permission Validation Flow

1. **Check if user is authenticated** (for non-public endpoints)
2. **Check if resource is shared** (for public endpoints)
   - Validate share token
   - Check expiration date
   - Verify password if required
3. **Check if user is owner** (for authenticated endpoints)
4. **Check share permission level** matches requested action
5. **Allow or deny** access

### Implementation Strategy

Create a middleware/hook in PocketBase:

```go
func ValidateFileAccess(fileId, userId, shareToken string, action string) bool {
    // 1. Check if user owns the file
    // 2. If not owner, check for valid share with appropriate permissions
    // 3. Return true/false
}
```

## S3 Integration

### Configuration

Store S3 credentials securely in environment variables or PocketBase settings:

```env
S3_ENDPOINT=https://s3.amazonaws.com
S3_REGION=us-east-1
S3_BUCKET=filesonthego
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
S3_USE_SSL=true
```

### Upload Process

1. Client initiates upload via API
2. Backend validates request and user permissions
3. Generate unique S3 key: `users/{user_id}/{file_id}/{filename}`
4. Upload file to S3 using AWS SDK
5. Store metadata in PocketBase database
6. Return success response to client

### Download Process

1. Client requests file download
2. Backend validates permissions
3. Generate pre-signed S3 URL (valid for 5-15 minutes)
4. Return signed URL or redirect client
5. Client downloads directly from S3

### Benefits of Pre-signed URLs

- Direct downloads from S3 (no proxying through backend)
- Reduced server bandwidth
- Better performance
- Automatic expiration for security

## Security Considerations

### Authentication & Authorization

- Use PocketBase's built-in JWT authentication
- Validate all requests with middleware
- Implement rate limiting on upload/download endpoints
- Use HTTPS only in production

### File Upload Security

- Validate file types and sizes
- Sanitize filenames (remove special characters, path traversal attempts)
- Limit upload size per user/per file
- Implement quotas per user

### Share Link Security

- Generate cryptographically secure random tokens (UUID v4)
- Use bcrypt for password-protected shares
- Implement access logging for audit trails
- Allow share revocation
- Support expiration dates

### S3 Security

- Use IAM roles with minimal permissions
- Enable S3 bucket versioning
- Configure bucket policies to prevent public access
- Use server-side encryption
- Regularly rotate access keys

## User Interface Design

### Key Pages/Views

1. **Dashboard/File Browser**
   - File/folder list with icons
   - Breadcrumb navigation
   - Upload button (drag-and-drop zone)
   - New folder button
   - Sort options (name, date, size)

2. **File/Folder Context Menu**
   - Download
   - Rename
   - Move
   - Delete
   - Share (opens share dialog)
   - View details

3. **Share Dialog**
   - Permission level selector (radio buttons)
   - Password protection toggle with input field
   - Expiration date picker
   - Generate link button
   - Copy link button
   - List of existing shares with revoke option

4. **Public Share Page**
   - Display shared resource name
   - File list (if directory)
   - Download buttons (if read permission)
   - Upload zone (if upload permission)
   - Password prompt (if protected)

5. **Settings Page**
   - Account details
   - Storage usage
   - S3 configuration (admin only)
   - Default share settings

### HTMX Implementation Notes

- Use `hx-get`, `hx-post`, `hx-delete` for AJAX-style interactions
- Server responds with HTML fragments
- `hx-target` and `hx-swap` for dynamic content updates
- `hx-indicator` for loading states during uploads
- Minimal JavaScript for drag-and-drop and copy-to-clipboard
- Server-side rendering for initial page load and SEO

### Mobile Responsive Design

- Stack layout for mobile
- Touch-friendly buttons and gestures
- Simplified context menus (bottom sheets)
- Progressive Web App (PWA) support

## Deployment Architecture

### Single Server Deployment

```
┌────────────────────────────────────┐
│          Server (VPS/Bare Metal)    │
│                                    │
│  ┌─────────────────────────────┐  │
│  │  Nginx (Reverse Proxy)      │  │
│  │  - HTTPS termination        │  │
│  │  - Static file serving      │  │
│  └─────────────────────────────┘  │
│              │                     │
│              ▼                     │
│  ┌─────────────────────────────┐  │
│  │  FilesOnTheGo (PocketBase)  │  │
│  │  - Port 8090                │  │
│  └─────────────────────────────┘  │
│              │                     │
│              ▼                     │
│  ┌─────────────────────────────┐  │
│  │  SQLite Database            │  │
│  │  (embedded in PocketBase)   │  │
│  └─────────────────────────────┘  │
└────────────────────────────────────┘
              │
              ▼ (over internet)
┌────────────────────────────────────┐
│   S3-Compatible Storage            │
│   (MinIO/AWS/Backblaze)            │
└────────────────────────────────────┘
```

### Container Deployment

Container and systemd deployment examples will be provided in separate deployment documentation files for flexible installation options.

## Development Roadmap

### Phase 1: MVP (Core Functionality)
- [ ] Set up PocketBase project structure
- [ ] Implement S3 client integration
- [ ] Create data models (collections)
- [ ] Build file upload API endpoint
- [ ] Build file download API endpoint
- [ ] Implement directory creation and navigation
- [ ] Basic authentication (using PocketBase built-in)
- [ ] Simple web UI for file browsing with HTMX and Tailwind
- [ ] Basic read-only sharing
- [ ] Password protection for shares

### Phase 2: Enhanced Sharing
- [ ] Implement all permission types (read, read/upload, upload-only)
- [ ] Implement share expiration
- [ ] Create share management UI
- [ ] Add access logging

### Phase 3: User Experience
- [ ] Drag-and-drop upload
- [ ] Chunked upload for large files
- [ ] Download folders as ZIP
- [ ] Mobile-responsive design
- [ ] Progressive Web App features

### Phase 4: Advanced Features
- [ ] File search functionality
- [ ] File versioning
- [ ] Storage quotas per user
- [ ] Admin panel for user management
- [ ] Activity logs and analytics
- [ ] Integration with external storage providers
- [ ] Public user registration vs invite-only mode
- [ ] Webhook system for extensibility (file upload/download/delete events)
- [ ] Optional malware scanning integration (ClamAV or similar)

### Phase 5: Enterprise Features (Optional)
- [ ] Team/organization support
- [ ] LDAP/SSO integration
- [ ] Advanced access controls
- [ ] Audit logs
- [ ] Data retention policies
- [ ] Multi-region S3 support

## Testing Strategy

### Unit Tests
- File upload/download logic
- Permission validation
- S3 client operations
- Share token generation

### Integration Tests
- End-to-end file upload to S3
- Authentication flow
- Share link access with various permissions
- Directory operations

### Security Tests
- Authentication bypass attempts
- Path traversal attempts
- Permission escalation attempts
- Share link brute force protection

### Performance Tests
- Large file uploads (1GB+)
- Concurrent user uploads
- Directory listing with many files
- Share link access under load

## Monitoring & Logging

### Metrics to Track
- Upload/download success rate
- Average upload/download speed
- S3 storage usage per user
- API response times
- Error rates
- Active shares

### Logging
- User authentication events
- File operations (upload, download, delete)
- Share link creation and access
- Failed permission checks
- S3 operation errors

## Backup & Recovery

### Database Backups
- Daily automated SQLite database backups
- Store backups in S3 or separate storage
- Retention policy (30 days)

### S3 Data Protection
- Enable S3 versioning
- Configure lifecycle policies
- Consider cross-region replication for critical data

### Disaster Recovery Plan
1. Restore database from latest backup
2. Verify S3 bucket integrity
3. Restart application services
4. Validate user access and functionality

## Configuration Management

### Environment Variables

```env
# Application
APP_ENV=production
APP_URL=https://filesonthego.example.com
PORT=8090

# Database
DB_PATH=/app/pb_data/data.db

# S3 Configuration
S3_ENDPOINT=https://s3.amazonaws.com
S3_REGION=us-east-1
S3_BUCKET=filesonthego
S3_ACCESS_KEY=your_access_key
S3_SECRET_KEY=your_secret_key
S3_USE_SSL=true

# Security
JWT_SECRET=your_jwt_secret
MAX_UPLOAD_SIZE=5368709120  # 5GB in bytes
ALLOWED_ORIGINS=https://filesonthego.example.com

# Features
ENABLE_PUBLIC_REGISTRATION=false
ENABLE_EMAIL_VERIFICATION=true
DEFAULT_USER_QUOTA=107374182400  # 100GB in bytes
```

## Future Considerations

### Potential Enhancements
- Webhook system for event notifications (upload, download, share created, etc.)
- Malware scanning integration (ClamAV or third-party API services)
- File search across names and metadata (full-text search as advanced feature)
- Real-time collaboration (shared editing)
- File preview generation (thumbnails, document previews)
- Integration with external services (Dropbox, Google Drive sync)
- Mobile apps (native iOS/Android)
- WebDAV support
- Encrypted storage option
- Two-factor authentication

### Webhook System Design (Future)

A webhook system would enable extensibility and integration with external services:

**Webhook Events:**
- `file.uploaded` - New file uploaded
- `file.downloaded` - File downloaded
- `file.deleted` - File deleted
- `share.created` - Share link created
- `share.accessed` - Share link accessed
- `directory.created` - New directory created
- `user.registered` - New user registration

**Implementation Approach:**
- Webhooks collection in PocketBase with URL, events, secret, and status
- Queue system for reliable delivery (using Go channels or external queue)
- Retry logic with exponential backoff
- HMAC signatures for webhook verification
- Webhook logs for debugging

**Use Cases:**
- Trigger malware scanning on file upload
- Send notifications to external services (Slack, Discord, email)
- Sync metadata to external systems
- Custom analytics and monitoring
- Automated backup triggers

### Scalability Options
- Move from SQLite to PostgreSQL for larger deployments
- Implement Redis for session management and caching
- Add CDN for frequently accessed files
- Horizontal scaling with load balancer
- Separate service for file processing (thumbnails, virus scanning)

## Conclusion

FilesOnTheGo provides a self-hosted alternative to commercial cloud storage solutions with fine-grained sharing controls. By leveraging PocketBase for rapid development and S3-compatible storage for scalability, the system can be deployed easily while maintaining professional-grade features.

The modular architecture allows for incremental development, starting with core file management and progressively adding advanced features based on user needs.

---

**Document Version:** 1.0  
**Last Updated:** November 21, 2025  
**Author:** Joshua D. Boyd
