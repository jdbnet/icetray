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
)

// PlayerController is implemented by the Wails app for tray actions.
type PlayerController interface {
	ShowPlayer()
	TrayPlay()
	TrayPause()
	TrayStop()
	QuitApp()
}

// TrayManager manages the system tray UI and events.
type TrayManager struct {
	cfg        *config.Config
	player     *player.Player
	supervisor *stream.Supervisor
	startupMgr startup.StartupManager
	app        PlayerController
	playbackMu sync.Mutex

	openItem        *systray.MenuItem
	playItem        *systray.MenuItem
	pauseItem       *systray.MenuItem
	stopItem        *systray.MenuItem
	settingsItem    *systray.MenuItem
	autoplayItem    *systray.MenuItem
	launchLoginItem *systray.MenuItem
	quitItem        *systray.MenuItem

	isPlaying bool
	quitOnce  sync.Once
	done      chan struct{}
}

// NewTrayManager creates a new tray manager.
func NewTrayManager(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager, app PlayerController) *TrayManager {
	return &TrayManager{
		cfg:        cfg,
		player:     p,
		supervisor: sup,
		startupMgr: sm,
		app:        app,
		isPlaying:  p.IsPlaying(),
		done:       make(chan struct{}),
	}
}

// Start registers or runs the system tray.
func (tm *TrayManager) Start() {
	tm.player.AddStateChangeListener(tm.syncPlaybackState)
	if runtime.GOOS == "linux" {
		// GTK/Wails must own the main thread; systray.Register hooks into that loop.
		systray.Register(tm.onReady, tm.onExit)
		return
	}
	go systray.Run(tm.onReady, tm.onExit)
}

// Wait blocks until the tray exits (Windows).
func (tm *TrayManager) Wait() {
	<-tm.done
}

// Quit stops the tray icon.
func (tm *TrayManager) Quit() {
	systray.Quit()
}

func (tm *TrayManager) syncPlaybackState() {
	tm.playbackMu.Lock()
	tm.isPlaying = tm.player.IsPlaying()
	tm.playbackMu.Unlock()
	tm.updateMenuState()
}

func (tm *TrayManager) onReady() {
	if runtime.GOOS == "windows" {
		systray.SetIcon(assets.IconICO)
	} else {
		systray.SetTemplateIcon(assets.Icon, assets.Icon)
	}
	systray.SetTitle("")
	systray.SetTooltip("IceTray - Internet Radio Player")

	tm.openItem = systray.AddMenuItem("Open Player", "Show the IceTray player window")
	systray.AddSeparator()

	tm.playItem = systray.AddMenuItem("Play", "Play current stream")
	tm.pauseItem = systray.AddMenuItem("Pause", "Pause playback")
	tm.pauseItem.Hide()
	tm.stopItem = systray.AddMenuItem("Stop", "Stop playback")
	tm.stopItem.Hide()

	systray.AddSeparator()

	tm.settingsItem = systray.AddMenuItem("Settings", "Settings")
	tm.autoplayItem = tm.settingsItem.AddSubMenuItem("Autoplay on Startup", "Toggle autoplay")
	if tm.cfg.GetAutoplay() {
		tm.autoplayItem.SetTitle("Autoplay on Startup (on)")
	} else {
		tm.autoplayItem.SetTitle("Autoplay on Startup (off)")
	}

	tm.launchLoginItem = tm.settingsItem.AddSubMenuItem("Launch on Login", "Toggle launch on login")
	if tm.startupMgr.IsEnabled() {
		tm.launchLoginItem.SetTitle("Launch on Login (on)")
	} else {
		tm.launchLoginItem.SetTitle("Launch on Login (off)")
	}

	systray.AddSeparator()
	tm.quitItem = systray.AddMenuItem("Quit", "Quit IceTray")

	tm.updateMenuState()
	go tm.eventLoop()
}

func (tm *TrayManager) onExit() {
	tm.quitOnce.Do(func() {
		close(tm.done)
	})
	logger.Log("Tray exited")
}

func (tm *TrayManager) eventLoop() {
	for {
		select {
		case <-tm.openItem.ClickedCh:
			tm.app.ShowPlayer()

		case <-tm.playItem.ClickedCh:
			tm.handlePlay()

		case <-tm.pauseItem.ClickedCh:
			tm.handlePause()

		case <-tm.stopItem.ClickedCh:
			tm.handleStop()

		case <-tm.autoplayItem.ClickedCh:
			tm.handleToggleAutoplay()

		case <-tm.launchLoginItem.ClickedCh:
			tm.handleToggleLaunchOnLogin()

		case <-tm.quitItem.ClickedCh:
			tm.app.QuitApp()
			systray.Quit()
			return
		}
	}
}

func (tm *TrayManager) handlePlay() {
	tm.playbackMu.Lock()
	defer tm.playbackMu.Unlock()
	tm.app.TrayPlay()
	tm.isPlaying = tm.player.IsPlaying()
	tm.updateMenuState()
}

func (tm *TrayManager) handlePause() {
	tm.playbackMu.Lock()
	defer tm.playbackMu.Unlock()
	tm.app.TrayPause()
	tm.isPlaying = tm.player.IsPlaying()
	tm.updateMenuState()
}

func (tm *TrayManager) handleStop() {
	tm.playbackMu.Lock()
	defer tm.playbackMu.Unlock()
	tm.app.TrayStop()
	tm.isPlaying = false
	tm.updateMenuState()
}

func (tm *TrayManager) handleToggleAutoplay() {
	newAutoplay := !tm.cfg.GetAutoplay()
	tm.cfg.SetAutoplay(newAutoplay)
	if newAutoplay {
		tm.autoplayItem.SetTitle("Autoplay on Startup (on)")
	} else {
		tm.autoplayItem.SetTitle("Autoplay on Startup (off)")
	}
	logger.Log(fmt.Sprintf("Autoplay: %v", newAutoplay))
}

func (tm *TrayManager) handleToggleLaunchOnLogin() {
	enabled := tm.startupMgr.IsEnabled()
	var err error
	if enabled {
		err = tm.startupMgr.Disable()
		tm.launchLoginItem.SetTitle("Launch on Login (off)")
	} else {
		err = tm.startupMgr.Enable()
		tm.launchLoginItem.SetTitle("Launch on Login (on)")
	}
	if err != nil {
		logger.LogError("Launch on Login: failed to update", err)
	}
	tm.cfg.SetLaunchOnLogin(!enabled)
}

func (tm *TrayManager) updateMenuState() {
	if tm.isPlaying {
		tm.playItem.Hide()
		tm.pauseItem.Show()
		tm.stopItem.Show()
	} else if tm.player.IsRunning() {
		tm.playItem.Show()
		tm.pauseItem.Hide()
		tm.stopItem.Show()
	} else {
		tm.playItem.Show()
		tm.pauseItem.Hide()
		tm.stopItem.Hide()
	}
}
