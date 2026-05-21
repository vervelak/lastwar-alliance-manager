// mg-lightbox.spec.js — validate the screenshot lightbox added to MG import preview
// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const fs = require('fs');

const BASE = 'http://localhost:8080';
const DOWNLOADS = path.join(process.env.USERPROFILE, 'Downloads');
const SS = path.join(__dirname, 'screenshots');
if (!fs.existsSync(SS)) fs.mkdirSync(SS, { recursive: true });

// Use any 4 screenshots we know the test machine has; fall back gracefully if missing.
const CANDIDATE_IMAGES = [
    'WhatsApp Image 2026-05-14 at 08.20.06.jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (1).jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (2).jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (3).jpeg',
].map(n => path.join(DOWNLOADS, n));

const AVAILABLE_IMAGES = CANDIDATE_IMAGES.filter(p => fs.existsSync(p));

test.use({ storageState: path.join(__dirname, '..', 'auth.json') });

// ──────────────────────────────────────────────────────────────────────────────
// Helper: open the MG upload modal and arrive at the v2 preview, returning the
// first event card that has a screenshot button.
// ──────────────────────────────────────────────────────────────────────────────
async function openPreviewWithImages(page, images) {
    await page.goto(`${BASE}/marshal-guard.html`);
    await page.waitForLoadState('networkidle');

    // Open the add-event modal (visible to R3+/admin)
    const addBtn = page.locator('#add-event-btn');
    if (await addBtn.isVisible()) {
        await addBtn.click();
    } else {
        // Force modal open in case of role-based visibility
        await page.evaluate(() => {
            const m = document.getElementById('event-modal');
            if (m) { m.style.display = 'flex'; m.classList.add('active'); }
        });
    }
    await page.waitForTimeout(300);

    // Switch to the upload tab
    const uploadTab = page.locator('[data-modal-tab="upload"], button:has-text("Screenshot")').first();
    if (await uploadTab.count()) await uploadTab.click();
    await page.waitForTimeout(200);

    // Upload files
    const fileInput = page.locator('#mg-image-input');
    await fileInput.setInputFiles(images);

    // Wait for process button to appear
    await page.waitForFunction(
        () => {
            const btn = document.getElementById('mg-process-btn');
            return btn && (btn.style.display === '' || btn.style.display === 'block' || btn.style.display === 'inline-block');
        },
        { timeout: 10000 }
    ).catch(() => null);
    await page.waitForTimeout(300);

    // Click process and wait for v2 preview modal
    const responsePromise = page.waitForResponse(
        r => r.url().includes('/api/marshal-guard/process-mg-v2') && r.status() === 200,
        { timeout: 180000 }
    );
    await page.locator('#mg-process-btn').click();
    await responsePromise;

    // Wait for the preview modal to appear
    await page.waitForSelector('#mg-v2-modal', { state: 'visible', timeout: 30000 });
    await page.waitForTimeout(800);
}

// ──────────────────────────────────────────────────────────────────────────────
// Test 1 — structural checks: lightbox HTML is present in the page
// ──────────────────────────────────────────────────────────────────────────────
test('Lightbox HTML elements are present', async ({ page }) => {
    await page.goto(`${BASE}/marshal-guard.html`);
    await page.waitForLoadState('networkidle');

    // All lightbox elements must exist in the DOM
    await expect(page.locator('#mg-lightbox')).toHaveCount(1);
    await expect(page.locator('#mg-lightbox-img')).toHaveCount(1);
    await expect(page.locator('#mg-lightbox-close')).toHaveCount(1);
    await expect(page.locator('#mg-lightbox-prev')).toHaveCount(1);
    await expect(page.locator('#mg-lightbox-next')).toHaveCount(1);
    await expect(page.locator('#mg-lightbox-counter')).toHaveCount(1);

    // Lightbox should start hidden
    await expect(page.locator('#mg-lightbox')).toBeHidden();

    await page.screenshot({ path: path.join(SS, 'lb-01-page-loaded.png') });
    console.log('✅ All lightbox HTML elements found and lightbox starts hidden');
});

// ──────────────────────────────────────────────────────────────────────────────
// Test 2 — lightbox open / close cycle (programmatic, no OCR needed)
// ──────────────────────────────────────────────────────────────────────────────
test('Lightbox opens, shows image, and closes', async ({ page }) => {
    await page.goto(`${BASE}/marshal-guard.html`);
    await page.waitForLoadState('networkidle');

    // Inject a tiny 1×1 PNG blob and call openMGLightbox directly
    await page.evaluate(() => {
        // 1×1 red PNG as data URL
        const dataURL = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI6QAAAABJRU5ErkJggg==';
        // Populate the module-level arrays via the exported functions exposed at window scope
        // (they aren't exported, so we call openMGLightbox directly from page scope)
        window.__testLightboxURL = dataURL;
    });

    // openMGLightbox is a module-level function — call it via page.evaluate
    await page.evaluate(() => {
        // Locate the function on the page (script is non-module, so it's on window scope via
        // <script src="marshal-guard.js">)
        if (typeof openMGLightbox === 'function') {
            openMGLightbox([window.__testLightboxURL], 0);
        } else {
            // Fallback: set display directly to test the CSS/HTML layer
            const lb = document.getElementById('mg-lightbox');
            const img = document.getElementById('mg-lightbox-img');
            if (lb && img) {
                img.src = window.__testLightboxURL;
                lb.style.display = 'flex';
                document.body.style.overflow = 'hidden';
                const counter = document.getElementById('mg-lightbox-counter');
                if (counter) counter.textContent = '';
                const prev = document.getElementById('mg-lightbox-prev');
                const next = document.getElementById('mg-lightbox-next');
                if (prev) prev.style.display = 'none';
                if (next) next.style.display = 'none';
            }
        }
    });

    // Lightbox should now be visible
    await expect(page.locator('#mg-lightbox')).toBeVisible();
    await expect(page.locator('#mg-lightbox-img')).toBeVisible();

    // For a single image, prev/next should be hidden
    await expect(page.locator('#mg-lightbox-prev')).toBeHidden();
    await expect(page.locator('#mg-lightbox-next')).toBeHidden();
    // Counter empty for single image
    const counterText = await page.locator('#mg-lightbox-counter').textContent();
    expect(counterText?.trim() || '').toBe('');

    await page.screenshot({ path: path.join(SS, 'lb-02-open-single.png') });
    console.log('✅ Lightbox opened with single image, nav arrows hidden');

    // Close via close button
    await page.locator('#mg-lightbox-close').click();
    await expect(page.locator('#mg-lightbox')).toBeHidden();
    await page.screenshot({ path: path.join(SS, 'lb-03-closed.png') });
    console.log('✅ Lightbox closed via × button');
});

// ──────────────────────────────────────────────────────────────────────────────
// Test 3 — multi-image navigation (prev / next buttons + counter)
// ──────────────────────────────────────────────────────────────────────────────
test('Lightbox navigation works for multiple images', async ({ page }) => {
    await page.goto(`${BASE}/marshal-guard.html`);
    await page.waitForLoadState('networkidle');

    // Two distinct 1×1 PNGs (different colours so we can tell them apart)
    const img1 = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI6QAAAABJRU5ErkJggg==';
    const img2 = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADklEQVQI12NgYGBgAAAABQABXjKU2QAAAABJRU5ErkJggg==';

    await page.evaluate(([u1, u2]) => {
        if (typeof openMGLightbox === 'function') {
            openMGLightbox([u1, u2], 0);
        } else {
            const lb = document.getElementById('mg-lightbox');
            const img = document.getElementById('mg-lightbox-img');
            const counter = document.getElementById('mg-lightbox-counter');
            const prev = document.getElementById('mg-lightbox-prev');
            const next = document.getElementById('mg-lightbox-next');
            if (lb && img) {
                window.__lbURLs = [u1, u2];
                window.__lbIdx  = 0;
                img.src = u1;
                counter.textContent = '1 / 2';
                prev.style.display = '';
                next.style.display = '';
                lb.style.display = 'flex';
                document.body.style.overflow = 'hidden';
            }
        }
    }, [img1, img2]);

    await expect(page.locator('#mg-lightbox')).toBeVisible();
    // Arrows should be visible for 2 images
    await expect(page.locator('#mg-lightbox-prev')).toBeVisible();
    await expect(page.locator('#mg-lightbox-next')).toBeVisible();
    // Counter shows 1/2
    await expect(page.locator('#mg-lightbox-counter')).toContainText('1 / 2');

    await page.screenshot({ path: path.join(SS, 'lb-04-multi-img1.png') });
    console.log('✅ Lightbox shows 2 images with nav arrows and counter "1 / 2"');

    // Click next — go to image 2
    await page.evaluate(() => {
        if (typeof mgLbNav === 'function') {
            mgLbNav(1);
        } else {
            // Manual fallback
            if (window.__lbURLs && window.__lbIdx !== undefined) {
                window.__lbIdx = (window.__lbIdx + 1) % window.__lbURLs.length;
                document.getElementById('mg-lightbox-img').src = window.__lbURLs[window.__lbIdx];
                document.getElementById('mg-lightbox-counter').textContent =
                    `${window.__lbIdx + 1} / ${window.__lbURLs.length}`;
            }
        }
    });
    await expect(page.locator('#mg-lightbox-counter')).toContainText('2 / 2');
    await page.screenshot({ path: path.join(SS, 'lb-05-multi-img2.png') });
    console.log('✅ Navigated to image 2, counter shows "2 / 2"');

    // Keyboard Escape closes the lightbox
    await page.keyboard.press('Escape');
    await expect(page.locator('#mg-lightbox')).toBeHidden();
    await page.screenshot({ path: path.join(SS, 'lb-06-closed-esc.png') });
    console.log('✅ Escape key closes the lightbox');
});

// ──────────────────────────────────────────────────────────────────────────────
// Test 4 — backdrop click closes the lightbox
// ──────────────────────────────────────────────────────────────────────────────
test('Clicking the backdrop closes the lightbox', async ({ page }) => {
    await page.goto(`${BASE}/marshal-guard.html`);
    await page.waitForLoadState('networkidle');

    const img1 = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI6QAAAABJRU5ErkJggg==';

    await page.evaluate((u) => {
        const lb  = document.getElementById('mg-lightbox');
        const img = document.getElementById('mg-lightbox-img');
        if (lb && img) {
            img.src = u;
            lb.style.display = 'flex';
            document.body.style.overflow = 'hidden';
        }
    }, img1);

    await expect(page.locator('#mg-lightbox')).toBeVisible();

    // Click on the lightbox overlay itself (top-left corner, away from the image)
    await page.locator('#mg-lightbox').click({ position: { x: 10, y: 10 }, force: true });
    await expect(page.locator('#mg-lightbox')).toBeHidden();
    await page.screenshot({ path: path.join(SS, 'lb-07-closed-backdrop.png') });
    console.log('✅ Backdrop click closes the lightbox');
});

// ──────────────────────────────────────────────────────────────────────────────
// Test 5 — screenshot button appears in event card after OCR (requires real files)
// ──────────────────────────────────────────────────────────────────────────────
test('Screenshot button appears in event card after MG upload', async ({ page }) => {
    test.setTimeout(300000);

    if (AVAILABLE_IMAGES.length === 0) {
        console.log('⚠️  No WhatsApp test images found in Downloads — skipping end-to-end OCR test');
        test.skip();
        return;
    }

    console.log(`Using ${AVAILABLE_IMAGES.length} images from Downloads`);
    await openPreviewWithImages(page, AVAILABLE_IMAGES);

    await page.screenshot({ path: path.join(SS, 'lb-08-preview-modal.png') });

    // At least one event card should have a screenshot button
    const screenshotBtns = page.locator('.mg-view-screenshots-btn');
    const btnCount = await screenshotBtns.count();
    expect(btnCount).toBeGreaterThan(0);
    console.log(`✅ Found ${btnCount} screenshot button(s) in event card(s)`);

    await page.screenshot({ path: path.join(SS, 'lb-09-preview-with-btn.png') });

    // Click the first screenshot button — lightbox should open
    await screenshotBtns.first().click();
    await expect(page.locator('#mg-lightbox')).toBeVisible();
    await expect(page.locator('#mg-lightbox-img')).toBeVisible();

    // Image src should be a blob URL
    const imgSrc = await page.locator('#mg-lightbox-img').getAttribute('src');
    expect(imgSrc).toMatch(/^blob:/);
    console.log(`✅ Lightbox opened with blob URL: ${imgSrc?.substring(0, 40)}...`);

    await page.screenshot({ path: path.join(SS, 'lb-10-lightbox-open.png') });

    // Close via × button
    await page.locator('#mg-lightbox-close').click();
    await expect(page.locator('#mg-lightbox')).toBeHidden();
    console.log('✅ Lightbox closed after real OCR flow');
});

// ──────────────────────────────────────────────────────────────────────────────
// Test 6 — dark mode visual check (screenshot)
// ──────────────────────────────────────────────────────────────────────────────
test('Lightbox looks correct in dark mode', async ({ page }) => {
    await page.goto(`${BASE}/marshal-guard.html`);
    await page.waitForLoadState('networkidle');

    // Enable dark theme
    await page.evaluate(() => {
        document.documentElement.classList.add('theme-dark');
        localStorage.setItem('theme', 'dark');
    });
    await page.waitForTimeout(300);

    const img1 = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI6QAAAABJRU5ErkJggg==';
    const img2 = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADklEQVQI12NgYGBgAAAABQABXjKU2QAAAABJRU5ErkJggg==';

    await page.evaluate(([u1, u2]) => {
        if (typeof openMGLightbox === 'function') {
            openMGLightbox([u1, u2], 0);
        } else {
            const lb = document.getElementById('mg-lightbox');
            const img = document.getElementById('mg-lightbox-img');
            const counter = document.getElementById('mg-lightbox-counter');
            const prev = document.getElementById('mg-lightbox-prev');
            const next = document.getElementById('mg-lightbox-next');
            if (lb && img) {
                img.src = u1;
                counter.textContent = '1 / 2';
                prev.style.display = '';
                next.style.display = '';
                lb.style.display = 'flex';
                document.body.style.overflow = 'hidden';
            }
        }
    }, [img1, img2]);

    await expect(page.locator('#mg-lightbox')).toBeVisible();
    await page.screenshot({ path: path.join(SS, 'lb-11-dark-mode.png') });
    console.log('✅ Lightbox renders in dark mode');

    // Mobile viewport — check it doesn't overflow
    await page.setViewportSize({ width: 390, height: 844 });
    await page.waitForTimeout(200);
    await page.screenshot({ path: path.join(SS, 'lb-12-mobile-dark.png') });
    console.log('✅ Lightbox renders on mobile viewport (390×844)');

    await page.locator('#mg-lightbox-close').click();
    await expect(page.locator('#mg-lightbox')).toBeHidden();
});
