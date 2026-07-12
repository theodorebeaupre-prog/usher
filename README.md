<div align="center">

<img src="assets/banner/usher-banner.gif" alt="usher — a tiny pixel usher walks in, lights up the letters u-s-h-e-r with a flashlight, and the tagline types out: right this way." width="640">

### One command. The right AI coding agent. Every time.

You already pay for Claude Code, Codex, Gemini CLI, Copilot… **usher decides which one gets the job** — routing on task type, each agent's strengths, and who still has quota left. Built on the subscriptions you have. **No API keys.**

[![release](https://img.shields.io/github/v/release/theodorebeaupre-prog/usher?color=4c1)](https://github.com/theodorebeaupre-prog/usher/releases/latest)
[![CI](https://github.com/theodorebeaupre-prog/usher/actions/workflows/ci.yml/badge.svg)](https://github.com/theodorebeaupre-prog/usher/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-MIT-8a8a8a)](LICENSE)
[![made for](https://img.shields.io/badge/made_for-your_terminal-56b6c2)](#how-it-works)
[![buy me a coffee](https://img.shields.io/badge/☕_buy_me_a_coffee-isonord-FFDD00)](https://buymeacoffee.com/isonord)

🇫🇷 [Version française](README.fr.md)

</div>

> **v0.3 — feature-complete for the 1.0 freeze.** Every terminal example below is live behavior, not aspiration. Found a rough edge? [Issues welcome](https://github.com/theodorebeaupre-prog/usher/issues).

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

## Quick start

```console
$ brew install theodorebeaupre-prog/tap/usher              # macOS
$ go install github.com/theodorebeaupre-prog/usher@latest  # anywhere with Go
```

Or grab a [prebuilt binary](https://github.com/theodorebeaupre-prog/usher/releases/latest) (macOS / Linux / Windows, amd64 + arm64).

Then take attendance, and go:

```console
$ usher doctor
$ usher "explain what this repo does"
```

## How it works

<div align="center"><img src="assets/demo/usher-demo.gif" alt="usher demo: doctor, --why routing table, headless failover, animated banner" width="760"></div>

Three things happen in the milliseconds before handoff — all local, no network call, works offline:

1. **Detect** — finds which agent CLIs are installed (Claude Code, Codex, Gemini CLI, opencode, Amp, GitHub Copilot CLI, Cursor, Qwen Code).
2. **Rank** — a transparent heuristic scores each agent: task-type classification (`debug / feature / refactor / review / docs / test / other`) × per-agent strength weights × a quota-confidence penalty × your pinned rules.
3. **Exec** — hands you the winner's interactive session. Full TTY, native experience, zero latency added where it counts.

Not sure why it picked what it picked? Ask it.

```console
$ usher --why "review this PR for security issues"
task type: review

  agent       strength   quota    pin   score
  codex           0.95    1.00      —    0.95  ← launching
  gemini          0.70    1.00      —    0.70
  opencode        0.70    1.00      —    0.70
  claude          0.85    0.71      —    0.60

→ codex  (review task · quota OK · override with --agent)
```

(That `0.71` is claude cooling down from a recent cap — strongest reviewer after codex, but usher routes around the wall instead of into it.)

### Where the scores come from

usher doesn't *know* which agent is best — it has **written-down opinions, multiplied by what it has observed.** Three ingredients:

1. **Task type** — plain keyword matching sorts your sentence into a bucket, checked in priority order: *fix/crash/flaky* → `debug` before *test* → `test`, which is why "fix the flaky test" counts as debugging, not testing. No match → `other`.
2. **Strength priors** — every adapter ships a hand-written table. An excerpt of the actual defaults:

   | | debug | feature | refactor | review | docs | test |
   |---|---|---|---|---|---|---|
   | claude | **.90** | **.90** | **.95** | .85 | .85 | .85 |
   | codex | .85 | .80 | .80 | **.95** | .70 | **.90** |
   | gemini | .70 | .75 | .70 | .70 | **.90** | .70 |
   | cursor | .80 | .85 | .85 | .75 | .70 | .80 |

   Full honesty: these are **editorial judgments of each tool's reputation, not benchmarks** — Claude Code's multi-file refactoring earns its `.95` the same way Codex's review mode earns its own. They're priors: deliberately visible, deliberately yours to override.
3. **Quota confidence** — each strength is multiplied by what the [ledger](#routing-around-rate-limits) has witnessed. A cap observed 40 minutes ago scales that agent down until its window recovers: best *available* beats best *on paper*.

Pins outrank scores, `--agent` outranks everything, and `--why` shows the whole calculation. usher can be wrong — it can never be mysterious.

Why no LLM in the loop? Routing would then cost latency, network, and quota — *spending quota to decide how to spend quota* — and you couldn't predict or override it. The default stays boring on purpose; the long-term plan is that **you teach it**: every routing decision you disagree with is a weight to nudge in your config, and the 1.0 pass tunes the shipped defaults from real-world use.

And when you disagree in the moment, you win: `usher --agent claude "…"` skips the scoring entirely, and the [config file](#configuration) makes your opinion permanent.

## Routing around rate limits

Vendors don't expose remaining quota, so usher doesn't pretend to know it. Instead it keeps a small local ledger of **observed** events: every launch, and every time a session ends in a rate-limit error. Each vendor's cap window (Claude's ~5-hour rolling window, and friends) decays the penalty back to zero.

It's a confidence signal, not accounting — documented honestly as exactly that. In practice it means the difference between usher and a shell alias:

```console
$ usher "add dark mode to the settings pane"
→ claude  (feature task · codex is capped — routed around it · override with --agent)
```

Hit a cap *mid-session*? When the agent exits, usher notices, records it, and offers the runner-up. Your prompt travels with you; the retyping ritual is dead.

## Scripting & CI

Headless mode: `-p` runs the winner's own print-and-exit mode — the answer lands on stdout, usher's routing chatter stays on stderr, and the agent's exit code is yours.

```console
$ usher -p "summarize the failing tests in one paragraph"
→ claude  (test task · headless)          # stderr
The three failures share one cause: …     # stdout — pipe it anywhere
```

And the part your nightly job will love: if the agent hits its usage cap, usher automatically fails over to the next-best one — each agent tried at most once, no human required. Piped stdin is buffered and replayed to every attempt, so a failover agent sees the same input the first one did. Because the first agent may have left partial work behind, the replayed prompt is prefixed with a continuation notice — "a previous agent was already working on this task; inspect `git status` before continuing" — and the attempt is marked `"continuation_guard": true` in the `--json` envelope (disable with `continuation_guard = false` in config).

```console
$ usher -p "fix the crash"
→ claude  (debug task · headless)
→ claude hit its cap — failing over to codex (with continuation notice)
Patched the nil-check in auth.go; tests pass.
```

For tooling, `--json` wraps the run in one machine-readable object (the only thing printed to stdout — final agent, task type, exit code, output, and a per-attempt array), and `--timeout` keeps a hung agent from stalling your pipeline: the whole process group is killed, exit 124, GNU convention.

```console
$ usher -p --json --timeout 5m "fix the crash" | jq .agent
"codex"
```

The timeout applies per attempt, so a failover chain can take up to N× the value. On total failure the envelope's top-level `exit_code` mirrors the process exit code (the last attempt's entry may show `-1` if it failed to start).

## Configuration

None required — usher works out of the box. When you want opinions, they live in one TOML file:

```toml
# ~/.config/usher/config.toml — every key optional

default_agent = "claude"        # wins ties
disabled = ["opencode"]         # never route here
continuation_guard = false      # failover agents get no "workspace may be dirty" notice

[weights.codex]                 # override any strength, per agent × task type
review = 0.99                   # types: debug feature refactor review docs test other

[pins.types]                    # this task type ALWAYS goes to this agent
review = "codex"

[pins.paths]                    # work under this directory goes to this agent
"/Users/you/work/monorepo" = "claude"   # longest matching prefix wins
```

Pins beat scores; `--agent` beats everything. Respects `XDG_CONFIG_HOME`; on Windows the file lives under `%AppData%\usher`.

## Check the room

`usher doctor` shows what's installed, each agent's quota confidence, and where your config and ledger live:

```console
$ usher doctor
  claude     2.1.201 (Claude Code) quota ██████████ 100%
  codex      codex-cli 0.142.0 quota ██████████ 100%
  gemini     0.47.0         quota ██████████ 100%
  opencode   1.17.18        quota ██████████ 100%
  amp        not installed → npm install -g @sourcegraph/amp
  copilot    not installed → npm install -g @github/copilot
  cursor     not installed → curl https://cursor.com/install -fsS | bash
  qwen       not installed → npm install -g @qwen-code/qwen-code
  config: ~/.config/usher/config.toml (not found — using defaults)
  ledger: ~/.config/usher/ledger.json (2 events, confidence not accounting)
```

## Supported agents

| Agent | Subscription it uses | Adapter |
|---|---|---|
| **Claude Code** | Claude Pro / Max | ✅ v0.1 |
| **Codex** | ChatGPT Plus / Pro | ✅ v0.1 |
| **Gemini CLI** | Google AI Pro / free tier | ✅ v0.1 |
| **opencode** | bring-your-own (incl. subscriptions) | ✅ v0.1 |
| **Amp** | Amp free / Pro | ✅ v0.3 |
| **GitHub Copilot CLI** | Copilot Free / Pro / Pro+ | ✅ v0.2 |
| **Cursor CLI** | Cursor Hobby / Pro | ✅ v0.2 |
| **Qwen Code** | Qwen free tier / ModelScope | ✅ v0.3 |
| *yours?* | | [an adapter is one file](#contributing) |

## FAQ

**How do I connect my Claude / ChatGPT / Copilot account?**
You don't — there's nothing to connect. usher runs the same CLI commands you already use (`claude`, `codex`, …), with the logins they already have. If an agent works when you type its name, usher can route to it; `usher doctor` shows you exactly who's in the room.

**Is this okay with the vendors?**
usher never touches an API, a token, or a login. It starts the same official CLI you'd start yourself — your install, your auth, their TUI — exactly like a shell alias with judgment. If you're allowed to type `claude`, you're allowed to have something type it for you.

**Does usher phone home?**
No. Zero network calls, zero telemetry. Detection is a PATH lookup, routing is local arithmetic, and the ledger is a JSON file in `~/.config/usher/` you can read (or delete) anytime.

**Where's the AI in the routing?**
There isn't one — keyword heuristics and your config, deterministic and explainable with `--why`. That's a feature: routing adds zero latency, works offline, and never burns quota deciding how to spend quota.

**How accurate is "quota confidence"?**
It's built from what usher has *seen* — launches and observed cap errors, decayed over each vendor's window. It's honest about being a signal, not accounting: vendors don't expose the real numbers to anyone.

**My agent isn't supported.**
[One file](CONTRIBUTING.md). Detection, launch args, cap-error patterns, a strength profile — copy an existing adapter and send the PR.

## Design principles

- **Launcher, not wrapper.** usher never parses, filters, or rebrands an agent's output. You get the native tool, always.
- **Your subscriptions, not API keys.** The economics you already chose keep working.
- **Transparent over clever.** Every routing decision is explainable (`--why`), overridable (`--agent`), and configurable (TOML). No magic, no LLM in the routing path.
- **Boring reliability.** Single static Go binary. No daemon, no background process, nothing to update when vendors tweak their models.

## Roadmap

- [x] Design spec — [read it](docs/superpowers/specs/2026-07-11-usher-design.md)
- [x] Identity: wordmark + [animated terminal banner](assets/banner/) (engineering inspired by [GitHub Copilot CLI's banner](https://github.blog/engineering/from-pixels-to-characters-the-engineering-behind-github-copilot-clis-animated-ascii-banner/))
- [x] v0.1 — adapters ×4, heuristic router, quota ledger, `doctor`, `--why`
- [x] v0.2 — headless mode (`-p`) with auto-failover, Copilot + Cursor adapters
- [x] v0.3 — `--json` envelope, `--timeout` guard, stdin replay, Qwen + Amp adapters
- [ ] v1.0 — interface freeze, shell completions, man page, adapters verified on real machines
- [ ] Later — optional LLM-assisted routing, multi-account support

## Contributing

The whole architecture bends toward one contribution: **adding an agent is writing one file.** An adapter declares how to detect the CLI, how to launch it (interactive and headless), how to recognize its rate-limit error, and its strength profile. That's the entire interface — see [CONTRIBUTING.md](CONTRIBUTING.md).

Adapters we'd love PRs for: **Aider**, **Goose**, **Droid** — or whichever CLI you're paying for that isn't in the table yet. A captured rate-limit error message from a real session is worth its weight in gold (all but one of our quota fixtures are reconstructions — help us replace them with the real thing).

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
<sub>If usher saved you a retype today, a ⭐ helps the next person find the door — and <a href="https://buymeacoffee.com/isonord">a ☕</a> keeps him in bow ties.</sub>
<br><br>
<sub>usher — <em>right this way.</em></sub>
</div>
