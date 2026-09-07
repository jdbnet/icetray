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

const maxEdge = 512

// SaveStreamImage decodes the image, scales it so the longest edge is at most
// maxEdge, and writes a PNG. Aspect ratio is preserved. Passing both a width
// and height to resize.Resize stretches to a square, which is what we avoid.
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

	resized := fitImage(img, width, height)
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

func fitImage(img image.Image, width, height int) image.Image {
	if width <= maxEdge && height <= maxEdge {
		return img
	}
	if width >= height {
		return resize.Resize(maxEdge, 0, img, resize.Lanczos3)
	}
	return resize.Resize(0, maxEdge, img, resize.Lanczos3)
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
