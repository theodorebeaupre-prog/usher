// Package launch runs an agent CLI as a child that owns the terminal.
package launch

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
)

const tailSize = 4096

// Run starts the agent and waits for it to exit. stdin and stdout are the
// real terminal files so the agent's TUI works untouched; stderr is teed
// into a small ring buffer so we can recognize quota errors after exit.
// While the child runs, the parent ignores SIGINT — Ctrl-C belongs to the
// agent's session, not to usher.
func Run(bin string, args []string, dir string) (int, string, error) {
	path, err := exec.LookPath(bin)
	if err != nil {
		return -1, "", err
	}
	tail := NewTail(tailSize)
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, tail)

	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	err = cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return -1, tail.String(), err // failed to start
	}
	return cmd.ProcessState.ExitCode(), tail.String(), nil
}
