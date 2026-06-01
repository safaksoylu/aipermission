import { KeyRound, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useGateway } from "../lib/gateway-context";
import { apiDelete, apiPost } from "../lib/api";
import { useAsyncAction } from "../lib/use-async-action";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { CopyButton } from "../components/ui/copy-button";
import { Drawer } from "../components/ui/drawer";
import { Field, Input, Textarea } from "../components/ui/form";
import { Notice } from "../components/ui/notice";

const emptyForm = { name: "main", key_type: "ed25519" };

export function SSHKeysPage() {
  const { sshKeys, servers, loadSSHKeys } = useGateway();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const { actionState: state, runAction } = useAsyncAction();

  async function createKey(event) {
    event.preventDefault();
    await runAction({
      pending: "saving",
      successMessage: "SSH key created.",
      action: async () => {
        await apiPost("/api/ssh-keys", form);
        setForm(emptyForm);
        setDrawerOpen(false);
        await loadSSHKeys();
      },
    });
  }

  async function deleteKey(id) {
    await runAction({
      pending: "deleting",
      successMessage: "SSH key deleted.",
      action: async () => {
        await apiDelete(`/api/ssh-keys/${id}`);
        await loadSSHKeys();
      },
    });
  }

  return (
    <section className="mx-auto grid w-full max-w-6xl gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold">Gateway SSH keys</h3>
          <p className="text-sm text-stone-500">Create named keys and attach them to servers.</p>
        </div>
        <Button type="button" onClick={() => setDrawerOpen(true)}>
          <Plus className="h-4 w-4" />
          Add key
        </Button>
      </div>

      {state.message ? <Notice tone="good">{state.message}</Notice> : null}
      {state.state === "error" ? <Notice tone="bad">{state.error}</Notice> : null}
      {sshKeys.state === "error" ? <Notice tone="bad">{sshKeys.error}</Notice> : null}

      <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
        <table className="w-full table-fixed border-collapse text-left text-sm">
          <thead className="bg-stone-50 text-xs uppercase text-stone-500">
            <tr>
              <th className="w-[22%] px-4 py-3 font-semibold">Name</th>
              <th className="w-[13%] px-4 py-3 font-semibold">Type</th>
              <th className="w-[28%] px-4 py-3 font-semibold">Fingerprint</th>
              <th className="w-[22%] px-4 py-3 font-semibold">Servers</th>
              <th className="w-[15%] px-4 py-3 text-right font-semibold">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-stone-200">
            {sshKeys.data.map((key) => {
              const linkedServers = servers.data.filter((server) => Number(server.ssh_key_id) === Number(key.id));
              return (
                <tr className="align-top" key={key.id}>
                  <td className="px-4 py-4">
                    <div className="flex min-w-0 items-center gap-2">
                      <KeyRound className="h-4 w-4 shrink-0 text-emerald-900" />
                      <span className="truncate font-semibold">{key.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-4">
                    <Badge>{key.key_type}</Badge>
                  </td>
                  <td className="px-4 py-4">
                    <span className="block truncate font-mono text-xs text-stone-500">{key.fingerprint}</span>
                  </td>
                  <td className="px-4 py-4">
                    <div className="grid gap-1">
                      <Badge tone={linkedServers.length > 0 ? "good" : "neutral"} className="w-fit">
                        {linkedServers.length} server{linkedServers.length === 1 ? "" : "s"}
                      </Badge>
                      {linkedServers.length > 0 ? (
                        <span className="truncate text-xs text-stone-500">
                          {linkedServers.map((server) => server.name).join(", ")}
                        </span>
                      ) : null}
                    </div>
                  </td>
                  <td className="px-4 py-4">
                    <div className="flex justify-end gap-2">
                      <CopyButton
                        value={key.install_command}
                        variant="outline"
                        className="h-9 w-9 px-0"
                        title="Copy install command"
                      >
                        {null}
                      </CopyButton>
                      <Button
                        type="button"
                        variant="outline"
                        className="h-9 w-9 px-0"
                        title={linkedServers.length > 0 ? "Remove server links first" : "Delete key"}
                        onClick={() => deleteKey(key.id)}
                        disabled={linkedServers.length > 0 || state.state === "deleting"}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {sshKeys.state === "ready" && sshKeys.data.length === 0 ? (
          <div className="p-4">
            <Notice>Create your first gateway SSH key.</Notice>
          </div>
        ) : null}
      </div>

      <Drawer
        open={drawerOpen}
        title="Add SSH key"
        description="ed25519 is recommended. RSA is available for older systems."
        onClose={() => setDrawerOpen(false)}
      >
        <form className="grid gap-4" onSubmit={createKey}>
          <Field>
            Name
            <Input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
          </Field>
          <div className="grid grid-cols-2 gap-2 rounded-md bg-stone-100 p-1">
            {["ed25519", "rsa"].map((type) => (
              <button
                className={`rounded-md px-3 py-2 text-sm font-semibold transition ${
                  form.key_type === type ? "bg-white text-emerald-950 shadow-sm" : "text-stone-500 hover:text-stone-900"
                }`}
                key={type}
                type="button"
                onClick={() => setForm({ ...form, key_type: type })}
              >
                {type}
              </button>
            ))}
          </div>
          {state.state === "error" ? <Notice tone="bad">{state.error}</Notice> : null}
          <Button type="submit" disabled={state.state === "saving"}>
            <Plus className="h-4 w-4" />
            {state.state === "saving" ? "Creating..." : `Generate ${form.key_type} key`}
          </Button>
          <Notice>
            After creating the key, copy the install command from the table and paste it on the VPS.
          </Notice>
          <Field>
            Install command preview
            <Textarea
              readOnly
              rows={4}
              className="font-mono text-xs"
              value={`mkdir -p ~/.ssh && chmod 700 ~/.ssh && printf '%s\\n' 'ssh-${form.key_type} ... aipermission-${form.name || "key"}' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`}
            />
          </Field>
        </form>
      </Drawer>
    </section>
  );
}
