# usher — Design Spec

**Date:** 2026-07-11
**Status:** Approved for planning
**Working name:** `usher` — the tool ushers your task to the right agent. Chosen 2026-07-11 after collision checks (GitHub, Homebrew, npm).

## Problem

Developers now pay for several AI coding agent subscriptions at once — Claude Code (Claude Max), Codex (ChatGPT), Gemini CLI, opencode, and more. Each has its own CLI, its own strengths, and its own usage caps. Picking which agent to launch for a given task is manual, and hitting a rate cap mid-session means manually falling back to another tool. No existing tool routes across *subscription* CLIs (as opposed to API keys).

## Product

A single cross-platform binary. You type:

```
usher "fix the flaky auth test"
```

It detects which agent CLIs are installed, scores the task against each agent's strengths and remaining-quota confidence, prints one explanatory line, and **execs into the winning agent's native interactive TUI** in the current directory with the prompt pre-seeded. No API keys — it uses the subscriptions you already pay for. The native experience of each agent is preserved entirely; usher is a launcher, not a wrapper around output.

### Goals

- One command that always starts the *right* agent, instantly (no network call to route).
- Route around exhausted quotas automatically.
- Transparent decisions (`--why`), manual override (`--agent`).
- Ship a polished v0.1 in days; earn GitHub stars in the agent-tooling niche.

### Non-goals (v0.1)

- Headless / scripted mode (`-p`-style unified output).
- Background daemon or menu bar UI.
- LLM-assisted routing.
- Multi-account switching within one vendor.
- Cost/usage *reporting* dashboards.

The router and ledger are separate modules specifically so these can be added later without rework.

## Architecture

Go, single static binary, released with goreleaser (macOS/Linux/Windows) plus a Homebrew tap.

Four modules:

### 1. Adapter registry (`internal/adapters/`)

One file per supported agent. v0.1 ships: **Claude Code, Codex, Gemini CLI, opencode**.

Each adapter implements one interface:

- `Detect() (installed bool, version string)` — e.g., `exec.LookPath` + `--version`.
- `Launch(prompt string, dir string) *exec.Cmd` — builds the interactive invocation with the prompt pre-seeded as an argument (e.g., `claude "<prompt>"`, `codex "<prompt>"`). Full TTY inheritance.
- `DetectQuotaError(exitCode int, capturedStderr string) bool` — recognizes that vendor's rate-limit/usage-cap signature.
- `Profile() Strengths` — static per-task-type weights (see Router) plus quota decay window metadata.

Adding a new agent = adding one file + registering it. This is the intended community-contribution surface.

### 2. Router (`internal/router/`)

A pure function:

```
Rank(task string, repo RepoContext, adapters []AdapterInfo, ledger LedgerState, cfg Config) []ScoredAgent
```

Scoring components, combined as a weighted sum:

1. **Task-type classification** — keyword/pattern heuristics bucket the task into: `debug`, `feature`, `refactor`, `review`, `docs`, `test`, `other`. Cheap, local, deterministic.
2. **Strength weights** — each adapter ships default weights per task type; users can override any weight in the config file.
3. **Quota penalty** — from the ledger: a recent observed rate-limit event applies a heavy penalty that decays over that vendor's window.
4. **Pinned rules** — user config can pin patterns or paths to an agent (e.g., `review → codex`, `~/Developer/PhotoCull → claude`). Pins beat scoring.

CLI behavior:

- `--agent <name>` overrides everything.
- `--why` prints the full scoring table before launching.
- Repo context in v0.1 is minimal: current directory path (for pinned rules) and primary language (for future use; collected but not weighted yet).

### 3. Usage ledger (`internal/ledger/`)

Local JSON file at `~/.config/usher/ledger.json` (no SQLite dependency; the data is tiny). Records:

- Each launch: agent, timestamp.
- Each observed quota event: agent, timestamp (reported by adapters after session exit).

Exposes `LedgerState` to the router: per-agent "quota confidence" computed with each vendor's decay window (e.g., Claude Max ~5-hour rolling window; per-adapter metadata). Explicitly a *confidence signal*, not accounting — the README must say so.

Writes are atomic (temp file + rename) so concurrent sessions can't corrupt it.

### 4. Launcher (`internal/launch/`)

- Prints one line before handoff: `→ claude (debug task, quota OK) — override with --agent`.
- Execs the chosen CLI in the current directory, inheriting stdin/stdout/stderr (true TTY passthrough; on Unix use syscall exec semantics where possible, otherwise process wait).
- On exit: passes exit code/stderr tail to the adapter's `DetectQuotaError`. If quota was hit, records it in the ledger and offers: `codex hit its usage cap — relaunch with claude (#2)? [Y/n]`.

### Supporting pieces

- **Config** — TOML at `~/.config/usher/config.toml`: weight overrides, pins, default agent, disabled agents.
- **`usher doctor`** — lists detected agents, versions, quota confidence, config path. First thing users run; also the debug tool for adapter issues.
- **`usher --version`, `usher list`** — the usual hygiene.

## Data flow

```
usher "task…"
  → registry.DetectAll()            (parallel, ~ms)
  → ledger.Load()
  → router.Rank(task, ctx, …)
  → print decision line (or full table with --why)
  → launch.Exec(winner)             (hands over the terminal)
  → on exit: adapter.DetectQuotaError → ledger.Record → optional relaunch offer
```

## Error handling

- **No agents installed** → friendly message listing supported agents with install one-liners; exit 1.
- **Exactly one agent installed** → skip scoring, launch it (still record to ledger).
- **Tie / low-confidence score** → launch the top agent anyway (never block on a prompt for the core flow); `--why` reveals closeness.
- **Ledger unreadable/corrupt** → warn once, start a fresh ledger; never fail a launch over ledger state.
- **Adapter launch failure** (CLI vanished between detect and exec) → offer runner-up.
- **Windows** — no exec syscall; run as child process and forward signals. Adapters must not assume POSIX.

## Testing

- **Router**: table-driven unit tests — task strings → expected ranking, including quota-penalty and pin cases. This is the highest-value test surface.
- **Adapters**: `Detect`/`DetectQuotaError` tested against fixture outputs (recorded real CLI error strings per vendor); `Launch` argument construction tested without executing.
- **Ledger**: decay-window math, atomic-write behavior, corrupt-file recovery.
- **End-to-end smoke**: a fake agent adapter (test binary) exercises detect → rank → exec → quota-record round trip in CI.
- CI: GitHub Actions, macOS + Linux + Windows matrix, `go test ./...` + goreleaser dry run.

## Release checklist (v0.1)

- README with demo GIF (same pipeline as AgentBar's), the "stop juggling your AI subscriptions" pitch up top, and an honest "how quota confidence works" section.
- Homebrew tap formula; goreleaser binaries for the three OSes.
- MIT license, copyright ISO NORD CA.
- `CONTRIBUTING.md` centered on "add an adapter in one file".

## Risks

- **Vendor CLI churn** — invocation flags and error strings change. Mitigation: adapters are one file each, fixtures make breakage visible, `doctor` helps users self-diagnose.
- **Quota signals are opaque** — vendors don't expose remaining quota. Mitigation: observed-event + decay model, honestly documented as a heuristic.
- **Heuristic routing feels wrong sometimes** — Mitigation: `--why` transparency, config overrides, pins; routing quality can improve incrementally without breaking the interface.
