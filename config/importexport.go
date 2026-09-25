package config

import (
	"os"

	"github.com/google/uuid"

	"github.com/jdbnet/icetray/exportbundle"
	"github.com/jdbnet/icetray/images"
)

// ApplyBundle replaces or merges streams (and portable settings) from an export bundle.
func (c *Config) ApplyBundle(bundle exportbundle.ReadResult, replace bool) (int, error) {
	if replace {
		return c.replaceBundle(bundle)
	}
	return c.mergeBundle(bundle)
}

func (c *Config) replaceBundle(bundle exportbundle.ReadResult) (int, error) {
	c.mu.Lock()
	for _, s := range c.Streams {
		images.DeleteStreamImage(c.imagesDir, s.Image)
	}
	c.Streams = []Stream{}
	c.LastStream = ""
	c.LastStreamID = ""
	c.mu.Unlock()

	if err := c.writeBundleImages(bundle); err != nil {
		return 0, err
	}

	c.mu.Lock()
	c.Streams = streamsFromBundle(bundle.Data.Streams)
	c.Autoplay = bundle.Data.Autoplay
	if bundle.Data.Volume > 0 {
		c.Volume = bundle.Data.Volume
	}
	c.mu.Unlock()

	if err := c.Save(); err != nil {
		return 0, err
	}
	return len(bundle.Data.Streams), nil
}

func (c *Config) mergeBundle(bundle exportbundle.ReadResult) (int, error) {
	if err := c.writeBundleImages(bundle); err != nil {
		return 0, err
	}

	c.mu.Lock()
	existingURL := make(map[string]struct{}, len(c.Streams))
	existingID := make(map[string]struct{}, len(c.Streams))
	for _, s := range c.Streams {
		existingURL[s.URL] = struct{}{}
		existingID[s.ID] = struct{}{}
	}

	added := 0
	for _, stream := range bundle.Data.Streams {
		if _, ok := existingURL[stream.URL]; ok {
			continue
		}
		imported := streamFromBundle(stream)
		if imported.ID == "" {
			imported.ID = uuid.NewString()
		}
		if _, ok := existingID[imported.ID]; ok {
			oldImage := imported.Image
			imported.ID = uuid.NewString()
			if oldImage != "" {
				oldPath := images.ImagePath(c.imagesDir, oldImage)
				newName := imported.ID + ".png"
				newPath := images.ImagePath(c.imagesDir, newName)
				if err := os.Rename(oldPath, newPath); err == nil {
					imported.Image = newName
				} else {
					imported.Image = ""
				}
			}
		}
		existingID[imported.ID] = struct{}{}
		existingURL[imported.URL] = struct{}{}
		c.Streams = append(c.Streams, imported)
		added++
	}
	c.mu.Unlock()

	if added == 0 {
		return 0, nil
	}
	if err := c.Save(); err != nil {
		return 0, err
	}
	return added, nil
}

func (c *Config) writeBundleImages(bundle exportbundle.ReadResult) error {
	c.mu.RLock()
	imagesDir := c.imagesDir
	c.mu.RUnlock()

	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return err
	}

	needed := make(map[string]struct{})
	for _, s := range bundle.Data.Streams {
		if s.Image != "" {
			needed[s.Image] = struct{}{}
		}
	}

	for name, body := range bundle.Images {
		if _, ok := needed[name]; !ok {
			continue
		}
		outPath := images.ImagePath(imagesDir, name)
		if err := os.WriteFile(outPath, body, 0644); err != nil {
			return err
		}
	}
	return nil
}

// SnapshotForExport returns portable bundle data for the current library.
func (c *Config) SnapshotForExport() exportbundle.Data {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return exportbundle.Data{
		Streams:  streamsToBundle(c.Streams),
		Autoplay: c.Autoplay,
		Volume:   c.Volume,
	}
}

func streamsToBundle(streams []Stream) []exportbundle.Stream {
	out := make([]exportbundle.Stream, len(streams))
	for i, s := range streams {
		out[i] = exportbundle.Stream{
			ID: s.ID, Name: s.Name, URL: s.URL, Image: s.Image,
		}
	}
	return out
}

func streamsFromBundle(streams []exportbundle.Stream) []Stream {
	out := make([]Stream, len(streams))
	for i, s := range streams {
		out[i] = streamFromBundle(s)
	}
	return out
}

func streamFromBundle(s exportbundle.Stream) Stream {
	return Stream{ID: s.ID, Name: s.Name, URL: s.URL, Image: s.Image}
}
