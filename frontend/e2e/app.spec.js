import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  let unlocked = false;
  let tokenPermissions = [];
  let mcpRuntimeEnabled = false;
  await page.route("http://localhost:8080/api/unlock/status", async (route) => {
    await route.fulfill({
      json: unlocked
        ? unlockedStatus()
        : {
            state: "session_required",
            database_id: "default",
            databases: [{ id: "default", name: "Default", state: "locked" }],
          },
    });
  });
  await page.route("http://localhost:8080/api/unlock", async (route) => {
    unlocked = true;
    await route.fulfill({
      headers: {
        "set-cookie": "aipermission_ui_session=test; Path=/; SameSite=Strict",
      },
      json: { state: "unlocked", database_id: "default", database_name: "Default" },
    });
  });
  await page.route("http://localhost:8080/api/backup/import", async (route) => {
    unlocked = true;
    await route.fulfill({ json: { state: "unlocked", database_id: "imported", database_name: "Imported project" } });
  });
  await page.route("http://localhost:8080/api/status", async (route) => {
    await route.fulfill({ json: { service: "aipermission", status: "running", config: {}, features: [] } });
  });
  await page.route("http://localhost:8080/api/servers", async (route) => {
    await route.fulfill({ json: [{ id: 1, name: "worker-1", host: "127.0.0.1", port: 22, username: "root" }] });
  });
  await page.route("http://localhost:8080/api/ssh-keys", async (route) => {
    await route.fulfill({ json: [] });
  });
  await page.route("http://localhost:8080/api/tokens", async (route) => {
    await route.fulfill({ json: [{ id: 1, name: "agent", token_prefix: "aip_test", created_at: "2026-05-31T00:00:00Z" }] });
  });
  await page.route("http://localhost:8080/api/console/sessions", async (route) => {
    await route.fulfill({ json: [] });
  });
  await page.route("http://localhost:8080/api/approvals", async (route) => {
    await route.fulfill({ json: [{ id: 7, server_id: 1, server_name: "worker-1", command: "docker ps", reason: "inspect", status: "pending_approval", created_at: "2026-05-31T00:00:00Z" }] });
  });
  await page.route("http://localhost:8080/api/messages", async (route) => {
    await route.fulfill({ json: [] });
  });
  await page.route("http://localhost:8080/api/settings/security", async (route) => {
    if (route.request().method() === "PUT") {
      await route.fulfill({ json: { reusable_tokens: false, expose_mcp_server_metadata: true, mcp_start_enabled: false, redaction_mode: "basic" } });
      return;
    }
    await route.fulfill({ json: { reusable_tokens: false, expose_mcp_server_metadata: false, mcp_start_enabled: false, redaction_mode: "basic" } });
  });
  await page.route("http://localhost:8080/api/settings/mcp-runtime", async (route) => {
    if (route.request().method() === "PUT") {
      mcpRuntimeEnabled = Boolean(route.request().postDataJSON().enabled);
    }
    await route.fulfill({ json: { enabled: mcpRuntimeEnabled, start_enabled: false, updated_at: "2026-05-31T00:00:00Z" } });
  });
  await page.route("http://localhost:8080/api/settings/redaction-rules", async (route) => {
    await route.fulfill({ json: [] });
  });
  await page.route("http://localhost:8080/api/settings/retention", async (route) => {
    if (route.request().method() === "PUT") {
      await route.fulfill({ json: { history_days: 14, audit_days: 14, console_days: 7, message_days: 7 } });
      return;
    }
    await route.fulfill({ json: { history_days: 0, audit_days: 0, console_days: 0, message_days: 0 } });
  });
  await page.route("http://localhost:8080/api/tokens/1/permissions", async (route) => {
    if (route.request().method() === "PUT") {
      const body = route.request().postDataJSON();
      tokenPermissions = body.permissions || [];
      await route.fulfill({ json: tokenPermissions });
      return;
    }
    await route.fulfill({ json: tokenPermissions });
  });
});

test("unlocks the local UI session and renders the dashboard", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("Your browser session is missing or expired.")).toBeVisible();
  await page.getByRole("textbox").fill("local-password");
  await page.getByRole("button", { name: "Unlock", exact: true }).click();

  await expect(page.locator('aside a[href="/console"]')).toBeVisible();
  await expect(page.getByRole("complementary").getByText("Gateway", { exact: true })).toBeVisible();
  await expect(page.getByRole("complementary").getByText("running", { exact: true })).toBeVisible();
  await expect(page.getByRole("complementary").getByText("Stopped", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Start MCP" }).click();
  await expect(page.getByRole("button", { name: "Stop MCP" })).toBeVisible();
});

test("renders security settings and updates MCP metadata exposure", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("textbox").fill("local-password");
  await page.getByRole("button", { name: "Unlock", exact: true }).click();
  await page.getByRole("link", { name: /Security/ }).click();

  await expect(page.getByRole("heading", { name: "Security" })).toBeVisible();
  await expect(page.getByText("MCP server lists hide host, port, and username by default.")).toBeVisible();
  await page.getByLabel("Expose endpoint metadata to MCP").click();
  await expect(page.getByText("MCP list_servers now includes host, port, and username metadata.")).toBeVisible();
});

test("imports a database from the unlock screen", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Import Database" }).click();
  await page.getByPlaceholder("Restored project").fill("Imported project");
  await page.locator('input[type="file"]').setInputFiles({
    name: "imported.aipdb",
    mimeType: "application/octet-stream",
    buffer: Buffer.from("encrypted-test-fixture"),
  });
  await page.locator('input[type="password"]').fill("ImportedPassword123");
  await page.locator('form button[type="submit"]').click();

  await expect(page.locator('aside a[href="/console"]')).toBeVisible();
});

test("renders settings retention controls", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("textbox").fill("local-password");
  await page.getByRole("button", { name: "Unlock", exact: true }).click();
  await page.getByRole("link", { name: /Settings/ }).click();

  await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Backup" })).toBeVisible();
  await page.getByLabel("Command history days").fill("14");
  await page.getByLabel("Audit log days").fill("14");
  await page.getByRole("button", { name: "Save retention" }).click();
  await expect(page.getByText("Retention settings saved and cleanup ran.")).toBeVisible();
});

test("updates token server permission from the Tokens page", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("textbox").fill("local-password");
  await page.getByRole("button", { name: "Unlock", exact: true }).click();
  await page.locator('aside a[href="/tokens"]').click();

  await page.getByTitle("worker-1: Disabled").click();
  await expect(page.getByRole("dialog", { name: "agent -> worker-1" })).toBeVisible();
  await page.getByRole("button", { name: /Prompt/ }).click();
  await expect(page.getByTitle("worker-1: Prompt")).toBeVisible();
});

function unlockedStatus() {
  return {
    state: "unlocked",
    database_id: "default",
    database_name: "Default",
    unlocked_databases: [{ id: "default", name: "Default", current: true }],
    databases: [{ id: "default", name: "Default", state: "unlocked" }],
  };
}
