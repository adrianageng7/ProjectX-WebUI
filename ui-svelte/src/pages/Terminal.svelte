<script>
    import { onMount, onDestroy, tick } from 'svelte';
    import { Terminal } from 'xterm';
    import { FitAddon } from 'xterm-addon-fit';
    import 'xterm/css/xterm.css';

    let terminalContainer;
    let term;
    let fitAddon;
    let ws;
    let isConnected = false;
    let isConnecting = false;
    let connectionError = '';
    let isFullscreen = false;
    let showVirtualKeys = true;
    let showPresetModal = false;

    // Mobile input bar
    let mobileInput = '';
    let isExecRunning = false;

    // Presets
    let presets = [
        { category: 'Network', icon: 'fa-wifi', name: 'IP & Route', cmd: 'ip a && echo "\n--- ROUTING TABLE ---" && ip route', desc: 'Daftar IP interface & routing kernel' },
        { category: 'Network', icon: 'fa-shield-halved', name: 'NAT iptables', cmd: 'iptables -t nat -nvL', desc: 'Aturan NAT & port forwarding' },
        { category: 'Network', icon: 'fa-shield-cat', name: 'Mangle & TTL', cmd: 'iptables -t mangle -nvL PROJECTX_TTL 2>/dev/null || iptables -t mangle -nvL', desc: 'Status chain TTL / Hop limit' },
        { category: 'Network', icon: 'fa-network-wired', name: 'Tabel ARP', cmd: 'cat /proc/net/arp', desc: 'Daftar MAC & IP klien hotspot' },
        { category: 'Network', icon: 'fa-tower-broadcast', name: 'Ping 1.1.1.1', cmd: 'ping -c 4 1.1.1.1', desc: 'Tes koneksi Cloudflare DNS' },

        { category: 'BFR', icon: 'fa-box', name: 'Proses Proxy', cmd: 'ps -ef | grep -E "box|clash|sing-box|xray|projectx" | grep -v grep', desc: 'Cek proses core yang berjalan' },
        { category: 'BFR', icon: 'fa-scroll', name: 'Log BFR Core', cmd: 'tail -n 40 /data/adb/box/clash/clash.log 2>/dev/null || tail -n 40 /data/adb/box/sing-box/sing-box.log 2>/dev/null || echo "Log proxy tidak ditemukan"', desc: '40 baris log terakhir BFR' },
        { category: 'BFR', icon: 'fa-rotate', name: 'Restart BFR', cmd: '/data/adb/box/scripts/box.service restart 2>&1 || echo "Script box.service tidak ditemukan"', desc: 'Restart service proxy core', confirm: true },

        { category: 'System', icon: 'fa-memory', name: 'RAM & Disk', cmd: 'free -m && echo "\n--- STORAGE ---" && df -h /data /system', desc: 'Penggunaan memori & kapasitas internal' },
        { category: 'System', icon: 'fa-clock', name: 'Uptime', cmd: 'uptime', desc: 'Lama sistem aktif & beban load' },
        { category: 'System', icon: 'fa-battery-half', name: 'Baterai & Suhu', cmd: 'dumpsys battery', desc: 'Level baterai, charger, dan suhu' },
        { category: 'System', icon: 'fa-signal', name: 'Sinyal Modem', cmd: 'dumpsys telephony.registry 2>/dev/null | grep -iE "mSignalStrength|mServiceState" | head -n 12', desc: 'Informasi sinyal radio modem' },
        { category: 'System', icon: 'fa-broom', name: 'Bersihkan RAM', cmd: 'sync && echo 3 > /proc/sys/vm/drop_caches && echo "Cache memori dibersihkan!" && free -m', desc: 'Drop caches kernel untuk melegakan RAM', confirm: true }
    ];

    // Universal Modal State
    let modal = {
        show: false,
        title: '',
        message: '',
        warning: '',
        type: 'confirm',
        confirmText: 'Jalankan',
        confirmIcon: 'fa-play',
        confirmAction: null
    };

    function initTerminal() {
        if (!terminalContainer) return;

        term = new Terminal({
            cursorBlink: true,
            cursorStyle: 'block',
            fontSize: 12,
            fontFamily: "'JetBrains Mono', Consolas, monospace",
            lineHeight: 1.2,
            theme: {
                background: '#0d0b11',
                foreground: '#e6edf3',
                cursor: '#ffe14a',
                cursorAccent: '#0d0b11',
                selectionBackground: 'rgba(255, 92, 168, 0.4)',
                black: '#17151a',
                red: '#ff5c77',
                green: '#39ff14',
                yellow: '#ffe14a',
                blue: '#55dfff',
                magenta: '#ff5ca8',
                cyan: '#00e5ff',
                white: '#f0f6fc',
                brightGreen: '#7ee787',
                brightYellow: '#ffea7f',
                brightCyan: '#56d4dd'
            },
            convertEol: true,
            scrollback: 1000
        });

        fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.open(terminalContainer);

        setTimeout(() => {
            if (fitAddon) {
                try { fitAddon.fit(); } catch (e) {}
            }
        }, 120);

        term.onData((data) => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(data);
            }
        });

        term.onResize(({ cols, rows }) => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'resize', cols, rows }));
            }
        });

        connectWebSocket();
    }

    function connectWebSocket() {
        if (ws) {
            try { ws.close(); } catch (e) {}
        }

        isConnecting = true;
        connectionError = '';

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/terminal/ws`;

        try {
            ws = new WebSocket(wsUrl);
            ws.binaryType = 'arraybuffer';

            ws.onopen = () => {
                isConnected = true;
                isConnecting = false;
                connectionError = '';
                term.clear();
                term.writeln('\x1b[38;5;82m╭───────────────────────────────────────────────╮\x1b[0m');
                term.writeln('\x1b[38;5;82m│\x1b[0m \x1b[1;38;5;220mPROJECTX ROOT SHELL\x1b[0m (\x1b[38;5;51mInteractive PTY\x1b[0m)        \x1b[38;5;82m│\x1b[0m');
                term.writeln('\x1b[38;5;82m│\x1b[0m Siap menerima perintah (ketik atau pilih chip)\x1b[38;5;82m│\x1b[0m');
                term.writeln('\x1b[38;5;82m╰───────────────────────────────────────────────╯\x1b[0m\r\n');

                if (term && fitAddon) {
                    try {
                        fitAddon.fit();
                        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
                    } catch (e) {}
                }
            };

            ws.onmessage = (event) => {
                if (event.data instanceof ArrayBuffer) {
                    const text = new TextDecoder().decode(event.data);
                    term.write(text);
                } else {
                    term.write(event.data);
                }
            };

            ws.onclose = () => {
                isConnected = false;
                isConnecting = false;
            };

            ws.onerror = (err) => {
                isConnected = false;
                isConnecting = false;
                connectionError = 'WebSocket PTY tidak terhubung. Mode Direct HTTP Runner aktif otomatis.';
            };
        } catch (e) {
            isConnecting = false;
            connectionError = e.message;
        }
    }

    function sendKey(seq) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(seq);
            term.focus();
        } else if (seq === '\x03') {
            term.writeln('^C');
        }
    }

    async function sendCommand(cmd) {
        if (!cmd) return;

        // If WebSocket is active, send to PTY
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(cmd + '\n');
            term.focus();
            return;
        }

        // Fallback: Execute via HTTP Exec API directly
        isExecRunning = true;
        term.writeln(`\r\n\x1b[38;5;220mroot@projectx:~# ${cmd}\x1b[0m`);
        try {
            const res = await fetch('/api/terminal/exec', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: cmd, timeout: 30 })
            });
            const data = await res.json();
            if (data.stdout) {
                term.write(data.stdout.replace(/\n/g, '\r\n'));
            }
            if (data.stderr) {
                term.writeln(`\x1b[31m${data.stderr.replace(/\n/g, '\r\n')}\x1b[0m`);
            }
            if (data.exit_code === 0) {
                term.writeln(`\x1b[32m[Selesai dalam ${data.duration_ms}ms]\x1b[0m`);
            } else {
                term.writeln(`\x1b[31m[Exit Code: ${data.exit_code}]\x1b[0m`);
            }
        } catch (e) {
            term.writeln(`\x1b[31m[Gagal mengeksekusi]: ${e.message}\x1b[0m`);
        } finally {
            isExecRunning = false;
        }
    }

    function handleMobileSubmit() {
        if (!mobileInput.trim()) return;
        sendCommand(mobileInput.trim());
        mobileInput = '';
    }

    function runPreset(p) {
        if (p.confirm) {
            modal = {
                show: true,
                title: 'Konfirmasi Perintah',
                message: `Jalankan perintah <b>${p.name}</b>?<br><br><code style="background:var(--panel-alt);padding:4px 8px;border-radius:4px;display:block;font-family:monospace;font-size:11px;border:1px solid var(--line);">${p.cmd}</code>`,
                warning: p.desc || 'Pastikan Anda memahami efek dari perintah ini.',
                type: 'confirm',
                confirmText: 'Eksekusi',
                confirmIcon: 'fa-bolt',
                confirmAction: () => {
                    modal.show = false;
                    sendCommand(p.cmd);
                }
            };
        } else {
            sendCommand(p.cmd);
        }
    }

    function clearTerminal() {
        if (term) {
            term.clear();
            term.focus();
        }
    }

    function toggleFullscreen() {
        isFullscreen = !isFullscreen;
        tick().then(() => {
            if (fitAddon) {
                setTimeout(() => fitAddon.fit(), 150);
            }
        });
    }

    function handleResize() {
        if (fitAddon) {
            try {
                fitAddon.fit();
                if (ws && ws.readyState === WebSocket.OPEN && term) {
                    ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
                }
            } catch (e) {}
        }
    }

    onMount(() => {
        initTerminal();
        window.addEventListener('resize', handleResize);
    });

    onDestroy(() => {
        window.removeEventListener('resize', handleResize);
        if (ws) {
            try { ws.close(); } catch (e) {}
        }
        if (term) {
            try { term.dispose(); } catch (e) {}
        }
    });
</script>

<div class="wrap {isFullscreen ? 'fullscreen-mode' : ''}">
    <!-- HEADER -->
    <div class="header">
        <div class="header-left">
            <div class="eyebrow">
                <span class="live-dot {isConnected ? 'is-connected' : isConnecting ? 'is-connecting' : 'is-disconnected'}"></span>
                Shell Konsol Android
            </div>
            <h1>Web Terminal</h1>
        </div>
        <div class="header-actions">
            <button
                class="btn-icon"
                on:click={connectWebSocket}
                title="Hubungkan Ulang"
                disabled={isConnecting}
            >
                <i class="fa-solid fa-rotate-right {isConnecting ? 'fa-spin' : ''}"></i>
            </button>
            <button
                class="btn-icon"
                on:click={clearTerminal}
                title="Bersihkan Layar"
            >
                <i class="fa-solid fa-broom"></i>
            </button>
            <button
                class="btn-icon {isFullscreen ? 'btn-active' : ''}"
                on:click={toggleFullscreen}
                title={isFullscreen ? 'Keluar Layar Penuh' : 'Layar Penuh'}
            >
                <i class="fa-solid {isFullscreen ? 'fa-compress' : 'fa-expand'}"></i>
            </button>
        </div>
    </div>

    {#if connectionError}
        <div class="alert alert-warning">
            <i class="fa-solid fa-circle-info"></i>
            <span>{connectionError}</span>
            <button class="btn-retry" on:click={connectWebSocket}>Hubungkan Ulang</button>
        </div>
    {/if}

    <!-- TERMINAL WINDOW CARD -->
    <div class="terminal-card">
        <!-- WINDOW TITLEBAR -->
        <div class="window-titlebar">
            <div class="window-dots">
                <span class="dot dot-red"></span>
                <span class="dot dot-yellow"></span>
                <span class="dot dot-green"></span>
            </div>
            <div class="window-title">
                <i class="fa-solid fa-terminal" style="color:var(--lime);"></i>
                <span>root@projectx:~#</span>
                {#if isConnected}
                    <span class="badge badge-online">PTY LIVE</span>
                {:else if isConnecting}
                    <span class="badge badge-connecting">CONNECTING...</span>
                {:else}
                    <span class="badge badge-http">DIRECT RUNNER</span>
                {/if}
            </div>
            <div class="window-tools">
                <button
                    class="tool-btn {showVirtualKeys ? 'active' : ''}"
                    on:click={() => showVirtualKeys = !showVirtualKeys}
                    title="Toggle Tombol Bantuan"
                >
                    <i class="fa-solid fa-keyboard"></i>
                </button>
            </div>
        </div>

        <!-- VIRTUAL KEY HELPER BAR (TOUCH/MOBILE FRIENDLY) -->
        {#if showVirtualKeys}
            <div class="virtual-keybar">
                <button type="button" class="vk-btn vk-ctrl" on:click={() => sendKey('\x03')} title="Kirim SIGINT">CTRL+C</button>
                <button type="button" class="vk-btn" on:click={() => sendKey('\t')}>TAB</button>
                <button type="button" class="vk-btn" on:click={() => sendKey('\x1b')}>ESC</button>
                <button type="button" class="vk-btn" on:click={() => sendKey('\x1b[A')}>▲ UP</button>
                <button type="button" class="vk-btn" on:click={() => sendKey('\x1b[B')}>▼ DOWN</button>
                <button type="button" class="vk-btn" on:click={() => sendKey('\x0c')}>CLEAR</button>
                <button type="button" class="vk-btn" on:click={() => sendKey('|')}>|</button>
                <button type="button" class="vk-btn" on:click={() => sendKey(' && ')}>&&</button>
                <button type="button" class="vk-btn" on:click={() => sendKey('/')}>/</button>
            </div>
        {/if}

        <!-- XTERM CONTAINER -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
            class="terminal-viewport {isFullscreen ? 'viewport-fullscreen' : ''}"
            bind:this={terminalContainer}
            on:click={() => term && term.focus()}
        ></div>

        <!-- INPUT BAR -->
        <form class="terminal-input-bar" on:submit|preventDefault={handleMobileSubmit}>
            <div class="input-prompt">#</div>
            <input
                type="text"
                class="command-input"
                bind:value={mobileInput}
                placeholder="Ketik perintah (contoh: ping 1.1.1.1, iptables -L)..."
                autocomplete="off"
                autocorrect="off"
                autocapitalize="off"
                spellcheck="false"
            />
            <button type="submit" class="btn-send" disabled={isExecRunning}>
                <i class="fa-solid {isExecRunning ? 'fa-spinner fa-spin' : 'fa-paper-plane'}"></i>
            </button>
        </form>
    </div>

    <!-- QUICK PRESET CHIPS (SLEEK COMPACT HORIZONTAL SCROLL) -->
    <div class="chips-container">
        <div class="chips-label">
            <i class="fa-solid fa-bolt" style="color:var(--yellow);"></i>
            <span>Pilihan Cepat:</span>
        </div>
        <div class="chips-scroll">
            {#each presets as p}
                <button
                    type="button"
                    class="chip-pill {p.confirm ? 'chip-danger' : ''}"
                    on:click={() => runPreset(p)}
                    title="{p.cmd} - {p.desc}"
                >
                    <i class="fa-solid {p.icon}"></i>
                    <span>{p.name}</span>
                </button>
            {/each}
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
        padding: 14px;
        max-width: 820px;
        margin: 0 auto;
        padding-bottom: 90px;
        box-sizing: border-box;
    }

    .wrap.fullscreen-mode {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 9999;
        background: var(--bg);
        max-width: 100vw;
        height: 100vh;
        padding: 10px;
        box-sizing: border-box;
        overflow-y: auto;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        gap: 10px;
    }

    .eyebrow {
        font-size: 0.7rem;
        font-weight: 800;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--lime);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 2px;
    }

    .live-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        display: inline-block;
    }

    .live-dot.is-connected {
        background: var(--lime);
        box-shadow: 0 0 6px var(--lime);
    }

    .live-dot.is-connecting {
        background: var(--yellow);
        box-shadow: 0 0 6px var(--yellow);
    }

    .live-dot.is-disconnected {
        background: var(--muted);
    }

    h1 {
        font-size: 1.5rem;
        font-weight: 900;
        letter-spacing: -0.03em;
        margin: 0;
        color: var(--ink);
        text-transform: uppercase;
    }

    .header-actions {
        display: flex;
        gap: 6px;
        align-items: center;
    }

    .btn-icon {
        width: 36px;
        height: 36px;
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 8px;
        color: var(--ink);
        box-shadow: 2px 2px 0 var(--line);
        cursor: pointer;
        display: grid;
        place-items: center;
        font-size: 13px;
        transition: .15s;
    }

    .btn-icon:hover {
        background: var(--yellow);
        color: #000;
        transform: translate(-1px, -1px);
    }

    .btn-icon.btn-active {
        background: var(--pink);
        color: #fff;
    }

    /* ALERTS */
    .alert {
        padding: 8px 12px;
        border-radius: 8px;
        font-size: 0.75rem;
        font-weight: 800;
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 12px;
        border: 2px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
    }

    .alert-warning {
        background: rgba(255, 225, 74, 0.15);
        color: var(--yellow);
        border-color: var(--yellow);
    }

    .btn-retry {
        margin-left: auto;
        padding: 3px 8px;
        background: var(--yellow);
        color: #000;
        border: 1px solid #000;
        border-radius: 4px;
        font-weight: 900;
        font-size: 10px;
        cursor: pointer;
        flex-shrink: 0;
    }

    /* TERMINAL CARD */
    .terminal-card {
        background: #0d0b11;
        border: 2px solid var(--line);
        border-radius: 14px;
        box-shadow: var(--shadow);
        overflow: hidden;
        margin-bottom: 14px;
        display: flex;
        flex-direction: column;
        width: 100%;
        max-width: 100%;
        box-sizing: border-box;
    }

    .window-titlebar {
        background: #181520;
        border-bottom: 2px solid var(--line);
        padding: 8px 12px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
    }

    .window-dots {
        display: flex;
        gap: 5px;
    }

    .dot {
        width: 9px;
        height: 9px;
        border-radius: 50%;
        border: 1px solid rgba(0,0,0,0.4);
    }

    .dot-red { background: #ff5c77; }
    .dot-yellow { background: #ffe14a; }
    .dot-green { background: #39ff14; }

    .window-title {
        font-family: 'JetBrains Mono', monospace;
        font-size: 11px;
        font-weight: 900;
        color: #fffaf4;
        display: flex;
        align-items: center;
        gap: 6px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .badge {
        font-size: 8px;
        font-weight: 900;
        padding: 2px 5px;
        border-radius: 4px;
        letter-spacing: 0.05em;
    }

    .badge-online { background: var(--lime); color: #000; }
    .badge-connecting { background: var(--yellow); color: #000; }
    .badge-http { background: var(--cyan); color: #000; }

    .window-tools {
        display: flex;
        gap: 6px;
    }

    .tool-btn {
        background: #24202b;
        border: 1.5px solid var(--line);
        color: #e2e8f0;
        width: 26px;
        height: 26px;
        border-radius: 6px;
        display: grid;
        place-items: center;
        font-size: 11px;
        cursor: pointer;
        transition: .1s;
    }

    .tool-btn.active, .tool-btn:hover {
        background: var(--yellow);
        color: #000;
    }

    /* VIRTUAL KEY BAR */
    .virtual-keybar {
        background: #14111a;
        border-bottom: 1.5px solid var(--line);
        padding: 6px 8px;
        display: flex;
        gap: 5px;
        overflow-x: auto;
        white-space: nowrap;
        scrollbar-width: none;
    }

    .virtual-keybar::-webkit-scrollbar {
        display: none;
    }

    .vk-btn {
        background: #221d2b;
        border: 1.5px solid var(--line);
        color: #f1f5f9;
        font-family: 'JetBrains Mono', monospace;
        font-size: 10px;
        font-weight: 800;
        padding: 4px 8px;
        border-radius: 5px;
        cursor: pointer;
        transition: .1s;
        flex-shrink: 0;
    }

    .vk-btn:hover {
        background: var(--yellow);
        color: #000;
    }

    .vk-ctrl {
        background: rgba(255, 92, 168, 0.2);
        color: var(--pink);
        border-color: var(--pink);
    }

    /* XTERM VIEWPORT */
    .terminal-viewport {
        height: 340px;
        padding: 8px 10px;
        background: #0d0b11;
        overflow: hidden;
        width: 100%;
        max-width: 100%;
        box-sizing: border-box;
    }

    .viewport-fullscreen {
        height: calc(100vh - 190px) !important;
    }

    /* COMMAND INPUT BAR */
    .terminal-input-bar {
        background: #181520;
        border-top: 2px solid var(--line);
        padding: 6px 10px;
        display: flex;
        gap: 8px;
        align-items: center;
    }

    .input-prompt {
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        font-weight: 900;
        color: var(--lime);
        flex-shrink: 0;
    }

    .command-input {
        flex: 1;
        background: #0d0b11;
        border: 1.5px solid var(--line);
        border-radius: 6px;
        padding: 7px 10px;
        color: #fffaf4;
        font-family: 'JetBrains Mono', monospace;
        font-size: 11px;
        outline: none;
        min-width: 0;
    }

    .command-input:focus {
        border-color: var(--yellow);
    }

    .btn-send {
        background: var(--yellow);
        color: #000;
        border: 1.5px solid var(--line);
        border-radius: 6px;
        width: 32px;
        height: 32px;
        cursor: pointer;
        display: grid;
        place-items: center;
        flex-shrink: 0;
        font-size: 12px;
        transition: .1s;
    }

    .btn-send:hover {
        transform: translate(-1px, -1px);
    }

    /* QUICK PRESET CHIPS */
    .chips-container {
        display: flex;
        flex-direction: column;
        gap: 6px;
        width: 100%;
        max-width: 100%;
        box-sizing: border-box;
        overflow: hidden;
    }

    .chips-label {
        font-size: 10px;
        font-weight: 850;
        text-transform: uppercase;
        color: var(--muted);
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .chips-scroll {
        display: flex;
        gap: 6px;
        overflow-x: auto;
        padding-bottom: 4px;
        scrollbar-width: none;
        width: 100%;
        max-width: 100%;
        box-sizing: border-box;
    }

    .chips-scroll::-webkit-scrollbar {
        display: none;
    }

    .chip-pill {
        background: var(--paper);
        border: 1.5px solid var(--line);
        color: var(--ink);
        padding: 5px 10px;
        border-radius: 20px;
        font-size: 10px;
        font-weight: 800;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 5px;
        flex-shrink: 0;
        box-shadow: 2px 2px 0 var(--line);
        transition: .1s;
        white-space: nowrap;
    }

    .chip-pill:hover {
        background: var(--yellow);
        color: #000;
        transform: translate(-1px, -1px);
    }

    .chip-danger {
        border-color: var(--pink);
    }

    .chip-danger:hover {
        background: var(--pink);
        color: #fff;
    }

    /* UNIVERSAL POPUP MODAL */
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
</style>
