// Donations leaderboard page

let donationType = 'daily'; // 'daily' | 'weekly'
let currentDate  = null;    // Date object (day for daily, Monday for weekly)

// ── Date helpers (mirrors vs.js) ──────────────────────────────────────────────

function getMostRecentMonday(d = new Date()) {
    const day  = d.getDay();
    const diff = day === 0 ? 6 : day - 1;
    const m    = new Date(d);
    m.setDate(d.getDate() - diff);
    m.setHours(0, 0, 0, 0);
    return m;
}

function fmtDate(d) {
    return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;
}

function fmtDateDisplay(d) {
    return d.toLocaleDateString('en-GB', { weekday:'long', day:'numeric', month:'long', year:'numeric' });
}

function fmtWeekDisplay(monday) {
    const saturday = new Date(monday);
    saturday.setDate(monday.getDate() + 5);
    return `${monday.toLocaleDateString('en-GB',{day:'numeric',month:'short'})} – ${saturday.toLocaleDateString('en-GB',{day:'numeric',month:'short',year:'numeric'})}`;
}

function stepDate(d, dir) {
    const next = new Date(d);
    if (donationType === 'daily') {
        next.setDate(d.getDate() + dir);
    } else {
        next.setDate(d.getDate() + dir * 7);
    }
    return next;
}

// ── Rank badge helper ─────────────────────────────────────────────────────────

const RANK_COLOURS = {
    R5: '#e57373',
    R4: '#ffb74d',
    R3: '#81c784',
    R2: '#64b5f6',
    R1: '#b0bec5'
};

function rankBadge(rank) {
    if (!rank) return '';
    const colour = RANK_COLOURS[rank] || '#b0bec5';
    return `<span style="display:inline-block; padding:2px 7px; border-radius:12px; font-size:11px; font-weight:700; background:${colour}22; color:${colour}; border:1px solid ${colour}66; white-space:nowrap;">${rank}</span>`;
}

function medalEmoji(pos) {
    if (pos === 1) return '🥇';
    if (pos === 2) return '🥈';
    if (pos === 3) return '🥉';
    return `#${pos}`;
}

// ── Data loading ──────────────────────────────────────────────────────────────

async function loadDonations() {
    const dateStr = fmtDate(currentDate);

    // Update label
    const label = donationType === 'daily'
        ? fmtDateDisplay(currentDate)
        : `Week of ${fmtWeekDisplay(currentDate)}`;
    document.getElementById('don-date-label').textContent = label;

    const tbody = document.getElementById('don-tbody');
    tbody.innerHTML = '<tr><td colspan="4" style="text-align:center; padding:32px; color:var(--text-muted);">Loading…</td></tr>';
    document.getElementById('don-empty').style.display = 'none';

    try {
        const res = await fetch(`${API_BASE}/donations?type=${donationType}&date=${dateStr}`);
        if (!res.ok) throw new Error(await res.text());
        const records = await res.json();
        renderTable(records);
    } catch (err) {
        console.error('loadDonations:', err);
        tbody.innerHTML = `<tr><td colspan="4" style="text-align:center; padding:32px; color:var(--error-color, #e57373);">Failed to load data</td></tr>`;
    }
}

// ── Render ────────────────────────────────────────────────────────────────────

function renderTable(records) {
    const tbody  = document.getElementById('don-tbody');
    const empty  = document.getElementById('don-empty');
    const table  = document.getElementById('don-table');

    if (!records || records.length === 0) {
        tbody.innerHTML = '';
        table.style.display = 'none';
        empty.style.display = 'block';
        return;
    }

    table.style.display = '';
    empty.style.display = 'none';

    // Assign display rank (sequential, ignoring stored rank_in_snapshot)
    tbody.innerHTML = records.map((rec, i) => {
        const pos      = i + 1;
        const name     = rec.member_name || rec.name_snapshot;
        const badge    = rankBadge(rec.member_rank);
        const medal    = medalEmoji(pos);
        const rowStyle = pos <= 3 ? `background:var(--card-bg);` : '';

        return `<tr style="${rowStyle}">
            <td style="text-align:center; font-weight:700; font-size:${pos <= 3 ? '1.2em' : '1em'}">${medal}</td>
            <td>
                <span style="font-weight:${pos <= 3 ? '700' : '500'}">${escapeHtml(name)}</span>
                ${badge ? ` ${badge}` : ''}
                ${rec.member_name && rec.member_name !== rec.name_snapshot
                    ? `<small style="color:var(--text-muted); margin-left:6px; font-size:11px;">(${escapeHtml(rec.name_snapshot)})</small>`
                    : ''}
            </td>
            <td style="text-align:center;">${badge}</td>
            <td style="text-align:right; padding-right:16px; font-weight:600; font-variant-numeric:tabular-nums;">
                ${rec.amount.toLocaleString()}
            </td>
        </tr>`;
    }).join('');
}

// ── Init ──────────────────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', async () => {
    const auth = await requireAuth();
    if (!auth) return;

    // Read ?type and ?date from URL params
    const params = new URLSearchParams(window.location.search);
    const typeParam = params.get('type');
    if (typeParam === 'daily' || typeParam === 'weekly') {
        donationType = typeParam;
    }
    const dateParam = params.get('date');
    if (dateParam && /^\d{4}-\d{2}-\d{2}$/.test(dateParam)) {
        currentDate = new Date(dateParam + 'T00:00:00');
    }

    // Set default date
    if (!currentDate) {
        currentDate = donationType === 'weekly'
            ? getMostRecentMonday()
            : new Date();
        currentDate.setHours(0, 0, 0, 0);
    }

    // Sync type-tab active state
    document.querySelectorAll('[data-dontype]').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.dontype === donationType);
    });

    // Type tab click
    document.querySelectorAll('[data-dontype]').forEach(btn => {
        btn.addEventListener('click', () => {
            if (btn.dataset.dontype === donationType) return;
            donationType = btn.dataset.dontype;
            document.querySelectorAll('[data-dontype]').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            // Reset to today / this week
            if (donationType === 'weekly') {
                currentDate = getMostRecentMonday();
            } else {
                currentDate = new Date();
                currentDate.setHours(0, 0, 0, 0);
            }
            loadDonations();
        });
    });

    // Date navigation
    document.getElementById('don-prev').addEventListener('click', () => {
        currentDate = stepDate(currentDate, -1);
        loadDonations();
    });
    document.getElementById('don-next').addEventListener('click', () => {
        currentDate = stepDate(currentDate, 1);
        loadDonations();
    });
    document.getElementById('don-today').addEventListener('click', () => {
        currentDate = donationType === 'weekly'
            ? getMostRecentMonday()
            : new Date();
        currentDate.setHours(0, 0, 0, 0);
        loadDonations();
    });

    await loadDonations();
});
