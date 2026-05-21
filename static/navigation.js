/**
 * Navigation: Header + Sidebar injection and toggle
 * Injects shared header/nav HTML into pages that declare data-subtitle on <body>.
 * Handles collapsible sidebar functionality with state persistence.
 */

(function() {
    'use strict';

    // Inject shared <header> and <nav> into .container
    function injectHeader() {
        const subtitle = document.body.dataset.subtitle;
        if (!subtitle) return; // login.html, modal-test.html etc. — skip

        const h1 = document.body.dataset.h1 || '🎮 Last War: Survival';

        const html = `
        <header>
            <h1>${h1}</h1>
            <h2>${subtitle}</h2>
            <div class="user-info">
                <div class="user-dropdown">
                    <button id="username-display" class="username-btn"></button>
                    <div id="user-dropdown-menu" class="dropdown-menu">
                        <a href="/profile.html" class="dropdown-item" id="profile-dropdown-link">👤 Profile</a>
                        <a href="/admin.html" class="dropdown-item admin-only" id="admin-dropdown-link" style="display: none;">🔐 Admin Panel</a>
                        <div class="dropdown-divider"></div>
                        <div class="theme-section">
                            <div class="theme-section-label">Theme</div>
                            <a href="#" class="dropdown-item theme-option" data-theme="auto">● Auto (System)</a>
                            <a href="#" class="dropdown-item theme-option" data-theme="light">○ Light</a>
                            <a href="#" class="dropdown-item theme-option" data-theme="dark">○ Dark</a>
                        </div>
                        <div class="dropdown-divider"></div>
                        <a href="#" class="dropdown-item" id="dropdown-logout-btn">🚪 Logout</a>
                    </div>
                </div>
            </div>
        </header>
        <nav class="nav-menu">
            <div class="nav-header">
                <span class="nav-logo">⚔️ Last War</span>
                <button class="nav-collapse-btn" aria-label="Collapse sidebar" title="Collapse">‹</button>
            </div>
            <div class="nav-links">
                <a href="/" class="nav-link">👥 Members</a>
                <a href="/train.html" class="nav-link">🚂 Train</a>
                <a href="/awards.html" class="nav-link">🏆 Awards</a>
                <a href="/recommendations.html" class="nav-link">⭐ Recommendations</a>
                <a href="/conduct.html" class="nav-link">📋 Conduct Reports</a>
                <a href="/rankings.html" class="nav-link">📊 Rankings</a>
                <a href="/storm.html" class="nav-link">🏜️ Storm</a>
                <a href="/marshal-guard.html" class="nav-link" id="mg-nav-link">🛡️ Marshal Guard</a>
                <a href="/vs.html" class="nav-link">⚔️ VS Points</a>
                <a href="/vs-compliance.html" class="nav-link">✅ VS Compliance</a>
                <a href="/upload.html" class="nav-link">📸 Upload</a>
                <a href="/settings.html" class="nav-link">⚙️ Settings</a>
                <a href="/recruit.html" class="nav-link">🎯 Recruit</a>
                <a href="/admin.html" class="nav-link admin-link" id="admin-nav-link" style="display: none;">🔐 Admin Panel</a>
                <a href="/graveyard.html" class="nav-link admin-link" id="graveyard-nav-link" style="display: none;">🪦 Graveyard</a>
            </div>
        </nav>`;

        const footer = `
        <footer class="app-footer">
            <span>&copy; ${new Date().getFullYear()} <a href="https://github.com/vervelak/lastwar-alliance-manager" target="_blank" rel="noopener noreferrer">Synnefo Ltd</a> &mdash; Last War Alliance Manager</span>
        </footer>`;

        const container = document.querySelector('.container');
        if (!container) return;
        container.insertAdjacentHTML('afterbegin', html);
        container.insertAdjacentHTML('beforeend', footer);
    }

    // Mark profile dropdown item as active when on profile page
    function setActiveDropdownLink() {
        if (window.location.pathname === '/profile.html') {
            const profileLink = document.getElementById('profile-dropdown-link');
            if (profileLink) profileLink.classList.add('active');
        }
    }

    // Mirror admin-dropdown-link visibility to admin-nav-link
    function mirrorAdminNavLink() {
        const dropdownAdminLink = document.getElementById('admin-dropdown-link');
        const navAdminLink = document.getElementById('admin-nav-link');
        if (!dropdownAdminLink || !navAdminLink) return;

        const observer = new MutationObserver(() => {
            navAdminLink.style.display = dropdownAdminLink.style.display;
            const graveyardNavLink = document.getElementById('graveyard-nav-link');
            if (graveyardNavLink) graveyardNavLink.style.display = dropdownAdminLink.style.display;
        });
        observer.observe(dropdownAdminLink, { attributes: true, attributeFilter: ['style'] });
    }

    // Create and insert toggle button
    function createToggleButton() {
        const existingBtn = document.querySelector('.sidebar-toggle');
        if (existingBtn) return existingBtn;

        const toggleBtn = document.createElement('button');
        toggleBtn.className = 'sidebar-toggle';
        toggleBtn.innerHTML = '☰';
        toggleBtn.setAttribute('aria-label', 'Toggle sidebar navigation');
        toggleBtn.title = 'Toggle navigation';
        
        document.body.appendChild(toggleBtn);
        return toggleBtn;
    }

    // Create overlay for mobile
    function createOverlay() {
        const existingOverlay = document.querySelector('.sidebar-overlay');
        if (existingOverlay) return existingOverlay;

        const overlay = document.createElement('div');
        overlay.className = 'sidebar-overlay';
        document.body.appendChild(overlay);
        
        overlay.addEventListener('click', toggleSidebar);
        
        return overlay;
    }

    // Initialize sidebar state from localStorage
    function initializeSidebarState() {
        const sidebar = document.querySelector('.nav-menu');
        const main = document.querySelector('main');
        const header = document.querySelector('header');
        const footer = document.querySelector('.app-footer');
        const savedState = localStorage.getItem('sidebarCollapsed');

        if (!sidebar || !main) return;

        const isMobile = window.innerWidth <= 768;
        const shouldCollapse = savedState === 'true' || (savedState === null && isMobile);

        if (shouldCollapse) {
            sidebar.classList.add('collapsed');
            main.classList.add('sidebar-collapsed');
            if (header) header.classList.add('sidebar-collapsed');
            if (footer) footer.classList.add('sidebar-collapsed');
        } else {
            sidebar.classList.remove('collapsed');
            main.classList.remove('sidebar-collapsed');
            if (header) header.classList.remove('sidebar-collapsed');
            if (footer) footer.classList.remove('sidebar-collapsed');
        }

        updateToggleIcon(shouldCollapse);
    }

    // Show/hide external toggle button — only needed when sidebar is collapsed
    function updateToggleIcon(isCollapsed) {
        const toggleBtn = document.querySelector('.sidebar-toggle');
        if (!toggleBtn) return;
        toggleBtn.style.display = isCollapsed ? 'flex' : 'none';
    }

    // Toggle sidebar
    function toggleSidebar() {
        const sidebar = document.querySelector('.nav-menu');
        const main = document.querySelector('main');
        const header = document.querySelector('header');
        const footer = document.querySelector('.app-footer');
        const overlay = document.querySelector('.sidebar-overlay');

        if (!sidebar || !main) return;

        const isCurrentlyCollapsed = sidebar.classList.contains('collapsed');
        const isMobile = window.innerWidth <= 768;

        if (isCurrentlyCollapsed) {
            sidebar.classList.remove('collapsed');
            main.classList.remove('sidebar-collapsed');
            if (header) header.classList.remove('sidebar-collapsed');
            if (footer) footer.classList.remove('sidebar-collapsed');
            if (isMobile && overlay) overlay.classList.add('active');
        } else {
            sidebar.classList.add('collapsed');
            main.classList.add('sidebar-collapsed');
            if (header) header.classList.add('sidebar-collapsed');
            if (footer) footer.classList.add('sidebar-collapsed');
            if (overlay) overlay.classList.remove('active');
        }

        const newState = !isCurrentlyCollapsed;
        localStorage.setItem('sidebarCollapsed', newState);
        updateToggleIcon(newState);
    }

    // Close sidebar when clicking outside on mobile
    function handleOutsideClick(e) {
        if (window.innerWidth > 768) return;

        const sidebar = document.querySelector('.nav-menu');
        const toggleBtn = document.querySelector('.sidebar-toggle');
        const overlay = document.querySelector('.sidebar-overlay');
        
        if (!sidebar || sidebar.classList.contains('collapsed')) return;

        const clickedInsideSidebar = sidebar.contains(e.target);
        const clickedToggleBtn = toggleBtn && toggleBtn.contains(e.target);
        const clickedOverlay = overlay && overlay.contains(e.target);

        if (!clickedInsideSidebar && !clickedToggleBtn && !clickedOverlay) {
            toggleSidebar();
        }
    }

    // Handle window resize
    function handleResize() {
        const sidebar = document.querySelector('.nav-menu');
        const main = document.querySelector('main');
        
        if (!sidebar || !main) return;

        // On desktop, respect saved state
        // On mobile, default to collapsed if no state saved
        const savedState = localStorage.getItem('sidebarCollapsed');
        const isMobile = window.innerWidth <= 768;

        if (isMobile && savedState === null) {
            sidebar.classList.add('collapsed');
            main.classList.add('sidebar-collapsed');
            updateToggleIcon(true);
        }
    }

    // Set active link based on current page
    function setActiveLink() {
        const currentPath = window.location.pathname;
        const navLinks = document.querySelectorAll('.nav-link');
        
        navLinks.forEach(link => {
            const linkPath = new URL(link.href).pathname;
            
            if (linkPath === currentPath || 
                (currentPath === '/' && linkPath === '/') ||
                (currentPath === '/index.html' && linkPath === '/')) {
                link.classList.add('active');
            } else {
                link.classList.remove('active');
            }
        });
    }

    // Initialize on DOM ready
    function init() {
        injectHeader();
        const toggleBtn = createToggleButton();
        const overlay = createOverlay();
        initializeSidebarState();
        setActiveLink();
        setActiveDropdownLink();
        mirrorAdminNavLink();
        applyFeatureGates();

        // External toggle (shown only when collapsed) + collapse btn inside nav header
        toggleBtn.addEventListener('click', toggleSidebar);
        const navCollapseBtn = document.querySelector('.nav-collapse-btn');
        if (navCollapseBtn) navCollapseBtn.addEventListener('click', toggleSidebar);

        document.addEventListener('click', handleOutsideClick);
        window.addEventListener('resize', debounce(handleResize, 250));

        // Close sidebar on navigation link click (mobile)
        const navLinks = document.querySelectorAll('.nav-link');
        navLinks.forEach(link => {
            link.addEventListener('click', () => {
                if (window.innerWidth <= 768) {
                    setTimeout(() => {
                        const sidebar = document.querySelector('.nav-menu');
                        const main = document.querySelector('main');
                        if (sidebar && !sidebar.classList.contains('collapsed')) {
                            sidebar.classList.add('collapsed');
                            main.classList.add('sidebar-collapsed');
                            localStorage.setItem('sidebarCollapsed', 'true');
                            updateToggleIcon(true);
                        }
                    }, 100);
                }
            });
        });
    }

    // Debounce utility
    function debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    // Hide feature-gated nav links based on settings
    function applyFeatureGates() {
        // Marshal Guard nav link is always visible; the setting only controls
        // whether officers can create new events on the page itself.
    }

    // Auto-initialize
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
