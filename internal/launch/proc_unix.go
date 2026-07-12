//go:build !windows

package launch

import (
	"os/exec"
	"syscall"
)

// procGroup puts the child in its own process group and makes context
// cancellation kill the whole group, so a hung agent's helpers die too.
func procGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
