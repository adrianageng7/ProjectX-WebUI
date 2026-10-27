package main

import (
	"fmt"
	"net/http"
)

func main() {
	resp, err := http.Get("https://speed.cloudflare.com/__down?bytes=0")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Status:", resp.Status)
	for k, v := range resp.Header {
		fmt.Printf("%s: %v\n", k, v)
	}
}

