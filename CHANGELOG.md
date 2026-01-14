# Changelog

All notable changes to this project will be documented in this file.

The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.7.0] - 2026-01-13

### Added

- `--email-auth` flag for `inbox create` to enable/disable SPF/DKIM/DMARC/PTR checks per inbox
- `--encryption` flag for `inbox create` to request encrypted or plain inboxes (when server policy allows)
- Support for `skipped` status in email authentication results display
- Reverse DNS (`reverseDns`) field in JSON output for `email audit --json`

### Changed

- Updated `client-go` to v0.7.0
- **Breaking:** ReverseDNS authentication result now uses `Result` field (`pass`/`fail`/`none`/`skipped`) instead of `Verified` boolean

## [0.6.1] - 2026-01-11

### Changed

- Made strategy configurable (SSE or polling)

### Fixed

- Fixed path traversal vulnerability in CleanupPreviews

## [0.6.0] - 2026-01-04

### Changed

- Updated `client-go` to v0.6.0 (export format: `SecretKeyB64` → `SecretKey`, public key now derived from secret key per spec Section 4.2)

## [0.5.1] - 2026-01-01

### Changed

- Updated `client-go` to v0.5.1 (email auth field renames: `Status` → `Result`)

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
