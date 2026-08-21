//go:build !headless

package main

import (
	"context"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"git.jdbnet.co.uk/jamie/icetray/assets"
	"git.jdbnet.co.uk/jamie/icetray/config"
	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
	"git.jdbnet.co.uk/jamie/icetray/startup"
	"git.jdbnet.co.uk/jamie/icetray/stream"
	"git.jdbnet.co.uk/jamie/icetray/tray"
)

func init() {
	// WebKitGTK hardware compositing can crash on Wayland with some Nvidia/Mesa drivers.
	_ = os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
}

// runHeaded initializes the Wails player UI and system tray.
func runHeaded(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager) {
	app := NewApp(cfg, p, sup, sm)
	trayMgr := tray.NewTrayManager(cfg, p, sup, sm, app)
	trayMgr.Start()

	background := options.RGBA{R: 18, G: 18, B: 20, A: 255}

	err := wails.Run(&options.App{
		Title:             "IceTray",
		Width:             1100,
		Height:            720,
		MinWidth:          800,
		MinHeight:         560,
		HideWindowOnClose: true,
		BackgroundColour:  &background,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		OnStartup: app.startup,
		OnShutdown: func(ctx context.Context) {
			app.Shutdown()
		},
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon: assets.Icon,
		},
	})
	if err != nil {
		logger.LogFatal("Wails application failed", err)
	}

	if runtime.GOOS == "windows" {
		trayMgr.Wait()
	}
}
