import { test, expect } from '@playwright/test';

test.describe('Debug Admin Link - Template Data Test', () => {
  const ADMIN_EMAIL = 'admin@filesonthego.local';
  const ADMIN_PASSWORD = 'admin123';

  test('Debug template data and page content', async ({ page }) => {
    // Navigate to the login page
    await page.goto('/login');

    // Log in as admin
    await page.fill('input[name="email"]', ADMIN_EMAIL);
    await page.fill('input[name="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL('**/dashboard');

    // Check the page content for any debugging info
    console.log('Page URL:', page.url());

    // Look for any admin-related content in the page
    const pageContent = await page.content();
    console.log('Page contains "Admin":', pageContent.includes('Admin'));
    console.log('Page contains "admin":', pageContent.includes('admin'));
    console.log('Page contains "IsAdmin":', pageContent.includes('IsAdmin'));

    // Look for settings data
    const hasSettings = pageContent.includes('Settings');
    console.log('Page contains "Settings":', hasSettings);

    // Look for dropdown menu structure
    const hasDropdown = pageContent.includes('dropdown') || pageContent.includes('role="menu"');
    console.log('Page has dropdown structure:', hasDropdown);

    // Try to find user menu button
    const userMenuButton = page.locator('button:has(div[class*="rounded-full"])');
    const buttonExists = await userMenuButton.count() > 0;
    console.log('User menu button exists:', buttonExists);

    if (buttonExists) {
      // Get the button's parent to see its HTML structure
      const buttonHTML = await userMenuButton.innerHTML();
      console.log('User menu button HTML snippet:', buttonHTML.substring(0, 200));

      // Click the button and wait for dropdown
      await userMenuButton.click();
      await page.waitForTimeout(1000); // Wait longer for Alpine.js

      // Get page content after clicking
      const pageAfterClick = await page.content();
      console.log('After click - Page contains "/admin":', pageAfterClick.includes('/admin'));
      console.log('After click - Page contains "Admin Panel":', pageAfterClick.includes('Admin Panel'));

      // Look specifically for any link href="/admin"
      const adminLinks = await page.locator('a[href="/admin"]').count();
      console.log('Number of admin links found:', adminLinks);

      // If still not found, let's examine the dropdown HTML structure
      if (adminLinks === 0) {
        // Find any dropdown menus
        const dropdowns = page.locator('[role="menu"], .dropdown, [x-show]');
        const dropdownCount = await dropdowns.count();
        console.log('Number of dropdown elements found:', dropdownCount);

        for (let i = 0; i < Math.min(dropdownCount, 3); i++) {
          const dropdownHTML = await dropdowns.nth(i).innerHTML();
          console.log(`Dropdown ${i} HTML:`, dropdownHTML.substring(0, 300));
        }
      }
    }

    // Take screenshot for visual inspection
    await page.screenshot({ path: 'debug-template-data.png', fullPage: true });
    console.log('Screenshot saved as debug-template-data.png');
  });
});