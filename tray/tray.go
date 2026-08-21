package tray

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/getlantern/systray"

	"git.jdbnet.co.uk/jamie/icetray/assets"
	"git.jdbnet.co.uk/jamie/icetray/config"
	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
	"git.jdbnet.co.uk/jamie/icetray/startup"
	"git.jdbnet.co.uk/jamie/icetray/stream"
	"git.jdbnet.co.uk/jamie/icetray/ui"
)

const streamMenuPoolSize = 32

// TrayManager manages the system tray UI and events.
type TrayManager struct {
	cfg        *config.Config
	player     *player.Player
	supervisor *stream.Supervisor
	startupMgr startup.StartupManager
	playbackMu sync.Mutex
	menuMu     sync.Mutex

	// Menu items
	playItem        *systray.MenuItem
	pauseItem       *systray.MenuItem
	stopItem        *systray.MenuItem
	streamsMenu     *systray.MenuItem
	addStreamItem   *systray.MenuItem
	volumeMenu      *systray.MenuItem
	volumeItems     map[int]*systray.MenuItem
	settingsItem    *systray.MenuItem
	autoplayItem    *systray.MenuItem
	launchLoginItem *systray.MenuItem
	quitItem        *systray.MenuItem

	streamMenuPool  []*systray.MenuItem
	streamURLs      []string
	streamEmptyItem *systray.MenuItem

	isPlaying bool
}

// NewTrayManager creates a new tray manager.
func NewTrayManager(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager) *TrayManager {
	return &TrayManager{
		cfg:         cfg,
		player:      p,
		supervisor:  sup,
		startupMgr:  sm,
		isPlaying:   p.IsRunning(),
		volumeItems: make(map[int]*systray.MenuItem),
	}
}

// Init initializes the system tray.
func (tm *TrayManager) Init() {
	systray.Run(tm.onReady, tm.onExit)
}

// onReady is called when the system tray is ready.
func (tm *TrayManager) onReady() {
	if runtime.GOOS == "windows" {
		systray.SetIcon(assets.IconICO)
	} else {
		systray.SetTemplateIcon(assets.Icon, assets.Icon)
	}
	systray.SetTitle("")
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
	tm.initStreamMenuPool()
	tm.rebuildStreamsMenu()

	// Add Stream item
	tm.addStreamItem = systray.AddMenuItem("➕ Add Stream", "Add a new stream")

	systray.AddSeparator()

	// Volume controls
	tm.volumeMenu = systray.AddMenuItem("🔊 Volume", "Set player volume")
	tm.initVolumeMenu()

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

	// Update initial menu state based on player running state
	tm.updateMenuState()

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

		case <-tm.autoplayItem.ClickedCh:
			tm.handleToggleAutoplay()

		case <-tm.launchLoginItem.ClickedCh:
			tm.handleToggleLaunchOnLogin()

		case <-tm.quitItem.ClickedCh:
			systray.Quit()
			return
		}
	}
}

// handlePlay starts playing the last stream or prompts to select one.
func (tm *TrayManager) handlePlay() {
	tm.playbackMu.Lock()
	defer tm.playbackMu.Unlock()

	lastStream := tm.cfg.GetLastStream()
	if lastStream == "" {
		logger.Log("Play: no stream selected")
		return
	}

	if tm.player.IsRunning() {
		tm.player.Resume()
		tm.isPlaying = true
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
	tm.playbackMu.Lock()
	defer tm.playbackMu.Unlock()

	if err := tm.player.Pause(); err != nil {
		logger.LogError("Pause: failed", err)
		return
	}
	tm.isPlaying = false
	tm.updateMenuState()
}

// handleStop stops playback.
func (tm *TrayManager) handleStop() {
	tm.playbackMu.Lock()
	defer tm.playbackMu.Unlock()

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
	input := ui.AddStreamInput{}

	for {
		name, url, ok := ui.ShowAddStreamDialog(input)
		if !ok {
			return
		}

		if err := tm.cfg.AddStream(name, url); err != nil {
			logger.LogError("Failed to save new stream", err)
			input = ui.AddStreamInput{
				Name:  name,
				URL:   url,
				Error: "Failed to save the stream. Please try again.",
			}
			continue
		}

		tm.rebuildStreamsMenu()
		logger.Log(fmt.Sprintf("Added new stream: %s (%s)", name, url))
		return
	}
}

// initVolumeMenu populates the volume submenu with presets.
func (tm *TrayManager) initVolumeMenu() {
	presets := []int{100, 90, 80, 70, 60, 50, 40, 30, 20, 10, 0}
	currentVol := tm.cfg.GetVolume()

	for _, vol := range presets {
		label := fmt.Sprintf("%d%%", vol)
		if vol == 0 {
			label = "Mute (0%)"
		}
		
		item := tm.volumeMenu.AddSubMenuItemCheckbox(label, "", vol == currentVol)
		tm.volumeItems[vol] = item

		v := vol
		go func(menuItem *systray.MenuItem, level int) {
			for range menuItem.ClickedCh {
				tm.handleVolumeChange(level)
			}
		}(item, v)
	}
}

// handleVolumeChange sets the new volume and updates the menu checkmarks.
func (tm *TrayManager) handleVolumeChange(newVol int) {
	tm.cfg.SetVolume(newVol)
	tm.player.SetVolume(newVol)

	for vol, item := range tm.volumeItems {
		if vol == newVol {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
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

// initStreamMenuPool pre-creates hidden stream menu slots. AppIndicator on Linux
// does not reliably show submenu items added after startup, but updating existing
// items via SetTitle/Show works.
func (tm *TrayManager) initStreamMenuPool() {
	tm.streamMenuPool = make([]*systray.MenuItem, streamMenuPoolSize)
	tm.streamURLs = make([]string, streamMenuPoolSize)

	for i := 0; i < streamMenuPoolSize; i++ {
		item := tm.streamsMenu.AddSubMenuItem("", "")
		item.Hide()
		tm.streamMenuPool[i] = item

		idx := i
		go func(menuItem *systray.MenuItem) {
			for range menuItem.ClickedCh {
				tm.menuMu.Lock()
				streamURL := tm.streamURLs[idx]
				tm.menuMu.Unlock()
				if streamURL != "" {
					tm.handleStreamSelected(streamURL)
				}
			}
		}(item)
	}
}

// rebuildStreamsMenu syncs the streams submenu with the saved config.
func (tm *TrayManager) rebuildStreamsMenu() {
	tm.menuMu.Lock()
	defer tm.menuMu.Unlock()

	streams := tm.cfg.GetStreams()
	for i := range tm.streamURLs {
		tm.streamURLs[i] = ""
	}

	for i, stream := range streams {
		if i >= streamMenuPoolSize {
			logger.Log(fmt.Sprintf("Stream menu full, could not show %q", stream.Name))
			continue
		}

		item := tm.streamMenuPool[i]
		tm.streamURLs[i] = stream.URL
		item.SetTitle(stream.Name)
		item.SetTooltip(stream.URL)
		item.Show()
	}

	for i := len(streams); i < streamMenuPoolSize; i++ {
		tm.streamMenuPool[i].Hide()
	}

	if len(streams) == 0 {
		if tm.streamEmptyItem == nil {
			tm.streamEmptyItem = tm.streamsMenu.AddSubMenuItem("(No streams saved)", "Add streams via Add Stream menu")
		} else {
			tm.streamEmptyItem.Show()
		}
	} else if tm.streamEmptyItem != nil {
		tm.streamEmptyItem.Hide()
	}
}

// handleStreamSelected handles selection of a stream from the menu.
func (tm *TrayManager) handleStreamSelected(streamURL string) {
	tm.playbackMu.Lock()
	defer tm.playbackMu.Unlock()

	// Stop current playback first
	tm.player.Stop()
	tm.supervisor.Stop()

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
