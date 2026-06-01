# MVP Scope

The first public MVP should stay small, usable, and honest about its boundaries.

Main target:

> After `docker compose up`, a user can create SSH keys, add servers, create API tokens, grant token/server permissions, connect an MCP client, call `list_servers`, and run commands through `exec`. Commands either run directly or wait for approval according to their execution rule.

## Product Boundary

The MVP is not a DevOps platform. It does not own production operations. It gives a developer a controlled, token-scoped, auditable execution channel for debugging, maintenance, incident triage, and temporary AI-assisted automation.

The MVP does not introduce first-class management modules for every external system. If an allowed server has the needed CLI tools and access, the AI can operate at command level through `exec`.

The gateway itself is local-only. It is not designed to run on a remote server for browser/MCP clients, to be shared on a LAN, or to act as a central team control plane.

## Included

- local Docker runtime
- React web dashboard
- Go backend
- SQLCipher SQLite storage
- encrypted credential vault
- named local databases
- backup/import with `.aipdb`
- SSH key management
- gateway-generated `ed25519` and `rsa` keypairs
- public key install command
- server management
- persistent console sessions
- API token creation and revocation
- token-to-server permissions
- execution rules: `always_run`, `approval_required`, `blocked`
- MCP `list_servers`
- MCP `exec`
- approval dialog and request polling
- Run / Decline / note
- live message queue
- command request history
- audit event storage
- operator instructions for AI clients

## Deferred

- advanced command risk analysis
- PostgreSQL query tools
- SQL parser and masking
- multi-server batch execution
- manual terminal command event parsing for structured history

For detailed sequencing, see [Implementation Roadmap](implementation-roadmap.md) and [Roadmap](../ROADMAP.md).
