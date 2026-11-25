//go:build !windows

package scripts

import "os/exec"

// launcherCommandWindows is only reached on Windows; launcherCommandShell
// checks runtime.GOOS first. It fails loudly if that ever changes.
func launcherCommandWindows(launcher string, args ...string) *exec.Cmd {
	panic("launcherCommandWindows called on a non-Windows platform")
}
