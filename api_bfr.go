package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const iniFilePath = "/data/adb/box/settings.ini"
const dummyIniFilePath = "scratch/settings.ini" // For local testing if needed

func getIniPath() string {
	if _, err := os.Stat(iniFilePath); err == nil {
		return iniFilePath
	}
	// Fallback for local dev
	os.MkdirAll("scratch", 0755)
	if _, err := os.Stat(dummyIniFilePath); os.IsNotExist(err) {
		os.WriteFile(dummyIniFilePath, []byte("bin_name=clash\nnetwork_mode=tproxy\nport_detect=true\n"), 0644)
	}
	return dummyIniFilePath
}

func handleBFRSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	path := getIniPath()

	if r.Method == http.MethodGet {
		settings := make(map[string]string)
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					// Remove quotes if present
					if len(val) >= 2 && ((strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) || (strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`))) {
						val = val[1 : len(val)-1]
					}
					settings[key] = val
				}
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"settings": settings,
		})
		return
	}

	if r.Method == http.MethodPost {
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			w.Write([]byte(`{"status": "error", "message": "Invalid JSON"}`))
			return
		}

		// Read existing lines
		var lines []string
		file, err := os.Open(path)
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			file.Close()
		}

		// Update lines
		updatedKeys := make(map[string]bool)
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				originalVal := strings.TrimSpace(parts[1])
				
				if newVal, exists := updates[key]; exists {
					updatedKeys[key] = true
					// Preserve quotes
					if len(originalVal) >= 2 && strings.HasPrefix(originalVal, `"`) && strings.HasSuffix(originalVal, `"`) {
						newVal = `"` + newVal + `"`
					} else if len(originalVal) >= 2 && strings.HasPrefix(originalVal, `'`) && strings.HasSuffix(originalVal, `'`) {
						newVal = `'` + newVal + `'`
					}
					
					re := regexp.MustCompile(`^(\s*` + regexp.QuoteMeta(key) + `\s*=\s*).*$`)
					lines[i] = re.ReplaceAllString(line, "${1}"+newVal)
				}
			}
		}

		// Append any new keys that weren't present in settings.ini
		for key, newVal := range updates {
			if !updatedKeys[key] {
				lines = append(lines, fmt.Sprintf("%s=\"%s\"", key, newVal))
			}
		}

		// Write back
		err = os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
		if err != nil {
			w.Write([]byte(`{"status": "error", "message": "Gagal menyimpan file settings.ini"}`))
			return
		}

		w.Write([]byte(`{"status": "success", "message": "Settings saved"}`))
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func handleBFRControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Write([]byte(`{"status": "error", "message": "Invalid JSON"}`))
		return
	}

	var cmd string
	switch req.Action {
	case "start":
		cmd = "su -c '/data/adb/box/scripts/box.service start && /data/adb/box/scripts/box.iptables renew'"
	case "stop":
		cmd = "su -c '/data/adb/box/scripts/box.iptables disable; /data/adb/box/scripts/box.service stop'"
	case "restart":
		cmd = "su -c '/data/adb/box/scripts/box.service restart'"
	default:
		w.Write([]byte(`{"status": "error", "message": "Aksi tidak dikenali"}`))
		return
	}

	// Exec the command directly
	err := exec.Command("sh", "-c", cmd).Run()
	if err != nil {
		w.Write([]byte(`{"status": "error", "message": "Gagal menjalankan perintah: ` + err.Error() + `"}`))
		return
	}

	w.Write([]byte(`{"status": "success", "message": "Action executed"}`))
}

func handleBFRLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	logPath := "/data/adb/box/run/runs.log"
	b, err := os.ReadFile(logPath)
	if err != nil {
		w.Write([]byte(`{"status": "error", "message": "Berkas log belum tersedia."}`))
		return
	}

	lines := strings.Split(string(b), "\n")
	var cleaned []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(strings.ReplaceAll(l, `"`, ""))
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"logs":   cleaned,
	})
}
