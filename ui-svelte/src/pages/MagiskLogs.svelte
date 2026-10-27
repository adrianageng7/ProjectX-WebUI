<script>
    import { onMount } from 'svelte';

    let logText = '';
    let loading = true;
    let error = null;
    let container;

    async function fetchLogs() {
        loading = true;
        error = null;
        try {
            const res = await fetch('/api/magisk_log?_=' + Date.now());
            const data = await res.json();
            if (data.status === 'success') {
                logText = data.log || 'Log kosong.';
                setTimeout(() => {
                    if (container) container.scrollTop = container.scrollHeight;
                }, 50);
            } else {
                error = data.message || 'Gagal membaca log';
            }
        } catch (e) {
            error = 'Gagal terhubung ke server';
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        fetchLogs();
    });
</script>

<div class="wrap">
    <div class="header">
        <div class="header-left">
            <div class="eyebrow"><span class="live-dot"></span> System Manager</div>
            <h1>Magisk Logs</h1>
            <div class="desc">Memeriksa log dari daemon Magisk / KernelSU</div>
        </div>
        <button class="btn" on:click={fetchLogs}><i class="fa-solid fa-rotate-right"></i> Refresh</button>
    </div>

    <div class="card">
        <div class="card-header">
            <i class="fa-solid fa-terminal"></i> /cache/magisk.log
        </div>
        <div class="log-terminal" bind:this={container}>
            {#if loading}
                <div class="loading"><i class="fa-solid fa-spinner fa-spin"></i> Membaca Log...</div>
            {:else if error}
                <div class="loading" style="color:var(--pink);"><i class="fa-solid fa-triangle-exclamation"></i> {error}</div>
            {:else}
                <pre>{logText}</pre>
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

    .btn {
        padding: 10px 14px;
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 8px;
        color: var(--ink);
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        cursor: pointer;
        box-shadow: 3px 3px 0 var(--line);
        transition: all 0.1s;
        font-family: inherit;
        font-size: 12px;
    }
    .btn:hover { background: var(--cyan); color: #000; transform: translate(-2px, -2px); box-shadow: 5px 5px 0 var(--line); }
    .btn:active { transform: translate(2px, 2px); box-shadow: none; }

    .card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 16px;
        padding: 16px;
        box-shadow: var(--shadow);
        display: flex;
        flex-direction: column;
        flex: 1;
        min-height: 380px;
    }

    .card-header {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 13px;
        font-weight: 900;
        text-transform: uppercase;
        margin-bottom: 12px;
        padding-bottom: 10px;
        border-bottom: 2px dashed var(--line);
        color: var(--ink);
    }
    .card-header i { color: var(--lime); }

    .log-terminal {
        flex: 1;
        background: #0a0a0a;
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 14px;
        overflow-y: auto;
        color: #39ff14;
        font-family: var(--mono);
        font-size: 11px;
        line-height: 1.5;
    }

    pre { margin: 0; white-space: pre-wrap; word-break: break-all; font-family: inherit; }

    .loading {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--muted);
        font-size: 13px;
        font-weight: 800;
        text-transform: uppercase;
        gap: 10px;
    }
</style>
