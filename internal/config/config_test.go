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

func TestLoadBadTOMLErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("this is = not [ toml"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}
