package exportbundle

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

)

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatal(err)
	}

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "station-a.png"), png, 0644); err != nil {
		t.Fatal(err)
	}

	data := Data{
		Streams: []Stream{
			{ID: "station-a", Name: "Test FM", URL: "https://example.com/stream", Image: "station-a.png"},
		},
		Autoplay: true,
		Volume:   42,
	}

	var buf bytes.Buffer
	if err := Write(&buf, data, imagesDir); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(dir, "export.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Read(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Data.Streams) != 1 || got.Data.Streams[0].Name != "Test FM" {
		t.Fatalf("unexpected streams: %+v", got.Data.Streams)
	}
	if !got.Data.Autoplay || got.Data.Volume != 42 {
		t.Fatalf("unexpected settings: autoplay=%v volume=%d", got.Data.Autoplay, got.Data.Volume)
	}
	if len(got.Images["station-a.png"]) == 0 {
		t.Fatal("expected image bytes in bundle")
	}
}
