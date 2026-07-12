# Continuation guard on headless failover — design (2026-07-12)

Origin: launch-day feedback from a Reddit commenter (r/ClaudeCode) who pulled
the repo and ran the gates. Scenario: agent A partially edits the workspace,
hits its cap, usher fails over and replays the *original* prompt to agent B —
which then has no idea the task is half-done. "Replaying the original prompt
is necessary, but it is not a handoff by itself."

Target release: v0.3.2 (patch — behavior addition gated to failover attempts).

## Behavior

In headless mode (`-p`), when an attempt follows a previous attempt whose
agent **actually ran** (recorded a real exit — i.e. the quota-failover path),
usher prepends a continuation notice to the prompt passed to the next agent:

```
[usher failover] A previous agent (claude) was already working on this task and
hit its usage cap; the workspace may contain partial changes. Inspect `git status`
and the diff before continuing — do not assume the task is untouched.

<original prompt, byte-for-byte>
```

Rules:

- The notice names every prior agent that actually ran, comma-separated in
  the order they were attempted (e.g. `(claude, gemini)` after two capped runs).
- An attempt whose agent **failed to start** (exec error) never ran and does
  NOT trigger the guard for the next agent.
- Forced `--agent` does no failover → never guarded (unchanged).
- Interactive mode: out of scope. A human watching the TUI already has the
  context; revisit only if users ask.
- Stdin replay is unchanged — the guard rides only in the prompt argument.

## JSON envelope

`jsonAttempt` gains one field:

```go
ContinuationGuard bool `json:"continuation_guard,omitempty"`
```

`true` on any attempt that received the guarded prompt. `omitempty` keeps
first attempts and all pre-v0.3.2 envelope shapes byte-identical.

## Config opt-out

New top-level TOML key, default on:

```toml
continuation_guard = false   # disable the failover continuation notice
```

Implemented as `*bool` in `config.Config` (`toml:"continuation_guard"`) so
that "absent" and "explicitly false" are distinguishable; a helper
`ContinuationGuardEnabled() bool` returns true when nil. Fits the project
ethos: every behavior that edits routing or prompts is an overridable prior.

## Stderr

The existing failover line gains a suffix when the guard is active:

```
→ claude hit its cap — failing over to codex (with continuation notice)
```

Stdout purity invariant untouched: the guard travels inside the agent's
prompt argument; usher's own line stays on stderr.

## Implementation shape

- New pure function in `main.go` (beside the other headless helpers):
  `guardedPrompt(task string, priorAgents []string) string`. No I/O — unit
  testable exactly like the router.
- `cmdHeadless` tracks `priorRan []string`, appends `choice.Name` after any
  attempt that got a real exit, and passes the guarded prompt to
  `a.HeadlessArgs(...)` when `len(priorRan) > 0` and config allows.

## Testing

Unit (`main` package or alongside): `guardedPrompt` — empty priors returns
task unchanged; one/two priors produce the exact notice text.

E2E (extend `tests/e2e_headless_test.go`, existing fake-agent harness):

1. **Four-step scenario (the commenter's test):** fake claude exits 1 with a
   quota line; fake codex dumps `$@` to a file and exits 0. Assert codex's
   received prompt = notice + original task, envelope has
   `continuation_guard` absent on attempt 1 and `true` on attempt 2.
2. **Failed-to-start is not a prior:** fake claude is non-executable (or
   missing binary path), codex runs. Assert codex's prompt is the bare task.
3. **Opt-out:** config with `continuation_guard = false`, quota failover.
   Assert bare task on attempt 2 and no `continuation_guard` in envelope.

## Docs

- README.md headless section: short paragraph + the notice text; re-capture
  the `--json` failover envelope example from real output (byte-for-byte
  rule) if it now differs.
- README.fr.md: same content, mirrored (bilingual invariant).
- Follow-up issue (#6): user-supplied handoff file (the commenter's
  "stronger optional version") — out of scope here.

## Invariants check

- Failover triggers unchanged (non-zero exit + quota pattern only; never on
  exit 0, timeout, or forced agent).
- Headless stdout purity preserved.
- Zero new dependencies; gofmt/vet clean.
- Ledger behavior untouched.
