package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CronJob merepresentasikan satu tugas terjadwal
type CronJob struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"` // Default OFF
	Time        string `json:"time"`    // HH:MM format, misal "04:00"
	Days        []int  `json:"days"`    // 0=Minggu, 1=Senin, ..., 6=Sabtu
	Type        string `json:"type"`    // "reboot", "soft_reboot", "clean_ram", "restart_bfr", "reconnect_data", "custom"
	Command     string `json:"command"` // Perintah custom shell jika type=="custom"
	LastRun     string `json:"last_run"`
	LastStatus  string `json:"last_status"` // "success", "failed", "never"
	LastOutput  string `json:"last_output"`
}

// CronConfig menyimpan pengaturan utama
type CronConfig struct {
	MasterEnabled bool      `json:"master_enabled"` // Default strictly FALSE (OFF)
	Jobs          []CronJob `json:"jobs"`
}

var (
	cronMu            sync.RWMutex
	cronConfigPath    string
	cronCfg           CronConfig
	cronLogs          = make([]string, 0, 50)
	cronWorkerRunning = false
	cronStopChan      chan struct{}
)

func getDefaultCronConfig() CronConfig {
	return CronConfig{
		MasterEnabled: false, // Default strictly OFF
		Jobs: []CronJob{
			{
				ID:          "reboot-daily",
				Title:       "Auto Reboot Terjadwal",
				Description: "Mulai ulang perangkat Android secara berkala untuk menjaga stabilitas & kesegaran memori HP.",
				Enabled:     false,
				Time:        "04:00",
				Days:        []int{0, 1, 2, 3, 4, 5, 6},
				Type:        "reboot",
				LastStatus:  "never",
			},
			{
				ID:          "clean-ram",
				Title:       "Auto Bersihkan RAM & Cache",
				Description: "Bebaskan pagecache, dentries, dan inodes kernel Linux (drop_caches).",
				Enabled:     false,
				Time:        "03:30",
				Days:        []int{0, 1, 2, 3, 4, 5, 6},
				Type:        "clean_ram",
				LastStatus:  "never",
			},
			{
				ID:          "restart-bfr",
				Title:       "Auto Restart BFR Tunnel",
				Description: "Restart core proxy Box (Clash/Sing-box) untuk merefresh koneksi & memori tunnel.",
				Enabled:     false,
				Time:        "05:00",
				Days:        []int{0, 1, 2, 3, 4, 5, 6},
				Type:        "restart_bfr",
				LastStatus:  "never",
			},
			{
				ID:          "reconnect-data",
				Title:       "Auto Refresh Sinyal Seluler",
				Description: "Reset data seluler sekejap untuk memperbarui alokasi IP publik dari operator.",
				Enabled:     false,
				Time:        "04:15",
				Days:        []int{0, 1, 2, 3, 4, 5, 6},
				Type:        "reconnect_data",
				LastStatus:  "never",
			},
		},
	}
}

func initCronPaths() {
	candDirs := []string{
		"/data/adb/modules/projectx-webui/data",
		"/data/adb/projectx",
		"./data",
		".",
	}
	for _, dir := range candDirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			cronConfigPath = filepath.Join(dir, "cron_config.json")
			break
		}
	}
	if cronConfigPath == "" {
		cronConfigPath = "./cron_config.json"
	}
}

func logCron(msg string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s", ts, msg)
	cronLogs = append(cronLogs, entry)
	if len(cronLogs) > 50 {
		cronLogs = cronLogs[len(cronLogs)-50:]
	}
}

func loadCronConfig() {
	initCronPaths()
	data, err := os.ReadFile(cronConfigPath)
	if err != nil {
		cronCfg = getDefaultCronConfig()
		saveCronConfigInternal()
		return
	}

	var cfg CronConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		cronCfg = getDefaultCronConfig()
		saveCronConfigInternal()
		return
	}

	cronCfg = cfg
	// Pastikan minimal default jobs ada jika jobs kosong
	if len(cronCfg.Jobs) == 0 {
		cronCfg.Jobs = getDefaultCronConfig().Jobs
	}
}

func saveCronConfigInternal() {
	data, err := json.MarshalIndent(cronCfg, "", "  ")
	if err == nil {
		_ = os.WriteFile(cronConfigPath, data, 0644)
	}
}

func executeJobCommand(job CronJob) (string, error) {
	var cmdStr string
	switch job.Type {
	case "reboot":
		cmdStr = "sync; sleep 1; reboot"
	case "soft_reboot":
		cmdStr = "sync; setprop ctl.restart zygote || killall -9 zygote"
	case "clean_ram":
		cmdStr = "sync; echo 3 > /proc/sys/vm/drop_caches; free -m || true"
	case "restart_bfr":
		cmdStr = "/data/adb/box/scripts/box.service restart"
	case "reconnect_data":
		cmdStr = "svc data disable; sleep 2; svc data enable"
	case "custom":
		cmdStr = strings.TrimSpace(job.Command)
		if cmdStr == "" {
			return "Perintah custom kosong", fmt.Errorf("empty command")
		}
	default:
		return "Tipe job tidak dikenali", fmt.Errorf("unknown job type: %s", job.Type)
	}

	out, err := exec.Command("su", "-c", cmdStr).CombinedOutput()
	outputStr := strings.TrimSpace(string(out))
	if outputStr == "" {
		if err == nil {
			outputStr = "Perintah berhasil dijalankan tanpa output."
		} else {
			outputStr = err.Error()
		}
	}
	return outputStr, err
}

func runCronWorker() {
	cronMu.Lock()
	if cronWorkerRunning {
		cronMu.Unlock()
		return
	}
	cronStopChan = make(chan struct{})
	cronWorkerRunning = true
	cronMu.Unlock()

	logCron("Cron Manager scheduler service aktif.")

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-cronStopChan:
				logCron("Cron Manager scheduler dihentikan.")
				return
			case <-ticker.C:
				cronMu.Lock()
				if !cronCfg.MasterEnabled {
					cronMu.Unlock()
					continue
				}

				now := time.Now()
				nowHM := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
				todayStr := now.Format("2006-01-02")
				nowWeekday := int(now.Weekday())

				for i := range cronCfg.Jobs {
					job := &cronCfg.Jobs[i]
					if !job.Enabled {
						continue
					}

					// Cek hari aktif
					dayMatch := false
					if len(job.Days) == 0 {
						dayMatch = true
					} else {
						for _, d := range job.Days {
							if d == nowWeekday {
								dayMatch = true
								break
							}
						}
					}
					if !dayMatch {
						continue
					}

					// Cek jam dan menit
					if job.Time != nowHM {
						continue
					}

					// Cek apakah sudah pernah berjalan di menit yang sama hari ini
					expectedRunPrefix := todayStr + " " + nowHM
					if strings.HasPrefix(job.LastRun, expectedRunPrefix) {
						continue
					}

					// Eksekusi job
					logCron(fmt.Sprintf("⏰ Menjalankan tugas terjadwal [%s]: %s (%s)...", job.ID, job.Title, job.Type))
					
					// Update last run info sebelum run (untuk mencegah double trigger)
					job.LastRun = now.Format("2006-01-02 15:04:05")

					// Eksekusi
					out, err := executeJobCommand(*job)
					if err != nil {
						job.LastStatus = "failed"
						job.LastOutput = fmt.Sprintf("Error: %v | %s", err, out)
						logCron(fmt.Sprintf("❌ Tugas [%s] gagal: %s", job.Title, out))
					} else {
						job.LastStatus = "success"
						job.LastOutput = out
						logCron(fmt.Sprintf("✅ Tugas [%s] berhasil dijalankan.", job.Title))
					}

					saveCronConfigInternal()
				}
				cronMu.Unlock()
			}
		}
	}()
}

func stopCronWorker() {
	cronMu.Lock()
	defer cronMu.Unlock()
	if !cronWorkerRunning {
		return
	}
	cronWorkerRunning = false
	if cronStopChan != nil {
		close(cronStopChan)
		cronStopChan = nil
	}
}

func init() {
	loadCronConfig()
	if cronCfg.MasterEnabled {
		runCronWorker()
	}
}

// ==================== HTTP HANDLERS ====================

func handleCronStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cronMu.RLock()
	defer cronMu.RUnlock()

	now := time.Now()
	systemTime := now.Format("2006-01-02 15:04:05 MST")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"master_enabled": cronCfg.MasterEnabled,
			"worker_running": cronWorkerRunning,
			"system_time":    systemTime,
			"jobs":           cronCfg.Jobs,
			"recent_logs":    cronLogs,
		},
	})
}

func handleCronSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Unauthorized"})
		return
	}

	var req CronConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON format"})
		return
	}

	cronMu.Lock()
	wasMasterEnabled := cronCfg.MasterEnabled
	cronCfg.MasterEnabled = req.MasterEnabled

	// Update list jobs
	if len(req.Jobs) > 0 {
		cronCfg.Jobs = req.Jobs
	}
	saveCronConfigInternal()

	if cronCfg.MasterEnabled && !wasMasterEnabled {
		cronMu.Unlock()
		runCronWorker()
		cronMu.Lock()
	} else if !cronCfg.MasterEnabled && wasMasterEnabled {
		cronMu.Unlock()
		stopCronWorker()
		cronMu.Lock()
	}
	cronMu.Unlock()

	logCron(fmt.Sprintf("Konfigurasi Cron diperbarui (Master: %v, Total Job: %d).", req.MasterEnabled, len(req.Jobs)))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Konfigurasi jadwal berhasil disimpan",
		"data":    cronCfg,
	})
}

func handleCronRunNow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Unauthorized"})
		return
	}

	var req struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	cronMu.Lock()
	var targetJob *CronJob
	for i := range cronCfg.Jobs {
		if cronCfg.Jobs[i].ID == req.ID {
			targetJob = &cronCfg.Jobs[i]
			break
		}
	}

	var jobToRun CronJob
	if targetJob != nil {
		jobToRun = *targetJob
	} else {
		jobToRun = CronJob{
			ID:      req.ID,
			Title:   "Manual Run",
			Type:    req.Type,
			Command: req.Command,
		}
	}
	cronMu.Unlock()

	logCron(fmt.Sprintf("Manual trigger: menjalankan tugas [%s] (%s)...", jobToRun.Title, jobToRun.Type))
	out, err := executeJobCommand(jobToRun)

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	statusStr := "success"
	if err != nil {
		statusStr = "failed"
		logCron(fmt.Sprintf("Manual run [%s] gagal: %s", jobToRun.Title, out))
	} else {
		logCron(fmt.Sprintf("Manual run [%s] berhasil.", jobToRun.Title))
	}

	cronMu.Lock()
	if targetJob != nil {
		targetJob.LastRun = nowStr
		targetJob.LastStatus = statusStr
		targetJob.LastOutput = out
		saveCronConfigInternal()
	}
	cronMu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"exec_status": statusStr,
		"output":      out,
		"timestamp":   nowStr,
	})
}

func handleCronToggleMaster(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Unauthorized"})
		return
	}

	cronMu.Lock()
	cronCfg.MasterEnabled = !cronCfg.MasterEnabled
	newVal := cronCfg.MasterEnabled
	saveCronConfigInternal()

	if newVal {
		cronMu.Unlock()
		runCronWorker()
	} else {
		cronMu.Unlock()
		stopCronWorker()
	}

	logCron(fmt.Sprintf("Master Cron switch diubah menjadi: %v", newVal))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"master_enabled": newVal,
		"message":        fmt.Sprintf("Status cron: %v", newVal),
	})
}

