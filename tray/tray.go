package tray

import (
	"fmt"

	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"

	"github.com/user/icetray/assets"
	"github.com/user/icetray/config"
	"github.com/user/icetray/logger"
	"github.com/user/icetray/player"
	"github.com/user/icetray/startup"
	"github.com/user/icetray/stream"
)

// TrayManager manages the system tray UI and events.
type TrayManager struct {
	cfg        *config.Config
	player     *player.Player
	supervisor *stream.Supervisor
	startupMgr startup.StartupManager

	// Menu items
	playItem        *systray.MenuItem
	pauseItem       *systray.MenuItem
	stopItem        *systray.MenuItem
	streamsMenu     *systray.MenuItem
	addStreamItem   *systray.MenuItem
	volumeUpItem    *systray.MenuItem
	volumeDownItem  *systray.MenuItem
	settingsItem    *systray.MenuItem
	autoplayItem    *systray.MenuItem
	launchLoginItem *systray.MenuItem
	quitItem        *systray.MenuItem

	isPlaying bool
}

// NewTrayManager creates a new tray manager.
func NewTrayManager(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager) *TrayManager {
	return &TrayManager{
		cfg:        cfg,
		player:     p,
		supervisor: sup,
		startupMgr: sm,
		isPlaying:  false,
	}
}

// Init initializes the system tray.
func (tm *TrayManager) Init() {
	systray.Run(tm.onReady, tm.onExit)
}

// onReady is called when the system tray is ready.
func (tm *TrayManager) onReady() {
	systray.SetTemplateIcon(assets.Icon, assets.Icon)
	systray.SetTitle("IceTray")
	systray.SetTooltip("IceTray - Internet Radio Player")

	// Play item
	tm.playItem = systray.AddMenuItem("▶ Play", "Play current stream")

	// Pause item
	tm.pauseItem = systray.AddMenuItem("⏸ Pause", "Pause playback")
	tm.pauseItem.Hide()

	// Stop item
	tm.stopItem = systray.AddMenuItem("⏹ Stop", "Stop playback")
	tm.stopItem.Hide()

	systray.AddSeparator()

	// Streams submenu
	tm.streamsMenu = systray.AddMenuItem("🎵 Streams", "Saved radio streams")
	tm.refreshStreamsMenu()

	// Add Stream item
	tm.addStreamItem = systray.AddMenuItem("➕ Add Stream", "Add a new stream")

	systray.AddSeparator()

	// Volume controls
	tm.volumeUpItem = systray.AddMenuItem("🔊 Volume Up", "Increase volume")
	tm.volumeDownItem = systray.AddMenuItem("🔉 Volume Down", "Decrease volume")

	systray.AddSeparator()

	// Settings
	tm.settingsItem = systray.AddMenuItem("⚙ Settings", "Settings")
	tm.autoplayItem = tm.settingsItem.AddSubMenuItem("Autoplay on Startup", "Toggle autoplay")
	if tm.cfg.GetAutoplay() {
		tm.autoplayItem.SetTitle("☑ Autoplay on Startup")
	} else {
		tm.autoplayItem.SetTitle("☐ Autoplay on Startup")
	}

	tm.launchLoginItem = tm.settingsItem.AddSubMenuItem("Launch on Login", "Toggle launch on login")
	if tm.startupMgr.IsEnabled() {
		tm.launchLoginItem.SetTitle("☑ Launch on Login")
	} else {
		tm.launchLoginItem.SetTitle("☐ Launch on Login")
	}

	systray.AddSeparator()

	// Quit item
	tm.quitItem = systray.AddMenuItem("❌ Quit", "Quit IceTray")

	// Start event loop
	go tm.eventLoop()
}

// onExit is called when systray is shutting down.
func (tm *TrayManager) onExit() {
	tm.player.Stop()
	tm.supervisor.Stop()
	logger.Log("Tray exited, application shutting down")
}

// eventLoop handles menu item clicks.
func (tm *TrayManager) eventLoop() {
	for {
		select {
		case <-tm.playItem.ClickedCh:
			tm.handlePlay()

		case <-tm.pauseItem.ClickedCh:
			tm.handlePause()

		case <-tm.stopItem.ClickedCh:
			tm.handleStop()

		case <-tm.addStreamItem.ClickedCh:
			tm.handleAddStream()

		case <-tm.volumeUpItem.ClickedCh:
			tm.handleVolumeUp()

		case <-tm.volumeDownItem.ClickedCh:
			tm.handleVolumeDown()

		case <-tm.autoplayItem.ClickedCh:
			tm.handleToggleAutoplay()

		case <-tm.launchLoginItem.ClickedCh:
			tm.handleToggleLaunchOnLogin()

		case <-tm.quitItem.ClickedCh:
			systray.Quit()
			return
		}

		// Handle stream submenu clicks
		tm.handleStreamMenuClicks()
	}
}

// handlePlay starts playing the last stream or prompts to select one.
func (tm *TrayManager) handlePlay() {
	lastStream := tm.cfg.GetLastStream()
	if lastStream == "" {
		logger.Log("Play: no stream selected")
		return
	}

	if tm.isPlaying {
		tm.player.Resume()
	} else {
		if err := tm.player.Play(lastStream); err != nil {
			logger.LogError("Play: failed to start playback", err)
			return
		}
		tm.supervisor.Start(lastStream)
		tm.isPlaying = true
	}

	tm.updateMenuState()
}

// handlePause pauses playback.
func (tm *TrayManager) handlePause() {
	if err := tm.player.Pause(); err != nil {
		logger.LogError("Pause: failed", err)
		return
	}
	tm.isPlaying = false
	tm.updateMenuState()
}

// handleStop stops playback.
func (tm *TrayManager) handleStop() {
	if err := tm.player.Stop(); err != nil {
		logger.LogError("Stop: failed", err)
		return
	}
	tm.supervisor.Stop()
	tm.isPlaying = false
	tm.updateMenuState()
}

// handleAddStream prompts the user to add a new stream.
func (tm *TrayManager) handleAddStream() {
	name, err := zenity.Entry("Enter stream name (e.g., Lofi Radio):", zenity.Title("Add Stream (1/2)"))
	if err != nil || name == "" {
		if err != zenity.ErrCanceled {
			logger.LogError("Failed to get stream name", err)
		}
		return
	}

	url, err := zenity.Entry("Enter stream URL:", zenity.Title("Add Stream (2/2)"))
	if err != nil || url == "" {
		if err != zenity.ErrCanceled {
			logger.LogError("Failed to get stream URL", err)
		}
		return
	}

	if err := tm.cfg.AddStream(name, url); err != nil {
		logger.LogError("Failed to save new stream", err)
		zenity.Error("Failed to save the new stream.", zenity.Title("Error"))
		return
	}

	// Add just the new stream to the menu dynamically
	item := tm.streamsMenu.AddSubMenuItem(name, url)
	go func(menuItem *systray.MenuItem, streamURL string) {
		for range menuItem.ClickedCh {
			tm.handleStreamSelected(streamURL)
		}
	}(item, url)

	logger.Log(fmt.Sprintf("Added new stream: %s (%s)", name, url))
}

// handleVolumeUp increases the volume.
func (tm *TrayManager) handleVolumeUp() {
	currentVol := tm.cfg.GetVolume()
	newVol := currentVol + 5
	if newVol > 100 {
		newVol = 100
	}
	tm.cfg.SetVolume(newVol)
	tm.player.SetVolume(newVol)
}

// handleVolumeDown decreases the volume.
func (tm *TrayManager) handleVolumeDown() {
	currentVol := tm.cfg.GetVolume()
	newVol := currentVol - 5
	if newVol < 0 {
		newVol = 0
	}
	tm.cfg.SetVolume(newVol)
	tm.player.SetVolume(newVol)
}

// handleToggleAutoplay toggles autoplay setting.
func (tm *TrayManager) handleToggleAutoplay() {
	newAutoplay := !tm.cfg.GetAutoplay()
	tm.cfg.SetAutoplay(newAutoplay)
	if newAutoplay {
		tm.autoplayItem.SetTitle("☑ Autoplay on Startup")
	} else {
		tm.autoplayItem.SetTitle("☐ Autoplay on Startup")
	}
	logger.Log(fmt.Sprintf("Autoplay: %v", newAutoplay))
}

// handleToggleLaunchOnLogin toggles launch on login setting.
func (tm *TrayManager) handleToggleLaunchOnLogin() {
	enabled := tm.startupMgr.IsEnabled()
	var err error
	if enabled {
		err = tm.startupMgr.Disable()
		tm.launchLoginItem.SetTitle("☐ Launch on Login")
	} else {
		err = tm.startupMgr.Enable()
		tm.launchLoginItem.SetTitle("☑ Launch on Login")
	}
	if err != nil {
		logger.LogError("Launch on Login: failed to update", err)
	}
	tm.cfg.SetLaunchOnLogin(!enabled)
}

// handleStreamMenuClicks handles clicks on stream menu items.
func (tm *TrayManager) handleStreamMenuClicks() {
	// Note: This is a simplified approach. In a real implementation,
	// you'd need to track each stream menu item's click channel individually.
	// For now, we refresh the streams menu periodically or on demand.
}

// refreshStreamsMenu updates the streams submenu.
func (tm *TrayManager) refreshStreamsMenu() {
	streams := tm.cfg.GetStreams()

	for _, stream := range streams {
		item := tm.streamsMenu.AddSubMenuItem(stream.Name, stream.URL)
		// Create a closure to capture the stream URL
		streamURL := stream.URL
		go func(menuItem *systray.MenuItem) {
			for range menuItem.ClickedCh {
				tm.handleStreamSelected(streamURL)
			}
		}(item)
	}

	if len(streams) == 0 {
		tm.streamsMenu.AddSubMenuItem("(No streams saved)", "Add streams via Add Stream menu")
	}
}

// handleStreamSelected handles selection of a stream from the menu.
func (tm *TrayManager) handleStreamSelected(streamURL string) {
	// Stop current playback if running
	if tm.isPlaying {
		tm.player.Stop()
		tm.supervisor.Stop()
	}

	// Save the selected stream and start playing
	tm.cfg.SetLastStream(streamURL)
	logger.Log("Stream Selected: " + streamURL)

	// Start playing the selected stream
	if err := tm.player.Play(streamURL); err != nil {
		logger.LogError("Failed to play stream", err)
		return
	}
	tm.supervisor.Start(streamURL)
	tm.isPlaying = true
	tm.updateMenuState()
}

// updateMenuState updates the visibility and state of menu items.
func (tm *TrayManager) updateMenuState() {
	if tm.isPlaying {
		tm.playItem.Hide()
		tm.pauseItem.Show()
		tm.stopItem.Show()
	} else {
		tm.playItem.Show()
		tm.pauseItem.Hide()
		tm.stopItem.Hide()
	}
}


