//go:build headless

package main

import (
	"fmt"
	"os"

	"github.com/jdbnet/icetray/config"
	"github.com/jdbnet/icetray/logger"
	"github.com/jdbnet/icetray/player"
	"github.com/jdbnet/icetray/startup"
	"github.com/jdbnet/icetray/stream"
)

// runHeaded fails with an error message because headless builds do not package the GUI tray.
func runHeaded(cfg *config.Config, p *player.Player, sup *stream.Supervisor, sm startup.StartupManager) {
	fmt.Fprintln(os.Stderr, "Error: This binary is compiled in headless mode and does not support the system tray UI.")
	fmt.Fprintln(os.Stderr, "Use --stream <URL> to play audio from the command line.")
	logger.Log("Headless binary tried to run in headed mode, exiting")
	os.Exit(1)
}
