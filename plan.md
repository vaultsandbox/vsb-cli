# VSB-CLI Improvement Plan

## Flag Consistency Issues

### `--json` flag missing on some commands
| Command | Has `--json` | Notes |
|---------|--------------|-------|
| `inbox list` | No | Would help CI/CD scripting |
| `view` | No | Could output email as JSON |
| `audit` | Yes | |
| `open` | Yes | |
| `wait-for` | Yes | |

### `--local` naming inconsistency
- `inbox delete --local-only`
- `import --local`

Should both be `--local`?

---

## Missing Features

### Non-interactive config
Currently `vsb config` is interactive only. Could add:
```bash
vsb config set api-key <key>
vsb config set base-url <url>
vsb config show
```

### Inbox management gaps
- `inbox info <email>` - Show details of a single inbox

---

## Naming Considerations

### `wait-for` hyphenation
- `wait-for` has a hyphen, all others don't
- Alternatives: `waitfor`, `await`, `poll`
- Keep as-is? It's readable and matches common CLI patterns (e.g., `kubectl wait`)

lets use wait
---

## Shorthand Flags

### Missing short flags
| Flag | Suggested Short |
|------|-----------------|
| `--quiet` | `-q` |
| `--json` | `-j` |
| `--text` | `-t` |
| `--raw` | `-r` |
| `--all` | `-a` |
| `--force` | `-f` |
| `--list` | `-l` |

---

## Possible Removals

### Flags to consider removing
- `--output` / `-o` global flag - is it actually used anywhere?
- `--base-url` global flag - rarely needed, config file is enough?

agree remove the flag for base-ur and apikeys ... use config only
---

## User Ideas

### Global `-o/--output` flag (Option A)
- Implement `-o json` globally instead of per-command `--json` flags
- Standard pattern used by kubectl, aws, gcloud, az
- Formats: `pretty` (default), `json`
- Remove individual `--json` flags from: `audit`, `open`, `wait-for`
- Add support to all commands: `inbox create`, `inbox list`, `view`, etc.

### `vsb` opens TUI directly (no subcommand)
- Like `k9s`, `lazygit`, `htop` - base command opens the UI
- Remove `watch` command
- `vsb` → opens TUI
- `vsb --all` → watch all inboxes
- `vsb --email abc@vsb.com` → watch specific inbox
- Help still available via `vsb --help` or `vsb help`

### Rename `open` → `link`
- `vsb link` → list links (default)
- `vsb link --open` → open first link in browser
- `vsb link --open 2` → open second link
- `vsb link abc123` → links from specific email
- Clearer: command is about links, `--open` is the action

### Non-interactive config for CI/CD
- `vsb config set api-key <key>` → set API key
- `vsb config set base-url <url>` → set base URL
- `vsb config show` → display current config
- Keep `VSB_API_KEY` env var support (remove only CLI flags)

### Partial matching for inbox selection
- If user types `34g` and only one inbox matches, use it
- Works for: `inbox delete`, `inbox use`, `--email` flag
- Example: `vsb inbox delete 34g` → deletes `34gabc@vaultsandbox.com`
- If multiple matches, show error with options
- Makes CLI faster to use


---

## Priority

| Priority | Change |
|----------|--------|
| High | |
| Medium | |
| Low | |
