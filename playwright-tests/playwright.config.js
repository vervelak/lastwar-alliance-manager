// @ts-check
const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  // Exclude tests that require local Downloads assets not available in CI
  testIgnore: process.env.CI ? ['**/mg-ocr.spec.js', '**/vs-ocr.spec.js', '**/mg-ui-review.spec.js'] : [],
  fullyParallel: false,
  retries: 0,
  workers: 1,
  globalSetup: require.resolve('./global-setup'),
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'report' }]],
  use: {
    baseURL: 'http://localhost:8080',
    headless: true,
    viewport: { width: 1280, height: 900 },
    screenshot: 'on',
    video: 'off',
    trace: 'off',
    // Reuse auth session saved by global-setup
    storageState: require('path').join(__dirname, 'auth.json'),
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
