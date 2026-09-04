//go:build !headless

package main

import (
	"os"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

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
	svc := NewApp(cfg, p, sup, sm)

	wailsApp := application.New(application.Options{
		Name:        "IceTray",
		Description: "Internet radio player for Icecast streams",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontendAssets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	startHidden := false
	if runtime.GOOS == "windows" || runtime.GOOS == "linux" {
		startHidden = cfg.GetLaunchMinimized()
	}

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "IceTray",
		Width:            1100,
		Height:           720,
		MinWidth:         800,
		MinHeight:        560,
		Hidden:           startHidden,
		BackgroundColour: application.NewRGB(18, 18, 20),
		URL:              "/",
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if runtime.GOOS == "android" {
			return
		}
		window.Hide()
		e.Cancel()
	})

	svc.setWindow(window)

	if runtime.GOOS != "android" {
		trayMgr := tray.NewTrayManager(wailsApp, cfg, p, sm, svc)
		trayMgr.Start()
	}

	if err := wailsApp.Run(); err != nil {
		logger.LogFatal("Wails application failed", err)
	}
}
