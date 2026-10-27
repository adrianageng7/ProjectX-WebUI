package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
)

func main() {
	srcPath := `C:\Users\Adrian\.gemini\antigravity\brain\9ca04858-b28d-4beb-8da3-8867ba589763\y2k_mobile_wallpaper_1791168778191.jpg`
	dstPath := `C:\Users\Adrian\.gemini\antigravity\brain\9ca04858-b28d-4beb-8da3-8867ba589763\y2k_mobile_dark.jpg`

	file, err := os.Open(srcPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		panic(err)
	}

	bounds := img.Bounds()
	out := image.NewRGBA(bounds)

	// Target dark canvas color: #17151a (R: 23, G: 21, B: 26)
	darkR, darkG, darkB := 23.0, 21.0, 26.0

	// Cream background target: R ~250, G ~242, B ~228
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r32, g32, b32, a32 := c.RGBA()
			r := float64(r32 >> 8)
			g := float64(g32 >> 8)
			b := float64(b32 >> 8)
			a := uint8(a32 >> 8)

			// Check if pixel is part of the warm cream paper background:
			// Cream is bright, with R > G > B, low saturation.
			// Distance from pure cream (250, 242, 228)
			diffCream := math.Sqrt(math.Pow(r-250, 2) + math.Pow(g-242, 2) + math.Pow(b-228, 2))
			
			// Also check if high luminance and warm:
			isCream := diffCream < 28 || (r > 235 && g > 225 && b > 210 && math.Abs(r-g) < 25 && math.Abs(g-b) < 25)

			if isCream {
				// Replace cream with dark background
				out.SetRGBA(x, y, color.RGBA{R: uint8(darkR), G: uint8(darkG), B: uint8(darkB), A: a})
			} else if diffCream < 45 {
				// Feathering / Anti-aliasing edge
				t := (diffCream - 28.0) / (45.0 - 28.0) // 0 (near cream) to 1 (near foreground)
				finalR := darkR*(1-t) + r*t
				finalG := darkG*(1-t) + g*t
				finalB := darkB*(1-t) + b*t
				out.SetRGBA(x, y, color.RGBA{R: uint8(finalR), G: uint8(finalG), B: uint8(finalB), A: a})
			} else {
				// Keep sticker pixel as is
				out.SetRGBA(x, y, color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: a})
			}
		}
	}

	outFile, err := os.Create(dstPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	err = jpeg.Encode(outFile, out, &jpeg.Options{Quality: 92})
	if err != nil {
		panic(err)
	}

	fmt.Println("Created mobile dark wallpaper successfully:", dstPath)
}
