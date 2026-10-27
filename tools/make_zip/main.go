package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	srcDir := "magisk_module"
	zipPath := "ProjectX_Magisk.zip"

	_ = os.Remove(zipPath)

	zipFile, err := os.Create(zipPath)
	if err != nil {
		fmt.Printf("Error creating zip: %v\n", err)
		os.Exit(1)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	count := 0
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// MUST USE FORWARD SLASHES FOR UNIX / ANDROID COMPATIBILITY
		zipEntryName := strings.ReplaceAll(relPath, `\`, `/`)

		if info.IsDir() {
			zipEntryName += "/"
			header := &zip.FileHeader{
				Name:   zipEntryName,
				Method: zip.Deflate,
			}
			header.SetMode(info.Mode() | 0755)
			_, err = w.CreateHeader(header)
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = zipEntryName
		header.Method = zip.Deflate

		// Executable permissions for scripts & binaries
		if strings.HasSuffix(zipEntryName, ".sh") || strings.HasPrefix(zipEntryName, "bin/") || strings.Contains(zipEntryName, "update-binary") {
			header.SetMode(0755)
		} else {
			header.SetMode(0644)
		}

		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			return err
		}

		count++
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created %s with %d files using UNIX forward-slash paths.\n", zipPath, count)
}
