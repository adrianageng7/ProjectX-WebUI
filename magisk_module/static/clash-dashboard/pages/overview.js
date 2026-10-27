/**
 * pages/overview.js
 * Overview page — real-time traffic chart, stats, mode switch
 */

const OverviewPage = (() => {
    // =========================================================
    // CHART SETUP
    // =========================================================
    const MAX_POINTS = 60;
    const upData   = new Array(MAX_POINTS).fill(0);
    const downData = new Array(MAX_POINTS).fill(0);
    let canvas, ctx;
    let animFrame;

    function initChart() {
        canvas = document.getElementById('speed-chart');
        if (!canvas) return;
        ctx = canvas.getContext('2d');
        resize();
        window.addEventListener('resize', resize);
        drawChart();
    }

    function resize() {
        if (!canvas) return;
        canvas.width  = canvas.offsetWidth * window.devicePixelRatio;
        canvas.height = canvas.offsetHeight * window.devicePixelRatio;
        ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    }

    function getCSSVar(name) {
        return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    }

    function drawChart() {
        if (!canvas || !ctx) return;
        const W = canvas.offsetWidth;
        const H = canvas.offsetHeight;

        ctx.clearRect(0, 0, W, H);

        const maxVal = Math.max(...upData, ...downData, 1024);

        function drawLine(data, color) {
            ctx.beginPath();
            ctx.strokeStyle = color;
            ctx.lineWidth = 1.5;
            ctx.lineJoin = 'round';

            const fill = ctx.createLinearGradient(0, 0, 0, H);
            fill.addColorStop(0, color + '40');
            fill.addColorStop(1, color + '00');

            for (let i = 0; i < data.length; i++) {
                const x = (i / (MAX_POINTS - 1)) * W;
                const y = H - (data[i] / maxVal) * (H - 8);
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            }

            // Fill area
            ctx.save();
            ctx.lineTo(W, H);
            ctx.lineTo(0, H);
            ctx.closePath();
            ctx.fillStyle = fill;
            ctx.fill();
            ctx.restore();

            // Re-draw stroke
            ctx.beginPath();
            ctx.strokeStyle = color;
            for (let i = 0; i < data.length; i++) {
                const x = (i / (MAX_POINTS - 1)) * W;
                const y = H - (data[i] / maxVal) * (H - 8);
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            }
            ctx.stroke();
        }

        drawLine(downData, getCSSVar('--accent'));
        drawLine(upData, getCSSVar('--secondary'));

        animFrame = requestAnimationFrame(drawChart);
    }

    // =========================================================
    // TRAFFIC & TOTALS
    // =========================================================
    let wsTraffic = null;
    let wsMemory  = null;
    let syncTimer = null;
    let totalUp = 0, totalDown = 0;

    function loadSavedTotals() {
        const savedUp = localStorage.getItem('clash_total_up');
        const savedDown = localStorage.getItem('clash_total_down');
        if (savedUp !== null) totalUp = parseInt(savedUp, 10) || 0;
        if (savedDown !== null) totalDown = parseInt(savedDown, 10) || 0;
        renderTotals();
    }

    function saveTotals() {
        try {
            localStorage.setItem('clash_total_up', String(totalUp));
            localStorage.setItem('clash_total_down', String(totalDown));
        } catch (e) {}
    }

    function renderTotals() {
        const tuFmt = formatBytes(totalUp);
        const tdFmt = formatBytes(totalDown);
        const elUp = document.getElementById('stat-total-up');
        const elUpUnit = document.getElementById('stat-total-up-unit');
        const elDown = document.getElementById('stat-total-down');
        const elDownUnit = document.getElementById('stat-total-down-unit');
        if (elUp) elUp.textContent = tuFmt.val;
        if (elUpUnit) elUpUnit.textContent = tuFmt.unit;
        if (elDown) elDown.textContent = tdFmt.val;
        if (elDownUnit) elDownUnit.textContent = tdFmt.unit;
    }

    function updateTotals(kernelUp, kernelDown) {
        let changed = false;
        if (typeof kernelUp === 'number' && !isNaN(kernelUp)) {
            if (kernelUp !== totalUp) {
                totalUp = kernelUp;
                changed = true;
            }
        }
        if (typeof kernelDown === 'number' && !isNaN(kernelDown)) {
            if (kernelDown !== totalDown) {
                totalDown = kernelDown;
                changed = true;
            }
        }
        if (changed) {
            saveTotals();
            renderTotals();
        }
    }

    async function syncFromConnections() {
        try {
            const conns = await ClashAPI.getConnections();
            if (conns) {
                updateTotals(conns.uploadTotal, conns.downloadTotal);
                const el = document.getElementById('ov-active-conn');
                if (el && Array.isArray(conns.connections)) {
                    el.textContent = conns.connections.length;
                }
            }
        } catch (e) {}
    }

    function startTraffic() {
        wsTraffic = ClashAPI.connectTraffic((data) => {
            const up   = data.up   || 0;
            const down = data.down || 0;

            upData.push(up); upData.shift();
            downData.push(down); downData.shift();

            // Update stats
            const upFmt   = formatBytes(up);
            const downFmt = formatBytes(down);
            const elUp = document.getElementById('stat-up');
            const elUpUnit = document.getElementById('stat-up-unit');
            const elDown = document.getElementById('stat-down');
            const elDownUnit = document.getElementById('stat-down-unit');
            if (elUp) elUp.textContent = upFmt.val;
            if (elUpUnit) elUpUnit.textContent = upFmt.unit + '/s';
            if (elDown) elDown.textContent = downFmt.val;
            if (elDownUnit) elDownUnit.textContent = downFmt.unit + '/s';

            ClashAPI.setConnected(true);
        });

        wsTraffic.onerror = () => ClashAPI.setConnected(false);
    }

    function startMemory() {
        wsMemory = ClashAPI.connectMemory((data) => {
            const mb = Math.round((data.inuse || 0) / 1024 / 1024);
            const el = document.getElementById('ov-memory');
            if (el) el.textContent = mb;
        });
    }

    // =========================================================
    // INIT
    // =========================================================
    async function init() {
        initChart();
        loadSavedTotals();
        startTraffic();
        startMemory();
        await syncFromConnections();

        if (syncTimer) clearInterval(syncTimer);
        syncTimer = setInterval(syncFromConnections, 5000);

        try {
            const ver = await ClashAPI.getVersion();
            const el = document.getElementById('ov-version');
            if (el) el.textContent = ver.version || '—';
            document.getElementById('info-version') && (document.getElementById('info-version').textContent = ver.version || '—');

            const cfg = await ClashAPI.getConfigs();
            if (cfg.mode) ClashAPI.updateModeBadge(cfg.mode);
        } catch (e) {}
    }

    function destroy() {
        cancelAnimationFrame(animFrame);
        if (wsTraffic) wsTraffic.close();
        if (wsMemory) wsMemory.close();
        if (syncTimer) {
            clearInterval(syncTimer);
            syncTimer = null;
        }
    }

    return { init, destroy, updateTotals, renderTotals };
})();

window.OverviewPage = OverviewPage;
