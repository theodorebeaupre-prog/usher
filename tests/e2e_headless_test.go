//go:build !windows

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runSplit is like run but keeps stdout and stderr separate — headless
// stdout purity is part of the contract.
func runSplit(t *testing.T, bin, fakeDir, cfgHome string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = []string{"PATH=" + fakeDir, "HOME=" + cfgHome, "XDG_CONFIG_HOME=" + cfgHome}
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	_ = cmd.Run()
	return out.String(), errb.String(), cmd.ProcessState.ExitCode()
}

func TestHeadlessPassthroughAndExitCode(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "-p" ]; then echo "HEADLESS_ANSWER: $2"; exit 0; fi
echo "unexpected: $@" >&2; exit 9`)

	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "-p", "fix the flaky auth test")
	if exit != 0 {
		t.Fatalf("exit = %d\nstderr: %s", exit, stderr)
	}
	if !strings.Contains(stdout, "HEADLESS_ANSWER: fix the flaky auth test") {
		t.Errorf("stdout missing agent output:\n%s", stdout)
	}
	if strings.Contains(stdout, "→") {
		t.Errorf("usher decision line leaked onto stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "→ claude") {
		t.Errorf("decision line missing from stderr:\n%s", stderr)
	}
}

func TestHeadlessFailover(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "usage limit reached" >&2; exit 1`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then echo "CODEX_RESULT: $2"; exit 0; fi
echo "unexpected: $@" >&2; exit 9`)

	// "fix the crash" classifies as debug: claude (0.90) outranks codex (0.85),
	// so claude goes first, hits its fake cap, and usher must fail over.
	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "-p", "fix the crash")
	if exit != 0 {
		t.Fatalf("exit = %d\nstderr: %s", exit, stderr)
	}
	if !strings.Contains(stdout, "CODEX_RESULT: fix the crash") {
		t.Errorf("failover output missing:\n%s", stdout)
	}
	if !strings.Contains(stderr, "failing over to codex") {
		t.Errorf("failover notice missing from stderr:\n%s", stderr)
	}
	data, err := os.ReadFile(filepath.Join(cfgHome, "usher", "ledger.json"))
	if err != nil {
		t.Fatalf("ledger not written: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"quota"`) || strings.Count(s, `"launch"`) < 2 {
		t.Errorf("ledger missing quota + two launch events: %s", s)
	}
}

func TestHeadlessForcedAgentNoFailover(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "usage limit reached" >&2; exit 1`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then echo "CODEX_RESULT: $2"; exit 0; fi`)

	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "--agent", "claude", "-p", "fix the crash")
	if exit == 0 {
		t.Fatalf("forced capped agent should exit non-zero\nstderr: %s", stderr)
	}
	if strings.Contains(stdout, "CODEX_RESULT") || strings.Contains(stderr, "failing over") {
		t.Errorf("forced agent must not fail over:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

func TestHeadlessNonQuotaExitPropagates(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "syntax error in prompt" >&2; exit 7`)

	_, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "-p", "fix the crash")
	if exit != 7 {
		t.Errorf("want agent's exit code 7, got %d\nstderr: %s", exit, stderr)
	}
	if strings.Contains(stderr, "failing over") {
		t.Errorf("non-quota failure must not trigger failover:\n%s", stderr)
	}
}
