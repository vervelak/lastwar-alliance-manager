// mg-ui-review.spec.js — upload 12 WhatsApp photos and capture MG UI screenshots
const { test, expect } = require('@playwright/test');
const path = require('path');
const fs = require('fs');

const DOWNLOADS = path.join(process.env.USERPROFILE, 'Downloads');
const IMAGES = [
    'WhatsApp Image 2026-05-14 at 08.20.06.jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (1).jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (2).jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (3).jpeg',
    'WhatsApp Image 2026-05-14 at 08.21.15.jpeg',
    'WhatsApp Image 2026-05-14 at 08.21.15 (1).jpeg',
    'WhatsApp Image 2026-05-14 at 08.21.15 (2).jpeg',
    'WhatsApp Image 2026-05-14 at 08.21.15 (3).jpeg',
    'WhatsApp Image 2026-05-14 at 08.22.00.jpeg',
    'WhatsApp Image 2026-05-14 at 08.22.00 (1).jpeg',
    'WhatsApp Image 2026-05-14 at 08.22.01.jpeg',
    'WhatsApp Image 2026-05-14 at 08.22.01 (1).jpeg',
].map(n => path.join(DOWNLOADS, n));

const SS = path.join(__dirname, 'screenshots');
if (!fs.existsSync(SS)) fs.mkdirSync(SS, { recursive: true });

test('MG UI — upload 12 photos, capture light + dark previews', async ({ page }) => {
    test.setTimeout(300000); // 5 minutes for OCR
    await page.goto('http://localhost:8080/marshal-guard.html');
    await page.waitForLoadState('networkidle');

    // ── Light theme screenshot of empty state ──────────────────────────────
    await page.screenshot({ path: path.join(SS, 'mg-01-empty-light.png'), fullPage: true });

    // ── Switch to dark theme ───────────────────────────────────────────────
    const themeToggle = page.locator('#theme-toggle, [data-theme-toggle], .theme-toggle').first();
    if (await themeToggle.count()) {
        await themeToggle.click();
        await page.waitForTimeout(400);
    } else {
        // Try setting data-theme attribute directly
        await page.evaluate(() => {
            document.documentElement.setAttribute('data-theme', 'dark');
            localStorage.setItem('theme', 'dark');
        });
        await page.waitForTimeout(400);
    }
    await page.screenshot({ path: path.join(SS, 'mg-02-empty-dark.png'), fullPage: true });

    // Switch back to light
    await page.evaluate(() => {
        document.documentElement.setAttribute('data-theme', 'light');
        localStorage.setItem('theme', 'light');
    });
    await page.waitForTimeout(300);

    // ── Open MG modal (officer-only "Add Event" btn, revealed after JS role check) ──
    await page.waitForSelector('#add-event-btn', { state: 'visible', timeout: 10000 })
        .catch(() => null); // might not be officer — fall back
    const addBtn = page.locator('#add-event-btn');
    const isVisible = await addBtn.isVisible().catch(() => false);
    if (isVisible) {
        await addBtn.click();
    } else {
        // Force-show the modal directly if we lack officer role in the test session
        await page.evaluate(() => {
            const modal = document.getElementById('event-modal');
            if (modal) { modal.style.display = 'flex'; modal.classList.add('active'); }
        });
    }
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(SS, 'mg-03-modal-open.png') });

    // ── Navigate to Screenshot Upload tab ─────────────────────────────────
    const uploadTab = page.locator('[data-modal-tab="upload"], button:has-text("Screenshot")').first();
    if (await uploadTab.count()) {
        await uploadTab.click();
        await page.waitForTimeout(300);
    }

    // ── Upload the 12 images ───────────────────────────────────────────────
    const input = page.locator('#mg-image-input');
    await input.setInputFiles(IMAGES);
    // Wait until the process button becomes visible (files rendered in gallery)
    await page.waitForFunction(
        () => document.getElementById('mg-process-btn')?.style?.display !== 'none',
        { timeout: 8000 }
    );
    await page.waitForTimeout(400);
    await page.screenshot({ path: path.join(SS, 'mg-04-files-selected-light.png') });

    // Dark mode with files selected
    await page.evaluate(() => {
        document.documentElement.setAttribute('data-theme', 'dark');
        localStorage.setItem('theme', 'dark');
    });
    await page.waitForTimeout(300);
    await page.screenshot({ path: path.join(SS, 'mg-05-files-selected-dark.png') });
    await page.evaluate(() => {
        document.documentElement.setAttribute('data-theme', 'light');
        localStorage.setItem('theme', 'light');
    });

    // ── Click Process ──────────────────────────────────────────────────────
    await page.locator('#mg-process-btn').click();

    // Wait for OCR (up to 3 minutes)
    await page.waitForSelector('#mg-v2-modal', { state: 'visible', timeout: 180000 });
    await page.waitForTimeout(1000);

    // ── Preview modal — light ──────────────────────────────────────────────
    await page.screenshot({ path: path.join(SS, 'mg-06-preview-light.png'), fullPage: false });

    // Scroll down inside the preview modal to see more
    const modalBody = page.locator('#mg-v2-events-container');
    if (await modalBody.count()) {
        await modalBody.evaluate(el => el.scrollTop += 600);
        await page.waitForTimeout(300);
        await page.screenshot({ path: path.join(SS, 'mg-07-preview-light-scroll.png') });
        await modalBody.evaluate(el => el.scrollTop += 600);
        await page.waitForTimeout(300);
        await page.screenshot({ path: path.join(SS, 'mg-08-preview-light-scroll2.png') });
        await modalBody.evaluate(el => el.scrollTop = 0);
    }

    // ── Dark mode preview ──────────────────────────────────────────────────
    await page.evaluate(() => {
        document.documentElement.classList.add('theme-dark');
        localStorage.setItem('theme', 'dark');
    });
    await page.waitForTimeout(400);
    await page.screenshot({ path: path.join(SS, 'mg-09-preview-dark.png'), fullPage: false });

    if (await modalBody.count()) {
        await modalBody.evaluate(el => el.scrollTop += 600);
        await page.waitForTimeout(300);
        await page.screenshot({ path: path.join(SS, 'mg-10-preview-dark-scroll.png') });
    }

    console.log('Screenshots saved to:', SS);
});
