<script>
    import { router } from 'tinro';
</script>

<div class="wrap">
    <div class="header">
        <img src="/assets/favicon.png" alt="ProjectX Logo" class="logo">
        <h1>ProjectX Web UI</h1>
        <div class="desc">A lightweight, powerful dashboard for Magisk & Box • <strong>v2.2</strong></div>
    </div>

    <div class="card">
        <div class="card-header" style="border-bottom-color:var(--pink);">
            <i class="fa-solid fa-code" style="color:var(--pink)"></i> Developer
        </div>
        
        <a href="https://t.me/ntpsbl" target="_blank" rel="noreferrer" class="credit-item">
            <div class="credit-icon" style="background:var(--pink); color:#fff;">
                <i class="fa-solid fa-user-ninja"></i>
            </div>
            <div class="credit-info">
                <div class="credit-role">Core Developer</div>
                <div class="credit-name">MarkX</div>
                <div class="credit-sub">@ntpsbl</div>
            </div>
            <i class="fa-solid fa-arrow-up-right-from-square arrow"></i>
        </a>
    </div>

    <div class="card">
        <div class="card-header" style="border-bottom-color:var(--cyan);">
            <i class="fa-solid fa-users" style="color:var(--cyan)"></i> Community
        </div>
        
        <a href="https://t.me/projectxwebui" target="_blank" rel="noreferrer" class="credit-item">
            <div class="credit-icon telegram">
                <i class="fa-brands fa-telegram"></i>
            </div>
            <div class="credit-info">
                <div class="credit-role">Official Group</div>
                <div class="credit-name">ProjectX Web UI</div>
                <div class="credit-sub">Join on Telegram</div>
            </div>
            <i class="fa-solid fa-arrow-up-right-from-square arrow"></i>
        </a>
    </div>

    <div class="card">
        <div class="card-header" style="border-bottom-color:var(--lime);">
            <i class="fa-solid fa-heart" style="color:var(--pink)"></i> Donasi & Dukungan
        </div>
        
        <button on:click={() => router.goto('/donate')} class="credit-item" style="width:100%; text-align:left; cursor:pointer;">
            <div class="credit-icon" style="background:var(--lime); color:#000;">
                <i class="fa-solid fa-qrcode"></i>
            </div>
            <div class="credit-info">
                <div class="credit-role">Scan QRIS</div>
                <div class="credit-name">Donasi via QRIS</div>
                <div class="credit-sub" style="color:var(--pink);">Dukung Pengembangan ProjectX</div>
            </div>
            <i class="fa-solid fa-chevron-right arrow"></i>
        </button>
    </div>

    <div class="footer-text">
        Made with <i class="fa-solid fa-heart"></i> by MarkX for the community.
    </div>
</div>

<style>
    .wrap { max-width: 600px; margin: 0 auto; padding: 20px; }
    .header { margin-bottom: 24px; text-align: center; }
    .logo { width: 72px; height: 72px; margin-bottom: 12px; border-radius: 16px; box-shadow: var(--shadow); border: 2px solid var(--line); }
    h1 { font-size: 26px; font-weight: 950; letter-spacing: -1px; text-transform: uppercase; line-height: 1; }
    .desc { font-size: 12px; font-weight: 700; color: var(--muted); margin-top: 6px; }

    .card { background: var(--paper); border: 2px solid var(--line); border-radius: 16px; padding: 16px; box-shadow: var(--shadow); margin-bottom: 16px; }
    .card-header { display: flex; align-items: center; gap: 10px; font-size: 13px; font-weight: 900; text-transform: uppercase; margin-bottom: 12px; padding-bottom: 8px; border-bottom: 2px dashed var(--line); }

    .credit-item { display: flex; align-items: center; gap: 14px; padding: 12px 14px; border-radius: 12px; background: var(--panel-alt); border: 2px solid var(--line); text-decoration: none; color: var(--ink); box-shadow: 2px 2px 0 var(--line); transition: transform 0.1s; }
    .credit-item:hover { transform: translate(-2px, -2px); box-shadow: 4px 4px 0 var(--line); }
    .credit-item:active { transform: translate(2px, 2px); box-shadow: none; }

    .credit-icon { width: 42px; height: 42px; border-radius: 10px; display: flex; align-items: center; justify-content: center; font-size: 18px; border: 2px solid var(--line); flex-shrink: 0; }
    .credit-icon.telegram { background: #0088cc; color: #fff; }

    .credit-info { flex: 1; }
    .credit-role { font-size: 9px; font-weight: 900; text-transform: uppercase; color: var(--muted); letter-spacing: 0.5px; margin-bottom: 2px; }
    .credit-name { font-size: 15px; font-weight: 950; }
    .credit-sub { font-size: 11px; font-weight: 700; color: var(--cyan); margin-top: 1px; }

    .arrow { color: var(--muted); font-size: 12px; }

    .footer-text { text-align: center; font-size: 11px; font-weight: 700; color: var(--muted); margin-top: 24px; }
    .footer-text i { color: var(--pink); }
</style>
