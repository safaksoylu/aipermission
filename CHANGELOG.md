# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses semantic versioning once public releases begin.

## [Unreleased]

## [0.1.0-rc.1] - Unreleased

### Added

- Local-only Docker gateway with React UI on `http://localhost:3210`.
- SQLCipher-encrypted named `.aipdb` databases with unlock, switch, import, backup, rename, delete, and password-change flows.
- Gateway-owned SSH key generation, public-key install commands, SSH host fingerprint approval, and server records.
- Token-scoped MCP command execution with `always_run`, `approval_required`, and `blocked` rules.
- Persistent web console sessions, live output streaming, approval dialogs, messages, command history, and audit logs.
- `@aipermission/mcp` package with init and operator-skill installer for common AI clients.
- Security controls for local-only HTTP boundaries, UI session cookies, CSRF, redaction rules, reusable-token opt-in, and supply-chain checks.

### Security

- SSH private keys and reusable token values stay inside the local encrypted gateway.
- API tokens are stored as hashes and shown once by default.
- Approval-required raw commands are encrypted separately so display redaction cannot mutate execution payloads.
- `read_console` requires `always_run` permission to avoid exposing unrelated manual transcripts to approval-only tokens.

### Notes

- This RC is a local developer gateway, not a remote DevOps platform.
- Do not expose the UI/API on a LAN or the public internet.
