<script>
    let showConfirm = false;
    let rebooting = false;

    async function executeReboot() {
        showConfirm = false;
        rebooting = true;

        try {
            await fetch('/api/reboot');
        } catch (e) {
            console.warn("Connection dropped during reboot");
        }
    }
</script>

<div class="wrap">
    {#if rebooting}
        <div class="rebooting-screen">
            <i class="fa-solid fa-spinner fa-spin spin-icon"></i>
            <h2>Mengeksekusi Reboot...</h2>
            <p>Perangkat akan mati dan menyala kembali.<br>Harap tunggu beberapa menit sebelum merefresh halaman ini.</p>
        </div>
    {:else}
        <div class="header">
            <i class="fa-solid fa-power-off icon-main"></i>
            <h1>Reboot System</h1>
            <div class="desc">Memulai ulang perangkat. Semua koneksi jaringan, service, dan antarmuka akan terputus untuk sementara.</div>
        </div>

        <div class="card">
            <button class="btn-reboot" on:click={() => showConfirm = true}>
                <i class="fa-solid fa-rotate"></i> Reboot Sekarang
            </button>
        </div>
    {/if}

    {#if showConfirm}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="custom-modal" on:click={() => showConfirm = false}>
            <div class="modal-card" on:click|stopPropagation>
                <div class="modal-title">
                    <i class="fa-solid fa-triangle-exclamation"></i> Konfirmasi
                </div>
                <div class="modal-body">
                    Apakah Anda yakin ingin melakukan <strong>Reboot</strong> perangkat ini? Dashboard tidak akan dapat diakses hingga perangkat selesai menyala kembali.
                </div>
                <div class="modal-actions">
                    <button class="btn-modal btn-cancel" on:click={() => showConfirm = false}>Batal</button>
                    <button class="btn-modal btn-danger" on:click={executeReboot}>Ya, Reboot</button>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .wrap { max-width: 600px; margin: 0 auto; padding: 20px; text-align: center; }
    .header { margin-bottom: 24px; }
    .icon-main { font-size: 46px; color: var(--pink); margin-bottom: 14px; filter: drop-shadow(3px 3px 0 var(--line)); }
    h1 { font-size: 26px; font-weight: 950; letter-spacing: -1px; text-transform: uppercase; line-height: 1; }
    .desc { font-size: 12px; font-weight: 700; color: var(--muted); margin-top: 8px; line-height: 1.5; }

    .card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 30px 20px; box-shadow: var(--shadow); margin-bottom: 20px; }

    .btn-reboot { display: inline-flex; align-items: center; justify-content: center; gap: 10px; width: 100%; max-width: 280px; padding: 14px; background: var(--pink); color: #fff; border: 2px solid var(--line); border-radius: 12px; font-size: 14px; font-weight: 950; text-transform: uppercase; letter-spacing: 0.5px; cursor: pointer; box-shadow: 4px 4px 0 var(--line); transition: all 0.1s; font-family: inherit; }
    .btn-reboot:hover { transform: translate(-2px, -2px); box-shadow: 6px 6px 0 var(--line); filter: brightness(1.1); }
    .btn-reboot:active { transform: translate(2px, 2px); box-shadow: none; }

    .rebooting-screen { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 50vh; }
    .spin-icon { font-size: 40px; color: var(--cyan); margin-bottom: 16px; }
    .rebooting-screen h2 { font-size: 20px; font-weight: 900; text-transform: uppercase; }
    .rebooting-screen p { font-size: 12px; color: var(--muted); margin-top: 10px; line-height: 1.5; }

    /* MODAL */
    .custom-modal { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.6); z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 16px; }
    .modal-card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 20px; width: 360px; max-width: 100%; box-shadow: var(--shadow); text-align: left; }
    .modal-title { font-size: 13px; font-weight: 950; text-transform: uppercase; padding: 4px 10px; border: 2px solid var(--line); border-radius: 6px; box-shadow: 2px 2px 0 var(--line); display: inline-block; margin-bottom: 12px; background: var(--pink); color: #fff; }
    .modal-body { font-size: 13px; line-height: 1.5; color: var(--ink); margin-bottom: 16px; font-weight: 700; }
    .modal-actions { display: flex; justify-content: flex-end; gap: 8px; }
    .btn-modal { padding: 8px 16px; font-weight: 950; font-size: 12px; border-radius: 8px; border: 2px solid var(--line); cursor: pointer; box-shadow: 3px 3px 0 var(--line); text-transform: uppercase; }
    .btn-cancel { background: var(--panel-alt); color: var(--ink); }
    .btn-danger { background: var(--pink); color: #fff; }
</style>
