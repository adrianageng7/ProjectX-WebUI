package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var smsMutex sync.Mutex

type SMSItem struct {
	ID            string `json:"id"`
	Address       string `json:"address"`
	Date          int64  `json:"date"`
	FormattedTime string `json:"formatted_time"`
	Body          string `json:"body"`
	Type          int    `json:"type"` // 1 = Masuk (Inbox), 2 = Keluar (Sent)
	OTP           string `json:"otp,omitempty"`
}

// Deteksi otomatis kode OTP / PIN / token verifikasi dalam pesan
func detectOTP(body string) string {
	// 1. Pola: kode/otp/pin/verifikasi diikuti angka 4-8 digit
	re1 := regexp.MustCompile(`(?i)(?:kode|code|otp|pin|verifikasi|verification|token|password)[^0-9]{0,10}([0-9]{4,8})\b`)
	if m := re1.FindStringSubmatch(body); len(m) > 1 {
		return m[1]
	}

	// 2. Pola kata kunci keamanan dengan angka 4-6 digit berdiri sendiri
	lower := strings.ToLower(body)
	if strings.Contains(lower, "otp") || strings.Contains(lower, "kode") || strings.Contains(lower, "verifikasi") ||
		strings.Contains(lower, "rahasia") || strings.Contains(lower, "login") || strings.Contains(lower, "security") {
		re2 := regexp.MustCompile(`\b([0-9]{4,6})\b`)
		if m := re2.FindStringSubmatch(body); len(m) > 1 {
			return m[1]
		}
	}

	return ""
}

// Format waktu milidetik Unix ke tanggal lokal WIB
func formatSMSTime(epochMillis int64) string {
	if epochMillis <= 0 {
		return "-"
	}
	// Jika format dalam detik (bukan milidetik)
	if epochMillis < 10000000000 {
		epochMillis *= 1000
	}
	sec := epochMillis / 1000
	nsec := (epochMillis % 1000) * 1000000
	t := time.Unix(sec, nsec)
	return t.Format("2006-01-02 15:04")
}

// Cari path database SMS telephony Android yang valid
func getTelephonyDBPath() string {
	candidates := []string{
		"/data/user_de/0/com.android.providers.telephony/databases/mmssms.db",
		"/data/data/com.android.providers.telephony/databases/mmssms.db",
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// Baca SMS dari database Telephony bawaan Android (mmssms.db)
func readSMSFromTelephonyDB() ([]SMSItem, error) {
	dbPath := getTelephonyDBPath()
	if dbPath == "" {
		return nil, fmt.Errorf("telephony db tidak ditemukan")
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT _id, address, date, body, type FROM sms ORDER BY date DESC LIMIT 150")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SMSItem
	for rows.Next() {
		var id int64
		var addr sql.NullString
		var date int64
		var body sql.NullString
		var msgType int

		if err := rows.Scan(&id, &addr, &date, &body, &msgType); err == nil {
			b := ""
			if body.Valid {
				b = body.String
			}
			a := "Unknown"
			if addr.Valid && addr.String != "" {
				a = addr.String
			}
			result = append(result, SMSItem{
				ID:            strconv.FormatInt(id, 10),
				Address:       a,
				Date:          date,
				FormattedTime: formatSMSTime(date),
				Body:          b,
				Type:          msgType,
				OTP:           detectOTP(b),
			})
		}
	}
	return result, nil
}

// Baca SMS via Android Content Provider CLI (Universal untuk semua Android)
func readSMSFromContentProvider() ([]SMSItem, error) {
	cmd := exec.Command("content", "query", "--uri", "content://sms", "--projection", "_id,address,date,body,type", "--sort", "date DESC")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result []SMSItem
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Row:") {
			continue
		}

		// Parse key-value: Row: 0 _id=3, address=SMARTFREN, date=1790571166845, body=..., type=1
		// Hilangkan prefix "Row: N "
		idx := strings.Index(line, " _id=")
		if idx == -1 {
			continue
		}
		dataPart := line[idx+1:]

		var item SMSItem
		// Ekstrak _id
		if m := regexp.MustCompile(`_id=([0-9]+)`).FindStringSubmatch(dataPart); len(m) > 1 {
			item.ID = m[1]
		}
		// Ekstrak address
		if m := regexp.MustCompile(`address=([^,]+)`).FindStringSubmatch(dataPart); len(m) > 1 {
			item.Address = strings.TrimSpace(m[1])
			if item.Address == "NULL" {
				item.Address = "Unknown"
			}
		}
		// Ekstrak date
		if m := regexp.MustCompile(`date=([0-9]+)`).FindStringSubmatch(dataPart); len(m) > 1 {
			item.Date, _ = strconv.ParseInt(m[1], 10, 64)
			item.FormattedTime = formatSMSTime(item.Date)
		}
		// Ekstrak type
		if m := regexp.MustCompile(`type=([0-9]+)`).FindStringSubmatch(dataPart); len(m) > 1 {
			item.Type, _ = strconv.Atoi(m[1])
		}
		// Ekstrak body
		if bIdx := strings.Index(dataPart, "body="); bIdx != -1 {
			bodyText := dataPart[bIdx+5:]
			// Cari penutup field berikutnya (misal ", type=" atau end of string)
			endPatterns := []string{", type=", ", read=", ", status=", ", service_center=", ", locked=", ", seen=", ", thread_id=", ", sub_id="}
			minEnd := len(bodyText)
			for _, pat := range endPatterns {
				if p := strings.Index(bodyText, pat); p != -1 && p < minEnd {
					minEnd = p
				}
			}
			item.Body = strings.TrimSpace(bodyText[:minEnd])
			if item.Body == "NULL" {
				item.Body = ""
			}
			// Pastikan suffix atribut seperti ', type=1' tidak bocor ke dalam body
			reTrailing := regexp.MustCompile(`, ?(?:type|read|status|service_center|locked|seen|thread_id|sub_id)=.*$`)
			item.Body = strings.TrimSpace(reTrailing.ReplaceAllString(item.Body, ""))

			item.OTP = detectOTP(item.Body)
		}

		if item.ID != "" {
			result = append(result, item)
		}
	}

	return result, nil
}

// Simulator SMS (Fallback hanya jika berjalan di PC Windows tanpa HP)
func getSimulatorSMS() []SMSItem {
	simFile := "./data/sms_simulator.json"
	os.MkdirAll("./data", 0755)

	content, err := os.ReadFile(simFile)
	if err != nil {
		initial := []SMSItem{
			{
				ID:            "1",
				Address:       "SMARTFREN",
				Date:          time.Now().UnixMilli(),
				FormattedTime: formatSMSTime(time.Now().UnixMilli()),
				Body:          "Batas kuota harian Anda telah mencapai 80%. Nikmati Speed Booster di *888#.",
				Type:          1,
				OTP:           "",
			},
			{
				ID:            "2",
				Address:       "BANK-BCA",
				Date:          time.Now().Add(-15 * time.Minute).UnixMilli(),
				FormattedTime: formatSMSTime(time.Now().Add(-15 * time.Minute).UnixMilli()),
				Body:          "JANGAN BERIKAN KE SIAPAPUN! Kode OTP verifikasi login m-BCA Anda adalah 839201. Berlaku 5 menit.",
				Type:          1,
				OTP:           "839201",
			},
			{
				ID:            "3",
				Address:       "TELKOMSEL",
				Date:          time.Now().Add(-2 * time.Hour).UnixMilli(),
				FormattedTime: formatSMSTime(time.Now().Add(-2 * time.Hour).UnixMilli()),
				Body:          "Paket Internet OMG! 20GB Anda aktif s.d 30 hari. Cek sisa kuota di MyTelkomsel.",
				Type:          1,
				OTP:           "",
			},
		}
		data, _ := json.MarshalIndent(initial, "", "  ")
		os.WriteFile(simFile, data, 0644)
		return initial
	}

	var list []SMSItem
	json.Unmarshal(content, &list)
	return list
}

// Handler HTTP untuk /api/sms/list
func handleSMSList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	smsMutex.Lock()
	defer smsMutex.Unlock()

	// 1. Coba baca via SQLite mmssms.db (Paling cepat & akurat)
	list, err := readSMSFromTelephonyDB()
	source := "telephony_sqlite"

	// 2. Coba via Android Content Provider CLI
	if err != nil || len(list) == 0 {
		cpList, cpErr := readSMSFromContentProvider()
		if cpErr == nil && len(cpList) > 0 {
			list = cpList
			source = "content_provider"
		}
	}

	// 3. Fallback ke simulator HANYA jika di PC Windows saat development lokal tanpa HP
	if len(list) == 0 && runtime.GOOS == "windows" {
		list = getSimulatorSMS()
		source = "simulator"
	}

	if list == nil {
		list = []SMSItem{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"source":   source,
		"total":    len(list),
		"messages": list,
	})
}

// Handler HTTP untuk /api/sms/delete
func handleSMSDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req struct {
		ID  string `json:"id"`
		All bool   `json:"all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Request tidak valid"})
		return
	}

	smsMutex.Lock()
	defer smsMutex.Unlock()

	dbPath := getTelephonyDBPath()
	if dbPath != "" {
		if db, err := sql.Open("sqlite", dbPath); err == nil {
			defer db.Close()
			if req.All {
				db.Exec("DELETE FROM sms")
			} else if req.ID != "" {
				db.Exec("DELETE FROM sms WHERE _id = ?", req.ID)
			}
		}
	}

	// Juga jalankan via content provider delete
	if req.All {
		exec.Command("content", "delete", "--uri", "content://sms").Run()
	} else if req.ID != "" {
		exec.Command("content", "delete", "--uri", "content://sms", "--where", fmt.Sprintf("_id=%s", req.ID)).Run()
	}

	// Bersihkan juga di file simulator jika sedang mode simulator
	simFile := "./data/sms_simulator.json"
	if content, err := os.ReadFile(simFile); err == nil {
		var list []SMSItem
		if json.Unmarshal(content, &list) == nil {
			var updated []SMSItem
			if !req.All {
				for _, item := range list {
					if item.ID != req.ID {
						updated = append(updated, item)
					}
				}
			}
			data, _ := json.MarshalIndent(updated, "", "  ")
			os.WriteFile(simFile, data, 0644)
		}
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Pesan berhasil dihapus",
	})
}
