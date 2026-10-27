package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BatteryConfig menyimpan pengaturan auto cut-off charger
type BatteryConfig struct {
	Enabled    bool `json:"enabled"`     // Apakah auto cut-off diaktifkan
	StartLevel int  `json:"start_level"` // Mulai cas kembali jika baterai <= level ini (default: 60)
	StopLevel  int  `json:"stop_level"`  // Hentikan cas jika baterai >= level ini (default: 80)
}

// BatteryStatusResponse respon JSON lengkap untuk UI
type BatteryStatusResponse struct {
	Level          int           `json:"level"`
	Status         string        `json:"status"`         // "Charging", "Discharging", "Not charging", "Full"
	DisplayStatus  string        `json:"display_status"` // "Charging", "Cut-off (Bypass)", "Discharging"
	TempC          float64       `json:"temp_c"`
	Health         string        `json:"health"`
	VoltageMv      int           `json:"voltage_mv"`
	CurrentMa      int           `json:"current_ma"`
	IsCutoff       bool          `json:"is_cutoff"`
	ControlMethod  string        `json:"control_method"`
	Config         BatteryConfig `json:"config"`
	Message        string        `json:"message,omitempty"`
}

type BatterySwitch struct {
	ID         string
	Name       string
	Path       string
	ChargeVal  string
	CutoffVal  string
	IsInverted bool
}

var (
	batteryMu      sync.RWMutex
	batteryCfg     BatteryConfig
	batteryCutoff  bool
	batteryMethod  string
	batteryCfgPath string
	activeSwitch   *BatterySwitch
)

// Inisialisasi Battery Controller saat server start
func initBatteryController() {
	// 1. Tentukan path file konfigurasi
	candDirs := []string{
		"./data",
		"/data/adb/modules/projectx-webui/data",
		"/data/adb/projectx",
	}
	for _, dir := range candDirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			batteryCfgPath = filepath.Join(dir, "battery_config.json")
			break
		}
	}
	if batteryCfgPath == "" {
		_ = os.MkdirAll("./data", 0755)
		batteryCfgPath = "./data/battery_config.json"
	}

	// 2. Default Config
	batteryCfg = BatteryConfig{
		Enabled:    false,
		StartLevel: 60,
		StopLevel:  80,
	}

	// 3. Load config dari disk jika ada
	if data, err := os.ReadFile(batteryCfgPath); err == nil {
		var loaded BatteryConfig
		if err := json.Unmarshal(data, &loaded); err == nil {
			// Validasi nilai tersimpan
			if loaded.StartLevel >= 10 && loaded.StartLevel <= 90 &&
				loaded.StopLevel >= 20 && loaded.StopLevel <= 100 &&
				loaded.StopLevel > loaded.StartLevel {
				batteryCfg = loaded
			} else {
				batteryCfg.Enabled = loaded.Enabled
			}
		}
	}

	// 4. Deteksi metode kontrol hardware charger
	activeSwitch = detectBatteryControlMethod()
	if activeSwitch != nil {
		batteryMethod = activeSwitch.ID
	} else {
		batteryMethod = "unsupported"
	}
	methodName := "Tidak Didukung"
	if activeSwitch != nil {
		methodName = activeSwitch.Name
	}
	fmt.Printf("[BatteryController] Metode kontrol terdeteksi: %s [%s] (Auto Cut-off: %v, Range: %d%% - %d%%)\n",
		batteryMethod, methodName, batteryCfg.Enabled, batteryCfg.StartLevel, batteryCfg.StopLevel)

	// 5. Jalankan evaluasi awal dan start background loop
	evaluateBatteryProtection()
	startBatteryMonitorLoop()
}

// Deteksi hardware atau sysfs yang tersedia secara universal (Samsung, MTK, Qualcomm, Oppo, dll)
func detectBatteryControlMethod() *BatterySwitch {
	// Prioritas 1: Hardware I2C SMB358 (Xiaomi Redmi 4A / Rolex / MSM8917 / MSM8937)
	if _, err := os.Stat("/sys/bus/i2c/drivers/smb358-charger-rolex"); err == nil {
		return &BatterySwitch{
			ID:   "smb358_i2c",
			Name: "Hardware I2C (SMB358 - Redmi 4A)",
		}
	}
	if _, err := os.Stat("/sys/devices/platform/soc/7af5000.i2c/i2c-5/5-0057"); err == nil {
		return &BatterySwitch{
			ID:   "smb358_i2c",
			Name: "Hardware I2C (SMB358 - Redmi 4A)",
		}
	}
	if _, err := os.Stat("/dev/i2c-5"); err == nil {
		bin := getI2CGetBin()
		out, err := exec.Command(bin, "-fy", "5", "0x57", "0x30").Output()
		if err == nil && strings.HasPrefix(strings.TrimSpace(string(out)), "0x") {
			return &BatterySwitch{
				ID:   "smb358_i2c",
				Name: "Hardware I2C (SMB358 - Redmi 4A)",
			}
		}
	}

	// Prioritas 2: Universal Sysfs Candidate List
	candidates := []BatterySwitch{
		// --- SAMSUNG (Exynos & Snapdragon: J7 Pro, Note, S-series, A-series) ---
		{
			ID:         "samsung_batt_slate",
			Name:       "Samsung (batt_slate_mode)",
			Path:       "/sys/class/power_supply/battery/batt_slate_mode",
			ChargeVal:  "0\n",
			CutoffVal:  "1\n",
			IsInverted: true,
		},
		{
			ID:         "samsung_store_mode",
			Name:       "Samsung (store_mode)",
			Path:       "/sys/class/power_supply/battery/store_mode",
			ChargeVal:  "0\n",
			CutoffVal:  "1\n",
			IsInverted: true,
		},
		{
			ID:         "samsung_siop_level",
			Name:       "Samsung (siop_level)",
			Path:       "/sys/class/power_supply/battery/siop_level",
			ChargeVal:  "100\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},

		// --- QUALCOMM / AOSP / XIAOMI / PIXEL / SONY / MOTOROLA ---
		{
			ID:         "sysfs_charging_enabled",
			Name:       "Kernel Sysfs (charging_enabled)",
			Path:       "/sys/class/power_supply/battery/charging_enabled",
			ChargeVal:  "1\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},
		{
			ID:         "sysfs_battery_charging_enabled",
			Name:       "Kernel Sysfs (battery_charging_enabled)",
			Path:       "/sys/class/power_supply/battery/battery_charging_enabled",
			ChargeVal:  "1\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},
		{
			ID:         "sysfs_main_charging_enabled",
			Name:       "Kernel Sysfs (main/charging_enabled)",
			Path:       "/sys/class/power_supply/main/charging_enabled",
			ChargeVal:  "1\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},

		// --- MEDIATEK (Infinix, Tecno, Poco/Redmi MTK, Realme MTK) ---
		{
			ID:         "mtk_platform_charger",
			Name:       "MediaTek (charger/charging_enabled)",
			Path:       "/sys/devices/platform/charger/charging_enabled",
			ChargeVal:  "1\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},
		{
			ID:         "mtk_mt_battery",
			Name:       "MediaTek (mt-battery/Charging_Enable)",
			Path:       "/sys/devices/platform/mt-battery/Charging_Enable",
			ChargeVal:  "1\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},
		{
			ID:         "mtk_device_charging",
			Name:       "MediaTek (battery/device/Charging_Enable)",
			Path:       "/sys/class/power_supply/battery/device/Charging_Enable",
			ChargeVal:  "1\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},

		// --- OPPO / REALME / ONEPLUS (VOOC / SuperVOOC) ---
		{
			ID:         "oppo_input_suspend",
			Name:       "Input Suspend (input_suspend)",
			Path:       "/sys/class/power_supply/battery/input_suspend",
			ChargeVal:  "0\n",
			CutoffVal:  "1\n",
			IsInverted: true,
		},
		{
			ID:         "oppo_mmi_charging",
			Name:       "MMI (mmi_charging_enable)",
			Path:       "/sys/class/power_supply/battery/mmi_charging_enable",
			ChargeVal:  "1\n",
			CutoffVal:  "0\n",
			IsInverted: false,
		},
	}

	for _, cand := range candidates {
		if _, err := os.Stat(cand.Path); err == nil {
			c := cand
			return &c
		}
		// Fallback cek dengan su cat / test jika os.Stat ditolak SELinux
		out, err := exec.Command("su", "-c", fmt.Sprintf("[ -f %s ] && echo 1", cand.Path)).Output()
		if err == nil && strings.TrimSpace(string(out)) == "1" {
			c := cand
			return &c
		}
	}

	return nil
}

func getI2CSetBin() string {
	if bin, err := exec.LookPath("i2cset"); err == nil {
		return bin
	}
	if _, err := os.Stat("/system/bin/i2cset"); err == nil {
		return "/system/bin/i2cset"
	}
	return "i2cset"
}

func getI2CGetBin() string {
	if bin, err := exec.LookPath("i2cget"); err == nil {
		return bin
	}
	if _, err := os.Stat("/system/bin/i2cget"); err == nil {
		return "/system/bin/i2cget"
	}
	return "i2cget"
}

func writeSysfsRoot(path string, val string) error {
	cleanVal := strings.TrimSpace(val)
	err := os.WriteFile(path, []byte(val), 0644)
	if err == nil {
		return nil
	}
	// Fallback via root su dengan chmod 666 untuk mengatasi SELinux Enforcing Samsung / Xiaomi
	cmdStr := fmt.Sprintf("chmod 666 %s 2>/dev/null; echo %s > %s", path, cleanVal, path)
	out, suErr := exec.Command("su", "-c", cmdStr).CombinedOutput()
	if suErr != nil {
		return fmt.Errorf("write error: %v (su error: %v, out: %s)", err, suErr, strings.TrimSpace(string(out)))
	}
	return nil
}

func readSysfsRoot(path string) string {
	if data, err := os.ReadFile(path); err == nil {
		return strings.TrimSpace(string(data))
	}
	out, err := exec.Command("su", "-c", "cat "+path).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// Terapkan charging on / off ke hardware charger
func applyHardwareCharging(charging bool) error {
	if activeSwitch == nil {
		return fmt.Errorf("charging control unsupported on this device")
	}

	if activeSwitch.ID == "smb358_i2c" {
		targetVal := "0x82" // Bit 1 = 1 (CMD_A_CHG_ENABLED)
		if !charging {
			targetVal = "0x80" // Bit 1 = 0 (CMD_A_CHG_DISABLED)
		}
		bin := getI2CSetBin()
		out, err := exec.Command(bin, "-fy", "5", "0x57", "0x30", targetVal, "b").CombinedOutput()
		if err != nil {
			fmt.Printf("[BatteryController] ❌ i2cset failed: %v (%s)\n", err, strings.TrimSpace(string(out)))
			return fmt.Errorf("i2cset failed: %v (%s)", err, strings.TrimSpace(string(out)))
		}
		fmt.Printf("[BatteryController] ✅ Hardware charging set to: %s (charging=%v)\n", targetVal, charging)
		return nil
	}

	// Untuk semua switch sysfs (Samsung, MTK, Qualcomm, Oppo, dll)
	valToWrite := activeSwitch.ChargeVal
	if !charging {
		valToWrite = activeSwitch.CutoffVal
	}

	err := writeSysfsRoot(activeSwitch.Path, valToWrite)
	if err != nil {
		fmt.Printf("[BatteryController] ❌ Gagal menulis ke %s (%s): %v\n", activeSwitch.Path, activeSwitch.Name, err)
		return err
	}
	fmt.Printf("[BatteryController] ✅ Berhasil menerapkan charging=%v ke %s (val=%s)\n", charging, activeSwitch.Path, strings.TrimSpace(valToWrite))
	return nil
}

// Cek apakah charger hardware saat ini sedang dalam posisi disable/cutoff
func isHardwareCutoffActive() bool {
	if activeSwitch == nil {
		return false
	}
	if activeSwitch.ID == "smb358_i2c" {
		bin := getI2CGetBin()
		out, err := exec.Command(bin, "-fy", "5", "0x57", "0x30").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			if strings.HasPrefix(s, "0x") {
				if v, err := strconv.ParseUint(s[2:], 16, 8); err == nil {
					// Bit 1 (0x02) adalah charge enabled. Jika 0, berarti cut-off aktif!
					return (v & 0x02) == 0
				}
			}
		}
	}

	val := readSysfsRoot(activeSwitch.Path)
	targetCutoff := strings.TrimSpace(activeSwitch.CutoffVal)
	return val == targetCutoff
}

// Background monitor loop yang mengevaluasi persentase baterai
func startBatteryMonitorLoop() {
	go func() {
		ticker := time.NewTicker(6 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			evaluateBatteryProtection()
		}
	}()
}

// Evaluasi proteksi baterai dengan algoritma Hysteresis
func evaluateBatteryProtection() {
	batteryMu.Lock()
	defer batteryMu.Unlock()

	level, _, _ := getBattery()
	if level <= 0 {
		return
	}

	// 1. Jika fitur dinonaktifkan
	if !batteryCfg.Enabled {
		if batteryCutoff {
			_ = applyHardwareCharging(true)
			batteryCutoff = false
		}
		return
	}

	// 2. Jika fitur aktif:
	// A. Baterai mencapai atau melampaui batas maksimal -> STOP CAS (Cut-off)
	if level >= batteryCfg.StopLevel {
		if !batteryCutoff {
			_ = applyHardwareCharging(false)
			batteryCutoff = true
		} else {
			// Pastikan hardware tetap dalam status cut-off (jika ada event driver/kabel yang meresetnya)
			if !isHardwareCutoffActive() {
				_ = applyHardwareCharging(false)
			}
		}
	} else if level <= batteryCfg.StartLevel {
		// B. Baterai turun ke atau di bawah batas minimal -> MULAI CAS KEMBALI
		if batteryCutoff {
			_ = applyHardwareCharging(true)
			batteryCutoff = false
		}
	} else {
		// C. Berada di rentang tengah (misal 61% - 79%):
		// Pertahankan status saat ini (Hysteresis):
		// Jika sedang cut-off, tetap cut-off. Jika sedang cas, tetap cas sampai stop level.
		if batteryCutoff {
			if !isHardwareCutoffActive() {
				_ = applyHardwareCharging(false)
			}
		}
	}
}

// Simpan config ke disk
func saveBatteryConfig(cfg BatteryConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(batteryCfgPath, data, 0644)
}

// Helper untuk sysinfo
func isBatteryCutoffActive() bool {
	batteryMu.RLock()
	defer batteryMu.RUnlock()
	return batteryCutoff
}

func isBatteryProtectionEnabled() bool {
	batteryMu.RLock()
	defer batteryMu.RUnlock()
	return batteryCfg.Enabled
}

func getBatteryConfig() BatteryConfig {
	batteryMu.RLock()
	defer batteryMu.RUnlock()
	return batteryCfg
}

// Ambil status baterai detail
func getDetailedBatteryStatus() BatteryStatusResponse {
	batteryMu.RLock()
	cfg := batteryCfg
	isCutoff := batteryCutoff
	batteryMu.RUnlock()

	base := "/sys/class/power_supply/battery"
	level, rawStatus, tempC := getBattery()

	health := strings.TrimSpace(readFile(base + "/health"))
	if health == "" {
		health = "Good"
	}

	voltageMv := 0
	if vStr := readFirstLine(base + "/voltage_now"); vStr != "" {
		if v, err := strconv.Atoi(vStr); err == nil {
			if v > 100000 {
				voltageMv = v / 1000
			} else {
				voltageMv = v
			}
		}
	}

	currentMa := 0
	if cStr := readFirstLine(base + "/current_now"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil {
			if c > 10000 || c < -10000 {
				currentMa = c / 1000
			} else {
				currentMa = c
			}
		}
	}

	// Tentukan display status yang ramah bagi user
	displayStatus := rawStatus
	if isCutoff {
		displayStatus = "Cut-off (Bypass)"
	} else if rawStatus == "Charging" {
		displayStatus = "Charging"
	} else if rawStatus == "Discharging" {
		displayStatus = "Discharging"
	}

	methodDisplay := "Tidak Didukung"
	if activeSwitch != nil {
		methodDisplay = activeSwitch.Name
	}

	return BatteryStatusResponse{
		Level:         level,
		Status:        rawStatus,
		DisplayStatus: displayStatus,
		TempC:         tempC,
		Health:        health,
		VoltageMv:     voltageMv,
		CurrentMa:     currentMa,
		IsCutoff:      isCutoff,
		ControlMethod: methodDisplay,
		Config:        cfg,
	}
}

// ==================== HTTP HANDLERS ====================

// GET /api/battery/status
func handleBatteryStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := getDetailedBatteryStatus()
	json.NewEncoder(w).Encode(resp)
}

// POST /api/battery/config
func handleBatteryConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"status":"error","message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Enabled    bool `json:"enabled"`
		StartLevel int  `json:"start_level"`
		StopLevel  int  `json:"stop_level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Format JSON tidak valid",
		})
		return
	}

	// Validasi nilai
	if req.StartLevel < 10 || req.StartLevel > 90 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Start level harus antara 10% dan 90%",
		})
		return
	}

	if req.StopLevel < 20 || req.StopLevel > 100 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Stop level harus antara 20% dan 100%",
		})
		return
	}

	if req.StopLevel-req.StartLevel < 5 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Selisih Stop level dan Start level minimal 5% untuk mencegah siklus switching cepat",
		})
		return
	}

	// Simpan ke memory
	newCfg := BatteryConfig{
		Enabled:    req.Enabled,
		StartLevel: req.StartLevel,
		StopLevel:  req.StopLevel,
	}

	batteryMu.Lock()
	batteryCfg = newCfg
	batteryMu.Unlock()

	// Simpan ke disk
	_ = saveBatteryConfig(newCfg)

	// Evaluasi langsung sekarang
	evaluateBatteryProtection()

	resp := getDetailedBatteryStatus()
	resp.Message = "Pengaturan proteksi baterai berhasil disimpan"
	json.NewEncoder(w).Encode(resp)
}
