// Alliance Branding Loader
// This file loads alliance settings and applies branding across the site

// Load and apply alliance branding
async function loadAllianceBranding() {
    // The login page is unauthenticated by definition; /api/settings would
    // 401 and log a console error on every load. Keep default branding there.
    if (window.location.pathname.endsWith('/login.html')) return;
    try {
        const response = await fetch('/api/settings');
        if (response.ok) {
            const settings = await response.json();
            const allianceName = settings.alliance_name || 'Last War: Survival';
            const allianceShort = settings.alliance_short_name || 'LWS';
            
            // Update page title
            const currentTitle = document.title;
            if (currentTitle.includes('Last War: Survival')) {
                document.title = currentTitle.replace('Last War: Survival', allianceName);
            } else if (currentTitle.includes(' - ')) {
                // Format: "PageName - Alliance"
                const parts = currentTitle.split(' - ');
                document.title = `${parts[0]} - ${allianceName}`;
            } else {
                document.title = `${allianceName} - Alliance Manager`;
            }
            
            // Update header h1 if it exists
            const headerH1 = document.querySelector('header h1');
            if (headerH1) {
                // Keep the emoji if present
                const text = headerH1.textContent;
                const emojiMatch = text.match(/^([\u{1F300}-\u{1F9FF}]\s*)/u);
                if (emojiMatch) {
                    headerH1.textContent = `${emojiMatch[1]}${allianceName}`;
                } else {
                    headerH1.textContent = allianceName;
                }
            }
            
            // Update header h2 if it exists (subtitle)
            const headerH2 = document.querySelector('header h2');
            if (headerH2) {
                const subtitle = headerH2.textContent;
                // Replace generic terms with alliance tag
                if (subtitle.includes('Alliance')) {
                    headerH2.textContent = subtitle.replace(/Alliance\s+Member\s+Manager/i, `${allianceShort} Member Manager`);
                } else if (!subtitle.includes(allianceShort)) {
                    // If subtitle doesn't have alliance info, prepend it
                    headerH2.textContent = `${allianceShort} - ${subtitle}`;
                }
            }
            
            // Store branding in session for other scripts
            sessionStorage.setItem('alliance_name', allianceName);
            sessionStorage.setItem('alliance_short_name', allianceShort);
            
        }
    } catch (error) {
        console.error('Failed to load alliance branding:', error);
        // Silently fail - use default branding from HTML
    }
}

// Auto-load branding when script is included
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', loadAllianceBranding);
} else {
    // DOM already loaded
    loadAllianceBranding();
}
