<script>
    import { onMount, onDestroy } from 'svelte';
    import JMuxer from 'jmuxer';

    let videoEl;
    let viewportEl;
    let ws;
    let jmuxer;
    let isConnected = false;
    let isFullscreen = false;
    let isCssFullscreen = false;
    let deviceW = 1080;
    let deviceH = 1920;

    $: isFsActive = isFullscreen || isCssFullscreen;

    function handleFsChange() {
        isFullscreen = !!(document.fullscreenElement || document.webkitFullscreenElement || document.mozFullScreenElement || document.msFullscreenElement);
        if (!isFullscreen) {
            isCssFullscreen = false;
        }
    }

    function toggleFullscreen() {
        const canNativeFs = document.fullscreenEnabled || document.webkitFullscreenEnabled;
        if (canNativeFs && viewportEl) {
            if (!document.fullscreenElement && !document.webkitFullscreenElement) {
                const req = viewportEl.requestFullscreen || viewportEl.webkitRequestFullscreen || viewportEl.mozRequestFullScreen || viewportEl.msRequestFullscreen;
                if (req) {
                    req.call(viewportEl).catch(() => {
                        isCssFullscreen = !isCssFullscreen;
                    });
                    return;
                }
            } else {
                const exit = document.exitFullscreen || document.webkitExitFullscreen || document.mozCancelFullScreen || document.msExitFullscreen;
                if (exit) {
                    exit.call(document).catch(() => {});
                    return;
                }
            }
        }
        isCssFullscreen = !isCssFullscreen;
    }

    function handleKeyDown(e) {
        if (e.key === 'Escape' && isCssFullscreen) {
            isCssFullscreen = false;
        }
        if ((e.key === 'f' || e.key === 'F') && !e.ctrlKey && !e.altKey && !e.metaKey) {
            if (document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
                e.preventDefault();
                toggleFullscreen();
            }
        }
    }

    // Scrcpy control message types
    const TYPE_INJECT_TOUCH = 2;
    const TYPE_INJECT_KEYCODE = 0;
    const TYPE_BACK_OR_SCREEN_ON = 4;

    // Android keycodes
    const KEYCODE = {
        HOME: 3,
        BACK: 4,
        APP_SWITCH: 187,  // Recents
        POWER: 26,
        VOLUME_UP: 24,
        VOLUME_DOWN: 25,
        SCREENSHOT: 120,  // SYSRQ
        ROTATE: 193,      // no standard key, will use adb for this
    };

    // Android MotionEvent actions
    const ACTION_DOWN = 0;
    const ACTION_UP = 1;
    const ACTION_MOVE = 2;

    let pointerDown = false;
    let lastPointerId = 0xFFFFFFFF; // POINTER_ID_GENERIC_FINGER

    onMount(() => {
        jmuxer = new JMuxer({
            node: videoEl,
            mode: 'video',
            flushingTime: 0,
            fps: 30,
            debug: false
        });

        connectWs();

        document.addEventListener('fullscreenchange', handleFsChange);
        document.addEventListener('webkitfullscreenchange', handleFsChange);
        document.addEventListener('mozfullscreenchange', handleFsChange);
        document.addEventListener('MSFullscreenChange', handleFsChange);
    });

    function connectWs() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/api/scrcpy/ws`);
        ws.binaryType = 'arraybuffer';

        ws.onopen = () => { isConnected = true; };

        ws.onmessage = (event) => {
            const data = new Uint8Array(event.data);
            // First byte 0xFF = metadata packet from our Go server
            if (data[0] === 0xFF && data.length === 5) {
                deviceW = (data[1] << 8) | data[2];
                deviceH = (data[3] << 8) | data[4];
                return;
            }
            jmuxer.feed({ video: data });
        };

        ws.onclose = () => {
            isConnected = false;
            setTimeout(connectWs, 3000);
        };
    }

    onDestroy(() => {
        if (ws) ws.close();
        if (jmuxer) jmuxer.destroy();
        document.removeEventListener('fullscreenchange', handleFsChange);
        document.removeEventListener('webkitfullscreenchange', handleFsChange);
        document.removeEventListener('mozfullscreenchange', handleFsChange);
        document.removeEventListener('MSFullscreenChange', handleFsChange);
    });

    // ─── Touch/Mouse → Scrcpy control packet ───────────────────────────────────

    function getScaledCoords(e) {
        const rect = videoEl.getBoundingClientRect();
        // The video element uses object-fit:contain — calculate actual rendered size
        const videoAspect = deviceW / deviceH;
        const boxAspect = rect.width / rect.height;

        let renderedW, renderedH, offsetX, offsetY;
        if (videoAspect > boxAspect) {
            renderedW = rect.width;
            renderedH = rect.width / videoAspect;
            offsetX = 0;
            offsetY = (rect.height - renderedH) / 2;
        } else {
            renderedH = rect.height;
            renderedW = rect.height * videoAspect;
            offsetX = (rect.width - renderedW) / 2;
            offsetY = 0;
        }

        const clientX = e.touches ? e.touches[0].clientX : e.clientX;
        const clientY = e.touches ? e.touches[0].clientY : e.clientY;

        const x = Math.round(((clientX - rect.left - offsetX) / renderedW) * deviceW);
        const y = Math.round(((clientY - rect.top - offsetY) / renderedH) * deviceH);
        return {
            x: Math.max(0, Math.min(deviceW, x)),
            y: Math.max(0, Math.min(deviceH, y))
        };
    }

    function buildTouchPacket(action, x, y, pressure) {
        // Scrcpy inject touch event (type 2) for scrcpy 2.x:
        // 1B type | 1B action | 8B pointerId | 4B x | 4B y | 2B screenW | 2B screenH | 2B pressure | 4B actionButton | 4B buttons
        const buf = new ArrayBuffer(32);
        const view = new DataView(buf);
        let offset = 0;
        view.setUint8(offset++, TYPE_INJECT_TOUCH);
        view.setUint8(offset++, action);
        // pointerId (int64) — use generic finger ID (1) instead of mouse (-1)
        view.setUint32(offset, 0); offset += 4;
        view.setUint32(offset, 1); offset += 4;
        // position
        view.setInt32(offset, x); offset += 4;
        view.setInt32(offset, y); offset += 4;
        // screen size
        view.setUint16(offset, deviceW); offset += 2;
        view.setUint16(offset, deviceH); offset += 2;
        // pressure (0xFFFF = 1.0f in fixed point)
        view.setUint16(offset, pressure ? 0xFFFF : 0); offset += 2;
        // action button (0 for touch)
        view.setUint32(offset, 0); offset += 4;
        // buttons (0 for touch)
        view.setUint32(offset, 0);
        return new Uint8Array(buf);
    }

    function sendTouch(action, e) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        const { x, y } = getScaledCoords(e);
        const packet = buildTouchPacket(action, x, y, action !== ACTION_UP);
        ws.send(packet);
    }

    function onPointerDown(e) {
        if (e.button !== undefined && e.button !== 0) return; // left button only
        e.preventDefault();
        pointerDown = true;
        sendTouch(ACTION_DOWN, e);
    }
    function onPointerMove(e) {
        if (!pointerDown) return;
        e.preventDefault();
        sendTouch(ACTION_MOVE, e);
    }
    function onPointerUp(e) {
        if (!pointerDown) return;
        e.preventDefault();
        pointerDown = false;
        sendTouch(ACTION_UP, e);
    }

    // Touch events (mobile browser)
    function onTouchStart(e) { e.preventDefault(); sendTouch(ACTION_DOWN, e); }
    function onTouchMove(e) { e.preventDefault(); sendTouch(ACTION_MOVE, e); }
    function onTouchEnd(e) {
        e.preventDefault();
        // Use last position for ACTION_UP — create synthetic event from changedTouches
        const last = e.changedTouches[0];
        sendTouch(ACTION_UP, { clientX: last.clientX, clientY: last.clientY });
    }

    // ─── Keycode shortcuts ──────────────────────────────────────────────────────

    function buildKeycodePacket(keycode, action = 0 /*ACTION_DOWN*/, repeat = 0, metaState = 0) {
        // Scrcpy inject keycode (type 0):
        // 1B type | 1B action | 4B keycode | 4B repeat | 4B metaState
        const buf = new ArrayBuffer(14);
        const view = new DataView(buf);
        view.setUint8(0, TYPE_INJECT_KEYCODE);
        view.setUint8(1, action);
        view.setInt32(2, keycode);
        view.setInt32(6, repeat);
        view.setInt32(10, metaState);
        return new Uint8Array(buf);
    }

    function pressKey(keycode) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        ws.send(buildKeycodePacket(keycode, 0)); // DOWN
        setTimeout(() => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(buildKeycodePacket(keycode, 1)); // UP
            }
        }, 50);
    }

    const shortcuts = [
        { label: 'Back',        icon: 'fa-solid fa-chevron-left',   color: 'var(--cyan)',   key: KEYCODE.BACK },
        { label: 'Home',        icon: 'fa-solid fa-house',          color: 'var(--primary)',key: KEYCODE.HOME },
        { label: 'Recent',      icon: 'fa-solid fa-square',         color: 'var(--yellow)', key: KEYCODE.APP_SWITCH },
        { label: 'Volume Up',   icon: 'fa-solid fa-volume-high',    color: 'var(--lime)',   key: KEYCODE.VOLUME_UP },
        { label: 'Volume Down', icon: 'fa-solid fa-volume-low',     color: 'var(--cyan)',   key: KEYCODE.VOLUME_DOWN },
        { label: 'Power',       icon: 'fa-solid fa-power-off',      color: 'var(--pink)',   key: KEYCODE.POWER },
        { label: 'Screenshot',  icon: 'fa-solid fa-camera',         color: 'var(--purple)', key: KEYCODE.SCREENSHOT },
    ];
</script>

<svelte:window on:keydown={handleKeyDown} />

<div class="wrap">
    <div class="header">
        <h1><i class="fa-solid fa-mobile-screen" style="color: var(--primary);"></i> Remote Screen</h1>
        <div class="header-right">
            <button class="fs-toggle-btn" on:click={toggleFullscreen} title={isFsActive ? "Keluar Fullscreen (Esc)" : "Layar Penuh (F)"}>
                <i class="fa-solid {isFsActive ? 'fa-compress' : 'fa-expand'}"></i>
                <span class="fs-text">{isFsActive ? 'Exit' : 'Fullscreen'}</span>
            </button>
            <div class="status {isConnected ? 'on' : 'off'}">
                {isConnected ? '● Live' : '○ Offline'}
            </div>
        </div>
    </div>

    <!-- Viewport Box (di-fullscreen-kan) -->
    <div class="viewport-box" bind:this={viewportEl} class:is-fs={isFsActive}>
        <div class="screen-wrap">
            <!-- svelte-ignore a11y_media_has_caption -->
            <video
                bind:this={videoEl}
                autoplay
                muted
                playsinline
                on:mousedown={onPointerDown}
                on:mousemove={onPointerMove}
                on:mouseup={onPointerUp}
                on:mouseleave={onPointerUp}
                on:touchstart={onTouchStart}
                on:touchmove={onTouchMove}
                on:touchend={onTouchEnd}
                style="cursor: {pointerDown ? 'crosshair' : 'pointer'}; touch-action: none;"
            ></video>

            {#if !isConnected}
                <div class="overlay">
                    <div class="connecting-msg">
                        <span class="spin">⟳</span> Menghubungkan ke perangkat…
                    </div>
                </div>
            {/if}

            {#if isFsActive}
                <button class="floating-exit-btn" on:click={toggleFullscreen} title="Keluar Fullscreen (Esc)">
                    <i class="fa-solid fa-xmark"></i>
                </button>
            {/if}
        </div>

        <!-- Toolbar Navigasi (di bawah layar) -->
        <div class="toolbar-wrap" class:fs-toolbar={isFsActive}>
            <div class="shortcuts">
                {#each shortcuts as s}
                    <button class="shortcut-btn" on:click={() => pressKey(s.key)} title={s.label}>
                        <i class="{s.icon}" style="color: {s.color};"></i>
                    </button>
                {/each}
                <button class="shortcut-btn fs-shortcut-btn" on:click={toggleFullscreen} title={isFsActive ? "Keluar Fullscreen (Esc)" : "Layar Penuh (F)"}>
                    <i class="fa-solid {isFsActive ? 'fa-compress' : 'fa-expand'}" style="color: var(--cyan);"></i>
                </button>
            </div>
        </div>
    </div>

    {#if !isFsActive}
        <div class="hint">Tap / drag pada layar untuk interaksi · Tekan [F] atau tombol Fullscreen untuk layar penuh</div>
    {/if}
</div>

<style>
    .wrap {
        padding: 8px 14px 4px;
        display: flex;
        flex-direction: column;
        height: 100%;
        box-sizing: border-box;
        gap: 8px;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-bottom: 2px;
    }

    h1 {
        font-size: 15px;
        font-weight: 950;
        margin: 0;
        display: flex;
        align-items: center;
        gap: 8px;
        text-transform: uppercase;
        letter-spacing: -0.3px;
    }

    .header-right {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .fs-toggle-btn {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        background: var(--paper);
        color: var(--ink);
        border: 1.5px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
        border-radius: 8px;
        padding: 4px 10px;
        font-size: 11px;
        font-weight: 850;
        cursor: pointer;
        transition: all 0.15s ease;
        user-select: none;
        font-family: inherit;
    }
    .fs-toggle-btn:hover {
        background: var(--primary);
        color: var(--primary-text);
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--line);
    }
    .fs-toggle-btn:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }

    .status {
        padding: 2px 8px;
        border-radius: 20px;
        font-size: 10.5px;
        font-weight: 900;
        border: 1.5px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
        letter-spacing: 0.3px;
    }
    .status.on  { background: var(--lime); color: #000; }
    .status.off { background: var(--pink); color: #fff; }

    /* ── Viewport Box ────────────────────────────────────── */
    .viewport-box {
        display: flex;
        flex-direction: column;
        flex: 1;
        gap: 8px;
        min-height: 0;
        position: relative;
    }

    /* ── Compact Toolbar ────────────────────────────────── */
    .toolbar-wrap {
        display: flex;
        justify-content: center;
        align-items: center;
        width: 100%;
    }

    .shortcuts {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 14px;
        padding: 4px 6px;
        box-shadow: var(--brutal-shadow-sm);
    }

    .shortcut-btn {
        width: 34px;
        height: 34px;
        border-radius: 9px;
        background: var(--paper);
        border: 1.5px solid var(--line);
        color: var(--ink);
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        font-size: 12px;
        box-shadow: 1.5px 1.5px 0 var(--line);
        transition: all 0.1s ease;
        user-select: none;
        font-family: inherit;
    }
    .shortcut-btn:hover {
        transform: translate(-1px, -1px);
        box-shadow: 2.5px 2.5px 0 var(--line);
        border-color: var(--primary);
    }
    .shortcut-btn:active {
        transform: translate(1px, 1px);
        box-shadow: none;
        background: var(--primary);
    }
    .shortcut-btn:active i {
        color: var(--primary-text) !important;
    }

    /* ── Video screen ────────────────────────────────── */
    .screen-wrap {
        flex: 1;
        background: #000000;
        border: 2px solid var(--line);
        border-radius: 16px;
        overflow: hidden;
        position: relative;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: var(--shadow);
        min-height: 0;
    }

    video {
        width: 100%;
        height: 100%;
        object-fit: contain;
        display: block;
        -webkit-user-select: none;
        user-select: none;
    }

    .overlay {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0,0,0,.65);
        pointer-events: none;
    }

    .connecting-msg {
        color: #fff;
        font-size: 13px;
        font-weight: 800;
        display: flex;
        align-items: center;
        gap: 8px;
        background: var(--paper);
        padding: 8px 16px;
        border-radius: 12px;
        border: 2px solid var(--line);
        box-shadow: var(--brutal-shadow);
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    .spin {
        display: inline-block;
        animation: spin 1s linear infinite;
        font-size: 16px;
    }

    .hint {
        text-align: center;
        font-size: 10px;
        font-weight: 700;
        color: var(--muted);
        letter-spacing: 0.2px;
    }

    /* ── FULLSCREEN MODE STYLES ────────────────────────── */
    .viewport-box.is-fs,
    .viewport-box:fullscreen,
    .viewport-box:-webkit-full-screen {
        position: fixed;
        inset: 0;
        z-index: 999999;
        width: 100vw;
        height: 100vh;
        background: #000000;
        margin: 0;
        padding: 0;
        gap: 0;
        display: flex;
        flex-direction: column;
    }

    .viewport-box.is-fs .screen-wrap,
    .viewport-box:fullscreen .screen-wrap,
    .viewport-box:-webkit-full-screen .screen-wrap {
        width: 100%;
        height: 100%;
        border: none;
        border-radius: 0;
        box-shadow: none;
        flex: 1;
    }

    .viewport-box.is-fs .toolbar-wrap,
    .viewport-box:fullscreen .toolbar-wrap,
    .viewport-box:-webkit-full-screen .toolbar-wrap {
        position: absolute;
        bottom: 24px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 1000;
        width: auto;
        opacity: 0.88;
        transition: all 0.2s ease;
    }

    .viewport-box.is-fs .toolbar-wrap:hover,
    .viewport-box:fullscreen .toolbar-wrap:hover,
    .viewport-box:-webkit-full-screen .toolbar-wrap:hover {
        opacity: 1;
        transform: translateX(-50%) translateY(-2px);
    }

    .viewport-box.is-fs .shortcuts,
    .viewport-box:fullscreen .shortcuts,
    .viewport-box:-webkit-full-screen .shortcuts {
        background: rgba(24, 21, 30, 0.88);
        backdrop-filter: blur(14px);
        -webkit-backdrop-filter: blur(14px);
        border: 2px solid rgba(255, 255, 255, 0.22);
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.7);
        padding: 6px 8px;
    }

    .floating-exit-btn {
        position: absolute;
        top: 18px;
        right: 18px;
        z-index: 1001;
        width: 38px;
        height: 38px;
        border-radius: 50%;
        background: rgba(20, 18, 26, 0.75);
        backdrop-filter: blur(8px);
        -webkit-backdrop-filter: blur(8px);
        border: 1.5px solid rgba(255, 255, 255, 0.25);
        color: #fff;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 15px;
        cursor: pointer;
        transition: all 0.15s ease;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.5);
    }
    .floating-exit-btn:hover {
        background: var(--pink);
        color: #fff;
        transform: scale(1.1);
        border-color: #fff;
    }
    .floating-exit-btn:active {
        transform: scale(0.95);
    }
</style>
