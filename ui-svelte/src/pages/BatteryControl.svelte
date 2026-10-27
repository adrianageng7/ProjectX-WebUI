<script>
    import { onMount, onDestroy } from 'svelte';

    let loading = true;
    let saving = false;
    let data = {
        level: 0,
        status: '—',
        display_status: '—',
        temp_c: 0,
        health: 'Good',
        voltage_mv: 0,
        is_cutoff: false,
        control_method: '—',
        config: {
            enabled: false,
            start_level: 60,
            stop_level: 80
        }
    };

    let form = {
        enabled: false,
        start_level: 60,
        stop_level: 80
    };

    let message = '';
    let error = '';
    let timer = null;

    async function fetchStatus(isInitial = false) {
        if (isInitial) loading = true;
        try {
            const res = await fetch('/api/battery/status?_=' + Date.now());
            if (res.ok) {
                const json = await res.json();
                data = json;
                if (isInitial && json.config) {
                    form = {
                        enabled: !!json.config.enabled,
                        start_level: json.config.start_level || 60,
                        stop_level: json.config.stop_level || 80
                    };
                }
            }
        } catch (e) {
            console.warn('Failed to fetch battery status:', e);
        } finally {
            if (isInitial) loading = false;
        }
    }

    function setPreset(start, stop) {
        if (!form.enabled) return;
        form.start_level = start;
        form.stop_level = stop;
    }

    async function saveConfig() {
        message = '';
        error = '';

        if (form.start_level >= form.stop_level) {
            error = 'Batas Berhenti (Cut-off) harus lebih besar dari Batas Mulai!';
            return;
        }

        if (form.stop_level - form.start_level < 5) {
            error = 'Selisih Batas Berhenti dan Batas Mulai minimal 5% (mencegah siklus cut-off berulang cepat)!';
            return;
        }

        saving = true;
        try {
            const res = await fetch('/api/battery/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    enabled: form.enabled,
                    start_level: Number(form.start_level),
                    stop_level: Number(form.stop_level)
                })
            });

            const json = await res.json();
            if (res.ok) {
                data = json;
                message = json.message || 'Pengaturan proteksi baterai berhasil diterapkan!';
                setTimeout(() => { message = ''; }, 4000);
            } else {
                error = json.message || 'Gagal menyimpan pengaturan proteksi baterai';
            }
        } catch (e) {
            error = 'Gagal menghubungi server: ' + e.message;
        } finally {
            saving = false;
        }
    }

    function getBatteryIcon(level, isCharging, isCutoff) {
        if (isCutoff) return 'fa-shield-halved';
        if (isCharging) return 'fa-bolt';
        if (level >= 80) return 'fa-battery-full';
        if (level >= 60) return 'fa-battery-three-quarters';
        if (level >= 40) return 'fa-battery-half';
        if (level >= 15) return 'fa-battery-quarter';
        return 'fa-battery-empty';
    }

    onMount(() => {
        fetchStatus(true);
        timer = setInterval(() => fetchStatus(false), 3000);
    });

    onDestroy(() => {
        if (timer) clearInterval(timer);
    });
</script>

<div class="wrap">
    <!-- HEADER -->
    <header class="header">
        <div class="brand">
            <div class="eyebrow"><span class="live-dot"></span> System Power & Protection</div>
            <h1>Proteksi Baterai</h1>
            <div class="desc">Auto Cut-off & Bypass Charging untuk pemakaian modem/router 24/7</div>
        </div>
        <div class="controls">
            <button class="btn" on:click={() => fetchStatus(false)} title="Segarkan Data">
                <i class="fa-solid fa-rotate-right"></i>
            </button>
        </div>
    </header>

    {#if message}
        <div class="alert alert-success">
            <i class="fa-solid fa-circle-check"></i>
            <span>{message}</span>
        </div>
    {/if}

    {#if error}
        <div class="alert alert-danger">
            <i class="fa-solid fa-triangle-exclamation"></i>
            <span>{error}</span>
        </div>
    {/if}

    <!-- LIVE HERO STATUS CARD -->
    <div class="card hero-card {data.is_cutoff ? 'hero-cutoff' : ''}">
        <div class="hero-top">
            <div class="hero-info">
                <div class="hero-badge">
                    {#if data.is_cutoff}
                        <span class="badge badge-cutoff">
                            <i class="fa-solid fa-shield-halved"></i> BYPASS CHARGING AKTIF
                        </span>
                    {:else if data.status === 'Charging'}
                        <span class="badge badge-charging">
                            <i class="fa-solid fa-bolt"></i> SEDANG MENGISI DAYA
                        </span>
                    {:else}
                        <span class="badge badge-idle">
                            <i class="fa-solid fa-battery-half"></i> PENGISIAN TERPUTUS / DISCHARGING
                        </span>
                    {/if}
                </div>
                <div class="hero-level">
                    <span class="level-num">{data.level}</span><span class="level-unit">%</span>
                </div>
                <div class="hero-status-text">
                    {#if data.is_cutoff}
                        Pengecasan dihentikan secara hardware. HP beroperasi dengan suplai daya USB.
                    {:else if data.status === 'Charging'}
                        Arus daya aktif mengalir mengisi sel baterai.
                    {:else}
                        Baterai tidak sedang diisi atau charger tidak terhubung.
                    {/if}
                </div>
            </div>
            <div class="hero-icon-box {data.is_cutoff ? 'icon-box-cutoff' : ''}">
                <i class="fa-solid {getBatteryIcon(data.level, data.status === 'Charging', data.is_cutoff)}"></i>
            </div>
        </div>

        <!-- PROGRESS BAR -->
        <div class="hero-progress">
            <div 
                class="hero-progress-fill {data.is_cutoff ? 'fill-cutoff' : (data.status === 'Charging' ? 'fill-charging' : 'fill-idle')}"
                style="width: {Math.max(5, Math.min(100, data.level))}%"
            ></div>
            {#if form.enabled}
                <!-- Threshold Markers -->
                <div class="marker marker-start" style="left: {form.start_level}%" title="Mulai Cas: {form.start_level}%">
                    <span class="marker-tag">ON {form.start_level}%</span>
                </div>
                <div class="marker marker-stop" style="left: {form.stop_level}%" title="Stop Cas: {form.stop_level}%">
                    <span class="marker-tag">OFF {form.stop_level}%</span>
                </div>
            {/if}
        </div>

        <!-- METRICS GRID -->
        <div class="stats-grid">
            <div class="stat-cell">
                <span class="stat-cell-label">Status Realtime</span>
                <span class="stat-cell-val {data.is_cutoff ? 'text-cyan' : (data.status === 'Charging' ? 'text-lime' : '')}">
                    {data.display_status || data.status || 'Unknown'}
                </span>
            </div>
            <div class="stat-cell">
                <span class="stat-cell-label">Suhu Baterai</span>
                <span class="stat-cell-val">
                    <i class="fa-solid fa-temperature-half" style="color:var(--pink);"></i> {data.temp_c}°C
                </span>
            </div>
            <div class="stat-cell">
                <span class="stat-cell-label">Tegangan</span>
                <span class="stat-cell-val">
                    {data.voltage_mv > 0 ? (data.voltage_mv / 1000).toFixed(2) + ' V' : '—'}
                </span>
            </div>
            <div class="stat-cell">
                <span class="stat-cell-label">Kesehatan Baterai</span>
                <span class="stat-cell-val text-lime">
                    <i class="fa-solid fa-heart-pulse"></i> {data.health || 'Good'}
                </span>
            </div>
            <div class="stat-cell stat-cell-wide">
                <span class="stat-cell-label">Hardware Controller</span>
                <span class="stat-cell-val font-mono">{data.control_method || 'Mendeteksi...'}</span>
            </div>
        </div>
    </div>

    <!-- CONFIGURATION CARD -->
    <div class="card config-card">
        <div class="card-header">
            <i class="fa-solid fa-sliders" style="color:var(--yellow);"></i>
            <span>Pengaturan Auto Cut-off</span>
        </div>

        <!-- Toggle Switch Row -->
        <div class="toggle-row">
            <div>
                <div class="toggle-title">Aktifkan Proteksi Otomatis</div>
                <div class="toggle-desc">
                    Otomatis putus arus cas saat mencapai batas atas, dan sambung kembali saat daya turun ke batas bawah.
                </div>
            </div>
            <label class="switch">
                <input type="checkbox" bind:checked={form.enabled}>
                <span class="slider"></span>
            </label>
        </div>

        <!-- Sliders -->
        <div class="sliders-wrap" class:disabled-sliders={!form.enabled}>
            <div class="slider-box">
                <div class="slider-header-row">
                    <span class="slider-name">
                        <i class="fa-solid fa-arrow-down" style="color:var(--lime);"></i> Mulai Mengisi (Start Level)
                    </span>
                    <span class="slider-val-badge badge-lime">{form.start_level}%</span>
                </div>
                <input 
                    type="range" 
                    min="20" 
                    max="85" 
                    step="5" 
                    bind:value={form.start_level} 
                    disabled={!form.enabled} 
                />
                <div class="slider-desc">
                    Pengecasan otomatis dinyalakan kembali saat baterai berada di level <strong>{form.start_level}%</strong> atau lebih rendah.
                </div>
            </div>

            <div class="slider-box">
                <div class="slider-header-row">
                    <span class="slider-name">
                        <i class="fa-solid fa-arrow-up" style="color:var(--pink);"></i> Berhenti Mengisi (Cut-off Stop Level)
                    </span>
                    <span class="slider-val-badge badge-pink">{form.stop_level}%</span>
                </div>
                <input 
                    type="range" 
                    min="30" 
                    max="95" 
                    step="5" 
                    bind:value={form.stop_level} 
                    disabled={!form.enabled} 
                />
                <div class="slider-desc">
                    Pengecasan dihentikan saat mencapai <strong>{form.stop_level}%</strong>. Perangkat tetap menyala menggunakan daya bypass USB.
                </div>
            </div>

            <!-- Quick Presets -->
            <div class="presets-section">
                <div class="presets-title">Rekomendasi Preset:</div>
                <div class="presets-buttons">
                    <button 
                        type="button" 
                        class="btn-preset {form.start_level === 50 && form.stop_level === 70 ? 'active' : ''}" 
                        on:click={() => setPreset(50, 70)} 
                        disabled={!form.enabled}
                    >
                        Awet 50% - 70%
                    </button>
                    <button 
                        type="button" 
                        class="btn-preset {form.start_level === 60 && form.stop_level === 80 ? 'active' : ''}" 
                        on:click={() => setPreset(60, 80)} 
                        disabled={!form.enabled}
                    >
                        Standar 60% - 80% ★
                    </button>
                    <button 
                        type="button" 
                        class="btn-preset {form.start_level === 70 && form.stop_level === 90 ? 'active' : ''}" 
                        on:click={() => setPreset(70, 90)} 
                        disabled={!form.enabled}
                    >
                        Kapasitas 70% - 90%
                    </button>
                </div>
            </div>
        </div>

        <!-- Save Button -->
        <button class="btn-save" on:click={saveConfig} disabled={saving}>
            {#if saving}
                <i class="fa-solid fa-spinner fa-spin"></i> Menyimpan & Menerapkan...
            {:else}
                <i class="fa-solid fa-floppy-disk"></i> Terapkan Pengaturan Sekarang
            {/if}
        </button>
    </div>

    <!-- EXPLANATION / FAQ CARD -->
    <div class="card info-card">
        <div class="card-header">
            <i class="fa-solid fa-circle-question" style="color:var(--cyan);"></i>
            <span>Panduan & Informasi Proteksi Baterai</span>
        </div>

        <div class="info-content">
            <div class="info-item">
                <div class="info-q"><i class="fa-solid fa-battery-full text-lime"></i> Kenapa Baterai Butuh Proteksi 24/7?</div>
                <div class="info-a">
                    Ponsel yang difungsikan sebagai router / modem seringkali dicolok charger terus-menerus selama berhari-hari. 
                    Jika baterai dipertahankan 100% pada suhu hangat secara terus menerus, tekanan elektrolit tinggi akan menyebabkan 
                    <strong>baterai kembung (bloating)</strong> dan berisiko merusak layar serta komponen ponsel.
                </div>
            </div>

            <div class="info-item">
                <div class="info-q"><i class="fa-solid fa-microchip text-cyan"></i> Bagaimana Bypass Charging Bekerja?</div>
                <div class="info-a">
                    ProjectX mengontrol langsung register IC Charger hardware (Qualcomm SMB358 / sysfs) via I2C. Saat cut-off aktif, 
                    aliran pengisian ke sel baterai dimatikan, namun daya USB tetap diteruskan untuk menghidupkan sistem ponsel (Bypass mode).
                </div>
            </div>

            <div class="info-item">
                <div class="info-q"><i class="fa-solid fa-arrow-right-arrow-left text-pink"></i> Fitur Hysteresis (Jeda Rentang)</div>
                <div class="info-a">
                    Sistem memberikan jeda minimal 5% antara batas mulai dan batas berhenti. Contohnya pada preset 60%-80%, saat baterai 
                    mencapai 80% pengecasan berhenti. Pengisian tidak akan dinyalakan lagi sampai baterai perlahan turun ke 60%, 
                    sehingga terhindar dari siklus on-off yang terlalu sering.
                </div>
            </div>
        </div>
    </div>
</div>

<style>
    .wrap {
        max-width: 760px;
        margin: 0 auto;
        padding: 16px;
        padding-bottom: 100px;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        margin-bottom: 20px;
    }

    .brand {
        display: flex;
        flex-direction: column;
    }

    .eyebrow {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--cyan);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 4px;
        letter-spacing: 0.5px;
    }

    .live-dot {
        width: 8px;
        height: 8px;
        background: var(--lime);
        border-radius: 50%;
        animation: pulse 1.8s infinite;
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.35; transform: scale(0.85); }
    }

    h1 {
        font-size: 28px;
        font-weight: 950;
        letter-spacing: -1.5px;
        text-transform: uppercase;
        line-height: 1;
        margin: 0;
        color: var(--ink);
    }

    .desc {
        font-size: 12px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 6px;
    }

    .btn {
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 8px 12px;
        font-size: 13px;
        font-weight: 900;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        transition: all 0.1s ease;
        box-shadow: 3px 3px 0px var(--line);
        text-transform: uppercase;
        background: var(--paper);
        color: var(--ink);
    }
    .btn:hover { transform: translate(-2px,-2px); box-shadow: 5px 5px 0 var(--line); }
    .btn:active { transform: translate(2px,2px); box-shadow: none; }

    /* ALERTS */
    .alert {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 12px 16px;
        border: 2px solid var(--line);
        border-radius: 12px;
        font-size: 12px;
        font-weight: 800;
        margin-bottom: 16px;
        box-shadow: 3px 3px 0 var(--line);
    }
    .alert-success { background: #dcfce7; color: #14532d; }
    .alert-danger { background: #fee2e2; color: #7f1d1d; }

    /* CARD BASE */
    .card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 16px;
        padding: 20px;
        box-shadow: var(--shadow);
        margin-bottom: 20px;
    }

    .card-header {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 14px;
        font-weight: 900;
        text-transform: uppercase;
        margin-bottom: 16px;
        padding-bottom: 10px;
        border-bottom: 2px dashed var(--line);
        color: var(--ink);
    }

    /* HERO CARD */
    .hero-card {
        background: var(--panel);
        border-color: var(--line);
        position: relative;
        overflow: hidden;
    }
    .hero-card.hero-cutoff {
        border-color: var(--cyan);
        box-shadow: 4px 4px 0 var(--cyan);
    }

    .hero-top {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 16px;
    }

    .hero-info {
        display: flex;
        flex-direction: column;
    }

    .hero-badge {
        margin-bottom: 8px;
    }

    .badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        border-radius: 999px;
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        border: 2px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
    }
    .badge-cutoff { background: var(--cyan); color: #000; }
    .badge-charging { background: var(--lime); color: #000; }
    .badge-idle { background: var(--muted); color: #fff; }

    .hero-level {
        display: flex;
        align-items: baseline;
        line-height: 1;
    }
    .level-num {
        font-size: 56px;
        font-weight: 950;
        letter-spacing: -2px;
        color: var(--ink);
    }
    .level-unit {
        font-size: 24px;
        font-weight: 900;
        color: var(--muted);
        margin-left: 4px;
    }

    .hero-status-text {
        font-size: 12px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 6px;
        max-width: 440px;
    }

    .hero-icon-box {
        width: 64px;
        height: 64px;
        border-radius: 16px;
        background: var(--paper);
        border: 2px solid var(--line);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 28px;
        box-shadow: 3px 3px 0 var(--line);
        color: var(--lime);
    }
    .hero-icon-box.icon-box-cutoff {
        background: var(--cyan);
        color: #000;
    }

    /* PROGRESS BAR */
    .hero-progress {
        position: relative;
        height: 24px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 999px;
        overflow: visible;
        margin: 18px 0 24px;
    }

    .hero-progress-fill {
        height: 100%;
        border-radius: 999px;
        transition: width 0.4s ease;
    }
    .fill-charging { background: var(--lime); }
    .fill-cutoff { background: var(--cyan); }
    .fill-idle { background: var(--yellow); }

    .marker {
        position: absolute;
        top: -6px;
        bottom: -6px;
        width: 3px;
        background: var(--line);
        z-index: 2;
    }
    .marker-start { background: var(--lime); }
    .marker-stop { background: var(--pink); }

    .marker-tag {
        position: absolute;
        bottom: -22px;
        transform: translateX(-50%);
        font-size: 9px;
        font-weight: 900;
        padding: 1px 5px;
        border-radius: 4px;
        border: 1px solid var(--line);
        white-space: nowrap;
        background: var(--paper);
        color: var(--ink);
    }

    /* STATS GRID */
    .stats-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 10px;
        margin-top: 10px;
    }
    @media (min-width: 580px) {
        .stats-grid {
            grid-template-columns: repeat(4, 1fr);
        }
    }

    .stat-cell {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 10px 12px;
        box-shadow: 2px 2px 0 var(--line);
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .stat-cell-wide {
        grid-column: 1 / -1;
    }

    .stat-cell-label {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
    }
    .stat-cell-val {
        font-size: 13px;
        font-weight: 900;
        color: var(--ink);
    }

    /* CONFIG CARD */
    .toggle-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 16px;
        padding: 14px 16px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 12px;
        margin-bottom: 20px;
        box-shadow: 2px 2px 0 var(--line);
    }

    .toggle-title {
        font-size: 14px;
        font-weight: 950;
        color: var(--ink);
    }
    .toggle-desc {
        font-size: 11px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 2px;
    }

    /* SWITCH */
    .switch {
        position: relative;
        display: inline-block;
        width: 50px;
        height: 28px;
        flex-shrink: 0;
    }
    .switch input { opacity: 0; width: 0; height: 0; }
    .slider {
        position: absolute;
        cursor: pointer;
        top: 0; left: 0; right: 0; bottom: 0;
        background: var(--muted);
        border: 2px solid var(--line);
        border-radius: 34px;
        transition: .3s;
    }
    .slider:before {
        position: absolute;
        content: "";
        height: 18px;
        width: 18px;
        left: 3px;
        bottom: 3px;
        background: white;
        border: 2px solid var(--line);
        border-radius: 50%;
        transition: .3s;
    }
    input:checked + .slider { background: var(--lime); }
    input:checked + .slider:before { transform: translateX(22px); }

    /* SLIDERS */
    .sliders-wrap {
        display: flex;
        flex-direction: column;
        gap: 16px;
        transition: opacity 0.2s;
    }
    .disabled-sliders {
        opacity: 0.45;
        pointer-events: none;
    }

    .slider-box {
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 14px 16px;
        box-shadow: 2px 2px 0 var(--line);
    }

    .slider-header-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
    }
    .slider-name {
        font-size: 12px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--ink);
    }
    .slider-val-badge {
        font-size: 12px;
        font-weight: 950;
        padding: 2px 8px;
        border: 2px solid var(--line);
        border-radius: 6px;
        box-shadow: 2px 2px 0 var(--line);
    }
    .badge-lime { background: var(--lime); color: #000; }
    .badge-pink { background: var(--pink); color: #fff; }

    input[type=range] {
        width: 100%;
        height: 8px;
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 4px;
        outline: none;
        -webkit-appearance: none;
        appearance: none;
        cursor: pointer;
    }
    input[type=range]::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 22px;
        height: 22px;
        border-radius: 50%;
        background: var(--yellow);
        border: 2px solid var(--line);
        cursor: pointer;
        box-shadow: 2px 2px 0 var(--line);
        transition: transform 0.1s;
    }
    input[type=range]::-webkit-slider-thumb:hover {
        transform: scale(1.15);
    }

    .slider-desc {
        font-size: 11px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 8px;
    }

    /* PRESETS */
    .presets-section {
        margin-top: 4px;
    }
    .presets-title {
        font-size: 11px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
        margin-bottom: 8px;
    }
    .presets-buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }
    .btn-preset {
        padding: 8px 12px;
        border: 2px solid var(--line);
        border-radius: 8px;
        background: var(--paper);
        color: var(--ink);
        font-size: 11px;
        font-weight: 900;
        cursor: pointer;
        box-shadow: 2px 2px 0 var(--line);
        transition: all 0.1s;
    }
    .btn-preset:hover { transform: translate(-1px, -1px); box-shadow: 3px 3px 0 var(--line); }
    .btn-preset:active { transform: translate(1px, 1px); box-shadow: none; }
    .btn-preset.active {
        background: var(--yellow);
        border-color: var(--line);
    }

    .btn-save {
        width: 100%;
        padding: 14px;
        background: var(--lime);
        color: #000;
        border: 2px solid var(--line);
        border-radius: 12px;
        font-size: 13px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        cursor: pointer;
        box-shadow: 4px 4px 0 var(--line);
        transition: all 0.1s;
        margin-top: 20px;
        font-family: inherit;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
    }
    .btn-save:hover { transform: translate(-2px, -2px); box-shadow: 6px 6px 0 var(--line); filter: brightness(1.05); }
    .btn-save:active { transform: translate(2px, 2px); box-shadow: none; }
    .btn-save:disabled { opacity: 0.6; cursor: not-allowed; transform: none; box-shadow: 4px 4px 0 var(--line); }

    /* INFO CARD */
    .info-content {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }
    .info-item {
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 12px 14px;
        box-shadow: 2px 2px 0 var(--line);
    }
    .info-q {
        font-size: 12px;
        font-weight: 950;
        color: var(--ink);
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 6px;
    }
    .info-a {
        font-size: 12px;
        font-weight: 600;
        color: var(--muted);
        line-height: 1.5;
    }

    .text-lime { color: var(--lime); }
    .text-cyan { color: var(--cyan); }
    .text-pink { color: var(--pink); }
    .font-mono { font-family: monospace; }
</style>

