package exportbundle

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"git.jdbnet.co.uk/jamie/icetray/images"
)

// Stream is a portable stream entry in a bundle.
type Stream struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Image string `json:"image,omitempty"`
}

const (
	FormatName    = "icetray-bundle"
	FormatVersion = 1
	manifestName  = "manifest.json"
	dataName      = "data.json"
	imagesPrefix  = "images/"
	maxZipBytes   = 50 * 1024 * 1024
	maxImageBytes = 4 * 1024 * 1024
)

// Manifest describes the archive format.
type Manifest struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
}

// Data is the portable stream library payload.
type Data struct {
	Streams  []Stream `json:"streams"`
	Autoplay bool     `json:"autoplay,omitempty"`
	Volume   int      `json:"volume,omitempty"`
}

// Write creates a zip archive with streams, optional settings, and artwork files.
func Write(w io.Writer, data Data, imagesDir string) error {
	zw := zip.NewWriter(w)

	manifest := Manifest{Format: FormatName, Version: FormatVersion}
	if err := writeJSONEntry(zw, manifestName, manifest); err != nil {
		return err
	}
	if err := writeJSONEntry(zw, dataName, data); err != nil {
		return err
	}

	for _, stream := range data.Streams {
		if stream.Image == "" {
			continue
		}
		src := images.ImagePath(imagesDir, stream.Image)
		info, err := os.Stat(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if info.Size() > maxImageBytes {
			return fmt.Errorf("image too large: %s", stream.Image)
		}
		body, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		name := imagesPrefix + filepath.Base(stream.Image)
		if err := writeFileEntry(zw, name, body); err != nil {
			return err
		}
	}

	return zw.Close()
}

// ReadResult is a validated import bundle.
type ReadResult struct {
	Data   Data
	Images map[string][]byte
}

// Read parses and validates a bundle from a zip file path.
func Read(zipPath string) (ReadResult, error) {
	info, err := os.Stat(zipPath)
	if err != nil {
		return ReadResult{}, err
	}
	if info.Size() > maxZipBytes {
		return ReadResult{}, fmt.Errorf("archive is too large")
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return ReadResult{}, fmt.Errorf("invalid archive: %w", err)
	}
	defer r.Close()

	var manifest Manifest
	var data Data
	imageFiles := make(map[string][]byte)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.UncompressedSize64 > maxImageBytes && strings.HasPrefix(f.Name, imagesPrefix) {
			return ReadResult{}, fmt.Errorf("image entry too large: %s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return ReadResult{}, err
		}
		body, err := io.ReadAll(io.LimitReader(rc, maxImageBytes+1))
		rc.Close()
		if err != nil {
			return ReadResult{}, err
		}

		switch f.Name {
		case manifestName:
			if err := json.Unmarshal(body, &manifest); err != nil {
				return ReadResult{}, fmt.Errorf("invalid manifest: %w", err)
			}
		case dataName:
			if err := json.Unmarshal(body, &data); err != nil {
				return ReadResult{}, fmt.Errorf("invalid data: %w", err)
			}
		default:
			if !strings.HasPrefix(f.Name, imagesPrefix) {
				continue
			}
			clean := path.Clean(f.Name)
			if clean != f.Name || strings.Contains(clean, "..") {
				return ReadResult{}, fmt.Errorf("invalid path in archive: %s", f.Name)
			}
			base := filepath.Base(clean)
			if base == "." || base == "" {
				continue
			}
			if len(body) > maxImageBytes {
				return ReadResult{}, fmt.Errorf("image too large: %s", base)
			}
			imageFiles[base] = body
		}
	}

	if manifest.Format != FormatName || manifest.Version != FormatVersion {
		return ReadResult{}, fmt.Errorf("unsupported bundle format (expected %s v%d)", FormatName, FormatVersion)
	}
	if len(data.Streams) == 0 {
		return ReadResult{}, fmt.Errorf("bundle contains no streams")
	}

	normalizeStreamImages(&data, imageFiles)
	return ReadResult{Data: data, Images: imageFiles}, nil
}

func normalizeStreamImages(data *Data, files map[string][]byte) {
	for i := range data.Streams {
		stream := &data.Streams[i]
		if stream.Image == "" {
			continue
		}
		base := filepath.Base(stream.Image)
		if _, ok := files[base]; !ok {
			stream.Image = ""
			continue
		}
		stream.Image = base
	}
}

func writeJSONEntry(zw *zip.Writer, name string, value any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return err
	}
	return writeFileEntry(zw, name, buf.Bytes())
}

func writeFileEntry(zw *zip.Writer, name string, body []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}
