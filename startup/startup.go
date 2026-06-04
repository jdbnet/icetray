package startup

// StartupManager defines the interface for platform-specific startup registration.
type StartupManager interface {
	// Enable registers the app to launch on login.
	Enable() error

	// Disable unregisters the app from launching on login.
	Disable() error

	// IsEnabled checks if the app is registered to launch on login.
	IsEnabled() bool
}

// New returns a platform-specific StartupManager.
func New() StartupManager {
	return getPlatformStartupManager()
}
