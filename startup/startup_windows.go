//go:build windows

package startup

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

type WindowsStartupManager struct{}

const (
	legacyRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName            = "IceTray"
	startupShortcut    = "IceTray.lnk"
)

func getPlatformStartupManager() StartupManager {
	return &WindowsStartupManager{}
}

func startupShortcutPath() string {
	return filepath.Join(
		os.Getenv("APPDATA"),
		"Microsoft", "Windows", "Start Menu", "Programs", "Startup",
		startupShortcut,
	)
}

func removeLegacyRunKey() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, legacyRegistryPath, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer key.Close()

	err = key.DeleteValue(appName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

func (w *WindowsStartupManager) Enable() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return err
	}

	// Registry Run keys are a common malware persistence vector and trigger Defender
	// heuristics (Behavior:Win32/Persistence.A!ml). Use a Startup folder shortcut instead.
	if err := removeLegacyRunKey(); err != nil {
		return err
	}

	return createShortcut(startupShortcutPath(), exePath, filepath.Dir(exePath))
}

func (w *WindowsStartupManager) Disable() error {
	if err := removeLegacyRunKey(); err != nil {
		return err
	}

	err := os.Remove(startupShortcutPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (w *WindowsStartupManager) IsEnabled() bool {
	if _, err := os.Stat(startupShortcutPath()); err == nil {
		return true
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, legacyRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appName)
	return err == nil
}
