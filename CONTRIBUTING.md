# Contributing to usher

The highest-value contribution is an **adapter** — support for another agent
CLI. An adapter is one file in `internal/adapters/` implementing:

```go
type Adapter interface {
    Name() string                                   // "myagent"
    Bin() string                                    // binary on PATH
    Detect() (installed bool, version string)       // use the shared detect() helper
    Installed() bool                                // use the shared installed() helper — cheap, no version probe
    LaunchArgs(prompt string) []string              // argv for an interactive session
    QuotaError(exitCode int, output string) bool    // recognize the vendor's cap error
    Profile() Profile                               // strengths, quota window, install hint
}
```

Copy `internal/adapters/opencode.go` as a template, register the adapter in
`All()` (keep the list alphabetical after the founding four), add cases to
`adapters_test.go`, and include a real captured rate-limit message in your PR
description so the quota patterns are grounded in reality.

Ground rules: stdlib + BurntSushi/toml only, `gofmt` clean, `go vet` clean,
tests green on macOS/Linux/Windows CI. The router must stay a pure function.

usher is MIT licensed; contributions are accepted under the same license.
