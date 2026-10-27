package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type ClashTarget struct {
	Path  string `json:"path"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type ProxyItem struct {
	Index            int                    `json:"index"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Server           string                 `json:"server"`
	Port             interface{}            `json:"port"`
	UUID             string                 `json:"uuid,omitempty"`
	Password         string                 `json:"password,omitempty"`
	Cipher           string                 `json:"cipher,omitempty"`
	TLS              bool                   `json:"tls"`
	SkipCertVerify   bool                   `json:"skip_cert_verify"`
	Servername       string                 `json:"servername,omitempty"`
	Network          string                 `json:"network,omitempty"`
	WsPath           string                 `json:"ws_path,omitempty"`
	WsHost           string                 `json:"ws_host,omitempty"`
	GrpcServiceName string                 `json:"grpc_service_name,omitempty"`
	RealityPublicKey string                 `json:"reality_public_key,omitempty"`
	RealityShortID   string                 `json:"reality_short_id,omitempty"`
	UDP              bool                   `json:"udp"`
	Raw              map[string]interface{} `json:"raw,omitempty"`
	RawYaml          string                 `json:"raw_yaml,omitempty"`
}

type ClashSaveRequest struct {
	Action      string `json:"action"`
	TargetFile  string `json:"target_file"`
	YamlContent string `json:"yaml_content"`
}

type ClashAccountAddRequest struct {
	TargetFile  string      `json:"target_file"`
	YamlContent string      `json:"yaml_content,omitempty"`
	Item        *ProxyItem  `json:"item,omitempty"`
	Items       []ProxyItem `json:"items,omitempty"`
}

type ClashAccountUpdateRequest struct {
	TargetFile  string     `json:"target_file"`
	Index       int        `json:"index"`
	OriginalName string    `json:"original_name,omitempty"`
	YamlContent string     `json:"yaml_content,omitempty"`
	Item        *ProxyItem `json:"item,omitempty"`
}

type ClashAccountDeleteRequest struct {
	TargetFile string `json:"target_file"`
	Index      int    `json:"index"`
	Name       string `json:"name,omitempty"`
}

type ClashTargetCreateRequest struct {
	Name string `json:"name"`
}

const (
	clashBaseDir       = "/data/adb/box/clash"
	clashDefaultConfig = "/data/adb/box/clash/config.yaml"
	clashProviderDir   = "/data/adb/box/clash/proxy_provider"
)

func isAllowedClashPath(target string) bool {
	clean := filepath.Clean(target)
	if strings.HasPrefix(clean, "/data/adb/box/clash") {
		return true
	}
	cleanSlash := filepath.ToSlash(clean)
	if strings.HasPrefix(cleanSlash, "scratch/clash") || strings.HasPrefix(clean, "scratch\\clash") {
		return true
	}
	return false
}

func countProxiesInFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return 0
	}
	seq := getProxiesSequenceNode(&root)
	if seq != nil {
		return len(seq.Content)
	}
	return 0
}

// getClashTargets memindai file YAML yang dapat dijadikan target penyimpanan node proxy
func getClashTargets() []ClashTarget {
	var targets []ClashTarget

	// 3. Cek BFR Main Config (/data/adb/box/clash/config.yaml)
	if fi, err := os.Stat(clashDefaultConfig); err == nil && !fi.IsDir() {
		if isFileWritable(clashDefaultConfig) {
			targets = append(targets, ClashTarget{
				Path:  clashDefaultConfig,
				Label: "[BFR] config.yaml",
				Count: countProxiesInFile(clashDefaultConfig),
			})
		}
	}

	// 4. Cek BFR Provider Directory (/data/adb/box/clash/proxy_provider/*.yaml)
	if fi, err := os.Stat(clashProviderDir); err == nil && fi.IsDir() {
		entries, err := os.ReadDir(clashProviderDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if ext == ".yaml" || ext == ".yml" {
					fullPath := filepath.Join(clashProviderDir, entry.Name())
					if isFileWritable(fullPath) {
						targets = append(targets, ClashTarget{
							Path:  fullPath,
							Label: "[BFR] " + entry.Name(),
							Count: countProxiesInFile(fullPath),
						})
					}
				}
			}
		}
	}

	// 3. Fallback untuk pengujian lokal jika /data/adb/box tidak ada
	if len(targets) == 0 {
		devBase := filepath.Join("scratch", "clash")
		devConfig := filepath.Join(devBase, "config.yaml")
		devProvider := filepath.Join(devBase, "proxy_provider")

		if _, err := os.Stat(clashBaseDir); os.IsNotExist(err) {
			_ = os.MkdirAll(devProvider, 0755)
			if _, err := os.Stat(devConfig); os.IsNotExist(err) {
				_ = os.WriteFile(devConfig, []byte("port: 7890\nproxies:\n"), 0644)
			}
			providerSample := filepath.Join(devProvider, "provider1.yaml")
			if _, err := os.Stat(providerSample); os.IsNotExist(err) {
				_ = os.WriteFile(providerSample, []byte("proxies:\n"), 0644)
			}
		}

		if fi, err := os.Stat(devConfig); err == nil && !fi.IsDir() {
			targets = append(targets, ClashTarget{
				Path:  devConfig,
				Label: "Main Config (config.yaml)",
				Count: countProxiesInFile(devConfig),
			})
		}
		if entries, err := os.ReadDir(devProvider); err == nil {
			for _, entry := range entries {
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if ext == ".yaml" || ext == ".yml" {
					fullPath := filepath.Join(devProvider, entry.Name())
					targets = append(targets, ClashTarget{
						Path:  fullPath,
						Label: "Provider: " + entry.Name(),
						Count: countProxiesInFile(fullPath),
					})
				}
			}
		}
	}

	return targets
}

func isFileWritable(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// getProxiesSequenceNode mengekstrak SequenceNode proxies dari AST YAML
func getProxiesSequenceNode(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	doc := root
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			docNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			root.Content = []*yaml.Node{docNode}
		}
		doc = root.Content[0]
	}

	if doc.Kind == yaml.SequenceNode {
		return doc
	}

	if doc.Kind == yaml.MappingNode {
		for i := 0; i < len(doc.Content); i += 2 {
			if doc.Content[i].Value == "proxies" {
				if doc.Content[i+1].Kind == yaml.SequenceNode {
					return doc.Content[i+1]
				}
				// Jika proxies bukan sequence (misal null atau empty), jadikan sequence
				doc.Content[i+1].Kind = yaml.SequenceNode
				doc.Content[i+1].Tag = "!!seq"
				doc.Content[i+1].Content = nil
				return doc.Content[i+1]
			}
		}

		// Tambahkan key "proxies" ke mapping jika belum ada
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "proxies", Tag: "!!str"}
		seqNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		doc.Content = append(doc.Content, keyNode, seqNode)
		return seqNode
	}

	return nil
}

func parseProxyNode(node *yaml.Node, index int) (ProxyItem, error) {
	var raw map[string]interface{}
	if err := node.Decode(&raw); err != nil {
		return ProxyItem{}, err
	}

	item := ProxyItem{
		Index: index,
		Raw:   raw,
	}

	if v, ok := raw["name"].(string); ok {
		item.Name = v
	}
	if v, ok := raw["type"].(string); ok {
		item.Type = strings.ToLower(v)
	}
	if v, ok := raw["server"].(string); ok {
		item.Server = v
	}
	if v, ok := raw["port"]; ok {
		item.Port = v
	}
	if v, ok := raw["uuid"].(string); ok {
		item.UUID = v
	}
	if v, ok := raw["password"].(string); ok {
		item.Password = v
	}
	if v, ok := raw["cipher"].(string); ok {
		item.Cipher = v
	}
	if v, ok := raw["tls"].(bool); ok {
		item.TLS = v
	}
	if v, ok := raw["skip-cert-verify"].(bool); ok {
		item.SkipCertVerify = v
	}
	if v, ok := raw["servername"].(string); ok {
		item.Servername = v
	}
	if v, ok := raw["network"].(string); ok {
		item.Network = strings.ToLower(v)
	}
	if v, ok := raw["udp"].(bool); ok {
		item.UDP = v
	}

	// ws-opts
	if wsOpts, ok := raw["ws-opts"].(map[string]interface{}); ok {
		if path, ok := wsOpts["path"].(string); ok {
			item.WsPath = path
		}
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			if h, ok := headers["Host"].(string); ok {
				item.WsHost = h
			}
		}
	}

	// grpc-opts
	if grpcOpts, ok := raw["grpc-opts"].(map[string]interface{}); ok {
		if sName, ok := grpcOpts["grpc-service-name"].(string); ok {
			item.GrpcServiceName = sName
		}
	}

	// reality-opts
	if realityOpts, ok := raw["reality-opts"].(map[string]interface{}); ok {
		if pubKey, ok := realityOpts["public-key"].(string); ok {
			item.RealityPublicKey = pubKey
		}
		if sid, ok := realityOpts["short-id"].(string); ok {
			item.RealityShortID = sid
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	_ = enc.Encode(node)
	enc.Close()
	item.RawYaml = strings.TrimSpace(buf.String())

	return item, nil
}

func buildProxyYamlString(item ProxyItem) string {
	if strings.TrimSpace(item.RawYaml) != "" {
		trimmed := strings.TrimSpace(item.RawYaml)
		if strings.HasPrefix(trimmed, "- ") {
			return trimmed
		}
		return "- " + strings.ReplaceAll(trimmed, "\n", "\n  ")
	}

	var b strings.Builder
	pType := strings.ToLower(item.Type)
	if pType == "" {
		pType = "vless"
	}
	name := item.Name
	if name == "" {
		name = "Proxy-Node"
	}

	b.WriteString(fmt.Sprintf("- name: %q\n", name))
	b.WriteString(fmt.Sprintf("  type: %s\n", pType))
	b.WriteString(fmt.Sprintf("  server: %s\n", item.Server))
	b.WriteString(fmt.Sprintf("  port: %v\n", item.Port))

	if pType == "vless" || pType == "vmess" {
		if item.UUID != "" {
			b.WriteString(fmt.Sprintf("  uuid: %s\n", item.UUID))
		}
		if item.Cipher != "" {
			b.WriteString(fmt.Sprintf("  cipher: %s\n", item.Cipher))
		} else {
			b.WriteString("  cipher: auto\n")
		}
		if pType == "vmess" {
			b.WriteString("  alterId: 0\n")
		}
	} else if pType == "trojan" || pType == "ss" {
		if item.Password != "" {
			b.WriteString(fmt.Sprintf("  password: %s\n", item.Password))
		}
		if pType == "ss" && item.Cipher != "" {
			b.WriteString(fmt.Sprintf("  cipher: %s\n", item.Cipher))
		}
	}

	b.WriteString(fmt.Sprintf("  tls: %t\n", item.TLS))
	b.WriteString(fmt.Sprintf("  skip-cert-verify: %t\n", item.SkipCertVerify))

	if item.Servername != "" {
		b.WriteString(fmt.Sprintf("  servername: %s\n", item.Servername))
	}

	if item.RealityPublicKey != "" {
		b.WriteString("  reality-opts:\n")
		b.WriteString(fmt.Sprintf("    public-key: %s\n", item.RealityPublicKey))
		if item.RealityShortID != "" {
			b.WriteString(fmt.Sprintf("    short-id: %s\n", item.RealityShortID))
		}
	}

	netType := strings.ToLower(item.Network)
	if netType == "" {
		netType = "ws"
	}
	b.WriteString(fmt.Sprintf("  network: %s\n", netType))

	if netType == "ws" {
		path := item.WsPath
		if path == "" {
			path = "/"
		}
		b.WriteString("  ws-opts:\n")
		b.WriteString(fmt.Sprintf("    path: %s\n", path))
		if item.WsHost != "" {
			b.WriteString("    headers:\n")
			b.WriteString(fmt.Sprintf("      Host: %s\n", item.WsHost))
		}
	} else if netType == "grpc" {
		sName := item.GrpcServiceName
		if sName == "" {
			sName = "v2ray"
		}
		b.WriteString("  grpc-opts:\n")
		b.WriteString(fmt.Sprintf("    grpc-service-name: %s\n", sName))
	}

	b.WriteString(fmt.Sprintf("  udp: %t\n", item.UDP))
	return b.String()
}

func loadTargetYaml(targetPath string) (*yaml.Node, *yaml.Node, error) {
	data, err := os.ReadFile(targetPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}

	var root yaml.Node
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &root); err != nil {
			return nil, nil, fmt.Errorf("Format YAML tidak valid: %v", err)
		}
	}

	if len(root.Content) == 0 {
		root.Kind = yaml.DocumentNode
		docNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = []*yaml.Node{docNode}
	}

	seqNode := getProxiesSequenceNode(&root)
	if seqNode == nil {
		return nil, nil, fmt.Errorf("Tidak dapat menemukan atau membuat bagian 'proxies'")
	}

	return &root, seqNode, nil
}

func saveTargetYaml(targetPath string, root *yaml.Node) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return err
	}
	enc.Close()

	return os.WriteFile(targetPath, buf.Bytes(), 0644)
}

const clashBFRTemplateYaml = `# =================================================================
# ProjectX WebUI - Template Config Clash / Mihomo untuk BFR
# Fitur: Pisah Trafik (Game, Sosmed/Streaming, Indo Direct, Adblock)
# Support: External Controller (0.0.0.0:9090) & DNS Android Fake-IP
# =================================================================

mixed-port: 7890
tproxy-port: 9898
redir-port: 9797
allow-lan: true
bind-address: '*'
mode: rule
log-level: info
ipv6: false
external-controller: 0.0.0.0:9090
external-ui: ./dashboard
secret: ''

dns:
  enable: true
  listen: 0.0.0.0:1053
  ipv6: false
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  fake-ip-filter:
    - '*.lan'
    - '*.local'
    - 'localhost.ptlogin2.qq.com'
    - '+.msftconnecttest.com'
    - '+.msftncsi.com'
  nameserver:
    - https://dns.google/dns-query
    - https://cloudflare-dns.com/dns-query
    - 8.8.8.8
    - 1.1.1.1
  fallback:
    - 8.8.4.4
    - 1.0.0.1
    - https://dns.quad9.net/dns-query

sniffer:
  enable: true
  parse-pure-ip: true
  sniff:
    TLS:
      ports: [443, 8443]
    HTTP:
      ports: [80, 8080-8880]
      override-destination: true
    QUIC:
      ports: [443]
  skip-domain:
    - 'Mijia Cloud'
    - 'dlg.io.mi.com'

proxy-providers:
  my_proxies:
    type: file
    path: ./proxy_provider/my_proxies.yaml
    health-check:
      enable: true
      interval: 300
      url: http://www.gstatic.com/generate_204

proxy-groups:
  - name: 🚀 PROXY UTAMA
    type: select
    use:
      - my_proxies
    proxies:
      - ♻️ AUTO PING
      - DIRECT

  - name: ♻️ AUTO PING
    type: url-test
    use:
      - my_proxies
    url: http://www.gstatic.com/generate_204
    interval: 300
    tolerance: 50

  - name: 🎮 GAME
    type: select
    proxies:
      - DIRECT
      - 🚀 PROXY UTAMA
      - ♻️ AUTO PING
    use:
      - my_proxies

  - name: 📺 STREAMING & SOSMED
    type: select
    proxies:
      - 🚀 PROXY UTAMA
      - ♻️ AUTO PING
      - DIRECT
    use:
      - my_proxies

  - name: 🇮🇩 INDO DIRECT
    type: select
    proxies:
      - DIRECT
      - 🚀 PROXY UTAMA

  - name: 🛡️ IKLAN / ADS
    type: select
    proxies:
      - REJECT
      - DIRECT

rules:
  # 🚫 Blokir Protokol QUIC (UDP 443 & UDP 80)
  # Wajib untuk mematikan QUIC agar Twitter/X Video, Spotify Web, dan Cloudflare (Dafont dll) instan fallback ke TCP
  - AND,((NETWORK,udp),(DST-PORT,443)),REJECT
  - AND,((NETWORK,udp),(DST-PORT,80)),REJECT

  # Port Game & Game Populer (Mobile Legends, PUBG, Free Fire, Garena, Steam, Riot, Hoyoverse, Roblox)
  - DST-PORT,30000-30010,🎮 GAME
  - DOMAIN-KEYWORD,mobilelegends,🎮 GAME
  - DOMAIN-KEYWORD,pubgmobile,🎮 GAME
  - DOMAIN-KEYWORD,freefire,🎮 GAME
  - DOMAIN-SUFFIX,garena.com,🎮 GAME
  - DOMAIN-SUFFIX,steampowered.com,🎮 GAME
  - DOMAIN-SUFFIX,steamcommunity.com,🎮 GAME
  - DOMAIN-SUFFIX,riotgames.com,🎮 GAME
  - DOMAIN-SUFFIX,hoyoverse.com,🎮 GAME
  - DOMAIN-SUFFIX,mihoyo.com,🎮 GAME
  - DOMAIN-SUFFIX,roblox.com,🎮 GAME

  # Twitter / X (Termasuk CDN Gambar & Streaming Video)
  - DOMAIN-KEYWORD,twitter,📺 STREAMING & SOSMED
  - DOMAIN-KEYWORD,twimg,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,x.com,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,twitter.com,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,twimg.com,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,t.co,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,periscope.tv,📺 STREAMING & SOSMED

  # Spotify (Termasuk Web Player, Aplikasi & Seluruh CDN Audio/Cover Album)
  - DOMAIN-KEYWORD,spotify,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,spotify.com,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,spotifycdn.com,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,spotifycdn.net,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,spoti.fi,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,scdn.co,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,pscdn.co,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,audio-ak-spotify-com.akamaized.net,📺 STREAMING & SOSMED

  # Streaming Video & Sosmed Lainnya (YouTube, TikTok, Instagram, Facebook, Netflix, Bstation)
  - DOMAIN-KEYWORD,youtube,📺 STREAMING & SOSMED
  - DOMAIN-KEYWORD,googlevideo,📺 STREAMING & SOSMED
  - DOMAIN-KEYWORD,tiktok,📺 STREAMING & SOSMED
  - DOMAIN-KEYWORD,instagram,📺 STREAMING & SOSMED
  - DOMAIN-KEYWORD,facebook,📺 STREAMING & SOSMED
  - DOMAIN-KEYWORD,netflix,📺 STREAMING & SOSMED
  - DOMAIN-KEYWORD,bilibili,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,byteoversea.com,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,ibytedtos.com,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,fbcdn.net,📺 STREAMING & SOSMED
  - DOMAIN-SUFFIX,cdninstagram.com,📺 STREAMING & SOSMED

  # Web Font & CDN Populer (Bypass Cloudflare Block pada Datacenter Proxy)
  - DOMAIN-SUFFIX,dafont.com,DIRECT
  - DOMAIN-KEYWORD,dafont,DIRECT

  # Indo Lokal / M-Banking / Marketplace (Direct / Tanpa Kuota VPN)
  - DOMAIN-SUFFIX,id,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,bca,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,mandiri,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,bri,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,bni,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,tokopedia,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,shopee,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,bukalapak,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,dana.id,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,ovo.id,🇮🇩 INDO DIRECT
  - DOMAIN-KEYWORD,gopay,🇮🇩 INDO DIRECT
  - GEOIP,ID,🇮🇩 INDO DIRECT

  # Blokir Iklan
  - DOMAIN-KEYWORD,adservice,🛡️ IKLAN / ADS
  - DOMAIN-KEYWORD,googleads,🛡️ IKLAN / ADS
  - DOMAIN-KEYWORD,doubleclick,🛡️ IKLAN / ADS
  - DOMAIN-KEYWORD,adsense,🛡️ IKLAN / ADS

  # Aturan Default / Sisanya
  - MATCH,🚀 PROXY UTAMA
`

const clashBFRStandardTemplateYaml = `# =================================================================
# ProjectX WebUI - Template Config Clash / Mihomo untuk BFR
# Mode: Standar (All-in-One Proxy / Tanpa Pisah Trafik)
# Support: External Controller (0.0.0.0:9090) & DNS Android Fake-IP
# =================================================================

mixed-port: 7890
tproxy-port: 9898
redir-port: 9797
allow-lan: true
bind-address: '*'
mode: rule
log-level: info
ipv6: false
external-controller: 0.0.0.0:9090
external-ui: ./dashboard
secret: ''

dns:
  enable: true
  listen: 0.0.0.0:1053
  ipv6: false
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  fake-ip-filter:
    - '*.lan'
    - '*.local'
    - 'localhost.ptlogin2.qq.com'
    - '+.msftconnecttest.com'
    - '+.msftncsi.com'
  nameserver:
    - https://dns.google/dns-query
    - https://cloudflare-dns.com/dns-query
    - 8.8.8.8
    - 1.1.1.1
  fallback:
    - 8.8.4.4
    - 1.0.0.1
    - https://dns.quad9.net/dns-query

sniffer:
  enable: true
  parse-pure-ip: true
  sniff:
    TLS:
      ports: [443, 8443]
    HTTP:
      ports: [80, 8080-8880]
      override-destination: true
    QUIC:
      ports: [443]
  skip-domain:
    - 'Mijia Cloud'
    - 'dlg.io.mi.com'

proxy-providers:
  my_proxies:
    type: file
    path: ./proxy_provider/my_proxies.yaml
    health-check:
      enable: true
      interval: 300
      url: http://www.gstatic.com/generate_204

proxy-groups:
  - name: 🚀 PROXY UTAMA
    type: select
    use:
      - my_proxies
    proxies:
      - ♻️ AUTO PING
      - DIRECT

  - name: ♻️ AUTO PING
    type: url-test
    use:
      - my_proxies
    url: http://www.gstatic.com/generate_204
    interval: 300
    tolerance: 50

rules:
  # 🚫 Blokir Protokol QUIC (UDP 443 & UDP 80)
  # Memaksa Twitter/X Video, Spotify Web, dan Web Cloudflare (Dafont dll) instan fallback ke TCP
  - AND,((NETWORK,udp),(DST-PORT,443)),REJECT
  - AND,((NETWORK,udp),(DST-PORT,80)),REJECT
  - DOMAIN-SUFFIX,dafont.com,DIRECT
  - DOMAIN-KEYWORD,dafont,DIRECT
  - GEOIP,LAN,DIRECT
  - MATCH,🚀 PROXY UTAMA
`

func detectConfigMode() string {
	configPath := clashDefaultConfig
	if _, err := os.Stat(clashBaseDir); os.IsNotExist(err) {
		configPath = filepath.Join("scratch", "clash", "config.yaml")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "none"
	}
	content := string(data)
	if strings.Contains(content, "🎮 GAME") || strings.Contains(content, "📺 STREAMING") {
		return "split"
	}
	if strings.Contains(content, "proxy-groups:") && strings.Contains(content, "rules:") {
		return "standard"
	}
	return "none"
}

func checkClashConfigNeedsTemplate() bool {
	configPath := clashDefaultConfig
	if _, err := os.Stat(clashBaseDir); os.IsNotExist(err) {
		configPath = filepath.Join("scratch", "clash", "config.yaml")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return true // File belum ada
	}

	content := string(data)
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return true
	}

	// Cek apakah hanya berisi komentar (#)
	lines := strings.Split(trimmed, "\n")
	hasNonComment := false
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#") {
			hasNonComment = true
			break
		}
	}
	if !hasNonComment {
		return true
	}

	// Cek apakah memiliki struktur proxy-groups dan rules
	if !strings.Contains(content, "proxy-groups:") || !strings.Contains(content, "rules:") {
		return true
	}

	return false
}

// Endpoint GET /api/clash/targets
func handleClashTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	targets := getClashTargets()
	needsTemplate := checkClashConfigNeedsTemplate()
	configMode := detectConfigMode()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"targets":        targets,
		"needs_template": needsTemplate,
		"config_mode":    configMode,
	})
}

// Endpoint POST /api/clash/template/apply
func handleClashTemplateApply(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"status": "error", "message": "Method not allowed"}`))
		return
	}

	var req struct {
		Mode string `json:"mode"` // "split" or "standard"
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode != "standard" {
		mode = "split"
	}

	templateContent := clashBFRTemplateYaml
	modeLabel := "Pisah Trafik (Multi-Routing)"
	if mode == "standard" {
		templateContent = clashBFRStandardTemplateYaml
		modeLabel = "Standar (All-in-One Proxy)"
	}

	baseDir := clashBaseDir
	configPath := clashDefaultConfig
	providerDir := clashProviderDir
	if _, err := os.Stat(clashBaseDir); os.IsNotExist(err) {
		baseDir = filepath.Join("scratch", "clash")
		configPath = filepath.Join(baseDir, "config.yaml")
		providerDir = filepath.Join(baseDir, "proxy_provider")
	}

	_ = os.MkdirAll(baseDir, 0755)
	_ = os.MkdirAll(providerDir, 0755)

	// Backup file config yang ada jika memiliki konten
	if curData, err := os.ReadFile(configPath); err == nil && len(strings.TrimSpace(string(curData))) > 0 {
		bakPath := configPath + ".bak"
		_ = os.WriteFile(bakPath, curData, 0644)
	}

	// Tulis Template Config Lengkap Sesuai Mode Pilihan
	if err := os.WriteFile(configPath, []byte(templateContent), 0644); err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Gagal menulis config.yaml: %v"}`, err)))
		return
	}

	// Buat default provider file jika belum ada
	defaultProviderFile := filepath.Join(providerDir, "my_proxies.yaml")
	if _, err := os.Stat(defaultProviderFile); os.IsNotExist(err) {
		_ = os.WriteFile(defaultProviderFile, []byte("proxies:\n"), 0644)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"message":          fmt.Sprintf("Template %s berhasil dipasang!", modeLabel),
		"config_path":      configPath,
		"default_provider": defaultProviderFile,
		"mode":             mode,
	})
}

// Endpoint POST /api/clash/targets/create
func handleClashTargetCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"status": "error", "message": "Method not allowed"}`))
		return
	}

	var req ClashTargetCreateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		w.Write([]byte(`{"status": "error", "message": "Nama file provider tidak boleh kosong"}`))
		return
	}

	// Bersihkan nama file
	name = strings.ReplaceAll(name, " ", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)
	name = reg.ReplaceAllString(name, "")
	if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
		name += ".yaml"
	}

	baseDir := clashProviderDir
	if _, err := os.Stat(clashBaseDir); os.IsNotExist(err) {
		baseDir = filepath.Join("scratch", "clash", "proxy_provider")
	}

	_ = os.MkdirAll(baseDir, 0755)
	targetPath := filepath.Join(baseDir, name)

	if _, err := os.Stat(targetPath); err == nil {
		w.Write([]byte(`{"status": "error", "message": "File provider dengan nama ini sudah ada"}`))
		return
	}

	initialContent := "proxies:\n"
	if err := os.WriteFile(targetPath, []byte(initialContent), 0644); err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Gagal membuat file: %v"}`, err)))
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("File provider %s berhasil dibuat!", name),
		"path":    targetPath,
		"label":   "Provider: " + name,
	})
}

// Endpoint GET /api/clash/accounts?target=...
func handleClashAccountsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	targetPath := r.URL.Query().Get("target")
	if targetPath == "" {
		// Gunakan default target pertama
		targets := getClashTargets()
		if len(targets) > 0 {
			targetPath = targets[0].Path
		}
	}

	cleanTarget := filepath.Clean(targetPath)
	if !isAllowedClashPath(cleanTarget) {
		w.Write([]byte(`{"status": "error", "message": "Lokasi target di luar batas aman sistem."}`))
		return
	}

	if _, err := os.Stat(cleanTarget); os.IsNotExist(err) {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "File target '%s' tidak ditemukan."}`, filepath.Base(cleanTarget))))
		return
	}

	_, seqNode, err := loadTargetYaml(cleanTarget)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Gagal membaca konfigurasi: %v"}`, err)))
		return
	}

	var accounts []ProxyItem
	stats := make(map[string]int)

	for idx, pNode := range seqNode.Content {
		item, err := parseProxyNode(pNode, idx)
		if err == nil {
			accounts = append(accounts, item)
			pType := item.Type
			if pType == "" {
				pType = "unknown"
			}
			stats[pType]++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"target":   cleanTarget,
		"total":    len(accounts),
		"stats":    stats,
		"accounts": accounts,
	})
}

// Endpoint POST /api/clash/accounts/add
func handleClashAccountsAdd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"status": "error", "message": "Method not allowed"}`))
		return
	}

	var req ClashAccountAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Write([]byte(`{"status": "error", "message": "Format request JSON tidak valid"}`))
		return
	}

	cleanTarget := filepath.Clean(req.TargetFile)
	if !isAllowedClashPath(cleanTarget) {
		w.Write([]byte(`{"status": "error", "message": "Lokasi target di luar batas aman sistem."}`))
		return
	}

	root, seqNode, err := loadTargetYaml(cleanTarget)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "%v"}`, err)))
		return
	}

	addedCount := 0

	// 1. Jika dikirim via raw YamlContent
	if strings.TrimSpace(req.YamlContent) != "" {
		var parsedNode yaml.Node
		if err := yaml.Unmarshal([]byte(req.YamlContent), &parsedNode); err == nil {
			if len(parsedNode.Content) > 0 {
				top := parsedNode.Content[0]
				if top.Kind == yaml.SequenceNode {
					for _, itemNode := range top.Content {
						seqNode.Content = append(seqNode.Content, itemNode)
						addedCount++
					}
				} else if top.Kind == yaml.MappingNode {
					seqNode.Content = append(seqNode.Content, top)
					addedCount++
				}
			}
		}
	}

	// 2. Jika dikirim via single Item
	if req.Item != nil {
		yamlStr := buildProxyYamlString(*req.Item)
		var tmp yaml.Node
		if err := yaml.Unmarshal([]byte(yamlStr), &tmp); err == nil && len(tmp.Content) > 0 {
			top := tmp.Content[0]
			if top.Kind == yaml.SequenceNode && len(top.Content) > 0 {
				seqNode.Content = append(seqNode.Content, top.Content[0])
				addedCount++
			} else if top.Kind == yaml.MappingNode {
				seqNode.Content = append(seqNode.Content, top)
				addedCount++
			}
		}
	}

	// 3. Jika dikirim via array of Items
	for _, item := range req.Items {
		yamlStr := buildProxyYamlString(item)
		var tmp yaml.Node
		if err := yaml.Unmarshal([]byte(yamlStr), &tmp); err == nil && len(tmp.Content) > 0 {
			top := tmp.Content[0]
			if top.Kind == yaml.SequenceNode && len(top.Content) > 0 {
				seqNode.Content = append(seqNode.Content, top.Content[0])
				addedCount++
			} else if top.Kind == yaml.MappingNode {
				seqNode.Content = append(seqNode.Content, top)
				addedCount++
			}
		}
	}

	if addedCount == 0 {
		w.Write([]byte(`{"status": "error", "message": "Tidak ada node akun yang valid untuk ditambahkan."}`))
		return
	}

	if err := saveTargetYaml(cleanTarget, root); err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Gagal menyimpan berkas: %v"}`, err)))
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Berhasil menambahkan %d akun ke %s!", addedCount, filepath.Base(cleanTarget)),
		"total":   len(seqNode.Content),
	})
}

// Endpoint POST /api/clash/accounts/update
func handleClashAccountsUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"status": "error", "message": "Method not allowed"}`))
		return
	}

	var req ClashAccountUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Write([]byte(`{"status": "error", "message": "Format request JSON tidak valid"}`))
		return
	}

	cleanTarget := filepath.Clean(req.TargetFile)
	if !isAllowedClashPath(cleanTarget) {
		w.Write([]byte(`{"status": "error", "message": "Lokasi target di luar batas aman sistem."}`))
		return
	}

	root, seqNode, err := loadTargetYaml(cleanTarget)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "%v"}`, err)))
		return
	}

	targetIndex := -1
	// 1. Prioritaskan pencarian berdasarkan OriginalName jika ada
	if req.OriginalName != "" {
		for idx, pNode := range seqNode.Content {
			item, _ := parseProxyNode(pNode, idx)
			if item.Name == req.OriginalName {
				targetIndex = idx
				break
			}
		}
	}

	// 2. Jika tidak ditemukan lewat OriginalName, gunakan Index jika valid
	if targetIndex == -1 {
		if req.Index >= 0 && req.Index < len(seqNode.Content) {
			targetIndex = req.Index
		}
	}

	if targetIndex == -1 {
		w.Write([]byte(`{"status": "error", "message": "Akun tidak ditemukan untuk diperbarui."}`))
		return
	}

	// Bangun node baru dari YamlContent atau Item
	var newNode *yaml.Node

	if strings.TrimSpace(req.YamlContent) != "" {
		var tmp yaml.Node
		content := strings.TrimSpace(req.YamlContent)
		if err := yaml.Unmarshal([]byte(content), &tmp); err == nil && len(tmp.Content) > 0 {
			top := tmp.Content[0]
			if top.Kind == yaml.MappingNode {
				newNode = top
			} else if top.Kind == yaml.SequenceNode && len(top.Content) > 0 {
				newNode = top.Content[0]
			}
		} else {
			wrapped := "- " + strings.ReplaceAll(content, "\n", "\n  ")
			if err := yaml.Unmarshal([]byte(wrapped), &tmp); err == nil && len(tmp.Content) > 0 {
				top := tmp.Content[0]
				if top.Kind == yaml.SequenceNode && len(top.Content) > 0 {
					newNode = top.Content[0]
				} else if top.Kind == yaml.MappingNode {
					newNode = top
				}
			}
		}
	} else if req.Item != nil {
		itemCopy := *req.Item
		itemCopy.RawYaml = "" // Pastikan dibangun dari atribut-atribut baru yang diedit, bukan raw YAML lama
		yamlStr := buildProxyYamlString(itemCopy)
		var tmp yaml.Node
		if err := yaml.Unmarshal([]byte(yamlStr), &tmp); err == nil && len(tmp.Content) > 0 {
			top := tmp.Content[0]
			if top.Kind == yaml.SequenceNode && len(top.Content) > 0 {
				newNode = top.Content[0]
			} else if top.Kind == yaml.MappingNode {
				newNode = top
			}
		}
	}

	if newNode == nil {
		w.Write([]byte(`{"status": "error", "message": "Format konfigurasi akun yang diperbarui tidak valid."}`))
		return
	}

	seqNode.Content[targetIndex] = newNode

	if err := saveTargetYaml(cleanTarget, root); err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Gagal menyimpan berkas: %v"}`, err)))
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Akun berhasil diperbarui!",
	})
}

// Endpoint POST /api/clash/accounts/delete
func handleClashAccountsDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"status": "error", "message": "Method not allowed"}`))
		return
	}

	var req ClashAccountDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Write([]byte(`{"status": "error", "message": "Format request JSON tidak valid"}`))
		return
	}

	cleanTarget := filepath.Clean(req.TargetFile)
	if !isAllowedClashPath(cleanTarget) {
		w.Write([]byte(`{"status": "error", "message": "Lokasi target di luar batas aman sistem."}`))
		return
	}

	root, seqNode, err := loadTargetYaml(cleanTarget)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "%v"}`, err)))
		return
	}

	targetIndex := req.Index
	if targetIndex < 0 || targetIndex >= len(seqNode.Content) {
		found := false
		if req.Name != "" {
			for idx, pNode := range seqNode.Content {
				item, _ := parseProxyNode(pNode, idx)
				if item.Name == req.Name {
					targetIndex = idx
					found = true
					break
				}
			}
		}
		if !found {
			w.Write([]byte(`{"status": "error", "message": "Akun yang ingin dihapus tidak ditemukan."}`))
			return
		}
	}

	// Hapus item pada targetIndex
	seqNode.Content = append(seqNode.Content[:targetIndex], seqNode.Content[targetIndex+1:]...)

	if err := saveTargetYaml(cleanTarget, root); err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Gagal menyimpan berkas: %v"}`, err)))
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Akun berhasil dihapus!",
		"total":   len(seqNode.Content),
	})
}

// Endpoint POST /api/clash/save (Legacy compatibility)
func handleClashSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"status": "error", "message": "Method not allowed"}`))
		return
	}

	var targetFile, yamlContent string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req ClashSaveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Write([]byte(`{"status": "error", "message": "Gagal: Format JSON tidak valid."}`))
			return
		}
		targetFile = req.TargetFile
		yamlContent = req.YamlContent
	} else {
		if err := r.ParseForm(); err == nil {
			targetFile = r.FormValue("target_file")
			yamlContent = r.FormValue("yaml_content")
		}
	}

	yamlContent = strings.TrimSpace(yamlContent)
	if yamlContent == "" {
		w.Write([]byte(`{"status": "error", "message": "Gagal: Konten YAML kosong."}`))
		return
	}

	cleanTarget := filepath.Clean(targetFile)
	if !isAllowedClashPath(cleanTarget) {
		w.Write([]byte(`{"status": "error", "message": "Gagal: Lokasi penyimpanan berada di luar batas aman sistem."}`))
		return
	}

	root, seqNode, err := loadTargetYaml(cleanTarget)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "%v"}`, err)))
		return
	}

	var parsedNode yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &parsedNode); err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Format YAML tidak valid: %v"}`, err)))
		return
	}

	addedCount := 0
	if len(parsedNode.Content) > 0 {
		top := parsedNode.Content[0]
		if top.Kind == yaml.SequenceNode {
			for _, itemNode := range top.Content {
				seqNode.Content = append(seqNode.Content, itemNode)
				addedCount++
			}
		} else if top.Kind == yaml.MappingNode {
			seqNode.Content = append(seqNode.Content, top)
			addedCount++
		}
	}

	if addedCount == 0 {
		w.Write([]byte(`{"status": "error", "message": "Tidak ada item proxy yang terdeteksi."}`))
		return
	}

	if err := saveTargetYaml(cleanTarget, root); err != nil {
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "message": "Gagal menulis berkas: %v"}`, err)))
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Sukses: %d konfigurasi akun ditambahkan ke %s", addedCount, filepath.Base(cleanTarget)),
	})
}
