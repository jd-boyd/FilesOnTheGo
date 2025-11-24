/**
 * FilesOnTheGo - JavaScript Utilities
 */

// Toast Notification System
let toastId = 0;

function showToast(type, title, message = '') {
    const id = `toast-${++toastId}`;
    const container = document.getElementById('toast-container');

    if (!container) {
        console.error('Toast container not found');
        return;
    }

    const toast = document.createElement('div');
    toast.id = id;
    toast.className = 'max-w-sm w-full bg-white shadow-lg rounded-lg pointer-events-auto ring-1 ring-black ring-opacity-5 overflow-hidden transform transition-all duration-300 toast-enter';

    const iconColor = {
        success: 'text-green-500',
        error: 'text-red-500',
        warning: 'text-yellow-500',
        info: 'text-blue-500'
    }[type] || 'text-gray-500';

    const icons = {
        success: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>',
        error: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path>',
        warning: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>',
        info: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>'
    };

    toast.innerHTML = `
        <div class="p-4">
            <div class="flex items-start">
                <div class="flex-shrink-0">
                    <svg class="h-6 w-6 ${iconColor}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        ${icons[type] || icons.info}
                    </svg>
                </div>
                <div class="ml-3 w-0 flex-1 pt-0.5">
                    <p class="text-sm font-medium text-gray-900">${escapeHtml(title)}</p>
                    ${message ? `<p class="mt-1 text-sm text-gray-500">${escapeHtml(message)}</p>` : ''}
                </div>
                <div class="ml-4 flex-shrink-0 flex">
                    <button type="button" onclick="dismissToast('${id}')" class="inline-flex text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary rounded-md">
                        <span class="sr-only">Close</span>
                        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                        </svg>
                    </button>
                </div>
            </div>
        </div>
    `;

    container.appendChild(toast);

    // Auto-dismiss after 5 seconds
    setTimeout(() => dismissToast(id), 5000);
}

function dismissToast(id) {
    const toast = document.getElementById(id);
    if (toast) {
        toast.classList.remove('toast-enter');
        toast.classList.add('toast-exit');
        setTimeout(() => toast.remove(), 300);
    }
}

// Format File Size
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

// Format Date
function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now - date;

    // Less than 1 minute
    if (diff < 60000) {
        return 'Just now';
    }

    // Less than 1 hour
    if (diff < 3600000) {
        const minutes = Math.floor(diff / 60000);
        return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
    }

    // Less than 1 day
    if (diff < 86400000) {
        const hours = Math.floor(diff / 3600000);
        return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    }

    // Less than 1 week
    if (diff < 604800000) {
        const days = Math.floor(diff / 86400000);
        return `${days} day${days > 1 ? 's' : ''} ago`;
    }

    // Default to formatted date
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
}

// Copy to Clipboard
async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast('success', 'Copied to clipboard');
        return true;
    } catch (err) {
        console.error('Failed to copy:', err);
        showToast('error', 'Failed to copy to clipboard');
        return false;
    }
}

// Escape HTML to prevent XSS
function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// Modal Functions
function openModal(modalId) {
    const event = new CustomEvent('modal-open', { detail: modalId });
    window.dispatchEvent(event);
}

function closeModal(modalId) {
    const event = new CustomEvent('modal-close', { detail: modalId });
    window.dispatchEvent(event);
}

// Note: openUploadModal is implemented in upload.js
// Note: createFolder functionality is in file-browser.js (openNewFolderModal)

// HTMX Event Listeners
document.addEventListener('DOMContentLoaded', function() {
    // Listen for HTMX events
    document.body.addEventListener('htmx:afterSwap', function(event) {
        // Re-initialize any components that were swapped in
        console.log('Content swapped:', event.detail.target);
    });

    document.body.addEventListener('htmx:responseError', function(event) {
        const xhr = event.detail.xhr;
        let message = 'An error occurred';

        try {
            const response = JSON.parse(xhr.responseText);
            message = response.message || response.error || message;
        } catch (e) {
            // Response is not JSON
        }

        showToast('error', 'Error', message);
    });

    document.body.addEventListener('htmx:sendError', function(event) {
        showToast('error', 'Network Error', 'Unable to connect to the server');
    });
});

// Expose functions globally
window.showToast = showToast;
window.dismissToast = dismissToast;
window.formatFileSize = formatFileSize;
window.formatDate = formatDate;
window.copyToClipboard = copyToClipboard;
window.escapeHtml = escapeHtml;
window.openModal = openModal;
window.closeModal = closeModal;
// Note: openUploadModal is exported in upload.js
// Note: createFolder functionality is in file-browser.js (openNewFolderModal)
