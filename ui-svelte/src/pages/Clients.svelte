<script>
    import { onMount, onDestroy } from 'svelte';
    import { router } from 'tinro';

    let clients = [];
    let loading = true;
    let timer = null;
    let searchQuery = '';

    let modal = {
        show: false,
        type: '', // 'block' | 'unblock' | 'limit' | 'rename' | 'alert'
        title: '',
        message: '',
        client: null,
        limitSpeed: 5,
        newAlias: ''
    };

    $: filteredClients = clients.filter(c => {
        if (!searchQuery.trim()) return true;
        const q = searchQuery.toLowerCase();
        return (c.hostname && c.hostname.toLowerCase().includes(q)) ||
               (c.ip && c.ip.toLowerCase().includes(q)) ||
               (c.mac && c.mac.toLowerCase().includes(q)) ||
               (c.vendor && c.vendor.toLowerCase().includes(q));
    });

    $: blockedCount = clients.filter(c => c.is_blocked).length;
    $: limitedCount = clients.filter(c => c.limit_speed).length;

    async function fetchClients() {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 5000);
        try {
            const res = await fetch('/api/clients?_=' + Date.now(), { signal: controller.signal });
            clearTimeout(timeoutId);
            if (!res.ok) throw new Error('HTTP ' + res.status);
            const data = await res.json();
            if (data && Array.isArray(data.clients)) {
                clients = data.clients;
            } else if (Array.isArray(data)) {
                clients = data;
            } else {
                clients = [];
            }
        } catch (e) {
            console.error('Error fetching clients:', e);
        } finally {
            clearTimeout(timeoutId);
            loading = false;
        }
    }

    function openBlockModal(c) {
        modal = {
            show: true,
            type: 'block',
            title: 'Blokir Perangkat',
            message: `Apakah Anda yakin ingin memblokir akses internet untuk ${c.hostname || c.ip} (${c.ip})?`,
            client: c,
            limitSpeed: 5,
            newAlias: ''
        };
    }

    function openUnblockModal(c) {
        modal = {
            show: true,
            type: 'unblock',
            title: 'Buka Blokir Perangkat',
            message: `Izinkan kembali akses internet untuk ${c.hostname || c.ip} (${c.ip})?`,
            client: c,
            limitSpeed: 5,
            newAlias: ''
        };
    }

    function openLimitModal(c) {
        let cur = 5;
        if (c.limit_speed) {
            const parsed = parseInt(c.limit_speed);
            if (!isNaN(parsed) && parsed > 0) cur = parsed;
        }
        modal = {
            show: true,
            type: 'limit',
            title: 'Limit Kecepatan',
            message: '',
            client: c,
            limitSpeed: cur,
            newAlias: ''
        };
    }

    function openRenameModal(c) {
        modal = {
            show: true,
            type: 'rename',
            title: 'Ubah Nama Perangkat',
            message: '',
            client: c,
            limitSpeed: 5,
            newAlias: c.hostname || ''
        };
    }

    async function doBlock() {
        const c = modal.client;
        modal.show = false;
        if (!c) return;
        try {
            const url = `/api/clients/block?ip=${encodeURIComponent(c.ip || '')}&mac=${encodeURIComponent(c.mac || '')}`;
            const res = await fetch(url);
            const data = await res.json();
            if (data.status === 'success') {
                showAlert('Sukses', `Perangkat ${c.hostname || c.ip} berhasil diblokir.`);
            } else {
                showAlert('Peringatan', data.message || 'Gagal memblokir client.');
            }
            await fetchClients();
        } catch (e) {
            showAlert('Error', 'Gagal memproses permintaan blokir.');
        }
    }

    async function doUnblock() {
        const c = modal.client;
        modal.show = false;
        if (!c) return;
        try {
            const url = `/api/clients/unblock?ip=${encodeURIComponent(c.ip || '')}&mac=${encodeURIComponent(c.mac || '')}`;
            const res = await fetch(url);
            const data = await res.json();
            if (data.status === 'success') {
                showAlert('Sukses', `Blokir untuk ${c.hostname || c.ip} berhasil dibuka.`);
            } else {
                showAlert('Peringatan', data.message || 'Gagal membuka blokir.');
            }
            await fetchClients();
        } catch (e) {
            showAlert('Error', 'Gagal memproses buka blokir.');
        }
    }

    async function doLimit() {
        const c = modal.client;
        const speed = modal.limitSpeed + 'Mbit';
        modal.show = false;
        if (!c) return;
        try {
            const url = `/api/clients/limit?ip=${encodeURIComponent(c.ip || '')}&mac=${encodeURIComponent(c.mac || '')}&interface=${encodeURIComponent(c.interface || '')}&speed=${encodeURIComponent(speed)}`;
            const res = await fetch(url);
            const data = await res.json();
            if (data.status === 'success') {
                showAlert('Sukses', `Kecepatan ${c.hostname || c.ip} dibatasi ke ${speed}.`);
            } else {
                showAlert('Peringatan', data.message || 'Gagal mengatur limit kecepatan.');
            }
            await fetchClients();
        } catch (e) {
            showAlert('Error', 'Gagal mengatur limit.');
        }
    }

    async function doRemoveLimit() {
        const c = modal.client;
        modal.show = false;
        if (!c) return;
        try {
            const url = `/api/clients/unlimit?ip=${encodeURIComponent(c.ip || '')}&mac=${encodeURIComponent(c.mac || '')}&interface=${encodeURIComponent(c.interface || '')}`;
            const res = await fetch(url);
            const data = await res.json();
            if (data.status === 'success') {
                showAlert('Sukses', `Limit kecepatan ${c.hostname || c.ip} telah dihapus.`);
            } else {
                showAlert('Peringatan', data.message || 'Gagal menghapus limit.');
            }
            await fetchClients();
        } catch (e) {
            showAlert('Error', 'Gagal menghapus limit.');
        }
    }

    async function doSaveAlias() {
        const c = modal.client;
        const alias = modal.newAlias.trim();
        modal.show = false;
        if (!c) return;
        try {
            const url = `/api/clients/alias?ip=${encodeURIComponent(c.ip || '')}&mac=${encodeURIComponent(c.mac || '')}&alias=${encodeURIComponent(alias)}`;
            const res = await fetch(url);
            const data = await res.json();
            if (data.status === 'success') {
                showAlert('Sukses', `Nama perangkat berhasil disimpan.`);
            } else {
                showAlert('Peringatan', data.message || 'Gagal menyimpan nama alias.');
            }
            await fetchClients();
        } catch (e) {
            showAlert('Error', 'Gagal menyimpan alias.');
        }
    }

    function showAlert(title, message) {
        modal = {
            show: true,
            type: 'alert',
            title,
            message,
            client: null,
            limitSpeed: 5,
            newAlias: ''
        };
    }

    onMount(() => {
        fetchClients();
        timer = setInterval(fetchClients, 8000);
    });

    onDestroy(() => {
        if (timer) clearInterval(timer);
    });
</script>

<div class="wrap">
    <div class="header">
        <div class="header-left">
            <div class="eyebrow"><span class="live-dot"></span> Hotspot & Jaringan</div>
            <h1>Clients</h1>
            <div class="desc">Perangkat yang terhubung ke tethering atau jaringan lokal</div>
        </div>
        <div class="header-actions">
            <button class="btn btn-ttl" on:click={() => router.goto('/ttl')} title="Bypass TTL Hotspot / Tethering">
                <i class="fa-solid fa-shield-halved" style="color:var(--pink);"></i> TTL
            </button>
            <button class="btn btn-watchdog" on:click={() => router.goto('/watchdog')} title="Connection Watchdog (Anti-RTO)">
                <i class="fa-solid fa-dog" style="color:var(--yellow);"></i> Watchdog
            </button>
            <button class="btn btn-refresh" on:click={fetchClients} title="Pindai Ulang">
                <i class="fa-solid fa-rotate-right {loading ? 'fa-spin' : ''}"></i>
            </button>
        </div>
    </div>

    <!-- STATS STRIP -->
    <div class="stats-row">
        <div class="stat-pill">
            <i class="fa-solid fa-users" style="color:var(--cyan);"></i>
            <span><strong>{clients.length}</strong> Terhubung</span>
        </div>
        {#if blockedCount > 0}
            <div class="stat-pill blocked-pill">
                <i class="fa-solid fa-ban" style="color:var(--pink);"></i>
                <span><strong>{blockedCount}</strong> Diblokir</span>
            </div>
        {/if}
        {#if limitedCount > 0}
            <div class="stat-pill limited-pill">
                <i class="fa-solid fa-gauge-high" style="color:var(--yellow);"></i>
                <span><strong>{limitedCount}</strong> Dilimit</span>
            </div>
        {/if}
    </div>

    <div class="card">
        <div class="card-header">
            <div class="card-title">
                <i class="fa-solid fa-laptop-mobile"></i> Connected Devices
            </div>
            {#if clients.length > 2}
                <div class="search-box">
                    <i class="fa-solid fa-magnifying-glass"></i>
                    <input type="text" placeholder="Cari IP / Nama..." bind:value={searchQuery}>
                </div>
            {/if}
        </div>

        <div class="client-list">
            {#if loading && clients.length === 0}
                <div class="empty-state">
                    <i class="fa-solid fa-spinner fa-spin empty-icon"></i>
                    <div class="empty-title">Memindai Jaringan...</div>
                    <div class="empty-sub">Mendeteksi perangkat dari DHCP leases & ARP cache</div>
                </div>
            {:else if clients.length === 0}
                <div class="empty-state">
                    <i class="fa-solid fa-wifi-slash empty-icon"></i>
                    <div class="empty-title">Tidak Ada Perangkat Terhubung</div>
                    <div class="empty-sub">
                        Pastikan Hotspot Wi-Fi atau USB Tethering HP aktif.<br>
                        Perangkat yang tersambung akan otomatis muncul di sini.
                    </div>
                    <button class="btn-rescan" on:click={fetchClients}>
                        <i class="fa-solid fa-rotate-right"></i> Pindai Ulang
                    </button>
                </div>
            {:else if filteredClients.length === 0}
                <div class="empty-state">
                    <i class="fa-solid fa-filter-circle-xmark empty-icon"></i>
                    <div class="empty-title">Tidak Ditemukan</div>
                    <div class="empty-sub">Tidak ada perangkat yang cocok dengan pencarian "{searchQuery}"</div>
                </div>
            {:else}
                {#each filteredClients as c (c.mac || c.ip)}
                    {@const isWifi = c.interface && (c.interface.includes('wlan') || c.interface.includes('ap') || c.interface.includes('softap'))}
                    <div class="client-item {c.is_blocked ? 'is-blocked' : ''}">
                        <div class="client-icon" style="background: {c.is_blocked ? 'var(--pink)' : isWifi ? 'var(--cyan)' : 'var(--purple)'}">
                            <i class="fa-solid {c.is_blocked ? 'fa-ban' : isWifi ? 'fa-wifi' : 'fa-network-wired'}"></i>
                        </div>

                        <div class="client-info">
                            <div class="client-title-row">
                                <span class="client-name">{c.hostname || 'Unknown Device'}</span>
                                <button class="btn-rename" on:click={() => openRenameModal(c)} title="Beri nama panggilan">
                                    <i class="fa-solid fa-pen"></i>
                                </button>
                                {#if c.is_blocked}
                                    <span class="badge badge-blocked"><i class="fa-solid fa-lock"></i> BLOCKED</span>
                                {/if}
                                {#if c.limit_speed}
                                    <span class="badge badge-limited"><i class="fa-solid fa-gauge"></i> {c.limit_speed}</span>
                                {/if}
                                {#if c.vendor}
                                    <span class="badge badge-vendor">{c.vendor}</span>
                                {/if}
                            </div>

                            <div class="client-meta">
                                <span class="client-ip"><i class="fa-solid fa-location-dot"></i> {c.ip || '-'}</span>
                                <span class="client-mac"><i class="fa-solid fa-fingerprint"></i> {c.mac || '-'}</span>
                                <span class="client-iface"><i class="fa-solid fa-ethernet"></i> {c.interface || 'wlan0'}</span>
                                {#if c.expires_in}
                                    <span class="client-exp"><i class="fa-solid fa-clock"></i> Lease: {c.expires_in}</span>
                                {/if}
                            </div>
                        </div>

                        <div class="client-actions">
                            <button
                                class="btn-action btn-limit {c.limit_speed ? 'active-limit' : ''}"
                                on:click={() => openLimitModal(c)}
                                title="Batas Kecepatan Internet"
                            >
                                <i class="fa-solid fa-gauge-high"></i>
                                <span>{c.limit_speed ? c.limit_speed : 'Limit'}</span>
                            </button>

                            {#if c.is_blocked}
                                <button
                                    class="btn-action btn-unblock"
                                    on:click={() => openUnblockModal(c)}
                                    title="Buka Blokir Internet"
                                >
                                    <i class="fa-solid fa-lock-open"></i>
                                    <span>Buka</span>
                                </button>
                            {:else}
                                <button
                                    class="btn-action btn-block"
                                    on:click={() => openBlockModal(c)}
                                    title="Blokir Akses Internet"
                                >
                                    <i class="fa-solid fa-ban"></i>
                                    <span>Blokir</span>
                                </button>
                            {/if}
                        </div>
                    </div>
                {/each}
            {/if}
        </div>
    </div>

    <!-- MODAL -->
    {#if modal.show}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="custom-modal" on:click={() => modal.show = false}>
            <div class="modal-card" on:click|stopPropagation>
                <div class="modal-title">
                    {modal.title}
                </div>

                <div class="modal-body">
                    {#if modal.type === 'block'}
                        <p class="modal-text">{modal.message}</p>
                        <div class="modal-warning">
                            <i class="fa-solid fa-triangle-exclamation"></i>
                            Perangkat ini tidak akan bisa mengakses internet melalui hotspot atau tethering ini.
                        </div>
                    {:else if modal.type === 'unblock'}
                        <p class="modal-text">{modal.message}</p>
                    {:else if modal.type === 'limit'}
                        <p class="modal-text">
                            Atur batas kecepatan bandwidth untuk <strong>{modal.client?.hostname || modal.client?.ip}</strong>:
                        </p>
                        <div class="slider-box">
                            <div class="slider-display">{modal.limitSpeed} Mbit/s</div>
                            <input type="range" min="1" max="50" bind:value={modal.limitSpeed}>
                            <div class="slider-labels">
                                <span>1 Mbit</span>
                                <span>25 Mbit</span>
                                <span>50 Mbit</span>
                            </div>
                        </div>

                        <!-- SPEED PRESETS -->
                        <div class="preset-label">Pilihan Cepat:</div>
                        <div class="preset-buttons">
                            {#each [1, 2, 5, 10, 20] as sp}
                                <button
                                    type="button"
                                    class="btn-preset {modal.limitSpeed === sp ? 'active' : ''}"
                                    on:click={() => modal.limitSpeed = sp}
                                >
                                    {sp}M
                                </button>
                            {/each}
                        </div>
                    {:else if modal.type === 'rename'}
                        <p class="modal-text">Masukkan nama alias untuk perangkat ({modal.client?.mac || modal.client?.ip}):</p>
                        <input
                            type="text"
                            class="input-rename"
                            bind:value={modal.newAlias}
                            placeholder="Contoh: Laptop Kerja, HP Ibu..."
                        >
                    {:else}
                        <p class="modal-text">{modal.message}</p>
                    {/if}
                </div>

                <div class="modal-actions">
                    <button class="btn-modal btn-cancel" on:click={() => modal.show = false}>
                        {modal.type === 'alert' ? 'Tutup' : 'Batal'}
                    </button>

                    {#if modal.type === 'block'}
                        <button class="btn-modal btn-danger" on:click={doBlock}>
                            <i class="fa-solid fa-ban"></i> Blokir Sekarang
                        </button>
                    {:else if modal.type === 'unblock'}
                        <button class="btn-modal btn-success" on:click={doUnblock}>
                            <i class="fa-solid fa-lock-open"></i> Buka Blokir
                        </button>
                    {:else if modal.type === 'limit'}
                        {#if modal.client?.limit_speed}
                            <button class="btn-modal btn-clear" on:click={doRemoveLimit}>
                                Hapus Limit
                            </button>
                        {/if}
                        <button class="btn-modal btn-primary" on:click={doLimit}>
                            Terapkan
                        </button>
                    {:else if modal.type === 'rename'}
                        <button class="btn-modal btn-primary" on:click={doSaveAlias}>
                            Simpan
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .wrap { max-width: 900px; margin: 0 auto; padding: 16px; padding-bottom: 90px; }
    .header { margin-bottom: 12px; display: flex; justify-content: space-between; align-items: flex-end; }
    .eyebrow { font-size: 10px; font-weight: 900; text-transform: uppercase; color: var(--lime); display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
    .live-dot { width: 8px; height: 8px; background: var(--lime); border-radius: 50%; animation: blink 1.5s infinite; }
    @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
    h1 { font-size: 28px; font-weight: 950; letter-spacing: -1.5px; text-transform: uppercase; line-height: 1; }
    .desc { font-size: 12px; font-weight: 700; color: var(--muted); margin-top: 6px; }

    .header-actions { display: flex; gap: 8px; align-items: center; }
    .btn-ttl { padding: 8px 14px; background: var(--paper); border: 2px solid var(--line); border-radius: 8px; cursor: pointer; color: var(--ink); box-shadow: 2px 2px 0 var(--line); font-size: 11px; font-weight: 900; text-transform: uppercase; display: flex; align-items: center; gap: 6px; transition: .15s; }
    .btn-ttl:hover { background: var(--pink); color: #fff; transform: translate(-1px, -1px); }
    .btn-watchdog { padding: 8px 14px; background: var(--paper); border: 2px solid var(--line); border-radius: 8px; cursor: pointer; color: var(--ink); box-shadow: 2px 2px 0 var(--line); font-size: 11px; font-weight: 900; text-transform: uppercase; display: flex; align-items: center; gap: 6px; transition: .15s; }
    .btn-watchdog:hover { background: var(--yellow); color: #000; transform: translate(-1px, -1px); }
    .btn-refresh { padding: 8px 12px; background: var(--paper); border: 2px solid var(--line); border-radius: 8px; cursor: pointer; color: var(--ink); box-shadow: 2px 2px 0 var(--line); transition: .15s; }
    .btn-refresh:hover { background: var(--yellow); color: #000; transform: translate(-1px, -1px); }

    /* STATS STRIP */
    .stats-row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
    .stat-pill { display: flex; align-items: center; gap: 6px; padding: 6px 12px; background: var(--panel-alt); border: 2px solid var(--line); border-radius: 8px; font-size: 11px; font-weight: 800; box-shadow: 2px 2px 0 var(--line); }
    .blocked-pill { border-color: var(--pink); color: var(--pink); }
    .limited-pill { border-color: var(--yellow); color: var(--yellow); }

    .card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 16px; box-shadow: var(--shadow); margin-bottom: 16px; }
    .card-header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px dashed var(--line); padding-bottom: 12px; margin-bottom: 14px; gap: 10px; flex-wrap: wrap; }
    .card-title { font-size: 13px; font-weight: 900; text-transform: uppercase; display: flex; align-items: center; gap: 8px; }

    .search-box { display: flex; align-items: center; gap: 8px; background: var(--panel-alt); padding: 4px 10px; border: 2px solid var(--line); border-radius: 8px; font-size: 11px; }
    .search-box input { background: transparent; border: none; color: var(--ink); font-weight: 700; font-size: 11px; outline: none; width: 130px; }

    .client-list { display: flex; flex-direction: column; gap: 10px; }
    .client-item { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 12px; padding: 12px; background: var(--panel-alt); border: 2px solid var(--line); border-radius: 12px; box-shadow: 2px 2px 0 var(--line); transition: .15s; }
    .client-item:hover { transform: translate(-2px, -2px); box-shadow: 4px 4px 0 var(--line); }
    .client-item.is-blocked { border-color: var(--pink); background: rgba(255, 92, 168, 0.08); }

    .client-icon { width: 40px; height: 40px; display: grid; place-items: center; color: #000; border: 2px solid var(--line); border-radius: 10px; font-size: 17px; box-shadow: 2px 2px 0 var(--line); flex-shrink: 0; }
    .client-info { display: flex; flex-direction: column; gap: 4px; overflow: hidden; }

    .client-title-row { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
    .client-name { font-size: 13px; font-weight: 950; word-break: break-all; }
    .btn-rename { background: none; border: none; color: var(--muted); cursor: pointer; padding: 2px 4px; font-size: 10px; border-radius: 4px; }
    .btn-rename:hover { color: var(--yellow); }

    .badge { font-size: 9px; font-weight: 900; padding: 2px 6px; border-radius: 4px; border: 1.5px solid var(--line); text-transform: uppercase; }
    .badge-blocked { background: var(--pink); color: #fff; }
    .badge-limited { background: var(--yellow); color: #000; }
    .badge-vendor { background: var(--cyan); color: #000; }

    .client-meta { font-size: 10px; font-weight: 700; color: var(--muted); display: flex; flex-wrap: wrap; gap: 8px; }
    .client-ip { color: var(--cyan); font-family: monospace; font-weight: 900; }
    .client-mac { color: var(--pink); font-family: monospace; }
    .client-iface { color: var(--lime); }
    .client-exp { color: var(--yellow); }

    .client-actions { display: flex; gap: 6px; }
    .btn-action { padding: 6px 10px; border: 2px solid var(--line); border-radius: 8px; font-size: 11px; font-weight: 950; text-transform: uppercase; cursor: pointer; box-shadow: 2px 2px 0 var(--line); display: flex; align-items: center; gap: 4px; transition: .1s; }
    .btn-action:hover { transform: translate(-1px, -1px); box-shadow: 3px 3px 0 var(--line); }
    .btn-action:active { transform: translate(1px, 1px); box-shadow: none; }

    .btn-limit { background: var(--paper); color: var(--ink); border-color: var(--yellow); }
    .btn-limit.active-limit { background: var(--yellow); color: #000; }
    .btn-block { background: var(--pink); color: #fff; }
    .btn-unblock { background: var(--lime); color: #000; }

    /* EMPTY STATE */
    .empty-state { text-align: center; padding: 36px 16px; color: var(--muted); font-weight: 700; display: flex; flex-direction: column; align-items: center; gap: 8px; }
    .empty-icon { font-size: 38px; color: var(--muted); margin-bottom: 6px; }
    .empty-title { font-size: 15px; font-weight: 950; color: var(--ink); text-transform: uppercase; }
    .empty-sub { font-size: 11px; line-height: 1.5; color: var(--muted); max-width: 380px; }
    .btn-rescan { margin-top: 10px; padding: 8px 16px; background: var(--yellow); color: #000; border: 2px solid var(--line); border-radius: 8px; font-size: 11px; font-weight: 950; text-transform: uppercase; cursor: pointer; box-shadow: 3px 3px 0 var(--line); }

    /* SLIDER & PRESETS */
    .slider-box { margin-top: 10px; background: var(--panel-alt); padding: 12px; border-radius: 10px; border: 2px dashed var(--line); }
    .slider-display { text-align: center; font-size: 20px; font-weight: 950; color: var(--yellow); margin-bottom: 6px; }
    input[type="range"] { width: 100%; accent-color: var(--yellow); }
    .slider-labels { display: flex; justify-content: space-between; font-size: 9px; font-weight: 800; color: var(--muted); margin-top: 4px; }

    .preset-label { font-size: 10px; font-weight: 800; color: var(--muted); margin-top: 12px; margin-bottom: 6px; text-transform: uppercase; }
    .preset-buttons { display: flex; gap: 6px; }
    .btn-preset { flex: 1; padding: 6px; background: var(--paper); border: 2px solid var(--line); border-radius: 6px; font-weight: 900; font-size: 11px; color: var(--ink); cursor: pointer; }
    .btn-preset.active { background: var(--yellow); color: #000; }

    .input-rename { width: 100%; padding: 10px 12px; border: 2px solid var(--line); border-radius: 8px; background: var(--panel-alt); color: var(--ink); font-weight: 800; font-size: 13px; margin-top: 8px; box-sizing: border-box; }

    /* MODAL */
    .custom-modal { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.65); z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 16px; }
    .modal-card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 20px; width: 380px; max-width: 100%; box-shadow: var(--shadow); }
    .modal-title { font-size: 13px; font-weight: 950; text-transform: uppercase; padding: 4px 10px; border: 2px solid var(--line); border-radius: 6px; box-shadow: 2px 2px 0 var(--line); display: inline-block; margin-bottom: 12px; background: var(--primary); color: #000; }
    .modal-body { font-size: 13px; line-height: 1.5; color: var(--ink); margin-bottom: 16px; font-weight: 700; }
    .modal-text { margin: 0 0 8px 0; }
    .modal-warning { background: rgba(255, 92, 168, 0.15); border: 2px dashed var(--pink); border-radius: 8px; padding: 8px 10px; font-size: 11px; color: var(--pink); display: flex; gap: 8px; align-items: flex-start; margin-top: 8px; }
    .modal-actions { display: flex; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
    .btn-modal { padding: 8px 14px; font-weight: 950; font-size: 11px; border-radius: 8px; border: 2px solid var(--line); cursor: pointer; box-shadow: 3px 3px 0 var(--line); text-transform: uppercase; transition: .1s; }
    .btn-modal:hover { transform: translate(-1px, -1px); box-shadow: 4px 4px 0 var(--line); }
    .btn-modal:active { transform: translate(1px, 1px); box-shadow: none; }
    .btn-cancel { background: var(--panel-alt); color: var(--ink); }
    .btn-danger { background: var(--pink); color: #fff; }
    .btn-success { background: var(--lime); color: #000; }
    .btn-clear { background: var(--orange); color: #000; }
    .btn-primary { background: var(--cyan); color: #000; }

    @media(max-width: 600px) {
        .client-item { grid-template-columns: auto 1fr; }
        .client-actions { grid-column: 1 / -1; justify-content: flex-end; padding-top: 6px; border-top: 1px dashed var(--line); }
    }
</style>
