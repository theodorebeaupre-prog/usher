<div align="center">

<img src="assets/banner/usher-banner.gif" alt="usher — a tiny pixel usher walks in, lights up the letters u-s-h-e-r with a flashlight, and the tagline types out: right this way." width="640">

### One command. The right AI coding agent. Every time.

You already pay for Claude Code, Codex, Gemini CLI… **usher decides which one gets the job** — routing on task type, each agent's strengths, and who still has quota left. Built on the subscriptions you have. **No API keys.**

[![status](https://img.shields.io/badge/status-v0.2-4c1)](#roadmap)
[![license](https://img.shields.io/badge/license-MIT-8a8a8a)](LICENSE)
[![made for](https://img.shields.io/badge/made_for-your_terminal-56b6c2)](#how-it-works)

</div>

> **v0.2.** Early but real — the examples below are live behavior. Found a rough edge? Issues welcome.

---

## Why

If you're serious about AI-assisted coding in 2026, you're probably paying for two or three agent subscriptions at once. Each one has its own CLI, its own personality, its own strengths — and its own usage caps that you discover mid-task, at the worst possible moment.

So every task starts with the same silent ritual: *which agent do I open for this?* And every rate limit ends with the same one: *fine, let me retype all of this into the other one.*

usher removes the ritual. You describe the task; it seats you with the right agent.

```console
$ usher "fix the flaky auth test in photocull"
→ claude  (debug task · quota OK · override with --agent)
```

…and you're inside a normal Claude Code session, in your repo, prompt already delivered. usher is a **launcher, not a wrapper** — it hands you the agent's real TUI and gets out of the way. Nothing sits between you and the tool you paid for.

## Install

```console
$ brew install theodorebeaupre-prog/tap/usher   # macOS (Homebrew)
$ go install github.com/theodorebeaupre-prog/usher@latest   # anywhere with Go
```

## How it works

Three things happen in the milliseconds before handoff — all local, no network call, works offline:

1. **Detect** — finds which agent CLIs are installed (Claude Code, Codex, Gemini CLI, opencode, GitHub Copilot CLI, Cursor).
2. **Rank** — a transparent heuristic scores each agent: task-type classification (debug / feature / refactor / review / docs / test) × per-agent strength weights × a quota-confidence penalty × your pinned rules.
3. **Exec** — replaces itself with the winner's interactive session. Full TTY, native experience, zero latency added where it counts.

Not sure why it picked what it picked? Ask it.

```console
$ usher --why "review this PR for security issues"
task type: review

  agent    strengths  quota   pins   score
  codex        0.92    1.00      —    0.92  ← launching
  claude       0.85    0.71      —    0.60      (cap hit 38 min ago)
  gemini       0.55    1.00      —    0.55

→ codex  (review task · best available)
```

And when you disagree, you win: `usher --agent claude "…"` skips the scoring entirely, and a TOML config lets you re-weight anything or pin patterns for good (`review → codex`, `this repo → claude`).

## Routing around rate limits

Vendors don't expose remaining quota, so usher doesn't pretend to know it. Instead it keeps a small local ledger of **observed** events: every launch, and every time a session ends in a rate-limit error. Each vendor's cap window (Claude's ~5-hour rolling window, and friends) decays the penalty back to zero.

It's a confidence signal, not accounting — documented honestly as exactly that. In practice it means the difference between usher and a shell alias:

```console
$ usher "add dark mode to the settings pane"
→ claude  (feature task · codex is capped — routed around it · override with --agent)
```

Hit a cap *mid-session*? When the agent exits, usher notices and offers the runner-up. Your prompt travels with you; the retyping ritual is dead.

## Scripting & CI

Headless mode: `-p` runs the winner's own print-and-exit mode — the answer
lands on stdout, usher's routing chatter stays on stderr, and the agent's
exit code is yours.

```console
$ usher -p "summarize the failing tests in one paragraph"
→ claude  (test task · headless)          # stderr
The three failures share one cause: …     # stdout — pipe it anywhere
```

And the part your nightly job will love: if the agent hits its usage cap,
usher automatically fails over to the next-best one — each agent tried at
most once, no human required.

```console
$ usher -p "fix the crash"
→ claude  (debug task · headless)
→ claude hit its cap — failing over to codex
Patched the nil-check in auth.go; tests pass.
```

Piped stdin is buffered and replayed to every attempt, so a failover agent sees the same input the first one did.

## Supported agents

| Agent | Subscription it uses | Adapter |
|---|---|---|
| **Claude Code** | Claude Pro / Max | ✅ v0.1 |
| **Codex** | ChatGPT Plus / Pro | ✅ v0.1 |
| **Gemini CLI** | Google AI Pro / free tier | ✅ v0.1 |
| **opencode** | bring-your-own (incl. subscriptions) | ✅ v0.1 |
| **GitHub Copilot CLI** | Copilot Free / Pro / Pro+ | ✅ v0.2 |
| **Cursor CLI** | Cursor Hobby / Pro | ✅ v0.2 |
| *yours?* | | [an adapter is one file](#contributing) |

## Design principles

- **Launcher, not wrapper.** usher never parses, filters, or rebrands an agent's output. You get the native tool, always.
- **Your subscriptions, not API keys.** The economics you already chose keep working.
- **Transparent over clever.** Every routing decision is explainable (`--why`), overridable (`--agent`), and configurable (TOML). No magic, no LLM in the routing path.
- **Boring reliability.** Single static Go binary. No daemon, no background process, nothing to update when vendors tweak their models.

## Roadmap

- [x] Design spec — [read it](docs/superpowers/specs/2026-07-11-usher-design.md)
- [x] Identity: wordmark + [animated terminal banner](assets/banner/) (engineering inspired by [GitHub Copilot CLI's banner](https://github.blog/engineering/from-pixels-to-characters-the-engineering-behind-github-copilot-clis-animated-ascii-banner/))
- [x] v0.1 — adapters ×4, heuristic router, quota ledger, `doctor`, `--why`, Homebrew tap
- [x] v0.2 — headless mode (-p) with auto-failover, Copilot + Cursor adapters
- [ ] v0.3 — --json envelope, --timeout guard, more adapters
- [ ] Later — optional LLM-assisted routing, multi-account support

## Contributing

The whole architecture bends toward one contribution: **adding an agent is writing one file.** An adapter declares how to detect the CLI, how to launch it with a seeded prompt, how to recognize its rate-limit error, and its strength profile. That's the entire interface — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Meet the usher

<img src="assets/usher-orange.png" alt="usher wordmark, orange pixel variant" width="360">

He has one job: walk you to the right seat. He takes it seriously — red bow tie and all. Watch him work — run usher with no arguments:

```console
$ usher
```

*(Respects `NO_COLOR` and `USHER_NO_BANNER`; prints a static frame when piped; never sent to screen readers. The frames are generated by [frames.py](assets/banner/frames.py) — the same data drives the Go banner and the GIF above.)*

## License

[MIT](LICENSE) © 2026 ISO NORD CA

<div align="center">
<sub>usher — <em>right this way.</em></sub>
</div>
