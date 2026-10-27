<script>
    import { onMount } from 'svelte';

    let modules = [];
    let loading = true;
    let error = null;

    async function fetchModules() {
        loading = true; error = null;
        try {
            const res = await fetch('/api/magisk/modules');
            const data = await res.json();
            if (data.status === 'success') {
                modules = data.data || [];
            } else {
                error = data.message;
            }
        } catch (e) {
            error = 'Gagal koneksi ke server API';
        }
        loading = false;
    }

    async function toggleModule(id, enable) {
        try {
            const res = await fetch('/api/magisk/modules/action', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({action: 'toggle', id, enable})
            });
            const data = await res.json();
            if(data.status !== 'success') {
                alert(data.message);
                fetchModules(); // revert
            } else {
                // optimistically update local state
                modules = modules.map(m => m.id === id ? {...m, enabled: enable} : m);
            }
        } catch(e) {
            alert('Error connecting');
        }
    }

    async function removeModule(id) {
        if(!confirm('Modul ini akan ditandai untuk dihapus saat reboot. Lanjutkan?')) return;
        try {
            const res = await fetch('/api/magisk/modules/action', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({action: 'remove', id})
            });
            const data = await res.json();
            if(data.status === 'success') fetchModules();
            else alert(data.message);
        } catch(e) {}
    }

    async function restoreModule(id) {
        try {
            const res = await fetch('/api/magisk/modules/action', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({action: 'restore', id})
            });
            const data = await res.json();
            if(data.status === 'success') fetchModules();
            else alert(data.message);
        } catch(e) {}
    }

    onMount(() => {
        fetchModules();
    });
</script>

<div class="wrap">
    <div class="header">
        <div class="header-left">
            <div class="eyebrow"><i class="fa-solid fa-puzzle-piece"></i> Extension</div>
            <h1>Magisk Modules</h1>
            <div class="desc">Kelola modul root Magisk secara Native.</div>
        </div>
        <button class="btn" on:click={fetchModules} title="Refresh"><i class="fa-solid fa-rotate-right"></i></button>
    </div>

    {#if loading}
        <div class="empty-state">
            <i class="fa-solid fa-spinner fa-spin"></i> Membaca Module.prop...
        </div>
    {:else if error}
        <div class="empty-state" style="color:var(--pink);">
            <i class="fa-solid fa-triangle-exclamation"></i> {error}
        </div>
    {:else if modules.length === 0}
        <div class="empty-state">
            <i class="fa-solid fa-box-open"></i> Tidak ada modul yang terinstall
        </div>
    {:else}
        <div class="module-list">
            {#each modules as m}
                <div class="module-card {m.enabled ? '' : 'disabled'} {m.pending_del ? 'pending-del' : ''}">
                    {#if m.pending_del}
                        <div class="badge-del">Akan Dihapus (Reboot)</div>
                    {/if}
                    
                    <div class="mod-header">
                        <div>
                            <div class="mod-title">
                                {m.name || m.id} 
                                <span class="mod-version">{m.version || 'v1.0'}</span>
                            </div>
                            <div class="mod-author">by {m.author || 'Unknown'}</div>
                        </div>
                    </div>
                    
                    <div class="mod-desc">{m.description || 'Tidak ada deskripsi.'}</div>
                    <div class="mod-id">ID: {m.id}</div>
                    
                    <div class="mod-actions">
                        <label class="brutal-switch" title={m.enabled ? 'Nonaktifkan' : 'Aktifkan'}>
                            <input type="checkbox" checked={m.enabled} disabled={m.pending_del} on:change={(e) => toggleModule(m.id, e.target.checked)}>
                            <span class="slider"></span>
                        </label>
                        
                        {#if m.pending_del}
                            <button class="btn btn-restore" on:click={() => restoreModule(m.id)}>
                                <i class="fa-solid fa-rotate-left"></i> Batal Hapus
                            </button>
                        {:else}
                            <button class="btn btn-trash" on:click={() => removeModule(m.id)}>
                                <i class="fa-solid fa-trash"></i> Hapus
                            </button>
                        {/if}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .wrap { max-width: 800px; margin: 0 auto; padding: 20px; }
    
    .header { margin-bottom: 20px; display: flex; justify-content: space-between; align-items: flex-end; }
    .eyebrow { font-size: 10px; font-weight: 900; text-transform: uppercase; color: var(--cyan); display: flex; align-items: center; gap: 6px; margin-bottom: 6px;}
    h1 { font-size: 28px; font-weight: 950; letter-spacing: -1.5px; text-transform: uppercase; line-height: 1; }
    .desc { font-size: 12px; font-weight: 700; color: var(--muted); margin-top: 8px; }

    .btn { border: 2px solid var(--line); border-radius: 8px; padding: 8px 12px; font-size: 13px; font-weight: 900; cursor: pointer; display: inline-flex; align-items: center; justify-content: center; gap: 8px; transition: all 0.1s ease; box-shadow: 3px 3px 0px var(--line); text-transform: uppercase; background: var(--paper); color: var(--ink); }
    .btn:hover { transform: translate(-2px, -2px); box-shadow: 5px 5px 0px var(--line); filter: brightness(1.1); }
    .btn:active { transform: translate(2px, 2px); box-shadow: none; }

    .module-list { display: flex; flex-direction: column; gap: 15px; }

    .module-card {
        background: var(--paper); border: 2px solid var(--line); border-radius: 12px;
        padding: 16px; box-shadow: var(--shadow);
        display: flex; flex-direction: column; gap: 12px; position: relative; transition: opacity 0.3s;
    }
    .module-card.disabled { opacity: 0.7; }
    .module-card.pending-del { border-color: var(--pink); border-style: dashed; }

    .mod-header { display: flex; justify-content: space-between; align-items: flex-start; }
    .mod-title { font-size: 16px; font-weight: 900; display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
    .mod-version { font-size: 11px; font-weight: 900; background: var(--panel-alt); padding: 2px 8px; border-radius: 50px; border: 1px solid var(--line); color: var(--cyan); }
    .mod-author { font-size: 11px; font-weight: 800; color: var(--muted); }
    .mod-desc { font-size: 12px; line-height: 1.5; font-weight: 500; }
    .mod-id { font-size: 10px; font-family: var(--mono); color: var(--muted); margin-top: 4px; }

    .mod-actions { display: flex; justify-content: space-between; align-items: center; margin-top: 5px; border-top: 2px dashed var(--line); padding-top: 12px; }

    .brutal-switch { position: relative; display: inline-block; width: 44px; height: 24px; }
    .brutal-switch input { opacity: 0; width: 0; height: 0; }
    .slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: var(--panel-alt); border: 2px solid var(--line); border-radius: 20px; transition: .2s; }
    .slider:before { position: absolute; content: ""; height: 14px; width: 14px; left: 3px; bottom: 3px; background-color: var(--muted); border-radius: 50%; border: 2px solid var(--line); transition: .2s; }
    input:checked + .slider { background-color: var(--lime); }
    input:checked + .slider:before { transform: translateX(20px); background-color: #fff; }

    .btn-trash { background: var(--pink); color: #fff; padding: 6px 10px; font-size: 11px; }
    .btn-restore { background: var(--yellow); color: #000; padding: 6px 10px; font-size: 11px; }

    .badge-del { position: absolute; top: -10px; right: 10px; background: var(--pink); color: #fff; font-size: 10px; font-weight: 900; padding: 4px 8px; border: 2px solid var(--line); border-radius: 6px; box-shadow: 2px 2px 0 var(--line); transform: rotate(2deg); z-index: 10; text-transform: uppercase; }

    .empty-state { text-align: center; padding: 40px 20px; color: var(--muted); font-weight: 800; font-size: 14px; text-transform: uppercase; }
</style>
