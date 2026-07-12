// Package launch runs an agent CLI as a child that owns the terminal.
package launch

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
)

const tailSize = 4096

// ErrTimeout is returned when the child is killed for exceeding Opts.Timeout.
var ErrTimeout = errors.New("agent timed out")

// Opts configures optional Run behavior. The zero value reproduces v0.2
// behavior exactly: no timeout, stdin from the real terminal, stdout
// streamed straight through (not captured).
type Opts struct {
	Timeout       time.Duration // 0 = none; child killed on expiry
	Stdin         io.Reader     // nil = os.Stdin
	CaptureStdout bool          // true = stdout returned, not streamed
}

// Run starts the agent and waits for it to exit. By default stdin and
// stdout are the real terminal files so the agent's TUI works untouched;
// stderr is teed into a small ring buffer so we can recognize quota errors
// after exit. While the child runs, the parent ignores SIGINT — Ctrl-C
// belongs to the agent's session, not to usher.
//
// Opts.Timeout, if set, kills the child when it expires; Run then returns
// (124, capturedSoFar, tail, ErrTimeout). Opts.Stdin overrides the child's
// stdin. Opts.CaptureStdout redirects stdout into a buffer that is returned
// instead of being streamed to the real terminal.
func Run(bin string, args []string, dir string, opts Opts) (int, string, string, error) {
	path, err := exec.LookPath(bin)
	if err != nil {
		return -1, "", "", err
	}

	ctx := context.Background()
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	tail := NewTail(tailSize)
	var outBuf strings.Builder
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = dir

	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	} else {
		cmd.Stdin = os.Stdin
	}

	if opts.CaptureStdout {
		cmd.Stdout = &outBuf
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = io.MultiWriter(os.Stderr, tail)

	// WaitDelay bounds how long Wait blocks after the context is canceled:
	// once ctx expires, Cmd abandons any pipes still open (e.g. held by an
	// orphaned grandchild) and Wait returns instead of hanging forever.
	cmd.WaitDelay = 2 * time.Second

	// Only put the child in its own process group when a timeout is in
	// play. Setpgid moves the child out of the terminal's foreground
	// process group, so in interactive mode (Timeout == 0) it would break
	// Ctrl-C delivery and cause SIGTTIN on any TTY read from a background
	// group. Timeout is headless-only, so there's no foreground-group
	// concern there — and killing the whole group is what lets a hung
	// agent's grandchildren (e.g. a stray `sleep`) die with it instead of
	// leaking and holding the stderr pipe open.
	if opts.Timeout > 0 {
		procGroup(cmd)
	}

	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	err = cmd.Run()
	// This must be checked before any error-type inspection below: once the
	// context deadline fires, cmd.Run's error could be exec.ErrWaitDelay (if
	// Wait had to abandon pipes) or an *exec.ExitError from the kill signal.
	// Either way a timeout is a timeout, not the generic error path.
	if ctx.Err() == context.DeadlineExceeded {
		return 124, outBuf.String(), tail.String(), ErrTimeout
	}
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return -1, outBuf.String(), tail.String(), err // failed to start
	}
	return cmd.ProcessState.ExitCode(), outBuf.String(), tail.String(), nil
}
