package main

import (
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
)

func compress(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Err open:", err)
		return
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		fmt.Println("Err decode:", err)
		return
	}

	tmpPath := path + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		fmt.Println("Err create:", err)
		return
	}

	err = jpeg.Encode(out, img, &jpeg.Options{Quality: 82})
	out.Close()
	if err != nil {
		fmt.Println("Err encode:", err)
		return
	}

	_ = os.Rename(tmpPath, path)
	fi, _ := os.Stat(path)
	fmt.Printf("Compressed %s -> %d KB\n", filepath.Base(path), fi.Size()/1024)
}

func main() {
	files := []string{
		"ui-svelte/public/assets/y2k-desktop-light.jpg",
		"ui-svelte/public/assets/y2k-desktop-dark.jpg",
		"ui-svelte/public/assets/y2k-mobile-light.jpg",
		"ui-svelte/public/assets/y2k-mobile-dark.jpg",
	}
	for _, p := range files {
		compress(p)
	}
}
