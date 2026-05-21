// Marshal Guard page logic
let canUpload = false; // R3, R4, R5, admin — can upload screenshots & import events
let isOfficer = false; // R4, R5, admin — can edit/delete events
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
    selectedFiles = selectedFiles.concat(files).slice(0, 40);
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
    // Revoke any outstanding blob URLs so the browser can free the image memory.
    mgV2BlobURLs.forEach(u => URL.revokeObjectURL(u));
    mgV2BlobURLs = [];
    document.getElementById('mg-image-input').value = '';
    renderFilePreview();
}

async function processScreenshots() {
    if (selectedFiles.length === 0) return;
    const processBtn = document.getElementById('mg-process-btn');
    const progressWrap = document.getElementById('mg-progress-wrap');
    const progressBar  = document.getElementById('mg-progress-bar');
    const progressLabel = document.getElementById('mg-progress-label');
    const progressTime  = document.getElementById('mg-progress-time');

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

        const res = await fetch('/api/marshal-guard/process-mg-v2', { method: 'POST', body: formData });
        if (!res.ok) { showToast('OCR processing failed', 'error'); return; }

        const events = await res.json();
        if (!events || events.length === 0) {
            showToast('No events detected. Check screenshot format.', 'warning');
            return;
        }
        document.getElementById('event-modal').style.display = 'none';
        progressBar.style.width = '100%';
        setTimeout(() => { progressWrap.style.display = 'none'; }, 600);
        showMGV2Preview(events);
    } catch (e) {
        showToast('OCR processing failed: ' + e.message, 'error');
        progressWrap.style.display = 'none';
    } finally {
        clearInterval(timer);
        processBtn.disabled = false;
        processBtn.textContent = '🔍 Process Screenshots with OCR';
    }
}

// ─── V2 multi-event preview ────────────────────────────────────────────────

// Live data store for the preview — mutated by inline edits.
let mgV2Events = [];
// All active members, loaded once for the member-select dropdowns.
let mgAllMembers = [];
// Blob URLs created from the uploaded files (revoked when preview closes).
let mgV2BlobURLs = [];
// Lightbox state
let mgLightboxURLs = [];
let mgLightboxIdx   = 0;
let mgLbTouchStartX = 0;

async function loadMGMembers() {
    try {
        const res = await fetch('/api/members');
        if (!res.ok) return;
        const data = await res.json();
        mgAllMembers = (data || []).sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
    } catch { /* non-critical, dropdown just stays empty */ }
}

// Build <option> elements for a member select, pre-selecting memberId.
// graveyardName is non-null when the match is a deleted member.
function buildMGMemberOptions(memberId, graveyardName) {
    let opts = '<option value="">— Unmatched —</option>';
    if (graveyardName) {
        const sel = memberId ? ' selected' : '';
        opts += `<option value="${memberId}"${sel} data-name="${escapeAttr(graveyardName.toLowerCase())}">⚰️ ${escapeHtml(graveyardName)} (graveyard)</option>`;
    }
    for (const m of mgAllMembers) {
        const nick = m.nickname ? ` [${m.nickname}]` : '';
        const searchName = escapeAttr((m.name + (m.nickname ? ' ' + m.nickname : '')).toLowerCase());
        const sel = (!graveyardName && m.id === memberId) ? ' selected' : '';
        opts += `<option value="${m.id}"${sel} data-name="${searchName}">${escapeHtml(m.name)}${escapeHtml(nick)} (${m.rank})</option>`;
    }
    return opts;
}

// Filter a .mg-member-select by search term and sync mgV2Events on single match.
function filterMGSelect(select, term) {
    let visible = 0, lastIdx = -1;
    for (let i = 0; i < select.options.length; i++) {
        const opt = select.options[i];
        if (i === 0) { opt.style.display = ''; continue; } // keep "Unmatched"
        const name = opt.dataset.name || '';
        const show = !term || name.includes(term) || fuzzyMatchMG(name, term);
        opt.style.display = show ? '' : 'none';
        if (show) { visible++; lastIdx = i; }
    }
    if (visible === 1 && term && lastIdx > 0) {
        select.selectedIndex = lastIdx;
        const evIdx = parseInt(select.dataset.ev), rowIdx = parseInt(select.dataset.row);
        if (!isNaN(evIdx) && !isNaN(rowIdx)) {
            mgV2Events[evIdx].rows[rowIdx].member_id = parseInt(select.value) || null;
        }
    }
}

function fuzzyMatchMG(str, pattern) {
    if (!pattern) return true;
    if (str.includes(pattern)) return true;
    let pi = 0;
    for (let i = 0; i < str.length && pi < pattern.length; i++) {
        if (str[i] === pattern[pi]) pi++;
    }
    return pi === pattern.length;
}

function showMGV2Preview(events) {
    // Create fresh blob URLs for all uploaded files (revoke old ones first).
    mgV2BlobURLs.forEach(u => URL.revokeObjectURL(u));
    mgV2BlobURLs = selectedFiles.map(f => URL.createObjectURL(f));

    mgV2Events = events.map(ev => ({
        ...ev,
        notes: '',
        // Attach blob URLs matching the source file indices returned by the server.
        blobURLs: (ev.source_file_indices || []).map(i => mgV2BlobURLs[i]).filter(Boolean),
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
        drawMGRowCrops(card);
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

    // Match quality summary for the card header.
    const named      = (ev.rows || []).filter(r => r.name);
    const nMatched   = named.filter(r => r.member_id && !r.graveyard_match).length;
    const nGraveyard = named.filter(r => r.graveyard_match).length;
    const nUnmatched = named.filter(r => !r.member_id).length;
    const summaryParts = [];
    if (nMatched)   summaryParts.push(`<span class="mg-sum-ok">✅ ${nMatched}</span>`);
    if (nGraveyard) summaryParts.push(`<span class="mg-sum-gy">⚰️ ${nGraveyard}</span>`);
    if (nUnmatched) summaryParts.push(`<span class="mg-sum-un">❓ ${nUnmatched}</span>`);
    const matchSummary = summaryParts.join(' ');

    let memberRows = '';
    (ev.rows || []).forEach((row, rIdx) => {
        const isGap  = !row.name && !row.damage_str;
        const isTop  = row.rank === 1;
        const noMember = !row.member_id && !isGap;
        const needsReview = isGap || !row.damage_ok || noMember;
        const rowClass = isTop ? 'mg-v2-top-row' : (isGap ? 'mg-v2-gap-row' : (needsReview ? 'mg-v2-warn-row' : ''));
        const rankCell = isTop ? '🏆' : row.rank;

        // Player cell: search input + member select + OCR hint
        let playerCellContent;
        if (isGap) {
            // Gap row — allow manual assignment of a member and damage
            const opts = buildMGMemberOptions(row.member_id || null, null);
            playerCellContent = `
                <input class="mg-search-input" placeholder="🔍 filter…" autocomplete="off" aria-label="Search member">
                <select class="mg-member-select mg-member-select--warn" data-ev="${evIdx}" data-row="${rIdx}">${opts}</select>
                <div class="mg-ocr-hint">Gap — no screenshot for rank ${row.rank}</div>`;
        } else {
            const ocrText = (row.alliance_tag ? `[${row.alliance_tag}] ` : '') + (row.name || '');
            const warnClass = noMember ? ' mg-member-select--warn' : '';
            const opts = buildMGMemberOptions(
                row.member_id,
                row.graveyard_match ? (row.member_name || null) : null
            );
            playerCellContent = `
                <input class="mg-search-input" placeholder="🔍 filter…" autocomplete="off" aria-label="Search member">
                <select class="mg-member-select${warnClass}" data-ev="${evIdx}" data-row="${rIdx}">${opts}</select>
                <div class="mg-ocr-hint">OCR: ${escapeHtml(ocrText)}</div>`;
        }

        const dmgInput = `<input class="mg-cell-input mg-dmg-input" data-ev="${evIdx}" data-row="${rIdx}" data-field="damage_str"
            value="${escapeAttr(row.damage_str || '')}" placeholder="—"
            style="${!row.damage_ok ? 'border-color:var(--warning,#f59e0b);' : ''}">`;

        // Status icon
        let statusIcon;
        if (isGap) {
            statusIcon = '<span class="mg-si mg-si-none" title="Rank gap — assign member manually">❓</span>';
        } else if (noMember) {
            statusIcon = '<span class="mg-si mg-si-none" title="No member matched — please select">❓</span>';
        } else if (!row.damage_ok) {
            statusIcon = '<span class="mg-si mg-si-warn" title="Damage OCR uncertain or out of order">⚠</span>';
        } else if (row.rank_fixed) {
            statusIcon = '<span class="mg-si mg-si-fixed" title="Rank inferred from sequence">✓<sup>+</sup></span>';
        } else {
            statusIcon = '<span class="mg-si mg-si-ok">✓</span>';
        }

        // Inline screenshot crop for ⚠ rows (damage_ok === false)
        // Shows a canvas thumbnail of the row's location in the source image.
        let cropCanvas = '';
        if (!row.damage_ok && row.source_file_idx != null) {
            // source_file_idx is a global index into mgV2BlobURLs, not into ev.blobURLs
            const blobURL = mgV2BlobURLs[row.source_file_idx] || (ev.blobURLs && ev.blobURLs[0]);
            if (blobURL && row.crop_y0_pct != null && row.crop_y1_pct != null && row.crop_y1_pct > row.crop_y0_pct) {
                cropCanvas = `<canvas class="mg-row-crop" aria-hidden="true"
                    data-src="${escapeAttr(blobURL)}"
                    data-y0="${row.crop_y0_pct}"
                    data-y1="${row.crop_y1_pct}"
                    title="Screenshot crop for this row"></canvas>`;
            }
        }

        memberRows += `<tr class="${rowClass}">
            <td class="mg-rank-col">${rankCell}</td>
            <td class="mg-name-col"><div class="mg-player-cell">${playerCellContent}</div></td>
            <td class="mg-dmg-col">${dmgInput}${cropCanvas}</td>
            <td class="mg-status-col">${statusIcon}</td>
        </tr>`;
    });

    // Screenshot viewer button — only when we have blob URLs for this event.
    const screenshotCount = (ev.blobURLs && ev.blobURLs.length) || 0;
    const screenshotBtn = screenshotCount > 0
        ? `<button class="mg-view-screenshots-btn" data-ev-shots="${evIdx}" title="View original screenshot(s) to fill gaps or fix OCR errors">📷${screenshotCount > 1 ? ' ' + screenshotCount : ''}</button>`
        : '';

    return `
        <div class="mg-v2-card-header">
            <div class="mg-card-meta">
                <strong class="mg-event-date">📅 ${escapeHtml(ev.event_date || 'Unknown date')}</strong>
                <span class="mg-match-summary">${ev.rows.length} players &nbsp;·&nbsp; ${matchSummary}</span>
            </div>
            <div class="mg-card-controls">
                ${screenshotBtn}
                <input type="date" class="mg-date-input" data-ev="${evIdx}"
                       value="${escapeAttr(ev.event_date || '')}" title="Edit event date">
                <input type="text" class="mg-notes-input" data-ev="${evIdx}"
                       value="${escapeAttr(ev.notes || '')}" placeholder="Notes (optional)" title="Event notes">
                <button class="mg-import-btn" data-import-event="${evIdx}">✔ Import</button>
            </div>
        </div>
        ${overwrite}
        <div class="rk-table-wrapper" style="max-height:420px;overflow-y:auto;">
            <table class="rk-table mg-v2-table">
                <thead><tr><th class="mg-rank-col">#</th><th>Player</th><th class="mg-dmg-col">Damage</th><th class="mg-status-col">✓</th></tr></thead>
                <tbody>${memberRows}</tbody>
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
    // Member search input — filter the adjacent select
    if (t.classList.contains('mg-search-input')) {
        const select = t.closest('.mg-player-cell')?.querySelector('.mg-member-select');
        if (select) filterMGSelect(select, t.value.toLowerCase().trim());
    }
});

// Member select change — update member_id in live data.
document.addEventListener('change', e => {
    const t = e.target;
    if (t.classList.contains('mg-member-select')) {
        const evIdx  = parseInt(t.dataset.ev);
        const rowIdx = parseInt(t.dataset.row);
        if (!isNaN(evIdx) && !isNaN(rowIdx)) {
            mgV2Events[evIdx].rows[rowIdx].member_id = t.value ? parseInt(t.value) : null;
        }
    }
});

async function importSingleEvent(evIdx) {
    const ev = mgV2Events[evIdx];
    if (!ev.event_date) { showToast('Event date is required', 'warning'); return; }

    const participants = [];
    for (const row of ev.rows) {
        const dmgStr = (row.damage_str || '').trim();
        if (!row.name && !dmgStr) continue; // skip empty gap rows
        // Prefer the matched member's canonical name; fall back to OCR name.
        const matchedMember = row.member_id
            ? mgAllMembers.find(m => m.id === row.member_id) || null
            : null;
        const name_snapshot = matchedMember
            ? matchedMember.name
            : (row.member_name || row.name || '').trim(); // graveyard or OCR fallback
        participants.push({
            rank_in_event: row.rank,
            name_snapshot,
            alliance_tag:  row.alliance_tag || '',
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
        // Free browser memory: revoke blob URLs and clear the file list.
        clearFiles();
    });
});

// ─── Inline screenshot crop thumbnails ───────────────────────────────────────

// After rendering an event card, find all canvas.mg-row-crop elements and draw
// the appropriate crop from the source blob URL onto each.
function drawMGRowCrops(container) {
    container.querySelectorAll('canvas.mg-row-crop').forEach(canvas => {
        const src  = canvas.dataset.src;
        const y0   = parseFloat(canvas.dataset.y0);
        const y1   = parseFloat(canvas.dataset.y1);
        if (!src || isNaN(y0) || isNaN(y1) || y1 <= y0) return;

        const img = new Image();
        img.onload = () => {
            const srcH   = img.naturalHeight;
            const srcW   = img.naturalWidth;
            const cropH  = Math.round(srcH * (y1 - y0));
            const cropY  = Math.round(srcH * y0);
            // Scale to a fixed display width (200px max), preserve aspect ratio
            const dispW  = Math.min(srcW, 220);
            const dispH  = Math.round(dispW * cropH / srcW);
            canvas.width  = dispW;
            canvas.height = Math.max(dispH, 20);
            const ctx = canvas.getContext('2d');
            ctx.drawImage(img, 0, cropY, srcW, cropH, 0, 0, dispW, dispH);
        };
        img.onerror = () => { canvas.style.display = 'none'; };
        img.src = src;
    });
}

// ─── Screenshot Lightbox ──────────────────────────────────────────────────────

function openMGLightbox(blobURLs, startIdx) {
    mgLightboxURLs = blobURLs;
    mgLightboxIdx  = startIdx || 0;
    mgLbUpdateLightbox();
    document.getElementById('mg-lightbox').style.display = 'flex';
    document.body.style.overflow = 'hidden';
}

function closeMGLightbox() {
    document.getElementById('mg-lightbox').style.display = 'none';
    document.body.style.overflow = '';
}

function mgLbNav(dir) {
    if (mgLightboxURLs.length <= 1) return;
    mgLightboxIdx = (mgLightboxIdx + dir + mgLightboxURLs.length) % mgLightboxURLs.length;
    mgLbUpdateLightbox();
}

function mgLbUpdateLightbox() {
    const img     = document.getElementById('mg-lightbox-img');
    const counter = document.getElementById('mg-lightbox-counter');
    const prevBtn = document.getElementById('mg-lightbox-prev');
    const nextBtn = document.getElementById('mg-lightbox-next');
    if (!img) return;
    img.src = mgLightboxURLs[mgLightboxIdx];
    const multi = mgLightboxURLs.length > 1;
    counter.textContent = multi ? `${mgLightboxIdx + 1} / ${mgLightboxURLs.length}` : '';
    prevBtn.style.display = multi ? '' : 'none';
    nextBtn.style.display = multi ? '' : 'none';
}

// ─── Damage / name parsing helpers ────────────────────────────────────────────

// parseMGName splits "[TAG]PlayerName" → { tag, name }. Also handles fuzzy ] (OCR reads ] as l, 1, | or I).
function parseMGName(raw) {
    const m = raw.match(/^\[([A-Za-z0-9]{1,10})\]\s*(.+)$/);
    if (m) return { tag: m[1], name: m[2].trim() };
    const f = raw.match(/^\[([A-Za-z0-9]{1,10})[lI1|]\s*(.+)$/);
    if (f) return { tag: f[1], name: f[2].trim() };
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

    await Promise.all([loadEvents(), loadMemberStats(), loadMGMembers()]);

    // ── Screenshot lightbox setup ─────────────────────────────────────────────
    const lb = document.getElementById('mg-lightbox');
    if (lb) {
        document.getElementById('mg-lightbox-close')
            .addEventListener('click', closeMGLightbox);
        document.getElementById('mg-lightbox-prev')
            .addEventListener('click', () => mgLbNav(-1));
        document.getElementById('mg-lightbox-next')
            .addEventListener('click', () => mgLbNav(1));
        // Close on backdrop click
        lb.addEventListener('click', e => {
            if (e.target === lb) closeMGLightbox();
        });
        // Touch swipe (left = next, right = prev)
        lb.addEventListener('touchstart', e => {
            mgLbTouchStartX = e.touches[0].clientX;
        }, { passive: true });
        lb.addEventListener('touchend', e => {
            const diff = e.changedTouches[0].clientX - mgLbTouchStartX;
            if (Math.abs(diff) > 48) mgLbNav(diff > 0 ? -1 : 1);
        }, { passive: true });
    }

    // ── Screenshot button click delegation ───────────────────────────────────
    document.addEventListener('click', e => {
        const btn = e.target.closest('[data-ev-shots]');
        if (!btn) return;
        const evIdx = parseInt(btn.dataset.evShots);
        if (!isNaN(evIdx) && mgV2Events[evIdx]?.blobURLs?.length) {
            openMGLightbox(mgV2Events[evIdx].blobURLs, 0);
        }
    });

    // ── Crop canvas click — open full screenshot annotated with row highlight ─
    document.addEventListener('click', e => {
        const cropCanvas = e.target.closest('canvas.mg-row-crop');
        if (!cropCanvas) return;
        const src = cropCanvas.dataset.src;
        const y0  = parseFloat(cropCanvas.dataset.y0);
        const y1  = parseFloat(cropCanvas.dataset.y1);
        if (!src || isNaN(y0) || isNaN(y1) || y1 <= y0) return;

        const img = new Image();
        img.onload = () => {
            const w = img.naturalWidth;
            const h = img.naturalHeight;
            const off = document.createElement('canvas');
            off.width  = w;
            off.height = h;
            const ctx = off.getContext('2d');
            // Draw full screenshot
            ctx.drawImage(img, 0, 0);
            // Dim rows above and below
            const ry0 = Math.round(h * y0);
            const ry1 = Math.round(h * y1);
            ctx.fillStyle = 'rgba(0,0,0,0.5)';
            if (ry0 > 0)    ctx.fillRect(0, 0,   w, ry0);
            if (ry1 < h)    ctx.fillRect(0, ry1,  w, h - ry1);
            // Orange highlight border around the row
            ctx.strokeStyle = '#f0a030';
            ctx.lineWidth   = Math.max(3, Math.round(w / 120));
            const pad = Math.round(ctx.lineWidth / 2);
            ctx.strokeRect(pad, ry0 + pad, w - pad * 2, (ry1 - ry0) - pad * 2);
            off.toBlob(blob => {
                if (!blob) return;
                const url = URL.createObjectURL(blob);
                openMGLightbox([url], 0);
                setTimeout(() => URL.revokeObjectURL(url), 120000);
            }, 'image/jpeg', 0.92);
        };
        img.onerror = () => { /* silent */ };
        img.src = src;
    });

    // ── Lightbox keyboard navigation ─────────────────────────────────────────
    document.addEventListener('keydown', e => {
        const lb = document.getElementById('mg-lightbox');
        if (!lb || lb.style.display === 'none') return;
        if (e.key === 'Escape')      { e.preventDefault(); closeMGLightbox(); }
        if (e.key === 'ArrowLeft')   { e.preventDefault(); mgLbNav(-1); }
        if (e.key === 'ArrowRight')  { e.preventDefault(); mgLbNav(1); }
    });
});
