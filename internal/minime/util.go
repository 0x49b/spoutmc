package minime

import (
	"image"
	"image/color"
)

func findSuitablePixel(img image.Image, overlayX, overlayY, normalX, normalY int) color.Color {
	r, g, b, a := img.At(overlayX, overlayY).RGBA()
	if a == 0 {
		r, g, b, a = img.At(normalX, normalY).RGBA()
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
