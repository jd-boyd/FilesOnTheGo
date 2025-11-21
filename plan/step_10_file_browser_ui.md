# Step 10: File Browser UI Component

## Overview
Create the interactive file browser UI using HTMX for dynamic file/folder listing, navigation, context menus, and file operations.

## Dependencies
- Step 05: Basic UI layout (requires templates structure)
- Step 06: File upload handler (requires upload API)
- Step 07: File download handler (requires download API)
- Step 08: Directory management (requires directory API)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 10 of the FilesOnTheGo project. Your task is to create an interactive file browser UI using HTMX and Tailwind CSS.

### Commit Message Instructions

When you complete this step and are ready to commit your changes, use the following commit message format:

**First line (used for PR):**
```
feat: create interactive file browser UI with HTMX
```

**Full commit message:**
```
feat: create interactive file browser UI with HTMX

Build comprehensive file browser interface with dynamic interactions,
context menus, and keyboard shortcuts using HTMX and Tailwind CSS.

Includes:
- File list component with grid and list view options
- Individual file/folder item components with icons
- Action toolbar with upload, new folder, view toggle, sort
- Context menu for file operations (download, rename, move, delete, share)
- File details modal for metadata display
- Main file browser page with breadcrumb navigation
- HTMX interactions for navigation, downloads, deletes
- Storage usage indicator
- Search functionality with debounced input
- Sort and filter options (name, date, size, type)
- Multi-select with batch operations
- Keyboard shortcuts (Delete, Ctrl+A, Escape, arrow keys)
- Drag and drop upload support (optional)
- Loading states and error handling
- Toast notifications for user feedback
- Responsive design for mobile, tablet, and desktop
- Accessibility features (ARIA labels, keyboard navigation)
- UI component tests
- Integration tests for complete flows

All tests passing
Accessibility: WCAG 2.1 AA compliant
```

Use this exact format when committing your work.

### Tasks

1. **Create templates/components/file-list.html**

   File/directory list component:
   ```html
   - Grid or list view toggle
   - Each item shows:
     - Icon (folder/file type)
     - Name
     - Size (for files)
     - Modified date
     - Context menu trigger
   - Empty state message
   - Loading state
   - Selection checkboxes (for batch operations)
   ```

2. **Create templates/components/file-item.html**

   Individual file/folder item:
   ```html
   - Clickable name (download file or navigate directory)
   - File icon based on MIME type
   - File metadata (size, date)
   - Context menu button (three dots)
   - Checkbox for selection
   - HTMX attributes for actions
   ```

3. **Create templates/components/file-actions.html**

   Action toolbar:
   ```html
   - New Folder button
   - Upload File button
   - View toggle (grid/list)
   - Sort dropdown (name, date, size, type)
   - Search box
   - Batch actions (download selected, delete selected)
   ```

4. **Create templates/components/context-menu.html**

   Right-click/action menu:
   ```html
   - Download
   - Rename
   - Move
   - Delete
   - Share
   - Properties/Details
   - Position dynamically near clicked item
   - Close on outside click
   ```

5. **Create templates/components/file-details-modal.html**

   File details dialog:
   ```html
   - File name
   - Size
   - Type (MIME)
   - Created date
   - Modified date
   - Owner
   - Path
   - Checksum
   - Share links (if any)
   ```

6. **Create templates/pages/files.html**

   Main file browser page:
   ```html
   - Breadcrumb navigation at top
   - Action toolbar
   - File/folder list
   - Storage usage indicator
   - HTMX attributes for dynamic loading
   ```

7. **Implement HTMX Interactions**

   **Navigate to directory:**
   ```html
   <a hx-get="/api/directories/{id}"
      hx-target="#file-list"
      hx-push-url="true">
     Folder Name
   </a>
   ```

   **Download file:**
   ```html
   <button hx-get="/api/files/{id}/download"
           hx-indicator="#loading">
     Download
   </button>
   ```

   **Delete file:**
   ```html
   <button hx-delete="/api/files/{id}"
           hx-confirm="Delete this file?"
           hx-target="closest .file-item"
           hx-swap="outerHTML swap:1s">
     Delete
   </button>
   ```

   **Rename file:**
   ```html
   <form hx-patch="/api/files/{id}"
         hx-target="closest .file-item">
     <input name="name" value="current name">
   </form>
   ```

8. **Implement JavaScript Utilities (static/js/file-browser.js)**

   Minimal JavaScript for:
   - Context menu positioning
   - Keyboard shortcuts (Delete, Ctrl+A, etc.)
   - File size formatting
   - Date formatting
   - Clipboard operations (copy share link)
   - Multi-select with Shift+Click

9. **Implement Drag and Drop (Optional)**

   ```javascript
   - Drop zone for file upload
   - Visual feedback on dragover
   - Multiple file support
   - Progress indicators
   ```

10. **Add Keyboard Shortcuts**

    - `Delete` - Delete selected files
    - `Ctrl/Cmd + A` - Select all
    - `Ctrl/Cmd + D` - Deselect all
    - `Ctrl/Cmd + N` - New folder
    - `Ctrl/Cmd + U` - Upload
    - `Escape` - Close modals/menus
    - Arrow keys - Navigate items

11. **Implement Sort and Filter**

    Server-side sorting:
    ```html
    <select hx-get="/api/files"
            hx-include="[name='directory_id']"
            hx-target="#file-list"
            name="sort">
      <option value="name">Name</option>
      <option value="date">Date Modified</option>
      <option value="size">Size</option>
      <option value="type">Type</option>
    </select>
    ```

12. **Implement Search**

    ```html
    <input type="search"
           name="search"
           hx-get="/api/files/search"
           hx-trigger="keyup changed delay:500ms"
           hx-target="#file-list"
           placeholder="Search files...">
    ```

13. **Add Loading States**

    ```html
    <div id="loading" class="htmx-indicator">
      <svg class="animate-spin...">...</svg>
      Loading...
    </div>
    ```

14. **Implement Error Handling**

    ```javascript
    document.body.addEventListener('htmx:responseError', function(evt) {
      showToast('Error: ' + evt.detail.error, 'error');
    });
    ```

15. **Write Tests**

    **UI Component Tests (tests/ui/file_browser_test.go):**
    - Test file list renders correctly
    - Test directory navigation works
    - Test sorting updates list
    - Test search filters results
    - Test context menu appears
    - Test modals open/close

    **Integration Tests:**
    - Test full navigation flow
    - Test file operations update UI
    - Test HTMX responses correctly
    - Test keyboard shortcuts work

### Success Criteria

- [ ] File browser displays files and folders
- [ ] Navigation works (breadcrumbs, folder clicks)
- [ ] Context menus functional
- [ ] Sorting and filtering work
- [ ] Search works
- [ ] File operations (rename, delete, download) work via HTMX
- [ ] Loading states display
- [ ] Error handling shows appropriate messages
- [ ] Keyboard shortcuts work
- [ ] Responsive on mobile/tablet/desktop
- [ ] All tests pass
- [ ] Code follows CLAUDE.md guidelines
- [ ] Accessibility requirements met

### Testing Commands

```bash
# Start the application
go run main.go serve

# Visit in browser
open http://localhost:8090/files

# Run UI tests (if using headless browser)
go test ./tests/ui/... -v
```

### Design Specifications

**File Icons:**
- Use icon library (Heroicons, Font Awesome, or Tabler Icons)
- Different icons for folders, PDFs, images, documents, etc.
- Consistent sizing (24x24px or 32x32px)

**Layout:**
- List view: Rows with columns (Name, Size, Modified)
- Grid view: Cards with centered icons and names below
- Responsive: Grid on desktop, list on mobile

**Colors:**
- Hover states: Light gray background
- Selected items: Blue background with white text
- Folders: Blue/yellow icon
- Files: Gray icon (type-specific)

**Spacing:**
- Adequate padding in list items (16px vertical, 12px horizontal)
- Grid items: 150px width with 16px gap

**Animations:**
- Smooth transitions on hover (0.2s)
- Fade in/out for modals (0.3s)
- Slide animations for context menus

### Example HTML Structure

```html
<!-- File list container -->
<div id="file-list" class="space-y-2">
  <!-- Folder item -->
  <div class="file-item folder flex items-center p-3 hover:bg-gray-100 rounded cursor-pointer"
       data-id="dir123">
    <input type="checkbox" class="mr-3">
    <svg class="w-6 h-6 text-blue-500 mr-3"><!-- folder icon --></svg>
    <div class="flex-1">
      <a href="/files/dir123"
         hx-get="/api/directories/dir123"
         hx-target="#file-list"
         hx-push-url="true"
         class="font-medium">
        Documents
      </a>
    </div>
    <span class="text-sm text-gray-500 mr-4">Nov 21, 2025</span>
    <button class="context-menu-btn" data-id="dir123">
      <svg class="w-5 h-5"><!-- three dots icon --></svg>
    </button>
  </div>

  <!-- File item -->
  <div class="file-item file flex items-center p-3 hover:bg-gray-100 rounded"
       data-id="file456">
    <input type="checkbox" class="mr-3">
    <svg class="w-6 h-6 text-red-500 mr-3"><!-- PDF icon --></svg>
    <div class="flex-1">
      <span class="font-medium">report.pdf</span>
    </div>
    <span class="text-sm text-gray-500 mr-4">2.4 MB</span>
    <span class="text-sm text-gray-500 mr-4">Nov 20, 2025</span>
    <button hx-get="/api/files/file456/download"
            class="text-blue-600 hover:text-blue-800 mr-2">
      Download
    </button>
    <button class="context-menu-btn" data-id="file456">
      <svg class="w-5 h-5"><!-- three dots icon --></svg>
    </button>
  </div>
</div>

<!-- Empty state -->
<div id="empty-state" class="text-center py-12">
  <svg class="w-24 h-24 mx-auto text-gray-300 mb-4"><!-- empty folder icon --></svg>
  <h3 class="text-xl font-medium text-gray-700 mb-2">No files yet</h3>
  <p class="text-gray-500 mb-4">Upload your first file to get started</p>
  <button class="bg-blue-600 text-white px-6 py-2 rounded hover:bg-blue-700">
    Upload File
  </button>
</div>
```

### References

- DESIGN.md: User Interface Design section
- CLAUDE.md: HTMX Development Guidelines
- HTMX docs: https://htmx.org/
- Tailwind UI: https://tailwindui.com/

### Notes

- Keep JavaScript minimal - HTMX handles most interactions
- Ensure accessibility (keyboard navigation, ARIA labels)
- Test with screen readers
- Optimize for performance (virtual scrolling for large lists)
- Implement infinite scroll or pagination
- Cache frequent requests
- Add breadcrumb navigation
- Show storage usage indicator
- Consider adding file preview (images, PDFs)
