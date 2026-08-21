package images

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
)

const targetSize = 512

// SaveStreamImage decodes, resizes to 512x512, and saves as PNG.
func SaveStreamImage(imagesDir, streamID string, data []byte) (string, error) {
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return "", err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("invalid image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 32 || height < 32 {
		return "", fmt.Errorf("image too small")
	}

	resized := resize.Resize(targetSize, targetSize, img, resize.Lanczos3)
	filename := streamID + ".png"
	outPath := filepath.Join(imagesDir, filename)

	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := png.Encode(f, resized); err != nil {
		os.Remove(outPath)
		return "", err
	}

	return filename, nil
}

// DeleteStreamImage removes a stream image file if present.
func DeleteStreamImage(imagesDir, filename string) {
	if filename == "" {
		return
	}
	_ = os.Remove(filepath.Join(imagesDir, filename))
}

// ImagePath returns the full path to a stream image.
func ImagePath(imagesDir, filename string) string {
	if filename == "" {
		return ""
	}
	return filepath.Join(imagesDir, filename)
}
