import { Dialog } from "../ui/dialog";

export function PermissionDialog({ value, permissionState, onClose, onSetRule }) {
  const token = value?.token;
  const server = value?.server;
  const rule = token && server ? permissionState.data[token.id]?.[server.id] || "" : "";
  const saving = token && server ? permissionState.savingKey === `${token.id}:${server.id}` : false;

  return (
    <Dialog
      open={Boolean(value)}
      title={token && server ? `${token.name} -> ${server.name}` : "Permission"}
      description="Choose how this token can use this server."
      onClose={onClose}
    >
      {token && server ? (
        <div className="grid gap-3">
          <PermissionChoice
            title="Always run"
            description="Commands can run without approval."
            active={rule === "always_run"}
            disabled={saving}
            onClick={() => onSetRule(token, server, "always_run")}
          />
          <PermissionChoice
            title="Prompt"
            description="Commands must wait for approval."
            active={rule === "approval_required"}
            disabled={saving}
            onClick={() => onSetRule(token, server, "approval_required")}
          />
          <PermissionChoice
            title="Disabled"
            description="This token cannot see or use this server."
            active={!rule}
            disabled={saving}
            onClick={() => onSetRule(token, server, "")}
          />
        </div>
      ) : null}
    </Dialog>
  );
}

function PermissionChoice({ title, description, active, ...props }) {
  return (
    <button
      type="button"
      className={`rounded-lg border p-3 text-left transition disabled:pointer-events-none disabled:opacity-50 ${
        active ? "border-emerald-900 bg-emerald-50" : "border-stone-200 bg-white hover:bg-stone-50"
      }`}
      {...props}
    >
      <span className="block text-sm font-semibold text-stone-950">{title}</span>
      <span className="mt-1 block text-xs text-stone-500">{description}</span>
    </button>
  );
}
