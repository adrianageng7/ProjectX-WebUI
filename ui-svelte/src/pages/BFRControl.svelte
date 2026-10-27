<script>
    import { onMount, onDestroy } from 'svelte';

    let logs = [];
    let loading = true;
    let error = null;
    let executing = false;
    let timer = null;
    let logContainer;

    async function fetchLogs() {
        try {
            const res = await fetch('/api/bfr/logs');
            const data = await res.json();
            if (data.status === 'success') {
                logs = data.logs || [];
                error = null;
                setTimeout(() => {
                    if (logContainer) logContainer.scrollTop = logContainer.scrollHeight;
                }, 50);
            } else {
                error = data.message;
            }
        } catch (e) {
            error = 'Gagal memuat log.';
        } finally {
            loading = false;
        }
    }

    async function execAction(act) {
        executing = true;
        try {
            await fetch('/api/bfr/control', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: act })
            });
            setTimeout(fetchLogs, 1000);
            setTimeout(fetchLogs, 2500);
            setTimeout(fetchLogs, 4000);
        } catch (e) {
            console.error(e);
        } finally {
            setTimeout(() => { executing = false; }, 2500);
        }
    }

    onMount(() => {
        fetchLogs();
        timer = setInterval(fetchLogs, 4000);
    });

    onDestroy(() => {
        if (timer) clearInterval(timer);
    });
</script>

<div class="wrap">
    <div class="header">
        <div class="header-left">
            <div class="eyebrow"><span class="live-dot"></span> Box For Root</div>
            <h1>BFR Control</h1>
            <div class="desc">Kontrol service tunneling dan proxy Box For Root.</div>
        </div>
        <button class="btn btn-refresh" on:click={fetchLogs} title="Refresh Logs">
            <i class="fa-solid fa-rotate-right"></i>
        </button>
    </div>

    <div class="card">
        <div class="btn-group">
            <button class="btn btn-start" disabled={executing} on:click={() => execAction('start')}>
                <i class="fa-solid fa-play"></i> Start
            </button>
            <button class="btn btn-restart" disabled={executing} on:click={() => execAction('restart')}>
                <i class="fa-solid fa-rotate"></i> Restart
            </button>
            <button class="btn btn-stop" disabled={executing} on:click={() => execAction('stop')}>
                <i class="fa-solid fa-stop"></i> Stop
            </button>
        </div>
    </div>

    <div class="card-terminal">
        <div class="terminal-header">
            <span><i class="fa-solid fa-terminal"></i> runs.log</span>
            <span class="live-badge"><i class="fa-solid fa-circle-check"></i> Live</span>
        </div>
        <div class="log-terminal" bind:this={logContainer}>
            {#if loading}
                <div class="loading-box"><i class="fa-solid fa-spinner fa-spin"></i> Memuat log...</div>
            {:else if error}
                <div class="loading-box" style="color:var(--pink);">
                    <i class="fa-solid fa-triangle-exclamation"></i> {error}
                </div>
            {:else if logs.length === 0}
                <div class="loading-box">Log kosong.</div>
            {:else}
                {#each logs as l}
                    <div class="log-entry">{l}</div>
                {/each}
            {/if}
        </div>
    </div>
</div>

<style>
    .wrap {
        max-width: 800px;
        margin: 0 auto;
        padding: 16px;
        display: flex;
        flex-direction: column;
        height: 100%;
    }

    .header {
        margin-bottom: 16px;
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
    }

    .eyebrow {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--cyan);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 4px;
    }

    .live-dot {
        width: 8px;
        height: 8px;
        background: var(--cyan);
        border-radius: 50%;
        animation: blink 1.5s infinite;
    }
    @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }

    h1 {
        font-size: 28px;
        font-weight: 950;
        letter-spacing: -1.5px;
        text-transform: uppercase;
        line-height: 1;
    }

    .desc {
        font-size: 12px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 6px;
    }

    .card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 16px;
        padding: 16px;
        box-shadow: var(--shadow);
        margin-bottom: 16px;
    }

    .btn-group {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 10px;
    }

    .btn {
        border: 2px solid var(--line);
        border-radius: 10px;
        padding: 12px;
        font-size: 13px;
        font-weight: 950;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        transition: all 0.1s ease;
        box-shadow: 3px 3px 0px var(--line);
        text-transform: uppercase;
        font-family: inherit;
    }
    .btn:hover { transform: translate(-2px, -2px); box-shadow: 5px 5px 0px var(--line); filter: brightness(1.1); }
    .btn:active { transform: translate(2px, 2px); box-shadow: none; }
    .btn:disabled { opacity: 0.5; pointer-events: none; }

    .btn-start   { background: #b8f34a; color: #000; }
    .btn-stop    { background: #ff5ca8; color: #fff; }
    .btn-restart { background: #ffe14a; color: #000; }
    .btn-refresh { background: var(--paper); color: var(--ink); padding: 8px 12px; }

    .card-terminal {
        flex: 1;
        display: flex;
        flex-direction: column;
        min-height: 350px;
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 16px;
        box-shadow: var(--shadow);
        overflow: hidden;
    }

    .terminal-header {
        padding: 12px 16px;
        background: var(--panel-alt);
        border-bottom: 2px solid var(--line);
        font-size: 11px;
        font-weight: 900;
        color: var(--muted);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .live-badge { color: var(--cyan); }

    .log-terminal {
        flex: 1;
        overflow-y: auto;
        padding: 14px;
        background: #08070a;
        color: #fffaf4;
        font-family: var(--mono);
        font-size: 11px;
        line-height: 1.5;
    }
    :global(:root[data-theme="light"]) .log-terminal {
        background: #ffffff;
        color: #17151a;
    }

    .log-entry { margin-bottom: 4px; word-break: break-all; }
    .loading-box { text-align: center; padding: 20px; color: var(--muted); font-weight: 800; text-transform: uppercase; }
</style>
