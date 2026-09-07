package images

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveStreamImagePreservesAspectRatio(t *testing.T) {
	dir := t.TempDir()
	src := image.NewRGBA(image.Rect(0, 0, 500, 118))
	for y := 0; y < 118; y++ {
		for x := 0; x < 500; x++ {
			src.Set(x, y, color.RGBA{R: 0, G: 80, B: 180, A: 255})
		}
	}

	var buf []byte
	{
		tmp := filepath.Join(dir, "src.png")
		f, err := os.Create(tmp)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, src); err != nil {
			t.Fatal(err)
		}
		f.Close()
		data, err := os.ReadFile(tmp)
		if err != nil {
			t.Fatal(err)
		}
		buf = data
	}

	filename, err := SaveStreamImage(dir, "kwik", buf)
	if err != nil {
		t.Fatal(err)
	}

	out, err := os.Open(filepath.Join(dir, filename))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	img, err := png.Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == h {
		t.Fatalf("saved image is square (%dx%d); wide source should stay wide", w, h)
	}
	if w < h {
		t.Fatalf("saved image is tall (%dx%d); expected landscape", w, h)
	}
	ratio := float64(w) / float64(h)
	want := 500.0 / 118.0
	if ratio < want-0.05 || ratio > want+0.05 {
		t.Fatalf("aspect ratio %v want about %v (%dx%d)", ratio, want, w, h)
	}
	if w > maxEdge || h > maxEdge {
		t.Fatalf("longest edge %dx%d exceeds %d", w, h, maxEdge)
	}
}
