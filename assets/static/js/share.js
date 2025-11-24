/**
 * FilesOnTheGo - Share Management JavaScript Utilities
 * Handles share modal interactions, form controls, and clipboard operations
 */

// ============================================
// Share Modal Functions
// ============================================

/**
 * Opens the share modal for a file or directory
 * @param {string} id - Resource ID
 * @param {string} type - Resource type ('file' or 'directory')
 */
function openShareModal(id, type) {
    const modal = document.getElementById('share-modal');
    const itemNameSpan = document.getElementById('share-item-name');
    const itemTypeLabel = document.getElementById('share-item-type-label');
    const resourceTypeField = document.getElementById('share-resource-type');
    const resourceIdField = document.getElementById('share-resource-id');
    const resultDiv = document.getElementById('share-result');
    const existingSharesList = document.getElementById('existing-shares-list');

    if (!modal) return;

    // Get item info from DOM
    const item = document.querySelector(`.file-item[data-id="${id}"]`);
    const name = item?.dataset.name || 'this item';

    // Update modal content
    if (itemNameSpan) itemNameSpan.textContent = name;
    if (itemTypeLabel) itemTypeLabel.textContent = type === 'directory' ? 'Folder' : 'File';
    if (resourceTypeField) resourceTypeField.value = type;
    if (resourceIdField) resourceIdField.value = id;
    if (resultDiv) resultDiv.innerHTML = '';

    // Reset form state
    resetShareForm();

    // Load existing shares for this resource
    if (existingSharesList) {
        existingSharesList.setAttribute('hx-get', `/api/shares/resource/${type}/${id}`);
        htmx.trigger(existingSharesList, 'revealed');
    }

    // Show modal
    modal.classList.remove('hidden');

    // Focus first radio button for accessibility
    const firstRadio = modal.querySelector('input[name="permission_type"]');
    if (firstRadio) firstRadio.focus();
}

/**
 * Closes the share modal
 */
function closeShareModal() {
    const modal = document.getElementById('share-modal');
    if (modal) {
        modal.classList.add('hidden');
        resetShareForm();
    }
}

/**
 * Resets the share form to default state
 */
function resetShareForm() {
    const form = document.getElementById('share-form');
    if (form) form.reset();

    // Reset password section
    const passwordToggle = document.getElementById('share-password-toggle');
    const passwordSection = document.getElementById('share-password-section');
    if (passwordToggle) passwordToggle.checked = false;
    if (passwordSection) passwordSection.classList.add('hidden');

    // Reset expiration section
    const expirationToggle = document.getElementById('share-expiration-toggle');
    const expirationSection = document.getElementById('share-expiration-section');
    if (expirationToggle) expirationToggle.checked = false;
    if (expirationSection) expirationSection.classList.add('hidden');

    // Clear result area
    const resultDiv = document.getElementById('share-result');
    if (resultDiv) resultDiv.innerHTML = '';

    // Reset permission selection highlight
    updatePermissionHighlight();
}

// ============================================
// Form Control Toggles
// ============================================

/**
 * Toggles password protection input visibility
 */
function toggleSharePassword() {
    const toggle = document.getElementById('share-password-toggle');
    const section = document.getElementById('share-password-section');
    const input = document.getElementById('share-password');

    if (section && toggle) {
        if (toggle.checked) {
            section.classList.remove('hidden');
            if (input) input.focus();
        } else {
            section.classList.add('hidden');
            if (input) input.value = '';
        }
    }
}

/**
 * Toggles expiration date input visibility
 */
function toggleShareExpiration() {
    const toggle = document.getElementById('share-expiration-toggle');
    const section = document.getElementById('share-expiration-section');
    const input = document.getElementById('share-expires-at');

    if (section && toggle) {
        if (toggle.checked) {
            section.classList.remove('hidden');
            // Set default to 7 days from now
            if (input && !input.value) {
                setShareExpiration(7);
            }
        } else {
            section.classList.add('hidden');
            if (input) input.value = '';
        }
    }
}

/**
 * Sets expiration date to X days from now
 * @param {number} days - Number of days from now
 */
function setShareExpiration(days) {
    const input = document.getElementById('share-expires-at');
    if (!input) return;

    const date = new Date();
    date.setDate(date.getDate() + days);

    // Format as datetime-local value (YYYY-MM-DDTHH:MM)
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');

    input.value = `${year}-${month}-${day}T${hours}:${minutes}`;
}

/**
 * Updates permission option highlighting based on selection
 */
function updatePermissionHighlight() {
    const options = document.querySelectorAll('.permission-option');
    options.forEach(option => {
        const radio = option.querySelector('input[type="radio"]');
        if (radio && radio.checked) {
            option.classList.add('border-blue-500', 'bg-blue-50');
            option.classList.remove('border-gray-200');
        } else {
            option.classList.remove('border-blue-500', 'bg-blue-50');
            option.classList.add('border-gray-200');
        }
    });
}

// ============================================
// Clipboard Functions
// ============================================

/**
 * Copies share link from the input field
 */
async function copyShareLinkFromInput() {
    const input = document.getElementById('share-url-input');
    if (!input) return;

    try {
        await navigator.clipboard.writeText(input.value);
        showToast('success', 'Link copied to clipboard');

        // Visual feedback - select the input
        input.select();
    } catch (err) {
        console.error('Failed to copy:', err);
        showToast('error', 'Failed to copy to clipboard');

        // Fallback: select the input for manual copy
        input.select();
    }
}

/**
 * Copies share link from a share item by element ID
 * @param {string} elementId - The ID of the share item element
 */
async function copyShareLinkById(elementId) {
    const element = document.getElementById(elementId);
    if (!element) return;

    const shareUrl = element.dataset.shareUrl;
    if (!shareUrl) return;

    try {
        await navigator.clipboard.writeText(shareUrl);
        showToast('success', 'Link copied to clipboard');
    } catch (err) {
        console.error('Failed to copy:', err);
        showToast('error', 'Failed to copy to clipboard');
    }
}

// ============================================
// Share Management Page Functions
// ============================================

/**
 * Filters shares based on selected criteria
 */
function filterShares() {
    const resourceType = document.getElementById('filter-resource-type')?.value || '';
    const status = document.getElementById('filter-status')?.value || '';

    // Build query parameters
    const params = new URLSearchParams();
    if (resourceType) params.set('resource_type', resourceType);
    if (status) params.set('status', status);

    const container = document.getElementById('shares-list-container');
    if (container) {
        container.setAttribute('hx-get', `/api/shares/list-htmx?${params.toString()}`);
        htmx.trigger(container, 'load');
    }
}

/**
 * Searches shares by keyword
 * @param {string} query - Search query
 */
function searchShares(query) {
    // Debounce the search
    clearTimeout(window.shareSearchTimeout);
    window.shareSearchTimeout = setTimeout(() => {
        const container = document.getElementById('shares-list-container');
        if (container) {
            const currentUrl = new URL(container.getAttribute('hx-get'), window.location.origin);
            if (query) {
                currentUrl.searchParams.set('search', query);
            } else {
                currentUrl.searchParams.delete('search');
            }
            container.setAttribute('hx-get', currentUrl.pathname + currentUrl.search);
            htmx.trigger(container, 'load');
        }
    }, 300);
}

/**
 * Sorts shares based on selected criteria
 */
function sortShares() {
    const sortValue = document.getElementById('sort-shares')?.value || 'created_desc';

    const container = document.getElementById('shares-list-container');
    if (container) {
        const currentUrl = new URL(container.getAttribute('hx-get') || '/api/shares/list-htmx', window.location.origin);
        currentUrl.searchParams.set('sort', sortValue);
        container.setAttribute('hx-get', currentUrl.pathname + currentUrl.search);
        htmx.trigger(container, 'load');
    }
}

// ============================================
// Bulk Selection Functions
// ============================================

const shareSelectionState = {
    selectedShares: new Set()
};

/**
 * Toggles share selection
 * @param {string} shareId - Share ID to toggle
 */
function toggleShareSelection(shareId) {
    if (shareSelectionState.selectedShares.has(shareId)) {
        shareSelectionState.selectedShares.delete(shareId);
    } else {
        shareSelectionState.selectedShares.add(shareId);
    }
    updateShareSelectionUI();
}

/**
 * Clears all share selections
 */
function clearShareSelection() {
    shareSelectionState.selectedShares.clear();
    updateShareSelectionUI();
}

/**
 * Updates the UI to reflect selection state
 */
function updateShareSelectionUI() {
    const count = shareSelectionState.selectedShares.size;
    const bulkActions = document.getElementById('bulk-actions');
    const selectedCount = document.getElementById('selected-count');

    if (bulkActions) {
        if (count > 0) {
            bulkActions.classList.remove('hidden');
        } else {
            bulkActions.classList.add('hidden');
        }
    }

    if (selectedCount) {
        selectedCount.textContent = count;
    }
}

/**
 * Revokes all selected shares
 */
async function revokeSelectedShares() {
    const selected = Array.from(shareSelectionState.selectedShares);
    if (selected.length === 0) return;

    const confirmed = confirm(`Are you sure you want to revoke ${selected.length} share(s)? This action cannot be undone.`);
    if (!confirmed) return;

    let revokedCount = 0;
    for (const shareId of selected) {
        try {
            const response = await fetch(`/api/shares/${shareId}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                revokedCount++;
                const item = document.getElementById(`share-item-${shareId}`);
                if (item) {
                    item.style.opacity = '0';
                    setTimeout(() => item.remove(), 300);
                }
            }
        } catch (error) {
            console.error(`Failed to revoke share ${shareId}:`, error);
        }
    }

    clearShareSelection();
    showToast('success', `${revokedCount} share(s) revoked`);
}

// ============================================
// Initialization
// ============================================

document.addEventListener('DOMContentLoaded', function() {
    // Add event listeners for permission option highlighting
    const permissionOptions = document.querySelectorAll('.permission-option input[type="radio"]');
    permissionOptions.forEach(radio => {
        radio.addEventListener('change', updatePermissionHighlight);
    });

    // Initial highlight update
    updatePermissionHighlight();

    // Listen for HTMX events
    document.body.addEventListener('htmx:afterSwap', function(event) {
        // Re-initialize HTMX on dynamically loaded content
        if (event.detail.target.id === 'share-result' ||
            event.detail.target.id === 'existing-shares-list' ||
            event.detail.target.id === 'shares-list-container') {
            htmx.process(event.detail.target);
        }
    });

    // Close modal on Escape key
    document.addEventListener('keydown', function(event) {
        if (event.key === 'Escape') {
            closeShareModal();
        }
    });
});

// ============================================
// Expose functions globally
// ============================================

window.openShareModal = openShareModal;
window.closeShareModal = closeShareModal;
window.resetShareForm = resetShareForm;
window.toggleSharePassword = toggleSharePassword;
window.toggleShareExpiration = toggleShareExpiration;
window.setShareExpiration = setShareExpiration;
window.copyShareLinkFromInput = copyShareLinkFromInput;
window.copyShareLinkById = copyShareLinkById;
window.filterShares = filterShares;
window.searchShares = searchShares;
window.sortShares = sortShares;
window.toggleShareSelection = toggleShareSelection;
window.clearShareSelection = clearShareSelection;
window.revokeSelectedShares = revokeSelectedShares;