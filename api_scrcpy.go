package main

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var scrcpyCmd *exec.Cmd

func getScrcpyJarPath() string {
	candidates := []string{
		"/data/adb/modules/projectx-webui/bin/scrcpy-server.jar",
		"/data/adb/modules/boxera_projectx/bin/scrcpy-server.jar",
		"/data/local/tmp/scrcpy-server.jar",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(exe), "scrcpy-server.jar")}, candidates...)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/data/local/tmp/scrcpy-server.jar"
}

func startScrcpyServer() error {
	if scrcpyCmd != nil && scrcpyCmd.Process != nil {
		err := scrcpyCmd.Process.Signal(nil)
		if err == nil {
			return nil
		}
		scrcpyCmd = nil
	}

	// Matikan proses lama scrcpy jika ada
	exec.Command("su", "-c", "pkill -9 -f scrcpy-server.jar").Run()
	time.Sleep(300 * time.Millisecond)

	jarPath := getScrcpyJarPath()
	log.Printf("Menggunakan scrcpy-server jar dari: %s", jarPath)

	args := []string{
		"-Djava.class.path=" + jarPath,
		"/system/bin",
		"com.genymobile.scrcpy.Server",
		"4.1",
		"log_level=info",
		"video_bit_rate=2000000",
		"max_fps=30",
		"tunnel_forward=true",
		"send_frame_meta=false",
		"control=true",
		"audio=false",
		"cleanup=true",
	}

	scrcpyCmd = exec.Command("app_process", args...)

	// Redirect output ke file log biar bisa didebug
	logFile, _ := os.OpenFile("/data/local/tmp/scrcpy_error.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	scrcpyCmd.Stdout = logFile
	scrcpyCmd.Stderr = logFile

	err := scrcpyCmd.Start()
	if err != nil {
		return err
	}

	// Tunggu sebentar agar server java siap
	time.Sleep(1200 * time.Millisecond)
	return nil
}

func handleScrcpyWs(w http.ResponseWriter, r *http.Request) {
	err := startScrcpyServer()
	if err != nil {
		log.Println("Gagal start scrcpy:", err)
		http.Error(w, "Gagal start scrcpy", 500)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade ws error:", err)
		return
	}
	defer ws.Close()

	// 1. Konek ke Video Socket (dengan retry sampai scrcpy siap menerima)
	var videoConn net.Conn
	for i := 0; i < 15; i++ {
		videoConn, err = net.Dial("unix", "@scrcpy")
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		log.Println("Gagal konek video socket:", err)
		return
	}
	defer videoConn.Close()

	// Handshake di Video Socket (scrcpy 2.x protocol):
	// a. 1 byte dummy (0x00)
	dummy := make([]byte, 1)
	if _, err := io.ReadFull(videoConn, dummy); err != nil {
		log.Println("Gagal baca dummy byte video socket:", err)
		return
	}

	// 2. Konek ke Control Socket SEGERA setelah dummy byte!
	// Scrcpy server's DesktopConnection.open blocks on accept() for the control socket
	// BEFORE sending the device metadata on the video socket.
	controlConn, err := net.Dial("unix", "@scrcpy")
	if err != nil {
		log.Println("Gagal konek control socket:", err)
		return
	}
	defer controlConn.Close()

	// b. 64 bytes nama device (NUL-padded UTF-8)
	devName := make([]byte, 64)
	if _, err := io.ReadFull(videoConn, devName); err != nil {
		log.Println("Gagal baca device name:", err)
		return
	}
	cleanDevName := strings.TrimRight(string(devName), "\x00")

	// c. 4 bytes Video Codec ID (contoh: "h264" = 0x68323634)
	codecId := make([]byte, 4)
	if _, err := io.ReadFull(videoConn, codecId); err != nil {
		log.Println("Gagal baca codec id:", err)
		return
	}

	// d. 12 bytes Video Meta: 4 bytes flag (0x80000000) + 4 bytes width (uint32 BE) + 4 bytes height (uint32 BE)
	dimBuf := make([]byte, 12)
	if _, err := io.ReadFull(videoConn, dimBuf); err != nil {
		log.Println("Gagal baca dimensi video:", err)
		return
	}
	deviceWidth := int(binary.BigEndian.Uint32(dimBuf[4:8]))
	deviceHeight := int(binary.BigEndian.Uint32(dimBuf[8:12]))

	log.Printf("✅ Scrcpy Stream Terhubung! Device: %s, Codec: %s, Resolusi: %dx%d", cleanDevName, string(codecId), deviceWidth, deviceHeight)

	// Kirim metadata ke browser sebagai paket 0xFF:
	// 1B: 0xFF | 2B: deviceWidth | 2B: deviceHeight
	meta := []byte{
		0xFF,
		byte(deviceWidth >> 8), byte(deviceWidth),
		byte(deviceHeight >> 8), byte(deviceHeight),
	}
	if err := ws.WriteMessage(websocket.BinaryMessage, meta); err != nil {
		log.Println("Gagal kirim meta ke WS:", err)
		return
	}

	done := make(chan struct{})

	// Goroutine: Baca raw H.264 dari video socket → kirim ke WebSocket
	go func() {
		defer close(done)
		buf := make([]byte, 65536)
		for {
			n, err := videoConn.Read(buf)
			if err != nil {
				break
			}
			err = ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				break
			}
		}
		log.Println("❌ Scrcpy Video Stream Terputus.")
	}()

	// Goroutine: Baca control events dari WebSocket → kirim ke control socket
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}
			// Debug: log every control packet
			if len(msg) > 0 {
				msgType := msg[0]
				typeName := "UNKNOWN"
				switch msgType {
				case 0:
					typeName = "INJECT_KEYCODE"
				case 1:
					typeName = "INJECT_TEXT"
				case 2:
					typeName = "INJECT_TOUCH"
				case 3:
					typeName = "INJECT_SCROLL"
				}
				log.Printf("🎮 Control packet: type=%d (%s) size=%d hex=%s", msgType, typeName, len(msg), hex.EncodeToString(msg))
				if msgType == 2 && len(msg) >= 32 {
					action := msg[1]
					ptrHi := binary.BigEndian.Uint32(msg[2:6])
					ptrLo := binary.BigEndian.Uint32(msg[6:10])
					posX := int32(binary.BigEndian.Uint32(msg[10:14]))
					posY := int32(binary.BigEndian.Uint32(msg[14:18]))
					scrW := binary.BigEndian.Uint16(msg[18:20])
					scrH := binary.BigEndian.Uint16(msg[20:22])
					pressure := binary.BigEndian.Uint16(msg[22:24])
					actionBtn := binary.BigEndian.Uint32(msg[24:28])
					buttons := binary.BigEndian.Uint32(msg[28:32])
					log.Printf("   → Touch: action=%d ptr=0x%08X%08X pos=(%d,%d) screen=%dx%d pressure=0x%04X actionBtn=%d buttons=%d",
						action, ptrHi, ptrLo, posX, posY, scrW, scrH, pressure, actionBtn, buttons)
				}
			}
			// Relay pesan control event (abaikan paket meta 0xFF jika ada)
			if len(msg) > 0 && msg[0] != 0xFF {
				_, writeErr := controlConn.Write(msg)
				if writeErr != nil {
					log.Println("Error forward control packet:", writeErr)
					break
				}
			}
		}
	}()

	<-done
	log.Println("❌ Scrcpy Session Berakhir.")
}
