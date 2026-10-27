package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func read(p string) string {
	b, _ := os.ReadFile(p)
	return strings.TrimSpace(string(b))
}

func main() {
	path := "/sys/class/power_supply/battery/charging_enabled"
	statPath := "/sys/class/power_supply/battery/status"
	currPath := "/sys/class/power_supply/battery/current_now"

	fmt.Printf("[INITIAL] charging_enabled: %s | status: %s | current: %s uA\n",
		read(path), read(statPath), read(currPath))

	// Turn charging OFF
	os.WriteFile(path, []byte("0\n"), 0666)
	time.Sleep(2 * time.Second)
	fmt.Printf("[AFTER OFF] charging_enabled: %s | status: %s | current: %s uA\n",
		read(path), read(statPath), read(currPath))

	// Turn charging ON
	os.WriteFile(path, []byte("1\n"), 0666)
	time.Sleep(2 * time.Second)
	fmt.Printf("[RESTORE ON] charging_enabled: %s | status: %s | current: %s uA\n",
		read(path), read(statPath), read(currPath))
}
