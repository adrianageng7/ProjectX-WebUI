<script>
    import { onMount, onDestroy } from 'svelte';

    let isRunning = false;
    let eventSource = null;

    // Test states
    let phase = 'idle'; // 'idle', 'init', 'ping', 'download', 'upload', 'complete', 'stopped', 'error'
    let progress = 0;
    let currentSpeed = 0;
    let ping = 0;
    let jitter = 0;
    let download = 0;
    let upload = 0;

    let ip = '-';
    let city = '-';
    let country = '-';
    let colo = '-';
    let isp = '-';
    let message = '';
    let lastTestedTime = '';

    // Load last result on mount
    async function loadLastResult() {
        try {
            const res = await fetch('/api/speedtest/last?_=' + Date.now());
            if (res.ok) {
                const data = await res.json();
                if (data.status === 'success' && data.data) {
                    const r = data.data;
                    ping = r.ping || 0;
                    jitter = r.jitter || 0;
                    download = r.download || 0;
                    upload = r.upload || 0;
                    ip = r.ip || '-';
                    city = r.city || '-';
                    country = r.country || '-';
                    colo = r.colo || '-';
                    isp = r.isp || '-';
                    lastTestedTime = r.timestamp || '';
                    phase = 'ready';
                }
            }
        } catch (e) {
            console.error('Error loading last speedtest:', e);
        }
    }

    function startSpeedtest() {
        if (isRunning) return;

        // Reset display values for current run
        isRunning = true;
        phase = 'init';
        progress = 2;
        currentSpeed = 0;
        ping = 0;
        jitter = 0;
        download = 0;
        upload = 0;
        message = 'Menyiapkan koneksi ke server Cloudflare...';

        if (eventSource) {
            eventSource.close();
        }

        eventSource = new EventSource('/api/speedtest/stream');

        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);

                phase = data.phase || phase;
                if (data.progress !== undefined) progress = data.progress;
                if (data.current_speed !== undefined) currentSpeed = data.current_speed;
                if (data.ping !== undefined) ping = data.ping;
                if (data.jitter !== undefined) jitter = data.jitter;
                if (data.download !== undefined) download = data.download;
                if (data.upload !== undefined) upload = data.upload;

                if (data.ip) ip = data.ip;
                if (data.city) city = data.city;
                if (data.country) country = data.country;
                if (data.colo) colo = data.colo;
                if (data.isp) isp = data.isp;
                if (data.message) message = data.message;

                if (phase === 'complete') {
                    isRunning = false;
                    currentSpeed = download;
                    if (data.result && data.result.timestamp) {
                        lastTestedTime = data.result.timestamp;
                    } else {
                        const now = new Date();
                        lastTestedTime = now.toLocaleDateString() + ' ' + now.toLocaleTimeString();
                    }
                    if (eventSource) {
                        eventSource.close();
                        eventSource = null;
                    }
                } else if (phase === 'stopped' || phase === 'error') {
                    isRunning = false;
                    if (eventSource) {
                        eventSource.close();
                        eventSource = null;
                    }
                }
            } catch (e) {
                console.error('Error parsing SSE data:', e);
            }
        };

        eventSource.onerror = (err) => {
            console.warn('SSE disconnected or completed:', err);
            if (isRunning) {
                isRunning = false;
                if (phase !== 'complete') {
                    phase = 'error';
                    message = 'Koneksi terputus ke server pengujian.';
                }
            }
            if (eventSource) {
                eventSource.close();
                eventSource = null;
            }
        };
    }

    async function stopSpeedtest() {
        if (!isRunning) return;
        try {
            await fetch('/api/speedtest/stop', { method: 'POST' });
        } catch (e) {
            console.error('Error stopping speedtest:', e);
        }
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        isRunning = false;
        phase = 'stopped';
        message = 'Pengujian dihentikan oleh pengguna.';
    }

    onMount(() => {
        loadLastResult();
    });

    onDestroy(() => {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
    });

    // Helper teks fase
    function getPhaseTitle(p) {
        switch (p) {
            case 'init': return 'MENYIAPKAN PENGUJIAN...';
            case 'ping': return 'MENGUKUR LATENSI & JITTER...';
            case 'download': return 'MENGUJI KECEPATAN UNDUH...';
            case 'upload': return 'MENGUJI KECEPATAN UNGGAH...';
            case 'complete': return 'PENGUJIAN SELESAI';
            case 'stopped': return 'PENGUJIAN DIBATALKAN';
            case 'error': return 'TERJADI KESALAHAN';
            default: return 'SIAP UNTUK PENGUJIAN';
        }
    }

    // Rekomendasi performa
    $: gamingStatus = ping > 0 ? (ping <= 45 ? 'Sangat Bagus' : ping <= 90 ? 'Cukup Baik' : 'Tinggi') : '-';
    $: streamingStatus = download > 0 ? (download >= 25 ? '4K Ultra HD' : download >= 10 ? 'Full HD 1080p' : 'HD 720p') : '-';
</script>

<div class="wrap">
    <!-- HEADER -->
    <div class="header">
        <div>
            <div class="eyebrow">
                <span class="live-dot" class:active-spin={isRunning}></span>
                <span>Alat Jaringan & Performa</span>
            </div>
            <h1>Speedtest Internet</h1>
            <div class="desc">
                Uji kecepatan unduh, unggah, dan kestabilan latensi koneksi internet / proxy HP secara real-time via Cloudflare Edge.
            </div>
        </div>
        {#if lastTestedTime}
            <div class="last-tested">
                <i class="fa-solid fa-clock-rotate-left"></i> Terakhir diuji: {lastTestedTime}
            </div>
        {/if}
    </div>

    <!-- CONNECTION INFO STRIP -->
    <div class="card meta-card">
        <div class="meta-item">
            <i class="fa-solid fa-network-wired" style="color:var(--cyan);"></i>
            <div class="meta-content">
                <div class="meta-label">IP Publik</div>
                <div class="meta-val">{ip}</div>
            </div>
        </div>
        <div class="meta-sep"></div>
        <div class="meta-item">
            <i class="fa-solid fa-tower-broadcast" style="color:var(--lime);"></i>
            <div class="meta-content">
                <div class="meta-label">Penyedia / ISP</div>
                <div class="meta-val">{isp}</div>
            </div>
        </div>
        <div class="meta-sep"></div>
        <div class="meta-item">
            <i class="fa-solid fa-server" style="color:var(--yellow);"></i>
            <div class="meta-content">
                <div class="meta-label">Server & Lokasi</div>
                <div class="meta-val">
                    {#if colo !== '-'}<strong>{colo}</strong>{/if}
                    {#if city !== '-' && city !== ''} ({city}, {country}){/if}
                    {#if colo === '-' && city === '-'}Mendeteksi...{/if}
                </div>
            </div>
        </div>
    </div>

    <!-- MAIN SPEEDOMETER HERO CARD -->
    <div class="card hero-card">
        <div class="hero-top">
            <span class="phase-badge phase-{phase}">
                <i class="fa-solid fa-circle-notch" class:fa-spin={isRunning}></i>
                {getPhaseTitle(phase)}
            </span>
            {#if isRunning}
                <span class="progress-num">{progress}%</span>
            {/if}
        </div>

        <!-- PROGRESS BAR -->
        <div class="progress-track">
            <div
                class="progress-fill phase-fill-{phase}"
                style="width: {progress}%;"
                class:animated-stripes={isRunning}
            ></div>
        </div>

        <!-- BIG LIVE SPEED NUMBER -->
        <div class="speed-display">
            {#if phase === 'download'}
                <div class="speed-number">{currentSpeed.toFixed(1)}</div>
                <div class="speed-unit">Mbps <span>(Unduh)</span></div>
            {:else if phase === 'upload'}
                <div class="speed-number">{currentSpeed.toFixed(1)}</div>
                <div class="speed-unit">Mbps <span>(Unggah)</span></div>
            {:else if phase === 'ping'}
                <div class="speed-number">{ping.toFixed(0)}</div>
                <div class="speed-unit">ms <span>(Latensi)</span></div>
            {:else if phase === 'complete'}
                <div class="speed-number">{download.toFixed(1)}</div>
                <div class="speed-unit">Mbps <span>(Unduh Maksimal)</span></div>
            {:else}
                <div class="speed-number">{download > 0 ? download.toFixed(1) : '0.0'}</div>
                <div class="speed-unit">Mbps</div>
            {/if}
        </div>

        {#if message}
            <div class="status-msg">
                <i class="fa-solid fa-circle-info"></i> {message}
            </div>
        {/if}

        <!-- ACTION BUTTON -->
        <div class="action-wrap">
            {#if !isRunning}
                <button class="btn-main btn-start" on:click={startSpeedtest}>
                    <i class="fa-solid fa-bolt-lightning"></i>
                    <span>{phase === 'complete' ? 'Uji Ulang Kecepatan' : 'Mulai Uji Kecepatan'}</span>
                </button>
            {:else}
                <button class="btn-main btn-stop" on:click={stopSpeedtest}>
                    <i class="fa-solid fa-hand"></i>
                    <span>Hentikan Pengujian</span>
                </button>
            {/if}
        </div>
    </div>

    <!-- 4 METRIC CARDS -->
    <div class="metrics-grid">
        <!-- PING -->
        <div class="card metric-card" class:metric-active={phase === 'ping'}>
            <div class="metric-head">
                <i class="fa-solid fa-bolt" style="color:var(--lime);"></i>
                <span>Latensi (Ping)</span>
            </div>
            <div class="metric-body">
                <div class="metric-val">{ping > 0 ? ping.toFixed(1) : '-'}</div>
                <div class="metric-unit">ms</div>
            </div>
            <div class="metric-footer">
                Koneksi awal ke server
            </div>
        </div>

        <!-- JITTER -->
        <div class="card metric-card" class:metric-active={phase === 'ping'}>
            <div class="metric-head">
                <i class="fa-solid fa-water" style="color:var(--yellow);"></i>
                <span>Jitter</span>
            </div>
            <div class="metric-body">
                <div class="metric-val">{jitter > 0 ? jitter.toFixed(1) : '-'}</div>
                <div class="metric-unit">ms</div>
            </div>
            <div class="metric-footer">
                Stabilitas delay paket
            </div>
        </div>

        <!-- DOWNLOAD -->
        <div class="card metric-card" class:metric-active={phase === 'download'}>
            <div class="metric-head">
                <i class="fa-solid fa-cloud-arrow-down" style="color:var(--cyan);"></i>
                <span>Kecepatan Unduh</span>
            </div>
            <div class="metric-body">
                <div class="metric-val">{download > 0 ? download.toFixed(2) : (phase === 'download' ? currentSpeed.toFixed(2) : '-')}</div>
                <div class="metric-unit">Mbps</div>
            </div>
            <div class="metric-footer">
                Download transfer rate
            </div>
        </div>

        <!-- UPLOAD -->
        <div class="card metric-card" class:metric-active={phase === 'upload'}>
            <div class="metric-head">
                <i class="fa-solid fa-cloud-arrow-up" style="color:var(--pink);"></i>
                <span>Kecepatan Unggah</span>
            </div>
            <div class="metric-body">
                <div class="metric-val">{upload > 0 ? upload.toFixed(2) : (phase === 'upload' ? currentSpeed.toFixed(2) : '-')}</div>
                <div class="metric-unit">Mbps</div>
            </div>
            <div class="metric-footer">
                Upload transfer rate
            </div>
        </div>
    </div>

    <!-- EVALUATION / INSIGHTS CARD -->
    {#if phase === 'complete'}
        <div class="card insight-card">
            <div class="insight-title">
                <i class="fa-solid fa-chart-simple" style="color:var(--lime);"></i>
                <span>Ringkasan Kemampuan Jaringan</span>
            </div>
            <div class="insight-grid">
                <div class="insight-item">
                    <div class="ii-icon"><i class="fa-solid fa-gamepad" style="color:var(--lime);"></i></div>
                    <div class="ii-text">
                        <div class="ii-name">Game Online (ML, PUBG, FF)</div>
                        <div class="ii-desc">Ping <strong>{ping.toFixed(0)} ms</strong> &bull; Status: <span class="badge-rec">{gamingStatus}</span></div>
                    </div>
                </div>
                <div class="insight-item">
                    <div class="ii-icon"><i class="fa-solid fa-video" style="color:var(--cyan);"></i></div>
                    <div class="ii-text">
                        <div class="ii-name">Streaming Video & Film</div>
                        <div class="ii-desc">Kapasitas: <span class="badge-rec">{streamingStatus}</span></div>
                    </div>
                </div>
                <div class="insight-item">
                    <div class="ii-icon"><i class="fa-solid fa-phone-volume" style="color:var(--yellow);"></i></div>
                    <div class="ii-text">
                        <div class="ii-name">Video Call & Meeting</div>
                        <div class="ii-desc">Jitter <strong>{jitter.toFixed(1)} ms</strong> &bull; Bebas putus-putus</div>
                    </div>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .wrap {
        max-width: 860px;
        margin: 0 auto;
        padding: 16px;
        padding-bottom: 90px;
    }

    .header {
        margin-bottom: 14px;
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        flex-wrap: wrap;
        gap: 8px;
    }

    .eyebrow {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--lime);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 4px;
    }

    .live-dot {
        width: 8px;
        height: 8px;
        background: var(--lime);
        border-radius: 50%;
        box-shadow: 0 0 6px var(--lime);
    }

    .active-spin {
        animation: pulse 1s infinite;
    }

    @keyframes pulse {
        0%, 100% { transform: scale(1); opacity: 1; }
        50% { transform: scale(1.4); opacity: 0.5; }
    }

    h1 {
        font-size: 26px;
        font-weight: 950;
        letter-spacing: -1.2px;
        text-transform: uppercase;
        line-height: 1;
        margin: 0;
    }

    .desc {
        font-size: 12px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 6px;
        max-width: 600px;
    }

    .last-tested {
        font-size: 11px;
        font-weight: 800;
        color: var(--muted);
        background: var(--panel-alt);
        padding: 6px 12px;
        border: 2px solid var(--line);
        border-radius: 8px;
        box-shadow: 2px 2px 0 var(--line);
    }

    /* CARD BASE */
    .card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 14px;
        padding: 16px;
        box-shadow: var(--shadow);
        margin-bottom: 14px;
    }

    /* META CARD */
    .meta-card {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 12px 18px;
        gap: 12px;
        flex-wrap: wrap;
        background: var(--panel-alt);
    }

    .meta-item {
        display: flex;
        align-items: center;
        gap: 10px;
        flex: 1;
        min-width: 170px;
    }

    .meta-item i {
        font-size: 18px;
    }

    .meta-content {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .meta-label {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
    }

    .meta-val {
        font-size: 12.5px;
        font-weight: 800;
        color: var(--ink);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .meta-sep {
        width: 2px;
        height: 32px;
        background: var(--line);
    }

    @media (max-width: 600px) {
        .meta-sep { display: none; }
    }

    /* HERO CARD */
    .hero-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 24px 20px;
        text-align: center;
        background: var(--paper);
        position: relative;
        overflow: hidden;
    }

    .hero-top {
        width: 100%;
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 14px;
    }

    .phase-badge {
        font-size: 11px;
        font-weight: 900;
        letter-spacing: 0.5px;
        text-transform: uppercase;
        padding: 5px 12px;
        border-radius: 6px;
        border: 2px solid var(--line);
        display: inline-flex;
        align-items: center;
        gap: 6px;
        box-shadow: 2px 2px 0 var(--line);
    }

    .phase-idle, .phase-ready { background: var(--panel-alt); color: var(--ink); }
    .phase-init { background: var(--yellow); color: #000; }
    .phase-ping { background: var(--lime); color: #000; }
    .phase-download { background: var(--cyan); color: #000; }
    .phase-upload { background: var(--pink); color: #fff; }
    .phase-complete { background: var(--lime); color: #000; }
    .phase-stopped, .phase-error { background: var(--pink); color: #fff; }

    .progress-num {
        font-size: 14px;
        font-weight: 950;
        color: var(--ink);
    }

    /* PROGRESS BAR */
    .progress-track {
        width: 100%;
        height: 12px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 10px;
        overflow: hidden;
        margin-bottom: 20px;
        box-shadow: inset 1px 1px 2px rgba(0,0,0,0.2);
    }

    .progress-fill {
        height: 100%;
        transition: width 0.15s ease-out;
    }

    .phase-fill-ping { background: var(--lime); }
    .phase-fill-download { background: var(--cyan); }
    .phase-fill-upload { background: var(--pink); }
    .phase-fill-complete { background: var(--lime); }
    .phase-fill-idle, .phase-fill-ready, .phase-fill-init { background: var(--yellow); }

    .animated-stripes {
        background-image: linear-gradient(
            45deg,
            rgba(255, 255, 255, 0.25) 25%,
            transparent 25%,
            transparent 50%,
            rgba(255, 255, 255, 0.25) 50%,
            rgba(255, 255, 255, 0.25) 75%,
            transparent 75%,
            transparent
        );
        background-size: 24px 24px;
        animation: moveStripes 1s linear infinite;
    }

    @keyframes moveStripes {
        from { background-position: 0 0; }
        to { background-position: 24px 0; }
    }

    /* BIG SPEED DISPLAY */
    .speed-display {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        margin: 10px 0 16px 0;
    }

    .speed-number {
        font-size: 72px;
        font-weight: 950;
        line-height: 1;
        letter-spacing: -2px;
        color: var(--ink);
        font-variant-numeric: tabular-nums;
        font-family: var(--font, system-ui, sans-serif);
    }

    .speed-unit {
        font-size: 16px;
        font-weight: 900;
        color: var(--muted);
        margin-top: 4px;
    }

    .speed-unit span {
        font-size: 13px;
        font-weight: 800;
        color: var(--primary);
    }

    .status-msg {
        font-size: 12px;
        font-weight: 700;
        color: var(--muted);
        margin-bottom: 20px;
        display: flex;
        align-items: center;
        gap: 6px;
    }

    /* BUTTONS */
    .action-wrap {
        width: 100%;
        max-width: 380px;
    }

    .btn-main {
        width: 100%;
        padding: 14px 24px;
        font-size: 15px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        border: 2px solid var(--line);
        border-radius: 12px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        box-shadow: 4px 4px 0 var(--line);
        transition: transform 0.1s, box-shadow 0.1s;
    }

    .btn-main:hover {
        transform: translate(-2px, -2px);
        box-shadow: 6px 6px 0 var(--line);
    }

    .btn-main:active {
        transform: translate(2px, 2px);
        box-shadow: 2px 2px 0 var(--line);
    }

    .btn-start {
        background: var(--yellow);
        color: #000;
    }

    .btn-stop {
        background: var(--pink);
        color: #fff;
    }

    /* METRICS GRID */
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
        gap: 12px;
        margin-bottom: 14px;
    }

    .metric-card {
        padding: 14px;
        display: flex;
        flex-direction: column;
        gap: 6px;
        transition: transform 0.15s, border-color 0.15s;
    }

    .metric-active {
        border-color: var(--primary);
        box-shadow: 3px 3px 0 var(--primary);
        transform: translate(-1px, -1px);
        background: rgba(212, 255, 0, 0.05);
    }

    .metric-head {
        font-size: 11px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .metric-body {
        display: flex;
        align-items: baseline;
        gap: 4px;
    }

    .metric-val {
        font-size: 28px;
        font-weight: 950;
        color: var(--ink);
        font-variant-numeric: tabular-nums;
        line-height: 1.1;
    }

    .metric-unit {
        font-size: 12px;
        font-weight: 800;
        color: var(--muted);
    }

    .metric-footer {
        font-size: 10px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 2px;
    }

    /* INSIGHT CARD */
    .insight-card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 14px;
        padding: 16px;
    }

    .insight-title {
        font-size: 13px;
        font-weight: 900;
        text-transform: uppercase;
        margin-bottom: 12px;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .insight-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 12px;
    }

    .insight-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 12px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 10px;
    }

    .ii-icon {
        width: 36px;
        height: 36px;
        border-radius: 8px;
        background: var(--paper);
        border: 2px solid var(--line);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 16px;
        flex-shrink: 0;
    }

    .ii-name {
        font-size: 12px;
        font-weight: 900;
        color: var(--ink);
    }

    .ii-desc {
        font-size: 11px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 2px;
    }

    .badge-rec {
        background: var(--lime);
        color: #000;
        font-weight: 900;
        padding: 1px 6px;
        border-radius: 4px;
        font-size: 10px;
    }
</style>

