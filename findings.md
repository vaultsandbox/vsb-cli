Findings (ordered by severity)

- High: Keystore writes don’t ensure the config directory exists, so a first-time `vsb inbox create/import/use` can fail with “no such file or directory.” `internal/config/keystore.go:99-110` and `internal/config/keystore.go:252-257`
- Medium: `vsb config` can panic when the stored API key is shorter than 11 chars due to slicing. `internal/cli/config.go:54-56`
- Medium: `--ttl 7d` is advertised but `time.ParseDuration` doesn’t accept days, so the example in help text fails. `internal/cli/inbox_create.go:38-55`
- Medium: Watch goroutines read from channels without checking close; if the SDK closes a channel before context cancellation, it can spin and burn CPU. `internal/tui/watch/model.go:212-249`
- Low: Errors from `cli.Execute()` are not printed in `main`, so users might see a silent non‑zero exit depending on Cobra error settings. `cmd/vsb/main.go:9-12`
- Low: A compiled `vsb` binary is in the repo root; for open source this is usually excluded. `vsb`
