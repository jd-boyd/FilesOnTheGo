/**
 * FilesOnTheGo - Upload Module
 * Comprehensive file upload functionality with drag-and-drop,
 * progress tracking, multi-file support, and error handling.
 */

// Upload configuration
const uploadConfig = {
    maxFileSize: 5 * 1024 * 1024 * 1024, // 5 GB
    maxConcurrentUploads: 3,
    chunkSize: 5 * 1024 * 1024, // 5 MB for chunked uploads
    allowedTypes: [], // Empty means all types allowed
    uploadEndpoint: '/api/files/upload',
    retryAttempts: 3,
    retryDelay: 1000
};

// Upload state
const uploadState = {
    files: [],
    uploads: new Map(), // Map of file index to XHR
    isUploading: false,
    uploadedCount: 0,
    totalSize: 0,
    uploadedSize: 0,
    startTime: null,
    currentDirectoryId: null
};

// ============================================
// Initialization
// ============================================

document.addEventListener('DOMContentLoaded', function() {
    initUploadModule();
});

function initUploadModule() {
    // Get current directory from data attribute
    const mainContent = document.getElementById('main-content');
    if (mainContent) {
        uploadState.currentDirectoryId = mainContent.dataset.directoryId || '';
    }

    // Initialize keyboard shortcuts for upload
    initUploadKeyboardShortcuts();

    // Listen for file uploaded event to refresh list
    document.body.addEventListener('fileUploaded', function() {
        refreshFileList();
    });
}

function initUploadKeyboardShortcuts() {
    document.addEventListener('keydown', function(event) {
        // Don't trigger in input fields
        if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
            return;
        }

        const ctrl = event.ctrlKey || event.metaKey;

        // Ctrl+U - Open upload modal
        if (ctrl && event.key === 'u') {
            event.preventDefault();
            openUploadModal();
        }

        // Escape - Close upload modal
        if (event.key === 'Escape') {
            const modal = document.getElementById('upload-modal');
            if (modal && !modal.classList.contains('hidden')) {
                closeUploadModal();
            }
        }
    });
}

// ============================================
// Modal Functions
// ============================================

function openUploadModal() {
    const modal = document.getElementById('upload-modal');
    if (modal) {
        modal.classList.remove('hidden');
        resetUploadState();

        // Update directory ID
        const modalDirId = document.getElementById('modal-directory-id');
        if (modalDirId) {
            modalDirId.value = uploadState.currentDirectoryId || '';
        }

        // Focus the drop zone for accessibility
        const dropZone = document.getElementById('modal-drop-zone');
        if (dropZone) {
            dropZone.focus();
        }
    }
}

function closeUploadModal() {
    const modal = document.getElementById('upload-modal');
    if (modal) {
        // Cancel any ongoing uploads
        if (uploadState.isUploading) {
            cancelAllUploads();
        }

        modal.classList.add('hidden');
        resetUploadState();
    }
}

function resetUploadState() {
    uploadState.files = [];
    uploadState.uploads.clear();
    uploadState.isUploading = false;
    uploadState.uploadedCount = 0;
    uploadState.totalSize = 0;
    uploadState.uploadedSize = 0;
    uploadState.startTime = null;

    // Reset UI elements
    const fileList = document.getElementById('upload-file-list');
    const progress = document.getElementById('upload-progress');
    const uploadBtn = document.getElementById('upload-btn');
    const fileInput = document.getElementById('modal-file-input');
    const errors = document.getElementById('upload-errors');

    if (fileList) {
        fileList.innerHTML = '';
        fileList.classList.add('hidden');
    }
    if (progress) {
        progress.classList.add('hidden');
    }
    if (uploadBtn) {
        uploadBtn.disabled = true;
    }
    if (fileInput) {
        fileInput.value = '';
    }
    if (errors) {
        errors.innerHTML = '';
    }

    // Reset progress bar
    const progressBar = document.getElementById('upload-progress-bar');
    const progressPercent = document.getElementById('upload-percent');
    if (progressBar) {
        progressBar.style.width = '0%';
    }
    if (progressPercent) {
        progressPercent.textContent = '0%';
    }
}

// ============================================
// Drag and Drop Handlers
// ============================================

function handleDragOver(event) {
    event.preventDefault();
    event.stopPropagation();
    const target = event.currentTarget;
    target.classList.add('dragover', 'border-primary', 'bg-blue-50');
}

function handleDragLeave(event) {
    event.preventDefault();
    event.stopPropagation();
    const target = event.currentTarget;
    target.classList.remove('dragover', 'border-primary', 'bg-blue-50');
}

function handleDrop(event) {
    event.preventDefault();
    event.stopPropagation();
    const target = event.currentTarget;
    target.classList.remove('dragover', 'border-primary', 'bg-blue-50');

    const files = event.dataTransfer.files;
    if (files.length > 0) {
        addFilesToQueue(Array.from(files));
    }
}

// ============================================
// File Selection and Validation
// ============================================

function handleFileSelect(event) {
    const files = event.target.files;
    if (files.length > 0) {
        addFilesToQueue(Array.from(files));
    }
}

function addFilesToQueue(files) {
    const errors = [];

    files.forEach(file => {
        const validation = validateFile(file);
        if (validation.valid) {
            uploadState.files.push({
                file: file,
                status: 'pending',
                progress: 0,
                error: null,
                preview: null
            });
            uploadState.totalSize += file.size;

            // Generate preview for images
            if (file.type.startsWith('image/')) {
                generateImagePreview(file, uploadState.files.length - 1);
            }
        } else {
            errors.push({ name: file.name, error: validation.error });
        }
    });

    // Show validation errors
    if (errors.length > 0) {
        showValidationErrors(errors);
    }

    updateFileListUI();
}

function validateFile(file) {
    // Check file size
    if (file.size > uploadConfig.maxFileSize) {
        return {
            valid: false,
            error: `File is too large. Maximum size is ${formatFileSize(uploadConfig.maxFileSize)}`
        };
    }

    // Check file size is not 0
    if (file.size === 0) {
        return {
            valid: false,
            error: 'File is empty'
        };
    }

    // Check file type if restrictions exist
    if (uploadConfig.allowedTypes.length > 0) {
        const fileType = file.type || getMimeTypeFromExtension(file.name);
        const allowed = uploadConfig.allowedTypes.some(type => {
            if (type.endsWith('/*')) {
                return fileType.startsWith(type.slice(0, -1));
            }
            return fileType === type;
        });

        if (!allowed) {
            return {
                valid: false,
                error: 'File type not allowed'
            };
        }
    }

    // Check for duplicate files
    const isDuplicate = uploadState.files.some(
        existing => existing.file.name === file.name && existing.file.size === file.size
    );

    if (isDuplicate) {
        return {
            valid: false,
            error: 'File already in queue'
        };
    }

    return { valid: true };
}

function getMimeTypeFromExtension(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    const mimeTypes = {
        'pdf': 'application/pdf',
        'doc': 'application/msword',
        'docx': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'xls': 'application/vnd.ms-excel',
        'xlsx': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'ppt': 'application/vnd.ms-powerpoint',
        'pptx': 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
        'txt': 'text/plain',
        'csv': 'text/csv',
        'html': 'text/html',
        'css': 'text/css',
        'js': 'application/javascript',
        'json': 'application/json',
        'xml': 'application/xml',
        'zip': 'application/zip',
        'tar': 'application/x-tar',
        'gz': 'application/gzip',
        'jpg': 'image/jpeg',
        'jpeg': 'image/jpeg',
        'png': 'image/png',
        'gif': 'image/gif',
        'webp': 'image/webp',
        'svg': 'image/svg+xml',
        'mp3': 'audio/mpeg',
        'wav': 'audio/wav',
        'mp4': 'video/mp4',
        'webm': 'video/webm',
        'avi': 'video/x-msvideo'
    };
    return mimeTypes[ext] || 'application/octet-stream';
}

function generateImagePreview(file, index) {
    const reader = new FileReader();
    reader.onload = function(e) {
        if (uploadState.files[index]) {
            uploadState.files[index].preview = e.target.result;
            updateFileItemPreview(index);
        }
    };
    reader.readAsDataURL(file);
}

function showValidationErrors(errors) {
    const errorsContainer = document.getElementById('upload-errors');
    if (!errorsContainer) return;

    errors.forEach(({ name, error }) => {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'bg-red-50 border border-red-200 text-red-800 rounded-md p-3 mb-2 flex items-start';
        errorDiv.innerHTML = `
            <svg class="w-5 h-5 text-red-500 mr-2 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <div class="flex-1">
                <p class="text-sm font-medium">${escapeHtml(name)}</p>
                <p class="text-xs mt-0.5">${escapeHtml(error)}</p>
            </div>
            <button type="button" class="text-red-400 hover:text-red-600" onclick="this.parentElement.remove()">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
            </button>
        `;
        errorsContainer.appendChild(errorDiv);

        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            if (errorDiv.parentElement) {
                errorDiv.remove();
            }
        }, 5000);
    });
}

// ============================================
// File List UI
// ============================================

function updateFileListUI() {
    const fileList = document.getElementById('upload-file-list');
    const uploadBtn = document.getElementById('upload-btn');

    if (!fileList) return;

    // Clear existing list
    fileList.innerHTML = '';

    if (uploadState.files.length === 0) {
        fileList.classList.add('hidden');
        if (uploadBtn) uploadBtn.disabled = true;
        return;
    }

    fileList.classList.remove('hidden');
    if (uploadBtn) uploadBtn.disabled = false;

    // Create file items
    uploadState.files.forEach((fileData, index) => {
        const item = createFileListItem(fileData, index);
        fileList.appendChild(item);
    });
}

function createFileListItem(fileData, index) {
    const { file, status, progress, error, preview } = fileData;

    const item = document.createElement('div');
    item.className = 'upload-file-item flex items-center justify-between bg-gray-50 rounded-md p-3';
    item.dataset.fileIndex = index;

    // Determine icon based on file type
    const iconSvg = getFileTypeIcon(file.type);

    // Create item HTML
    item.innerHTML = `
        <div class="flex items-center min-w-0 flex-1">
            ${preview ? `
                <img src="${preview}" alt="" class="w-10 h-10 object-cover rounded mr-3 flex-shrink-0">
            ` : `
                <div class="w-10 h-10 bg-gray-200 rounded flex items-center justify-center mr-3 flex-shrink-0">
                    ${iconSvg}
                </div>
            `}
            <div class="min-w-0 flex-1">
                <p class="text-sm font-medium text-gray-900 truncate" title="${escapeHtml(file.name)}">
                    ${escapeHtml(file.name)}
                </p>
                <div class="flex items-center space-x-2">
                    <span class="text-xs text-gray-500">${formatFileSize(file.size)}</span>
                    ${status === 'uploading' ? `
                        <div class="flex-1 max-w-32">
                            <div class="bg-gray-200 rounded-full h-1.5">
                                <div class="progress-bar bg-primary h-1.5 rounded-full transition-all duration-200" style="width: ${progress}%"></div>
                            </div>
                        </div>
                        <span class="text-xs text-primary font-medium">${progress}%</span>
                    ` : ''}
                    ${status === 'completed' ? `
                        <span class="text-xs text-green-600 flex items-center">
                            <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                            </svg>
                            Done
                        </span>
                    ` : ''}
                    ${status === 'error' ? `
                        <span class="text-xs text-red-600 flex items-center" title="${escapeHtml(error || 'Upload failed')}">
                            <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                            </svg>
                            Failed
                        </span>
                    ` : ''}
                </div>
            </div>
        </div>
        ${status === 'pending' || status === 'uploading' ? `
            <button type="button"
                    class="text-gray-400 hover:text-red-500 ml-3 flex-shrink-0"
                    onclick="removeUploadFile(${index})"
                    aria-label="Remove file">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
            </button>
        ` : ''}
    `;

    return item;
}

function getFileTypeIcon(mimeType) {
    // Document types
    if (mimeType.includes('pdf')) {
        return `<svg class="h-5 w-5 text-red-500" fill="currentColor" viewBox="0 0 20 20"><path d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z"/></svg>`;
    }
    if (mimeType.includes('word') || mimeType.includes('document')) {
        return `<svg class="h-5 w-5 text-blue-600" fill="currentColor" viewBox="0 0 20 20"><path d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z"/></svg>`;
    }
    if (mimeType.includes('sheet') || mimeType.includes('excel')) {
        return `<svg class="h-5 w-5 text-green-600" fill="currentColor" viewBox="0 0 20 20"><path d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z"/></svg>`;
    }

    // Media types
    if (mimeType.startsWith('image/')) {
        return `<svg class="h-5 w-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>`;
    }
    if (mimeType.startsWith('video/')) {
        return `<svg class="h-5 w-5 text-pink-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>`;
    }
    if (mimeType.startsWith('audio/')) {
        return `<svg class="h-5 w-5 text-yellow-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3"></path></svg>`;
    }

    // Archive types
    if (mimeType.includes('zip') || mimeType.includes('tar') || mimeType.includes('gzip') || mimeType.includes('rar')) {
        return `<svg class="h-5 w-5 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"></path></svg>`;
    }

    // Code/text types
    if (mimeType.includes('text') || mimeType.includes('javascript') || mimeType.includes('json')) {
        return `<svg class="h-5 w-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path></svg>`;
    }

    // Default file icon
    return `<svg class="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>`;
}

function updateFileItemPreview(index) {
    const item = document.querySelector(`.upload-file-item[data-file-index="${index}"]`);
    if (!item || !uploadState.files[index]) return;

    const fileData = uploadState.files[index];
    if (fileData.preview) {
        const iconDiv = item.querySelector('.w-10.h-10.bg-gray-200');
        if (iconDiv) {
            const img = document.createElement('img');
            img.src = fileData.preview;
            img.alt = '';
            img.className = 'w-10 h-10 object-cover rounded mr-3 flex-shrink-0';
            iconDiv.replaceWith(img);
        }
    }
}

function updateFileItemProgress(index, progress) {
    const item = document.querySelector(`.upload-file-item[data-file-index="${index}"]`);
    if (!item) return;

    const progressBar = item.querySelector('.progress-bar');
    if (progressBar) {
        progressBar.style.width = `${progress}%`;
    }

    const progressText = item.querySelector('.text-xs.text-primary');
    if (progressText) {
        progressText.textContent = `${progress}%`;
    }
}

// ============================================
// Upload Functions
// ============================================

function removeUploadFile(index) {
    // Cancel upload if in progress
    const xhr = uploadState.uploads.get(index);
    if (xhr) {
        xhr.abort();
        uploadState.uploads.delete(index);
    }

    // Remove file from state
    const file = uploadState.files[index];
    if (file) {
        uploadState.totalSize -= file.file.size;
        uploadState.files.splice(index, 1);
    }

    updateFileListUI();
}

function clearUploadQueue() {
    // Cancel all uploads
    cancelAllUploads();

    // Clear state
    uploadState.files = [];
    uploadState.totalSize = 0;

    updateFileListUI();
}

async function startUpload() {
    if (uploadState.files.length === 0 || uploadState.isUploading) {
        return;
    }

    uploadState.isUploading = true;
    uploadState.uploadedCount = 0;
    uploadState.uploadedSize = 0;
    uploadState.startTime = Date.now();

    // Show progress section
    const progress = document.getElementById('upload-progress');
    const uploadBtn = document.getElementById('upload-btn');
    const totalSpan = document.getElementById('upload-total');

    if (progress) progress.classList.remove('hidden');
    if (uploadBtn) uploadBtn.disabled = true;
    if (totalSpan) totalSpan.textContent = uploadState.files.length;

    // Get directory ID
    const directoryId = document.getElementById('modal-directory-id')?.value || '';

    // Upload files sequentially (or in parallel with limit)
    const pendingFiles = uploadState.files.filter(f => f.status === 'pending');

    for (let i = 0; i < pendingFiles.length; i++) {
        const fileIndex = uploadState.files.indexOf(pendingFiles[i]);

        try {
            await uploadFile(fileIndex, directoryId);
            uploadState.uploadedCount++;
        } catch (error) {
            console.error('Upload error:', error);
        }

        // Update overall progress
        updateOverallProgress();
    }

    // Upload complete
    uploadState.isUploading = false;

    const successCount = uploadState.files.filter(f => f.status === 'completed').length;
    const errorCount = uploadState.files.filter(f => f.status === 'error').length;

    if (successCount > 0) {
        showToast('success', `${successCount} file(s) uploaded successfully`);
        refreshFileList();
    }

    if (errorCount > 0) {
        showToast('error', `${errorCount} file(s) failed to upload`);
    }

    // Close modal after short delay if all successful
    if (errorCount === 0 && successCount > 0) {
        setTimeout(() => {
            closeUploadModal();
        }, 1000);
    }
}

function uploadFile(index, directoryId) {
    return new Promise((resolve, reject) => {
        const fileData = uploadState.files[index];
        if (!fileData) {
            reject(new Error('File not found'));
            return;
        }

        fileData.status = 'uploading';
        fileData.progress = 0;
        updateFileListUI();

        const formData = new FormData();
        formData.append('file', fileData.file);
        formData.append('directory_id', directoryId || '');

        const xhr = new XMLHttpRequest();
        uploadState.uploads.set(index, xhr);

        xhr.upload.addEventListener('progress', function(event) {
            if (event.lengthComputable) {
                const percent = Math.round((event.loaded / event.total) * 100);
                fileData.progress = percent;
                updateFileItemProgress(index, percent);
                updateOverallProgress();
            }
        });

        xhr.addEventListener('load', function() {
            uploadState.uploads.delete(index);

            if (xhr.status >= 200 && xhr.status < 300) {
                fileData.status = 'completed';
                fileData.progress = 100;
                uploadState.uploadedSize += fileData.file.size;
                updateFileListUI();
                resolve();
            } else {
                let errorMessage = 'Upload failed';
                try {
                    const response = JSON.parse(xhr.responseText);
                    errorMessage = response.error?.message || response.message || errorMessage;
                } catch (e) {
                    // Response is not JSON
                }
                fileData.status = 'error';
                fileData.error = errorMessage;
                updateFileListUI();
                reject(new Error(errorMessage));
            }
        });

        xhr.addEventListener('error', function() {
            uploadState.uploads.delete(index);
            fileData.status = 'error';
            fileData.error = 'Network error occurred';
            updateFileListUI();
            reject(new Error('Network error'));
        });

        xhr.addEventListener('abort', function() {
            uploadState.uploads.delete(index);
            fileData.status = 'pending';
            updateFileListUI();
            reject(new Error('Upload cancelled'));
        });

        xhr.open('POST', uploadConfig.uploadEndpoint);

        // Set HTMX header for proper response handling
        xhr.setRequestHeader('HX-Request', 'true');

        xhr.send(formData);
    });
}

function updateOverallProgress() {
    const current = document.getElementById('upload-current');
    const percent = document.getElementById('upload-percent');
    const progressBar = document.getElementById('upload-progress-bar');
    const status = document.getElementById('upload-status');

    // Calculate overall progress
    let totalProgress = 0;
    uploadState.files.forEach(f => {
        if (f.status === 'completed') {
            totalProgress += 100;
        } else if (f.status === 'uploading') {
            totalProgress += f.progress;
        }
    });

    const overallPercent = Math.round(totalProgress / uploadState.files.length);

    if (current) current.textContent = uploadState.uploadedCount;
    if (percent) percent.textContent = `${overallPercent}%`;
    if (progressBar) {
        progressBar.style.width = `${overallPercent}%`;
        progressBar.setAttribute('aria-valuenow', overallPercent);
    }

    // Calculate upload speed and time remaining
    if (status && uploadState.startTime) {
        const elapsed = (Date.now() - uploadState.startTime) / 1000;
        const speed = uploadState.uploadedSize / elapsed;
        const remaining = (uploadState.totalSize - uploadState.uploadedSize) / speed;

        if (isFinite(speed) && speed > 0) {
            status.textContent = `${formatFileSize(Math.round(speed))}/s`;
            if (isFinite(remaining) && remaining > 0) {
                status.textContent += ` - ${formatTimeRemaining(remaining)} remaining`;
            }
        } else {
            status.textContent = 'Uploading...';
        }
    }
}

function cancelUpload(button) {
    const item = button.closest('.upload-file-item');
    if (!item) return;

    const index = parseInt(item.dataset.fileIndex);
    const xhr = uploadState.uploads.get(index);

    if (xhr) {
        xhr.abort();
    }
}

function cancelAllUploads() {
    uploadState.uploads.forEach((xhr, index) => {
        xhr.abort();
    });
    uploadState.uploads.clear();
    uploadState.isUploading = false;
}

// ============================================
// Helper Functions
// ============================================

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

function formatTimeRemaining(seconds) {
    if (seconds < 60) {
        return `${Math.round(seconds)}s`;
    } else if (seconds < 3600) {
        return `${Math.round(seconds / 60)}m`;
    } else {
        return `${Math.round(seconds / 3600)}h`;
    }
}

function escapeHtml(unsafe) {
    if (!unsafe) return '';
    return unsafe
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function refreshFileList() {
    // Trigger HTMX to refresh the file list
    const fileListWrapper = document.getElementById('file-list-wrapper');
    if (fileListWrapper) {
        htmx.trigger(fileListWrapper, 'load');
    }
}

// ============================================
// Expose Functions Globally
// ============================================

window.openUploadModal = openUploadModal;
window.closeUploadModal = closeUploadModal;
window.handleDragOver = handleDragOver;
window.handleDragLeave = handleDragLeave;
window.handleDrop = handleDrop;
window.handleFileSelect = handleFileSelect;
window.removeUploadFile = removeUploadFile;
window.clearUploadQueue = clearUploadQueue;
window.startUpload = startUpload;
window.cancelUpload = cancelUpload;
window.cancelAllUploads = cancelAllUploads;
window.removeFromQueue = removeUploadFile;
