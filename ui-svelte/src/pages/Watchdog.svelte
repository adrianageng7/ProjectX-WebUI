<script>
    import { onMount, onDestroy } from 'svelte';

    let loading = true;
    let saving = false;
    let testingAction = false;
    let checkingNow = false;

    let status = {
        enabled: false,
        running: false,
        consecutive_fails: 0,
        last_check_time: '',
        last_check_success: true,
        last_check_latency_ms: 0,
        last_action_time: '',
        last_action_message: '',
        total_checks: 0,
        total_failures: 0,
        total_recoveries: 0,
        recent_logs: [],
        config: {
            enabled: false,
            interval_seconds: 30,
            failure_threshold: 3,
            target_host: '1.1.1.1',
            check_type: 'ping',
            action: 'data_reconnect',
            cooldown_seconds: 30,
            cooldown_minutes: 1
        }
    };

    let form = {
        enabled: false,
        interval_seconds: 30,
        failure_threshold: 3,
        target_host: '1.1.1.1',
        check_type: 'ping',
        action: 'data_reconnect',
        cooldown_seconds: 30,
        cooldown_minutes: 1
    };

    let message = '';
    let error = '';
    let timer = null;

    let modal = {
        show: false,
        title: '',
        message: '',
        warning: '',
        type: 'confirm',
        confirmText: 'Eksekusi Sekarang',
        confirmIcon: 'fa-bolt',
        confirmAction: null
    };

    async function fetchStatus(isInitial = false) {
        if (isInitial) loading = true;
        try {
            const res = await fetch('/api/watchdog/status?_=' + Date.now());
            if (res.ok) {
                const json = await res.json();
                if (json.status === 'success' && json.data) {
                    status = json.data;
                    if (isInitial && json.data.config) {
                        form = { ...json.data.config };
                        if (form.cooldown_seconds === undefined && form.cooldown_minutes !== undefined) {
                            form.cooldown_seconds = form.cooldown_minutes * 60;
                        }
                    }
                }
            }
        } catch (e) {
            console.warn('Gagal memuat status watchdog:', e);
        } finally {
            if (isInitial) loading = false;
        }
    }

    async function saveConfig() {
        message = '';
        error = '';
        saving = true;

        try {
            const payload = {
                ...form,
                cooldown_minutes: Math.floor((form.cooldown_seconds || 0) / 60)
            };
            const res = await fetch('/api/watchdog/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            const json = await res.json();
            if (res.ok && json.status === 'success') {
                message = json.message || 'Pengaturan watchdog berhasil disimpan!';
                await fetchStatus(false);
                setTimeout(() => { message = ''; }, 3500);
            } else {
                error = json.message || 'Gagal menyimpan pengaturan watchdog.';
            }
        } catch (e) {
            error = 'Gagal menghubungi server: ' + e.message;
        } finally {
            saving = false;
        }
    }

    async function toggleMaster() {
        form.enabled = !form.enabled;
        await saveConfig();
    }

    async function checkNow() {
        checkingNow = true;
        try {
            const res = await fetch('/api/watchdog/check-now', { method: 'POST' });
            const json = await res.json();
            if (json.status === 'success') {
                if (json.success) {
                    message = `Pengecekan berhasil! Latensi ke ${json.target}: ${json.latency_ms} ms.`;
                } else {
                    error = `Pengecekan gagal (RTO) ke ${json.target}!`;
                }
                await fetchStatus(false);
                setTimeout(() => { message = ''; error = ''; }, 4000);
            }
        } catch (e) {
            error = 'Gagal melakukan tes koneksi: ' + e.message;
        } finally {
            checkingNow = false;
        }
    }

    function promptTestAction() {
        let actionDesc = '';
        if (form.action === 'data_reconnect') actionDesc = 'Reset Data Seluler (svc data)';
        else if (form.action === 'airplane_mode') actionDesc = 'Mode Pesawat Cycle';
        else if (form.action === 'bfr_restart') actionDesc = 'Restart Box For Root (BFR)';
        else if (form.action === 'both') actionDesc = 'Reset Data + Restart BFR';

        modal = {
            show: true,
            title: 'Konfirmasi Uji Coba Aksi',
            message: `Jalankan uji coba aksi <b>"${actionDesc}"</b> sekarang? Koneksi data internet akan di-reset sekejap.`,
            warning: 'Koneksi data seluler atau layanan core BFR akan di-restart sekejap untuk simulasi pemulihan.',
            type: 'confirm',
            confirmText: 'Eksekusi Sekarang',
            confirmIcon: 'fa-bolt',
            confirmAction: executeTestAction
        };
    }

    async function executeTestAction() {
        modal.show = false;
        testingAction = true;
        try {
            const res = await fetch('/api/watchdog/test-action?action=' + encodeURIComponent(form.action), { method: 'POST' });
            const json = await res.json();
            if (json.status === 'success') {
                message = `Uji coba berhasil: ${json.message}`;
                await fetchStatus(false);
                setTimeout(() => { message = ''; }, 4000);
            } else {
                error = json.message || 'Gagal menjalankan uji coba tindakan.';
            }
        } catch (e) {
            error = 'Gagal menjalankan aksi: ' + e.message;
        } finally {
            testingAction = false;
        }
    }

    onMount(() => {
        fetchStatus(true);
        timer = setInterval(() => fetchStatus(false), 4000);
    });

    onDestroy(() => {
        if (timer) clearInterval(timer);
    });
</script>

<div class="wrap">
    <!-- HEADER -->
    <div class="header">
        <div class="header-left">
            <div class="eyebrow">
                <span class="live-dot {status.enabled ? 'is-active' : ''}"></span>
                Anti RTO & Auto Reconnect
            </div>
            <h1>Connection Watchdog</h1>
            <div class="desc">Otomatis refresh data seluler atau proxy BFR saat internet macet / RTO</div>
        </div>
        <button class="btn-refresh" on:click={() => fetchStatus(true)} title="Pindai Ulang">
            <i class="fa-solid fa-rotate-right {loading ? 'fa-spin' : ''}"></i>
        </button>
    </div>

    {#if message}
        <div class="alert alert-success">
            <i class="fa-solid fa-circle-check"></i>
            <span>{message}</span>
        </div>
    {/if}

    {#if error}
        <div class="alert alert-error">
            <i class="fa-solid fa-triangle-exclamation"></i>
            <span>{error}</span>
        </div>
    {/if}

    <!-- CARD 1: MASTER TOGGLE & STATUS -->
    <div class="card hero-card {form.enabled ? 'is-enabled' : ''}">
        <div class="hero-top">
            <div class="hero-info">
                <div class="status-badge {form.enabled ? 'badge-on' : 'badge-off'}">
                    <i class="fa-solid {form.enabled ? 'fa-circle-check' : 'fa-circle-pause'}"></i>
                    <span>{form.enabled ? 'WATCHDOG AKTIF' : 'NONAKTIF (OFF)'}</span>
                </div>
                <div class="hero-sub">
                    {#if form.enabled}
                        Memantau tiap {form.interval_seconds}s ke {form.target_host} ({form.failure_threshold}x RTO)
                    {:else}
                        Watchdog mati secara default. Klik tombol di kanan untuk menyalakan.
                    {/if}
                </div>
            </div>
            <button
                type="button"
                class="toggle-switch {form.enabled ? 'checked' : ''}"
                on:click={toggleMaster}
                aria-label="Toggle Watchdog"
                disabled={saving}
            >
                <span class="toggle-knob"></span>
            </button>
        </div>

        <div class="kpi-strip">
            <div class="kpi-item">
                <span class="kpi-label">Latensi Target</span>
                <span class="kpi-val {status.last_check_success ? 'color-lime' : 'color-pink'}">
                    {#if status.last_check_latency_ms > 0}
                        {status.last_check_latency_ms} ms
                    {:else}
                        —
                    {/if}
                </span>
            </div>
            <div class="kpi-item">
                <span class="kpi-label">Kondisi Saat Ini</span>
                <span class="kpi-val {status.consecutive_fails > 0 ? 'color-pink' : 'color-cyan'}">
                    {#if status.consecutive_fails > 0}
                        RTO ({status.consecutive_fails}x)
                    {:else}
                        Normal
                    {/if}
                </span>
            </div>
            <div class="kpi-item">
                <span class="kpi-label">Total Pulih</span>
                <span class="kpi-val color-yellow">{status.total_recoveries || 0}x</span>
            </div>
        </div>
    </div>

    <!-- CARD 2: PENGATURAN RINGKAS -->
    <div class="card form-card">
        <div class="card-header">
            <i class="fa-solid fa-sliders" style="color:var(--cyan);"></i>
            <span>Pengaturan Watchdog</span>
        </div>

        <div class="form-body">
            <!-- HOST TARGET -->
            <div class="form-row">
                <label class="form-label">Target Pemeriksaan (Host / IP)</label>
                <div class="target-input-wrap">
                    <input
                        type="text"
                        class="form-input"
                        bind:value={form.target_host}
                        placeholder="1.1.1.1"
                    />
                    <div class="quick-chips">
                        <button
                            type="button"
                            class="chip-btn {form.target_host === '1.1.1.1' ? 'chip-active' : ''}"
                            on:click={() => form.target_host = '1.1.1.1'}
                        >
                            1.1.1.1 (Cloudflare)
                        </button>
                        <button
                            type="button"
                            class="chip-btn {form.target_host === '8.8.8.8' ? 'chip-active' : ''}"
                            on:click={() => form.target_host = '8.8.8.8'}
                        >
                            8.8.8.8 (Google)
                        </button>
                    </div>
                </div>
            </div>

            <!-- GRID SELECTS -->
            <div class="form-grid">
                <div class="form-group">
                    <label class="form-label">Frekuensi Cek</label>
                    <select class="form-select" bind:value={form.interval_seconds}>
                        <option value={15}>Setiap 15 Detik</option>
                        <option value={30}>Setiap 30 Detik (Standar)</option>
                        <option value={60}>Setiap 1 Menit</option>
                        <option value={120}>Setiap 2 Menit</option>
                    </select>
                </div>

                <div class="form-group">
                    <label class="form-label">Toleransi RTO (Ambang)</label>
                    <select class="form-select" bind:value={form.failure_threshold}>
                        <option value={1}>1x Gagal (Agresif / Instan)</option>
                        <option value={2}>2x Gagal Berturut</option>
                        <option value={3}>3x Gagal (Rekomendasi)</option>
                        <option value={5}>5x Gagal</option>
                        <option value={8}>8x Gagal</option>
                    </select>
                </div>

                <div class="form-group">
                    <label class="form-label">Tindakan Pemulihan</label>
                    <select class="form-select" bind:value={form.action}>
                        <option value="data_reconnect">Reset Data Seluler (svc data) [Rekomendasi]</option>
                        <option value="airplane_mode">Mode Pesawat Cycle (Airplane)</option>
                        <option value="bfr_restart">Restart Box For Root (BFR)</option>
                        <option value="both">Reset Data + Restart BFR</option>
                    </select>
                </div>

                <div class="form-group">
                    <label class="form-label">Jeda Cooldown</label>
                    <select class="form-select" bind:value={form.cooldown_seconds}>
                        <option value={0}>Tanpa Jeda (0 Detik / Instan)</option>
                        <option value={30}>30 Detik (Rekomendasi Cepat)</option>
                        <option value={60}>1 Menit</option>
                        <option value={120}>2 Menit (Standar)</option>
                        <option value={300}>5 Menit</option>
                        <option value={600}>10 Menit</option>
                    </select>
                </div>
            </div>

            <button
                type="button"
                class="btn-save {saving ? 'is-loading' : ''}"
                on:click={saveConfig}
                disabled={saving}
            >
                <i class="fa-solid {saving ? 'fa-spinner fa-spin' : 'fa-floppy-disk'}"></i>
                <span>{saving ? 'Menyimpan...' : 'Simpan Pengaturan'}</span>
            </button>
        </div>
    </div>

    <!-- CARD 3: UJI COBA MANUAL -->
    <div class="card action-card">
        <div class="card-header">
            <i class="fa-solid fa-play" style="color:var(--yellow);"></i>
            <span>Uji Coba Manual</span>
        </div>
        <div class="action-buttons-row">
            <button
                type="button"
                class="btn-action-test"
                on:click={checkNow}
                disabled={checkingNow}
            >
                <i class="fa-solid fa-satellite-dish {checkingNow ? 'fa-spin' : ''}"></i>
                <span>{checkingNow ? 'Mengecek...' : 'Tes Ping Sekarang'}</span>
            </button>

            <button
                type="button"
                class="btn-action-test btn-action-recovery"
                on:click={promptTestAction}
                disabled={testingAction}
            >
                <i class="fa-solid fa-bolt {testingAction ? 'fa-spin' : ''}"></i>
                <span>{testingAction ? 'Mengeksekusi...' : 'Uji Coba Aksi'}</span>
            </button>
        </div>
    </div>

    <!-- CARD 4: LOG AKTIVITAS -->
    <div class="card log-card">
        <div class="card-header">
            <i class="fa-solid fa-clock-rotate-left" style="color:var(--purple);"></i>
            <span>Riwayat Aktivitas Watchdog</span>
        </div>
        <div class="log-box">
            {#if status.recent_logs && status.recent_logs.length > 0}
                {#each status.recent_logs as log}
                    <div class="log-item">
                        {#if log.includes('GAGAL') || log.includes('Threshold')}
                            <span class="tag tag-fail">RTO</span>
                        {:else if log.includes('PULIH') || log.includes('selesai')}
                            <span class="tag tag-ok">OK</span>
                        {:else}
                            <span class="tag tag-info">INFO</span>
                        {/if}
                        <span class="log-msg">{log}</span>
                    </div>
                {/each}
            {:else}
                <div class="log-empty">Belum ada riwayat aktivitas. Watchdog siap digunakan.</div>
            {/if}
        </div>
    </div>

    <!-- UNIVERSAL POPUP MODAL (THEME & COLOR SYNCED) -->
    {#if modal.show}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="custom-modal" on:click={() => modal.show = false}>
            <div class="modal-card" on:click|stopPropagation>
                <div class="modal-title">
                    <i class="fa-solid fa-triangle-exclamation"></i> {modal.title}
                </div>
                <div class="modal-body">
                    <p class="modal-text">{@html modal.message}</p>
                    {#if modal.warning}
                        <div class="modal-warning">
                            <i class="fa-solid fa-triangle-exclamation"></i>
                            <span>{modal.warning}</span>
                        </div>
                    {/if}
                </div>
                <div class="modal-actions">
                    <button class="btn-modal btn-cancel" on:click={() => modal.show = false}>
                        {modal.type === 'alert' ? 'Tutup' : 'Batal'}
                    </button>
                    {#if modal.type === 'confirm' && modal.confirmAction}
                        <button class="btn-modal btn-confirm" on:click={modal.confirmAction}>
                            <i class="fa-solid {modal.confirmIcon || 'fa-bolt'}"></i> {modal.confirmText || 'Lanjutkan'}
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .wrap {
        padding: 16px;
        max-width: 760px;
        margin: 0 auto;
        padding-bottom: 90px;
        box-sizing: border-box;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 14px;
        gap: 10px;
    }

    .eyebrow {
        font-size: 0.72rem;
        font-weight: 800;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--yellow);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 2px;
    }

    .live-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: var(--muted);
        display: inline-block;
    }

    .live-dot.is-active {
        background: var(--lime);
        box-shadow: 0 0 6px var(--lime);
    }

    h1 {
        font-size: 1.6rem;
        font-weight: 900;
        letter-spacing: -0.03em;
        margin: 0 0 3px 0;
        color: var(--ink);
        text-transform: uppercase;
    }

    .desc {
        font-size: 0.78rem;
        color: var(--muted);
        line-height: 1.4;
    }

    .btn-refresh {
        background: var(--paper);
        border: 2px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
        color: var(--ink);
        width: 38px;
        height: 38px;
        border-radius: 8px;
        cursor: pointer;
        display: grid;
        place-items: center;
        flex-shrink: 0;
        transition: .15s;
    }

    .btn-refresh:hover {
        background: var(--yellow);
        color: #000;
        transform: translate(-1px, -1px);
    }

    /* ALERTS */
    .alert {
        padding: 10px 14px;
        border-radius: 10px;
        font-size: 0.8rem;
        font-weight: 800;
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 12px;
        border: 2px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
    }

    .alert-success {
        background: rgba(184, 243, 74, 0.15);
        color: var(--lime);
        border-color: var(--lime);
    }

    .alert-error {
        background: rgba(255, 92, 168, 0.15);
        color: var(--pink);
        border-color: var(--pink);
    }

    /* CARDS */
    .card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 14px;
        padding: 16px;
        margin-bottom: 14px;
        box-shadow: var(--shadow);
        box-sizing: border-box;
    }

    .card-header {
        font-size: 12px;
        font-weight: 950;
        text-transform: uppercase;
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--ink);
        border-bottom: 2px dashed var(--line);
        padding-bottom: 10px;
        margin-bottom: 12px;
    }

    /* HERO CARD */
    .hero-top {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        margin-bottom: 14px;
    }

    .status-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        border-radius: 6px;
        font-size: 11px;
        font-weight: 950;
        text-transform: uppercase;
        border: 1.5px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
        margin-bottom: 6px;
    }

    .badge-on {
        background: var(--lime);
        color: #000;
    }

    .badge-off {
        background: var(--panel-alt);
        color: var(--muted);
    }

    .hero-sub {
        font-size: 11px;
        color: var(--muted);
        font-weight: 700;
        line-height: 1.35;
    }

    /* TOGGLE SWITCH */
    .toggle-switch {
        width: 54px;
        height: 30px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 30px;
        position: relative;
        cursor: pointer;
        box-shadow: 2px 2px 0 var(--line);
        flex-shrink: 0;
        transition: .15s;
        padding: 0;
    }

    .toggle-switch.checked {
        background: var(--lime);
    }

    .toggle-knob {
        width: 22px;
        height: 22px;
        background: var(--ink);
        border: 2px solid var(--line);
        border-radius: 50%;
        position: absolute;
        top: 2px;
        left: 2px;
        transition: transform .15s ease-out;
    }

    .toggle-switch.checked .toggle-knob {
        transform: translateX(24px);
        background: #000;
    }

    /* KPI STRIP */
    .kpi-strip {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 8px;
        background: var(--panel-alt);
        padding: 10px;
        border-radius: 10px;
        border: 1.5px solid var(--line);
    }

    .kpi-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
        gap: 2px;
    }

    .kpi-label {
        font-size: 9px;
        font-weight: 850;
        text-transform: uppercase;
        color: var(--muted);
    }

    .kpi-val {
        font-size: 13px;
        font-weight: 950;
        font-family: 'JetBrains Mono', monospace;
    }

    .color-lime { color: var(--lime); }
    .color-pink { color: var(--pink); }
    .color-cyan { color: var(--cyan); }
    .color-yellow { color: var(--yellow); }

    /* FORM ELEMENTS */
    .form-body {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .form-row {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .form-label {
        font-size: 10px;
        font-weight: 850;
        text-transform: uppercase;
        color: var(--muted);
    }

    .target-input-wrap {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .form-input {
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 8px 12px;
        color: var(--ink);
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        font-weight: 800;
        outline: none;
    }

    .form-input:focus {
        border-color: var(--yellow);
    }

    .quick-chips {
        display: flex;
        gap: 6px;
        flex-wrap: wrap;
    }

    .chip-btn {
        background: var(--panel-alt);
        border: 1.5px solid var(--line);
        border-radius: 6px;
        padding: 4px 8px;
        font-size: 10px;
        font-weight: 800;
        color: var(--ink);
        cursor: pointer;
        transition: .1s;
    }

    .chip-btn.chip-active, .chip-btn:hover {
        background: var(--yellow);
        color: #000;
    }

    .form-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 10px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 5px;
    }

    .form-select {
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 8px 10px;
        color: var(--ink);
        font-size: 11px;
        font-weight: 800;
        outline: none;
        cursor: pointer;
    }

    .form-select:focus {
        border-color: var(--cyan);
    }

    .btn-save {
        margin-top: 4px;
        background: var(--lime);
        color: #000;
        border: 2px solid var(--line);
        border-radius: 10px;
        padding: 10px 16px;
        font-size: 12px;
        font-weight: 950;
        text-transform: uppercase;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        box-shadow: 3px 3px 0 var(--line);
        transition: .12s;
    }

    .btn-save:hover {
        transform: translate(-1px, -1px);
        box-shadow: 4px 4px 0 var(--line);
    }

    /* ACTION CARD */
    .action-buttons-row {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 10px;
    }

    .btn-action-test {
        background: var(--panel-alt);
        color: var(--ink);
        border: 2px solid var(--line);
        border-radius: 10px;
        padding: 10px 14px;
        font-size: 11px;
        font-weight: 950;
        text-transform: uppercase;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        box-shadow: 2px 2px 0 var(--line);
        transition: .1s;
    }

    .btn-action-test:hover {
        background: var(--yellow);
        color: #000;
        transform: translate(-1px, -1px);
    }

    .btn-action-recovery {
        background: rgba(255, 92, 168, 0.15);
        color: var(--pink);
        border-color: var(--pink);
    }

    .btn-action-recovery:hover {
        background: var(--pink);
        color: #fff;
    }

    /* LOGS */
    .log-box {
        background: #121016;
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 10px;
        font-family: 'JetBrains Mono', monospace;
        font-size: 11px;
        max-height: 160px;
        overflow-y: auto;
        display: flex;
        flex-direction: column-reverse;
        gap: 6px;
    }

    .log-item {
        display: flex;
        gap: 6px;
        align-items: baseline;
    }

    .tag {
        font-size: 9px;
        font-weight: 900;
        padding: 1px 4px;
        border-radius: 3px;
        flex-shrink: 0;
    }

    .tag-ok { background: var(--lime); color: #000; }
    .tag-fail { background: var(--pink); color: #fff; }
    .tag-info { background: var(--cyan); color: #000; }

    .log-msg {
        color: #e2e8f0;
        word-break: break-all;
    }

    .log-empty {
        color: #64748b;
        font-style: italic;
    }

    /* UNIVERSAL POPUP MODAL (THEME & COLOR SYNCED) */
    .custom-modal {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        z-index: 10000;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 16px;
        backdrop-filter: blur(3px);
    }

    .modal-card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 16px;
        padding: 20px;
        width: 380px;
        max-width: 100%;
        box-shadow: var(--shadow);
        color: var(--ink);
    }

    .modal-title {
        font-size: 13px;
        font-weight: 950;
        text-transform: uppercase;
        padding: 4px 10px;
        border: 2px solid var(--line);
        border-radius: 6px;
        box-shadow: 2px 2px 0 var(--line);
        display: inline-flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 12px;
        background: var(--yellow);
        color: #000;
    }

    .modal-body {
        font-size: 13px;
        line-height: 1.5;
        color: var(--ink);
        margin-bottom: 16px;
        font-weight: 700;
    }

    .modal-text { margin: 0 0 8px 0; }

    .modal-warning {
        background: rgba(255, 92, 168, 0.12);
        border: 2px dashed var(--pink);
        border-radius: 8px;
        padding: 8px 10px;
        font-size: 11px;
        color: var(--pink);
        display: flex;
        gap: 8px;
        align-items: flex-start;
        margin-top: 8px;
        font-weight: 800;
        line-height: 1.4;
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
    }

    .btn-modal {
        padding: 8px 14px;
        font-weight: 950;
        font-size: 11px;
        border-radius: 8px;
        border: 2px solid var(--line);
        cursor: pointer;
        box-shadow: 3px 3px 0 var(--line);
        text-transform: uppercase;
        transition: .1s;
        font-family: inherit;
    }

    .btn-cancel { background: var(--panel-alt); color: var(--ink); }
    .btn-confirm { background: var(--yellow); color: #000; }

    @media (max-width: 600px) {
        .form-grid {
            grid-template-columns: 1fr;
        }
        .action-buttons-row {
            grid-template-columns: 1fr;
        }
    }
</style>
