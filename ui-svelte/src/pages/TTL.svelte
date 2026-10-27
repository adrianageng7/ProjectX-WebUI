<script>
    import { onMount, onDestroy } from 'svelte';

    let loading = true;
    let saving = false;
    let data = {
        enabled: false,
        ttl: 64,
        ipv6_hl: true,
        sysctl_ttl: 64,
        active_rules_v4: [],
        active_rules_v6: []
    };

    let form = {
        enabled: false,
        ttl: 64,
        ipv6_hl: true
    };

    let message = '';
    let error = '';
    let timer = null;

    async function fetchStatus(isInitial = false) {
        if (isInitial) loading = true;
        try {
            const res = await fetch('/api/ttl/status?_=' + Date.now());
            if (res.ok) {
                const json = await res.json();
                data = json;
                if (isInitial) {
                    form = {
                        enabled: !!json.enabled,
                        ttl: json.ttl || 64,
                        ipv6_hl: json.ipv6_hl !== false
                    };
                }
            }
        } catch (e) {
            console.warn('Gagal memuat status TTL:', e);
        } finally {
            if (isInitial) loading = false;
        }
    }

    function setPreset(val) {
        form.ttl = val;
    }

    async function saveConfig() {
        message = '';
        error = '';
        saving = true;

        const ttlNum = parseInt(form.ttl);
        if (isNaN(ttlNum) || ttlNum < 1 || ttlNum > 255) {
            error = 'Nilai TTL harus berada di antara 1 dan 255!';
            saving = false;
            return;
        }

        try {
            const res = await fetch('/api/ttl/set', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    enabled: form.enabled,
                    ttl: ttlNum,
                    ipv6_hl: form.ipv6_hl
                })
            });

            const json = await res.json();
            if (res.ok && json.status === 'success') {
                data = json;
                message = json.message || 'Pengaturan TTL berhasil diterapkan!';
                setTimeout(() => { message = ''; }, 4000);
            } else {
                error = json.message || 'Gagal menyimpan pengaturan TTL.';
            }
        } catch (e) {
            error = 'Gagal menghubungi server: ' + e.message;
        } finally {
            saving = false;
        }
    }

    async function quickToggle() {
        form.enabled = !form.enabled;
        await saveConfig();
    }

    onMount(() => {
        fetchStatus(true);
        timer = setInterval(() => fetchStatus(false), 5000);
    });

    onDestroy(() => {
        if (timer) clearInterval(timer);
    });
</script>

<div class="wrap">
    <!-- HEADER -->
    <div class="header">
        <div class="header-left">
            <div class="eyebrow"><span class="live-dot"></span> Bypass Tethering & Kuota Hotspot</div>
            <h1>TTL & Hop Limit</h1>
            <div class="desc">Ubah Time To Live (TTL) paket IP agar traffic tethering terdeteksi sebagai kuota reguler HP</div>
        </div>
        <button class="btn btn-refresh" on:click={() => fetchStatus(true)} title="Pindai Ulang">
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

    <!-- STATS ROW -->
    <div class="stats-row">
        <div class="stat-pill {data.enabled ? 'active-pill' : ''}">
            <i class="fa-solid {data.enabled ? 'fa-shield-halved' : 'fa-shield-slash'}" style="color:{data.enabled ? 'var(--lime)' : 'var(--pink)'};"></i>
            <span>Bypass: <strong>{data.enabled ? 'AKTIF' : 'NONAKTIF'}</strong></span>
        </div>
        <div class="stat-pill">
            <i class="fa-solid fa-network-wired" style="color:var(--cyan);"></i>
            <span>Target TTL: <strong>{data.enabled ? data.ttl : (data.sysctl_ttl || 64)}</strong></span>
        </div>
        <div class="stat-pill">
            <i class="fa-solid fa-globe" style="color:var(--purple);"></i>
            <span>IPv6 HL: <strong>{data.ipv6_hl ? 'Sinkron' : 'Off'}</strong></span>
        </div>
    </div>

    <!-- MAIN CONTROL CARD -->
    <div class="card">
        <div class="card-header">
            <div class="card-title">
                <i class="fa-solid fa-sliders"></i> Konfigurasi TTL Modifier
            </div>
            <div class="switch-box">
                <span class="switch-label">{form.enabled ? 'Bypass Menyala' : 'Bypass Mati'}</span>
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="toggle-switch {form.enabled ? 'checked' : ''}" on:click={quickToggle}>
                    <div class="switch-handle"></div>
                </div>
            </div>
        </div>

        <div class="card-body">
            <!-- PRESETS -->
            <div class="section-label">Pilihan Preset Populer:</div>
            <div class="preset-grid">
                <button
                    type="button"
                    class="preset-card {form.ttl === 64 ? 'active' : ''}"
                    on:click={() => setPreset(64)}
                >
                    <div class="preset-badge badge-rec">Rekomendasi</div>
                    <div class="preset-title">TTL 64</div>
                    <div class="preset-sub">Standar Android / Linux. Paling cocok untuk tethering langsung ke Laptop / PC Windows.</div>
                </button>

                <button
                    type="button"
                    class="preset-card {form.ttl === 65 ? 'active' : ''}"
                    on:click={() => setPreset(65)}
                >
                    <div class="preset-badge">Router Mode</div>
                    <div class="preset-title">TTL 65</div>
                    <div class="preset-sub">Cocok jika HP dicolokkan ke Router OpenWrt / GL.iNet sebelum dibagi ke banyak client.</div>
                </button>

                <button
                    type="button"
                    class="preset-card {form.ttl === 128 ? 'active' : ''}"
                    on:click={() => setPreset(128)}
                >
                    <div class="preset-badge">Windows Native</div>
                    <div class="preset-title">TTL 128</div>
                    <div class="preset-sub">Nilai default bawaan Windows. Digunakan jika ingin menyamakan seluruh traffic ke Windows.</div>
                </button>
            </div>

            <!-- SLIDER / MANUAL INPUT -->
            <div class="slider-section">
                <div class="slider-header">
                    <span class="slider-title">Nilai TTL Manual:</span>
                    <span class="slider-val-badge">{form.ttl}</span>
                </div>
                <div class="slider-input-row">
                    <input
                        type="range"
                        min="1"
                        max="255"
                        bind:value={form.ttl}
                        class="range-slider"
                    >
                    <input
                        type="number"
                        min="1"
                        max="255"
                        bind:value={form.ttl}
                        class="number-input"
                    >
                </div>
                <div class="slider-marks">
                    <span>1 (Min)</span>
                    <span>64 (HP)</span>
                    <span>65 (Router)</span>
                    <span>128 (Windows)</span>
                    <span>255 (Max)</span>
                </div>
            </div>

            <!-- IPV6 TOGGLE -->
            <div class="checkbox-row">
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <label class="custom-checkbox-wrap">
                    <input type="checkbox" bind:checked={form.ipv6_hl}>
                    <span class="checkmark"></span>
                    <span class="checkbox-text">
                        <strong>Sinkronkan IPv6 Hop Limit (HL)</strong>
                        <span class="checkbox-sub">Otomatis atur Hop Limit paket IPv6 ke nilai yang sama ({form.ttl}) via <code>ip6tables</code>.</span>
                    </span>
                </label>
            </div>

            <!-- ACTION BUTTON -->
            <div class="action-footer">
                <button
                    class="btn-save {saving ? 'is-loading' : ''}"
                    on:click={saveConfig}
                    disabled={saving}
                >
                    {#if saving}
                        <i class="fa-solid fa-spinner fa-spin"></i> Menerapkan...
                    {:else}
                        <i class="fa-solid fa-floppy-disk"></i> Simpan & Terapkan
                    {/if}
                </button>
            </div>
        </div>
    </div>

    <!-- RULES INSPECTOR CARD -->
    <div class="card">
        <div class="card-header">
            <div class="card-title">
                <i class="fa-solid fa-terminal"></i> Status Kernel & Iptables
            </div>
            <span class="badge {data.enabled ? 'badge-online' : 'badge-offline'}">
                {data.enabled ? 'Rules Active' : 'No Rules Applied'}
            </span>
        </div>
        <div class="card-body">
            <div class="rules-box">
                <div class="rules-header">IPv4 Mangle Chain (PROJECTX_TTL):</div>
                {#if data.active_rules_v4 && data.active_rules_v4.length > 0}
                    {#each data.active_rules_v4 as r}
                        <div class="code-line"><i class="fa-solid fa-check" style="color:var(--lime);"></i> {r}</div>
                    {/each}
                {:else}
                    <div class="code-empty">Tidak ada aturan aktif (Bypass dimatikan)</div>
                {/if}

                {#if data.ipv6_hl}
                    <div class="rules-header" style="margin-top:10px;">IPv6 Mangle Chain (PROJECTX_TTL):</div>
                    {#if data.active_rules_v6 && data.active_rules_v6.length > 0}
                        {#each data.active_rules_v6 as r}
                            <div class="code-line"><i class="fa-solid fa-check" style="color:var(--purple);"></i> {r}</div>
                        {/each}
                    {:else}
                        <div class="code-empty">Tidak ada aturan IPv6 aktif</div>
                    {/if}
                {/if}
            </div>
        </div>
    </div>

    <!-- EXPLANATION CARD -->
    <div class="card info-card">
        <div class="card-header">
            <div class="card-title">
                <i class="fa-solid fa-circle-question"></i> Cara Kerja & Manfaat TTL Bypass
            </div>
        </div>
        <div class="card-body info-body">
            <div class="info-item">
                <div class="info-icon" style="background:var(--pink);"><i class="fa-solid fa-mobile"></i></div>
                <div class="info-content">
                    <h4>Kenapa Kuota Hotspot Cepat Habis?</h4>
                    <p>Operator seluler membedakan kuota utama vs kuota hotspot melalui nilai <strong>TTL (Time To Live)</strong>. Windows mengirim TTL 128, sedangkan Android mengirim TTL 64. Saat tethering aktif, operator melihat TTL 127/128 dan langsung memotong kuota hotspot Anda.</p>
                </div>
            </div>

            <div class="info-item">
                <div class="info-icon" style="background:var(--lime);"><i class="fa-solid fa-shield-heart"></i></div>
                <div class="info-content">
                    <h4>Bagaimana Fitur Ini Membantu?</h4>
                    <p>Dengan mengaktifkan fitur ini, seluruh paket dari PC / Laptop / TV yang melewati HP akan dimodifikasi di tabel kernel <code>mangle POSTROUTING</code> menjadi <strong>TTL 64</strong>. Operator melihat semua traffic seolah-olah Anda sedang browsing langsung di HP, sehingga <strong>hanya memotong kuota reguler utama</strong>.</p>
                </div>
            </div>

            <div class="info-item">
                <div class="info-icon" style="background:var(--yellow);"><i class="fa-solid fa-route"></i></div>
                <div class="info-content">
                    <h4>Kapan Harus Menggunakan TTL 65?</h4>
                    <p>Gunakan <strong>TTL 65</strong> jika topologi jaringan Anda adalah: <code>HP Modem ➔ Router OpenWrt ➔ Laptop</code>. Setiap kali melewati router, TTL akan dikurangi 1 (65 - 1 = 64), sehingga saat keluar dari HP nilainya tepat 64.</p>
                </div>
            </div>
        </div>
    </div>
</div>

<style>
    .wrap {
        padding: 18px;
        max-width: 900px;
        margin: 0 auto;
        padding-bottom: 90px;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 16px;
    }

    .eyebrow {
        font-size: 0.72rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--pink);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 2px;
    }

    .live-dot {
        width: 7px;
        height: 7px;
        background: var(--lime);
        border-radius: 50%;
        display: inline-block;
        box-shadow: 0 0 6px var(--lime);
    }

    h1 {
        font-size: 1.8rem;
        font-weight: 900;
        letter-spacing: -0.03em;
        margin: 0 0 4px 0;
        color: var(--text);
    }

    .desc {
        font-size: 0.82rem;
        color: var(--text-muted);
        line-height: 1.4;
    }

    .btn-refresh {
        background: var(--surface);
        border: 2px solid var(--border);
        box-shadow: 2px 2px 0 var(--border);
        color: var(--text);
        width: 40px;
        height: 40px;
        border-radius: 10px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1rem;
        transition: transform 0.1s;
    }

    .btn-refresh:active {
        transform: translate(2px, 2px);
        box-shadow: none;
    }

    .alert {
        padding: 10px 14px;
        border-radius: 10px;
        font-size: 0.82rem;
        font-weight: 700;
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 14px;
        border: 2px solid var(--border);
    }

    .alert-success {
        background: rgba(166, 227, 161, 0.2);
        color: var(--lime);
    }

    .alert-error {
        background: rgba(243, 139, 168, 0.2);
        color: var(--pink);
    }

    .stats-row {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin-bottom: 16px;
    }

    .stat-pill {
        background: var(--surface);
        border: 2px solid var(--border);
        box-shadow: 2px 2px 0 var(--border);
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 0.78rem;
        display: flex;
        align-items: center;
        gap: 6px;
        color: var(--text);
    }

    .stat-pill.active-pill {
        border-color: var(--lime);
        background: rgba(166, 227, 161, 0.1);
    }

    .card {
        background: var(--surface);
        border: 2px solid var(--border);
        box-shadow: 4px 4px 0 var(--border);
        border-radius: 14px;
        margin-bottom: 16px;
        overflow: hidden;
    }

    .card-header {
        padding: 12px 16px;
        border-bottom: 2px solid var(--border);
        background: var(--card-header-bg, rgba(0, 0, 0, 0.1));
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .card-title {
        font-size: 0.95rem;
        font-weight: 800;
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--text);
    }

    .card-body {
        padding: 16px;
    }

    .switch-box {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .switch-label {
        font-size: 0.78rem;
        font-weight: 700;
        color: var(--text-muted);
    }

    .toggle-switch {
        width: 46px;
        height: 26px;
        background: var(--surface-secondary, #444);
        border: 2px solid var(--border);
        border-radius: 14px;
        cursor: pointer;
        position: relative;
        transition: background 0.2s;
    }

    .toggle-switch.checked {
        background: var(--lime);
    }

    .switch-handle {
        width: 18px;
        height: 18px;
        background: #fff;
        border: 2px solid var(--border);
        border-radius: 50%;
        position: absolute;
        top: 2px;
        left: 2px;
        transition: left 0.2s;
    }

    .toggle-switch.checked .switch-handle {
        left: 22px;
    }

    .section-label {
        font-size: 0.8rem;
        font-weight: 800;
        text-transform: uppercase;
        color: var(--text-muted);
        letter-spacing: 0.05em;
        margin-bottom: 10px;
    }

    .preset-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 12px;
        margin-bottom: 20px;
    }

    .preset-card {
        background: var(--surface);
        border: 2px solid var(--border);
        box-shadow: 2px 2px 0 var(--border);
        border-radius: 12px;
        padding: 14px;
        text-align: left;
        cursor: pointer;
        transition: transform 0.1s, border-color 0.15s;
        display: flex;
        flex-direction: column;
    }

    .preset-card:active {
        transform: translate(2px, 2px);
        box-shadow: none;
    }

    .preset-card.active {
        border-color: var(--accent, var(--yellow));
        background: rgba(249, 226, 175, 0.08);
        box-shadow: 3px 3px 0 var(--accent, var(--yellow));
    }

    .preset-badge {
        font-size: 0.65rem;
        font-weight: 800;
        text-transform: uppercase;
        padding: 2px 6px;
        border-radius: 6px;
        background: var(--surface-secondary, rgba(255, 255, 255, 0.1));
        color: var(--text-muted);
        display: inline-block;
        margin-bottom: 6px;
        align-self: flex-start;
        border: 1px solid var(--border);
    }

    .preset-badge.badge-rec {
        background: var(--lime);
        color: #000;
        border-color: #000;
    }

    .preset-title {
        font-size: 1.2rem;
        font-weight: 900;
        color: var(--text);
        margin-bottom: 4px;
    }

    .preset-sub {
        font-size: 0.74rem;
        color: var(--text-muted);
        line-height: 1.35;
    }

    .slider-section {
        background: var(--surface-secondary, rgba(0, 0, 0, 0.1));
        border: 2px solid var(--border);
        border-radius: 12px;
        padding: 14px;
        margin-bottom: 16px;
    }

    .slider-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
    }

    .slider-title {
        font-size: 0.85rem;
        font-weight: 800;
        color: var(--text);
    }

    .slider-val-badge {
        font-size: 1.1rem;
        font-weight: 900;
        color: var(--accent, var(--yellow));
        background: var(--surface);
        border: 2px solid var(--border);
        padding: 2px 10px;
        border-radius: 8px;
    }

    .slider-input-row {
        display: flex;
        gap: 12px;
        align-items: center;
        margin-bottom: 6px;
    }

    .range-slider {
        flex: 1;
        accent-color: var(--accent, var(--yellow));
        cursor: pointer;
        height: 6px;
    }

    .number-input {
        width: 70px;
        background: var(--surface);
        border: 2px solid var(--border);
        color: var(--text);
        font-size: 1rem;
        font-weight: 800;
        text-align: center;
        padding: 6px;
        border-radius: 8px;
    }

    .slider-marks {
        display: flex;
        justify-content: space-between;
        font-size: 0.68rem;
        color: var(--text-muted);
        font-weight: 600;
    }

    .checkbox-row {
        margin-bottom: 20px;
    }

    .custom-checkbox-wrap {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        cursor: pointer;
        font-size: 0.85rem;
    }

    .custom-checkbox-wrap input {
        display: none;
    }

    .checkmark {
        width: 20px;
        height: 20px;
        background: var(--surface);
        border: 2px solid var(--border);
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        margin-top: 2px;
    }

    .custom-checkbox-wrap input:checked ~ .checkmark {
        background: var(--lime);
    }

    .custom-checkbox-wrap input:checked ~ .checkmark::after {
        content: '✓';
        color: #000;
        font-weight: 900;
        font-size: 0.8rem;
    }

    .checkbox-text {
        display: flex;
        flex-direction: column;
        color: var(--text);
    }

    .checkbox-sub {
        font-size: 0.72rem;
        color: var(--text-muted);
        margin-top: 2px;
    }

    .action-footer {
        display: flex;
        justify-content: flex-end;
    }

    .btn-save {
        background: var(--lime);
        color: #000;
        border: 2px solid var(--border);
        box-shadow: 3px 3px 0 var(--border);
        padding: 10px 20px;
        border-radius: 10px;
        font-size: 0.9rem;
        font-weight: 900;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: transform 0.1s;
    }

    .btn-save:active {
        transform: translate(2px, 2px);
        box-shadow: none;
    }

    .badge {
        font-size: 0.68rem;
        font-weight: 800;
        padding: 3px 8px;
        border-radius: 6px;
        border: 1px solid var(--border);
    }

    .badge-online {
        background: rgba(166, 227, 161, 0.2);
        color: var(--lime);
    }

    .badge-offline {
        background: rgba(243, 139, 168, 0.2);
        color: var(--pink);
    }

    .rules-box {
        background: #0d0f12;
        border: 2px solid var(--border);
        border-radius: 8px;
        padding: 12px;
        font-family: monospace;
        font-size: 0.78rem;
    }

    .rules-header {
        font-weight: 800;
        color: var(--text-muted);
        margin-bottom: 6px;
    }

    .code-line {
        color: #e0e0e0;
        padding: 2px 0;
        word-break: break-all;
    }

    .code-empty {
        color: #666;
        font-style: italic;
    }

    .info-card {
        background: var(--surface);
    }

    .info-body {
        display: flex;
        flex-direction: column;
        gap: 14px;
    }

    .info-item {
        display: flex;
        gap: 12px;
        align-items: flex-start;
    }

    .info-icon {
        width: 34px;
        height: 34px;
        border-radius: 8px;
        border: 2px solid var(--border);
        display: flex;
        align-items: center;
        justify-content: center;
        color: #000;
        font-size: 0.9rem;
        flex-shrink: 0;
    }

    .info-content h4 {
        margin: 0 0 2px 0;
        font-size: 0.85rem;
        font-weight: 800;
        color: var(--text);
    }

    .info-content p {
        margin: 0;
        font-size: 0.76rem;
        color: var(--text-muted);
        line-height: 1.4;
    }
</style>

