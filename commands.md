# VSB CLI Commands Reference

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Config file path (default: `~/.config/vsb/config.yaml`) |
| `--output` | `-o` | Output format: `pretty`, `json` |

---

## Inbox Management

### `vsb inbox create`

Create a new temporary inbox with quantum-safe encryption.

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--ttl` | `24h` | Inbox lifetime (e.g., `1h`, `24h`, `7d`) |

**Examples:**
```bash
vsb inbox create
vsb inbox create --ttl 1h
vsb inbox create --ttl 7d
```

---

### `vsb inbox list`

List all stored inboxes. Alias: `ls`

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--all` | `-a` | Show expired inboxes too |

**Examples:**
```bash
vsb inbox list
vsb inbox ls -a
```

---

### `vsb inbox use <email>`

Switch active inbox. Supports partial matching.

**Examples:**
```bash
vsb inbox use abc123@vaultsandbox.com
vsb inbox use abc     # Partial match
```

---

### `vsb inbox delete <email>`

Delete an inbox. Alias: `rm`. Supports partial matching.

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--local` | `-l` | Only remove from local keystore, don't delete on server |

**Examples:**
```bash
vsb inbox delete abc123@vaultsandbox.com
vsb inbox delete abc       # Partial match
vsb inbox delete abc -l    # Local only
```

---

### `vsb inbox info [email]`

Show detailed information about an inbox.

**Examples:**
```bash
vsb inbox info           # Info for active inbox
vsb inbox info abc       # Info for inbox matching 'abc'
vsb inbox info -o json   # JSON output
```

---

## Email Operations

### `vsb list`

List all emails in the active inbox. Alias: `ls`

**Flags:**
| Flag | Description |
|------|-------------|
| `--email` | Use specific inbox (default: active) |

**Examples:**
```bash
vsb list              # List emails in active inbox
vsb list --email abc  # List emails in specific inbox
vsb list -o json      # JSON output
```

---

### `vsb view [email-id]`

Preview email content. Defaults to latest email.

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--text` | `-t` | Show plain text version in terminal |
| `--raw` | `-r` | Show raw email source (RFC 5322) |
| `--email` | | Use specific inbox (default: active) |

**Examples:**
```bash
vsb view              # View latest email HTML in browser
vsb view abc123       # View specific email
vsb view -t           # Print plain text to terminal
vsb view -r           # Print raw email source
vsb view -o json      # JSON output
```

---

### `vsb audit [email-id]`

Deep-dive security analysis of an email. Displays SPF, DKIM, DMARC validation, TLS info, and MIME structure.

**Flags:**
| Flag | Description |
|------|-------------|
| `--email` | Use specific inbox (default: active) |

**Examples:**
```bash
vsb audit              # Audit most recent email
vsb audit abc123       # Audit specific email
vsb audit -o json      # JSON output for scripting
```

---

### `vsb link [email-id]`

Extract and optionally open links from an email.

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--open` | `-O` | Open the Nth link in browser (1=first, 0=don't open) |
| `--email` | | Use specific inbox (default: active) |

**Examples:**
```bash
vsb link              # List links from latest email
vsb link abc123       # List links from specific email
vsb link --open 1     # Open first link in browser
vsb link --open 2     # Open second link in browser
vsb link -o json      # JSON output for CI/CD
```

---

## CI/CD Integration

### `vsb wait`

Block until an email matching criteria arrives. Exit code 0 on match, 1 on timeout.

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--email` | | Watch specific inbox (default: active) |
| `--subject` | | Exact subject match |
| `--subject-regex` | | Subject regex pattern |
| `--from` | | Exact sender match |
| `--from-regex` | | Sender regex pattern |
| `--timeout` | `60s` | Maximum time to wait |
| `--count` | `1` | Number of matching emails to wait for |
| `--quiet` | | No output, exit code only |
| `--extract-link` | | Output first link from email |

**Examples:**
```bash
vsb wait
vsb wait --subject-regex "password reset" --timeout 30s
LINK=$(vsb wait --subject "Verify" --extract-link)
vsb wait --from "noreply@example.com" -o json | jq .subject
```

---

## Backup & Restore

### `vsb export [email-address]`

Export inbox with private keys to a JSON file.

**Flags:**
| Flag | Description |
|------|-------------|
| `--out` | Output file path (default: `<email>.json`) |

**Examples:**
```bash
vsb export                     # Export active inbox
vsb export abc@vsb.com         # Export specific inbox
vsb export --out ~/backup.json # Specify output file
```

---

### `vsb import <file>`

Import inbox from export file.

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--local` | `-l` | Skip server verification |
| `--force` | `-f` | Overwrite existing inbox with same email |

**Examples:**
```bash
vsb import backup.json      # Import and verify
vsb import backup.json -l   # Skip server verification
vsb import backup.json -f   # Force overwrite existing
```

---

## Configuration

### `vsb config`

Interactive configuration wizard.

---

### `vsb config show`

Show current configuration (API key masked).

---

### `vsb config set <key> <value>`

Set a configuration value.

**Available keys:**
- `api-key` - Your VaultSandbox API key
- `base-url` - API server URL (default: `https://api.vaultsandbox.com`)

**Examples:**
```bash
vsb config set api-key vsb_abc123
vsb config set base-url https://api.vaultsandbox.com
```
