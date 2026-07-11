package ledger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var t0 = time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "ledger.json")
	l := Load(path)
	l.RecordLaunch("claude", t0)
	l.RecordQuota("codex", t0)
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	l2 := Load(path)
	if len(l2.Events) != 2 {
		t.Fatalf("want 2 events after reload, got %d", len(l2.Events))
	}
}

func TestCorruptFileRecovers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	os.WriteFile(path, []byte("{not json"), 0o644)
	l := Load(path)
	if l.Warning == "" {
		t.Error("want a warning for corrupt ledger")
	}
	if len(l.Events) != 0 {
		t.Error("want empty events for corrupt ledger")
	}
	if err := l.Save(); err != nil { // must be able to overwrite the corrupt file
		t.Fatal(err)
	}
}

func TestPruneOldEvents(t *testing.T) {
	l := Load(filepath.Join(t.TempDir(), "ledger.json"))
	l.RecordLaunch("claude", t0.Add(-8*24*time.Hour))
	l.RecordLaunch("claude", t0) // this prunes the 8-day-old event
	if len(l.Events) != 1 {
		t.Fatalf("want old events pruned, got %d", len(l.Events))
	}
}

func TestConfidence(t *testing.T) {
	l := Load(filepath.Join(t.TempDir(), "ledger.json"))
	window := 5 * time.Hour

	if c := l.Confidence("claude", window, t0); c != 1.0 {
		t.Errorf("no events: want 1.0, got %f", c)
	}
	l.RecordQuota("claude", t0.Add(-time.Hour)) // cap hit 1h ago, 5h window
	c := l.Confidence("claude", window, t0)
	if c < 0.19 || c > 0.21 { // 1h/5h = 0.2
		t.Errorf("want ~0.2, got %f", c)
	}
	if c := l.Confidence("claude", window, t0.Add(5*time.Hour)); c != 1.0 {
		t.Errorf("window elapsed: want 1.0, got %f", c)
	}
	if c := l.Confidence("codex", window, t0); c != 1.0 {
		t.Errorf("other agent unaffected: want 1.0, got %f", c)
	}
}
