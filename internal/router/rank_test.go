package router

import (
	"testing"

	"github.com/theodorebeaupre-prog/usher/internal/config"
)

func agents() []AgentInfo {
	return []AgentInfo{
		{Name: "claude", Strengths: map[string]float64{"debug": 0.9, "review": 0.85}, Quota: 1.0},
		{Name: "codex", Strengths: map[string]float64{"debug": 0.85, "review": 0.95}, Quota: 1.0},
	}
}

func TestRankByStrength(t *testing.T) {
	got := Rank("fix the crash", "/tmp", agents(), config.Config{})
	if got[0].Name != "claude" || got[0].TaskType != "debug" {
		t.Fatalf("want claude first for debug, got %+v", got)
	}
	if got[0].Score <= got[1].Score {
		t.Fatalf("scores not descending: %+v", got)
	}
}

func TestQuotaPenaltyFlipsRanking(t *testing.T) {
	a := agents()
	a[0].Quota = 0.2 // claude hit a cap recently
	got := Rank("fix the crash", "/tmp", a, config.Config{})
	if got[0].Name != "codex" {
		t.Fatalf("want codex first when claude is quota-limited, got %+v", got)
	}
}

func TestTypePinWins(t *testing.T) {
	cfg := config.Config{Pins: config.Pins{Types: map[string]string{"review": "claude"}}}
	got := Rank("review this PR", "/tmp", agents(), cfg)
	if got[0].Name != "claude" || !got[0].Pinned {
		t.Fatalf("want pinned claude first, got %+v", got)
	}
}

func TestPathPinWins(t *testing.T) {
	cfg := config.Config{Pins: config.Pins{Paths: map[string]string{"/repos/photocull": "codex"}}}
	got := Rank("fix the crash", "/repos/photocull/sub/dir", agents(), cfg)
	if got[0].Name != "codex" || !got[0].Pinned {
		t.Fatalf("want pinned codex first, got %+v", got)
	}
}

func TestWeightOverride(t *testing.T) {
	cfg := config.Config{Weights: map[string]map[string]float64{"codex": {"debug": 0.99}}}
	got := Rank("fix the crash", "/tmp", agents(), cfg)
	if got[0].Name != "codex" {
		t.Fatalf("want codex first after weight override, got %+v", got)
	}
}

func TestDisabledExcluded(t *testing.T) {
	cfg := config.Config{Disabled: []string{"claude"}}
	got := Rank("fix the crash", "/tmp", agents(), cfg)
	if len(got) != 1 || got[0].Name != "codex" {
		t.Fatalf("want only codex, got %+v", got)
	}
}

func TestDefaultAgentBreaksTies(t *testing.T) {
	a := []AgentInfo{
		{Name: "claude", Strengths: map[string]float64{"other": 0.8}, Quota: 1.0},
		{Name: "codex", Strengths: map[string]float64{"other": 0.8}, Quota: 1.0},
	}
	cfg := config.Config{DefaultAgent: "codex"}
	got := Rank("hello", "/tmp", a, cfg)
	if got[0].Name != "codex" {
		t.Fatalf("want default agent to win ties, got %+v", got)
	}
}
