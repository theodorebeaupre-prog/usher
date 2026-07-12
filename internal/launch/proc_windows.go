//go:build windows

package launch

import "os/exec"

// procGroup is a no-op on Windows; WaitDelay alone unblocks Wait.
func procGroup(cmd *exec.Cmd) {}
