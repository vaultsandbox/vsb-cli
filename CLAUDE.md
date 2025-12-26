# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o vsb ./cmd/vsb   # Build binary
./vsb                        # Launch TUI dashboard
```

## Architecture

**vsb-cli** is a CLI tool for VaultSandbox, a service providing temporary email inboxes with quantum-safe encryption (ML-KEM-768, ML-DSA-65).

### Package Structure

- `cmd/vsb/main.go` - Entry point, calls `cli.Execute()`
- `internal/cli/` - Cobra command implementations
  - `root.go` - Root command, launches the TUI dashboard when run without subcommands
  - Commands: `inbox` (create/list/info/use/delete), `email` (list/view/audit/url/delete), `wait`, `config`, `export/import`
- `internal/config/` - Configuration and keystore
  - `config.go` - YAML config (~/.config/vsb/config.yaml), env var support (VSB_* prefix)
  - `keystore.go` - Encrypted inbox storage (~/.config/vsb/keystore.json), thread-safe with mutex
  - `client.go` - SDK client factory using config values
- `internal/tui/watch/` - Bubble Tea TUI for real-time email monitoring
  - `model.go` - Main TUI model, handles multi-inbox watching via SDK's `WatchInboxes`
  - Tabs: Content, Security, Links, Attachments, Raw
- `internal/styles/` - Lipgloss styling constants
- `internal/browser/` - Opens URLs/HTML in system browser

### SDK Dependency

Uses local `github.com/vaultsandbox/client-go` SDK (replace directive in go.mod points to `/home/vs/Desktop/dev/client-go`). Key types from SDK:
- `vaultsandbox.Client` - API client
- `vaultsandbox.Inbox` - Inbox with crypto keys
- `vaultsandbox.Email` - Decrypted email with Links, Attachments

### Data Flow

1. Inboxes created via API, keys stored locally in keystore
2. On `vsb` (root command), all stored inboxes are imported into SDK client
3. TUI watches all inboxes via SSE, decrypts emails locally using stored private keys
4. Config priority: CLI flags > env vars (VSB_*) > config file > defaults
