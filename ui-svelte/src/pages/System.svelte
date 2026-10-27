<script>
    import { router } from 'tinro';

    const sections = [
        {
            title: 'Utilitas & Alat',
            icon: 'fa-screwdriver-wrench',
            accent: 'var(--yellow)',
            items: [
                {
                    title: 'Scrcpy Live',
                    desc: 'Screen mirroring & kendali layar HP real-time',
                    icon: 'fa-mobile-screen',
                    iconBg: 'var(--pink)',
                    iconColor: '#fff',
                    badge: 'LIVE',
                    badgeBg: 'var(--pink)',
                    badgeColor: '#fff',
                    path: '/remote-screen'
                },
                {
                    title: 'Wi-Fi Hotspot',
                    desc: 'Ubah SSID, password hotspot, & QR Code koneksi cepat',
                    icon: 'fa-wifi',
                    iconBg: 'var(--yellow)',
                    iconColor: '#000',
                    path: '/hotspot'
                },
                {
                    title: 'Web Terminal',
                    desc: 'Konsol shell root interaktif langsung di browser',
                    icon: 'fa-terminal',
                    iconBg: 'var(--lime)',
                    iconColor: '#000',
                    badge: 'ROOT',
                    path: '/terminal'
                },
                {
                    title: 'File Manager',
                    desc: 'Jelajah penyimpanan internal & kartu SD',
                    icon: 'fa-folder-open',
                    iconBg: 'var(--yellow)',
                    iconColor: '#000',
                    path: '/filemanager?dir=/sdcard'
                }
            ]
        },
        {
            title: 'Magisk & Root',
            icon: 'fa-box-archive',
            accent: 'var(--purple)',
            items: [
                {
                    title: 'Magisk Modules',
                    desc: 'Kelola modul Magisk yang terpasang',
                    icon: 'fa-puzzle-piece',
                    iconBg: 'var(--purple)',
                    iconColor: '#fff',
                    path: '/magisk-modules'
                },
                {
                    title: 'Magisk Logs',
                    desc: 'Pantau riwayat log aktivitas Magisk & boot',
                    icon: 'fa-file-lines',
                    iconBg: 'var(--cyan)',
                    iconColor: '#000',
                    path: '/magisk-logs'
                }
            ]
        },
        {
            title: 'Konfigurasi Perangkat',
            icon: 'fa-sliders',
            accent: 'var(--cyan)',
            items: [
                {
                    title: 'Proteksi Baterai',
                    desc: 'Bypass charging & batas auto cut-off daya',
                    icon: 'fa-battery-half',
                    iconBg: 'var(--lime)',
                    iconColor: '#000',
                    path: '/battery'
                },
                {
                    title: 'Keamanan WebUI',
                    desc: 'Ubah password masuk & port akses dashboard',
                    icon: 'fa-user-shield',
                    iconBg: 'var(--cyan)',
                    iconColor: '#000',
                    path: '/security'
                },
                {
                    title: 'Scheduled Reboot & Cron',
                    desc: 'Jadwal reboot otomatis, pembersihan RAM, & tugas berkala',
                    icon: 'fa-clock',
                    iconBg: 'var(--yellow)',
                    iconColor: '#000',
                    badge: 'OFF',
                    path: '/cron'
                }
            ]
        },
        {
            title: 'Pemeliharaan & Info',
            icon: 'fa-power-off',
            accent: 'var(--pink)',
            items: [
                {
                    title: 'Reboot Device',
                    desc: 'Mulai ulang perangkat, recovery, atau bootloader',
                    icon: 'fa-power-off',
                    iconBg: 'var(--pink)',
                    iconColor: '#fff',
                    isDanger: true,
                    path: '/reboot'
                },
                {
                    title: 'Credits & Lisensi',
                    desc: 'Informasi pengembang & kontributor ProjectX',
                    icon: 'fa-circle-info',
                    iconBg: 'var(--panel-alt)',
                    iconColor: 'var(--cyan)',
                    path: '/credits'
                },
                {
                    title: 'Donasi QRIS',
                    desc: 'Dukung pengembang kopi untuk ProjectX',
                    icon: 'fa-heart',
                    iconBg: 'rgba(255, 92, 168, 0.15)',
                    iconColor: 'var(--pink)',
                    path: '/donate'
                }
            ]
        }
    ];

    function openPage(path) {
        router.goto(path);
    }
</script>

<div class="wrap">
    <!-- HEADER -->
    <div class="header">
        <div class="eyebrow">
            <span class="live-dot"></span>
            Pusat Pengaturan Perangkat
        </div>
        <h1>System & Tools</h1>
        <div class="desc">Kelola modul Magisk, akses konsol root, dan konfigurasi sistem HP</div>
    </div>

    <!-- SECTIONS LIST -->
    <div class="sections-list">
        {#each sections as sec}
            <div class="section-card">
                <div class="section-header">
                    <i class="fa-solid {sec.icon}" style="color: {sec.accent};"></i>
                    <span>{sec.title}</span>
                </div>

                <div class="items-list">
                    {#each sec.items as item}
                        <button
                            type="button"
                            class="setting-item {item.isDanger ? 'item-danger' : ''}"
                            on:click={() => openPage(item.path)}
                        >
                            <div
                                class="item-icon"
                                style="background: {item.iconBg}; color: {item.iconColor};"
                            >
                                <i class="fa-solid {item.icon}"></i>
                            </div>

                            <div class="item-content">
                                <div class="item-title-row">
                                    <span class="item-title">{item.title}</span>
                                    {#if item.badge}
                                        <span 
                                            class="item-badge" 
                                            style="{item.badgeBg ? `background:${item.badgeBg}; color:${item.badgeColor || '#000'}; border-color:var(--line);` : ''}"
                                        >
                                            {item.badge}
                                        </span>
                                    {/if}
                                </div>
                                <div class="item-desc">{item.desc}</div>
                            </div>

                            <i class="fa-solid fa-chevron-right item-arrow"></i>
                        </button>
                    {/each}
                </div>
            </div>
        {/each}
    </div>
</div>

<style>
    .wrap {
        padding: 16px;
        max-width: 680px;
        margin: 0 auto;
        padding-bottom: 90px;
        box-sizing: border-box;
    }

    .header {
        margin-bottom: 16px;
    }

    .eyebrow {
        font-size: 0.72rem;
        font-weight: 800;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--cyan);
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 3px;
    }

    .live-dot {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: var(--lime);
        box-shadow: 0 0 6px var(--lime);
        display: inline-block;
    }

    h1 {
        font-size: 1.65rem;
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

    /* SECTIONS */
    .sections-list {
        display: flex;
        flex-direction: column;
        gap: 14px;
    }

    .section-card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 14px;
        padding: 14px;
        box-shadow: var(--shadow);
    }

    .section-header {
        font-size: 11px;
        font-weight: 900;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--muted);
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 10px;
        padding-bottom: 8px;
        border-bottom: 1.5px dashed var(--line);
    }

    /* ITEMS LIST */
    .items-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .setting-item {
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 10px;
        padding: 10px 12px;
        display: flex;
        align-items: center;
        gap: 12px;
        cursor: pointer;
        text-align: left;
        box-shadow: 2px 2px 0 var(--line);
        transition: all 0.12s ease-out;
        width: 100%;
        box-sizing: border-box;
        font-family: inherit;
    }

    .setting-item:hover {
        background: var(--paper);
        border-color: var(--yellow);
        transform: translate(-1px, -1px);
        box-shadow: 3px 3px 0 var(--line);
    }

    .setting-item:active {
        transform: translate(1px, 1px);
        box-shadow: none;
    }

    .setting-item.item-danger:hover {
        border-color: var(--pink);
    }

    .item-icon {
        width: 38px;
        height: 38px;
        border-radius: 8px;
        border: 1.5px solid var(--line);
        display: grid;
        place-items: center;
        font-size: 16px;
        flex-shrink: 0;
        box-shadow: 1px 1px 0 var(--line);
    }

    .item-content {
        flex: 1;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .item-title-row {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .item-title {
        font-size: 12px;
        font-weight: 900;
        color: var(--ink);
        text-transform: uppercase;
        letter-spacing: -0.01em;
    }

    .item-badge {
        font-size: 8px;
        font-weight: 950;
        background: var(--lime);
        color: #000;
        padding: 1px 5px;
        border-radius: 4px;
        border: 1px solid var(--line);
        letter-spacing: 0.05em;
    }

    .item-desc {
        font-size: 10px;
        color: var(--muted);
        font-weight: 700;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .item-arrow {
        font-size: 11px;
        color: var(--muted);
        flex-shrink: 0;
        transition: transform 0.1s;
    }

    .setting-item:hover .item-arrow {
        color: var(--yellow);
        transform: translateX(2px);
    }

    @media (max-width: 480px) {
        .wrap {
            padding: 12px;
        }
        .item-icon {
            width: 34px;
            height: 34px;
            font-size: 14px;
        }
        .item-title {
            font-size: 11.5px;
        }
        .item-desc {
            font-size: 9.5px;
        }
    }
</style>

