//go:build windows

package scripts

import (
	"os/exec"
	"syscall"
)

// launcherCommandWindows runs the launcher through cmd.exe. The command line is
// set verbatim with CmdLine, because exec's own quoting would re-escape the
// quotes that windowsCommandArgs wrote.
func launcherCommandWindows(launcher string, args ...string) *exec.Cmd {
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `cmd.exe /S /C "` + windowsCommandArgs(launcher, args...) + `"`,
	}
	return cmd
}
