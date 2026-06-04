package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Stream represents a saved radio stream.
type Stream struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Config represents the application configuration.
type Config struct {
	Streams       []Stream `json:"streams"`
	LastStream    string   `json:"last_stream"`
	Autoplay      bool     `json:"autoplay"`
	Volume        int      `json:"volume"`
	LaunchOnLogin bool     `json:"launch_on_login"`

	// Internal fields
	configPath string
	mu         sync.RWMutex
}

// LoadConfig loads configuration from the config directory.
// If the config file doesn't exist, it creates a default config.
func LoadConfig(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.json")

	cfg := &Config{
		configPath:    configPath,
		Streams:       []Stream{},
		LastStream:    "",
		Autoplay:      false,
		Volume:        50,
		LaunchOnLogin: false,
	}

	// Try to read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			if err := cfg.Save(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.configPath = configPath
	return cfg, nil
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
func (c *Config) AddStream(name, url string) error {
	c.mu.Lock()
	c.Streams = append(c.Streams, Stream{Name: name, URL: url})
	c.mu.Unlock()
	return c.Save()
}

// RemoveStream removes a stream by URL and saves the config.
func (c *Config) RemoveStream(url string) error {
	c.mu.Lock()
	for i, s := range c.Streams {
		if s.URL == url {
			c.Streams = append(c.Streams[:i], c.Streams[i+1:]...)
			c.mu.Unlock()
			return c.Save()
		}
	}
	c.mu.Unlock()
	return nil
}

// SetLastStream sets the last played stream and saves the config.
func (c *Config) SetLastStream(url string) error {
	c.mu.Lock()
	c.LastStream = url
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
