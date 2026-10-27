<script>
    import { createEventDispatcher, onMount } from 'svelte';
    import { router } from 'tinro';

    const dispatch = createEventDispatcher();

    export let isOpen = false;
    export let brandTitle = 'ProjectX';

    let currentPath = '/';
    router.subscribe(r => currentPath = r.path || '/');

    const DEFAULT_CONFIG_PATH = '/data/adb/box/clash/config.yaml';
    const DEFAULT_ACCOUNT_PATH = '/data/adb/box/clash/proxy_provider';

    let configPath = DEFAULT_CONFIG_PATH;
    let configLabel = 'Edit config.yaml';
    let accountPath = DEFAULT_ACCOUNT_PATH;
    let accountLabel = 'Edit Akun';

    // State accordion grup
    let sections = {
        main: true,
        stats: true,
        net: true,
        box: true,
        system: false
    };

    function toggleSection(key) {
        sections[key] = !sections[key];
    }

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

    function navigate(path) {
        router.goto(path);
        dispatch('close');
    }

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
</script>

<!-- BACKDROP OVERLAY UNTUK MOBILE -->
{#if isOpen}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="sidebar-backdrop" on:click={() => dispatch('close')}></div>
{/if}

<aside class="sidebar-aside {isOpen ? 'open' : ''}">
    <!-- MOBILE HEADER -->
    <div class="sidebar-header-mobile">
        <div class="sh-brand">
            <img src="./assets/favicon.png" alt="Logo" class="sh-logo" />
            <div class="sh-title-wrap">
                <span class="sh-title">{brandTitle}</span>
                <span class="sh-badge"><span class="live-dot"></span> SIDEBAR</span>
            </div>
        </div>
        <button class="sh-close" on:click={() => dispatch('close')} title="Tutup Menu">
            <i class="fa-solid fa-xmark"></i>
        </button>
    </div>

    <!-- MENU LIST SCROLLABLE -->
    <div class="sidebar-content">
        <!-- GRUP 1: UTAMA -->
        <div class="menu-section">
            <div class="section-title">
                <span>Utama</span>
            </div>
            <div class="section-items">
                <button
                    class="nav-btn {currentPath === '/' ? 'active' : ''}"
                    on:click={() => navigate('/')}
                >
                    <div class="nav-icon" style="background:var(--yellow); color:#000;">
                        <i class="fa-solid fa-house"></i>
                    </div>
                    <span class="nav-text">Overview</span>
                    {#if currentPath === '/'}
                        <div class="active-indicator"></div>
                    {/if}
                </button>

                <button
                    class="nav-btn {currentPath === '/sms' ? 'active' : ''}"
                    on:click={() => navigate('/sms')}
                >
                    <div class="nav-icon" style="background:var(--pink); color:#fff;">
                        <i class="fa-solid fa-envelope"></i>
                    </div>
                    <span class="nav-text">SMS Inbox</span>
                    {#if currentPath === '/sms'}
                        <div class="active-indicator"></div>
                    {/if}
                </button>

                <button
                    class="nav-btn {currentPath === '/system' ? 'active' : ''}"
                    on:click={() => navigate('/system')}
                >
                    <div class="nav-icon" style="background:var(--cyan); color:#000;">
                        <i class="fa-solid fa-gears"></i>
                    </div>
                    <span class="nav-text">System Info</span>
                    {#if currentPath === '/system'}
                        <div class="active-indicator"></div>
                    {/if}
                </button>
            </div>
        </div>

        <!-- GRUP 2: STATUS & CHARTS -->
        <div class="menu-section">
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="section-title toggleable" on:click={() => toggleSection('stats')}>
                <span><i class="fa-solid fa-chart-pie" style="color:var(--pink);"></i> Status & Grafik</span>
                <i class="fa-solid {sections.stats ? 'fa-chevron-down' : 'fa-chevron-right'} chevron"></i>
            </div>
            {#if sections.stats}
                <div class="section-items">
                    <button
                        class="nav-btn {currentPath.includes('/vnstat') ? 'active' : ''}"
                        on:click={() => navigate('/vnstat')}
                    >
                        <div class="nav-icon" style="background:var(--cyan); color:#000;">
                            <i class="fa-solid fa-chart-line"></i>
                        </div>
                        <span class="nav-text">Traffic Monitor</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/clashboard') ? 'active' : ''}"
                        on:click={() => navigate('/clashboard')}
                    >
                        <div class="nav-icon" style="background:var(--yellow); color:#000;">
                            <i class="fa-solid fa-shield-halved"></i>
                        </div>
                        <span class="nav-text">Clashboard</span>
                    </button>
                </div>
            {/if}
        </div>

        <!-- GRUP 3: JARINGAN & PERANGKAT -->
        <div class="menu-section">
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="section-title toggleable" on:click={() => toggleSection('net')}>
                <span><i class="fa-solid fa-wifi" style="color:var(--lime);"></i> Jaringan & Wi-Fi</span>
                <i class="fa-solid {sections.net ? 'fa-chevron-down' : 'fa-chevron-right'} chevron"></i>
            </div>
            {#if sections.net}
                <div class="section-items">
                    <button
                        class="nav-btn {currentPath.includes('/hotspot') ? 'active' : ''}"
                        on:click={() => navigate('/hotspot')}
                    >
                        <div class="nav-icon" style="background:var(--yellow); color:#000;">
                            <i class="fa-solid fa-wifi"></i>
                        </div>
                        <span class="nav-text">Wi-Fi Hotspot</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/clients') ? 'active' : ''}"
                        on:click={() => navigate('/clients')}
                    >
                        <div class="nav-icon" style="background:var(--lime); color:#000;">
                            <i class="fa-solid fa-users"></i>
                        </div>
                        <span class="nav-text">Connected Devices</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/speedtest') ? 'active' : ''}"
                        on:click={() => navigate('/speedtest')}
                    >
                        <div class="nav-icon" style="background:var(--cyan); color:#000;">
                            <i class="fa-solid fa-gauge-high"></i>
                        </div>
                        <span class="nav-text">Speedtest</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/ttl') ? 'active' : ''}"
                        on:click={() => navigate('/ttl')}
                    >
                        <div class="nav-icon" style="background:var(--pink); color:#fff;">
                            <i class="fa-solid fa-shield-virus"></i>
                        </div>
                        <span class="nav-text">TTL & Hop Limit</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/watchdog') ? 'active' : ''}"
                        on:click={() => navigate('/watchdog')}
                    >
                        <div class="nav-icon" style="background:var(--yellow); color:#000;">
                            <i class="fa-solid fa-dog"></i>
                        </div>
                        <span class="nav-text">Watchdog Ping</span>
                    </button>
                </div>
            {/if}
        </div>

        <!-- GRUP 4: BOX PROXY (BFR) -->
        <div class="menu-section">
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="section-title toggleable" on:click={() => toggleSection('box')}>
                <span><i class="fa-solid fa-box-open" style="color:var(--yellow);"></i> Box Proxy (BFR)</span>
                <i class="fa-solid {sections.box ? 'fa-chevron-down' : 'fa-chevron-right'} chevron"></i>
            </div>
            {#if sections.box}
                <div class="section-items">
                    <button
                        class="nav-btn {currentPath === '/bfr/control' ? 'active' : ''}"
                        on:click={() => navigate('/bfr/control')}
                    >
                        <div class="nav-icon" style="background:var(--lime); color:#000;">
                            <i class="fa-solid fa-box"></i>
                        </div>
                        <span class="nav-text">BFR Control</span>
                    </button>

                    <button
                        class="nav-btn {currentPath === '/bfr/settings' ? 'active' : ''}"
                        on:click={() => navigate('/bfr/settings')}
                    >
                        <div class="nav-icon" style="background:var(--cyan); color:#000;">
                            <i class="fa-solid fa-sliders"></i>
                        </div>
                        <span class="nav-text">Box Settings</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/clash-generator') ? 'active' : ''}"
                        on:click={() => navigate('/clash-generator')}
                    >
                        <div class="nav-icon" style="background:var(--yellow); color:#000;">
                            <i class="fa-solid fa-server"></i>
                        </div>
                        <span class="nav-text">Kelola Akun Proxy</span>
                    </button>

                    <!-- SHORTCUTS FILE -->
                    <button
                        class="nav-btn shortcut-btn"
                        on:click={() => openShortcut(configPath)}
                        title={configPath}
                    >
                        <div class="nav-icon" style="background:var(--yellow); color:#000;">
                            <i class="fa-solid fa-file-signature"></i>
                        </div>
                        <span class="nav-text">{configLabel}</span>
                    </button>

                    <button
                        class="nav-btn shortcut-btn"
                        on:click={() => openShortcut(accountPath)}
                        title={accountPath}
                    >
                        <div class="nav-icon" style="background:var(--purple); color:#fff;">
                            <i class="fa-solid fa-user-gear"></i>
                        </div>
                        <span class="nav-text">{accountLabel}</span>
                    </button>
                </div>
            {/if}
        </div>

        <!-- GRUP 5: SISTEM & ALAT LAINNYA -->
        <div class="menu-section">
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="section-title toggleable" on:click={() => toggleSection('system')}>
                <span><i class="fa-solid fa-screwdriver-wrench" style="color:var(--cyan);"></i> Alat & Konfigurasi</span>
                <i class="fa-solid {sections.system ? 'fa-chevron-down' : 'fa-chevron-right'} chevron"></i>
            </div>
            {#if sections.system}
                <div class="section-items">
                    <button
                        class="nav-btn {currentPath.includes('/remote-screen') ? 'active' : ''}"
                        on:click={() => navigate('/remote-screen')}
                    >
                        <div class="nav-icon" style="background:var(--pink); color:#fff;">
                            <i class="fa-solid fa-mobile-screen"></i>
                        </div>
                        <span class="nav-text">Scrcpy Live</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/terminal') ? 'active' : ''}"
                        on:click={() => navigate('/terminal')}
                    >
                        <div class="nav-icon" style="background:var(--lime); color:#000;">
                            <i class="fa-solid fa-terminal"></i>
                        </div>
                        <span class="nav-text">Web Terminal</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/filemanager') ? 'active' : ''}"
                        on:click={() => navigate('/filemanager?dir=/sdcard')}
                    >
                        <div class="nav-icon" style="background:var(--yellow); color:#000;">
                            <i class="fa-solid fa-folder-open"></i>
                        </div>
                        <span class="nav-text">File Manager</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/battery') ? 'active' : ''}"
                        on:click={() => navigate('/battery')}
                    >
                        <div class="nav-icon" style="background:var(--lime); color:#000;">
                            <i class="fa-solid fa-battery-half"></i>
                        </div>
                        <span class="nav-text">Proteksi Baterai</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/cron') ? 'active' : ''}"
                        on:click={() => navigate('/cron')}
                    >
                        <div class="nav-icon" style="background:var(--yellow); color:#000;">
                            <i class="fa-solid fa-clock"></i>
                        </div>
                        <span class="nav-text">Cron Manager</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/magisk-modules') ? 'active' : ''}"
                        on:click={() => navigate('/magisk-modules')}
                    >
                        <div class="nav-icon" style="background:var(--purple); color:#fff;">
                            <i class="fa-solid fa-puzzle-piece"></i>
                        </div>
                        <span class="nav-text">Magisk Modules</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/magisk-logs') ? 'active' : ''}"
                        on:click={() => navigate('/magisk-logs')}
                    >
                        <div class="nav-icon" style="background:var(--cyan); color:#000;">
                            <i class="fa-solid fa-file-lines"></i>
                        </div>
                        <span class="nav-text">Magisk Logs</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/security') ? 'active' : ''}"
                        on:click={() => navigate('/security')}
                    >
                        <div class="nav-icon" style="background:var(--cyan); color:#000;">
                            <i class="fa-solid fa-user-shield"></i>
                        </div>
                        <span class="nav-text">Keamanan WebUI</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/reboot') ? 'active' : ''}"
                        on:click={() => navigate('/reboot')}
                    >
                        <div class="nav-icon" style="background:var(--pink); color:#fff;">
                            <i class="fa-solid fa-power-off"></i>
                        </div>
                        <span class="nav-text">Reboot Perangkat</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/credits') ? 'active' : ''}"
                        on:click={() => navigate('/credits')}
                    >
                        <div class="nav-icon" style="background:var(--purple); color:#fff;">
                            <i class="fa-solid fa-circle-info"></i>
                        </div>
                        <span class="nav-text">Credits</span>
                    </button>

                    <button
                        class="nav-btn {currentPath.includes('/donate') ? 'active' : ''}"
                        on:click={() => navigate('/donate')}
                    >
                        <div class="nav-icon" style="background:var(--pink); color:#fff;">
                            <i class="fa-solid fa-heart"></i>
                        </div>
                        <span class="nav-text">Donasi</span>
                    </button>
                </div>
            {/if}
        </div>
    </div>
</aside>

<style>
    /* BACKDROP MOBILE */
    .sidebar-backdrop {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(0, 0, 0, 0.65);
        backdrop-filter: blur(2px);
        z-index: 199;
    }

    /* ASIDE CONTAINER */
    .sidebar-aside {
        width: 250px;
        background: var(--panel-bg);
        border-right: var(--border-width) solid var(--border-color);
        display: flex;
        flex-direction: column;
        height: 100%;
        flex-shrink: 0;
        transition: transform 0.25s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        z-index: 200;
        box-shadow: 2px 0 0 var(--border-color);
    }

    /* HEADER MOBILE */
    .sidebar-header-mobile {
        display: none;
        align-items: center;
        justify-content: space-between;
        padding: 12px 14px;
        border-bottom: var(--border-width) solid var(--border-color);
        background: var(--panel-bg-alt);
    }
    .sh-brand {
        display: flex;
        align-items: center;
        gap: 8px;
    }
    .sh-logo {
        width: 28px;
        height: 28px;
        border-radius: 8px;
        border: 2px solid var(--border-color);
    }
    .sh-title-wrap {
        display: flex;
        flex-direction: column;
    }
    .sh-title {
        font-weight: 900;
        font-size: 13px;
        color: var(--text);
    }
    .sh-badge {
        font-size: 9px;
        font-weight: 800;
        color: var(--text-muted);
        display: flex;
        align-items: center;
        gap: 4px;
    }
    .live-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: var(--lime);
        box-shadow: 0 0 6px var(--lime);
    }
    .sh-close {
        width: 32px;
        height: 32px;
        background: var(--pink);
        color: #fff;
        border: 2px solid var(--border-color);
        border-radius: 8px;
        box-shadow: 2px 2px 0 var(--border-color);
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        font-size: 14px;
    }

    /* SCROLLABLE CONTENT */
    .sidebar-content {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
        padding: 10px 10px 24px;
        display: flex;
        flex-direction: column;
        gap: 12px;
        scrollbar-width: thin;
        scrollbar-color: var(--border-color) transparent;
    }
    .sidebar-content::-webkit-scrollbar {
        width: 4px;
    }
    .sidebar-content::-webkit-scrollbar-thumb {
        background: var(--border-color);
        border-radius: 4px;
    }

    /* SECTIONS */
    .menu-section {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .section-title {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 0.8px;
        color: var(--text-muted);
        padding: 4px 6px;
        display: flex;
        align-items: center;
        justify-content: space-between;
    }
    .section-title.toggleable {
        cursor: pointer;
        border-radius: 6px;
        user-select: none;
    }
    .section-title.toggleable:hover {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text);
    }
    .section-title span {
        display: flex;
        align-items: center;
        gap: 6px;
    }
    .chevron {
        font-size: 10px;
    }

    .section-items {
        display: flex;
        flex-direction: column;
        gap: 3px;
    }

    /* NAV BUTTONS */
    .nav-btn {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 6px 8px;
        background: transparent;
        border: 1.5px solid transparent;
        border-radius: 10px;
        color: var(--text);
        cursor: pointer;
        text-align: left;
        font-family: inherit;
        font-size: 11.5px;
        font-weight: 700;
        transition: all 0.12s ease;
        position: relative;
    }
    .nav-btn:hover {
        background: var(--panel-bg-alt);
        border-color: var(--border-color);
        transform: translateX(2px);
    }
    .nav-btn.active {
        background: var(--panel-bg-alt);
        border-color: var(--border-color);
        box-shadow: 2px 2px 0 var(--border-color);
        color: var(--primary);
    }
    .nav-icon {
        width: 26px;
        height: 26px;
        border-radius: 7px;
        border: 1.5px solid var(--border-color);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 11px;
        flex-shrink: 0;
        box-shadow: 1px 1px 0 var(--border-color);
    }
    .nav-text {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .active-indicator {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: var(--primary);
        box-shadow: 0 0 6px var(--primary);
    }

    .shortcut-btn {
        opacity: 0.9;
    }
    .shortcut-btn:hover {
        opacity: 1;
    }

    /* RESPONSIVE LAYOUT */
    @media (max-width: 768px) {
        .sidebar-aside {
            position: fixed;
            top: 0;
            left: 0;
            bottom: 0;
            width: 270px;
            height: 100dvh;
            transform: translateX(-100%);
            box-shadow: 4px 0 0 var(--border-color);
        }
        .sidebar-aside.open {
            transform: translateX(0);
        }
        .sidebar-header-mobile {
            display: flex;
        }
    }
</style>

