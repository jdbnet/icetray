//go:build !windows && !(linux && !android)

package startup

// Platforms without a login-startup implementation (Android, iOS, macOS).
type noopStartupManager struct{}

func getPlatformStartupManager() StartupManager {
	return noopStartupManager{}
}

func (noopStartupManager) Enable() error   { return nil }
func (noopStartupManager) Disable() error  { return nil }
func (noopStartupManager) IsEnabled() bool { return false }
