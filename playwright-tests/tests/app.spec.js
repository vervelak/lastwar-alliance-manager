// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const fs = require('fs');

const screenshotsDir = path.join(__dirname, '..', 'screenshots');
if (!fs.existsSync(screenshotsDir)) fs.mkdirSync(screenshotsDir, { recursive: true });

function collectConsoleErrors(page) {
  const messages = [];
  page.on('console', msg => {
    if (msg.type() !== 'error') return;
    const text = msg.text();
    const skip = ['favicon','ERR_BLOCKED_BY_CLIENT','401 (Unauthorized)','Failed to load resource'];
    if (!skip.some(p => text.includes(p))) messages.push(text);
  });
  return messages;
}

async function expectNavPresent(page) {
  await expect(page.locator('nav.nav-menu').first()).toBeVisible({ timeout: 6000 });
}

test.describe('Login page', () => {
  test.use({ storageState: { cookies: [], origins: [] } });
  test('renders form', async ({ page }) => {
    await page.goto('/login.html');
    await page.screenshot({ path: path.join(screenshotsDir, '00-login.png') });
    await expect(page.locator('#username')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('#login-btn')).toBeVisible();
    const box = await page.locator('.login-box').boundingBox();
    expect(box?.height ?? 0, '.login-box collapsed').toBeGreaterThan(100);
  });
  test('shows error on bad credentials', async ({ page }) => {
    await page.goto('/login.html');
    await page.fill('#username', 'wrong');
    await page.fill('#password', 'wrong');
    await page.click('#login-btn');
    await expect(page.locator('#error-message')).toBeVisible({ timeout: 5000 });
  });
  test('redirects after successful login', async ({ page }) => {
    await page.goto('/login.html');
    await page.fill('#username', 'admin');
    await page.fill('#password', 'admin123');
    await page.click('#login-btn');
    await page.waitForURL(url => !url.href.includes('login.html'), { timeout: 8000 });
    expect(page.url()).not.toContain('login.html');
  });
});

const PAGES = [
  { name: 'Members (index)', path: '/', slug: '01-members', keySelectors: ['.container, table, .member-card'] },
  {
    name: 'Train Schedule', path: '/train.html', slug: '02-train',
    keySelectors: ['.schedule-controls', '#week-display'],
    extraChecks: async (page) => {
      await expect(page.locator('#week-mode-bar')).toBeVisible();
      await expect(page.locator('#mode-win')).toBeChecked();
      await expect(page.locator('#lucky-draw-panel')).toBeHidden();
      await page.click('#mode-save');
      await expect(page.locator('#lucky-draw-panel')).toBeVisible({ timeout: 3000 });
      await page.click('#mode-win');
      await expect(page.locator('#lucky-draw-panel')).toBeHidden({ timeout: 3000 });
    },
  },
  { name: 'Awards', path: '/awards.html', slug: '03-awards', keySelectors: ['table, section, main'] },
  { name: 'Recommendations', path: '/recommendations.html', slug: '04-recommendations', keySelectors: ['main, section'] },
  { name: 'Conduct Reports', path: '/conduct.html', slug: '05-conduct', keySelectors: ['main, section'] },
  { name: 'Rankings', path: '/rankings.html', slug: '06-rankings', keySelectors: ['main, section, table'] },
  { name: 'Storm', path: '/storm.html', slug: '07-storm', keySelectors: ['main, section'] },
  { name: 'VS Points', path: '/vs.html', slug: '08-vs', keySelectors: ['main, section, table'] },
  { name: 'VS Compliance', path: '/vs-compliance.html', slug: '09-vs-compliance', keySelectors: ['main, section, table'] },
  { name: 'Upload', path: '/upload.html', slug: '10-upload', keySelectors: ['select, form, main'] },
  {
    name: 'Settings', path: '/settings.html', slug: '11-settings',
    keySelectors: ['.form-section, form, section'],
    extraChecks: async (page) => {
      const btn = page.locator('#save-rotation-btn');
      if (await btn.count() > 0) await expect(btn).toBeVisible();
    },
  },
  { name: 'Admin', path: '/admin.html', slug: '12-admin', keySelectors: ['main, section, .container'] },
  { name: 'Graveyard', path: '/graveyard.html', slug: '13-graveyard', keySelectors: ['main, section, table'] },
  {
    name: 'Profile', path: '/profile.html', slug: '14-profile',
    keySelectors: ['.pf-header, .pf-top-row, main, section, .container'],
    extraChecks: async (page) => {
      // Score card and radar canvas should appear once a member is linked
      const memberSection = page.locator('#pf-member-section');
      const accountSection = page.locator('#pf-account-section');
      // At minimum the account section is always present
      await expect(accountSection).toBeVisible({ timeout: 8000 });
      // If member linked, member section should be visible
      const isMemberVisible = await memberSection.isVisible();
      if (isMemberVisible) {
        await expect(page.locator('#pf-radar')).toBeVisible();
        await expect(page.locator('#pf-score-table')).toBeVisible();
        await expect(page.locator('#pf-total-score')).not.toHaveText('—');
      }
    },
  },
  {
    name: 'Recruit', path: '/recruit.html', slug: '15-recruit',
    keySelectors: ['main, section, .form-section'],
    extraChecks: async (page) => {
      await expect(page.locator('#status-filter')).toBeVisible();
      await expect(page.locator('#applicant-list')).toBeVisible();
      // add button visible because we're logged in as admin (officer)
      await expect(page.locator('#add-applicant-btn')).toBeVisible();
    },
  },
  {
    name: 'Participation', path: '/participation.html', slug: '16-participation',
    keySelectors: ['.pt-summary-row, main, .container'],
    extraChecks: async (page) => {
      await expect(page.locator('#pt-summary-row')).toBeVisible({ timeout: 8000 });
      await expect(page.locator('#pt-total')).not.toHaveText('—', { timeout: 10000 });
      await expect(page.locator('#pt-tbody')).toBeVisible({ timeout: 8000 });
      // Toolbar controls present
      await expect(page.locator('#pt-search')).toBeVisible();
    },
  },
  {
    name: 'Leadership', path: '/leadership.html', slug: '17-leadership',
    keySelectors: ['.ld-cards, main, .container'],
    extraChecks: async (page) => {
      // Admin-only: should not redirect to login
      expect(page.url()).not.toContain('login.html');
      await expect(page.locator('#ld-summary-cards')).toBeVisible({ timeout: 8000 });
      await expect(page.locator('#ld-total-members')).not.toHaveText('—', { timeout: 10000 });
      // Charts rendered
      await expect(page.locator('#ld-rank-doughnut')).toBeVisible();
      await expect(page.locator('#ld-rank-score-bar')).toBeVisible();
      // Tables or empty messages present
      const promoteTbody = page.locator('#ld-promote-tbody');
      const promoteEmpty = page.locator('#ld-promote-empty');
      const hasRows = await promoteTbody.locator('tr').count() > 0;
      const hasEmpty = await promoteEmpty.isVisible();
      expect(hasRows || hasEmpty, 'Promotion section has neither rows nor empty message').toBeTruthy();
      // Full leaderboard: rows visible if data present, otherwise table still renders
      const fullRowCount = await page.locator('#ld-full-tbody tr').count();
      if (fullRowCount > 0) {
        await expect(page.locator('#ld-full-tbody tr').first()).toBeVisible({ timeout: 6000 });
        // Search filter works
        await page.fill('#ld-search', 'zzzznotexist');
        await expect(page.locator('#ld-full-tbody tr')).toHaveCount(0);
        await page.fill('#ld-search', '');
        await expect(page.locator('#ld-full-tbody tr').first()).toBeVisible();
      }
    },
  },
];

for (const pg of PAGES) {
  test.describe(pg.name, () => {
    test('loads without JS errors and nav is present', async ({ page }) => {
      const errors = collectConsoleErrors(page);
      await page.goto(pg.path);
      expect(page.url(), pg.name + ' redirected to login').not.toContain('login.html');
      await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
      await page.screenshot({ path: path.join(screenshotsDir, pg.slug + '-' + pg.name.replace(/[\s()]/g,'_') + '.png') });
      await expectNavPresent(page);
      let anyVisible = false;
      for (const sel of pg.keySelectors) {
        const el = page.locator(sel).first();
        if (await el.count() > 0) {
          const box = await el.boundingBox();
          if (box && box.height > 0) { anyVisible = true; break; }
        }
      }
      expect(anyVisible, 'No visible selectors on ' + pg.name).toBeTruthy();
      expect(errors, 'JS errors on ' + pg.name + ':\n' + errors.join('\n')).toHaveLength(0);
    });
    if (pg.extraChecks) {
      test('extra feature checks', async ({ page }) => {
        await page.goto(pg.path);
        await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
        await pg.extraChecks(page);
      });
    }
  });
}

test.describe('CSS layout checks', () => {
  test('stylesheet loads (200)', async ({ page }) => {
    let ok = false;
    page.on('response', r => { if (r.url().includes('styles.css') && r.status() === 200) ok = true; });
    await page.goto('/');
    await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
    expect(ok, 'styles.css returned non-200').toBeTruthy();
  });

  test('CSS custom properties resolve', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
    for (const prop of ['--accent-primary','--accent-secondary','--card-bg','--border-color']) {
      const val = await page.evaluate(p => getComputedStyle(document.documentElement).getPropertyValue(p).trim(), prop);
      expect(val, prop + ' is empty').not.toBe('');
    }
  });

  test('nav renders correctly', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
    const nav = page.locator('nav.nav-menu');
    await expect(nav).toBeVisible();
    const box = await nav.boundingBox();
    expect(box?.width ?? 0, 'nav zero width').toBeGreaterThan(50);
    expect(box?.height ?? 0, 'nav zero height').toBeGreaterThan(200);
    expect(await nav.locator('a.nav-link').count(), 'too few nav links').toBeGreaterThan(5);
    await page.screenshot({ path: path.join(screenshotsDir, '90-nav-desktop.png') });
  });

  test('mobile viewport no major overflow', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto('/');
    await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
    await page.screenshot({ path: path.join(screenshotsDir, '91-mobile.png') });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 5);
    if (overflow) console.warn('HorizontalOverflow on mobile 375px');
  });

  test('train week-mode bar height > 0', async ({ page }) => {
    await page.goto('/train.html');
    await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
    const bar = page.locator('#week-mode-bar');
    await expect(bar).toBeVisible();
    const box = await bar.boundingBox();
    expect(box?.height ?? 0, '#week-mode-bar collapsed').toBeGreaterThan(20);
    await page.screenshot({ path: path.join(screenshotsDir, '92-train-week-mode.png') });
  });

  test('no visible button has zero dimensions', async ({ page }) => {
    for (const pg of PAGES) {
      await page.goto(pg.path);
      await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => {});
      const buttons = page.locator('button:visible');
      const count = await buttons.count();
      for (let i = 0; i < count; i++) {
        const box = await buttons.nth(i).boundingBox();
        if (box) {
          expect(box.height, 'Button on ' + pg.name + ' zero height').toBeGreaterThan(0);
          expect(box.width, 'Button on ' + pg.name + ' zero width').toBeGreaterThan(0);
        }
      }
    }
  });
});

test.describe('API smoke tests', () => {
  let cookieHeader = '';
  test.beforeAll(async ({ browser }) => {
    const ctx = await browser.newContext({ storageState: path.join(__dirname, '..', 'auth.json') });
    const cookies = await ctx.cookies();
    cookieHeader = cookies.map(c => c.name + '=' + c.value).join('; ');
    await ctx.close();
  });
  async function apiGet(request, url) {
    return request.get('http://localhost:8080' + url, { headers: { Cookie: cookieHeader } });
  }
  test('GET /api/members returns array', async ({ request }) => {
    const res = await apiGet(request, '/api/members');
    expect(res.status()).toBe(200);
    expect(Array.isArray(await res.json())).toBeTruthy();
  });
  test('GET /api/settings returns 200', async ({ request }) => {
    expect((await apiGet(request, '/api/settings')).status()).toBe(200);
  });
  test('GET /api/settings/backup-rotation has order array', async ({ request }) => {
    const res = await apiGet(request, '/api/settings/backup-rotation');
    expect(res.status()).toBe(200);
    expect(Array.isArray((await res.json()).order)).toBeTruthy();
  });
  test('GET /api/settings/train-week-mode returns win or save', async ({ request }) => {
    const res = await apiGet(request, '/api/settings/train-week-mode');
    expect(res.status()).toBe(200);
    expect(['win','save']).toContain((await res.json()).mode);
  });
  test('GET /api/train-schedules returns 200', async ({ request }) => {
    expect((await apiGet(request, '/api/train-schedules')).status()).toBe(200);
  });
  test('GET /api/awards returns 200', async ({ request }) => {
    expect((await apiGet(request, '/api/awards')).status()).toBe(200);
  });
  test('GET /api/rankings returns rankings array', async ({ request }) => {
    const res = await apiGet(request, '/api/rankings');
    expect(res.status()).toBe(200);
    expect(Array.isArray((await res.json()).rankings)).toBeTruthy();
  });
  test('GET /api/applicants returns array', async ({ request }) => {
    const res = await apiGet(request, '/api/applicants');
    expect(res.status()).toBe(200);
    expect(Array.isArray(await res.json())).toBeTruthy();
  });
  test('POST /api/applicants creates and DELETE removes', async ({ request }) => {
    const createRes = await request.post('http://localhost:8080/api/applicants', {
      headers: { 'Content-Type': 'application/json', Cookie: cookieHeader },
      data: { name: 'TestApplicant_PW', power: 3000000, rank: 'R2', status: 'pending' },
    });
    expect(createRes.status()).toBe(201);
    const created = await createRes.json();
    expect(created.id).toBeGreaterThan(0);
    expect(created.name).toBe('TestApplicant_PW');

    // Status update
    const statusRes = await request.put(`http://localhost:8080/api/applicants/${created.id}/status`, {
      headers: { 'Content-Type': 'application/json', Cookie: cookieHeader },
      data: { status: 'on_trial', trial_end_date: '2099-12-31' },
    });
    expect(statusRes.status()).toBe(200);
    expect((await statusRes.json()).status).toBe('on_trial');

    // Cleanup
    const delRes = await request.delete(`http://localhost:8080/api/applicants/${created.id}`, {
      headers: { Cookie: cookieHeader },
    });
    expect(delRes.status()).toBe(200);
  });
  test('GET /api/settings has min_power field', async ({ request }) => {
    const res = await apiGet(request, '/api/settings');
    expect(res.status()).toBe(200);
    const s = await res.json();
    expect(typeof s.min_power).toBe('number');
    expect(typeof s.min_hq_level).toBe('number');
  });
});
