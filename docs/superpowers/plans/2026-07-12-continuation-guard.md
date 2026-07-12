# Continuation Guard on Headless Failover — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When headless failover replays a prompt to the next agent, prepend a notice that a previous agent may have partially modified the workspace, and expose that in the `--json` envelope.

**Architecture:** One pure helper (`guardedPrompt`) in `main.go`, wired into the `cmdHeadless` attempt loop; one new `jsonAttempt` field; one optional TOML kill-switch in `internal/config`. Spec: `docs/superpowers/specs/2026-07-12-continuation-guard-design.md`.

**Tech Stack:** Go stdlib only (invariant: BurntSushi/toml is the sole dependency; do not add any).

## Global Constraints

- `gofmt -l .` empty and `go vet ./...` clean on every commit.
- Headless stdout purity: usher's own lines go to stderr only; the guard rides inside the agent's prompt argument.
- Failover triggers unchanged: non-zero exit + quota pattern only; never on exit 0, timeout, or forced `--agent`.
- README.md and README.fr.md change together; console examples match real output.
- Commits end with `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- Notice text (exact, single source of truth):
  `[usher failover] A previous agent (NAMES) was already working on this task and hit its usage cap; the workspace may contain partial changes. Inspect ` + "`git status`" + ` and the diff before continuing — do not assume the task is untouched.` then a blank line, then the original task.

---

### Task 1: Config kill-switch `continuation_guard`

**Files:**
- Modify: `internal/config/config.go` (Config struct, ~line 19)
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `Config.ContinuationGuard *bool` (toml `continuation_guard`) and `func (c Config) ContinuationGuardEnabled() bool` — true when the key is absent or true.

- [ ] **Step 1: Write the failing tests** (append to `internal/config/config_test.go`; match the file's existing helper style for writing temp TOML — read the file first):

```go
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
```

- [ ] **Step 2: Run to verify failure** — `go test ./internal/config/` → FAIL: `c.ContinuationGuardEnabled undefined`.

- [ ] **Step 3: Implement** in `internal/config/config.go`:

```go
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
```

- [ ] **Step 4: Verify** — `go test ./internal/config/` PASS; `gofmt -l . && go vet ./...` clean.
- [ ] **Step 5: Commit** — `feat(config): continuation_guard kill-switch (default on)`

---

### Task 2: `guardedPrompt` pure helper

**Files:**
- Modify: `main.go` (place near `sumDurationsMS`, ~line 216)
- Test: Create `main_test.go` (package main)

**Interfaces:**
- Produces: `func guardedPrompt(task string, priorAgents []string) string` — returns `task` unchanged when `priorAgents` is empty; otherwise notice + `\n\n` + task.

- [ ] **Step 1: Write the failing test** — create `main_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

func TestGuardedPromptNoPriors(t *testing.T) {
	if got := guardedPrompt("fix the crash", nil); got != "fix the crash" {
		t.Errorf("prompt must pass through unchanged, got %q", got)
	}
}

func TestGuardedPromptWithPriors(t *testing.T) {
	got := guardedPrompt("fix the crash", []string{"claude", "gemini"})
	wantPrefix := "[usher failover] A previous agent (claude, gemini) was already working on this task"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("notice prefix missing:\n%q", got)
	}
	if !strings.Contains(got, "Inspect `git status` and the diff before continuing") {
		t.Errorf("git-status instruction missing:\n%q", got)
	}
	if !strings.HasSuffix(got, "\n\nfix the crash") {
		t.Errorf("original task must end the prompt untouched:\n%q", got)
	}
}
```

- [ ] **Step 2: Run to verify failure** — `go test .` → FAIL: `undefined: guardedPrompt`.

- [ ] **Step 3: Implement** in `main.go`:

```go
// guardedPrompt prepends the failover continuation notice when a previous
// agent actually ran: replaying the bare prompt would let the next agent
// assume an untouched workspace. priorAgents is in attempt order.
func guardedPrompt(task string, priorAgents []string) string {
	if len(priorAgents) == 0 {
		return task
	}
	return fmt.Sprintf("[usher failover] A previous agent (%s) was already working on this task and hit its usage cap; the workspace may contain partial changes. Inspect `git status` and the diff before continuing — do not assume the task is untouched.\n\n%s",
		strings.Join(priorAgents, ", "), task)
}
```

- [ ] **Step 4: Verify** — `go test .` PASS; gates clean.
- [ ] **Step 5: Commit** — `feat: guardedPrompt — failover continuation notice builder`

---

### Task 3: Wire into `cmdHeadless` + envelope + E2E

**Files:**
- Modify: `main.go` — `jsonAttempt` (~line 181), `cmdHeadless` attempt loop (~lines 279–340)
- Test: `tests/e2e_headless_test.go` (append three tests)

**Interfaces:**
- Consumes: `guardedPrompt` (Task 2), `cfg.ContinuationGuardEnabled()` (Task 1).
- Produces: `jsonAttempt.ContinuationGuard bool` (json `continuation_guard,omitempty`); stderr failover line suffix ` (with continuation notice)`.

- [ ] **Step 1: Write the failing E2E tests** (append to `tests/e2e_headless_test.go`; helpers `buildUsher`/`fakeAgent`/`runSplit` already exist there):

```go
// The commenter's four-step scenario: A edits, A hits quota, usher launches
// B — B's prompt must announce the possibly-dirty workspace, and the JSON
// envelope must show which attempt was guarded.
func TestHeadlessFailoverContinuationGuard(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	promptFile := filepath.Join(t.TempDir(), "codex-prompt.txt")
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "usage limit reached" >&2; exit 1`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then printf '%s' "$2" > "`+promptFile+`"; echo "CODEX_RESULT"; exit 0; fi
echo "unexpected: $@" >&2; exit 9`)

	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "-p", "--json", "fix the crash")
	if exit != 0 {
		t.Fatalf("exit = %d\nstderr: %s", exit, stderr)
	}
	if !strings.Contains(stderr, "failing over to codex (with continuation notice)") {
		t.Errorf("stderr missing continuation hint:\n%s", stderr)
	}
	var env struct {
		Attempts []struct {
			Agent             string `json:"agent"`
			ContinuationGuard bool   `json:"continuation_guard"`
		} `json:"attempts"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("envelope: %v\n%s", err, stdout)
	}
	if len(env.Attempts) != 2 || env.Attempts[0].ContinuationGuard || !env.Attempts[1].ContinuationGuard {
		t.Errorf("want guard on attempt 2 only: %+v", env.Attempts)
	}
	if strings.Contains(strings.SplitN(stdout, `"attempts"`, 2)[0], "continuation_guard") {
		t.Errorf("top level must not grow fields; omitempty on attempt 1: %s", stdout)
	}
	got, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("codex never received the prompt: %v", err)
	}
	p := string(got)
	if !strings.HasPrefix(p, "[usher failover] A previous agent (claude) ") ||
		!strings.HasSuffix(p, "\n\nfix the crash") {
		t.Errorf("guarded prompt wrong:\n%q", p)
	}
}

// An agent that fails to START never touched the workspace — no guard.
// DEVIATION (found during execution): headless detection uses LookPath only,
// never --version, so a chmod-during-version trick can't work. Instead the
// claude fake is an executable file with a dead shebang: detection accepts
// it, exec fails with ENOENT.
func TestHeadlessStartFailureIsNotAPrior(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	promptFile := filepath.Join(t.TempDir(), "codex-prompt.txt")
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then chmod -x "$0"; echo "1.0.0 (fake)"; exit 0; fi
echo "must not run" >&2; exit 9`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then printf '%s' "$2" > "`+promptFile+`"; echo "CODEX_RESULT"; exit 0; fi
echo "unexpected: $@" >&2; exit 9`)

	_, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "-p", "fix the crash")
	if exit != 0 {
		t.Fatalf("exit = %d\nstderr: %s", exit, stderr)
	}
	got, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("codex never received the prompt: %v", err)
	}
	if string(got) != "fix the crash" {
		t.Errorf("start-failure must not trigger the guard, got:\n%q", string(got))
	}
}

// continuation_guard = false restores pre-v0.3.2 behavior byte-for-byte.
func TestHeadlessContinuationGuardOptOut(t *testing.T) {
	bin := buildUsher(t)
	fakeDir, cfgHome := t.TempDir(), t.TempDir()
	promptFile := filepath.Join(t.TempDir(), "codex-prompt.txt")
	if err := os.MkdirAll(filepath.Join(cfgHome, "usher"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgHome, "usher", "config.toml"),
		[]byte("continuation_guard = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeAgent(t, fakeDir, "claude", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
echo "usage limit reached" >&2; exit 1`)
	fakeAgent(t, fakeDir, "codex", `
if [ "$1" = "--version" ]; then echo "1.0.0 (fake)"; exit 0; fi
if [ "$1" = "exec" ]; then printf '%s' "$2" > "`+promptFile+`"; echo "CODEX_RESULT"; exit 0; fi
echo "unexpected: $@" >&2; exit 9`)

	stdout, stderr, exit := runSplit(t, bin, fakeDir, cfgHome, "-p", "--json", "fix the crash")
	if exit != 0 {
		t.Fatalf("exit = %d\nstderr: %s", exit, stderr)
	}
	if strings.Contains(stdout, "continuation_guard") {
		t.Errorf("opted-out envelope must not mention the guard: %s", stdout)
	}
	if strings.Contains(stderr, "(with continuation notice)") {
		t.Errorf("opted-out stderr must not mention the notice:\n%s", stderr)
	}
	if got, _ := os.ReadFile(promptFile); string(got) != "fix the crash" {
		t.Errorf("opted-out prompt must be bare, got:\n%q", string(got))
	}
}
```

(`encoding/json` may need adding to the test file's imports.)

- [ ] **Step 2: Run to verify failure** — `go test ./tests/ -run ContinuationGuard -v` and `-run StartFailure` → FAIL (no guard, no envelope field, no stderr suffix).

- [ ] **Step 3: Implement in `main.go`.** Four edits:

(a) `jsonAttempt`:

```go
type jsonAttempt struct {
	Agent             string `json:"agent"`
	ExitCode          int    `json:"exit_code"`
	QuotaError        bool   `json:"quota_error"`
	TimedOut          bool   `json:"timed_out"`
	DurationMS        int64  `json:"duration_ms"`
	ContinuationGuard bool   `json:"continuation_guard,omitempty"`
}
```

(b) top of the attempt loop (replaces the current fixed use of `task` in the launch call only — the candidate-filter `HeadlessArgs(task)` nil-check above the loop stays as is):

```go
	var priorRan []string // agents that actually executed (real exit), in attempt order
	for i, choice := range ranked {
		a, _ := adapters.Get(choice.Name)
		prompt := task
		guarded := false
		if len(priorRan) > 0 && cfg.ContinuationGuardEnabled() {
			prompt = guardedPrompt(task, priorRan)
			guarded = true
		}
```

and the launch call becomes `launch.Run(a.Bin(), a.HeadlessArgs(prompt), dir, opts)`.

(c) every `jsonAttempt{...}` literal in the loop gains `ContinuationGuard: guarded` (timeout, start-failure, and normal-exit appends — all three). After the normal-exit append (the line that records `exit, quotaErr`), add:

```go
		priorRan = append(priorRan, choice.Name)
```

(start-failure `continue`s BEFORE this append — an agent that never started is not a prior; timeout `os.Exit`s so it never matters.)

(d) the failover stderr line:

```go
		if i < len(ranked)-1 {
			note := ""
			if cfg.ContinuationGuardEnabled() {
				note = " (with continuation notice)"
			}
			fmt.Fprintf(os.Stderr, "%s\n", sgr(color, 93,
				fmt.Sprintf("→ %s hit its cap — failing over to %s%s", choice.Name, ranked[i+1].Name, note)))
		}
```

- [ ] **Step 4: Verify** — `go test ./...` all PASS (including the pre-existing failover tests, which must not break: they assert `strings.Contains(stderr, "failing over to codex")`, still true with the suffix). `gofmt -l .` empty, `go vet ./...` clean.
- [ ] **Step 5: Commit** — `feat: continuation guard on headless failover (+ envelope flag, opt-out)`

---

### Task 4: Docs — README.md + README.fr.md

**Files:**
- Modify: `README.md` (~lines 127–134 and the Configuration TOML block ~line 150)
- Modify: `README.fr.md` (mirrored sections)

- [ ] **Step 1: README.md.** After the failover paragraph (line 127), the example (line 132) gains the suffix, and a new sentence documents the guard. Replace lines 127–134 with:

```markdown
And the part your nightly job will love: if the agent hits its usage cap, usher automatically fails over to the next-best one — each agent tried at most once, no human required. Piped stdin is buffered and replayed to every attempt, so a failover agent sees the same input the first one did. Because the first agent may have left partial work behind, the replayed prompt is prefixed with a continuation notice — "a previous agent was already working on this task; inspect `git status` before continuing" — and the attempt is marked `"continuation_guard": true` in the `--json` envelope (disable with `continuation_guard = false` in config).

```console
$ usher -p "fix the crash"
→ claude  (debug task · headless)
→ claude hit its cap — failing over to codex (with continuation notice)
Patched the nil-check in auth.go; tests pass.
```
```

In the Configuration TOML example add one line after `disabled = [...]`:

```toml
continuation_guard = false      # failover agents get no "workspace may be dirty" notice
```

- [ ] **Step 2: README.fr.md.** Mirror the same two edits in French (translate the paragraph; keep code/example lines identical to English output — usher's output is not localized).
- [ ] **Step 3: Verify** — the example line matches the implementation string exactly: `→ claude hit its cap — failing over to codex (with continuation notice)`. Cross-check against a real run: reuse the E2E fakes manually or trust the E2E assertion (it asserts the same byte string).
- [ ] **Step 4: Commit** — `docs: continuation guard — headless section + config key (EN/FR)`

---

### Task 5: Release v0.3.2

**Files:** none (tag + GitHub release notes)

- [ ] **Step 1:** `go test ./... && gofmt -l . && go vet ./...` — all clean at HEAD.
- [ ] **Step 2:** `git push` and confirm CI green: `gh run watch` (3-OS matrix).
- [ ] **Step 3:** `git tag v0.3.2 && git push origin v0.3.2` — goreleaser builds binaries + updates the Homebrew cask automatically (HOMEBREW_TAP_GITHUB_TOKEN already configured, `skip_upload: auto`).
- [ ] **Step 4:** Watch the Release workflow; then verify the cask bumped: `gh api repos/theodorebeaupre-prog/homebrew-tap/contents/Casks/usher.rb | jq -r .content | base64 -d | grep version`.
- [ ] **Step 5:** Edit release notes (`gh release edit v0.3.2 --notes ...`): credit the r/ClaudeCode commenter's scenario, describe the guard, the envelope field, the opt-out. One paragraph + bullets, tone matching v0.3.1's notes.
- [ ] **Step 6:** Open follow-up issue #6 — "Continuation handoff file (user-supplied) for failover" — labeled `help wanted`, referencing the spec's out-of-scope note.
- [ ] **Step 7:** Update `~/Developer/usher-launch-plan.md` launch log + memory file `usher-project.md`. Draft the Reddit reply for Théodore (he posts it — never post on his behalf).

## Self-review

- Spec coverage: behavior (T2+T3), envelope (T3), opt-out (T1+T3 test 3), stderr (T3), start-failure exclusion (T3 test 2), forced-agent unaffected (no failover path — existing `TestHeadlessForcedAgentNoFailover` still passes), docs EN/FR (T4), follow-up issue (T5). Interactive mode: correctly absent (out of scope).
- No placeholders; all code complete.
- Type consistency: `guardedPrompt(task string, priorAgents []string) string` used identically in T2/T3; `ContinuationGuardEnabled()` matches T1.
