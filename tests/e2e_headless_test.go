//go:build !windows

package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHeadlessStdinReplayedToFailoverAgent verifies piped stdin is buffered
// once and replayed to each attempt: claude drains stdin then quota-fails,
// and codex — the failover — must still see the original piped content.
func TestHeadlessStdinReplayedToFailoverAgent(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
/bin/cat >/dev/null; echo "usage limit reached" >&2; exit 1`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then echo "GOT_STDIN: $(/bin/cat)"; exit 0; fi`)

	cmd := exec.Command(bin, "-p", "fix the crash")
	cmd.Env = []string{"PATH=" + fakeDir, "HOME=" + cfgHome, "XDG_CONFIG_HOME=" + cfgHome}
	cmd.Stdin = strings.NewReader("context-from-pipe")
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	_ = cmd.Run()
	if !strings.Contains(out.String(), "GOT_STDIN: context-from-pipe") {
		t.Errorf("failover agent did not receive replayed stdin:\nstdout: %s\nstderr: %s", out.String(), errb.String())
	}
}

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
	// Launch events are recorded once a run starts (regardless of outcome),
	// not just on success — both claude and codex started here, so two
	// "launch" events are expected even though claude's attempt quota-failed.
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

func TestHeadlessSuccessWithQuotaWordNoFailover(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "note: discussing rate limit strategies" >&2
echo "ANSWER: use a token bucket"; exit 0`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then echo "CODEX_SHOULD_NOT_RUN"; exit 0; fi`)

	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "-p", "fix the crash")
	if exit != 0 {
		t.Fatalf("exit = %d\nstderr: %s", exit, stderr)
	}
	if strings.Contains(stdout, "CODEX_SHOULD_NOT_RUN") || strings.Contains(stderr, "failing over") {
		t.Errorf("exit-0 run must never fail over:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stdout, "ANSWER: use a token bucket") {
		t.Errorf("agent answer missing:\n%s", stdout)
	}
}

func TestHeadlessNoAgentsError(t *testing.T) {
	bin := buildUsher(t)
	_, stderr, exit := runSplit(t, bin, t.TempDir(), t.TempDir(), "-p", "fix the crash")
	if exit == 0 {
		t.Error("want non-zero exit with no agents installed")
	}
	if !strings.Contains(stderr, "headless") {
		t.Errorf("expected headless-agents error on stderr:\n%s", stderr)
	}
}

func TestHeadlessTimeout(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
/bin/sleep 30`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then echo "SHOULD_NOT_RUN"; exit 0; fi`)

	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "--timeout", "1s", "-p", "fix the crash")
	if exit != 124 {
		t.Errorf("exit = %d, want 124\nstderr: %s", exit, stderr)
	}
	if !strings.Contains(stderr, "timed out after 1s") {
		t.Errorf("missing timeout notice:\n%s", stderr)
	}
	if strings.Contains(stdout, "SHOULD_NOT_RUN") || strings.Contains(stderr, "failing over") {
		t.Error("timeout must not fail over")
	}
}

func TestTimeoutRequiresHeadless(t *testing.T) {
	bin := buildUsher(t)
	out, exit := run(t, bin, t.TempDir(), t.TempDir(), "--timeout", "5s", "fix the crash")
	if exit == 0 || !strings.Contains(out, "requires -p") {
		t.Errorf("want requires--p error, got exit %d: %s", exit, out)
	}
}

func TestHeadlessJSONEnvelopeAfterFailover(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "usage limit reached" >&2; exit 1`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then echo "the answer"; exit 0; fi`)

	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "--json", "-p", "fix the crash")
	if exit != 0 {
		t.Fatalf("exit = %d\nstderr: %s", exit, stderr)
	}
	var env struct {
		Agent    string `json:"agent"`
		TaskType string `json:"task_type"`
		ExitCode int    `json:"exit_code"`
		Output   string `json:"output"`
		Attempts []struct {
			Agent      string `json:"agent"`
			QuotaError bool   `json:"quota_error"`
		} `json:"attempts"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout is not a single JSON object: %v\n%s", err, stdout)
	}
	if env.Agent != "codex" || env.TaskType != "debug" || env.ExitCode != 0 ||
		!strings.Contains(env.Output, "the answer") ||
		len(env.Attempts) != 2 || !env.Attempts[0].QuotaError || env.Attempts[0].Agent != "claude" {
		t.Errorf("envelope wrong: %+v", env)
	}
}

func TestJSONRequiresHeadless(t *testing.T) {
	bin := buildUsher(t)
	out, exit := run(t, bin, t.TempDir(), t.TempDir(), "--json", "fix the crash")
	if exit == 0 || !strings.Contains(out, "requires -p") {
		t.Errorf("want requires--p error, got exit %d: %s", exit, out)
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
