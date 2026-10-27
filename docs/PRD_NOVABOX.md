# 📄 Product Requirements Document (PRD): NovaBox Core
**Proyek:** ProjectX WebUI  
**Komponen:** NovaBox (Standalone Native Go TProxy & Core Manager)  
**Versi Dokumen:** v1.0.0  
**Status:** In Planning / Design Phase  
**Author:** MarkX (@ntpsbl) & AI Architect  
**Tanggal:** 5 Oktober 2026  

---

## 1. Executive Summary & Visi Produk

### 1.1 Latar Belakang
Selama ini, implementasi *Transparent Proxy* (TProxy) di Android berakar pada modul shell script tradisional seperti **Box for Root (BFR)** oleh Taamarin. Meskipun berjalan dengan baik, shell script memiliki keterbatasan mendasar:
- **Overhead Proses Tinggi:** Setiap loop monitoring dan pengecekan mengeksekusi puluhan sub-proses (`awk`, `grep`, `sed`, `iptables`) yang membebani CPU, terutama pada ponsel spesifikasi rendah (HP modem).
- **Kerapuhan Scripting:** Perilaku Busybox dan shell berbeda di tiap versi Android (Android 9 hingga Android 15/16).
- **Kurangnya State Management:** Tidak ada kontrol memori internal, sulit mendeteksi crash secara instan, dan rentan terhadap aturan routing nyangkut (*internet dead lock*) jika eksekusi script terinterupsi.

### 1.2 Visi NovaBox
**NovaBox** adalah subsistem core proxy **Standalone** berbasis **Native Go** yang terintegrasi di dalam ProjectX WebUI. NovaBox bertindak sebagai *init supervisor*, pengatur routing TProxy (iptables/nftables), dan controller proxy modern tanpa bergantung pada shell script luar.

### 1.3 Prinsip Desain Utama
1. **100% Standalone & Zero-Conflict:**  
   NovaBox memiliki direktori, file konfigurasi, dan proses sendiri. Tidak mencampuri atau menimpa berkas milik BFR Taamarin (`/data/adb/box/`).
2. **Seamless Engine Switcher:**  
   Pengguna dapat memilih dengan bebas ingin menjalankan **"Engine NovaBox (Go)"** atau **"Engine BFR (Eksternal)"** secara instan lewat satu sakelar di antarmuka pengguna.
3. **Pemisahan Peran (Separation of Concerns):**  
   - **Clash / Account Generator:** Tetap mandiri sebagai alat pembuat, konversi link proxy, dan pengelola file provider akun.
   - **NovaBox:** Khusus sebagai mesin eksekusi TProxy, pengawas proses (*process supervisor*), dan pengontrol koneksi.
4. **Dedicated UI Experience:**  
   NovaBox memiliki menu sendiri di bilah navigasi bawah (*Bottom Dock*) untuk kemudahan akses dan pemantauan real-time.

---

## 2. Struktur Direktori & Lingkup Kerja (Zero-Conflict)

Untuk memastikan tidak terjadi bentrok berkas maupun konfigurasi dengan BFR Taamarin:

```
/data/adb/
├── box/                     <-- [TIDAK DISENTUH] Milik BFR Taamarin
│   ├── clash/
│   ├── scripts/
│   └── bin/
│
└── novabox/                 <-- [STANDALONE] Khusus NovaBox Core
    ├── bin/
    │   ├── mihomo           <-- Binary Core Mihomo (ARM64 / ARM32)
    │   └── sing-box         <-- (Opsional) Binary Core Sing-box
    ├── conf/
    │   ├── config.yaml      <-- Konfigurasi utama NovaBox
    │   └── template.yaml    <-- Template dasar NovaBox
    ├── providers/           <-- Folder proxy provider / akun NovaBox
    ├── run/
    │   ├── novabox.pid      <-- PID Core
    │   └── novabox.sock     <-- Unix socket / IPC
    └── logs/
        ├── core.log         <-- Log stdout/stderr core
        └── tproxy.log       <-- Log status iptables
```

> **Keuntungan:** Pengguna yang sudah menginstal BFR Taamarin tidak akan kehilangan data apa pun. Jika ingin beralih kembali ke BFR, seluruh konfigurasi lama mereka tetap utuh.

---

## 3. Fitur Utama & Kebutuhan Fungsional (Functional Requirements)

### 3.1 Dual-Engine Switcher & Anti-Collision (Pencegah Bentrok)
- **Deteksi Otomatis:** NovaBox secara berkala mendeteksi apakah service BFR Taamarin sedang berjalan (`pgrep -f /data/adb/box`).
- **Mutual Exclusion Lock:**
  - Jika NovaBox diaktifkan, backend Go otomatis memanggil penghentian BFR eksternal terlebih dahulu (`/data/adb/box/scripts/box.service stop`) agar tidak terjadi bentrok port (DNS `53`, API `9090`, Mixed `7890`) dan bentrok tabel routing `table 100`.
  - Jika pengguna mengaktifkan BFR eksternal, NovaBox secara otomatis masuk ke status standby/mati.
- **Pilihan Engine di WebUI:**  
  Toggle Segmented Control: `[ ⚡ NovaBox Engine (Go) ]` vs `[ 📦 BFR Engine (Eksternal) ]`.

### 3.2 Native Process Supervisor (Pengawas Core)
- Dijalankan langsung oleh backend Go menggunakan `os/exec.CommandContext`.
- **Self-Healing (Auto-Restart):** Jika binary Mihomo mati mendadak (out-of-memory atau crash), Go mendeteksi dalam hitungan milidetik dan merestart proses secara otomatis.
- **Graceful Reload (Tanpa Putus Koneksi):** Saat config atau akun diperbarui, Go mengirim sinyal `SIGHUP` atau memanggil REST API Clash `/configs?force=true` sehingga koneksi yang sedang aktif tidak terputus total.
- **Pemantauan Resource Real-Time:** Menghitung penggunaan CPU & RAM khusus untuk binary core.

### 3.3 Atomic TProxy & Safe Network Routing
- **Aturan Transparan (iptables & ip rule):**
  - Mengatur `ip rule add fwmark 1 table 100` dan `ip route add local 0.0.0.0/0 dev lo table 100`.
  - Mengatur tabel `mangle` untuk menangkap paket TCP dan UDP ke port TProxy NovaBox (default: `9898`).
  - **Bypass Safeguards:**
    - Bypass jaringan lokal & intranet (`127.0.0.0/8`, `192.168.0.0/16`, `10.0.0.0/8`, `172.16.0.0/12`).
    - Bypass traffic UID milik NovaBox/ProjectX agar tidak terjadi looping paket.
    - Bypass port DHCP (`67`, `68`) dan DNS lokal penting.
- **Atomic Rollback (Fitur Anti-Bongkar Jaringan):**
  - Jika terjadi kegagalan saat injeksi iptables, Go otomatis melakukan rollback/flush aturan untuk mencegah internet HP mati total (*dead-lock*).
- **Tombol Darurat Reset Jaringan:** Tombol 1-klik di WebUI untuk membersihkan seluruh aturan iptables jika terjadi anomali koneksi.

### 3.4 Integrasi dengan Clash Dashboard & Account Generator
- **Clash Dashboard Terintegrasi:** Menghubungkan langsung ke REST API core NovaBox di `127.0.0.1:9090` (menggunakan secret token aman).
- **Generator Akun Tetap Mandiri:**
  - Halaman **Kelola & Tambah Akun** tetap berdiri sendiri sebagai *tool*.
  - Di Generator Akun, ditambahkan target baru: **`NovaBox (Default)`** (`/data/adb/novabox/providers/`), selain target BFR eksternal (`/data/adb/box/clash/proxy_provider/`).

---

## 4. Antarmuka Pengguna & Navigasi (UI/UX Specification)

### 4.1 Posisi Menu di Navigasi Bawah (Bottom Dock)
Bilah navigasi bawah disesuaikan agar NovaBox memiliki akses utama:
```
[ 🏠 Home ]  [ 📊 Stats ]  [ 📶 Net ]  [ ⚡ NovaBox ]  [ ✉️ Inbox ]  [ ⚙️ System ]
```
- Mengganti tab `[ Box ]` lama menjadi **`[ NovaBox ]`** dengan ikon `fa-solid fa-cube` atau `fa-solid fa-bolt-lightning`.
- Tombol ini langsung membuka halaman utama kontrol NovaBox.

### 4.2 Halaman Utama NovaBox (`ui-svelte/src/pages/NovaBox.svelte`)
Halaman kontrol didesain dengan konsep **Neo-Brutalism Modern & Non-Intimidating**:

1. **Header & Engine Switcher Strip:**
   - Menampilkan status engine saat ini dengan pemilih segmen:
     `[ ⚡ NovaBox Core (Native Go) ]` | `[ 📦 BFR Eksternal (Shell) ]`
2. **Hero Core Status Card:**
   - Indikator LED Aktif: Hijau (Running) / Merah (Stopped).
   - Info Metric: Uptime, RAM Core, CPU Usage, Port TProxy (`9898`).
   - Tombol Utama Aksi Besar:
     - `[ ⚡ AKTIFKAN NOVABOX ]` (Kuning Volt / Cyan)
     - `[ ⏸ HENTIKAN NOVABOX ]`
     - `[ 🔄 RESTART CORE ]`
3. **Mode Proxy Quick Switcher:**
   - Tiga tombol pill: `[ Rule ]` | `[ Global ]` | `[ Direct ]`.
4. **Quick Node / Proxy Selector:**
   - Menampilkan proxy aktif saat ini, tombol cek latensi (ping), dan akses 1-klik untuk membuka **Clash Dashboard**.
5. **Shortcut & Utilitas Cepat:**
   - `[ 📑 Tambah / Kelola Akun Proxy ]` (mengarahkan ke Clash Generator).
   - `[ ⚙️ Pengaturan Routing & DNS ]`.
   - `[ 🧹 Reset Aturan Jaringan (Darurat) ]`.
6. **Live Log Viewer:**
   - Jendela terminal ringkas untuk melihat output log langsung dari Mihomo/Sing-box secara real-time.

---

## 5. API Backend Spesifikasi (Go REST Endpoints)

| Endpoint | Method | Fungsi |
| :--- | :--- | :--- |
| `/api/novabox/status` | `GET` | Mengambil status NovaBox (running, PID, RAM, CPU, engine aktif, TProxy status). |
| `/api/novabox/start` | `POST` | Mematikan BFR eksternal (jika aktif), menyalakan core NovaBox, & menerapkan iptables. |
| `/api/novabox/stop` | `POST` | Menghentikan core NovaBox dan membersihkan aturan iptables secara bersih. |
| `/api/novabox/restart` | `POST` | Melakukan restart cepat pada core NovaBox. |
| `/api/novabox/switch-engine` | `POST` | Beralih antara mode `novabox` dan `bfr_external`. |
| `/api/novabox/config` | `GET/POST` | Membaca dan menyimpan file konfigurasi utama NovaBox (`config.yaml`). |
| `/api/novabox/logs` | `GET (SSE)` | Streaming log stdout/stderr core secara real-time via Server-Sent Events. |
| `/api/novabox/reset-network`| `POST` | Membersihkan seluruh aturan routing & iptables darurat. |

---

## 6. Rencana Implementasi Bertahap (Phased Roadmap)

### Fase 1: Fondasi Backend & Standalone Environment
- [ ] Buat struktur folder mandiri `/data/adb/novabox/` (bin, conf, providers, run, logs).
- [ ] Buat modul backend Go `novabox_supervisor.go` untuk mengelola child process binary core (start, stop, monitor).
- [ ] Buat modul `novabox_iptables.go` untuk injeksi dan pelepasan aturan TProxy yang aman.

### Fase 2: API Endpoints & Engine Switcher
- [ ] Pasang REST API `/api/novabox/*` di `main.go`.
- [ ] Implementasikan deteksi BFR Taamarin dan mekanisme saling kunci (*mutual exclusion*) agar kedua core tidak pernah bentrok.

### Fase 3: Antarmuka UI/UX NovaBox
- [ ] Buat halaman `NovaBox.svelte` di `ui-svelte/src/pages/`.
- [ ] Perbarui `BottomNav.svelte` untuk menyematkan menu `NovaBox`.
- [ ] Hubungkan kontrol status, switcher mode, live log, dan tombol start/stop ke API backend.

### Fase 4: Integrasi Generator & Dashboard
- [ ] Tambahkan target path NovaBox (`/data/adb/novabox/providers/`) di `ClashGenerator.svelte`.
- [ ] Pastikan Clashboard bawaan dapat membaca REST API NovaBox pada port `9090`.

### Fase 5: Pengujian & Rilis Modul
- [ ] Uji coba skenario transisi: User tanpa BFR vs User dengan BFR aktif.
- [ ] Uji stabilitas: Tes internet bypass saat reboot, saat ganti WiFi/Data, dan saat restart core.
- [ ] Build binary Go (ARM64 & ARM32), update paket flashable zip, dan rilis versi baru.

---

## 7. Kesimpulan
Dengan arsitektur **NovaBox** ini, pengguna mendapatkan yang terbaik dari kedua dunia:
1. Pengguna BFR lama tetap aman 100% tanpa risiko kehilangan settingan.
2. Pengguna baru mendapatkan solusi *all-in-one* yang jauh lebih cepat, stabil, hemat baterai, dan ramah pengguna karena ditenagai langsung oleh native Go.
