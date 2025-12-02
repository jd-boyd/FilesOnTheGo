import { test, expect } from '@playwright/test';

test.describe('Dashboard New Folder Functionality', () => {
  const ADMIN_EMAIL = 'admin@filesonthego.local';
  const ADMIN_PASSWORD = 'admin123';

  test.beforeEach(async ({ page }) => {
    // Log in as admin before each test
    await page.goto('/login');
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard');
  });

  test.describe('Login Verification', () => {
    test('Admin can successfully log in and access dashboard', async ({ page }) => {
      // Verify we're on the dashboard page
      await expect(page).toHaveURL(/.*dashboard/);

      // Check for dashboard elements
      await expect(page.locator('h1')).toContainText('My Files');

      // Look for storage usage card
      const storageCard = page.locator('text=Storage Usage');
      await expect(storageCard).toBeVisible();

      console.log('✅ Admin login successful - dashboard loaded');
    });

    test('Dashboard displays page header and actions correctly', async ({ page }) => {
      // Check page header
      const pageHeader = page.locator('h1');
      await expect(pageHeader).toContainText('My Files');
      await expect(pageHeader).toBeVisible();

      // Check that the page actions section exists
      const pageActions = page.locator('button:has-text("New Folder")');
      await expect(pageActions).toBeVisible();

      console.log('✅ Dashboard page structure verified');
    });
  });

  test.describe('New Folder Button Functionality', () => {
    test('New Folder button is visible and clickable', async ({ page }) => {
      // Look for the New Folder button specifically
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');
      await expect(newFolderButton).toBeVisible();

      // Check button text and icon
      await expect(newFolderButton).toContainText('New Folder');

      // Verify the button has the correct styling
      await expect(newFolderButton).toHaveClass(/inline-flex.*items-center.*px-4.*py-2/);

      console.log('✅ New Folder button is visible and properly styled');
    });

    test('Clicking New Folder button opens modal', async ({ page }) => {
      // Click the New Folder button
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');
      await newFolderButton.click();

      // Wait for modal to appear
      const modal = page.locator('#new-folder-modal');
      await expect(modal).toBeVisible();

      // Check modal content
      await expect(modal.locator('h3')).toContainText('Create New Folder');

      // Check for input field
      const folderNameInput = page.locator('#folder-name');
      await expect(folderNameInput).toBeVisible();
      await expect(folderNameInput).toHaveAttribute('placeholder', 'New Folder');

      // Check for form buttons
      const createButton = modal.locator('button[type="submit"]');
      await expect(createButton).toBeVisible();
      await expect(createButton).toContainText('Create');

      const cancelButton = modal.locator('button:has-text("Cancel")');
      await expect(cancelButton).toBeVisible();

      console.log('✅ New Folder modal opens correctly with all elements');

      // Take screenshot for verification
      await page.screenshot({ path: 'new-folder-modal-open.png', fullPage: false });
      console.log('📸 Screenshot saved as new-folder-modal-open.png');
    });

    test('Modal can be closed with Cancel button', async ({ page }) => {
      // Open modal
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');
      await newFolderButton.click();

      // Verify modal is open
      const modal = page.locator('#new-folder-modal');
      await expect(modal).toBeVisible();

      // Click cancel button
      const cancelButton = modal.locator('button:has-text("Cancel")');
      await cancelButton.click();

      // Verify modal is closed
      await expect(modal).toBeHidden();

      console.log('✅ Modal can be closed with Cancel button');
    });

    test('Modal can be closed with Escape key', async ({ page }) => {
      // Open modal
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');
      await newFolderButton.click();

      // Verify modal is open
      const modal = page.locator('#new-folder-modal');
      await expect(modal).toBeVisible();

      // Press Escape key
      await page.keyboard.press('Escape');

      // Verify modal is closed
      await expect(modal).toBeHidden();

      console.log('✅ Modal can be closed with Escape key');
    });

    test('Modal can be closed by clicking overlay', async ({ page }) => {
      // Open modal
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');
      await newFolderButton.click();

      // Verify modal is open
      const modal = page.locator('#new-folder-modal');
      await expect(modal).toBeVisible();

      // Click on overlay (background)
      const overlay = modal.locator('.fixed.inset-0.bg-gray-500');
      await overlay.click({ position: { x: 10, y: 10 } }); // Click in corner

      // Verify modal is closed
      await expect(modal).toBeHidden();

      console.log('✅ Modal can be closed by clicking overlay');
    });

    test('Modal input validation and form submission', async ({ page }) => {
      // Open modal
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');
      await newFolderButton.click();

      // Verify modal is open
      const modal = page.locator('#new-folder-modal');
      await expect(modal).toBeVisible();

      // Find the input field
      const folderNameInput = page.locator('#folder-name');
      await expect(folderNameInput).toBeVisible();

      // Type a folder name
      const testFolderName = 'Test Folder ' + Date.now();
      await folderNameInput.fill(testFolderName);

      // Verify the input has the text
      await expect(folderNameInput).toHaveValue(testFolderName);

      // Check that the Create button is enabled
      const createButton = modal.locator('button[type="submit"]');
      await expect(createButton).toBeEnabled();

      // Find the form to check its action
      const form = modal.locator('form[hx-post="/api/directories"]');
      await expect(form).toBeVisible();

      // Verify the form has the correct HTMX attributes
      await expect(form).toHaveAttribute('hx-post', '/api/directories');
      await expect(form).toHaveAttribute('hx-target', '#file-list-container');

      console.log('✅ Modal form validation and structure verified');
      console.log(`📝 Test folder name: ${testFolderName}`);

      // Take screenshot before potential submission
      await page.screenshot({ path: 'new-folder-modal-filled.png', fullPage: false });
      console.log('📸 Screenshot saved as new-folder-modal-filled.png');

      // Note: We don't actually submit the form in this test to avoid creating
      // actual folders, but we verify all the structure is correct
    });
  });

  test.describe('JavaScript Function Availability', () => {
    test('openNewFolderModal function is defined and accessible', async ({ page }) => {
      // Wait for page to load completely
      await page.waitForLoadState('networkidle');

      // Check if the function is defined by evaluating it in the page context
      const isFunctionDefined = await page.evaluate(() => {
        return typeof window.openNewFolderModal === 'function';
      });

      expect(isFunctionDefined).toBe(true);
      console.log('✅ openNewFolderModal function is properly defined');
    });

    test('closeNewFolderModal function is defined and accessible', async ({ page }) => {
      // Wait for page to load completely
      await page.waitForLoadState('networkidle');

      // Check if the function is defined
      const isFunctionDefined = await page.evaluate(() => {
        return typeof window.closeNewFolderModal === 'function';
      });

      expect(isFunctionDefined).toBe(true);
      console.log('✅ closeNewFolderModal function is properly defined');
    });

    test('Required JavaScript files are loaded', async ({ page }) => {
      // Wait for page to load
      await page.waitForLoadState('networkidle');

      // Check if file-browser.js script was loaded by looking for a specific function
      const hasFileBrowserFunctions = await page.evaluate(() => {
        return typeof window.toggleFileSelection === 'function' &&
               typeof window.showContextMenu === 'function' &&
               typeof window.openNewFolderModal === 'function';
      });

      expect(hasFileBrowserFunctions).toBe(true);
      console.log('✅ All required file-browser.js functions are available');
    });
  });

  test.describe('Error Scenarios', () => {
    test('New Folder button does not throw JavaScript errors', async ({ page }) => {
      // Listen for console errors
      let consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });

      // Wait for page load
      await page.waitForLoadState('networkidle');

      // Click the New Folder button
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');
      await newFolderButton.click();

      // Wait a moment for any JavaScript to execute
      await page.waitForTimeout(1000);

      // Check for console errors
      expect(consoleErrors.length).toBe(0);

      if (consoleErrors.length > 0) {
        console.log('❌ Console errors found:', consoleErrors);
      } else {
        console.log('✅ No JavaScript errors when clicking New Folder button');
      }
    });

    test('Modal closes cleanly without errors', async ({ page }) => {
      // Listen for console errors
      let consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });

      // Open and close modal multiple times
      const newFolderButton = page.locator('button[onclick*="openNewFolderModal()"]');

      for (let i = 0; i < 3; i++) {
        await newFolderButton.click();
        await page.waitForTimeout(500);

        const cancelButton = page.locator('#new-folder-modal button:has-text("Cancel")');
        await cancelButton.click();
        await page.waitForTimeout(500);
      }

      // Check for console errors
      expect(consoleErrors.length).toBe(0);

      if (consoleErrors.length > 0) {
        console.log('❌ Console errors found during modal operations:', consoleErrors);
      } else {
        console.log('✅ No JavaScript errors during modal open/close operations');
      }
    });
  });
});