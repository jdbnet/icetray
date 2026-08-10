//go:build linux && !headless

package main

import (
	"fmt"
	"os"

	"git.jdbnet.co.uk/jamie/icetray/ui"
)

func runAddStreamDialogIfRequested(resultPath, name, url, errMsg string) bool {
	if resultPath == "" {
		return false
	}

	if err := ui.RunDialogMode(resultPath, ui.AddStreamInput{
		Name:  name,
		URL:   url,
		Error: errMsg,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Add stream dialog failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
	return true
}
