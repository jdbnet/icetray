//go:build !headless

package tray

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"git.jdbnet.co.uk/jamie/icetray/assets"
	"git.jdbnet.co.uk/jamie/icetray/config"
	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
	"git.jdbnet.co.uk/jamie/icetray/startup"
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
	wails      *application.App
	cfg        *config.Config
	player     *player.Player
	startupMgr startup.StartupManager
	app        PlayerController
	systray    *application.SystemTray
	menu       *application.Menu

	playItem        *application.MenuItem
	pauseItem       *application.MenuItem
	stopItem        *application.MenuItem
	autoplayItem    *application.MenuItem
	launchLoginItem *application.MenuItem
}

// NewTrayManager creates a new tray manager.
func NewTrayManager(wailsApp *application.App, cfg *config.Config, p *player.Player, sm startup.StartupManager, app PlayerController) *TrayManager {
	return &TrayManager{
		wails:      wailsApp,
		cfg:        cfg,
		player:     p,
		startupMgr: sm,
		app:        app,
	}
}

// Start registers the system tray icon and menu.
func (tm *TrayManager) Start() {
	if runtime.GOOS == "android" {
		return
	}

	tm.systray = tm.wails.SystemTray.New()
	if runtime.GOOS == "windows" {
		tm.systray.SetIcon(assets.IconICO)
	} else {
		tm.systray.SetIcon(assets.Icon)
	}
	tm.systray.SetTooltip("IceTray - Internet Radio Player")
	tm.systray.OnClick(func() {
		tm.app.ShowPlayer()
	})

	tm.rebuildMenu()
	tm.player.AddStateChangeListener(func() {
		tm.rebuildMenu()
	})
}

func (tm *TrayManager) rebuildMenu() {
	if tm.wails == nil || tm.systray == nil {
		return
	}

	menu := tm.wails.NewMenu()
	menu.Add("Open Player").OnClick(func(_ *application.Context) {
		tm.app.ShowPlayer()
	})
	menu.AddSeparator()

	tm.playItem = menu.Add("Play").OnClick(func(_ *application.Context) {
		tm.app.TrayPlay()
	})
	tm.pauseItem = menu.Add("Pause").OnClick(func(_ *application.Context) {
		tm.app.TrayPause()
	})
	tm.stopItem = menu.Add("Stop").OnClick(func(_ *application.Context) {
		tm.app.TrayStop()
	})

	playing := tm.player.IsPlaying()
	running := tm.player.IsRunning()
	tm.playItem.SetHidden(playing)
	tm.pauseItem.SetHidden(!playing)
	tm.stopItem.SetHidden(!playing && !running)

	menu.AddSeparator()
	settings := menu.AddSubmenu("Settings")
	tm.autoplayItem = settings.AddCheckbox("Autoplay on Startup", tm.cfg.GetAutoplay()).OnClick(func(ctx *application.Context) {
		enabled := ctx.ClickedMenuItem().Checked()
		_ = tm.cfg.SetAutoplay(enabled)
	})
	tm.launchLoginItem = settings.AddCheckbox("Launch on Login", tm.cfg.GetLaunchOnLogin()).OnClick(func(ctx *application.Context) {
		enabled := ctx.ClickedMenuItem().Checked()
		if enabled {
			if err := tm.startupMgr.Enable(); err != nil {
				logger.LogError("Launch on Login: failed to enable", err)
				ctx.ClickedMenuItem().SetChecked(false)
				return
			}
		} else if err := tm.startupMgr.Disable(); err != nil {
			logger.LogError("Launch on Login: failed to disable", err)
			ctx.ClickedMenuItem().SetChecked(true)
			return
		}
		_ = tm.cfg.SetLaunchOnLogin(enabled)
	})

	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		tm.app.QuitApp()
	})

	tm.menu = menu
	tm.systray.SetMenu(menu)
}
