package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// Stream represents a saved radio stream.
type Stream struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Image string `json:"image,omitempty"`
}

// Config represents the application configuration.
type Config struct {
	Streams       []Stream `json:"streams"`
	LastStream    string   `json:"last_stream"`
	LastStreamID  string   `json:"last_stream_id,omitempty"`
	Autoplay         bool `json:"autoplay"`
	Volume           int  `json:"volume"`
	LaunchOnLogin    bool `json:"launch_on_login"`
	LaunchMinimized  bool `json:"launch_minimized"`

	configPath string
	imagesDir  string
	mu         sync.RWMutex
}

// LoadConfig loads configuration from the config directory.
func LoadConfig(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.json")
	imagesDir := filepath.Join(configDir, "images")

	cfg := &Config{
		configPath: configPath,
		imagesDir:  imagesDir,
		Streams:    []Stream{},
		Volume:     50,
	}

	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create images directory: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := cfg.Save(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.configPath = configPath
	cfg.imagesDir = imagesDir
	cfg.migrate()

	return cfg, nil
}

func (c *Config) migrate() {
	changed := false
	for i := range c.Streams {
		if c.Streams[i].ID == "" {
			c.Streams[i].ID = uuid.NewString()
			changed = true
		}
	}
	if c.LastStreamID == "" && c.LastStream != "" {
		for _, s := range c.Streams {
			if s.URL == c.LastStream {
				c.LastStreamID = s.ID
				changed = true
				break
			}
		}
	}
	if changed {
		_ = c.Save()
	}
}

// ImagesDir returns the path where stream artwork is stored.
func (c *Config) ImagesDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.imagesDir
}

// Save writes the current config to disk.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddStream adds a new stream and saves the config.
func (c *Config) AddStream(name, url string) (Stream, error) {
	c.mu.Lock()
	stream := Stream{
		ID:   uuid.NewString(),
		Name: name,
		URL:  url,
	}
	c.Streams = append(c.Streams, stream)
	c.mu.Unlock()
	return stream, c.Save()
}

// UpdateStream updates an existing stream by ID.
func (c *Config) UpdateStream(id, name, url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.Streams {
		if c.Streams[i].ID == id {
			c.Streams[i].Name = name
			c.Streams[i].URL = url
			return c.saveLocked()
		}
	}
	return fmt.Errorf("stream not found: %s", id)
}

// SetStreamImage sets the image filename for a stream.
func (c *Config) SetStreamImage(id, image string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.Streams {
		if c.Streams[i].ID == id {
			c.Streams[i].Image = image
			return c.saveLocked()
		}
	}
	return fmt.Errorf("stream not found: %s", id)
}

// ReorderStreams rewrites the stream list to match the given IDs and saves.
func (c *Config) ReorderStreams(ids []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(ids) != len(c.Streams) {
		return fmt.Errorf("stream order length mismatch")
	}

	byID := make(map[string]Stream, len(c.Streams))
	for _, s := range c.Streams {
		if _, exists := byID[s.ID]; exists {
			return fmt.Errorf("duplicate stream id: %s", s.ID)
		}
		byID[s.ID] = s
	}

	next := make([]Stream, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		s, ok := byID[id]
		if !ok {
			return fmt.Errorf("stream not found: %s", id)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("duplicate stream id: %s", id)
		}
		seen[id] = struct{}{}
		next = append(next, s)
	}

	c.Streams = next
	return c.saveLocked()
}

// RemoveStreamByID removes a stream by ID and saves the config.
func (c *Config) RemoveStreamByID(id string) (Stream, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, s := range c.Streams {
		if s.ID == id {
			c.Streams = append(c.Streams[:i], c.Streams[i+1:]...)
			if c.LastStreamID == id {
				c.LastStreamID = ""
				c.LastStream = ""
			}
			if err := c.saveLocked(); err != nil {
				return Stream{}, err
			}
			return s, nil
		}
	}
	return Stream{}, fmt.Errorf("stream not found: %s", id)
}

// RemoveStream removes a stream by URL and saves the config.
func (c *Config) RemoveStream(url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, s := range c.Streams {
		if s.URL == url {
			c.Streams = append(c.Streams[:i], c.Streams[i+1:]...)
			if c.LastStream == url {
				c.LastStream = ""
				c.LastStreamID = ""
			}
			return c.saveLocked()
		}
	}
	return nil
}

func (c *Config) saveLocked() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// GetStreamByID returns a stream by ID.
func (c *Config) GetStreamByID(id string) (Stream, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, s := range c.Streams {
		if s.ID == id {
			return s, true
		}
	}
	return Stream{}, false
}

// SetLastStream sets the last played stream and saves the config.
func (c *Config) SetLastStream(url string) error {
	c.mu.Lock()
	c.LastStream = url
	c.LastStreamID = ""
	for _, s := range c.Streams {
		if s.URL == url {
			c.LastStreamID = s.ID
			break
		}
	}
	c.mu.Unlock()
	return c.Save()
}

// SetLastStreamID sets the last played stream by ID.
func (c *Config) SetLastStreamID(id string) error {
	c.mu.Lock()
	c.LastStreamID = id
	c.LastStream = ""
	for _, s := range c.Streams {
		if s.ID == id {
			c.LastStream = s.URL
			break
		}
	}
	c.mu.Unlock()
	return c.Save()
}

// SetAutoplay sets the autoplay setting and saves the config.
func (c *Config) SetAutoplay(autoplay bool) error {
	c.mu.Lock()
	c.Autoplay = autoplay
	c.mu.Unlock()
	return c.Save()
}

// SetVolume sets the volume and saves the config.
func (c *Config) SetVolume(volume int) error {
	c.mu.Lock()
	if volume < 0 {
		volume = 0
	} else if volume > 100 {
		volume = 100
	}
	c.Volume = volume
	c.mu.Unlock()
	return c.Save()
}

// SetLaunchOnLogin sets the launch on login setting and saves the config.
func (c *Config) SetLaunchOnLogin(launchOnLogin bool) error {
	c.mu.Lock()
	c.LaunchOnLogin = launchOnLogin
	c.mu.Unlock()
	return c.Save()
}

// SetLaunchMinimized sets whether the player window stays hidden until opened from the tray.
func (c *Config) SetLaunchMinimized(launchMinimized bool) error {
	c.mu.Lock()
	c.LaunchMinimized = launchMinimized
	c.mu.Unlock()
	return c.Save()
}

// GetVolume returns the current volume.
func (c *Config) GetVolume() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Volume
}

// GetStreams returns a copy of the streams list.
func (c *Config) GetStreams() []Stream {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Stream{}, c.Streams...)
}

// GetLastStream returns the last stream URL.
func (c *Config) GetLastStream() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastStream
}

// GetLastStreamID returns the last played stream ID.
func (c *Config) GetLastStreamID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastStreamID
}

// GetAutoplay returns the autoplay setting.
func (c *Config) GetAutoplay() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Autoplay
}

// GetLaunchOnLogin returns the launch on login setting.
func (c *Config) GetLaunchOnLogin() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LaunchOnLogin
}

// GetLaunchMinimized returns whether the player window should start hidden.
func (c *Config) GetLaunchMinimized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LaunchMinimized
}
