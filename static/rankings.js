// Rankings page
const RANKINGS_URL = `${API_BASE}/rankings`;

function getCSSColor(varName) {
    return getComputedStyle(document.documentElement).getPropertyValue(varName).trim();
}

function applyChartTheme() {
    Chart.defaults.color = getCSSColor('--text-primary');
    Chart.defaults.borderColor = getCSSColor('--border-color');
}

let currentData = null;
let filteredRankings = null;
let maxScore = 1;
let activeRankFilter = '';
let breakdownChart = null;
let memberTimelineCharts = [];
let cachedTimelineData = null;

// Persist rank order across page loads for change indicators
let previousRankingsMap = (function () {
    try { return JSON.parse(localStorage.getItem('rankingsOrder') || 'null'); } catch { return null; }
})();

function saveRankingsOrder(rankings) {
    try {
        const map = {};
        rankings.forEach((r, i) => { map[r.member.id] = i; });
        localStorage.setItem('rankingsOrder', JSON.stringify(map));
    } catch {}
}

// ── Data load ─────────────────────────────────────────────────────────────────

async function loadRankings() {
    const btn = document.getElementById('refresh-btn');
    setButtonLoading(btn, '...');
    try {
        const response = await fetch(RANKINGS_URL);
        if (!response.ok) throw new Error('Failed to load rankings');
        currentData = await response.json();
        if (!Array.isArray(currentData.rankings)) currentData.rankings = [];

        maxScore = Math.max(...currentData.rankings.map(r => r.total_score), 1);
        filteredRankings = currentData.rankings;

        document.getElementById('avg-count').textContent =
            currentData.average_conductor_count.toFixed(1);

        displayUpNext(currentData.rankings);
        populateFormulaModal(currentData.settings, currentData.average_conductor_count);
        applyFiltersAndSort();

        // Save for next visit (after displaying, so change indicators already computed)
        saveRankingsOrder(currentData.rankings);

    } catch (error) {
        console.error('Error loading rankings:', error);
        document.getElementById('rankings-tbody').innerHTML =
            '<tr><td colspan="8" class="error-message">Failed to load rankings. Please try again.</td></tr>';
    } finally {
        clearButtonLoading(btn);
    }
}

// ── Up Next ───────────────────────────────────────────────────────────────────

function displayUpNext(rankings) {
    const top = rankings.slice(0, 3);
    const container = document.getElementById('up-next-list');

    if (!top.length) {
        container.innerHTML = '<div class="empty-state">No data available.</div>';
        return;
    }

    const labels = ['1st Choice', '2nd Choice', '3rd Choice'];
    container.innerHTML = top.map((r, i) => {
        const isFirst = r.conductor_count === 0;
        const sinceText = isFirst
            ? '<span class="rk-never">🆕 Never</span>'
            : r.days_since_last_conductor !== null
                ? `${r.days_since_last_conductor}d ago`
                : '—';

        return `
            <div class="rk-up-card rk-up-${i + 1}">
                <div class="rk-up-label">${labels[i]}</div>
                <div class="rk-up-name">
                    ${escapeHtml(r.member.name)}${r.member.nickname ? '<span class="member-nickname"> aka ' + escapeHtml(r.member.nickname) + '</span>' : ''}
                    <span class="rank-badge rank-badge-${r.member.rank.toLowerCase()}">${r.member.rank}</span>
                    ${isFirst ? '<span class="first-timer-badge" title="Never been conductor">🆕</span>' : ''}
                </div>
                <div class="rk-up-score">${r.total_score} <span class="rk-up-pts">pts</span></div>
                <div class="rk-up-since">${sinceText}</div>
            </div>`;
    }).join('');
}

// ── Filters & Sort ────────────────────────────────────────────────────────────

function applyFiltersAndSort() {
    if (!currentData) return;
    const nameFilter = document.getElementById('filter-name').value.toLowerCase().trim();
    const sortBy = document.getElementById('sort-by').value;

    filteredRankings = currentData.rankings.filter(r => {
        const nameOk = !nameFilter || r.member.name.toLowerCase().includes(nameFilter) || (r.member.nickname && r.member.nickname.toLowerCase().includes(nameFilter));
        const rankOk = !activeRankFilter
            || (activeRankFilter === 'first-timer' ? r.conductor_count === 0 : r.member.rank === activeRankFilter);
        return nameOk && rankOk;
    });

    filteredRankings.sort((a, b) => {
        switch (sortBy) {
            case 'total_score':          return b.total_score - a.total_score;
            case 'conductor_count':      return b.conductor_count - a.conductor_count;
            case 'days_since':           return (b.days_since_last_conductor ?? 9999) - (a.days_since_last_conductor ?? 9999);
            case 'award_points':         return b.award_points - a.award_points;
            case 'recommendation_points':return b.recommendation_points - a.recommendation_points;
            case 'name':                 return a.member.name.localeCompare(b.member.name);
            default: return 0;
        }
    });

    displayRankings(filteredRankings);
}

function setActiveChip(rank) {
    activeRankFilter = rank;
    document.querySelectorAll('#rank-chips .filter-chip').forEach(chip => {
        chip.classList.toggle('active', chip.dataset.rank === rank);
    });
    applyFiltersAndSort();
}

// ── Rankings table ────────────────────────────────────────────────────────────

function displayRankings(rankings) {
    memberTimelineCharts.forEach(c => c.destroy());
    memberTimelineCharts = [];
    cachedTimelineData = null;

    const tbody = document.getElementById('rankings-tbody');

    if (!rankings.length) {
        tbody.innerHTML = '<tr><td colspan="8" class="empty-state">🧙 No survivors match these parameters.</td></tr>';
        return;
    }

    tbody.innerHTML = rankings.map((r, i) => buildRowPair(r, i)).join('');

    // Timeline option change → rebuild chart for that member
    tbody.querySelectorAll('.timeline-opt').forEach(el => {
        el.addEventListener('change', e => rebuildSingleMemberChart(parseInt(e.target.dataset.memberId)));
    });

    // Inactive awards toggle
    tbody.querySelectorAll('.show-inactive-cb').forEach(cb => {
        cb.addEventListener('change', e => toggleInactiveAwards(parseInt(e.target.dataset.memberId), e.target.checked));
    });

    // Re-render the analytics chart if the section is open
    const details = document.querySelector('.rk-analytics');
    if (details && details.open) displayBreakdownChart(currentData.rankings);
}

function buildRowPair(ranking, index) {
    const medal = ['🥇','🥈','🥉'][index] || `#${index + 1}`;
    const change = getRankChangeIndicator(ranking.member.id, index);
    const isFirst = ranking.conductor_count === 0;
    const barPct = ((ranking.total_score / maxScore) * 100).toFixed(1);
    const lastRunHtml = formatLastRunCell(ranking.days_since_last_conductor);

    const dataRow = `
        <tr class="rk-row" data-member-id="${ranking.member.id}" onclick="toggleRow(${ranking.member.id})">
            <td class="rk-col-pos">${medal}${change}</td>
            <td class="rk-col-name">
                <span class="rk-member-name">${escapeHtml(ranking.member.name)}${ranking.member.nickname ? '<span class="member-nickname"> aka ' + escapeHtml(ranking.member.nickname) + '</span>' : ''}</span>
                <span class="rank-badge rank-badge-${ranking.member.rank.toLowerCase()}">${ranking.member.rank}</span>
                ${isFirst ? '<span class="first-timer-badge" title="Never been conductor">🆕</span>' : ''}
            </td>
            <td class="rk-col-score">
                <span class="rk-score-num">${ranking.total_score}</span>
                <div class="rk-score-bar"><div class="rk-score-fill" style="width:${barPct}%"></div></div>
            </td>
            <td class="rk-col-runs">${ranking.conductor_count}</td>
            <td class="rk-col-since">${lastRunHtml}</td>
            <td class="rk-col-awards">${ranking.award_points > 0 ? '+' + ranking.award_points : '<span class="rk-zero">—</span>'}</td>
            <td class="rk-col-recs">${ranking.recommendation_points > 0 ? '+' + ranking.recommendation_points : '<span class="rk-zero">—</span>'}</td>
            <td class="rk-col-mg">${ranking.mg_event_count > 0 ? ranking.mg_event_count : '<span class="rk-zero">—</span>'}</td>
            <td class="rk-col-expand"><span class="rk-expand-icon">▶</span></td>
        </tr>`;

    const detailRow = `
        <tr class="rk-detail-row" id="detail-row-${ranking.member.id}">
            <td colspan="9"><div class="rk-detail-inner">${buildDetailContent(ranking)}</div></td>
        </tr>`;

    return dataRow + detailRow;
}

function buildDetailContent(r) {
    // Score breakdown bars
    const total = r.total_score || 1;
    const bItems = [
        { label: '🏆 Awards',         val: r.award_points,               cls: 'positive', color: 'gold'   },
        { label: '⭐ Recs',            val: r.recommendation_points,       cls: 'positive', color: 'teal'   },
        { label: '🏅 Rank Boost',      val: r.rank_boost,                  cls: 'positive', color: 'purple' },
        { label: '🎯 First Timer',     val: r.first_time_conductor_boost,  cls: 'positive', color: 'blue'   },
        { label: '⏱️ Recent Penalty', val: -r.recent_conductor_penalty,   cls: 'negative', color: 'red'    },
        { label: '📈 Above Avg',       val: -r.above_average_penalty,      cls: 'negative', color: 'orange' },
    ].filter(x => x.val !== 0);

    const breakdownHtml = bItems.map(item => {
        const pct = Math.min(Math.abs(item.val) / Math.abs(total) * 120, 100).toFixed(1);
        return `
            <div class="rk-brow">
                <span class="rk-blabel">${item.label}</span>
                <div class="rk-bbar-wrap">
                    <div class="rk-bbar rk-bbar-${item.color}" style="width:${pct}%"></div>
                </div>
                <span class="rk-bval ${item.cls}">${item.val > 0 ? '+' : ''}${item.val}</span>
            </div>`;
    }).join('');

    // Awards
    let awardsHtml = '';
    const active = (r.award_details || []).filter(a => !a.expired);
    const expired = (r.award_details || []).filter(a => a.expired);
    if (r.award_details && r.award_details.length) {
        awardsHtml = `
            <div class="rk-detail-section">
                <div class="rk-dsection-header">
                    <h5>🏆 Awards <span class="rk-count">${active.length} active${expired.length ? `, ${expired.length} expired` : ''}</span></h5>
                    ${expired.length ? `<label class="rk-toggle-label"><input type="checkbox" class="show-inactive-cb" data-member-id="${r.member.id}"> Show expired</label>` : ''}
                </div>
                <div class="awards-compact-list" id="awards-list-${r.member.id}">
                    ${active.length ? active.map(a => awardItemHtml(a)).join('') : '<p class="no-data">No active awards.</p>'}
                </div>
            </div>`;
    } else {
        awardsHtml = `<div class="rk-detail-section"><h5>🏆 Awards</h5><p class="no-data">No awards yet.</p></div>`;
    }

    // Conductor stats
    const statsHtml = `
        <div class="rk-detail-section">
            <h5>📈 Conductor History</h5>
            <div class="rk-stat-row">
                <div class="rk-stat"><span class="rk-stat-val">${r.conductor_count}</span><span class="rk-stat-lbl">total runs</span></div>
                <div class="rk-stat"><span class="rk-stat-val">${r.last_conductor_date ? formatDate(r.last_conductor_date) : '—'}</span><span class="rk-stat-lbl">last date</span></div>
                ${r.days_since_last_conductor !== null ? `<div class="rk-stat"><span class="rk-stat-val">${r.days_since_last_conductor}d</span><span class="rk-stat-lbl">days ago</span></div>` : ''}
            </div>
        </div>`;

    // Timeline
    const timelineHtml = `
        <div class="rk-detail-section rk-timeline-section">
            <h5>📊 Timeline — Last 3 Months <span class="rk-trend" id="trend-${r.member.id}"></span></h5>
            <div class="rk-timeline-controls">
                <label><input type="checkbox" class="timeline-opt" id="show-reset-${r.member.id}" data-member-id="${r.member.id}" checked> Points</label>
                <label><input type="checkbox" class="timeline-opt" id="show-vs-${r.member.id}" data-member-id="${r.member.id}" checked> VS Points</label>
                <label><input type="checkbox" class="timeline-opt" id="show-power-${r.member.id}" data-member-id="${r.member.id}"> Power</label>
                <label><input type="checkbox" class="timeline-opt" id="show-donations-${r.member.id}" data-member-id="${r.member.id}"> Donations</label>
                <button class="rk-advanced-btn" onclick="toggleAdvanced(${r.member.id}, this)">More ▸</button>
            </div>
            <div class="rk-timeline-advanced" id="advanced-${r.member.id}">
                <label><input type="checkbox" class="timeline-opt" id="show-no-reset-${r.member.id}" data-member-id="${r.member.id}" checked> Cumulative (no resets)</label>
                <label><input type="checkbox" class="timeline-opt" id="show-breakdown-${r.member.id}" data-member-id="${r.member.id}"> Point breakdown</label>
                <label><input type="radio" name="scale-type-${r.member.id}" class="timeline-opt" data-member-id="${r.member.id}" value="linear" checked> Linear</label>
                <label><input type="radio" name="scale-type-${r.member.id}" class="timeline-opt" data-member-id="${r.member.id}" value="logarithmic"> Log</label>
            </div>
            <div class="timeline-loading" id="timeline-loading-${r.member.id}"><p>Loading timeline...</p></div>
            <canvas id="timeline-${r.member.id}" class="member-timeline-canvas" style="display:none"></canvas>
            <canvas id="vs-timeline-${r.member.id}" class="rk-power-canvas" style="display:none"></canvas>
            <canvas id="power-timeline-${r.member.id}" class="rk-power-canvas" style="display:none"></canvas>
            <canvas id="donation-timeline-${r.member.id}" class="rk-power-canvas" style="display:none"></canvas>
        </div>`;

    return `
        <div class="rk-detail-columns">
            <div class="rk-detail-col">
                <div class="rk-detail-section">
                    <h5>📊 Score Breakdown</h5>
                    <div class="rk-breakdown">${breakdownHtml || '<p class="no-data">No contributions.</p>'}</div>
                    <div class="rk-breakdown-total">Total: <strong>${r.total_score} pts</strong></div>
                </div>
                ${statsHtml}
            </div>
            <div class="rk-detail-col">${awardsHtml}</div>
        </div>
        ${timelineHtml}`;
}

function awardItemHtml(award, expired = false) {
    return `
        <div class="award-compact-item ${expired ? 'expired-award' : ''}">
            <span class="award-icon-compact">${getRankEmoji(award.rank)}</span>
            <div class="award-info-compact">
                <span class="award-type-compact">${escapeHtml(award.award_type)}${expired ? ' <em>(expired)</em>' : ''}</span>
                <span class="award-week-compact">${getWeeksAgo(award.week_date)}</span>
            </div>
            <span class="award-points-compact">+${award.points}</span>
        </div>`;
}

function formatLastRunCell(days) {
    if (days === null || days === undefined) return '<span class="rk-last-never">Never</span>';
    if (days <= 7)  return `<span class="rk-last-recent">${days}d</span>`;
    if (days <= 30) return `<span class="rk-last-mid">${days}d</span>`;
    const wks = Math.floor(days / 7);
    return `<span class="rk-last-old">${wks}w</span>`;
}

// ── Row toggle ────────────────────────────────────────────────────────────────

function toggleRow(memberId) {
    const detailRow = document.getElementById(`detail-row-${memberId}`);
    const dataRow = document.querySelector(`tr.rk-row[data-member-id="${memberId}"]`);
    if (!detailRow || !dataRow) return;

    const isOpen = dataRow.classList.contains('rk-row-open');
    dataRow.classList.toggle('rk-row-open', !isOpen);
    detailRow.classList.toggle('rk-detail-open', !isOpen);
    dataRow.querySelector('.rk-expand-icon').textContent = isOpen ? '▶' : '▼';

    if (!isOpen && !detailRow.dataset.chartLoaded) {
        detailRow.dataset.chartLoaded = 'true';
        const ranking = filteredRankings.find(r => r.member.id === memberId);
        if (ranking) loadSingleMemberTimeline(ranking);
    }
}

function toggleAdvanced(memberId, btn) {
    const el = document.getElementById(`advanced-${memberId}`);
    if (!el) return;
    const open = el.classList.toggle('rk-advanced-open');
    btn.textContent = open ? 'Less ▾' : 'More ▸';
}

// ── Rank change indicators ────────────────────────────────────────────────────

function getRankChangeIndicator(memberId, currentIndex) {
    if (!previousRankingsMap || previousRankingsMap[memberId] === undefined) return '';
    const diff = previousRankingsMap[memberId] - currentIndex;
    if (diff > 0) return `<span class="rank-change rank-up" title="Up ${diff}">▲${diff}</span>`;
    if (diff < 0) return `<span class="rank-change rank-down" title="Down ${Math.abs(diff)}">▼${Math.abs(diff)}</span>`;
    return '<span class="rank-change rank-same">—</span>';
}

// ── Analytics chart ───────────────────────────────────────────────────────────

function displayBreakdownChart(rankings) {
    applyChartTheme();
    if (breakdownChart) { breakdownChart.destroy(); breakdownChart = null; }

    const top10 = rankings.slice(0, 10);
    const ctx = document.getElementById('pointsBreakdownChart');
    if (!ctx) return;

    // Use theme-aware colors where possible
    const gold    = getCSSColor('--medal-gold') || '#ffd700';
    const teal    = getCSSColor('--bs-info')    || '#17a2b8';
    const purple  = '#9b6fcc';
    const blue    = getCSSColor('--info')       || '#3b82f6';
    const red     = getCSSColor('--danger')     || '#ef4444';

    breakdownChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: top10.map(r => r.member.name),
            datasets: [
                { label: 'Awards',       data: top10.map(r => r.award_points),              backgroundColor: gold   + 'cc' },
                { label: 'Recs',         data: top10.map(r => r.recommendation_points),      backgroundColor: teal   + 'cc' },
                { label: 'Rank Boost',   data: top10.map(r => r.rank_boost),                 backgroundColor: purple + 'cc' },
                { label: 'First Timer',  data: top10.map(r => r.first_time_conductor_boost), backgroundColor: blue   + 'cc' },
                { label: 'Penalties',    data: top10.map(r => -(r.recent_conductor_penalty + r.above_average_penalty)), backgroundColor: red + 'cc' },
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: true, position: 'bottom' } },
            scales: {
                x: { stacked: true },
                y: { stacked: true, title: { display: true, text: 'Points' } }
            }
        }
    });
}

// ── Formula modal ─────────────────────────────────────────────────────────────

function openFormulaModal() {
    document.getElementById('formula-modal').style.display = 'flex';
}

function closeFormulaModal() {
    document.getElementById('formula-modal').style.display = 'none';
}

function populateFormulaModal(settings, avgCount) {
    document.getElementById('formula-content').innerHTML = `
        <div class="system-info-grid">
            <div class="system-info-item"><span class="info-label">🥇 1st Award</span><span class="info-value">+${settings.award_first_points} pts</span></div>
            <div class="system-info-item"><span class="info-label">🥈 2nd Award</span><span class="info-value">+${settings.award_second_points} pts</span></div>
            <div class="system-info-item"><span class="info-label">🥉 3rd Award</span><span class="info-value">+${settings.award_third_points} pts</span></div>
            <div class="system-info-item"><span class="info-label">⭐ Recs</span><span class="info-value">5×√n pts / day</span></div>
            <div class="system-info-item"><span class="info-label">🏅 R4/R5 Boost</span><span class="info-value">${settings.r4r5_rank_boost} × 2^(days/7)</span></div>
            <div class="system-info-item"><span class="info-label">🎯 First Timer</span><span class="info-value">+${settings.first_time_conductor_boost} pts</span></div>
            <div class="system-info-item"><span class="info-label">⏱️ Recent Penalty</span><span class="info-value">−${settings.recent_conductor_penalty_days} pts max</span></div>
            <div class="system-info-item"><span class="info-label">📈 Above Avg Penalty</span><span class="info-value">−${settings.above_average_conductor_penalty} pts</span></div>
        </div>
        <p class="system-note">
            Awards and recommendations accumulate week over week until the member runs as conductor, then they reset to zero.
            <br>Avg conductor count: <strong>${avgCount.toFixed(2)}</strong>
            &ensp;·&ensp; 1 rec = 5pts, 4 recs = 10pts, 9 recs = 15pts
            &ensp;·&ensp; R4/R5 boost doubles every 7 days
        </p>`;
}

// ── Inactive awards toggle ────────────────────────────────────────────────────

function toggleInactiveAwards(memberId, show) {
    const ranking = filteredRankings.find(r => r.member.id === memberId);
    if (!ranking || !ranking.award_details) return;

    const list = document.getElementById(`awards-list-${memberId}`);
    if (!list) return;

    const items = show ? ranking.award_details : ranking.award_details.filter(a => !a.expired);
    list.innerHTML = items.length
        ? items.map(a => awardItemHtml(a, a.expired)).join('')
        : '<p class="no-data">No awards.</p>';
}

// ── Timeline chart (per-member, lazy) ─────────────────────────────────────────

async function loadSingleMemberTimeline(ranking) {
    if (!cachedTimelineData) {
        try {
            const resp = await fetch(`${API_BASE}/member-timelines?months=3`);
            if (!resp.ok) throw new Error('Failed to load timeline');
            cachedTimelineData = await resp.json();
        } catch (error) {
            console.error(error);
            const el = document.getElementById(`timeline-loading-${ranking.member.id}`);
            if (el) el.innerHTML = '<p class="error-message">Failed to load timeline.</p>';
            return;
        }
    }
    buildMemberChart(ranking, cachedTimelineData);
}

function rebuildSingleMemberChart(memberId) {
    if (!cachedTimelineData) return;
    const ranking = filteredRankings.find(r => r.member.id === memberId);
    if (ranking) buildMemberChart(ranking, cachedTimelineData);
}

function computeTrend(values) {
    if (!values || values.length < 4) return 'flat';
    const last4 = values.slice(-4);
    const first2avg = (last4[0] + last4[1]) / 2;
    const last2avg  = (last4[2] + last4[3]) / 2;
    if (first2avg === 0 && last2avg === 0) return 'flat';
    const pct = first2avg > 0 ? (last2avg - first2avg) / first2avg : (last2avg > 0 ? 1 : 0);
    if (pct > 0.05) return 'up';
    if (pct < -0.05) return 'down';
    return 'flat';
}

function buildMemberChart(ranking, timelineData) {
    const memberData = timelineData[ranking.member.id];
    const loadingEl   = document.getElementById(`timeline-loading-${ranking.member.id}`);
    const canvas      = document.getElementById(`timeline-${ranking.member.id}`);
    const vsCanvas    = document.getElementById(`vs-timeline-${ranking.member.id}`);
    const powerCanvas = document.getElementById(`power-timeline-${ranking.member.id}`);
    const donationCanvas = document.getElementById(`donation-timeline-${ranking.member.id}`);

    if (!memberData || memberData.dates.length === 0) {
        if (loadingEl) loadingEl.innerHTML = '<p class="empty">No timeline data available.</p>';
        return;
    }
    if (!canvas) return;

    if (loadingEl) loadingEl.style.display = 'none';
    canvas.style.display = '';

    // Destroy previous instances
    [canvas, vsCanvas, powerCanvas, donationCanvas].forEach(c => {
        if (!c) return;
        const existing = Chart.getChart(c);
        if (existing) existing.destroy();
        memberTimelineCharts = memberTimelineCharts.filter(ch => ch.canvas !== c);
    });

    const showReset     = document.getElementById(`show-reset-${ranking.member.id}`)?.checked ?? true;
    const showNoReset   = document.getElementById(`show-no-reset-${ranking.member.id}`)?.checked ?? true;
    const showBreakdown = document.getElementById(`show-breakdown-${ranking.member.id}`)?.checked ?? false;
    const showVS        = document.getElementById(`show-vs-${ranking.member.id}`)?.checked ?? true;
    const showPower     = document.getElementById(`show-power-${ranking.member.id}`)?.checked ?? false;
    const showDonations = document.getElementById(`show-donations-${ranking.member.id}`)?.checked ?? false;
    const scaleType     = document.querySelector(`input[name="scale-type-${ranking.member.id}"]:checked`)?.value || 'linear';

    applyChartTheme();

    // Trend indicator
    const trendEl = document.getElementById(`trend-${ranking.member.id}`);
    if (trendEl && memberData.points_with_reset) {
        const trend = computeTrend(memberData.points_with_reset);
        trendEl.className = `rk-trend rk-trend-${trend}`;
        const labels = { up: '↗ Trending up', down: '↘ Trending down', flat: '→ Stable' };
        trendEl.textContent = labels[trend] || '';
        trendEl.title = 'Based on last 4 weeks';
    }

    // ── Points + VS chart ─────────────────────────────────────────────────────
    const datasets = [];
    const accentColor = getCSSColor('--accent-primary');
    const infoColor   = getCSSColor('--bs-info') || '#17a2b8';

    if (showBreakdown) {
        const addStack = (label, data, color, stack) => {
            if (!data) return;
            datasets.push({ label, data, backgroundColor: color + '99', borderColor: color, borderWidth: 1, fill: true, stack, yAxisID: 'y', type: 'bar' });
        };
        const colors = { awards: '#ffd700', recs: '#17a2b8', boost: '#9b6fcc', first: '#3b82f6', penalty: '#ef4444', above: '#f59e0b' };
        if (showReset) {
            addStack('Awards',         memberData.awards_with_reset,              colors.awards,   'reset');
            addStack('Recs',           memberData.recommendations_with_reset,     colors.recs,     'reset');
            addStack('Rank Boost',     memberData.rank_boost_with_reset,          colors.boost,    'reset');
            addStack('First Timer',    memberData.first_time_boost_with_reset,    colors.first,    'reset');
            addStack('Recent Penalty', (memberData.recent_penalty_with_reset || []).map(v => -v),  colors.penalty, 'reset');
            addStack('Above Avg',      (memberData.above_avg_penalty_with_reset || []).map(v => -v), colors.above, 'reset');
        }
        if (showNoReset) {
            addStack('Awards (Cum)',   memberData.awards_cumulative,              colors.awards + '88', 'cumul');
            addStack('Recs (Cum)',     memberData.recommendations_cumulative,     colors.recs   + '88', 'cumul');
            addStack('Boost (Cum)',    memberData.rank_boost_cumulative,          colors.boost  + '88', 'cumul');
            addStack('First (Cum)',    memberData.first_time_boost_cumulative,    colors.first  + '88', 'cumul');
        }
    } else {
        if (showReset && memberData.points_with_reset) {
            datasets.push({ label: 'With Resets', data: memberData.points_with_reset, borderColor: accentColor, backgroundColor: accentColor + '22', borderWidth: 2, fill: true, tension: 0.2, yAxisID: 'y', type: 'line' });
        }
        if (showNoReset && memberData.points_cumulative) {
            datasets.push({ label: 'Cumulative', data: memberData.points_cumulative, borderColor: infoColor, backgroundColor: infoColor + '22', borderWidth: 2, fill: true, tension: 0.2, yAxisID: 'y', type: 'line' });
        }
    }

    if (datasets.length) {
        // Conductor annotations
        const annotations = {};
        (memberData.conductor_dates || []).forEach((d, idx) => {
            const wi = memberData.dates.indexOf(d);
            if (wi !== -1) {
                annotations[`c-${idx}`] = {
                    type: 'line', xMin: wi, xMax: wi,
                    borderColor: 'rgba(255,159,64,0.8)', borderWidth: 2, borderDash: [5, 5],
                    label: { display: true, content: '🚂', position: 'start', yAdjust: -10, font: { size: 13 } }
                };
            }
        });

        const chart = new Chart(canvas, {
            type: 'line',
            data: { labels: memberData.dates, datasets },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                plugins: {
                    legend: { display: true, position: 'top', labels: { font: { size: 11 }, boxWidth: 12 } },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                        callbacks: {
                            label: item => {
                                const v = item.parsed.y;
                                if (v === null || v === undefined || v === 0) return null;
                                return ` ${item.dataset.label}: ${v > 0 ? '+' : ''}${v}`;
                            }
                        }
                    },
                    annotation: { annotations }
                },
                scales: {
                    x: { title: { display: false }, ticks: { maxRotation: 45, minRotation: 45, font: { size: 9 } }, stacked: showBreakdown },
                    y: { type: scaleType, beginAtZero: true, position: 'left', title: { display: true, text: 'Points', font: { size: 11 } }, ticks: { font: { size: 10 } }, stacked: showBreakdown },
                },
                interaction: { mode: 'nearest', axis: 'x', intersect: false }
            }
        });
        memberTimelineCharts.push(chart);
    }

    // ── VS mini-chart ─────────────────────────────────────────────────────────
    if (vsCanvas) {
        const hasVS = memberData.vs_weekly_total && memberData.vs_weekly_total.some(v => v > 0);
        if (showVS && hasVS) {
            vsCanvas.style.display = '';
            const vsTarget = currentData?.settings?.vs_points_weekly_target || 0;
            const vsAnnotations = {};
            if (vsTarget > 0) {
                vsAnnotations.target = {
                    type: 'line', yMin: vsTarget, yMax: vsTarget,
                    borderColor: 'rgba(239,68,68,0.7)', borderWidth: 1.5, borderDash: [4, 4],
                    label: { display: true, content: `Target: ${vsTarget.toLocaleString()}`, position: 'end', font: { size: 10 } }
                };
            }
            const vsChart = new Chart(vsCanvas, {
                type: 'bar',
                data: {
                    labels: memberData.dates,
                    datasets: [{
                        label: 'VS Points',
                        data: memberData.vs_weekly_total,
                        backgroundColor: memberData.vs_weekly_total.map(v =>
                            vsTarget > 0
                                ? (v >= vsTarget ? 'rgba(34,197,94,0.55)' : 'rgba(239,68,68,0.45)')
                                : 'rgba(155,111,204,0.55)'
                        ),
                        borderColor: memberData.vs_weekly_total.map(v =>
                            vsTarget > 0
                                ? (v >= vsTarget ? '#22c55e' : '#ef4444')
                                : '#9b6fcc'
                        ),
                        borderWidth: 1,
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            callbacks: {
                                title: items => items[0]?.label || '',
                                label: item => ` VS: ${item.parsed.y.toLocaleString()}${vsTarget > 0 ? ' / ' + vsTarget.toLocaleString() + ' target' : ''}`
                            }
                        },
                        annotation: { annotations: vsAnnotations }
                    },
                    scales: {
                        x: { ticks: { maxRotation: 45, minRotation: 45, font: { size: 9 } } },
                        y: {
                            beginAtZero: true,
                            title: { display: true, text: 'VS Pts', font: { size: 11 } },
                            ticks: { font: { size: 10 }, callback: v => v >= 1000 ? (v / 1000).toFixed(0) + 'K' : v }
                        }
                    },
                    interaction: { mode: 'nearest', axis: 'x', intersect: false }
                }
            });
            memberTimelineCharts.push(vsChart);
        } else {
            vsCanvas.style.display = 'none';
        }
    }

    // ── Power mini-chart ──────────────────────────────────────────────────────
    if (powerCanvas) {
        const hasPower = memberData.power && memberData.power.some(v => v > 0);
        if (showPower && hasPower) {
            powerCanvas.style.display = '';
            const powerChart = new Chart(powerCanvas, {
                type: 'line',
                data: {
                    labels: memberData.dates,
                    datasets: [{
                        label: 'Power',
                        data: memberData.power,
                        borderColor: accentColor,
                        backgroundColor: accentColor + '22',
                        borderWidth: 2,
                        fill: true,
                        tension: 0.2,
                        pointRadius: 2,
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            mode: 'index',
                            intersect: false,
                            callbacks: {
                                title: items => items[0]?.label || '',
                                label: item => ` Power: ${formatPowerShort(item.parsed.y)}`
                            }
                        },
                        annotation: { annotations: {} }
                    },
                    scales: {
                        x: { ticks: { maxRotation: 45, minRotation: 45, font: { size: 9 } } },
                        y: {
                            beginAtZero: false,
                            title: { display: true, text: 'Power', font: { size: 11 } },
                            ticks: { font: { size: 10 }, callback: v => formatPowerShort(v) }
                        }
                    },
                    interaction: { mode: 'nearest', axis: 'x', intersect: false }
                }
            });
            memberTimelineCharts.push(powerChart);
        } else {
            powerCanvas.style.display = 'none';
        }
    }

    // ── Donations mini-chart ─────────────────────────────────────────────────
    if (donationCanvas) {
        const hasDonations = memberData.donation_weekly_total && memberData.donation_weekly_total.some(v => v > 0);
        if (showDonations && hasDonations) {
            donationCanvas.style.display = '';
            const donationTarget = currentData?.settings?.donation_weekly_target || 0;
            const donationAnnotations = {};
            if (donationTarget > 0) {
                donationAnnotations.target = {
                    type: 'line', yMin: donationTarget, yMax: donationTarget,
                    borderColor: 'rgba(239,68,68,0.7)', borderWidth: 1.5, borderDash: [4, 4],
                    label: { display: true, content: `Target: ${donationTarget.toLocaleString()}`, position: 'end', font: { size: 10 } }
                };
            }
            const donationChart = new Chart(donationCanvas, {
                type: 'bar',
                data: {
                    labels: memberData.dates,
                    datasets: [{
                        label: 'Tech Donations',
                        data: memberData.donation_weekly_total,
                        backgroundColor: memberData.donation_weekly_total.map(v =>
                            donationTarget > 0
                                ? (v >= donationTarget ? 'rgba(34,197,94,0.55)' : 'rgba(239,68,68,0.45)')
                                : 'rgba(245,158,11,0.55)'
                        ),
                        borderColor: memberData.donation_weekly_total.map(v =>
                            donationTarget > 0
                                ? (v >= donationTarget ? '#22c55e' : '#ef4444')
                                : '#f59e0b'
                        ),
                        borderWidth: 1,
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            callbacks: {
                                title: items => items[0]?.label || '',
                                label: item => ` Donations: ${item.parsed.y.toLocaleString()}${donationTarget > 0 ? ' / ' + donationTarget.toLocaleString() + ' target' : ''}`
                            }
                        },
                        annotation: { annotations: donationAnnotations }
                    },
                    scales: {
                        x: { ticks: { maxRotation: 45, minRotation: 45, font: { size: 9 } } },
                        y: {
                            beginAtZero: true,
                            title: { display: true, text: 'Donations', font: { size: 11 } },
                            ticks: { font: { size: 10 }, callback: v => v >= 1000 ? (v / 1000).toFixed(0) + 'K' : v }
                        }
                    },
                    interaction: { mode: 'nearest', axis: 'x', intersect: false }
                }
            });
            memberTimelineCharts.push(donationChart);
        } else {
            donationCanvas.style.display = 'none';
        }
    }
}

// ── CSV export ────────────────────────────────────────────────────────────────

function exportCSV() {
    if (!filteredRankings || !filteredRankings.length) return;
    const headers = ['Rank','Name','Alliance Rank','Total Score','Award Points','Rec Points','Rank Boost','First Timer','Recent Penalty','Above Avg Penalty','Conductor Count','Last Conductor Date'];
    const rows = filteredRankings.map((r, i) => [
        i + 1,
        '"' + r.member.name.replace(/"/g, '""') + '"',
        r.member.rank,
        r.total_score,
        r.award_points,
        r.recommendation_points,
        r.rank_boost,
        r.first_time_conductor_boost,
        r.recent_conductor_penalty,
        r.above_average_penalty,
        r.conductor_count,
        r.last_conductor_date || ''
    ]);
    const csv = [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8;' }));
    Object.assign(document.createElement('a'), { href: url, download: `rankings_${new Date().toISOString().slice(0,10)}.csv` }).click();
    URL.revokeObjectURL(url);
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatDate(dateStr) {
    if (!dateStr) return '—';
    const [y, m, d] = dateStr.split('-');
    return `${d}/${m}/${y}`;
}

function formatPowerShort(v) {
    if (!v) return '0';
    if (v >= 1e9) return (v / 1e9).toFixed(1) + 'B';
    if (v >= 1e6) return (v / 1e6).toFixed(1) + 'M';
    if (v >= 1e3) return (v / 1e3).toFixed(0) + 'K';
    return String(v);
}

function getRankEmoji(rank) {
    return { 1: '🥇', 2: '🥈', 3: '🥉' }[rank] || '🏅';
}

function getWeeksAgo(weekDate) {
    const award = new Date(weekDate);
    const today = new Date();
    const day = today.getDay();
    const monday = new Date(today);
    monday.setDate(today.getDate() - (day === 0 ? 6 : day - 1));
    monday.setHours(0, 0, 0, 0);
    const weeks = Math.floor((monday - award) / (7 * 24 * 60 * 60 * 1000));
    if (weeks <= 0) return 'This week';
    if (weeks === 1) return 'Last week';
    return `${weeks} weeks ago`;
}

function escapeHtml(text) {
    const d = document.createElement('div');
    d.textContent = text;
    return d.innerHTML;
}

function debounce(fn, delay) {
    let t;
    return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), delay); };
}

// ── Init ──────────────────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', async () => {
    const auth = await requireAuth();
    if (!auth) return;
    await loadRankings();

    document.getElementById('filter-name').addEventListener('input', debounce(applyFiltersAndSort, 300));
    document.getElementById('sort-by').addEventListener('change', applyFiltersAndSort);
    document.getElementById('refresh-btn').addEventListener('click', loadRankings);
    document.getElementById('export-csv-btn').addEventListener('click', exportCSV);

    document.getElementById('rank-chips').addEventListener('click', e => {
        const chip = e.target.closest('.filter-chip');
        if (chip) setActiveChip(chip.dataset.rank);
    });

    document.getElementById('formula-modal').addEventListener('click', e => {
        if (e.target === e.currentTarget) closeFormulaModal();
    });

    // Render analytics chart when the collapsible opens
    document.querySelector('.rk-analytics').addEventListener('toggle', e => {
        if (e.target.open && currentData) displayBreakdownChart(currentData.rankings);
    });
});
