<script>
    import { onMount, onDestroy } from 'svelte';

    let messages = [];
    let loading = true;
    let searchQuery = '';
    let activeFilter = 'all'; // 'all' | 'otp' | 'inbox'
    let toast = { show: false, message: '' };
    let timer = null;

    let modal = {
        show: false,
        type: '', // 'delete_one' | 'delete_all' | 'alert'
        title: '',
        message: '',
        targetId: null,
        targetSender: ''
    };

    async function fetchSMS() {
        try {
            const res = await fetch('/api/sms/list?_=' + Date.now());
            if (!res.ok) throw new Error('HTTP ' + res.status);
            const data = await res.json();
            messages = data.messages || [];
        } catch (e) {
            console.error('Gagal fetch SMS:', e);
        } finally {
            loading = false;
        }
    }

    function showToast(msg) {
        toast = { show: true, message: msg };
        setTimeout(() => { toast.show = false; }, 2000);
    }

    function copyOTP(otp) {
        if (navigator.clipboard) {
            navigator.clipboard.writeText(otp);
        }
        showToast('OTP ' + otp + ' disalin!');
    }

    function openDeleteModal(m) {
        modal = {
            show: true,
            type: 'delete_one',
            title: 'Hapus Pesan',
            message: `Hapus pesan dari <strong>${m.address || 'Unknown'}</strong>? Tindakan ini tidak dapat dibatalkan.`,
            targetId: m.id,
            targetSender: m.address || 'Unknown'
        };
    }

    function openDeleteAllModal() {
        if (messages.length === 0) {
            showToast('Tidak ada pesan untuk dihapus');
            return;
        }
        modal = {
            show: true,
            type: 'delete_all',
            title: 'Kosongkan Inbox',
            message: `Apakah Anda yakin ingin menghapus <strong>SEMUA (${messages.length}) pesan</strong>? Semua riwayat SMS akan hilang permanen.`,
            targetId: null,
            targetSender: ''
        };
    }

    async function doDelete() {
        const id = modal.targetId;
        modal.show = false;
        if (!id) return;
        try {
            const res = await fetch('/api/sms/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ id })
            });
            const data = await res.json();
            showToast(data.message || 'Pesan dihapus');
            fetchSMS();
        } catch (e) {
            showAlert('Gagal', 'Gagal menghapus pesan.');
        }
    }

    async function doDeleteAll() {
        modal.show = false;
        try {
            const res = await fetch('/api/sms/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ all: true })
            });
            const data = await res.json();
            showToast(data.message || 'Semua pesan dibersihkan');
            fetchSMS();
        } catch (e) {
            showAlert('Gagal', 'Gagal mengosongkan pesan.');
        }
    }

    function showAlert(title, message) {
        modal = {
            show: true,
            type: 'alert',
            title,
            message,
            targetId: null,
            targetSender: ''
        };
    }

    $: filteredMessages = messages.filter(m => {
        if (activeFilter === 'otp' && !m.otp) return false;
        if (activeFilter === 'inbox' && m.type !== 1) return false;

        if (searchQuery.trim()) {
            const q = searchQuery.toLowerCase();
            const inAddr = (m.address || '').toLowerCase().includes(q);
            const inBody = (m.body || '').toLowerCase().includes(q);
            const inOTP = (m.otp || '').toLowerCase().includes(q);
            return inAddr || inBody || inOTP;
        }
        return true;
    });

    onMount(() => {
        fetchSMS();
        timer = setInterval(fetchSMS, 6000);
    });

    onDestroy(() => {
        if (timer) clearInterval(timer);
    });
</script>

<div class="wrap">
    <div class="card">
        <div class="header">
            <div class="header-left">
                <i class="fa-solid fa-comment-sms icon-title"></i>
                <h1 class="header-title">SMS Inbox</h1>
                <span class="sms-badge">{messages.length} Pesan</span>
            </div>
            <div class="header-actions">
                <button class="btn btn-refresh" on:click={fetchSMS} title="Refresh">
                    <i class="fa-solid fa-rotate-right {loading ? 'fa-spin' : ''}"></i>
                </button>
                <button class="btn btn-danger" on:click={openDeleteAllModal} title="Hapus Semua">
                    <i class="fa-solid fa-trash-can"></i> Hapus Semua
                </button>
            </div>
        </div>

        <div class="search-area">
            <div class="search-box">
                <i class="fa-solid fa-magnifying-glass"></i>
                <input type="text" placeholder="Cari pesan, nomor, OTP..." bind:value={searchQuery}>
            </div>
            <div class="filter-chips">
                <button class="chip {activeFilter === 'all' ? 'active' : ''}" on:click={() => activeFilter = 'all'}>
                    Semua ({messages.length})
                </button>
                <button class="chip {activeFilter === 'otp' ? 'active' : ''}" on:click={() => activeFilter = 'otp'}>
                    ⚡ Kode OTP ({messages.filter(m => m.otp).length})
                </button>
                <button class="chip {activeFilter === 'inbox' ? 'active' : ''}" on:click={() => activeFilter = 'inbox'}>
                    Masuk ({messages.filter(m => m.type === 1).length})
                </button>
            </div>
        </div>

        <ul class="sms-list">
            {#if loading && messages.length === 0}
                <li class="empty-state">
                    <i class="fa-solid fa-spinner fa-spin"></i>
                    <p>Memuat pesan...</p>
                </li>
            {:else if filteredMessages.length === 0}
                <li class="empty-state">
                    <i class="fa-regular fa-envelope-open"></i>
                    <p>{searchQuery ? 'Tidak ada pesan yang cocok.' : 'Tidak ada pesan masuk.'}</p>
                </li>
            {:else}
                {#each filteredMessages as m (m.id || m.date)}
                    {@const isInbox = m.type === 1}
                    <li class="sms-item">
                        <div class="sms-top">
                            <div class="sms-sender-info">
                                <span class="sms-sender">{m.address || 'Unknown'}</span>
                                <span class="type-tag {isInbox ? 'inbox' : ''}">{isInbox ? 'Masuk' : 'Keluar'}</span>
                            </div>
                            <span class="sms-date">{m.formatted_time || '-'}</span>
                        </div>
                        <div class="sms-body">{m.body || ''}</div>

                        {#if m.otp}
                            <div class="otp-box">
                                <span class="otp-label">Kode OTP:</span>
                                <span class="otp-code">{m.otp}</span>
                                <button class="btn-copy-otp" on:click={() => copyOTP(m.otp)}>
                                    <i class="fa-solid fa-copy"></i> Salin
                                </button>
                            </div>
                        {/if}

                        <div class="sms-footer">
                            <button class="btn-del" on:click={() => openDeleteModal(m)} title="Hapus pesan ini">
                                <i class="fa-solid fa-trash-can"></i> Hapus
                            </button>
                        </div>
                    </li>
                {/each}
            {/if}
        </ul>
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
                    {@html modal.message}
                </div>
                <div class="modal-actions">
                    <button class="btn-modal btn-cancel" on:click={() => modal.show = false}>
                        {modal.type === 'alert' ? 'Tutup' : 'Batal'}
                    </button>
                    {#if modal.type === 'delete_one'}
                        <button class="btn-modal btn-danger" on:click={doDelete}>
                            <i class="fa-solid fa-trash-can"></i> Hapus
                        </button>
                    {:else if modal.type === 'delete_all'}
                        <button class="btn-modal btn-danger" on:click={doDeleteAll}>
                            <i class="fa-solid fa-trash-can"></i> Hapus Semua
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {/if}

    {#if toast.show}
        <div class="toast">{toast.message}</div>
    {/if}
</div>

<style>
    .wrap { max-width: 820px; margin: 0 auto; padding: 16px; padding-bottom: 90px; }
    .card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; overflow: hidden; box-shadow: var(--shadow); margin-bottom: 16px; }

    .header { padding: 14px 18px; background: var(--panel-alt); border-bottom: 2px solid var(--line); display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
    .header-left { display: flex; align-items: center; gap: 10px; }
    .icon-title { font-size: 20px; color: var(--yellow); }
    .header-title { font-size: 16px; font-weight: 900; margin: 0; text-transform: uppercase; letter-spacing: -0.5px; }
    .sms-badge { background: var(--yellow); color: #000; padding: 2px 8px; border-radius: 6px; font-size: 11px; font-weight: 900; border: 1px solid var(--line); }

    .header-actions { display: flex; gap: 8px; }
    .btn { background: var(--paper); color: var(--ink); border: 2px solid var(--line); border-radius: 8px; padding: 8px 12px; font-size: 12px; font-weight: 900; cursor: pointer; display: flex; align-items: center; gap: 6px; box-shadow: 2px 2px 0 var(--line); font-family: inherit; transition: .1s; }
    .btn:hover { transform: translate(-1px, -1px); box-shadow: 3px 3px 0 var(--line); }
    .btn:active { transform: translate(1px, 1px); box-shadow: none; }
    .btn-danger { background: var(--pink); color: #fff; }

    .search-area { padding: 14px 18px; border-bottom: 2px dashed var(--line); display: flex; flex-direction: column; gap: 10px; }
    .search-box { display: flex; align-items: center; gap: 10px; background: var(--panel-alt); border: 2px solid var(--line); border-radius: 10px; padding: 10px 14px; }
    .search-box i { color: var(--muted); }
    .search-box input { border: none; background: transparent; color: var(--ink); width: 100%; font-size: 13px; font-weight: 700; font-family: inherit; outline: none; }

    .filter-chips { display: flex; gap: 6px; flex-wrap: wrap; }
    .chip { padding: 6px 12px; background: var(--panel-alt); border: 2px solid var(--line); border-radius: 8px; font-size: 11px; font-weight: 900; cursor: pointer; color: var(--muted); font-family: inherit; transition: .1s; }
    .chip.active { background: var(--yellow); color: #000; box-shadow: 2px 2px 0 var(--line); }

    .sms-list { list-style: none; padding: 0; margin: 0; }
    .sms-item { padding: 16px 18px; border-bottom: 1px solid var(--line); display: flex; flex-direction: column; gap: 8px; }
    .sms-item:last-child { border-bottom: none; }

    .sms-top { display: flex; justify-content: space-between; align-items: center; }
    .sms-sender-info { display: flex; align-items: center; gap: 8px; }
    .sms-sender { font-size: 14px; font-weight: 900; }
    .type-tag { font-size: 9px; font-weight: 900; padding: 2px 6px; border-radius: 4px; border: 1px solid var(--line); background: var(--panel-alt); color: var(--muted); text-transform: uppercase; }
    .type-tag.inbox { background: var(--cyan); color: #000; }
    .sms-date { font-size: 10px; font-weight: 700; color: var(--muted); font-family: var(--mono, monospace); }

    .sms-body { font-size: 13px; line-height: 1.5; color: var(--ink); word-break: break-word; }

    .otp-box { display: flex; align-items: center; gap: 8px; background: var(--panel-alt); border: 2px solid var(--yellow); border-radius: 8px; padding: 8px 12px; width: fit-content; margin-top: 4px; box-shadow: 2px 2px 0 var(--line); }
    .otp-label { font-size: 11px; font-weight: 900; color: var(--muted); text-transform: uppercase; }
    .otp-code { font-size: 16px; font-weight: 950; color: var(--yellow); letter-spacing: 2px; font-family: var(--mono, monospace); }
    .btn-copy-otp { background: var(--yellow); color: #000; border: 1px solid var(--line); border-radius: 4px; padding: 4px 8px; font-size: 10px; font-weight: 900; cursor: pointer; box-shadow: 1px 1px 0 var(--line); }
    .btn-copy-otp:active { transform: translate(1px, 1px); box-shadow: none; }

    .sms-footer { display: flex; justify-content: flex-end; }
    .btn-del { background: transparent; border: 1.5px solid transparent; color: var(--pink); font-size: 11px; font-weight: 900; cursor: pointer; display: flex; align-items: center; gap: 4px; padding: 4px 8px; border-radius: 6px; transition: .1s; }
    .btn-del:hover { background: rgba(255, 92, 168, 0.12); border-color: var(--pink); }

    .empty-state { padding: 40px; text-align: center; color: var(--muted); display: flex; flex-direction: column; align-items: center; gap: 10px; font-size: 13px; font-weight: 800; }
    .empty-state i { font-size: 32px; }

    .toast { position: fixed; bottom: 85px; left: 50%; transform: translateX(-50%); background: var(--yellow); color: #000; padding: 10px 20px; border-radius: 50px; border: 2px solid var(--line); font-size: 12px; font-weight: 950; box-shadow: 3px 3px 0 var(--line); z-index: 10000; }

    /* UNIVERSAL POPUP MODAL (THEME & COLOR SYNCED) */
    .custom-modal { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.65); z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 16px; }
    .modal-card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 20px; width: 380px; max-width: 100%; box-shadow: var(--shadow); }
    .modal-title { font-size: 13px; font-weight: 950; text-transform: uppercase; padding: 4px 10px; border: 2px solid var(--line); border-radius: 6px; box-shadow: 2px 2px 0 var(--line); display: inline-flex; align-items: center; gap: 6px; margin-bottom: 12px; background: var(--pink); color: #fff; }
    .modal-body { font-size: 13px; line-height: 1.5; color: var(--ink); margin-bottom: 16px; font-weight: 700; }
    .modal-actions { display: flex; justify-content: flex-end; gap: 8px; }
    .btn-modal { padding: 8px 16px; font-weight: 950; font-size: 12px; border-radius: 8px; border: 2px solid var(--line); cursor: pointer; box-shadow: 3px 3px 0 var(--line); text-transform: uppercase; transition: .1s; font-family: inherit; }
    .btn-modal:hover { transform: translate(-1px, -1px); box-shadow: 4px 4px 0 var(--line); }
    .btn-modal:active { transform: translate(1px, 1px); box-shadow: none; }
    .btn-cancel { background: var(--panel-alt); color: var(--ink); }
    .btn-danger { background: var(--pink); color: #fff; }
</style>
