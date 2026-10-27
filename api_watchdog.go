package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type WatchdogConfig struct {
	Enabled          bool   `json:"enabled"`
	IntervalSeconds  int    `json:"interval_seconds"`
	FailureThreshold int    `json:"failure_threshold"`
	TargetHost       string `json:"target_host"`
	CheckType        string `json:"check_type"` // "ping", "http", "dns"
	Action           string `json:"action"`     // "data_reconnect", "airplane_mode", "bfr_restart", "both"
	CooldownSeconds  int    `json:"cooldown_seconds"`
	CooldownMinutes  int    `json:"cooldown_minutes"`
}

type WatchdogStatus struct {
	Enabled             bool      `json:"enabled"`
	Running             bool      `json:"running"`
	ConsecutiveFails    int       `json:"consecutive_fails"`
	LastCheckTime       string    `json:"last_check_time"`
	LastCheckSuccess    bool      `json:"last_check_success"`
	LastCheckLatencyMs  int64     `json:"last_check_latency_ms"`
	LastActionTime      string    `json:"last_action_time,omitempty"`
	LastActionMessage   string    `json:"last_action_message,omitempty"`
	TotalChecks         int64     `json:"total_checks"`
	TotalFailures       int64     `json:"total_failures"`
	TotalRecoveries     int64     `json:"total_recoveries"`
	RecentLogs          []string  `json:"recent_logs"`
	Config              WatchdogConfig `json:"config"`
}

var (
	watchdogMu          sync.RWMutex
	watchdogStopChan    chan struct{}
	watchdogWorkerRunning = false

	watchdogCfg = WatchdogConfig{
		Enabled:          false, // Default strictly OFF
		IntervalSeconds:  30,
		FailureThreshold: 3,
		TargetHost:       "1.1.1.1",
		CheckType:        "ping",
		Action:           "data_reconnect",
		CooldownSeconds:  30,
		CooldownMinutes:  1,
	}

	watchdogConsecutiveFails = 0
	watchdogLastCheckTime    time.Time
	watchdogLastSuccess      = true
	watchdogLastLatencyMs    int64
	watchdogLastActionTime   time.Time
	watchdogLastActionMsg    = ""
	watchdogTotalChecks      int64
	watchdogTotalFailures    int64
	watchdogTotalRecoveries  int64
	watchdogLogs             = make([]string, 0, 50)
)

func getWatchdogConfigPath() string {
	paths := []string{
		"/data/adb/modules/projectx-webui/data/watchdog_config.json",
		"/data/adb/projectx/watchdog_config.json",
		"/data/local/tmp/projectx_watchdog.json",
		"./watchdog_config.json",
	}
	for _, p := range paths {
		dir := filepath.Dir(p)
		if _, err := os.Stat(dir); err == nil {
			return p
		}
	}
	return "./watchdog_config.json"
}

func logWatchdog(msg string) {
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	watchdogLogs = append(watchdogLogs, entry)
	if len(watchdogLogs) > 40 {
		watchdogLogs = watchdogLogs[len(watchdogLogs)-40:]
	}
}

func loadWatchdogConfig() {
	watchdogMu.Lock()
	defer watchdogMu.Unlock()

	p := getWatchdogConfigPath()
	data, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(data, &watchdogCfg)
	} else {
		// Default strictly OFF
		watchdogCfg.Enabled = false
		watchdogCfg.IntervalSeconds = 30
		watchdogCfg.FailureThreshold = 3
		watchdogCfg.TargetHost = "1.1.1.1"
		watchdogCfg.CheckType = "ping"
		watchdogCfg.Action = "data_reconnect"
		watchdogCfg.CooldownSeconds = 30
		watchdogCfg.CooldownMinutes = 1
		saveWatchdogConfigInternal()
	}

	// Sanitize
	if watchdogCfg.IntervalSeconds < 5 {
		watchdogCfg.IntervalSeconds = 30
	}
	if watchdogCfg.FailureThreshold < 1 {
		watchdogCfg.FailureThreshold = 1
	}
	if watchdogCfg.TargetHost == "" {
		watchdogCfg.TargetHost = "1.1.1.1"
	}
	if watchdogCfg.CooldownSeconds == 0 && watchdogCfg.CooldownMinutes > 0 {
		watchdogCfg.CooldownSeconds = watchdogCfg.CooldownMinutes * 60
	}
	if watchdogCfg.CooldownSeconds < 0 {
		watchdogCfg.CooldownSeconds = 30
	}

	if watchdogCfg.Enabled && !watchdogWorkerRunning {
		startWatchdogWorkerLocked()
	}
}

func saveWatchdogConfigInternal() {
	p := getWatchdogConfigPath()
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	data, err := json.MarshalIndent(watchdogCfg, "", "  ")
	if err == nil {
		_ = os.WriteFile(p, data, 0644)
	}
}

func init() {
	loadWatchdogConfig()
}

func startWatchdogWorkerLocked() {
	if watchdogWorkerRunning {
		return
	}
	watchdogStopChan = make(chan struct{})
	watchdogWorkerRunning = true
	logWatchdog("Watchdog worker diaktifkan.")

	go func() {
		for {
			watchdogMu.RLock()
			interval := watchdogCfg.IntervalSeconds
			enabled := watchdogCfg.Enabled
			stopCh := watchdogStopChan
			watchdogMu.RUnlock()

			if !enabled {
				break
			}

			// Run connectivity test
			performCheck()

			select {
			case <-stopCh:
				return
			case <-time.After(time.Duration(interval) * time.Second):
			}
		}
	}()
}

func stopWatchdogWorkerLocked() {
	if !watchdogWorkerRunning {
		return
	}
	close(watchdogStopChan)
	watchdogWorkerRunning = false
	logWatchdog("Watchdog worker dinonaktifkan.")
}

func checkConnectivity(host, checkType string) (bool, int64) {
	start := time.Now()
	timeout := 2500 * time.Millisecond

	switch checkType {
	case "http":
		url := host
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}
		client := http.Client{Timeout: timeout}
		resp, err := client.Get(url)
		latency := time.Since(start).Milliseconds()
		if err == nil {
			_ = resp.Body.Close()
			return true, latency
		}
		return false, 0

	case "dns":
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				return d.DialContext(ctx, "udp", "1.1.1.1:53")
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err := resolver.LookupHost(ctx, host)
		latency := time.Since(start).Milliseconds()
		return err == nil, latency

	default: // "ping"
		if runtime.GOOS == "linux" {
			out, err := exec.Command("ping", "-c", "1", "-W", "2", host).Output()
			latency := time.Since(start).Milliseconds()
			if err == nil {
				// Parse latency from ping output
				re := regexp.MustCompile(`time=([0-9\.]+)\s*ms`)
				matches := re.FindStringSubmatch(string(out))
				if len(matches) > 1 {
					if flt, err := strconv.ParseFloat(matches[1], 64); err == nil {
						latency = int64(flt)
					}
				}
				return true, latency
			}
			return false, 0
		} else {
			// Windows fallback: TCP ping to port 80/53
			target := host
			if !strings.Contains(target, ":") {
				target = target + ":80"
			}
			conn, err := net.DialTimeout("tcp", target, timeout)
			latency := time.Since(start).Milliseconds()
			if err == nil {
				_ = conn.Close()
				return true, latency
			}
			return false, 0
		}
	}
}

func runRootSh(cmdStr string) error {
	// 1. Coba lewat su -c terlebih dahulu
	err := exec.Command("su", "-c", cmdStr).Run()
	if err == nil {
		return nil
	}
	// 2. Fallback via sh -c (jika process sudah berjalan sebagai root uid 0)
	err = exec.Command("sh", "-c", cmdStr).Run()
	if err == nil {
		return nil
	}
	// 3. Fallback direct path /system/bin/sh
	return exec.Command("/system/bin/sh", "-c", cmdStr).Run()
}

func executeAction(action string) string {
	if runtime.GOOS != "linux" {
		return fmt.Sprintf("Simulasi aksi: %s (Bukan Linux)", action)
	}

	switch action {
	case "airplane_mode":
		// Multi-method Universal Airplane Mode (Kompatibel Xiaomi MIUI/HyperOS, Samsung OneUI, AOSP, Android 7-14+):
		// Method 1: Modern Android 10/11+ via `cmd connectivity`
		// Method 2: Global Settings + Broadcast Intent (Standar universal Android untuk OEM ROM kustom)
		_ = runRootSh("cmd connectivity airplane-mode enable 2>/dev/null; settings put global airplane_mode_on 1 && am broadcast -a android.intent.action.AIRPLANE_MODE --ez state true 2>/dev/null")
		
		// Jeda 3 detik agar RIL baseband modem benar-benar memutus koneksi dan unregister dari BTS
		time.Sleep(3 * time.Second)

		// Matikan Mode Pesawat:
		_ = runRootSh("cmd connectivity airplane-mode disable 2>/dev/null; settings put global airplane_mode_on 0 && am broadcast -a android.intent.action.AIRPLANE_MODE --ez state false 2>/dev/null")
		
		// Pastikan data seluler tersambung kembali
		time.Sleep(1 * time.Second)
		_ = runRootSh("svc data enable 2>/dev/null")

		return "Mode pesawat di-toggle (Universal Root + Broadcast)"

	case "bfr_restart":
		_ = runRootSh("/data/adb/box/scripts/box.service restart")
		return "Box For Root (BFR) di-restart"

	case "both":
		_ = runRootSh("svc data disable")
		time.Sleep(1 * time.Second)
		_ = runRootSh("svc data enable")
		time.Sleep(2 * time.Second)
		_ = runRootSh("/data/adb/box/scripts/box.service restart")
		return "Data seluler di-reset & BFR di-restart"

	default: // "data_reconnect"
		_ = runRootSh("svc data disable")
		time.Sleep(1 * time.Second)
		_ = runRootSh("svc data enable")
		return "Data seluler di-reset (svc data disable/enable)"
	}
}

func performCheck() {
	watchdogMu.Lock()
	host := watchdogCfg.TargetHost
	cType := watchdogCfg.CheckType
	action := watchdogCfg.Action
	threshold := watchdogCfg.FailureThreshold
	cooldownSecs := watchdogCfg.CooldownSeconds
	if cooldownSecs == 0 && watchdogCfg.CooldownMinutes > 0 {
		cooldownSecs = watchdogCfg.CooldownMinutes * 60
	}
	watchdogTotalChecks++
	watchdogMu.Unlock()

	ok, latency := checkConnectivity(host, cType)

	watchdogMu.Lock()
	defer watchdogMu.Unlock()

	watchdogLastCheckTime = time.Now()
	watchdogLastSuccess = ok
	watchdogLastLatencyMs = latency

	if ok {
		if watchdogConsecutiveFails > 0 {
			logWatchdog(fmt.Sprintf("Koneksi PULIH ke %s (%dms).", host, latency))
		}
		watchdogConsecutiveFails = 0
	} else {
		watchdogConsecutiveFails++
		watchdogTotalFailures++
		logWatchdog(fmt.Sprintf("Koneksi GAGAL (%d/%d RTO) ke %s.", watchdogConsecutiveFails, threshold, host))

		if watchdogConsecutiveFails >= threshold {
			// Check cooldown
			if cooldownSecs == 0 || time.Since(watchdogLastActionTime) >= time.Duration(cooldownSecs)*time.Second {
				logWatchdog(fmt.Sprintf("Threshold tercapai! Menjalankan tindakan: %s...", action))
				msg := executeAction(action)
				watchdogLastActionTime = time.Now()
				watchdogLastActionMsg = msg
				watchdogTotalRecoveries++
				watchdogConsecutiveFails = 0
				logWatchdog(fmt.Sprintf("Tindakan selesai: %s", msg))
			} else {
				logWatchdog(fmt.Sprintf("Aksi ditunda (Cooldown %d detik aktif).", cooldownSecs))
			}
		}
	}
}

// ==================== HTTP HANDLERS ====================

func handleWatchdogStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	watchdogMu.RLock()
	defer watchdogMu.RUnlock()

	lastCheckStr := ""
	if !watchdogLastCheckTime.IsZero() {
		lastCheckStr = watchdogLastCheckTime.Format("2006-01-02 15:04:05")
	}

	lastActionStr := ""
	if !watchdogLastActionTime.IsZero() {
		lastActionStr = watchdogLastActionTime.Format("2006-01-02 15:04:05")
	}

	status := WatchdogStatus{
		Enabled:             watchdogCfg.Enabled,
		Running:             watchdogWorkerRunning,
		ConsecutiveFails:    watchdogConsecutiveFails,
		LastCheckTime:       lastCheckStr,
		LastCheckSuccess:    watchdogLastSuccess,
		LastCheckLatencyMs:  watchdogLastLatencyMs,
		LastActionTime:      lastActionStr,
		LastActionMessage:   watchdogLastActionMsg,
		TotalChecks:         watchdogTotalChecks,
		TotalFailures:       watchdogTotalFailures,
		TotalRecoveries:     watchdogTotalRecoveries,
		RecentLogs:          watchdogLogs,
		Config:              watchdogCfg,
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   status,
	})
}

func handleWatchdogConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req WatchdogConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON"})
		return
	}

	watchdogMu.Lock()
	if req.IntervalSeconds >= 5 {
		watchdogCfg.IntervalSeconds = req.IntervalSeconds
	}
	if req.FailureThreshold >= 1 {
		watchdogCfg.FailureThreshold = req.FailureThreshold
	}
	if req.TargetHost != "" {
		watchdogCfg.TargetHost = req.TargetHost
	}
	if req.CheckType != "" {
		watchdogCfg.CheckType = req.CheckType
	}
	if req.Action != "" {
		watchdogCfg.Action = req.Action
	}
	if req.CooldownSeconds >= 0 {
		watchdogCfg.CooldownSeconds = req.CooldownSeconds
		watchdogCfg.CooldownMinutes = req.CooldownSeconds / 60
	} else if req.CooldownMinutes >= 0 {
		watchdogCfg.CooldownMinutes = req.CooldownMinutes
		watchdogCfg.CooldownSeconds = req.CooldownMinutes * 60
	}

	wasEnabled := watchdogCfg.Enabled
	watchdogCfg.Enabled = req.Enabled
	saveWatchdogConfigInternal()

	if req.Enabled && !wasEnabled {
		startWatchdogWorkerLocked()
	} else if !req.Enabled && wasEnabled {
		stopWatchdogWorkerLocked()
	}

	watchdogMu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Pengaturan watchdog berhasil diperbarui",
		"config":  watchdogCfg,
	})
}

func handleWatchdogTestAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	action := r.URL.Query().Get("action")
	if action == "" {
		watchdogMu.RLock()
		action = watchdogCfg.Action
		watchdogMu.RUnlock()
	}

	msg := executeAction(action)
	watchdogMu.Lock()
	watchdogLastActionTime = time.Now()
	watchdogLastActionMsg = msg + " (Manual Test)"
	logWatchdog("Uji coba manual tindakan: " + msg)
	watchdogMu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": msg,
	})
}

func handleWatchdogCheckNow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	watchdogMu.RLock()
	host := watchdogCfg.TargetHost
	cType := watchdogCfg.CheckType
	watchdogMu.RUnlock()

	ok, latency := checkConnectivity(host, cType)

	watchdogMu.Lock()
	watchdogLastCheckTime = time.Now()
	watchdogLastSuccess = ok
	watchdogLastLatencyMs = latency
	if ok {
		logWatchdog(fmt.Sprintf("Pengecekan manual OK ke %s (%dms)", host, latency))
	} else {
		logWatchdog(fmt.Sprintf("Pengecekan manual GAGAL ke %s", host))
	}
	watchdogMu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"success":    ok,
		"latency_ms": latency,
		"target":     host,
		"check_type": cType,
	})
}

