<script>
    import { onMount } from 'svelte';
    import { router } from 'tinro';
    import QRCode from 'qrcode';

    // Data QRIS SHOPREM resmi dari gambar user
    const QRIS_PAYLOAD = "00020101021126570011ID.DANA.WWW011893600915385625217502098562521750303UMI51440014ID.CO.QRIS.WWW0215ID10232887478770303UMI5204899953033605802ID5907SHOPREM6013Kab. Wonogiri6105576626304AEE6";

    let qrSvg = '';

    // Generate dynamic QR SVG with theme color stroke
    async function generateQR() {
        try {
            const rawSvg = await QRCode.toString(QRIS_PAYLOAD, {
                type: 'svg',
                margin: 1,
                errorCorrectionLevel: 'M',
                color: {
                    dark: '#000000',
                    light: '#ffffff'
                }
            });
            // Ganti stroke hitam dengan currentColor agar otomatis mengikuti CSS color: var(--primary)
            qrSvg = rawSvg.replace(/stroke="[^"]+"/g, 'stroke="currentColor"');
        } catch (err) {
            console.error('Failed to generate QR SVG:', err);
        }
    }

    onMount(() => {
        generateQR();
    });
</script>

<div class="wrap">
    <div class="header">
        <div class="header-badge">
            <i class="fa-solid fa-heart" style="color: var(--pink);"></i> Support Developer
        </div>
        <h1>Donasi QRIS</h1>
        <div class="desc">Dukung kelanjutan pengembangan ProjectX Web UI</div>
    </div>

    <!-- MAIN QRIS CARD -->
    <div class="qris-card">
        <!-- HEADER RESMI QRIS -->
        <div class="official-header">
            <div class="qris-logo-text">
                <span class="qris-badge-txt">QRIS</span>
                <span class="qris-sub-txt">QR Code Standar<br>Pembayaran Nasional</span>
            </div>
            <div class="gpn-logo-wrap">
                <span class="gpn-badge"><i class="fa-solid fa-shield-halved"></i> GPN</span>
            </div>
        </div>

        <!-- MERCHANT NAME -->
        <div class="merchant-info">
            <div class="merchant-name">SHOPREM</div>
        </div>

        <!-- THEMED QR CODE BOX -->
        <div class="qr-canvas-box">
            <div class="qr-svg-holder">
                {#if qrSvg}
                    {@html qrSvg}
                {:else}
                    <div class="qr-loading">
                        <i class="fa-solid fa-spinner fa-spin"></i> Menyiapkan QR...
                    </div>
                {/if}
            </div>
        </div>

        <div class="qris-tagline">
            <strong>SATU QRIS UNTUK SEMUA</strong>
            <span>Cek aplikasi penyelenggara di: aspi-qris.id</span>
        </div>

        <!-- PAYMENT PILLS -->
        <div class="supported-methods">
            <span class="method-pill"><i class="fa-solid fa-wallet"></i> GoPay</span>
            <span class="method-pill"><i class="fa-solid fa-wallet"></i> OVO</span>
            <span class="method-pill"><i class="fa-solid fa-wallet"></i> DANA</span>
            <span class="method-pill"><i class="fa-solid fa-wallet"></i> ShopeePay</span>
            <span class="method-pill"><i class="fa-solid fa-building-columns"></i> Semua Bank</span>
        </div>
    </div>

    <!-- INFO BOX -->
    <div class="info-card">
        <div class="info-header">
            <i class="fa-solid fa-circle-info" style="color: var(--yellow);"></i>
            <span>Catatan Donasi</span>
        </div>
        <p>
            Bisa discan langsung dari aplikasi <strong>m-Banking</strong> (BCA, Mandiri, BRI, BNI, Seabank, Jago, dll) maupun <strong>e-Wallet</strong> (DANA, GoPay, OVO, ShopeePay, LinkAja). Terima kasih banyak atas dukungan Anda! 🙏
        </p>
    </div>

    <div class="nav-back">
        <button class="btn-back" on:click={() => router.goto('/')}>
            <i class="fa-solid fa-arrow-left"></i> Kembali ke Dashboard
        </button>
    </div>
</div>

<style>
    .wrap {
        max-width: 480px;
        margin: 0 auto;
        padding: 20px 16px 40px;
    }

    .header {
        text-align: center;
        margin-bottom: 20px;
    }

    .header-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 20px;
        font-size: 11px;
        font-weight: 900;
        text-transform: uppercase;
        margin-bottom: 8px;
        box-shadow: 2px 2px 0 var(--line);
    }

    h1 {
        font-size: 26px;
        font-weight: 950;
        text-transform: uppercase;
        letter-spacing: -0.5px;
        margin: 0 0 4px;
    }

    .desc {
        font-size: 12px;
        font-weight: 700;
        color: var(--muted);
    }

    /* MAIN CARD */
    .qris-card {
        background: var(--paper);
        border: 2px solid var(--line);
        border-radius: 20px;
        padding: 20px;
        box-shadow: var(--shadow);
        margin-bottom: 16px;
    }

    /* OFFICIAL HEADER */
    .official-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-bottom: 12px;
        border-bottom: 2px dashed var(--line);
        margin-bottom: 16px;
    }

    .qris-logo-text {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .qris-badge-txt {
        font-size: 20px;
        font-weight: 950;
        letter-spacing: -1px;
        background: var(--ink);
        color: var(--paper);
        padding: 2px 8px;
        border-radius: 6px;
        border: 2px solid var(--line);
        box-shadow: 2px 2px 0 var(--line);
    }

    .qris-sub-txt {
        font-size: 9px;
        font-weight: 800;
        line-height: 1.2;
        color: var(--muted);
        text-transform: uppercase;
    }

    .gpn-badge {
        background: #e21b1b;
        color: #fff;
        font-size: 11px;
        font-weight: 950;
        padding: 4px 10px;
        border: 2px solid var(--line);
        border-radius: 8px;
        box-shadow: 2px 2px 0 var(--line);
        letter-spacing: 0.5px;
    }

    /* MERCHANT INFO */
    .merchant-info {
        text-align: center;
        margin-bottom: 16px;
    }

    .merchant-name {
        font-size: 24px;
        font-weight: 950;
        letter-spacing: -0.5px;
        text-transform: uppercase;
        color: var(--ink);
    }

    /* THEMED QR CODE BOX */
    .qr-canvas-box {
        display: flex;
        justify-content: center;
        align-items: center;
        background: #ffffff; /* Background putih bersih */
        border: 3px solid var(--line);
        border-radius: 18px;
        padding: 16px;
        margin: 0 auto 16px;
        box-shadow: 4px 4px 0 var(--line);
        max-width: 300px;
        aspect-ratio: 1 / 1;
        /* WARNA BARCODE IKUT TEMA */
        color: var(--primary);
    }

    .qr-svg-holder {
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .qr-svg-holder :global(svg) {
        width: 100%;
        height: 100%;
        display: block;
    }

    /* Pastikan path barcode mengambil warna tema secara mutlak */
    .qr-svg-holder :global(svg path[stroke]) {
        stroke: var(--primary) !important;
    }

    .qr-loading {
        color: #666;
        font-size: 14px;
        font-weight: 800;
    }

    .qris-tagline {
        text-align: center;
        font-size: 11px;
        margin-bottom: 16px;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .qris-tagline strong {
        font-size: 12px;
        letter-spacing: 0.5px;
    }

    .qris-tagline span {
        color: var(--muted);
        font-size: 10.5px;
    }

    /* PAYMENT PILLS */
    .supported-methods {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        justify-content: center;
    }

    .method-pill {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        font-size: 10px;
        font-weight: 800;
        padding: 4px 10px;
        background: var(--panel-alt);
        border: 1.5px solid var(--line);
        border-radius: 20px;
        color: var(--ink);
    }

    /* INFO CARD */
    .info-card {
        background: var(--panel-alt);
        border: 2px solid var(--line);
        border-radius: 16px;
        padding: 14px 16px;
        box-shadow: var(--shadow);
        margin-bottom: 20px;
    }

    .info-header {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;
        font-weight: 900;
        text-transform: uppercase;
        margin-bottom: 8px;
    }

    .info-card p {
        margin: 0;
        font-size: 11.5px;
        line-height: 1.5;
        font-weight: 600;
        color: var(--ink);
    }

    /* BACK BUTTON */
    .nav-back {
        text-align: center;
    }

    .btn-back {
        background: var(--paper);
        color: var(--ink);
        border: 2px solid var(--line);
        border-radius: 12px;
        padding: 10px 20px;
        font-size: 12px;
        font-weight: 900;
        box-shadow: 2px 2px 0 var(--line);
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        transition: transform 0.1s;
        font-family: inherit;
    }
    .btn-back:hover {
        transform: translate(-2px, -2px);
        box-shadow: 4px 4px 0 var(--line);
    }
    .btn-back:active {
        transform: translate(2px, 2px);
        box-shadow: none;
    }
</style>
