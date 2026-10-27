/**
 * pages/api.js
 * Clash / Mihomo API Client
 * Semua request ke backend Clash lewat sini
 */

const ClashAPI = (() => {
    // =========================================================
    // STATE
    // =========================================================
    let baseUrl = localStorage.getItem('clash_base_url') || 'http://127.0.0.1:9090';
    let secret  = localStorage.getItem('clash_secret') || '';
    let _wsTraffic = null;
    let _wsConnections = null;
    let _wsLogs = null;
    let _connected = false;

    // =========================================================
    // HEADERS
    // =========================================================
    function headers(extra = {}) {
        const h = { 'Content-Type': 'application/json', ...extra };
        if (secret) h['Authorization'] = `Bearer ${secret}`;
        return h;
    }

    // =========================================================
    // HTTP HELPERS
    // =========================================================
    async function get(path) {
        const r = await fetch(`${baseUrl}${path}`, { headers: headers() });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
    }

    async function patch(path, body) {
        const r = await fetch(`${baseUrl}${path}`, {
            method: 'PATCH',
            headers: headers(),
            body: JSON.stringify(body)
        });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r;
    }

    async function put(path, body) {
        const r = await fetch(`${baseUrl}${path}`, {
            method: 'PUT',
            headers: headers(),
            body: JSON.stringify(body)
        });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r;
    }

    async function del(path) {
        const r = await fetch(`${baseUrl}${path}`, {
            method: 'DELETE',
            headers: headers()
        });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r;
    }

    async function putRaw(path, body, contentType = 'text/plain') {
        const h = {};
        if (secret) h['Authorization'] = `Bearer ${secret}`;
        h['Content-Type'] = contentType;
        const r = await fetch(`${baseUrl}${path}`, {
            method: 'PUT',
            headers: h,
            body
        });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r;
    }

    // =========================================================
    // API METHODS
    // =========================================================

    /** Get Clash version */
    async function getVersion() {
        return get('/version');
    }

    /** Get general configs */
    async function getConfigs() {
        return get('/configs');
    }

    /** Update configs (PATCH) */
    async function updateConfigs(body) {
        return patch('/configs', body);
    }

    /** Reload configs from file */
    async function reloadConfigs() {
        return put('/configs?force=false', {});
    }

    /** Get all proxies */
    async function getProxies() {
        return get('/proxies');
    }

    /** Get all proxy providers */
    async function getProviders() {
        try {
            return await get('/providers/proxies');
        } catch {
            return { providers: {} };
        }
    }

    /** Healthcheck an entire proxy provider */
    async function healthcheckProvider(providerName) {
        return get(`/providers/proxies/${encodeURIComponent(providerName)}/healthcheck`);
    }

    /** Select proxy in a group */
    async function selectProxy(groupName, proxyName) {
        return put(`/proxies/${encodeURIComponent(groupName)}`, { name: proxyName });
    }

    /** Test latency of a proxy (supports both regular and provider proxies) */
    async function testLatency(proxyName, providerName = '', url = 'http://cp.cloudflare.com/generate_204', timeout = 5000) {
        if (providerName) {
            try {
                return await get(`/providers/proxies/${encodeURIComponent(providerName)}/${encodeURIComponent(proxyName)}/healthcheck?url=${encodeURIComponent(url)}&timeout=${timeout}`);
            } catch (e) {
                // fallback to standard delay endpoint if provider route doesn't match
            }
        }
        return get(`/proxies/${encodeURIComponent(proxyName)}/delay?url=${encodeURIComponent(url)}&timeout=${timeout}`);
    }

    /** Get all rules */
    async function getRules() {
        return get('/rules');
    }

    /** Toggle rule enabled/disabled (Mihomo PATCH /rules/disable) */
    async function toggleRule(index, disabled) {
        return patch('/rules/disable', { [String(index)]: disabled });
    }

    /** Get rule providers */
    async function getRuleProviders() {
        try {
            return await get('/providers/rules');
        } catch {
            return { providers: {} };
        }
    }

    /** Update a rule provider */
    async function updateRuleProvider(name) {
        return put(`/providers/rules/${encodeURIComponent(name)}`, {});
    }

    /** Get all connections */
    async function getConnections() {
        return get('/connections');
    }

    /** Kill a specific connection */
    async function killConnection(id) {
        return del(`/connections/${id}`);
    }

    /** Kill all connections */
    async function killAllConnections() {
        return del('/connections');
    }

    /** Set proxy mode: rule | global | direct */
    async function setMode(mode) {
        try {
            await patch('/configs', { mode });
            Toast.show(`Mode diubah ke: ${mode.toUpperCase()}`, 'success');
            updateModeBadge(mode);
        } catch (e) {
            Toast.show('Gagal mengubah mode', 'error');
        }
    }

    // =========================================================
    // WEBSOCKETS
    // =========================================================

    /** Traffic WebSocket — callback(up, down, totalUp, totalDown) */
    function connectTraffic(callback) {
        closeWs(_wsTraffic);
        const protocol = baseUrl.startsWith('https') ? 'wss' : 'ws';
        const host = baseUrl.replace(/^https?:\/\//, '');
        const url = `${protocol}://${host}/traffic${secret ? `?token=${secret}` : ''}`;
        _wsTraffic = new WebSocket(url);
        _wsTraffic.onmessage = (e) => {
            try { callback(JSON.parse(e.data)); } catch {}
        };
        _wsTraffic.onerror = () => setConnected(false);
        _wsTraffic.onclose = () => {};
        return _wsTraffic;
    }

    /** Memory WebSocket — callback({inuse}) */
    function connectMemory(callback) {
        const protocol = baseUrl.startsWith('https') ? 'wss' : 'ws';
        const host = baseUrl.replace(/^https?:\/\//, '');
        const url = `${protocol}://${host}/memory${secret ? `?token=${secret}` : ''}`;
        const ws = new WebSocket(url);
        ws.onmessage = (e) => {
            try { callback(JSON.parse(e.data)); } catch {}
        };
        return ws;
    }

    /** Connections WebSocket — callback(data) */
    function connectConnectionsWs(callback) {
        closeWs(_wsConnections);
        const protocol = baseUrl.startsWith('https') ? 'wss' : 'ws';
        const host = baseUrl.replace(/^https?:\/\//, '');
        const url = `${protocol}://${host}/connections${secret ? `?token=${secret}` : ''}`;
        _wsConnections = new WebSocket(url);
        _wsConnections.onmessage = (e) => {
            try { callback(JSON.parse(e.data)); } catch {}
        };
        return _wsConnections;
    }

    /** Logs WebSocket — callback({type, payload}) */
    function connectLogs(level, callback) {
        closeWs(_wsLogs);
        const protocol = baseUrl.startsWith('https') ? 'wss' : 'ws';
        const host = baseUrl.replace(/^https?:\/\//, '');
        const url = `${protocol}://${host}/logs?level=${level}${secret ? `&token=${secret}` : ''}`;
        _wsLogs = new WebSocket(url);
        _wsLogs.onmessage = (e) => {
            try { callback(JSON.parse(e.data)); } catch {}
        };
        return _wsLogs;
    }

    function closeWs(ws) {
        if (ws && ws.readyState !== WebSocket.CLOSED) ws.close();
    }

    function closeAllWs() {
        closeWs(_wsTraffic);
        closeWs(_wsConnections);
        closeWs(_wsLogs);
    }

    // =========================================================
    // CONNECTION STATUS
    // =========================================================
    function setConnected(ok) {
        _connected = ok;
        const dot = document.getElementById('conn-dot');
        const lbl = document.getElementById('conn-label');
        if (dot) {
            dot.className = 'conn-dot ' + (ok ? 'connected' : 'error');
        }
        if (lbl) lbl.textContent = ok ? 'Connected' : 'Offline';
    }

    async function testConnection() {
        try {
            await getVersion();
            setConnected(true);
            return true;
        } catch {
            setConnected(false);
            return false;
        }
    }

    function updateModeBadge(mode) {
        const badge = document.getElementById('mode-badge');
        if (!badge) return;
        badge.textContent = mode.charAt(0).toUpperCase() + mode.slice(1);
        badge.className = `mode-badge ${mode}`;

        // Update mode buttons
        ['rule','global','direct'].forEach(m => {
            const btn = document.getElementById(`mode-${m}`);
            if (btn) btn.className = `btn ${m === mode ? 'btn-primary' : 'btn-ghost'}`;
        });
    }

    // =========================================================
    // CONFIG SETTERS
    // =========================================================
    function setConfig(url, sec) {
        baseUrl = url.replace(/\/$/, '');
        secret = sec;
        localStorage.setItem('clash_base_url', baseUrl);
        localStorage.setItem('clash_secret', secret);
    }

    function getBaseUrl() { return baseUrl; }
    function getSecret() { return secret; }

    // =========================================================
    // PUBLIC API
    // =========================================================
    return {
        get, patch, put, del, putRaw,
        getVersion, getConfigs, updateConfigs, reloadConfigs,
        getProxies, getProviders, healthcheckProvider, selectProxy, testLatency,
        getRules, toggleRule, getRuleProviders, updateRuleProvider,
        getConnections, killConnection, killAllConnections,
        setMode,
        connectTraffic, connectMemory, connectConnectionsWs, connectLogs,
        closeAllWs,
        setConnected, testConnection, updateModeBadge,
        setConfig, getBaseUrl, getSecret
    };
})();

// =========================================================
// TOAST SYSTEM
// =========================================================
const Toast = (() => {
    function show(msg, type = 'info', duration = 3000) {
        const container = document.getElementById('toast-container');
        const t = document.createElement('div');
        t.className = `toast ${type}`;
        const icons = { success: 'fa-check-circle', error: 'fa-circle-xmark', info: 'fa-circle-info' };
        t.innerHTML = `<i class="fa-solid ${icons[type] || 'fa-circle-info'}"></i> ${msg}`;
        container.appendChild(t);
        setTimeout(() => {
            t.style.animation = 'slideOut 0.2s ease forwards';
            setTimeout(() => t.remove(), 200);
        }, duration);
    }
    return { show };
})();

// =========================================================
// UTILITIES
// =========================================================
function formatBytes(bytes) {
    if (bytes === 0) return { val: '0', unit: 'B' };
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return { val: (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0), unit: sizes[i] };
}

function formatBytesStr(bytes) {
    const f = formatBytes(bytes);
    return `${f.val} ${f.unit}`;
}

function formatSpeed(bytesPerSec) {
    const f = formatBytes(bytesPerSec);
    return `${f.val} ${f.unit}/s`;
}

function timeSince(dateStr) {
    const diff = Date.now() - new Date(dateStr).getTime();
    const s = Math.floor(diff / 1000);
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m`;
    const h = Math.floor(m / 60);
    return `${h}h ${m % 60}m`;
}
