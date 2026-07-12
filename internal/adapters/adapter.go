// Package adapters knows how to detect, launch, and read the failure modes
// of each supported agent CLI. Adding an agent = adding one file here.
package adapters

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// Test seams: swapped out in unit tests, never in production code.
var (
	lookPath   = exec.LookPath
	versionCmd = realVersionCmd
)

type Profile struct {
	Strengths   map[string]float64 // task type -> 0..1
	QuotaWindow time.Duration      // vendor's cap window for ledger decay
	InstallHint string             // one-liner shown when not installed
}

type Adapter interface {
	Name() string
	Bin() string
	Detect() (installed bool, version string)
	// Installed reports whether the CLI is on PATH without probing its
	// version — cheap enough for the launch path.
	Installed() bool
	LaunchArgs(prompt string) []string
	// HeadlessArgs is the argv for the agent's print-and-exit mode
	// (usher -p). nil means headless is unsupported or the prompt is empty.
	HeadlessArgs(prompt string) []string
	QuotaError(exitCode int, output string) bool
	Profile() Profile
}

// All returns every adapter in stable display order.
func All() []Adapter {
	return []Adapter{
		claudeAdapter{}, codexAdapter{}, geminiAdapter{}, opencodeAdapter{},
		ampAdapter{}, copilotAdapter{}, cursorAdapter{}, qwenAdapter{},
	}
}

func Get(name string) (Adapter, bool) {
	for _, a := range All() {
		if a.Name() == name {
			return a, true
		}
	}
	return nil, false
}

// detect is the shared Detect implementation.
func detect(bin string) (bool, string) {
	if _, err := lookPath(bin); err != nil {
		return false, ""
	}
	return true, versionCmd(bin)
}

// installed is the shared Installed implementation: a lookPath-only check,
// with no version probe, for use on the launch path.
func installed(bin string) bool {
	_, err := lookPath(bin)
	return err == nil
}

func realVersionCmd(bin string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return "unknown"
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(line)
}

// matchAny reports whether any pattern occurs in the output, case-insensitively.
// Patterns are recorded from real CLI failures — refine them as vendors change.
func matchAny(output string, patterns []string) bool {
	lower := strings.ToLower(output)
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
