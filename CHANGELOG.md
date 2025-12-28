# Changelog

All notable changes to this project will be documented in this file.

The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.5.0] - 2025-12-28

### Initial release

- Interactive TUI dashboard for real-time email monitoring across multiple inboxes
- Inbox management commands: `inbox create`, `inbox list`, `inbox info`, `inbox use`, `inbox delete`
- Email commands: `email list`, `email view`, `email audit`, `email url`, `email delete`
- `wait` command for polling emails in CI/CD pipelines
- Quantum-safe encryption via ML-KEM-768 (keys stored locally in keystore)
- Real-time email streaming via SSE with local decryption
- SPF/DKIM/DMARC authentication display in Security tab
- Link and attachment extraction with browser preview
- Inbox import/export functionality for backup and portability
- Configuration via YAML file, environment variables (`VSB_*`), or CLI flags
