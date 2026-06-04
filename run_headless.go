//go:build headless

package main

import (
	"fmt"
	"os"

	"git.jdbnet.co.uk/jamie/icetray/config"
	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
	"git.jdbnet.co.uk/jamie/icetray/startup"
	"git.jdbnet.co.uk/jamie/icetray/stream"
)

// runHeaded fails with an error message because headless builds do not package the GUI tray.
func runHeaded(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager) {
	fmt.Fprintln(os.Stderr, "Error: This binary is compiled in headless mode and does not support the system tray UI.")
	fmt.Fprintln(os.Stderr, "Use --stream <URL> to play audio from the command line.")
	logger.Log("Headless binary tried to run in headed mode, exiting")
	os.Exit(1)
}
