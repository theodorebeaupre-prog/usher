package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileIsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if cfg.DefaultAgent != "" || len(cfg.Disabled) != 0 {
		t.Fatalf("expected zero config, got %+v", cfg)
	}
}

func TestLoadFullConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `
default_agent = "claude"
disabled = ["opencode"]

[weights.claude]
review = 0.9

[pins.types]
review = "codex"

[pins.paths]
"/Users/new/Developer/PhotoCull" = "claude"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultAgent != "claude" {
		t.Errorf("DefaultAgent = %q", cfg.DefaultAgent)
	}
	if !cfg.IsDisabled("opencode") || cfg.IsDisabled("claude") {
		t.Errorf("IsDisabled wrong: %+v", cfg.Disabled)
	}
	if cfg.Weights["claude"]["review"] != 0.9 {
		t.Errorf("Weights = %+v", cfg.Weights)
	}
	if cfg.Pins.Types["review"] != "codex" {
		t.Errorf("Pins.Types = %+v", cfg.Pins.Types)
	}
	if cfg.Pins.Paths["/Users/new/Developer/PhotoCull"] != "claude" {
		t.Errorf("Pins.Paths = %+v", cfg.Pins.Paths)
	}
}

func TestContinuationGuardEnabledDefault(t *testing.T) {
	var c Config
	if !c.ContinuationGuardEnabled() {
		t.Error("absent continuation_guard must default to enabled")
	}
}

func TestContinuationGuardExplicit(t *testing.T) {
	off, on := false, true
	c := Config{ContinuationGuard: &off}
	if c.ContinuationGuardEnabled() {
		t.Error("continuation_guard = false must disable")
	}
	c.ContinuationGuard = &on
	if !c.ContinuationGuardEnabled() {
		t.Error("continuation_guard = true must enable")
	}
}

func TestLoadContinuationGuardFalse(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte("continuation_guard = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ContinuationGuardEnabled() {
		t.Error("loaded continuation_guard = false must disable")
	}
}

func TestLoadBadTOMLErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("this is = not [ toml"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}
