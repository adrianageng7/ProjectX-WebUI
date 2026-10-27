<script>
    import { onMount } from 'svelte';

    // State Navigasi
    let activeTab = 'add'; // Default langsung ke 'add' agar user langsung bisa paste link & lihat form!
    let formViewTab = 'form'; // 'form' | 'raw' (Tabs di dalam form: Formulir Mudah vs Kode YAML)

    // Targets & Akun Tersimpan
    let targetFiles = [];
    let selectedTarget = '';
    let loadingTargets = true;
    let loadingAccounts = false;
    let accounts = [];
    let accountsStats = {};
    let totalAccounts = 0;

    // Filter & Search di Tab Kelola Akun
    let searchQuery = '';
    let filterType = 'all';

    // State Link Input & Akun Aktif di Form
    let rawLinks = '';
    let parsedAccounts = [];
    let activeAccountIndex = 0;
    let showAdvancedSettings = false;

    // Model Akun Kosong Default untuk Form
    let currentAccount = {
        name: '',
        type: 'vless',
        server: '',
        port: 443,
        uuid: '',
        password: '',
        cipher: 'auto',
        tls: true,
        skip_cert_verify: true,
        servername: '',
        network: 'ws',
        ws_path: '/',
        ws_host: '',
        grpc_service_name: '',
        reality_public_key: '',
        reality_short_id: '',
        udp: true
    };

    let rawYamlLive = '';

    // Modal Edit Akun (Kelola Akun)
    let showEditModal = false;
    let editModalTab = 'form';
    let editingAccount = null;
    let editingRawYaml = '';

    // Modal Hapus Akun
    let showDeleteModal = false;
    let deletingAccount = null;

    // Modal Buat Provider Baru
    let showCreateProviderModal = false;
    let newProviderName = '';

    // Modal & Status Template BFR
    let needsTemplate = false;
    let configMode = 'none'; // 'split', 'standard', 'none'
    let selectedTemplateMode = 'split'; // 'split' or 'standard'
    let isApplyingTemplate = false;
    let showTemplateModal = false;
    let showRestartBFRConfirm = false;
    let isRestartingBFR = false;

    // Status Notifikasi Toast
    let isSaving = false;
    let toast = { show: false, title: '', desc: '' };
    let toastTimer = null;

    function showToast(title, desc) {
        if (toastTimer) clearTimeout(toastTimer);
        toast = { show: true, title, desc };
        toastTimer = setTimeout(() => {
            toast.show = false;
        }, 3200);
    }

    // Parser Link VPN ke Objek Akun Terstruktur
    function parseVpnLink(link) {
        link = link.trim();
        if (!link) return null;

        // 1. VMess
        if (link.toLowerCase().startsWith('vmess://')) {
            try {
                let b64 = link.substring(8).replace(/-/g, '+').replace(/_/g, '/');
                while (b64.length % 4) b64 += '=';
                let decoded = atob(b64);
                try { decoded = decodeURIComponent(escape(decoded)); } catch (e) {}
                let json = JSON.parse(decoded);
                return {
                    name: (json.ps || 'VMess-Node').replace(/^["']|["']$/g, ''),
                    type: 'vmess',
                    server: json.add || '',
                    port: parseInt(json.port) || 443,
                    uuid: json.id || '',
                    password: '',
                    cipher: 'auto',
                    tls: json.tls === 'tls',
                    skip_cert_verify: true,
                    servername: json.sni || json.host || '',
                    network: json.net || 'ws',
                    ws_path: json.path || '/',
                    ws_host: json.host || '',
                    grpc_service_name: json.path || 'v2ray',
                    reality_public_key: '',
                    reality_short_id: '',
                    udp: true
                };
            } catch (e) { return null; }
        }

        // 2. VLESS / Trojan
        if (link.toLowerCase().startsWith('vless://') || link.toLowerCase().startsWith('trojan://')) {
            try {
                let isVless = link.toLowerCase().startsWith('vless://');
                let url = new URL(link);
                let uuid = url.username;
                let server = url.hostname;
                let port = parseInt(url.port) || 443;
                let name = decodeURIComponent(url.hash.substring(1)) || (isVless ? 'VLESS-Node' : 'Trojan-Node');
                name = name.replace(/^["']|["']$/g, '');
                let security = url.searchParams.get('security');
                let sni = url.searchParams.get('sni') || '';
                let tls = (security === 'tls' || security === 'reality' || sni !== '');
                let netType = url.searchParams.get('type') || 'ws';

                let rawPath = url.searchParams.get('path') || '/';
                try { rawPath = decodeURIComponent(rawPath); } catch(e) {}

                let rawHost = url.searchParams.get('host') || '';
                try { rawHost = decodeURIComponent(rawHost); } catch(e) {}

                let item = {
                    name: name,
                    type: isVless ? 'vless' : 'trojan',
                    server: server,
                    port: port,
                    uuid: isVless ? uuid : '',
                    password: !isVless ? uuid : '',
                    cipher: 'auto',
                    tls: tls,
                    skip_cert_verify: true,
                    servername: sni,
                    network: netType,
                    ws_path: rawPath,
                    ws_host: rawHost,
                    grpc_service_name: url.searchParams.get('serviceName') ? decodeURIComponent(url.searchParams.get('serviceName')) : 'v2ray',
                    reality_public_key: '',
                    reality_short_id: '',
                    udp: true
                };

                if (security === 'reality') {
                    item.reality_public_key = url.searchParams.get('pbk') || '';
                    item.reality_short_id = url.searchParams.get('sid') || '';
                }
                return item;
            } catch (e) { return null; }
        }

        // 3. Shadowsocks
        if (link.toLowerCase().startsWith('ss://')) {
            try {
                let rest = link.substring(5);
                let name = 'SS-Node';
                if (rest.includes('#')) {
                    const parts = rest.split('#');
                    rest = parts[0];
                    name = decodeURIComponent(parts[1]).replace(/^["']|["']$/g, '');
                }
                let server = '', port = 443, method = 'aes-256-gcm', password = '';
                if (rest.includes('@')) {
                    const atParts = rest.split('@');
                    let b64 = atParts[0].replace(/-/g, '+').replace(/_/g, '/');
                    while (b64.length % 4) b64 += '=';
                    const creds = atob(b64).split(':');
                    method = creds[0];
                    password = creds.slice(1).join(':');
                    const hostParts = atParts[1].split(':');
                    server = hostParts[0];
                    port = parseInt(hostParts[1]) || 443;
                } else {
                    let b64 = rest.replace(/-/g, '+').replace(/_/g, '/');
                    while (b64.length % 4) b64 += '=';
                    const decoded = atob(b64);
                    const atParts = decoded.split('@');
                    const creds = atParts[0].split(':');
                    method = creds[0];
                    password = creds.slice(1).join(':');
                    const hostParts = atParts[1].split(':');
                    server = hostParts[0];
                    port = parseInt(hostParts[1]) || 443;
                }
                return {
                    name: name, type: 'ss', server: server, port: port,
                    cipher: method, password: password, tls: false, skip_cert_verify: false,
                    network: 'tcp', udp: true, reality_public_key: '', reality_short_id: ''
                };
            } catch (e) { return null; }
        }
        return null;
    }

    // Bangun string Clash YAML dari objek akun
    function buildYamlFromItem(item) {
        if (!item || !item.server) return '';
        let y = `- name: "${item.name || 'Node'}"\n` +
                `  type: ${item.type}\n` +
                `  server: ${item.server}\n` +
                `  port: ${item.port || 443}\n`;

        if (item.type === 'vless' || item.type === 'vmess') {
            y += `  uuid: ${item.uuid}\n` +
                 `  cipher: ${item.cipher || 'auto'}\n`;
            if (item.type === 'vmess') y += `  alterId: 0\n`;
        } else if (item.type === 'trojan' || item.type === 'ss') {
            y += `  password: ${item.password}\n`;
            if (item.type === 'ss') y += `  cipher: ${item.cipher || 'aes-256-gcm'}\n`;
        }

        y += `  tls: ${item.tls ? 'true' : 'false'}\n` +
             `  skip-cert-verify: ${item.skip_cert_verify ? 'true' : 'false'}\n`;

        if (item.servername) y += `  servername: ${item.servername}\n`;

        if (item.reality_public_key) {
            y += `  reality-opts:\n    public-key: ${item.reality_public_key}\n`;
            if (item.reality_short_id) y += `    short-id: ${item.reality_short_id}\n`;
        }

        let net = item.network || 'ws';
        y += `  network: ${net}\n`;
        if (net === 'ws') {
            y += `  ws-opts:\n    path: ${item.ws_path || '/'}\n`;
            if (item.ws_host) y += `    headers:\n      Host: ${item.ws_host}\n`;
        } else if (net === 'grpc') {
            y += `  grpc-opts:\n    grpc-service-name: ${item.grpc_service_name || 'v2ray'}\n`;
        }

        y += `  udp: ${item.udp ? 'true' : 'false'}\n`;
        return y;
    }

    // Reconstruct shareable VPN Link
    function generateVpnLink(item) {
        if (!item || !item.server) return '';
        const nameEnc = encodeURIComponent(item.name || 'node');

        if (item.type === 'vless') {
            let q = [];
            if (item.tls) q.push('security=tls');
            if (item.servername) q.push('sni=' + encodeURIComponent(item.servername));
            let net = item.network || 'ws';
            q.push('type=' + encodeURIComponent(net));
            if (net === 'ws') {
                q.push('path=' + encodeURIComponent(item.ws_path || '/'));
                if (item.ws_host) q.push('host=' + encodeURIComponent(item.ws_host));
            } else if (net === 'grpc') {
                q.push('serviceName=' + encodeURIComponent(item.grpc_service_name || 'v2ray'));
            }
            return `vless://${item.uuid}@${item.server}:${item.port}?${q.join('&')}#${nameEnc}`;
        }

        if (item.type === 'trojan') {
            let q = [];
            if (item.tls) q.push('security=tls');
            if (item.servername) q.push('sni=' + encodeURIComponent(item.servername));
            let net = item.network || 'ws';
            q.push('type=' + encodeURIComponent(net));
            if (net === 'ws') {
                q.push('path=' + encodeURIComponent(item.ws_path || '/'));
                if (item.ws_host) q.push('host=' + encodeURIComponent(item.ws_host));
            }
            return `trojan://${item.password}@${item.server}:${item.port}?${q.join('&')}#${nameEnc}`;
        }

        if (item.type === 'vmess') {
            let obj = {
                v: "2", ps: item.name, add: item.server, port: String(item.port || 443),
                id: item.uuid, aid: "0", scy: "auto", net: item.network || "ws", type: "none",
                host: item.ws_host || "", path: item.ws_path || "/", tls: item.tls ? "tls" : "",
                sni: item.servername || ""
            };
            try { return "vmess://" + btoa(unescape(encodeURIComponent(JSON.stringify(obj)))); } catch (e) { return ''; }
        }

        if (item.type === 'ss') {
            let creds = (item.cipher || 'aes-256-gcm') + ':' + (item.password || '');
            return `ss://${btoa(creds)}@${item.server}:${item.port}#${nameEnc}`;
        }

        return '';
    }

    // Reaktif: Mengurai link yang diketik/ditempel
    $: {
        const text = (rawLinks || '').trim();
        if (!text) {
            parsedAccounts = [];
            activeAccountIndex = 0;
        } else {
            const lines = text.split('\n');
            const items = [];
            lines.forEach(l => {
                const line = l.trim();
                if (line) {
                    const parsed = parseVpnLink(line);
                    if (parsed) items.push(parsed);
                }
            });
            parsedAccounts = items;
            if (activeAccountIndex >= parsedAccounts.length) {
                activeAccountIndex = 0;
            }
            if (parsedAccounts.length > 0) {
                // Salin nilai akun pertama ke formulir
                currentAccount = { ...parsedAccounts[activeAccountIndex] };
            }
        }
    }

    // Update real-time YAML ketika formulir berubah
    $: {
        if (currentAccount && currentAccount.server) {
            rawYamlLive = buildYamlFromItem(currentAccount);
        } else {
            rawYamlLive = '';
        }
    }

    // Pilih akun jika ada multiple links
    function selectAccountChip(idx) {
        activeAccountIndex = idx;
        if (parsedAccounts[idx]) {
            currentAccount = { ...parsedAccounts[idx] };
        }
    }

    // Sinkronisasi perubahan kembali ke array parsedAccounts jika mengedit
    function onFormFieldChange() {
        if (parsedAccounts.length > 0 && parsedAccounts[activeAccountIndex]) {
            parsedAccounts[activeAccountIndex] = { ...currentAccount };
        }
    }

    // Filter akun tersimpan
    $: filteredAccounts = accounts.filter(acc => {
        if (filterType !== 'all' && (acc.type || '').toLowerCase() !== filterType) {
            return false;
        }
        if (searchQuery.trim()) {
            const q = searchQuery.toLowerCase().trim();
            const matchName = (acc.name || '').toLowerCase().includes(q);
            const matchServer = (acc.server || '').toLowerCase().includes(q);
            const matchSni = (acc.servername || acc.ws_host || '').toLowerCase().includes(q);
            return matchName || matchServer || matchSni;
        }
        return true;
    });

    // API: Muat Targets
    async function loadTargets(keepSelection = true) {
        loadingTargets = true;
        try {
            const res = await fetch('/api/clash/targets?_=' + Date.now());
            const data = await res.json();
            if (data.status === 'success' && Array.isArray(data.targets)) {
                targetFiles = data.targets;
                needsTemplate = data.needs_template === true;
                configMode = data.config_mode || 'none';
                if (!keepSelection || !selectedTarget || !targetFiles.find(t => t.path === selectedTarget)) {
                    if (targetFiles.length > 0) {
                        const defProv = targetFiles.find(t => t.path.includes('my_proxies'));
                        if (defProv) {
                            selectedTarget = defProv.path;
                        } else {
                            selectedTarget = targetFiles[0].path;
                        }
                    }
                }
                if (selectedTarget) {
                    await loadAccounts(selectedTarget);
                }
            }
        } catch (e) {
            console.error('Gagal memuat target files:', e);
        } finally {
            loadingTargets = false;
        }
    }

    // Terapkan Template BFR Sesuai Mode (Pisah Trafik / Standar)
    async function applyBFRTemplate(mode = null) {
        isApplyingTemplate = true;
        const targetMode = mode || selectedTemplateMode;
        try {
            const res = await fetch('/api/clash/template/apply', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ mode: targetMode })
            });
            const data = await res.json();
            if (data.status === 'success') {
                showToast('Template Berhasil Dipasang!', data.message || 'Config BFR telah diperbarui.');
                showTemplateModal = false;
                needsTemplate = false;
                await loadTargets(false);
                if (data.default_provider) {
                    selectedTarget = data.default_provider;
                    await loadAccounts(selectedTarget);
                }
                showRestartBFRConfirm = true;
            } else {
                showToast('Gagal', data.message || 'Gagal memasang template');
            }
        } catch (e) {
            showToast('Error', 'Gagal memproses pemasangan template');
        } finally {
            isApplyingTemplate = false;
        }
    }

    // Restart Layanan BFR
    async function restartBFRService() {
        isRestartingBFR = true;
        try {
            const res = await fetch('/api/bfr/control', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: 'restart' })
            });
            const data = await res.json();
            if (data.status === 'success') {
                showToast('BFR Direstart', 'Layanan Box for Root berhasil direstart!');
                showRestartBFRConfirm = false;
            } else {
                showToast('Info', data.message || 'Perintah restart dikirim');
            }
        } catch (e) {
            showToast('Error', 'Gagal merestart layanan BFR');
        } finally {
            isRestartingBFR = false;
        }
    }

    // API: Muat Akun dari File Target
    async function loadAccounts(targetPath) {
        if (!targetPath) return;
        loadingAccounts = true;
        try {
            const res = await fetch('/api/clash/accounts?target=' + encodeURIComponent(targetPath) + '&_=' + Date.now());
            const data = await res.json();
            if (data.status === 'success') {
                accounts = data.accounts || [];
                totalAccounts = data.total || 0;
                accountsStats = data.stats || {};
            } else {
                showToast('Info', data.message || 'Gagal membaca akun');
            }
        } catch (e) {
            console.error('Gagal memuat akun:', e);
        } finally {
            loadingAccounts = false;
        }
    }

    function onTargetChange() {
        if (selectedTarget) {
            loadAccounts(selectedTarget);
        }
    }

    // Simpan Akun dari Formulir Tambah
    async function saveCurrentFormAccount() {
        if (!currentAccount.server) {
            showToast('Peringatan', 'Harap isi Alamat Server / IP Host');
            return;
        }
        if (!selectedTarget) {
            showToast('Peringatan', 'Pilih file target penyimpanan terlebih dahulu');
            return;
        }

        isSaving = true;
        try {
            let payload = {
                target_file: selectedTarget
            };

            if (formViewTab === 'raw' && rawYamlLive.trim()) {
                payload.yaml_content = rawYamlLive;
            } else {
                payload.item = currentAccount;
            }

            const res = await fetch('/api/clash/accounts/add', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            const data = await res.json();
            if (data.status === 'success') {
                showToast('Sukses Ditambahkan', `Akun ${currentAccount.name || 'Proxy'} berhasil disimpan!`);
                // Reset form
                rawLinks = '';
                parsedAccounts = [];
                currentAccount = {
                    name: '', type: 'vless', server: '', port: 443, uuid: '',
                    password: '', cipher: 'auto', tls: true, skip_cert_verify: true,
                    servername: '', network: 'ws', ws_path: '/', ws_host: '', udp: true
                };
                activeTab = 'manage';
                await loadTargets(true);
            } else {
                showToast('Gagal Menambahkan', data.message);
            }
        } catch (e) {
            showToast('Error', 'Gagal menghubungi server.');
        } finally {
            isSaving = false;
        }
    }

    // Simpan Semua Akun Sekaligus (Bulk Import)
    async function saveAllParsedAccounts() {
        if (parsedAccounts.length === 0 || !selectedTarget) return;
        isSaving = true;
        try {
            const res = await fetch('/api/clash/accounts/add', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    target_file: selectedTarget,
                    items: parsedAccounts
                })
            });
            const data = await res.json();
            if (data.status === 'success') {
                showToast('Semua Akun Disimpan', data.message);
                rawLinks = '';
                parsedAccounts = [];
                activeTab = 'manage';
                await loadTargets(true);
            } else {
                showToast('Gagal', data.message);
            }
        } catch (e) {
            showToast('Error', 'Terjadi kesalahan sistem.');
        } finally {
            isSaving = false;
        }
    }

    // Modal Edit Akun
    function openEdit(acc) {
        editingAccount = JSON.parse(JSON.stringify(acc));
        editingRawYaml = buildYamlFromItem(editingAccount);
        editModalTab = 'form';
        showEditModal = true;
    }

    async function submitEdit() {
        if (!editingAccount || !selectedTarget) return;
        isSaving = true;
        try {
            let payload = {
                target_file: selectedTarget,
                index: editingAccount.index,
                original_name: editingAccount.name
            };

            if (editModalTab === 'raw') {
                payload.yaml_content = editingRawYaml;
            } else {
                payload.item = editingAccount;
            }

            const res = await fetch('/api/clash/accounts/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            const data = await res.json();
            if (data.status === 'success') {
                showToast('Berhasil Diperbarui', `Akun ${editingAccount.name} telah diperbarui.`);
                showEditModal = false;
                await loadAccounts(selectedTarget);
            } else {
                showToast('Gagal Memperbarui', data.message);
            }
        } catch (e) {
            showToast('Error', 'Terjadi kesalahan jaringan.');
        } finally {
            isSaving = false;
        }
    }

    // Modal Hapus Akun
    function openDelete(acc) {
        deletingAccount = acc;
        showDeleteModal = true;
    }

    async function confirmDelete() {
        if (!deletingAccount || !selectedTarget) return;
        isSaving = true;
        try {
            const res = await fetch('/api/clash/accounts/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    target_file: selectedTarget,
                    index: deletingAccount.index,
                    name: deletingAccount.name
                })
            });
            const data = await res.json();
            if (data.status === 'success') {
                showToast('Akun Dihapus', `Akun ${deletingAccount.name} berhasil dihapus.`);
                showDeleteModal = false;
                deletingAccount = null;
                await loadTargets(true);
            } else {
                showToast('Gagal Menghapus', data.message);
            }
        } catch (e) {
            showToast('Error', 'Gagal memproses penghapusan.');
        } finally {
            isSaving = false;
        }
    }

    // Modal Buat Provider Baru
    async function createNewProvider() {
        if (!newProviderName.trim()) return;
        isSaving = true;
        try {
            const res = await fetch('/api/clash/targets/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: newProviderName.trim() })
            });
            const data = await res.json();
            if (data.status === 'success') {
                showToast('Provider Dibuat', data.message);
                showCreateProviderModal = false;
                newProviderName = '';
                await loadTargets(false);
                if (data.path) {
                    selectedTarget = data.path;
                    await loadAccounts(selectedTarget);
                }
            } else {
                showToast('Gagal', data.message);
            }
        } catch (e) {
            showToast('Error', 'Gagal membuat file provider.');
        } finally {
            isSaving = false;
        }
    }

    // Clipboard Helpers
    function copyText(txt, title = 'Tersalin!') {
        if (!txt) return;
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(txt).then(() => showToast(title, 'Disalin ke clipboard'))
                .catch(() => fallbackCopy(txt, title));
        } else {
            fallbackCopy(txt, title);
        }
    }

    function fallbackCopy(txt, title) {
        const ta = document.createElement('textarea');
        ta.value = txt;
        document.body.appendChild(ta);
        ta.select();
        try {
            document.execCommand('copy');
            showToast(title, 'Disalin ke clipboard');
        } catch (e) {}
        document.body.removeChild(ta);
    }

    function copyAccountYaml(acc) {
        copyText(buildYamlFromItem(acc), 'YAML Akun Tersalin');
    }

    function copyAccountLink(acc) {
        const link = generateVpnLink(acc);
        if (link) copyText(link, 'Link Akun Tersalin');
        else showToast('Info', 'Link akun tidak didukung untuk tipe ini.');
    }

    async function pasteFromClipboard() {
        try {
            const text = await navigator.clipboard.readText();
            if (text) {
                rawLinks = text;
                showToast('Link Ditempel', 'Link berhasil dibaca dari clipboard.');
            }
        } catch(e) {
            showToast('Peringatan', 'Silakan tempel langsung ke kotak teks.');
        }
    }

    onMount(() => {
        loadTargets();
    });
</script>

<div class="wrap">
    <!-- COMPACT HEADER & TARGET TOOLBAR -->
    <div class="compact-header">
        <div class="header-main-row">
            <div>
                <div class="eyebrow">
                    <i class="fa-solid fa-bolt" style="color:var(--primary);"></i> Manajemen Proksi Clash
                </div>
                <h1>Kelola & Tambah Akun</h1>
            </div>
            <div class="header-quick-actions">
                <button class="btn btn-sm btn-outline" on:click={() => showTemplateModal = true} title="Pasang/Reset Template Pisah Trafik BFR">
                    <i class="fa-solid fa-wand-magic-sparkles" style="color:var(--accent);"></i>
                    <span>Template BFR</span>
                </button>
                <button class="btn btn-sm btn-outline" on:click={() => showCreateProviderModal = true} title="Buat File Provider YAML Baru">
                    <i class="fa-solid fa-plus"></i>
                    <span>Provider</span>
                </button>
                <button class="btn btn-sm btn-icon" on:click={() => loadTargets(true)} title="Segarkan Data">
                    <i class="fa-solid fa-rotate-right" class:fa-spin={loadingAccounts || loadingTargets}></i>
                </button>
            </div>
        </div>

        <!-- TARGET SELECTOR STRIP -->
        <div class="target-strip">
            <div class="target-strip-left">
                <label for="targetSelect" class="ts-label"><i class="fa-solid fa-folder-open"></i> File Target:</label>
                <div class="select-wrap-compact">
                    <select id="targetSelect" bind:value={selectedTarget} on:change={onTargetChange}>
                        {#if loadingTargets}
                            <option value="">Memindai file...</option>
                        {:else}
                            {#each targetFiles as target}
                                <option value={target.path}>
                                    {target.label} ({target.count} Akun)
                                </option>
                            {/each}
                        {/if}
                    </select>
                </div>
            </div>
            {#if configMode === 'split'}
                {#if (selectedTarget || '').includes('my_proxies')}
                    <span class="target-badge-pill pill-success" title="Target ini terhubung ke grup Game & Streaming">
                        <i class="fa-solid fa-circle-check"></i> Pisah Trafik
                    </span>
                {:else}
                    <span class="target-badge-pill pill-warning" title="Pilih 'Provider: my_proxies.yaml'">
                        <i class="fa-solid fa-triangle-exclamation"></i> Saran: my_proxies.yaml
                    </span>
                {/if}
            {/if}
        </div>
    </div>

    <!-- BANNER REKOMENDASI TEMPLATE JIKA CONFIG MASIH KOSONG -->
    {#if needsTemplate}
        <div class="template-alert-banner">
            <div class="tab-icon-wrap">
                <i class="fa-solid fa-bolt-lightning"></i>
            </div>
            <div class="tab-text-wrap">
                <div class="tab-heading">CONFIG BFR MASIH KOSONG / BELUM PISAH TRAFIK</div>
                <div class="tab-description">
                    File <code>config.yaml</code> bawaan BFR belum memiliki aturan routing pisah trafik & DNS Android. Pasang template siap pakai sekarang agar internet langsung berjalan otomatis!
                </div>
            </div>
            <div class="tab-action-wrap">
                <button class="btn btn-template-yellow" on:click={() => showTemplateModal = true}>
                    <i class="fa-solid fa-wand-magic-sparkles"></i> Pilih & Pasang Template BFR
                </button>
            </div>
        </div>
    {/if}

    <!-- SEGMENTED TABS -->
    <div class="segmented-tabs">
        <button
            class="seg-tab {activeTab === 'add' ? 'active' : ''}"
            on:click={() => activeTab = 'add'}
        >
            <i class="fa-solid fa-circle-plus"></i>
            <span>Tambah Akun</span>
        </button>

        <button
            class="seg-tab {activeTab === 'manage' ? 'active' : ''}"
            on:click={() => activeTab = 'manage'}
        >
            <i class="fa-solid fa-list-check"></i>
            <span>Kelola Akun Tersimpan</span>
            <span class="seg-badge">{totalAccounts}</span>
        </button>
    </div>

    <!-- TAB: TAMBAH AKUN BARU (STREAMLINED & USER-FRIENDLY) -->
    {#if activeTab === 'add'}
        <div class="tab-content">
            <!-- 1. KOTAK TEMPEL LINK CEPAT -->
            <div class="quick-paste-card">
                <div class="qpc-header">
                    <div class="qpc-title">
                        <i class="fa-solid fa-link" style="color:var(--primary);"></i>
                        <div>
                            <span class="qpc-heading">Tempel Link VPN</span>
                            <span class="qpc-sub">VLESS, VMess, Trojan, Shadowsocks</span>
                        </div>
                    </div>
                    <button class="btn-paste-clip" on:click={pasteFromClipboard} title="Tempel otomatis dari Clipboard">
                        <i class="fa-solid fa-clipboard"></i> <span>Tempel Clipboard</span>
                    </button>
                </div>

                <div class="textarea-wrap">
                    <textarea
                        class="paste-textarea"
                        bind:value={rawLinks}
                        placeholder="Tempel tautan vless://, vmess://, atau trojan:// di sini...&#10;Data akun otomatis terisi dan siap disimpan langsung!"
                        rows={parsedAccounts.length > 0 ? 2 : 3}
                    ></textarea>
                    {#if rawLinks}
                        <button class="btn-clear-paste" on:click={() => { rawLinks = ''; parsedAccounts = []; }} title="Hapus teks">
                            <i class="fa-solid fa-xmark"></i>
                        </button>
                    {/if}
                </div>

                {#if parsedAccounts.length > 1}
                    <!-- Selector Chip jika banyak link sekaligus -->
                    <div class="chip-bar">
                        <span class="chip-label"><i class="fa-solid fa-layer-group"></i> {parsedAccounts.length} Akun Terdeteksi:</span>
                        <div class="chips-scroll">
                            {#each parsedAccounts as item, idx}
                                <button
                                    class="chip-btn {activeAccountIndex === idx ? 'active' : ''}"
                                    on:click={() => selectAccountChip(idx)}
                                >
                                    <span class="proto-tag proto-{item.type}">{item.type.toUpperCase()}</span>
                                    <span class="chip-name">{item.name || item.server}</span>
                                </button>
                            {/each}
                        </div>
                        <button class="btn btn-primary btn-sm" on:click={saveAllParsedAccounts} disabled={isSaving}>
                            <i class="fa-solid fa-floppy-disk"></i> Simpan Semua ({parsedAccounts.length})
                        </button>
                    </div>
                {/if}
            </div>

            <!-- 2. HASIL DETEKSI CEPAT & TOMBOL SIMPAN 1-KLIK -->
            {#if currentAccount && currentAccount.server}
                <div class="quick-summary-card">
                    <div class="qsc-header">
                        <div class="qsc-status">
                            <span class="dot-live"></span>
                            <span class="qsc-ready-label">Akun Terdeteksi & Siap Simpan</span>
                        </div>
                        {#if currentAccount.type}
                            <span class="proto-tag proto-{currentAccount.type}">{currentAccount.type.toUpperCase()}</span>
                        {/if}
                    </div>

                    <div class="qsc-body">
                        <div class="qsc-name-box">
                            <label for="quickNameInput">Nama Akun Tampilan:</label>
                            <input
                                id="quickNameInput"
                                type="text"
                                bind:value={currentAccount.name}
                                on:input={onFormFieldChange}
                                placeholder="Contoh: Biznet SG VIP"
                                class="qsc-input-name"
                            />
                        </div>

                        <div class="qsc-chips">
                            <div class="qsc-chip" title="Server Host & Port">
                                <i class="fa-solid fa-server"></i>
                                <span>{currentAccount.server}:{currentAccount.port}</span>
                            </div>
                            <div class="qsc-chip" title="Tipe Jaringan">
                                <i class="fa-solid fa-network-wired"></i>
                                <span>{(currentAccount.network || 'ws').toUpperCase()}</span>
                            </div>
                            {#if currentAccount.servername}
                                <div class="qsc-chip chip-highlight" title="Bug SNI">
                                    <i class="fa-solid fa-bug"></i>
                                    <span>{currentAccount.servername}</span>
                                </div>
                            {/if}
                            <div class="qsc-chip" title="Keamanan TLS">
                                <i class="fa-solid {currentAccount.tls ? 'fa-lock' : 'fa-lock-open'}"></i>
                                <span>{currentAccount.tls ? 'TLS Aktif' : 'Non-TLS'}</span>
                            </div>
                        </div>
                    </div>

                    <div class="qsc-actions">
                        <button
                            class="btn btn-save-primary"
                            on:click={saveCurrentFormAccount}
                            disabled={isSaving || !currentAccount.server}
                        >
                            <i class="fa-solid fa-floppy-disk"></i>
                            <span>{isSaving ? 'Menyimpan...' : `Simpan Akun ke ${selectedTarget.split('/').pop()}`}</span>
                        </button>

                        <button
                            class="btn-toggle-adv"
                            on:click={() => showAdvancedSettings = !showAdvancedSettings}
                            type="button"
                        >
                            <i class="fa-solid fa-sliders"></i>
                            <span>{showAdvancedSettings ? 'Tutup Pengaturan Lanjutan' : 'Edit Detail / Pengaturan Lanjutan'}</span>
                            <i class="fa-solid {showAdvancedSettings ? 'fa-chevron-up' : 'fa-chevron-down'}"></i>
                        </button>
                    </div>
                </div>
            {:else if !rawLinks}
                <!-- HINT BERSIH JIKA BELUM MENEMPEL LINK -->
                <div class="clean-hint-card">
                    <div class="ch-text">
                        <i class="fa-regular fa-lightbulb" style="color:var(--primary);"></i>
                        <div>
                            <strong>Alur Cepat:</strong> Cukup salin link proxy dari penyedia (misal <code>vless://...</code>), lalu tekan tombol <strong>Tempel Clipboard</strong> di atas untuk menyimpan akun secara instan.
                        </div>
                    </div>
                    <button class="btn-text-manual" on:click={() => {
                        currentAccount = {
                            name: 'Akun Proxy Baru', type: 'vless', server: '104.16.1.1', port: 443, uuid: '',
                            password: '', cipher: 'auto', tls: true, skip_cert_verify: true,
                            servername: '', network: 'ws', ws_path: '/', ws_host: '', udp: true
                        };
                        showAdvancedSettings = true;
                    }}>
                        <i class="fa-solid fa-pen-to-square"></i> Buat Manual Tanpa Link
                    </button>
                </div>
            {/if}

            <!-- 3. ACCORDION DETAIL & PENGATURAN LANJUTAN (HANYA DIBUKA JIKA USER KLIK) -->
            {#if showAdvancedSettings}
                <div class="advanced-editor-card">
                    <div class="aec-header">
                        <div class="aec-title">
                            <i class="fa-solid fa-sliders" style="color:var(--primary);"></i>
                            <span>Detail Parameter & Pengaturan Lanjutan</span>
                            {#if currentAccount.type}
                                <span class="proto-tag proto-{currentAccount.type}">{currentAccount.type.toUpperCase()}</span>
                            {/if}
                        </div>
                        <button class="aec-close" on:click={() => showAdvancedSettings = false} title="Tutup Detail Lanjutan">
                            <i class="fa-solid fa-xmark"></i>
                        </button>
                    </div>

                    <!-- Tabs: Formulir Lengkap vs Kode YAML -->
                    <div class="em-tabs">
                        <button
                            class="em-tab {formViewTab === 'form' ? 'active' : ''}"
                            on:click={() => formViewTab = 'form'}
                        >
                            <i class="fa-solid fa-sliders"></i>
                            <span>Formulir Lengkap</span>
                        </button>
                        <button
                            class="em-tab {formViewTab === 'raw' ? 'active' : ''}"
                            on:click={() => formViewTab = 'raw'}
                        >
                            <i class="fa-solid fa-code"></i>
                            <span>Kode YAML</span>
                        </button>
                    </div>

                    <!-- Body Form -->
                    <div class="em-body">
                        {#if formViewTab === 'form'}
                            <div class="form-grid">
                                <!-- NAMA AKUN TAMPILAN -->
                                <div class="form-group full-width">
                                    <label for="formAccName">NAMA AKUN TAMPILAN</label>
                                    <input
                                        id="formAccName"
                                        type="text"
                                        bind:value={currentAccount.name}
                                        on:input={onFormFieldChange}
                                        placeholder="Contoh: Biznet Singapura VIP"
                                    />
                                </div>

                                <!-- PROTOKOL & PORT -->
                                <div class="form-group">
                                    <label for="formAccProto">PROTOKOL</label>
                                    <select id="formAccProto" bind:value={currentAccount.type} on:change={onFormFieldChange}>
                                        <option value="vless">VLESS</option>
                                        <option value="vmess">VMess</option>
                                        <option value="trojan">Trojan</option>
                                        <option value="ss">Shadowsocks</option>
                                    </select>
                                </div>

                                <div class="form-group">
                                    <label for="formAccPort">PORT</label>
                                    <input
                                        id="formAccPort"
                                        type="number"
                                        bind:value={currentAccount.port}
                                        on:input={onFormFieldChange}
                                        placeholder="443"
                                    />
                                </div>

                                <!-- SERVER HOST / IP -->
                                <div class="form-group full-width">
                                    <label for="formAccServer">SERVER HOST / IP</label>
                                    <input
                                        id="formAccServer"
                                        type="text"
                                        bind:value={currentAccount.server}
                                        on:input={onFormFieldChange}
                                        placeholder="domain.com atau 104.16.1.1"
                                    />
                                </div>

                                <!-- UUID ATAU PASSWORD -->
                                {#if currentAccount.type === 'vless' || currentAccount.type === 'vmess'}
                                    <div class="form-group full-width">
                                        <label for="formAccUuid">UUID</label>
                                        <input
                                            id="formAccUuid"
                                            type="text"
                                            bind:value={currentAccount.uuid}
                                            on:input={onFormFieldChange}
                                            placeholder="b3b4a229-873b-4861-9c17-48b594b29bb8"
                                        />
                                    </div>
                                {:else}
                                    <div class="form-group full-width">
                                        <label for="formAccPass">PASSWORD / KATA SANDI</label>
                                        <input
                                            id="formAccPass"
                                            type="text"
                                            bind:value={currentAccount.password}
                                            on:input={onFormFieldChange}
                                            placeholder="Kata sandi proxy"
                                        />
                                    </div>
                                {/if}

                                <!-- TIPE JARINGAN & BUG SNI (SERVERNAME) -->
                                <div class="form-group">
                                    <label for="formAccNet">TIPE JARINGAN</label>
                                    <select id="formAccNet" bind:value={currentAccount.network} on:change={onFormFieldChange}>
                                        <option value="ws">WebSocket (WS)</option>
                                        <option value="grpc">gRPC</option>
                                        <option value="tcp">TCP</option>
                                    </select>
                                </div>

                                <div class="form-group">
                                    <label for="formAccSni">BUG SNI (SERVERNAME)</label>
                                    <input
                                        id="formAccSni"
                                        type="text"
                                        bind:value={currentAccount.servername}
                                        on:input={onFormFieldChange}
                                        placeholder="zoom.us atau bug host"
                                    />
                                </div>

                                <!-- WEBSOCKET PATH & HOST (JIKA WEBSOCKET) -->
                                {#if currentAccount.network === 'ws'}
                                    <div class="form-group">
                                        <label for="formAccWsPath">WEBSOCKET PATH</label>
                                        <input
                                            id="formAccWsPath"
                                            type="text"
                                            bind:value={currentAccount.ws_path}
                                            on:input={onFormFieldChange}
                                            placeholder="/"
                                        />
                                    </div>

                                    <div class="form-group">
                                        <label for="formAccWsHost">WEBSOCKET HOST HEADER</label>
                                        <input
                                            id="formAccWsHost"
                                            type="text"
                                            bind:value={currentAccount.ws_host}
                                            on:input={onFormFieldChange}
                                            placeholder="domain.com (Opsional)"
                                        />
                                    </div>
                                {/if}

                                <!-- REALITY OPTIONS (JIKA ADA) -->
                                {#if currentAccount.reality_public_key}
                                    <div class="form-group">
                                        <label for="formAccPbk">REALITY PUBLIC KEY</label>
                                        <input id="formAccPbk" type="text" bind:value={currentAccount.reality_public_key} on:input={onFormFieldChange} />
                                    </div>
                                    <div class="form-group">
                                        <label for="formAccSid">SHORT ID</label>
                                        <input id="formAccSid" type="text" bind:value={currentAccount.reality_short_id} on:input={onFormFieldChange} />
                                    </div>
                                {/if}

                                <!-- CHECKBOX TOGGLE LIST -->
                                <div class="form-group full-width toggle-list">
                                    <label class="toggle-box">
                                        <input type="checkbox" bind:checked={currentAccount.tls} on:change={onFormFieldChange} />
                                        <span>TLS AKTIF</span>
                                    </label>

                                    <label class="toggle-box">
                                        <input type="checkbox" bind:checked={currentAccount.skip_cert_verify} on:change={onFormFieldChange} />
                                        <span>SKIP CERT VERIFY</span>
                                    </label>

                                    <label class="toggle-box">
                                        <input type="checkbox" bind:checked={currentAccount.udp} on:change={onFormFieldChange} />
                                        <span>UDP</span>
                                    </label>
                                </div>
                            </div>
                        {:else}
                            <!-- TAB RAW YAML VIEW -->
                            <div class="raw-yaml-wrap">
                                <div class="raw-hint">
                                    <i class="fa-solid fa-code"></i> Output konfigurasi Clash YAML otomatis diperbarui saat Anda mengubah formulir.
                                </div>
                                <textarea
                                    class="raw-textarea mono"
                                    bind:value={rawYamlLive}
                                    rows="12"
                                    placeholder="- name: My-Proxy..."
                                ></textarea>
                            </div>
                        {/if}
                    </div>

                    <!-- Footer Card Form -->
                    <div class="em-footer">
                        <button
                            class="btn btn-dark"
                            on:click={() => {
                                rawLinks = '';
                                parsedAccounts = [];
                                showAdvancedSettings = false;
                                currentAccount = {
                                    name: '', type: 'vless', server: '', port: 443, uuid: '',
                                    password: '', cipher: 'auto', tls: true, skip_cert_verify: true,
                                    servername: '', network: 'ws', ws_path: '/', ws_host: '', udp: true
                                };
                            }}
                        >
                            Reset Form
                        </button>

                        <button
                            class="btn btn-save-primary"
                            on:click={saveCurrentFormAccount}
                            disabled={isSaving || !currentAccount.server}
                        >
                            <i class="fa-solid fa-floppy-disk"></i>
                            {isSaving ? 'Menyimpan...' : `Simpan Akun`}
                        </button>
                    </div>
                </div>
            {/if}
        </div>
    {/if}

    <!-- TAB: KELOLA AKUN TERSIMPAN -->
    {#if activeTab === 'manage'}
        <div class="tab-content">
            <!-- FILTER & SEARCH BAR -->
            <div class="filter-bar">
                <div class="search-input-wrap">
                    <i class="fa-solid fa-magnifying-glass"></i>
                    <input
                        type="text"
                        bind:value={searchQuery}
                        placeholder="Cari nama akun, server, atau bug SNI..."
                    />
                    {#if searchQuery}
                        <button class="clear-search" on:click={() => searchQuery = ''}>
                            <i class="fa-solid fa-xmark"></i>
                        </button>
                    {/if}
                </div>

                <div class="protocol-pills">
                    <button
                        class="pill {filterType === 'all' ? 'active' : ''}"
                        on:click={() => filterType = 'all'}
                    >
                        Semua ({totalAccounts})
                    </button>
                    {#each Object.entries(accountsStats) as [type, count]}
                        <button
                            class="pill {filterType === type ? 'active' : ''}"
                            on:click={() => filterType = type}
                        >
                            {type.toUpperCase()} ({count})
                        </button>
                    {/each}
                </div>
            </div>

            <!-- LIST KARTU AKUN -->
            {#if loadingAccounts}
                <div class="loading-state">
                    <i class="fa-solid fa-spinner fa-spin"></i>
                    <p>Memuat daftar akun proxy tersimpan...</p>
                </div>
            {:else if filteredAccounts.length === 0}
                <div class="empty-state">
                    <div class="empty-icon"><i class="fa-solid fa-layer-group"></i></div>
                    <h3>Tidak ada akun ditemukan</h3>
                    <p>
                        {#if searchQuery || filterType !== 'all'}
                            Tidak ada akun yang cocok dengan filter pencarian ini.
                        {:else}
                            File ini masih kosong. Silakan tambahkan akun pertama Anda sekarang!
                        {/if}
                    </p>
                    <button class="btn btn-save-yellow" on:click={() => activeTab = 'add'}>
                        <i class="fa-solid fa-plus"></i> Tambah Akun Sekarang
                    </button>
                </div>
            {:else}
                <div class="account-grid">
                    {#each filteredAccounts as acc (acc.index + '_' + acc.name)}
                        <div class="account-card type-{acc.type || 'vless'}">
                            <!-- Card Header -->
                            <div class="acc-header">
                                <div class="acc-title-wrap">
                                    <span class="proto-badge proto-{acc.type}">{(acc.type || 'VLESS').toUpperCase()}</span>
                                    <h3 class="acc-name" title={acc.name}>{acc.name}</h3>
                                </div>
                                <div class="acc-actions">
                                    <button class="action-btn edit" on:click={() => openEdit(acc)} title="Edit Akun">
                                        <i class="fa-solid fa-pen-to-square"></i>
                                    </button>
                                    <button class="action-btn share" on:click={() => copyAccountLink(acc)} title="Salin Link VPN">
                                        <i class="fa-solid fa-link"></i>
                                    </button>
                                    <button class="action-btn copy" on:click={() => copyAccountYaml(acc)} title="Salin Potongan YAML">
                                        <i class="fa-solid fa-code"></i>
                                    </button>
                                    <button class="action-btn delete" on:click={() => openDelete(acc)} title="Hapus Akun">
                                        <i class="fa-solid fa-trash-can"></i>
                                    </button>
                                </div>
                            </div>

                            <!-- Card Body Info Chips Rapi & Ramping -->
                            <div class="acc-body-compact">
                                <div class="acc-chips-row">
                                    <span class="acc-chip" title="Server & Port">
                                        <i class="fa-solid fa-server"></i> {acc.server}:{acc.port}
                                    </span>
                                    <span class="acc-chip" title="Jaringan">
                                        <i class="fa-solid fa-network-wired"></i> {(acc.network || 'ws').toUpperCase()}
                                    </span>
                                    {#if acc.servername || acc.ws_host}
                                        <span class="acc-chip chip-sni" title="Bug SNI">
                                            <i class="fa-solid fa-bug"></i> {acc.servername || acc.ws_host}
                                        </span>
                                    {/if}
                                    <span class="acc-chip {acc.tls ? 'chip-tls' : ''}">
                                        <i class="fa-solid {acc.tls ? 'fa-lock' : 'fa-lock-open'}"></i> {acc.tls ? 'TLS' : 'Non-TLS'}
                                    </span>
                                    {#if acc.udp}
                                        <span class="acc-chip chip-mini">UDP</span>
                                    {/if}
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    {/if}
</div>

<!-- MODAL EDIT AKUN (DENGAN TAMPILAN PERSIS SEPERTI GAMBAR USER) -->
{#if showEditModal && editingAccount}
    <div class="modal-overlay active">
        <div class="modal-card">
            <div class="modal-header">
                <div class="m-title">
                    <i class="fa-solid fa-pen-to-square" style="color:var(--yellow);"></i>
                    <span>Edit Akun Proxy</span>
                </div>
                <button class="modal-close" on:click={() => showEditModal = false}>
                    <i class="fa-solid fa-xmark"></i>
                </button>
            </div>

            <!-- Tabs Modal: Formulir Mudah vs Kode YAML -->
            <div class="em-tabs">
                <button
                    class="em-tab {editModalTab === 'form' ? 'active' : ''}"
                    on:click={() => editModalTab = 'form'}
                >
                    <i class="fa-solid fa-sliders"></i>
                    <span>Formulir Mudah</span>
                </button>
                <button
                    class="em-tab {editModalTab === 'raw' ? 'active' : ''}"
                    on:click={() => {
                        editingRawYaml = buildYamlFromItem(editingAccount);
                        editModalTab = 'raw';
                    }}
                >
                    <i class="fa-solid fa-code"></i>
                    <span>Kode YAML</span>
                </button>
            </div>

            <div class="modal-body">
                {#if editModalTab === 'form'}
                    <div class="form-grid">
                        <div class="form-group full-width">
                            <label for="editName">NAMA AKUN TAMPILAN</label>
                            <input id="editName" type="text" bind:value={editingAccount.name} />
                        </div>

                        <div class="form-group">
                            <label for="editType">PROTOKOL</label>
                            <select id="editType" bind:value={editingAccount.type}>
                                <option value="vless">VLESS</option>
                                <option value="vmess">VMess</option>
                                <option value="trojan">Trojan</option>
                                <option value="ss">Shadowsocks</option>
                            </select>
                        </div>

                        <div class="form-group">
                            <label for="editPort">PORT</label>
                            <input id="editPort" type="number" bind:value={editingAccount.port} />
                        </div>

                        <div class="form-group full-width">
                            <label for="editServer">SERVER HOST / IP</label>
                            <input id="editServer" type="text" bind:value={editingAccount.server} />
                        </div>

                        {#if editingAccount.type === 'vless' || editingAccount.type === 'vmess'}
                            <div class="form-group full-width">
                                <label for="editUUID">UUID</label>
                                <input id="editUUID" type="text" bind:value={editingAccount.uuid} />
                            </div>
                        {:else}
                            <div class="form-group full-width">
                                <label for="editPassword">PASSWORD / KATA SANDI</label>
                                <input id="editPassword" type="text" bind:value={editingAccount.password} />
                            </div>
                        {/if}

                        <div class="form-group">
                            <label for="editNet">TIPE JARINGAN</label>
                            <select id="editNet" bind:value={editingAccount.network}>
                                <option value="ws">WebSocket (WS)</option>
                                <option value="grpc">gRPC</option>
                                <option value="tcp">TCP</option>
                            </select>
                        </div>

                        <div class="form-group">
                            <label for="editSni">BUG SNI (SERVERNAME)</label>
                            <input id="editSni" type="text" bind:value={editingAccount.servername} placeholder="zoom.us atau bug host" />
                        </div>

                        {#if editingAccount.network === 'ws'}
                            <div class="form-group">
                                <label for="editWsPath">WEBSOCKET PATH</label>
                                <input id="editWsPath" type="text" bind:value={editingAccount.ws_path} />
                            </div>
                            <div class="form-group">
                                <label for="editWsHost">WEBSOCKET HOST HEADER</label>
                                <input id="editWsHost" type="text" bind:value={editingAccount.ws_host} />
                            </div>
                        {/if}

                        <div class="form-group full-width toggle-list">
                            <label class="toggle-box">
                                <input type="checkbox" bind:checked={editingAccount.tls} />
                                <span>TLS AKTIF</span>
                            </label>
                            <label class="toggle-box">
                                <input type="checkbox" bind:checked={editingAccount.skip_cert_verify} />
                                <span>SKIP CERT VERIFY</span>
                            </label>
                            <label class="toggle-box">
                                <input type="checkbox" bind:checked={editingAccount.udp} />
                                <span>UDP</span>
                            </label>
                        </div>
                    </div>
                {:else}
                    <textarea class="raw-textarea mono" rows="12" bind:value={editingRawYaml}></textarea>
                {/if}
            </div>

            <div class="modal-footer">
                <button class="btn btn-dark" on:click={() => showEditModal = false}>Batal</button>
                <button class="btn btn-save-yellow" on:click={submitEdit} disabled={isSaving}>
                    <i class="fa-solid fa-floppy-disk"></i>
                    {isSaving ? 'Menyimpan...' : 'Simpan Perubahan'}
                </button>
            </div>
        </div>
    </div>
{/if}

<!-- MODAL HAPUS AKUN -->
{#if showDeleteModal && deletingAccount}
    <div class="modal-overlay active">
        <div class="modal-card modal-sm">
            <div class="modal-header modal-header-danger">
                <div class="m-title">
                    <i class="fa-solid fa-triangle-exclamation"></i>
                    <span>Hapus Akun Proxy</span>
                </div>
                <button class="modal-close" on:click={() => showDeleteModal = false}>
                    <i class="fa-solid fa-xmark"></i>
                </button>
            </div>

            <div class="modal-body">
                <p>Apakah Anda yakin ingin menghapus akun berikut?</p>
                <div class="del-preview">
                    <strong>{deletingAccount.name}</strong>
                    <div class="small mono">{deletingAccount.server}:{deletingAccount.port} ({deletingAccount.type})</div>
                </div>
                <p class="warn-text"><i class="fa-solid fa-info-circle"></i> Akun akan dihapus dari berkas YAML target.</p>
            </div>

            <div class="modal-footer">
                <button class="btn btn-dark" on:click={() => showDeleteModal = false}>Batal</button>
                <button class="btn btn-danger" on:click={confirmDelete} disabled={isSaving}>
                    <i class="fa-solid fa-trash-can"></i>
                    {isSaving ? 'Menghapus...' : 'Ya, Hapus Akun'}
                </button>
            </div>
        </div>
    </div>
{/if}

<!-- MODAL BUAT PROVIDER BARU -->
{#if showCreateProviderModal}
    <div class="modal-overlay active">
        <div class="modal-card modal-sm">
            <div class="modal-header">
                <div class="m-title">
                    <i class="fa-solid fa-file-circle-plus" style="color:var(--lime);"></i>
                    <span>Buat Provider Baru</span>
                </div>
                <button class="modal-close" on:click={() => showCreateProviderModal = false}>
                    <i class="fa-solid fa-xmark"></i>
                </button>
            </div>

            <div class="modal-body">
                <div class="form-group full-width">
                    <label for="newProvName">Nama File Provider (YAML)</label>
                    <input
                        id="newProvName"
                        type="text"
                        bind:value={newProviderName}
                        placeholder="Contoh: vip_singapore atau akun_game"
                        on:keydown={(e) => e.key === 'Enter' && createNewProvider()}
                    />
                    <div class="form-hint">Otomatis disimpan di <code>proxy_provider/</code></div>
                </div>
            </div>

            <div class="modal-footer">
                <button class="btn btn-dark" on:click={() => showCreateProviderModal = false}>Batal</button>
                <button class="btn btn-save-yellow" on:click={createNewProvider} disabled={isSaving || !newProviderName.trim()}>
                    <i class="fa-solid fa-plus"></i>
                    {isSaving ? 'Membuat...' : 'Buat Provider'}
                </button>
            </div>
        </div>
    </div>
{/if}

<!-- MODAL KONFIRMASI PASANG TEMPLATE BFR -->
{#if showTemplateModal}
    <div class="modal-overlay" on:click={() => showTemplateModal = false}>
        <div class="modal-card" on:click|stopPropagation>
            <div class="modal-header">
                <div class="m-title">
                    <i class="fa-solid fa-wand-magic-sparkles" style="color:var(--primary);"></i>
                    <span>Pilih Template Konfigurasi BFR</span>
                </div>
                <button class="modal-close" on:click={() => showTemplateModal = false}>✕</button>
            </div>
            <div class="modal-body">
                <p style="font-size:13px; line-height:1.5; margin:0; font-weight:700;">
                    Pilih model perutean trafik yang sesuai dengan kebutuhan Anda:
                </p>

                <!-- PILIHAN MODE -->
                <div class="template-mode-cards">
                    <!-- MODE 1: PISAH TRAFIK -->
                    <label class="mode-card {selectedTemplateMode === 'split' ? 'active' : ''}">
                        <div class="mode-radio">
                            <input type="radio" name="tmplMode" value="split" bind:group={selectedTemplateMode}>
                        </div>
                        <div class="mode-info">
                            <div class="mode-title-row">
                                <span class="mode-title">🔀 Mode Pisah Trafik (Multi-Routing)</span>
                                <span class="mode-badge-rec">REKOMENDASI</span>
                            </div>
                            <div class="mode-desc">
                                Memisahkan koneksi otomatis ke grup khusus: <strong>Game</strong> (ML, PUBG, FF, Steam), <strong>Sosmed & Streaming</strong> (YouTube, TikTok, Netflix, IG), <strong>Indo Direct</strong> (M-Banking, Marketplace), dan <strong>Blokir Iklan</strong>.
                            </div>
                        </div>
                    </label>

                    <!-- MODE 2: STANDAR / TANPA PISAH TRAFIK -->
                    <label class="mode-card {selectedTemplateMode === 'standard' ? 'active' : ''}">
                        <div class="mode-radio">
                            <input type="radio" name="tmplMode" value="standard" bind:group={selectedTemplateMode}>
                        </div>
                        <div class="mode-info">
                            <div class="mode-title-row">
                                <span class="mode-title">🚀 Mode Standar (All-in-One Proxy)</span>
                            </div>
                            <div class="mode-desc">
                                Seluruh trafik internet dialirkan langsung ke satu grup proxy utama (dengan auto-ping terendah & bypass LAN). Lebih sederhana dan ringan tanpa filter terpisah.
                            </div>
                        </div>
                    </label>
                </div>

                <!-- PETUNJUK TARGET FILE PENYIMPANAN -->
                <div class="target-guidance-box">
                    <div class="tgb-title"><i class="fa-solid fa-lightbulb" style="color:var(--yellow);"></i> Panduan Target Akun:</div>
                    <div class="tgb-desc">
                        {#if selectedTemplateMode === 'split'}
                            Saat menambah akun, pastikan memilih <strong>Provider: my_proxies.yaml</strong> agar akun Anda otomatis terhubung ke seluruh grup pisah trafik (Game, Streaming, dll).
                        {:else}
                            Saat menambah akun, pilih <strong>Provider: my_proxies.yaml</strong> agar tersimpan rapi tanpa merusak setting sistem <code>config.yaml</code>.
                        {/if}
                    </div>
                </div>

                <div class="warn-box" style="background:rgba(255, 225, 74, 0.12); border:1.5px solid var(--yellow); border-radius:8px; padding:8px 12px; font-size:11.5px; color:var(--text);">
                    <i class="fa-solid fa-shield-halved" style="color:var(--yellow); margin-right:4px;"></i>
                    File <code>config.yaml</code> lama akan otomatis dibackup ke <code>config.yaml.bak</code>.
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-dark" on:click={() => showTemplateModal = false}>Batal</button>
                <button class="btn btn-save-yellow" on:click={() => applyBFRTemplate(selectedTemplateMode)} disabled={isApplyingTemplate}>
                    <i class="fa-solid fa-check"></i>
                    {isApplyingTemplate ? 'Memasang...' : 'Pasang Template ' + (selectedTemplateMode === 'split' ? 'Pisah Trafik' : 'Standar')}
                </button>
            </div>
        </div>
    </div>
{/if}

<!-- MODAL KONFIRMASI RESTART BFR SETELAH PASANG TEMPLATE -->
{#if showRestartBFRConfirm}
    <div class="modal-overlay" on:click={() => showRestartBFRConfirm = false}>
        <div class="modal-card modal-sm" on:click|stopPropagation>
            <div class="modal-header">
                <div class="m-title">
                    <i class="fa-solid fa-circle-check" style="color:var(--lime);"></i>
                    <span>Template Berhasil Dipasang!</span>
                </div>
                <button class="modal-close" on:click={() => showRestartBFRConfirm = false}>✕</button>
            </div>
            <div class="modal-body">
                <p style="font-size:13px; line-height:1.5; margin:0;">
                    File <code>config.yaml</code> dan provider <code>my_proxies.yaml</code> telah aktif.
                </p>
                <p style="font-size:12px; color:var(--text-muted); margin:4px 0 0 0;">
                    Agar perubahan konfigurasi langsung dimuat oleh core Clash/Mihomo, restart layanan BFR sekarang.
                </p>
            </div>
            <div class="modal-footer">
                <button class="btn btn-dark" on:click={() => showRestartBFRConfirm = false}>Nanti Saja</button>
                <button class="btn btn-save-yellow" on:click={restartBFRService} disabled={isRestartingBFR}>
                    <i class="fa-solid fa-rotate-right" class:fa-spin={isRestartingBFR}></i>
                    {isRestartingBFR ? 'Merestart...' : 'Restart Layanan BFR'}
                </button>
            </div>
        </div>
    </div>
{/if}

<!-- TOAST NOTIFICATION -->
{#if toast.show}
    <div class="brutal-toast">
        <div class="toast-title"><i class="fa-solid fa-bell"></i> {toast.title}</div>
        <div class="toast-desc">{toast.desc}</div>
    </div>
{/if}

<style>
    * { box-sizing: border-box; }

    .wrap {
        max-width: 900px;
        margin: 0 auto;
        padding: 16px;
        padding-bottom: 90px;
    }

    /* COMPACT HEADER & TARGET STRIP */
    .compact-header {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 16px;
    }

    .header-main-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: 10px;
    }

    .header-quick-actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .target-strip {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        border-radius: 10px;
        padding: 8px 12px;
        box-shadow: var(--brutal-shadow-sm);
        flex-wrap: wrap;
    }

    .target-strip-left {
        display: flex;
        align-items: center;
        gap: 8px;
        flex: 1;
        min-width: 200px;
    }

    .ts-label {
        font-size: 11.5px;
        font-weight: 850;
        text-transform: uppercase;
        color: var(--text-muted);
        display: flex;
        align-items: center;
        gap: 5px;
        white-space: nowrap;
    }

    .select-wrap-compact {
        flex: 1;
        max-width: 340px;
    }

    .select-wrap-compact select {
        padding: 5px 10px;
        font-size: 12px;
        height: 32px;
        border-radius: 6px;
    }

    .eyebrow {
        font-size: 11px;
        font-weight: 900;
        text-transform: uppercase;
        color: var(--text-muted);
        letter-spacing: 0.8px;
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 2px;
    }

    h1 {
        margin: 0;
        font-size: 22px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: -0.5px;
        color: var(--text);
    }

    .target-badge-pill {
        font-size: 10px;
        font-weight: 850;
        padding: 3px 8px;
        border-radius: 6px;
        border: 1px solid var(--border-color);
        display: inline-flex;
        align-items: center;
        gap: 5px;
    }

    .pill-success {
        background: rgba(184, 243, 74, 0.25);
        color: var(--lime);
        border-color: var(--lime);
    }

    .pill-warning {
        background: rgba(255, 225, 74, 0.2);
        color: var(--yellow);
        border-color: var(--yellow);
    }

    .pill-info {
        background: rgba(85, 223, 255, 0.2);
        color: var(--cyan);
        border-color: var(--cyan);
    }

    /* TEMPLATE MODE CARDS */
    .template-mode-cards {
        display: flex;
        flex-direction: column;
        gap: 10px;
        margin: 10px 0;
    }

    .mode-card {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        padding: 12px 14px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: var(--panel-bg-alt);
        cursor: pointer;
        transition: all 0.15s;
    }

    .mode-card:hover {
        border-color: var(--primary);
    }

    .mode-card.active {
        border-color: var(--primary);
        background: rgba(212, 255, 0, 0.08);
        box-shadow: 2px 2px 0 var(--primary);
    }

    .mode-radio input[type="radio"] {
        margin-top: 3px;
        accent-color: var(--primary);
        width: 17px;
        height: 17px;
        cursor: pointer;
    }

    .mode-info {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .mode-title-row {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .mode-title {
        font-size: 13px;
        font-weight: 900;
        color: var(--text);
    }

    .mode-badge-rec {
        font-size: 9px;
        font-weight: 900;
        background: var(--primary);
        color: var(--primary-text);
        padding: 2px 6px;
        border-radius: 4px;
        letter-spacing: 0.5px;
    }

    .mode-desc {
        font-size: 11.5px;
        color: var(--text-muted);
        line-height: 1.4;
    }

    .target-guidance-box {
        background: rgba(85, 223, 255, 0.1);
        border: 1.5px solid var(--cyan);
        border-radius: 8px;
        padding: 10px 12px;
        font-size: 11.5px;
        margin: 6px 0;
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .tgb-title {
        font-weight: 900;
        color: var(--cyan);
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .tgb-desc {
        color: var(--text);
        line-height: 1.4;
    }

    .select-wrap { flex: 1; }

    select, input[type="text"], input[type="number"], .paste-textarea, .raw-textarea {
        width: 100%;
        background: var(--panel-bg-alt);
        color: var(--text);
        border: 2px solid var(--border-color);
        border-radius: 8px;
        padding: 10px 14px;
        font-family: var(--font);
        font-size: 13px;
        font-weight: 700;
        outline: none;
    }

    .mono { font-family: var(--font-mono, monospace); }

    .target-actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    /* TEMPLATE ALERT BANNER */
    .template-alert-banner {
        background: linear-gradient(135deg, rgba(255, 225, 74, 0.15), rgba(85, 223, 255, 0.12));
        border: 2px solid var(--primary);
        border-radius: 12px;
        padding: 14px 18px;
        margin-bottom: 18px;
        display: flex;
        align-items: center;
        gap: 14px;
        box-shadow: 3px 3px 0 var(--border-color);
    }

    .tab-icon-wrap {
        width: 42px;
        height: 42px;
        border-radius: 10px;
        background: var(--primary);
        color: #000;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 20px;
        flex-shrink: 0;
    }

    .tab-text-wrap {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 3px;
    }

    .tab-heading {
        font-size: 13px;
        font-weight: 900;
        color: var(--primary);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .tab-description {
        font-size: 11.5px;
        color: var(--text);
        line-height: 1.4;
    }

    .tab-description code {
        background: rgba(0,0,0,0.3);
        padding: 2px 6px;
        border-radius: 4px;
        font-family: var(--mono);
        color: var(--yellow);
    }

    .btn-template-yellow {
        background: var(--yellow);
        color: #000;
        border: 2px solid #000;
        padding: 8px 14px;
        border-radius: 8px;
        font-weight: 800;
        font-size: 12px;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        cursor: pointer;
        box-shadow: 2px 2px 0 #000;
        white-space: nowrap;
        transition: transform 0.1s;
    }

    .btn-template-yellow:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 #000;
    }

    /* SEGMENTED TABS (RAPI, MODERN, & COMPACT) */
    .segmented-tabs {
        display: flex;
        gap: 6px;
        margin-bottom: 16px;
        background: var(--panel-bg-alt);
        padding: 4px;
        border: var(--border-width) solid var(--border-color);
        border-radius: 12px;
        box-shadow: var(--brutal-shadow-sm);
    }

    .seg-tab {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 10px 16px;
        border: none;
        border-radius: 8px;
        background: transparent;
        color: var(--text-muted);
        font-size: 13px;
        font-weight: 800;
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .seg-tab:hover {
        color: var(--text);
        background: rgba(255, 255, 255, 0.05);
    }

    .seg-tab.active {
        background: var(--primary);
        color: var(--primary-text);
        box-shadow: 2px 2px 0 var(--border-color);
        font-weight: 900;
    }

    .seg-badge {
        background: rgba(0, 0, 0, 0.2);
        color: inherit;
        padding: 2px 7px;
        border-radius: 10px;
        font-size: 11px;
        font-weight: 900;
    }

    /* KOTAK TEMPEL LINK RINGKAS */
    .quick-paste-card {
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        border-radius: 12px;
        padding: 14px 16px;
        box-shadow: var(--brutal-shadow-sm);
        display: flex;
        flex-direction: column;
        gap: 10px;
        margin-bottom: 16px;
    }

    .qpc-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 10px;
        flex-wrap: wrap;
    }

    .qpc-title {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .qpc-heading {
        font-size: 13px;
        font-weight: 900;
        color: var(--text);
        display: block;
        letter-spacing: 0.3px;
    }

    .qpc-sub {
        font-size: 11px;
        font-weight: 700;
        color: var(--text-muted);
        display: block;
    }

    .btn-paste-clip {
        background: var(--yellow);
        color: #000;
        border: 1.5px solid var(--border-color);
        border-radius: 8px;
        font-size: 11px;
        font-weight: 900;
        padding: 6px 12px;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        box-shadow: 2px 2px 0 var(--border-color);
        transition: transform 0.1s;
    }

    .btn-paste-clip:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--border-color);
    }

    .textarea-wrap {
        position: relative;
        display: flex;
        width: 100%;
    }

    .paste-textarea {
        width: 100%;
        resize: vertical;
        min-height: 60px;
        padding: 10px 38px 10px 12px;
        font-size: 12.5px;
        font-family: monospace;
        border: 1.5px solid var(--border-color);
        border-radius: 8px;
        background: var(--panel-bg-alt);
        color: var(--text);
        line-height: 1.4;
    }

    .btn-clear-paste {
        position: absolute;
        right: 8px;
        top: 8px;
        background: transparent;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 4px 6px;
        font-size: 14px;
        border-radius: 4px;
    }

    .btn-clear-paste:hover {
        color: var(--pink);
    }

    /* CHIP BAR */
    .chip-bar {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
        padding-top: 6px;
        border-top: 1px dashed var(--border-color);
    }

    .chip-label {
        font-size: 11px;
        font-weight: 800;
        color: var(--text-muted);
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .chips-scroll {
        display: flex;
        gap: 6px;
        overflow-x: auto;
        flex: 1;
    }

    .chip-btn {
        background: var(--panel-bg-alt);
        border: 1.5px solid var(--border-color);
        border-radius: 20px;
        padding: 4px 10px;
        font-size: 11px;
        font-weight: 700;
        color: var(--text);
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 6px;
        white-space: nowrap;
    }

    .chip-btn.active {
        background: var(--yellow);
        color: #000;
        font-weight: 900;
    }

    .chip-name {
        max-width: 140px;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    /* KARTU PREVIEW CEPAT & 1-KLIK SIMPAN */
    .quick-summary-card {
        background: var(--panel-bg);
        border: 2px solid var(--primary);
        border-radius: 12px;
        padding: 14px 16px;
        box-shadow: var(--brutal-shadow);
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 18px;
        animation: fadeInScale 0.15s ease-out;
    }

    @keyframes fadeInScale {
        from { opacity: 0; transform: translateY(-4px); }
        to { opacity: 1; transform: translateY(0); }
    }

    .qsc-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 8px;
    }

    .qsc-status {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .dot-live {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background: var(--lime);
        box-shadow: 0 0 8px rgba(184, 243, 74, 0.7);
        display: inline-block;
    }

    .qsc-ready-label {
        font-size: 12.5px;
        font-weight: 900;
        color: var(--text);
        letter-spacing: 0.2px;
    }

    .qsc-body {
        display: flex;
        flex-direction: column;
        gap: 10px;
        background: var(--panel-bg-alt);
        border: 1px solid var(--border-color);
        border-radius: 8px;
        padding: 10px 12px;
    }

    .qsc-name-box {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .qsc-name-box label {
        font-size: 10.5px;
        font-weight: 800;
        text-transform: uppercase;
        color: var(--text-muted);
    }

    .qsc-input-name {
        background: var(--panel-bg);
        border: 1.5px solid var(--border-color);
        border-radius: 6px;
        padding: 7px 10px;
        font-size: 13px;
        font-weight: 800;
        color: var(--text);
    }

    .qsc-chips {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }

    .qsc-chip {
        background: var(--panel-bg);
        border: 1px solid var(--border-color);
        border-radius: 6px;
        padding: 4px 8px;
        font-size: 11px;
        font-weight: 700;
        color: var(--text);
        display: inline-flex;
        align-items: center;
        gap: 5px;
    }

    .qsc-chip i {
        font-size: 10px;
        color: var(--text-muted);
    }

    .qsc-chip.chip-highlight {
        background: rgba(255, 225, 74, 0.18);
        border-color: rgba(255, 225, 74, 0.5);
        font-weight: 800;
    }

    .qsc-actions {
        display: flex;
        align-items: center;
        gap: 10px;
        flex-wrap: wrap;
    }

    .btn-save-primary {
        background: var(--primary);
        color: var(--primary-text);
        border: 2px solid var(--border-color);
        border-radius: 8px;
        padding: 10px 18px;
        font-size: 13px;
        font-weight: 900;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        box-shadow: 2px 2px 0 var(--border-color);
        transition: transform 0.1s;
    }

    .btn-save-primary:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--border-color);
    }

    .btn-save-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
        transform: none;
    }

    .btn-toggle-adv {
        background: transparent;
        border: 1.5px solid var(--border-color);
        border-radius: 8px;
        padding: 9px 12px;
        font-size: 12px;
        font-weight: 800;
        color: var(--text-muted);
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        transition: all 0.15s;
    }

    .btn-toggle-adv:hover {
        background: var(--panel-bg-alt);
        color: var(--text);
    }

    /* HINT BERSIH KALAU KOSONG */
    .clean-hint-card {
        background: var(--panel-bg-alt);
        border: 1.5px dashed var(--border-color);
        border-radius: 12px;
        padding: 14px 16px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap;
        margin-bottom: 18px;
    }

    .ch-text {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 12px;
        color: var(--text-muted);
        flex: 1;
        min-width: 250px;
        line-height: 1.5;
    }

    .ch-text strong {
        color: var(--text);
    }

    .ch-text code {
        background: rgba(0,0,0,0.15);
        padding: 1px 5px;
        border-radius: 4px;
        font-size: 11px;
    }

    .btn-text-manual {
        background: transparent;
        border: 1px dashed var(--border-color);
        border-radius: 6px;
        padding: 6px 12px;
        font-size: 11.5px;
        font-weight: 800;
        color: var(--text);
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }

    .btn-text-manual:hover {
        background: var(--panel-bg);
        border-style: solid;
    }

    /* ACCORDION ADVANCED SETTINGS */
    .advanced-editor-card {
        background: #1e1b24;
        border: 2px solid #000000;
        border-radius: 14px;
        box-shadow: 4px 5px 0 #000000;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        margin-bottom: 24px;
        animation: fadeInScale 0.15s ease-out;
    }

    :root[data-theme="light"] .advanced-editor-card {
        background: #ffffff;
        border-color: #17151a;
        box-shadow: 4px 5px 0 rgba(0,0,0,0.15);
    }

    .aec-header {
        padding: 12px 16px;
        background: #18151e;
        border-bottom: 2px solid #000000;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    :root[data-theme="light"] .aec-header {
        background: #f5f2e9;
    }

    .aec-title {
        font-size: 13.5px;
        font-weight: 900;
        display: flex;
        align-items: center;
        gap: 8px;
        color: #ffffff;
    }

    :root[data-theme="light"] .aec-title {
        color: #17151a;
    }

    .aec-close {
        background: transparent;
        border: none;
        color: var(--text-muted);
        font-size: 16px;
        cursor: pointer;
        padding: 4px 8px;
        border-radius: 4px;
    }

    .aec-close:hover {
        color: var(--pink);
    }

    /* TABS KUNING VOLT PERSIS GAMBAR */
    .em-tabs {
        display: flex;
        background: #18151e;
        border-bottom: 2px solid #000000;
    }

    :root[data-theme="light"] .em-tabs {
        background: #e8e5da;
    }

    .em-tab {
        flex: 1;
        padding: 12px 16px;
        background: transparent;
        border: none;
        font-weight: 900;
        font-size: 13px;
        color: #8f8899;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        transition: all 0.15s;
    }

    .em-tab.active {
        background: #ffe14a; /* Volt Yellow persis gambar */
        color: #000000;
    }

    .em-body {
        padding: 20px;
        background: #1e1b24;
    }

    :root[data-theme="light"] .em-body {
        background: #ffffff;
    }

    /* FORM GRID & LABELS PERSIS GAMBAR */
    .form-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
    }

    @media (max-width: 640px) {
        .wrap {
            padding: 10px 10px 85px 10px;
        }
        .page-header {
            gap: 10px;
            margin-bottom: 12px;
        }
        .page-header h1 {
            font-size: 20px;
        }
        .subtitle {
            font-size: 11px;
            margin-top: 2px;
        }
        .target-bar {
            padding: 8px 10px;
            gap: 8px;
        }
        .target-field {
            min-width: 100%;
        }
        .target-field label {
            font-size: 11px;
        }
        .target-actions {
            width: 100%;
            justify-content: space-between;
        }
        .target-actions .btn-outline {
            flex: 1;
            justify-content: center;
        }
        .template-alert-banner {
            flex-direction: column;
            align-items: flex-start;
            padding: 12px;
            gap: 10px;
            margin-bottom: 14px;
        }
        .tab-action-wrap {
            width: 100%;
        }
        .btn-template-yellow {
            width: 100%;
            justify-content: center;
        }
        .segmented-tabs {
            margin-bottom: 12px;
            padding: 3px;
        }
        .seg-tab {
            padding: 8px 10px;
            font-size: 12px;
            gap: 6px;
        }
        .quick-paste-card {
            padding: 10px 12px;
            gap: 8px;
            margin-bottom: 12px;
        }
        .qpc-heading {
            font-size: 12px;
        }
        .qpc-sub {
            font-size: 10px;
        }
        .btn-paste-clip {
            padding: 5px 10px;
            font-size: 10.5px;
        }
        .paste-textarea {
            min-height: 50px;
            padding: 7px 10px;
            font-size: 12px;
        }
        .quick-summary-card {
            padding: 12px;
            gap: 10px;
            margin-bottom: 14px;
        }
        .qsc-ready-label {
            font-size: 11.5px;
        }
        .qsc-actions {
            flex-direction: column;
            width: 100%;
            gap: 8px;
        }
        .btn-save-primary {
            width: 100%;
            justify-content: center;
            padding: 9px 14px;
            font-size: 12px;
        }
        .btn-toggle-adv {
            width: 100%;
            justify-content: center;
            font-size: 11.5px;
            padding: 8px 12px;
        }
        .clean-hint-card {
            padding: 10px 12px;
            gap: 10px;
            flex-direction: column;
            align-items: flex-start;
        }
        .ch-text {
            min-width: 100%;
            font-size: 11.5px;
        }
        .btn-text-manual {
            width: 100%;
            justify-content: center;
        }
        .advanced-editor-card {
            border-radius: 10px;
            margin-bottom: 16px;
        }
        .aec-header {
            padding: 10px 12px;
        }
        .aec-title {
            font-size: 12px;
            gap: 6px;
        }
        .em-tabs .em-tab {
            padding: 8px 10px;
            font-size: 12px;
        }
        .em-body {
            padding: 12px;
        }
        .form-grid {
            grid-template-columns: 1fr 1fr;
            gap: 8px 10px;
        }
        .form-group {
            gap: 3px;
        }
        .form-group label {
            font-size: 9.5px;
            letter-spacing: 0.3px;
        }
        select, input[type="text"], input[type="number"], .raw-textarea {
            padding: 6px 10px;
            font-size: 12px;
            border-radius: 6px;
            height: 36px;
        }
        .toggle-list {
            flex-direction: row;
            flex-wrap: wrap;
            gap: 6px 14px;
            margin-top: 4px;
            padding: 6px 8px;
            background: rgba(0,0,0,0.15);
            border-radius: 6px;
        }
        .toggle-box {
            font-size: 10.5px;
            gap: 6px;
        }
        .toggle-box input[type="checkbox"] {
            width: 17px;
            height: 17px;
        }
        .em-footer {
            padding: 10px 12px;
            gap: 8px;
        }
        .btn-dark {
            padding: 8px 12px;
            font-size: 12px;
        }
        .btn-save-yellow {
            padding: 8px 14px;
            font-size: 12px;
            flex: 1;
            justify-content: center;
        }
        .account-grid {
            grid-template-columns: 1fr;
            gap: 10px;
        }
        .account-card {
            padding: 10px 12px;
            gap: 8px;
        }
        .acc-name {
            font-size: 14px;
        }
        .acc-details {
            padding: 8px;
            gap: 4px;
            font-size: 11px;
        }
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .form-group.full-width {
        grid-column: 1 / -1;
    }

    .form-group label {
        font-size: 11px;
        font-weight: 800;
        text-transform: uppercase;
        color: #8f8899; /* Abu-abu teks persis screenshot */
        letter-spacing: 0.5px;
    }

    /* TOGGLE BOXES PERSIS GAMBAR */
    .toggle-list {
        display: flex;
        flex-direction: row;
        flex-wrap: wrap;
        gap: 10px 20px;
        margin-top: 8px;
        align-items: center;
    }

    .toggle-box {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        cursor: pointer;
        color: #ffffff;
    }

    :root[data-theme="light"] .toggle-box {
        color: #17151a;
    }

    .toggle-box input[type="checkbox"] {
        width: 18px;
        height: 18px;
        accent-color: #ffe14a; /* Checkbox kuning volt */
        cursor: pointer;
    }

    /* FOOTER PERSIS GAMBAR */
    .em-footer {
        padding: 14px 20px;
        background: #18151e;
        border-top: 2px solid #000000;
        display: flex;
        justify-content: flex-end;
        align-items: center;
        gap: 12px;
    }

    :root[data-theme="light"] .em-footer {
        background: #f5f2e9;
    }

    .btn-dark {
        background: #2b2734;
        color: #ffffff;
        border: 2px solid #000000;
        border-radius: 8px;
        padding: 10px 18px;
        font-size: 13px;
        font-weight: 800;
        cursor: pointer;
        box-shadow: 2px 2px 0 #000000;
        transition: transform 0.1s;
    }

    .btn-dark:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 #000000;
    }

    .btn-save-yellow {
        background: #ffe14a; /* Volt yellow persis tombol Simpan Perubahan di gambar */
        color: #000000;
        border: 2px solid #000000;
        border-radius: 8px;
        padding: 10px 20px;
        font-size: 13px;
        font-weight: 900;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        box-shadow: 3px 3px 0 #000000;
        transition: transform 0.1s;
    }

    .btn-save-yellow:hover {
        transform: translate(-1px, -1px);
        box-shadow: 4px 4px 0 #000000;
    }

    .btn-save-yellow:disabled {
        opacity: 0.5;
        cursor: not-allowed;
        transform: none;
    }

    .raw-yaml-wrap {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .raw-hint {
        font-size: 11px;
        color: var(--text-muted);
        font-weight: 700;
        display: flex;
        align-items: center;
        gap: 6px;
    }

    /* PROTOCOL TAGS */
    .proto-tag {
        font-size: 9px;
        font-weight: 900;
        text-transform: uppercase;
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid #000;
        box-shadow: 1px 1px 0 #000;
    }

    .proto-vless { background: var(--cyan); color: #000; }
    .proto-vmess { background: var(--pink); color: #fff; }
    .proto-trojan { background: var(--orange); color: #000; }
    .proto-ss { background: var(--lime); color: #000; }

    /* ACCOUNT GRID (KELOLA AKUN) */
    .filter-bar {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 18px;
    }

    .search-input-wrap {
        position: relative;
        display: flex;
        align-items: center;
    }

    .search-input-wrap i.fa-magnifying-glass {
        position: absolute;
        left: 14px;
        color: var(--text-muted);
        pointer-events: none;
    }

    .search-input-wrap input {
        padding-left: 38px;
        padding-right: 36px;
        height: 42px;
    }

    .clear-search {
        position: absolute;
        right: 10px;
        background: transparent;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 6px;
    }

    .protocol-pills {
        display: flex;
        gap: 6px;
        overflow-x: auto;
        padding-bottom: 4px;
    }

    .pill {
        background: var(--panel-bg);
        color: var(--text);
        border: 1.5px solid var(--border-color);
        border-radius: 20px;
        padding: 5px 12px;
        font-size: 11px;
        font-weight: 800;
        cursor: pointer;
        white-space: nowrap;
    }

    .pill.active {
        background: var(--accent);
        color: #000;
        box-shadow: 2px 2px 0 var(--border-color);
    }

    .account-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 16px;
    }

    .account-card {
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        border-radius: 12px;
        padding: 14px;
        box-shadow: var(--brutal-shadow-sm);
        display: flex;
        flex-direction: column;
        gap: 12px;
        transition: transform 0.15s, box-shadow 0.15s;
    }

    .account-card:hover {
        transform: translateY(-2px);
        box-shadow: var(--brutal-shadow);
    }

    .acc-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 8px;
    }

    .acc-title-wrap {
        display: flex;
        flex-direction: column;
        gap: 4px;
        min-width: 0;
    }

    .proto-badge {
        font-size: 9px;
        font-weight: 900;
        text-transform: uppercase;
        padding: 2px 6px;
        border-radius: 4px;
        border: 1.5px solid var(--border-color);
        width: fit-content;
        box-shadow: 1px 1px 0 var(--border-color);
    }

    .acc-name {
        margin: 0;
        font-size: 15px;
        font-weight: 800;
        color: var(--text);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .acc-actions {
        display: flex;
        gap: 4px;
        flex-shrink: 0;
    }

    .action-btn {
        background: var(--panel-bg-alt);
        border: 1.5px solid var(--border-color);
        border-radius: 6px;
        color: var(--text);
        width: 32px;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        font-size: 13px;
        transition: transform 0.1s;
    }

    .action-btn:hover {
        transform: translate(-1px, -1px);
        box-shadow: 1px 1px 0 var(--border-color);
    }

    .action-btn.edit:hover { background: var(--yellow); color: #000; }
    .action-btn.share:hover { background: var(--cyan); color: #000; }
    .action-btn.copy:hover { background: var(--lime); color: #000; }
    .action-btn.delete:hover { background: var(--pink); color: #fff; }

    /* CARD BODY COMPACT CHIPS (LEBIH RAMPING & TIDAK BIKIN PUSING) */
    .acc-body-compact {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .acc-chips-row {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        align-items: center;
    }

    .acc-chip {
        background: var(--panel-bg-alt);
        border: 1px solid var(--border-color);
        border-radius: 6px;
        padding: 4px 8px;
        font-size: 11px;
        font-weight: 700;
        color: var(--text);
        display: inline-flex;
        align-items: center;
        gap: 5px;
    }

    .acc-chip i {
        font-size: 10px;
        color: var(--text-muted);
    }

    .acc-chip.chip-sni {
        background: rgba(255, 225, 74, 0.15);
        border-color: rgba(255, 225, 74, 0.4);
        color: var(--text);
        font-weight: 800;
    }

    .acc-chip.chip-tls {
        color: var(--lime);
    }

    .acc-chip.chip-mini {
        font-size: 9.5px;
        font-weight: 900;
        padding: 2px 5px;
        background: rgba(85, 223, 255, 0.15);
        border-color: rgba(85, 223, 255, 0.4);
        color: var(--cyan);
    }

    .acc-details {
        display: flex;
        flex-direction: column;
        gap: 6px;
        background: var(--panel-bg-alt);
        border: 1.5px solid var(--border-color);
        border-radius: 8px;
        padding: 10px;
        font-size: 12px;
    }

    .detail-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 8px;
    }

    .detail-row.highlight {
        background: rgba(255, 225, 74, 0.12);
        padding: 2px 6px;
        border-radius: 4px;
    }

    .detail-label {
        color: var(--text-muted);
        font-weight: 700;
        font-size: 11px;
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .detail-val {
        font-weight: 800;
        color: var(--text);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 190px;
    }

    .badges-row {
        display: flex;
        gap: 6px;
        margin-top: 4px;
        flex-wrap: wrap;
    }

    .tag-status {
        font-size: 10px;
        font-weight: 800;
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid var(--border-color);
    }

    .tls-on { background: rgba(184, 243, 74, 0.25); color: var(--lime); }
    .tls-off { background: rgba(255, 92, 168, 0.15); color: var(--pink); }
    .tag-udp { background: rgba(85, 223, 255, 0.2); color: var(--cyan); }
    .tag-skip { background: rgba(255, 225, 74, 0.2); color: var(--yellow); }

    /* STATES */
    .empty-state, .loading-state {
        text-align: center;
        padding: 48px 16px;
        background: var(--panel-bg);
        border: var(--border-width) dashed var(--border-color);
        border-radius: 12px;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 12px;
    }

    .empty-icon i, .loading-state i {
        font-size: 38px;
        color: var(--text-muted);
    }

    .empty-state h3 { margin: 0; font-size: 18px; font-weight: 800; }
    .empty-state p, .loading-state p { margin: 0; font-size: 13px; color: var(--text-muted); max-width: 400px; }

    /* MODAL */
    .modal-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.75);
        backdrop-filter: blur(4px);
        z-index: 1000;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 16px;
    }

    .modal-card {
        background: #1e1b24;
        border: 2px solid #000;
        border-radius: 14px;
        width: 100%;
        max-width: 620px;
        max-height: 90vh;
        display: flex;
        flex-direction: column;
        box-shadow: 6px 6px 0 #000;
        overflow: hidden;
    }

    :root[data-theme="light"] .modal-card {
        background: #ffffff;
    }

    .modal-card.modal-sm { max-width: 420px; }

    .modal-header {
        padding: 14px 18px;
        background: #18151e;
        border-bottom: 2px solid #000;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    :root[data-theme="light"] .modal-header {
        background: #f5f2e9;
    }

    .modal-header-danger { background: var(--pink); color: #fff; }
    .modal-header-danger .m-title span { color: #fff; }
    .modal-header-danger .modal-close { color: #fff; }

    .m-title {
        font-size: 15px;
        font-weight: 900;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .modal-close {
        background: transparent;
        border: none;
        color: var(--text);
        cursor: pointer;
        font-size: 16px;
    }

    .modal-body {
        padding: 20px;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .modal-footer {
        padding: 14px 20px;
        background: #18151e;
        border-top: 2px solid #000;
        display: flex;
        justify-content: flex-end;
        gap: 10px;
    }

    :root[data-theme="light"] .modal-footer {
        background: #f5f2e9;
    }

    .del-preview {
        background: var(--panel-bg-alt);
        border: 1.5px solid var(--border-color);
        border-radius: 8px;
        padding: 10px 12px;
        margin: 6px 0;
    }

    .warn-text {
        font-size: 11px;
        color: var(--pink);
        font-weight: 700;
        margin: 0;
    }

    /* BUTTONS */
    .btn {
        background: var(--panel-bg-alt);
        color: var(--text);
        border: var(--border-width) solid var(--border-color);
        border-radius: 8px;
        padding: 8px 14px;
        font-size: 12px;
        font-weight: 800;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        box-shadow: 2px 2px 0 var(--border-color);
        transition: transform 0.1s;
    }

    .btn:hover {
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--border-color);
    }

    .btn-primary { background: var(--primary); color: var(--primary-text); }
    .btn-danger { background: var(--pink); color: #fff; }
    .btn-outline { background: var(--panel-bg); }
    .btn-icon { width: 38px; height: 38px; padding: 0; display: flex; align-items: center; justify-content: center; }
    .btn-sm { padding: 4px 10px; font-size: 11px; }

    /* TOAST */
    .brutal-toast {
        position: fixed;
        bottom: 24px;
        right: 24px;
        background: var(--panel-bg);
        border: var(--border-width) solid var(--border-color);
        border-radius: 10px;
        padding: 12px 16px;
        box-shadow: 4px 4px 0 var(--border-color);
        z-index: 2000;
        display: flex;
        flex-direction: column;
        gap: 4px;
        max-width: 320px;
        animation: slideUp 0.2s ease-out;
    }

    @keyframes slideUp {
        from { transform: translateY(20px); opacity: 0; }
        to { transform: translateY(0); opacity: 1; }
    }

    .toast-title {
        font-size: 13px;
        font-weight: 900;
        color: var(--primary);
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .toast-desc {
        font-size: 11px;
        font-weight: 700;
        color: var(--text);
    }
</style>
