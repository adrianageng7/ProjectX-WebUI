package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver (tidak butuh CGO / NDK)
)

func main() {
	// 0. Inisialisasi Traffic Monitor Database & Background Collector
	initTrafficDB()
	startTrafficCollector()

	// 0.1 Inisialisasi Battery Controller & Auto Cut-off Monitor
	initBatteryController()

	// 1. Serve file statis dengan SPA fallback
	staticDir := "./static"
	if fi, err := os.Stat(staticDir); err != nil || !fi.IsDir() {
		if exePath, err := os.Executable(); err == nil {
			cand := filepath.Join(filepath.Dir(exePath), "..", "static")
			if fi2, err := os.Stat(cand); err == nil && fi2.IsDir() {
				staticDir = cand
			}
		}
	}
	if fi, err := os.Stat(staticDir); err != nil || !fi.IsDir() {
		cand := "/data/adb/modules/projectx-webui/static"
		if fi2, err := os.Stat(cand); err == nil && fi2.IsDir() {
			staticDir = cand
		}
	}
	
	fs := http.FileServer(http.Dir(staticDir))
	
	// 2. Setup API Routes Backend
	http.HandleFunc("/api/sysinfo", handleSysinfo)
	http.HandleFunc("/api/vnstat", handleVnstat)
	http.HandleFunc("/api/magisk_log", handleMagiskLog)
	http.HandleFunc("/api/reboot", handleReboot)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/bfr/settings", handleBFRSettings)
	http.HandleFunc("/api/bfr/control", handleBFRControl)
	http.HandleFunc("/api/bfr/logs", handleBFRLogs)

	http.HandleFunc("/api/check_auth", handleCheckAuth)
	http.HandleFunc("/api/auth/update", handleUpdateAuth)
	http.HandleFunc("/api/logout", handleLogout)

	// Clash Generator & Account Manager Endpoints
	http.HandleFunc("/api/clash/targets", handleClashTargets)
	http.HandleFunc("/api/clash/targets/create", handleClashTargetCreate)
	http.HandleFunc("/api/clash/accounts", handleClashAccountsList)
	http.HandleFunc("/api/clash/accounts/add", handleClashAccountsAdd)
	http.HandleFunc("/api/clash/accounts/update", handleClashAccountsUpdate)
	http.HandleFunc("/api/clash/accounts/delete", handleClashAccountsDelete)
	http.HandleFunc("/api/clash/template/apply", handleClashTemplateApply)
	http.HandleFunc("/api/clash/save", handleClashSave)

	// Magisk Modules Endpoints
	http.HandleFunc("/api/magisk/modules", handleMagiskModules)
	http.HandleFunc("/api/magisk/modules/action", handleMagiskModuleAction)

	// File Manager Endpoints
	http.HandleFunc("/api/fm/list", handleFMList)
	http.HandleFunc("/api/fm/action", handleFMAction)
	http.HandleFunc("/api/fm/upload", handleFMUpload)
	http.HandleFunc("/api/fm/read", handleFMRead)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Jika prefix adalah /api/ biarkan (akan dihandle oleh mux)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		
		// Redirect /svelte/ ke root /
		if strings.HasPrefix(r.URL.Path, "/svelte/") {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}

		// Cek apakah file ada di static directory
		path := filepath.Join(staticDir, r.URL.Path)
		if _, err := os.Stat(path); err != nil {
			// SPA Fallback: rewrite path ke root / dan serve dengan FileServer
			r.URL.Path = "/"
		}

		// Serve static file
		fs.ServeHTTP(w, r)
	})

	// SMS Viewer Endpoints
	http.HandleFunc("/api/sms/list", handleSMSList)
	http.HandleFunc("/api/sms/delete", handleSMSDelete)

	// Clients Endpoints
	http.HandleFunc("/api/clients", ApiClientsHandler)
	http.HandleFunc("/api/clients/block", ApiBlockClientHandler)
	http.HandleFunc("/api/clients/unblock", ApiUnblockClientHandler)
	http.HandleFunc("/api/clients/limit", ApiLimitClientHandler)
	http.HandleFunc("/api/clients/unlimit", ApiUnlimitClientHandler)
	http.HandleFunc("/api/clients/alias", ApiClientAliasHandler)

	// TTL & Hop Limit Endpoints
	http.HandleFunc("/api/ttl/status", handleTTLStatus)
	http.HandleFunc("/api/ttl/set", handleTTLSet)

	// Watchdog (Anti-RTO / Auto-Reconnect) Endpoints
	http.HandleFunc("/api/watchdog/status", handleWatchdogStatus)
	http.HandleFunc("/api/watchdog/config", handleWatchdogConfig)
	http.HandleFunc("/api/watchdog/test-action", handleWatchdogTestAction)
	http.HandleFunc("/api/watchdog/check-now", handleWatchdogCheckNow)

	// Web Terminal (Interactive Root Shell) Endpoints
	http.HandleFunc("/api/terminal/ws", handleTerminalWs)
	http.HandleFunc("/api/terminal/exec", handleTerminalExec)
	http.HandleFunc("/api/terminal/presets", handleTerminalPresets)

	// Cron & Scheduled Tasks Endpoints
	http.HandleFunc("/api/cron/status", handleCronStatus)
	http.HandleFunc("/api/cron/save", handleCronSave)
	http.HandleFunc("/api/cron/run_now", handleCronRunNow)
	http.HandleFunc("/api/cron/toggle_master", handleCronToggleMaster)

	// Wi-Fi Hotspot & Share Manager Endpoints
	http.HandleFunc("/api/hotspot/status", handleHotspotStatus)
	http.HandleFunc("/api/hotspot/toggle", handleHotspotToggle)
	http.HandleFunc("/api/hotspot/config", handleHotspotSaveConfig)

	// Screen Remote Endpoints
	http.HandleFunc("/api/screen/stream", handleScreenStream)
	http.HandleFunc("/api/screen/control", handleScreenControl)
	http.HandleFunc("/api/scrcpy/ws", handleScrcpyWs)

	// Speedtest Endpoints
	http.HandleFunc("/api/speedtest/stream", handleSpeedtestStream)
	http.HandleFunc("/api/speedtest/stop", handleSpeedtestStop)
	http.HandleFunc("/api/speedtest/last", handleSpeedtestLast)

	// Battery Auto Cut-off Endpoints
	http.HandleFunc("/api/battery/status", handleBatteryStatus)
	http.HandleFunc("/api/battery/config", handleBatteryConfig)



	// 4. Jalankan Web Server
	port := "80"
	fmt.Printf("🚀 Server ProjectX berjalan di http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ==================== HELPER FUNCTIONS ====================

func readFirstLine(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Jalankan perintah shell, return output (atau "" kalau error)
func sh(args ...string) string {
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Baca Android system property via getprop
func getProp(prop string) string {
	v := sh("getprop", prop)
	if v == "" || strings.ToLower(v) == "unknown" {
		return ""
	}
	return v
}

// Format bytes ke human-readable string
func fmtBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ==================== CPU USAGE TRACKER ====================

type cpuStat struct {
	total  int64
	active int64
}

var (
	prevCPUStat  cpuStat
	cpuHistory   []float64
	cpuStatMutex sync.Mutex
)

// Baca /proc/stat baris pertama "cpu ..."
func readCPUStat() (cpuStat, bool) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return cpuStat{}, false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			break
		}
		toInt := func(s string) int64 {
			v, _ := strconv.ParseInt(s, 10, 64)
			return v
		}
		user := toInt(fields[1])
		nice := toInt(fields[2])
		sys := toInt(fields[3])
		idle := toInt(fields[4])
		iowait := int64(0)
		irq := int64(0)
		soft := int64(0)
		if len(fields) > 5 {
			iowait = toInt(fields[5])
		}
		if len(fields) > 6 {
			irq = toInt(fields[6])
		}
		if len(fields) > 7 {
			soft = toInt(fields[7])
		}
		total := user + nice + sys + idle + iowait + irq + soft
		active := user + nice + sys + irq + soft
		return cpuStat{total: total, active: active}, true
	}
	return cpuStat{}, false
}

// Hitung CPU usage % dibanding sample sebelumnya
func getCPUUsage() float64 {
	cpuStatMutex.Lock()
	defer cpuStatMutex.Unlock()

	cur, ok := readCPUStat()
	if !ok {
		return 0
	}

	usage := 0.0
	diffTotal := cur.total - prevCPUStat.total
	diffActive := cur.active - prevCPUStat.active
	if prevCPUStat.total > 0 && diffTotal > 0 {
		usage = float64(diffActive) / float64(diffTotal) * 100
		if usage < 0 {
			usage = 0
		}
		if usage > 100 {
			usage = 100
		}
	}
	usage = math.Round(usage*10) / 10
	prevCPUStat = cur

	// Update history (max 8 titik)
	cpuHistory = append(cpuHistory, usage)
	if len(cpuHistory) > 8 {
		cpuHistory = cpuHistory[len(cpuHistory)-8:]
	}

	return usage
}

func getCPUHistoryCopy() []float64 {
	cpuStatMutex.Lock()
	defer cpuStatMutex.Unlock()
	if len(cpuHistory) == 0 {
		return []float64{0, 0, 0, 0, 0}
	}
	cp := make([]float64, len(cpuHistory))
	copy(cp, cpuHistory)
	return cp
}

// ==================== MEMORY ====================

func getMemInfo() (total, free, available int64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemFree:":
			free = val
		case "MemAvailable:":
			available = val
		}
	}
	if available == 0 {
		available = free
	}
	return
}

func getSwapInfo() (swapTotal, swapFree int64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "SwapTotal:":
			swapTotal = val
		case "SwapFree:":
			swapFree = val
		}
	}
	return
}

// ==================== UPTIME ====================

func getUptime() (secs int64, formatted string) {
	raw := readFirstLine("/proc/uptime")
	if raw != "" {
		parts := strings.Fields(raw)
		if len(parts) > 0 {
			if f, err := strconv.ParseFloat(parts[0], 64); err == nil {
				secs = int64(f)
			}
		}
	}
	d := secs / 86400
	h := (secs % 86400) / 3600
	m := (secs % 3600) / 60
	parts := []string{}
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if len(parts) == 0 {
		parts = append(parts, "0m")
	}
	formatted = strings.Join(parts, " ")
	return
}

// ==================== CPU INFO ====================

func getCPUCores() int {
	// Coba baca dari /sys/devices/system/cpu/present
	present := readFirstLine("/sys/devices/system/cpu/present")
	if present != "" {
		// Format: "0-7" atau "0-3"
		if strings.Contains(present, "-") {
			ps := strings.SplitN(present, "-", 2)
			if len(ps) == 2 {
				lo, _ := strconv.Atoi(ps[0])
				hi, _ := strconv.Atoi(ps[1])
				return hi - lo + 1
			}
		}
	}
	// Fallback: hitung folder cpu0, cpu1, ...
	matches, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	if len(matches) > 0 {
		return len(matches)
	}
	return 4
}

func getCPUModel() string {
	// Coba dari /proc/cpuinfo field Hardware
	file, err := os.Open("/proc/cpuinfo")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Hardware") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					model := strings.TrimSpace(parts[1])
					if model != "" && strings.ToLower(model) != "unknown" {
						return model
					}
				}
			}
		}
	}
	// Coba getprop
	for _, prop := range []string{"ro.board.platform", "ro.chipname", "ro.hardware", "ro.product.board"} {
		if v := getProp(prop); v != "" {
			return v
		}
	}
	return "Unknown CPU"
}

func getCPUFreq() string {
	for _, path := range []string{
		"/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq",
		"/sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq",
		"/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq",
	} {
		raw := readFirstLine(path)
		if raw != "" {
			if f, err := strconv.ParseInt(raw, 10, 64); err == nil && f > 0 {
				return fmt.Sprintf("%d MHz", f/1000)
			}
		}
	}
	return ""
}

func getCPUTemp() int {
	// Coba berbagai thermal zone
	for i := 0; i <= 10; i++ {
		raw := readFirstLine(fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i))
		if raw == "" {
			continue
		}
		if t, err := strconv.Atoi(raw); err == nil {
			if t > 1000 {
				return t / 1000
			}
			if t > 0 {
				return t
			}
		}
	}
	return 0
}

// ==================== BATTERY ====================

func getBattery() (level int, status string, tempC float64) {
	base := "/sys/class/power_supply/battery"
	level = 100

	// Level
	if s := readFirstLine(base + "/capacity"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			level = v
		}
	}

	// Status
	status = strings.TrimSpace(readFile(base + "/status"))
	if status == "" {
		status = "Unknown"
	}

	// Temperature (dalam 0.1°C)
	if t := readFirstLine(base + "/temp"); t != "" {
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			tempC = v / 10.0
		}
	}
	return
}

// ==================== GPU ====================

func getGPUModel() (model, gl string) {
	model = strings.TrimSpace(readFile("/sys/kernel/gpu/gpu_model"))
	if model == "" {
		model = strings.TrimSpace(readFile("/sys/class/kgsl/kgsl-3d0/gpu_model"))
	}
	if model == "" {
		// Coba dari dumpsys SurfaceFlinger
		out := sh("dumpsys", "SurfaceFlinger")
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "GLES:") {
				parts := strings.SplitN(line, ",", 3)
				if len(parts) >= 3 {
					model = strings.TrimSpace(parts[1])
					if strings.Contains(parts[2], "OpenGL ES") {
						// Extract versi
						sub := strings.TrimSpace(parts[2])
						if idx := strings.Index(sub, "OpenGL ES "); idx >= 0 {
							gl = "ES " + strings.Fields(sub[idx+10:])[0]
						}
					}
				}
				break
			}
		}
	}
	if model == "" {
		model = "Unknown GPU"
	}

	// GL version fallback
	if gl == "" {
		if raw := getProp("ro.opengles.version"); raw != "" {
			if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
				major := v >> 16
				gl = fmt.Sprintf("ES %d.0", major)
			}
		}
		if gl == "" {
			gl = "OpenGL"
		}
	}
	return
}

// ==================== DEVICE INFO ====================

func getDeviceInfo() (manuf, model, device, osVer, kernel string) {
	model = getProp("ro.product.model")
	if m := getProp("ro.product.marketname"); m != "" {
		model = m
	}
	manuf = getProp("ro.product.manufacturer")
	device = getProp("ro.product.device")
	osVer = getProp("ro.build.version.release")
	kernel = sh("uname", "-r")

	if model == "" {
		model = "Android Device"
	}
	if manuf == "" {
		manuf = "Unknown"
	}
	if osVer == "" {
		osVer = "Android"
	} else {
		// SDK mapping
		sdk, _ := strconv.Atoi(getProp("ro.build.version.sdk"))
		codeNames := map[int]string{
			19: "KitKat", 21: "Lollipop", 22: "Lollipop",
			23: "Marshmallow", 24: "Nougat", 25: "Nougat",
			26: "Oreo", 27: "Oreo", 28: "Pie",
			29: "Q", 30: "R", 31: "S", 32: "S", 33: "Tiramisu",
			34: "Upside Down Cake", 35: "Vanilla Ice Cream",
		}
		if cn, ok := codeNames[sdk]; ok {
			osVer = fmt.Sprintf("Android %s (%s)", osVer, cn)
		} else {
			osVer = "Android " + osVer
		}
	}
	if kernel == "" {
		kernel = readFirstLine("/proc/sys/kernel/osrelease")
	}
	return
}

// ==================== NETWORK ====================

type NetIface struct {
	Name    string `json:"name"`
	IP      string `json:"ip"`
	RxTotal string `json:"rx_total"`
	TxTotal string `json:"tx_total"`
	State   string `json:"state"`
	RxRaw   int64  `json:"rx_raw"`
	TxRaw   int64  `json:"tx_raw"`
}

func getNetworkInterfaces() []NetIface {
	matches, _ := filepath.Glob("/sys/class/net/*")
	var result []NetIface

	for _, path := range matches {
		iface := filepath.Base(path)
		if iface == "lo" {
			continue
		}

		state := readFile(path + "/operstate")
		rxRaw := int64(0)
		txRaw := int64(0)
		if s := readFirstLine(path + "/statistics/rx_bytes"); s != "" {
			rxRaw, _ = strconv.ParseInt(s, 10, 64)
		}
		if s := readFirstLine(path + "/statistics/tx_bytes"); s != "" {
			txRaw, _ = strconv.ParseInt(s, 10, 64)
		}

		// Cari IP
		ip := ""
		// Coba dari /proc/net/fib_trie atau ip addr
		ipOut := sh("/system/bin/ip", "-4", "addr", "show", "dev", iface)
		if ipOut == "" {
			ipOut = sh("ip", "-4", "addr", "show", "dev", iface)
		}
		if ipOut != "" {
			for _, line := range strings.Split(ipOut, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "inet ") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						addr := strings.SplitN(fields[1], "/", 2)[0]
						if !strings.HasPrefix(addr, "127.") {
							ip = addr
							break
						}
					}
				}
			}
		}

		// Hanya tampilkan interface yang aktif atau punya traffic
		isUp := state == "up" || state == "lower_up" || state == "unknown"
		hasTraffic := rxRaw+txRaw > 0
		hasIP := ip != ""

		if !hasIP && !isUp && !hasTraffic {
			continue
		}

		result = append(result, NetIface{
			Name:    iface,
			IP:      ip,
			RxTotal: fmtBytes(rxRaw),
			TxTotal: fmtBytes(txRaw),
			State:   state,
			RxRaw:   rxRaw,
			TxRaw:   txRaw,
		})
	}
	return result
}

// ==================== TELEPHONY / CARRIER INFO ====================

type TelephonyInfo struct {
	Operator   string `json:"operator"`
	NetType    string `json:"net_type"`
	SignalBars int    `json:"signal_bars"`
	Dbm        int    `json:"dbm"`
	SignalText string `json:"signal_text"`
}

var (
	telephonyCacheLock sync.Mutex
	cachedTelephony    TelephonyInfo
	lastTelephonyCheck time.Time
)

func cleanOperatorName(raw string) string {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "smartfren") {
		return "Smartfren"
	}
	if strings.Contains(lower, "telkomsel") || strings.Contains(lower, "tsel") || strings.Contains(lower, "by.u") {
		return "Telkomsel"
	}
	if strings.Contains(lower, "indosat") || strings.Contains(lower, "im3") || strings.Contains(lower, "ooredoo") {
		return "Indosat Ooredoo"
	}
	if strings.Contains(lower, "xl") || strings.Contains(lower, "axiata") {
		return "XL Axiata"
	}
	if strings.Contains(lower, "tri") || strings.Contains(lower, "three") || strings.Contains(lower, "3") {
		return "Tri (3)"
	}
	if raw != "" {
		return strings.Title(strings.ToLower(raw))
	}
	return ""
}

func getTelephonyInfo() TelephonyInfo {
	telephonyCacheLock.Lock()
	defer telephonyCacheLock.Unlock()

	if time.Since(lastTelephonyCheck) < 3*time.Second && cachedTelephony.Operator != "" {
		return cachedTelephony
	}

	info := TelephonyInfo{
		Operator:   "Tidak Ada SIM",
		NetType:    "Tidak Ada Jaringan",
		SignalBars: 0,
		Dbm:        0,
		SignalText: "Tidak Ada Sinyal",
	}

	// 1. Get Operator
	rawSim := getProp("gsm.sim.operator.alpha")
	rawAlpha := getProp("gsm.operator.alpha")
	rawNum := getProp("gsm.operator.numeric")

	var candidates []string
	for _, s := range strings.Split(rawSim, ",") {
		if t := strings.TrimSpace(s); t != "" && !strings.EqualFold(t, "unknown") {
			candidates = append(candidates, t)
		}
	}
	for _, s := range strings.Split(rawAlpha, ",") {
		if t := strings.TrimSpace(s); t != "" && !strings.EqualFold(t, "unknown") {
			candidates = append(candidates, t)
		}
	}

	opName := ""
	if len(candidates) > 0 {
		opName = candidates[0]
	}

	info.Operator = cleanOperatorName(opName)
	if info.Operator == "" {
		if strings.HasPrefix(rawNum, "51009") || strings.HasPrefix(rawNum, "51028") {
			info.Operator = "Smartfren"
		} else if strings.HasPrefix(rawNum, "51010") {
			info.Operator = "Telkomsel"
		} else if strings.HasPrefix(rawNum, "51001") || strings.HasPrefix(rawNum, "51021") {
			info.Operator = "Indosat Ooredoo"
		} else if strings.HasPrefix(rawNum, "51011") {
			info.Operator = "XL Axiata"
		} else if strings.HasPrefix(rawNum, "51089") {
			info.Operator = "Tri (3)"
		} else {
			info.Operator = "Tidak Ada SIM"
		}
	}

	// 2. Get Network Type (RAT)
	rawNet := getProp("gsm.network.type")
	for _, n := range strings.Split(rawNet, ",") {
		t := strings.TrimSpace(n)
		if t != "" && !strings.EqualFold(t, "unknown") {
			upper := strings.ToUpper(t)
			if strings.Contains(upper, "LTE") {
				info.NetType = "4G LTE"
			} else if strings.Contains(upper, "NR") {
				info.NetType = "5G NR"
			} else if strings.Contains(upper, "HSPA") || strings.Contains(upper, "UMTS") || strings.Contains(upper, "HSDPA") {
				info.NetType = "3G HSPA"
			} else if strings.Contains(upper, "GSM") || strings.Contains(upper, "EDGE") || strings.Contains(upper, "GPRS") {
				info.NetType = "2G EDGE"
			} else {
				info.NetType = upper
			}
			break
		}
	}

	// 3. Get Signal Strength via dumpsys telephony.registry
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	outDump, err := exec.CommandContext(ctx, "dumpsys", "telephony.registry").Output()
	if err == nil {
		dumpStr := string(outDump)
		lines := strings.Split(dumpStr, "\n")
		reLte := regexp.MustCompile(`mLte=CellSignalStrengthLte:\s*rssi=(-?\d+)\s+rsrp=(-?\d+)[^}]*level=(\d+)`)
		reNr := regexp.MustCompile(`mNr=CellSignalStrengthNr:\s*\{[^}]*ssRsrp\s*=\s*(-?\d+)[^}]*level\s*=\s*(\d+)`)
		reGsm := regexp.MustCompile(`mGsm=CellSignalStrengthGsm:\s*rssi=(-?\d+)[^}]*level=(\d+)`)
		reWcdma := regexp.MustCompile(`mWcdma=CellSignalStrengthWcdma:\s*ss=(-?\d+)[^}]*level=(\d+)`)

		for _, l := range lines {
			if !strings.Contains(l, "mSignalStrength=SignalStrength:{") {
				continue
			}

			// Prioritaskan NR jika ada
			if matchNr := reNr.FindStringSubmatch(l); len(matchNr) > 2 {
				rsrp, _ := strconv.Atoi(matchNr[1])
				lvl, _ := strconv.Atoi(matchNr[2])
				if rsrp < 0 && rsrp != 2147483647 {
					info.Dbm = rsrp
					info.SignalBars = lvl
					if info.NetType == "Tidak Ada Jaringan" {
						info.NetType = "5G NR"
					}
					break
				}
			}

			// Cek LTE
			if matchLte := reLte.FindStringSubmatch(l); len(matchLte) > 3 {
				rssi, _ := strconv.Atoi(matchLte[1])
				rsrp, _ := strconv.Atoi(matchLte[2])
				lvl, _ := strconv.Atoi(matchLte[3])

				if (rsrp < 0 && rsrp != 2147483647) || (rssi < 0 && rssi != 2147483647) {
					if rsrp < 0 && rsrp != 2147483647 {
						info.Dbm = rsrp
					} else {
						info.Dbm = rssi
					}
					info.SignalBars = lvl
					if info.NetType == "Tidak Ada Jaringan" {
						info.NetType = "4G LTE"
					}
					break
				}
			}

			// Cek WCDMA / 3G
			if matchWcdma := reWcdma.FindStringSubmatch(l); len(matchWcdma) > 2 {
				ss, _ := strconv.Atoi(matchWcdma[1])
				lvl, _ := strconv.Atoi(matchWcdma[2])
				if ss < 0 && ss != 2147483647 {
					info.Dbm = ss
					info.SignalBars = lvl
					if info.NetType == "Tidak Ada Jaringan" {
						info.NetType = "3G HSPA"
					}
					break
				}
			}

			// Cek GSM / 2G
			if matchGsm := reGsm.FindStringSubmatch(l); len(matchGsm) > 2 {
				rssi, _ := strconv.Atoi(matchGsm[1])
				lvl, _ := strconv.Atoi(matchGsm[2])
				if rssi < 0 && rssi != 2147483647 {
					info.Dbm = rssi
					info.SignalBars = lvl
					if info.NetType == "Tidak Ada Jaringan" {
						info.NetType = "2G EDGE"
					}
					break
				}
			}
		}
	}

	// 4. Format SignalText
	if info.NetType != "Tidak Ada Jaringan" {
		if info.Dbm != 0 && info.SignalBars > 0 {
			info.SignalText = fmt.Sprintf("%s · %d/4 Bar (%d dBm)", info.NetType, info.SignalBars, info.Dbm)
		} else if info.SignalBars > 0 {
			info.SignalText = fmt.Sprintf("%s · %d/4 Bar", info.NetType, info.SignalBars)
		} else if info.Dbm != 0 {
			info.SignalText = fmt.Sprintf("%s (%d dBm)", info.NetType, info.Dbm)
		} else {
			info.SignalText = info.NetType
		}
	}

	cachedTelephony = info
	lastTelephonyCheck = time.Now()
	return info
}

// ==================== CHECK ROOT ====================

func checkRoot() string {
	// Cek Magisk
	if _, err := os.Stat("/data/adb/magisk"); err == nil {
		return "Magisk (Rooted)"
	}
	if _, err := os.Stat("/sbin/.magisk"); err == nil {
		return "Magisk (Rooted)"
	}
	// Cek KernelSU
	if _, err := os.Stat("/data/adb/ksud"); err == nil {
		return "KernelSU (Rooted)"
	}
	// Cek su binary
	suPaths := []string{"/system/bin/su", "/system/xbin/su", "/su/bin/su"}
	for _, p := range suPaths {
		if _, err := os.Stat(p); err == nil {
			return "Rooted (su)"
		}
	}
	return "Not Rooted"
}

// ==================== MAGISK LOG ====================

func handleMagiskLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	content, err := os.ReadFile("/cache/magisk.log")
	if err != nil {
		// Coba path alternatif
		content, err = os.ReadFile("/data/adb/magisk.log")
	}

	if err != nil {
		w.Write([]byte(`{"status": "error", "message": "Log Magisk tidak ditemukan (atau bukan root Magisk)."}`))
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"log":    string(content),
	})
}

// ==================== REBOOT ====================

func handleReboot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	w.Write([]byte(`{"status": "success", "message": "Mengeksekusi reboot..."}`))

	// Run the reboot command asynchronously so we can return the response first
	go func() {
		exec.Command("su", "-c", "reboot").Run()
	}()
}

// ==================== MAIN HANDLER ====================

func handleSysinfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// A. Device & OS Info
	manuf, model, device, osVer, kernel := getDeviceInfo()
	deviceDisplay := manuf + " " + model
	if manuf == "Unknown" || manuf == "" {
		deviceDisplay = model
	}

	// B. Uptime
	_, uptimeStr := getUptime()

	// C. CPU
	cpuUsage := getCPUUsage()
	cpuCores := getCPUCores()
	cpuModel := getCPUModel()
	cpuFreq := getCPUFreq()
	cpuTemp := getCPUTemp()
	cpuHist := getCPUHistoryCopy()

	// D. RAM
	memTotal, _, memAvail := getMemInfo()
	memUsed := memTotal - memAvail
	if memTotal == 0 {
		memTotal = 1
	}
	memPct := int(float64(memUsed) / float64(memTotal) * 100)
	swTotal, swFree := getSwapInfo()
	swUsed := swTotal - swFree
	swPct := 0
	if swTotal > 0 {
		swPct = int(float64(swUsed) / float64(swTotal) * 100)
	}

	// E. Battery
	batLevel, batStatus, batTemp := getBattery()

	// F. GPU
	gpuModel, gpuGL := getGPUModel()
	gpuFreq := ""
	for _, p := range []string{
		"/sys/class/devfreq/gpufreq/cur_freq",
		"/sys/class/kgsl/kgsl-3d0/gpuclk",
	} {
		if raw := readFirstLine(p); raw != "" {
			if f, err := strconv.ParseInt(raw, 10, 64); err == nil && f > 0 {
				if f > 1000000 {
					gpuFreq = fmt.Sprintf("%d MHz", f/1000000)
				} else if f > 1000 {
					gpuFreq = fmt.Sprintf("%d MHz", f/1000)
				}
				break
			}
		}
	}

	// G. Network
	netList := getNetworkInterfaces()

	// H. Root
	rootStatus := checkRoot()

	// I. Telephony (Carrier & Signal)
	telInfo := getTelephonyInfo()

	// Kirim JSON
	data := map[string]interface{}{
		"sys": map[string]interface{}{
			"device":      deviceDisplay,
			"model":       device,
			"uptime":      uptimeStr,
			"os":          osVer,
			"kernel":      kernel,
			"root":        rootStatus,
			"provider":    telInfo.Operator,
			"signal":      telInfo.SignalText,
			"signal_bars": telInfo.SignalBars,
			"ts":          time.Now().Unix(),
		},
		"cpu": map[string]interface{}{
			"usage":   cpuUsage,
			"cores":   cpuCores,
			"model":   cpuModel,
			"freq":    cpuFreq,
			"temp":    cpuTemp,
			"history": cpuHist,
		},
		"ram": map[string]interface{}{
			"raw_lvl":  memPct,
			"used":     fmtBytes(memUsed * 1024),
			"total":    fmtBytes(memTotal * 1024),
			"swap_raw": swPct,
			"swap_txt": fmtBytes(swUsed*1024) + " / " + fmtBytes(swTotal*1024),
		},
		"bat": map[string]interface{}{
			"raw_lvl":             batLevel,
			"status":              batStatus,
			"display_status":      func() string {
				if isBatteryCutoffActive() {
					return "Cut-off (Bypass)"
				}
				return batStatus
			}(),
			"temp":                fmt.Sprintf("%.1f", batTemp),
			"is_cutoff":           isBatteryCutoffActive(),
			"auto_cutoff_enabled": isBatteryProtectionEnabled(),
			"start_level":         getBatteryConfig().StartLevel,
			"stop_level":          getBatteryConfig().StopLevel,
		},
		"gpu": map[string]interface{}{
			"model": gpuModel,
			"gl":    gpuGL,
			"freq":  gpuFreq,
		},
		"net": map[string]interface{}{
			"list": netList,
		},
	}
	json.NewEncoder(w).Encode(data)
}

// ==================== AUTH HANDLERS ====================

type AuthCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func getCredentials() AuthCreds {
	// Default credentials
	creds := AuthCreds{Username: "admin", Password: "admin"}
	
	file, err := os.Open("auth.json")
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&creds)
	}
	return creds
}

func saveCredentials(creds AuthCreds) error {
	file, err := os.Create("auth.json")
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(creds)
}

func handleCheckAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.Write([]byte(`{"status": "error"}`))
		return
	}
	w.Write([]byte(`{"status": "success"}`))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req AuthCreds

	err := json.NewDecoder(r.Body).Decode(&req)
	creds := getCredentials()

	if err == nil && req.Username == creds.Username && req.Password == creds.Password {
		http.SetCookie(w, &http.Cookie{
			Name:     "projectx_session",
			Value:    "loggedin",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   86400,
		})
		w.Write([]byte(`{"status": "success", "message": "Login berhasil"}`))
		return
	}

	w.Write([]byte(`{"status": "error", "message": "Username atau password salah!"}`))
}

func handleUpdateAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	var req struct {
		OldUser string `json:"old_user"`
		OldPass string `json:"old_pass"`
		NewUser string `json:"new_user"`
		NewPass string `json:"new_pass"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Write([]byte(`{"status": "error", "message": "Invalid request"}`))
		return
	}

	creds := getCredentials()
	if req.OldUser != creds.Username || req.OldPass != creds.Password {
		w.Write([]byte(`{"status": "error", "message": "Username atau Password saat ini salah!"}`))
		return
	}

	if req.NewUser == "" && req.NewPass == "" {
		w.Write([]byte(`{"status": "error", "message": "Tidak ada data baru yang diubah!"}`))
		return
	}

	if req.NewUser != "" {
		creds.Username = req.NewUser
	}
	if req.NewPass != "" {
		creds.Password = req.NewPass
	}

	if err := saveCredentials(creds); err != nil {
		w.Write([]byte(`{"status": "error", "message": "Gagal menyimpan kredensial"}`))
		return
	}

	w.Write([]byte(`{"status": "success", "message": "Kredensial berhasil diubah! Silakan login kembali."}`))
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "projectx_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login.html", http.StatusFound)
}

func isAuthorized(r *http.Request) bool {
	cookie, err := r.Cookie("projectx_session")
	if err != nil || cookie.Value != "loggedin" {
		return false
	}
	return true
}

// ==================== SCREEN REMOTE (MJPEG) ====================

func handleScreenStream(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Baca parameter quality (default 75)
	quality := 75
	if q := r.URL.Query().Get("q"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v >= 10 && v <= 100 {
			quality = v
		}
	}

	mimeWriter := multipart.NewWriter(w)
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", mimeWriter.Boundary()))
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := r.Context()
	flusher, canFlush := w.(http.Flusher)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Capture layar sebagai PNG
			pngData, err := exec.Command("screencap", "-p").Output()
			if err != nil || len(pngData) < 100 {
				// Fallback: coba dengan su
				pngData, err = exec.Command("su", "-c", "screencap -p").Output()
			}

			if err == nil && len(pngData) > 100 {
				// Decode PNG
				img, decErr := png.Decode(bytes.NewReader(pngData))
				if decErr == nil {
					// Encode ke JPEG (jauh lebih kecil = lebih cepat)
					var jpegBuf bytes.Buffer
					encErr := jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: quality})
					if encErr == nil {
						partHeader := make(map[string][]string)
						partHeader["Content-Type"] = []string{"image/jpeg"}
						partWriter, pErr := mimeWriter.CreatePart(partHeader)
						if pErr == nil {
							partWriter.Write(jpegBuf.Bytes())
							if canFlush {
								flusher.Flush()
							}
						}
					}
				}
			}

			// ~8 FPS — cukup smooth untuk remote control
			time.Sleep(120 * time.Millisecond)
		}
	}
}

func handleScreenControl(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodPost {
		var data struct {
			Action   string `json:"action"` // "tap" | "swipe" | "key"
			X        int    `json:"x"`
			Y        int    `json:"y"`
			X1       int    `json:"x1"`
			Y1       int    `json:"y1"`
			X2       int    `json:"x2"`
			Y2       int    `json:"y2"`
			Duration int    `json:"duration"` // ms, untuk swipe
			Keycode  int    `json:"keycode"`   // Android keycode
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err == nil {
			switch data.Action {
			case "tap":
				exec.Command("su", "-c", fmt.Sprintf("input tap %d %d", data.X, data.Y)).Run()
			case "swipe":
				dur := data.Duration
				if dur <= 0 {
					dur = 300
				}
				exec.Command("su", "-c", fmt.Sprintf("input swipe %d %d %d %d %d",
					data.X1, data.Y1, data.X2, data.Y2, dur)).Run()
			case "key":
				if data.Keycode > 0 {
					exec.Command("su", "-c", fmt.Sprintf("input keyevent %d", data.Keycode)).Run()
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
