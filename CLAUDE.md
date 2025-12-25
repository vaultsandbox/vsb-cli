# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VSB-CLI is a command-line interface for VaultSandbox, a developer tool for testing email flows with quantum-safe end-to-end encryption (ML-KEM-768, ML-DSA-65). Features include temporary disposable email inboxes, real-time email monitoring via TUI, security auditing, and CI/CD-friendly structured output.

## Build and Run

```bash
# Build the binary
go build -o vsb ./cmd/vsb

# Run
./vsb
```

## Architecture

The project follows a modular architecture with clear separation of concerns:

```
cmd/vsb/main.go          # Entry point, delegates to cli.Execute()

internal/
├── cli/                 # Cobra command handlers
│   ├── root.go          # Root command, global flags, config init
│   ├── helpers.go       # Shared functions (LoadKeystoreOrError, GetInbox, etc.)
│   ├── inbox*.go        # Inbox management (create, list, use, delete)
│   ├── watch.go         # Real-time TUI email watching
│   ├── view.go          # Email viewing (HTML/text/raw)
│   ├── audit.go         # Security auditing
│   ├── waitfor.go       # CI/CD email waiting with filters
│   └── import.go/export.go  # Inbox backup and restore
│
├── config/              # Configuration and persistence
│   ├── config.go        # Config struct, priority: flags > env > file > defaults
│   ├── keystore.go      # Persistent inbox storage (JSON with RWMutex)
│   └── client.go        # VaultSandbox SDK client creation
│
├── tui/watch/           # Bubble Tea TUI for real-time monitoring
│   ├── model.go         # Main state machine (list + detail view)
│   ├── security.go      # Security audit tab rendering
│   ├── links.go         # Link list tab rendering
│   └── raw.go           # Raw headers tab rendering
│
├── styles/              # Lipgloss visual styling (colors, components)
├── output/              # Message formatting helpers (success/error/info)
└── browser/             # URL opening and HTML temp file viewing
```

## Key Patterns

**Adding a new command:**
1. Create `internal/cli/mycommand.go`
2. Use shared helpers from `helpers.go`: `LoadKeystoreOrError()`, `GetInbox()`, `LoadAndImportInbox()`, `GetEmailByIDOrLatest()`
3. Register command in `init()` to parent (rootCmd or inboxCmd)

**Configuration priority:** flags > env vars (VSB_*) > config file (~/.config/vsb/config.yaml) > defaults

**Persistence:**
- Config: `~/.config/vsb/config.yaml` (YAML)
- Keystore: `~/.config/vsb/keystore.json` (JSON, stores inboxes with crypto keys)

**Data flow pattern:**
```
command → config.LoadKeystore() → config.NewClient() → LoadAndImportInbox() → SDK operation → output/TUI
```

**TUI (watch command):**
- Bubble Tea framework with dual-pane UI
- SSE streaming for real-time emails + batch loading existing
- Tabs: content, security audit, links, raw headers

## SDK Dependency

Uses `vaultsandbox/client-go` SDK. Local development path: `/home/vs/Desktop/dev/client-go`

## Command Structure

```
vsb
├── inbox create/list/use/delete   # Inbox management
├── watch [--all] [--email]        # Real-time TUI
├── view [id] [--text|--raw]       # View email content
├── audit [id] [--json]            # Security analysis
├── open [id] [--list|--nth]       # Extract/open links
├── wait-for [filters]             # CI/CD email waiting
├── import/export                  # Backup and restore
└── config                         # Interactive API setup
```
