package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Cloudflare Anycast IPs cadangan jika DNS sistem Android / VPN bermasalah (misal IPv6 loopback [::1]:53 mati)
var cfSpeedtestFallbackIPs = []string{
	"162.159.140.220",
	"172.66.0.218",
	"104.16.1.1",
}

// robustResolver menangani kegagalan DNS khas Android saat VPN / Clash / Sing-box aktif
// Mencegah error 'dial udp [::1]:53: connect: cannot assign requested address'
var robustResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 2 * time.Second}

		var candidates []string

		// 1. Coba address yang diminta HANYA JIKA bukan IPv6 [::1] loopback yang rusak
		if !strings.HasPrefix(address, "[") && !strings.Contains(address, "::") {
			candidates = append(candidates, address)
		}

		// 2. Baca DNS sistem Android dari getprop
		for _, prop := range []string{"net.dns1", "net.dns2", "net.dns3"} {
			if val := strings.TrimSpace(getProp(prop)); val != "" && !strings.Contains(val, ":") {
				candidates = append(candidates, val+":53")
			}
		}

		// 3. DNS lokal VPN / BFR / Clash / Sing-box / dnsmasq
		candidates = append(candidates, "127.0.0.1:53")

		// 4. DNS publik IPv4 terpercaya
		candidates = append(candidates, "1.1.1.1:53", "8.8.8.8:53", "9.9.9.9:53")

		var lastErr error
		for _, srv := range candidates {
			conn, err := d.DialContext(ctx, "udp", srv)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}

		// Jika UDP terblokir oleh firewall / routing, coba via TCP
		for _, srv := range []string{"1.1.1.1:53", "8.8.8.8:53"} {
			conn, err := d.DialContext(ctx, "tcp", srv)
			if err == nil {
				return conn, nil
			}
		}

		return nil, lastErr
	},
}

func newSpeedtestTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  robustResolver,
	}

	return &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		ForceAttemptHTTP2:   true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
				port = "443"
			}

			// 1. Coba koneksi normal menggunakan robustResolver
			conn, err := dialer.DialContext(ctx, network, addr)
			if err == nil {
				return conn, nil
			}

			// 2. Jika resolusi DNS gagal (terutama di Android dengan VPN),
			// langsung hubungkan ke Cloudflare Anycast IP fallback dengan SNI speed.cloudflare.com
			if host == "speed.cloudflare.com" {
				for _, fbIP := range cfSpeedtestFallbackIPs {
					target := net.JoinHostPort(fbIP, port)
					conn2, err2 := dialer.DialContext(ctx, network, target)
					if err2 == nil {
						return conn2, nil
					}
				}
			}

			return nil, err
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}

// SpeedtestResult menyimpan ringkasan hasil pengujian terakhir
type SpeedtestResult struct {
	Timestamp string  `json:"timestamp"`
	Ping      float64 `json:"ping"`     // ms
	Jitter    float64 `json:"jitter"`   // ms
	Download  float64 `json:"download"` // Mbps
	Upload    float64 `json:"upload"`   // Mbps
	IP        string  `json:"ip"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Colo      string  `json:"colo"`
	ISP       string  `json:"isp"`
}

type SpeedtestProgress struct {
	Phase        string           `json:"phase"` // "init", "ping", "download", "upload", "complete", "stopped", "error"
	Progress     int              `json:"progress"`
	CurrentSpeed float64          `json:"current_speed,omitempty"`
	Speed        float64          `json:"speed,omitempty"`
	Bytes        int64            `json:"bytes,omitempty"`
	Ping         float64          `json:"ping,omitempty"`
	Jitter       float64          `json:"jitter,omitempty"`
	Download     float64          `json:"download,omitempty"`
	Upload       float64          `json:"upload,omitempty"`
	IP           string           `json:"ip,omitempty"`
	City         string           `json:"city,omitempty"`
	Country      string           `json:"country,omitempty"`
	Colo         string           `json:"colo,omitempty"`
	ISP          string           `json:"isp,omitempty"`
	Message      string           `json:"message,omitempty"`
	Result       *SpeedtestResult `json:"result,omitempty"`
}

var (
	speedtestMu         sync.Mutex
	speedtestCancel     context.CancelFunc
	speedtestRunning    bool
	lastSpeedtestResult *SpeedtestResult
	lastResultMu        sync.RWMutex
)

// Mapping ASN populer Indonesia sebagai fallback nama ISP
var knownASNs = map[string]string{
	"17974":  "Telkomsel",
	"7713":   "Telkom IndiHome",
	"18004":  "Telkom Indonesia",
	"24203":  "Indosat Ooredoo Hutchison",
	"4761":   "Indosat IM2",
	"24558":  "XL Axiata",
	"131759": "Smartfren",
	"17451":  "Biznet Networks",
	"24204":  "First Media / Link Net",
	"38081":  "MyRepublic",
	"9907":   "Oxygen.id",
	"13335":  "Cloudflare 1.1.1.1 / WARP",
}

func getFallbackISP(asn, defaultVal string) string {
	cleanASN := strings.TrimPrefix(strings.ToUpper(asn), "AS")
	if name, ok := knownASNs[cleanASN]; ok {
		return name
	}
	if asn != "" {
		return "AS" + cleanASN
	}
	return defaultVal
}

// Handler untuk menghentikan speedtest yang sedang berjalan
func handleSpeedtestStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":"error","message":"Unauthorized"}`))
		return
	}

	speedtestMu.Lock()
	if speedtestCancel != nil {
		speedtestCancel()
		speedtestCancel = nil
	}
	speedtestRunning = false
	speedtestMu.Unlock()

	w.Write([]byte(`{"status":"success","message":"Pengujian dihentikan"}`))
}

// Handler untuk mendapatkan hasil speedtest terakhir
func handleSpeedtestLast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":"error","message":"Unauthorized"}`))
		return
	}

	lastResultMu.RLock()
	defer lastResultMu.RUnlock()

	if lastSpeedtestResult == nil {
		w.Write([]byte(`{"status":"success","data":null}`))
		return
	}

	resBytes, _ := json.Marshal(map[string]interface{}{
		"status": "success",
		"data":   lastSpeedtestResult,
	})
	w.Write(resBytes)
}

// Handler SSE Streaming untuk Speedtest
func handleSpeedtestStream(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":"error","message":"Unauthorized"}`))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sendSSE := func(prog SpeedtestProgress) {
		data, err := json.Marshal(prog)
		if err == nil {
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}

	speedtestMu.Lock()
	if speedtestRunning {
		// Batalkan tes sebelumnya jika ada klien baru yang mulai
		if speedtestCancel != nil {
			speedtestCancel()
		}
	}
	speedtestRunning = true
	ctx, cancel := context.WithCancel(r.Context())
	speedtestCancel = cancel
	speedtestMu.Unlock()

	defer func() {
		speedtestMu.Lock()
		speedtestRunning = false
		speedtestCancel = nil
		speedtestMu.Unlock()
	}()

	client := &http.Client{
		Timeout:   0, // Timeout diatur per-fase via context
		Transport: newSpeedtestTransport(),
	}

	// ==========================================
	// FASE 1: Inisialisasi & Metadata
	// ==========================================
	sendSSE(SpeedtestProgress{Phase: "init", Progress: 3, Message: "Menghubungkan ke server Cloudflare..."})

	var ip, city, country, colo, asn, isp string

	metaReq, err := http.NewRequestWithContext(ctx, "GET", "https://speed.cloudflare.com/__down?bytes=0", nil)
	if err != nil {
		sendSSE(SpeedtestProgress{Phase: "error", Message: "Gagal membuat koneksi: " + err.Error()})
		return
	}
	metaResp, err := client.Do(metaReq)
	if err != nil {
		if ctx.Err() != nil {
			sendSSE(SpeedtestProgress{Phase: "stopped", Message: "Pengujian dibatalkan"})
		} else {
			sendSSE(SpeedtestProgress{Phase: "error", Message: "Koneksi terputus: " + err.Error()})
		}
		return
	}
	metaResp.Body.Close()

	ip = metaResp.Header.Get("Cf-Meta-Ip")
	city = metaResp.Header.Get("City")
	country = metaResp.Header.Get("Country")
	colo = metaResp.Header.Get("Colo")
	asn = metaResp.Header.Get("Asn")
	isp = getFallbackISP(asn, "Cloudflare Edge")

	// Coba cari nama ISP lengkap via ip-api.com (timeout cepat 1.2s agar tidak menghambat)
	go func() {
		apiCtx, apiCancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
		defer apiCancel()
		apiReq, err := http.NewRequestWithContext(apiCtx, "GET", "http://ip-api.com/json/"+ip+"?fields=isp,city,country", nil)
		if err == nil {
			apiClient := &http.Client{
				Timeout: 1200 * time.Millisecond,
				Transport: &http.Transport{
					DialContext: (&net.Dialer{
						Timeout:  1200 * time.Millisecond,
						Resolver: robustResolver,
					}).DialContext,
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
			apiResp, err := apiClient.Do(apiReq)
			if err == nil {
				defer apiResp.Body.Close()
				var apiData struct {
					ISP     string `json:"isp"`
					City    string `json:"city"`
					Country string `json:"country"`
				}
				if json.NewDecoder(apiResp.Body).Decode(&apiData) == nil && apiData.ISP != "" {
					isp = apiData.ISP
					if city == "" && apiData.City != "" {
						city = apiData.City
					}
				}
			}
		}
	}()

	sendSSE(SpeedtestProgress{
		Phase:    "init",
		Progress: 6,
		IP:       ip,
		City:     city,
		Country:  country,
		Colo:     colo,
		ISP:      isp,
		Message:  "Server terdeteksi: " + colo + " (" + city + ")",
	})

	// ==========================================
	// FASE 2: Latensi (Ping) & Jitter
	// ==========================================
	var pings []float64
	pingSamples := 6
	for i := 0; i < pingSamples; i++ {
		if ctx.Err() != nil {
			sendSSE(SpeedtestProgress{Phase: "stopped", Message: "Pengujian dibatalkan"})
			return
		}
		pStart := time.Now()
		pReq, err := http.NewRequestWithContext(ctx, "GET", "https://speed.cloudflare.com/__down?bytes=0", nil)
		if err != nil {
			break
		}
		pResp, err := client.Do(pReq)
		if err != nil {
			continue
		}
		pResp.Body.Close()
		rtt := float64(time.Since(pStart).Microseconds()) / 1000.0 // ms
		pings = append(pings, rtt)

		// Hitung rata-rata ping & jitter saat ini
		currentPing := rtt
		currentJitter := 0.0
		if len(pings) > 1 {
			diffSum := 0.0
			for j := 1; j < len(pings); j++ {
				diff := pings[j] - pings[j-1]
				if diff < 0 {
					diff = -diff
				}
				diffSum += diff
			}
			currentJitter = diffSum / float64(len(pings)-1)
		}

		sendSSE(SpeedtestProgress{
			Phase:    "ping",
			Progress: 6 + int(float64(i+1)/float64(pingSamples)*14),
			Ping:     roundFloat(currentPing, 1),
			Jitter:   roundFloat(currentJitter, 1),
			IP:       ip,
			City:     city,
			Country:  country,
			Colo:     colo,
			ISP:      isp,
		})
		time.Sleep(60 * time.Millisecond)
	}

	finalPing := 0.0
	finalJitter := 0.0
	if len(pings) > 0 {
		sum := 0.0
		minPing := pings[0]
		for _, p := range pings {
			sum += p
			if p < minPing {
				minPing = p
			}
		}
		finalPing = roundFloat(minPing, 1) // Gunakan best ping
		if len(pings) > 1 {
			diffSum := 0.0
			for j := 1; j < len(pings); j++ {
				diff := pings[j] - pings[j-1]
				if diff < 0 {
					diff = -diff
				}
				diffSum += diff
			}
			finalJitter = roundFloat(diffSum/float64(len(pings)-1), 1)
		}
	}

	sendSSE(SpeedtestProgress{
		Phase:    "ping",
		Progress: 20,
		Ping:     finalPing,
		Jitter:   finalJitter,
		IP:       ip,
		City:     city,
		Country:  country,
		Colo:     colo,
		ISP:      isp,
	})

	// ==========================================
	// FASE 3: Uji Unduh (Download)
	// ==========================================
	// Target durasi download: ~6 detik
	downloadTargetDuration := 6 * time.Second
	dlOverallStart := time.Now()
	var totalDlBytes int64
	var lastDlUpdate time.Time = time.Now()
	var windowDlBytes int64
	var windowDlStart time.Time = time.Now()
	var currentDlSpeed float64

	// Chunk ukuran bertahap: 1MB -> 2.5MB -> 5MB -> 10MB
	chunkSizes := []int{1000000, 2500000, 5000000, 10000000, 15000000}
	chunkIdx := 0

	for time.Since(dlOverallStart) < downloadTargetDuration {
		if ctx.Err() != nil {
			sendSSE(SpeedtestProgress{Phase: "stopped", Message: "Pengujian dibatalkan"})
			return
		}

		cSize := chunkSizes[chunkIdx]
		if chunkIdx < len(chunkSizes)-1 {
			chunkIdx++
		}

		dlUrl := "https://speed.cloudflare.com/__down?bytes=" + strconv.Itoa(cSize)
		dlReq, err := http.NewRequestWithContext(ctx, "GET", dlUrl, nil)
		if err != nil {
			break
		}
		dlResp, err := client.Do(dlReq)
		if err != nil {
			if ctx.Err() != nil {
				sendSSE(SpeedtestProgress{Phase: "stopped", Message: "Pengujian dibatalkan"})
				return
			}
			break
		}

		buf := make([]byte, 32*1024)
		for {
			if ctx.Err() != nil {
				dlResp.Body.Close()
				sendSSE(SpeedtestProgress{Phase: "stopped", Message: "Pengujian dibatalkan"})
				return
			}
			n, rErr := dlResp.Body.Read(buf)
			if n > 0 {
				totalDlBytes += int64(n)
				windowDlBytes += int64(n)

				now := time.Now()
				if now.Sub(lastDlUpdate) >= 120*time.Millisecond {
					wDur := now.Sub(windowDlStart).Seconds()
					if wDur > 0 {
						currentDlSpeed = (float64(windowDlBytes) * 8.0) / (wDur * 1e6)
					}
					totDur := now.Sub(dlOverallStart).Seconds()
					avgSpeed := 0.0
					if totDur > 0 {
						avgSpeed = (float64(totalDlBytes) * 8.0) / (totDur * 1e6)
					}
					elapsedRatio := totDur / downloadTargetDuration.Seconds()
					if elapsedRatio > 1.0 {
						elapsedRatio = 1.0
					}

					sendSSE(SpeedtestProgress{
						Phase:        "download",
						Progress:     20 + int(elapsedRatio*40.0),
						CurrentSpeed: roundFloat(currentDlSpeed, 2),
						Speed:        roundFloat(avgSpeed, 2),
						Bytes:        totalDlBytes,
						Ping:         finalPing,
						Jitter:       finalJitter,
						IP:           ip,
						City:         city,
						Country:      country,
						Colo:         colo,
						ISP:          isp,
					})

					lastDlUpdate = now
					windowDlStart = now
					windowDlBytes = 0
				}
			}
			if rErr != nil {
				break
			}
			if time.Since(dlOverallStart) >= downloadTargetDuration {
				break
			}
		}
		dlResp.Body.Close()
	}

	finalDlDuration := time.Since(dlOverallStart).Seconds()
	finalDownload := 0.0
	if finalDlDuration > 0 && totalDlBytes > 0 {
		finalDownload = roundFloat((float64(totalDlBytes)*8.0)/(finalDlDuration*1e6), 2)
	}

	sendSSE(SpeedtestProgress{
		Phase:        "download",
		Progress:     60,
		CurrentSpeed: finalDownload,
		Speed:        finalDownload,
		Download:     finalDownload,
		Ping:         finalPing,
		Jitter:       finalJitter,
		IP:           ip,
		City:         city,
		Country:      country,
		Colo:         colo,
		ISP:          isp,
	})

	time.Sleep(100 * time.Millisecond)

	// ==========================================
	// FASE 4: Uji Unggah (Upload)
	// ==========================================
	// Target durasi upload: ~5 detik
	uploadTargetDuration := 5 * time.Second
	ulOverallStart := time.Now()
	var totalUlBytes int64
	var lastUlUpdate time.Time = time.Now()
	var windowUlBytes int64
	var windowUlStart time.Time = time.Now()
	var currentUlSpeed float64

	// Siapkan buffer upload 512KB di memori
	ulBufferSize := 512 * 1024
	ulPayload := make([]byte, ulBufferSize)
	rand.Read(ulPayload)

	for time.Since(ulOverallStart) < uploadTargetDuration {
		if ctx.Err() != nil {
			sendSSE(SpeedtestProgress{Phase: "stopped", Message: "Pengujian dibatalkan"})
			return
		}

		// Bungkus buffer dengan reader yang melacak byte terkirim
		pipeReader, pipeWriter := io.Pipe()

		go func() {
			defer pipeWriter.Close()
			buf := make([]byte, 32*1024)
			r := bytes.NewReader(ulPayload)
			for {
				if ctx.Err() != nil {
					return
				}
				n, err := r.Read(buf)
				if n > 0 {
					_, wErr := pipeWriter.Write(buf[:n])
					if wErr != nil {
						return
					}
					totalUlBytes += int64(n)
					windowUlBytes += int64(n)

					now := time.Now()
					if now.Sub(lastUlUpdate) >= 120*time.Millisecond {
						wDur := now.Sub(windowUlStart).Seconds()
						if wDur > 0 {
							currentUlSpeed = (float64(windowUlBytes) * 8.0) / (wDur * 1e6)
						}
						totDur := now.Sub(ulOverallStart).Seconds()
						avgSpeed := 0.0
						if totDur > 0 {
							avgSpeed = (float64(totalUlBytes) * 8.0) / (totDur * 1e6)
						}
						elapsedRatio := totDur / uploadTargetDuration.Seconds()
						if elapsedRatio > 1.0 {
							elapsedRatio = 1.0
						}

						sendSSE(SpeedtestProgress{
							Phase:        "upload",
							Progress:     60 + int(elapsedRatio*38.0),
							CurrentSpeed: roundFloat(currentUlSpeed, 2),
							Speed:        roundFloat(avgSpeed, 2),
							Bytes:        totalUlBytes,
							Download:     finalDownload,
							Ping:         finalPing,
							Jitter:       finalJitter,
							IP:           ip,
							City:         city,
							Country:      country,
							Colo:         colo,
							ISP:          isp,
						})

						lastUlUpdate = now
						windowUlStart = now
						windowUlBytes = 0
					}
				}
				if err != nil {
					break
				}
			}
		}()

		ulReq, err := http.NewRequestWithContext(ctx, "POST", "https://speed.cloudflare.com/__up", pipeReader)
		if err != nil {
			break
		}
		ulReq.Header.Set("Content-Type", "application/octet-stream")
		ulResp, err := client.Do(ulReq)
		if err != nil {
			if ctx.Err() != nil {
				sendSSE(SpeedtestProgress{Phase: "stopped", Message: "Pengujian dibatalkan"})
				return
			}
			break
		}
		io.Copy(io.Discard, ulResp.Body)
		ulResp.Body.Close()
	}

	finalUlDuration := time.Since(ulOverallStart).Seconds()
	finalUpload := 0.0
	if finalUlDuration > 0 && totalUlBytes > 0 {
		finalUpload = roundFloat((float64(totalUlBytes)*8.0)/(finalUlDuration*1e6), 2)
	}

	// ==========================================
	// FASE 5: Selesai (Complete)
	// ==========================================
	completedResult := &SpeedtestResult{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Ping:      finalPing,
		Jitter:    finalJitter,
		Download:  finalDownload,
		Upload:    finalUpload,
		IP:        ip,
		City:      city,
		Country:   country,
		Colo:      colo,
		ISP:       isp,
	}

	lastResultMu.Lock()
	lastSpeedtestResult = completedResult
	lastResultMu.Unlock()

	sendSSE(SpeedtestProgress{
		Phase:    "complete",
		Progress: 100,
		Ping:     finalPing,
		Jitter:   finalJitter,
		Download: finalDownload,
		Upload:   finalUpload,
		IP:       ip,
		City:     city,
		Country:  country,
		Colo:     colo,
		ISP:      isp,
		Result:   completedResult,
		Message:  "Pengujian kecepatan selesai!",
	})
}

func roundFloat(val float64, precision int) float64 {
	format := fmt.Sprintf("%%.%df", precision)
	s := fmt.Sprintf(format, val)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

