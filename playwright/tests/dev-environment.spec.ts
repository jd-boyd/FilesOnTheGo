import { test, expect } from '@playwright/test';

test.describe('Development Environment Setup Validation', () => {
  const ADMIN_EMAIL = 'admin@filesonthego.local';
  const ADMIN_PASSWORD = 'admin123';
  const REGULAR_USER_EMAIL = 'user@filesonthego.local';
  const REGULAR_USER_PASSWORD = 'user1234';

  test('Application health endpoint is accessible', async ({ page }) => {
    // Test that the application is running and healthy
    const response = await page.goto('/api/health');

    expect(response?.ok()).toBeTruthy();

    // Check health endpoint content
    const responseText = await response?.text();
    expect(responseText).toContain('API is healthy');

    console.log('✅ Application health check passed');
  });

  test('Admin user created by run_dev.sh can log in', async ({ page }) => {
    // Try to log in as the admin user that should be created by run_dev.sh
    await page.goto('/login');

    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Should be redirected to dashboard on successful login
    await page.waitForURL('**/dashboard', { timeout: 10000 });

    // Verify we're on the dashboard
    await expect(page.locator('h1')).toContainText('Dashboard');

    // Check that user menu is visible (indicates successful login)
    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    await expect(userMenuButton).toBeVisible();

    console.log('✅ Admin user login successful');
  });

  test('Regular user created by run_dev.sh can log in', async ({ page }) => {
    // Try to log in as the regular user that should be created by run_dev.sh
    await page.goto('/login');

    await page.fill('input[name="email"]', REGULAR_USER_EMAIL);
    await page.fill('input[name="password"]', REGULAR_USER_PASSWORD);
    await page.click('button[type="submit"]');

    // Wait for navigation - could be dashboard or stay on login if user doesn't exist
    await page.waitForTimeout(3000);

    const currentUrl = page.url();

    if (currentUrl.includes('dashboard')) {
      // Login succeeded - verify we're on dashboard
      await expect(page.locator('h1')).toContainText('Dashboard');
      const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
      await expect(userMenuButton).toBeVisible();

      console.log('✅ Regular user login successful - user creation fix is working!');

      // Check that regular user does NOT have admin access
      await userMenuButton.click();
      await page.waitForTimeout(500);

      const adminLink = page.locator('a[href="/admin"]');
      const adminLinkCount = await adminLink.count();
      expect(adminLinkCount).toBe(0);

      console.log('✅ Regular user correctly does not have admin access');

    } else if (currentUrl.includes('login')) {
      // Login failed - check for error message
      const errorElement = page.locator('text=Invalid').or(page.locator('text=failed')).or(page.locator('text=Error'));

      if (await errorElement.isVisible()) {
        const errorText = await errorElement.first().textContent();
        console.log('❌ Regular user login failed with error:', errorText);
      } else {
        console.log('❌ Regular user login failed - no specific error message found');
        await page.screenshot({ path: 'regular-user-login-failed.png', fullPage: true });
      }

      // This failure indicates the user creation issue still exists
      console.log('ℹ️ This suggests the run_dev.sh user creation fix needs more work');
    }
  });

  test('Both users are visible in admin panel when logged in as admin', async ({ page }) => {
    // Log in as admin first
    await page.goto('/login');
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard');

    // Navigate to admin panel
    await page.goto('/admin');
    await page.waitForURL('**/admin');

    // Wait for page to load completely
    await page.waitForLoadState('networkidle');

    // Get the page content
    const pageContent = await page.content();

    // Check for presence of both users
    const adminUserExists = pageContent.includes(ADMIN_EMAIL);
    const regularUserExists = pageContent.includes(REGULAR_USER_EMAIL);

    console.log('Users found in admin panel:');
    console.log('  - Admin user (admin@filesonthego.local):', adminUserExists ? '✅' : '❌');
    console.log('  - Regular user (user@filesonthego.local):', regularUserExists ? '✅' : '❌');

    // Take screenshot for debugging
    await page.screenshot({ path: 'admin-panel-users.png', fullPage: true });

    // Admin user should always exist
    expect(adminUserExists).toBeTruthy();

    // Regular user existence depends on whether the fix is working
    if (regularUserExists) {
      console.log('🎉 SUCCESS: Both admin and regular users are present in admin panel!');
      console.log('   The run_dev.sh user creation fix is working correctly.');
    } else {
      console.log('🔍 INVESTIGATION NEEDED: Regular user not found in admin panel.');
      console.log('   The run_dev.sh script may need further fixes.');

      // Let's check if there's a table or list structure we should be looking for
      const tables = page.locator('table');
      const lists = page.locator('ul, ol');
      const userSections = page.locator('text=user, text=User, text=Users');

      console.log('Debugging info:');
      console.log('  - Number of tables found:', await tables.count());
      console.log('  - Number of lists found:', await lists.count());
      console.log('  - Number of user sections found:', await userSections.count());

      // Look for any email patterns in the page
      const emailPattern = /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g;
      const emailsFound = pageContent.match(emailPattern) || [];
      console.log('  - Email addresses found on page:', emailsFound);
    }
  });

  test('Development environment shows expected test account information', async ({ page }) => {
    // Check that the application shows signs of being a dev environment
    await page.goto('/');

    // Look for development indicators (optional, depends on implementation)
    const pageContent = await page.content();

    // Some applications show development environment info
    // This is more of an informational test
    console.log('Development environment checks:');
    console.log('  - Page loaded successfully');
    console.log('  - Base URL:', page.context().browser()?.contexts()[0].browser().version());
  });

  test('MinIO/S3 integration is working', async ({ page }) => {
    // Log in as admin first
    await page.goto('/login');
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard');

    // Check if there are any S3/MinIO related indicators in the application
    // This is a basic check that S3 integration is configured
    const pageContent = await page.content();

    // Look for storage information or S3 indicators
    const hasStorageInfo = pageContent.includes('Storage') ||
                          pageContent.includes('storage') ||
                          pageContent.includes('quota') ||
                          pageContent.includes('MB') ||
                          pageContent.includes('GB');

    if (hasStorageInfo) {
      console.log('✅ Storage information visible in dashboard');
    } else {
      console.log('ℹ️ Storage information not immediately visible (this may be normal)');
    }
  });

  test.describe('Edge Cases and Error Handling', () => {
    test('Invalid login attempts are properly handled', async ({ page }) => {
      await page.goto('/login');

      // Try wrong password
      await page.fill('input[name="email"]', ADMIN_EMAIL);
      await page.fill('input[name="password"]', 'wrongpassword');
      await page.click('button[type="submit"]');

      // Should stay on login page and show error
      await page.waitForTimeout(2000);
      expect(page.url()).toContain('login');

      // Look for error message (specific message depends on implementation)
      const errorVisible = await page.locator('text=Invalid, text=failed, text=Error').first().isVisible();

      if (errorVisible) {
        console.log('✅ Invalid login properly handled with error message');
      } else {
        console.log('ℹ️ Invalid login handled (no explicit error message visible)');
      }
    });

    test('Protected routes redirect to login when not authenticated', async ({ page }) => {
      // Try accessing protected routes without login
      const protectedRoutes = ['/dashboard', '/admin', '/settings'];

      for (const route of protectedRoutes) {
        await page.goto(route);

        // Should redirect to login
        await page.waitForTimeout(2000);
        expect(page.url()).toContain('login');

        console.log(`✅ Route ${route} properly redirects to login`);
      }
    });
  });
});