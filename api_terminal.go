package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var terminalUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all local LAN connections
	},
}

type TerminalExecRequest struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // in seconds, default 15
}

type TerminalExecResponse struct {
	Status     string `json:"status"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Message    string `json:"message,omitempty"`
}

type TerminalPreset struct {
	Category string   `json:"category"`
	Name     string   `json:"name"`
	Command  string   `json:"command"`
	Desc     string   `json:"desc"`
}

var defaultTerminalPresets = []TerminalPreset{
	// NETWORK
	{Category: "Network", Name: "Cek IP & Route", Command: "ip a && echo '\n--- IP ROUTE ---' && ip route", Desc: "Tampilkan interface IP dan tabel routing kernel"},
	{Category: "Network", Name: "Tabel NAT (iptables)", Command: "iptables -t nat -nvL", Desc: "Lihat seluruh aturan NAT / port forwarding"},
	{Category: "Network", Name: "Tabel Mangle & TTL", Command: "iptables -t mangle -nvL PROJECTX_TTL 2>/dev/null || iptables -t mangle -nvL", Desc: "Lihat status chain mangle dan TTL / Hop Limit"},
	{Category: "Network", Name: "Routing Rules (ip rule)", Command: "ip rule show", Desc: "Daftar policy routing Android / BFR"},
	{Category: "Network", Name: "Tabel ARP (Klien Hotspot)", Command: "cat /proc/net/arp", Desc: "Tampilkan tabel ARP perangkat yang terhubung"},
	{Category: "Network", Name: "Tes Ping 1.1.1.1", Command: "ping -c 4 1.1.1.1", Desc: "Tes latensi dan konektivitas ke Cloudflare DNS"},
	{Category: "Network", Name: "Tes Ping 8.8.8.8", Command: "ping -c 4 8.8.8.8", Desc: "Tes latensi dan konektivitas ke Google DNS"},

	// BFR & PROXY
	{Category: "BFR & Core", Name: "Cek Proses Core BFR", Command: "ps -ef | grep -E 'box|clash|sing-box|xray|projectx' | grep -v grep", Desc: "Cek apakah binary core proxy sedang berjalan"},
	{Category: "BFR & Core", Name: "Log Core Proxy Terakhir", Command: "tail -n 40 /data/adb/box/clash/clash.log 2>/dev/null || tail -n 40 /data/adb/box/sing-box/sing-box.log 2>/dev/null || echo 'Log tidak ditemukan'", Desc: "Lihat 40 baris log terakhir BFR"},
	{Category: "BFR & Core", Name: "Restart Layanan BFR", Command: "/data/adb/box/scripts/box.service restart 2>&1 || echo 'Script box.service tidak ditemukan'", Desc: "Restart service proxy BFR"},
	{Category: "BFR & Core", Name: "Preview Config YAML BFR", Command: "head -n 40 /data/adb/box/clash/config.yaml 2>/dev/null || echo 'File config tidak ditemukan'", Desc: "Lihat 40 baris awal file konfigurasi"},

	// SISTEM & PERANGKAT
	{Category: "System", Name: "RAM & Storage (free/df)", Command: "free -m && echo '\n--- STORAGE (df -h) ---' && df -h /data /system", Desc: "Lihat penggunaan memori dan kapasitas partisi internal"},
	{Category: "System", Name: "Uptime Sistem", Command: "uptime", Desc: "Lihat durasi HP menyala dan load average"},
	{Category: "System", Name: "Status Baterai & Suhu", Command: "dumpsys battery", Desc: "Lihat level baterai, status charging, dan temperatur"},
	{Category: "System", Name: "Info Sinyal Operator", Command: "dumpsys telephony.registry 2>/dev/null | grep -iE 'mSignalStrength|mServiceState' | head -n 12", Desc: "Informasi kekuatan sinyal radio modem HP"},
	{Category: "System", Name: "Logcat Android Terbaru", Command: "logcat -d -t 35", Desc: "Lihat 35 baris pesan sistem Android terakhir"},
	{Category: "System", Name: "Bersihkan Cache Memori", Command: "sync && echo 3 > /proc/sys/vm/drop_caches && echo 'Cache RAM berhasil dibersihkan!' && free -m", Desc: "Drop kernel caches untuk melegakan RAM"},
}

// Handler WebSocket untuk Interactive Root Shell
func handleTerminalWs(w http.ResponseWriter, r *http.Request) {
	ws, err := terminalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Terminal] Gagal upgrade websocket: %v\n", err)
		return
	}
	defer ws.Close()

	// Deteksi shell yang tersedia di Android / Linux
	shellCmd := "/system/bin/sh"
	if runtime.GOOS == "windows" {
		shellCmd = "powershell.exe"
	} else {
		for _, candidate := range []string{"/system_ext/bin/bash", "/system/bin/sh", "/bin/sh", "sh"} {
			if _, err := os.Stat(candidate); err == nil {
				shellCmd = candidate
				break
			}
		}
	}

	cmd := exec.Command(shellCmd)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"HOME=/data/adb",
		"USER=root",
		"PATH=/sbin:/system/sbin:/product/bin:/apex/com.android.runtime/bin:/system/bin:/system/xbin:/odm/bin:/vendor/bin:/vendor/xbin:/data/adb/magisk:/data/adb/ksu/bin:/data/adb/ap/bin",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("[Terminal] Gagal start PTY (%s): %v\n", shellCmd, err)
		_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n\x1b[31;1m[ERROR] Gagal menginisialisasi terminal: %v\x1b[0m\r\n", err)))
		return
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Set initial window size default
	_ = pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	var closeOnce sync.Once
	cleanup := func() {
		closeOnce.Do(func() {
			_ = ptmx.Close()
			_ = ws.Close()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})
	}

	// Goroutine: Baca output dari PTY master dan kirim ke WebSocket
	go func() {
		defer cleanup()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					// PTY tertutup
				}
				_ = ws.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[33m[Proses Terminal Ditutup]\x1b[0m\r\n"))
				return
			}
			if n > 0 {
				err = ws.WriteMessage(websocket.BinaryMessage, buf[:n])
				if err != nil {
					return
				}
			}
		}
	}()

	// Loop utama: Baca input dari WebSocket dan teruskan ke PTY master
	defer cleanup()
	for {
		msgType, payload, err := ws.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
			// Cek apakah pesan adalah event JSON kontrol (misal resize terminal)
			if len(payload) > 0 && payload[0] == '{' {
				var ctrl struct {
					Type string `json:"type"`
					Rows uint16 `json:"rows"`
					Cols uint16 `json:"cols"`
				}
				if err := json.Unmarshal(payload, &ctrl); err == nil && ctrl.Type == "resize" {
					if ctrl.Rows > 0 && ctrl.Cols > 0 {
						_ = pty.Setsize(ptmx, &pty.Winsize{
							Rows: ctrl.Rows,
							Cols: ctrl.Cols,
						})
						continue
					}
				}
			}

			// Kirim payload mentah (karakter/keystroke) ke PTY stdin
			_, err = ptmx.Write(payload)
			if err != nil {
				break
			}
		}
	}
}

// Handler HTTP one-shot exec (untuk single command execution & fallback)
func handleTerminalExec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"status":"error","message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req TerminalExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Command == "" {
		http.Error(w, `{"status":"error","message":"Command tidak boleh kosong"}`, http.StatusBadRequest)
		return
	}

	timeoutSec := req.Timeout
	if timeoutSec <= 0 || timeoutSec > 120 {
		timeoutSec = 15
	}

	start := time.Now()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", req.Command)
	} else {
		// Jalankan via su jika ada root, fallback ke sh
		if _, err := exec.LookPath("su"); err == nil {
			cmd = exec.Command("su", "-c", req.Command)
		} else {
			cmd = exec.Command("sh", "-c", req.Command)
		}
	}

	done := make(chan error, 1)
	var stdoutBuf, stderrBuf []byte

	// Output pipes
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		duration := time.Since(start).Milliseconds()
		resp := TerminalExecResponse{
			Status:     "error",
			Message:    err.Error(),
			ExitCode:   1,
			DurationMs: duration,
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	go func() {
		stdoutBuf, _ = io.ReadAll(stdoutPipe)
		stderrBuf, _ = io.ReadAll(stderrPipe)
		done <- cmd.Wait()
	}()

	var exitCode int
	select {
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		duration := time.Since(start).Milliseconds()
		resp := TerminalExecResponse{
			Status:     "timeout",
			Stdout:     string(stdoutBuf),
			Stderr:     "Perintah dihentikan karena melebihi batas waktu (timeout).",
			ExitCode:   124,
			DurationMs: duration,
		}
		json.NewEncoder(w).Encode(resp)
		return

	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}
	}

	duration := time.Since(start).Milliseconds()
	resp := TerminalExecResponse{
		Status:     "success",
		Stdout:     string(stdoutBuf),
		Stderr:     string(stderrBuf),
		ExitCode:   exitCode,
		DurationMs: duration,
	}
	json.NewEncoder(w).Encode(resp)
}

// Handler HTTP untuk presets
func handleTerminalPresets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"presets": defaultTerminalPresets,
	})
}

