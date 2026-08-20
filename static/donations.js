// Tech Donations page logic
let canUpload = false; // R3, R4, R5, admin — can upload screenshots & manage records
let isOfficer = false; // R4, R5, admin — can delete records
let isAdmin = false;
let selectedFiles = [];
let ocrPreview = null; // { donation_type, record_date, rows: [...] }
let allMembers = [];

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
            const tab = document.getElementById('tab-' + btn.dataset.tab);
            if (tab) tab.classList.add('active');
            if (btn.dataset.tab === 'leaderboard') loadBoard();
            if (btn.dataset.tab === 'compliance') loadCompliance();
        });
    });
}

// ---- Upload ----
function initUpload() {
    const dropZone = document.getElementById('donation-drop-zone');
    const fileInput = document.getElementById('donation-image-input');
    const processBtn = document.getElementById('donation-process-btn');
    const clearBtn = document.getElementById('donation-clear-btn');

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
    const gallery = document.getElementById('donation-preview-gallery');
    const container = document.getElementById('donation-preview-container');
    const dropContent = document.getElementById('donation-drop-content');
    const processBtn = document.getElementById('donation-process-btn');
    const countEl = document.getElementById('donation-files-count');

    if (selectedFiles.length === 0) {
        container.style.display = 'none';
        dropContent.style.display = '';
        if (canUpload) processBtn.style.display = 'none';
        return;
    }

    dropContent.style.display = 'none';
    container.style.display = 'block';
    if (canUpload) processBtn.style.display = 'block';
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
    document.getElementById('donation-image-input').value = '';
    renderFilePreview();
}

async function processScreenshots() {
    if (selectedFiles.length === 0) return;
    const donationType = document.getElementById('donation-type-select').value;
    const recordDate = document.getElementById('donation-record-date').value;
    if (!recordDate) { showToast('Select the record date first', 'warning'); return; }

    const formData = new FormData();
    formData.append('donation_type', donationType);
    formData.append('record_date', recordDate);
    selectedFiles.forEach(f => formData.append('images', f));

    const progressWrap = document.getElementById('donation-progress-wrap');
    const progressBar = document.getElementById('donation-progress-bar');
    const progressLabel = document.getElementById('donation-progress-label');
    const progressTime = document.getElementById('donation-progress-time');
    const processBtn = document.getElementById('donation-process-btn');
    processBtn.disabled = true;
    progressWrap.style.display = 'block';
    progressBar.style.width = '15%';
    progressLabel.textContent = 'Processing screenshots with OCR…';
    const started = Date.now();
    const timer = setInterval(() => {
        progressTime.textContent = Math.floor((Date.now() - started) / 1000) + 's';
        const w = Math.min(90, 15 + (Date.now() - started) / 200);
        progressBar.style.width = w + '%';
    }, 500);

    try {
        const res = await fetch('/api/donations/process-screenshots', { method: 'POST', body: formData });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message || 'OCR processing failed');
        progressBar.style.width = '100%';
        progressLabel.textContent = 'Done';
        if (!data.entries || data.entries.length === 0) {
            showToast('No rows detected in the screenshots', 'warning');
            return;
        }
        showPreview(data);
    } catch (e) {
        showToast('OCR failed: ' + e.message, 'error');
    } finally {
        clearInterval(timer);
        processBtn.disabled = false;
        setTimeout(() => { progressWrap.style.display = 'none'; progressBar.style.width = '0%'; }, 800);
    }
}

// ---- Members ----
async function loadMembers() {
    try {
        const res = await fetch('/api/members');
        if (!res.ok) return;
        const data = await res.json();
        allMembers = (data || []).sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
    } catch { /* non-critical */ }
}

function buildMemberOptions(memberId) {
    let opts = '<option value="">— Unmatched —</option>';
    for (const m of allMembers) {
        const nick = m.nickname ? ` [${m.nickname}]` : '';
        const sel = m.id === memberId ? ' selected' : '';
        opts += `<option value="${m.id}"${sel}>${escapeHtml(m.name)}${escapeHtml(nick)} (${escapeHtml(m.rank)})</option>`;
    }
    return opts;
}

// ---- OCR preview ----
async function showPreview(result) {
    ocrPreview = {
        donation_type: result.donation_type || 'weekly',
        record_date: result.record_date || '',
        rows: (result.entries || []).map(e => ({
            rank_in_snapshot: e.rank_in_snapshot || 0,
            name_snapshot: e.name_snapshot || '',
            points: e.points || 0,
            member_id: e.member_id || null,
            member_name: e.member_name || '',
            match_confidence: e.match_confidence || 0,
            match_type: e.match_type || 'none',
            points_crop_b64: e.points_crop_b64 || '',
            name_crop_b64: e.name_crop_b64 || '',
        })),
    };

    // Detect existing records for the same type+date (import will overwrite).
    ocrPreview.existing_count = 0;
    try {
        const res = await fetch(`/api/donations?type=${ocrPreview.donation_type}&date=${ocrPreview.record_date}`);
        if (res.ok) {
            const existing = await res.json();
            ocrPreview.existing_count = (existing || []).length;
        }
    } catch { /* non-critical */ }

    renderPreview();
    document.getElementById('donation-ocr-modal').style.display = 'flex';
}

function matchBadge(row) {
    const labels = { exact: '✓ exact', nickname: '✓ nickname', fuzzy: '~ fuzzy', none: '✗ none' };
    const colors = { exact: '#22c55e', nickname: '#22c55e', fuzzy: '#f59e0b', none: '#ef4444' };
    const t = row.match_type || 'none';
    return `<span class="match-badge" style="color:${colors[t]};font-size:.78em;" title="Match confidence: ${row.match_confidence}%">${labels[t]}</span>`;
}

function renderPreview() {
    const container = document.getElementById('donation-ocr-content');
    const overwrite = ocrPreview.existing_count > 0
        ? `<div class="info-banner info-banner--warning" style="margin-bottom:.5rem;">
               <div class="info-content"><div class="info-icon">⚠️</div>
               <div class="info-text">${ocrPreview.existing_count} existing record(s) for this type + date will be replaced.</div>
               </div></div>`
        : '';

    let rows = '';
    ocrPreview.rows.forEach((row, rIdx) => {
        const needsReview = row.points === 0 || row.match_type === 'none' || row.match_type === 'fuzzy';
        const cls = needsReview ? ' class="zs-warn-row"' : '';
        const cropThumb = row.points_crop_b64
            ? `<img src="data:image/png;base64,${row.points_crop_b64}" alt="points crop" style="height:22px;vertical-align:middle;border-radius:3px;margin-left:4px;" title="OCR crop — verify value">`
            : '';
        rows += `<tr${cls}>
            <td class="mg-rank-col">${row.rank_in_snapshot || '—'}</td>
            <td class="mg-name-col">
                <input class="mg-cell-input" data-field="name_snapshot" data-row="${rIdx}"
                    value="${escapeAttr(row.name_snapshot)}" placeholder="Player name">
                ${matchBadge(row)}
            </td>
            <td class="mg-dmg-col">
                <input class="mg-cell-input" data-field="points" data-row="${rIdx}"
                    value="${escapeAttr(String(row.points))}" placeholder="0" style="width:130px;">
                ${cropThumb}
            </td>
            <td class="mg-name-col">
                <select class="mg-member-select" data-row="${rIdx}">${buildMemberOptions(row.member_id)}</select>
            </td>
        </tr>`;
    });

    container.innerHTML = `
        <div class="mg-v2-card-header">
            <div class="mg-card-meta">
                <strong class="mg-event-date">💰 ${ocrPreview.donation_type === 'daily' ? 'Daily' : 'Weekly'} donations — ${escapeHtml(ocrPreview.record_date)}</strong>
            </div>
        </div>
        ${overwrite}
        <div class="rk-table-wrapper" style="max-height:420px;overflow-y:auto;">
            <table class="rk-table">
                <thead><tr><th class="mg-rank-col">#</th><th>Player</th><th class="mg-dmg-col">Points</th><th>Member</th></tr></thead>
                <tbody>${rows}</tbody>
            </table>
        </div>`;
}

// Sync inline edits back into ocrPreview live data.
document.addEventListener('input', e => {
    const t = e.target;
    if (t.classList.contains('mg-cell-input') && ocrPreview) {
        const rIdx = parseInt(t.dataset.row);
        const field = t.dataset.field;
        if (!isNaN(rIdx)) {
            if (field === 'points') {
                ocrPreview.rows[rIdx].points = parseInt(t.value) || 0;
            } else {
                ocrPreview.rows[rIdx][field] = t.value;
            }
        }
    }
});

document.addEventListener('change', e => {
    const t = e.target;
    if (t.classList.contains('mg-member-select') && ocrPreview) {
        const rIdx = parseInt(t.dataset.row);
        if (!isNaN(rIdx)) {
            ocrPreview.rows[rIdx].member_id = t.value ? parseInt(t.value) : null;
        }
    }
});

async function importRecords() {
    if (!ocrPreview) return;
    const entries = ocrPreview.rows
        .filter(r => (r.name_snapshot || '').trim() !== '')
        .map((r, i) => ({
            rank_in_snapshot: r.rank_in_snapshot || i + 1,
            name_snapshot: (r.name_snapshot || '').trim(),
            points: r.points || 0,
            member_id: r.member_id || null,
        }));
    if (entries.length === 0) { showToast('No rows to import', 'warning'); return; }

    const btn = document.getElementById('donation-ocr-import-btn');
    try {
        if (btn) { btn.disabled = true; btn.textContent = '⏳ Importing…'; }
        const res = await fetch('/api/donations', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                donation_type: ocrPreview.donation_type,
                record_date: ocrPreview.record_date,
                entries,
            }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message || 'Import failed');
        showToast(`Imported ${data.saved} donation record(s)`, 'success');
        document.getElementById('donation-ocr-modal').style.display = 'none';
        ocrPreview = null;
        clearFiles();
        // Refresh leaderboard with the imported data.
        document.getElementById('board-type-select').value = document.getElementById('donation-type-select').value;
        document.getElementById('board-date').value = document.getElementById('donation-record-date').value;
        loadBoard();
    } catch (e) {
        showToast('Import failed: ' + e.message, 'error');
    } finally {
        if (btn) { btn.disabled = false; btn.textContent = '✔ Import Records'; }
    }
}

// ---- Leaderboard ----
async function loadBoard() {
    const type = document.getElementById('board-type-select').value;
    const date = document.getElementById('board-date').value;
    const list = document.getElementById('board-list');
    list.innerHTML = '<p class="loading">⏳ Loading records...</p>';
    try {
        const res = await fetch(`/api/donations?type=${type}&date=${date}`);
        if (!res.ok) throw new Error('Failed to load records');
        const records = await res.json();
        renderBoard(records || [], type, date);
    } catch (e) {
        list.innerHTML = `<p class="error-message">${escapeHtml(e.message)}</p>`;
    }
}

function renderBoard(records, type, date) {
    const list = document.getElementById('board-list');
    if (!records.length) {
        list.innerHTML = `<p class="no-data">No ${type} donation records for ${escapeHtml(date)}. Upload screenshots or add records manually.</p>`;
        return;
    }
    const total = records.reduce((s, r) => s + r.amount, 0);
    let rows = '';
    records.forEach((r, i) => {
        const displayName = r.member_id
            ? `${escapeHtml(r.member_name || r.name_snapshot)}`
            : `${escapeHtml(r.name_snapshot)} <span class="text-muted" style="font-size:.8em;">(unmatched)</span>`;
        const deleteBtn = canUpload
            ? `<button class="secondary-btn donation-delete-btn" data-id="${r.id}" title="Delete record" style="padding:2px 8px;">🗑️</button>`
            : '';
        rows += `<tr>
            <td class="mg-rank-col">${i + 1}</td>
            <td>${displayName}${r.member_rank ? ` <span class="text-muted" style="font-size:.8em;">(${escapeHtml(r.member_rank)})</span>` : ''}</td>
            <td class="mg-dmg-col"><strong>${r.amount.toLocaleString()}</strong></td>
            <td class="mg-rank-col">${deleteBtn}</td>
        </tr>`;
    });
    list.innerHTML = `
        <p class="text-muted" style="font-size:.85em;margin:.25rem 0 .5rem;">${records.length} record(s) — total: <strong>${total.toLocaleString()}</strong></p>
        <table class="rk-table">
            <thead><tr><th class="mg-rank-col">#</th><th>Member</th><th class="mg-dmg-col">Amount</th><th></th></tr></thead>
            <tbody>${rows}</tbody>
        </table>`;

    list.querySelectorAll('.donation-delete-btn').forEach(btn => {
        btn.addEventListener('click', () => deleteRecord(btn.dataset.id));
    });
}

async function deleteRecord(id) {
    if (!confirm('Delete this donation record?')) return;
    try {
        const res = await fetch(`/api/donations/${id}`, { method: 'DELETE' });
        if (!res.ok) throw new Error('Delete failed');
        showToast('Record deleted', 'success');
        loadBoard();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

// ---- Add record modal ----
function openAddRecordModal() {
    const sel = document.getElementById('record-member');
    sel.innerHTML = buildMemberOptions(null).replace('<option value="">— Unmatched —</option>', '<option value="">— Select member —</option>');
    document.getElementById('record-amount').value = '';
    document.getElementById('add-record-modal').style.display = 'flex';
}

async function submitAddRecord(e) {
    e.preventDefault();
    const memberId = document.getElementById('record-member').value;
    const amount = parseInt(document.getElementById('record-amount').value) || 0;
    if (!memberId) { showToast('Select a member', 'warning'); return; }
    if (amount <= 0) { showToast('Amount must be positive', 'warning'); return; }

    const type = document.getElementById('board-type-select').value;
    const date = document.getElementById('board-date').value;
    const member = allMembers.find(m => m.id === parseInt(memberId));

    try {
        // Replace-semantics API: fetch existing records, append, save all.
        const res = await fetch(`/api/donations?type=${type}&date=${date}`);
        const existing = res.ok ? await res.json() : [];
        const entries = (existing || []).map(r => ({
            rank_in_snapshot: r.rank_in_snapshot,
            name_snapshot: r.name_snapshot || r.member_name,
            points: r.amount,
            member_id: r.member_id || null,
        }));
        entries.push({
            rank_in_snapshot: entries.length + 1,
            name_snapshot: member ? member.name : 'Unknown',
            points: amount,
            member_id: parseInt(memberId),
        });
        const saveRes = await fetch('/api/donations', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ donation_type: type, record_date: date, entries }),
        });
        if (!saveRes.ok) throw new Error('Save failed');
        showToast('Record added', 'success');
        document.getElementById('add-record-modal').style.display = 'none';
        loadBoard();
    } catch (err) {
        showToast('Failed to add record: ' + err.message, 'error');
    }
}

// ---- Compliance ----
async function loadCompliance() {
    const weeks = document.getElementById('compliance-weeks').value;
    const list = document.getElementById('compliance-list');
    list.innerHTML = '<p class="loading">⏳ Loading compliance...</p>';
    try {
        const res = await fetch(`/api/donations/compliance?weeks=${weeks}`);
        if (!res.ok) throw new Error('Failed to load compliance');
        const data = await res.json();
        renderCompliance(data);
    } catch (e) {
        list.innerHTML = `<p class="error-message">${escapeHtml(e.message)}</p>`;
    }
}

function renderCompliance(data) {
    const list = document.getElementById('compliance-list');
    const target = data.target || 0;
    const banner = target > 0
        ? `<p class="text-muted" style="font-size:.85em;">Weekly target: <strong>${target.toLocaleString()}</strong> — <span style="color:#22c55e;">■ met</span> <span style="color:#f59e0b;">■ below</span> <span style="color:#ef4444;">■ no data</span></p>`
        : `<div class="info-banner" style="margin-bottom:.75rem;"><div class="info-content"><div class="info-icon">ℹ️</div>
           <div class="info-text">No weekly donation target set — configure it in Settings → Tech Donation Targets.</div></div></div>`;

    let html = banner;
    (data.weeks || []).forEach(week => {
        let rows = '';
        week.members.forEach(m => {
            let status, color;
            if (!m.has_data) { status = '—'; color = target > 0 ? '#ef4444' : '#9ca3af'; }
            else if (target > 0 && m.met) { status = '✔ met'; color = '#22c55e'; }
            else if (target > 0) { status = 'below'; color = '#f59e0b'; }
            else { status = '✔'; color = '#22c55e'; }
            rows += `<tr>
                <td>${escapeHtml(m.name)}${m.nickname ? ` <span class="text-muted" style="font-size:.8em;">[${escapeHtml(m.nickname)}]</span>` : ''}</td>
                <td class="mg-rank-col">${escapeHtml(m.rank)}</td>
                <td class="mg-dmg-col">${m.has_data ? m.amount.toLocaleString() : '—'}</td>
                <td style="color:${color};font-weight:600;">${status}</td>
            </tr>`;
        });
        html += `
            <div class="form-section" style="margin-bottom:1rem;">
                <h4>📅 ${escapeHtml(week.week_label)}</h4>
                <div class="rk-table-wrapper">
                    <table class="rk-table">
                        <thead><tr><th>Member</th><th class="mg-rank-col">Rank</th><th class="mg-dmg-col">Donated</th><th>Status</th></tr></thead>
                        <tbody>${rows}</tbody>
                    </table>
                </div>
            </div>`;
    });
    list.innerHTML = html;
}

// ---- Modals ----
function initModals() {
    document.querySelectorAll('.modal .close').forEach(btn => {
        btn.addEventListener('click', () => btn.closest('.modal').style.display = 'none');
        btn.addEventListener('keydown', (e) => { if (e.key === 'Enter' || e.key === ' ') btn.closest('.modal').style.display = 'none'; });
    });
    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => { if (e.target === modal) modal.style.display = 'none'; });
    });
    document.getElementById('donation-ocr-cancel-btn').addEventListener('click', () => {
        document.getElementById('donation-ocr-modal').style.display = 'none';
    });
    document.getElementById('donation-ocr-import-btn').addEventListener('click', importRecords);
    document.getElementById('add-record-btn').addEventListener('click', openAddRecordModal);
    document.getElementById('add-record-form').addEventListener('submit', submitAddRecord);
    document.getElementById('board-type-select').addEventListener('change', loadBoard);
    document.getElementById('board-date').addEventListener('change', loadBoard);
    document.getElementById('compliance-weeks').addEventListener('change', loadCompliance);
}

function escapeAttr(str) {
    return String(str ?? '').replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// ---- Init ----
document.addEventListener('DOMContentLoaded', async () => {
    const ok = await checkAuth();
    if (!ok) return;

    const today = new Date().toISOString().slice(0, 10);
    document.getElementById('donation-record-date').value = today;
    document.getElementById('board-date').value = today;

    initTabs();
    initUpload();
    initModals();
    await loadMembers();
});
