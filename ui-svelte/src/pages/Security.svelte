<script>
    let user_oldUser = '';
    let user_oldPass = '';
    let user_new = '';

    let pass_oldUser = '';
    let pass_oldPass = '';
    let pass_new = '';

    let modal = { show: false, title: '', message: '', isSuccess: false };

    async function sendUpdate(old_u, old_p, new_u, new_p) {
        try {
            const res = await fetch('/api/auth/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    old_user: old_u,
                    old_pass: old_p,
                    new_user: new_u,
                    new_pass: new_p
                })
            });
            const data = await res.json();
            if (data.status === 'success') {
                modal = { show: true, title: 'Berhasil', message: data.message, isSuccess: true };
            } else {
                modal = { show: true, title: 'Gagal', message: data.message, isSuccess: false };
            }
        } catch (e) {
            modal = { show: true, title: 'Error', message: 'Gagal terhubung ke server', isSuccess: false };
        }
    }

    function updateUsername() {
        if (!user_oldUser || !user_oldPass || !user_new) {
            modal = { show: true, title: 'Error', message: 'Semua kolom username wajib diisi!', isSuccess: false };
            return;
        }
        sendUpdate(user_oldUser, user_oldPass, user_new, '');
    }

    function updatePassword() {
        if (!pass_oldUser || !pass_oldPass || !pass_new) {
            modal = { show: true, title: 'Error', message: 'Semua kolom password wajib diisi!', isSuccess: false };
            return;
        }
        sendUpdate(pass_oldUser, pass_oldPass, '', pass_new);
    }
</script>

<div class="wrap">
    <div class="header">
        <div class="eyebrow"><span class="live-dot"></span> System Manager</div>
        <h1>Security</h1>
        <div class="desc">Ubah kredensial login Dashboard ProjectX</div>
    </div>

    <!-- GANTI USERNAME -->
    <div class="card">
        <div class="card-header">
            <i class="fa-solid fa-user-pen"></i> Ganti Username
        </div>
        <div class="form-group">
            <label>Username Saat Ini</label>
            <input type="text" bind:value={user_oldUser} placeholder="Username saat ini">
        </div>
        <div class="form-group">
            <label>Password Saat Ini</label>
            <input type="password" bind:value={user_oldPass} placeholder="Password saat ini">
        </div>
        <div class="form-group">
            <label>Username Baru</label>
            <input type="text" bind:value={user_new} placeholder="Masukkan username baru">
        </div>
        <button class="btn-save" on:click={updateUsername}>Simpan Username</button>
    </div>

    <!-- GANTI PASSWORD -->
    <div class="card">
        <div class="card-header" style="border-bottom-color:var(--lime);">
            <i class="fa-solid fa-key" style="color:var(--lime)"></i> Ganti Password
        </div>
        <div class="form-group">
            <label>Username Saat Ini</label>
            <input type="text" bind:value={pass_oldUser} placeholder="Username saat ini">
        </div>
        <div class="form-group">
            <label>Password Saat Ini</label>
            <input type="password" bind:value={pass_oldPass} placeholder="Password saat ini">
        </div>
        <div class="form-group">
            <label>Password Baru</label>
            <input type="password" bind:value={pass_new} placeholder="Masukkan password baru">
        </div>
        <button class="btn-save" style="background:var(--lime); color:#000;" on:click={updatePassword}>
            Simpan Password
        </button>
    </div>

    <!-- MODAL -->
    {#if modal.show}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="custom-modal" on:click={() => modal.show = false}>
            <div class="modal-card" on:click|stopPropagation>
                <div class="modal-title" style="background: {modal.isSuccess ? 'var(--lime)' : 'var(--pink)'}; color: {modal.isSuccess ? '#000' : '#fff'}">
                    {modal.title}
                </div>
                <div class="modal-body">{modal.message}</div>
                <div class="modal-actions">
                    <button class="btn-modal" on:click={() => {
                        modal.show = false;
                        if (modal.isSuccess) window.location.reload();
                    }}>
                        {modal.isSuccess ? 'OK, Login Ulang' : 'Tutup'}
                    </button>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .wrap { max-width: 600px; margin: 0 auto; padding: 16px; padding-bottom: 90px; }
    .header { margin-bottom: 20px; }
    .eyebrow { font-size: 10px; font-weight: 900; text-transform: uppercase; color: var(--primary); display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
    .live-dot { width: 8px; height: 8px; background: var(--primary); border-radius: 50%; animation: blink 1.5s infinite; }
    @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
    h1 { font-size: 28px; font-weight: 950; letter-spacing: -1.5px; text-transform: uppercase; line-height: 1; }
    .desc { font-size: 12px; font-weight: 700; color: var(--muted); margin-top: 6px; }

    .card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 20px; box-shadow: var(--shadow); margin-bottom: 20px; }
    .card-header { display: flex; align-items: center; gap: 10px; font-size: 14px; font-weight: 900; text-transform: uppercase; margin-bottom: 16px; padding-bottom: 10px; border-bottom: 2px dashed var(--line); color: var(--ink); }
    .card-header i { color: var(--cyan); }

    .form-group { margin-bottom: 14px; }
    .form-group label { display: block; font-size: 11px; font-weight: 900; text-transform: uppercase; color: var(--muted); margin-bottom: 6px; }
    .form-group input { width: 100%; background: var(--panel-alt); border: 2px solid var(--line); border-radius: 8px; padding: 12px; color: var(--ink); font-size: 13px; font-weight: 700; transition: all 0.2s; font-family: inherit; }
    .form-group input:focus { outline: none; border-color: var(--primary); box-shadow: 2px 2px 0 var(--primary); }

    .btn-save { width: 100%; padding: 12px; background: var(--primary); color: #000; border: 2px solid var(--line); border-radius: 10px; font-size: 13px; font-weight: 950; text-transform: uppercase; letter-spacing: 0.5px; cursor: pointer; box-shadow: 3px 3px 0 var(--line); transition: all 0.1s; margin-top: 6px; font-family: inherit; }
    .btn-save:hover { transform: translate(-2px, -2px); box-shadow: 5px 5px 0 var(--line); filter: brightness(1.1); }
    .btn-save:active { transform: translate(2px, 2px); box-shadow: none; }

    /* MODAL */
    .custom-modal { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.6); z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 16px; }
    .modal-card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 20px; width: 360px; max-width: 100%; box-shadow: var(--shadow); }
    .modal-title { font-size: 13px; font-weight: 950; text-transform: uppercase; padding: 4px 10px; border: 2px solid var(--line); border-radius: 6px; box-shadow: 2px 2px 0 var(--line); display: inline-block; margin-bottom: 12px; }
    .modal-body { font-size: 13px; line-height: 1.5; color: var(--ink); margin-bottom: 16px; font-weight: 700; }
    .modal-actions { display: flex; justify-content: flex-end; }
    .btn-modal { padding: 8px 16px; font-weight: 950; font-size: 12px; border-radius: 8px; border: 2px solid var(--line); cursor: pointer; box-shadow: 3px 3px 0 var(--line); text-transform: uppercase; background: var(--paper); color: var(--ink); }
</style>
