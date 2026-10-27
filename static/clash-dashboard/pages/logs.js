/**
 * pages/logs.js
 * Logs page — real-time log viewer via WebSocket
 */

const LogsPage = (() => {
    let ws = null;
    let currentLevel = 'info';
    const MAX_LOGS = 300;
    let logBuffer = [];

    // =========================================================
    // WEBSOCKET
    // =========================================================
    function connect(level) {
        if (ws) { ws.close(); ws = null; }

        ws = ClashAPI.connectLogs(level, (entry) => {
            addLog(entry);
        });
    }

    // =========================================================
    // ADD LOG ENTRY
    // =========================================================
    function addLog(entry) {
        const scroll = document.getElementById('log-scroll');
        if (!scroll) return;

        const type    = (entry.type || 'info').toLowerCase();
        const payload = entry.payload || '';
        const now = new Date();
        const time = now.toTimeString().slice(0, 8);

        logBuffer.push({ type, payload, time });
        if (logBuffer.length > MAX_LOGS) logBuffer.shift();

        const el = document.createElement('div');
        el.className = 'log-entry';
        el.innerHTML = `
            <span class="log-time">${time}</span>
            <span class="log-level ${type}">${type.toUpperCase().slice(0,4)}</span>
            <span class="log-msg">${escapeHtml(payload)}</span>
        `;

        scroll.appendChild(el);

        // Trim old entries
        while (scroll.children.length > MAX_LOGS) {
            scroll.removeChild(scroll.firstChild);
        }

        // Auto scroll
        const autoScroll = document.getElementById('log-autoscroll');
        if (!autoScroll || autoScroll.checked) {
            scroll.scrollTop = scroll.scrollHeight;
        }
    }

    function escapeHtml(text) {
        const d = document.createElement('div');
        d.textContent = text;
        return d.innerHTML;
    }

    // =========================================================
    // LEVEL
    // =========================================================
    function setLevel(level, btn) {
        currentLevel = level;

        // Update active button
        document.querySelectorAll('.log-level-btn').forEach(b => b.classList.remove('active'));
        if (btn) btn.classList.add('active');

        connect(level);
        Toast.show(`Log level: ${level}`, 'info');
    }

    // =========================================================
    // CLEAR
    // =========================================================
    function clear() {
        const scroll = document.getElementById('log-scroll');
        if (scroll) scroll.innerHTML = '';
        logBuffer = [];
    }

    // =========================================================
    // INIT / DESTROY
    // =========================================================
    function init() {
        connect(currentLevel);
    }

    function destroy() {
        if (ws) { ws.close(); ws = null; }
    }

    return { init, destroy, setLevel, clear };
})();
