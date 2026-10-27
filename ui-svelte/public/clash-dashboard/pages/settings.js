/**
 * pages/settings.js
 * Settings page — backend URL, secret, theme, color scheme
 */

const SettingsPage = (() => {
    // =========================================================
    // INIT — populate inputs from localStorage
    // =========================================================
    function init() {
        const urlInput    = document.getElementById('cfg-base-url');
        const secretInput = document.getElementById('cfg-secret');
        const themeToggle = document.getElementById('toggle-theme');

        if (urlInput)    urlInput.value    = ClashAPI.getBaseUrl();
        if (secretInput) secretInput.value = ClashAPI.getSecret();

        // Theme
        const isDark = document.documentElement.getAttribute('data-theme') !== 'light';
        if (themeToggle) themeToggle.checked = !isDark;

        // Color
        const savedColor = localStorage.getItem('clash_color') || 'default';
        document.querySelectorAll('[data-color]').forEach(btn => {
            btn.classList.toggle('btn-primary', btn.dataset.color === savedColor);
            btn.classList.toggle('btn-ghost',   btn.dataset.color !== savedColor);
        });
        document.documentElement.setAttribute('data-color', savedColor);

        // Update info panel
        const infoBackend = document.getElementById('info-backend');
        if (infoBackend) infoBackend.textContent = ClashAPI.getBaseUrl();
    }

    // =========================================================
    // SAVE CONNECTION SETTINGS
    // =========================================================
    async function save() {
        const url    = document.getElementById('cfg-base-url')?.value?.trim() || '';
        const secret = document.getElementById('cfg-secret')?.value?.trim() || '';

        if (!url) {
            Toast.show('URL backend tidak boleh kosong', 'error');
            return;
        }

        ClashAPI.setConfig(url, secret);

        const infoBackend = document.getElementById('info-backend');
        if (infoBackend) infoBackend.textContent = url;

        Toast.show('Pengaturan disimpan, menghubungkan...', 'info');

        const ok = await ClashAPI.testConnection();
        if (ok) {
            Toast.show('Berhasil terhubung ke Clash!', 'success');
            // Re-init overview
            OverviewPage.init();
        } else {
            Toast.show('Tidak dapat terhubung. Periksa URL dan secret.', 'error');
        }
    }

    // =========================================================
    // TEST CONNECTION
    // =========================================================
    async function test() {
        Toast.show('Menguji koneksi...', 'info');
        const ok = await ClashAPI.testConnection();
        if (ok) {
            Toast.show('Koneksi berhasil! ✓', 'success');
        } else {
            Toast.show('Koneksi gagal. Periksa backend.', 'error');
        }
    }

    // =========================================================
    // THEME
    // =========================================================
    function toggleTheme(isLight) {
        if (isLight) {
            document.documentElement.setAttribute('data-theme', 'light');
            localStorage.setItem('clash_theme', 'light');
        } else {
            document.documentElement.removeAttribute('data-theme');
            localStorage.removeItem('clash_theme');
        }

        // Sync with parent if inside iframe
        try {
            if (window.parent && window.parent.document) {
                const parentHtml = window.parent.document.documentElement;
                const parentTheme = parentHtml.getAttribute('data-theme');
                // Just set our own theme
            }
        } catch (e) {}
    }

    // =========================================================
    // COLOR SCHEME
    // =========================================================
    function setColor(color, btn) {
        document.documentElement.setAttribute('data-color', color);
        localStorage.setItem('clash_color', color);

        // Update buttons
        document.querySelectorAll('[data-color]').forEach(b => {
            b.classList.toggle('btn-primary', b.dataset.color === color);
            b.classList.toggle('btn-ghost',   b.dataset.color !== color);
        });
    }

    // =========================================================
    // PARENT THEME SYNC (ProjectX integration)
    // =========================================================
    function syncFromParent() {
        try {
            if (window.parent && window.parent !== window && window.parent.document) {
                const parentHtml = window.parent.document.documentElement;
                const parentTheme = parentHtml.getAttribute('data-theme');
                const parentColor = parentHtml.getAttribute('data-color');

                if (parentTheme === 'light') {
                    document.documentElement.setAttribute('data-theme', 'light');
                } else {
                    document.documentElement.removeAttribute('data-theme');
                }

                if (parentColor) {
                    document.documentElement.setAttribute('data-color', parentColor);
                }
            }
        } catch (e) {}
    }

    return { init, save, test, toggleTheme, setColor, syncFromParent };
})();
