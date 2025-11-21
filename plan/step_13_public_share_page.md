# Step 13: Public Share Page

## Overview
Create the public-facing page for accessing shared files and directories, with password protection, permission-based UI, and file operations.

## Dependencies
- Step 05: Basic UI layout (requires templates structure)
- Step 07: File download handler (requires download API)
- Step 09: Share service (requires share validation)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 13 of the FilesOnTheGo project. Your task is to create a public share access page that respects permissions and provides appropriate UI based on share type.

### Tasks

1. **Create templates/pages/public-share.html**

   Public share page layout:
   ```html
   - Minimal header (no authentication required)
   - Share information display
   - Password prompt (if protected)
   - File/directory content based on share type
   - Appropriate actions based on permission level
   - Footer with branding
   ```

2. **Create templates/components/password-prompt.html**

   Password entry form:
   ```html
   - Password input field
   - Submit button
   - Error message display
   - Remember password option (session)
   - Clean, centered design
   ```

3. **Create templates/components/shared-file-view.html**

   Single file share view:
   ```html
   - File icon (based on type)
   - File name
   - File size
   - File type
   - Download button (if read/read_upload)
   - Preview (for images/PDFs, optional)
   - Share expiration notice
   ```

4. **Create templates/components/shared-directory-view.html**

   Directory share view:
   ```html
   - Directory name
   - File list (read-only for upload-only)
   - Upload zone (if read_upload or upload_only)
   - Download button (if read/read_upload)
   - Sort options
   - Breadcrumb (for nested navigation)
   ```

5. **Create templates/components/upload-only-view.html**

   Upload-only share view:
   ```html
   - Upload instructions
   - Drag-and-drop zone
   - File list (names only, no download links)
   - Upload progress
   - Success messages
   - No file previews or download buttons
   ```

6. **Implement Share Validation Handler (handlers/public_share_handler.go)**

   `GET /share/{share_token}`

   **Process:**
   1. Validate share token
   2. Check expiration
   3. If password-protected and not validated:
      - Show password prompt
   4. If valid:
      - Log access
      - Show appropriate view based on resource type and permission
   5. Handle errors gracefully

7. **Implement Password Validation Endpoint**

   `POST /share/{share_token}/validate`

   **Process:**
   1. Get share by token
   2. Verify password
   3. Set session/cookie for validated access
   4. Return success or error
   5. Rate limit to prevent brute force

8. **Implement Permission-Based UI**

   **Read-only:**
   - Show download buttons
   - Show file previews
   - No upload functionality
   - View-only directory listing

   **Read & Upload:**
   - Show download buttons
   - Show file previews
   - Show upload zone
   - Full directory interaction

   **Upload-only:**
   - NO download buttons
   - NO file previews
   - Show upload zone
   - Show file names only (no sizes or metadata)
   - Confirm upload success

9. **Implement File Download from Share**

   `GET /share/{share_token}/download/{file_id}`

   **Process:**
   1. Validate share token
   2. Check password if required
   3. Verify permission allows download
   4. Log download access
   5. Generate pre-signed URL or stream file

10. **Implement File Upload to Share**

    `POST /share/{share_token}/upload`

    **Process:**
    1. Validate share token
    2. Check password if required
    3. Verify permission allows upload
    4. Process file upload
    5. Store in shared directory
    6. Log upload access
    7. Return success

11. **Implement Access Logging**

    Log all share page access:
    - View
    - Password attempt (success/failure)
    - Download
    - Upload
    - IP address
    - User agent
    - Timestamp

12. **Add Share Expiration Notice**

    ```html
    <div class="bg-yellow-50 border border-yellow-200 rounded p-3 mb-4">
      <svg class="inline w-5 h-5 text-yellow-600"><!-- clock icon --></svg>
      <span class="text-yellow-800">
        This share expires on {expiration_date}
      </span>
    </div>
    ```

13. **Add Share Type Indicator**

    ```html
    <div class="bg-blue-50 border border-blue-200 rounded p-3 mb-4">
      <div class="font-medium text-blue-900">Permission: {permission_type}</div>
      <div class="text-sm text-blue-700">
        {permission_description}
      </div>
    </div>
    ```

14. **Implement Error States**

    **Share not found:**
    ```html
    <div class="text-center py-12">
      <svg class="w-24 h-24 mx-auto text-gray-300 mb-4"><!-- error icon --></svg>
      <h2 class="text-2xl font-bold text-gray-700 mb-2">Share Not Found</h2>
      <p class="text-gray-500">This share link is invalid or has been revoked.</p>
    </div>
    ```

    **Share expired:**
    ```html
    <div class="text-center py-12">
      <svg class="w-24 h-24 mx-auto text-gray-300 mb-4"><!-- clock icon --></svg>
      <h2 class="text-2xl font-bold text-gray-700 mb-2">Share Expired</h2>
      <p class="text-gray-500">This share link has expired.</p>
    </div>
    ```

    **Wrong password:**
    ```html
    <div class="bg-red-50 border border-red-200 rounded p-3 mb-4">
      <span class="text-red-800">Incorrect password. Please try again.</span>
    </div>
    ```

15. **Write Tests**

    **Handler Tests (handlers/public_share_handler_test.go):**
    - Test share page renders correctly
    - Test password-protected share shows prompt
    - Test password validation
    - Test expired share blocked
    - Test invalid token shows error
    - Test permission-based UI differences

    **Integration Tests (tests/integration/public_share_test.go):**
    - Test accessing read-only share
    - Test downloading from read share
    - Test uploading to read_upload share
    - Test upload-only cannot download
    - Test password-protected flow
    - Test access logging

    **Security Tests (tests/security/public_share_test.go):**
    - Test rate limiting on password attempts
    - Test expired share cannot be accessed
    - Test revoked share cannot be accessed
    - Test upload-only cannot download files
    - Test permission escalation attempts

### Success Criteria

- [ ] Public share page accessible without authentication
- [ ] Password protection works
- [ ] Share validation enforces expiration
- [ ] Permission-based UI displays correctly
- [ ] Download works for read/read_upload shares
- [ ] Upload works for read_upload/upload_only shares
- [ ] Upload-only shares hide download functionality
- [ ] Access logging implemented
- [ ] Error states display appropriately
- [ ] Rate limiting prevents password brute force
- [ ] All tests pass
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Start the application
go run main.go serve

# Create a test share
curl -X POST http://localhost:8090/api/shares \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"resource_type":"file","resource_id":"file123","permission_type":"read"}'

# Visit share in browser (no auth required)
open http://localhost:8090/share/{share_token}

# Run tests
go test ./handlers/... -run TestPublicShare -v
go test ./tests/integration/... -run TestPublicShare -v
go test ./tests/security/... -run TestPublicShare -v
```

### Design Specifications

**Public Share Layout:**
- Clean, minimal design
- No user account UI elements
- Centered content (max-width 1200px)
- Ample whitespace
- Clear call-to-action buttons

**Password Prompt:**
- Centered on page
- Max-width 400px
- Card style with shadow
- Large input field
- Primary button for submit

**File Display:**
- Large file icon
- File name in heading
- Metadata in gray text
- Download button prominent (blue)
- Preview centered below

**Directory Listing:**
- Similar to authenticated view but simplified
- No context menus
- No selection checkboxes
- Download/upload buttons clear

**Colors:**
- Neutral palette (grays)
- Blue for primary actions
- Yellow for warnings (expiration)
- Red for errors

### Example HTML Structure

```html
<!-- Public Share Page -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Shared: {resource_name} - FilesOnTheGo</title>
  <link rel="stylesheet" href="/static/css/output.css">
</head>
<body class="bg-gray-50">
  <!-- Header -->
  <header class="bg-white border-b">
    <div class="max-w-7xl mx-auto px-4 py-4">
      <h1 class="text-xl font-bold text-gray-800">FilesOnTheGo</h1>
    </div>
  </header>

  <!-- Content -->
  <main class="max-w-4xl mx-auto px-4 py-8">
    <!-- Share Info -->
    <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
      <div class="flex items-center">
        <svg class="w-5 h-5 text-blue-600 mr-2"><!-- share icon --></svg>
        <div>
          <div class="font-medium text-blue-900">Read-Only Share</div>
          <div class="text-sm text-blue-700">You can view and download this file</div>
        </div>
      </div>
    </div>

    <!-- Expiration Warning (if applicable) -->
    {% if expires_soon %}
    <div class="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6">
      <svg class="inline w-5 h-5 text-yellow-600 mr-2"><!-- clock --></svg>
      <span class="text-yellow-800">This share expires on {expiration_date}</span>
    </div>
    {% endif %}

    <!-- File View -->
    <div class="bg-white rounded-lg shadow-sm p-8 text-center">
      <svg class="w-24 h-24 mx-auto text-red-500 mb-4"><!-- PDF icon --></svg>
      <h2 class="text-2xl font-bold mb-2">{filename}</h2>
      <p class="text-gray-600 mb-4">{file_size} · {mime_type}</p>

      {% if can_download %}
      <button hx-get="/share/{token}/download/{file_id}"
              class="bg-blue-600 text-white px-8 py-3 rounded-lg hover:bg-blue-700 text-lg">
        <svg class="inline w-5 h-5 mr-2"><!-- download icon --></svg>
        Download File
      </button>
      {% endif %}
    </div>
  </main>

  <!-- Footer -->
  <footer class="mt-12 text-center text-gray-500 text-sm py-4">
    <p>Powered by FilesOnTheGo</p>
  </footer>
</body>
</html>

<!-- Password Prompt (if protected) -->
<div class="min-h-screen flex items-center justify-center">
  <div class="bg-white rounded-lg shadow-lg p-8 max-w-md w-full">
    <h2 class="text-2xl font-bold mb-4">Password Required</h2>
    <p class="text-gray-600 mb-6">This share is protected. Please enter the password to access.</p>

    <form hx-post="/share/{token}/validate"
          hx-target="#error-message">
      <input type="password"
             name="password"
             placeholder="Enter password"
             class="w-full border rounded-lg p-3 mb-4"
             autofocus>

      <div id="error-message" class="mb-4"></div>

      <button type="submit"
              class="w-full bg-blue-600 text-white py-3 rounded-lg hover:bg-blue-700">
        Access Share
      </button>
    </form>
  </div>
</div>
```

### References

- DESIGN.md: Sharing & Permissions section
- CLAUDE.md: Security Guidelines
- OWASP: Authentication Best Practices

### Notes

- Implement session-based password validation (don't require password on every request)
- Set appropriate cache headers (no-cache for password-protected)
- Log all access attempts for security auditing
- Implement CAPTCHA for password attempts after failures (optional)
- Add share preview metadata for social sharing (Open Graph tags)
- Consider adding download limits per share
- Implement view-only mode for documents (Google Docs style, optional)
- Add watermarking for shared files (optional)
- Track unique visitors vs total accesses
