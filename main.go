package main

import (
	"fmt"
	"os"
	"path/filepath"

	"git.jdbnet.co.uk/jamie/icetray/config"
	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
	"git.jdbnet.co.uk/jamie/icetray/startup"
	"git.jdbnet.co.uk/jamie/icetray/stream"
	"git.jdbnet.co.uk/jamie/icetray/tray"
)

func main() {
	// Get or create config directory
	configDir, err := getConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get config directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(configDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Log("IceTray starting up")

	// Load configuration
	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		logger.LogFatal("Failed to load config", err)
	}

	logger.Log("Config loaded successfully")

	// Initialize player
	player := player.NewPlayer()
	defer player.Close()

	// Initialize stream supervisor
	supervisor := stream.NewSupervisor(player)
	defer supervisor.Stop()

	// Initialize startup manager
	startupMgr := startup.New()

	// Apply launch on login setting from config
	if cfg.GetLaunchOnLogin() {
		if err := startupMgr.Enable(); err != nil {
			logger.LogError("Failed to enable launch on login", err)
		}
	}

	// If autoplay is enabled and we have a last stream, start playing
	if cfg.GetAutoplay() && cfg.GetLastStream() != "" {
		logger.Log("Autoplay enabled, starting stream: " + cfg.GetLastStream())
		if err := player.Play(cfg.GetLastStream()); err != nil {
			logger.LogError("Failed to start autoplay stream", err)
		} else {
			supervisor.Start(cfg.GetLastStream())
			// Set initial volume
			player.SetVolume(cfg.GetVolume())
		}
	}

	// Initialize and run system tray
	logger.Log("Initializing system tray")
	trayMgr := tray.NewTrayManager(cfg, player, supervisor, startupMgr)
	trayMgr.Init() // This is a blocking call

	logger.Log("IceTray shutting down")
}

// getConfigDir returns the IceTray config directory, creating it if necessary.
func getConfigDir() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(baseDir, "IceTray")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}
