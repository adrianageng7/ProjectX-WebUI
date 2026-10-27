package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TTLConfig struct {
	Enabled bool `json:"enabled"`
	TTL     int  `json:"ttl"`
	IPv6HL  bool `json:"ipv6_hl"`
}

var (
	ttlMu     sync.RWMutex
	ttlConfig = TTLConfig{
		Enabled: false,
		TTL:     64,
		IPv6HL:  false,
	}
	ttlConfigLoaded = false
)

func getTTLConfigPath() string {
	paths := []string{
		"/data/adb/modules/projectx-webui/data/ttl_config.json",
		"/data/adb/projectx/ttl_config.json",
		"/data/local/tmp/projectx_ttl.json",
		"./ttl_config.json",
	}
	for _, p := range paths {
		dir := filepath.Dir(p)
		if _, err := os.Stat(dir); err == nil {
			return p
		}
	}
	return "./ttl_config.json"
}

func loadTTLConfig() {
	ttlMu.Lock()
	defer ttlMu.Unlock()

	p := getTTLConfigPath()
	data, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(data, &ttlConfig)
	} else {
		ttlConfig.Enabled = false
		ttlConfig.TTL = 64
		ttlConfig.IPv6HL = false
		saveTTLConfig()
	}
	if ttlConfig.TTL <= 0 || ttlConfig.TTL > 255 {
		ttlConfig.TTL = 64
	}
	ttlConfigLoaded = true

	if runtime.GOOS == "linux" && ttlConfig.Enabled {
		applyTTLRules(ttlConfig.TTL, ttlConfig.IPv6HL)
	}
}

func saveTTLConfig() {
	p := getTTLConfigPath()
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	data, err := json.MarshalIndent(ttlConfig, "", "  ")
	if err == nil {
		_ = os.WriteFile(p, data, 0644)
	}
}

func init() {
	loadTTLConfig()
}

func runIp6tables(args ...string) error {
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		cmdArgs := append([]string{"-w", "2"}, args...)
		cmd := exec.Command("ip6tables", cmdArgs...)
		err = cmd.Run()
		if err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

func ensureTTLChains() {
	if runtime.GOOS != "linux" {
		return
	}
	// IPv4 Mangle Table
	_ = runIptables("-t", "mangle", "-N", "PROJECTX_TTL")
	if exec.Command("iptables", "-t", "mangle", "-C", "POSTROUTING", "-j", "PROJECTX_TTL").Run() != nil {
		_ = runIptables("-t", "mangle", "-I", "POSTROUTING", "1", "-j", "PROJECTX_TTL")
	}

	// IPv6 Mangle Table
	_ = runIp6tables("-t", "mangle", "-N", "PROJECTX_TTL")
	if exec.Command("ip6tables", "-t", "mangle", "-C", "POSTROUTING", "-j", "PROJECTX_TTL").Run() != nil {
		_ = runIp6tables("-t", "mangle", "-I", "POSTROUTING", "1", "-j", "PROJECTX_TTL")
	}
}

func applyTTLRules(ttl int, ipv6HL bool) {
	if runtime.GOOS != "linux" {
		return
	}
	if ttl <= 0 || ttl > 255 {
		ttl = 64
	}

	ensureTTLChains()

	// 1. Flush IPv4 chain & apply new rule
	_ = runIptables("-t", "mangle", "-F", "PROJECTX_TTL")
	ttlStr := strconv.Itoa(ttl)
	_ = runIptables("-t", "mangle", "-A", "PROJECTX_TTL", "-j", "TTL", "--ttl-set", ttlStr)

	// 2. Set Linux default TTL in procfs
	_ = os.WriteFile("/proc/sys/net/ipv4/ip_default_ttl", []byte(ttlStr+"\n"), 0644)

	// 3. IPv6 Hop Limit
	_ = runIp6tables("-t", "mangle", "-F", "PROJECTX_TTL")
	if ipv6HL {
		// RFC 4861: ICMPv6 Neighbor Discovery (Router & Neighbor Solicitation/Advertisement)
		// WAJIB memiliki Hop Limit = 255. Jika Hop Limit diubah ke 64, router ISP / Wi-Fi
		// akan mendrop paket secara diam-diam (silently discard).
		// Ini menyebabkan seluruh koneksi IPv6 mati total (Black Hole) sehingga web seperti Spotify,
		// Dafont (Cloudflare), dan Video Twitter/X tidak bisa diakses!
		_ = runIp6tables("-t", "mangle", "-A", "PROJECTX_TTL", "-p", "icmpv6", "-j", "RETURN")
		_ = runIp6tables("-t", "mangle", "-A", "PROJECTX_TTL", "-p", "tcp", "-j", "HL", "--hl-set", ttlStr)
		_ = runIp6tables("-t", "mangle", "-A", "PROJECTX_TTL", "-p", "udp", "-j", "HL", "--hl-set", ttlStr)
	}
}

func removeTTLRules() {
	if runtime.GOOS != "linux" {
		return
	}
	ensureTTLChains()
	_ = runIptables("-t", "mangle", "-F", "PROJECTX_TTL")
	_ = runIp6tables("-t", "mangle", "-F", "PROJECTX_TTL")
	// Restore default TTL to 64
	_ = os.WriteFile("/proc/sys/net/ipv4/ip_default_ttl", []byte("64\n"), 0644)
}

func getSysctlTTL() int {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_default_ttl")
	if err == nil {
		val, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil {
			return val
		}
	}
	return 64
}

func getActiveTTLRules() ([]string, []string) {
	var v4Rules, v6Rules []string
	if runtime.GOOS != "linux" {
		return v4Rules, v6Rules
	}

	if out, err := exec.Command("iptables", "-t", "mangle", "-S", "PROJECTX_TTL").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-A PROJECTX_TTL") {
				v4Rules = append(v4Rules, line)
			}
		}
	}

	if out, err := exec.Command("ip6tables", "-t", "mangle", "-S", "PROJECTX_TTL").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-A PROJECTX_TTL") {
				v6Rules = append(v6Rules, line)
			}
		}
	}

	return v4Rules, v6Rules
}

// ==================== HTTP HANDLERS ====================

func handleTTLStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ttlMu.RLock()
	defer ttlMu.RUnlock()

	v4Rules, v6Rules := getActiveTTLRules()
	sysctlTTL := getSysctlTTL()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"enabled":         ttlConfig.Enabled,
		"ttl":             ttlConfig.TTL,
		"ipv6_hl":         ttlConfig.IPv6HL,
		"sysctl_ttl":      sysctlTTL,
		"active_rules_v4": v4Rules,
		"active_rules_v6": v6Rules,
	})
}

func handleTTLSet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Enabled *bool `json:"enabled"`
		TTL     *int  `json:"ttl"`
		IPv6HL  *bool `json:"ipv6_hl"`
	}

	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// Also support query parameters
	q := r.URL.Query()
	if val := q.Get("enabled"); val != "" {
		b := val == "true" || val == "1"
		req.Enabled = &b
	}
	if val := q.Get("ttl"); val != "" {
		if t, err := strconv.Atoi(val); err == nil {
			req.TTL = &t
		}
	}
	if val := q.Get("ipv6_hl"); val != "" {
		b := val == "true" || val == "1"
		req.IPv6HL = &b
	}

	ttlMu.Lock()
	if req.Enabled != nil {
		ttlConfig.Enabled = *req.Enabled
	}
	if req.TTL != nil && *req.TTL > 0 && *req.TTL <= 255 {
		ttlConfig.TTL = *req.TTL
	}
	if req.IPv6HL != nil {
		ttlConfig.IPv6HL = *req.IPv6HL
	}

	currentEnabled := ttlConfig.Enabled
	currentTTL := ttlConfig.TTL
	currentIPv6HL := ttlConfig.IPv6HL
	saveTTLConfig()
	ttlMu.Unlock()

	if currentEnabled {
		applyTTLRules(currentTTL, currentIPv6HL)
	} else {
		removeTTLRules()
	}

	v4Rules, v6Rules := getActiveTTLRules()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"message":         fmt.Sprintf("TTL berhasil diatur ke %d (Enabled: %v)", currentTTL, currentEnabled),
		"enabled":         currentEnabled,
		"ttl":             currentTTL,
		"ipv6_hl":         currentIPv6HL,
		"active_rules_v4": v4Rules,
		"active_rules_v6": v6Rules,
	})
}

