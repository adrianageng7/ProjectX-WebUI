<script>
    import { createEventDispatcher, onMount } from 'svelte';
    const dispatch = createEventDispatcher();
    
    export let theme;
    export let color;
    export let navMode = 'dock';
    
    let isPaletteOpen = false;
    let isSettingsOpen = false;
    let isLogoutOpen = false;

    let brandTitle = 'ProjectX';
    let brandInput = 'ProjectX';

    const colors = [
        { id: 'default', label: 'Retro Pop', bg: '#0d0b10', text: '#fff' },
        { id: 'synthwave', label: 'Synthwave', bg: '#ff2d95', text: '#fff' },
        { id: 'toxic', label: 'Toxic', bg: '#39ff14', text: '#000' },
        { id: 'arctic', label: 'Arctic', bg: '#40c4ff', text: '#000' },
        { id: 'sunset', label: 'Sunset', bg: '#ff6d00', text: '#000' },
        { id: 'lavender', label: 'Lavender', bg: '#b388ff', text: '#000' }
    ];

    onMount(() => {
        brandTitle = localStorage.getItem('dashboard_brand_title') || 'ProjectX';
        brandInput = brandTitle;
    });

    function saveBrand() {
        brandTitle = brandInput.trim() || 'ProjectX';
        localStorage.setItem('dashboard_brand_title', brandTitle);
        isSettingsOpen = false;
    }

    async function proceedLogout() {
        try {
            await fetch('/api/logout');
        } catch (e) {}
        dispatch('logout');
        isLogoutOpen = false;
    }
</script>

<div class="topbar">
    <div class="topbar-brand">
        {#if navMode === 'sidebar'}
            <button class="btn-icon btn-hamburger" on:click={() => dispatch('toggleMobileSidebar')} title="Buka Menu Sidebar">
                <i class="fa-solid fa-bars"></i>
            </button>
        {/if}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <img
            src="./assets/favicon.png"
            alt="Logo"
            class="brand-icon"
            on:click={() => isSettingsOpen = true}
            title="Ubah nama header"
        />
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="brand-text" on:click={() => isSettingsOpen = true} title="Klik untuk ubah nama dashboard">
            <span class="brand-name">{brandTitle}</span>
            <span class="brand-badge"><div class="live-dot"></div> BFR BOX</span>
        </div>
    </div>
    <div class="actions">
        <!-- SETTINGS BUTTON -->
        <button class="btn-icon" on:click={() => isSettingsOpen = true} title="Setelan Dashboard">
            <i class="fa-solid fa-gear"></i>
        </button>

        <!-- PALETTE DROPDOWN -->
        <div style="position:relative;">
            <button class="btn-icon" on:click={() => isPaletteOpen = !isPaletteOpen} title="Ubah Tema Warna">
                <i class="fa-solid fa-palette"></i>
            </button>
            {#if isPaletteOpen}
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="palette-dropdown" on:click={() => isPaletteOpen = false}>
                    {#each colors as c}
                        <button
                            class="color-btn {color === c.id ? 'active' : ''}" 
                            style="background: {c.bg}; color: {c.text}"
                            on:click={() => dispatch('changeColor', c.id)}
                            title={c.label}
                        >
                            <i class="fa-solid fa-droplet"></i>
                        </button>
                    {/each}
                </div>
            {/if}
        </div>
        
        <!-- THEME TOGGLE -->
        <button class="btn-icon" on:click={() => dispatch('toggleTheme')} title="Mode Terang/Gelap">
            <i class="fa-solid {theme === 'dark' ? 'fa-sun' : 'fa-moon'}"></i>
        </button>

        <!-- LOGOUT BUTTON -->
        <button class="btn-icon btn-logout" on:click={() => isLogoutOpen = true} title="Keluar / Logout">
            <i class="fa-solid fa-right-from-bracket"></i>
        </button>
    </div>
</div>

<!-- MODAL: SETTING BRAND TITLE & NAV MODE -->
{#if isSettingsOpen}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="custom-modal" on:click={() => isSettingsOpen = false}>
        <div class="modal-card" on:click|stopPropagation>
            <div class="modal-title">Setelan Dashboard</div>
            <div class="modal-body">
                <div class="setting-field-group">
                    <label class="setting-label">Nama Header (Brand Name):</label>
                    <input
                        type="text"
                        class="modal-input"
                        bind:value={brandInput}
                        placeholder="Masukkan nama header baru..."
                    />
                </div>

                <div class="setting-field-group">
                    <label class="setting-label">Gaya Navigasi Tampilan:</label>
                    <div class="nav-mode-selector">
                        <button
                            type="button"
                            class="nav-mode-card {navMode === 'dock' ? 'active' : ''}"
                            on:click={() => dispatch('changeNavMode', 'dock')}
                        >
                            <div class="nm-icon"><i class="fa-solid fa-layer-group"></i></div>
                            <div class="nm-info">
                                <strong>Dynamic Bar</strong>
                                <span>Dock mengambang modern di bawah layar</span>
                            </div>
                        </button>
                        <button
                            type="button"
                            class="nav-mode-card {navMode === 'sidebar' ? 'active' : ''}"
                            on:click={() => dispatch('changeNavMode', 'sidebar')}
                        >
                            <div class="nm-icon"><i class="fa-solid fa-table-columns"></i></div>
                            <div class="nm-info">
                                <strong>Sidebar Menu</strong>
                                <span>Bilah navigasi samping kiri bergaya admin</span>
                            </div>
                        </button>
                    </div>
                </div>
            </div>
            <div class="modal-actions">
                <button class="btn-modal btn-cancel" on:click={() => isSettingsOpen = false}>Batal</button>
                <button class="btn-modal btn-primary" on:click={saveBrand}>Simpan</button>
            </div>
        </div>
    </div>
{/if}

<!-- MODAL: LOGOUT CONFIRMATION -->
{#if isLogoutOpen}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="custom-modal" on:click={() => isLogoutOpen = false}>
        <div class="modal-card" on:click|stopPropagation>
            <div class="modal-title modal-title-danger">Keluar Sistem</div>
            <div class="modal-body">
                Apakah Anda yakin ingin mengakhiri sesi masuk aktif Anda dan keluar dari dashboard ProjectX?
            </div>
            <div class="modal-actions">
                <button class="btn-modal btn-cancel" on:click={() => isLogoutOpen = false}>Batal</button>
                <button class="btn-modal btn-danger" on:click={proceedLogout}>Keluar Sesi</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .topbar {
        position: relative;
        height: 58px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 14px;
        background: var(--panel-bg);
        border-bottom: var(--border-width) solid var(--border-color);
        z-index: 90;
        flex-shrink: 0;
        transition: background-color 0.2s ease, border-color 0.2s ease;
    }
    
    .topbar::after {
        content: '';
        position: absolute;
        bottom: -4px;
        left: 0; right: 0;
        height: 3px;
        background: linear-gradient(90deg, var(--pink), var(--primary), var(--cyan), var(--purple));
        z-index: 91;
    }

    .topbar-brand {
        display: flex;
        align-items: center;
        gap: 10px;
        cursor: pointer;
    }

    .brand-icon {
        width: 32px;
        height: 32px;
        border-radius: 8px;
        border: var(--border-width) solid var(--border-color);
        box-shadow: var(--brutal-shadow-sm);
        object-fit: cover;
        transform: rotate(-3deg);
        transition: transform 0.15s;
    }
    .brand-icon:hover { transform: rotate(0deg) scale(1.08); }
    
    .brand-text { display: flex; flex-direction: column; gap: 1px; }
    
    .brand-name {
        font-size: 15px;
        font-weight: 950;
        color: var(--text);
        text-transform: uppercase;
        letter-spacing: -0.5px;
        line-height: 1;
    }
    
    .brand-badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        font-size: 7px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 0.8px;
        background: var(--primary);
        color: var(--primary-text);
        padding: 1px 6px;
        border: 1px solid var(--border-color);
        border-radius: 4px;
        width: fit-content;
        box-shadow: 1px 1px 0 var(--border-color);
    }
    
    .brand-badge .live-dot {
        width: 4px;
        height: 4px;
        border-radius: 50%;
        background: var(--primary-text);
        animation: blink 1.5s infinite;
    }
    @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

    .actions { display: flex; gap: 6px; }
    
    .brand-badge.engine-bfr { background: var(--primary); color: var(--primary-text); }
    
    .brand-badge.engine-loading { background: var(--purple); color: #fff; }
    
    .brand-badge.engine-none { 
        background: var(--panel-alt); 
        color: var(--ink);
        border-color: var(--line);
        box-shadow: none;
    }
    .brand-badge.engine-none .live-dot { display: none; }

    .btn-active-engine {
        background: var(--primary);
        color: var(--primary-text);
        box-shadow: 2px 2px 0 var(--border-color);
        transform: translate(-1px, -1px);
    }

    /* ENGINE SWITCHER GRID */
    .engine-grid {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }
    .engine-opt {
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 12px 16px;
        background: var(--panel-bg);
        border: 2px solid var(--line);
        border-radius: 12px;
        cursor: pointer;
        text-align: left;
        transition: all 0.2s;
        box-shadow: var(--shadow-sm);
        position: relative;
    }
    .engine-opt:hover:not(:disabled) {
        transform: translateX(4px);
    }
    .engine-opt i.fa-solid:first-child {
        font-size: 20px;
        width: 24px;
        text-align: center;
        color: var(--ink);
    }
    .engine-opt.active {
        background: var(--primary);
        border-color: var(--border-color);
        box-shadow: 3px 3px 0 var(--border-color);
    }
    .engine-opt.active i.fa-solid:first-child {
        color: var(--primary-text);
    }
    .engine-opt.stop-opt.active {
        background: var(--pink);
    }
    .engine-info {
        display: flex;
        flex-direction: column;
        gap: 2px;
        flex: 1;
    }
    .etitle {
        font-size: 14px;
        font-weight: 900;
        color: var(--text);
    }
    .active .etitle {
        color: var(--primary-text);
    }
    .stop-opt.active .etitle {
        color: #fff;
    }
    .edesc {
        font-size: 11px;
        font-weight: 700;
        color: var(--ink);
    }
    .active .edesc {
        color: var(--primary-text);
        opacity: 0.8;
    }
    .stop-opt.active .edesc {
        color: #fff;
        opacity: 0.8;
    }
    .check-icon {
        position: absolute;
        right: 16px;
        font-size: 18px;
        color: var(--primary-text);
    }
    .stop-opt.active .check-icon {
        color: #fff;
    }
    .palette-dropdown {
        position: absolute;
        top: 120%; right: 0;
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        border-radius: 12px;
        padding: 10px;
        box-shadow: var(--brutal-shadow);
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 8px;
        z-index: 100;
    }
    .color-btn {
        width: 36px;
        height: 36px;
        border-radius: 8px;
        border: var(--border-width) solid var(--border-color);
        cursor: pointer;
        transition: transform 0.1s;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .color-btn:hover { transform: scale(1.1) rotate(5deg); }
    .color-btn.active { box-shadow: 3px 3px 0 var(--border-color); }

    .btn-logout {
        color: var(--pink);
    }
    .btn-logout:hover {
        background: var(--pink);
        color: #fff;
    }

    /* MODAL */
    .custom-modal {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(0, 0, 0, 0.6);
        z-index: 1000;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 16px;
    }
    .modal-card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 16px;
        padding: 20px;
        width: 360px;
        max-width: 100%;
        box-shadow: var(--shadow);
    }
    .modal-title {
        font-size: 13px;
        font-weight: 950;
        text-transform: uppercase;
        padding: 4px 10px;
        border: 2px solid var(--line);
        border-radius: 6px;
        box-shadow: 2px 2px 0 var(--line);
        display: inline-block;
        margin-bottom: 12px;
        background: var(--primary);
        color: var(--primary-text);
        transform: rotate(-1.5deg);
    }
    .modal-title-danger {
        background: var(--pink);
        color: #fff;
    }
    .modal-body {
        font-size: 13px;
        line-height: 1.5;
        color: var(--ink);
        margin-bottom: 16px;
        font-weight: 700;
    }
    .modal-input {
        width: 100%;
        margin-top: 10px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 10px 12px;
        color: var(--ink);
        font-size: 13px;
        font-family: inherit;
        font-weight: 800;
    }
    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
    }
    .btn-modal {
        padding: 8px 16px;
        font-weight: 950;
        font-size: 12px;
        border-radius: 8px;
        border: 2px solid var(--line);
        cursor: pointer;
        box-shadow: 3px 3px 0 var(--line);
        text-transform: uppercase;
        font-family: inherit;
    }
    .btn-cancel { background: var(--panel-alt); color: var(--ink); }
    .btn-primary { background: var(--primary); color: var(--primary-text); }
    .btn-danger { background: var(--pink); color: #fff; }

    /* HAMBURGER & NAV TOGGLE STYLES */
    .btn-hamburger {
        display: none;
        margin-right: 4px;
        flex-shrink: 0;
    }
    @media (max-width: 768px) {
        .btn-hamburger {
            display: flex;
        }
    }
    .btn-nav-active {
        background: var(--primary);
        color: var(--primary-text);
        box-shadow: 2px 2px 0 var(--border-color);
    }
    .setting-field-group {
        margin-top: 14px;
        display: flex;
        flex-direction: column;
        gap: 6px;
    }
    .setting-label {
        font-size: 11px;
        font-weight: 800;
        text-transform: uppercase;
        color: var(--text-muted);
    }
    .nav-mode-selector {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 8px;
    }
    @media (max-width: 480px) {
        .nav-mode-selector {
            grid-template-columns: 1fr;
        }
    }
    .nav-mode-card {
        background: var(--panel-alt);
        border: 2px solid var(--border-color);
        border-radius: 10px;
        padding: 8px 10px;
        display: flex;
        align-items: center;
        gap: 8px;
        cursor: pointer;
        text-align: left;
        color: var(--text);
        box-shadow: 2px 2px 0 var(--border-color);
        transition: all 0.15s;
        font-family: inherit;
    }
    .nav-mode-card:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--border-color);
    }
    .nav-mode-card.active {
        border-color: var(--primary);
        background: rgba(255, 225, 74, 0.14);
        box-shadow: 3px 3px 0 var(--primary);
    }
    .nm-icon {
        font-size: 16px;
        color: var(--primary);
    }
    .nm-info {
        display: flex;
        flex-direction: column;
    }
    .nm-info strong {
        font-size: 11.5px;
        font-weight: 800;
    }
    .nm-info span {
        font-size: 9.5px;
        color: var(--text-muted);
        line-height: 1.2;
    }
</style>
