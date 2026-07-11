package launch

import (
	"runtime"
	"strings"
	"testing"
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
	exit, tail, err := Run("sh", []string{"-c", "echo boom >&2; exit 3"}, t.TempDir())
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
	_, _, err := Run("definitely-not-a-real-binary-xyz", nil, t.TempDir())
	if err == nil {
		t.Fatal("want error for missing binary")
	}
}
