/**
 * pages/config.js
 * Config page — view & edit Clash YAML config
 */

const ConfigPage = (() => {
    let originalConfig = '';

    // =========================================================
    // LOAD CONFIG
    // =========================================================
    async function load() {
        const textarea = document.getElementById('config-textarea');
        textarea.value = 'Memuat config...';
        textarea.disabled = true;

        try {
            // Mihomo supports GET /configs to get current config
            // But it returns JSON. For YAML, we use PUT /configs with path.
            // Most implementations serve raw YAML via GET /configs with accept header.
            const response = await fetch(`${ClashAPI.getBaseUrl()}/configs`, {
                headers: {
                    'Accept': 'text/yaml, application/yaml, application/json',
                    ...(ClashAPI.getSecret() ? { 'Authorization': `Bearer ${ClashAPI.getSecret()}` } : {})
                }
            });

            const contentType = response.headers.get('content-type') || '';
            const text = await response.text();

            if (contentType.includes('json')) {
                // Format JSON nicely
                try {
                    const json = JSON.parse(text);
                    originalConfig = JSON.stringify(json, null, 2);
                    textarea.value = originalConfig;
                    textarea.disabled = false;
                } catch {
                    textarea.value = text;
                    textarea.disabled = false;
                }
            } else {
                // YAML
                originalConfig = text;
                textarea.value = text;
                textarea.disabled = false;
            }
        } catch (e) {
            textarea.value = `# Gagal memuat config: ${e.message}\n# Pastikan Clash berjalan dan URL/secret benar.`;
            textarea.disabled = false;
        }
    }

    // =========================================================
    // SAVE & RELOAD CONFIG
    // =========================================================
    async function save() {
        const textarea = document.getElementById('config-textarea');
        const content = textarea.value;

        if (!content.trim()) {
            Toast.show('Config kosong!', 'error');
            return;
        }

        // Check if user changed anything
        if (content === originalConfig) {
            Toast.show('Tidak ada perubahan', 'info');
            return;
        }

        try {
            // Reload with force=true triggers Clash to reload from file
            // Alternatively, PATCH /configs with allowed keys
            // Mihomo: PUT /configs?force=true with body { path: "/path/to/config" }
            // For direct YAML update, we need to know the path.
            // Most dashboards just PATCH individual keys.
            // We'll try to send patch of supported keys.

            Toast.show('Config hanya bisa diedit via file langsung. Gunakan PATCH untuk key spesifik.', 'info');

            // Try reloadConfigs
            await ClashAPI.reloadConfigs();
            Toast.show('Clash config di-reload!', 'success');
        } catch (e) {
            Toast.show(`Gagal reload config: ${e.message}`, 'error');
        }
    }

    return { load, save };
})();
