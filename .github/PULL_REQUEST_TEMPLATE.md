## What

<!-- One or two sentences. Link the issue if there is one. -->

## Checklist

- [ ] `gofmt -l .` prints nothing, `go vet ./...` is clean
- [ ] `go test ./...` is green (unit + E2E)
- [ ] No new dependencies (stdlib + BurntSushi/toml only)

### If this adds an adapter

- [ ] One file in `internal/adapters/`, registered in `All()` (alphabetical after the founding four)
- [ ] Rows added to the LaunchArgs/HeadlessArgs table tests
- [ ] Quota fixture in `internal/adapters/testdata/quota/<name>.txt` (say in the PR whether it's captured-real or reconstructed)
- [ ] Flags verified against the CLI's `--help` (paste the relevant excerpt in the PR description)
- [ ] README agents table row
