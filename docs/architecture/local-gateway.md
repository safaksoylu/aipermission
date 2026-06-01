# Local Gateway

`aipermission` is a local-first developer gateway.

The user runs it on their own machine with Docker. The gateway owns the web UI, HTTP API, MCP bridge behavior, SQLite storage, encrypted vault, approval flow, and audit/history records.

This is a product design decision, not only a default setting. The gateway is not intended to be installed on a remote VPS, shared across a LAN, or exposed as a team DevOps service. Remote servers are only SSH targets reached from the local gateway.

## Runtime Shape

```txt
Developer machine
  Docker Compose
    frontend      -> http://localhost:3210
    web API proxy -> http://localhost:3210/api
    backend       -> loopback-only inside the local container namespace
    sqlite        -> Docker volume / data path
```

No agent is installed on remote servers. The gateway decrypts credentials locally and opens SSH connections only when an approved or permitted action needs them.

## Responsibilities

The gateway is responsible for:

- server records
- credential encryption and decryption
- API token creation and revocation
- token-to-server permissions
- execution rule checks
- MCP `list_servers` and `exec` requests
- pending approval flow
- live message queue
- command request history
- audit events

## Product Boundary

This is not a DevOps platform or a permanent production control plane.

Unsupported gateway shapes:

- `0.0.0.0` bind for browser or MCP access
- LAN-shared gateway for multiple machines
- public internet gateway
- central hosted team gateway
- reverse-proxying the unlocked UI/API for remote users

If an allowed server already has the needed CLI tools, config files, and network access, an AI agent can use those through the gateway `exec` surface. The gateway does not give operational knowledge or credentials to the AI. It runs AI-proposed commands inside local permission, approval, and audit boundaries.

When the work is done, the user can:

- revoke the token
- remove server permissions
- lock the database
- stop Docker

For the detailed MCP path, see [MCP Permission Flow](mcp-permission-flow.md).
