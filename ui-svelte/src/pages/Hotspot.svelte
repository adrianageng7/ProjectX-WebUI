<script>
    import { onMount } from 'svelte';
    import QRCode from 'qrcode';

    let loading = true;
    let saving = false;
    let toggling = false;
    let showPassword = false;
    let copied = false;

    let status = {
        enabled: false,
        interface: '',
        ip_address: '',
        config: {
            ssid: '',
            password: '',
            security: 'wpa2',
            band: '2.4',
            hidden: false
        },
        qr_payload: ''
    };

    let form = {
        ssid: '',
        password: '',
        band: '2.4',
        hidden: false
    };

    let qrDataUrl = '';

    let toast = {
        show: false,
        message: '',
        type: 'success'
    };

    function showToast(message, type = 'success') {
        toast = { show: true, message, type };
        setTimeout(() => {
            toast.show = false;
        }, 3500);
    }

    async function generateQRCode() {
        const ssid = form.ssid || status.config.ssid || 'Wi-Fi';
        const password = form.password || status.config.password || '';
        const payload = status.qr_payload || (password ? `WIFI:S:${ssid};T:WPA;P:${password};;` : `WIFI:S:${ssid};T:nopass;;`);
        try {
            qrDataUrl = await QRCode.toDataURL(payload, {
                width: 240,
                margin: 2,
                color: {
                    dark: '#000000',
                    light: '#ffffff'
                }
            });
        } catch (e) {
            console.error('QR Code error:', e);
        }
    }

    async function fetchStatus() {
        try {
            const res = await fetch('/api/hotspot/status?_=' + Date.now());
            const json = await res.json();
            if (json.status === 'success' && json.data) {
                status = json.data;
                form = {
                    ssid: status.config.ssid || '',
                    password: status.config.password || '',
                    band: status.config.band || '2.4',
                    hidden: !!status.config.hidden
                };
                await generateQRCode();
            }
        } catch (e) {
            console.error('Failed to fetch hotspot status', e);
        } finally {
            loading = false;
        }
    }

    async function toggleHotspot() {
        if (toggling) return;
        toggling = true;
        const newTarget = !status.enabled;
        try {
            const res = await fetch('/api/hotspot/toggle', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ enable: newTarget })
            });
            const json = await res.json();
            if (json.status === 'success') {
                showToast(
                    newTarget ? 'Hotspot Wi-Fi berhasil dinyalakan!' : 'Hotspot Wi-Fi telah dimatikan.',
                    'success'
                );
            } else {
                showToast(json.message || 'Gagal mengubah status hotspot', 'error');
            }
        } catch (e) {
            showToast('Terjadi kesalahan jaringan', 'error');
        } finally {
            toggling = false;
            fetchStatus();
        }
    }

    async function saveConfig() {
        if (saving) return;
        if (!form.ssid.trim()) {
            showToast('Nama SSID tidak boleh kosong', 'error');
            return;
        }
        if (form.password && form.password.length < 8) {
            showToast('Kata sandi Wi-Fi minimal 8 karakter', 'error');
            return;
        }

        saving = true;
        try {
            const res = await fetch('/api/hotspot/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(form)
            });
            const json = await res.json();
            if (json.status === 'success') {
                showToast('Nama SSID & kata sandi berhasil diperbarui!', 'success');
                status.config = { ...form };
                if (json.qr_payload) {
                    status.qr_payload = json.qr_payload;
                }
                await generateQRCode();
            } else {
                showToast(json.message || 'Gagal menyimpan konfigurasi', 'error');
            }
        } catch (e) {
            showToast('Gagal menghubungi server', 'error');
        } finally {
            saving = false;
            fetchStatus();
        }
    }

    function copyWifiInfo() {
        const text = `Wi-Fi: ${form.ssid}\nPassword: ${form.password || '(Tanpa Password)'}`;
        navigator.clipboard.writeText(text);
        copied = true;
        showToast('Info Wi-Fi disalin ke papan klip!', 'info');
        setTimeout(() => copied = false, 2000);
    }

    onMount(() => {
        fetchStatus();
    });
</script>

<div class="wrap">
    <!-- HEADER -->
    <div class="header">
        <div class="eyebrow">
            <span class="live-dot {status.enabled ? 'dot-active' : ''}"></span>
            Hotspot & Tethering Gateway
        </div>
        <h1>Wi-Fi Hotspot</h1>
        <div class="desc">Kelola nama hotspot, kata sandi, pita frekuensi, dan bagikan koneksi via QR Code.</div>
    </div>

    {#if toast.show}
        <div class="toast-bar {toast.type}">
            <i class="fa-solid {toast.type === 'success' ? 'fa-circle-check' : (toast.type === 'error' ? 'fa-triangle-exclamation' : 'fa-circle-info')}"></i>
            <span>{toast.message}</span>
        </div>
    {/if}

    {#if loading}
        <div class="loading-state">
            <i class="fa-solid fa-spinner fa-spin"></i>
            <span>Membaca data Hotspot perangkat...</span>
        </div>
    {:else}
        <!-- CARD 1: MASTER TOGGLE HERO -->
        <div class="card hero-card {status.enabled ? 'is-active' : 'is-inactive'}">
            <div class="hero-left">
                <div class="status-badge {status.enabled ? 'badge-on' : 'badge-off'}">
                    <span class="pulse-indicator"></span>
                    {status.enabled ? 'HOTSPOT AKTIF' : 'HOTSPOT MATI'}
                </div>
                <div class="hero-title">
                    {status.enabled ? form.ssid || 'Hotspot Menyala' : 'Hotspot Tidak Aktif'}
                </div>
                <div class="hero-sub">
                    {#if status.enabled}
                        <span>Interface: <strong>{status.interface || 'wlan'}</strong></span>
                        {#if status.ip_address}
                            <span> • Gateway: <strong>{status.ip_address}</strong></span>
                        {/if}
                    {:else}
                        Aktifkan tethering agar perangkat lain dapat tersambung ke HP ini.
                    {/if}
                </div>
            </div>

            <button 
                type="button" 
                class="btn-toggle {status.enabled ? 'btn-stop' : 'btn-start'}" 
                disabled={toggling}
                on:click={toggleHotspot}
            >
                <i class="fa-solid {toggling ? 'fa-spinner fa-spin' : (status.enabled ? 'fa-power-off' : 'fa-wifi')}"></i>
                <span>{status.enabled ? 'Matikan' : 'Nyalakan'}</span>
            </button>
        </div>

        <div class="grid-layout">
            <!-- CARD 2: QR CODE QUICK CONNECT -->
            <div class="card qr-card">
                <div class="card-header">
                    <i class="fa-solid fa-qrcode" style="color:var(--yellow);"></i>
                    <span>Scan untuk Konek</span>
                </div>
                <div class="qr-box">
                    {#if qrDataUrl}
                        <img src={qrDataUrl} alt="Wi-Fi QR Code" class="qr-img" />
                    {:else}
                        <div class="qr-placeholder">
                            <i class="fa-solid fa-spinner fa-spin"></i>
                            <span>Membuat QR...</span>
                        </div>
                    {/if}
                </div>
                <div class="qr-info">
                    <div class="qr-ssid">{form.ssid || status.config.ssid || 'Wi-Fi Hotspot'}</div>
                    <div class="qr-hint">Scan dengan kamera HP tamu untuk otomatis terhubung tanpa repot mengetik.</div>
                    <button type="button" class="btn-copy-info" on:click={copyWifiInfo}>
                        <i class="fa-solid {copied ? 'fa-check' : 'fa-copy'}"></i>
                        <span>{copied ? 'Tersalin' : 'Salin Info Wi-Fi'}</span>
                    </button>
                </div>
            </div>

            <!-- CARD 3: EDIT CONFIG FORM -->
            <div class="card form-card">
                <div class="card-header">
                    <i class="fa-solid fa-sliders" style="color:var(--cyan);"></i>
                    <span>Pengaturan Jaringan</span>
                </div>

                <div class="form-body">
                    <!-- SSID -->
                    <div class="form-group">
                        <label class="form-label" for="ssid-input">Nama Hotspot (SSID)</label>
                        <div class="input-wrap">
                            <i class="fa-solid fa-wifi input-icon"></i>
                            <input 
                                id="ssid-input"
                                type="text" 
                                class="form-input" 
                                bind:value={form.ssid} 
                                placeholder="Nama Jaringan Wi-Fi" 
                            />
                        </div>
                    </div>

                    <!-- PASSWORD -->
                    <div class="form-group">
                        <label class="form-label" for="pass-input">Kata Sandi Wi-Fi</label>
                        <div class="input-wrap">
                            <i class="fa-solid fa-key input-icon"></i>
                            <input 
                                id="pass-input"
                                type={showPassword ? 'text' : 'password'} 
                                class="form-input" 
                                bind:value={form.password} 
                                placeholder="Minimal 8 karakter (kosongkan jika tanpa sandi)" 
                            />
                            <button 
                                type="button" 
                                class="btn-reveal" 
                                on:click={() => showPassword = !showPassword}
                                title={showPassword ? 'Sembunyikan' : 'Lihat Sandi'}
                            >
                                <i class="fa-solid {showPassword ? 'fa-eye-slash' : 'fa-eye'}"></i>
                            </button>
                        </div>
                    </div>

                    <!-- FREKUENSI BAND -->
                    <div class="form-group">
                        <span class="form-label">Pita Frekuensi Hotspot</span>
                        <div class="band-pills">
                            <label class="band-pill {form.band === '2.4' ? 'active' : ''}">
                                <input type="radio" bind:group={form.band} value="2.4" />
                                <i class="fa-solid fa-signal"></i>
                                <span>2.4 GHz (Jangkauan Luas)</span>
                            </label>
                            <label class="band-pill {form.band === '5' ? 'active' : ''}">
                                <input type="radio" bind:group={form.band} value="5" />
                                <i class="fa-solid fa-bolt"></i>
                                <span>5 GHz (Kecepatan Tinggi)</span>
                            </label>
                        </div>
                    </div>

                    <!-- HIDDEN SSID CHECKBOX -->
                    <div class="form-group-check">
                        <label class="check-label">
                            <input type="checkbox" bind:checked={form.hidden} />
                            <span>Sembunyikan Nama Hotspot (Hidden SSID)</span>
                        </label>
                    </div>

                    <!-- SUBMIT BUTTON -->
                    <button 
                        type="button" 
                        class="btn-save" 
                        disabled={saving}
                        on:click={saveConfig}
                    >
                        <i class="fa-solid {saving ? 'fa-spinner fa-spin' : 'fa-check'}"></i>
                        <span>{saving ? 'Menerapkan Perubahan...' : 'Simpan & Terapkan'}</span>
                    </button>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .wrap {
        padding: 16px;
        max-width: 680px;
        margin: 0 auto;
        padding-bottom: 95px;
        box-sizing: border-box;
    }

    /* HEADER */
    .header {
        margin-bottom: 16px;
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
        margin-bottom: 3px;
    }

    .live-dot {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: var(--muted);
        display: inline-block;
    }

    .live-dot.dot-active {
        background: var(--lime);
        box-shadow: 0 0 6px var(--lime);
    }

    h1 {
        font-size: 1.6rem;
        font-weight: 950;
        letter-spacing: -0.03em;
        margin: 0 0 4px 0;
        color: var(--ink);
        text-transform: uppercase;
    }

    .desc {
        font-size: 0.8rem;
        color: var(--muted);
        line-height: 1.4;
        font-weight: 700;
    }

    /* TOAST */
    .toast-bar {
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 10px 14px;
        margin-bottom: 14px;
        font-size: 11px;
        font-weight: 900;
        display: flex;
        align-items: center;
        gap: 8px;
        box-shadow: 2px 2px 0 var(--line);
        animation: fadeIn 0.2s ease-out;
    }

    .toast-bar.success { background: var(--lime); color: #000; }
    .toast-bar.error { background: var(--pink); color: #fff; }
    .toast-bar.info { background: var(--cyan); color: #000; }

    @keyframes fadeIn {
        from { opacity: 0; transform: translateY(-4px); }
        to { opacity: 1; transform: translateY(0); }
    }

    /* CARD BASE */
    .card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 14px;
        box-shadow: var(--shadow);
        box-sizing: border-box;
    }

    .card-header {
        font-size: 11px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--muted);
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 12px;
        padding-bottom: 8px;
        border-bottom: 1.5px dashed var(--line);
    }

    /* HERO CARD */
    .hero-card {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 14px;
        margin-bottom: 14px;
        flex-wrap: wrap;
        border-width: 2.5px;
    }

    .hero-card.is-active {
        background: linear-gradient(135deg, rgba(85, 239, 196, 0.15), var(--paper));
        border-color: var(--lime);
    }

    .hero-left {
        flex: 1;
        min-width: 200px;
    }

    .status-badge {
        font-size: 9.5px;
        font-weight: 950;
        letter-spacing: 0.05em;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 3px 8px;
        border-radius: 6px;
        border: 1px solid var(--line);
        margin-bottom: 6px;
    }

    .badge-on { background: var(--lime); color: #000; }
    .badge-off { background: var(--panel-alt); color: var(--muted); }

    .pulse-indicator {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: currentColor;
    }

    .hero-title {
        font-size: 15px;
        font-weight: 950;
        color: var(--ink);
        margin-bottom: 3px;
    }

    .hero-sub {
        font-size: 10.5px;
        font-weight: 700;
        color: var(--muted);
    }

    .btn-toggle {
        padding: 10px 18px;
        border-radius: 10px;
        font-size: 11.5px;
        font-weight: 950;
        text-transform: uppercase;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        border: 2px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
        font-family: inherit;
        transition: all 0.12s;
    }

    .btn-toggle:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--line);
    }

    .btn-toggle:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }

    .btn-start { background: var(--lime); color: #000; }
    .btn-stop { background: var(--pink); color: #fff; }

    /* GRID LAYOUT */
    .grid-layout {
        display: flex;
        flex-direction: column;
        gap: 14px;
    }

    /* QR CARD */
    .qr-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
    }

    .qr-box {
        background: #ffffff;
        padding: 10px;
        border: 2px solid var(--line);
        border-radius: 10px;
        box-shadow: 2px 2px 0 var(--line);
        margin-bottom: 10px;
        display: inline-block;
    }

    .qr-img {
        display: block;
        width: 170px;
        height: 170px;
        image-rendering: pixelated;
        border-radius: 4px;
    }

    .qr-placeholder {
        width: 170px;
        height: 170px;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 8px;
        color: var(--muted, #888);
        font-size: 11px;
        font-weight: 800;
    }

    .qr-info {
        max-width: 320px;
    }

    .qr-ssid {
        font-size: 13.5px;
        font-weight: 950;
        color: var(--ink);
        margin-bottom: 2px;
    }

    .qr-hint {
        font-size: 10px;
        font-weight: 700;
        color: var(--muted);
        line-height: 1.35;
        margin-bottom: 8px;
    }

    .btn-copy-info {
        background: var(--panel-alt);
        border: 1.5px solid var(--line);
        border-radius: 6px;
        padding: 5px 12px;
        font-size: 10px;
        font-weight: 900;
        cursor: pointer;
        color: var(--ink);
        display: inline-flex;
        align-items: center;
        gap: 5px;
        box-shadow: 1.5px 1.5px 0 var(--line);
        font-family: inherit;
        transition: all 0.1s;
    }

    .btn-copy-info:hover {
        background: var(--paper);
        border-color: var(--yellow);
    }

    /* FORM CARD */
    .form-body {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .form-label {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
        letter-spacing: 0.04em;
    }

    .input-wrap {
        position: relative;
        display: flex;
        align-items: center;
    }

    .input-icon {
        position: absolute;
        left: 10px;
        font-size: 12px;
        color: var(--muted);
        pointer-events: none;
    }

    .form-input {
        width: 100%;
        padding: 8px 10px 8px 32px;
        border: 1.5px solid var(--line);
        border-radius: 8px;
        font-family: inherit;
        font-size: 12px;
        font-weight: 800;
        background: var(--bg);
        color: var(--ink);
        outline: none;
        box-sizing: border-box;
        transition: border-color 0.15s;
    }

    .form-input:focus {
        border-color: var(--yellow);
    }

    .btn-reveal {
        position: absolute;
        right: 8px;
        background: none;
        border: none;
        color: var(--muted);
        cursor: pointer;
        font-size: 12px;
        padding: 4px;
    }

    .btn-reveal:hover {
        color: var(--ink);
    }

    /* BAND PILLS */
    .band-pills {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 8px;
    }

    .band-pill {
        border: 1.5px solid var(--line);
        border-radius: 8px;
        padding: 8px 10px;
        background: var(--bg);
        color: var(--muted);
        font-size: 10px;
        font-weight: 900;
        display: flex;
        align-items: center;
        gap: 6px;
        cursor: pointer;
        transition: all 0.12s;
    }

    .band-pill input {
        display: none;
    }

    .band-pill.active {
        background: var(--yellow);
        color: #000;
        border-color: var(--line);
        box-shadow: 2px 2px 0 var(--line);
    }

    .form-group-check {
        padding-top: 2px;
    }

    .check-label {
        font-size: 10.5px;
        font-weight: 800;
        color: var(--ink);
        display: flex;
        align-items: center;
        gap: 6px;
        cursor: pointer;
    }

    .btn-save {
        margin-top: 4px;
        background: var(--lime);
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 10px;
        font-size: 11.5px;
        font-weight: 950;
        color: #000;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        box-shadow: 2px 2px 0 var(--line);
        font-family: inherit;
        transition: all 0.12s;
    }

    .btn-save:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--line);
    }

    .btn-save:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }

    .loading-state {
        text-align: center;
        padding: 30px;
        color: var(--muted);
        font-size: 13px;
        font-weight: 800;
        display: flex;
        flex-direction: column;
        gap: 10px;
        align-items: center;
    }

    @media (max-width: 480px) {
        .wrap {
            padding: 12px;
        }
        .hero-card {
            padding: 12px;
        }
        .btn-toggle {
            width: 100%;
            justify-content: center;
        }
        .band-pills {
            grid-template-columns: 1fr;
        }
    }
</style>

