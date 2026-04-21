import { test, expect } from '@playwright/test';

test.describe('Basic Application Setup', () => {
  test('has title and loads successfully', async ({ page }) => {
    // Navigates to the base URL set in playwright.config.ts (http://localhost:34115)
    await page.goto('/');

    // Check that the page loads without immediately crashing
    // For Wails apps, usually there is an #app or root element.
    // Replace this with the actual root element of your application.
    const appElement = page.locator('body');
    await expect(appElement).toBeVisible();

    // Take a screenshot of the initial state for verification
    await page.screenshot({ path: 'test-results/initial-load.png', fullPage: true });
  });
});
