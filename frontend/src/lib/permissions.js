export function permissionsToMap(permissions) {
  return Object.fromEntries(permissions.map((permission) => [permission.server_id, permission.execution_rule]));
}

export function ruleLabel(rule) {
  if (rule === "always_run") return "always run";
  if (rule === "approval_required") return "prompt";
  return "disabled";
}

export function ruleDotClass(rule) {
  if (rule === "always_run") return "bg-emerald-500";
  if (rule === "approval_required") return "bg-amber-400";
  return "bg-red-500";
}

export function permissionCardClass(rule) {
  if (rule === "always_run") return "border-emerald-100 bg-emerald-50/45 permission-card-good";
  if (rule === "approval_required") return "border-amber-100 bg-amber-50/45 permission-card-warn";
  return "border-red-100 bg-red-50/35 permission-card-bad";
}

export function maskedToken(value) {
  if (!value) return "legacy token unavailable";
  if (value.length <= 14) return value;
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}
