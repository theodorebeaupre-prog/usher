# Demo recording

`usher-demo.{gif,mp4,webm}` — four scenes, recorded with [vhs](https://github.com/charmbracelet/vhs)
against the real released binary. Fake agent scripts shadow the real CLIs on
PATH (so no quota is spent and output is reproducible), a seeded ledger gives
codex a 71% quota-confidence, and everything printed is genuine usher behavior:
`doctor`, `--why` with a routed-around decision, a live headless failover
chain, and the animated banner.

To re-record: recreate the fake agents + setup script per the tape's comments,
then `vhs usher-demo.tape`.
