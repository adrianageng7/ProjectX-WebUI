<script>
    import { Route, router } from 'tinro';
    import { onMount } from 'svelte';
    import Topbar from './components/Topbar.svelte';
    import BottomNav from './components/BottomNav.svelte';
    import DrawerMenu from './components/DrawerMenu.svelte';
    import Sidebar from './components/Sidebar.svelte';
    import Login from './pages/Login.svelte';
    
    // Import all pages
    import Overview from './pages/Overview.svelte';
    import Vnstat from './pages/Vnstat.svelte';
    import ClashDashboard from './pages/ClashDashboard.svelte';
    import BFRControl from './pages/BFRControl.svelte';
    import BFRSettings from './pages/BFRSettings.svelte';
    import FileManager from './pages/FileManager.svelte';
    import MagiskModules from './pages/MagiskModules.svelte';
    import MagiskLogs from './pages/MagiskLogs.svelte';
    import RemoteScreen from './pages/RemoteScreen.svelte';
    import Clients from './pages/Clients.svelte';
    import SMS from './pages/SMS.svelte';
    import Security from './pages/Security.svelte';
    import Reboot from './pages/Reboot.svelte';
    import Credits from './pages/Credits.svelte';
    import Donate from './pages/Donate.svelte';
    import ClashGenerator from './pages/ClashGenerator.svelte';
    import Speedtest from './pages/Speedtest.svelte';
    import BatteryControl from './pages/BatteryControl.svelte';
    import TTL from './pages/TTL.svelte';
    import Watchdog from './pages/Watchdog.svelte';
    import Terminal from './pages/Terminal.svelte';
    import System from './pages/System.svelte';
    import CronManager from './pages/CronManager.svelte';
    import Hotspot from './pages/Hotspot.svelte';


    // State
    let isAuthenticated = true; // Default true, then checked via API
    let isCheckingAuth = true;
    let activeDrawer = null;
    let theme = 'dark';
    let color = 'default';
    let navMode = 'dock'; // 'dock' | 'sidebar'
    let isMobileSidebarOpen = false;
    let showLoader = false;
    let loaderTimer = null;

    router.subscribe(() => {
        if (!isAuthenticated) return;
        showLoader = true;
        if (loaderTimer) clearTimeout(loaderTimer);
        loaderTimer = setTimeout(() => {
            showLoader = false;
        }, 300);
    });

    async function checkAuth() {
        try {
            const res = await fetch('/api/check_auth?_=' + Date.now());
            const data = await res.json();
            isAuthenticated = data.status === 'success';
        } catch (e) {
            isAuthenticated = true; // Fallback jika offline/bypass
        } finally {
            isCheckingAuth = false;
        }
    }

    onMount(() => {
        router.mode.hash();
        theme = localStorage.getItem('theme') || 'dark';
        color = localStorage.getItem('color') || 'default';
        navMode = localStorage.getItem('nav_mode') || 'dock';
        document.documentElement.setAttribute('data-theme', theme);
        document.documentElement.setAttribute('data-color', color);
        document.documentElement.removeAttribute('data-y2k');
        localStorage.removeItem('y2k');
        checkAuth();
    });

    function toggleTheme() {
        theme = theme === 'dark' ? 'light' : 'dark';
        localStorage.setItem('theme', theme);
        document.documentElement.setAttribute('data-theme', theme);
    }

    function changeColor(newColor) {
        color = newColor;
        localStorage.setItem('color', color);
        document.documentElement.setAttribute('data-color', color);
    }

    function toggleNavMode() {
        navMode = navMode === 'dock' ? 'sidebar' : 'dock';
        localStorage.setItem('nav_mode', navMode);
        closeDrawers();
    }

    function setNavMode(mode) {
        navMode = mode;
        localStorage.setItem('nav_mode', navMode);
        closeDrawers();
    }

    function toggleMobileSidebar() {
        isMobileSidebarOpen = !isMobileSidebarOpen;
    }

    function toggleDrawer(drawerId) {
        activeDrawer = activeDrawer === drawerId ? null : drawerId;
    }

    function closeDrawers() {
        activeDrawer = null;
    }

    function handleLoginSuccess() {
        isAuthenticated = true;
        router.goto('/');
    }

    function handleLogout() {
        isAuthenticated = false;
    }
</script>

{#if isCheckingAuth}
    <div class="iframe-loader">
        <div class="loader-sticker">
            <div class="loader-shape">
                <img src="./assets/smiley.png" alt="Loading..." class="smiley-img">
            </div>
            <div class="loader-label">Loading...</div>
        </div>
    </div>
{:else if !isAuthenticated}
    <Login on:loginSuccess={handleLoginSuccess} />
{:else}
    <Topbar
        {theme}
        {color}
        {navMode}
        on:toggleTheme={toggleTheme}
        on:changeColor={(e) => changeColor(e.detail)}
        on:toggleNavMode={toggleNavMode}
        on:changeNavMode={(e) => setNavMode(e.detail)}
        on:toggleMobileSidebar={toggleMobileSidebar}
        on:logout={handleLogout}
    />

    <div class="app-body {navMode === 'sidebar' ? 'layout-sidebar' : ''}">
        {#if navMode === 'sidebar'}
            <Sidebar
                bind:isOpen={isMobileSidebarOpen}
                brandTitle={localStorage.getItem('dashboard_brand_title') || 'ProjectX'}
                on:close={() => isMobileSidebarOpen = false}
                on:switchNavMode={(e) => setNavMode(e.detail)}
            />
        {/if}

        <main class="main-content {navMode === 'sidebar' ? 'sidebar-mode' : ''}">
            {#if showLoader}
                <div class="iframe-loader">
                    <div class="loader-sticker">
                        <div class="loader-shape">
                            <img src="./assets/smiley.png" alt="Loading..." class="smiley-img">
                        </div>
                        <div class="loader-label">Loading...</div>
                    </div>
                </div>
            {/if}

            <Route path="/">
                <Overview />
        </Route>
        <Route path="/vnstat">
            <Vnstat />
        </Route>
        <Route path="/clashboard">
            <ClashDashboard />
        </Route>
        <Route path="/clash-generator">
            <ClashGenerator />
        </Route>
        <Route path="/bfr/control">
            <BFRControl />
        </Route>
        <Route path="/bfr/settings">
            <BFRSettings />
        </Route>
        <Route path="/filemanager">
            <FileManager />
        </Route>
        <Route path="/magisk-modules">
            <MagiskModules />
        </Route>
        <Route path="/magisk-logs">
            <MagiskLogs />
        </Route>
        <Route path="/remote-screen">
            <RemoteScreen />
        </Route>
        <Route path="/clients">
            <Clients />
        </Route>
        <Route path="/ttl">
            <TTL />
        </Route>
        <Route path="/watchdog">
            <Watchdog />
        </Route>
        <Route path="/terminal">
            <Terminal />
        </Route>
        <Route path="/speedtest">
            <Speedtest />
        </Route>
        <Route path="/sms">
            <SMS />
        </Route>
        <Route path="/battery">
            <BatteryControl />
        </Route>
        <Route path="/security">
            <Security />
        </Route>
        <Route path="/reboot">
            <Reboot />
        </Route>
        <Route path="/credits">
            <Credits />
        </Route>
        <Route path="/donate">
            <Donate />
        </Route>
        <Route path="/system">
            <System />
        </Route>
        <Route path="/cron">
            <CronManager />
        </Route>
        <Route path="/hotspot">
            <Hotspot />
        </Route>

    </main>
    </div>

    {#if navMode === 'dock'}
        <BottomNav
            {activeDrawer}
            on:toggleDrawer={(e) => toggleDrawer(e.detail)}
            on:closeDrawers={closeDrawers}
        />

        {#if activeDrawer}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="overlay" on:click={closeDrawers}></div>
        {/if}

        <DrawerMenu {activeDrawer} on:close={closeDrawers} />
    {/if}
{/if}

<style>
    .app-body {
        display: flex;
        flex: 1;
        height: calc(100dvh - 58px);
        overflow: hidden;
        position: relative;
    }

    .main-content {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
        position: relative;
        padding-bottom: 80px;
        height: 100%;
    }

    .main-content.sidebar-mode {
        padding-bottom: 24px;
    }
    
    .overlay {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(0,0,0,0.6);
        z-index: 95;
    }

    /* ========== ANIMASI LOADING SMILEY ========== */
    .iframe-loader {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: var(--bg);
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        z-index: 999;
        pointer-events: none;
        transition: opacity 0.3s ease;
    }
    
    .loader-sticker {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0;
        animation: stickerBounce 1.2s ease-in-out infinite;
    }
    
    .loader-shape {
        width: 90px;
        height: 90px;
        background: var(--primary);
        border: 3px solid var(--border-color);
        border-radius: 50%;
        box-shadow: var(--brutal-shadow);
        position: relative;
        animation: stickerWobble 2s ease-in-out infinite;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    
    .smiley-img {
        width: 100%;
        height: 100%;
        object-fit: contain;
        border-radius: 50%;
    }
    
    .loader-label {
        margin-top: -6px;
        background: var(--secondary);
        color: #ffffff;
        font-size: 9px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: 1.5px;
        padding: 3px 8px;
        border: 2px solid var(--border-color);
        border-radius: 6px;
        box-shadow: 2px 2px 0 var(--border-color);
        transform: rotate(-3deg);
        z-index: 2;
    }

    @keyframes stickerBounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-8px); }
    }
    @keyframes stickerWobble {
        0%, 100% { transform: rotate(0deg); }
        25% { transform: rotate(-6deg); }
        75% { transform: rotate(6deg); }
    }
</style>
