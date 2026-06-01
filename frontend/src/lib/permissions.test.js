import assert from "node:assert/strict";
import test from "node:test";

import { maskedToken, permissionsToMap, ruleLabel } from "./permissions.js";

test("permissionsToMap indexes execution rules by server id", () => {
  assert.deepEqual(
    permissionsToMap([
      { server_id: 4, execution_rule: "always_run" },
      { server_id: 7, execution_rule: "approval_required" },
    ]),
    {
      4: "always_run",
      7: "approval_required",
    }
  );
});

test("token and rule helpers produce compact labels", () => {
  assert.equal(maskedToken("aip_1234567890abcdef"), "aip_1234...abcdef");
  assert.equal(maskedToken(""), "legacy token unavailable");
  assert.equal(ruleLabel("approval_required"), "prompt");
});
