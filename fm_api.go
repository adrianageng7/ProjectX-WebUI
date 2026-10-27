package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type FMItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"` // "dir", "file", "up"
	Size  string `json:"size"`
	MTime string `json:"mtime"`
	Perms string `json:"perms"`
}

// ==================== LIST DIRECTORY ====================

func handleFMList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "/sdcard"
	}
	dir = filepath.Clean(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Tidak bisa membuka direktori: " + err.Error(),
		})
		return
	}

	var folders []FMItem
	var files []FMItem

	// Navigasi ke atas
	if dir != "/" {
		folders = append(folders, FMItem{
			Name:  "..",
			Path:  filepath.Dir(dir),
			Type:  "up",
			Size:  "-",
			MTime: "-",
			Perms: "-",
		})
	}

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(dir, e.Name())
		mtime := info.ModTime().Format("2006-01-02 15:04")
		perms := fmt.Sprintf("%04o", info.Mode().Perm())

		item := FMItem{
			Name:  e.Name(),
			Path:  path,
			Type:  "file",
			Size:  formatSize(info.Size()),
			MTime: mtime,
			Perms: perms,
		}

		if e.IsDir() {
			item.Type = "dir"
			item.Size = "-"
			folders = append(folders, item)
		} else {
			files = append(files, item)
		}
	}

	// Sort: folder dulu, lalu file, keduanya alphabetical
	sort.Slice(folders[1:], func(i, j int) bool { // skip ".." di index 0
		if len(folders) > 1 {
			return folders[i+1].Name < folders[j+1].Name
		}
		return false
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	// Disk info via df command (works di Android)
	diskFree := ""
	diskTotal := ""
	diskPct := 0
	dfOut := sh("df", "-k", dir)
	if dfOut != "" {
		lines := strings.Split(dfOut, "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				// df -k output: Filesystem 1K-blocks Used Available Use% Mounted
				totalKB, e1 := strconv.ParseInt(fields[1], 10, 64)
				availKB, e2 := strconv.ParseInt(fields[3], 10, 64)
				if e1 == nil && e2 == nil && totalKB > 0 {
					diskTotal = formatSize(totalKB * 1024)
					diskFree = formatSize(availKB * 1024)
					diskPct = int(float64(totalKB-availKB) / float64(totalKB) * 100)
				}
			}
		}
	}

	resp := map[string]interface{}{
		"current_dir": dir,
		"items":       append(folders, files...),
		"disk_free":   diskFree,
		"disk_total":  diskTotal,
		"disk_pct":    diskPct,
	}
	json.NewEncoder(w).Encode(resp)
}

// ==================== FILE ACTIONS ====================

func handleFMAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req struct {
		Action  string `json:"action"`
		Target  string `json:"target"`
		Dest    string `json:"dest"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendFMResult(w, fmt.Errorf("request tidak valid"), "")
		return
	}

	// Sanitasi path: pastikan tidak ada path traversal berbahaya
	req.Target = filepath.Clean(req.Target)

	switch req.Action {
	case "mkdir":
		if req.Dest == "" {
			sendFMResult(w, fmt.Errorf("nama folder tidak boleh kosong"), "")
			return
		}
		err := os.Mkdir(filepath.Join(req.Target, req.Dest), 0755)
		sendFMResult(w, err, "Folder berhasil dibuat")

	case "mkfile":
		if req.Dest == "" {
			sendFMResult(w, fmt.Errorf("nama file tidak boleh kosong"), "")
			return
		}
		destPath := filepath.Join(req.Target, req.Dest)
		if _, err := os.Stat(destPath); err == nil {
			sendFMResult(w, fmt.Errorf("file sudah ada"), "")
			return
		}
		err := os.WriteFile(destPath, []byte(""), 0644)
		sendFMResult(w, err, "File berhasil dibuat")

	case "delete":
		err := os.RemoveAll(req.Target)
		sendFMResult(w, err, "Berhasil dihapus")

	case "rename":
		if req.Dest == "" {
			sendFMResult(w, fmt.Errorf("nama baru tidak boleh kosong"), "")
			return
		}
		newPath := filepath.Join(filepath.Dir(req.Target), req.Dest)
		err := os.Rename(req.Target, newPath)
		sendFMResult(w, err, "Berhasil diubah nama")

	case "move":
		if req.Dest == "" {
			sendFMResult(w, fmt.Errorf("tujuan tidak boleh kosong"), "")
			return
		}
		newPath := filepath.Join(req.Dest, filepath.Base(req.Target))
		err := os.Rename(req.Target, newPath)
		sendFMResult(w, err, "Berhasil dipindahkan")

	case "copy":
		if req.Dest == "" {
			sendFMResult(w, fmt.Errorf("tujuan tidak boleh kosong"), "")
			return
		}
		destPath := filepath.Join(req.Dest, filepath.Base(req.Target))
		cmd := exec.Command("sh", "-c", "cp -rf '"+req.Target+"' '"+destPath+"'")
		out, err := cmd.CombinedOutput()
		if err != nil {
			sendFMResult(w, fmt.Errorf("gagal menyalin: "+strings.TrimSpace(string(out))), "")
		} else {
			sendFMResult(w, nil, "Berhasil disalin")
		}

	case "chmod":
		if req.Dest == "" {
			sendFMResult(w, fmt.Errorf("nilai chmod tidak boleh kosong"), "")
			return
		}
		// Hilangkan prefix "0" kalau ada, parse oktal
		modeStr := req.Dest
		mode, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			sendFMResult(w, fmt.Errorf("format chmod tidak valid (gunakan oktal, contoh: 0755)"), "")
			return
		}
		err = os.Chmod(req.Target, os.FileMode(mode))
		sendFMResult(w, err, "Izin akses berhasil diubah")

	case "save":
		if req.Target == "" {
			sendFMResult(w, fmt.Errorf("path file tidak valid"), "")
			return
		}
		err := os.WriteFile(req.Target, []byte(req.Content), 0644)
		sendFMResult(w, err, "File berhasil disimpan")

	case "extract":
		// Extract archive ke dir yang sama, atau dest jika diisi
		if req.Target == "" {
			sendFMResult(w, fmt.Errorf("path archive tidak valid"), "")
			return
		}
		destDir := req.Dest
		if destDir == "" {
			destDir = filepath.Dir(req.Target)
		}
		if err := os.MkdirAll(destDir, 0755); err != nil {
			sendFMResult(w, fmt.Errorf("gagal membuat folder tujuan: "+err.Error()), "")
			return
		}
		lower := strings.ToLower(req.Target)
		ext := strings.ToLower(filepath.Ext(req.Target))

		if ext == ".zip" {
			err := extractZip(req.Target, destDir)
			sendFMResult(w, err, "Archive berhasil diekstrak ke: "+destDir)
			return
		}

		var cmd *exec.Cmd
		switch {
		case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; tar -xzf '"+req.Target+"' -C '"+destDir+"' 2>&1")
		case strings.HasSuffix(lower, ".tar.bz2") || strings.HasSuffix(lower, ".tbz2"):
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; tar -xjf '"+req.Target+"' -C '"+destDir+"' 2>&1")
		case strings.HasSuffix(lower, ".tar.xz") || strings.HasSuffix(lower, ".txz"):
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; tar -xJf '"+req.Target+"' -C '"+destDir+"' 2>&1")
		case ext == ".tar":
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; tar -xf '"+req.Target+"' -C '"+destDir+"' 2>&1")
		case ext == ".gz":
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; cp '"+req.Target+"' '"+destDir+"/"+filepath.Base(req.Target)+"' && gunzip -f '"+destDir+"/"+filepath.Base(req.Target)+"' 2>&1")
		case ext == ".bz2":
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; cp '"+req.Target+"' '"+destDir+"/"+filepath.Base(req.Target)+"' && bunzip2 -f '"+destDir+"/"+filepath.Base(req.Target)+"' 2>&1")
		case ext == ".xz":
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; cp '"+req.Target+"' '"+destDir+"/"+filepath.Base(req.Target)+"' && unxz -f '"+destDir+"/"+filepath.Base(req.Target)+"' 2>&1")
		case ext == ".7z":
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; 7z x '"+req.Target+"' -o'"+destDir+"' -y 2>&1 || 7za x '"+req.Target+"' -o'"+destDir+"' -y 2>&1")
		default:
			sendFMResult(w, fmt.Errorf("format archive tidak didukung: "+ext), "")
			return
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			sendFMResult(w, fmt.Errorf("gagal ekstrak: "+strings.TrimSpace(string(out))), "")
		} else {
			sendFMResult(w, nil, "Archive berhasil diekstrak ke: "+destDir)
		}

	case "compress":
		// Compress file/folder ke archive
		if req.Target == "" {
			sendFMResult(w, fmt.Errorf("target tidak valid"), "")
			return
		}
		format := req.Dest // "zip", "tar.gz", "tar.bz2"
		baseName := filepath.Base(req.Target)
		parentDir := filepath.Dir(req.Target)

		// Nama output archive (req.Content bisa diisi nama custom)
		archiveName := baseName
		if req.Content != "" {
			archiveName = req.Content
		}

		var outPath string
		var cmd *exec.Cmd
		switch format {
		case "zip":
			outPath = filepath.Join(parentDir, archiveName+".zip")
			err := createZip(req.Target, outPath)
			sendFMResult(w, err, "Berhasil dikompres: "+filepath.Base(outPath))
			return
		case "tar.gz":
			outPath = filepath.Join(parentDir, archiveName+".tar.gz")
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; tar -czf '"+outPath+"' -C '"+parentDir+"' '"+baseName+"' 2>&1")
		case "tar.bz2":
			outPath = filepath.Join(parentDir, archiveName+".tar.bz2")
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; tar -cjf '"+outPath+"' -C '"+parentDir+"' '"+baseName+"' 2>&1")
		case "tar.xz":
			outPath = filepath.Join(parentDir, archiveName+".tar.xz")
			cmd = exec.Command("sh", "-c", "export PATH=/data/adb/magisk:$PATH; tar -cJf '"+outPath+"' -C '"+parentDir+"' '"+baseName+"' 2>&1")
		default:
			sendFMResult(w, fmt.Errorf("format tidak didukung: "+format), "")
			return
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			sendFMResult(w, fmt.Errorf("gagal kompres: "+strings.TrimSpace(string(out))), "")
		} else {
			sendFMResult(w, nil, "Berhasil dikompres: "+filepath.Base(outPath))
		}

	default:
		sendFMResult(w, fmt.Errorf("aksi tidak dikenal: "+req.Action), "")
	}
}

// ==================== READ FILE ====================

func handleFMRead(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	target := filepath.Clean(r.URL.Query().Get("file"))
	if target == "" || target == "." {
		http.Error(w, "path file tidak valid", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(target)
	if err != nil {
		http.Error(w, "file tidak ditemukan: "+err.Error(), http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "target adalah direktori, bukan file", http.StatusBadRequest)
		return
	}

	// Jika request ada ?download=1, set Content-Disposition
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(target)+"\"")
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		ctype := mime.TypeByExtension(filepath.Ext(target))
		if ctype != "" {
			w.Header().Set("Content-Type", ctype)
		} else {
			// Serve sebagai teks biasa (untuk editor)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
	}
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))

	f, err := os.Open(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

// ==================== UPLOAD ====================

func handleFMUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Max 100 MB per file
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		sendFMResult(w, err, "")
		return
	}

	dir := r.FormValue("dir")
	if dir == "" {
		dir = "/sdcard"
	}
	dir = filepath.Clean(dir)

	// Bisa upload multiple files
	files := r.MultipartForm.File["file_upload"]
	if len(files) == 0 {
		sendFMResult(w, fmt.Errorf("tidak ada file yang diterima"), "")
		return
	}

	uploaded := 0
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			continue
		}
		destPath := filepath.Join(dir, filepath.Base(fh.Filename))
		dst, err := os.Create(destPath)
		if err != nil {
			file.Close()
			continue
		}
		io.Copy(dst, file)
		file.Close()
		dst.Close()
		uploaded++
	}

	if uploaded == 0 {
		sendFMResult(w, fmt.Errorf("gagal menyimpan file"), "")
	} else {
		sendFMResult(w, nil, fmt.Sprintf("%d file berhasil diunggah", uploaded))
	}
}

// ==================== HELPERS ====================

func sendFMResult(w http.ResponseWriter, err error, msg string) {
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": err.Error(),
		})
	} else {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": msg,
		})
	}
}

func formatSize(bytes int64) string {
	if bytes < 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func createZip(srcPath, destZipPath string) error {
	zipFile, err := os.Create(destZipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(srcPath)
	}

	return filepath.Walk(srcPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		var zipPath string
		if baseDir != "" {
			zipPath = filepath.Join(baseDir, relPath)
		} else {
			zipPath = relPath
		}
		zipPath = filepath.ToSlash(zipPath)

		header, err := zip.FileInfoHeader(f)
		if err != nil {
			return err
		}

		header.Name = zipPath
		if f.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Check for ZipSlip vulnerability
		cleanDest := filepath.Clean(destDir)
		cleanPath := filepath.Clean(fpath)
		if !strings.HasPrefix(cleanPath, cleanDest+string(os.PathSeparator)) && cleanPath != cleanDest {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

