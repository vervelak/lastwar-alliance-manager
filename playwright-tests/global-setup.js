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

  await page.goto(`${BASE}/login.html`);
  await page.fill('#username', 'admin');
  await page.fill('#password', process.env.TEST_PASSWORD || 'admin123');
  await page.click('#login-btn');
  await page.waitForURL(url => !url.href.includes('login.html'), { timeout: 10000 });

  // Save cookies and localStorage to reuse in tests
  await context.storageState({ path: AUTH_FILE });
  await browser.close();
};
