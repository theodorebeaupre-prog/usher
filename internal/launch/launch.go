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

	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return 124, outBuf.String(), tail.String(), ErrTimeout
	}
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return -1, outBuf.String(), tail.String(), err // failed to start
	}
	return cmd.ProcessState.ExitCode(), outBuf.String(), tail.String(), nil
}
