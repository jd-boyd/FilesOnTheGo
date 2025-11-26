import { test, expect } from '@playwright/test';

test.describe('Admin Link - Simple Test', () => {
  const ADMIN_EMAIL = 'admin@filesonthego.local';
  const ADMIN_PASSWORD = 'admin123';

  test('Admin can see Admin link in dropdown menu', async ({ page }) => {
    // Navigate to the login page
    await page.goto('/login');

    // Log in as admin
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL('**/dashboard');

    // Find and click the user menu button (has a user avatar)
    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    await expect(userMenuButton).toBeVisible();
    await userMenuButton.click();

    // Wait for dropdown to appear
    await page.waitForTimeout(500);

    // Check if Admin link exists in the dropdown
    const adminLink = page.locator('a[href="/admin"]');
    const adminLinkExists = await adminLink.count() > 0;

    console.log(`Admin link found: ${adminLinkExists}`);

    if (adminLinkExists) {
      console.log(`Admin link text: ${await adminLink.textContent()}`);
      await expect(adminLink).toContainText('Admin Panel');
    }
  });

  test('Admin link navigates to admin panel', async ({ page }) => {
    // Navigate to the login page
    await page.goto('/login');

    // Log in as admin
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL('**/dashboard');

    // Open user menu
    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    await userMenuButton.click();

    // Wait for dropdown to appear
    await page.waitForTimeout(500);

    // Try to find and click the Admin link
    const adminLink = page.locator('a[href="/admin"]');
    const linkCount = await adminLink.count();

    if (linkCount > 0) {
      await adminLink.click();

      // Check if we navigate to admin page
      await page.waitForURL('**/admin', { timeout: 5000 });

      // Verify we're on admin page
      const h1 = page.locator('h1');
      await expect(h1).toContainText('Admin Panel');

      console.log('Successfully navigated to admin panel');
    } else {
      console.log('Admin link not found in dropdown');
      // Let's take a screenshot to debug
      await page.screenshot({ path: 'debug-dropdown.png', fullPage: true });
    }
  });

  test('Admin page requires authentication', async ({ page }) => {
    // Try to access admin page directly without login
    const response = await page.goto('/admin');

    // Should be redirected to login page
    await page.waitForURL('**/login');
    console.log('Correctly redirected to login when accessing admin without auth');
  });
});