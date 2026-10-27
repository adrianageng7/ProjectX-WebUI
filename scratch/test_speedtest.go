package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("Testing Cloudflare speed test ping & download...")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Ping
	var rtts []int64
	for i := 0; i < 5; i++ {
		start := time.Now()
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://speed.cloudflare.com/__down?bytes=0", nil)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Ping %d failed: %v\n", i, err)
			return
		}
		resp.Body.Close()
		rtt := time.Since(start).Milliseconds()
		rtts = append(rtts, rtt)
		fmt.Printf("Ping %d: %d ms, Colo: %s, IP: %s\n", i+1, rtt, resp.Header.Get("cf-meta-colo"), resp.Header.Get("cf-meta-ip"))
	}

	// 2. Download test (5MB)
	fmt.Println("\nTesting 5MB Download...")
	dlStart := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://speed.cloudflare.com/__down?bytes=5000000", nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	buf := make([]byte, 32*1024)
	var totalRead int64
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			totalRead += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			break
		}
	}
	dlDuration := time.Since(dlStart).Seconds()
	mbps := (float64(totalRead) * 8) / (dlDuration * 1e6)
	fmt.Printf("Downloaded %d bytes in %.2f s -> Speed: %.2f Mbps\n", totalRead, dlDuration, mbps)
}

