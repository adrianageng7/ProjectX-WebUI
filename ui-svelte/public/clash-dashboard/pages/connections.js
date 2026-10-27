/**
 * pages/connections.js
 * Connections page — real-time via WebSocket, kill per-conn, kill all
 */

const ConnectionsPage = (() => {
    let ws = null;
    let paused = false;
    let lastData = { connections: [] };
    let filterQuery = '';

    // =========================================================
    // WEBSOCKET
    // =========================================================
    function start() {
        ws = ClashAPI.connectConnectionsWs((data) => {
            lastData = data;
            if (!paused) render(data);
        });
    }

    function stop() {
        if (ws) { ws.close(); ws = null; }
    }

    function toggle() {
        paused = !paused;
        const icon = document.getElementById('conn-toggle-icon');
        if (icon) icon.className = paused ? 'fa-solid fa-play' : 'fa-solid fa-pause';
        if (!paused) render(lastData);
    }

    // =========================================================
    // RENDER
    // =========================================================
    function render(data) {
        const conns = (data.connections || []).filter(c => {
            if (!filterQuery) return true;
            const q = filterQuery.toLowerCase();
            const host = (c.metadata?.host || c.metadata?.destinationIP || '').toLowerCase();
            return host.includes(q);
        });

        const container = document.getElementById('conn-cards');
        const badge = document.getElementById('conn-count');
        const subtitle = document.getElementById('conn-subtitle');

        const total = (data.connections || []).length;

        if (badge) {
            badge.textContent = total;
            badge.style.display = total > 0 ? 'grid' : 'none';
        }
        if (subtitle) {
            const dlTotal = formatBytesStr(data.downloadTotal || 0);
            const ulTotal = formatBytesStr(data.uploadTotal || 0);
            subtitle.textContent = `${total} koneksi • ↑${ulTotal} ↓${dlTotal}`;
        }

        // Update overview
        const ovConn = document.getElementById('ov-active-conn');
        if (ovConn) ovConn.textContent = total;
        if (window.OverviewPage && typeof window.OverviewPage.updateTotals === 'function') {
            window.OverviewPage.updateTotals(data.uploadTotal, data.downloadTotal);
        }

        if (conns.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fa-solid fa-network-wired"></i>
                    <p>${total === 0 ? 'Tidak ada koneksi aktif' : 'Tidak ada hasil untuk filter'}</p>
                </div>`;
            return;
        }

        // Efficient DOM update: rebuild
        container.innerHTML = conns.map(c => {
            const meta = c.metadata || {};
            const host = meta.host || meta.destinationIP || '?';
            const port = meta.destinationPort || '';
            const chain = (c.chains || []).join(' → ') || c.rule || '—';
            const type  = meta.type || '?';
            const network = meta.network || '?';
            const dl  = formatBytesStr(c.download || 0);
            const ul  = formatBytesStr(c.upload || 0);
            const age = timeSince(c.start);
            const ruleType = c.rulePayload || c.rule || '—';

            return `<div class="conn-card">
                <div class="conn-card-top">
                    <div class="conn-host">${host}${port ? ':' + port : ''}</div>
                    <button class="conn-kill-btn" title="Kill connection" onclick="ConnectionsPage.kill('${c.id}')">
                        <i class="fa-solid fa-xmark"></i>
                    </button>
                </div>
                <div class="conn-card-meta">
                    <span class="conn-tag">${type}</span>
                    <span class="conn-tag">${network.toUpperCase()}</span>
                    <span class="conn-tag" style="color:var(--accent);">${chain}</span>
                    <span class="conn-tag">${ruleType}</span>
                    <span class="conn-tag" style="color:var(--text-muted);">${age}</span>
                </div>
                <div class="conn-traffic">
                    <span style="color:var(--secondary);">↑ ${ul}</span>
                    <span style="color:var(--accent);">↓ ${dl}</span>
                </div>
            </div>`;
        }).join('');
    }

    // =========================================================
    // KILL
    // =========================================================
    async function kill(id) {
        try {
            await ClashAPI.killConnection(id);
            Toast.show('Koneksi dimatikan', 'success');
        } catch (e) {
            Toast.show(`Gagal: ${e.message}`, 'error');
        }
    }

    async function killAll() {
        if (!confirm('Matikan semua koneksi aktif?')) return;
        try {
            await ClashAPI.killAllConnections();
            Toast.show('Semua koneksi dimatikan', 'success');
        } catch (e) {
            Toast.show(`Gagal: ${e.message}`, 'error');
        }
    }

    // =========================================================
    // FILTER
    // =========================================================
    function filter(query) {
        filterQuery = query;
        render(lastData);
    }

    return { start, stop, toggle, kill, killAll, filter };
})();
