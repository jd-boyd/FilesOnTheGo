# Step 12: Share Creation UI

## Overview
Create the user interface for creating and managing share links with permission selection, expiration settings, password protection, and link management.

## Dependencies
- Step 05: Basic UI layout (requires templates structure)
- Step 09: Share service (requires share API)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 12 of the FilesOnTheGo project. Your task is to create an intuitive share link management UI with HTMX.

### Tasks

1. **Create templates/components/share-button.html**

   Share button in context menu:
   ```html
   - Share icon
   - "Share" text
   - Opens share modal
   - Available for files and directories
   - Shows existing share indicator
   ```

2. **Create templates/components/share-modal.html**

   Share creation/management dialog:
   ```html
   - Resource name and type display
   - Permission type selector (radio buttons):
     * Read-only
     * Read & Upload
     * Upload-only
   - Password protection toggle and input
   - Expiration date picker
   - Generate/Update Share button
   - Existing shares list
   - Close button
   ```

3. **Create templates/components/share-link-display.html**

   Share link display:
   ```html
   - Generated share URL (read-only input)
   - Copy to clipboard button
   - QR code (optional)
   - Social share buttons (optional)
   - Instructions text
   ```

4. **Create templates/components/share-list-item.html**

   Existing share item:
   ```html
   - Share URL (truncated)
   - Permission type badge
   - Expiration date (if set)
   - Access count
   - Created date
   - Copy link button
   - Revoke button
   - Edit button (change expiration)
   ```

5. **Create templates/pages/shares.html**

   Share management page:
   ```html
   - List all user's shares
   - Filter by resource type
   - Sort by date, access count
   - Bulk revoke option
   - Search shares
   ```

6. **Implement Permission Type Selector**

   ```html
   <div class="space-y-3">
     <label class="flex items-start p-3 border rounded cursor-pointer hover:bg-gray-50">
       <input type="radio" name="permission_type" value="read" class="mt-1 mr-3">
       <div>
         <div class="font-medium">Read-only</div>
         <div class="text-sm text-gray-600">
           Recipients can view and download files
         </div>
       </div>
     </label>

     <label class="flex items-start p-3 border rounded cursor-pointer hover:bg-gray-50">
       <input type="radio" name="permission_type" value="read_upload" class="mt-1 mr-3">
       <div>
         <div class="font-medium">Read & Upload</div>
         <div class="text-sm text-gray-600">
           Recipients can view, download, and upload files
         </div>
       </div>
     </label>

     <label class="flex items-start p-3 border rounded cursor-pointer hover:bg-gray-50">
       <input type="radio" name="permission_type" value="upload_only" class="mt-1 mr-3">
       <div>
         <div class="font-medium">Upload-only</div>
         <div class="text-sm text-gray-600">
           Recipients can upload files but cannot download
         </div>
       </div>
     </label>
   </div>
   ```

7. **Implement Password Protection**

   ```html
   <div class="mb-4">
     <label class="flex items-center mb-2">
       <input type="checkbox" id="password-toggle" class="mr-2">
       <span class="font-medium">Password protect this share</span>
     </label>

     <div id="password-input" class="hidden">
       <input type="password"
              name="password"
              placeholder="Enter password"
              class="w-full border rounded p-2">
       <p class="text-sm text-gray-600 mt-1">
         Recipients will need this password to access the share
       </p>
     </div>
   </div>
   ```

8. **Implement Expiration Picker**

   ```html
   <div class="mb-4">
     <label class="flex items-center mb-2">
       <input type="checkbox" id="expiration-toggle" class="mr-2">
       <span class="font-medium">Set expiration date</span>
     </label>

     <div id="expiration-input" class="hidden">
       <input type="datetime-local"
              name="expires_at"
              class="w-full border rounded p-2">
       <div class="flex space-x-2 mt-2">
         <button type="button" onclick="setExpiration(1)">1 day</button>
         <button type="button" onclick="setExpiration(7)">7 days</button>
         <button type="button" onclick="setExpiration(30)">30 days</button>
       </div>
     </div>
   </div>
   ```

9. **Implement Share Creation Form**

   ```html
   <form hx-post="/api/shares"
         hx-target="#share-result"
         id="create-share-form">

     <input type="hidden" name="resource_type" value="{type}">
     <input type="hidden" name="resource_id" value="{id}">

     <!-- Permission selector -->
     <!-- Password protection -->
     <!-- Expiration picker -->

     <button type="submit"
             class="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700">
       Generate Share Link
     </button>
   </form>

   <div id="share-result" class="mt-4"></div>
   ```

10. **Implement Copy to Clipboard**

    ```javascript
    function copyShareLink(url) {
      navigator.clipboard.writeText(url).then(() => {
        showToast('Link copied to clipboard!', 'success');
      }).catch(err => {
        showToast('Failed to copy link', 'error');
      });
    }
    ```

11. **Implement Share Revocation**

    ```html
    <button hx-delete="/api/shares/{share_id}"
            hx-confirm="Revoke this share link? It will immediately stop working."
            hx-target="closest .share-item"
            hx-swap="outerHTML swap:1s"
            class="text-red-600 hover:text-red-800">
      Revoke
    </button>
    ```

12. **Implement Access Logs Display**

    ```html
    <button hx-get="/api/shares/{share_id}/logs"
            hx-target="#access-logs"
            class="text-sm text-blue-600 hover:text-blue-800">
      View Access Logs ({count} accesses)
    </button>

    <div id="access-logs" class="mt-2"></div>
    ```

13. **Add QR Code Generation (Optional)**

    ```html
    <div class="qr-code text-center p-4 border rounded">
      <img src="/api/shares/{share_id}/qr" alt="QR Code" class="mx-auto">
      <p class="text-sm text-gray-600 mt-2">Scan to access share</p>
    </div>
    ```

14. **Implement JavaScript Utilities (static/js/share.js)**

    ```javascript
    - Toggle password input visibility
    - Toggle expiration input visibility
    - Set expiration presets (1 day, 7 days, etc.)
    - Copy link to clipboard
    - Format dates for display
    - Generate QR code (using library)
    ```

15. **Write Tests**

    **UI Tests (tests/ui/share_test.go):**
    - Test share modal opens
    - Test permission selection works
    - Test password protection toggle
    - Test expiration picker
    - Test share creation form submission
    - Test share link display
    - Test copy to clipboard
    - Test share revocation
    - Test access logs display

    **Integration Tests:**
    - Test end-to-end share creation
    - Test share appears in list
    - Test share can be revoked
    - Test share settings are saved correctly

### Success Criteria

- [ ] Share button available in file/folder context menu
- [ ] Share modal displays correctly
- [ ] Permission type selection works
- [ ] Password protection functional
- [ ] Expiration date picker works
- [ ] Share link generated and displayed
- [ ] Copy to clipboard works
- [ ] Existing shares listed
- [ ] Share revocation works
- [ ] Access logs viewable
- [ ] Responsive on all devices
- [ ] Accessible (keyboard navigation, ARIA labels)
- [ ] All tests pass
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Start the application
go run main.go serve

# Visit in browser
open http://localhost:8090/files

# Test share creation:
# 1. Right-click a file
# 2. Click "Share"
# 3. Select permission type
# 4. Optionally add password/expiration
# 5. Click "Generate Share Link"
# 6. Copy link and test in incognito window

# Run automated tests
go test ./tests/ui/... -run TestShare -v
```

### Design Specifications

**Share Modal:**
- Max width: 600px
- Padding: 24px
- White background
- Drop shadow
- Centered on screen

**Permission Badges:**
- Read-only: Blue (#3B82F6)
- Read & Upload: Green (#10B981)
- Upload-only: Yellow (#F59E0B)
- Small padding, rounded, uppercase text

**Share Link Display:**
- Light gray background
- Monospace font
- Select all on click
- Copy button on right
- Success feedback on copy

**Existing Shares:**
- List with borders
- Hover effect
- Metadata in gray text
- Action buttons aligned right

### Example HTML Structure

```html
<!-- Share Modal -->
<div id="share-modal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
  <div class="bg-white rounded-lg p-6 max-w-2xl w-full mx-4">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-xl font-bold">Share "{resource_name}"</h2>
      <button onclick="closeShareModal()" class="text-gray-500 hover:text-gray-700">
        <svg class="w-6 h-6"><!-- X icon --></svg>
      </button>
    </div>

    <!-- Create New Share -->
    <form hx-post="/api/shares" hx-target="#share-result">
      <input type="hidden" name="resource_type" value="file">
      <input type="hidden" name="resource_id" value="file123">

      <div class="mb-4">
        <label class="block font-medium mb-2">Permission Level</label>
        <!-- Permission radio buttons -->
      </div>

      <div class="mb-4">
        <!-- Password protection -->
      </div>

      <div class="mb-4">
        <!-- Expiration -->
      </div>

      <button type="submit"
              class="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700">
        Generate Share Link
      </button>
    </form>

    <!-- Share Result -->
    <div id="share-result" class="mt-4"></div>

    <!-- Existing Shares -->
    <div class="mt-6">
      <h3 class="font-medium mb-2">Existing Shares</h3>
      <div class="space-y-2">
        <div class="share-item border rounded p-3">
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="flex items-center space-x-2 mb-1">
                <span class="text-sm font-mono bg-gray-100 px-2 py-1 rounded truncate">
                  https://files.example.com/share/abc123...
                </span>
                <button onclick="copyShareLink('...')"
                        class="text-blue-600 hover:text-blue-800">
                  Copy
                </button>
              </div>
              <div class="flex items-center space-x-2 text-sm text-gray-600">
                <span class="badge bg-blue-100 text-blue-800 px-2 py-1 rounded text-xs">
                  READ-ONLY
                </span>
                <span>Expires: Nov 28, 2025</span>
                <span>Accesses: 5</span>
              </div>
            </div>
            <button hx-delete="/api/shares/share123"
                    hx-confirm="Revoke this share?"
                    hx-target="closest .share-item"
                    hx-swap="outerHTML swap:1s"
                    class="text-red-600 hover:text-red-800 ml-2">
              Revoke
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
```

### References

- DESIGN.md: Sharing & Permissions section
- CLAUDE.md: HTMX Development Guidelines
- Clipboard API: MDN documentation

### Notes

- Generate unique share tokens on server (never client-side)
- Validate expiration dates on both client and server
- Show clear permission descriptions
- Implement share link analytics (views, downloads)
- Consider adding custom share slugs (optional)
- Add share preview before finalizing
- Implement share templates for common scenarios
- Allow updating existing shares (change expiration, password)
- Show share link security recommendations
