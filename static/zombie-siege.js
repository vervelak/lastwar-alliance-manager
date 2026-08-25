// Zombie Siege page logic
let canUpload = false; // R3, R4, R5, admin — can upload screenshots & import events
let isOfficer = false; // R4, R5, admin — can edit/delete events
let isAdmin = false;
let selectedFiles = [];
let ocrResult = null; // single DesertStormOCRResult preview
let zsAllMembers = [];

async function checkAuth() {
    try {
        const res = await fetch('/api/check-auth');
        const data = await res.json();
        if (!data.authenticated) { window.location.href = '/login.html'; return false; }
        if (data.must_change_password) { window.location.href = '/profile.html?must_change_password=1'; return false; }

        let display = `👤 ${data.username}`;
        if (data.rank) display += ` (${data.rank})`;
        const usernameDisplay = document.getElementById('username-display');
        if (usernameDisplay) {
            usernameDisplay.textContent = display;
            usernameDisplay.addEventListener('click', toggleUserDropdown);
        }

        const logoutBtn = document.getElementById('dropdown-logout-btn');
        if (logoutBtn) logoutBtn.addEventListener('click', handleLogout);

        document.addEventListener('click', (event) => {
            const dropdown = document.getElementById('user-dropdown-menu');
            const btn = document.getElementById('username-display');
            if (dropdown && btn && !btn.contains(event.target) && !dropdown.contains(event.target)) {
                dropdown.classList.remove('show');
            }
        });

        isAdmin = data.is_admin || false;
        const rank = (data.rank || '').toUpperCase();
        canUpload = isAdmin || rank === 'R3' || rank === 'R4' || rank === 'R5';
        isOfficer = isAdmin || rank === 'R4' || rank === 'R5';

        if (canUpload) {
            document.querySelectorAll('.uploader-only').forEach(el => el.style.display = '');
        }
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
        const res = await fetch('/api/zombie-siege');
        const events = await res.json();
        renderEventList(events);
    } catch (e) {
        document.getElementById('event-list').innerHTML = '<p class="empty">⚠️ Failed to load events.</p>';
    }
}

function renderEventList(events) {
    const el = document.getElementById('event-list');
    if (!events || events.length === 0) {
        el.innerHTML = '<p class="empty">🧟 No Zombie Siege events recorded yet.</p>';
        return;
    }
    let html = `<table class="rk-table"><thead><tr>
        <th>Date</th><th>Total Waves</th><th># Players</th><th>Top Defender</th><th>Top Waves</th>`;
    if (isOfficer) html += '<th>Actions</th>';
    html += '</tr></thead><tbody>';
    for (const ev of events) {
        html += `<tr class="mg-event-row" data-id="${ev.id}">
            <td>${ev.event_date}</td>
            <td>${formatWaves(ev.total_waves)}</td>
            <td>${ev.participant_count}</td>
            <td>${escapeHtml(ev.top_wave_defender)}</td>
            <td>${formatWaves(ev.top_waves)}</td>`;
        if (isOfficer) {
            html += `<td class="list-actions">
                <button class="edit-schedule-btn" onclick="event.stopPropagation(); deleteEvent(${ev.id})" title="Delete">🗑️</button>
            </td>`;
        }
        html += '</tr>';
    }
    html += '</tbody></table>';
    el.innerHTML = html;

    el.querySelectorAll('.mg-event-row').forEach(row => {
        row.style.cursor = 'pointer';
        row.addEventListener('click', () => viewEventDetail(parseInt(row.dataset.id)));
    });
}

async function viewEventDetail(id) {
    try {
        const res = await fetch('/api/zombie-siege/' + id);
        const ev = await res.json();
        document.getElementById('detail-modal-title').textContent = `🧟 Event: ${ev.event_date}`;
        document.getElementById('detail-summary').innerHTML = `
            <p><strong>Total Alliance Waves:</strong> ${formatWaves(ev.total_waves)}</p>
            ${ev.notes ? '<p><strong>Notes:</strong> ' + escapeHtml(ev.notes) + '</p>' : ''}
            <p><strong>Participants:</strong> ${ev.participants.length}</p>`;
        let thtml = `<table class="rk-table"><thead><tr>
            <th>#</th><th>Player</th><th>Waves</th><th>Member</th>
        </tr></thead><tbody>`;
        for (const p of ev.participants) {
            const matched = p.member_id ? `✅ ${escapeHtml(p.member_name)}` : '❌ Unmatched';
            thtml += `<tr${p.rank_in_event === 1 ? ' class="mg-mvp-row"' : ''}>
                <td>${p.rank_in_event === 1 ? '🏆' : p.rank_in_event}</td>
                <td>${escapeHtml(p.name_snapshot)}${p.alliance_tag ? ' <span class="text-muted">[' + escapeHtml(p.alliance_tag) + ']</span>' : ''}</td>
                <td>${formatWaves(p.waves)}</td>
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
    if (!confirm('Delete this Zombie Siege event and all its participants?')) return;
    try {
        const res = await fetch('/api/zombie-siege/' + id, { method: 'DELETE' });
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
        const res = await fetch('/api/zombie-siege/member-stats');
        const stats = await res.json();
        renderMemberStats(stats);
    } catch {
        document.getElementById('stats-list').innerHTML = '<p class="empty">⚠️ Failed to load stats.</p>';
    }
}

function renderMemberStats(stats) {
    const el = document.getElementById('stats-list');
    if (!stats || stats.length === 0) {
        el.innerHTML = '<p class="empty">🧟 No participation data yet.</p>';
        return;
    }
    let html = `<table class="rk-table" id="zs-stats-table"><thead><tr>
        <th>Member</th><th>Rank</th><th>Events</th><th>Total Waves</th><th>Avg Rank</th><th>Best Waves</th>
    </tr></thead><tbody>`;
    for (const s of stats) {
        html += `<tr>
            <td>${escapeHtml(s.member_name)}</td>
            <td>${escapeHtml(s.member_rank)}</td>
            <td>${s.event_count}</td>
            <td>${formatWaves(s.total_waves)}</td>
            <td>${s.avg_rank.toFixed(1)}</td>
            <td>${formatWaves(s.best_waves)}</td>
        </tr>`;
    }
    html += '</tbody></table>';
    el.innerHTML = html;
}

// ---- Upload / OCR flow ----
function initUpload() {
    const dropZone = document.getElementById('zs-drop-zone');
    const fileInput = document.getElementById('zs-image-input');
    const processBtn = document.getElementById('zs-process-btn');
    const clearBtn = document.getElementById('zs-clear-btn');

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
    selectedFiles = selectedFiles.concat(files).slice(0, 40);
    renderFilePreview();
}

function renderFilePreview() {
    const gallery = document.getElementById('zs-preview-gallery');
    const container = document.getElementById('zs-preview-container');
    const dropContent = document.getElementById('zs-drop-content');
    const processBtn = document.getElementById('zs-process-btn');
    const countEl = document.getElementById('zs-files-count');

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
    document.getElementById('zs-image-input').value = '';
    renderFilePreview();
}

async function processScreenshots() {
    if (selectedFiles.length === 0) return;
    const processBtn = document.getElementById('zs-process-btn');
    const progressWrap = document.getElementById('zs-progress-wrap');
    const progressBar  = document.getElementById('zs-progress-bar');
    const progressLabel = document.getElementById('zs-progress-label');
    const progressTime  = document.getElementById('zs-progress-time');

    processBtn.disabled = true;
    processBtn.textContent = '⏳ Processing…';
    progressWrap.style.display = 'block';
    progressBar.style.width = '0%';
    progressLabel.textContent = `Processing ${selectedFiles.length} image${selectedFiles.length > 1 ? 's' : ''}…`;

    let pct = 0;
    const startMs = Date.now();
    const timer = setInterval(() => {
        pct += (90 - pct) * 0.12;
        progressBar.style.width = pct.toFixed(1) + '%';
        progressTime.textContent = ((Date.now() - startMs) / 1000).toFixed(0) + 's';
    }, 300);

    try {
        const formData = new FormData();
        selectedFiles.forEach(f => formData.append('images[]', f));

        const res = await fetch('/api/zombie-siege/process-screenshots', { method: 'POST', body: formData });
        if (!res.ok) { showToast('OCR processing failed', 'error'); return; }

        const result = await res.json();
        if (!result || !result.participants || result.participants.length === 0) {
            showToast('No players detected. Check screenshot format.', 'warning');
            return;
        }
        document.getElementById('event-modal').style.display = 'none';
        progressBar.style.width = '100%';
        setTimeout(() => { progressWrap.style.display = 'none'; }, 600);
        showPreview(result);
    } catch (e) {
        showToast('OCR processing failed: ' + e.message, 'error');
        progressWrap.style.display = 'none';
    } finally {
        clearInterval(timer);
        processBtn.disabled = false;
        processBtn.textContent = '🔍 Process Screenshots with OCR';
    }
}

// ---- OCR preview (single event) ----
async function loadZSMembers() {
    try {
        const res = await fetch('/api/members');
        if (!res.ok) return;
        const data = await res.json();
        zsAllMembers = (data || []).sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
        buildZSMemberDatalist();
    } catch { /* non-critical */ }
}

function buildZSMemberDatalist() {
    const dl = document.getElementById('zs-member-names');
    if (!dl) return;
    dl.innerHTML = zsAllMembers.map(m => `<option value="${escapeAttr(m.name)}">`).join('');
}

function buildZSMemberOptions(memberId) {
    let opts = '<option value="">— Unmatched —</option>';
    for (const m of zsAllMembers) {
        const nick = m.nickname ? ` [${m.nickname}]` : '';
        const sel = m.id === memberId ? ' selected' : '';
        opts += `<option value="${m.id}"${sel}>${escapeHtml(m.name)}${escapeHtml(nick)} (${escapeHtml(m.rank)})</option>`;
    }
    return opts;
}

function showPreview(result) {
    ocrResult = {
        event_date: result.event_date || '',
        total_waves: result.total_waves || 0,
        existing_event_id: result.existing_event_id || null,
        notes: '',
        rows: (result.participants || []).map(p => ({
            rank_in_event: p.rank_in_event,
            name_snapshot: p.name_snapshot || '',
            alliance_tag: p.alliance_tag || '',
            waves: p.waves || 0,
            member_id: p.member_id || null,
            member_name: p.member_name || '',
            warnings: p.warnings || [],
        })),
    };
    renderPreview();
    document.getElementById('zs-ocr-modal').style.display = 'flex';
}

function renderPreview() {
    const container = document.getElementById('zs-ocr-event');
    const overwrite = ocrResult.existing_event_id
        ? `<div class="info-banner info-banner--warning" style="margin-bottom:.5rem;">
               <div class="info-content"><div class="info-icon">⚠️</div>
               <div class="info-text">Event already exists for this date — importing will overwrite it.</div>
               </div></div>`
        : '';

    let memberRows = '';
    ocrResult.rows.forEach((row, rIdx) => {
        const isTop = row.rank_in_event === 1;
        const hasWarnings = (row.warnings || []).length > 0;
        const cls = [isTop ? 'mg-mvp-row' : '', hasWarnings ? 'zs-warn-row' : ''].filter(Boolean).join(' ');
        const warnBadge = hasWarnings
            ? `<span class="zs-warn-badge" title="${escapeAttr(row.warnings.join('; '))}">⚠️</span>`
            : '';
        const opts = buildZSMemberOptions(row.member_id);
        memberRows += `<tr${cls ? ` class="${cls}"` : ''}>
            <td class="mg-rank-col">${isTop ? '🏆' : row.rank_in_event}</td>
            <td class="mg-name-col">
                ${warnBadge}
                <input class="mg-cell-input" data-field="name_snapshot" data-row="${rIdx}"
                    value="${escapeAttr(row.name_snapshot)}" placeholder="Player name">
                <input class="mg-cell-input" data-field="alliance_tag" data-row="${rIdx}"
                    value="${escapeAttr(row.alliance_tag)}" placeholder="[TAG]" style="width:90px;">
            </td>
            <td class="mg-dmg-col">
                <input class="mg-cell-input" data-field="waves" data-row="${rIdx}"
                    value="${escapeAttr(String(row.waves))}" placeholder="0" style="width:110px;">
            </td>
            <td class="mg-name-col">
                <select class="mg-member-select" data-row="${rIdx}">${opts}</select>
            </td>
        </tr>`;
    });

    container.innerHTML = `
        <div class="mg-v2-card-header">
            <div class="mg-card-meta">
                <strong class="mg-event-date">📅 Zombie Siege ranking</strong>
            </div>
            <div class="mg-card-controls">
                <input type="date" id="zs-preview-date" value="${escapeAttr(ocrResult.event_date)}" title="Event date">
                <input type="text" id="zs-preview-notes" value="" placeholder="Notes (optional)" title="Event notes">
            </div>
        </div>
        ${overwrite}
        <div class="rk-table-wrapper" style="max-height:420px;overflow-y:auto;">
            <table class="rk-table">
                <thead><tr><th class="mg-rank-col">#</th><th>Player</th><th class="mg-dmg-col">Waves</th><th>Member</th></tr></thead>
                <tbody>${memberRows}</tbody>
            </table>
        </div>`;
}

// Sync inline edits back into ocrResult live data.
document.addEventListener('input', e => {
    const t = e.target;
    if (t.classList.contains('mg-cell-input')) {
        const rIdx = parseInt(t.dataset.row);
        const field = t.dataset.field;
        if (!isNaN(rIdx)) {
            ocrResult.rows[rIdx][field] = t.value;
            // User-corrected values are authoritative — clear OCR warnings.
            if (field === 'name_snapshot' || field === 'waves') {
                ocrResult.rows[rIdx].warnings = [];
                const tr = t.closest('tr');
                if (tr) {
                    tr.classList.remove('zs-warn-row');
                    tr.querySelector('.zs-warn-badge')?.remove();
                }
            }
        }
    }
    if (t.id === 'zs-preview-date') ocrResult.event_date = t.value;
    if (t.id === 'zs-preview-notes') ocrResult.notes = t.value;
});

document.addEventListener('change', e => {
    const t = e.target;
    if (t.classList.contains('mg-member-select')) {
        const rIdx = parseInt(t.dataset.row);
        if (!isNaN(rIdx)) {
            ocrResult.rows[rIdx].member_id = t.value ? parseInt(t.value) : null;
        }
    }
});

async function importEvent() {
    if (!ocrResult) return;
    if (!ocrResult.event_date) { showToast('Event date is required', 'warning'); return; }

    const participants = [];
    for (const row of ocrResult.rows) {
        const name = (row.name_snapshot || '').trim();
        const wavesStr = String(row.waves || '').trim();
        if (!name && !wavesStr) continue;
        participants.push({
            rank_in_event: row.rank_in_event,
            name_snapshot: name,
            alliance_tag: row.alliance_tag || '',
            waves: parseWaves(wavesStr),
            member_id: row.member_id || null,
        });
    }

    const totalWaves = participants.reduce((s, p) => s + (p.waves || 0), 0);

    const body = {
        event_date: ocrResult.event_date,
        total_waves: totalWaves,
        notes: ocrResult.notes || '',
        participants,
    };
    if (ocrResult.existing_event_id) body.overwrite_event_id = ocrResult.existing_event_id;

    try {
        const btn = document.getElementById('zs-ocr-import-btn');
        if (btn) { btn.disabled = true; btn.textContent = '⏳ Importing…'; }

        const res = await fetch('/api/zombie-siege/confirm', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        const data = await res.json();
        if (res.ok) {
            showToast(data.message || 'Event imported', 'success');
            document.getElementById('zs-ocr-modal').style.display = 'none';
            ocrResult = null;
            clearFiles();
            loadEvents();
            loadMemberStats();
        } else {
            showToast(data.message || 'Import failed', 'error');
            if (btn) { btn.disabled = false; btn.textContent = '✔ Import Event'; }
        }
    } catch (e) {
        showToast('Import failed: ' + e.message, 'error');
    }
}

// ---- Manual event creation ----
function addZSManualRow(name = '', waves = '') {
    const container = document.getElementById('zs-manual-participants');
    if (!container) return;
    const row = document.createElement('div');
    row.className = 'manual-participant-row';
    row.innerHTML =
        `<input type="text" class="zs-manual-name" list="zs-member-names" placeholder="Member name" value="${escapeAttr(name)}">` +
        `<input type="number" class="zs-manual-waves" min="0" placeholder="Waves" value="${escapeAttr(waves)}">` +
        `<button type="button" class="zs-manual-remove btn btn-secondary">✕</button>`;
    container.appendChild(row);
    row.querySelector('.zs-manual-remove').addEventListener('click', () => row.remove());
}

function collectZSManualParticipants() {
    const participants = [];
    document.querySelectorAll('#zs-manual-participants .manual-participant-row').forEach(row => {
        const name = row.querySelector('.zs-manual-name').value.trim();
        const waves = parseInt(row.querySelector('.zs-manual-waves').value, 10) || 0;
        if (name || waves) participants.push({ name_snapshot: name, waves });
    });
    return participants;
}

function initManualForm() {
    const addBtn = document.getElementById('zs-add-participant-btn');
    if (addBtn) addBtn.addEventListener('click', () => addZSManualRow());

    document.getElementById('event-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const date = document.getElementById('zs-date').value;
        if (!date) { showToast('Date is required', 'warning'); return; }

        try {
            const res = await fetch('/api/zombie-siege', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    event_date: date,
                    total_waves: parseInt(document.getElementById('zs-total-waves').value) || 0,
                    notes: document.getElementById('zs-notes').value,
                    participants: collectZSManualParticipants(),
                }),
            });
            if (res.ok) {
                showToast('Event created', 'success');
                document.getElementById('event-modal').style.display = 'none';
                document.getElementById('event-form').reset();
                document.getElementById('zs-manual-participants').innerHTML = '';
                loadEvents();
            } else {
                showToast('Failed to create event', 'error');
            }
        } catch { showToast('Failed to create event', 'error'); }
    });
}

// ---- Modal management ----
function initModals() {
    document.getElementById('add-event-btn').addEventListener('click', () => {
        document.getElementById('event-modal').style.display = 'flex';
    });

    document.querySelectorAll('.modal .close').forEach(btn => {
        btn.addEventListener('click', () => btn.closest('.modal').style.display = 'none');
    });

    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) modal.style.display = 'none';
        });
    });

    document.getElementById('zs-ocr-cancel-btn').addEventListener('click', () => {
        document.getElementById('zs-ocr-modal').style.display = 'none';
        ocrResult = null;
        clearFiles();
    });
    document.getElementById('zs-ocr-import-btn').addEventListener('click', importEvent);
}

// ---- Search filter ----
function initSearch() {
    const input = document.getElementById('stats-search');
    input.addEventListener('input', () => {
        const q = input.value.toLowerCase();
        const rows = document.querySelectorAll('#zs-stats-table tbody tr');
        rows.forEach(row => {
            const name = row.cells[0].textContent.toLowerCase();
            row.style.display = name.includes(q) ? '' : 'none';
        });
    });
}

// ---- Waves parsing/formatting ----
function parseWaves(s) {
    if (!s) return 0;
    const clean = String(s).replace(/[^0-9]/g, '');
    const n = parseInt(clean, 10);
    return isNaN(n) ? 0 : n;
}

function formatWaves(val) {
    const n = parseInt(val, 10);
    if (isNaN(n) || n === 0) return '0';
    return n.toLocaleString('en-US');
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

    await Promise.all([loadEvents(), loadMemberStats(), loadZSMembers()]);
});
