// Package config loads usher's TOML configuration.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type Pins struct {
	Types map[string]string `toml:"types"` // task type -> agent name
	Paths map[string]string `toml:"paths"` // directory prefix -> agent name
}

type Config struct {
	DefaultAgent      string                        `toml:"default_agent"`
	Disabled          []string                      `toml:"disabled"`
	Weights           map[string]map[string]float64 `toml:"weights"` // agent -> task type -> weight
	Pins              Pins                          `toml:"pins"`
	ContinuationGuard *bool                         `toml:"continuation_guard"`
}

// ContinuationGuardEnabled reports whether failover prompts carry the
// continuation notice. Pointer so "absent" (default on) is distinguishable
// from an explicit false.
func (c Config) ContinuationGuardEnabled() bool {
	return c.ContinuationGuard == nil || *c.ContinuationGuard
}

// Load reads a config file. A missing file is not an error — usher must
// work with zero configuration.
func Load(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) IsDisabled(agent string) bool {
	for _, d := range c.Disabled {
		if d == agent {
			return true
		}
	}
	return false
}

// Dir returns usher's config directory. Dev-tool convention: ~/.config even
// on macOS (not ~/Library/Application Support).
func Dir() string {
	if runtime.GOOS == "windows" {
		if d, err := os.UserConfigDir(); err == nil {
			return filepath.Join(d, "usher")
		}
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "usher")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "usher")
}
