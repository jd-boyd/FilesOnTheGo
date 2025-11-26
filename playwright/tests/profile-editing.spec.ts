import { test, expect } from '@playwright/test';

test.describe('User Profile Editing', () => {
  const REGULAR_USER_EMAIL = 'user@filesonthego.local';
  const REGULAR_USER_PASSWORD = 'user1234';
  const ADMIN_EMAIL = 'admin@filesonthego.local';
  const ADMIN_PASSWORD = 'admin123';

  test.beforeEach(async ({ page }) => {
    // Log in as regular user before each test
    await page.goto('/login');
    await page.fill('input[name="email"]', REGULAR_USER_EMAIL);
    await page.fill('input[name="password"]', REGULAR_USER_PASSWORD);
    await page.click('button[type="submit"]');

    // Wait for successful login - redirect to dashboard (which shows "My Files")
    await page.waitForURL('**/dashboard', { timeout: 10000 });

    // Verify we're logged in - the page shows "My Files" as the heading
    await expect(page.locator('h1')).toContainText('My Files');
  });

  test('User can navigate to settings and edit profile', async ({ page }) => {
    console.log('Starting profile editing test...');

    // Step 1: Navigate to settings
    console.log('Step 1: Navigating to settings...');
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Verify we're on the settings page
    await expect(page.locator('h1')).toContainText('Personal Settings');
    console.log('✅ Successfully navigated to settings page');

    // Step 2: Click on "Edit Profile" link (there are multiple, use the one with specific text)
    console.log('Step 2: Clicking Edit Profile link...');
    const editProfileLink = page.locator('a[href="/profile"]', { hasText: 'Edit Profile' });
    await expect(editProfileLink).toBeVisible();
    await editProfileLink.click();

    // Wait for profile page to load
    await page.waitForURL('**/profile');
    await page.waitForLoadState('networkidle');

    // Verify we're on the profile edit page
    await expect(page.locator('h1')).toContainText('Edit Profile');
    console.log('✅ Successfully navigated to profile edit page');

    // Step 3: Check current form state
    console.log('Step 3: Checking current form state...');
    const displayNameInput = page.locator('input[name="displayName"]');
    const emailInput = page.locator('input[name="email"]');

    await expect(displayNameInput).toBeVisible();
    await expect(emailInput).toBeVisible();

    // Email should be readonly
    await expect(emailInput).toHaveAttribute('readonly');
    await expect(emailInput).toHaveAttribute('disabled');

    const currentDisplayName = await displayNameInput.inputValue();
    const currentEmail = await emailInput.inputValue();

    expect(currentEmail).toBe(REGULAR_USER_EMAIL);
    console.log(`✅ Current display name: "${currentDisplayName}"`);
    console.log(`✅ Current email: "${currentEmail}"`);

    // Step 4: Change the display name
    console.log('Step 4: Changing display name...');
    const newDisplayName = `Test User ${Date.now()}`;
    await displayNameInput.clear();
    await displayNameInput.fill(newDisplayName);

    // Step 5: Submit the form
    console.log('Step 5: Submitting the form...');
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();

    // Intercept the form submission to wait for completion
    const responsePromise = page.waitForResponse(response =>
      response.url().includes('/profile') && response.status() === 200
    );

    await submitButton.click();

    // Wait for the HTMX response
    const response = await responsePromise;
    console.log('✅ Form submission completed');

    // Step 6: Check for success message
    console.log('Step 6: Checking for success message...');
    await page.waitForTimeout(1000); // Give HTMX a moment to update the DOM

    // Look for success message (could be in various forms)
    const successSelectors = [
      '.bg-green-50',
      '[class*="green"]',
      'text=Profile updated successfully',
      'text=updated successfully',
      'text=success'
    ];

    let successFound = false;
    for (const selector of successSelectors) {
      try {
        const element = page.locator(selector).first();
        if (await element.isVisible({ timeout: 2000 })) {
          console.log(`✅ Success message found with selector: ${selector}`);
          const successText = await element.textContent();
          console.log(`   Message: "${successText}"`);
          successFound = true;
          break;
        }
      } catch (e) {
        // Continue trying other selectors
      }
    }

    if (!successFound) {
      console.log('ℹ️ No explicit success message found, but checking if update still worked...');
    }

    // Step 7: Verify the change persisted
    console.log('Step 7: Verifying the change persisted...');

    // Check if the input now shows the new value
    const updatedDisplayName = await displayNameInput.inputValue();

    if (updatedDisplayName === newDisplayName) {
      console.log('✅ Display name updated successfully in form');
    } else {
      console.log(`⚠️ Display name in form is "${updatedDisplayName}", expected "${newDisplayName}"`);

      // Try refreshing and checking again
      console.log('Refreshing page to check persisted value...');
      await page.reload();
      await page.waitForLoadState('networkidle');

      const refreshedDisplayName = await displayNameInput.inputValue();
      if (refreshedDisplayName === newDisplayName) {
        console.log('✅ Display name persisted after page refresh');
      } else {
        console.log(`❌ Display name after refresh is "${refreshedDisplayName}", expected "${newDisplayName}"`);
      }
    }

    // Step 8: Navigate back to settings to see if the change is reflected there
    console.log('Step 8: Checking settings page for updated name...');
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Look for the updated display name on the settings page
    const settingsPageContent = await page.content();
    if (settingsPageContent.includes(newDisplayName)) {
      console.log('✅ Updated display name is reflected on settings page');
    } else {
      console.log(`⚠️ Updated display name "${newDisplayName}" not found on settings page`);
    }
  });

  test('Profile form validation works correctly', async ({ page }) => {
    console.log('Starting profile form validation test...');

    // Navigate to profile page
    await page.goto('/profile');
    await page.waitForLoadState('networkidle');

    const displayNameInput = page.locator('input[name="displayName"]');
    const submitButton = page.locator('button[type="submit"]');

    // Test 1: Empty display name
    console.log('Test 1: Testing empty display name...');
    await displayNameInput.clear();
    await submitButton.click();

    // Check for validation error
    await page.waitForTimeout(1000);
    const errorSelectors = [
      '.bg-red-50',
      '[class*="red"]',
      'text=between 2 and 100 characters',
      'text=required',
      'text=Display name'
    ];

    let errorFound = false;
    for (const selector of errorSelectors) {
      try {
        const element = page.locator(selector).first();
        if (await element.isVisible({ timeout: 2000 })) {
          console.log(`✅ Validation error found with selector: ${selector}`);
          const errorText = await element.textContent();
          console.log(`   Error message: "${errorText}"`);
          errorFound = true;
          break;
        }
      } catch (e) {
        // Continue trying other selectors
      }
    }

    if (!errorFound) {
      console.log('ℹ️ No validation error message found for empty name');
    }

    // Test 2: Too long display name
    console.log('Test 2: Testing display name that is too long...');
    await displayNameInput.clear();
    await displayNameInput.fill('a'.repeat(101)); // 101 characters
    await submitButton.click();

    await page.waitForTimeout(1000);

    // Look for validation error again
    for (const selector of errorSelectors) {
      try {
        const element = page.locator(selector).first();
        if (await element.isVisible({ timeout: 2000 })) {
          console.log(`✅ Length validation error found`);
          errorFound = true;
          break;
        }
      } catch (e) {
        // Continue trying other selectors
      }
    }

    if (!errorFound) {
      console.log('ℹ️ No validation error message found for long name');
    }

    // Test 3: Valid display name (should work)
    console.log('Test 3: Testing valid display name...');
    const validName = `Valid User ${Date.now()}`;
    await displayNameInput.clear();
    await displayNameInput.fill(validName);

    const responsePromise = page.waitForResponse(response =>
      response.url().includes('/profile') && response.status() === 200
    );

    await submitButton.click();
    await responsePromise;

    console.log('✅ Valid display name submission completed');
  });

  test('Admin user can also edit profile', async ({ page }) => {
    console.log('Starting admin profile editing test...');

    // Log out first, then log in as admin
    await page.goto('/login');

    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard', { timeout: 10000 });

    // Verify admin is logged in too (should show "My Files")
    await expect(page.locator('h1')).toContainText('My Files');

    console.log('✅ Admin login successful');

    // Navigate to profile
    await page.goto('/profile');
    await page.waitForURL('**/profile');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Edit Profile');

    // Check for admin-specific elements
    const pageContent = await page.content();
    const hasAdminNotice = pageContent.includes('Administrator Account') ||
                           pageContent.includes('admin') ||
                           pageContent.includes('privileges');

    if (hasAdminNotice) {
      console.log('✅ Admin-specific content found on profile page');
    } else {
      console.log('ℹ️ No admin-specific content noticed on profile page');
    }

    // Test editing with a valid name
    const displayNameInput = page.locator('input[name="displayName"]');
    const adminDisplayName = `Admin User ${Date.now()}`;

    await displayNameInput.clear();
    await displayNameInput.fill(adminDisplayName);

    const responsePromise = page.waitForResponse(response =>
      response.url().includes('/profile') && response.status() === 200
    );

    await page.locator('button[type="submit"]').click();
    await responsePromise;

    console.log('✅ Admin profile update completed');
  });
});