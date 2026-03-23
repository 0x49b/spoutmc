package processor

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
	"github.com/twistedasylummc/minime"
)

// DefaultAvatarSize is the edge length (px) of the square avatar after upscaling the minime.
// Pixel-art upscales stay sharp with NearestNeighbor; 512 gives a crisp header/profile image.
const DefaultAvatarSize = 512

func LoadImage(input string, isURL bool) (image.Image, error) {
	var reader io.Reader
	if isURL {
		resp, err := http.Get(input)
		if err != nil {
			return nil, fmt.Errorf("failed to download image: %v", err)
		}
		defer resp.Body.Close()
		reader = resp.Body
	} else {
		file, err := os.Open(input)
		if err != nil {
			return nil, fmt.Errorf("failed to open input file: %v", err)
		}
		defer file.Close()
		reader = file
	}
	return png.Decode(reader)
}

func ProcessSkin(input string, isURL bool, slim bool, size int) (image.Image, error) {
	src, err := LoadImage(input, isURL)
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	if !(bounds.Dx() == 64 && bounds.Dy() == 64) && !(bounds.Dx() == 128 && bounds.Dy() == 128) {
		return nil, fmt.Errorf("input must be 64x64 or 128x128, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// resize.Resize(160, 0, original_image, resize.Lanczos3)
	var skinImage image.Image

	if bounds.Dx() == 64 {
		skinImage = minime.Skin64(src)
	} else {
		skinImage = minime.Skin128(src, slim)
	}

	// Minime output is very small (e.g. 18×28); Bicubic blurs pixel edges and thickens outlines.
	// Nearest-neighbor preserves hard edges for Minecraft-style sprites.
	return resize.Resize(uint(size), uint(size), skinImage, resize.NearestNeighbor), nil
}

func EncodeToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func SaveImageToFile(img image.Image, outputPath string) error {
	if outputPath == "" {
		return errors.New("output path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %v", err)
	}
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()
	return png.Encode(outFile, img)
}
