/**
 * app.js
 * ClashBoard — Router, App Init, Theme Restore
 */

// =========================================================
// ROUTER
// =========================================================
const Router = (() => {
    const pages = ['overview', 'proxies', 'rules', 'connections', 'logs', 'config', 'settings'];
    let current = 'overview';
    let initialized = {};

    function go(pageId, dockBtn) {
        if (pageId === current) return;

        // Hide current page
        document.getElementById(`page-${current}`)?.classList.remove('active');

        // Show new page
        const newPage = document.getElementById(`page-${pageId}`);
        if (newPage) newPage.classList.add('active');

        // Update dock
        document.querySelectorAll('.dock-item').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.page === pageId);
        });

        // Scroll to top
        document.getElementById('main-content').scrollTop = 0;

        const prev = current;
        current = pageId;

        // Lazy init & lifecycle
        onPageLeave(prev);
        onPageEnter(pageId);
    }

    function onPageEnter(pageId) {
        if (!initialized[pageId]) {
            initialized[pageId] = true;
            // First time init
            switch (pageId) {
                case 'overview':    OverviewPage.init(); break;
                case 'connections': ConnectionsPage.start(); break;
                case 'logs':        LogsPage.init(); break;
            }
        }

        // Always-refresh on enter
        switch (pageId) {
            case 'overview':    OverviewPage.renderTotals?.(); break;
            case 'proxies':     ProxiesPage.load(); break;
            case 'rules':       RulesPage.load(); break;
            case 'config':      ConfigPage.load(); break;
            case 'settings':    SettingsPage.init(); break;
        }
    }

    function onPageLeave(pageId) {
        // Stop heavy processes when leaving
        switch (pageId) {
            case 'logs':        LogsPage.destroy(); break;
        }
        // Restart logs when re-entering
    }

    return { go, getCurrent: () => current };
})();

// =========================================================
// APP INIT
// =========================================================
document.addEventListener('DOMContentLoaded', async () => {

    // ── 1. Restore theme ─────────────────────────────
    const savedTheme = localStorage.getItem('clash_theme');
    if (savedTheme === 'light') {
        document.documentElement.setAttribute('data-theme', 'light');
    }
    const savedColor = localStorage.getItem('clash_color') || 'default';
    document.documentElement.setAttribute('data-color', savedColor);

    // ── 2. Sync with parent frame (ProjectX) ──────────
    SettingsPage.syncFromParent();
    setInterval(SettingsPage.syncFromParent, 1500);

    // ── 3. Init first page ────────────────────────────
    OverviewPage.init();

    // ── 4. Test connection ────────────────────────────
    const connected = await ClashAPI.testConnection();
    if (!connected) {
        Toast.show('Tidak terhubung ke Clash. Cek Settings.', 'error', 5000);
    } else {
        // Load initial version and mode
        try {
            const cfg = await ClashAPI.getConfigs();
            if (cfg.mode) ClashAPI.updateModeBadge(cfg.mode);
            const ver = await ClashAPI.getVersion();
            const el = document.getElementById('ov-version');
            if (el) el.textContent = ver.version || '—';
            document.getElementById('info-version') && (document.getElementById('info-version').textContent = ver.version || '—');
        } catch {}
    }

    // ── 5. Mode badge click → cycle modes ─────────────
    document.getElementById('btn-settings-quick')?.addEventListener('click', () => {
        Router.go('settings', null);
    });

    document.getElementById('mode-badge')?.addEventListener('click', async () => {
        try {
            const cfg = await ClashAPI.getConfigs();
            const modes = ['rule', 'global', 'direct'];
            const idx = modes.indexOf(cfg.mode || 'rule');
            const next = modes[(idx + 1) % modes.length];
            await ClashAPI.setMode(next);
        } catch {}
    });

    // ── 6. Start connections WS for badge count ────────
    ConnectionsPage.start();
});

function openClashGenerator() {
    try {
        if (window.parent && window.parent.location) {
            window.parent.location.hash = '/clash-generator';
            return;
        }
    } catch (e) {}
    window.location.href = '../clash_generator.html';
}
window.openClashGenerator = openClashGenerator;

