const API_URL = '/api/awards';
const MEMBERS_URL = '/api/members';

const AWARD_TYPES = [
    'Alliance Champion',
    'Star of Desert Storm',
    'Soldier Crusher',
    'Divine Healer',
    'Great Destroyer',
    'Grind King',
    'Alliance Exercise MVP',
    'Doom Elite Slayer',
    'Best Manager',
    'Alliance Sponsor',
    'Firefighting Leader',
    'Excavator Radar',
    'Shining Star',
    'MVP',
    'Devil Trainer',
    'Trial Assist King',
    'Good Helper'
];

let currentWeekDate = null;
let allMembers = [];
let currentAwards = {};
let allHistory = [];
let currentUsername = '';
let activeAwardTypes = new Set(AWARD_TYPES); // All awards active by default

// Check authentication
async function checkAuth() {
    try {
        const response = await fetch('/api/check-auth');
        const data = await response.json();
        
        if (!data.authenticated) {
            window.location.href = '/login.html';
            return false;
        }
        
        currentUsername = data.username;
        let displayText = `👤 ${currentUsername}`;
        if (data.rank) {
            displayText += ` (${data.rank})`;
        }
        document.getElementById('username-display').textContent = displayText;
        
        return true;
    } catch (error) {
        console.error('Auth check error:', error);
        window.location.href = '/login.html';
        return false;
    }
}

// Logout handler
document.getElementById('logout-btn').addEventListener('click', async () => {
    if (!confirm('Are you sure you want to logout?')) {
        return;
    }
    
    try {
        await fetch('/api/logout', { method: 'POST' });
        window.location.href = '/login.html';
    } catch (error) {
        console.error('Logout error:', error);
        alert('Error logging out. Please try again.');
    }
});

// Get most recent Monday
function getMostRecentMonday(date = new Date()) {
    const d = new Date(date);
    const day = d.getDay();
    const diff = day === 0 ? 6 : day - 1; // If Sunday, go back 6 days, else go back to Monday
    d.setDate(d.getDate() - diff);
    return d;
}

// Format date as YYYY-MM-DD
function formatDate(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
}

// Format date for display (European style: dd/mm/yyyy)
function formatDisplayDate(dateStr) {
    const date = new Date(dateStr + 'T00:00:00');
    const day = String(date.getDate()).padStart(2, '0');
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const year = date.getFullYear();
    
    return `Week of ${day}/${month}/${year}`;
}

// Initialize current week
function initializeWeek() {
    currentWeekDate = getMostRecentMonday();
    updateWeekDisplay();
}

// Update week display
function updateWeekDisplay() {
    document.getElementById('week-display').textContent = formatDisplayDate(formatDate(currentWeekDate));
}

// Navigate weeks functions
function navigatePrevWeek() {
    currentWeekDate.setDate(currentWeekDate.getDate() - 7);
    updateWeekDisplay();
    loadAwards();
}

function navigateNextWeek() {
    currentWeekDate.setDate(currentWeekDate.getDate() + 7);
    updateWeekDisplay();
    loadAwards();
}

// Load members
async function loadMembers() {
    try {
        const response = await fetch(MEMBERS_URL);
        allMembers = await response.json();
        allMembers.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
    } catch (error) {
        console.error('Error loading members:', error);
    }
}

// Load awards for current week
async function loadAwards() {
    const weekDate = formatDate(currentWeekDate);
    
    try {
        const response = await fetch(`${API_URL}?week=${weekDate}`);
        const awards = await response.json();
        
        // Organize awards by type and rank
        currentAwards = {};
        awards.forEach(award => {
            if (!currentAwards[award.award_type]) {
                currentAwards[award.award_type] = {};
            }
            currentAwards[award.award_type][award.rank] = award.member_id;
        });
        
        renderAwardsForm();
    } catch (error) {
        console.error('Error loading awards:', error);
    }
}

// Save current form state
function saveFormState() {
    const formState = {};
    document.querySelectorAll('.member-select').forEach(select => {
        const award = select.dataset.award;
        const rank = select.dataset.rank;
        const key = `${award}|${rank}`;
        formState[key] = select.value;
    });
    return formState;
}

// Restore form state
function restoreFormState(formState) {
    if (!formState) return;
    
    document.querySelectorAll('.member-select').forEach(select => {
        const award = select.dataset.award;
        const rank = select.dataset.rank;
        const key = `${award}|${rank}`;
        if (formState[key]) {
            select.value = formState[key];
        }
    });
}

// Render awards form
function renderAwardsForm(preserveFormState = false) {
    const grid = document.getElementById('awards-grid');
    
    // Save current form state before re-rendering (only if preserving)
    const formState = preserveFormState ? saveFormState() : null;
    
    let html = '';
    
    // Get active award types as array and sort
    const activeTypes = Array.from(activeAwardTypes).sort();
    const inactiveTypes = AWARD_TYPES.filter(type => !activeAwardTypes.has(type)).sort();
    
    activeTypes.forEach(awardType => {
        html += `<div class="award-card">`;
        html += `<div class="award-header">`;
        html += `<h4 class="award-title">🏆 ${awardType}</h4>`;
        html += `<button class="toggle-award-btn" data-award="${awardType}" title="Hide this award">✕</button>`;
        html += `</div>`;
        
        for (let rank = 1; rank <= 3; rank++) {
            const selectedMemberId = currentAwards[awardType]?.[rank] || '';
            const rankLabel = rank === 1 ? '🥇 1st Place' : rank === 2 ? '🥈 2nd Place' : '🥉 3rd Place';
            
            html += `<div class="award-position">`;
            html += `<label>${rankLabel}</label>`;
            html += `<input type="text" class="member-search" placeholder="🔍 Search member..." data-award="${awardType}" data-rank="${rank}">`;
            html += `<select class="member-select" data-award="${awardType}" data-rank="${rank}">`;
            html += `<option value="">-- Select Member --</option>`;
            
            allMembers.forEach(member => {
                const selected = member.id === selectedMemberId ? 'selected' : '';
                html += `<option value="${member.id}" ${selected} data-name="${member.name.toLowerCase()}">${escapeHtml(member.name)} (${member.rank})</option>`;
            });
            
            html += `</select>`;
            html += `</div>`;
        }
        
        html += `</div>`;
    });
    
    // Show inactive awards section if any
    if (inactiveTypes.length > 0) {
        html += `<div class="inactive-awards-section">`;
        html += `<h4 class="inactive-title">Inactive Awards (Click to activate)</h4>`;
        html += `<div class="inactive-awards-chips">`;
        inactiveTypes.forEach(awardType => {
            html += `<button class="inactive-award-chip" data-award="${awardType}">${awardType}</button>`;
        });
        html += `</div>`;
        html += `</div>`;
    }
    
    grid.innerHTML = html;
    
    // Restore form state after rendering (only if preserving)
    if (preserveFormState) {
        restoreFormState(formState);
    }
    
    // Setup toggle buttons
    document.querySelectorAll('.toggle-award-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            const award = e.target.dataset.award;
            activeAwardTypes.delete(award);
            renderAwardsForm(true); // Preserve form state when hiding
        });
    });
    
    // Setup inactive award chips
    document.querySelectorAll('.inactive-award-chip').forEach(chip => {
        chip.addEventListener('click', (e) => {
            e.preventDefault();
            const award = e.target.dataset.award;
            activeAwardTypes.add(award);
            renderAwardsForm(true); // Preserve form state when showing
        });
    });
    
    // Setup search filters for all dropdowns
    setupSearchFilters();
    
    // Update toggle button text
    updateToggleButton();
}

// Setup search filters
function setupSearchFilters() {
    const searchInputs = document.querySelectorAll('.member-search');
    
    searchInputs.forEach(input => {
        input.addEventListener('input', (e) => {
            const award = e.target.dataset.award;
            const rank = e.target.dataset.rank;
            const searchTerm = e.target.value.toLowerCase().trim();
            const select = document.querySelector(`select[data-award="${award}"][data-rank="${rank}"]`);
            
            filterSelectOptions(select, searchTerm);
        });
    });
}

// Filter select options
function filterSelectOptions(selectElement, searchTerm) {
    const options = selectElement.options;
    let visibleCount = 0;
    
    for (let i = 0; i < options.length; i++) {
        const option = options[i];
        if (i === 0) { // Keep first option visible
            option.style.display = '';
            continue;
        }
        
        const name = option.dataset.name || '';
        
        if (name.includes(searchTerm)) {
            option.style.display = '';
            visibleCount++;
        } else {
            option.style.display = 'none';
        }
    }
    
    // Auto-select if only one visible option
    if (visibleCount === 1 && searchTerm) {
        for (let i = 1; i < options.length; i++) {
            if (options[i].style.display !== 'none') {
                selectElement.selectedIndex = i;
                break;
            }
        }
    }
}

// Save awards
async function saveAwards() {
    const weekDate = formatDate(currentWeekDate);
    
    // Check if there are hidden awards with saved data
    const hiddenAwardsWithData = [];
    const inactiveTypes = AWARD_TYPES.filter(type => !activeAwardTypes.has(type));
    inactiveTypes.forEach(awardType => {
        if (currentAwards[awardType]) {
            const ranks = Object.keys(currentAwards[awardType]);
            if (ranks.length > 0) {
                hiddenAwardsWithData.push(awardType);
            }
        }
    });
    
    // Warn user if saving will delete hidden awards
    if (hiddenAwardsWithData.length > 0) {
        const message = `Warning: ${hiddenAwardsWithData.length} hidden award(s) have saved data that will be deleted:\n\n${hiddenAwardsWithData.join(', ')}\n\nOnly visible awards will be saved. Continue?`;
        if (!confirm(message)) {
            return;
        }
    }
    
    const awards = [];
    
    // Collect awards from visible form only
    const selects = document.querySelectorAll('.member-select');
    selects.forEach(select => {
        const memberId = parseInt(select.value);
        if (memberId) {
            awards.push({
                award_type: select.dataset.award,
                rank: parseInt(select.dataset.rank),
                member_id: memberId
            });
        }
    });
    
    try {
        const response = await fetch(API_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ week_date: weekDate, awards })
        });
        
        if (!response.ok) {
            throw new Error('Failed to save awards');
        }
        
        alert('✓ Awards saved successfully!');
        await loadHistory();
    } catch (error) {
        console.error('Error saving awards:', error);
        alert('Failed to save awards: ' + error.message);
    }
}

// Clear awards
async function clearAwards() {
    const weekDate = formatDate(currentWeekDate);
    
    if (!confirm('Clear all awards for this week? This cannot be undone.')) {
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/${weekDate}`, {
            method: 'DELETE'
        });
        
        if (!response.ok && response.status !== 204) {
            throw new Error('Failed to clear awards');
        }
        
        currentAwards = {};
        renderAwardsForm();
        await loadHistory();
        alert('✓ Awards cleared for this week.');
    } catch (error) {
        console.error('Error clearing awards:', error);
        alert('Failed to clear awards: ' + error.message);
    }
}

// Load history
async function loadHistory() {
    try {
        const response = await fetch(API_URL);
        allHistory = await response.json();
        
        // Get unique weeks
        const weeks = [...new Set(allHistory.map(a => a.week_date))].sort().reverse();
        
        // Populate week filter
        const weekFilter = document.getElementById('week-filter');
        weekFilter.innerHTML = '<option value="">All Weeks</option>';
        weeks.forEach(week => {
            const option = document.createElement('option');
            option.value = week;
            option.textContent = formatDisplayDate(week);
            weekFilter.appendChild(option);
        });
        
        renderHistory();
    } catch (error) {
        console.error('Error loading history:', error);
        document.getElementById('history-content').innerHTML = 
            '<p class="empty">Error loading history.</p>';
    }
}

// Render history
function renderHistory() {
    const searchTerm = document.getElementById('history-search').value.toLowerCase().trim();
    const weekFilter = document.getElementById('week-filter').value;
    
    let filtered = allHistory;
    
    if (weekFilter) {
        filtered = filtered.filter(a => a.week_date === weekFilter);
    }
    
    if (searchTerm) {
        filtered = filtered.filter(a => a.member_name.toLowerCase().includes(searchTerm));
    }
    
    const content = document.getElementById('history-content');
    
    if (filtered.length === 0) {
        content.innerHTML = '<p class="empty">No awards found.</p>';
        return;
    }
    
    // Group by week
    const groupedByWeek = {};
    filtered.forEach(award => {
        if (!groupedByWeek[award.week_date]) {
            groupedByWeek[award.week_date] = {};
        }
        if (!groupedByWeek[award.week_date][award.award_type]) {
            groupedByWeek[award.week_date][award.award_type] = [];
        }
        groupedByWeek[award.week_date][award.award_type].push(award);
    });
    
    let html = '';
    
    Object.keys(groupedByWeek).sort().reverse().forEach(week => {
        html += `<div class="week-history">`;
        html += `<h4 class="week-history-title">📅 ${formatDisplayDate(week)}</h4>`;
        html += `<div class="awards-history-grid">`;
        
        Object.keys(groupedByWeek[week]).sort().forEach(awardType => {
            const awards = groupedByWeek[week][awardType].sort((a, b) => a.rank - b.rank);
            
            html += `<div class="history-award-card">`;
            html += `<h5 class="history-award-title">${awardType}</h5>`;
            
            awards.forEach(award => {
                const rankEmoji = award.rank === 1 ? '🥇' : award.rank === 2 ? '🥈' : '🥉';
                html += `<div class="history-award-item rank-${award.rank}">`;
                html += `${rankEmoji} ${escapeHtml(award.member_name)}`;
                html += `</div>`;
            });
            
            html += `</div>`;
        });
        
        html += `</div>`;
        html += `</div>`;
    });
    
    content.innerHTML = html;
}

// Escape HTML
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    const isAuthenticated = await checkAuth();
    if (isAuthenticated) {
        await loadMembers();
        initializeWeek();
        await loadAwards();
        await loadHistory();
        
        // Set up event listeners after DOM is ready
        document.getElementById('prev-week').addEventListener('click', navigatePrevWeek);
        document.getElementById('next-week').addEventListener('click', navigateNextWeek);
        document.getElementById('save-awards-btn').addEventListener('click', saveAwards);
        document.getElementById('clear-awards-btn').addEventListener('click', clearAwards);
        document.getElementById('history-search').addEventListener('input', renderHistory);
        document.getElementById('week-filter').addEventListener('change', renderHistory);
        
        // Toggle all awards button
        document.getElementById('toggle-all-awards-btn').addEventListener('click', () => {
            if (activeAwardTypes.size === AWARD_TYPES.length) {
                // Hide all
                activeAwardTypes.clear();
            } else {
                // Show all
                activeAwardTypes = new Set(AWARD_TYPES);
            }
            renderAwardsForm(true); // Preserve form state when toggling all
            updateToggleButton();
        });
    }
});

// Update toggle button text
function updateToggleButton() {
    const btn = document.getElementById('toggle-all-awards-btn');
    if (btn) {
        if (activeAwardTypes.size === AWARD_TYPES.length) {
            btn.textContent = '⚙️ Hide All Awards';
        } else if (activeAwardTypes.size === 0) {
            btn.textContent = '⚙️ Show All Awards';
        } else {
            btn.textContent = `⚙️ Show All (${activeAwardTypes.size}/${AWARD_TYPES.length})`;
        }
    }
}
