<script>
    import { createEventDispatcher, onMount } from 'svelte';
    import { router } from 'tinro';
    const dispatch = createEventDispatcher();
    
    export let activeDrawer;

    const DEFAULT_CONFIG_PATH = '/data/adb/box/clash/config.yaml';
    const DEFAULT_ACCOUNT_PATH = '/data/adb/box/clash/proxy_provider';

    let configPath = DEFAULT_CONFIG_PATH;
    let configLabel = 'Edit config.yaml';
    let accountPath = DEFAULT_ACCOUNT_PATH;
    let accountLabel = 'Edit Akun';

    let editModal = {
        show: false,
        key: '', // 'config' | 'account'
        title: '',
        label: '',
        path: ''
    };

    function loadShortcuts() {
        configPath = localStorage.getItem('bfr_config_path') || DEFAULT_CONFIG_PATH;
        configLabel = localStorage.getItem('bfr_config_label') || '';
        if (!configLabel) {
            const parts = configPath.split('/');
            const filename = parts[parts.length - 1] || 'config.yaml';
            configLabel = `Edit ${filename}`;
        }

        accountPath = localStorage.getItem('bfr_account_path') || DEFAULT_ACCOUNT_PATH;
        accountLabel = localStorage.getItem('bfr_account_label') || 'Edit Akun';
    }

    onMount(() => {
        loadShortcuts();
        window.addEventListener('storage', loadShortcuts);
        return () => window.removeEventListener('storage', loadShortcuts);
    });

    function isFilePath(path) {
        if (!path) return false;
        const lastPart = path.split('/').pop() || '';
        return lastPart.includes('.') && !path.endsWith('/');
    }

    function openShortcut(path) {
        if (isFilePath(path)) {
            navigate(`/filemanager?edit=${encodeURIComponent(path)}`);
        } else {
            navigate(`/filemanager?dir=${encodeURIComponent(path)}`);
        }
    }

    function openEditModal(key, e) {
        if (e) e.stopPropagation();
        if (key === 'config') {
            editModal = {
                show: true,
                key: 'config',
                title: 'Kustomisasi Shortcut Config',
                label: configLabel,
                path: configPath
            };
        } else {
            editModal = {
                show: true,
                key: 'account',
                title: 'Kustomisasi Shortcut Akun',
                label: accountLabel,
                path: accountPath
            };
        }
    }

    function saveShortcut() {
        if (editModal.key === 'config') {
            configPath = editModal.path.trim() || DEFAULT_CONFIG_PATH;
            configLabel = editModal.label.trim();
            if (!configLabel) {
                const parts = configPath.split('/');
                const filename = parts[parts.length - 1] || 'config.yaml';
                configLabel = `Edit ${filename}`;
            }
            localStorage.setItem('bfr_config_path', configPath);
            localStorage.setItem('bfr_config_label', configLabel);
        } else {
            accountPath = editModal.path.trim() || DEFAULT_ACCOUNT_PATH;
            accountLabel = editModal.label.trim() || 'Edit Akun';
            localStorage.setItem('bfr_account_path', accountPath);
            localStorage.setItem('bfr_account_label', accountLabel);
        }
        editModal.show = false;
        window.dispatchEvent(new Event('storage'));
    }

    function resetShortcut() {
        if (editModal.key === 'config') {
            localStorage.removeItem('bfr_config_path');
            localStorage.removeItem('bfr_config_label');
            configPath = DEFAULT_CONFIG_PATH;
            configLabel = 'Edit config.yaml';
        } else {
            localStorage.removeItem('bfr_account_path');
            localStorage.removeItem('bfr_account_label');
            accountPath = DEFAULT_ACCOUNT_PATH;
            accountLabel = 'Edit Akun';
        }
        editModal.show = false;
        window.dispatchEvent(new Event('storage'));
    }

    function navigate(url) {
        router.goto(url);
        dispatch('close');
    }
</script>

<!-- STATS DRAWER (menu-status) -->
<div class="drawer-panel {activeDrawer === 'menu-status' ? 'active' : ''}">
    <div class="drawer-header">
        <span class="drawer-title"><i class="fa-solid fa-chart-pie" style="color:var(--pink);"></i> Status & Charts</span>
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <i class="fa-solid fa-xmark drawer-close" on:click={() => dispatch('close')}></i>
    </div>
    <div class="menu-grid">
        <button class="menu-link" on:click={() => navigate('/vnstat')}>
            <i class="fa-solid fa-chart-line" style="color:var(--cyan);"></i> Traffic Monitor
        </button>
        <button class="menu-link" on:click={() => navigate('/clashboard')}>
            <i class="fa-solid fa-shield-halved" style="color:var(--yellow);"></i> Clashboard
        </button>
    </div>
</div>

<!-- NETWORK DRAWER (menu-net) -->
<div class="drawer-panel {activeDrawer === 'menu-net' ? 'active' : ''}">
    <div class="drawer-header">
        <span class="drawer-title"><i class="fa-solid fa-wifi" style="color:var(--lime);"></i> Network & Wi-Fi</span>
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <i class="fa-solid fa-xmark drawer-close" on:click={() => dispatch('close')}></i>
    </div>
    <div class="menu-grid">
        <button class="menu-link" on:click={() => navigate('/hotspot')}>
            <i class="fa-solid fa-wifi" style="color:var(--yellow);"></i> Wi-Fi Hotspot
        </button>
        <button class="menu-link" on:click={() => navigate('/clients')}>
            <i class="fa-solid fa-users" style="color:var(--lime);"></i> Connected Devices
        </button>
        <button class="menu-link" on:click={() => navigate('/ttl')}>
            <i class="fa-solid fa-shield-halved" style="color:var(--pink);"></i> TTL & Hop Limit
        </button>
        <button class="menu-link" on:click={() => navigate('/watchdog')}>
            <i class="fa-solid fa-dog" style="color:var(--yellow);"></i> Connection Watchdog
        </button>
        <button class="menu-link" on:click={() => navigate('/speedtest')}>
            <i class="fa-solid fa-gauge-high" style="color:var(--cyan);"></i> Speedtest Internet
        </button>
    </div>
</div>

<!-- BOX DRAWER (menu-box) -->
<div class="drawer-panel {activeDrawer === 'menu-box' ? 'active' : ''}">
    <div class="drawer-header">
        <span class="drawer-title"><i class="fa-solid fa-box-open" style="color:var(--yellow);"></i> Box Tunneling (BFR)</span>
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <i class="fa-solid fa-xmark drawer-close" on:click={() => dispatch('close')}></i>
    </div>
    <div class="menu-grid">

        <button class="menu-link" on:click={() => navigate('/bfr/control')}>
            <i class="fa-solid fa-box" style="color:var(--lime);"></i> BFR Control
        </button>
        <button class="menu-link" on:click={() => navigate('/bfr/settings')}>
            <i class="fa-solid fa-sliders" style="color:var(--cyan);"></i> Box Settings
        </button>
        <button class="menu-link" on:click={() => navigate('/clash-generator')}>
            <i class="fa-solid fa-server" style="color:var(--accent);"></i> Kelola Akun Proxy
        </button>
        <div class="menu-item-wrap">
            <button class="menu-link" on:click={() => openShortcut(configPath)}>
                <i class="fa-solid fa-file-signature" style="color:var(--yellow);"></i>
                <span class="menu-label-text">{configLabel}</span>
            </button>
            <button class="btn-edit-shortcut" title="Kustomisasi Path Config" on:click={(e) => openEditModal('config', e)}>
                <i class="fa-solid fa-gear"></i>
            </button>
        </div>

        <div class="menu-item-wrap">
            <button class="menu-link" on:click={() => openShortcut(accountPath)}>
                <i class="fa-solid fa-user-gear" style="color:var(--purple);"></i>
                <span class="menu-label-text">{accountLabel}</span>
            </button>
            <button class="btn-edit-shortcut" title="Kustomisasi Path Akun" on:click={(e) => openEditModal('account', e)}>
                <i class="fa-solid fa-gear"></i>
            </button>
        </div>
    </div>
</div>




{#if editModal.show}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="shortcut-modal-overlay" on:click={() => editModal.show = false}>
        <div class="shortcut-modal" on:click|stopPropagation>
            <div class="shortcut-modal-header">
                <span class="shortcut-modal-title"><i class="fa-solid fa-gear" style="color:var(--yellow);"></i> {editModal.title}</span>
                <i class="fa-solid fa-xmark shortcut-close" on:click={() => editModal.show = false}></i>
            </div>
            <div class="shortcut-modal-body">
                <div class="modal-form-group">
                    <label>Nama Tombol</label>
                    <input type="text" class="modal-input" bind:value={editModal.label} placeholder="Misal: Edit config.yaml / Edit Akun" />
                </div>
                <div class="modal-form-group">
                    <label>Path Tujuan (File atau Folder)</label>
                    <input type="text" class="modal-input" bind:value={editModal.path} placeholder="/data/adb/box/..." />
                    <div class="path-hint">
                        {#if isFilePath(editModal.path)}
                            <span class="badge-type file"><i class="fa-solid fa-file-code"></i> File (Editor Langsung)</span>
                        {:else}
                            <span class="badge-type folder"><i class="fa-solid fa-folder-open"></i> Folder (File Manager)</span>
                        {/if}
                    </div>
                </div>
            </div>
            <div class="shortcut-modal-actions">
                <button type="button" class="btn-modal-reset" on:click={resetShortcut}>
                    <i class="fa-solid fa-rotate-left"></i> Reset
                </button>
                <button type="button" class="btn-modal-save" on:click={saveShortcut}>
                    <i class="fa-solid fa-check"></i> Simpan
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .drawer-panel {
        position: fixed;
        bottom: 80px; left: 15px; right: 15px;
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        border-radius: 16px; padding: 16px;
        box-shadow: var(--brutal-shadow);
        z-index: 100;
        transform: translateY(120%); opacity: 0; pointer-events: none;
        transition: transform 0.25s cubic-bezier(0.175, 0.885, 0.32, 1.275), opacity 0.2s;
        max-height: calc(100vh - 110px);
        overflow-y: auto;
    }
    .drawer-panel.active {
        transform: translateY(0); opacity: 1; pointer-events: auto;
    }
    
    .drawer-header {
        display: flex; justify-content: space-between; align-items: center;
        margin-bottom: 12px; border-bottom: 2px dashed var(--border-color); padding-bottom: 8px;
    }
    .drawer-title { font-size: 13px; font-weight: 900; text-transform: uppercase; color: var(--text); display: flex; align-items: center; gap: 8px; }
    .drawer-close {
        width: 28px; height: 28px; background: var(--pink); color: #fff;
        border-radius: 8px; display: flex; align-items: center; justify-content: center;
        border: 2px solid var(--border-color); box-shadow: 2px 2px 0 var(--border-color);
        cursor: pointer; font-size: 13px;
    }

    .menu-grid {
        display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px;
    }

    .menu-item-wrap {
        position: relative;
        display: flex;
        width: 100%;
    }
    .menu-item-wrap .menu-link {
        width: 100%;
    }
    .menu-label-text {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 100%;
        display: block;
    }
    .btn-edit-shortcut {
        position: absolute;
        top: 5px;
        right: 5px;
        width: 22px;
        height: 22px;
        border-radius: 6px;
        background: var(--panel-bg);
        border: 1.5px solid var(--border-color);
        color: var(--text-muted);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 10px;
        cursor: pointer;
        box-shadow: 1px 1px 0 var(--border-color);
        transition: all 0.15s;
        z-index: 2;
    }
    .btn-edit-shortcut:hover {
        background: var(--primary);
        color: var(--primary-text);
        transform: scale(1.1);
    }
    .btn-edit-shortcut:active {
        transform: scale(0.95);
    }

    .menu-link {
        display: flex; flex-direction: column; align-items: center; gap: 6px;
        background: var(--panel-bg-alt);
        border: 2px solid var(--border-color); border-radius: 12px;
        padding: 12px 8px; font-size: 10.5px; font-weight: 800; text-align: center;
        color: var(--text); text-transform: uppercase; letter-spacing: 0.4px;
        box-shadow: 2px 2px 0 var(--border-color); transition: all 0.15s;
        cursor: pointer; font-family: inherit;
    }
    .menu-link i { font-size: 20px; }
    .menu-link:hover { transform: translate(-2px, -2px); box-shadow: 4px 4px 0 var(--border-color); }
    .menu-link:active { transform: translate(2px, 2px); box-shadow: none; }

    /* SHORTCUT MODAL */
    .shortcut-modal-overlay {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        z-index: 1000;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 16px;
    }
    .shortcut-modal {
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        border-radius: 16px;
        padding: 18px;
        width: 380px;
        max-width: 100%;
        box-shadow: var(--brutal-shadow);
        animation: modalPop 0.15s ease-out;
    }
    @keyframes modalPop {
        from { transform: scale(0.92); opacity: 0; }
        to { transform: scale(1); opacity: 1; }
    }
    .shortcut-modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 14px;
        padding-bottom: 8px;
        border-bottom: 2px dashed var(--border-color);
    }
    .shortcut-modal-title {
        font-size: 12px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--text);
        display: flex;
        align-items: center;
        gap: 6px;
    }
    .shortcut-close {
        width: 26px;
        height: 26px;
        background: var(--pink);
        color: #fff;
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        border: 1.5px solid var(--border-color);
        cursor: pointer;
        font-size: 11px;
    }
    .modal-form-group {
        display: flex;
        flex-direction: column;
        gap: 5px;
        margin-bottom: 12px;
    }
    .modal-form-group label {
        font-size: 10.5px;
        font-weight: 850;
        color: var(--text);
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }
    .modal-input {
        background: var(--panel-bg-alt);
        border: 2px solid var(--border-color);
        border-radius: 8px;
        padding: 8px 10px;
        font-size: 12px;
        color: var(--text);
        font-family: inherit;
        font-weight: 700;
        outline: none;
        transition: border-color 0.15s;
    }
    .modal-input:focus {
        border-color: var(--primary);
    }
    .path-hint {
        margin-top: 4px;
    }
    .badge-type {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        font-size: 9px;
        font-weight: 850;
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid var(--border-color);
    }
    .badge-type.file {
        background: #0284c7;
        color: #fff;
    }
    .badge-type.folder {
        background: #eab308;
        color: #000;
    }
    .shortcut-modal-actions {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 8px;
        margin-top: 16px;
        padding-top: 10px;
        border-top: 1px dashed var(--border-color);
    }
    .btn-modal-reset {
        background: var(--panel-bg-alt);
        border: 1.5px solid var(--border-color);
        color: var(--text-muted);
        border-radius: 8px;
        padding: 7px 12px;
        font-size: 11px;
        font-weight: 850;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 4px;
        transition: all 0.1s;
    }
    .btn-modal-reset:hover {
        color: var(--text);
        border-color: var(--text);
    }
    .btn-modal-save {
        background: var(--primary);
        color: var(--primary-text);
        border: 2px solid var(--border-color);
        border-radius: 8px;
        padding: 7px 16px;
        font-size: 11px;
        font-weight: 900;
        cursor: pointer;
        box-shadow: 2px 2px 0 var(--border-color);
        display: inline-flex;
        align-items: center;
        gap: 5px;
        text-transform: uppercase;
        transition: all 0.1s;
    }
    .btn-modal-save:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--border-color);
    }
    .btn-modal-save:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }
</style>
