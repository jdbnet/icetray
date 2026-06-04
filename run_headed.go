//go:build !headless

package main

import (
	"git.jdbnet.co.uk/jamie/icetray/config"
	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
	"git.jdbnet.co.uk/jamie/icetray/startup"
	"git.jdbnet.co.uk/jamie/icetray/stream"
	"git.jdbnet.co.uk/jamie/icetray/tray"
)

// runHeaded initializes and runs the graphical system tray interface.
func runHeaded(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager) {
	logger.Log("Initializing system tray")
	trayMgr := tray.NewTrayManager(cfg, p, sup, sm)
	trayMgr.Init() // This is a blocking call
}
