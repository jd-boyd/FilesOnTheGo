import { test, expect } from '@playwright/test';

test.describe('Admin Link Functionality', () => {
  const ADMIN_EMAIL = 'admin@filesonthego.local';
  const ADMIN_PASSWORD = 'admin123';

  test.beforeEach(async ({ page }) => {
    // Navigate to the login page
    await page.goto('/login');
  });

  test('Admin link appears in header after login as admin user', async ({ page }) => {
    // Log in as admin
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL('**/dashboard');

    // Look for the user menu button (it has a user avatar with specific styling)
    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    await expect(userMenuButton).toBeVisible();

    // Click to open the dropdown menu
    await userMenuButton.click();

    // Wait for dropdown to be visible
    await page.waitForTimeout(500); // Small delay for Alpine.js to show dropdown

    // Check that Admin Panel link is visible in the dropdown
    const adminLink = page.locator('a[href="/admin"]');
    await expect(adminLink).toBeVisible();

    // Verify the link text
    await expect(adminLink).toContainText('Admin Panel');

    // Verify it has the correct icon
    const adminIcon = adminLink.locator('svg');
    await expect(adminIcon).toBeVisible();
  });

  test('Admin link navigation works correctly', async ({ page }) => {
    // Log in as admin
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL('**/dashboard');

    // Open user menu and click Admin link
    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    await userMenuButton.click();

    await page.waitForTimeout(500);

    const adminLink = page.locator('a[href="/admin"]');
    await adminLink.click();

    // Should navigate to admin page
    await page.waitForURL('**/admin');

    // Verify we're on the admin page by checking for admin content
    await expect(page.locator('h1')).toContainText('Admin Panel');
  });

  test('Admin link is not present for regular user', async ({ page }) => {
    // Log in as regular user
    await page.fill('input[name="email"]', 'user@filesonthego.local');
    await page.fill('input[name="password"]', 'user123');
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL('**/dashboard');

    // Open user menu
    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    await userMenuButton.click();

    await page.waitForTimeout(500);

    // Admin link should NOT be visible for regular users
    const adminLink = page.locator('a[href="/admin"]');
    await expect(adminLink).not.toBeVisible();
  });

  test('Direct access to /admin requires authentication', async ({ page }) => {
    // Try to access admin page without being logged in
    await page.goto('/admin');

    // Should be redirected to login page
    await page.waitForURL('**/login');
  });

  test('Admin panel shows correct content after navigation', async ({ page }) => {
    // Log in as admin and navigate to admin panel
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    await page.waitForURL('**/dashboard');

    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    await userMenuButton.click();

    await page.waitForTimeout(500);

    const adminLink = page.locator('a[href="/admin"]');
    await adminLink.click();

    // Verify admin panel content
    await page.waitForURL('**/admin');
    await expect(page.locator('h1')).toContainText('Admin Panel');

    // Look for admin-specific sections
    await expect(page.locator('text=System Settings')).toBeVisible();
    await expect(page.locator('text=User Management')).toBeVisible();
  });
});