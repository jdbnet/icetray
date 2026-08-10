//go:build linux && !headless

package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.jdbnet.co.uk/jamie/icetray/logger"
)

func showAddStreamDialog(initial AddStreamInput) (name, url string, ok bool) {
	exe, err := os.Executable()
	if err != nil {
		logger.LogError("Add stream dialog: could not resolve executable", err)
		return "", "", false
	}

	resultFile, err := os.CreateTemp("", "icetray-add-stream-*.json")
	if err != nil {
		logger.LogError("Add stream dialog: could not create result file", err)
		return "", "", false
	}
	resultPath := resultFile.Name()
	resultFile.Close()

	args := []string{
		"--add-stream-result", resultPath,
	}
	if initial.Name != "" {
		args = append(args, "--add-stream-name", initial.Name)
	}
	if initial.URL != "" {
		args = append(args, "--add-stream-url", initial.URL)
	}
	if initial.Error != "" {
		args = append(args, "--add-stream-error", initial.Error)
	}

	cmd := exec.Command(exe, args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.LogError(
			fmt.Sprintf("Add stream dialog subprocess failed: %s", strings.TrimSpace(string(output))),
			err,
		)
		os.Remove(resultPath)
		return "", "", false
	}

	name, url, ok, err = readDialogResult(resultPath)
	os.Remove(resultPath)
	if err != nil {
		logger.LogError("Add stream dialog: could not read result file", err)
		return "", "", false
	}

	return name, url, ok
}

// RunDialogMode shows the add stream dialog on the main thread and writes the result.
// This is invoked by re-executing the IceTray binary with internal CLI flags.
func RunDialogMode(resultPath string, initial AddStreamInput) error {
	name, url, ok := runAddStreamDialogMain(initial)
	return writeDialogResult(resultPath, name, url, ok)
}
