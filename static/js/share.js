/**
 * FilesOnTheGo - Share Management JavaScript Utilities
 * Handles share creation, management, and UI interactions
 */

// ============================================
// State Management
// ============================================

const shareState = {
    currentResourceId: null,
    currentResourceType: null,
    currentResourceName: null,
    selectedShares: new Set(),
    pendingRevokeShareId: null,
    pendingRevokeBulk: false,
    existingShares: [],
    baseUrl: window.location.origin
};

// ============================================
// Share Modal Functions
// ============================================

/**
 * Opens the share modal for a file or directory
 * @param {string} id - Resource ID
 * @param {string} type - Resource type ('file' or 'directory')
 * @param {string} name - Resource name (optional, will be fetched if not provided)
 */
function openShareModal(id, type, name) {
    shareState.currentResourceId = id;
    shareState.currentResourceType = type;
    shareState.currentResourceName = name || 'item';

    const modal = document.getElementById('share-modal');
    const itemNameSpan = document.getElementById('share-item-name');
    const resourceTypeDisplay = document.getElementById('share-resource-type-display');
    const resourceTypeField = document.getElementById('share-resource-type');
    const resourceIdField = document.getElementById('share-resource-id');
    const resultDiv = document.getElementById('share-result');

    if (!modal) return;

    // Update modal content
    if (itemNameSpan) itemNameSpan.textContent = name || 'item';
    if (resourceTypeDisplay) resourceTypeDisplay.textContent = type === 'directory' ? 'Folder' : 'File';
    if (resourceTypeField) resourceTypeField.value = type;
    if (resourceIdField) resourceIdField.value = id;
    if (resultDiv) {
        resultDiv.innerHTML = '';
        resultDiv.classList.add('hidden');
    }

    // Reset form to defaults
    resetShareForm();

    // Switch to create tab
    switchShareTab('create');

    // Load existing shares for this resource
    loadExistingShares(id, type);

    // Show modal
    modal.classList.remove('hidden');
}

/**
 * Closes the share modal
 */
function closeShareModal() {
    const modal = document.getElementById('share-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
    shareState.currentResourceId = null;
    shareState.currentResourceType = null;
    shareState.currentResourceName = null;
}

/**
 * Resets the share form to default values
 */
function resetShareForm() {
    const form = document.getElementById('create-share-form');
    if (form) form.reset();

    // Reset permission selection
    const readRadio = document.querySelector('input[name="permission_type"][value="read"]');
    if (readRadio) readRadio.checked = true;
    updatePermissionSelection();

    // Hide password and expiration containers
    document.getElementById('share-password-container')?.classList.add('hidden');
    document.getElementById('share-expiration-container')?.classList.add('hidden');
    document.getElementById('share-password-toggle').checked = false;
    document.getElementById('share-expiration-toggle').checked = false;
}

/**
 * Switches between create and existing tabs
 * @param {string} tab - Tab to switch to ('create' or 'existing')
 */
function switchShareTab(tab) {
    // Update tab buttons
    document.querySelectorAll('.share-tab').forEach(btn => {
        btn.classList.remove('border-primary', 'text-primary');
        btn.classList.add('border-transparent', 'text-gray-500');
    });

    const activeTab = document.getElementById(`share-tab-${tab}`);
    if (activeTab) {
        activeTab.classList.remove('border-transparent', 'text-gray-500');
        activeTab.classList.add('border-primary', 'text-primary');
    }

    // Update tab content
    document.querySelectorAll('.share-tab-content').forEach(content => {
        content.classList.add('hidden');
    });

    const activeContent = document.getElementById(`share-content-${tab}`);
    if (activeContent) {
        activeContent.classList.remove('hidden');
    }
}

// ============================================
// Permission Selection
// ============================================

/**
 * Updates the visual state of permission selection
 */
function updatePermissionSelection() {
    document.querySelectorAll('.permission-option').forEach(option => {
        const radio = option.querySelector('input[type="radio"]');
        if (radio.checked) {
            option.classList.add('border-primary', 'bg-blue-50');
            option.classList.remove('border-gray-200');
        } else {
            option.classList.remove('border-primary', 'bg-blue-50');
            option.classList.add('border-gray-200');
        }
    });
}

// ============================================
// Password Protection
// ============================================

/**
 * Toggles password input visibility
 */
function toggleSharePassword() {
    const toggle = document.getElementById('share-password-toggle');
    const container = document.getElementById('share-password-container');
    const input = document.getElementById('share-password');

    if (container && toggle) {
        if (toggle.checked) {
            container.classList.remove('hidden');
            if (input) input.focus();
        } else {
            container.classList.add('hidden');
            if (input) input.value = '';
        }
    }
}

// ============================================
// Expiration Date
// ============================================

/**
 * Toggles expiration input visibility
 */
function toggleShareExpiration() {
    const toggle = document.getElementById('share-expiration-toggle');
    const container = document.getElementById('share-expiration-container');

    if (container && toggle) {
        if (toggle.checked) {
            container.classList.remove('hidden');
            // Set default to 7 days from now
            setShareExpiration(7);
        } else {
            container.classList.add('hidden');
            const input = document.getElementById('share-expiration');
            if (input) input.value = '';
        }
    }
}

/**
 * Sets the expiration date to a number of days from now
 * @param {number} days - Number of days from now
 */
function setShareExpiration(days) {
    const input = document.getElementById('share-expiration');
    if (input) {
        const date = new Date();
        date.setDate(date.getDate() + days);
        // Format for datetime-local input (YYYY-MM-DDTHH:mm)
        const formatted = date.toISOString().slice(0, 16);
        input.value = formatted;
    }

    // Ensure toggle is checked
    const toggle = document.getElementById('share-expiration-toggle');
    if (toggle && !toggle.checked) {
        toggle.checked = true;
        document.getElementById('share-expiration-container')?.classList.remove('hidden');
    }
}

// ============================================
// Share Creation
// ============================================

/**
 * Submits the share creation form
 * @param {Event} event - Form submit event
 * @returns {boolean}
 */
async function submitShareForm(event) {
    event.preventDefault();

    const form = document.getElementById('create-share-form');
    const submitBtn = document.getElementById('create-share-btn');
    const resultDiv = document.getElementById('share-result');

    if (!form) return false;

    // Disable submit button
    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.innerHTML = `
            <svg class="animate-spin h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            Creating...
        `;
    }

    try {
        // Collect form data
        const formData = new FormData(form);
        const data = {
            resource_type: formData.get('resource_type'),
            resource_id: formData.get('resource_id'),
            permission_type: formData.get('permission_type')
        };

        // Add password if enabled
        const password = formData.get('password');
        if (password) {
            data.password = password;
        }

        // Add expiration if enabled
        const expiresAt = formData.get('expires_at');
        if (expiresAt) {
            data.expires_at = new Date(expiresAt).toISOString();
        }

        // Send request
        const response = await fetch('/api/shares', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.error || 'Failed to create share');
        }

        // Show success result
        if (resultDiv) {
            resultDiv.classList.remove('hidden');
            resultDiv.innerHTML = createShareLinkDisplayHTML(result.url, result.share);
        }

        // Reload existing shares
        loadExistingShares(data.resource_id, data.resource_type);

        showToast('success', 'Share link created successfully!');

    } catch (error) {
        console.error('Share creation error:', error);
        showToast('error', 'Failed to create share', error.message);
    } finally {
        // Re-enable submit button
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.innerHTML = `
                <svg class="h-5 w-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"></path>
                </svg>
                Generate Share Link
            `;
        }
    }

    return false;
}

/**
 * Creates the HTML for share link display
 * @param {string} url - Share URL
 * @param {Object} share - Share object
 * @returns {string} HTML string
 */
function createShareLinkDisplayHTML(url, share) {
    const permissionText = {
        'read': 'Read-only',
        'read_upload': 'Read & Upload',
        'upload_only': 'Upload-only'
    }[share.permission_type] || share.permission_type;

    const permissionClass = {
        'read': 'bg-blue-100 text-blue-800',
        'read_upload': 'bg-green-100 text-green-800',
        'upload_only': 'bg-yellow-100 text-yellow-800'
    }[share.permission_type] || 'bg-gray-100 text-gray-800';

    let expiresText = '';
    if (share.expires_at) {
        const expiresDate = new Date(share.expires_at);
        expiresText = `
            <div class="flex items-center">
                <span class="text-gray-500 w-24">Expires:</span>
                <span>${expiresDate.toLocaleDateString()} ${expiresDate.toLocaleTimeString()}</span>
            </div>
        `;
    }

    let passwordText = '';
    if (share.is_password_protected) {
        passwordText = `
            <div class="flex items-center">
                <span class="text-gray-500 w-24">Password:</span>
                <span class="flex items-center text-amber-700">
                    <svg class="h-3.5 w-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
                    </svg>
                    Protected
                </span>
            </div>
        `;
    }

    return `
        <div class="bg-green-50 border border-green-200 rounded-lg p-4" data-share-url="${escapeHtml(url)}">
            <div class="flex items-center mb-3">
                <div class="flex-shrink-0">
                    <svg class="h-5 w-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
                    </svg>
                </div>
                <p class="ml-2 text-sm font-medium text-green-800">
                    Share link created successfully!
                </p>
            </div>

            <div class="flex items-center space-x-2">
                <div class="flex-1 min-w-0">
                    <input type="text"
                           id="share-url-input"
                           value="${escapeHtml(url)}"
                           readonly
                           class="block w-full bg-white border border-gray-300 rounded-md py-2 px-3 text-sm font-mono text-gray-700 focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary cursor-text"
                           onclick="this.select()">
                </div>
                <button type="button"
                        onclick="copyShareLinkFromInput()"
                        class="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary">
                    <svg class="h-4 w-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
                    </svg>
                    Copy
                </button>
            </div>

            <div class="mt-4 text-xs text-gray-600 space-y-1">
                <div class="flex items-center">
                    <span class="text-gray-500 w-24">Permission:</span>
                    <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${permissionClass}">
                        ${permissionText}
                    </span>
                </div>
                ${expiresText}
                ${passwordText}
            </div>

            <div class="mt-4 bg-white rounded-md p-3 border border-green-100">
                <p class="text-xs text-gray-600">
                    <svg class="inline h-4 w-4 text-gray-400 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                    </svg>
                    Share this link with anyone you want to give access.
                    ${share.is_password_protected ? "Don't forget to share the password separately!" : ''}
                </p>
            </div>
        </div>
    `;
}

// ============================================
// Existing Shares
// ============================================

/**
 * Loads existing shares for a resource
 * @param {string} resourceId - Resource ID
 * @param {string} resourceType - Resource type
 */
async function loadExistingShares(resourceId, resourceType) {
    const loadingEl = document.getElementById('existing-shares-loading');
    const emptyEl = document.getElementById('existing-shares-empty');
    const containerEl = document.getElementById('existing-shares-container');
    const countBadge = document.getElementById('existing-shares-count');

    if (loadingEl) loadingEl.classList.remove('hidden');
    if (emptyEl) emptyEl.classList.add('hidden');
    if (containerEl) containerEl.innerHTML = '';

    try {
        const response = await fetch(`/api/shares?resource_type=${resourceType}&resource_id=${resourceId}`);
        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.error || 'Failed to load shares');
        }

        shareState.existingShares = result.shares || [];

        // Update count badge
        if (countBadge) {
            if (shareState.existingShares.length > 0) {
                countBadge.textContent = shareState.existingShares.length;
                countBadge.classList.remove('hidden');
            } else {
                countBadge.classList.add('hidden');
            }
        }

        // Render shares
        if (shareState.existingShares.length === 0) {
            if (emptyEl) emptyEl.classList.remove('hidden');
        } else {
            if (containerEl) {
                containerEl.innerHTML = shareState.existingShares.map(share =>
                    createShareListItemHTML(share, true)
                ).join('');
            }
        }

    } catch (error) {
        console.error('Failed to load existing shares:', error);
        showToast('error', 'Failed to load existing shares');
    } finally {
        if (loadingEl) loadingEl.classList.add('hidden');
    }
}

/**
 * Creates HTML for a share list item
 * @param {Object} share - Share object
 * @param {boolean} compact - Whether to use compact layout
 * @returns {string} HTML string
 */
function createShareListItemHTML(share, compact = false) {
    const shareUrl = `${shareState.baseUrl}/share/${share.share_token}`;
    const truncatedUrl = truncateUrl(shareUrl, 35);
    const isExpired = share.is_expired;

    const permissionBadgeClass = {
        'read': 'bg-blue-100 text-blue-700',
        'read_upload': 'bg-green-100 text-green-700',
        'upload_only': 'bg-yellow-100 text-yellow-700'
    }[share.permission_type] || 'bg-gray-100 text-gray-700';

    const permissionText = {
        'read': 'Read',
        'read_upload': 'Read+Upload',
        'upload_only': 'Upload'
    }[share.permission_type] || share.permission_type;

    let expiresText = '';
    if (share.expires_at) {
        const expiresDate = new Date(share.expires_at);
        expiresText = isExpired ? 'Expired' : formatRelativeDate(expiresDate);
    }

    return `
        <div class="share-item border border-gray-200 rounded-md p-3 hover:bg-gray-50 transition-colors ${isExpired ? 'opacity-60' : ''}"
             data-share-id="${share.id}"
             data-share-url="${escapeHtml(shareUrl)}">
            <div class="flex items-center justify-between">
                <div class="flex-1 min-w-0 mr-4">
                    <div class="flex items-center space-x-2 mb-1.5">
                        <span class="text-xs font-mono bg-gray-100 px-2 py-0.5 rounded truncate max-w-[200px]"
                              title="${escapeHtml(shareUrl)}">
                            ${escapeHtml(truncatedUrl)}
                        </span>
                        <button type="button"
                                onclick="copyShareLink('${escapeHtml(shareUrl)}')"
                                class="text-gray-400 hover:text-primary"
                                title="Copy">
                            <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
                            </svg>
                        </button>
                    </div>
                    <div class="flex items-center gap-2 text-xs text-gray-500">
                        <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium uppercase ${permissionBadgeClass}">
                            ${permissionText}
                        </span>
                        ${share.is_password_protected ? `
                            <svg class="w-3 h-3 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" title="Password protected">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
                            </svg>
                        ` : ''}
                        ${expiresText ? `<span class="${isExpired ? 'text-red-500' : ''}">${expiresText}</span>` : ''}
                        <span>${share.access_count} views</span>
                    </div>
                </div>
                <button type="button"
                        onclick="revokeShareFromModal('${share.id}')"
                        class="text-red-600 hover:text-red-800 p-1"
                        title="Revoke">
                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                    </svg>
                </button>
            </div>
        </div>
    `;
}

// ============================================
// Share Revocation
// ============================================

/**
 * Opens revoke confirmation for a share from the modal
 * @param {string} shareId - Share ID
 */
function revokeShareFromModal(shareId) {
    revokeShare(shareId);
}

/**
 * Opens revoke confirmation dialog
 * @param {string} shareId - Share ID
 */
function revokeShare(shareId) {
    shareState.pendingRevokeShareId = shareId;
    shareState.pendingRevokeBulk = false;

    const modal = document.getElementById('revoke-confirm-modal');
    const message = document.getElementById('revoke-confirm-message');

    if (message) {
        message.textContent = 'Are you sure you want to revoke this share? The link will immediately stop working.';
    }

    if (modal) {
        modal.classList.remove('hidden');
    }
}

/**
 * Confirms and executes share revocation
 */
async function confirmRevokeShare() {
    const shareId = shareState.pendingRevokeShareId;
    if (!shareId) return;

    closeRevokeConfirmModal();

    try {
        const response = await fetch(`/api/shares/${shareId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const result = await response.json();
            throw new Error(result.error || 'Failed to revoke share');
        }

        // Remove share item from DOM
        const shareItem = document.querySelector(`[data-share-id="${shareId}"]`);
        if (shareItem) {
            shareItem.style.opacity = '0';
            shareItem.style.transform = 'translateX(-10px)';
            setTimeout(() => shareItem.remove(), 300);
        }

        // Reload existing shares if in modal
        if (shareState.currentResourceId) {
            loadExistingShares(shareState.currentResourceId, shareState.currentResourceType);
        }

        showToast('success', 'Share revoked successfully');

    } catch (error) {
        console.error('Revoke error:', error);
        showToast('error', 'Failed to revoke share', error.message);
    }

    shareState.pendingRevokeShareId = null;
}

/**
 * Closes the revoke confirmation modal
 */
function closeRevokeConfirmModal() {
    const modal = document.getElementById('revoke-confirm-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
}

// ============================================
// Copy Functions
// ============================================

/**
 * Copies a share link to clipboard
 * @param {string} url - URL to copy
 */
async function copyShareLink(url) {
    try {
        await navigator.clipboard.writeText(url);
        showToast('success', 'Link copied to clipboard');
    } catch (error) {
        console.error('Copy failed:', error);
        showToast('error', 'Failed to copy link');
    }
}

/**
 * Copies the share link from the URL input field
 */
async function copyShareLinkFromInput() {
    const input = document.getElementById('share-url-input');
    if (input) {
        input.select();
        await copyShareLink(input.value);
    }
}

// ============================================
// QR Code
// ============================================

/**
 * Shows QR code modal for a share URL
 * @param {string} url - Share URL
 */
function showShareQRCode(url) {
    const modal = document.getElementById('qr-code-modal');
    const container = document.getElementById('qr-code-display');

    if (!modal || !container) return;

    // Generate QR code using a simple text-based approach
    // In production, you'd use a library like qrcode.js
    container.innerHTML = `
        <div class="text-center">
            <p class="text-sm text-gray-500 mb-2">QR Code generation requires qrcode.js library</p>
            <div class="text-xs text-gray-400 break-all p-2 bg-gray-100 rounded">${escapeHtml(url)}</div>
        </div>
    `;

    modal.classList.remove('hidden');
}

/**
 * Closes the QR code modal
 */
function closeQRCodeModal() {
    const modal = document.getElementById('qr-code-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
}

// ============================================
// Access Logs
// ============================================

/**
 * Views access logs for a share
 * @param {string} shareId - Share ID
 */
async function viewShareAccessLogs(shareId) {
    const modal = document.getElementById('access-logs-modal');
    const content = document.getElementById('access-logs-content');

    if (!modal || !content) return;

    modal.classList.remove('hidden');
    content.innerHTML = `
        <div class="flex justify-center py-8">
            <svg class="animate-spin h-6 w-6 text-gray-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
        </div>
    `;

    try {
        const response = await fetch(`/api/shares/${shareId}/logs`);
        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.error || 'Failed to load logs');
        }

        const logs = result.logs || [];

        if (logs.length === 0) {
            content.innerHTML = `
                <div class="text-center py-8">
                    <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"></path>
                    </svg>
                    <p class="mt-2 text-sm text-gray-500">No access logs yet.</p>
                </div>
            `;
        } else {
            content.innerHTML = `
                <div class="space-y-2">
                    ${logs.map(log => createAccessLogEntryHTML(log)).join('')}
                </div>
            `;
        }

    } catch (error) {
        console.error('Failed to load access logs:', error);
        content.innerHTML = `
            <div class="text-center py-8 text-red-500">
                <p>Failed to load access logs.</p>
            </div>
        `;
    }
}

/**
 * Creates HTML for an access log entry
 * @param {Object} log - Access log object
 * @returns {string} HTML string
 */
function createAccessLogEntryHTML(log) {
    const actionIcon = {
        'view': '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path>',
        'download': '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path>',
        'upload': '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path>'
    }[log.action] || '';

    const actionColor = {
        'view': 'bg-blue-100 text-blue-600',
        'download': 'bg-green-100 text-green-600',
        'upload': 'bg-yellow-100 text-yellow-600'
    }[log.action] || 'bg-gray-100 text-gray-600';

    const accessDate = new Date(log.accessed_at);
    const truncatedIP = log.ip_address ? log.ip_address.split(':')[0] : 'Unknown';

    return `
        <div class="flex items-center justify-between py-2 border-b border-gray-100 last:border-b-0">
            <div class="flex items-center space-x-3">
                <span class="inline-flex items-center justify-center w-6 h-6 rounded-full ${actionColor}">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        ${actionIcon}
                    </svg>
                </span>
                <div>
                    <span class="text-sm text-gray-700 capitalize">${log.action}</span>
                    ${log.file_name ? `<span class="text-xs text-gray-500 ml-1">(${escapeHtml(log.file_name)})</span>` : ''}
                </div>
            </div>
            <div class="text-right">
                <div class="text-xs text-gray-500">${formatRelativeDate(accessDate)}</div>
                <div class="text-[10px] text-gray-400" title="${escapeHtml(log.ip_address || '')}">${escapeHtml(truncatedIP)}</div>
            </div>
        </div>
    `;
}

/**
 * Closes the access logs modal
 */
function closeAccessLogsModal() {
    const modal = document.getElementById('access-logs-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
}

// ============================================
// Edit Expiration
// ============================================

/**
 * Opens the edit expiration modal
 * @param {string} shareId - Share ID
 * @param {string} currentExpiration - Current expiration date (ISO string)
 */
function editShareExpiration(shareId, currentExpiration) {
    const modal = document.getElementById('edit-expiration-modal');
    const shareIdField = document.getElementById('edit-share-id');
    const hasExpiration = document.getElementById('edit-has-expiration');
    const expirationInput = document.getElementById('edit-expiration-input');
    const container = document.getElementById('edit-expiration-container');

    if (!modal) return;

    if (shareIdField) shareIdField.value = shareId;

    if (currentExpiration) {
        if (hasExpiration) hasExpiration.checked = true;
        if (container) container.classList.remove('hidden');
        if (expirationInput) {
            const date = new Date(currentExpiration);
            expirationInput.value = date.toISOString().slice(0, 16);
        }
    } else {
        if (hasExpiration) hasExpiration.checked = false;
        if (container) container.classList.add('hidden');
        if (expirationInput) expirationInput.value = '';
    }

    modal.classList.remove('hidden');
}

/**
 * Toggles expiration input in edit modal
 */
function toggleEditExpiration() {
    const toggle = document.getElementById('edit-has-expiration');
    const container = document.getElementById('edit-expiration-container');

    if (container && toggle) {
        if (toggle.checked) {
            container.classList.remove('hidden');
            setEditExpiration(7);
        } else {
            container.classList.add('hidden');
        }
    }
}

/**
 * Sets expiration in edit modal
 * @param {number} days - Days from now
 */
function setEditExpiration(days) {
    const input = document.getElementById('edit-expiration-input');
    if (input) {
        const date = new Date();
        date.setDate(date.getDate() + days);
        input.value = date.toISOString().slice(0, 16);
    }
}

/**
 * Submits expiration update
 * @param {Event} event - Form submit event
 * @returns {boolean}
 */
async function submitExpirationUpdate(event) {
    event.preventDefault();

    const shareId = document.getElementById('edit-share-id').value;
    const hasExpiration = document.getElementById('edit-has-expiration').checked;
    const expirationValue = document.getElementById('edit-expiration-input').value;

    closeEditExpirationModal();

    try {
        const data = {
            expires_at: hasExpiration && expirationValue ? new Date(expirationValue).toISOString() : null
        };

        const response = await fetch(`/api/shares/${shareId}`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            const result = await response.json();
            throw new Error(result.error || 'Failed to update expiration');
        }

        // Reload shares
        if (shareState.currentResourceId) {
            loadExistingShares(shareState.currentResourceId, shareState.currentResourceType);
        }

        showToast('success', 'Expiration updated successfully');

    } catch (error) {
        console.error('Update error:', error);
        showToast('error', 'Failed to update expiration', error.message);
    }

    return false;
}

/**
 * Closes the edit expiration modal
 */
function closeEditExpirationModal() {
    const modal = document.getElementById('edit-expiration-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
}

// ============================================
// Utility Functions
// ============================================

/**
 * Truncates a URL for display
 * @param {string} url - URL to truncate
 * @param {number} maxLength - Maximum length
 * @returns {string} Truncated URL
 */
function truncateUrl(url, maxLength = 40) {
    if (url.length <= maxLength) return url;
    const start = url.substring(0, Math.floor(maxLength / 2) - 2);
    const end = url.substring(url.length - Math.floor(maxLength / 2) + 2);
    return `${start}...${end}`;
}

/**
 * Formats a date relative to now
 * @param {Date} date - Date to format
 * @returns {string} Formatted date string
 */
function formatRelativeDate(date) {
    const now = new Date();
    const diff = date - now;
    const absDiff = Math.abs(diff);

    const minutes = Math.floor(absDiff / 60000);
    const hours = Math.floor(absDiff / 3600000);
    const days = Math.floor(absDiff / 86400000);

    if (diff < 0) {
        // Past
        if (minutes < 1) return 'just now';
        if (minutes < 60) return `${minutes}m ago`;
        if (hours < 24) return `${hours}h ago`;
        if (days < 7) return `${days}d ago`;
        return date.toLocaleDateString();
    } else {
        // Future
        if (minutes < 60) return `in ${minutes}m`;
        if (hours < 24) return `in ${hours}h`;
        if (days < 7) return `in ${days}d`;
        return date.toLocaleDateString();
    }
}

/**
 * Escapes HTML special characters
 * @param {string} str - String to escape
 * @returns {string} Escaped string
 */
function escapeHtml(str) {
    if (!str) return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

// ============================================
// Shares Page Functions
// ============================================

/**
 * Filters shares based on selected filters
 */
function filterShares() {
    // This would typically trigger a new API request with filters
    const typeFilter = document.getElementById('filter-type')?.value;
    const statusFilter = document.getElementById('filter-status')?.value;

    // Reload shares with filters
    let url = '/api/shares?';
    if (typeFilter) url += `resource_type=${typeFilter}&`;
    if (statusFilter === 'expired') url += 'expired=true&';
    if (statusFilter === 'active') url += 'expired=false&';

    htmx.ajax('GET', url, { target: '#shares-list-content', swap: 'innerHTML' });
}

/**
 * Sorts shares
 */
function sortShares() {
    const sortBy = document.getElementById('sort-by')?.value;
    filterShares(); // Reapply filters with new sort
}

/**
 * Searches shares
 * @param {string} query - Search query
 */
function searchShares(query) {
    // Debounced search
    clearTimeout(window.shareSearchTimeout);
    window.shareSearchTimeout = setTimeout(() => {
        filterShares();
    }, 300);
}

/**
 * Toggles selection of all shares
 * @param {boolean} checked - Whether to select all
 */
function toggleSelectAllShares(checked) {
    const checkboxes = document.querySelectorAll('.share-checkbox');
    checkboxes.forEach(cb => {
        cb.checked = checked;
        const shareId = cb.dataset.shareId;
        if (checked) {
            shareState.selectedShares.add(shareId);
        } else {
            shareState.selectedShares.delete(shareId);
        }
    });
    updateBulkActions();
}

/**
 * Updates bulk action visibility
 */
function updateBulkActions() {
    const bulkActions = document.getElementById('bulk-actions');
    const selectedCount = document.getElementById('selected-count');
    const countContainer = document.getElementById('selected-shares-count');

    const count = shareState.selectedShares.size;

    if (bulkActions) {
        bulkActions.classList.toggle('hidden', count === 0);
    }
    if (selectedCount) {
        selectedCount.textContent = count;
    }
    if (countContainer) {
        countContainer.classList.toggle('hidden', count === 0);
    }
}

/**
 * Bulk revokes selected shares
 */
function bulkRevokeShares() {
    if (shareState.selectedShares.size === 0) return;

    shareState.pendingRevokeBulk = true;

    const modal = document.getElementById('revoke-confirm-modal');
    const message = document.getElementById('revoke-confirm-message');

    if (message) {
        message.textContent = `Are you sure you want to revoke ${shareState.selectedShares.size} share(s)? These links will immediately stop working.`;
    }

    if (modal) {
        modal.classList.remove('hidden');
    }
}

// ============================================
// Expose Functions Globally
// ============================================

window.openShareModal = openShareModal;
window.closeShareModal = closeShareModal;
window.switchShareTab = switchShareTab;
window.updatePermissionSelection = updatePermissionSelection;
window.toggleSharePassword = toggleSharePassword;
window.toggleShareExpiration = toggleShareExpiration;
window.setShareExpiration = setShareExpiration;
window.submitShareForm = submitShareForm;
window.revokeShare = revokeShare;
window.revokeShareFromModal = revokeShareFromModal;
window.confirmRevokeShare = confirmRevokeShare;
window.closeRevokeConfirmModal = closeRevokeConfirmModal;
window.copyShareLink = copyShareLink;
window.copyShareLinkFromInput = copyShareLinkFromInput;
window.showShareQRCode = showShareQRCode;
window.closeQRCodeModal = closeQRCodeModal;
window.viewShareAccessLogs = viewShareAccessLogs;
window.closeAccessLogsModal = closeAccessLogsModal;
window.editShareExpiration = editShareExpiration;
window.toggleEditExpiration = toggleEditExpiration;
window.setEditExpiration = setEditExpiration;
window.submitExpirationUpdate = submitExpirationUpdate;
window.closeEditExpirationModal = closeEditExpirationModal;
window.filterShares = filterShares;
window.sortShares = sortShares;
window.searchShares = searchShares;
window.toggleSelectAllShares = toggleSelectAllShares;
window.bulkRevokeShares = bulkRevokeShares;
