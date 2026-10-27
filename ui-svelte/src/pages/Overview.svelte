<script>
    import { onMount, onDestroy } from 'svelte';
    import { router } from 'tinro';

    let data = {
        sys: { device: '—', os: '—', uptime: '—', kernel: '—', root: '—' },
        cpu: { model: '—', cores: 0, freq: '—', usage: 0, temp: 0, history: [4,4,4,4,4] },
        ram: { used: '—', total: '—', raw_lvl: 0, swap_raw: 0 },
        bat: { raw_lvl: 0, status: '—', temp: 0 },
        gpu: { model: '—', gl: '—', freq: '—' },
        net: { list: [] }
    };

    let timer = null;

    async function fetchData() {
        try {
            const res = await fetch('/api/sysinfo?_=' + Date.now());
            if (res.ok) {
                const json = await res.json();
                data = json;
            }
        } catch (e) {
            console.warn('sysinfo fetch error:', e);
        }
    }

    onMount(() => {
        fetchData();
        timer = setInterval(fetchData, 3000);
    });

    onDestroy(() => {
        if (timer) clearInterval(timer);
    });
</script>

<div class="wrap">
    <header class="header">
        <div class="brand">
            <div class="eyebrow"><span class="live-dot"></span> Live System Monitor</div>
            <h1>Overview</h1>
            <div class="device-name">{data.sys?.device || 'Unknown Device'}</div>
        </div>
        <div class="controls">
            <button class="btn" on:click={fetchData} title="Refresh">
                <i class="fa-solid fa-rotate-right"></i>
            </button>
        </div>
    </header>

    <!-- DEVICE CARD -->
    <section class="device-card">
        <div>
            <div class="kpi-label">Device</div>
            <div class="device-model-text">{data.sys?.device || '—'}</div>
            <div class="device-meta">
                <span class="chip"><i class="fa-solid fa-mobile-screen"></i> <span>{data.sys?.device || '—'}</span></span>
                <span class="chip"><i class="fa-brands fa-android"></i> <span>{data.sys?.os || '—'}</span></span>
                <span class="chip"><i class="fa-solid fa-microchip"></i> <span>{data.cpu?.model || '—'}</span></span>
            </div>
        </div>
        <div class="uptime">
            <div class="uptime-label">Uptime</div>
            <div class="uptime-value">{data.sys?.uptime || '—'}</div>
        </div>
    </section>

    <!-- KPI CARDS -->
    <section class="grid">
        <!-- MEMORY -->
        <div class="card kpi-card kpi-mem">
            <div>
                <div class="kpi-top">
                    <div class="kpi-label">Memory</div>
                    <div class="kpi-icon"><i class="fa-solid fa-memory"></i></div>
                </div>
                <div class="kpi-val">{data.ram?.used || '—'}</div>
                <div class="kpi-sub">/ {data.ram?.total || '—'} · Swap {data.ram?.swap_raw ?? 0}%</div>
            </div>
            <div class="progress"><i style="width: {data.ram?.raw_lvl ?? 0}%"></i></div>
        </div>

        <!-- BATTERY -->
        <div class="card kpi-card kpi-bat clickable-card" on:click={() => router.goto('/battery')} role="button" tabindex="0" title="Buka Pengaturan Proteksi Baterai">
            <div>
                <div class="kpi-top">
                    <div class="kpi-label">
                        <span>Battery</span>
                        {#if data.bat?.auto_cutoff_enabled}
                            <span class="bat-badge {data.bat?.is_cutoff ? 'badge-cutoff' : 'badge-charging'}">
                                <i class="fa-solid {data.bat?.is_cutoff ? 'fa-shield-halved' : 'fa-bolt'}"></i>
                                {data.bat?.is_cutoff ? 'Cut-off' : 'Auto'}
                            </span>
                        {/if}
                    </div>
                    <div class="kpi-icon {data.bat?.is_cutoff ? 'icon-cutoff' : ''}">
                        <i class="fa-solid {data.bat?.is_cutoff ? 'fa-shield-halved' : (data.bat?.status === 'Charging' ? 'fa-bolt' : 'fa-battery-half')}"></i>
                    </div>
                </div>
                <div class="kpi-val">
                    {data.bat?.raw_lvl ?? '—'}%
                    {#if data.bat?.auto_cutoff_enabled}
                        <span class="target-range" title="Batas Pengisian: {data.bat.start_level}% - {data.bat.stop_level}%">
                            {data.bat.start_level}%-{data.bat.stop_level}%
                        </span>
                    {/if}
                </div>
                <div class="kpi-sub">
                    <span class="kpi-sub-status {data.bat?.is_cutoff ? 'text-cutoff' : ''}">
                        {(data.bat?.display_status || data.bat?.status || 'Unknown')}
                    </span> · {(data.bat?.temp ?? '?')}°C
                </div>
            </div>
            <div class="progress battery-progress {data.bat?.is_cutoff ? 'progress-cutoff' : ''}">
                <i style="width: {data.bat?.raw_lvl ?? 0}%"></i>
            </div>
        </div>

        <!-- CPU -->
        <div class="card kpi-card kpi-cpu">
            <div>
                <div class="kpi-top">
                    <div class="kpi-label">CPU Usage</div>
                    <div class="kpi-icon"><i class="fa-solid fa-microchip"></i></div>
                </div>
                <div class="kpi-val">
                    {typeof data.cpu?.usage === 'number' ? data.cpu.usage.toFixed(1) : (data.cpu?.usage || '—')}%
                </div>
                <div class="kpi-sub">{(data.cpu?.cores ?? '?')} cores · {(data.cpu?.freq || 'Dynamic')}</div>
            </div>
            <div class="chart-box">
                {#if Array.isArray(data.cpu?.history)}
                    {#each data.cpu.history as h}
                        <div class="chart-bar" style="height: {Math.max(4, h)}%"></div>
                    {/each}
                {:else}
                    <div class="chart-bar" style="height: 10%"></div>
                    <div class="chart-bar" style="height: 20%"></div>
                    <div class="chart-bar" style="height: 15%"></div>
                {/if}
            </div>
        </div>

        <!-- GPU -->
        <div class="card kpi-card kpi-gpu">
            <div>
                <div class="kpi-top">
                    <div class="kpi-label">GPU</div>
                    <div class="kpi-icon"><i class="fa-solid fa-display"></i></div>
                </div>
                <div class="kpi-val" style="font-size:16px">{data.gpu?.model || '—'}</div>
                <div class="kpi-sub">{(data.gpu?.gl || 'OpenGL')} · {(data.gpu?.freq || 'Dynamic')}</div>
            </div>
            <div class="chart-box">
                <div class="chart-bar" style="height:32%"></div>
                <div class="chart-bar" style="height:54%"></div>
                <div class="chart-bar" style="height:25%"></div>
                <div class="chart-bar" style="height:68%"></div>
                <div class="chart-bar" style="height:46%"></div>
            </div>
        </div>
    </section>

    <!-- NETWORK & SYSTEM DETAILS -->
    <section class="container-grid">
        <!-- NETWORK -->
        <div class="card section-card">
            <div class="section-head">
                <div class="section-title"><i class="fa-solid fa-network-wired"></i> Network</div>
                <div class="section-caption">LIVE INTERFACES</div>
            </div>
            <table class="info-table">
                <thead>
                    <tr>
                        <th style="text-align:left">Interface</th>
                        <th class="text-right">Upload</th>
                        <th class="text-right">Download</th>
                    </tr>
                </thead>
                <tbody>
                    {#if !data.net?.list || data.net.list.length === 0}
                        <tr><td colspan="3" style="color:var(--muted);font-size:10px;padding:12px 0;text-align:center">No active interface</td></tr>
                    {:else}
                        {#each data.net.list as n}
                            <tr>
                                <td>
                                    <div class="iface">
                                        <div class="iface-icon">
                                            <i class="fa-solid {n.name.startsWith('wlan') ? 'fa-wifi' : n.name.startsWith('rmnet') || n.name.startsWith('ccmni') ? 'fa-signal' : 'fa-ethernet'}"></i>
                                        </div>
                                        <div>
                                            <div class="iface-name">{n.name}</div>
                                            <div class="ip">IP: {n.ip || '-'}</div>
                                        </div>
                                    </div>
                                </td>
                                <td class="text-right">{n.tx_total || '—'}</td>
                                <td class="text-right">{n.rx_total || '—'}</td>
                            </tr>
                        {/each}
                    {/if}
                </tbody>
            </table>
        </div>

        <!-- SYSTEM INFO -->
        <div class="card section-card">
            <div class="section-head">
                <div class="section-title"><i class="fa-solid fa-circle-info"></i> System Info</div>
                <div class="section-caption">DEVICE STATUS</div>
            </div>
            <div class="system-list">
                <div class="system-row">
                    <div class="system-key">Provider</div>
                    <div class="system-value accent"><i class="fa-solid fa-sim-card"></i> {data.sys?.provider || '—'}</div>
                </div>
                <div class="system-row">
                    <div class="system-key">Sinyal</div>
                    <div class="system-value green"><i class="fa-solid fa-signal"></i> {data.sys?.signal || '—'}</div>
                </div>
                <div class="system-row">
                    <div class="system-key">OS</div>
                    <div class="system-value">{data.sys?.os || '—'}</div>
                </div>
                <div class="system-row">
                    <div class="system-key">Kernel</div>
                    <div class="system-value kernel-val">{data.sys?.kernel || '—'}</div>
                </div>
                <div class="system-row">
                    <div class="system-key">Uptime</div>
                    <div class="system-value green">{data.sys?.uptime || '—'}</div>
                </div>
                <div class="system-row">
                    <div class="system-key">CPU Temp</div>
                    <div class="system-value">{(data.cpu?.temp ?? '—')}°C</div>
                </div>
                <div class="system-row">
                    <div class="system-key">Root</div>
                    <div class="system-value red">{data.sys?.root || '—'}</div>
                </div>
            </div>
        </div>
    </section>

</div>

<style>
    .wrap {
        max-width: 900px;
        margin: 0 auto;
        padding: 16px;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        margin-bottom: 16px;
    }

    .eyebrow {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
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
    @keyframes blink { 0%,100%{opacity:1} 50%{opacity:0.3} }

    h1 {
        font-size: 28px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: -1.5px;
        line-height: 1;
    }

    .device-name {
        font-size: 12px;
        font-weight: 700;
        color: var(--muted);
        margin-top: 4px;
    }

    .btn {
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 8px 12px;
        font-size: 13px;
        font-weight: 900;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        transition: all 0.1s ease;
        box-shadow: 3px 3px 0px var(--line);
        text-transform: uppercase;
        background: var(--paper);
        color: var(--ink);
    }
    .btn:hover { transform: translate(-2px,-2px); box-shadow: 5px 5px 0 var(--line); }
    .btn:active { transform: translate(2px,2px); box-shadow: none; }

    /* DEVICE CARD */
    .device-card {
        position: relative;
        background: var(--yellow);
        color: #17151a;
        border: 2px solid var(--line);
        border-radius: 16px;
        padding: 16px;
        margin-bottom: 16px;
        box-shadow: var(--shadow);
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .device-model-text {
        font-size: 20px;
        font-weight: 950;
        margin-top: 4px;
        letter-spacing: -0.5px;
    }

    .device-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin-top: 10px;
    }

    .chip {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        background: var(--paper);
        color: var(--ink);
        border: 2px solid var(--line);
        border-radius: 999px;
        box-shadow: 2px 2px 0 var(--line);
        font-size: 9px;
        font-weight: 900;
    }

    .uptime { text-align: right; }
    .uptime-label { font-size: 9px; font-weight: 950; letter-spacing: 1px; text-transform: uppercase; }
    .uptime-value { margin-top: 4px; font-size: 22px; line-height: 1; font-weight: 950; letter-spacing: -1px; }

    /* KPI CARDS */
    .grid {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 12px;
        margin-bottom: 16px;
    }
    @media (max-width: 768px) {
        .grid { grid-template-columns: repeat(2, 1fr); }
    }

    .card {
        background: var(--paper);
        border: 2px solid var(--line);
        color: var(--ink);
        box-shadow: var(--shadow);
    }

    .kpi-card {
        position: relative;
        min-height: 150px;
        padding: 14px;
        border-radius: 14px;
        display: flex;
        flex-direction: column;
        justify-content: space-between;
    }

    .kpi-top { display: flex; justify-content: space-between; align-items: flex-start; }
    .kpi-label { color: var(--muted); font-size: 9px; font-weight: 950; letter-spacing: .9px; text-transform: uppercase; }
    .kpi-icon {
        width: 28px; height: 28px;
        display: grid; place-items: center;
        border: 2px solid var(--line); border-radius: 8px;
        color: #17151a; box-shadow: 2px 2px 0 var(--line); font-size: 13px;
    }
    .kpi-mem .kpi-icon { background: var(--purple); }
    .kpi-bat .kpi-icon { background: var(--lime); }
    .kpi-cpu .kpi-icon { background: var(--pink); }
    .kpi-gpu .kpi-icon { background: var(--cyan); }

    .kpi-val { margin-top: 6px; font-size: 24px; line-height: 1; font-weight: 950; letter-spacing: -1px; }
    .kpi-sub { margin-top: 4px; color: var(--muted); font-size: 10px; font-weight: 700; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

    .progress {
        height: 10px; padding: 1px; margin-top: 10px;
        border: 2px solid var(--line); border-radius: 999px;
        background: var(--panel-alt);
    }
    .progress > i { display: block; height: 100%; border-radius: 999px; background: var(--purple); transition: width .4s; }
    .battery-progress > i { background: var(--lime); }

    .chart-box {
        height: 38px; display: flex; align-items: flex-end; gap: 3px;
        margin-top: 10px; padding: 2px;
        border: 2px solid var(--line); border-radius: 6px;
        background: var(--panel-alt);
    }
    .chart-bar { flex: 1; background: var(--pink); border-radius: 2px; min-height: 4px; }

    /* CONTAINER DETAILS */
    .container-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
    }
    @media (max-width: 768px) {
        .container-grid { grid-template-columns: 1fr; }
    }

    .section-card {
        border-radius: 14px;
        padding: 16px;
    }
    .section-head {
        display: flex; justify-content: space-between; align-items: center;
        margin-bottom: 12px; border-bottom: 2px dashed var(--line); padding-bottom: 8px;
    }
    .section-title { font-size: 13px; font-weight: 900; text-transform: uppercase; display: flex; align-items: center; gap: 6px; }
    .section-caption { font-size: 8px; font-weight: 900; color: var(--muted); letter-spacing: .5px; }

    .info-table { width: 100%; border-collapse: collapse; font-size: 11px; }
    .info-table th { color: var(--muted); font-size: 9px; font-weight: 900; text-transform: uppercase; padding: 6px 0; border-bottom: 1px solid var(--line); }
    .info-table td { padding: 8px 0; border-bottom: 1px solid rgba(255,255,255,0.05); font-weight: 700; }
    .text-right { text-align: right; }

    .iface { display: flex; align-items: center; gap: 8px; }
    .iface-icon {
        width: 24px; height: 24px; border-radius: 6px; border: 1.5px solid var(--line);
        background: var(--cyan); color: #000; display: grid; place-items: center; font-size: 11px;
    }
    .iface-name { font-weight: 900; font-size: 12px; }
    .ip { font-size: 9px; color: var(--muted); }

    .system-list { display: flex; flex-direction: column; gap: 9px; }
    .system-row {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 12px;
        font-size: 11px;
    }
    .system-key {
        font-weight: 800;
        color: var(--muted);
        text-transform: uppercase;
        flex-shrink: 0;
        letter-spacing: 0.3px;
    }
    .system-value {
        font-weight: 900;
        text-align: right;
        word-break: break-word;
        display: flex;
        align-items: center;
        gap: 5px;
        justify-content: flex-end;
    }
    .system-value.kernel-val {
        font-family: monospace;
        font-size: 10px;
        word-break: break-all;
        line-height: 1.3;
        display: block;
    }
    .system-value.accent { color: var(--cyan); }
    .system-value.green { color: var(--lime); }
    .system-value.red { color: var(--pink); }

    /* BATTERY AUTO CUT-OFF & MODAL */
    .clickable-card {
        cursor: pointer;
        transition: transform 0.15s ease, box-shadow 0.15s ease;
    }
    .clickable-card:hover {
        transform: translateY(-2px);
        box-shadow: 4px 4px 0 var(--line);
    }
    .bat-badge {
        font-size: 9px;
        font-weight: 900;
        padding: 1px 6px;
        border-radius: 999px;
        border: 1.5px solid var(--line);
        display: inline-flex;
        align-items: center;
        gap: 3px;
        margin-left: 6px;
        vertical-align: middle;
        text-transform: uppercase;
    }
    .badge-cutoff { background: var(--cyan); color: #000; }
    .badge-charging { background: var(--lime); color: #000; }
    .badge-idle { background: var(--panel-alt); color: var(--muted); }
    .icon-cutoff { background: var(--cyan) !important; color: #000 !important; }
    .target-range {
        font-size: 11px;
        font-weight: 800;
        color: var(--muted);
        margin-left: 6px;
        letter-spacing: 0;
    }
    .text-cutoff { color: var(--cyan); font-weight: 900; }
    .progress-cutoff > i { background: var(--cyan) !important; }
</style>
