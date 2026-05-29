// @ts-check
// Regression test for GitHub issue #30:
// "Unable to Input VS Score Manually"
//
// Root cause: r4r5Middleware in main.go reads the session from the wrong cookie
// name ("session-name" instead of "session"), so member_id is never present and
// every PATCH /api/vs-points/patch request returns 401 "Not authenticated" even
// for fully-authenticated admin users.
//
// This test uses the shared admin auth session (auth.json produced by global-setup).
// It fails (401) on the buggy code and passes (200) after the fix.

const { test, expect } = require('@playwright/test');

test.describe('VS Points manual save — issue #30', () => {
  test('PATCH /api/vs-points/patch returns 200 for authenticated admin (not 401)', async ({ page }) => {
    // Navigate to vs.html to activate the auth session in the browser context
    await page.goto('/vs.html');
    await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {});
    expect(page.url(), 'Expected to remain on vs.html (not redirected to login)').not.toContain('login.html');

    // Send the PATCH request using the same authenticated session as the browser.
    // member_id 999 is used as a sentinel — patchVSPoint does a blind upsert so
    // no member record is required; the important thing is that auth passes.
    const response = await page.request.patch('/api/vs-points/patch', {
      headers: { 'Content-Type': 'application/json' },
      data: {
        week_date: '2026-05-25',
        member_id: 999,
        day: 'monday',
        value: 42,
      },
    });

    const body = await response.text();

    // The primary assertion: must NOT be 401 "Not authenticated"
    expect(
      response.status(),
      `Expected 200 but got ${response.status()}: "${body.trim()}"\n` +
      'If this is 401 "Not authenticated", r4r5Middleware is reading the wrong session cookie.',
    ).toBe(200);
  });

  test('UI: editing a VS cell shows no "Not authenticated" error', async ({ page }) => {
    await page.goto('/vs.html');
    await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {});
    expect(page.url()).not.toContain('login.html');

    // Wait for the table body to render
    await page.waitForSelector('#vs2-tbody tr', { timeout: 8000 }).catch(() => {});

    // Skip if there are no editable cells (empty member list in this environment)
    const editableCell = page.locator('td.vs2-col-day.vs2-editable').first();
    const count = await editableCell.count();
    if (count === 0) {
      test.skip(true, 'No editable cells present — no members in DB. Covered by the API test above.');
      return;
    }

    // Intercept the PATCH so we can assert the response code
    const patchPromise = page.waitForResponse(
      r => r.url().includes('/api/vs-points/patch') && r.request().method() === 'PATCH',
      { timeout: 8000 },
    );

    // Click the first editable cell to enter edit mode
    await editableCell.click();

    // Fill in a new value and confirm
    const input = page.locator('td.vs2-col-day.vs2-editable input.vs2-edit-input').first();
    await input.fill('1');
    await page.locator('td.vs2-col-day.vs2-editable .vs2-edit-ok').first().click();

    const patchResponse = await patchPromise;
    const patchBody = await patchResponse.text();

    // Assert no auth error in the response
    expect(
      patchResponse.status(),
      `PATCH returned ${patchResponse.status()}: "${patchBody.trim()}"`,
    ).toBe(200);

    // Assert no error toast containing "Not authenticated" is visible
    const toastLocator = page.locator('.toast, [class*="toast"]');
    if (await toastLocator.count() > 0) {
      const toastText = await toastLocator.first().textContent();
      expect(toastText ?? '', 'Error toast should not contain "Not authenticated"').not.toContain('Not authenticated');
    }
  });
});
