package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type HotspotConfig struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
	Security string `json:"security"` // "wpa2", "open"
	Band     string `json:"band"`     // "2.4", "5", "any"
	Hidden   bool   `json:"hidden"`
}

type HotspotStatus struct {
	Enabled    bool          `json:"enabled"`
	Interface  string        `json:"interface"`
	IPAddress  string        `json:"ip_address"`
	Config     HotspotConfig `json:"config"`
	QRPayload  string        `json:"qr_payload"`
	LastUpdate string        `json:"last_update"`
}

var (
	hotspotMu sync.Mutex
)

func getHotspotLocalConfigPath() string {
	candDirs := []string{
		"/data/adb/modules/projectx-webui/data",
		"/data/adb/projectx",
		"./data",
		".",
	}
	for _, dir := range candDirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return filepath.Join(dir, "hotspot_config.json")
		}
	}
	return "./hotspot_config.json"
}

func saveHotspotLocal(cfg HotspotConfig) {
	p := getHotspotLocalConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err == nil {
		_ = os.WriteFile(p, data, 0644)
	}
}

func readHotspotLocal() (HotspotConfig, bool) {
	p := getHotspotLocalConfigPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return HotspotConfig{}, false
	}
	var cfg HotspotConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return HotspotConfig{}, false
	}
	return cfg, true
}

// getSoftApXmlPaths mengembalikan kandidat lokasi file konfigurasi SoftAP di Android
func getSoftApXmlPaths() []string {
	return []string{
		"/data/misc/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/wifi/softap.conf",
	}
}

// readSoftApXml membaca SSID, password, band, dan hidden dari XML konfigurasi Android
func readSoftApXml() (HotspotConfig, string) {
	cfg := HotspotConfig{
		SSID:     "ProjectX_WiFi",
		Password: "",
		Security: "wpa2",
		Band:     "2.4",
		Hidden:   false,
	}
	foundPath := ""

	for _, path := range getSoftApXmlPaths() {
		data, err := exec.Command("su", "-c", "cat "+path).Output()
		if err != nil || len(data) == 0 {
			continue
		}
		foundPath = path
		content := string(data)

		// Regex SSID: <string name="WifiSsid">&quot;NamaSSID&quot;</string>
		reSSID := regexp.MustCompile(`<string name="WifiSsid">(?:&quot;|")?(.*?)(?:&quot;|")?</string>`)
		if m := reSSID.FindStringSubmatch(content); len(m) > 1 {
			cfg.SSID = strings.TrimSpace(m[1])
		}

		// Regex Passphrase: <string name="Passphrase">(.*?)</string>
		rePass := regexp.MustCompile(`<string name="Passphrase">(.*?)</string>`)
		if m := rePass.FindStringSubmatch(content); len(m) > 1 {
			cfg.Password = strings.TrimSpace(m[1])
		}

		// Regex HiddenSSID: <boolean name="HiddenSSID" value="(.*?)" />
		reHidden := regexp.MustCompile(`<boolean name="HiddenSSID" value="(true|false)"`)
		if m := reHidden.FindStringSubmatch(content); len(m) > 1 {
			cfg.Hidden = m[1] == "true"
		}

		// Regex Band: <BandChannel><int name="Band" value="(1|2)" />
		reBand := regexp.MustCompile(`<int name="Band" value="(\d+)"`)
		if m := reBand.FindStringSubmatch(content); len(m) > 1 {
			if m[1] == "2" {
				cfg.Band = "5"
			} else {
				cfg.Band = "2.4"
			}
		}

		// Security type: <int name="SecurityType" value="(0|1|4)" />
		reSec := regexp.MustCompile(`<int name="SecurityType" value="(\d+)"`)
		if m := reSec.FindStringSubmatch(content); len(m) > 1 {
			if m[1] == "0" {
				cfg.Security = "open"
			} else {
				cfg.Security = "wpa2"
			}
		}

		break
	}

	// Jika ada lokal config yang tersimpan, prioritaskan jika XML belum terupdate
	if local, ok := readHotspotLocal(); ok && local.SSID != "" {
		if cfg.SSID == "" || cfg.SSID == "ProjectX_WiFi" {
			cfg = local
		}
	}

	return cfg, foundPath
}

// saveSoftApXml memperbarui konfigurasi XML Android secara atomic via base64
func saveSoftApXml(cfg HotspotConfig, xmlPath string) error {
	if xmlPath == "" {
		return fmt.Errorf("XML config path not found")
	}

	data, err := exec.Command("su", "-c", "cat "+xmlPath).Output()
	if err != nil || len(data) == 0 {
		return fmt.Errorf("gagal membaca %s: %v", xmlPath, err)
	}

	content := string(data)

	// 1. Update WifiSsid
	cleanSSID := strings.ReplaceAll(cfg.SSID, `"`, "")
	escapedSSID := fmt.Sprintf("&quot;%s&quot;", cleanSSID)
	reSSID := regexp.MustCompile(`<string name="WifiSsid">.*?</string>`)
	content = reSSID.ReplaceAllString(content, fmt.Sprintf(`<string name="WifiSsid">%s</string>`, escapedSSID))

	// 2. Update Passphrase
	rePass := regexp.MustCompile(`<string name="Passphrase">.*?</string>`)
	content = rePass.ReplaceAllString(content, fmt.Sprintf(`<string name="Passphrase">%s</string>`, cfg.Password))

	// 3. Update HiddenSSID
	hiddenVal := "false"
	if cfg.Hidden {
		hiddenVal = "true"
	}
	reHidden := regexp.MustCompile(`<boolean name="HiddenSSID" value="(?:true|false)" />`)
	content = reHidden.ReplaceAllString(content, fmt.Sprintf(`<boolean name="HiddenSSID" value="%s" />`, hiddenVal))

	// 4. Update Band
	bandInt := "1"
	if cfg.Band == "5" {
		bandInt = "2"
	}
	reBand := regexp.MustCompile(`<int name="Band" value="\d+" />`)
	content = reBand.ReplaceAllString(content, fmt.Sprintf(`<int name="Band" value="%s" />`, bandInt))

	// 5. Update SecurityType (0 = open, 1 = wpa2)
	secVal := "1"
	if cfg.Password == "" || cfg.Security == "open" {
		secVal = "0"
	}
	reSec := regexp.MustCompile(`<int name="SecurityType" value="\d+" />`)
	content = reSec.ReplaceAllString(content, fmt.Sprintf(`<int name="SecurityType" value="%s" />`, secVal))

	// Tulis kembali file via root menggunakan base64 (bebas dari masalah tanda kutip / shell escaping)
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > %s && chown system:system %s && chmod 600 %s", b64, xmlPath, xmlPath, xmlPath)
	out, err := exec.Command("su", "-c", writeCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gagal menulis ke %s: %v, output: %s", xmlPath, err, string(out))
	}

	saveHotspotLocal(cfg)
	return nil
}

// runCmdWifiStartSoftAp menjalankan cmd wifi start-softap dengan argumen aman
func runCmdWifiStartSoftAp(ssid, password, security, band string) (string, error) {
	secType := "wpa2"
	passVal := password
	if passVal == "" || security == "open" {
		secType = "open"
		passVal = "none"
	}
	bandFlag := "2"
	if band == "5" {
		bandFlag = "5"
	}

	// Selalu hentikan softap yang sedang berjalan terlebih dahulu
	_ = exec.Command("su", "-c", "cmd wifi stop-softap").Run()
	time.Sleep(600 * time.Millisecond)

	// Jalankan perintah start-softap
	cleanSSID := strings.ReplaceAll(ssid, "'", "")
	cleanPass := strings.ReplaceAll(passVal, "'", "")

	cmdStr := fmt.Sprintf("cmd wifi start-softap '%s' %s '%s' -b %s", cleanSSID, secType, cleanPass, bandFlag)
	out, err := exec.Command("su", "-c", cmdStr).CombinedOutput()
	return string(out), err
}

// detectActiveHotspot mendeteksi apakah SoftAP sedang aktif dan mengembalikan nama interface serta IP
func detectActiveHotspot() (bool, string, string) {
	out, err := exec.Command("su", "-c", "ip -o addr show").Output()
	if err != nil {
		return false, "", ""
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		iface := fields[1]

		isHotspotIface := false
		if (strings.HasPrefix(iface, "wlan") && iface != "wlan0") ||
			strings.HasPrefix(iface, "softap") ||
			strings.HasPrefix(iface, "ap") ||
			strings.HasPrefix(iface, "swlan") {
			isHotspotIface = true
		}

		if isHotspotIface && fields[2] == "inet" {
			ipWithMask := fields[3]
			ipParts := strings.Split(ipWithMask, "/")
			return true, iface, ipParts[0]
		}
	}

	// Fallback cek dumpsys wifi
	dsOut, _ := exec.Command("su", "-c", "dumpsys wifi | grep -m 1 'SAP is enabled successfully'").Output()
	if len(dsOut) > 0 {
		return true, "wlan_ap", "192.168.43.1"
	}

	return false, "", ""
}

// buildQRPayload membuat string standar koneksi Wi-Fi untuk QR Code
func buildQRPayload(cfg HotspotConfig) string {
	authType := "WPA"
	if cfg.Password == "" || cfg.Security == "open" {
		authType = "nopass"
	}

	hiddenStr := ""
	if cfg.Hidden {
		hiddenStr = "H:true;"
	}

	if authType == "nopass" {
		return fmt.Sprintf("WIFI:S:%s;T:nopass;%s;", cfg.SSID, hiddenStr)
	}
	return fmt.Sprintf("WIFI:S:%s;T:%s;P:%s;%s;", cfg.SSID, authType, cfg.Password, hiddenStr)
}

// ==================== HTTP HANDLERS ====================

func handleHotspotStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	hotspotMu.Lock()
	defer hotspotMu.Unlock()

	cfg, _ := readSoftApXml()
	isActive, iface, ipAddr := detectActiveHotspot()

	status := HotspotStatus{
		Enabled:    isActive,
		Interface:  iface,
		IPAddress:  ipAddr,
		Config:     cfg,
		QRPayload:  buildQRPayload(cfg),
		LastUpdate: time.Now().Format("2006-01-02 15:04:05"),
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   status,
	})
}

func handleHotspotToggle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Unauthorized"})
		return
	}

	var req struct {
		Enable bool `json:"enable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON format"})
		return
	}

	hotspotMu.Lock()
	defer hotspotMu.Unlock()

	cfg, _ := readSoftApXml()

	if req.Enable {
		_, _ = runCmdWifiStartSoftAp(cfg.SSID, cfg.Password, cfg.Security, cfg.Band)
		time.Sleep(1 * time.Second)
	} else {
		exec.Command("su", "-c", "cmd wifi stop-softap").Run()
		time.Sleep(800 * time.Millisecond)
	}

	isActive, iface, ipAddr := detectActiveHotspot()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"enabled":    isActive,
		"interface":  iface,
		"ip_address": ipAddr,
		"message":    fmt.Sprintf("Hotspot status: %v", isActive),
	})
}

func handleHotspotSaveConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Unauthorized"})
		return
	}

	var req HotspotConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON format"})
		return
	}

	req.SSID = strings.TrimSpace(req.SSID)
	req.Password = strings.TrimSpace(req.Password)

	if req.SSID == "" {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Nama SSID tidak boleh kosong"})
		return
	}

	if req.Password != "" && len(req.Password) < 8 {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Kata sandi Wi-Fi minimal 8 karakter"})
		return
	}

	hotspotMu.Lock()
	defer hotspotMu.Unlock()

	_, xmlPath := readSoftApXml()
	if xmlPath == "" {
		xmlPath = "/data/misc/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml"
	}

	// 1. Simpan ke file XML sistem & lokal config
	if err := saveSoftApXml(req, xmlPath); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": fmt.Sprintf("Gagal menyimpan ke sistem: %v", err)})
		return
	}

	// 2. Terapkan konfigurasi baru ke SoftAP
	isActive, _, _ := detectActiveHotspot()
	if isActive {
		// Jika hotspot sedang ON, restart langsung dengan nama & sandi baru
		_, _ = runCmdWifiStartSoftAp(req.SSID, req.Password, req.Security, req.Band)
		time.Sleep(1 * time.Second)
	} else {
		// Jika hotspot sedang OFF, trigger update konfigurasi ke framework lalu hentikan
		_, _ = runCmdWifiStartSoftAp(req.SSID, req.Password, req.Security, req.Band)
		time.Sleep(500 * time.Millisecond)
		exec.Command("su", "-c", "cmd wifi stop-softap").Run()
		time.Sleep(500 * time.Millisecond)
	}

	updatedActive, updatedIface, updatedIP := detectActiveHotspot()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"message":    "Nama SSID dan kata sandi berhasil diperbarui!",
		"enabled":    updatedActive,
		"interface":  updatedIface,
		"ip_address": updatedIP,
		"config":     req,
		"qr_payload": buildQRPayload(req),
	})
}
