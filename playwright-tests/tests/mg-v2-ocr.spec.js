// @ts-check
// Test the v2 OCR pipeline (process-mg-v2 endpoint).
// Sends all screenshots for one event group and validates the response.
const { test, expect } = require('@playwright/test');
const path = require('path');
const fs = require('fs');

const BASE = 'http://localhost:8080';
const DL = 'C:\\Users\\verve\\Downloads';

test.use({ storageState: path.join(__dirname, '..', 'auth.json') });

const groups = {
  '08.20.06': [
    'WhatsApp Image 2026-05-14 at 08.20.06.jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (1).jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (2).jpeg',
    'WhatsApp Image 2026-05-14 at 08.20.06 (3).jpeg',
  ],
  '08.21.15': [
    'WhatsApp Image 2026-05-14 at 08.21.15.jpeg',
    'WhatsApp Image 2026-05-14 at 08.21.15 (1).jpeg',
    'WhatsApp Image 2026-05-14 at 08.21.15 (2).jpeg',
    'WhatsApp Image 2026-05-14 at 08.21.15 (3).jpeg',
  ],
  '08.22.00': [
    'WhatsApp Image 2026-05-14 at 08.22.00.jpeg',
    'WhatsApp Image 2026-05-14 at 08.22.00 (1).jpeg',
  ],
  '08.22.01': [
    'WhatsApp Image 2026-05-14 at 08.22.01.jpeg',
    'WhatsApp Image 2026-05-14 at 08.22.01 (1).jpeg',
  ],
};

function formatDamage(d) {
  if (d >= 1e9) return (d / 1e9).toFixed(2) + 'G';
  if (d >= 1e6) return (d / 1e6).toFixed(2) + 'M';
  return String(d);
}

for (const [groupName, files] of Object.entries(groups)) {
  test(`MG v2 OCR — group ${groupName} (${files.length} images)`, async () => {
    console.log(`\n=== Group ${groupName} (${files.length} images) ===`);

    // Read session cookie so we can authenticate the native fetch call.
    const authState = JSON.parse(fs.readFileSync(path.join(__dirname, '..', 'auth.json'), 'utf8'));
    const sessionCookie = authState.cookies.find(c => c.name === 'session');
    const cookieHeader = sessionCookie ? `session=${sessionCookie.value}` : '';

    // Use native fetch+FormData (Node 18+) to send multiple files under the same key.
    const form = new FormData();
    for (const fileName of files) {
      const imgPath = path.join(DL, fileName);
      expect(fs.existsSync(imgPath), `File exists: ${fileName}`).toBeTruthy();
      const buffer = fs.readFileSync(imgPath);
      form.append('images[]', new Blob([buffer], { type: 'image/jpeg' }), fileName);
    }

    const fetchResponse = await fetch(`${BASE}/api/marshal-guard/process-mg-v2`, {
      method: 'POST',
      headers: { cookie: cookieHeader },
      body: form,
    });

    if (!fetchResponse.ok) {
      const body = await fetchResponse.text();
      console.log(`ERROR ${fetchResponse.status}: ${body}`);
    }

    expect(fetchResponse.ok, 'API returned non-200').toBeTruthy();

    const events = await fetchResponse.json();
    console.log(`\nEvents returned: ${events.length}`);

    expect(Array.isArray(events), 'Response should be array').toBeTruthy();
    expect(events.length, 'Should have at least one event').toBeGreaterThan(0);

    for (const ev of events) {
      console.log(`\n📅 event_date: ${ev.event_date || '(EMPTY)'}`);
      console.log(`   top_player_name: ${ev.top_player_name || '(empty)'}`);
      console.log(`   top_player_damage_str: ${ev.top_player_damage_str || '(empty)'}`);
      console.log(`   rows: ${ev.rows ? ev.rows.length : 0}`);

      if (ev.rows) {
        for (const row of ev.rows) {
          const tag = row.alliance_tag ? `[${row.alliance_tag}]` : '[???]';
          const dmg = row.damage_str || formatDamage(row.damage) || 'N/A';
          const member = row.member_id
            ? (row.graveyard_match ? `⚰️ ${row.member_name}` : `✅ ${row.member_name}`)
            : '❌ no match';
          console.log(`   #${String(row.rank).padStart(2)}: ${tag}${(row.name || '(empty)').padEnd(22)} dmg=${dmg.padEnd(10)} ${member}`);
        }
      }
    }

    // Assertions on the first (and likely only) event.
    const ev = events[0];
    expect(ev.event_date, 'event_date must not be empty').toBeTruthy();
    expect(ev.event_date, 'event_date must not be "unknown"').not.toBe('unknown');
    expect(ev.rows.length, 'Should have rows').toBeGreaterThan(3);
  });
}

// Also test sending a single image to check date detection independently.
test('MG v2 OCR — single image date detection', async ({ request }) => {
  const fileName = 'WhatsApp Image 2026-05-14 at 08.20.06.jpeg';
  const imgPath = path.join(DL, fileName);
  expect(fs.existsSync(imgPath)).toBeTruthy();

  const response = await request.post(`${BASE}/api/marshal-guard/process-mg-v2`, {
    multipart: {
      'images[]': {
        name: fileName,
        mimeType: 'image/jpeg',
        buffer: fs.readFileSync(imgPath),
      },
    },
  });

  expect(response.status()).toBe(200);
  const events = await response.json();

  console.log('\n--- Single image response ---');
  console.log(JSON.stringify(events, null, 2));

  expect(events.length).toBeGreaterThan(0);
  const ev = events[0];
  console.log(`event_date: ${ev.event_date || '(EMPTY)'}`);
  console.log(`top_player_name: ${ev.top_player_name || '(empty)'}`);

  // This assertion will tell us if date detection is working.
  expect(ev.event_date, 'event_date must be captured from the screenshot').toBeTruthy();
});
