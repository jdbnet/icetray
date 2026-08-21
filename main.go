package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"git.jdbnet.co.uk/jamie/icetray/assets"
	"git.jdbnet.co.uk/jamie/icetray/config"
	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
	"git.jdbnet.co.uk/jamie/icetray/startup"
	"git.jdbnet.co.uk/jamie/icetray/stream"
)

func main() {
	streamURL := flag.String("stream", "", "Stream URL to play directly in terminal mode")
	flag.Parse()

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

	// Terminal direct play mode
	if *streamURL != "" {
		logger.Log("Running in terminal mode, playing stream: " + *streamURL)
		
		player := player.NewPlayer()
		defer player.Close()

		cfg, err := config.LoadConfig(configDir)
		if err == nil {
			player.SetVolume(cfg.GetVolume())
		} else {
			player.SetVolume(100)
		}

		supervisor := stream.NewSupervisor(player)
		defer supervisor.Stop()

		if err := player.Play(*streamURL); err != nil {
			logger.LogFatal("Failed to start stream", err)
		}
		supervisor.Start(*streamURL)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		
		fmt.Printf("Playing stream: %s\nPress Ctrl+C to stop...\n", *streamURL)
		<-sigChan
		fmt.Println("\nStopping playback...")
		return
	}

	// If running on Linux, check and install if running from non-target path
	checkAndInstallLinux()

	// If running on Windows, check and install if running from non-target path
	checkAndInstallWindows()

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

	// Initialize and run system tray (or block if headless)
	runHeaded(cfg, player, supervisor, startupMgr)

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

// checkAndInstallLinux checks if the application is running from the target local bin directory.
// If not, it copies itself there, installs the desktop entry and icon, launches the installed binary, and exits.
func checkAndInstallLinux() {
	if runtime.GOOS != "linux" {
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	targetDir := filepath.Join(homeDir, ".local", "bin")
	targetPath := filepath.Join(targetDir, "icetray")

	// If the current executable is already the installed one, do nothing
	if execPath == targetPath {
		return
	}

	logger.Log("Running from temporary location. Installing to " + targetPath)
	fmt.Printf("Installing IceTray to %s...\n", targetPath)

	// Create directories
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		logger.LogError("Failed to create bin directory", err)
		return
	}

	iconDir := filepath.Join(homeDir, ".local", "share", "icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		logger.LogError("Failed to create icons directory", err)
		return
	}

	appDir := filepath.Join(homeDir, ".local", "share", "applications")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		logger.LogError("Failed to create applications directory", err)
		return
	}

	// Remove old binary if it exists to avoid "text file busy" error
	os.Remove(targetPath)

	// Copy binary
	srcFile, err := os.Open(execPath)
	if err != nil {
		logger.LogError("Failed to open source binary", err)
		return
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		logger.LogError("Failed to open destination binary", err)
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		logger.LogError("Failed to copy binary", err)
		return
	}
	destFile.Close()

	// Write icon
	iconPath := filepath.Join(iconDir, "icetray.png")
	if err := os.WriteFile(iconPath, assets.Icon, 0644); err != nil {
		logger.LogError("Failed to write icon", err)
		return
	}

	// Write desktop entry
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=IceTray
Comment=Internet Radio Player
Exec=%s
Icon=%s
Terminal=false
Categories=AudioVideo;Audio;Player;
`, targetPath, iconPath)

	desktopPath := filepath.Join(appDir, "icetray.desktop")
	if err := os.WriteFile(desktopPath, []byte(desktopContent), 0644); err != nil {
		logger.LogError("Failed to write desktop entry", err)
		return
	}

	// Run update-desktop-database if available
	if path, err := exec.LookPath("update-desktop-database"); err == nil {
		exec.Command(path, appDir).Run()
	}

	logger.Log("Installation successful. Launching target binary: " + targetPath)
	fmt.Println("IceTray installed successfully! Launching...")

	// Launch the newly installed binary
	cmd := exec.Command(targetPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		logger.LogError("Failed to start installed binary", err)
		return
	}

	// Exit the current process
	os.Exit(0)
}

// checkAndInstallWindows checks if the application is running from the target local AppData directory.
// If not, it copies itself there, creates a Start Menu shortcut, launches the installed binary, and exits.
func checkAndInstallWindows() {
	if runtime.GOOS != "windows" {
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return
	}

	appData := os.Getenv("APPDATA")
	if appData == "" {
		return
	}

	targetDir := filepath.Join(appData, "IceTray")
	targetPath := filepath.Join(targetDir, "icetray.exe")

	// If the current executable is already the installed one, do nothing
	if execPath == targetPath {
		return
	}

	logger.Log("Running from temporary location. Installing to " + targetPath)
	fmt.Printf("Installing IceTray to %s...\n", targetPath)

	// Create directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		logger.LogError("Failed to create target directory", err)
		return
	}

	// Remove old binary if it exists to avoid "text file busy" error
	os.Remove(targetPath)

	// Copy binary
	srcFile, err := os.Open(execPath)
	if err != nil {
		logger.LogError("Failed to open source binary", err)
		return
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		logger.LogError("Failed to open destination binary", err)
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		logger.LogError("Failed to copy binary", err)
		return
	}
	destFile.Close()

	// Create Start Menu shortcut
	shortcutDir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs")
	shortcutPath := filepath.Join(shortcutDir, "IceTray.lnk")

	powershellCmd := fmt.Sprintf(`$WshShell = New-Object -ComObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%s'); $Shortcut.TargetPath = '%s'; $Shortcut.WorkingDirectory = '%s'; $Shortcut.Save()`, shortcutPath, targetPath, targetDir)

	// Execute PowerShell to create the shortcut
	ps := exec.Command("powershell", "-NoProfile", "-Command", powershellCmd)
	if err := ps.Run(); err != nil {
		logger.LogError("Failed to create Start Menu shortcut", err)
	}

	logger.Log("Installation successful. Launching target binary: " + targetPath)
	fmt.Println("IceTray installed successfully! Launching...")

	// Launch the newly installed binary
	cmd := exec.Command(targetPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		logger.LogError("Failed to start installed binary", err)
		return
	}

	// Exit the current process
	os.Exit(0)
}
