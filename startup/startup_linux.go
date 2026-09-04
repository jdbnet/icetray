//go:build linux && !android

package startup

import (
	"fmt"
	"os"
	"path/filepath"
)

type LinuxStartupManager struct{}

func getPlatformStartupManager() StartupManager {
	return &LinuxStartupManager{}
}

func (l *LinuxStartupManager) Enable() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	autoStartDir := filepath.Join(os.Getenv("HOME"), ".config", "autostart")
	if err := os.MkdirAll(autoStartDir, 0755); err != nil {
		return err
	}

	desktopFilePath := filepath.Join(autoStartDir, "icetray.desktop")
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Exec=%s
Name=IceTray
Comment=Icecast Internet Radio Player
Icon=audio-x-generic
Categories=Audio;
Terminal=false
`, exePath)

	return os.WriteFile(desktopFilePath, []byte(desktopContent), 0644)
}

func (l *LinuxStartupManager) Disable() error {
	autoStartDir := filepath.Join(os.Getenv("HOME"), ".config", "autostart")
	desktopFilePath := filepath.Join(autoStartDir, "icetray.desktop")
	return os.Remove(desktopFilePath)
}

func (l *LinuxStartupManager) IsEnabled() bool {
	autoStartDir := filepath.Join(os.Getenv("HOME"), ".config", "autostart")
	desktopFilePath := filepath.Join(autoStartDir, "icetray.desktop")
	_, err := os.Stat(desktopFilePath)
	return err == nil
}
