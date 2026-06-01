import { useEffect, useState } from "react";
import { LockKeyhole, Upload } from "lucide-react";
import { apiPost, apiPostForm } from "../lib/api";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/form";
import { Notice } from "../components/ui/notice";
import { isValidDatabasePassword } from "../lib/password";

export function UnlockPage({ status, onUnlocked }) {
  const databases = status?.databases || [];
  const firstDatabaseID = status?.database_id || databases[0]?.id || "default";
  const [selectedDatabaseID, setSelectedDatabaseID] = useState(firstDatabaseID);
  const selectedDatabase = databases.find((database) => database.id === selectedDatabaseID) || databases[0] || null;
  const hasDatabase = Boolean(selectedDatabase);
  const selectedUnsupported = selectedDatabase?.state === "unsupported_plaintext";
  const [activeTab, setActiveTab] = useState(hasDatabase ? "unlock" : "create");
  const [createForm, setCreateForm] = useState({ database_name: "", password: "", confirm_password: "" });
  const [unlockForm, setUnlockForm] = useState({ password: "" });
  const [importForm, setImportForm] = useState({ file: null, database_password: "" });
  const [createState, setCreateState] = useState({ state: "idle", error: null });
  const [unlockState, setUnlockState] = useState({ state: "idle", error: null });
  const [importState, setImportState] = useState({ state: "idle", error: null });

  useEffect(() => {
    const nextID = status?.database_id || databases[0]?.id || "default";
    setSelectedDatabaseID(nextID);
  }, [status?.database_id, databases.length]);

  useEffect(() => {
    if (!selectedDatabase) {
      setActiveTab("create");
      return;
    }
    setActiveTab("unlock");
  }, [selectedDatabase?.id, selectedDatabase?.state]);

  const tabs = [
    ...(hasDatabase ? [["unlock", "Unlock Database"]] : []),
    ["create", hasDatabase ? "New Database" : "Create Database"],
    ["import", "Import Database"],
  ];
  const createPasswordValid = isValidDatabasePassword(createForm.password);

  async function createDatabase(event) {
    event.preventDefault();
    setCreateState({ state: "saving", error: null });
    try {
      await apiPost("/api/unlock/setup", {
        password: createForm.password,
        confirm_password: createForm.confirm_password,
        database_name: createForm.database_name,
      });
      await onUnlocked();
    } catch (error) {
      setCreateState({ state: "error", error: error.message });
    }
  }

  async function unlockDatabase(event) {
    event.preventDefault();
    setUnlockState({ state: "unlocking", error: null });
    try {
      await apiPost("/api/unlock", { database_id: selectedDatabase?.id, password: unlockForm.password });
      await onUnlocked();
    } catch (error) {
      setUnlockState({ state: "error", error: error.message });
    }
  }

	async function importDatabase(event) {
		event.preventDefault();
		if (!importForm.file) {
      setImportState({ state: "error", error: "Database file is required" });
      return;
		}
		setImportState({ state: "importing", error: null });
		try {
			const formData = new FormData();
			formData.set("sqlite", importForm.file, importForm.file.name);
			formData.set("database_password", importForm.database_password);
			formData.set("database_name", createForm.database_name);
			await apiPostForm("/api/backup/import", formData);
			await onUnlocked();
		} catch (error) {
      setImportState({ state: "error", error: error.message });
    }
  }

  return (
    <UnlockShell title={hasDatabase ? "Select database" : "Database setup"}>
      {databases.length > 0 ? (
        <div className="grid gap-2">
          <label className="text-sm font-semibold text-stone-800">Database</label>
          <select
            className="h-10 rounded-md border border-stone-300 bg-white px-3 text-sm outline-none focus:border-emerald-800"
            value={selectedDatabase?.id || selectedDatabaseID}
            onChange={(event) => setSelectedDatabaseID(event.target.value)}
          >
            {databases.map((database) => (
              <option key={database.id} value={database.id}>
                {database.name} {database.state === "unsupported_plaintext" ? "(unsupported plaintext)" : ""}
              </option>
            ))}
          </select>
        </div>
      ) : null}
      {status?.state === "session_required" ? (
        <Notice tone="warn">Your browser session is missing or expired. Enter the database password to continue.</Notice>
      ) : null}
      {selectedUnsupported ? (
        <Notice tone="bad">This file is a plaintext SQLite database. AIPermission only supports SQLCipher-encrypted .aipdb databases.</Notice>
      ) : null}
      <div className="grid rounded-md border border-stone-200 bg-stone-100 p-1" style={{ gridTemplateColumns: `repeat(${tabs.length}, minmax(0, 1fr))` }}>
        {tabs.map(([value, label]) => (
          <button
            key={value}
            type="button"
            className={`h-10 whitespace-nowrap rounded px-2 text-xs font-semibold transition sm:text-sm ${
              activeTab === value ? "bg-white text-emerald-950 shadow-sm" : "text-stone-500 hover:text-stone-900"
            }`}
            onClick={() => setActiveTab(value)}
          >
            {label}
          </button>
        ))}
      </div>

      {activeTab === "unlock" ? (
        <form className="grid gap-4" onSubmit={unlockDatabase}>
          <div className="grid gap-2">
            <label className="text-sm font-semibold text-stone-800">Database password</label>
            <Input
              type="password"
              value={unlockForm.password}
              onChange={(event) => setUnlockForm((current) => ({ ...current, password: event.target.value }))}
              autoFocus
              required
            />
          </div>
          {unlockState.state === "error" ? <Notice tone="bad">{unlockState.error}</Notice> : null}
          <Button type="submit" disabled={unlockState.state === "unlocking" || selectedUnsupported}>
            <LockKeyhole className="h-4 w-4" />
            {unlockState.state === "unlocking" ? "Unlocking..." : "Unlock"}
          </Button>
        </form>
      ) : null}

      {activeTab === "create" ? (
        <form className="grid gap-4" onSubmit={createDatabase}>
          <div className="grid gap-2">
            <label className="text-sm font-semibold text-stone-800">Database name</label>
            <Input
              type="text"
              value={createForm.database_name}
              onChange={(event) => setCreateForm((current) => ({ ...current, database_name: event.target.value }))}
              placeholder={hasDatabase ? "Project name" : "Default"}
              required={hasDatabase}
            />
          </div>
          <div className="grid gap-2">
            <label className="text-sm font-semibold text-stone-800">Database password</label>
            <Input
              type="password"
              value={createForm.password}
              onChange={(event) => setCreateForm((current) => ({ ...current, password: event.target.value }))}
              minLength={14}
              autoFocus={!hasDatabase}
              required
            />
          </div>
          <div className="grid gap-2">
            <label className="text-sm font-semibold text-stone-800">Confirm password</label>
            <Input
              type="password"
              value={createForm.confirm_password}
              onChange={(event) => setCreateForm((current) => ({ ...current, confirm_password: event.target.value }))}
              minLength={14}
              required
            />
          </div>
          <Notice>Use at least 14 characters with uppercase letters, lowercase letters, and numbers. This password cannot be recovered.</Notice>
          {createState.state === "error" ? <Notice tone="bad">{createState.error}</Notice> : null}
          <Button type="submit" disabled={createState.state === "saving" || !createPasswordValid || createForm.password !== createForm.confirm_password}>
            <LockKeyhole className="h-4 w-4" />
            {createState.state === "saving" ? "Working..." : "Create encrypted database"}
          </Button>
        </form>
      ) : null}

      {activeTab === "import" ? (
        <form className="grid gap-4" onSubmit={importDatabase}>
          <div>
            <h2 className="text-sm font-semibold text-stone-900">Import encrypted database</h2>
            <p className="mt-1 text-sm text-stone-500">Choose an exported .aipdb or SQLCipher .db file, then enter that database password. Imports always create a new named database.</p>
          </div>
          <div className="grid gap-2">
            <label className="text-sm font-semibold text-stone-800">Database name</label>
            <Input
              type="text"
              value={createForm.database_name}
              onChange={(event) => setCreateForm((current) => ({ ...current, database_name: event.target.value }))}
              placeholder="Restored project"
              required
            />
          </div>
          <div className="grid gap-2">
            <label className="text-sm font-semibold text-stone-800">Database file</label>
            <Input
              type="file"
              accept=".aipdb,.db,application/octet-stream"
              onChange={(event) => setImportForm((current) => ({ ...current, file: event.target.files?.[0] || null }))}
              required
            />
          </div>
          <div className="grid gap-2">
            <label className="text-sm font-semibold text-stone-800">Database password</label>
            <Input
              type="password"
              value={importForm.database_password}
              onChange={(event) => setImportForm((current) => ({ ...current, database_password: event.target.value }))}
              required
            />
          </div>
          {importState.state === "error" ? <Notice tone="bad">{importState.error}</Notice> : null}
          <Button type="submit" variant="outline" disabled={importState.state === "importing"}>
            <Upload className="h-4 w-4" />
            {importState.state === "importing" ? "Importing..." : "Import database"}
          </Button>
        </form>
      ) : null}
    </UnlockShell>
  );
}

export function UnlockShell({ title, children }) {
  return (
    <main className="grid min-h-screen place-items-center bg-stone-100 p-5 text-stone-950">
      <section className="grid w-full max-w-2xl gap-5 rounded-lg border border-stone-200 bg-white p-6 shadow-xl">
        <div className="flex items-center gap-3">
          <img src="/icon.svg" alt="" className="h-10 w-10 rounded-lg" />
          <div>
            <h1 className="text-lg font-semibold">aipermission</h1>
            <p className="text-sm text-stone-500">{title}</p>
          </div>
        </div>
        <Notice tone="warn">
          Local-only gateway. Keep Docker ports bound to <span className="font-mono">127.0.0.1</span>; do not expose this UI or API on LAN or the public internet.
        </Notice>
        {children}
      </section>
    </main>
  );
}
