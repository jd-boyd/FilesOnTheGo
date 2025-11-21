# Step 11: Upload UI Component

## Overview
Create a user-friendly file upload interface with drag-and-drop, progress indicators, multi-file support, and error handling using HTMX.

## Dependencies
- Step 05: Basic UI layout (requires templates structure)
- Step 06: File upload handler (requires upload API)

## Duration Estimate
30 minutes

## Agent Prompt

You are implementing Step 11 of the FilesOnTheGo project. Your task is to create an intuitive file upload UI with progress tracking and drag-and-drop support.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
feat: create file upload UI with drag-and-drop support
```

**Full commit message:**
```
feat: create file upload UI with drag-and-drop support

Build comprehensive file upload interface with drag-and-drop,
progress tracking, and multi-file support using HTMX and JavaScript.

Includes:
- Upload button component in toolbar
- Upload modal dialog with file selection
- Drag-and-drop zone with visual feedback
- Progress indicators with percentage and bars
- Multi-file upload with queue management
- File validation (size, type) before upload
- Sequential or parallel upload options
- Error handling for all upload scenarios
- Success feedback with toast notifications
- Upload cancellation support
- File previews for images
- Keyboard shortcuts (Ctrl+U, Escape)
- Upload restrictions display
- HTMX integration for seamless updates
- JavaScript utilities for drag-and-drop
- Responsive design for all devices
- Accessibility features
- UI component tests
- Integration tests for upload flows

All tests passing
Accessibility: Keyboard navigation and ARIA labels
```

Use this exact format when committing your work.

### Tasks

1. **Create templates/components/upload-button.html**

   Primary upload button:
   ```html
   - Prominent button in toolbar
   - Opens file picker on click
   - Shows keyboard shortcut (Ctrl+U)
   - Icon + text
   - Responsive (icon only on mobile)
   ```

2. **Create templates/components/upload-modal.html**

   Upload dialog:
   ```html
   - Modal overlay
   - File selection area
   - Drag-and-drop zone
   - File list with progress bars
   - Cancel/close button
   - Upload button (if not auto-upload)
   - Error messages
   ```

3. **Create templates/components/upload-progress.html**

   Progress indicator:
   ```html
   - File name
   - File size
   - Progress bar (0-100%)
   - Percentage text
   - Cancel button per file
   - Success/error icons
   - Overall progress (if multiple files)
   ```

4. **Create templates/components/drop-zone.html**

   Drag-and-drop area:
   ```html
   - Dashed border
   - Large icon
   - "Drag files here or click to browse" text
   - Visual feedback on dragover
   - Support for multiple files
   - File type restrictions display
   - Max size display
   ```

5. **Implement Upload Form (templates/components/upload-form.html)**

   ```html
   <form hx-post="/api/files/upload"
         hx-encoding="multipart/form-data"
         hx-target="#file-list"
         hx-swap="afterbegin"
         id="upload-form">
     <input type="file"
            name="file"
            multiple
            id="file-input"
            class="hidden">
     <input type="hidden"
            name="directory_id"
            value="{current_directory}">
     <!-- Drop zone UI -->
   </form>
   ```

6. **Implement JavaScript (static/js/upload.js)**

   **File Selection:**
   ```javascript
   - Handle file input change
   - Validate file size and type
   - Add files to upload queue
   - Display file previews (for images)
   ```

   **Drag and Drop:**
   ```javascript
   - Prevent default drag behaviors
   - Highlight drop zone on dragover
   - Handle drop event
   - Extract files from DataTransfer
   - Add to upload queue
   ```

   **Upload Progress:**
   ```javascript
   - Use HTMX events or XMLHttpRequest
   - Update progress bars
   - Handle success/error per file
   - Show completion messages
   - Auto-close on success (optional)
   ```

   **Validation:**
   ```javascript
   - Check file size limits
   - Validate file types
   - Show error messages
   - Prevent upload if invalid
   ```

7. **Implement Progress Tracking**

   Using HTMX extensions or custom JavaScript:
   ```javascript
   document.getElementById('upload-form').addEventListener('htmx:xhr:progress', function(evt) {
     const progress = (evt.detail.loaded / evt.detail.total) * 100;
     updateProgressBar(progress);
   });
   ```

8. **Implement Multi-File Upload**

   **Sequential upload:**
   - Upload files one at a time
   - Show individual progress
   - Continue on error
   - Report success/failure per file

   **Parallel upload (optional):**
   - Upload multiple files simultaneously
   - Limit concurrent uploads (e.g., 3 at a time)
   - Show overall and individual progress

9. **Implement Error Handling**

   Display errors for:
   - File too large
   - Invalid file type
   - Network errors
   - Server errors (quota exceeded, etc.)
   - Timeout errors

   ```html
   <div class="error-message bg-red-100 text-red-800 p-3 rounded">
     <svg class="w-5 h-5 inline"><!-- error icon --></svg>
     <span>{error_message}</span>
     <button class="float-right">×</button>
   </div>
   ```

10. **Implement Success Feedback**

    ```html
    <div class="success-message bg-green-100 text-green-800 p-3 rounded">
      <svg class="w-5 h-5 inline"><!-- checkmark icon --></svg>
      <span>File uploaded successfully!</span>
    </div>
    ```

11. **Add File Previews (for images)**

    ```html
    <div class="file-preview">
      <img src="{preview_url}" alt="{filename}" class="w-20 h-20 object-cover">
      <span class="text-sm">{filename}</span>
    </div>
    ```

12. **Implement Cancel Upload**

    ```javascript
    - Abort XMLHttpRequest
    - Remove from upload queue
    - Show cancellation message
    - Clean up UI
    ```

13. **Add Upload Restrictions Display**

    ```html
    <div class="upload-info text-sm text-gray-600">
      <p>Max file size: 5 GB</p>
      <p>Allowed types: All files</p>
      <p>Multiple files supported</p>
    </div>
    ```

14. **Implement Keyboard Shortcuts**

    - `Ctrl/Cmd + U` - Open upload dialog
    - `Escape` - Close upload dialog
    - `Enter` - Start upload (if not auto-upload)

15. **Write Tests**

    **UI Tests (tests/ui/upload_test.go):**
    - Test upload button opens modal
    - Test file selection adds files to queue
    - Test drag-and-drop works
    - Test progress updates
    - Test error messages display
    - Test success messages display
    - Test file list updates after upload

    **Integration Tests:**
    - Test end-to-end upload flow
    - Test multi-file upload
    - Test upload with errors
    - Test cancel upload
    - Test quota enforcement shows error

### Success Criteria

- [ ] Upload button functional
- [ ] File picker works
- [ ] Drag-and-drop works
- [ ] Progress bars update correctly
- [ ] Multi-file upload supported
- [ ] Error handling displays messages
- [ ] Success feedback shown
- [ ] File list updates after upload
- [ ] Cancel upload works
- [ ] Validation prevents invalid uploads
- [ ] Responsive on all devices
- [ ] Accessible (keyboard navigation)
- [ ] All tests pass
- [ ] Code follows CLAUDE.md guidelines

### Testing Commands

```bash
# Start the application
go run main.go serve

# Visit in browser
open http://localhost:8090/files

# Test upload manually:
# 1. Click upload button
# 2. Select files
# 3. Verify progress
# 4. Verify files appear in list

# Run automated tests
go test ./tests/ui/... -run TestUpload -v
```

### Design Specifications

**Upload Button:**
- Primary blue color (#3B82F6)
- White text
- Icon + "Upload" text
- Hover: Darker blue (#2563EB)
- Active: Even darker (#1D4ED8)
- Shadow on hover

**Drop Zone:**
- Dashed border (2px, gray)
- Large size (min 200px height)
- Centered icon and text
- Dragover state: Blue border, blue background (light)
- Cursor: pointer

**Progress Bar:**
- Height: 8px
- Background: Light gray (#E5E7EB)
- Fill: Blue (#3B82F6)
- Animated (smooth transition)
- Rounded corners

**File Item in Queue:**
- File icon on left
- File name (truncate if long)
- File size
- Progress bar below
- Cancel button on right
- Success/error icon when complete

### Example HTML Structure

```html
<!-- Upload Modal -->
<div id="upload-modal" class="fixed inset-0 bg-black bg-opacity-50 hidden">
  <div class="bg-white rounded-lg p-6 max-w-2xl mx-auto mt-20">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-xl font-bold">Upload Files</h2>
      <button onclick="closeUploadModal()" class="text-gray-500 hover:text-gray-700">
        <svg class="w-6 h-6"><!-- X icon --></svg>
      </button>
    </div>

    <!-- Drop Zone -->
    <div id="drop-zone"
         class="border-2 border-dashed border-gray-300 rounded-lg p-12 text-center cursor-pointer hover:border-blue-500 transition">
      <svg class="w-16 h-16 mx-auto text-gray-400 mb-4"><!-- upload icon --></svg>
      <p class="text-lg mb-2">Drag files here or click to browse</p>
      <p class="text-sm text-gray-500">Max file size: 5 GB</p>
      <input type="file" id="file-input" multiple class="hidden">
    </div>

    <!-- Upload Queue -->
    <div id="upload-queue" class="mt-6 space-y-2">
      <!-- File items will be added here -->
    </div>

    <!-- Actions -->
    <div class="mt-6 flex justify-end space-x-2">
      <button onclick="closeUploadModal()"
              class="px-4 py-2 border rounded hover:bg-gray-50">
        Cancel
      </button>
      <button onclick="startUpload()"
              class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
        Upload
      </button>
    </div>
  </div>
</div>

<!-- File Item Template -->
<template id="file-item-template">
  <div class="file-upload-item border rounded p-3 flex items-center">
    <svg class="w-8 h-8 text-gray-400 mr-3"><!-- file icon --></svg>
    <div class="flex-1">
      <div class="flex justify-between items-center mb-1">
        <span class="font-medium file-name">filename.pdf</span>
        <span class="text-sm text-gray-500 file-size">2.4 MB</span>
      </div>
      <div class="progress-bar bg-gray-200 rounded h-2 overflow-hidden">
        <div class="progress-fill bg-blue-600 h-full transition-all" style="width: 0%"></div>
      </div>
      <div class="flex justify-between items-center mt-1">
        <span class="text-xs text-gray-500 progress-text">0%</span>
        <button class="text-red-600 text-xs cancel-btn">Cancel</button>
      </div>
    </div>
  </div>
</template>
```

### JavaScript Example

```javascript
// Drop zone handling
const dropZone = document.getElementById('drop-zone');
const fileInput = document.getElementById('file-input');

dropZone.addEventListener('click', () => fileInput.click());

dropZone.addEventListener('dragover', (e) => {
  e.preventDefault();
  dropZone.classList.add('border-blue-500', 'bg-blue-50');
});

dropZone.addEventListener('dragleave', () => {
  dropZone.classList.remove('border-blue-500', 'bg-blue-50');
});

dropZone.addEventListener('drop', (e) => {
  e.preventDefault();
  dropZone.classList.remove('border-blue-500', 'bg-blue-50');

  const files = Array.from(e.dataTransfer.files);
  addFilesToQueue(files);
});

fileInput.addEventListener('change', (e) => {
  const files = Array.from(e.target.files);
  addFilesToQueue(files);
});

function addFilesToQueue(files) {
  files.forEach(file => {
    if (validateFile(file)) {
      createFileItem(file);
    }
  });
}

function validateFile(file) {
  const maxSize = 5 * 1024 * 1024 * 1024; // 5GB
  if (file.size > maxSize) {
    showError(`File ${file.name} is too large`);
    return false;
  }
  return true;
}
```

### References

- DESIGN.md: User Interface Design section
- CLAUDE.md: HTMX Development Guidelines
- MDN: File API and Drag and Drop

### Notes

- Use chunked uploads for large files
- Implement resume functionality for interrupted uploads
- Show estimated time remaining
- Allow upload queue management (reorder, remove)
- Auto-retry on network errors
- Compress images before upload (optional)
- Generate thumbnails client-side for preview
- Support paste from clipboard
