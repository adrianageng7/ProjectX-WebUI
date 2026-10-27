package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	trafficDB   *sql.DB
	trafficLock sync.RWMutex

	// In-memory cache untuk live speed (agar respons ajax_speed instan < 1ms)
	liveTraffic struct {
		sync.RWMutex
		lastTime   int64
		interfaces map[string]IfaceStat
	}
)

type IfaceStat struct {
	Rx float64 `json:"rx"`
	Tx float64 `json:"tx"`
}

// Inisialisasi Database SQLite Bandwidth
func initTrafficDB() {
	dbDir := "./data"
	os.MkdirAll(dbDir, 0755)
	dbPath := filepath.Join(dbDir, "bandwidth.db")

	var err error
	trafficDB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("⚠️ Gagal membuka SQLite traffic DB: %v\n", err)
		return
	}

	// Performance pragmas
	trafficDB.Exec("PRAGMA journal_mode = WAL;")
	trafficDB.Exec("PRAGMA synchronous = NORMAL;")

	// Schema tabel identik dengan standar vnStat database
	tables := []string{
		"CREATE TABLE IF NOT EXISTS meta (iface TEXT PRIMARY KEY, last_rx REAL, last_tx REAL, last_ts INTEGER);",
		"CREATE TABLE IF NOT EXISTS samples (id INTEGER PRIMARY KEY AUTOINCREMENT, ts INTEGER, iface TEXT, rx REAL, tx REAL);",
		"CREATE TABLE IF NOT EXISTS hourly (hour_start INTEGER, iface TEXT, rx_sum REAL, tx_sum REAL, PRIMARY KEY(hour_start, iface));",
		"CREATE TABLE IF NOT EXISTS daily (date TEXT, iface TEXT, rx_sum REAL, tx_sum REAL, PRIMARY KEY(date, iface));",
		"CREATE TABLE IF NOT EXISTS monthly (month TEXT, iface TEXT, rx_sum REAL, tx_sum REAL, PRIMARY KEY(month, iface));",
	}

	for _, q := range tables {
		if _, err := trafficDB.Exec(q); err != nil {
			fmt.Printf("⚠️ Error init table: %v\n", err)
		}
	}

	liveTraffic.interfaces = make(map[string]IfaceStat)
	fmt.Println("📊 SQLite Traffic Database siap.")
}

// Ambil data rx & tx dari semua interface aktif di /sys/class/net atau /proc/net/dev
func scanNetworkStats() map[string]IfaceStat {
	result := make(map[string]IfaceStat)

	// Cara 1: Scan via /sys/class/net/*
	matches, _ := filepath.Glob("/sys/class/net/*")
	for _, path := range matches {
		iface := filepath.Base(path)
		if iface == "lo" {
			continue
		}
		rxBytesStr := readFirstLine(filepath.Join(path, "statistics/rx_bytes"))
		txBytesStr := readFirstLine(filepath.Join(path, "statistics/tx_bytes"))
		if rxBytesStr != "" || txBytesStr != "" {
			rx, _ := strconv.ParseFloat(rxBytesStr, 64)
			tx, _ := strconv.ParseFloat(txBytesStr, 64)
			if rx > 0 || tx > 0 {
				result[iface] = IfaceStat{Rx: rx, Tx: tx}
			}
		}
	}

	// Cara 2: Fallback /proc/net/dev jika /sys/class/net kosong
	if len(result) == 0 {
		content := readFile("/proc/net/dev")
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "|") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			iface := strings.TrimSpace(parts[0])
			if iface == "lo" {
				continue
			}
			fields := strings.Fields(parts[1])
			if len(fields) >= 9 {
				rx, _ := strconv.ParseFloat(fields[0], 64)
				tx, _ := strconv.ParseFloat(fields[8], 64)
				result[iface] = IfaceStat{Rx: rx, Tx: tx}
			}
		}
	}

	return result
}

// Background goroutine pengumpul statistik setiap 5 detik
func startTrafficCollector() {
	if trafficDB == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		pruneTicker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		defer pruneTicker.Stop()

		for {
			select {
			case <-ticker.C:
				recordTrafficSample()
			case <-pruneTicker.C:
				pruneOldTrafficData()
			}
		}
	}()
}

func recordTrafficSample() {
	if trafficDB == nil {
		return
	}

	now := time.Now()
	nowUnix := now.Unix()
	hourStart := (nowUnix / 3600) * 3600
	today := now.Format("2006-01-02")
	thisMonth := now.Format("2006-01")

	active := scanNetworkStats()
	if len(active) == 0 {
		return
	}

	// Simpan ke in-memory live traffic untuk ajax_speed
	liveTraffic.Lock()
	liveTraffic.lastTime = nowUnix
	liveTraffic.interfaces = active
	liveTraffic.Unlock()

	trafficLock.Lock()
	defer trafficLock.Unlock()

	tx, err := trafficDB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	for iface, stats := range active {
		curRx := stats.Rx
		curTx := stats.Tx

		// Baca data terakhir dari meta
		var prevRx, prevTx float64
		var prevTs int64
		err := tx.QueryRow("SELECT last_rx, last_tx, last_ts FROM meta WHERE iface = ?", iface).Scan(&prevRx, &prevTx, &prevTs)

		var deltaRx, deltaTx float64
		if err == nil && prevTs > 0 {
			deltaRx = curRx - prevRx
			deltaTx = curTx - prevTx

			// Handle restart counter atau reboot
			if deltaRx < 0 {
				deltaRx = curRx
			}
			if deltaTx < 0 {
				deltaTx = curTx
			}
			// Batasi anomali jika delta > 1 TB
			if deltaRx > 1099511627776 {
				deltaRx = 0
			}
			if deltaTx > 1099511627776 {
				deltaTx = 0
			}
		} else {
			// Data pertama kali
			deltaRx = 0
			deltaTx = 0
		}

		// Insert samples jika ada traffic
		if deltaRx > 0 || deltaTx > 0 {
			tx.Exec("INSERT INTO samples(ts, iface, rx, tx) VALUES(?,?,?,?)", nowUnix, iface, deltaRx, deltaTx)

			// Update hourly
			res, _ := tx.Exec("UPDATE hourly SET rx_sum = rx_sum + ?, tx_sum = tx_sum + ? WHERE hour_start = ? AND iface = ?",
				deltaRx, deltaTx, hourStart, iface)
			if rows, _ := res.RowsAffected(); rows == 0 {
				tx.Exec("INSERT INTO hourly(hour_start, iface, rx_sum, tx_sum) VALUES(?,?,?,?)",
					hourStart, iface, deltaRx, deltaTx)
			}

			// Update daily
			res, _ = tx.Exec("UPDATE daily SET rx_sum = rx_sum + ?, tx_sum = tx_sum + ? WHERE date = ? AND iface = ?",
				deltaRx, deltaTx, today, iface)
			if rows, _ := res.RowsAffected(); rows == 0 {
				tx.Exec("INSERT INTO daily(date, iface, rx_sum, tx_sum) VALUES(?,?,?,?)",
					today, iface, deltaRx, deltaTx)
			}

			// Update monthly
			res, _ = tx.Exec("UPDATE monthly SET rx_sum = rx_sum + ?, tx_sum = tx_sum + ? WHERE month = ? AND iface = ?",
				deltaRx, deltaTx, thisMonth, iface)
			if rows, _ := res.RowsAffected(); rows == 0 {
				tx.Exec("INSERT INTO monthly(month, iface, rx_sum, tx_sum) VALUES(?,?,?,?)",
					thisMonth, iface, deltaRx, deltaTx)
			}
		}

		// Update / insert meta
		res, _ := tx.Exec("UPDATE meta SET last_rx = ?, last_tx = ?, last_ts = ? WHERE iface = ?",
			curRx, curTx, nowUnix, iface)
		if rows, _ := res.RowsAffected(); rows == 0 {
			tx.Exec("INSERT INTO meta(iface, last_rx, last_tx, last_ts) VALUES(?,?,?,?)",
				iface, curRx, curTx, nowUnix)
		}
	}

	tx.Commit()
}

func pruneOldTrafficData() {
	if trafficDB == nil {
		return
	}
	trafficLock.Lock()
	defer trafficLock.Unlock()

	now := time.Now().Unix()
	trafficDB.Exec("DELETE FROM samples WHERE ts < ?", now-172800)                     // 2 hari
	trafficDB.Exec("DELETE FROM hourly WHERE hour_start < ?", now-(7*24*3600))         // 7 hari
	trafficDB.Exec("DELETE FROM daily WHERE date < date('now', '-30 day')")            // 30 hari
	trafficDB.Exec("DELETE FROM monthly WHERE month < strftime('%Y-%m', 'now', '-12 month')") // 12 bulan
}

// ==================== HTTP HANDLER ====================

func handleVnstat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 1. AJAX Speed (Realtime polling per detik)
	if r.URL.Query().Get("ajax_speed") == "1" {
		liveTraffic.RLock()
		snapshot := make(map[string]IfaceStat)
		for k, v := range liveTraffic.interfaces {
			snapshot[k] = v
		}
		t := liveTraffic.lastTime
		liveTraffic.RUnlock()

		if t == 0 {
			t = time.Now().Unix()
			snapshot = scanNetworkStats()
		}

		// Hitung Total Klien (Hotspot + USB rndis)
		var clientRx, clientTx float64
		for ifc, stats := range snapshot {
			if strings.Contains(ifc, "wlan") || strings.Contains(ifc, "rndis") {
				clientRx += stats.Rx
				clientTx += stats.Tx
			}
		}
		snapshot["Total Klien (Hotspot+USB)"] = IfaceStat{Rx: clientTx, Tx: clientRx}

		// Swap RX/TX untuk klien agar Download/Upload dilihat dari sisi user klien
		for ifc, stats := range snapshot {
			if ifc != "Total Klien (Hotspot+USB)" && (strings.Contains(ifc, "wlan") || strings.Contains(ifc, "rndis")) {
				snapshot[ifc] = IfaceStat{Rx: stats.Tx, Tx: stats.Rx}
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"time":       t,
			"interfaces": snapshot,
		})
		return
	}

	// 2. Full JSON Dashboard Data
	if r.URL.Query().Get("json") == "1" {
		filterIface := strings.TrimSpace(r.URL.Query().Get("iface"))
		if filterIface == "" {
			filterIface = "All"
		}

		trafficLock.RLock()
		defer trafficLock.RUnlock()

		// Ambil daftar interface yang pernah tercatat
		var loggedIfaces []string
		rows, err := trafficDB.Query("SELECT DISTINCT iface FROM meta")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var ifc string
				if err := rows.Scan(&ifc); err == nil {
					loggedIfaces = append(loggedIfaces, ifc)
				}
			}
		}

		allOptions := append([]string{"All", "Total Klien (Hotspot+USB)"}, loggedIfaces...)

		now := time.Now()
		nowUnix := now.Unix()
		hourStart := (nowUnix / 3600) * 3600
		today := now.Format("2006-01-02")
		yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
		thisMonth := now.Format("2006-01")

		swapRxTx := false
		type kpiVal struct {
			Rx float64 `json:"rx"`
			Tx float64 `json:"tx"`
		}
		kpi := map[string]*kpiVal{
			"today":     {Rx: 0, Tx: 0},
			"yesterday": {Rx: 0, Tx: 0},
			"month":     {Rx: 0, Tx: 0},
		}

		type HourUsage struct {
			HourStart int64
			Rx        float64
			Tx        float64
		}
		var hourlyRows []HourUsage

		type TableRow struct {
			Key   string  `json:"-"`
			Date  string  `json:"date,omitempty"`
			Month string  `json:"month,omitempty"`
			Rx    float64 `json:"rx"`
			Tx    float64 `json:"tx"`
			Total float64 `json:"total"`
		}
		var dailyData []TableRow
		var monthlyData []TableRow

		if filterIface == "Total Klien (Hotspot+USB)" {
			swapRxTx = true

			// Hourly
			hRows, _ := trafficDB.Query("SELECT hour_start, SUM(rx_sum), SUM(tx_sum) FROM hourly WHERE iface LIKE '%wlan%' OR iface LIKE '%rndis%' GROUP BY hour_start ORDER BY hour_start DESC LIMIT 24")
			if hRows != nil {
				defer hRows.Close()
				for hRows.Next() {
					var h HourUsage
					hRows.Scan(&h.HourStart, &h.Rx, &h.Tx)
					hourlyRows = append(hourlyRows, h)
				}
			}

			// Daily
			dRows, _ := trafficDB.Query("SELECT date, SUM(rx_sum), SUM(tx_sum) FROM daily WHERE iface LIKE '%wlan%' OR iface LIKE '%rndis%' GROUP BY date ORDER BY date DESC LIMIT 30")
			if dRows != nil {
				defer dRows.Close()
				for dRows.Next() {
					var r TableRow
					dRows.Scan(&r.Date, &r.Rx, &r.Tx)
					dailyData = append(dailyData, r)
				}
			}

			// Monthly
			mRows, _ := trafficDB.Query("SELECT month, SUM(rx_sum), SUM(tx_sum) FROM monthly WHERE iface LIKE '%wlan%' OR iface LIKE '%rndis%' GROUP BY month ORDER BY month DESC LIMIT 12")
			if mRows != nil {
				defer mRows.Close()
				for mRows.Next() {
					var r TableRow
					mRows.Scan(&r.Month, &r.Rx, &r.Tx)
					monthlyData = append(monthlyData, r)
				}
			}

			// KPI
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM daily WHERE date = ? AND (iface LIKE '%wlan%' OR iface LIKE '%rndis%')", today).Scan(&kpi["today"].Rx, &kpi["today"].Tx)
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM daily WHERE date = ? AND (iface LIKE '%wlan%' OR iface LIKE '%rndis%')", yesterday).Scan(&kpi["yesterday"].Rx, &kpi["yesterday"].Tx)
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM monthly WHERE month = ? AND (iface LIKE '%wlan%' OR iface LIKE '%rndis%')", thisMonth).Scan(&kpi["month"].Rx, &kpi["month"].Tx)

		} else if filterIface == "All" {
			// Hourly
			hRows, _ := trafficDB.Query("SELECT hour_start, SUM(rx_sum), SUM(tx_sum) FROM hourly GROUP BY hour_start ORDER BY hour_start DESC LIMIT 24")
			if hRows != nil {
				defer hRows.Close()
				for hRows.Next() {
					var h HourUsage
					hRows.Scan(&h.HourStart, &h.Rx, &h.Tx)
					hourlyRows = append(hourlyRows, h)
				}
			}

			// Daily
			dRows, _ := trafficDB.Query("SELECT date, SUM(rx_sum), SUM(tx_sum) FROM daily GROUP BY date ORDER BY date DESC LIMIT 30")
			if dRows != nil {
				defer dRows.Close()
				for dRows.Next() {
					var r TableRow
					dRows.Scan(&r.Date, &r.Rx, &r.Tx)
					dailyData = append(dailyData, r)
				}
			}

			// Monthly
			mRows, _ := trafficDB.Query("SELECT month, SUM(rx_sum), SUM(tx_sum) FROM monthly GROUP BY month ORDER BY month DESC LIMIT 12")
			if mRows != nil {
				defer mRows.Close()
				for mRows.Next() {
					var r TableRow
					mRows.Scan(&r.Month, &r.Rx, &r.Tx)
					monthlyData = append(monthlyData, r)
				}
			}

			// KPI
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM daily WHERE date = ?", today).Scan(&kpi["today"].Rx, &kpi["today"].Tx)
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM daily WHERE date = ?", yesterday).Scan(&kpi["yesterday"].Rx, &kpi["yesterday"].Tx)
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM monthly WHERE month = ?", thisMonth).Scan(&kpi["month"].Rx, &kpi["month"].Tx)

		} else {
			// Single Interface
			if strings.Contains(strings.ToLower(filterIface), "wlan") || strings.Contains(strings.ToLower(filterIface), "rndis") {
				swapRxTx = true
			}

			hRows, _ := trafficDB.Query("SELECT hour_start, rx_sum, tx_sum FROM hourly WHERE iface = ? ORDER BY hour_start DESC LIMIT 24", filterIface)
			if hRows != nil {
				defer hRows.Close()
				for hRows.Next() {
					var h HourUsage
					hRows.Scan(&h.HourStart, &h.Rx, &h.Tx)
					hourlyRows = append(hourlyRows, h)
				}
			}

			dRows, _ := trafficDB.Query("SELECT date, rx_sum, tx_sum FROM daily WHERE iface = ? ORDER BY date DESC LIMIT 30", filterIface)
			if dRows != nil {
				defer dRows.Close()
				for dRows.Next() {
					var r TableRow
					dRows.Scan(&r.Date, &r.Rx, &r.Tx)
					dailyData = append(dailyData, r)
				}
			}

			mRows, _ := trafficDB.Query("SELECT month, rx_sum, tx_sum FROM monthly WHERE iface = ? ORDER BY month DESC LIMIT 12", filterIface)
			if mRows != nil {
				defer mRows.Close()
				for mRows.Next() {
					var r TableRow
					mRows.Scan(&r.Month, &r.Rx, &r.Tx)
					monthlyData = append(monthlyData, r)
				}
			}

			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM daily WHERE date = ? AND iface = ?", today, filterIface).Scan(&kpi["today"].Rx, &kpi["today"].Tx)
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM daily WHERE date = ? AND iface = ?", yesterday, filterIface).Scan(&kpi["yesterday"].Rx, &kpi["yesterday"].Tx)
			trafficDB.QueryRow("SELECT COALESCE(SUM(rx_sum),0), COALESCE(SUM(tx_sum),0) FROM monthly WHERE month = ? AND iface = ?", thisMonth, filterIface).Scan(&kpi["month"].Rx, &kpi["month"].Tx)
		}

		if swapRxTx {
			kpi["today"].Rx, kpi["today"].Tx = kpi["today"].Tx, kpi["today"].Rx
			kpi["yesterday"].Rx, kpi["yesterday"].Tx = kpi["yesterday"].Tx, kpi["yesterday"].Rx
			kpi["month"].Rx, kpi["month"].Tx = kpi["month"].Tx, kpi["month"].Rx
		}

		// Timeline 24 Jam terurut
		timelineMap := make(map[int64]HourUsage)
		for i := 23; i >= 0; i-- {
			ts := hourStart - int64(i*3600)
			timelineMap[ts] = HourUsage{HourStart: ts, Rx: 0, Tx: 0}
		}
		for _, r := range hourlyRows {
			if _, exists := timelineMap[r.HourStart]; exists {
				rx := r.Rx
				tx := r.Tx
				if swapRxTx {
					rx, tx = tx, rx
				}
				timelineMap[r.HourStart] = HourUsage{HourStart: r.HourStart, Rx: rx, Tx: tx}
			}
		}

		type HourOutput struct {
			Time string  `json:"time"`
			Rx   float64 `json:"rx"`
			Tx   float64 `json:"tx"`
		}
		var hourList []HourOutput
		for i := 23; i >= 0; i-- {
			ts := hourStart - int64(i*3600)
			slot := timelineMap[ts]
			hourList = append(hourList, HourOutput{
				Time: time.Unix(ts, 0).Format(time.RFC3339),
				Rx:   slot.Rx,
				Tx:   slot.Tx,
			})
		}

		// Format output tabel daily & monthly
		var outDaily []TableRow
		for _, r := range dailyData {
			rx := r.Rx
			tx := r.Tx
			if swapRxTx {
				rx, tx = tx, rx
			}
			outDaily = append(outDaily, TableRow{
				Date:  r.Date,
				Rx:    rx,
				Tx:    tx,
				Total: rx + tx,
			})
		}

		var outMonthly []TableRow
		for _, r := range monthlyData {
			rx := r.Rx
			tx := r.Tx
			if swapRxTx {
				rx, tx = tx, rx
			}
			outMonthly = append(outMonthly, TableRow{
				Month: r.Month,
				Rx:    rx,
				Tx:    tx,
				Total: rx + tx,
			})
		}

		resp := map[string]interface{}{
			"meta": map[string]interface{}{
				"ifaces":       allOptions,
				"generated_at": now.Format(time.RFC3339),
				"tz":           "Asia/Jakarta",
			},
			"kpi": map[string]interface{}{
				"today":     kpi["today"],
				"yesterday": kpi["yesterday"],
				"month":     kpi["month"],
			},
			"usage": map[string]interface{}{
				"hour": hourList,
			},
			"tables": map[string]interface{}{
				"daily":   outDaily,
				"monthly": outMonthly,
			},
		}

		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Write([]byte(`{"status": "ok"}`))
}
