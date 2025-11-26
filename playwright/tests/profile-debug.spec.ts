import { test, expect } from '@playwright/test';

test.describe('Profile Debug Test', () => {
  const REGULAR_USER_EMAIL = 'user@filesonthego.local';
  const REGULAR_USER_PASSWORD = 'user1234';

  test('Debug profile page navigation', async ({ page }) => {
    console.log('Starting profile debug test...');

    // Log in as regular user
    await page.goto('/login');
    await page.fill('input[name="email"]', REGULAR_USER_EMAIL);
    await page.fill('input[name="password"]', REGULAR_USER_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard', { timeout: 10000 });

    console.log('✅ Logged in successfully');

    // Check current URL and page content
    console.log('Current URL after login:', page.url());

    // Try direct navigation to profile
    console.log('Navigating directly to /profile...');
    await page.goto('/profile');

    // Wait a bit for any redirects
    await page.waitForTimeout(3000);

    console.log('Current URL after profile navigation:', page.url());

    // Check page content
    const pageContent = await page.content();
    console.log('Page title:', await page.title());

    // Look for any h1 elements
    const h1Elements = await page.locator('h1').all();
    console.log('Number of h1 elements found:', h1Elements.length);

    for (let i = 0; i < h1Elements.length; i++) {
      const text = await h1Elements[i].textContent();
      console.log(`  h1[${i}]: "${text}"`);
    }

    // Look for any form elements
    const formElements = await page.locator('form').all();
    console.log('Number of form elements found:', formElements.length);

    // Look for input elements
    const inputElements = await page.locator('input').all();
    console.log('Number of input elements found:', inputElements.length);

    // Take screenshot for debugging
    await page.screenshot({ path: 'profile-debug-screenshot.png', fullPage: true });
    console.log('Screenshot saved to profile-debug-screenshot.png');

    // Check if we got redirected to login (authentication issue)
    if (page.url().includes('login')) {
      console.log('❌ We were redirected to login - authentication issue!');
    }

    // Look for error messages
    const errorSelectors = [
      '.error',
      '.alert',
      '[class*="error"]',
      '[class*="danger"]'
    ];

    for (const selector of errorSelectors) {
      const elements = await page.locator(selector).all();
      if (elements.length > 0) {
        console.log(`Found ${elements.length} error element(s) with selector ${selector}`);
        for (let i = 0; i < elements.length; i++) {
          const text = await elements[i].textContent();
          console.log(`  Error[${i}]: "${text}"`);
        }
      }
    }

    // Try the settings page first to see if that works
    console.log('Trying to navigate to settings...');
    await page.goto('/settings');
    await page.waitForTimeout(3000);

    console.log('Current URL after settings navigation:', page.url());
    console.log('Page title on settings:', await page.title());

    const settingsH1Elements = await page.locator('h1').all();
    console.log('Number of h1 elements on settings:', settingsH1Elements.length);

    for (let i = 0; i < settingsH1Elements.length; i++) {
      const text = await settingsH1Elements[i].textContent();
      console.log(`  Settings h1[${i}]: "${text}"`);
    }

    // Look for profile links on settings page
    const profileLinks = await page.locator('a[href="/profile"]').all();
    console.log('Number of profile links found on settings:', profileLinks.length);

    for (let i = 0; i < profileLinks.length; i++) {
      const text = await profileLinks[i].textContent();
      console.log(`  Profile link[${i}]: "${text}"`);
    }
  });
});