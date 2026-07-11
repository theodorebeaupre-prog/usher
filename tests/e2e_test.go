//go:build !windows

// Package tests holds black-box tests: they build the real binary and run
// it against fake agent CLIs placed on PATH.
package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildUsher(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "usher")
	cmd := exec.Command("go", "build", "-o", bin, "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func fakeAgent(t *testing.T, dir, name, script string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, bin, fakeDir, cfgHome string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = []string{
		"PATH=" + fakeDir, // ONLY the fake dir: no real agents visible
		"HOME=" + cfgHome,
		"XDG_CONFIG_HOME=" + cfgHome,
	}
	out, _ := cmd.CombinedOutput()
	return string(out), cmd.ProcessState.ExitCode()
}

func TestRoutesAndLaunchesFakeAgent(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "FAKE_CLAUDE_RAN: $1"`)

	out, exit := run(t, bin, fakeDir, cfgHome, "fix the flaky auth test")
	if exit != 0 {
		t.Fatalf("exit = %d\n%s", exit, out)
	}
	if !strings.Contains(out, "→ claude") {
		t.Errorf("missing decision line:\n%s", out)
	}
	if !strings.Contains(out, "FAKE_CLAUDE_RAN: fix the flaky auth test") {
		t.Errorf("fake agent did not receive the prompt:\n%s", out)
	}
}

func TestQuotaErrorRecordedInLedger(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "usage limit reached" >&2; exit 1`)

	out, _ := run(t, bin, fakeDir, cfgHome, "fix the flaky auth test")
	ledgerPath := filepath.Join(cfgHome, "usher", "ledger.json")
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("ledger not written: %v\n%s", err, out)
	}
	if !strings.Contains(string(data), `"quota"`) {
		t.Errorf("ledger missing quota event: %s", data)
	}
}

func TestNoAgentsInstalled(t *testing.T) {
	bin := buildUsher(t)
	out, exit := run(t, bin, t.TempDir(), t.TempDir(), "fix something")
	if exit == 0 {
		t.Error("want non-zero exit with no agents")
	}
	if !strings.Contains(out, "no agent CLIs found") {
		t.Errorf("missing install hints:\n%s", out)
	}
}
