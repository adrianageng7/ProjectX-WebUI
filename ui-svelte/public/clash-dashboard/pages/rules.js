/**
 * pages/rules.js
 * ClashBoard — Rules & Rule Providers Management with Live Toggle
 * Mendukung filter status, tipe rule, pencarian, pengurutan, dan update provider
 */

const RulesPage = (() => {
    let allRules = [];
    let ruleProviders = {};
    let activeTab = 'rules'; // 'rules' | 'providers'
    let statusFilter = 'all'; // 'all' | 'enabled' | 'disabled'
    let selectedType = null;
    let sortOrder = 'asc'; // 'asc' | 'desc'
    let searchQuery = '';

    // =========================================================
    // UTILS
    // =========================================================
    function esc(s) {
        return String(s || '')
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;');
    }

    // =========================================================
    // LOAD
    // =========================================================
    async function load() {
        const rulesList = document.getElementById('rules-list');
        const providersList = document.getElementById('rule-providers-list');

        if (rulesList && activeTab === 'rules') {
            rulesList.innerHTML = `
                <div class="empty-state">
                    <i class="fa-solid fa-spinner fa-spin"></i>
                    <p>Memuat rules & providers...</p>
                </div>
            `;
        }

        try {
            const [rulesRes, provRes] = await Promise.allSettled([
                ClashAPI.getRules(),
                ClashAPI.getRuleProviders()
            ]);

            allRules = (rulesRes.status === 'fulfilled' && rulesRes.value?.rules)
                ? rulesRes.value.rules.map((r, i) => ({
                    ...r,
                    _index: i,
                    disabled: !!r.disabled
                }))
                : [];

            ruleProviders = (provRes.status === 'fulfilled' && provRes.value?.providers)
                ? provRes.value.providers
                : {};

            // Update badge counters
            const rBadge = document.getElementById('rules-count-badge');
            if (rBadge) rBadge.textContent = allRules.length;

            const pBadge = document.getElementById('providers-count-badge');
            const provCount = Object.keys(ruleProviders).length;
            if (pBadge) pBadge.textContent = provCount;

            const subtitle = document.getElementById('rules-subtitle');
            if (subtitle) {
                subtitle.textContent = `${allRules.length} rules aktif, ${provCount} rule providers`;
            }

            renderTypeChips();
            renderActiveView();

        } catch (e) {
            if (rulesList) {
                rulesList.innerHTML = `
                    <div class="empty-state">
                        <i class="fa-solid fa-triangle-exclamation" style="color:var(--danger)"></i>
                        <p>Gagal memuat rules: ${esc(e.message)}</p>
                    </div>
                `;
            }
        }
    }

    // =========================================================
    // TAB SWITCHING
    // =========================================================
    function switchTab(tab) {
        activeTab = tab;

        const btnRules = document.getElementById('tab-btn-rules');
        const btnProviders = document.getElementById('tab-btn-providers');
        const rulesControls = document.getElementById('rules-controls');
        const rulesList = document.getElementById('rules-list');
        const providersList = document.getElementById('rule-providers-list');

        if (btnRules) btnRules.classList.toggle('active', tab === 'rules');
        if (btnProviders) btnProviders.classList.toggle('active', tab === 'providers');

        if (tab === 'rules') {
            if (rulesControls) rulesControls.style.display = 'flex';
            if (rulesList) rulesList.style.display = 'flex';
            if (providersList) providersList.style.display = 'none';
            renderRules();
        } else {
            if (rulesControls) rulesControls.style.display = 'none';
            if (rulesList) rulesList.style.display = 'none';
            if (providersList) providersList.style.display = 'flex';
            renderProviders();
        }
    }

    // =========================================================
    // TYPE CHIPS
    // =========================================================
    function renderTypeChips() {
        const container = document.getElementById('rules-type-chips');
        if (!container) return;

        // Count per type
        const typeCounts = {};
        for (const r of allRules) {
            const t = r.type || 'UNKNOWN';
            typeCounts[t] = (typeCounts[t] || 0) + 1;
        }

        const types = Object.keys(typeCounts).sort((a, b) => typeCounts[b] - typeCounts[a]);

        if (types.length === 0) {
            container.innerHTML = '';
            return;
        }

        container.innerHTML = types.map(t => {
            const isActive = selectedType === t;
            return `
                <button class="rule-type-chip ${isActive ? 'active' : ''}" 
                        onclick="RulesPage.toggleTypeFilter('${esc(t)}')">
                    <span>${esc(t)}</span>
                    <span class="chip-count">${typeCounts[t]}</span>
                </button>
            `;
        }).join('');
    }

    function toggleTypeFilter(type) {
        selectedType = (selectedType === type) ? null : type;
        renderTypeChips();
        renderRules();
    }

    // =========================================================
    // STATUS & SORT & SEARCH
    // =========================================================
    function setStatusFilter(status) {
        statusFilter = status;
        document.querySelectorAll('.rule-filter-pill').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.status === status);
        });
        renderRules();
    }

    function toggleSort() {
        sortOrder = (sortOrder === 'asc') ? 'desc' : 'asc';
        const btn = document.getElementById('rule-sort-btn');
        if (btn) {
            btn.innerHTML = sortOrder === 'asc' 
                ? '<i class="fa-solid fa-arrow-down-up-across-line"></i>' 
                : '<i class="fa-solid fa-arrow-up-down-across-line"></i>';
            btn.title = `Urutan: ${sortOrder.toUpperCase()}`;
        }
        renderRules();
    }

    function onSearch(query) {
        searchQuery = (query || '').trim().toLowerCase();
        renderActiveView();
    }

    function renderActiveView() {
        if (activeTab === 'rules') {
            renderRules();
        } else {
            renderProviders();
        }
    }

    // =========================================================
    // RENDER RULES
    // =========================================================
    function renderRules() {
        const container = document.getElementById('rules-list');
        if (!container) return;

        let filtered = allRules.filter(r => {
            // Status filter
            if (statusFilter === 'enabled' && r.disabled) return false;
            if (statusFilter === 'disabled' && !r.disabled) return false;

            // Type filter
            if (selectedType && r.type !== selectedType) return false;

            // Search query
            if (searchQuery) {
                const matchType = (r.type || '').toLowerCase().includes(searchQuery);
                const matchPayload = (r.payload || '').toLowerCase().includes(searchQuery);
                const matchProxy = (r.proxy || '').toLowerCase().includes(searchQuery);
                if (!matchType && !matchPayload && !matchProxy) return false;
            }

            return true;
        });

        // Sort
        filtered.sort((a, b) => {
            return sortOrder === 'asc' ? a._index - b._index : b._index - a._index;
        });

        if (filtered.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fa-solid fa-filter-circle-xmark"></i>
                    <p>Tidak ada rule yang cocok</p>
                </div>
            `;
            return;
        }

        container.innerHTML = filtered.map(rule => {
            const isChecked = !rule.disabled;
            const title = rule.payload || rule.type || '—';
            
            // Check provider rule count if RuleSet
            let extraStat = '';
            if (rule.type && rule.type.toLowerCase() === 'ruleset' && ruleProviders[rule.payload]) {
                const p = ruleProviders[rule.payload];
                if (typeof p.ruleCount === 'number') {
                    extraStat = `<span class="rule-stat-badge" title="Jumlah rule provider">${p.ruleCount}</span>`;
                }
            } else if (rule.size) {
                extraStat = `<span class="rule-stat-badge">${rule.size}</span>`;
            }

            return `
                <div class="rule-card ${rule.disabled ? 'disabled' : ''}" id="rule-card-${rule._index}">
                    <div class="rule-card-top">
                        <div class="rule-name" title="${esc(title)}">${esc(title)}</div>
                        <div class="rule-index">#${rule._index + 1}</div>
                    </div>
                    <div class="rule-card-bottom">
                        <div class="rule-toggle-wrap">
                            <label class="rule-toggle" title="${isChecked ? 'Nonaktifkan rule' : 'Aktifkan rule'}">
                                <input type="checkbox" ${isChecked ? 'checked' : ''} onchange="RulesPage.toggle(${rule._index}, this.checked)">
                                <span class="rule-toggle-slider"></span>
                            </label>
                        </div>
                        <span class="rule-type-badge ${esc(rule.type)}">${esc(rule.type)}</span>
                        <span class="rule-arrow">→</span>
                        <span class="rule-proxy-badge" title="${esc(rule.proxy)}">${esc(rule.proxy)}</span>
                        ${extraStat}
                    </div>
                </div>
            `;
        }).join('');
    }

    // =========================================================
    // TOGGLE RULE (ON / OFF)
    // =========================================================
    async function toggle(index, isChecked) {
        const rule = allRules.find(r => r._index === index);
        if (!rule) return;

        const targetDisabled = !isChecked;
        const prevDisabled = rule.disabled;

        // Optimistic UI update
        rule.disabled = targetDisabled;
        const card = document.getElementById(`rule-card-${index}`);
        if (card) {
            card.classList.toggle('disabled', targetDisabled);
        }

        try {
            await ClashAPI.toggleRule(index, targetDisabled);
            Toast.show(`Rule #${index + 1} ${targetDisabled ? 'dinonaktifkan' : 'diaktifkan'}`, 'success', 2000);
        } catch (e) {
            // Revert state on error
            rule.disabled = prevDisabled;
            if (card) {
                card.classList.toggle('disabled', prevDisabled);
                const checkbox = card.querySelector('input[type="checkbox"]');
                if (checkbox) checkbox.checked = !prevDisabled;
            }
            Toast.show(
                `Gagal mengubah status rule: ${e.message || 'Mihomo/Clash.Meta diperlukan'}`,
                'error',
                4000
            );
        }
    }

    // =========================================================
    // RENDER PROVIDERS
    // =========================================================
    function renderProviders() {
        const container = document.getElementById('rule-providers-list');
        if (!container) return;

        let keys = Object.keys(ruleProviders);

        if (searchQuery) {
            keys = keys.filter(k => {
                const p = ruleProviders[k];
                return k.toLowerCase().includes(searchQuery) ||
                       (p.behavior || '').toLowerCase().includes(searchQuery) ||
                       (p.vehicleType || '').toLowerCase().includes(searchQuery);
            });
        }

        if (keys.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fa-solid fa-folder-open"></i>
                    <p>Tidak ada rule provider yang ditemukan</p>
                </div>
            `;
            return;
        }

        container.innerHTML = keys.map(k => {
            const p = ruleProviders[k];
            const vehicle = p.vehicleType || 'HTTP';
            const behavior = p.behavior || 'classical';
            const count = typeof p.ruleCount === 'number' ? p.ruleCount : '—';
            
            let updatedText = '';
            if (p.updatedAt) {
                try {
                    updatedText = timeSince(p.updatedAt);
                } catch {
                    updatedText = '';
                }
            }

            return `
                <div class="provider-card">
                    <div class="provider-info">
                        <div class="provider-name">${esc(p.name || k)}</div>
                        <div class="provider-meta">
                            <span class="provider-badge ${vehicle.toLowerCase()}">${esc(vehicle)}</span>
                            <span class="provider-badge">${esc(behavior)}</span>
                            <span class="provider-badge count">${count} rules</span>
                            ${updatedText ? `<span class="provider-time" title="${esc(p.updatedAt)}"><i class="fa-regular fa-clock"></i> ${updatedText}</span>` : ''}
                        </div>
                    </div>
                    <button class="provider-update-btn" onclick="RulesPage.updateProvider('${esc(p.name || k)}', this)">
                        <i class="fa-solid fa-rotate"></i> Update
                    </button>
                </div>
            `;
        }).join('');
    }

    // =========================================================
    // UPDATE PROVIDER
    // =========================================================
    async function updateProvider(name, btn) {
        const icon = btn?.querySelector('i');
        if (icon) icon.className = 'fa-solid fa-spinner fa-spin';
        if (btn) btn.disabled = true;

        try {
            await ClashAPI.updateRuleProvider(name);
            Toast.show(`Provider "${name}" berhasil diperbarui`, 'success');
            // Reload provider data
            const res = await ClashAPI.getRuleProviders();
            ruleProviders = res.providers || {};
            renderProviders();
        } catch (e) {
            Toast.show(`Gagal memperbarui provider: ${e.message}`, 'error');
        } finally {
            if (icon) icon.className = 'fa-solid fa-rotate';
            if (btn) btn.disabled = false;
        }
    }

    return {
        load,
        switchTab,
        toggleTypeFilter,
        setStatusFilter,
        toggleSort,
        onSearch,
        toggle,
        updateProvider
    };
})();
