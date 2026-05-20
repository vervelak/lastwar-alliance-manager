// Marshal Guard page logic
let isOfficer = false;
let isAdmin = false;
let ocrPreviewData = null;
let selectedFiles = [];

async function checkAuth() {
    try {
        const res = await fetch('/api/check-auth');
        const data = await res.json();
        if (!data.authenticated) { window.location.href = '/login.html'; return false; }
        if (data.must_change_password) { window.location.href = '/profile.html?must_change_password=1'; return false; }

        let display = `👤 ${data.username}`;
        if (data.rank) display += ` (${data.rank})`;
        document.getElementById('username-display').textContent = display;

        isAdmin = data.is_admin || false;
        const rank = (data.rank || '').toUpperCase();
        isOfficer = isAdmin || rank === 'R4' || rank === 'R5';

        if (isOfficer) {
            document.querySelectorAll('.officer-only').forEach(el => el.style.display = '');
        }
        if (isAdmin) {
            const adminLink = document.getElementById('admin-nav-link');
            const gyLink = document.getElementById('graveyard-nav-link');
            if (adminLink) adminLink.style.display = 'block';
            if (gyLink) gyLink.style.display = 'block';
        }
        return true;
    } catch { return false; }
}

// ---- Tab switching ----
function initTabs() {
    document.querySelectorAll('.tab-btn[data-tab]').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.tab-btn[data-tab]').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
            btn.classList.add('active');
            document.getElementById('tab-' + btn.dataset.tab).classList.add('active');
        });
    });
    // Modal sub-tabs
    document.querySelectorAll('[data-modal-tab]').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('[data-modal-tab]').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.modal-tab-content').forEach(c => c.classList.remove('active'));
            btn.classList.add('active');
            document.getElementById('modal-tab-' + btn.dataset.modalTab).classList.add('active');
        });
    });
}

// ---- Event list ----
async function loadEvents() {
    try {
        const res = await fetch('/api/marshal-guard');
        const events = await res.json();
        renderEventList(events);
    } catch (e) {
        document.getElementById('event-list').innerHTML = '<p class="empty">⚠️ Failed to load events.</p>';
    }
}

function renderEventList(events) {
    const el = document.getElementById('event-list');
    if (!events || events.length === 0) {
        el.innerHTML = '<p class="empty">🛡️ No Marshal Guard events recorded yet.</p>';
        return;
    }
    let html = `<table class="rk-table"><thead><tr>
        <th>Date</th><th>Total Damage</th><th># Players</th><th>Top Dealer</th><th>Top Damage</th>`;
    if (isOfficer) html += '<th>Actions</th>';
    html += '</tr></thead><tbody>';
    for (const ev of events) {
        html += `<tr class="mg-event-row" data-id="${ev.id}">
            <td>${ev.event_date}</td>
            <td>${formatDamage(ev.total_alliance_damage)}</td>
            <td>${ev.participant_count}</td>
            <td>${escapeHtml(ev.top_damage_dealer)}</td>
            <td>${formatDamage(ev.top_damage)}</td>`;
        if (isOfficer) {
            html += `<td class="list-actions">
                <button class="edit-schedule-btn" onclick="event.stopPropagation(); deleteEvent(${ev.id})" title="Delete">🗑️</button>
            </td>`;
        }
        html += '</tr>';
    }
    html += '</tbody></table>';
    el.innerHTML = html;

    // Click row to view details
    el.querySelectorAll('.mg-event-row').forEach(row => {
        row.style.cursor = 'pointer';
        row.addEventListener('click', () => viewEventDetail(parseInt(row.dataset.id)));
    });
}

async function viewEventDetail(id) {
    try {
        const res = await fetch('/api/marshal-guard/' + id);
        const ev = await res.json();
        document.getElementById('detail-modal-title').textContent = `🛡️ Event: ${ev.event_date}`;
        document.getElementById('detail-summary').innerHTML = `
            <p><strong>Total Alliance Damage:</strong> ${formatDamage(ev.total_alliance_damage)}</p>
            ${ev.notes ? '<p><strong>Notes:</strong> ' + escapeHtml(ev.notes) + '</p>' : ''}
            <p><strong>Participants:</strong> ${ev.participants.length}</p>`;
        let thtml = `<table class="rk-table"><thead><tr>
            <th>#</th><th>Player</th><th>Damage</th><th>Attacks</th><th>Member</th>
        </tr></thead><tbody>`;
        for (const p of ev.participants) {
            const matched = p.member_id ? `✅ ${escapeHtml(p.member_name)}` : '❌ Unmatched';
            thtml += `<tr${p.rank_in_event === 1 ? ' class="mg-mvp-row"' : ''}>
                <td>${p.rank_in_event === 1 ? '🏆' : p.rank_in_event}</td>
                <td>${escapeHtml(p.name_snapshot)}${p.alliance_tag ? ' <span class="text-muted">[' + escapeHtml(p.alliance_tag) + ']</span>' : ''}</td>
                <td>${formatDamage(p.damage)}</td>
                <td>${p.attack_count != null ? p.attack_count : '—'}</td>
                <td>${matched}</td>
            </tr>`;
        }
        thtml += '</tbody></table>';
        document.getElementById('detail-participants').innerHTML = thtml;
        document.getElementById('detail-modal').style.display = 'flex';
    } catch (e) {
        showToast('Failed to load event details', 'error');
    }
}

async function deleteEvent(id) {
    if (!confirm('Delete this Marshal Guard event and all its participants?')) return;
    try {
        const res = await fetch('/api/marshal-guard/' + id, { method: 'DELETE' });
        if (res.ok) {
            showToast('Event deleted', 'success');
            loadEvents();
        } else {
            showToast('Failed to delete event', 'error');
        }
    } catch { showToast('Failed to delete event', 'error'); }
}

// ---- Member stats ----
async function loadMemberStats() {
    try {
        const res = await fetch('/api/marshal-guard/member-stats');
        const stats = await res.json();
        renderMemberStats(stats);
    } catch {
        document.getElementById('stats-list').innerHTML = '<p class="empty">⚠️ Failed to load stats.</p>';
    }
}

function renderMemberStats(stats) {
    const el = document.getElementById('stats-list');
    if (!stats || stats.length === 0) {
        el.innerHTML = '<p class="empty">🛡️ No participation data yet.</p>';
        return;
    }
    let html = `<table class="rk-table" id="mg-stats-table"><thead><tr>
        <th>Member</th><th>Rank</th><th>Events</th><th>Total Damage</th><th>Avg Rank</th><th>Best Damage</th>
    </tr></thead><tbody>`;
    for (const s of stats) {
        html += `<tr>
            <td>${escapeHtml(s.member_name)}</td>
            <td>${escapeHtml(s.member_rank)}</td>
            <td>${s.event_count}</td>
            <td>${formatDamage(s.total_damage)}</td>
            <td>${s.avg_rank.toFixed(1)}</td>
            <td>${formatDamage(s.best_damage)}</td>
        </tr>`;
    }
    html += '</tbody></table>';
    el.innerHTML = html;
}

// ---- Upload / OCR flow ----
function initUpload() {
    const dropZone = document.getElementById('mg-drop-zone');
    const fileInput = document.getElementById('mg-image-input');
    const processBtn = document.getElementById('mg-process-btn');
    const clearBtn = document.getElementById('mg-clear-btn');

    dropZone.addEventListener('click', (e) => {
        if (e.target.closest('button')) return;
        fileInput.click();
    });
    dropZone.addEventListener('dragover', (e) => { e.preventDefault(); dropZone.classList.add('dragover'); });
    dropZone.addEventListener('dragleave', () => dropZone.classList.remove('dragover'));
    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('dragover');
        handleFiles(e.dataTransfer.files);
    });
    fileInput.addEventListener('change', () => handleFiles(fileInput.files));
    clearBtn.addEventListener('click', clearFiles);
    processBtn.addEventListener('click', processScreenshots);
}

function handleFiles(fileList) {
    const files = Array.from(fileList).filter(f => f.type.startsWith('image/'));
    if (files.length === 0) return;
    selectedFiles = selectedFiles.concat(files).slice(0, 10);
    renderFilePreview();
}

function renderFilePreview() {
    const gallery = document.getElementById('mg-preview-gallery');
    const container = document.getElementById('mg-preview-container');
    const dropContent = document.getElementById('mg-drop-content');
    const processBtn = document.getElementById('mg-process-btn');
    const countEl = document.getElementById('mg-files-count');

    if (selectedFiles.length === 0) {
        container.style.display = 'none';
        dropContent.style.display = '';
        processBtn.style.display = 'none';
        return;
    }

    dropContent.style.display = 'none';
    container.style.display = 'block';
    processBtn.style.display = 'block';
    countEl.textContent = `${selectedFiles.length} file${selectedFiles.length > 1 ? 's' : ''} selected`;

    gallery.innerHTML = '';
    selectedFiles.forEach((file, i) => {
        const div = document.createElement('div');
        div.className = 'preview-item';
        const img = document.createElement('img');
        img.className = 'preview-img';
        img.src = URL.createObjectURL(file);
        const removeBtn = document.createElement('button');
        removeBtn.className = 'remove-file';
        removeBtn.textContent = '×';
        removeBtn.onclick = (e) => { e.stopPropagation(); selectedFiles.splice(i, 1); renderFilePreview(); };
        const nameEl = document.createElement('div');
        nameEl.className = 'file-name';
        nameEl.textContent = file.name;
        div.append(img, removeBtn, nameEl);
        gallery.appendChild(div);
    });
}

function clearFiles() {
    selectedFiles = [];
    document.getElementById('mg-image-input').value = '';
    renderFilePreview();
}

async function processScreenshots() {
    if (selectedFiles.length === 0) return;
    const processBtn = document.getElementById('mg-process-btn');
    processBtn.disabled = true;
    processBtn.textContent = '⏳ Processing…';

    try {
        const formData = new FormData();
        selectedFiles.forEach(f => formData.append('images[]', f));

        const res = await fetch('/api/marshal-guard/process-mg-v2', { method: 'POST', body: formData });
        if (!res.ok) { showToast('OCR processing failed', 'error'); return; }

        const events = await res.json();
        if (!events || events.length === 0) {
            showToast('No events detected. Check screenshot format.', 'warning');
            return;
        }
        document.getElementById('event-modal').style.display = 'none';
        showMGV2Preview(events);
    } catch (e) {
        showToast('OCR processing failed: ' + e.message, 'error');
    } finally {
        processBtn.disabled = false;
        processBtn.textContent = '🔍 Process Screenshots with OCR';
    }
}

// ─── V2 multi-event preview ────────────────────────────────────────────────

// Live data store for the preview — mutated by inline edits.
let mgV2Events = [];

function showMGV2Preview(events) {
    mgV2Events = events.map(ev => ({
        ...ev,
        notes: '',
        rows: (ev.rows || []).map(r => ({ ...r })),
    }));
    renderMGV2Events();
    document.getElementById('mg-v2-modal').style.display = 'flex';
}

function renderMGV2Events() {
    const container = document.getElementById('mg-v2-events');
    container.innerHTML = '';
    mgV2Events.forEach((ev, evIdx) => {
        const card = document.createElement('div');
        card.className = 'mg-v2-event-card';
        card.innerHTML = buildEventCardHTML(ev, evIdx);
        container.appendChild(card);
    });

    // Wire up per-event import buttons.
    container.querySelectorAll('[data-import-event]').forEach(btn => {
        btn.addEventListener('click', () => importSingleEvent(parseInt(btn.dataset.importEvent)));
    });
}

function buildEventCardHTML(ev, evIdx) {
    const overwrite = ev.existing_event_id
        ? `<div class="info-banner info-banner--warning" style="margin-bottom:.5rem;">
               <div class="info-content"><div class="info-icon">⚠️</div>
               <div class="info-text">Event already exists for this date — importing will overwrite it.</div>
               </div></div>`
        : '';

    const topRow = (ev.top_player_name || ev.top_player_damage_str)
        ? `<tr class="mg-v2-top-row">
               <td>🏆</td>
               <td>${escapeHtml(ev.top_player_name || '—')}</td>
               <td>${escapeHtml(ev.top_player_damage_str || '—')}</td>
               <td>–</td>
               <td class="text-muted" style="font-size:.8em;">top player</td>
           </tr>`
        : '';

    let memberRows = '';
    (ev.rows || []).forEach((row, rIdx) => {
        const isGap   = !row.name && !row.damage_str;         // rank gap
        const needsReview = isGap || !row.name_ok || !row.damage_ok;
        const rowClass = isGap ? 'mg-v2-gap-row' : (needsReview ? 'mg-v2-warn-row' : '');

        const nameCell = `<input class="mg-cell-input" data-ev="${evIdx}" data-row="${rIdx}" data-field="name"
                                 value="${escapeAttr(row.name || '')}"
                                 placeholder="[TAG]PlayerName" style="${!row.name_ok ? 'border-color:var(--warning,#f59e0b);' : ''}">`;
        const dmgCell  = `<input class="mg-cell-input mg-dmg-input" data-ev="${evIdx}" data-row="${rIdx}" data-field="damage_str"
                                 value="${escapeAttr(row.damage_str || '')}"
                                 placeholder="e.g. 15.20G" style="${!row.damage_ok ? 'border-color:var(--warning,#f59e0b);' : ''}">`;
        const status = isGap
            ? '<span class="badge badge-warn">gap</span>'
            : (row.rank_fixed
                ? '<span class="badge badge-info" title="Rank inferred from sequence">fixed</span>'
                : (needsReview
                    ? '<span class="badge badge-warn">review</span>'
                    : '<span class="badge badge-ok">✓</span>'));
        const member = row.member_id
            ? `<span class="text-muted" style="font-size:.8em;">✅ ${escapeHtml(row.member_name)}</span>`
            : '<span class="text-muted" style="font-size:.8em;">—</span>';

        memberRows += `<tr class="${rowClass}">
            <td>${row.rank}</td>
            <td>${nameCell}</td>
            <td>${dmgCell}</td>
            <td>${status}</td>
            <td>${member}</td>
        </tr>`;
    });

    return `
        <div class="mg-v2-card-header">
            <div>
                <strong>📅 ${escapeHtml(ev.event_date || 'Unknown date')}</strong>
                <span class="text-muted" style="margin-left:.5rem;font-size:.85em;">${ev.rows.length} player${ev.rows.length !== 1 ? 's' : ''}</span>
            </div>
            <button class="primary-btn" style="padding:.3rem .8rem;font-size:.85em;" data-import-event="${evIdx}">
                ✔ Import this event
            </button>
        </div>
        ${overwrite}
        <div class="form-group" style="display:flex;gap:1rem;align-items:center;flex-wrap:wrap;margin-bottom:.5rem;">
            <label style="margin:0;">Date:
                <input type="date" class="form-control mg-date-input" data-ev="${evIdx}"
                       value="${escapeAttr(ev.event_date || '')}" style="display:inline-block;width:auto;margin-left:.4rem;">
            </label>
            <label style="margin:0;">Notes:
                <input type="text" class="form-control mg-notes-input" data-ev="${evIdx}"
                       value="${escapeAttr(ev.notes || '')}" placeholder="optional"
                       style="display:inline-block;width:18rem;margin-left:.4rem;">
            </label>
        </div>
        <div class="rk-table-wrapper" style="max-height:400px;overflow-y:auto;">
            <table class="rk-table">
                <thead><tr><th>#</th><th>Player</th><th>Damage</th><th>Status</th><th>Member</th></tr></thead>
                <tbody>${topRow}${memberRows}</tbody>
            </table>
        </div>`;
}

// Sync inline edits back into mgV2Events live data.
document.addEventListener('input', e => {
    const t = e.target;
    if (t.classList.contains('mg-cell-input')) {
        const evIdx  = parseInt(t.dataset.ev);
        const rowIdx = parseInt(t.dataset.row);
        const field  = t.dataset.field;
        if (!isNaN(evIdx) && !isNaN(rowIdx)) {
            mgV2Events[evIdx].rows[rowIdx][field] = t.value;
        }
    }
    if (t.classList.contains('mg-date-input')) {
        const evIdx = parseInt(t.dataset.ev);
        if (!isNaN(evIdx)) mgV2Events[evIdx].event_date = t.value;
    }
    if (t.classList.contains('mg-notes-input')) {
        const evIdx = parseInt(t.dataset.ev);
        if (!isNaN(evIdx)) mgV2Events[evIdx].notes = t.value;
    }
});

async function importSingleEvent(evIdx) {
    const ev = mgV2Events[evIdx];
    if (!ev.event_date) { showToast('Event date is required', 'warning'); return; }

    const participants = [];
    // Top player as rank 1.
    if (ev.top_player_name || ev.top_player_damage) {
        const parsed = parseMGName(ev.top_player_name || '');
        participants.push({
            rank_in_event: 1,
            name_snapshot: parsed.name,
            alliance_tag:  parsed.tag,
            damage:        ev.top_player_damage || 0,
            attack_count:  null,
        });
    }
    for (const row of ev.rows) {
        const name  = (row.name || '').trim();
        const dmgStr = (row.damage_str || '').trim();
        if (!name && !dmgStr) continue; // skip empty gap rows
        const parsed = parseMGName(name);
        participants.push({
            rank_in_event: row.rank,
            name_snapshot: parsed.name,
            alliance_tag:  parsed.tag,
            damage:        parseMGDamageStr(dmgStr),
            attack_count:  null,
            member_id:     row.member_id || null,
        });
    }

    const totalDamage = participants.reduce((s, p) => s + (p.damage || 0), 0);

    const body = {
        event_date:  ev.event_date,
        total_damage: totalDamage,
        notes:        ev.notes || '',
        participants,
    };
    if (ev.existing_event_id) body.overwrite_event_id = ev.existing_event_id;

    try {
        const btn = document.querySelector(`[data-import-event="${evIdx}"]`);
        if (btn) { btn.disabled = true; btn.textContent = '⏳ Importing…'; }

        const res = await fetch('/api/marshal-guard/confirm', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        const data = await res.json();
        if (res.ok) {
            showToast(data.message || 'Event imported', 'success');
            // Remove imported event card.
            mgV2Events.splice(evIdx, 1);
            if (mgV2Events.length === 0) {
                document.getElementById('mg-v2-modal').style.display = 'none';
                clearFiles();
                loadEvents();
                loadMemberStats();
            } else {
                renderMGV2Events();
            }
        } else {
            showToast(data.message || 'Import failed', 'error');
            if (btn) { btn.disabled = false; btn.textContent = '✔ Import this event'; }
        }
    } catch (e) {
        showToast('Import failed: ' + e.message, 'error');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const importAllBtn = document.getElementById('mg-v2-import-all-btn');
    if (importAllBtn) {
        importAllBtn.addEventListener('click', async () => {
            // Import sequentially.
            const indices = mgV2Events.map((_, i) => i);
            for (let i = indices.length - 1; i >= 0; i--) {
                await importSingleEvent(0); // always import index 0 since array shrinks
            }
        });
    }
    document.getElementById('mg-v2-cancel-btn').addEventListener('click', () => {
        document.getElementById('mg-v2-modal').style.display = 'none';
    });
});

// ─── Damage / name parsing helpers ────────────────────────────────────────────

// parseMGName splits "[TAG]PlayerName" → { tag, name }.
function parseMGName(raw) {
    const m = raw.match(/^\[([A-Za-z0-9]{1,4})\]\s*(.+)$/);
    if (m) return { tag: m[1], name: m[2].trim() };
    return { tag: '', name: raw.trim() };
}

// parseMGDamageStr converts "27.35G" / "15.20M" / "8G" to an integer.
function parseMGDamageStr(s) {
    if (!s) return 0;
    // Accept "Total Damage: X.XXG" or just "X.XXG"
    const clean = s.replace(/Total Damage:\s*/i, '').trim();
    const m = clean.match(/^(\d+)(?:\.(\d{1,2}))?([GM])$/i);
    if (!m) return 0;
    const int  = parseInt(m[1], 10);
    const dec  = m[2] ? m[2].padEnd(2, '0') : '00';
    const unit = m[3].toUpperCase();
    const mult = unit === 'G' ? 1_000_000_000 : 1_000_000;
    return int * mult + parseInt(dec, 10) * (mult / 100);
}


// ---- Manual event creation ----
function initManualForm() {
    document.getElementById('event-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const date = document.getElementById('mg-date').value;
        if (!date) { showToast('Date is required', 'warning'); return; }

        try {
            const res = await fetch('/api/marshal-guard', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    event_date: date,
                    total_alliance_damage: parseInt(document.getElementById('mg-total-damage').value) || 0,
                    notes: document.getElementById('mg-notes').value,
                }),
            });
            if (res.ok) {
                showToast('Event created', 'success');
                document.getElementById('event-modal').style.display = 'none';
                document.getElementById('event-form').reset();
                loadEvents();
            } else {
                showToast('Failed to create event', 'error');
            }
        } catch { showToast('Failed to create event', 'error'); }
    });
}

// ---- Modal management ----
function initModals() {
    // Add event button
    document.getElementById('add-event-btn').addEventListener('click', () => {
        document.getElementById('event-modal').style.display = 'flex';
    });

    // Close buttons
    document.querySelectorAll('.modal .close').forEach(btn => {
        btn.addEventListener('click', () => btn.closest('.modal').style.display = 'none');
    });

    // Overlay click
    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) modal.style.display = 'none';
        });
    });
}

// ---- Search filter ----
function initSearch() {
    const input = document.getElementById('stats-search');
    input.addEventListener('input', () => {
        const q = input.value.toLowerCase();
        const rows = document.querySelectorAll('#mg-stats-table tbody tr');
        rows.forEach(row => {
            const name = row.cells[0].textContent.toLowerCase();
            row.style.display = name.includes(q) ? '' : 'none';
        });
    });
}

// ---- Damage formatting ----
function formatDamage(val) {
    if (!val || val === 0) return '0';
    if (val >= 1e9) return (val / 1e9).toFixed(2) + 'G';
    if (val >= 1e6) return (val / 1e6).toFixed(2) + 'M';
    if (val >= 1e3) return (val / 1e3).toFixed(1) + 'K';
    return val.toString();
}

function escapeAttr(str) {
    if (!str) return '';
    return str.replace(/&/g,'&amp;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

// ---- Init ----
document.addEventListener('DOMContentLoaded', async () => {
    const authed = await checkAuth();
    if (!authed) return;

    initTabs();
    initModals();
    initUpload();
    initManualForm();
    initSearch();

    await Promise.all([loadEvents(), loadMemberStats()]);
});
