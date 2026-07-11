package banner

import (
	"strings"
	"testing"
)

func TestStaticPrintsFinalFrame(t *testing.T) {
	var sb strings.Builder
	if err := Play(&sb, false); err != nil {
		t.Fatal(err)
	}
	out := sb.String()
	if !strings.Contains(out, "█") {
		t.Error("static banner should contain block characters")
	}
	if !strings.Contains(out, "right this way.") {
		t.Error("static banner should contain the tagline")
	}
	if strings.Contains(out, "\x1b[?25l") {
		t.Error("static banner must not hide the cursor")
	}
}

func TestBatchedANSIRuns(t *testing.T) {
	// Two adjacent cells with the same role must share one SGR sequence.
	line := renderLine("██", map[int]string{0: "letter_lit", 1: "letter_lit"})
	if strings.Count(line, "\x1b[37m") != 1 {
		t.Errorf("want exactly one SGR for a same-color run, got %q", line)
	}
}

func TestShouldAnimateGates(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if ShouldAnimate(false) {
		t.Error("NO_COLOR must disable animation")
	}
}
