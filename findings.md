# Findings

## Security
- High: `internal/tui/watch/browser.go:31` writes HTML previews to a fixed filename in a shared temp directory; `internal/tui/watch/browser.go:33` uses mode `0644`. This can leak decrypted content to other users and is vulnerable to symlink/race attacks. Use `os.CreateTemp` and `0600`, and consider cleanup after viewing.
- Medium: `internal/tui/watch/model.go:509` opens the first link without validating the URL scheme, and `internal/tui/watch/browser.go:11` passes it directly to the OS handler. A malicious email could open `file://`, `javascript:`, or custom schemes. Consider whitelisting `http`/`https` (optionally `mailto`) unless explicitly overridden.
- Medium: `internal/cli/config.go:62` reads the API key with echo enabled; this can leak secrets in terminal history/screen recordings. Use `term.ReadPassword` and guard against short existing keys before slicing at `internal/cli/config.go:59`.

## Reliability
- High: `internal/cli/waitfor.go:102` uses `os.Exit` inside `RunE`, which bypasses defers like `client.Close()` at `internal/cli/waitfor.go:130` and `cancel()`. Return errors instead and let Cobra handle exit codes.
- Low: `internal/tui/watch/model.go:200` and `internal/tui/watch/model.go:218` read from channels without checking `ok`. If a channel closes, the goroutine can spin.

## Config / Consistency
- Low: `internal/cli/root.go:46` binds `output`, but `internal/config/config.go:13` expects `default_output`. This makes config load/save inconsistent.
