export const appVersion = "0.1.0-rc.1";

export const changelogEntries = [
  {
    version: "0.1.0-rc.1",
    label: "Unreleased RC",
    sections: [
      {
        title: "Added",
        items: [
          "Local-only Docker gateway with React UI on http://localhost:3210.",
          "SQLCipher-encrypted named databases with unlock, switch, import, backup, rename, delete, and password-change flows.",
          "Gateway-owned SSH keys, SSH host fingerprint approval, token-scoped MCP execution, approvals, console sessions, history, and audit logs.",
        ],
      },
      {
        title: "Security",
        items: [
          "Private SSH keys and reusable token values stay inside the local encrypted gateway.",
          "API tokens are stored as hashes and shown once by default.",
          "Approval-required raw commands are encrypted separately so display redaction cannot mutate execution payloads.",
        ],
      },
    ],
  },
];
