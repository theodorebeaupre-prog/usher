# Quota-error fixtures

One file per agent: a rate-limit/usage-cap error string the adapter's
QuotaError patterns MUST match. Provenance:

- claude.txt — format captured from a real Claude Code cap event (v0.1).
- codex.txt, gemini.txt, opencode.txt, copilot.txt, cursor.txt — best-guess
  reconstructions from vendor docs/issue trackers; NOT captured live.
  Replace with a real capture when one is observed and note it here.

If a vendor changes their error copy, update the fixture AND the pattern
list in the adapter together.
