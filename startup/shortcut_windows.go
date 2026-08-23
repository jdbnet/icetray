//go:build windows

package startup

import (
	"fmt"
	"os/exec"
	"strings"
)

func createShortcut(shortcutPath, targetPath, workingDir string) error {
	ps := fmt.Sprintf(
		"$WshShell = New-Object -ComObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%s'); $Shortcut.TargetPath = '%s'; $Shortcut.WorkingDirectory = '%s'; $Shortcut.Save()",
		escapePowerShellSingleQuoted(shortcutPath),
		escapePowerShellSingleQuoted(targetPath),
		escapePowerShellSingleQuoted(workingDir),
	)
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
