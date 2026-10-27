# 🚀 ProjectX WebUI

<p align="center">
  <img src="magisk_module/static/favicon.png" width="96" alt="ProjectX Logo" />
</p>

<p align="center">
  <b>Dashboard WebUI Serbaguna Berbasis Native Go & Svelte untuk Android Root</b><br>
  <i>Mendukung Magisk, KernelSU, dan APatch. Port 80 (Akses langsung via http://IP_HP_KAMU).</i>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Version-v2.3.1-blue.svg" alt="Version" />
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/Frontend-Svelte%20SPA-FF3E00.svg?logo=svelte" alt="Svelte" />
  <img src="https://img.shields.io/badge/Root-Magisk%20%7C%20KSU%20%7C%20APatch-green.svg" alt="Root Solution" />
  <img src="https://img.shields.io/badge/Community-Telegram-0088cc.svg?logo=telegram" alt="Telegram" />
</p>

---

## 📖 Tentang ProjectX

**ProjectX WebUI** adalah aplikasi dashboard web berkinerja tinggi yang dikompilasi secara *native* menggunakan bahasa pemrograman **Go** dengan antarmuka modern berbasis **Svelte SPA (Single Page Application)**. 

Modul ini dirancang khusus untuk mengubah ponsel Android ber-root menjadi perangkat router/modem pintar berdaya guna tinggi tanpa perlu instalasi aplikasi runtime berat (seperti Python, PHP, atau Termux). Cukup pasang sebagai modul Magisk / KernelSU / APatch, dan WebUI langsung dapat diakses dari browser PC, laptop, maupun perangkat lain melalui alamat IP ponsel di port 80.

---

## ✨ Fitur Utama

### 📡 1. Wi-Fi Hotspot & Share Manager
* Mengaktifkan/menonaktifkan Hotspot Wi-Fi langsung dari WebUI tanpa menyentuh layar HP.
* Ganti Nama Jaringan (SSID) dan Kata Sandi Wi-Fi secara *live* & aman (atomic XML write).
* **Kode QR Wi-Fi Instan**: Tamu atau perangkat lain dapat langsung memindai QR Code untuk terhubung otomatis.
* Deteksi otomatis antarmuka aktif (`wlan0`, `wlan2`, `wlan_ap`) dan IP Gateway.

### 📶 2. TTL & Hop Limit Manager (Tethering Bypass)
* Mengubah nilai TTL IPv4 secara permanen di `/proc/sys/net/ipv4/ip_default_ttl` dan iptables mangle.
* **RFC 4861 Compliant IPv6 Hop Limit**: Dilengkapi proteksi bypass untuk paket ICMPv6 Neighbor Discovery agar koneksi IPv6 tidak mengalami *black hole* di operator seluler.
* Sangat berguna untuk menyamarkan penggunaan kuota hotspot pada operator seluler di Indonesia.

### 🔋 3. Universal Auto Cut-Off Battery Controller
* Mencegah baterai kembung saat HP difungsikan sebagai modem/router 24/7.
* **Dukungan Vendor Luas**:
  * **Samsung (Exynos & Snapdragon)**: Driver kernel `batt_slate_mode` (inversi 1/0), `store_mode`, dan `siop_level` (KNOX SELinux safe).
  * **MediaTek (Infinix, Tecno, Poco, Realme MTK)**: `Charging_Enable` & `charger/charging_enabled`.
  * **Qualcomm / AOSP / Pixel / Sony**: `charging_enabled` & `battery_charging_enabled`.
  * **Oppo / Realme / OnePlus**: `input_suspend` & `mmi_charging_enable`.
  * **Xiaomi Lawas (Redmi 4A / Rolex)**: Direct I2C Bus Register SMB358.
* Otomatis memutus pengisian daya saat persentase baterai mencapai batas atas dan menyambungkan kembali saat turun ke batas bawah.

### 🐕 4. Connection Watchdog (Anti-RTO & Auto Reconnect)
* Memonitor konektivitas internet secara berkala via Ping, HTTP, atau DNS.
* **Universal Auto Mode Pesawat (Airplane Mode)**:
  * Menggunakan sistem *dual-trigger* (`cmd connectivity` + `settings put global` & `am broadcast`) sehingga kompatibel di **seluruh merek HP (Samsung OneUI, Xiaomi MIUI/HyperOS, Oppo ColorOS, dan Android 7–14+)**.
  * Siklus jeda modem 3 detik untuk memastikan modem baseband benar-benar melepaskan koneksi BTS sebelum meminta IP baru.
* Opsi reset data seluler instan (`svc data disable/enable`) dan restart core BFR.

### 🛡️ 5. Box for Root (BFR) & Clash Configuration Generator
* Kontrol layanan **Box for Root (Taamarin)**: Start, Stop, dan Restart layanan BFR langsung dari WebUI.
* **Template Siap Pakai (1-Klik)**:
  * **Mode Pisah Trafik**: Memisahkan jalur trafik Game, Streaming/Sosmed (YouTube, TikTok, Netflix, Spotify, X/Twitter), Indo Direct (m-Banking & Marketplace), dan Blokir Iklan.
  * **Mode Standar**: Mode all-in-one proxy.
* **Fix Kritis QUIC & Cloudflare**: Dilengkapi penolakan terarah paket QUIC (UDP 443/80) agar video Twitter/X langsung berputar lancar tanpa *buffering* dan web Cloudflare (seperti Dafont) terbuka seketika.
* Manajemen Akun & Node Proxy (VMess, VLESS, Trojan, Shadowsocks, Hysteria2).

### 👥 6. Client Manager & Bandwidth Limiter
* Melihat seluruh perangkat yang terhubung ke jaringan Hotspot/Tethering (IP, MAC, Hostname, Vendor OUI).
* Membatasi kecepatan unduh/unggah (*speed limiting*) per perangkat menggunakan iptables `hashlimit`.
* Fitur blokir (*kick*) perangkat yang tidak diinginkan secara instan.

### 🖥️ 7. Remote Screen Web (Interactive Scrcpy)
* Streaming tampilan layar HP secara *real-time* ke browser menggunakan WebSockets dan scrcpy-server.
* Kontrol sentuhan layar penuh (*touch*, *tap*, *swipe*, *back*, *home*, *power*).

### 💬 8. SMS & USSD Viewer
* Membaca kotak masuk SMS modem langsung dari browser (integrasi SQLite Telephony DB & Content Provider).
* Deteksi otomatis kode OTP untuk kemudahan login akun.

### 📊 9. Vnstat & Traffic Monitor
* Grafik penggunaan bandwidth harian dan bulanan secara *real-time* berbasis Chart.js.
* Penyimpanan riwayat penggunaan kuota ke database SQLite lokal.

### 💻 10. Web Terminal (Root Shell)
* Terminal interaktif berbasis WebSocket langsung di dalam WebUI dengan hak akses Root penuh (`uid 0`).

---

## 📥 Cara Instalasi

1. Download berkas zip rilis terbaru: **`ProjectX_Magisk.zip`**.
2. Buka aplikasi root manager di HP Anda (**Magisk**, **KernelSU**, atau **APatch**).
3. Masuk ke menu **Modul**, lalu pilih **Pasang dari penyimpanan / Install from storage**.
4. Pilih file **`ProjectX_Magisk.zip`**, tunggu hingga proses instalasi selesai, lalu **Reboot HP**.
5. Setelah HP menyala kembali, buka browser dari laptop, PC, atau HP lain yang terhubung ke Wi-Fi / Hotspot yang sama, lalu akses:
   ```
   http://(IP_HP_KAMU) atau http://localhost
   ```
6. **Kredensial Default**:
   * **Username**: `admin`
   * **Password**: `admin`

---

## 🛠️ Kompilasi dari Source

### Kebutuhan:
* [Go (Golang) v1.21+](https://go.dev/dl/)
* [Node.js & npm](https://nodejs.org/) (untuk membangun UI Svelte)

### Langkah Build Modul Magisk:
1. Clone repositori:
   ```bash
   git clone https://github.com/USERNAME/ProjectX-WebUI.git
   cd ProjectX-WebUI
   ```
2. Jalankan skrip kompilasi otomatis:
   ```bash
   build_magisk.bat
   ```
   *Skrip ini akan mengompilasi binary ARM64 dan ARM32 secara multi-arch, mem-build bundle Svelte SPA, dan menghasilkan berkas siap flash `ProjectX_Magisk.zip`.*

---

## 📜 Riwayat Pembaruan (Changelog Ringkas)

* **v2.3.1 (Hotfix - 2026-10-27)**:
  * 🚑 **Fix Kritis IPv6 Hop Limit**: Menambahkan proteksi `-p icmpv6 -j RETURN` pada `PROJECTX_TTL` agar paket Neighbor Discovery (RFC 4861) tidak didrop router/BTS. Mengatasi masalah web seperti Spotify, Dafont, dan video Twitter/X yang tidak bisa dibuka saat modul aktif.
  * 🌐 **QUIC Blocking & Routing X/Spotify**: Menolak UDP 443 QUIC pada kernel Clash/BFR untuk memaksa *fallback* instan ke TCP HTTP/2 (memperbaiki pemutar video Twitter/X dan CDN Cloudflare). Menambahkan CDN lengkap Spotify (`scdn.co`) dan Twitter/X.
  * ✈️ **Universal Watchdog Airplane Mode**: Menambahkan dual-trigger Intent broadcast + root wrapper `su -c` dengan durasi jeda 3 detik, bekerja di seluruh merek HP (Xiaomi, Samsung, Oppo, dll.).
  * ⚡ **TCP MSS Clamping**: Clamping MSS otomatis pada iptables untuk mencegah *packet drop* di jaringan seluler 4G/LTE.
* **v2.3 (2026-10-07)**:
  * Penambahan fitur Wi-Fi Hotspot & Share Manager (Quick QR Connect).
  * Universal Battery Auto Cut-off untuk Samsung Exynos/Snapdragon dan MTK.
  * Scheduled Reboot & Cron Manager.

---

## 🤝 Komunitas & Kontribusi

* **Pengembang**: MarkX ([@ntpsbl](https://t.me/ntpsbl))
* **Grup Komunitas Telegram**: [ProjectX WebUI Community](https://t.me/projectxwebui)
* **Kredit Khusus**: 
  * [Taamarin](https://github.com/taamarin) untuk ekosistem dasar *Box for Root (BFR)*.
  * Tim pengembang Clash, Mihomo, Sing-box, dan Scrcpy.

---

<p align="center">Made with ❤️ for the Android Root & Modding Community.</p>
