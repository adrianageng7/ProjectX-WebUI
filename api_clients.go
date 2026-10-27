package main

import (
	"bufio"
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

type Client struct {
	Hostname   string `json:"hostname"`
	IP         string `json:"ip"`
	MAC        string `json:"mac"`
	Interface  string `json:"interface"`
	IsBlocked  bool   `json:"is_blocked"`
	LimitSpeed string `json:"limit_speed,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	ExpiresIn  string `json:"expires_in,omitempty"`
}

type ClientSetting struct {
	Alias      string `json:"alias,omitempty"`
	IsBlocked  bool   `json:"is_blocked"`
	LimitSpeed string `json:"limit_speed,omitempty"`
	IP         string `json:"ip,omitempty"`
}

var (
	clientMu       sync.RWMutex
	clientSettings = make(map[string]ClientSetting) // key: MAC (normalized lowercase) or IP
	settingsLoaded = false
)

func init() {
	loadClientSettings()
}

func getSettingsFilePath() string {
	paths := []string{
		"/data/adb/projectx/client_settings.json",
		"/data/local/tmp/projectx_clients.json",
		"./client_settings.json",
	}
	for _, p := range paths {
		dir := filepath.Dir(p)
		if _, err := os.Stat(dir); err == nil {
			return p
		}
	}
	return "./client_settings.json"
}

func loadClientSettings() {
	clientMu.Lock()
	defer clientMu.Unlock()

	filePath := getSettingsFilePath()
	data, err := os.ReadFile(filePath)
	if err == nil {
		_ = json.Unmarshal(data, &clientSettings)
	}
	settingsLoaded = true

	// Apply blocked & speed limited rules to iptables on startup if on Linux/Android
	if runtime.GOOS == "linux" {
		ensureLimitChain()
		for key, setting := range clientSettings {
			if setting.IsBlocked {
				if strings.Contains(key, ":") || strings.Contains(key, "-") {
					applyBlock("", key)
				} else {
					applyBlock(key, "")
				}
			}
			if setting.LimitSpeed != "" {
				targetIP := setting.IP
				if targetIP == "" && !strings.Contains(key, ":") && !strings.Contains(key, "-") {
					targetIP = key
				}
				if targetIP != "" {
					applyLimit(targetIP, "", setting.LimitSpeed)
				}
			}
		}
	}
}

func saveClientSettings() {
	filePath := getSettingsFilePath()
	_ = os.MkdirAll(filepath.Dir(filePath), 0755)
	data, err := json.MarshalIndent(clientSettings, "", "  ")
	if err == nil {
		_ = os.WriteFile(filePath, data, 0644)
	}
}

// Top Vendor OUI database (first 3 octets of MAC, lowercase without colons)
var vendorMap = map[string]string{
	"000393": "Apple", "000502": "Apple", "000a27": "Apple", "000a95": "Apple",
	"000d93": "Apple", "0010fa": "Apple", "001124": "Apple", "001451": "Apple",
	"0016cb": "Apple", "0017f2": "Apple", "0019e3": "Apple", "001b63": "Apple",
	"001cb3": "Apple", "001d4f": "Apple", "001e52": "Apple", "001ec2": "Apple",
	"0021e9": "Apple", "002241": "Apple", "002312": "Apple", "002332": "Apple",
	"00236c": "Apple", "0023df": "Apple", "002436": "Apple", "002500": "Apple",
	"00254b": "Apple", "0025bc": "Apple", "002608": "Apple", "00264a": "Apple",
	"0026b0": "Apple", "0026bb": "Apple", "186590": "Apple", "20a2e4": "Apple",
	"28cfe9": "Apple", "34159e": "Apple", "38cada": "Apple", "406c8f": "Apple",
	"48437c": "Apple", "4c3275": "Apple", "50bc96": "Apple", "544e90": "Apple",
	"5c969d": "Apple", "600308": "Apple", "6476ba": "Apple", "689c70": "Apple",
	"701124": "Apple", "74e182": "Apple", "787b8a": "Apple", "7c04d0": "Apple",
	"80e650": "Apple", "84fcac": "Apple", "8866a5": "Apple", "8c8590": "Apple",
	"907240": "Apple", "941625": "Apple", "9801a7": "Apple", "9c04eb": "Apple",
	"a0999b": "Apple", "a4c361": "Apple", "a8667f": "Apple", "ac87a3": "Apple",
	"b065bd": "Apple", "b418d1": "Apple", "b87826": "Apple", "bcd074": "Apple",
	"c0847d": "Apple", "c4b301": "Apple", "c81ee7": "Apple", "cc29f5": "Apple",
	"d023db": "Apple", "d4619d": "Apple", "d8004d": "Apple", "dc2b61": "Apple",
	"e0c767": "Apple", "e4ce8f": "Apple", "e8802e": "Apple", "ecadb8": "Apple",
	"f01898": "Apple", "f40f24": "Apple", "f8ffc2": "Apple", "fc257f": "Apple",

	"009ec8": "Xiaomi", "04cf8c": "Xiaomi", "0c1daf": "Xiaomi", "102ab3": "Xiaomi",
	"14f65a": "Xiaomi", "18f0e4": "Xiaomi", "2082c0": "Xiaomi", "286c07": "Xiaomi",
	"34800d": "Xiaomi", "38a4ed": "Xiaomi", "3cbd3e": "Xiaomi", "44237c": "Xiaomi",
	"488705": "Xiaomi", "50642b": "Xiaomi", "5448e6": "Xiaomi", "584120": "Xiaomi",
	"5cc307": "Xiaomi", "640980": "Xiaomi", "64cc2e": "Xiaomi", "68dfdd": "Xiaomi",
	"7070fc": "Xiaomi", "742344": "Xiaomi", "7811dc": "Xiaomi", "7c1dd9": "Xiaomi",
	"80ad16": "Xiaomi", "84f3eb": "Xiaomi", "8cbebe": "Xiaomi", "98fae3": "Xiaomi",
	"9c99a0": "Xiaomi", "a43b35": "Xiaomi", "acc1ee": "Xiaomi", "b0e5ed": "Xiaomi",
	"b40b44": "Xiaomi", "b83765": "Xiaomi", "c40bcb": "Xiaomi", "d4970b": "Xiaomi",
	"dc85de": "Xiaomi", "e4aaea": "Xiaomi", "ecd09f": "Xiaomi", "f460e2": "Xiaomi",
	"f8a45f": "Xiaomi", "fc643a": "Xiaomi",

	"0007ab": "Samsung", "0012fb": "Samsung", "001599": "Samsung", "001632": "Samsung",
	"0018af": "Samsung", "001a8a": "Samsung", "001d25": "Samsung", "001e7d": "Samsung",
	"002119": "Samsung", "002339": "Samsung", "002454": "Samsung", "002637": "Samsung",
	"04180f": "Samsung", "0808c2": "Samsung", "0c1420": "Samsung", "14bb6e": "Samsung",
	"183a2d": "Samsung", "244b03": "Samsung", "28987b": "Samsung", "34c059": "Samsung",
	"380195": "Samsung", "400e85": "Samsung", "4844f7": "Samsung", "4cbca5": "Samsung",
	"50ccf8": "Samsung", "5440ad": "Samsung", "5c0a5b": "Samsung", "647791": "Samsung",
	"68ebae": "Samsung", "702c1f": "Samsung", "784b87": "Samsung", "78f882": "Samsung",
	"8425db": "Samsung", "88365f": "Samsung", "8c7712": "Samsung", "9401c2": "Samsung",
	"9852b1": "Samsung", "a0821f": "Samsung", "a80600": "Samsung", "b0c4e7": "Samsung",
	"b857d8": "Samsung", "c4731e": "Samsung", "cc07ab": "Samsung", "d0176a": "Samsung",
	"e47cf9": "Samsung", "e8e5d6": "Samsung", "f0ee10": "Samsung", "f877b8": "Samsung",
	"fca13e": "Samsung",

	"001a11": "Google", "3c5a37": "Google", "546009": "Google", "641666": "Google",
	"a47733": "Google", "d86c63": "Google", "f40304": "Google", "f88fca": "Google",

	"001882": "Huawei", "001e10": "Huawei", "00259e": "Huawei", "00464b": "Huawei",
	"04c06f": "Huawei", "0819a6": "Huawei", "0c96bf": "Huawei", "104780": "Huawei",
	"200bc7": "Huawei", "24df6a": "Huawei", "38bc01": "Huawei", "4846fb": "Huawei",
	"5c0214": "Huawei", "68a0f6": "Huawei", "786a89": "Huawei", "80b686": "Huawei",
	"8853d4": "Huawei", "9c28ef": "Huawei", "ace87b": "Huawei", "c07009": "Huawei",
	"d8490b": "Huawei", "e01954": "Huawei", "f4559c": "Huawei",

	"0022f4": "OPPO", "10d07a": "OPPO", "1c77f6": "OPPO", "24e271": "OPPO",
	"307512": "OPPO", "404e36": "OPPO", "508f4c": "OPPO", "582af7": "OPPO",
	"683e34": "OPPO", "70bf92": "OPPO", "7c6166": "OPPO", "80ea96": "OPPO",
	"8c8580": "OPPO", "980d2e": "OPPO", "a4c494": "OPPO", "b0f1ec": "OPPO",
	"c40938": "OPPO", "d0c1b1": "OPPO", "e0dcff": "OPPO", "f01dbc": "OPPO",

	"145bd1": "Vivo", "24876d": "Vivo", "38a851": "Vivo", "50a4c8": "Vivo",
	"589396": "Vivo", "74a78e": "Vivo", "842e27": "Vivo", "a06986": "Vivo",
	"b43052": "Vivo", "c49f4c": "Vivo", "e84e06": "Vivo", "f4b7e2": "Vivo",

	"18fe34": "Espressif", "240ac4": "Espressif", "2462ab": "Espressif", "246f28": "Espressif",
	"24dcc3": "Espressif", "2c3ae8": "Espressif", "2cf432": "Espressif", "30aea4": "Espressif",
	"3c6105": "Espressif", "3c71bf": "Espressif", "4022d8": "Espressif", "409151": "Espressif",
	"483fda": "Espressif", "485519": "Espressif", "4c7525": "Espressif", "500291": "Espressif",
	"5443b2": "Espressif", "5ccf7f": "Espressif", "600194": "Espressif", "68b68a": "Espressif",
	"68c63a": "Espressif", "70039f": "Espressif", "7cdfa1": "Espressif", "807d3a": "Espressif",
	"840d8e": "Espressif", "84cca8": "Espressif", "84f703": "Espressif", "8caab5": "Espressif",
	"9097d5": "Espressif", "94b555": "Espressif", "94b97e": "Espressif", "98cdac": "Espressif",
	"a020a6": "Espressif", "a4cf12": "Espressif", "a4e57c": "Espressif", "ac67b2": "Espressif",
	"acd074": "Espressif", "b4e62d": "Espressif", "bcddc2": "Espressif", "c44f33": "Espressif",
	"c82b96": "Espressif", "cc50e3": "Espressif", "d4d4da": "Espressif", "d8a01d": "Espressif",
	"dc4f22": "Espressif", "e09806": "Espressif", "e831cd": "Espressif", "e868e7": "Espressif",
	"e89f6d": "Espressif", "e8db84": "Espressif", "ec6260": "Espressif", "ecfabc": "Espressif",
	"f4cfa2": "Espressif",

	"28cdc1": "Raspberry Pi", "b827eb": "Raspberry Pi", "dca632": "Raspberry Pi", "e45f01": "Raspberry Pi",

	"000aeb": "TP-Link", "001478": "TP-Link", "0019e0": "TP-Link", "001d0f": "TP-Link",
	"002127": "TP-Link", "0023cd": "TP-Link", "002586": "TP-Link", "002719": "TP-Link",
	"14cf92": "TP-Link", "14cc20": "TP-Link", "30b5c2": "TP-Link", "503aa0": "TP-Link",
	"50c7bf": "TP-Link", "6032b1": "TP-Link", "6466b3": "TP-Link", "704f57": "TP-Link",
	"7405a5": "TP-Link", "90f652": "TP-Link", "984827": "TP-Link", "ac84c6": "TP-Link",
	"c025e9": "TP-Link", "c46e1f": "TP-Link", "d807b6": "TP-Link", "e848b8": "TP-Link",
	"ec086b": "TP-Link", "f4ec38": "TP-Link",

	"0002b3": "Intel", "000347": "Intel", "000423": "Intel", "000e0c": "Intel",
	"001302": "Intel", "001500": "Intel", "0016ea": "Intel", "001b21": "Intel",
	"001e64": "Intel", "00216a": "Intel", "0024d7": "Intel", "28704e": "Intel",
	"3413e8": "Intel", "3c6aa7": "Intel", "4851b7": "Intel", "4c796e": "Intel",
	"5891cf": "Intel", "6805ca": "Intel", "7c5758": "Intel", "8086f2": "Intel",
	"8c8d28": "Intel", "94659c": "Intel", "a036bc": "Intel", "b49691": "Intel",
	"c85b76": "Intel", "d43b04": "Intel", "dc7196": "Intel", "e4a7a0": "Intel",
	"f85971": "Intel",

	"0003ff": "Microsoft", "000d3a": "Microsoft", "00125a": "Microsoft",
	"00155d": "Microsoft", "0017fa": "Microsoft", "001dd8": "Microsoft",
	"002248": "Microsoft", "0025ae": "Microsoft", "281878": "Microsoft",
	"3059b7": "Microsoft", "70b3d5": "Microsoft", "dcb4c4": "Microsoft",
}

func getVendorFromMAC(mac string) string {
	clean := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(mac, ":", ""), "-", ""))
	if len(clean) >= 6 {
		prefix := clean[:6]
		if v, ok := vendorMap[prefix]; ok {
			return v
		}
	}
	return ""
}

func resolveHostname(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var r net.Resolver
	names, err := r.LookupAddr(ctx, ip)
	if err == nil && len(names) > 0 {
		name := strings.TrimSuffix(names[0], ".")
		if name != "" && !strings.Contains(name, "ip6.arpa") && !strings.Contains(name, "in-addr.arpa") {
			return name
		}
	}
	return ""
}

// Read DHCP leases from standard Android locations
type dhcpLease struct {
	mac       string
	ip        string
	hostname  string
	expiresIn string
}

func parseDnsmasqLeases() map[string]dhcpLease {
	leases := make(map[string]dhcpLease)
	paths := []string{
		"/data/misc/dhcp/dnsmasq.leases",
		"/data/misc/wifi/dnsmasq.leases",
		"/data/misc/apexdata/com.android.tethering/misc/dnsmasq.leases",
		"/data/misc/apexdata/com.android.wifi/dnsmasq.leases",
		"/apex/com.android.tethering/misc/dnsmasq.leases",
		"/data/vendor/wifi/dnsmasq.leases",
		"/data/local/tmp/dnsmasq_hotspot.leases",
		"/data/system/users/0/tethering_leases",
	}

	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			// Format: <timestamp> <mac> <ip> <hostname> <client-id>
			if len(fields) >= 3 {
				mac := strings.ToLower(fields[1])
				ip := fields[2]
				host := ""
				if len(fields) >= 4 && fields[3] != "*" {
					host = fields[3]
				}

				expStr := ""
				if expEpoch, err := strconv.ParseInt(fields[0], 10, 64); err == nil && expEpoch > 0 {
					rem := expEpoch - time.Now().Unix()
					if rem > 0 {
						expStr = fmt.Sprintf("%dm", rem/60)
					}
				}

				leases[mac] = dhcpLease{
					mac:       mac,
					ip:        ip,
					hostname:  host,
					expiresIn: expStr,
				}
			}
		}
		f.Close()
	}
	return leases
}

// Read modern Android NetworkStack DHCP leases (Android 11, 12, 13, 14, 15, 16)
func parseNetworkStackDumpsys(leases map[string]dhcpLease) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "dumpsys", "network_stack").Output()
	if err != nil {
		out, err = exec.CommandContext(ctx, "su", "-c", "dumpsys network_stack").Output()
		if err != nil {
			return
		}
	}

	reMac := regexp.MustCompile(`hwAddr:\s*([0-9a-fA-F:]{17})`)
	reNet := regexp.MustCompile(`netAddr:\s*([0-9.]+)/\d+`)
	reHost := regexp.MustCompile(`hostname:\s*([^\s,]+)`)

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "hwAddr:") {
			continue
		}

		macMatch := reMac.FindStringSubmatch(line)
		if len(macMatch) < 2 {
			continue
		}
		mac := strings.ToLower(macMatch[1])

		ip := ""
		if netMatch := reNet.FindStringSubmatch(line); len(netMatch) >= 2 {
			ip = netMatch[1]
		}

		host := ""
		if hostMatch := reHost.FindStringSubmatch(line); len(hostMatch) >= 2 {
			h := strings.TrimSpace(hostMatch[1])
			if h != "null" && h != "*" && h != "" {
				host = h
			}
		}

		if existing, exists := leases[mac]; exists {
			if host != "" {
				existing.hostname = host
			}
			if ip != "" {
				existing.ip = ip
			}
			leases[mac] = existing
		} else {
			leases[mac] = dhcpLease{
				mac:      mac,
				ip:       ip,
				hostname: host,
			}
		}
	}
}

// Query NetBIOS Name (UDP port 137) for Windows / Samba LAN clients
func queryNetBIOSName(ip string) string {
	conn, err := net.DialTimeout("udp", ip+":137", 150*time.Millisecond)
	if err != nil {
		return ""
	}
	defer conn.Close()

	query := []byte{
		0x82, 0x28, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x20, 0x43, 0x4b, 0x41,
		0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
		0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
		0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
		0x41, 0x41, 0x41, 0x41, 0x41, 0x00, 0x00, 0x21,
		0x00, 0x01,
	}

	conn.SetDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := conn.Write(query); err != nil {
		return ""
	}

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil || n < 57 {
		return ""
	}

	numNames := int(buf[56])
	offset := 57
	for i := 0; i < numNames && offset+18 <= n; i++ {
		name := strings.TrimSpace(string(buf[offset : offset+15]))
		nameType := buf[offset+15]
		offset += 18
		if nameType == 0x00 && name != "" && !strings.HasPrefix(name, "IS~") {
			return name
		}
	}
	return ""
}

// Validasi interface LAN / Tethering yang aktif (bukan WAN/Cellular/VPN/Loopback)
func isClientInterfaceValid(dev string, activeIfaces map[string]bool) bool {
	if dev == "" {
		return false
	}
	dev = strings.ToLower(dev)
	// Abaikan WAN, Cellular (rmnet/ccmni/pdp), VPN tunnel, loopback, dummy
	if strings.HasPrefix(dev, "rmnet") ||
		strings.HasPrefix(dev, "r_rmnet") ||
		strings.HasPrefix(dev, "ccmni") ||
		strings.HasPrefix(dev, "pdp") ||
		strings.HasPrefix(dev, "tun") ||
		strings.HasPrefix(dev, "dummy") ||
		strings.HasPrefix(dev, "lo") ||
		strings.HasPrefix(dev, "sit") ||
		strings.HasPrefix(dev, "ip6tnl") ||
		strings.HasPrefix(dev, "ip_vti") ||
		strings.HasPrefix(dev, "ip6_vti") {
		return false
	}
	if len(activeIfaces) > 0 && !activeIfaces[dev] {
		return false
	}
	return true
}

// Read /proc/net/arp
func parseProcNetArp(clientMap map[string]*Client, activeIfaces map[string]bool) {
	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Skip header
	if scanner.Scan() {
		// header
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		// IP address, HW type, Flags, HW address, Mask, Device
		if len(fields) >= 6 {
			ip := fields[0]
			flags := fields[2]
			mac := strings.ToLower(fields[3])
			dev := strings.ToLower(fields[5])

			// Abaikan interface yang tidak aktif atau WAN
			if !isClientInterfaceValid(dev, activeIfaces) {
				continue
			}

			// Flags 0x2 = ATF_COM (complete ARP entry)
			if (flags == "0x2" || flags == "2") && mac != "00:00:00:00:00:00" && mac != "ff:ff:ff:ff:ff:ff" {
				if c, exists := clientMap[mac]; exists {
					if c.IP == "" || strings.Contains(c.IP, ":") {
						c.IP = ip
					}
					if c.Interface == "" {
						c.Interface = dev
					}
				} else {
					clientMap[mac] = &Client{
						IP:        ip,
						MAC:       mac,
						Interface: dev,
					}
				}
			}
		}
	}
}

// Parse Windows arp -a for local testing
func parseWindowsArp(clientMap map[string]*Client) {
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		return
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && strings.Contains(fields[0], ".") {
			ip := fields[0]
			mac := strings.ToLower(strings.ReplaceAll(fields[1], "-", ":"))
			typ := fields[2]

			if typ == "dynamic" && mac != "ff:ff:ff:ff:ff:ff" && !strings.HasPrefix(ip, "224.") && !strings.HasPrefix(ip, "239.") && !strings.HasSuffix(ip, ".255") {
				clientMap[mac] = &Client{
					IP:        ip,
					MAC:       mac,
					Interface: "lan",
				}
			}
		}
	}
}

// Parse ip neigh show
func parseIpNeigh(clientMap map[string]*Client, activeIfaces map[string]bool) {
	out, err := exec.Command("ip", "neigh", "show").Output()
	if err != nil {
		// Coba path android /system/bin/ip
		out, err = exec.Command("/system/bin/ip", "neigh", "show").Output()
		if err != nil {
			return
		}
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[3] == "lladdr" {
			ip := fields[0]
			dev := strings.ToLower(fields[2])
			mac := strings.ToLower(fields[4])
			state := fields[len(fields)-1]

			// Abaikan interface yang tidak aktif atau WAN
			if !isClientInterfaceValid(dev, activeIfaces) {
				continue
			}

			if mac == "00:00:00:00:00:00" || mac == "ff:ff:ff:ff:ff:ff" {
				continue
			}

			// Jika state FAILED atau INCOMPLETE untuk IP tersebut, hapus dari daftar aktif
			if state == "FAILED" || state == "INCOMPLETE" {
				if c, exists := clientMap[mac]; exists && (c.IP == ip || strings.Contains(c.IP, ":")) {
					delete(clientMap, mac)
				}
				continue
			}

			if state == "REACHABLE" || state == "STALE" || state == "DELAY" || state == "PROBE" {
				if c, exists := clientMap[mac]; exists {
					if strings.Contains(ip, ".") && strings.Contains(c.IP, ":") {
						c.IP = ip
					}
					if c.Interface == "" {
						c.Interface = dev
					}
				} else {
					clientMap[mac] = &Client{
						IP:        ip,
						MAC:       mac,
						Interface: dev,
					}
				}
			}
		}
	}
}

func getConnectedClients() ([]Client, error) {
	if !settingsLoaded {
		loadClientSettings()
	}

	// 1. Dapatkan daftar interface jaringan yang aktif (UP dan bukan loopback)
	activeIfaces := make(map[string]bool)
	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			if (iface.Flags&net.FlagUp != 0) && (iface.Flags&net.FlagLoopback == 0) {
				activeIfaces[strings.ToLower(iface.Name)] = true
			}
		}
	}

	clientMap := make(map[string]*Client)

	// 2. Kumpulkan HANYA perangkat yang aktif di tabel ARP / Neighbor Linux atau Windows
	// (JANGAN buat client dari riwayat DHCP leases yang sudah kadaluarsa/putus)
	if runtime.GOOS == "windows" {
		parseWindowsArp(clientMap)
	} else {
		parseProcNetArp(clientMap, activeIfaces)
		parseIpNeigh(clientMap, activeIfaces)
	}

	// 3. Ambil DHCP Leases (dnsmasq & Android NetworkStack) HANYA sebagai metadata (Hostname & Masa Berlaku)
	leases := parseDnsmasqLeases()
	if runtime.GOOS == "linux" {
		parseNetworkStackDumpsys(leases)
	}

	ipToHost := make(map[string]string)
	for _, l := range leases {
		if l.ip != "" && l.hostname != "" {
			ipToHost[l.ip] = l.hostname
		}
	}

	// 4. Perkaya clientMap aktif dengan metadata dari leases
	for mac, c := range clientMap {
		normMac := strings.ToLower(mac)
		if l, ok := leases[normMac]; ok {
			if (c.Hostname == "" || c.Hostname == "*") && l.hostname != "" {
				c.Hostname = l.hostname
			}
			if c.ExpiresIn == "" {
				c.ExpiresIn = l.expiresIn
			}
			if c.IP == "" && l.ip != "" {
				c.IP = l.ip
			}
		}
	}

	clientMu.RLock()
	settingsCopy := make(map[string]ClientSetting, len(clientSettings))
	for k, v := range clientSettings {
		settingsCopy[k] = v
	}
	clientMu.RUnlock()

	clients := make([]Client, 0, len(clientMap))
	for _, c := range clientMap {
		if c.MAC == "" && c.IP == "" {
			continue
		}

		// Validasi interface: harus aktif dan bukan WAN/kosong
		if runtime.GOOS != "windows" && !isClientInterfaceValid(c.Interface, activeIfaces) {
			continue
		}

		// Vendor lookup
		v := getVendorFromMAC(c.MAC)
		c.Vendor = v

		// Cek setting tersimpan (Alias, IsBlocked, LimitSpeed)
		normMac := strings.ToLower(c.MAC)
		setting, hasSetting := settingsCopy[normMac]
		if !hasSetting && c.IP != "" {
			setting, hasSetting = settingsCopy[c.IP]
		}

		if hasSetting {
			if setting.Alias != "" {
				c.Hostname = setting.Alias
			}
			c.IsBlocked = setting.IsBlocked
			c.LimitSpeed = setting.LimitSpeed

			// If client has LimitSpeed and IP is now known/changed, record IP and re-apply limit
			if c.LimitSpeed != "" && c.IP != "" && setting.IP != c.IP {
				clientMu.Lock()
				setting.IP = c.IP
				if normMac != "" {
					clientSettings[normMac] = setting
				} else {
					clientSettings[c.IP] = setting
				}
				saveClientSettings()
				clientMu.Unlock()
				applyLimit(c.IP, c.Interface, c.LimitSpeed)
			}
		}

		// Jika hostname kosong, cocokkan dengan IP dari DHCP leases
		if (c.Hostname == "" || c.Hostname == "*") && c.IP != "" {
			if h, ok := ipToHost[c.IP]; ok {
				c.Hostname = h
			}
		}

		// Jika hostname masih kosong, coba resolve DNS PTR atau NetBIOS
		if c.Hostname == "" || c.Hostname == "*" {
			if c.IP != "" {
				dnsName := resolveHostname(c.IP)
				if dnsName != "" {
					c.Hostname = dnsName
				} else {
					nbName := queryNetBIOSName(c.IP)
					if nbName != "" {
						c.Hostname = nbName
					}
				}
			}
		}

		// Fallback nama perangkat yang bersih
		if c.Hostname == "" || c.Hostname == "*" {
			if c.Vendor != "" {
				shortMac := ""
				if len(c.MAC) >= 5 {
					shortMac = c.MAC[len(c.MAC)-5:]
				}
				c.Hostname = fmt.Sprintf("%s (%s)", c.Vendor, shortMac)
			} else if strings.HasPrefix(c.Interface, "ncm") || strings.HasPrefix(c.Interface, "rndis") || strings.HasPrefix(c.Interface, "usb") {
				shortMac := ""
				if len(c.MAC) >= 5 {
					shortMac = c.MAC[len(c.MAC)-5:]
				}
				c.Hostname = fmt.Sprintf("USB-PC (%s)", shortMac)
			} else if len(c.MAC) >= 5 {
				c.Hostname = fmt.Sprintf("Device-%s", c.MAC[len(c.MAC)-5:])
			} else if c.IP != "" {
				c.Hostname = fmt.Sprintf("Device-%s", c.IP)
			} else {
				c.Hostname = "Unknown Device"
			}
		}

		// Kirim non-blocking probe ringan di background agar tabel ARP/neigh Linux tetap terverifikasi
		if c.IP != "" && strings.Contains(c.IP, ".") {
			go func(targetIP string) {
				conn, err := net.DialTimeout("udp", targetIP+":137", 50*time.Millisecond)
				if err == nil {
					_ = conn.Close()
				}
			}(c.IP)
		}

		clients = append(clients, *c)
	}

	return clients, nil
}

func applyBlock(ip, mac string) {
	if runtime.GOOS != "linux" {
		return
	}
	removeBlock(ip, mac)
	if ip != "" {
		_ = runIptables("-I", "FORWARD", "1", "-s", ip, "-j", "DROP")
		_ = runIptables("-I", "FORWARD", "1", "-d", ip, "-j", "DROP")
		_ = runIptables("-I", "INPUT", "1", "-s", ip, "-j", "DROP")
	}
	if mac != "" {
		_ = runIptables("-I", "FORWARD", "1", "-m", "mac", "--mac-source", mac, "-j", "DROP")
		_ = runIptables("-I", "INPUT", "1", "-m", "mac", "--mac-source", mac, "-j", "DROP")
	}
}

func removeBlock(ip, mac string) {
	if runtime.GOOS != "linux" {
		return
	}
	if ip != "" {
		for i := 0; i < 5; i++ {
			err := runIptables("-D", "FORWARD", "-s", ip, "-j", "DROP")
			if err != nil {
				break
			}
		}
		for i := 0; i < 5; i++ {
			err := runIptables("-D", "FORWARD", "-d", ip, "-j", "DROP")
			if err != nil {
				break
			}
		}
		for i := 0; i < 5; i++ {
			err := runIptables("-D", "INPUT", "-s", ip, "-j", "DROP")
			if err != nil {
				break
			}
		}
	}
	if mac != "" {
		for i := 0; i < 5; i++ {
			err := runIptables("-D", "FORWARD", "-m", "mac", "--mac-source", mac, "-j", "DROP")
			if err != nil {
				break
			}
		}
		for i := 0; i < 5; i++ {
			err := runIptables("-D", "INPUT", "-m", "mac", "--mac-source", mac, "-j", "DROP")
			if err != nil {
				break
			}
		}
	}
}

func runIptables(args ...string) error {
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		cmdArgs := append([]string{"-w", "2"}, args...)
		cmd := exec.Command("iptables", cmdArgs...)
		err = cmd.Run()
		if err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

func ensureLimitChain() {
	if runtime.GOOS != "linux" {
		return
	}
	_ = runIptables("-N", "PROJECTX_LIMIT")

	// PROJECTX_LIMIT khusus untuk client hotspot/tethering (FORWARD chain saja)
	if exec.Command("iptables", "-C", "FORWARD", "-j", "PROJECTX_LIMIT").Run() != nil {
		_ = runIptables("-I", "FORWARD", "1", "-j", "PROJECTX_LIMIT")
	}

	// Bersihkan hook legacy dari OUTPUT dan INPUT agar tidak mengganggu aplikasi lokal HP
	_ = runIptables("-D", "OUTPUT", "-j", "PROJECTX_LIMIT")
	_ = runIptables("-D", "INPUT", "-j", "PROJECTX_LIMIT")

	// TCP MSS Clamping to prevent PMTUD black hole / packet drop on 4G/LTE mobile networks
	if exec.Command("iptables", "-t", "mangle", "-C", "POSTROUTING", "-p", "tcp", "--tcp-flags", "SYN,RST", "SYN", "-j", "TCPMSS", "--clamp-mss-to-pmtu").Run() != nil {
		_ = runIptables("-t", "mangle", "-A", "POSTROUTING", "-p", "tcp", "--tcp-flags", "SYN,RST", "SYN", "-j", "TCPMSS", "--clamp-mss-to-pmtu")
	}
	if exec.Command("iptables", "-t", "mangle", "-C", "FORWARD", "-p", "tcp", "--tcp-flags", "SYN,RST", "SYN", "-j", "TCPMSS", "--clamp-mss-to-pmtu").Run() != nil {
		_ = runIptables("-t", "mangle", "-A", "FORWARD", "-p", "tcp", "--tcp-flags", "SYN,RST", "SYN", "-j", "TCPMSS", "--clamp-mss-to-pmtu")
	}
}

func parseSpeedLimit(speed string) (rateStr string, burstStr string, ok bool) {
	s := strings.ToLower(strings.TrimSpace(speed))
	if s == "" || s == "0" || s == "unlimit" {
		return "", "", false
	}

	if strings.HasSuffix(s, "b/s") {
		return s, "2mb", true
	}

	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([a-z]*)$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "", "", false
	}

	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil || val <= 0 {
		return "", "", false
	}

	unit := matches[2]
	var kbps float64

	if strings.HasPrefix(unit, "k") {
		kbps = val
	} else {
		kbps = val * 1000.0
	}

	// 1 Byte = 8 bits -> KB/s = kbps / 8
	rateKB := int(kbps / 8.0)
	if rateKB < 16 {
		rateKB = 16
	}

	burstKB := rateKB * 2
	if burstKB < 128 {
		burstKB = 128
	}
	if burstKB > 4096 {
		burstKB = 4096
	}

	return fmt.Sprintf("%dkb/s", rateKB), fmt.Sprintf("%dkb", burstKB), true
}

func cleanHashName(prefix, ip string) string {
	clean := strings.ReplaceAll(ip, ".", "_")
	clean = strings.ReplaceAll(clean, ":", "_")
	if len(clean) > 10 {
		clean = clean[len(clean)-10:]
	}
	return fmt.Sprintf("%s_%s", prefix, clean)
}

func applyLimit(ip, iface, speed string) {
	if runtime.GOOS != "linux" || ip == "" {
		return
	}
	ensureLimitChain()
	removeLimit(ip, iface)

	rate, burst, ok := parseSpeedLimit(speed)
	if !ok {
		return
	}

	dlName := cleanHashName("px_dl", ip)
	ulName := cleanHashName("px_ul", ip)

	// Download limit (match destination IP)
	_ = runIptables("-A", "PROJECTX_LIMIT", "-d", ip, "-m", "hashlimit",
		"--hashlimit-above", rate,
		"--hashlimit-burst", burst,
		"--hashlimit-mode", "dstip",
		"--hashlimit-name", dlName,
		"-j", "DROP")

	// Upload limit (match source IP)
	_ = runIptables("-A", "PROJECTX_LIMIT", "-s", ip, "-m", "hashlimit",
		"--hashlimit-above", rate,
		"--hashlimit-burst", burst,
		"--hashlimit-mode", "srcip",
		"--hashlimit-name", ulName,
		"-j", "DROP")
}

func removeLimit(ip, iface string) {
	if runtime.GOOS != "linux" || ip == "" {
		return
	}
	ensureLimitChain()

	out, err := exec.Command("iptables", "-S", "PROJECTX_LIMIT").Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-A PROJECTX_LIMIT ") {
			continue
		}
		if strings.Contains(line, " "+ip+"/") || strings.Contains(line, " "+ip+" ") {
			ruleArgs := strings.Fields(line[len("-A PROJECTX_LIMIT "):])
			delArgs := append([]string{"-D", "PROJECTX_LIMIT"}, ruleArgs...)
			_ = runIptables(delArgs...)
		}
	}
}

// ==================== HTTP HANDLERS ====================

func ApiClientsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clients, err := getConnectedClients()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
			"clients": []Client{},
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"clients": clients,
	})
}

func ApiBlockClientHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	mac := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mac")))
	action := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("action")))

	if ip == "" && mac == "" {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "IP or MAC is required"})
		return
	}

	key := mac
	if key == "" {
		key = ip
	}

	clientMu.Lock()
	setting := clientSettings[key]

	if action == "unblock" {
		setting.IsBlocked = false
		clientSettings[key] = setting
		saveClientSettings()
		clientMu.Unlock()

		removeBlock(ip, mac)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Client unblocked"})
		return
	}

	setting.IsBlocked = true
	clientSettings[key] = setting
	saveClientSettings()
	clientMu.Unlock()

	applyBlock(ip, mac)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Client blocked"})
}

func ApiUnblockClientHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Set("action", "unblock")
	r.URL.RawQuery = q.Encode()
	ApiBlockClientHandler(w, r)
}

func ApiLimitClientHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	mac := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mac")))
	speed := strings.TrimSpace(r.URL.Query().Get("speed"))
	iface := strings.TrimSpace(r.URL.Query().Get("interface"))
	action := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("action")))

	if ip == "" && mac == "" {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "IP or MAC is required"})
		return
	}

	key := mac
	if key == "" {
		key = ip
	}

	clientMu.Lock()
	setting := clientSettings[key]

	if action == "unlimit" || speed == "" || speed == "0" {
		setting.LimitSpeed = ""
		clientSettings[key] = setting
		saveClientSettings()
		clientMu.Unlock()

		removeLimit(ip, iface)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Speed limit removed"})
		return
	}

	setting.LimitSpeed = speed
	if ip != "" {
		setting.IP = ip
	}
	clientSettings[key] = setting
	saveClientSettings()
	clientMu.Unlock()

	applyLimit(ip, iface, speed)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": fmt.Sprintf("Speed limited to %s", speed)})
}

func ApiUnlimitClientHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Set("action", "unlimit")
	r.URL.RawQuery = q.Encode()
	ApiLimitClientHandler(w, r)
}

func ApiClientAliasHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	mac := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mac")))
	alias := strings.TrimSpace(r.URL.Query().Get("alias"))

	if r.Method == http.MethodPost {
		var req struct {
			IP    string `json:"ip"`
			MAC   string `json:"mac"`
			Alias string `json:"alias"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.IP != "" {
				ip = strings.TrimSpace(req.IP)
			}
			if req.MAC != "" {
				mac = strings.ToLower(strings.TrimSpace(req.MAC))
			}
			if req.Alias != "" {
				alias = strings.TrimSpace(req.Alias)
			}
		}
	}

	if mac == "" && ip == "" {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "IP or MAC is required"})
		return
	}

	key := mac
	if key == "" {
		key = ip
	}

	clientMu.Lock()
	setting := clientSettings[key]
	setting.Alias = alias
	clientSettings[key] = setting
	saveClientSettings()
	clientMu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Alias updated successfully"})
}
