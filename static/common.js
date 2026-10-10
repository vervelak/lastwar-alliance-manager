/**
 * Common utilities for Last War Alliance Manager
 * Shared auth, fetch, and utility functions used across all pages.
 * Include via <script src="common.js"></script> before page-specific JS.
 */

const API_BASE = '/api';

/**
 * Escape HTML to prevent XSS when inserting user-supplied text into the DOM.
 */
function escapeHtml(text) {
    if (text == null) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

/**
 * Centralized fetch wrapper with auth error handling.
 * Automatically redirects to login on 401, and to profile on 403 with must_change_password.
 * Returns the Response object on success.
 */
async function apiFetch(path, opts = {}) {
    const url = path.startsWith('http') ? path : `${API_BASE}${path}`;
    const response = await fetch(url, opts);

    if (response.status === 401) {
        window.location.href = '/login.html';
        throw new Error('Unauthorized');
    }
    if (response.status === 403) {
        try {
            const body = await response.clone().json();
            if (body.must_change_password) {
                window.location.href = '/profile.html?must_change_password=1';
                throw new Error('Password change required');
            }
        } catch (e) {
            if (e.message === 'Password change required') throw e;
        }
    }
    return response;
}

/**
 * Check authentication and set up common UI elements.
 * Returns the auth data object on success, or false if redirected.
 *
 * Sets up: username display, dropdown toggle, logout button, admin link visibility.
 * Handles must_change_password redirect (except on profile page).
 */
async function requireAuth() {
    try {
        const response = await fetch(`${API_BASE}/check-auth`);
        if (!response.ok) {
            window.location.href = '/login.html';
            return false;
        }

        const data = await response.json();
        if (!data.authenticated) {
            window.location.href = '/login.html';
            return false;
        }

        // Force password change redirect (skip if already on profile page)
        if (data.must_change_password && !window.location.pathname.includes('profile.html')) {
            window.location.href = '/profile.html?must_change_password=1';
            return false;
        }

        // Set username display
        let displayText = `👤 ${data.username}`;
        if (data.rank) {
            displayText += ` (${data.rank})`;
        }
        const usernameDisplay = document.getElementById('username-display');
        if (usernameDisplay) {
            usernameDisplay.textContent = displayText;
            usernameDisplay.addEventListener('click', toggleUserDropdown);
        }

        // Setup logout
        const logoutBtn = document.getElementById('dropdown-logout-btn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', handleLogout);
        }

        // Show admin link if admin
        if (data.is_admin) {
            const adminLink = document.getElementById('admin-dropdown-link');
            if (adminLink) adminLink.style.display = 'block';
        }

        // Close dropdown on outside click
        document.addEventListener('click', (event) => {
            const dropdown = document.getElementById('user-dropdown-menu');
            const usernameBtn = document.getElementById('username-display');
            if (dropdown && usernameBtn && !usernameBtn.contains(event.target) && !dropdown.contains(event.target)) {
                dropdown.classList.remove('show');
            }
        });

        return data;
    } catch (error) {
        console.error('Auth check failed:', error);
        window.location.href = '/login.html';
        return false;
    }
}

/**
 * Toggle the user dropdown menu.
 */
function toggleUserDropdown(event) {
    if (event) event.stopPropagation();
    const dropdown = document.getElementById('user-dropdown-menu');
    if (dropdown) {
        dropdown.classList.toggle('show');
    }
}

/**
 * Logout and redirect to login page.
 */
async function handleLogout(event) {
    if (event) event.preventDefault();
    try {
        await fetch(`${API_BASE}/logout`, { method: 'POST' });
    } catch (e) {
        // Ignore logout errors
    }
    window.location.href = '/login.html';
}

/**
 * Button loading state helpers.
 */
function setButtonLoading(btn, text) {
    if (!btn) return;
    btn.dataset.originalText = btn.textContent;
    btn.textContent = text || 'Loading...';
    btn.disabled = true;
}

function clearButtonLoading(btn) {
    if (!btn) return;
    btn.textContent = btn.dataset.originalText || btn.textContent;
    btn.disabled = false;
}

/**
 * Escape key closes the topmost open modal.
 *
 * Clicks the modal's own close button when present so each page's close
 * logic (form resets, etc.) runs; falls back to hiding the modal otherwise.
 * Skips while the confirm dialog or the MG lightbox owns Escape.
 */
document.addEventListener('keydown', (e) => {
    if (e.key !== 'Escape') return;
    if (document.querySelector('.confirm-overlay')) return;
    const lightbox = document.getElementById('mg-lightbox');
    if (lightbox && lightbox.style.display !== 'none') return;

    const modals = Array.from(document.querySelectorAll('.modal')).filter(
        (m) => m.style.display !== 'none' && getComputedStyle(m).display !== 'none'
    );
    const top = modals[modals.length - 1];
    if (!top) return;

    const closeBtn = top.querySelector('.close');
    if (closeBtn) closeBtn.click();
    else top.style.display = 'none';
});
