package launch

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTailKeepsLastBytes(t *testing.T) {
	tail := NewTail(8)
	tail.Write([]byte("0123456789ABCDEF")) // 16 bytes into an 8-byte tail
	if got := tail.String(); got != "89ABCDEF" {
		t.Errorf("tail = %q, want %q", got, "89ABCDEF")
	}
}

func TestTailAcrossWrites(t *testing.T) {
	tail := NewTail(4)
	tail.Write([]byte("abc"))
	tail.Write([]byte("def"))
	if got := tail.String(); got != "cdef" {
		t.Errorf("tail = %q, want %q", got, "cdef")
	}
}

func TestRunCapturesExitAndStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	exit, _, tail, err := Run("sh", []string{"-c", "echo boom >&2; exit 3"}, t.TempDir(), Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if exit != 3 {
		t.Errorf("exit = %d, want 3", exit)
	}
	if !strings.Contains(tail, "boom") {
		t.Errorf("tail = %q, want it to contain boom", tail)
	}
}

func TestRunMissingBinary(t *testing.T) {
	_, _, _, err := Run("definitely-not-a-real-binary-xyz", nil, t.TempDir(), Opts{})
	if err == nil {
		t.Fatal("want error for missing binary")
	}
}

func TestRunTimeoutKillsChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	start := time.Now()
	exit, _, _, err := Run("sh", []string{"-c", "sleep 5"}, t.TempDir(), Opts{Timeout: 300 * time.Millisecond})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
	if exit != 124 {
		t.Errorf("exit = %d, want 124", exit)
	}
	if time.Since(start) > 3*time.Second {
		t.Error("child was not killed promptly")
	}
}

func TestRunStdinOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	exit, out, _, err := Run("sh", []string{"-c", "cat"}, t.TempDir(),
		Opts{Stdin: strings.NewReader("replayed"), CaptureStdout: true})
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if out != "replayed" {
		t.Errorf("stdout = %q, want %q", out, "replayed")
	}
}

func TestRunCaptureStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	_, out, tail, err := Run("sh", []string{"-c", "echo OUT; echo ERR >&2"}, t.TempDir(), Opts{CaptureStdout: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "OUT" {
		t.Errorf("stdout = %q", out)
	}
	if !strings.Contains(tail, "ERR") {
		t.Errorf("stderr tail = %q", tail)
	}
}
