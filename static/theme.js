// Theme Management
const THEME_KEY = 'lastwar-theme-preference';
const THEMES = {
    AUTO: 'auto',
    LIGHT: 'light',
    DARK: 'dark'
};

// Get current theme preference
function getThemePreference() {
    return localStorage.getItem(THEME_KEY) || THEMES.AUTO;
}

// Set theme preference
function setThemePreference(theme) {
    localStorage.setItem(THEME_KEY, theme);
    applyTheme(theme);
}

// Apply theme to document
function applyTheme(theme) {
    const html = document.documentElement;
    
    // Remove all theme classes
    html.classList.remove('theme-light', 'theme-dark', 'theme-auto');
    
    // Add the selected theme class
    if (theme === THEMES.LIGHT) {
        html.classList.add('theme-light');
        html.style.colorScheme = 'light';
    } else if (theme === THEMES.DARK) {
        html.classList.add('theme-dark');
        html.style.colorScheme = 'dark';
    } else {
        html.classList.add('theme-auto');
        html.style.colorScheme = 'light dark';
    }
    
    // Update theme selector if it exists
    updateThemeSelector(theme);

    // Set Chart.js global defaults so charts that don't specify explicit
    // label/grid colors inherit readable ones for the current theme.
    applyChartThemeDefaults();

    // Notify chart/visualization code so it can re-render with the correct colors
    // (Chart.js reads the theme once at build time; axis labels and grid lines
    // would otherwise stay in the previous theme's colors after a toggle).
    document.dispatchEvent(new CustomEvent('themechange', { detail: { theme } }));
}

// Set Chart.js global defaults (text + grid colors) to match the active theme.
// Called from applyTheme, which runs before a page's charts are created, so every
// chart inherits readable defaults even if it doesn't set its own colors.
function applyChartThemeDefaults() {
    if (typeof Chart === 'undefined' || !Chart.defaults) return;
    const isDark = document.documentElement.classList.contains('theme-dark');
    Chart.defaults.color = isDark ? '#ccc' : '#444';
    Chart.defaults.borderColor = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)';
}

// Update theme selector display
function updateThemeSelector(theme) {
    const themeOptions = document.querySelectorAll('.theme-option');
    themeOptions.forEach(option => {
        const optionTheme = option.dataset.theme;
        if (optionTheme === theme) {
            option.classList.add('active');
            option.innerHTML = option.innerHTML.replace('○', '●');
        } else {
            option.classList.remove('active');
            option.innerHTML = option.innerHTML.replace('●', '○');
        }
    });
}

// Initialize theme on page load
function initTheme() {
    const savedTheme = getThemePreference();
    applyTheme(savedTheme);
    
    // Setup theme option click handlers
    document.addEventListener('DOMContentLoaded', () => {
        setupThemeHandlers();
    });
}

// Setup theme option handlers
function setupThemeHandlers() {
    const themeOptions = document.querySelectorAll('.theme-option');
    themeOptions.forEach(option => {
        option.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            const theme = option.dataset.theme;
            setThemePreference(theme);
        });
    });
}

// Re-theme all existing Chart.js instances so axis labels, grid lines, and legends
// stay readable when the theme is toggled. Charts read the theme once at build time,
// so without this their labels/grid would remain in the previous theme's colors.
function rethemeCharts() {
    if (typeof Chart === 'undefined' || !Chart.instances) return;
    const isDark = document.documentElement.classList.contains('theme-dark');
    const labelColor    = isDark ? '#ccc' : '#444';
    const gridColor     = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.08)';
    const legendColor   = isDark ? '#ccc' : '#333';
    const segmentBorder = isDark ? '#1a202c' : '#fff';

    for (const key of Object.keys(Chart.instances)) {
        const chart = Chart.instances[key];
        if (!chart || !chart.options) continue;
        const opts = chart.options;

        // Cartesian (x/y) and radial (r) axes
        const scales = opts.scales || {};
        for (const axisName of Object.keys(scales)) {
            const axis = scales[axisName];
            if (axis.ticks) axis.ticks.color = labelColor;
            if (axis.grid) axis.grid.color = gridColor;
            if (axis.pointLabels) axis.pointLabels.color = labelColor; // radar
            if (axis.angleLines) axis.angleLines.color = gridColor;    // radar
        }

        const legend = opts.plugins && opts.plugins.legend;
        if (legend && legend.labels) legend.labels.color = legendColor;

        // Doughnut/pie segment separators
        if (chart.config && (chart.config.type === 'doughnut' || chart.config.type === 'pie')) {
            (chart.data.datasets || []).forEach(ds => {
                if (ds && !Array.isArray(ds.borderColor)) ds.borderColor = segmentBorder;
            });
        }

        chart.update('none');
    }
}

// Re-theme charts whenever the theme changes.
document.addEventListener('themechange', rethemeCharts);

// Initialize theme immediately
initTheme();
