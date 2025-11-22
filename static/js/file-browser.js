/**
 * FilesOnTheGo - File Browser JavaScript Utilities
 * Handles file browser interactions, context menus, modals, and keyboard shortcuts
 */

// State management
const fileBrowserState = {
    selectedFiles: new Set(),
    currentDirectory: null,
    viewMode: 'list',
    contextMenuTarget: null,
    pendingUploadFiles: [],
    isUploading: false
};

// ============================================
// Initialization
// ============================================

document.addEventListener('DOMContentLoaded', function() {
    initFileBrowser();
});

function initFileBrowser() {
    // Get current directory from data attribute
    const mainContent = document.getElementById('main-content');
    if (mainContent) {
        fileBrowserState.currentDirectory = mainContent.dataset.directoryId || null;
    }

    // Initialize event listeners
    initKeyboardShortcuts();
    initDragAndDrop();
    initContextMenuClose();
    initSearchInput();

    // Listen for HTMX events
    document.body.addEventListener('htmx:afterSwap', handleAfterSwap);
    document.body.addEventListener('htmx:responseError', handleHtmxError);
}

function handleAfterSwap(event) {
    // Re-initialize components after HTMX swaps content
    updateSelectionUI();
    closeContextMenu();
}

function handleHtmxError(event) {
    const xhr = event.detail.xhr;
    let message = 'An error occurred';

    try {
        const response = JSON.parse(xhr.responseText);
        message = response.error?.message || response.message || message;
    } catch (e) {
        // Response is not JSON
    }

    showToast('error', 'Error', message);
}

// ============================================
// Selection Management
// ============================================

function toggleFileSelection(id) {
    if (fileBrowserState.selectedFiles.has(id)) {
        fileBrowserState.selectedFiles.delete(id);
    } else {
        fileBrowserState.selectedFiles.add(id);
    }
    updateSelectionUI();
}

function toggleSelectAll(checked) {
    const checkboxes = document.querySelectorAll('.file-checkbox');
    checkboxes.forEach(checkbox => {
        const id = checkbox.dataset.id;
        if (checked) {
            fileBrowserState.selectedFiles.add(id);
        } else {
            fileBrowserState.selectedFiles.delete(id);
        }
        checkbox.checked = checked;
    });
    updateSelectionUI();
}

function clearSelection() {
    fileBrowserState.selectedFiles.clear();
    const checkboxes = document.querySelectorAll('.file-checkbox');
    checkboxes.forEach(checkbox => checkbox.checked = false);
    updateSelectionUI();
}

function updateSelectionUI() {
    const count = fileBrowserState.selectedFiles.size;
    const batchActions = document.getElementById('batch-actions');
    const selectedCount = document.getElementById('selected-count');
    const selectAllCheckbox = document.getElementById('select-all');

    if (batchActions) {
        if (count > 0) {
            batchActions.classList.remove('hidden');
            batchActions.classList.add('flex');
        } else {
            batchActions.classList.add('hidden');
            batchActions.classList.remove('flex');
        }
    }

    if (selectedCount) {
        selectedCount.textContent = count;
    }

    // Update select all checkbox state
    if (selectAllCheckbox) {
        const totalCheckboxes = document.querySelectorAll('.file-checkbox').length;
        selectAllCheckbox.checked = count > 0 && count === totalCheckboxes;
        selectAllCheckbox.indeterminate = count > 0 && count < totalCheckboxes;
    }

    // Update individual checkboxes
    fileBrowserState.selectedFiles.forEach(id => {
        const checkbox = document.querySelector(`.file-checkbox[data-id="${id}"]`);
        if (checkbox) checkbox.checked = true;
    });
}

// ============================================
// Context Menu
// ============================================

function showContextMenu(event, id, type) {
    event.preventDefault();
    event.stopPropagation();

    const menu = document.getElementById('context-menu');
    if (!menu) return;

    // Store target info
    fileBrowserState.contextMenuTarget = { id, type };

    // Show/hide type-specific options
    const fileOptions = document.getElementById('context-menu-file-options');
    const dirOptions = document.getElementById('context-menu-directory-options');

    if (fileOptions) fileOptions.classList.toggle('hidden', type !== 'file');
    if (dirOptions) dirOptions.classList.toggle('hidden', type !== 'directory');

    // Position menu
    const x = event.clientX;
    const y = event.clientY;

    // Ensure menu stays within viewport
    const menuWidth = 200; // approximate width
    const menuHeight = 300; // approximate height
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    const left = Math.min(x, viewportWidth - menuWidth - 10);
    const top = Math.min(y, viewportHeight - menuHeight - 10);

    menu.style.left = left + 'px';
    menu.style.top = top + 'px';
    menu.classList.remove('hidden');

    // Focus first menu item for accessibility
    const firstItem = menu.querySelector('a[role="menuitem"]');
    if (firstItem) firstItem.focus();
}

function closeContextMenu() {
    const menu = document.getElementById('context-menu');
    if (menu) {
        menu.classList.add('hidden');
    }
    fileBrowserState.contextMenuTarget = null;
}

function initContextMenuClose() {
    // Close context menu on click outside
    document.addEventListener('click', function(event) {
        const menu = document.getElementById('context-menu');
        if (menu && !menu.contains(event.target)) {
            closeContextMenu();
        }
    });

    // Close on escape
    document.addEventListener('keydown', function(event) {
        if (event.key === 'Escape') {
            closeContextMenu();
        }
    });
}

function contextMenuAction(action) {
    const target = fileBrowserState.contextMenuTarget;
    if (!target) return;

    closeContextMenu();

    switch (action) {
        case 'download':
            downloadFile(target.id);
            break;
        case 'open':
            navigateToDirectory(target.id);
            break;
        case 'share':
            openShareModal(target.id, target.type);
            break;
        case 'rename':
            openRenameModal(target.id, target.type);
            break;
        case 'move':
            openMoveModal(target.id, target.type);
            break;
        case 'copy-link':
            copyDirectLink(target.id, target.type);
            break;
        case 'details':
            openFileDetailsModal(target.id, target.type);
            break;
        case 'delete':
            openDeleteModal(target.id, target.type);
            break;
    }
}

// ============================================
// File Operations
// ============================================

function downloadFile(id) {
    window.location.href = `/api/files/${id}/download`;
}

function navigateToDirectory(id) {
    htmx.ajax('GET', `/api/directories/${id}`, {
        target: '#file-list-container',
        swap: 'innerHTML'
    });
    history.pushState(null, '', `/files/${id}`);
    fileBrowserState.currentDirectory = id;
}

function downloadSelected() {
    const selected = Array.from(fileBrowserState.selectedFiles);
    if (selected.length === 0) {
        showToast('warning', 'No files selected');
        return;
    }

    if (selected.length === 1) {
        downloadFile(selected[0]);
    } else {
        // For multiple files, trigger batch download
        showToast('info', 'Preparing download...');
        // TODO: Implement batch download endpoint
        window.location.href = `/api/files/download-batch?ids=${selected.join(',')}`;
    }
}

function deleteSelected() {
    const selected = Array.from(fileBrowserState.selectedFiles);
    if (selected.length === 0) {
        showToast('warning', 'No files selected');
        return;
    }

    // Get first selected item info
    const firstItem = document.querySelector(`.file-item[data-id="${selected[0]}"]`);
    const itemType = firstItem?.dataset.type || 'item';

    if (selected.length === 1) {
        openDeleteModal(selected[0], itemType);
    } else {
        // Batch delete confirmation
        openDeleteModal('batch', 'items', selected);
    }
}

// ============================================
// Modal Functions
// ============================================

// New Folder Modal
function openNewFolderModal() {
    const modal = document.getElementById('new-folder-modal');
    const input = document.getElementById('folder-name');
    if (modal) {
        modal.classList.remove('hidden');
        if (input) {
            input.value = '';
            input.focus();
        }
    }
}

function closeNewFolderModal() {
    const modal = document.getElementById('new-folder-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
}

// Upload Modal
function openUploadModal() {
    const modal = document.getElementById('upload-modal');
    if (modal) {
        modal.classList.remove('hidden');
        resetUploadState();
    }
}

function closeUploadModal() {
    const modal = document.getElementById('upload-modal');
    if (modal) {
        modal.classList.add('hidden');
        resetUploadState();
    }
}

function resetUploadState() {
    fileBrowserState.pendingUploadFiles = [];
    fileBrowserState.isUploading = false;

    const fileList = document.getElementById('upload-file-list');
    const progress = document.getElementById('upload-progress');
    const uploadBtn = document.getElementById('upload-btn');
    const fileInput = document.getElementById('file-input');

    if (fileList) {
        fileList.innerHTML = '';
        fileList.classList.add('hidden');
    }
    if (progress) progress.classList.add('hidden');
    if (uploadBtn) uploadBtn.disabled = true;
    if (fileInput) fileInput.value = '';
}

// Rename Modal
function openRenameModal(id, type) {
    const modal = document.getElementById('rename-modal');
    const input = document.getElementById('rename-input');
    const idField = document.getElementById('rename-id');
    const typeField = document.getElementById('rename-type');
    const form = document.getElementById('rename-form');

    if (!modal) return;

    // Get current name
    const item = document.querySelector(`.file-item[data-id="${id}"]`);
    const currentName = item?.dataset.name || '';

    if (input) {
        input.value = currentName;
        // Select filename without extension for files
        if (type === 'file') {
            const lastDot = currentName.lastIndexOf('.');
            if (lastDot > 0) {
                setTimeout(() => input.setSelectionRange(0, lastDot), 0);
            }
        }
    }
    if (idField) idField.value = id;
    if (typeField) typeField.value = type;

    // Set form action based on type
    if (form) {
        form.setAttribute('hx-patch', type === 'directory'
            ? `/api/directories/${id}`
            : `/api/files/${id}`);
        htmx.process(form);
    }

    modal.classList.remove('hidden');
    if (input) input.focus();
}

function closeRenameModal() {
    const modal = document.getElementById('rename-modal');
    if (modal) modal.classList.add('hidden');
}

// Delete Modal
let deleteItems = [];

function openDeleteModal(id, type, items = null) {
    const modal = document.getElementById('delete-modal');
    const itemTypeSpan = document.getElementById('delete-item-type');
    const itemNameSpan = document.getElementById('delete-item-name');
    const directoryWarning = document.getElementById('delete-directory-warning');
    const itemIdField = document.getElementById('delete-item-id');
    const itemTypeField = document.getElementById('delete-item-type-value');

    if (!modal) return;

    // Handle batch delete
    if (id === 'batch' && items) {
        deleteItems = items;
        if (itemTypeSpan) itemTypeSpan.textContent = 'items';
        if (itemNameSpan) itemNameSpan.textContent = `${items.length} items`;
        if (directoryWarning) directoryWarning.classList.add('hidden');
    } else {
        deleteItems = [id];
        const item = document.querySelector(`.file-item[data-id="${id}"]`);
        const name = item?.dataset.name || 'this item';

        if (itemTypeSpan) itemTypeSpan.textContent = type === 'directory' ? 'folder' : 'file';
        if (itemNameSpan) itemNameSpan.textContent = name;
        if (directoryWarning) {
            directoryWarning.classList.toggle('hidden', type !== 'directory');
        }
    }

    if (itemIdField) itemIdField.value = id;
    if (itemTypeField) itemTypeField.value = type;

    modal.classList.remove('hidden');
}

function closeDeleteModal() {
    const modal = document.getElementById('delete-modal');
    if (modal) modal.classList.add('hidden');
    deleteItems = [];
}

async function confirmDelete() {
    if (deleteItems.length === 0) return;

    const id = document.getElementById('delete-item-id').value;
    const type = document.getElementById('delete-item-type-value').value;

    closeDeleteModal();

    try {
        if (deleteItems.length === 1 && id !== 'batch') {
            // Single item delete
            const endpoint = type === 'directory'
                ? `/api/directories/${id}?recursive=true`
                : `/api/files/${id}`;

            const response = await fetch(endpoint, { method: 'DELETE' });

            if (!response.ok) {
                throw new Error('Failed to delete');
            }

            // Remove item from DOM
            const item = document.querySelector(`.file-item[data-id="${id}"]`);
            if (item) {
                item.style.opacity = '0';
                setTimeout(() => item.remove(), 300);
            }

            showToast('success', 'Deleted successfully');
        } else {
            // Batch delete
            for (const itemId of deleteItems) {
                const item = document.querySelector(`.file-item[data-id="${itemId}"]`);
                const itemType = item?.dataset.type || 'file';
                const endpoint = itemType === 'directory'
                    ? `/api/directories/${itemId}?recursive=true`
                    : `/api/files/${itemId}`;

                await fetch(endpoint, { method: 'DELETE' });

                if (item) {
                    item.style.opacity = '0';
                    setTimeout(() => item.remove(), 300);
                }
            }

            showToast('success', `${deleteItems.length} items deleted`);
        }

        clearSelection();

    } catch (error) {
        console.error('Delete error:', error);
        showToast('error', 'Failed to delete', error.message);
    }
}

// Move Modal
function openMoveModal(id, type) {
    const modal = document.getElementById('move-modal');
    const itemIdField = document.getElementById('move-item-id');
    const itemTypeField = document.getElementById('move-item-type');
    const itemNameSpan = document.getElementById('move-item-name');

    if (!modal) return;

    const item = document.querySelector(`.file-item[data-id="${id}"]`);
    const name = item?.dataset.name || 'this item';

    if (itemIdField) itemIdField.value = id;
    if (itemTypeField) itemTypeField.value = type;
    if (itemNameSpan) itemNameSpan.textContent = name;

    // Refresh directory tree
    htmx.trigger('#move-directory-tree', 'revealed');

    modal.classList.remove('hidden');
}

function closeMoveModal() {
    const modal = document.getElementById('move-modal');
    if (modal) modal.classList.add('hidden');
}

function selectMoveTarget(targetId) {
    document.getElementById('move-target-id').value = targetId;
    document.getElementById('move-confirm-btn').disabled = false;

    // Highlight selected directory
    document.querySelectorAll('.move-directory-option').forEach(el => {
        el.classList.remove('bg-blue-50', 'border-blue-500');
    });
    const selected = document.querySelector(`.move-directory-option[data-id="${targetId}"]`);
    if (selected) {
        selected.classList.add('bg-blue-50', 'border-blue-500');
    }
}

async function confirmMove() {
    const itemId = document.getElementById('move-item-id').value;
    const itemType = document.getElementById('move-item-type').value;
    const targetId = document.getElementById('move-target-id').value;

    if (!itemId || !targetId) return;

    closeMoveModal();

    try {
        const endpoint = itemType === 'directory'
            ? `/api/directories/${itemId}/move`
            : `/api/files/${itemId}/move`;

        const response = await fetch(endpoint, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ target_directory_id: targetId })
        });

        if (!response.ok) {
            throw new Error('Failed to move');
        }

        // Remove item from current view
        const item = document.querySelector(`.file-item[data-id="${itemId}"]`);
        if (item) {
            item.style.opacity = '0';
            setTimeout(() => item.remove(), 300);
        }

        showToast('success', 'Moved successfully');

    } catch (error) {
        console.error('Move error:', error);
        showToast('error', 'Failed to move', error.message);
    }
}

// Share Modal
function openShareModal(id, type) {
    const modal = document.getElementById('share-modal');
    const itemNameSpan = document.getElementById('share-item-name');
    const fileIdField = document.getElementById('share-file-id');
    const dirIdField = document.getElementById('share-directory-id');
    const resultDiv = document.getElementById('share-result');

    if (!modal) return;

    const item = document.querySelector(`.file-item[data-id="${id}"]`);
    const name = item?.dataset.name || 'this item';

    if (itemNameSpan) itemNameSpan.textContent = name;
    if (fileIdField) fileIdField.value = type === 'file' ? id : '';
    if (dirIdField) dirIdField.value = type === 'directory' ? id : '';
    if (resultDiv) resultDiv.innerHTML = '';

    modal.classList.remove('hidden');
}

function closeShareModal() {
    const modal = document.getElementById('share-modal');
    if (modal) modal.classList.add('hidden');
}

function toggleSharePassword() {
    const passwordField = document.getElementById('share-password');
    const toggle = document.getElementById('share-password-toggle');

    if (passwordField && toggle) {
        passwordField.classList.toggle('hidden', !toggle.checked);
        if (toggle.checked) {
            passwordField.focus();
        }
    }
}

// File Details Modal
async function openFileDetailsModal(id, type) {
    const modal = document.getElementById('file-details-modal');
    if (!modal) return;

    // Get item info from DOM
    const item = document.querySelector(`.file-item[data-id="${id}"]`);

    // Set basic info
    document.getElementById('details-item-id').value = id;
    document.getElementById('details-item-type').value = type;
    document.getElementById('details-name').textContent = item?.dataset.name || 'Unknown';
    document.getElementById('details-type-label').textContent = type === 'directory' ? 'Folder' : 'File';

    // Show/hide type-specific elements
    const downloadBtn = document.getElementById('details-download-btn');
    const openBtn = document.getElementById('details-open-btn');
    const sizeRow = document.getElementById('details-size-row');
    const checksumRow = document.getElementById('details-checksum-row');

    if (type === 'file') {
        if (downloadBtn) {
            downloadBtn.href = `/api/files/${id}/download`;
            downloadBtn.classList.remove('hidden');
        }
        if (openBtn) openBtn.classList.add('hidden');
        if (sizeRow) sizeRow.classList.remove('hidden');
    } else {
        if (downloadBtn) downloadBtn.classList.add('hidden');
        if (openBtn) openBtn.classList.remove('hidden');
        if (sizeRow) sizeRow.classList.add('hidden');
        if (checksumRow) checksumRow.classList.add('hidden');
    }

    modal.classList.remove('hidden');

    // Fetch detailed info from API
    try {
        const endpoint = type === 'directory'
            ? `/api/directories/${id}/details`
            : `/api/files/${id}/details`;

        const response = await fetch(endpoint);
        if (response.ok) {
            const data = await response.json();
            updateFileDetails(data, type);
        }
    } catch (error) {
        console.error('Failed to fetch details:', error);
    }
}

function updateFileDetails(data, type) {
    if (data.size !== undefined) {
        document.getElementById('details-size').textContent = formatFileSize(data.size);
    }
    if (data.mime_type) {
        document.getElementById('details-mime-type').textContent = data.mime_type;
    } else if (type === 'directory') {
        document.getElementById('details-mime-type').textContent = 'Folder';
    }
    if (data.path) {
        const pathEl = document.getElementById('details-path');
        pathEl.textContent = data.path;
        pathEl.title = data.path;
    }
    if (data.created) {
        document.getElementById('details-created').textContent = formatDate(data.created);
    }
    if (data.updated) {
        document.getElementById('details-modified').textContent = formatDate(data.updated);
    }
    if (data.checksum) {
        document.getElementById('details-checksum').textContent = data.checksum;
        document.getElementById('details-checksum-row').classList.remove('hidden');
    }
    if (data.shares && data.shares.length > 0) {
        // TODO: Display share links
        document.getElementById('details-shares-row').classList.remove('hidden');
    }
    // Image preview
    if (data.mime_type && data.mime_type.startsWith('image/')) {
        const previewContainer = document.getElementById('details-preview-container');
        const previewImage = document.getElementById('details-preview-image');
        if (previewContainer && previewImage) {
            previewImage.src = `/api/files/${data.id}/thumbnail`;
            previewContainer.classList.remove('hidden');
        }
    }
}

function closeFileDetailsModal() {
    const modal = document.getElementById('file-details-modal');
    if (modal) modal.classList.add('hidden');
}

function openDirectoryFromDetails() {
    const id = document.getElementById('details-item-id').value;
    closeFileDetailsModal();
    navigateToDirectory(id);
}

function shareFromDetails() {
    const id = document.getElementById('details-item-id').value;
    const type = document.getElementById('details-item-type').value;
    closeFileDetailsModal();
    openShareModal(id, type);
}

// ============================================
// Copy Link
// ============================================

async function copyDirectLink(id, type) {
    const baseUrl = window.location.origin;
    let url;

    if (type === 'directory') {
        url = `${baseUrl}/files/${id}`;
    } else {
        url = `${baseUrl}/api/files/${id}/download`;
    }

    await copyToClipboard(url);
}

async function copyShareLink(button) {
    const linkUrl = button.closest('[data-share-url]')?.dataset.shareUrl;
    if (linkUrl) {
        await copyToClipboard(linkUrl);
    }
}

// ============================================
// View Mode
// ============================================

function setViewMode(mode) {
    fileBrowserState.viewMode = mode;

    const listView = document.getElementById('file-list');
    const gridView = document.getElementById('file-grid');
    const listBtn = document.getElementById('view-list-btn');
    const gridBtn = document.getElementById('view-grid-btn');

    if (mode === 'grid') {
        if (listView) listView.classList.add('hidden');
        if (gridView) gridView.classList.remove('hidden');
        if (listBtn) {
            listBtn.classList.remove('bg-gray-100', 'text-gray-700');
            listBtn.classList.add('text-gray-500');
        }
        if (gridBtn) {
            gridBtn.classList.add('bg-gray-100', 'text-gray-700');
            gridBtn.classList.remove('text-gray-500');
        }
    } else {
        if (listView) listView.classList.remove('hidden');
        if (gridView) gridView.classList.add('hidden');
        if (listBtn) {
            listBtn.classList.add('bg-gray-100', 'text-gray-700');
            listBtn.classList.remove('text-gray-500');
        }
        if (gridBtn) {
            gridBtn.classList.remove('bg-gray-100', 'text-gray-700');
            gridBtn.classList.add('text-gray-500');
        }
    }

    // Save preference
    localStorage.setItem('fileBrowserViewMode', mode);
}

// ============================================
// Search
// ============================================

function initSearchInput() {
    const searchInput = document.getElementById('file-search');
    const clearBtn = document.getElementById('clear-search');

    if (searchInput) {
        searchInput.addEventListener('input', function() {
            if (clearBtn) {
                clearBtn.classList.toggle('hidden', !this.value);
            }
        });
    }
}

function clearSearch() {
    const searchInput = document.getElementById('file-search');
    const clearBtn = document.getElementById('clear-search');

    if (searchInput) {
        searchInput.value = '';
        htmx.trigger(searchInput, 'keyup');
    }
    if (clearBtn) {
        clearBtn.classList.add('hidden');
    }
}

// ============================================
// Drag and Drop Upload
// ============================================

function initDragAndDrop() {
    const fileBrowser = document.getElementById('main-content');
    const overlay = document.getElementById('drag-drop-overlay');

    if (!fileBrowser) return;

    let dragCounter = 0;

    fileBrowser.addEventListener('dragenter', function(event) {
        event.preventDefault();
        dragCounter++;
        if (overlay) overlay.classList.remove('hidden');
    });

    fileBrowser.addEventListener('dragleave', function(event) {
        event.preventDefault();
        dragCounter--;
        if (dragCounter === 0 && overlay) {
            overlay.classList.add('hidden');
        }
    });

    fileBrowser.addEventListener('dragover', function(event) {
        event.preventDefault();
    });

    fileBrowser.addEventListener('drop', function(event) {
        event.preventDefault();
        dragCounter = 0;
        if (overlay) overlay.classList.add('hidden');

        const files = event.dataTransfer.files;
        if (files.length > 0) {
            handleDroppedFiles(files);
        }
    });
}

function handleDroppedFiles(files) {
    openUploadModal();
    addFilesToUpload(Array.from(files));
}

function handleDragOver(event) {
    event.preventDefault();
    event.currentTarget.classList.add('border-primary', 'bg-blue-50');
}

function handleDragLeave(event) {
    event.preventDefault();
    event.currentTarget.classList.remove('border-primary', 'bg-blue-50');
}

function handleDrop(event) {
    event.preventDefault();
    event.currentTarget.classList.remove('border-primary', 'bg-blue-50');

    const files = event.dataTransfer.files;
    if (files.length > 0) {
        addFilesToUpload(Array.from(files));
    }
}

function handleFileSelect(event) {
    const files = event.target.files;
    if (files.length > 0) {
        addFilesToUpload(Array.from(files));
    }
}

function addFilesToUpload(files) {
    fileBrowserState.pendingUploadFiles = fileBrowserState.pendingUploadFiles.concat(files);
    updateUploadFileList();
}

function updateUploadFileList() {
    const fileList = document.getElementById('upload-file-list');
    const uploadBtn = document.getElementById('upload-btn');

    if (!fileList) return;

    fileList.innerHTML = '';

    fileBrowserState.pendingUploadFiles.forEach((file, index) => {
        const item = document.createElement('div');
        item.className = 'flex items-center justify-between bg-gray-50 rounded-md p-2';
        item.innerHTML = `
            <div class="flex items-center min-w-0 flex-1">
                <svg class="h-5 w-5 text-gray-400 mr-2 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                </svg>
                <span class="text-sm text-gray-700 truncate">${escapeHtml(file.name)}</span>
                <span class="text-xs text-gray-500 ml-2">(${formatFileSize(file.size)})</span>
            </div>
            <button type="button" onclick="removeUploadFile(${index})" class="text-gray-400 hover:text-red-500 ml-2">
                <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
            </button>
        `;
        fileList.appendChild(item);
    });

    fileList.classList.toggle('hidden', fileBrowserState.pendingUploadFiles.length === 0);

    if (uploadBtn) {
        uploadBtn.disabled = fileBrowserState.pendingUploadFiles.length === 0;
    }
}

function removeUploadFile(index) {
    fileBrowserState.pendingUploadFiles.splice(index, 1);
    updateUploadFileList();
}

async function startUpload() {
    if (fileBrowserState.pendingUploadFiles.length === 0 || fileBrowserState.isUploading) {
        return;
    }

    fileBrowserState.isUploading = true;

    const progress = document.getElementById('upload-progress');
    const progressBar = document.getElementById('upload-progress-bar');
    const progressPercent = document.getElementById('upload-percent');
    const uploadBtn = document.getElementById('upload-btn');

    if (progress) progress.classList.remove('hidden');
    if (uploadBtn) uploadBtn.disabled = true;

    const totalFiles = fileBrowserState.pendingUploadFiles.length;
    let uploadedCount = 0;

    for (const file of fileBrowserState.pendingUploadFiles) {
        try {
            const formData = new FormData();
            formData.append('file', file);
            if (fileBrowserState.currentDirectory) {
                formData.append('parent_directory', fileBrowserState.currentDirectory);
            }

            const response = await fetch('/api/files/upload', {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                throw new Error(`Failed to upload ${file.name}`);
            }

            uploadedCount++;
            const percent = Math.round((uploadedCount / totalFiles) * 100);

            if (progressBar) progressBar.style.width = percent + '%';
            if (progressPercent) progressPercent.textContent = percent + '%';

        } catch (error) {
            console.error('Upload error:', error);
            showToast('error', 'Upload failed', error.message);
        }
    }

    fileBrowserState.isUploading = false;

    if (uploadedCount > 0) {
        showToast('success', `${uploadedCount} file(s) uploaded`);
        // Refresh file list
        htmx.trigger('#file-list-wrapper', 'load');
    }

    closeUploadModal();
}

// ============================================
// Keyboard Shortcuts
// ============================================

function initKeyboardShortcuts() {
    document.addEventListener('keydown', function(event) {
        // Don't trigger shortcuts when typing in input fields
        if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
            return;
        }

        const ctrl = event.ctrlKey || event.metaKey;

        // Delete - Delete selected files
        if (event.key === 'Delete') {
            event.preventDefault();
            deleteSelected();
        }

        // Ctrl+A - Select all
        if (ctrl && event.key === 'a') {
            event.preventDefault();
            toggleSelectAll(true);
        }

        // Ctrl+D - Deselect all
        if (ctrl && event.key === 'd') {
            event.preventDefault();
            clearSelection();
        }

        // Ctrl+N - New folder
        if (ctrl && event.key === 'n') {
            event.preventDefault();
            openNewFolderModal();
        }

        // Ctrl+U - Upload
        if (ctrl && event.key === 'u') {
            event.preventDefault();
            openUploadModal();
        }

        // Escape - Close modals/menus
        if (event.key === 'Escape') {
            closeContextMenu();
            closeNewFolderModal();
            closeUploadModal();
            closeRenameModal();
            closeDeleteModal();
            closeMoveModal();
            closeShareModal();
            closeFileDetailsModal();
            hideKeyboardShortcuts();
        }

        // ? - Show keyboard shortcuts
        if (event.key === '?') {
            event.preventDefault();
            showKeyboardShortcuts();
        }
    });
}

function showKeyboardShortcuts() {
    const help = document.getElementById('keyboard-shortcuts-help');
    if (help) help.classList.remove('hidden');
}

function hideKeyboardShortcuts() {
    const help = document.getElementById('keyboard-shortcuts-help');
    if (help) help.classList.add('hidden');
}

// ============================================
// Mobile File Options
// ============================================

function showMobileFileOptions(id, type) {
    // On mobile, show context menu as bottom sheet or dialog
    openFileDetailsModal(id, type);
}

// ============================================
// Expose functions globally
// ============================================

window.toggleFileSelection = toggleFileSelection;
window.toggleSelectAll = toggleSelectAll;
window.clearSelection = clearSelection;
window.showContextMenu = showContextMenu;
window.closeContextMenu = closeContextMenu;
window.contextMenuAction = contextMenuAction;
window.downloadFile = downloadFile;
window.navigateToDirectory = navigateToDirectory;
window.downloadSelected = downloadSelected;
window.deleteSelected = deleteSelected;
window.openNewFolderModal = openNewFolderModal;
window.closeNewFolderModal = closeNewFolderModal;
window.openUploadModal = openUploadModal;
window.closeUploadModal = closeUploadModal;
window.openRenameModal = openRenameModal;
window.closeRenameModal = closeRenameModal;
window.openDeleteModal = openDeleteModal;
window.closeDeleteModal = closeDeleteModal;
window.confirmDelete = confirmDelete;
window.openMoveModal = openMoveModal;
window.closeMoveModal = closeMoveModal;
window.selectMoveTarget = selectMoveTarget;
window.confirmMove = confirmMove;
window.openShareModal = openShareModal;
window.closeShareModal = closeShareModal;
window.toggleSharePassword = toggleSharePassword;
window.openFileDetailsModal = openFileDetailsModal;
window.closeFileDetailsModal = closeFileDetailsModal;
window.openDirectoryFromDetails = openDirectoryFromDetails;
window.shareFromDetails = shareFromDetails;
window.setViewMode = setViewMode;
window.clearSearch = clearSearch;
window.handleDragOver = handleDragOver;
window.handleDragLeave = handleDragLeave;
window.handleDrop = handleDrop;
window.handleFileSelect = handleFileSelect;
window.removeUploadFile = removeUploadFile;
window.startUpload = startUpload;
window.showKeyboardShortcuts = showKeyboardShortcuts;
window.hideKeyboardShortcuts = hideKeyboardShortcuts;
window.showMobileFileOptions = showMobileFileOptions;
window.copyShareLink = copyShareLink;
