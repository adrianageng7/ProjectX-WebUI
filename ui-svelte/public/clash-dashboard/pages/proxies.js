/**
 * pages/proxies.js
 * ClashBoard Proxies Page — Sleek 2-Grid, Mobile-Friendly, Header Ping on Collapse
 */

const ProxiesPage = (() => {
    let allProxies = {};   // Map of proxyName -> proxyObject (merged from /proxies & /providers/proxies)
    let allGroups  = [];   // List of group objects
    let activeFilter = 'all'; // 'all' | 'selector' | 'auto'
    let searchQuery  = '';
    let isAllExpanded = true;
    let isTestingAll = false;

    // =========================================================
    // HELPER: ESCAPE ATTRIBUTES & HTML
    // =========================================================
    function escapeAttr(str) {
        return (str || '').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
    }

    function escapeHtml(str) {
        return (str || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
    }

    // =========================================================
    // LATENCY HELPERS
    // =========================================================
    function latencyClass(ms) {
        if (!ms || ms <= 0) return 'none';
        if (ms < 180) return 'low';
        if (ms < 450) return 'mid';
        return 'high';
    }

    function getProxyLatency(proxy) {
        if (!proxy) return { ms: 0, status: 'untested' };
        
        // 1. Direct history in proxy object
        const history = proxy.history || [];
        if (history.length > 0) {
            const last = history[history.length - 1];
            if (last && typeof last.delay === 'number') {
                if (last.delay > 0) return { ms: last.delay, status: 'ok' };
                return { ms: 0, status: 'timeout' };
            }
        }

        // 2. Extra history in Mihomo (extra[testUrl].history)
        if (proxy.extra) {
            for (const key of Object.keys(proxy.extra)) {
                const item = proxy.extra[key];
                if (item && item.history && item.history.length > 0) {
                    const last = item.history[item.history.length - 1];
                    if (last && typeof last.delay === 'number') {
                        if (last.delay > 0) return { ms: last.delay, status: 'ok' };
                        return { ms: 0, status: 'timeout' };
                    }
                }
            }
        }

        return { ms: 0, status: 'untested' };
    }

    // Safe UI update for all badge occurrences of proxyName
    function updateProxyLatencyUI(proxyName, ms, status) {
        // 1. Update node badges
        document.querySelectorAll('.latency-badge').forEach(badge => {
            if (badge.getAttribute('data-proxy-name') === proxyName) {
                if (status === 'testing') {
                    badge.className = 'latency-badge testing';
                    badge.innerHTML = '<i class="fa-solid fa-bolt fa-spin"></i>';
                } else if (status === 'timeout' || ms <= 0) {
                    badge.className = 'latency-badge timeout';
                    badge.textContent = 'Timeout';
                } else {
                    badge.className = `latency-badge ${latencyClass(ms)}`;
                    badge.textContent = `${ms}ms`;
                }
            }
        });

        // 2. Update header active ping badge if this proxy is currently active in any group
        allGroups.forEach(g => {
            if (g.now === proxyName) {
                document.querySelectorAll('.active-ping-badge').forEach(hBadge => {
                    if (hBadge.getAttribute('data-header-ping') === g.name) {
                        if (status === 'testing') {
                            hBadge.className = 'active-ping-badge testing';
                            hBadge.innerHTML = '<i class="fa-solid fa-bolt fa-spin"></i>';
                        } else if (status === 'timeout' || ms <= 0) {
                            hBadge.className = 'active-ping-badge timeout';
                            hBadge.textContent = 'Timeout';
                        } else {
                            hBadge.className = `active-ping-badge ${latencyClass(ms)}`;
                            hBadge.textContent = `${ms}ms`;
                        }
                    }
                });
            }
        });
    }

    // =========================================================
    // PROTOCOL & GROUP ICONS & BADGES
    // =========================================================
    function getGroupIcon(type) {
        const map = {
            'Selector':    'fa-sliders',
            'URLTest':     'fa-wand-magic-sparkles',
            'Fallback':    'fa-shield-heart',
            'LoadBalance': 'fa-scale-balanced',
            'Relay':       'fa-route',
            'Compatible':  'fa-puzzle-piece'
        };
        return map[type] || 'fa-layer-group';
    }

    function getProtoBadge(type) {
        if (!type) return '<span class="proto-tag proto-other">?</span>';
        const t = type.toUpperCase();
        if (t.includes('VLESS')) return '<span class="proto-tag proto-vless">VLESS</span>';
        if (t.includes('VMESS')) return '<span class="proto-tag proto-vmess">VMESS</span>';
        if (t.includes('TROJAN')) return '<span class="proto-tag proto-trojan">TROJAN</span>';
        if (t.includes('SHADOWSOCKS') || t === 'SS') return '<span class="proto-tag proto-ss">SS</span>';
        if (t.includes('HYSTERIA') || t.includes('HY2')) return '<span class="proto-tag proto-hy2">HY2</span>';
        if (t.includes('WIREGUARD') || t === 'WG') return '<span class="proto-tag proto-wg">WG</span>';
        if (t.includes('DIRECT')) return '<span class="proto-tag proto-direct">DIR</span>';
        if (t.includes('REJECT')) return '<span class="proto-tag proto-reject">REJ</span>';
        if (t.includes('PASS')) return '<span class="proto-tag proto-pass">PASS</span>';
        return `<span class="proto-tag proto-other">${escapeHtml(t.slice(0, 4))}</span>`;
    }

    // =========================================================
    // RENDER GROUP CARD (2-GRID MOBILE FRIENDLY)
    // =========================================================
    function renderGroup(group) {
        const proxies = group.all || [];
        const current = group.now || '';
        const type    = group.type || 'Selector';
        const isSelector = type === 'Selector';

        const card = document.createElement('div');
        card.className = `proxy-group-card ${isAllExpanded ? 'open' : ''}`;
        card.dataset.name = group.name;
        card.dataset.type = type;

        // Active node latency for header (visible even when collapsed!)
        const activeNode = allProxies[current] || { name: current };
        const activeLatency = getProxyLatency(activeNode);
        let activePingHtml = '';
        if (activeLatency.status === 'ok') {
            activePingHtml = `<span class="active-ping-badge ${latencyClass(activeLatency.ms)}" data-header-ping="${escapeAttr(group.name)}" onclick="event.stopPropagation(); ProxiesPage.testProxy('${escapeAttr(current)}', event)" title="Klik untuk tes ping ${escapeAttr(current)}">${activeLatency.ms}ms</span>`;
        } else if (activeLatency.status === 'timeout') {
            activePingHtml = `<span class="active-ping-badge timeout" data-header-ping="${escapeAttr(group.name)}" onclick="event.stopPropagation(); ProxiesPage.testProxy('${escapeAttr(current)}', event)" title="Klik untuk tes ping ${escapeAttr(current)}">Timeout</span>`;
        } else {
            activePingHtml = `<span class="active-ping-badge untested" data-header-ping="${escapeAttr(group.name)}" onclick="event.stopPropagation(); ProxiesPage.testProxy('${escapeAttr(current)}', event)" title="Klik untuk tes ping ${escapeAttr(current)}"><i class="fa-solid fa-bolt"></i> Ping</span>`;
        }

        // Header: Top row (title + actions), Bottom row (Active Node + Live Ping)
        const header = document.createElement('div');
        header.className = 'proxy-group-header';
        header.innerHTML = `
            <div class="group-header-top">
                <div class="group-title-wrap">
                    <i class="fa-solid ${getGroupIcon(type)} group-mini-icon"></i>
                    <span class="group-name" title="${escapeAttr(group.name)}">${escapeHtml(group.name)}</span>
                    <span class="group-type-badge">${escapeHtml(type)}</span>
                </div>
                <div class="group-header-actions">
                    <button class="btn-group-ping" title="Test Group Latency" onclick="event.stopPropagation(); ProxiesPage.testGroup('${escapeAttr(group.name)}', this)">
                        <i class="fa-solid fa-bolt"></i>
                    </button>
                    <i class="fa-solid fa-chevron-down group-chevron"></i>
                </div>
            </div>
            <div class="group-active-bar" data-group-active="${escapeAttr(group.name)}" title="Node Aktif: ${escapeAttr(current)}">
                <div class="group-active-info">
                    <span class="active-dot"></span>
                    <span class="active-node-name" data-active-label="${escapeAttr(group.name)}">${escapeHtml(current || 'Direct')}</span>
                </div>
                ${activePingHtml}
            </div>
        `;

        header.addEventListener('click', () => {
            card.classList.toggle('open');
        });

        card.appendChild(header);

        // Proxy Nodes List (2-column grid of vertical cardlets)
        const list = document.createElement('div');
        list.className = 'proxy-list-grid';

        proxies.forEach(pName => {
            const proxy = allProxies[pName] || { name: pName, type: 'Unknown', history: [] };
            const isSelected = pName === current;
            const latencyInfo = getProxyLatency(proxy);

            let latencyHtml = '';
            if (latencyInfo.status === 'ok') {
                latencyHtml = `<span class="latency-badge ${latencyClass(latencyInfo.ms)}" data-proxy-name="${escapeAttr(pName)}">${latencyInfo.ms}ms</span>`;
            } else if (latencyInfo.status === 'timeout') {
                latencyHtml = `<span class="latency-badge timeout" data-proxy-name="${escapeAttr(pName)}">Timeout</span>`;
            } else {
                latencyHtml = `<span class="latency-badge untested" data-proxy-name="${escapeAttr(pName)}"><i class="fa-solid fa-bolt"></i></span>`;
            }

            const item = document.createElement('div');
            item.className = `proxy-compact-item ${isSelected ? 'selected' : ''} ${!isSelector ? 'read-only' : ''}`;
            item.dataset.proxy = pName;
            item.title = pName;

            item.innerHTML = `
                <div class="p-node-top">
                    ${getProtoBadge(proxy.type)}
                    <div class="p-latency-wrap" onclick="event.stopPropagation(); ProxiesPage.testProxy('${escapeAttr(pName)}', event)" title="Klik untuk tes ping">
                        ${latencyHtml}
                    </div>
                </div>
                <div class="p-node-bottom">
                    <span class="p-check-icon"><i class="fa-solid fa-check"></i></span>
                    <span class="p-node-name" title="${escapeAttr(pName)}">${escapeHtml(pName)}</span>
                </div>
            `;

            // Selection click
            if (isSelector) {
                item.addEventListener('click', () => selectProxy(group.name, pName, card));
            }

            list.appendChild(item);
        });

        card.appendChild(list);
        return card;
    }

    // =========================================================
    // SELECT PROXY
    // =========================================================
    async function selectProxy(groupName, proxyName, card) {
        try {
            await ClashAPI.selectProxy(groupName, proxyName);

            // Update UI card items selection
            card.querySelectorAll('.proxy-compact-item').forEach(el => {
                const match = el.dataset.proxy === proxyName;
                el.classList.toggle('selected', match);
            });

            // Update group object now
            const grp = allGroups.find(g => g.name === groupName);
            if (grp) grp.now = proxyName;

            // Update header active label
            const labelEl = card.querySelector('.active-node-name');
            if (labelEl) {
                labelEl.textContent = proxyName;
            }

            // Update header active ping
            const targetNode = allProxies[proxyName] || { name: proxyName };
            const lat = getProxyLatency(targetNode);
            const pingEl = card.querySelector('.active-ping-badge');
            if (pingEl) {
                pingEl.onclick = (e) => {
                    e.stopPropagation();
                    ProxiesPage.testProxy(proxyName, e);
                };
                pingEl.title = `Klik untuk tes ping ${proxyName}`;
                if (lat.status === 'ok') {
                    pingEl.className = `active-ping-badge ${latencyClass(lat.ms)}`;
                    pingEl.textContent = `${lat.ms}ms`;
                } else if (lat.status === 'timeout') {
                    pingEl.className = 'active-ping-badge timeout';
                    pingEl.textContent = 'Timeout';
                } else {
                    pingEl.className = 'active-ping-badge untested';
                    pingEl.innerHTML = '<i class="fa-solid fa-bolt"></i> Ping';
                }
            }

            Toast.show(`Aktif: ${proxyName}`, 'success');
        } catch (e) {
            Toast.show(`Gagal memilih: ${e.message}`, 'error');
        }
    }

    // =========================================================
    // TEST SINGLE PROXY PING
    // =========================================================
    async function testProxy(proxyName, event) {
        if (event && event.stopPropagation) event.stopPropagation();

        const node = allProxies[proxyName] || { name: proxyName };
        const providerName = node.provider || '';

        updateProxyLatencyUI(proxyName, 0, 'testing');

        try {
            const res = await ClashAPI.testLatency(proxyName, providerName);
            const delay = res.delay || 0;
            if (delay > 0) {
                updateProxyLatencyUI(proxyName, delay, 'ok');
                if (!node.history) node.history = [];
                node.history.push({ time: new Date().toISOString(), delay });
            } else {
                updateProxyLatencyUI(proxyName, 0, 'timeout');
            }
        } catch (e) {
            updateProxyLatencyUI(proxyName, 0, 'timeout');
        }
    }

    // =========================================================
    // TEST LATENCY FOR A GROUP
    // =========================================================
    async function testGroup(groupName, btn) {
        if (btn) {
            btn.classList.add('testing');
            btn.innerHTML = '<i class="fa-solid fa-bolt fa-spin"></i>';
            btn.disabled = true;
        }

        const group = allGroups.find(g => g.name === groupName);
        if (!group) {
            if (btn) {
                btn.classList.remove('testing');
                btn.innerHTML = '<i class="fa-solid fa-bolt"></i>';
                btn.disabled = false;
            }
            return;
        }

        const proxiesToTest = group.all || [];
        
        // Mark all as testing
        proxiesToTest.forEach(pName => updateProxyLatencyUI(pName, 0, 'testing'));

        // Batch test with concurrency 4
        const batchSize = 4;
        for (let i = 0; i < proxiesToTest.length; i += batchSize) {
            const chunk = proxiesToTest.slice(i, i + batchSize);
            await Promise.all(chunk.map(async (pName) => {
                const node = allProxies[pName] || { name: pName };
                try {
                    const res = await ClashAPI.testLatency(pName, node.provider);
                    const delay = res.delay || 0;
                    if (delay > 0) {
                        updateProxyLatencyUI(pName, delay, 'ok');
                        if (!node.history) node.history = [];
                        node.history.push({ time: new Date().toISOString(), delay });
                    } else {
                        updateProxyLatencyUI(pName, 0, 'timeout');
                    }
                } catch {
                    updateProxyLatencyUI(pName, 0, 'timeout');
                }
            }));
        }

        if (btn) {
            btn.classList.remove('testing');
            btn.innerHTML = '<i class="fa-solid fa-bolt"></i>';
            btn.disabled = false;
        }
        Toast.show(`Test selesai: ${groupName}`, 'success');
    }

    // =========================================================
    // TEST ALL PROXIES
    // =========================================================
    async function testAll() {
        if (isTestingAll) return;
        isTestingAll = true;

        const btn = document.getElementById('btn-test-all');
        const originalText = btn ? btn.innerHTML : '';
        if (btn) {
            btn.disabled = true;
            btn.innerHTML = '<i class="fa-solid fa-bolt fa-spin"></i> <span>0%</span>';
        }

        // Get unique leaf nodes
        const groupTypes = ['Selector', 'URLTest', 'Fallback', 'LoadBalance', 'Relay'];
        const uniqueNames = Object.keys(allProxies).filter(n => {
            const p = allProxies[n];
            return p && !groupTypes.includes(p.type) && p.type !== 'Reject' && p.type !== 'Direct';
        });

        if (uniqueNames.length === 0) {
            if (btn) {
                btn.disabled = false;
                btn.innerHTML = originalText;
            }
            isTestingAll = false;
            return;
        }

        // Mark all as testing
        uniqueNames.forEach(name => updateProxyLatencyUI(name, 0, 'testing'));

        let completed = 0;
        const total = uniqueNames.length;
        const batchSize = 5;

        for (let i = 0; i < total; i += batchSize) {
            const chunk = uniqueNames.slice(i, i + batchSize);
            await Promise.all(chunk.map(async (name) => {
                const node = allProxies[name] || { name };
                try {
                    const res = await ClashAPI.testLatency(name, node.provider);
                    const delay = res.delay || 0;
                    if (delay > 0) {
                        updateProxyLatencyUI(name, delay, 'ok');
                        if (!node.history) node.history = [];
                        node.history.push({ time: new Date().toISOString(), delay });
                    } else {
                        updateProxyLatencyUI(name, 0, 'timeout');
                    }
                } catch {
                    updateProxyLatencyUI(name, 0, 'timeout');
                } finally {
                    completed++;
                    if (btn) {
                        btn.innerHTML = `<i class="fa-solid fa-bolt fa-spin"></i> <span>${completed}/${total}</span>`;
                    }
                }
            }));
        }

        if (btn) {
            btn.disabled = false;
            btn.innerHTML = '<i class="fa-solid fa-bolt"></i> <span>Test Semua</span>';
        }
        isTestingAll = false;
        Toast.show(`Semua test (${completed} node) selesai!`, 'success');
    }

    // =========================================================
    // SEARCH & FILTER
    // =========================================================
    function filter(query) {
        searchQuery = (query || '').trim().toLowerCase();
        applyFilter();
    }

    function setFilter(type, el) {
        activeFilter = type;
        document.querySelectorAll('.proxy-chip').forEach(c => c.classList.remove('active'));
        if (el) el.classList.add('active');
        applyFilter();
    }

    function toggleAll() {
        isAllExpanded = !isAllExpanded;
        const btnText = document.getElementById('toggle-all-text');
        if (btnText) btnText.textContent = isAllExpanded ? 'Ciutkan' : 'Perluas';

        document.querySelectorAll('.proxy-group-card').forEach(card => {
            card.classList.toggle('open', isAllExpanded);
        });
    }

    function applyFilter() {
        document.querySelectorAll('.proxy-group-card').forEach(card => {
            const groupType = card.dataset.type || '';
            const groupName = (card.dataset.name || '').toLowerCase();

            // Type tab filter
            let matchType = true;
            if (activeFilter === 'selector') matchType = groupType === 'Selector';
            else if (activeFilter === 'auto') matchType = (groupType === 'URLTest' || groupType === 'Fallback' || groupType === 'LoadBalance');

            // Search query filter
            let matchSearch = true;
            if (searchQuery) {
                const items = card.querySelectorAll('.proxy-compact-item');
                let foundAnyItem = false;
                items.forEach(it => {
                    const itName = (it.dataset.proxy || '').toLowerCase();
                    const visible = itName.includes(searchQuery);
                    it.style.display = visible ? '' : 'none';
                    if (visible) foundAnyItem = true;
                });
                matchSearch = groupName.includes(searchQuery) || foundAnyItem;
            } else {
                card.querySelectorAll('.proxy-compact-item').forEach(it => it.style.display = '');
            }

            card.style.display = (matchType && matchSearch) ? '' : 'none';
        });
    }

    // =========================================================
    // LOAD PROXIES & PROVIDERS
    // =========================================================
    async function load() {
        const container = document.getElementById('proxy-groups-list');
        if (!container) return;

        container.innerHTML = `
            <div class="empty-state">
                <i class="fa-solid fa-spinner fa-spin"></i>
                <p>Memuat proxies & providers...</p>
            </div>`;

        try {
            // Load both proxies and providers simultaneously
            const [proxiesRes, providersRes] = await Promise.all([
                ClashAPI.getProxies().catch(() => ({ proxies: {} })),
                ClashAPI.getProviders().catch(() => ({ providers: {} }))
            ]);

            allProxies = { ...(proxiesRes.proxies || {}) };

            // Merge proxies from providers (subscription files)
            if (providersRes && providersRes.providers) {
                for (const [provName, prov] of Object.entries(providersRes.providers)) {
                    if (prov && Array.isArray(prov.proxies)) {
                        prov.proxies.forEach(p => {
                            if (p && p.name) {
                                allProxies[p.name] = {
                                    ...p,
                                    provider: provName
                                };
                            }
                        });
                    }
                }
            }

            // Extract proxy groups
            const groupTypes = ['Selector', 'URLTest', 'Fallback', 'LoadBalance', 'Relay'];
            allGroups = Object.values(allProxies)
                .filter(p => groupTypes.includes(p.type))
                .sort((a, b) => {
                    // Put primary selectors (GLOBAL, Proxy) first
                    const aIsPrimary = a.name === 'GLOBAL' || a.name.toLowerCase() === 'proxy';
                    const bIsPrimary = b.name === 'GLOBAL' || b.name.toLowerCase() === 'proxy';
                    if (aIsPrimary && !bIsPrimary) return -1;
                    if (!aIsPrimary && bIsPrimary) return 1;

                    // Selector before URLTest/Fallback
                    if (a.type === 'Selector' && b.type !== 'Selector') return -1;
                    if (a.type !== 'Selector' && b.type === 'Selector') return 1;

                    return a.name.localeCompare(b.name);
                });

            // Update count badge
            const countAllEl = document.getElementById('count-all');
            if (countAllEl) countAllEl.textContent = allGroups.length;

            if (allGroups.length === 0) {
                container.innerHTML = `
                    <div class="empty-state">
                        <i class="fa-solid fa-circle-xmark"></i>
                        <p>Tidak ada proxy groups ditemukan</p>
                    </div>`;
                return;
            }

            container.innerHTML = '';
            allGroups.forEach(group => {
                container.appendChild(renderGroup(group));
            });

            // Apply any active search or tab filter
            applyFilter();

        } catch (e) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fa-solid fa-circle-xmark"></i>
                    <p>Gagal memuat proxies: ${e.message}</p>
                </div>`;
        }
    }

    return {
        load,
        testAll,
        testGroup,
        testProxy,
        filter,
        setFilter,
        toggleAll
    };
})();
