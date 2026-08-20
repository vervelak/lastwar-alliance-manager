// Settings page
const SETTINGS_URL = `${API_BASE}/settings`;

let isR5OrAdmin = false;
let currentTimezones = ["Europe/London"];

// Load settings
async function loadSettings() {
    try {
        const response = await fetch(SETTINGS_URL);
        if (!response.ok) throw new Error('Failed to load settings');
        
        const settings = await response.json();
        
        // Alliance branding
        document.getElementById('alliance-name').value = settings.alliance_name || 'Last War: Survival';
        document.getElementById('alliance-short-name').value = settings.alliance_short_name || 'LWS';
        
        document.getElementById('award-first').value = settings.award_first_points;
        document.getElementById('award-second').value = settings.award_second_points;
        document.getElementById('award-third').value = settings.award_third_points;
        document.getElementById('recent-conductor-days').value = settings.recent_conductor_penalty_days;
        document.getElementById('above-average-penalty').value = settings.above_average_conductor_penalty;
        document.getElementById('r4r5-rank-boost').value = settings.r4r5_rank_boost;
        document.getElementById('first-time-boost').value = settings.first_time_conductor_boost || 5;
        document.getElementById('schedule-message-template').value = settings.schedule_message_template || 'Train Schedule - Week {WEEK}\n\n{SCHEDULES}\n\nNext in line:\n{NEXT_3}';
        document.getElementById('daily-message-template').value = settings.daily_message_template || 'Daily train reminder for {DAY}, {DATE}:\n🚂 Conductor: {CONDUCTOR_NAME} - Please be online at {CONDUCTOR_TIME}\n🔄 Backup: {BACKUP_NAME} - Please be ready at {BACKUP_TIME}\n\nAsk in alliance chat for the train to be assigned. Thanks for keeping the train golden!';
        
        // Server timezone
        document.getElementById('server-timezone').value = settings.server_timezone || 'UTC';
        
        // Train times
        document.getElementById('conductor-time').value = settings.conductor_time || '15:00';
        document.getElementById('backup-time').value = settings.backup_time || '16:30';
        
        // Display timezones
        currentTimezones = JSON.parse(settings.display_timezones || '["Europe/London"]');
        renderTimezoneTags();
        updateTimePreview();
        
        // VS targets
        document.getElementById('vs-daily-target').value = settings.vs_points_daily_target || 0;
        document.getElementById('vs-weekly-target').value = settings.vs_points_weekly_target || 0;

        // Donation targets
        document.getElementById('donation-weekly-target').value = settings.donation_weekly_target || 0;
        document.getElementById('conductor-donation-threshold').value =
            settings.conductor_donation_threshold !== undefined ? settings.conductor_donation_threshold : 60000;
        document.getElementById('vip-donation-threshold').value =
            settings.vip_donation_threshold !== undefined ? settings.vip_donation_threshold : 40000;

        // Recruitment requirements
        document.getElementById('min-power').value = settings.min_power || 0;
        document.getElementById('min-hq-level').value = settings.min_hq_level || 0;

        // Power tracking
        const powerTrackingEnabled = settings.power_tracking_enabled || false;
        document.getElementById('power-tracking-enabled').checked = powerTrackingEnabled;
        togglePowerUploadSection(powerTrackingEnabled);

        // VIP seat
        const vipSeatEnabled = settings.vip_seat_enabled !== undefined ? settings.vip_seat_enabled : true;
        document.getElementById('vip-seat-enabled').checked = vipSeatEnabled;

        // Marshal Guard
        const mgEnabled = settings.marshal_guard_enabled !== undefined ? settings.marshal_guard_enabled : true;
        document.getElementById('marshal-guard-enabled').checked = mgEnabled;

        // Desert Storm
        const dsEnabled = settings.desert_storm_enabled !== undefined ? settings.desert_storm_enabled : true;
        document.getElementById('desert-storm-enabled').checked = dsEnabled;

        // Zombie Siege
        const zsEnabled = settings.zombie_siege_enabled !== undefined ? settings.zombie_siege_enabled : true;
        document.getElementById('zombie-siege-enabled').checked = zsEnabled;
    } catch (error) {
        console.error('Error loading settings:', error);
        showToast('Failed to load settings.', 'error');
    }
}

// Save settings
document.getElementById('settings-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    if (!isR5OrAdmin) {
        showToast('You do not have permission to modify settings. Only R5 members and admins can do this.', 'error');
        return;
    }
    
    const settings = {
        alliance_name: document.getElementById('alliance-name').value,
        alliance_short_name: document.getElementById('alliance-short-name').value,
        award_first_points: parseInt(document.getElementById('award-first').value),
        award_second_points: parseInt(document.getElementById('award-second').value),
        award_third_points: parseInt(document.getElementById('award-third').value),
        recent_conductor_penalty_days: parseInt(document.getElementById('recent-conductor-days').value),
        above_average_conductor_penalty: parseInt(document.getElementById('above-average-penalty').value),
        r4r5_rank_boost: parseInt(document.getElementById('r4r5-rank-boost').value),
        first_time_conductor_boost: parseInt(document.getElementById('first-time-boost').value),
        schedule_message_template: document.getElementById('schedule-message-template').value,
        daily_message_template: document.getElementById('daily-message-template').value,
        vs_points_daily_target: parseInt(document.getElementById('vs-daily-target').value) || 0,
        vs_points_weekly_target: parseInt(document.getElementById('vs-weekly-target').value) || 0,
        donation_weekly_target: parseInt(document.getElementById('donation-weekly-target').value) || 0,
        conductor_donation_threshold: parseInt(document.getElementById('conductor-donation-threshold').value) || 0,
        vip_donation_threshold: parseInt(document.getElementById('vip-donation-threshold').value) || 0,
        min_power: parseInt(document.getElementById('min-power').value) || 0,
        min_hq_level: parseInt(document.getElementById('min-hq-level').value) || 0,
        power_tracking_enabled: document.getElementById('power-tracking-enabled').checked,
        vip_seat_enabled: document.getElementById('vip-seat-enabled').checked,
        marshal_guard_enabled: document.getElementById('marshal-guard-enabled').checked,
        desert_storm_enabled: document.getElementById('desert-storm-enabled').checked,
        zombie_siege_enabled: document.getElementById('zombie-siege-enabled').checked,
        server_timezone: document.getElementById('server-timezone').value,
        conductor_time: document.getElementById('conductor-time').value,
        backup_time: document.getElementById('backup-time').value,
        display_timezones: JSON.stringify(currentTimezones)
    };
    
    try {
        const response = await fetch(SETTINGS_URL, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(settings)
        });
        
        if (!response.ok) {
            const error = await response.text();
            throw new Error(error);
        }
        
        showToast('Settings saved successfully!', 'success');
    } catch (error) {
        console.error('Error saving settings:', error);
        showToast('Failed to save settings: ' + error.message, 'error');
    }
});

// Reset to defaults
document.getElementById('reset-btn').addEventListener('click', async () => {
    const confirmed = await showConfirm('Reset all settings to default values?', 'Reset Settings', 'Reset', 'Cancel', true);
    if (confirmed) {
        document.getElementById('alliance-name').value = 'Last War: Survival';
        document.getElementById('alliance-short-name').value = 'LWS';
        document.getElementById('award-first').value = 3;
        document.getElementById('award-second').value = 2;
        document.getElementById('award-third').value = 1;
        document.getElementById('recent-conductor-days').value = 30;
        document.getElementById('above-average-penalty').value = 10;
        document.getElementById('r4r5-rank-boost').value = 5;
        document.getElementById('first-time-boost').value = 5;
        document.getElementById('schedule-message-template').value = 'Train Schedule - Week {WEEK}\n\n{SCHEDULES}\n\nNext in line:\n{NEXT_3}';
        document.getElementById('daily-message-template').value = 'Daily train reminder for {DAY}, {DATE}:\\n🚂 Conductor: {CONDUCTOR} - Please be online at {CONDUCTOR_TIME}\\n🔄 Backup: {BACKUP} - Please be ready at {BACKUP_TIME}\\n\\nAsk in alliance chat for the train to be assigned. Thanks for keeping the train golden!';
        document.getElementById('server-timezone').value = 'Etc/GMT+2';
        document.getElementById('conductor-time').value = '15:00';
        document.getElementById('backup-time').value = '16:30';
        currentTimezones = ['Europe/London'];
        renderTimezoneTags();
        updateTimePreview();
        document.getElementById('vs-daily-target').value = 0;
        document.getElementById('vs-weekly-target').value = 0;
        document.getElementById('power-tracking-enabled').checked = false;
        document.getElementById('vip-seat-enabled').checked = true;
    }
});

// Power tracking toggle
function togglePowerUploadSection(enabled) {
    const uploadLink = document.getElementById('power-upload-link');
    if (uploadLink) {
        uploadLink.style.display = enabled ? 'block' : 'none';
    }
}

// Timezone management
function renderTimezoneTags() {
    const container = document.getElementById('timezone-tags');
    container.innerHTML = '';
    
    if (currentTimezones.length === 0) {
        container.innerHTML = '<em style="color: var(--text-muted);">No additional timezones configured</em>';
        return;
    }
    
    currentTimezones.forEach((tz, index) => {
        const tag = document.createElement('div');
        tag.className = 'timezone-tag';
        tag.innerHTML = `
            <span>${tz}</span>
            <button class="remove-btn" data-index="${index}" type="button">×</button>
        `;
        container.appendChild(tag);
    });
    
    // Add event listeners to remove buttons
    container.querySelectorAll('.remove-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const index = parseInt(e.target.dataset.index);
            currentTimezones.splice(index, 1);
            renderTimezoneTags();
            updateTimePreview();
        });
    });
}

document.getElementById('add-timezone-btn').addEventListener('click', () => {
    const select = document.getElementById('timezone-selector');
    const timezone = select.value;
    
    if (!timezone) {
        showToast('Please select a timezone first.', 'warning');
        return;
    }

    if (currentTimezones.includes(timezone)) {
        showToast('This timezone is already added.', 'warning');
        return;
    }
    
    currentTimezones.push(timezone);
    renderTimezoneTags();
    updateTimePreview();
    select.value = '';
});

// Update time preview
function updateTimePreview() {
    const conductorTime = document.getElementById('conductor-time').value;
    const backupTime = document.getElementById('backup-time').value;
    
    if (!conductorTime || currentTimezones.length === 0) {
        document.getElementById('time-preview').textContent = 'Configure times above to see preview';
        return;
    }
    
    // Simple client-side preview (actual formatting happens server-side with DST)
    const preview = `${conductorTime} ST / ...`;
    document.getElementById('time-preview').textContent = `Conductor: ${conductorTime} ST (will show in ${currentTimezones.length} timezone${currentTimezones.length !== 1 ? 's' : ''})`;
}

document.getElementById('conductor-time').addEventListener('change', updateTimePreview);
document.getElementById('backup-time').addEventListener('change', updateTimePreview);

document.getElementById('power-tracking-enabled').addEventListener('change', (e) => {
    togglePowerUploadSection(e.target.checked);
});

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    const auth = await requireAuth();
    if (!auth) return;

    isR5OrAdmin = auth.is_r5_or_admin || false;
    if (!isR5OrAdmin) {
        const form = document.getElementById('settings-form');
        const inputs = form.querySelectorAll('input, textarea, button[type="submit"]');
        inputs.forEach(input => input.disabled = true);
        const notice = document.createElement('div');
        notice.className = 'permission-notice';
        notice.style.cssText = 'background: var(--bs-warning-bg); border-left: 4px solid var(--bs-warning); padding: 15px; margin-bottom: 20px; color: var(--bs-warning-text);';
        notice.innerHTML = '<p>ℹ️ Only R5 members and admins can modify settings.</p>';
        form.parentNode.insertBefore(notice, form);
    }

    await loadSettings();
    await loadBackupRotation();
});

// ---- Backup rotation ----

let rotationOrder = []; // ordered array of member IDs

async function loadBackupRotation() {
    try {
        const res = await fetch('/api/settings/backup-rotation');
        if (!res.ok) return;
        const data = await res.json();
        rotationOrder = data.order || [];
        renderRotationList(data.members || []);
    } catch (e) {
        console.error('Failed to load backup rotation', e);
    }
}

function renderRotationList(members) {
    const list = document.getElementById('rotation-list');
    if (!list) return;

    if (members.length === 0) {
        list.innerHTML = '<p class="info-text">No R4/R5 members found.</p>';
        return;
    }

    list.innerHTML = members.map((m, i) => `
        <div class="rotation-item" data-id="${m.id}" style="display:flex; align-items:center; gap:10px; padding:8px 12px; border-bottom:1px solid var(--border-color);">
            <span style="color:var(--text-muted); font-size:13px; min-width:24px;">${i + 1}.</span>
            <span style="flex:1; font-weight:600;">${escapeHtml(m.name)}${m.nickname ? ' <span style="font-weight:400; color:var(--text-muted); font-size:13px;">aka ' + escapeHtml(m.nickname) + '</span>' : ''}</span>
            <span style="color:var(--text-muted); font-size:12px; margin-right:8px;">${m.rank}</span>
            <button class="rotation-up-btn" data-index="${i}" title="Move up" style="background:none; border:1px solid var(--border-color); border-radius:4px; padding:2px 8px; cursor:pointer; color:var(--text-primary);"
                ${i === 0 ? 'disabled' : ''}>▲</button>
            <button class="rotation-down-btn" data-index="${i}" title="Move down" style="background:none; border:1px solid var(--border-color); border-radius:4px; padding:2px 8px; cursor:pointer; color:var(--text-primary);"
                ${i === members.length - 1 ? 'disabled' : ''}>▼</button>
        </div>`
    ).join('');

    list.querySelectorAll('.rotation-up-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const idx = parseInt(btn.dataset.index);
            if (idx > 0) {
                const items = getCurrentRotationItems();
                [items[idx - 1], items[idx]] = [items[idx], items[idx - 1]];
                renderRotationList(items);
            }
        });
    });

    list.querySelectorAll('.rotation-down-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const idx = parseInt(btn.dataset.index);
            const items = getCurrentRotationItems();
            if (idx < items.length - 1) {
                [items[idx], items[idx + 1]] = [items[idx + 1], items[idx]];
                renderRotationList(items);
            }
        });
    });
}

function getCurrentRotationItems() {
    const list = document.getElementById('rotation-list');
    return Array.from(list.querySelectorAll('.rotation-item')).map(el => ({
        id: parseInt(el.dataset.id),
        name: el.querySelector('span:nth-child(2)').textContent,
        rank: el.querySelector('span:nth-child(3)').textContent.trim()
    }));
}

function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

document.addEventListener('DOMContentLoaded', () => {
    const saveRotationBtn = document.getElementById('save-rotation-btn');
    const statusEl = document.getElementById('rotation-save-status');
    if (!saveRotationBtn) return;

    saveRotationBtn.addEventListener('click', async () => {
        const items = getCurrentRotationItems();
        const order = items.map(m => m.id);

        saveRotationBtn.disabled = true;
        saveRotationBtn.textContent = 'Saving...';
        statusEl.textContent = '';

        try {
            const res = await fetch('/api/settings/backup-rotation', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ order })
            });
            if (!res.ok) throw new Error(await res.text());
            statusEl.textContent = '✓ Saved';
            statusEl.style.color = 'var(--success-color, #81c784)';
            rotationOrder = order;
        } catch (e) {
            statusEl.textContent = '✗ ' + e.message;
            statusEl.style.color = 'var(--danger-color, #e57373)';
        } finally {
            saveRotationBtn.disabled = false;
            saveRotationBtn.textContent = '💾 Save Rotation Order';
        }
    });
});
