// Leadership Dashboard
// Admin-only. Fetches /api/rankings, computes tiers, renders charts + tables.

const RANK_ORDER = ['R5', 'R4', 'R3', 'R2', 'R1']; // high → low

// ── helpers ───────────────────────────────────────────────────────────────────
function median(arr) {
    if (!arr.length) return 0;
    const sorted = [...arr].sort((a, b) => a - b);
    const m = Math.floor(sorted.length / 2);
    return sorted.length % 2 ? sorted[m] : (sorted[m - 1] + sorted[m]) / 2;
}
function percentile25(arr) {
    if (!arr.length) return 0;
    const sorted = [...arr].sort((a, b) => a - b);
    return sorted[Math.floor(sorted.length * 0.25)];
}
function avg(arr) {
    if (!arr.length) return 0;
    return arr.reduce((s, v) => s + v, 0) / arr.length;
}

function rankBadgeHtml(rank) {
    return `<span class="rank-badge rank-${rank.toLowerCase()}">${rank}</span>`;
}

function profileLink(member) {
    const name = member.nickname
        ? `${member.name} <span class="pf-nick">"${member.nickname}"</span>`
        : member.name;
    return `<a href="/profile.html?id=${member.id}" class="ld-member-link">${name}</a>`;
}

// ── main ──────────────────────────────────────────────────────────────────────
async function loadDashboard() {
    const res = await fetch('/api/rankings');
    if (!res.ok) { showToast('Failed to load rankings', 'error'); return; }
    const payload = await res.json();
    const rankings = Array.isArray(payload) ? payload : (payload.rankings || []);

    // Build tier map: rank → [{member, score, mgEvents, conductorCount, …}]
    const tiers = {};
    RANK_ORDER.forEach(r => tiers[r] = []);

    rankings.forEach(row => {
        const rank = row.member.rank;
        if (!tiers[rank]) tiers[rank] = [];
        tiers[rank].push(row);
    });

    // Leadership score = total_score without the recent-conductor penalty
    // (that penalty is a scheduling fairness tool, not a performance indicator).
    // total_score already has the penalty subtracted, so add it back.
    rankings.forEach(row => {
        row.ld_score = (row.total_score || 0) + (row.recent_conductor_penalty || 0);
    });

    // Alliance-wide score list (for percentile calc)
    const allScores = rankings.map(r => r.ld_score).sort((a, b) => a - b);

    function overallPercentile(score) {
        const below = allScores.filter(s => s < score).length;
        return Math.round(100 * below / Math.max(allScores.length - 1, 1));
    }

    // ── Summary cards ──────────────────────────────────────────────────────
    document.getElementById('ld-total-members').textContent = rankings.length;

    // ── Rank distribution doughnut ─────────────────────────────────────────
    buildDoughnut(tiers);

    // ── Avg score bar ──────────────────────────────────────────────────────
    buildScoreBar(tiers);

    // ── Promotion candidates ───────────────────────────────────────────────
    // A member at rank X is a candidate if score > median of rank X+1
    // Rank ladder (up): R1→R2→R3→R4→R5
    const promotable = [];
    RANK_ORDER.slice(0, -1).forEach((nextRank, idx) => {
        const currentRank = RANK_ORDER[idx + 1]; // next in array = one below
        const nextScores = tiers[nextRank].map(r => r.ld_score);
        const nextMedian = median(nextScores);
        tiers[currentRank].forEach(row => {
            if (row.ld_score > nextMedian) {
                promotable.push({ row, currentRank, nextRank, nextMedian });
            }
        });
    });
    promotable.sort((a, b) => (b.row.ld_score - b.nextMedian) - (a.row.ld_score - a.nextMedian));
    renderPromotable(promotable);
    document.getElementById('ld-promote-count').textContent = promotable.length;

    // ── Demotion watch ─────────────────────────────────────────────────────
    // R3–R5, bottom 25% overall AND 2+ activity gaps
    const p25 = percentile25(allScores);
    const demotable = [];
    ['R3', 'R4', 'R5'].forEach(rank => {
        tiers[rank].forEach(row => {
            if (row.ld_score > p25) return;
            const issues = buildIssues(row);
            if (issues.length >= 2) {
                demotable.push({ row, rank, issues, pct: overallPercentile(row.ld_score) });
            }
        });
    });
    demotable.sort((a, b) => a.row.ld_score - b.row.ld_score);
    renderDemotable(demotable);
    document.getElementById('ld-demote-count').textContent = demotable.length;

    // ── Rising stars ───────────────────────────────────────────────────────
    const rising = [];
    ['R1', 'R2'].forEach(rank => {
        const tierAvg = avg(tiers[rank].map(r => r.ld_score));
        tiers[rank].forEach(row => {
            if (row.ld_score > tierAvg) {
                rising.push({ row, rank, tierAvg, aboveBy: row.ld_score - tierAvg });
            }
        });
    });
    rising.sort((a, b) => b.aboveBy - a.aboveBy);
    renderRising(rising);
    document.getElementById('ld-rising-count').textContent = rising.length;

    // ── Full leaderboard ───────────────────────────────────────────────────
    renderFullTable(rankings);

    // ── Search / filter ────────────────────────────────────────────────────
    document.getElementById('ld-search').addEventListener('input', () => filterFull(rankings));
    document.getElementById('ld-rank-select').addEventListener('change', () => filterFull(rankings));
}

// ── issue tags ────────────────────────────────────────────────────────────────
function buildIssues(row) {
    const issues = [];
    if (row.mg_event_count === 0) issues.push('No MG');
    else if ((row.days_since_last_mg ?? 999) > 60) issues.push('MG stale 60d+');
    if (row.conductor_count === 0) issues.push('Never trained');
    if (row.award_points === 0 && row.recommendation_points === 0) issues.push('No awards/recs');
    return issues;
}

function issueTags(issues) {
    return issues.map(i => `<span class="ld-issue-tag">${i}</span>`).join(' ');
}

// ── render helpers ────────────────────────────────────────────────────────────
function renderPromotable(list) {
    const tbody = document.getElementById('ld-promote-tbody');
    const empty = document.getElementById('ld-promote-empty');
    if (!list.length) {
        tbody.closest('table').style.display = 'none';
        empty.style.display = '';
        return;
    }
    empty.style.display = 'none';
    tbody.closest('table').style.display = '';
    tbody.innerHTML = list.map(({ row, currentRank, nextRank, nextMedian }) => {
        const gap = row.ld_score - nextMedian;
        const notes = [];
        if (row.mg_event_count >= 3) notes.push('Active MG');
        if (row.conductor_count >= 2) notes.push('Reliable train');
        if (row.recommendation_points > 0) notes.push('Recommended');
        return `<tr>
            <td>${profileLink(row.member)}</td>
            <td>${rankBadgeHtml(currentRank)}</td>
            <td>${row.ld_score}</td>
            <td>${Math.round(nextMedian)} <small>(${nextRank} med.)</small></td>
            <td class="pf-pos">+${Math.round(gap)}</td>
            <td>${notes.map(n => `<span class="ld-note-tag">${n}</span>`).join(' ')}</td>
        </tr>`;
    }).join('');
}

function renderDemotable(list) {
    const tbody = document.getElementById('ld-demote-tbody');
    const empty = document.getElementById('ld-demote-empty');
    if (!list.length) {
        tbody.closest('table').style.display = 'none';
        empty.style.display = '';
        return;
    }
    empty.style.display = 'none';
    tbody.closest('table').style.display = '';
    tbody.innerHTML = list.map(({ row, rank, issues, pct }) => `<tr class="ld-row-warn">
        <td>${profileLink(row.member)}</td>
        <td>${rankBadgeHtml(rank)}</td>
        <td>${row.ld_score}</td>
        <td>${pct}th pct.</td>
        <td>${issueTags(issues)}</td>
    </tr>`).join('');
}

function renderRising(list) {
    const tbody = document.getElementById('ld-rising-tbody');
    const empty = document.getElementById('ld-rising-empty');
    if (!list.length) {
        tbody.closest('table').style.display = 'none';
        empty.style.display = '';
        return;
    }
    empty.style.display = 'none';
    tbody.closest('table').style.display = '';
    tbody.innerHTML = list.map(({ row, rank, tierAvg, aboveBy }) => `<tr>
        <td>${profileLink(row.member)}</td>
        <td>${rankBadgeHtml(rank)}</td>
        <td>${row.ld_score}</td>
        <td>${Math.round(tierAvg)}</td>
        <td class="pf-pos">+${Math.round(aboveBy)}</td>
    </tr>`).join('');
}

let fullRankings = [];
function renderFullTable(rankings) {
    fullRankings = rankings;
    filterFull(rankings);
}

function filterFull(rankings) {
    const q = document.getElementById('ld-search').value.toLowerCase();
    const rankFilter = document.getElementById('ld-rank-select').value;
    let filtered = rankings.filter(row => {
        const nameMatch = row.member.name.toLowerCase().includes(q) ||
            (row.member.nickname || '').toLowerCase().includes(q);
        const rankMatch = !rankFilter || row.member.rank === rankFilter;
        return nameMatch && rankMatch;
    });
    const tbody = document.getElementById('ld-full-tbody');
    tbody.innerHTML = filtered.map((row, i) => `<tr>
        <td>${i + 1}</td>
        <td>${profileLink(row.member)}</td>
        <td>${rankBadgeHtml(row.member.rank)}</td>
        <td>${row.ld_score}</td>
        <td>${row.mg_event_count}</td>
        <td>${row.conductor_count}</td>
        <td>${row.recommendation_count}</td>
        <td>${row.award_points}</td>
    </tr>`).join('');
}

// ── Charts ────────────────────────────────────────────────────────────────────
let doughnutChart = null;
let scoreBarChart = null;

function buildDoughnut(tiers) {
    const isDark = document.documentElement.classList.contains('theme-dark');
    const colors = ['#f6ad55', '#63b3ed', '#68d391', '#fc8181', '#b794f4'];
    const labels = RANK_ORDER.filter(r => tiers[r].length);
    const data   = labels.map(r => tiers[r].length);
    const ctx = document.getElementById('ld-rank-doughnut').getContext('2d');
    if (doughnutChart) doughnutChart.destroy();
    doughnutChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels,
            datasets: [{ data, backgroundColor: colors, borderWidth: 2,
                borderColor: isDark ? '#1a202c' : '#fff' }]
        },
        options: {
            responsive: true, maintainAspectRatio: false,
            plugins: {
                legend: { position: 'right', labels: { color: isDark ? '#ccc' : '#333', boxWidth: 14 } }
            }
        }
    });
}

function buildScoreBar(tiers) {
    const isDark = document.documentElement.classList.contains('theme-dark');
    const labelColor = isDark ? '#ccc' : '#444';
    const gridColor  = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.08)';
    const barColor   = isDark ? '#63b3ed' : '#3182ce';
    const ranks = RANK_ORDER.filter(r => tiers[r].length);
    const avgs  = ranks.map(r => Math.round(avg(tiers[r].map(x => x.ld_score))));
    const ctx = document.getElementById('ld-rank-score-bar').getContext('2d');
    if (scoreBarChart) scoreBarChart.destroy();
    scoreBarChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: ranks,
            datasets: [{ label: 'Avg Score', data: avgs, backgroundColor: barColor, borderRadius: 5 }]
        },
        options: {
            responsive: true, maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                x: { ticks: { color: labelColor }, grid: { display: false } },
                y: { ticks: { color: labelColor }, grid: { color: gridColor } }
            }
        }
    });
}

// ── init ──────────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', async () => {
    const authData = await requireAuth();
    if (!authData) return;
    if (!authData.is_admin) { window.location.href = '/'; return; }

    await loadDashboard();
});
