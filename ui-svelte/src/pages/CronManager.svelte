<script>
    import { onMount } from 'svelte';

    let loading = true;
    let saving = false;
    let masterEnabled = false;
    let systemTime = '';
    let recentLogs = [];
    let jobs = [];

    // Modal universal popup
    let modal = {
        show: false,
        title: '',
        message: '',
        output: '',
        type: 'info', // 'success', 'error', 'info'
    };

    let showAddModal = false;
    let newJob = {
        title: '',
        description: '',
        type: 'custom',
        time: '04:00',
        command: '',
        days: [0, 1, 2, 3, 4, 5, 6]
    };

    const daysList = [
        { id: 1, label: 'Sen' },
        { id: 2, label: 'Sel' },
        { id: 3, label: 'Rab' },
        { id: 4, label: 'Kam' },
        { id: 5, label: 'Jum' },
        { id: 6, label: 'Sab' },
        { id: 0, label: 'Min' }
    ];

    async function fetchStatus() {
        try {
            const res = await fetch('/api/cron/status?_=' + Date.now());
            const json = await res.json();
            if (json.status === 'success' && json.data) {
                masterEnabled = !!json.data.master_enabled;
                systemTime = json.data.system_time || '';
                recentLogs = json.data.recent_logs || [];
                jobs = json.data.jobs || [];
            }
        } catch (e) {
            console.error('Failed to fetch cron status', e);
        } finally {
            loading = false;
        }
    }

    async function toggleMaster() {
        saving = true;
        try {
            const res = await fetch('/api/cron/toggle_master', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const json = await res.json();
            if (json.status === 'success') {
                masterEnabled = json.master_enabled;
                showNotification(
                    'Status Diperbarui',
                    masterEnabled
                        ? 'Scheduler aktif! Tugas terjadwal yang diaktifkan akan dieksekusi tepat waktu.'
                        : 'Scheduler dinonaktifkan. Semua tugas terjadwal dijeda sementara.',
                    'info'
                );
            }
        } catch (e) {
            showNotification('Gagal', 'Terjadi kesalahan saat mengubah status scheduler.', 'error');
        } finally {
            saving = false;
            fetchStatus();
        }
    }

    async function saveAllConfig() {
        saving = true;
        try {
            const payload = {
                master_enabled: masterEnabled,
                jobs: jobs
            };
            const res = await fetch('/api/cron/save', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            const json = await res.json();
            if (json.status === 'success') {
                showNotification('Tersimpan', 'Konfigurasi jadwal & tugas berhasil disimpan ke sistem.', 'success');
            } else {
                showNotification('Gagal', json.message || 'Gagal menyimpan pengaturan.', 'error');
            }
        } catch (e) {
            showNotification('Error', 'Gagal menghubungi server.', 'error');
        } finally {
            saving = false;
            fetchStatus();
        }
    }

    let runningJobId = null;
    async function runJobNow(job) {
        if (runningJobId) return;
        runningJobId = job.id;
        try {
            const res = await fetch('/api/cron/run_now', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    id: job.id,
                    type: job.type,
                    command: job.command
                })
            });
            const json = await res.json();
            if (json.status === 'success') {
                modal = {
                    show: true,
                    title: `Uji Coba: ${job.title}`,
                    message: json.exec_status === 'success' 
                        ? 'Tugas berhasil dieksekusi dengan sukses!' 
                        : 'Tugas dieksekusi namun mengembalikan kode peringatan/error.',
                    output: json.output || '(Tidak ada output shell)',
                    type: json.exec_status === 'success' ? 'success' : 'error'
                };
            } else {
                showNotification('Gagal Eksekusi', json.message || 'Terjadi kesalahan sistem.', 'error');
            }
        } catch (e) {
            showNotification('Error', 'Gagal memicu eksekusi tugas.', 'error');
        } finally {
            runningJobId = null;
            fetchStatus();
        }
    }

    function toggleDay(job, dayId) {
        if (!job.days) job.days = [];
        const idx = job.days.indexOf(dayId);
        if (idx >= 0) {
            job.days = job.days.filter(d => d !== dayId);
        } else {
            job.days = [...job.days, dayId];
        }
        jobs = [...jobs];
    }

    function setAllDays(job) {
        job.days = [0, 1, 2, 3, 4, 5, 6];
        jobs = [...jobs];
    }

    function setWeekdays(job) {
        job.days = [1, 2, 3, 4, 5];
        jobs = [...jobs];
    }

    function setWeekends(job) {
        job.days = [0, 6];
        jobs = [...jobs];
    }

    function deleteJob(jobId) {
        if (confirm('Hapus tugas terjadwal ini?')) {
            jobs = jobs.filter(j => j.id !== jobId);
            saveAllConfig();
        }
    }

    function addNewJob() {
        if (!newJob.title.trim()) {
            alert('Silakan isi judul tugas.');
            return;
        }
        if (newJob.type === 'custom' && !newJob.command.trim()) {
            alert('Silakan isi perintah shell yang akan dijalankan.');
            return;
        }

        const id = 'custom-' + Date.now().toString(36);
        jobs = [
            ...jobs,
            {
                id: id,
                title: newJob.title.trim(),
                description: newJob.description.trim() || 'Tugas kustom pengguna',
                enabled: false, // Default strictly OFF
                time: newJob.time || '04:00',
                days: [...newJob.days],
                type: newJob.type,
                command: newJob.command.trim(),
                last_status: 'never'
            }
        ];

        showAddModal = false;
        newJob = {
            title: '',
            description: '',
            type: 'custom',
            time: '04:00',
            command: '',
            days: [0, 1, 2, 3, 4, 5, 6]
        };
        saveAllConfig();
    }

    function showNotification(title, message, type = 'info') {
        modal = {
            show: true,
            title,
            message,
            output: '',
            type
        };
    }

    function closeModal() {
        modal.show = false;
    }

    function getJobBadge(type) {
        switch (type) {
            case 'reboot': return { label: 'SYSTEM REBOOT', bg: 'var(--pink)', text: '#fff', icon: 'fa-power-off' };
            case 'soft_reboot': return { label: 'SOFT REBOOT', bg: 'var(--purple)', text: '#fff', icon: 'fa-rotate' };
            case 'clean_ram': return { label: 'RAM CLEANER', bg: 'var(--lime)', text: '#000', icon: 'fa-broom' };
            case 'restart_bfr': return { label: 'BFR PROXY', bg: 'var(--cyan)', text: '#000', icon: 'fa-box-open' };
            case 'reconnect_data': return { label: 'DATA RESET', bg: 'var(--yellow)', text: '#000', icon: 'fa-signal' };
            default: return { label: 'CUSTOM SHELL', bg: 'var(--panel-alt)', text: 'var(--ink)', icon: 'fa-terminal' };
        }
    }

    onMount(() => {
        fetchStatus();
        const interval = setInterval(fetchStatus, 15000);
        return () => clearInterval(interval);
    });
</script>

<div class="wrap">
    <!-- HEADER -->
    <div class="header">
        <div class="header-main">
            <div class="eyebrow">
                <span class="live-dot {masterEnabled ? 'dot-active' : ''}"></span>
                Otomatisasi Sistem Android
            </div>
            <h1>Scheduled Tasks & Reboot</h1>
            <div class="desc">Jadwalkan reboot harian, pembersihan RAM otomatis, dan restart proxy saat jam hening.</div>
        </div>

        {#if systemTime}
            <div class="time-badge" title="Waktu server saat ini">
                <i class="fa-regular fa-clock"></i>
                <span>{systemTime}</span>
            </div>
        {/if}
    </div>

    <!-- MASTER TOGGLE HERO -->
    <div class="master-card {masterEnabled ? 'is-active' : 'is-inactive'}">
        <div class="master-info">
            <div class="master-badge">
                <i class="fa-solid {masterEnabled ? 'fa-circle-check' : 'fa-circle-pause'}"></i>
                {masterEnabled ? 'SCHEDULER AKTIF' : 'DEFAULT OFF (NONAKTIF)'}
            </div>
            <div class="master-title">
                {masterEnabled ? 'Semua Jadwal Berjalan Sesuai Waktu' : 'Scheduler Sedang Nonaktif'}
            </div>
            <div class="master-desc">
                {masterEnabled 
                    ? 'Daemon server memeriksa waktu setiap 30 detik untuk mengeksekusi tugas aktif.'
                    : 'Fitur dijaga tetap OFF secara default. Aktifkan sakelar jika ingin jadwal otomatis berjalan.'}
            </div>
        </div>

        <button 
            type="button" 
            class="master-btn {masterEnabled ? 'btn-stop' : 'btn-start'}" 
            disabled={saving}
            on:click={toggleMaster}
        >
            <i class="fa-solid {saving ? 'fa-spinner fa-spin' : (masterEnabled ? 'fa-power-off' : 'fa-play')}"></i>
            <span>{masterEnabled ? 'Matikan Scheduler' : 'Aktifkan Scheduler'}</span>
        </button>
    </div>

    <!-- ACTION BAR -->
    <div class="action-bar">
        <div class="action-left">
            <span class="section-title"><i class="fa-solid fa-list-check"></i> Daftar Tugas Terjadwal ({jobs.length})</span>
        </div>
        <div class="action-right">
            <button type="button" class="btn-sub" on:click={() => showAddModal = true}>
                <i class="fa-solid fa-plus"></i> Tambah Tugas
            </button>
            <button type="button" class="btn-primary" disabled={saving} on:click={saveAllConfig}>
                <i class="fa-solid {saving ? 'fa-spinner fa-spin' : 'fa-floppy-disk'}"></i> Simpan
            </button>
        </div>
    </div>

    <!-- JOBS LIST -->
    {#if loading}
        <div class="loading-state">
            <i class="fa-solid fa-spinner fa-spin"></i>
            <span>Memuat jadwal sistem...</span>
        </div>
    {:else}
        <div class="jobs-list">
            {#each jobs as job (job.id)}
                {@const badge = getJobBadge(job.type)}
                <div class="job-card {job.enabled && masterEnabled ? 'job-running' : ''} {!job.enabled ? 'job-disabled' : ''}">
                    <!-- CARD HEADER -->
                    <div class="job-top">
                        <div class="job-meta">
                            <span class="job-badge" style="background:{badge.bg}; color:{badge.text};">
                                <i class="fa-solid {badge.icon}"></i> {badge.label}
                            </span>
                            <span class="job-time-chip">
                                <i class="fa-regular fa-clock"></i> {job.time}
                            </span>
                        </div>

                        <!-- TOGGLE SWITCH -->
                        <label class="toggle-switch" title="Aktifkan/Nonaktifkan tugas ini">
                            <input 
                                type="checkbox" 
                                bind:checked={job.enabled} 
                            />
                            <span class="slider"></span>
                        </label>
                    </div>

                    <!-- CARD BODY -->
                    <div class="job-body">
                        <div class="job-title-row">
                            <h3 class="job-title">{job.title}</h3>
                            {#if job.id.startsWith('custom-')}
                                <button type="button" class="btn-del" on:click={() => deleteJob(job.id)} title="Hapus tugas kustom">
                                    <i class="fa-solid fa-trash-can"></i>
                                </button>
                            {/if}
                        </div>
                        <p class="job-desc">{job.description}</p>

                        <!-- COMMAND PREVIEW IF CUSTOM -->
                        {#if job.type === 'custom'}
                            <div class="command-box">
                                <span class="cmd-prompt">#</span>
                                <input 
                                    type="text" 
                                    class="cmd-input" 
                                    bind:value={job.command} 
                                    placeholder="echo 'test'" 
                                />
                            </div>
                        {:else if job.type === 'reboot' || job.type === 'soft_reboot'}
                            <div class="reboot-type-select">
                                <label class="radio-label">
                                    <input 
                                        type="radio" 
                                        bind:group={job.type} 
                                        value="reboot" 
                                    />
                                    <span>Full Device Reboot</span>
                                </label>
                                <label class="radio-label">
                                    <input 
                                        type="radio" 
                                        bind:group={job.type} 
                                        value="soft_reboot" 
                                    />
                                    <span>Userspace Soft Reboot (Cepat)</span>
                                </label>
                            </div>
                        {/if}

                        <!-- TIME & DAYS CONFIG -->
                        <div class="job-schedule-box">
                            <div class="time-input-row">
                                <label class="time-label">Jam Eksekusi:</label>
                                <input 
                                    type="time" 
                                    class="time-picker" 
                                    bind:value={job.time} 
                                />
                                <div class="quick-times">
                                    <button type="button" class="chip-qtime" on:click={() => job.time = '03:00'}>03:00</button>
                                    <button type="button" class="chip-qtime" on:click={() => job.time = '04:00'}>04:00</button>
                                    <button type="button" class="chip-qtime" on:click={() => job.time = '05:00'}>05:00</button>
                                </div>
                            </div>

                            <!-- DAYS SELECTOR -->
                            <div class="days-row">
                                <span class="days-label">Hari Aktif:</span>
                                <div class="days-chips">
                                    {#each daysList as d}
                                        <button 
                                            type="button" 
                                            class="day-chip {job.days && job.days.includes(d.id) ? 'active' : ''}"
                                            on:click={() => toggleDay(job, d.id)}
                                        >
                                            {d.label}
                                        </button>
                                    {/each}
                                </div>
                                <div class="days-presets">
                                    <button type="button" class="btn-preset-day" on:click={() => setAllDays(job)}>Semua</button>
                                    <button type="button" class="btn-preset-day" on:click={() => setWeekdays(job)}>Kerja</button>
                                    <button type="button" class="btn-preset-day" on:click={() => setWeekends(job)}>Weekend</button>
                                </div>
                            </div>
                        </div>

                        <!-- LAST RUN & TEST ACTION -->
                        <div class="job-footer">
                            <div class="last-run-info">
                                {#if job.last_run}
                                    <i class="fa-solid {job.last_status === 'success' ? 'fa-check text-success' : 'fa-triangle-exclamation text-error'}"></i>
                                    <span>Terakhir: <strong>{job.last_run}</strong> ({job.last_status})</span>
                                {:else}
                                    <i class="fa-regular fa-clock text-muted"></i>
                                    <span class="text-muted">Belum pernah dijalankan</span>
                                {/if}
                            </div>

                            <button 
                                type="button" 
                                class="btn-run-now" 
                                disabled={runningJobId === job.id}
                                on:click={() => runJobNow(job)}
                                title="Uji coba jalankan perintah ini sekarang"
                            >
                                <i class="fa-solid {runningJobId === job.id ? 'fa-spinner fa-spin' : 'fa-play'}"></i>
                                <span>{runningJobId === job.id ? 'Mengeksekusi...' : 'Jalankan Sekarang'}</span>
                            </button>
                        </div>
                    </div>
                </div>
            {/each}
        </div>
    {/if}

    <!-- RECENT LOGS -->
    <div class="logs-card">
        <div class="logs-header">
            <span class="logs-title"><i class="fa-solid fa-receipt"></i> Log Eksekusi Scheduler</span>
            <button type="button" class="btn-refresh-log" on:click={fetchStatus} title="Refresh log">
                <i class="fa-solid fa-rotate-right"></i>
            </button>
        </div>
        <div class="logs-body">
            {#if recentLogs && recentLogs.length > 0}
                {#each recentLogs.slice().reverse() as log}
                    <div class="log-line">{log}</div>
                {/each}
            {:else}
                <div class="log-empty">Belum ada riwayat eksekusi tercatat.</div>
            {/if}
        </div>
    </div>
</div>

<!-- MODAL TAMBAH TUGAS KUSTOM -->
{#if showAddModal}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal-overlay" on:click={() => showAddModal = false}>
        <div class="modal-card" on:click|stopPropagation>
            <div class="modal-header">
                <h3><i class="fa-solid fa-plus-circle" style="color:var(--yellow);"></i> Tambah Tugas Kustom</h3>
                <button type="button" class="modal-close" on:click={() => showAddModal = false}>
                    <i class="fa-solid fa-xmark"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="form-group">
                    <label class="form-label">Nama / Judul Tugas</label>
                    <input type="text" class="form-input" bind:value={newJob.title} placeholder="Contoh: Backup File Hosts / Ping Server" />
                </div>
                <div class="form-group">
                    <label class="form-label">Deskripsi Singkat</label>
                    <input type="text" class="form-input" bind:value={newJob.description} placeholder="Keterangan fungsi tugas ini..." />
                </div>
                <div class="form-group">
                    <label class="form-label">Jam Eksekusi (WIB / Lokal)</label>
                    <input type="time" class="form-input" bind:value={newJob.time} />
                </div>
                <div class="form-group">
                    <label class="form-label">Perintah Shell Root (sh / bash)</label>
                    <textarea 
                        class="form-textarea" 
                        bind:value={newJob.command} 
                        rows="3" 
                        placeholder="Contoh: ping -c 3 1.1.1.1 atau sh /data/myscript.sh"
                    ></textarea>
                </div>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn-sub" on:click={() => showAddModal = false}>Batal</button>
                <button type="button" class="btn-primary" on:click={addNewJob}>Simpan Tugas</button>
            </div>
        </div>
    </div>
{/if}

<!-- MODAL OUTPUT RUN / UNIVERSAL POPUP -->
{#if modal.show}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal-overlay" on:click={closeModal}>
        <div class="modal-card" on:click|stopPropagation>
            <div class="modal-header">
                <h3>
                    <i class="fa-solid {modal.type === 'success' ? 'fa-circle-check text-success' : (modal.type === 'error' ? 'fa-triangle-exclamation text-error' : 'fa-circle-info text-info')}"></i>
                    {modal.title}
                </h3>
                <button type="button" class="modal-close" on:click={closeModal}>
                    <i class="fa-solid fa-xmark"></i>
                </button>
            </div>
            <div class="modal-body">
                <p class="modal-msg">{modal.message}</p>
                {#if modal.output}
                    <div class="output-header">Hasil Output Shell:</div>
                    <pre class="terminal-output">{modal.output}</pre>
                {/if}
            </div>
            <div class="modal-footer">
                <button type="button" class="btn-primary" on:click={closeModal}>Tutup</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .wrap {
        padding: 16px;
        max-width: 720px;
        margin: 0 auto;
        padding-bottom: 95px;
        box-sizing: border-box;
    }

    /* HEADER */
    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 12px;
        margin-bottom: 16px;
        flex-wrap: wrap;
    }

    .header-main {
        flex: 1;
        min-width: 240px;
    }

    .eyebrow {
        font-size: 0.72rem;
        font-weight: 800;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--yellow);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 3px;
    }

    .live-dot {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: var(--muted);
        display: inline-block;
    }

    .live-dot.dot-active {
        background: var(--lime);
        box-shadow: 0 0 6px var(--lime);
    }

    h1 {
        font-size: 1.6rem;
        font-weight: 950;
        letter-spacing: -0.03em;
        margin: 0 0 4px 0;
        color: var(--ink);
        text-transform: uppercase;
    }

    .desc {
        font-size: 0.8rem;
        color: var(--muted);
        line-height: 1.4;
        font-weight: 700;
    }

    .time-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 8px;
        padding: 6px 10px;
        font-size: 11px;
        font-weight: 850;
        color: var(--cyan);
        box-shadow: 2px 2px 0 var(--line);
    }

    /* MASTER CARD */
    .master-card {
        border: 2.5px solid var(--line);
        border-radius: 14px;
        padding: 16px;
        margin-bottom: 18px;
        box-shadow: var(--shadow);
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        flex-wrap: wrap;
        transition: all 0.15s ease;
    }

    .master-card.is-active {
        background: linear-gradient(135deg, rgba(85, 239, 196, 0.15), var(--paper));
        border-color: var(--lime);
    }

    .master-card.is-inactive {
        background: var(--paper);
    }

    .master-info {
        flex: 1;
        min-width: 220px;
    }

    .master-badge {
        font-size: 10px;
        font-weight: 950;
        letter-spacing: 0.05em;
        text-transform: uppercase;
        display: inline-flex;
        align-items: center;
        gap: 5px;
        margin-bottom: 4px;
        color: var(--muted);
    }

    .master-card.is-active .master-badge {
        color: var(--lime);
    }

    .master-title {
        font-size: 15px;
        font-weight: 900;
        color: var(--ink);
        margin-bottom: 4px;
    }

    .master-desc {
        font-size: 11px;
        font-weight: 700;
        color: var(--muted);
        line-height: 1.35;
    }

    .master-btn {
        padding: 12px 18px;
        border-radius: 10px;
        font-size: 12px;
        font-weight: 950;
        text-transform: uppercase;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        border: 2px solid var(--line);
        font-family: inherit;
        box-shadow: 2px 2px 0 var(--line);
        transition: all 0.12s;
    }

    .master-btn:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--line);
    }

    .master-btn:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }

    .btn-start {
        background: var(--lime);
        color: #000;
    }

    .btn-stop {
        background: var(--pink);
        color: #fff;
    }

    /* ACTION BAR */
    .action-bar {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        flex-wrap: wrap;
        gap: 10px;
    }

    .section-title {
        font-size: 12px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        color: var(--ink);
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }

    .action-right {
        display: flex;
        gap: 8px;
    }

    .btn-sub {
        background: var(--panel-alt);
        border: 2px solid var(--line);
        color: var(--ink);
        padding: 7px 12px;
        border-radius: 8px;
        font-size: 11px;
        font-weight: 850;
        cursor: pointer;
        box-shadow: 2px 2px 0 var(--line);
        font-family: inherit;
        display: inline-flex;
        align-items: center;
        gap: 5px;
    }

    .btn-primary {
        background: var(--yellow);
        border: 2px solid var(--line);
        color: #000;
        padding: 7px 14px;
        border-radius: 8px;
        font-size: 11px;
        font-weight: 950;
        cursor: pointer;
        box-shadow: 2px 2px 0 var(--line);
        font-family: inherit;
        display: inline-flex;
        align-items: center;
        gap: 5px;
    }

    .btn-sub:hover, .btn-primary:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--line);
    }

    .btn-sub:active, .btn-primary:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }

    /* JOBS LIST */
    .jobs-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 20px;
    }

    .job-card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 14px;
        box-shadow: var(--shadow);
        transition: all 0.12s;
    }

    .job-card.job-running {
        border-color: var(--lime);
    }

    .job-card.job-disabled {
        opacity: 0.85;
    }

    .job-top {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
        gap: 8px;
    }

    .job-meta {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .job-badge {
        font-size: 8.5px;
        font-weight: 950;
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid var(--line);
        letter-spacing: 0.04em;
        display: inline-flex;
        align-items: center;
        gap: 4px;
    }

    .job-time-chip {
        font-size: 11px;
        font-weight: 900;
        background: var(--panel-alt);
        border: 1.5px solid var(--line);
        border-radius: 6px;
        padding: 2px 6px;
        color: var(--ink);
        display: inline-flex;
        align-items: center;
        gap: 4px;
    }

    /* TOGGLE SWITCH */
    .toggle-switch {
        position: relative;
        display: inline-block;
        width: 44px;
        height: 24px;
        flex-shrink: 0;
    }

    .toggle-switch input {
        opacity: 0;
        width: 0;
        height: 0;
    }

    .slider {
        position: absolute;
        cursor: pointer;
        top: 0; left: 0; right: 0; bottom: 0;
        background-color: var(--panel-alt);
        border: 2px solid var(--line);
        transition: 0.2s;
        border-radius: 24px;
    }

    .slider:before {
        position: absolute;
        content: "";
        height: 14px;
        width: 14px;
        left: 3px;
        bottom: 3px;
        background-color: var(--muted);
        border: 1.5px solid var(--line);
        transition: 0.2s;
        border-radius: 50%;
    }

    input:checked + .slider {
        background-color: var(--lime);
    }

    input:checked + .slider:before {
        transform: translateX(18px);
        background-color: #000;
    }

    .job-title-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 3px;
    }

    .job-title {
        font-size: 13.5px;
        font-weight: 950;
        margin: 0;
        color: var(--ink);
    }

    .btn-del {
        background: none;
        border: none;
        color: var(--pink);
        cursor: pointer;
        font-size: 13px;
        padding: 2px;
    }

    .job-desc {
        font-size: 10.5px;
        color: var(--muted);
        font-weight: 700;
        margin: 0 0 10px 0;
        line-height: 1.35;
    }

    .command-box {
        background: var(--bg);
        border: 1.5px solid var(--line);
        border-radius: 6px;
        padding: 4px 8px;
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 10px;
    }

    .cmd-prompt {
        color: var(--lime);
        font-weight: 900;
        font-family: monospace;
    }

    .cmd-input {
        flex: 1;
        background: none;
        border: none;
        color: var(--ink);
        font-family: monospace;
        font-size: 11px;
        outline: none;
    }

    .reboot-type-select {
        display: flex;
        gap: 14px;
        margin-bottom: 10px;
        flex-wrap: wrap;
        font-size: 10.5px;
        font-weight: 800;
        color: var(--ink);
    }

    .radio-label {
        display: flex;
        align-items: center;
        gap: 5px;
        cursor: pointer;
    }

    /* SCHEDULE BOX */
    .job-schedule-box {
        background: var(--panel-alt);
        border: 1.5px solid var(--line);
        border-radius: 8px;
        padding: 10px;
        margin-bottom: 10px;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .time-input-row {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .time-label {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
    }

    .time-picker {
        border: 1.5px solid var(--line);
        border-radius: 6px;
        padding: 3px 8px;
        font-family: inherit;
        font-size: 11.5px;
        font-weight: 900;
        background: var(--paper);
        color: var(--ink);
    }

    .quick-times {
        display: flex;
        gap: 4px;
    }

    .chip-qtime {
        background: var(--paper);
        border: 1px solid var(--line);
        border-radius: 4px;
        padding: 2px 6px;
        font-size: 9.5px;
        font-weight: 850;
        cursor: pointer;
        color: var(--muted);
    }

    .chip-qtime:hover {
        color: var(--yellow);
        border-color: var(--yellow);
    }

    .days-row {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .days-label {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
    }

    .days-chips {
        display: flex;
        gap: 3px;
        flex-wrap: wrap;
    }

    .day-chip {
        padding: 3px 7px;
        border-radius: 5px;
        font-size: 9.5px;
        font-weight: 900;
        border: 1.5px solid var(--line);
        background: var(--paper);
        color: var(--muted);
        cursor: pointer;
        transition: all 0.1s;
    }

    .day-chip.active {
        background: var(--yellow);
        color: #000;
        border-color: var(--line);
    }

    .days-presets {
        display: flex;
        gap: 4px;
        margin-left: auto;
    }

    .btn-preset-day {
        background: none;
        border: 1px dashed var(--line);
        border-radius: 4px;
        padding: 2px 5px;
        font-size: 9px;
        font-weight: 800;
        color: var(--cyan);
        cursor: pointer;
    }

    /* FOOTER */
    .job-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 8px;
        padding-top: 6px;
        flex-wrap: wrap;
    }

    .last-run-info {
        font-size: 10px;
        font-weight: 800;
        display: flex;
        align-items: center;
        gap: 5px;
    }

    .btn-run-now {
        background: var(--panel-alt);
        border: 1.5px solid var(--line);
        border-radius: 6px;
        padding: 5px 10px;
        font-size: 10px;
        font-weight: 900;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 5px;
        color: var(--ink);
        box-shadow: 1.5px 1.5px 0 var(--line);
        font-family: inherit;
    }

    .btn-run-now:hover {
        background: var(--paper);
        color: var(--lime);
        border-color: var(--lime);
    }

    .btn-run-now:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }

    /* RECENT LOGS */
    .logs-card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 12px;
        box-shadow: var(--shadow);
    }

    .logs-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
    }

    .logs-title {
        font-size: 11px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--ink);
        display: inline-flex;
        align-items: center;
        gap: 5px;
    }

    .btn-refresh-log {
        background: none;
        border: none;
        color: var(--cyan);
        cursor: pointer;
        font-size: 12px;
    }

    .logs-body {
        background: var(--bg);
        border: 1.5px solid var(--line);
        border-radius: 8px;
        padding: 10px;
        max-height: 140px;
        overflow-y: auto;
        font-family: monospace;
        font-size: 9.5px;
        line-height: 1.4;
    }

    .log-line {
        color: var(--ink);
        margin-bottom: 3px;
    }

    .log-empty {
        color: var(--muted);
        text-align: center;
        padding: 12px;
    }

    /* MODAL */
    .modal-overlay {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        padding: 16px;
    }

    .modal-card {
        background: var(--paper);
        border: 2.5px solid var(--line);
        border-radius: 14px;
        width: 100%;
        max-width: 440px;
        box-shadow: 6px 6px 0 var(--line);
        box-sizing: border-box;
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 14px;
        border-bottom: 2px solid var(--line);
    }

    .modal-header h3 {
        margin: 0;
        font-size: 13.5px;
        font-weight: 950;
        display: flex;
        align-items: center;
        gap: 6px;
        color: var(--ink);
    }

    .modal-close {
        background: none;
        border: none;
        font-size: 15px;
        cursor: pointer;
        color: var(--muted);
    }

    .modal-body {
        padding: 14px;
    }

    .modal-msg {
        font-size: 11.5px;
        font-weight: 800;
        color: var(--ink);
        margin: 0 0 10px 0;
    }

    .output-header {
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
        margin-bottom: 4px;
    }

    .terminal-output {
        background: var(--bg);
        border: 1.5px solid var(--line);
        border-radius: 8px;
        padding: 10px;
        font-family: monospace;
        font-size: 10px;
        color: var(--lime);
        max-height: 180px;
        overflow-y: auto;
        white-space: pre-wrap;
        word-break: break-all;
        margin: 0;
    }

    .modal-footer {
        padding: 10px 14px;
        border-top: 2px solid var(--line);
        display: flex;
        justify-content: flex-end;
        gap: 8px;
    }

    /* FORM IN MODAL */
    .form-group {
        margin-bottom: 10px;
    }

    .form-label {
        display: block;
        font-size: 10px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--muted);
        margin-bottom: 4px;
    }

    .form-input, .form-textarea {
        width: 100%;
        padding: 8px 10px;
        border: 1.5px solid var(--line);
        border-radius: 6px;
        font-family: inherit;
        font-size: 11px;
        font-weight: 700;
        background: var(--bg);
        color: var(--ink);
        box-sizing: border-box;
    }

    .form-textarea {
        font-family: monospace;
    }

    .text-success { color: var(--lime); }
    .text-error { color: var(--pink); }
    .text-info { color: var(--cyan); }
    .text-muted { color: var(--muted); }

    .loading-state {
        text-align: center;
        padding: 30px;
        color: var(--muted);
        font-size: 13px;
        font-weight: 800;
        display: flex;
        flex-direction: column;
        gap: 10px;
        align-items: center;
    }

    @media (max-width: 480px) {
        .wrap {
            padding: 12px;
        }
        .header {
            margin-bottom: 12px;
        }
        .master-card {
            padding: 12px;
            gap: 12px;
        }
        .master-btn {
            width: 100%;
            justify-content: center;
        }
        .action-bar {
            flex-direction: column;
            align-items: stretch;
        }
        .action-right {
            justify-content: flex-end;
        }
        .days-presets {
            margin-left: 0;
            width: 100%;
            justify-content: flex-end;
        }
    }
</style>

