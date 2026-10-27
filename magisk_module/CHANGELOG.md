# 📜 ProjectX WebUI - Changelog

Seluruh riwayat pembaruan, peningkatan performa, dan perbaikan bug untuk modul **ProjectX WebUI**.

---

## 🚑 [v2.3.1 - Hotfix] - 2026-10-27

### 🌐 Perbaikan Kritis: QUIC / HTTP3 & Streaming Video X / Web Cloudflare
* **Blokir Protokol QUIC (UDP 443 & 80)**: Menambahkan aturan penolakan (`REJECT`) protokol QUIC pada kernel Clash/Mihomo dan iptables BFR. Mengatasi masalah pemutar video **Twitter/X yang macet (infinite buffer)** dan situs ber-CDN Cloudflare (seperti **Dafont**) yang tidak bisa dibuka akibat paket UDP QUIC terblokir/menggantung di proxy.
* **Fallback Instan ke TCP**: Dengan menolak QUIC secara terarah, aplikasi X dan browser langsung beralih ke protokol HTTP/2 berbasis TCP yang didukung penuh oleh seluruh protokol proxy (VMess, VLESS, Trojan, Shadowsocks).
* **Aturan Routing Lengkap untuk Spotify & X**:
  * Menambahkan seluruh domain CDN Spotify (`scdn.co`, `spoti.fi`, `pscdn.co`, `spotifycdn.com/net`) ke dalam grup `📺 STREAMING & SOSMED` agar pemutar musik dan gambar album termuat sempurna.
  * Menambahkan domain Twitter/X (`x.com`, `twitter.com`, `twimg.com`, `t.co`) ke dalam grup sosmed.
  * Menambahkan bypass rute langsung (`DIRECT`) untuk situs font & aset seperti `dafont.com` agar terbebas dari proteksi Cloudflare Turnstile/Captcha pada IP proxy datacenter.
* **TCP MSS Clamping Otomatis**: Mengaktifkan clamping MSS ke Path MTU (`TCPMSS --clamp-mss-to-pmtu`) secara otomatis pada tabel iptables Android untuk mencegah masalah *Path MTU Black Hole* pada jaringan seluler 4G/LTE (Telkomsel, Indosat, XL, Tri) dan hotspot tethering.
* **DNS BFR Lebih Cepat & Bersih**: Menghapus upstream DNS Tiongkok (`114.114.114.114`) dari template Clash, memprioritaskan DoH Google dan Cloudflare untuk mencegah DNS hijacking ISP Indonesia.
* **Pilihan Pengaturan QUIC di WebUI**: Menambahkan saklar kontrol QUIC pada menu *Box Settings* WebUI agar pengguna dapat mengaktifkan/menonaktifkan protokol QUIC dengan mudah.
* **Universal Auto Mode Pesawat (Watchdog Fix)**: Memperbaiki aksi toggle *Airplane Mode* pada Watchdog agar kompatibel di semua merek HP (Samsung OneUI, Xiaomi MIUI/HyperOS, Oppo ColorOS, Android 7–14+) dengan menggabungkan perintah `cmd connectivity` dan `settings put global` + `am broadcast` root wrapper dengan durasi jeda modem 3 detik.

---

## 🚀 [v2.3] - 2026-10-07

### 📡 Fitur Baru: Wi-Fi Hotspot & Share Manager
* **Kontrol Tethering Lengkap**: Mengaktifkan dan mematikan Hotspot Wi-Fi perangkat langsung dari WebUI tanpa perlu menyentuh layar HP.
* **Ubah SSID & Password Live**: Konfigurasi nama jaringan (SSID) dan kata sandi Wi-Fi langsung tersimpan secara *atomic* ke file konfigurasi sistem Android (`WifiConfigStoreSoftAp.xml`).
* **Kode QR Wi-Fi Instan (Quick Connect)**: Klien/tamu dapat langsung memindai QR Code menggunakan kamera ponsel untuk terhubung otomatis tanpa repot mengetik sandi.
* **Deteksi Jaringan Cerdas**: Otomatis mendeteksi antarmuka aktif (`wlan0`, `wlan2`, `wlan_ap`) beserta alamat IP Gateway hotspot.
* **UI Bersih & Responsif**: Antarmuka Neo-Brutalis yang rapi, ringan, dan nyaman digunakan di layar mobile maupun desktop.

---

### 🔋 Peningkatan Besar: Universal Auto Cut-Off Battery Controller
* **Dukungan Samsung Exynos & Snapdragon**: Mengatasi masalah cut-off pada seri Samsung (seperti **Samsung Galaxy J7 Pro**, Galaxy Note, Seri S, dan Seri A) menggunakan driver kernel Samsung (`batt_slate_mode`, `store_mode`, dan `siop_level`).
* **Multi-Switch Scanner Otomatis**: Sistem secara cerdas memindai dan memilih saklar pengisian daya yang cocok untuk berbagai vendor:
  * **Samsung**: `batt_slate_mode` (inversi 1/0) & `store_mode`.
  * **MediaTek (Infinix, Tecno, Poco/Redmi MTK, Realme MTK)**: `Charging_Enable` & `charger/charging_enabled`.
  * **Oppo / Realme / OnePlus**: `input_suspend` & `mmi_charging_enable`.
  * **Qualcomm / AOSP / Pixel / Motorola / Sony**: `charging_enabled` & `battery_charging_enabled`.
  * **Xiaomi Lawas (Redmi 4A / Rolex)**: Direct I2C Bus Register SMB358.
* **Bypass SELinux Enforcing**: Mekanisme penulisan root aman (`chmod 666` + fallback `su`) sehingga tidak terblokir oleh Samsung KNOX.
* **Deteksi Nama Driver Dinamis**: Menampilkan metode kontrol yang aktif secara transparan pada dashboard baterai.

---

### 🐕 Peningkatan: Connection Watchdog (Anti-RTO)
* **Opsi Toleransi RTO Baru**:
  * Menambahkan pilihan **`1x Gagal (Agresif / Instan)`** bagi pengguna yang membutuhkan respons pemulihan kilat saat ping terputus.
* **Opsi Jeda Cooldown Baru**:
  * Menambahkan pilihan **`Tanpa Jeda (0 Detik / Instan)`** untuk pemulihan tanpa masa tunggu.
  * Menambahkan pilihan **`30 Detik (Rekomendasi Cepat & Pas)`** sebagai *sweet spot* ideal — memberi waktu cukup bagi modem untuk registrasi sinyal sekaligus tetap responsif.
* **Penyempurnaan Backend**: Konfigurasi cooldown kini mendukung satuan detik dan menit dengan kompatibilitas penuh.

---

### ⏰ Fitur Baru: Scheduled Reboot & Cron Manager
* **Manajemen Tugas Terjadwal**: Otomatisasi reboot sistem harian, restart data berkala, atau eksekusi perintah shell root kustom pada jam yang ditentukan.
* **Default Aman**: Fitur ini dinonaktifkan secara *default* (OFF) untuk mencegah tindakan reboot yang tidak diinginkan.

---

### 🐛 Perbaikan Bug & Penyempurnaan Sistem
* **Perbaikan QR Code Canvas Blank**: Mengganti render canvas berbasis DOM menjadi *Data URL murni* (`QRCode.toDataURL`), memastikan QR Code langsung tampil tanpa kendala *mounting* pada Svelte SPA.
* **Pembersihan Total Config Generator**: Menghapus seluruh residu target NovaBox dari pemindai backend dan antarmuka Generator, sehingga daftar target kini 100% fokus murni untuk Box For Root (BFR) / Clash / Mihomo.
* **Pembersihan Kode**: Mengeliminasi variabel, file usang, dan dependensi yang tidak terpakai pada bundle frontend.

---

## 📦 [v2.2] - Sebelumnya
* Implementasi modul dasar ProjectX WebUI.
* Dashboard Box for Root (BFR) / Clash / Sing-box / Mihomo.
* SMS Gateway & USSD Manager.
* Vnstat Traffic Monitor & Bandwidth Tracker.
* Magisk Modules & System Logs Viewer.
