package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type MagiskModule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	VersionCode string `json:"versionCode"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	PendingDel  bool   `json:"pending_del"`
}

func handleMagiskModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "error", "message": "Unauthorized"}`))
		return
	}

	modulesDir := "/data/adb/modules"
	dirs, err := ioutil.ReadDir(modulesDir)
	if err != nil {
		w.Write([]byte(`{"status": "error", "message": "Gagal membaca direktori modul"}`))
		return
	}

	var modules []MagiskModule

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		
		modID := d.Name()
		modPath := filepath.Join(modulesDir, modID)
		propPath := filepath.Join(modPath, "module.prop")
		
		// Baca module.prop
		propBytes, err := ioutil.ReadFile(propPath)
		if err != nil {
			continue // bukan modul valid jika tidak ada module.prop
		}

		mod := MagiskModule{
			ID:      modID,
			Enabled: true,
		}

		// Parsing module.prop
		lines := strings.Split(string(propBytes), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "id=") {
				// ID dari prop diabaikan, pakai nama folder karena itu yg jadi patokan path
			} else if strings.HasPrefix(line, "name=") {
				mod.Name = strings.TrimPrefix(line, "name=")
			} else if strings.HasPrefix(line, "version=") {
				mod.Version = strings.TrimPrefix(line, "version=")
			} else if strings.HasPrefix(line, "versionCode=") {
				mod.VersionCode = strings.TrimPrefix(line, "versionCode=")
			} else if strings.HasPrefix(line, "author=") {
				mod.Author = strings.TrimPrefix(line, "author=")
			} else if strings.HasPrefix(line, "description=") {
				mod.Description = strings.TrimPrefix(line, "description=")
			}
		}

		// Cek state disable
		if _, err := os.Stat(filepath.Join(modPath, "disable")); err == nil {
			mod.Enabled = false
		}
		
		// Cek state remove
		if _, err := os.Stat(filepath.Join(modPath, "remove")); err == nil {
			mod.PendingDel = true
		}

		modules = append(modules, mod)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   modules,
	})
}

func handleMagiskModuleAction(w http.ResponseWriter, r *http.Request) {
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
		Action string `json:"action"` // "toggle", "remove", "restore"
		ID     string `json:"id"`
		Enable bool   `json:"enable"` // untuk toggle
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Write([]byte(`{"status": "error", "message": "Invalid JSON"}`))
		return
	}

	// Sanitasi ID untuk mencegah path traversal
	if strings.Contains(req.ID, "/") || strings.Contains(req.ID, "\\") || req.ID == ".." {
		w.Write([]byte(`{"status": "error", "message": "Invalid Module ID"}`))
		return
	}

	modPath := filepath.Join("/data/adb/modules", req.ID)
	
	// Pastikan folder ada
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		w.Write([]byte(`{"status": "error", "message": "Modul tidak ditemukan"}`))
		return
	}

	disableFile := filepath.Join(modPath, "disable")
	removeFile := filepath.Join(modPath, "remove")

	switch req.Action {
	case "toggle":
		if req.Enable {
			os.Remove(disableFile)
		} else {
			ioutil.WriteFile(disableFile, []byte(""), 0644)
		}
	case "remove":
		ioutil.WriteFile(removeFile, []byte(""), 0644)
	case "restore":
		os.Remove(removeFile)
	default:
		w.Write([]byte(`{"status": "error", "message": "Action tidak dikenal"}`))
		return
	}

	w.Write([]byte(`{"status": "success", "message": "Action completed"}`))
}
