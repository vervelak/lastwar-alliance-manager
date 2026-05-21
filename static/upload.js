// Upload page

let selectedFiles = []; // Array to hold multiple files
const MAX_FILES = 25;

// Tab switching
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        const tabName = btn.dataset.tab;
        
        // Update buttons
        document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        
        // Update content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`tab-${tabName}`).classList.add('active');
        
        // Clear results
        document.getElementById('result-container').innerHTML = '';
    });
});

// Image upload handling
const imageInput = document.getElementById('image-input');
const dropZone = document.getElementById('drop-zone');
const dropContent = document.getElementById('drop-content');
const previewContainer = document.getElementById('preview-container');
const previewGallery = document.getElementById('preview-gallery');
const filesCount = document.getElementById('files-count');
const processImageBtn = document.getElementById('process-image-btn');
const clearBtn = document.getElementById('clear-btn');

// Click to upload
dropZone.addEventListener('click', (e) => {
    if (e.target === clearBtn || clearBtn.contains(e.target)) {
        return; // Don't trigger file input if clicking clear button
    }
    if (selectedFiles.length === 0) {
        imageInput.click();
    }
});

// File selection
imageInput.addEventListener('change', (e) => {
    const files = Array.from(e.target.files);
    if (files.length > 0) {
        handleFiles(files);
    }
});

// Drag and drop
dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
});

dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
});

dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    
    const files = Array.from(e.dataTransfer.files);
    if (files.length > 0) {
        handleFiles(files);
    }
});

function handleFiles(files) {
    // Filter image files only
    let imageFiles = files.filter(file => file.type.startsWith('image/'));
    
    if (imageFiles.length === 0) {
        showResult('Please upload image files (PNG, JPG, JPEG)', 'error');
        return;
    }
    
    // Check file size
    const oversizedFiles = imageFiles.filter(file => file.size > 10 * 1024 * 1024);
    if (oversizedFiles.length > 0) {
        showResult(`${oversizedFiles.length} file(s) exceed 10MB limit and will be skipped`, 'error');
        imageFiles = imageFiles.filter(file => file.size <= 10 * 1024 * 1024);
    }
    
    // Check total count
    if (selectedFiles.length + imageFiles.length > MAX_FILES) {
        showResult(`Maximum ${MAX_FILES} files allowed. Only adding first ${MAX_FILES - selectedFiles.length} files.`, 'error');
        imageFiles = imageFiles.slice(0, MAX_FILES - selectedFiles.length);
    }
    
    // Add to selected files
    imageFiles.forEach(file => {
        selectedFiles.push(file);
    });
    
    updatePreview();
    
    // Clear any previous results
    document.getElementById('result-container').innerHTML = '';
}

function updatePreview() {
    if (selectedFiles.length === 0) {
        previewContainer.style.display = 'none';
        dropContent.style.display = 'block';
        processImageBtn.style.display = 'none';
        return;
    }
    
    filesCount.textContent = `${selectedFiles.length} file${selectedFiles.length > 1 ? 's' : ''} selected`;
    previewGallery.innerHTML = '';
    
    selectedFiles.forEach((file, index) => {
        const reader = new FileReader();
        reader.onload = (e) => {
            const previewItem = document.createElement('div');
            previewItem.className = 'preview-item';
            previewItem.dataset.fileIndex = index;
            previewItem.innerHTML = `
                <button class="remove-file" data-index="${index}" title="Remove">×</button>
                <img src="${e.target.result}" class="preview-img" alt="${file.name}">
                <div class="file-name" title="${file.name}">${file.name}</div>
                <div class="preview-status"></div>
            `;
            
            // Add remove handler
            previewItem.querySelector('.remove-file').addEventListener('click', (evt) => {
                evt.stopPropagation();
                removeFile(index);
            });
            
            previewGallery.appendChild(previewItem);
        };
        reader.readAsDataURL(file);
    });
    
    dropContent.style.display = 'none';
    previewContainer.style.display = 'block';
    processImageBtn.style.display = 'block';
}

// Update a tile's status overlay during OCR processing
function setTileStatus(index, status) {
    const tile = previewGallery.querySelector(`[data-file-index="${index}"]`);
    if (!tile) return;
    const statusEl = tile.querySelector('.preview-status');
    if (!statusEl) return;
    tile.classList.remove('tile-pending', 'tile-processing', 'tile-done', 'tile-error');
    switch (status) {
        case 'pending':
            tile.classList.add('tile-pending');
            statusEl.textContent = '⏳';
            break;
        case 'processing':
            tile.classList.add('tile-processing');
            statusEl.innerHTML = '<span class="tile-spinner"></span>';
            break;
        case 'done':
            tile.classList.add('tile-done');
            statusEl.textContent = '✅';
            break;
        case 'error':
            tile.classList.add('tile-error');
            statusEl.textContent = '❌';
            break;
    }
}

function removeFile(index) {
    selectedFiles.splice(index, 1);
    updatePreview();
}

// Clear all images
clearBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    selectedFiles = [];
    imageInput.value = '';
    updatePreview();
    document.getElementById('result-container').innerHTML = '';
});

// Process image with OCR
processImageBtn.addEventListener('click', async () => {
    if (selectedFiles.length === 0) {
        showResult('Please select at least one image', 'error');
        return;
    }
    
    // Get selected screenshot type
    const screenshotType = document.getElementById('screenshot-type').value;

    // Member list uses a separate flow with a preview/confirm step
    if (screenshotType === 'member-list') {
        await processMemberListScreenshots();
        return;
    }

    // Donations use a separate OCR-then-confirm flow
    if (screenshotType === 'donations') {
        await processDonationScreenshots();
        return;
    }
    
    const originalText = processImageBtn.innerHTML;
    processImageBtn.innerHTML = '<span class="loading"></span> Processing...';
    processImageBtn.disabled = true;
    processImageBtn.setAttribute('aria-busy', 'true');
    
    try {
        const typeLabel = screenshotType === 'power' ? 'Power Rankings' : 'VS Points';
        showResult(`🔍 Processing ${selectedFiles.length} ${typeLabel} screenshot${selectedFiles.length > 1 ? 's' : ''} with OCR...`, 'info');
        
        // Mark all tiles as pending
        for (let j = 0; j < selectedFiles.length; j++) setTileStatus(j, 'pending');
        
        let totalSuccess = 0;
        let totalFailed = 0;
        const allErrors = [];
        const detectedDays = []; // For VS points
        const allNotFound = []; // Accumulate unmatched names for auto-register
        
        // Determine API endpoint based on screenshot type
        const apiEndpoint = screenshotType === 'power' 
            ? `${API_BASE}/power-history/process-screenshot`
            : `${API_BASE}/vs-points/process-screenshot`;
        
        // Process files sequentially to avoid overwhelming the server
        for (let i = 0; i < selectedFiles.length; i++) {
            const file = selectedFiles[i];
            setTileStatus(i, 'processing');
            
            showResult(`🔍 Processing ${typeLabel} screenshot ${i + 1} of ${selectedFiles.length}: ${file.name}...`, 'info');
            
            try {
                const formData = new FormData();
                formData.append('image', file);
                
                // Add week parameter for VS Points
                if (screenshotType === 'vs-points') {
                    const week = document.getElementById('vs-week').value;
                    formData.append('week', week);
                }
                
                const response = await fetch(apiEndpoint, {
                    method: 'POST',
                    body: formData
                });
                
                if (!response.ok) {
                    const error = await response.text();
                    throw new Error(error);
                }
                
                const result = await response.json();
                totalSuccess += result.success_count || 0;
                setTileStatus(i, 'done');
                
                // Track detected day for VS points
                if (screenshotType === 'vs-points' && result.day) {
                    detectedDays.push(`${file.name} → ${result.day}`);
                }
                
                if (result.errors && result.errors.length > 0) {
                    allErrors.push(`<strong>${file.name}:</strong> ${result.errors.join(', ')}`);
                    totalFailed++;
                } else if (result.not_found_members && result.not_found_members.length > 0) {
                    allErrors.push(`<strong>${file.name}:</strong> ${result.not_found_members.length} members not found in database`);
                    result.not_found_members.forEach(n => { if (!allNotFound.includes(n)) allNotFound.push(n); });
                }
            } catch (error) {
                console.error(`Error processing ${file.name}:`, error);
                allErrors.push(`<strong>${file.name}:</strong> ${error.message}`);
                totalFailed++;
                setTileStatus(i, 'error');
            }
        }
        
        // Show final results
        let html = `<div class="result-box result-success">
            <strong>✅ Processed ${selectedFiles.length} ${typeLabel} screenshot${selectedFiles.length > 1 ? 's' : ''}</strong><br>
            <div style="margin-top: 10px;">
                <strong>Total Records Updated:</strong> ${totalSuccess}`;
                
        if (totalFailed > 0) {
            html += ` | <strong>Failed:</strong> ${totalFailed}`;
        }
        
        html += `</div>`;
        
        // Show detected days for VS points
        if (screenshotType === 'vs-points' && detectedDays.length > 0) {
            html += `<br><br><strong>Detected Days:</strong><br><div style="max-height: 150px; overflow-y: auto; margin-top: 5px; font-size: 13px;">${detectedDays.join('<br>')}</div>`;
        }
        
        if (allErrors.length > 0) {
            html += `<br><br><strong>Issues:</strong><br><div style="max-height: 200px; overflow-y: auto; margin-top: 5px;">${allErrors.join('<br>')}</div>`;
        }
        
        // Auto-register section for unmatched players
        if (allNotFound.length > 0) {
            html += `<br><br><div id="auto-register-section" style="padding:10px; background: var(--card-bg); border-radius:6px; border: 1px solid var(--border-color);">
                <strong>👤 ${allNotFound.length} player${allNotFound.length > 1 ? 's' : ''} not found in member database:</strong>
                <div style="font-size:12px; color: var(--text-muted); margin: 4px 0 8px;">${allNotFound.map(n => escapeHtml(n)).join(', ')}</div>
                <button id="auto-register-btn" class="btn btn-secondary" style="padding:6px 14px; font-size:13px;">
                    ➕ Register as new R1 members
                </button>
            </div>`;
        }
        
        html += '</div>';
        document.getElementById('result-container').innerHTML = html;

        // Wire up auto-register button if present
        if (allNotFound.length > 0) {
            const autoBtn = document.getElementById('auto-register-btn');
            if (autoBtn) {
                autoBtn.addEventListener('click', () => autoRegisterUnmatched(allNotFound, autoBtn));
            }
        }
        
        // Clear on success after delay
        if (totalSuccess > 0) {
            setTimeout(() => {
                selectedFiles = [];
                imageInput.value = '';
                updatePreview();
            }, 5000);
        }
    } catch (error) {
        console.error('Error processing images:', error);
        showResult(`❌ Processing failed: ${error.message}`, 'error');
    } finally {
        processImageBtn.innerHTML = originalText;
        processImageBtn.disabled = false;
        processImageBtn.removeAttribute('aria-busy');
    }
});

// Process manual text entry
document.getElementById('process-text-btn').addEventListener('click', async () => {
    const textInput = document.getElementById('text-input');
    const text = textInput.value.trim();
    
    if (!text) {
        showResult('Please enter some data', 'error');
        return;
    }
    
    const btn = document.getElementById('process-text-btn');
    const originalText = btn.innerHTML;
    btn.innerHTML = '<span class="loading"></span> Uploading...';
    btn.disabled = true;
    
    try {
        showResult('📤 Processing text data...', 'info');
        
        // Parse the text input
        const lines = text.split('\n');
        const records = [];
        const errors = [];
        
        lines.forEach((line, index) => {
            line = line.trim();
            if (!line) return;
            
            // Try to parse: Name, Power or Name Power
            const parts = line.split(/[,\s]+/);
            if (parts.length < 2) {
                errors.push(`Line ${index + 1}: Invalid format - need Name, Power`);
                return;
            }
            
            const name = parts.slice(0, -1).join(' ').trim();
            const powerStr = parts[parts.length - 1].replace(/,/g, '');
            const power = parseInt(powerStr, 10);
            
            if (!name || isNaN(power) || power < 1000000) {
                errors.push(`Line ${index + 1}: Invalid data - "${line}"`);
                return;
            }
            
            records.push({ member_name: name, power: power });
        });
        
        if (errors.length > 0) {
            showResult(`<strong>Parsing errors:</strong><br>${errors.join('<br>')}`, 'error');
            return;
        }
        
        if (records.length === 0) {
            showResult('No valid records to upload', 'error');
            return;
        }
        
        const response = await fetch(`${API_BASE}/power-history/process-screenshot`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ records: records })
        });
        
        if (!response.ok) {
            const error = await response.text();
            throw new Error(error);
        }
        
        const result = await response.json();
        
        let html = `<div class="result-box result-success">
            <strong>✅ ${result.message}</strong><br>
            <div style="margin-top: 10px;">
                <strong>Successful:</strong> ${result.success_count} | 
                <strong>Failed:</strong> ${result.failed_count}
            </div>`;
        
        if (result.errors && result.errors.length > 0) {
            html += `<br><br><strong>Errors:</strong><br><div style="max-height: 200px; overflow-y: auto; margin-top: 5px;">${result.errors.join('<br>')}</div>`;
        }
        
        html += '</div>';
        document.getElementById('result-container').innerHTML = html;
        
        // Clear on success after delay
        if (result.success_count > 0) {
            setTimeout(() => {
                textInput.value = '';
            }, 2000);
        }
    } catch (error) {
        console.error('Error uploading data:', error);
        showResult(`❌ Upload failed: ${error.message}`, 'error');
    } finally {
        btn.innerHTML = originalText;
        btn.disabled = false;
    }
});

function showResult(message, type) {
    if (type === 'raw') {
        document.getElementById('result-container').innerHTML = message;
        return;
    }
    const resultClass = type === 'error' ? 'result-error' : 
                       type === 'info' ? 'result-info' : 'result-success';
    document.getElementById('result-container').innerHTML = 
        `<div class="result-box ${resultClass}">${message}</div>`;
}

// Screenshot type selector handler
function updateScreenshotTypeHint() {
    const screenshotType = document.getElementById('screenshot-type').value;
    const hintElement = document.getElementById('screenshot-type-hint');
    const weekSelector = document.getElementById('week-selector');
    const donSelector = document.getElementById('donation-selector');

    if (weekSelector) weekSelector.style.display = 'none';
    if (donSelector) donSelector.style.display = 'none';

    if (screenshotType === 'power') {
        hintElement.textContent = 'Upload power ranking screenshots from the alliance member list.';
    } else if (screenshotType === 'vs-points') {
        hintElement.innerHTML = '<strong>⚔️ VS Points Instructions:</strong> Make sure to screenshot the "Daily Rank" tab. The system will automatically detect which day (Mon-Sat) is selected from the screenshot.';
        if (weekSelector) weekSelector.style.display = 'block';
    } else if (screenshotType === 'member-list') {
        hintElement.innerHTML = '<strong>👥 Alliance Member List:</strong> Screenshot the full alliance roster (names + ranks). OCR will extract members and prompt you to confirm changes before saving.';
    } else if (screenshotType === 'donations') {
        hintElement.innerHTML = '<strong>💎 Donation Rankings:</strong> Upload screenshots of the Strength Ranking → Donation tab. Upload all scrolled screenshots together — they are merged automatically.';
        if (donSelector) donSelector.style.display = 'block';
    }
}

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    const auth = await requireAuth();
    if (!auth) return;
    
    // Setup screenshot type selector
    const screenshotTypeSelector = document.getElementById('screenshot-type');
    if (screenshotTypeSelector) {
        screenshotTypeSelector.addEventListener('change', updateScreenshotTypeHint);
        updateScreenshotTypeHint(); // Set initial hint
    }

    // Pre-fill donation date with today, wire type-toggle buttons
    const donDateInput = document.getElementById('don-date');
    if (donDateInput) {
        donDateInput.value = new Date().toISOString().slice(0, 10);
    }
    document.querySelectorAll('[data-dontype]').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('[data-dontype]').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            // When switching to weekly, snap date to nearest Monday
            if (btn.dataset.dontype === 'weekly' && donDateInput) {
                const d = new Date(donDateInput.value || new Date());
                const day = d.getDay(); // 0=Sun,1=Mon,...
                const diff = (day === 0) ? -6 : 1 - day;
                d.setDate(d.getDate() + diff);
                donDateInput.value = d.toISOString().slice(0, 10);
            }
        });
    });
    
    // Check if power tracking is enabled
    try {
        const response = await fetch(`${API_BASE}/settings`);
        if (response.ok) {
            const settings = await response.json();
            if (!settings.power_tracking_enabled) {
                showResult('⚠️ Power tracking is not enabled. Some features may be limited. Please enable it in Settings.', 'info');
            }
        }
    } catch (error) {
        console.error('Failed to check power tracking status:', error);
    }

    // Member OCR confirm/cancel buttons
    document.getElementById('confirm-member-ocr-btn').addEventListener('click', confirmMemberOCR);
    document.getElementById('cancel-member-ocr-btn').addEventListener('click', () => {
        document.getElementById('member-ocr-preview').style.display = 'none';
        document.getElementById('result-container').innerHTML = '';
    });

    // Donation OCR confirm/cancel buttons
    document.getElementById('confirm-donation-ocr-btn').addEventListener('click', confirmDonationOCR);
    document.getElementById('cancel-donation-ocr-btn').addEventListener('click', () => {
        document.getElementById('donation-ocr-preview').style.display = 'none';
        document.getElementById('result-container').innerHTML = '';
    });
});

// ---- Donations OCR flow ----

let donationOCRData = null; // holds DonationOCRResult from backend

async function processDonationScreenshots() {
    if (selectedFiles.length === 0) return;

    const donTypeBtn = document.querySelector('[data-dontype].active');
    const donType = donTypeBtn ? donTypeBtn.dataset.dontype : 'weekly';
    const donDate = document.getElementById('don-date').value || new Date().toISOString().slice(0, 10);

    const originalText = processImageBtn.innerHTML;
    processImageBtn.innerHTML = '<span class="loading"></span> Processing...';
    processImageBtn.disabled = true;
    document.getElementById('donation-ocr-preview').style.display = 'none';
    document.getElementById('result-container').innerHTML = '';

    try {
        showResult(`🔍 Processing ${selectedFiles.length} donation screenshot${selectedFiles.length > 1 ? 's' : ''} with OCR...`, 'info');
        for (let j = 0; j < selectedFiles.length; j++) setTileStatus(j, 'pending');

        const formData = new FormData();
        formData.append('donation_type', donType);
        formData.append('record_date', donDate);
        for (let i = 0; i < selectedFiles.length; i++) {
            setTileStatus(i, 'processing');
            formData.append('images', selectedFiles[i]);
        }

        const response = await fetch(`${API_BASE}/donations/process-screenshots`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            const errText = await response.text();
            throw new Error(errText || 'OCR processing failed');
        }

        donationOCRData = await response.json();

        for (let j = 0; j < selectedFiles.length; j++) setTileStatus(j, 'done');

        if (!donationOCRData.entries || donationOCRData.entries.length === 0) {
            showResult('⚠️ No donation entries detected. Try a cleaner screenshot with better contrast.', 'error');
            return;
        }

        renderDonationOCRPreview(donationOCRData);
        document.getElementById('donation-ocr-preview').style.display = 'block';
        document.getElementById('result-container').innerHTML = '';

    } catch (error) {
        console.error('Donation OCR error:', error);
        showResult(`❌ OCR failed: ${error.message}`, 'error');
        for (let j = 0; j < selectedFiles.length; j++) setTileStatus(j, 'error');
    } finally {
        processImageBtn.innerHTML = originalText;
        processImageBtn.disabled = false;
    }
}

function renderDonationOCRPreview(data) {
    const meta = document.getElementById('donation-ocr-meta');
    const typeLabel = data.donation_type === 'daily' ? '📅 Daily' : '📆 Weekly';
    meta.innerHTML = `${typeLabel} · <strong>${data.record_date}</strong> · ${data.total_rows} entries from ${data.total_images} screenshot${data.total_images !== 1 ? 's' : ''}`;

    const tbody = document.getElementById('donation-ocr-tbody');
    tbody.innerHTML = data.entries.map((e, i) => {
        const matchColor = e.match_type === 'exact' ? 'var(--success-color, #81c784)'
            : e.match_type === 'nickname' ? 'var(--success-color, #81c784)'
            : e.match_type === 'fuzzy' ? 'var(--warning-color, #ffb74d)'
            : 'var(--text-muted)';
        const matchLabel = e.match_type === 'exact' ? '✓ exact'
            : e.match_type === 'nickname' ? '✓ alias'
            : e.match_type === 'fuzzy' ? `~ fuzzy (${e.match_confidence}%)`
            : '✗ none';
        const memberDisplay = e.member_name
            ? `<span style="color:${matchColor}">${escapeHtml(e.member_name)}</span> <small style="color:var(--text-muted)">${matchLabel}</small>`
            : `<span style="color:var(--text-muted)">— ${matchLabel}</span>`;

        return `<tr data-idx="${i}">
            <td><input type="checkbox" data-idx="${i}" checked></td>
            <td style="text-align:center; color:var(--text-muted)">${e.rank_in_snapshot}</td>
            <td>${escapeHtml(e.name_snapshot)}</td>
            <td>${memberDisplay}</td>
            <td><input type="number" class="form-input don-pts-input" data-idx="${i}" value="${e.points}" min="0" style="width:90px; padding:4px 6px; font-size:13px;"></td>
        </tr>`;
    }).join('');
}

async function confirmDonationOCR() {
    if (!donationOCRData) return;

    const checkboxes = document.querySelectorAll('#donation-ocr-tbody input[type=checkbox]');
    const ptsInputs = document.querySelectorAll('.don-pts-input');
    const entries = [];

    checkboxes.forEach(cb => {
        if (!cb.checked) return;
        const idx = parseInt(cb.dataset.idx);
        const e = donationOCRData.entries[idx];
        const pts = parseInt(ptsInputs[idx]?.value ?? e.points) || 0;
        entries.push({
            rank_in_snapshot: e.rank_in_snapshot,
            name_snapshot: e.name_snapshot,
            points: pts,
            member_id: e.member_id || null,
            member_name: e.member_name || '',
            member_rank: e.member_rank || '',
            match_confidence: e.match_confidence,
            match_type: e.match_type
        });
    });

    if (entries.length === 0) {
        showResult('Please select at least one entry to save.', 'error');
        return;
    }

    const confirmBtn = document.getElementById('confirm-donation-ocr-btn');
    const originalText = confirmBtn.innerHTML;
    confirmBtn.innerHTML = '<span class="loading"></span> Saving...';
    confirmBtn.disabled = true;

    try {
        const response = await fetch(`${API_BASE}/donations`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                donation_type: donationOCRData.donation_type,
                record_date: donationOCRData.record_date,
                entries
            })
        });

        if (!response.ok) {
            const errText = await response.text();
            throw new Error(errText || 'Save failed');
        }

        const result = await response.json();
        document.getElementById('donation-ocr-preview').style.display = 'none';
        showResult(`✅ Saved ${result.saved} donation records for ${donationOCRData.donation_type} ${donationOCRData.record_date}. <a href="/donations.html?type=${donationOCRData.donation_type}&date=${donationOCRData.record_date}" style="color:var(--accent-primary)">View Leaderboard →</a>`, 'success');
        donationOCRData = null;
        selectedFiles = [];
        imageInput.value = '';
        updatePreview();
    } catch (error) {
        console.error('Donation save error:', error);
        showResult(`❌ Save failed: ${error.message}`, 'error');
    } finally {
        confirmBtn.innerHTML = originalText;
        confirmBtn.disabled = false;
    }
}

// ---- Member list OCR flow ----

let memberOCRDetected = [];
let memberOCRToRemove = [];

async function processMemberListScreenshots() {
    if (selectedFiles.length === 0) return;

    const originalText = processImageBtn.innerHTML;
    processImageBtn.innerHTML = '<span class="loading"></span> Processing...';
    processImageBtn.disabled = true;
    document.getElementById('member-ocr-preview').style.display = 'none';

    try {
        // Only process first file for member list (roster is one page)
        const file = selectedFiles[0];
        showResult(`🔍 Processing alliance roster screenshot: ${file.name}...`, 'info');

        const formData = new FormData();
        formData.append('image', file);

        const response = await fetch(`${API_BASE}/members/import-screenshot`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            const errText = await response.text();
            throw new Error(errText || 'OCR processing failed');
        }

        const result = await response.json();

        if (!result.detected_members || result.detected_members.length === 0) {
            showResult('⚠️ No members detected. Try a cleaner screenshot with better contrast.', 'error');
            return;
        }

        memberOCRDetected = result.detected_members;
        memberOCRToRemove = result.members_to_remove || [];
        renderMemberOCRPreview(result);
        document.getElementById('member-ocr-preview').style.display = 'block';
        document.getElementById('result-container').innerHTML = '';

    } catch (error) {
        console.error('Member OCR error:', error);
        showResult(`❌ OCR failed: ${error.message}`, 'error');
    } finally {
        processImageBtn.innerHTML = originalText;
        processImageBtn.disabled = false;
    }
}

function renderMemberOCRPreview(result) {
    const listEl = document.getElementById('member-ocr-list');
    const removeSection = document.getElementById('member-ocr-remove');
    const removeListEl = document.getElementById('member-ocr-remove-list');
    const errorsEl = document.getElementById('member-ocr-errors');

    // Detected members list
    listEl.innerHTML = result.detected_members.map((m, i) => {
        let badge = '';
        if (m.is_new) badge = `<span style="color: var(--success-color, #81c784); font-size:11px; margin-left:6px;">NEW</span>`;
        else if (m.rank_changed) badge = `<span style="color: var(--accent-primary); font-size:11px; margin-left:6px;">${m.old_rank} → ${m.rank}</span>`;

        const similar = m.similar_match && m.similar_match.length > 0
            ? `<div style="font-size:11px; color: var(--warning-color, #ffb74d); margin-top:2px;">Similar existing: ${m.similar_match.join(', ')}</div>`
            : '';

        return `<label style="display:flex; align-items:flex-start; gap:8px; padding:6px 8px; border-bottom:1px solid var(--border-color); cursor:pointer;">
            <input type="checkbox" data-index="${i}" checked style="margin-top:3px;">
            <div>
                <span style="font-weight:600;">${escapeHtml(m.name)}</span>
                <span style="color: var(--text-muted); margin-left:6px; font-size:13px;">${m.rank}</span>
                ${badge}
                ${similar}
            </div>
        </label>`;
    }).join('');

    // Members to remove
    if (memberOCRToRemove.length > 0) {
        removeListEl.innerHTML = memberOCRToRemove.map(m =>
            `<label style="display:flex; align-items:center; gap:8px; padding:4px 8px; border-bottom:1px solid var(--border-color); cursor:pointer;">
                <input type="checkbox" data-remove-id="${m.id}">
                <span>${escapeHtml(m.name)}</span>
                <span style="color: var(--text-muted); font-size:12px;">${m.rank}</span>
            </label>`
        ).join('');
        removeSection.style.display = 'block';
    } else {
        removeSection.style.display = 'none';
    }

    // Parse errors
    if (result.errors && result.errors.length > 0) {
        errorsEl.innerHTML = `<div style="font-size:12px; color: var(--text-muted); padding:8px; background: var(--card-bg); border-radius:6px;">
            <strong>Parse notes:</strong><br>${result.errors.map(e => escapeHtml(e)).join('<br>')}
        </div>`;
        errorsEl.style.display = 'block';
    } else {
        errorsEl.style.display = 'none';
    }
}

async function confirmMemberOCR() {
    const checkboxes = document.querySelectorAll('#member-ocr-list input[type=checkbox]');
    const selectedMembers = [];
    checkboxes.forEach(cb => {
        if (cb.checked) {
            const idx = parseInt(cb.dataset.index);
            selectedMembers.push(memberOCRDetected[idx]);
        }
    });

    if (selectedMembers.length === 0) {
        showResult('Please select at least one member to import', 'error');
        return;
    }

    const removeCheckboxes = document.querySelectorAll('#member-ocr-remove-list input[type=checkbox]');
    const removeMemberIDs = [];
    removeCheckboxes.forEach(cb => {
        if (cb.checked) removeMemberIDs.push(parseInt(cb.dataset.removeId));
    });

    const confirmBtn = document.getElementById('confirm-member-ocr-btn');
    const originalText = confirmBtn.textContent;
    confirmBtn.disabled = true;
    confirmBtn.textContent = 'Importing...';

    try {
        const response = await fetch(`${API_BASE}/members/import/confirm`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ members: selectedMembers, remove_member_ids: removeMemberIDs, renames: [] })
        });

        if (!response.ok) throw new Error(await response.text() || 'Import failed');

        const result = await response.json();
        document.getElementById('member-ocr-preview').style.display = 'none';

        let msg = `✅ Imported ${result.added + result.updated} member(s)`;
        if (result.unchanged > 0) msg += ` | ${result.unchanged} unchanged`;
        if (result.removed > 0) msg += ` | 🗑️ ${result.removed} removed`;

        showResult(`<div class="result-box result-success"><strong>${msg}</strong></div>`, 'raw');

        // Clear uploaded files
        selectedFiles = [];
        imageInput.value = '';
        updatePreview();

    } catch (error) {
        console.error('Confirm error:', error);
        showResult(`❌ Import failed: ${error.message}`, 'error');
    } finally {
        confirmBtn.disabled = false;
        confirmBtn.textContent = originalText;
    }
}

function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

async function autoRegisterUnmatched(names, btn) {
    const originalText = btn.textContent;
    btn.disabled = true;
    btn.textContent = 'Registering...';

    try {
        const response = await fetch(`${API_BASE}/members/auto-register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ names })
        });

        if (!response.ok) throw new Error(await response.text() || 'Registration failed');

        const result = await response.json();
        const section = document.getElementById('auto-register-section');
        if (section) {
            section.innerHTML = `<span style="color: var(--success-color, #81c784);">✅ ${result.message}</span>` +
                (result.skipped && result.skipped.length > 0
                    ? `<span style="color: var(--text-muted); font-size:12px; margin-left:8px;">Skipped (already exist): ${result.skipped.map(s => escapeHtml(s)).join(', ')}</span>`
                    : '');
        }
    } catch (error) {
        console.error('Auto-register error:', error);
        btn.textContent = '❌ ' + error.message;
        btn.disabled = false;
    }
}


