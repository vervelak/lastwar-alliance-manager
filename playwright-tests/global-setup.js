// Global setup: logs in once and saves auth state to auth.json
// @ts-check
const { chromium } = require('@playwright/test');
const path = require('path');

const BASE = 'http://localhost:8080';
const AUTH_FILE = path.join(__dirname, 'auth.json');

module.exports = async function globalSetup() {
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();

  const password = process.env.TEST_PASSWORD || 'admin123';

  await page.goto(`${BASE}/login.html`);
  await page.fill('#username', 'admin');
  await page.fill('#password', password);
  await page.click('#login-btn');
  await page.waitForURL(url => !url.href.includes('login.html'), { timeout: 10000 });

  // In a fresh environment the admin account has must_change_password=true, which
  // blocks all API calls. Call change-password via page.evaluate so the request
  // runs inside the browser with the correct session cookie, then waits for the
  // Set-Cookie response to be applied before saving storageState.
  await page.evaluate(async (pwd) => {
    await fetch('/api/change-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ current_password: pwd, new_password: pwd }),
      credentials: 'same-origin',
    });
  }, password);

  // Save cookies and localStorage to reuse in tests
  await context.storageState({ path: AUTH_FILE });
  await browser.close();
};
