

<script>
    import { createEventDispatcher } from 'svelte';
    import { router } from 'tinro';
    const dispatch = createEventDispatcher();
    
    export let activeDrawer;

    let currentPath = '/';
    router.subscribe(r => currentPath = r.path);

    function goHome() {
        dispatch('closeDrawers');
        router.goto('/');
    }

    function goInbox() {
        dispatch('closeDrawers');
        router.goto('/sms');
    }

    function goSystem() {
        dispatch('closeDrawers');
        router.goto('/system');
    }

</script>

<div class="dock-container">
    <div class="dock">
        <!-- HOME -->
        <button
            class="dock-item {currentPath === '/' && !activeDrawer ? 'active' : ''}"
            on:click={goHome}
            title="Overview"
        >
            <i class="fa-solid fa-house"></i>
            <span>Home</span>
        </button>

        <div class="dock-sep"></div>

        <!-- STATS -->
        <button
            class="dock-item {activeDrawer === 'menu-status' || currentPath.includes('/vnstat') || currentPath.includes('/clashboard') ? 'active' : ''}"
            on:click={() => dispatch('toggleDrawer', 'menu-status')}
            title="Status & Graphs"
        >
            <i class="fa-solid fa-chart-pie"></i>
            <span>Stats</span>
        </button>

        <!-- NET -->
        <button
            class="dock-item {activeDrawer === 'menu-net' || currentPath.includes('/hotspot') || currentPath.includes('/clients') || currentPath.includes('/speedtest') || currentPath.includes('/ttl') || currentPath.includes('/watchdog') ? 'active' : ''}"
            on:click={() => dispatch('toggleDrawer', 'menu-net')}
            title="Network & Devices"
        >
            <i class="fa-solid fa-wifi"></i>
            <span>Net</span>
        </button>

        <!-- BOX -->
        <button
            class="dock-item {activeDrawer === 'menu-box' || currentPath.includes('/bfr') || currentPath.includes('/clash-generator') ? 'active' : ''}"
            on:click={() => dispatch('toggleDrawer', 'menu-box')}
            title="Box Proxy (BFR)"
        >
            <i class="fa-solid fa-box-open"></i>
            <span>Box</span>
        </button>


        <div class="dock-sep"></div>

        <!-- INBOX -->
        <button
            class="dock-item {currentPath === '/sms' && !activeDrawer ? 'active' : ''}"
            on:click={goInbox}
            title="Inbox SMS"
        >
            <i class="fa-solid fa-envelope"></i>
            <span>Inbox</span>
        </button>

        <!-- SYSTEM -->
        <button
            class="dock-item {(currentPath === '/system' || currentPath.includes('/cron') || currentPath.includes('/battery') || currentPath.includes('/magisk') || currentPath.includes('/filemanager') || currentPath.includes('/security') || currentPath.includes('/remote-screen') || currentPath.includes('/terminal') || currentPath.includes('/reboot') || currentPath.includes('/credits') || currentPath.includes('/donate')) && !activeDrawer ? 'active' : ''}"
            on:click={goSystem}
            title="System & Tools"
        >
            <i class="fa-solid fa-gears"></i>
            <span>System</span>
        </button>
    </div>
</div>

<style>
    .dock-container {
        position: fixed;
        bottom: 12px;
        left: 0; right: 0;
        display: flex;
        justify-content: center;
        z-index: 90;
        pointer-events: none;
        padding: 0 6px;
    }

    .dock {
        pointer-events: auto;
        display: flex;
        align-items: center;
        gap: 3px;
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        padding: 4px 6px;
        border-radius: 16px;
        box-shadow: var(--brutal-shadow);
        transition: background-color 0.2s ease, border-color 0.2s ease;
        max-width: 100%;
        overflow-x: auto;
        scrollbar-width: none;
    }
    .dock::-webkit-scrollbar {
        display: none;
    }

    .dock-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        width: 44px;
        height: 46px;
        border-radius: 10px;
        border: none;
        background: transparent;
        color: var(--text-muted);
        cursor: pointer;
        transition: all 0.15s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        font-family: inherit;
        padding: 2px 0;
        flex-shrink: 0;
    }

    .dock-item i {
        font-size: 16px;
        margin-bottom: 2px;
        transition: transform 0.2s ease;
    }

    .dock-item span {
        font-size: 8px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 0.2px;
        line-height: 1;
    }

    .dock-item:hover {
        color: var(--text);
        background: var(--panel-bg-alt);
    }

    .dock-item.active {
        color: var(--primary-text);
        background: var(--primary);
        box-shadow: 2px 2px 0 var(--border-color);
        border: 1.5px solid var(--border-color);
        transform: translateY(-2px);
    }
    .dock-item.active i {
        transform: scale(1.1);
    }

    .dock-sep {
        width: 2px;
        height: 22px;
        background: var(--border-color);
        margin: 0 1px;
        opacity: 0.6;
        flex-shrink: 0;
    }

    @media (max-width: 375px) {
        .dock-item { width: 38px; height: 42px; }
        .dock-item i { font-size: 14px; }
        .dock-item span { font-size: 7px; }
    }
</style>
