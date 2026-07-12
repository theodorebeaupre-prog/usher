# Security Policy

## Scope

usher is a local launcher: it makes no network calls, handles no credentials, and stores nothing but a local JSON ledger and TOML config under `~/.config/usher/`. The interesting attack surface is what it executes — usher launches agent CLIs found on your `PATH` with arguments built from your own command line.

Reports we especially care about:

- Argument construction that could be abused (injection through task text, config values, or environment)
- Anything that makes usher launch a binary the user didn't intend
- Ledger/config parsing issues with security consequences

## Reporting

Please use [GitHub private vulnerability reporting](https://github.com/theodorebeaupre-prog/usher/security/advisories/new) — don't open a public issue for anything exploitable. You'll get a response within a few days; fixes ship as patch releases with credit (if you want it).

## Supported versions

The latest minor release line receives fixes. Pre-1.0, that means: update to the latest tag.
