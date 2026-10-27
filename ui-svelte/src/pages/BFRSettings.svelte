<script>
    import { onMount } from 'svelte';

    let settings = {
        bin_name: 'clash',
        xclash_option: 'mihomo',
        network_mode: 'tproxy',
        proxy_mode: 'blacklist',
        ipv6: 'false',
        port_detect: 'true',
        cgroup_cpuset: 'false',
        cgroup_blkio: 'false',
        cgroup_memcg: 'false',
        memcg_limit: '',
        run_crontab: 'false',
        update_geo: 'false',
        renew: 'false',
        update_subscription: 'false',
        tproxy_port: '',
        redir_port: '',
        subscription_url_clash: '',
        name_clash_config: '',
        name_sing_config: ''
    };

    let loading = true;
    let saving = false;
    let modal = { show: false, title: '', message: '', isError: false };

    let shortcutConfigPath = '/data/adb/box/clash/config.yaml';
    let shortcutConfigLabel = 'Edit config.yaml';
    let shortcutAccountPath = '/data/adb/box/clash/proxy_provider';
    let shortcutAccountLabel = 'Edit Akun';

    function loadShortcuts() {
        shortcutConfigPath = localStorage.getItem('bfr_config_path') || '/data/adb/box/clash/config.yaml';
        shortcutConfigLabel = localStorage.getItem('bfr_config_label') || '';
        if (!shortcutConfigLabel) {
            const parts = shortcutConfigPath.split('/');
            shortcutConfigLabel = `Edit ${parts[parts.length - 1] || 'config.yaml'}`;
        }
        shortcutAccountPath = localStorage.getItem('bfr_account_path') || '/data/adb/box/clash/proxy_provider';
        shortcutAccountLabel = localStorage.getItem('bfr_account_label') || 'Edit Akun';
    }

    async function loadSettings() {
        loading = true;
        try {
            const res = await fetch('/api/bfr/settings');
            const data = await res.json();
            if (data.status === 'success' && data.settings) {
                settings = { ...settings, ...data.settings };
            }
        } catch (e) {
            console.error('Error load settings', e);
        } finally {
            loading = false;
        }
    }

    async function saveSettings() {
        saving = true;
        try {
            // Simpan shortcut path ke localStorage
            localStorage.setItem('bfr_config_path', shortcutConfigPath.trim() || '/data/adb/box/clash/config.yaml');
            localStorage.setItem('bfr_config_label', shortcutConfigLabel.trim() || 'Edit config.yaml');
            localStorage.setItem('bfr_account_path', shortcutAccountPath.trim() || '/data/adb/box/clash/proxy_provider');
            localStorage.setItem('bfr_account_label', shortcutAccountLabel.trim() || 'Edit Akun');
            window.dispatchEvent(new Event('storage'));

            const res = await fetch('/api/bfr/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(settings)
            });
            const data = await res.json();
            if (data.status === 'success') {
                modal = { show: true, title: 'Sukses', message: 'Setelan BFR dan Shortcut berhasil disimpan!', isError: false };
            } else {
                modal = { show: true, title: 'Gagal', message: data.message || 'Gagal menyimpan', isError: true };
            }
        } catch (e) {
            modal = { show: true, title: 'Error', message: 'Koneksi terputus saat menyimpan', isError: true };
        } finally {
            saving = false;
        }
    }

    onMount(() => {
        loadSettings();
        loadShortcuts();
    });
</script>

<div class="wrap">
    <div class="header">
        <div class="eyebrow"><span class="live-dot"></span> Core System</div>
        <h1>Box Settings</h1>
        <div class="desc">Konfigurasi file settings.ini untuk Box for Root.</div>
    </div>

    {#if loading}
        <div class="empty-state">
            <i class="fa-solid fa-spinner fa-spin"></i> Memuat Konfigurasi...
        </div>
    {:else}
        <form on:submit|preventDefault={saveSettings}>
            <!-- CORE & NETWORK -->
            <div class="card">
                <div class="card-header" style="border-bottom-color:var(--cyan);">
                    <i class="fa-solid fa-server" style="color:var(--cyan);"></i> Core & Network
                </div>

                <div class="form-group">
                    <label>Bin Name</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.bin_name}>
                            <option value="clash">CLASH</option>
                            <option value="sing-box">SING-BOX</option>
                            <option value="xray">XRAY</option>
                            <option value="v2fly">V2FLY</option>
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label>XClash Option</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.xclash_option}>
                            <option value="mihomo">MIHOMO</option>
                            <option value="premium">PREMIUM</option>
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label>Network Mode</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.network_mode}>
                            <option value="redirect">REDIRECT</option>
                            <option value="tproxy">TPROXY</option>
                            <option value="mixed">MIXED</option>
                            <option value="enhance">ENHANCE</option>
                            <option value="tun">TUN</option>
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label>Proxy Mode</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.proxy_mode}>
                            <option value="blacklist">BLACKLIST</option>
                            <option value="whitelist">WHITELIST</option>
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label>IPv6</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.ipv6}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label>Port Detect</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.port_detect}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>
            </div>

            <!-- CGROUP & TUNING -->
            <div class="card">
                <div class="card-header" style="border-bottom-color:var(--pink);">
                    <i class="fa-solid fa-microchip" style="color:var(--pink);"></i> Cgroup & Tuning
                </div>

                <div class="form-group">
                    <label>Cgroup Cpuset</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.cgroup_cpuset}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>
                <div class="form-group">
                    <label>Cgroup Blkio</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.cgroup_blkio}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>
                <div class="form-group">
                    <label>Cgroup Memcg</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.cgroup_memcg}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>
                <div class="form-group">
                    <label>Memcg Limit</label>
                    <input type="text" bind:value={settings.memcg_limit} placeholder="e.g. 500M">
                </div>
            </div>

            <!-- AUTOMATION & CONFIGS -->
            <div class="card">
                <div class="card-header" style="border-bottom-color:var(--yellow);">
                    <i class="fa-solid fa-cloud-arrow-down" style="color:var(--yellow);"></i> Automation & Configs
                </div>

                <div class="form-group">
                    <label>Run Crontab</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.run_crontab}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>
                <div class="form-group">
                    <label>Update Geo</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.update_geo}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>
                <div class="form-group">
                    <label>Renew</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.renew}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>
                <div class="form-group">
                    <label>Update Subscription</label>
                    <div class="select-wrapper">
                        <select bind:value={settings.update_subscription}>
                            <option value="true">TRUE</option>
                            <option value="false">FALSE</option>
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label>TProxy Port</label>
                    <input type="text" bind:value={settings.tproxy_port} placeholder="e.g. 9898">
                </div>
                <div class="form-group">
                    <label>Redir Port</label>
                    <input type="text" bind:value={settings.redir_port} placeholder="e.g. 9797">
                </div>
                <div class="form-group">
                    <label>Subscription URL Clash</label>
                    <input type="text" bind:value={settings.subscription_url_clash} placeholder="https://...">
                </div>
                <div class="form-group">
                    <label>Name Clash Config</label>
                    <input type="text" bind:value={settings.name_clash_config} placeholder="config.yaml">
                </div>
                <div class="form-group">
                    <label>Name Sing Config</label>
                    <input type="text" bind:value={settings.name_sing_config} placeholder="config.json">
                </div>
            </div>

            <!-- SHORTCUT FILE & AKUN -->
            <div class="card">
                <div class="card-header" style="border-bottom-color:var(--purple);">
                    <i class="fa-solid fa-folder-tree" style="color:var(--purple);"></i> Shortcut File & Akun
                </div>
                <div class="desc-card" style="margin-bottom: 14px; font-size: 11px; color: var(--muted); font-weight: 700; line-height: 1.4;">
                    Sesuaikan jalur file konfigurasi dan folder/file akun untuk tombol navigasi shortcut Box di Drawer menu.
                </div>

                <div class="form-group">
                    <label>Path File Config</label>
                    <input type="text" bind:value={shortcutConfigPath} placeholder="/data/adb/box/clash/config.yaml">
                </div>
                <div class="form-group">
                    <label>Label Tombol Config</label>
                    <input type="text" bind:value={shortcutConfigLabel} placeholder="Edit config.yaml">
                </div>

                <div class="form-group" style="margin-top: 14px; border-top: 1px dashed var(--line); padding-top: 14px;">
                    <label>Path Folder / File Akun</label>
                    <input type="text" bind:value={shortcutAccountPath} placeholder="/data/adb/box/clash/proxy_provider">
                </div>
                <div class="form-group">
                    <label>Label Tombol Akun</label>
                    <input type="text" bind:value={shortcutAccountLabel} placeholder="Edit Akun">
                </div>
            </div>

            <div class="save-bar">
                <button type="submit" class="btn-save" disabled={saving}>
                    {#if saving}
                        <i class="fa-solid fa-spinner fa-spin"></i> Menyimpan...
                    {:else}
                        <i class="fa-solid fa-floppy-disk"></i> Simpan Setelan
                    {/if}
                </button>
            </div>
        </form>
    {/if}

    <!-- POPUP MODAL -->
    {#if modal.show}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="custom-modal" on:click={() => modal.show = false}>
            <div class="modal-card" on:click|stopPropagation>
                <div class="modal-title" style="background: {modal.isError ? 'var(--pink)' : 'var(--cyan)'}; color: {modal.isError ? '#fff' : '#000'}">
                    {modal.title}
                </div>
                <div class="modal-body">{modal.message}</div>
                <div class="modal-actions">
                    <button class="btn-modal" on:click={() => modal.show = false}>Tutup</button>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .wrap { max-width: 800px; margin: 0 auto; padding: 16px; padding-bottom: 90px; }
    .header { margin-bottom: 20px; }
    .eyebrow { font-size: 10px; font-weight: 900; text-transform: uppercase; color: var(--cyan); display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
    .live-dot { width: 8px; height: 8px; background: var(--cyan); border-radius: 50%; animation: blink 1.5s infinite; }
    @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
    h1 { font-size: 28px; font-weight: 950; letter-spacing: -1.5px; text-transform: uppercase; line-height: 1; }
    .desc { font-size: 12px; font-weight: 700; color: var(--muted); margin-top: 6px; }

    .card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 20px; box-shadow: var(--shadow); margin-bottom: 20px; }
    .card-header { display: flex; align-items: center; gap: 10px; font-size: 14px; font-weight: 900; text-transform: uppercase; margin-bottom: 16px; padding-bottom: 10px; border-bottom: 2px dashed var(--line); color: var(--ink); }

    .form-group { margin-bottom: 15px; }
    .form-group label { display: block; font-size: 11px; font-weight: 900; text-transform: uppercase; color: var(--muted); margin-bottom: 6px; letter-spacing: 0.8px; }

    input[type="text"], select {
        width: 100%;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 10px;
        padding: 12px 16px;
        color: var(--ink);
        font-size: 13px;
        font-weight: 800;
        transition: all 0.2s;
        appearance: none;
        -webkit-appearance: none;
        font-family: inherit;
    }
    input:focus, select:focus { border-color: var(--cyan); box-shadow: 2px 2px 0 var(--cyan); }

    .select-wrapper { position: relative; }
    .select-wrapper::after {
        content: '\f078';
        font-family: "Font Awesome 6 Free";
        font-weight: 900;
        position: absolute;
        right: 16px; top: 50%;
        transform: translateY(-50%);
        color: var(--muted);
        pointer-events: none;
        font-size: 12px;
    }

    .save-bar {
        display: flex;
        justify-content: center;
        margin-top: 10px;
    }
    .btn-save {
        background: var(--cyan); color: #000;
        border: 2px solid var(--line); padding: 14px 36px; border-radius: 50px;
        font-size: 13px; font-weight: 950; text-transform: uppercase; cursor: pointer;
        box-shadow: 4px 4px 0 var(--line); display: flex; align-items: center; gap: 10px;
        transition: transform 0.1s, box-shadow 0.1s; letter-spacing: 0.5px; font-family: inherit;
    }
    .btn-save:hover { transform: translate(-2px, -2px); box-shadow: 6px 6px 0 var(--line); }
    .btn-save:active { transform: translate(2px, 2px); box-shadow: none; }
    .btn-save:disabled { opacity: 0.5; pointer-events: none; }

    .empty-state { text-align: center; padding: 40px 20px; color: var(--muted); font-weight: 800; font-size: 14px; text-transform: uppercase; }

    /* MODAL */
    .custom-modal { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.6); z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 16px; }
    .modal-card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 20px; width: 360px; max-width: 100%; box-shadow: var(--shadow); }
    .modal-title { font-size: 13px; font-weight: 950; text-transform: uppercase; padding: 4px 10px; border: 2px solid var(--line); border-radius: 6px; box-shadow: 2px 2px 0 var(--line); display: inline-block; margin-bottom: 12px; }
    .modal-body { font-size: 13px; line-height: 1.5; color: var(--ink); margin-bottom: 16px; font-weight: 700; }
    .modal-actions { display: flex; justify-content: flex-end; }
    .btn-modal { padding: 8px 16px; font-weight: 950; font-size: 12px; border-radius: 8px; border: 2px solid var(--line); cursor: pointer; box-shadow: 3px 3px 0 var(--line); text-transform: uppercase; background: var(--paper); color: var(--ink); }
</style>
