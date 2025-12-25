# VSB-CLI Commands Reference

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Config file path (default: `~/.config/vsb/config.yaml`) |
| `--api-key` | | API key (overrides config) |
| `--base-url` | | API base URL |
| `--output` | `-o` | Output format: `pretty`, `json`, `minimal` |

---

## Commands Overview

| Command | Description |
|---------|-------------|
| `vsb config` | Interactive API key setup |
| `vsb inbox` | Inbox management (create, list, use, delete) |
| `vsb watch` | Real-time TUI email monitoring |
| `vsb view` | View email content |
| `vsb audit` | Security analysis of email |
| `vsb open` | Extract and open links from email |
| `vsb wait-for` | Wait for email (CI/CD) |
| `vsb export` | Export inbox with private keys |
| `vsb import` | Import inbox from file |

---

## `vsb config`

Interactive configuration wizard.

**Usage:** `vsb config`

**Prompts for:**
- Server URL (default: `https://api.vaultsandbox.com`)
- API Key

**Flags:** None

---

## `vsb inbox`

Parent command for inbox management.

### `vsb inbox create [label]`

Create a new temporary inbox with quantum-safe encryption.

| Flag | Description | Default |
|------|-------------|---------|
| `--ttl` | Inbox lifetime (e.g., `1h`, `24h`, `7d`) | `24h` |

**Examples:**
```bash
vsb inbox create
vsb inbox create auth-tests
vsb inbox create --ttl 1h
```

### `vsb inbox list`

List all stored inboxes.

**Aliases:** `ls`

| Flag | Description | Default |
|------|-------------|---------|
| `--all` | Show expired inboxes too | `false` |

### `vsb inbox use <email>`

Switch active inbox.

**Arguments:** Email address (required)

**Flags:** None

### `vsb inbox delete <email>`

Delete an inbox from server and local keystore.

**Aliases:** `rm`

| Flag | Description | Default |
|------|-------------|---------|
| `--local-only` | Only remove from local keystore | `false` |

---

## `vsb watch`

Real-time TUI dashboard for incoming emails.

| Flag | Description | Default |
|------|-------------|---------|
| `--all` | Watch all stored inboxes | `false` |
| `--email` | Watch specific inbox by email | active inbox |

**Examples:**
```bash
vsb watch
vsb watch --all
vsb watch --email abc@vaultsandbox.com
```

---

## `vsb view [email-id]`

View email content in various formats.

| Flag | Description | Default |
|------|-------------|---------|
| `--text` | Show plain text in terminal | `false` |
| `--raw` | Show raw RFC 5322 source | `false` |
| `--email` | Use specific inbox | active inbox |

**Default behavior:** Opens HTML in browser

**Examples:**
```bash
vsb view              # Latest email HTML in browser
vsb view abc123       # Specific email
vsb view --text       # Plain text to terminal
vsb view --raw        # Raw email source
```

---

## `vsb audit [email-id]`

Deep security analysis of an email.

| Flag | Description | Default |
|------|-------------|---------|
| `--email` | Use specific inbox | active inbox |
| `--json` | Output as JSON | `false` |

**Analyzes:**
- Authentication: SPF, DKIM, DMARC
- Transport Security: TLS version, cipher suite
- MIME Structure: headers, body parts, attachments
- Security Score (0-100)

**Examples:**
```bash
vsb audit
vsb audit abc123
vsb audit --json
```

---

## `vsb open [email-id]`

Extract and open links from an email.

| Flag | Description | Default |
|------|-------------|---------|
| `--list` | List all links without opening | `false` |
| `--nth` | Open the Nth link (1-indexed) | `1` |
| `--email` | Use specific inbox | active inbox |
| `--json` | Output as JSON | `false` |

**Examples:**
```bash
vsb open              # Open first link from latest email
vsb open abc123       # From specific email
vsb open --list       # List all links
vsb open --nth 2      # Open second link
vsb open --json       # JSON output
```

---

## `vsb wait-for`

Wait for an email matching criteria. Designed for CI/CD pipelines.

### Filter Flags

| Flag | Description |
|------|-------------|
| `--subject` | Exact subject match |
| `--subject-regex` | Subject regex pattern |
| `--from` | Exact sender match |
| `--from-regex` | Sender regex pattern |

### Timing Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--timeout` | Maximum wait time | `60s` |
| `--count` | Number of emails to wait for | `1` |

### Output Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output as JSON | `false` |
| `--quiet` | No output, exit code only | `false` |
| `--extract-link` | Output first link from email | `false` |
| `--email` | Use specific inbox | active inbox |

**Exit codes:** `0` = success, `1` = timeout

**Examples:**
```bash
vsb wait-for
vsb wait-for --subject-regex "password reset" --timeout 30s
LINK=$(vsb wait-for --subject "Verify" --extract-link)
vsb wait-for --from "noreply@example.com" --json | jq .subject
```

---

## `vsb export [email-address]`

Export inbox with private keys for backup or sharing.

| Flag | Description | Default |
|------|-------------|---------|
| `--out` | Output file path | `<email>.json` |

**Warning:** Exported file contains PRIVATE KEY. Handle securely!

**Examples:**
```bash
vsb export
vsb export abc@vsb.com
vsb export --out ~/backup.json
```

---

## `vsb import <file>`

Import inbox from export file.

| Flag | Description | Default |
|------|-------------|---------|
| `--local` | Skip server verification | `false` |
| `--label` | Override label for imported inbox | from file |
| `--force` | Overwrite existing inbox | `false` |

**Examples:**
```bash
vsb import backup.json
vsb import backup.json --local
vsb import backup.json --label "shared-inbox"
vsb import backup.json --force
```

---

## TUI Keyboard Shortcuts (watch command)

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate email list |
| `Enter` | Select email / expand |
| `Tab` | Switch tabs (content, security, links, raw) |
| `v` | View HTML in browser |
| `o` | Open first link |
| `q` | Quit |

---

## Configuration Priority

1. Command-line flags (highest)
2. Environment variables (`VSB_API_KEY`, `VSB_BASE_URL`)
3. Config file (`~/.config/vsb/config.yaml`)
4. Defaults (lowest)

## File Locations

| File | Purpose |
|------|---------|
| `~/.config/vsb/config.yaml` | API configuration |
| `~/.config/vsb/keystore.json` | Stored inboxes with crypto keys |
