//go:build windows
// +build windows

package startup

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

type WindowsStartupManager struct{}

const registryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const appName = "IceTray"

func getPlatformStartupManager() StartupManager {
	return &WindowsStartupManager{}
}

func (w *WindowsStartupManager) Enable() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.SetStringValue(appName, exePath)
}

func (w *WindowsStartupManager) Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.DeleteValue(appName)
}

func (w *WindowsStartupManager) IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.READ)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appName)
	return err == nil
}
