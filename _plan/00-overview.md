# VaultSandbox CLI Implementation Plan

## Overview

This plan outlines the implementation of the `vsb` CLI tool, a developer companion for testing email flows using the VaultSandbox service. The CLI leverages the unpublished `client-go` SDK located at `/home/vs/Desktop/dev/client-go`.

## Core Principles

1. **Zero-Knowledge**: Server never sees plaintext; CLI handles all crypto locally
2. **Developer UX**: Fast, beautiful, and intuitive terminal experience
3. **CI/CD Ready**: First-class support for automated testing pipelines

## Technical Stack

| Component | Library | Purpose |
|-----------|---------|---------|
| CLI Framework | `cobra` | Command structure and argument parsing |
| TUI | `bubbletea` | Real-time interactive dashboard (`vsb watch`) |
| Styling | `lipgloss` | Pretty terminal output and formatting |
| Config | `viper` | Configuration file management |
| SDK | `client-go` (local) | VaultSandbox API and crypto operations |

## Project Structure

```
vsb-cli/
├── cmd/
│   └── vsb/
│       └── main.go           # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go           # Root command
│   │   ├── inbox.go          # inbox create/list/use
│   │   ├── watch.go          # Real-time TUI
│   │   ├── waitfor.go        # CI/CD wait command
│   │   ├── audit.go          # Email audit
│   │   ├── open.go           # Link extractor
│   │   ├── view.go           # HTML preview
│   │   ├── export.go         # Export inbox
│   │   └── import.go         # Import inbox
│   ├── config/
│   │   ├── config.go         # Config management
│   │   └── keystore.go       # Keystore management
│   ├── tui/
│   │   ├── watch/            # Watch TUI components
│   │   └── styles/           # Shared lipgloss styles
│   └── output/
│       └── printer.go        # Formatted output helpers
├── go.mod
├── go.sum
└── README.md
```

## Implementation Phases

### Phase 1: Foundation
- [01-project-setup.md](01-project-setup.md) - Go module, dependencies, main.go
- [02-config-keystore.md](02-config-keystore.md) - Config and keystore management

### Phase 2: Core Commands
- [03-inbox-commands.md](03-inbox-commands.md) - `inbox create`, `list`, `use`
- [04-watch-command.md](04-watch-command.md) - Real-time TUI dashboard

### Phase 3: Developer Tools
- [05-wait-for-command.md](05-wait-for-command.md) - CI/CD wait-for command
- [06-audit-command.md](06-audit-command.md) - Deep-dive email audit

### Phase 4: Convenience Commands
- [07-open-view-commands.md](07-open-view-commands.md) - `open` and `view` commands
- [08-export-import-commands.md](08-export-import-commands.md) - Portable identity

## SDK Integration Notes

The `client-go` SDK at `/home/vs/Desktop/dev/client-go` is not yet published. Use a `replace` directive in `go.mod`:

```go
replace github.com/vaultsandbox/client-go => /home/vs/Desktop/dev/client-go
```

### Key SDK APIs Used

| CLI Command | SDK Method(s) |
|-------------|---------------|
| `inbox create` | `client.CreateInbox()` |
| `inbox list` | Local keystore only |
| `inbox use` | Local keystore only |
| `watch` | `inbox.Watch()`, `client.WatchInboxes()` |
| `wait-for` | `inbox.WaitForEmail()` with options |
| `audit` | `inbox.GetEmail()`, `email.AuthResults` |
| `open` | `inbox.GetEmail()`, `email.Links` |
| `view` | `inbox.GetEmail()`, `email.HTML` |
| `export` | `inbox.Export()` |
| `import` | `client.ImportInbox()` |

## Configuration Files

### `~/.config/vsb/config.yaml`
```yaml
api_key: "vsb_xxxx..."
base_url: "https://api.vaultsandbox.com"
default_output: "pretty"  # pretty, json, minimal
```

### `~/.config/vsb/keystore.json`
```json
{
  "inboxes": [...],
  "active_inbox": "user@example.com"
}
```

## Command Summary

| Command | Description |
|---------|-------------|
| `vsb inbox create [label]` | Create new inbox with quantum-safe keys |
| `vsb inbox list` | List all stored inboxes |
| `vsb inbox use <email>` | Switch active inbox |
| `vsb watch [--all]` | Real-time email feed with TUI |
| `vsb wait-for [options]` | Block until matching email arrives |
| `vsb audit <id\|--latest>` | Deep-dive security analysis |
| `vsb open [--latest]` | Extract and open first link |
| `vsb view <id\|--latest>` | Preview HTML in browser |
| `vsb export [email] --out file` | Export inbox with keys |
| `vsb import <file>` | Import inbox from file |
