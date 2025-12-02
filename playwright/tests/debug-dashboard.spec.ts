import { test, expect } from '@playwright/test';

test.describe('Debug Dashboard Page', () => {
  test('Debug what is actually on dashboard page', async ({ page }) => {
    // Log in as admin
    await page.goto('/login');
    await page.fill('input[name="email"]', 'admin@filesonthego.local');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard');

    // Take screenshot to see what's actually there
    await page.screenshot({ path: 'debug-dashboard.png', fullPage: true });

    // Get page content
    const pageContent = await page.content();
    console.log('=== PAGE CONTENT START ===');
    console.log(pageContent.substring(0, 2000)); // First 2000 chars
    console.log('=== PAGE CONTENT END ===');

    // Check for any buttons with "New Folder" text
    const newFolderByText = page.locator('button:has-text("New Folder")');
    const countByText = await newFolderByText.count();
    console.log(`Buttons with "New Folder" text: ${countByText}`);

    // Check for any buttons with onclick attribute
    const buttonsWithOnclick = page.locator('button[onclick]');
    const countOnclick = await buttonsWithOnclick.count();
    console.log(`Buttons with onclick: ${countOnclick}`);

    if (countOnclick > 0) {
      for (let i = 0; i < Math.min(countOnclick, 5); i++) {
        const button = buttonsWithOnclick.nth(i);
        const onclick = await button.getAttribute('onclick');
        console.log(`Button ${i} onclick: ${onclick}`);
      }
    }

    // Check for file-actions inclusion
    const hasFileActions = pageContent.includes('file-actions') || pageContent.includes('new-folder-modal');
    console.log(`Has file-actions: ${hasFileActions}`);

    // Check for script tags
    const scripts = page.locator('script[src*="file-browser.js"]');
    const scriptCount = await scripts.count();
    console.log(`File-browser.js script count: ${scriptCount}`);

    // Wait for 5 seconds to make sure everything is loaded
    await page.waitForTimeout(5000);

    // Take another screenshot after waiting
    await page.screenshot({ path: 'debug-dashboard-after-wait.png', fullPage: true });
  });
});