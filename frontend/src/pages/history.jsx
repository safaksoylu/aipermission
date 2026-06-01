import { RefreshCcw, Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { apiGet } from "../lib/api";
import { useGateway } from "../lib/gateway-context";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { CopyButton } from "../components/ui/copy-button";
import { Dialog } from "../components/ui/dialog";
import { Input, Select } from "../components/ui/form";
import { Notice } from "../components/ui/notice";
import { PaginationBar } from "../components/ui/pagination-bar";

const statusOptions = [
  { value: "", label: "All statuses" },
  { value: "pending_approval", label: "Pending" },
  { value: "running", label: "Running" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
  { value: "declined", label: "Declined" },
  { value: "error", label: "Error" },
];

export function HistoryPage() {
  const { servers, approvals, loadApprovals } = useGateway();
  const [filters, setFilters] = useState({ query: "", status: "", serverID: "" });
  const [state, setState] = useState({
    state: "idle",
    data: [],
    total: 0,
    limit: 50,
    offset: 0,
    next_offset: null,
    error: null,
  });
  const [selected, setSelected] = useState(null);

  const stats = useMemo(() => {
    const data = state.data;
    return {
      total: state.total,
      shown: data.length,
      pending: data.filter((item) => item.status === "pending_approval").length,
      failed: data.filter((item) => item.status === "failed" || item.status === "error").length,
    };
  }, [state.data, state.total]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadHistory(0);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [filters.query, filters.status, filters.serverID]);

  async function loadHistory(offset = state.offset) {
    setState((current) => ({ ...current, state: "loading", error: null }));
    const params = new URLSearchParams({
      paginated: "true",
      limit: String(state.limit),
      offset: String(Math.max(0, offset)),
    });
    if (filters.query.trim()) params.set("q", filters.query.trim());
    if (filters.status) params.set("status", filters.status);
    if (filters.serverID) params.set("server_id", filters.serverID);
    try {
      const data = await apiGet(`/api/approvals?${params.toString()}`);
      setState({
        state: "ready",
        data: data.items || [],
        total: data.total || 0,
        limit: data.limit || state.limit,
        offset: data.offset || 0,
        next_offset: data.next_offset ?? null,
        error: null,
      });
      await loadApprovals();
    } catch (error) {
      setState((current) => ({ ...current, state: "error", data: [], total: 0, error: error.message }));
    }
  }

  async function openHistoryItem(item) {
    setSelected(item);
    try {
      const detail = await apiGet(`/api/approvals/${item.id}`);
      setSelected(detail);
    } catch {
      setSelected(item);
    }
  }

  const pageStart = state.total === 0 ? 0 : state.offset + 1;
  const pageEnd = Math.min(state.offset + state.data.length, state.total);

  return (
    <section className="mx-auto grid w-full max-w-7xl gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold">History</h3>
          <p className="text-sm text-stone-500">Review MCP and approval command requests executed through the gateway.</p>
        </div>
        <Button type="button" variant="outline" onClick={() => loadHistory(state.offset)} disabled={state.state === "loading"}>
          <RefreshCcw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        <HistoryStat label="Total" value={stats.total} />
        <HistoryStat label="Shown" value={stats.shown} />
        <HistoryStat label="Pending" value={stats.pending} tone="warn" />
        <HistoryStat label="Failed/error" value={stats.failed} tone="bad" />
      </div>

      <div className="grid gap-3 rounded-lg border border-stone-200 bg-white p-4 md:grid-cols-[minmax(0,1fr)_220px_220px]">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-stone-400" />
          <Input
            value={filters.query}
            onChange={(event) => setFilters((current) => ({ ...current, query: event.target.value }))}
            placeholder="Search commands, reasons, output, or tokens"
            className="pl-9"
          />
        </div>
        <Select
          value={filters.serverID}
          onChange={(event) => setFilters((current) => ({ ...current, serverID: event.target.value }))}
        >
          <option value="">All servers</option>
          {servers.data.map((server) => (
            <option key={server.id} value={server.id}>
              {server.name}
            </option>
          ))}
        </Select>
        <Select
          value={filters.status}
          onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value }))}
        >
          {statusOptions.map((option) => (
            <option key={option.value || "all"} value={option.value}>
              {option.label}
            </option>
          ))}
        </Select>
      </div>

      {state.state === "error" ? <Notice tone="bad">{state.error}</Notice> : null}
      {approvals.state === "error" ? <Notice tone="bad">{approvals.error}</Notice> : null}

      <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
        <table className="w-full table-fixed border-collapse text-left text-sm">
          <thead className="bg-stone-50 text-xs uppercase text-stone-500">
            <tr>
              <th className="w-[12%] px-4 py-3 font-semibold">Status</th>
              <th className="w-[16%] px-4 py-3 font-semibold">Server</th>
              <th className="w-[18%] px-4 py-3 font-semibold">Token</th>
              <th className="w-[34%] px-4 py-3 font-semibold">Command</th>
              <th className="w-[12%] px-4 py-3 font-semibold">Exit</th>
              <th className="w-[8%] px-4 py-3 text-right font-semibold">Time</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-stone-100">
            {state.state === "loading" ? (
              <tr>
                <td className="px-4 py-8 text-center text-sm text-stone-500" colSpan={6}>
                  Loading history...
                </td>
              </tr>
            ) : null}
            {state.state !== "loading" && state.data.length === 0 ? (
              <tr>
                <td className="px-4 py-8 text-center text-sm text-stone-500" colSpan={6}>
                  No command history yet.
                </td>
              </tr>
            ) : null}
            {state.state !== "loading"
              ? state.data.map((item) => (
                  <tr
                    key={item.id}
                    className="cursor-pointer transition hover:bg-stone-50"
                    onClick={() => openHistoryItem(item)}
                  >
                    <td className="px-4 py-3">
                      <StatusBadge status={item.status} />
                    </td>
                    <td className="truncate px-4 py-3 font-medium text-stone-900">{item.server_name}</td>
                    <td className="truncate px-4 py-3 text-stone-600">{item.token_name || "manual"}</td>
                    <td className="truncate px-4 py-3 font-mono text-xs text-stone-700">{oneLine(item.command)}</td>
                    <td className="px-4 py-3 text-stone-600">{item.exit_code ?? "-"}</td>
                    <td className="px-4 py-3 text-right text-xs text-stone-500">{formatShortTime(item.created_at)}</td>
                  </tr>
                ))
              : null}
          </tbody>
        </table>
      </div>

      <PaginationBar
        start={pageStart}
        end={pageEnd}
        total={state.total}
        disabled={state.state === "loading"}
        onPrevious={() => loadHistory(Math.max(0, state.offset - state.limit))}
        onNext={() => loadHistory(state.next_offset)}
        hasPrevious={state.offset > 0}
        hasNext={state.next_offset !== null && state.next_offset !== undefined}
      />

      <HistoryDialog item={selected} onClose={() => setSelected(null)} />
    </section>
  );
}

function HistoryStat({ label, value, tone = "neutral" }) {
  return (
    <div className="rounded-lg border border-stone-200 bg-white p-4">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-stone-500">{label}</span>
        <Badge tone={tone}>{value}</Badge>
      </div>
    </div>
  );
}

function HistoryDialog({ item, onClose }) {
  if (!item) return null;
  const output = [item.stdout, item.stderr, item.error].filter(Boolean).join("\n\n");

  return (
    <Dialog
      open={Boolean(item)}
      title={`Request #${item.id}`}
      description={`${item.server_name} · ${formatDateTime(item.created_at)}`}
      onClose={onClose}
      size="wide"
      className="h-[calc(100vh-100px)] grid-rows-[auto_minmax(0,1fr)]"
      bodyClassName="min-h-0 overflow-hidden p-0"
    >
      <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
        <div className="grid gap-2 border-b border-stone-200 px-5 py-3">
          <div className="grid min-w-0 gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-center">
            <span className="min-w-0 truncate text-sm text-stone-600">
              {item.reason ? item.reason : "No reason provided."}
            </span>
            <div className="flex min-w-0 flex-wrap items-center gap-2 md:justify-end">
              <StatusBadge status={item.status} />
              {item.token_name ? <Badge>{item.token_name}</Badge> : null}
              {item.session_id ? <Badge>session {item.session_id}</Badge> : null}
              {item.exit_code !== undefined && item.exit_code !== null ? <Badge>exit {item.exit_code}</Badge> : null}
            </div>
          </div>
          {item.user_note ? (
            <p className="truncate rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950">
              <span className="font-semibold">User note:</span> {item.user_note}
            </p>
          ) : null}
        </div>

        <div className="grid min-h-0 gap-4 p-5 lg:grid-cols-2">
          <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-2">
            <SectionHeader label="Command" value={item.command} />
            <pre className="min-h-0 overflow-auto whitespace-pre-wrap break-words rounded-md bg-stone-950 p-4 text-xs leading-5 text-stone-100">
              {item.command}
            </pre>
          </div>
          <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-2">
            <SectionHeader label="Output" value={output} />
            <pre className="min-h-0 overflow-auto whitespace-pre-wrap break-words rounded-md bg-stone-950 p-4 text-xs leading-5 text-stone-100">
              {output || "No output captured."}
            </pre>
          </div>
        </div>
      </div>
    </Dialog>
  );
}

function SectionHeader({ label, value }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="text-xs font-semibold uppercase text-stone-500">{label}</span>
      <CopyButton value={value || ""} variant="outline" className="h-8 px-2 text-xs" iconClassName="h-3.5 w-3.5" />
    </div>
  );
}

function StatusBadge({ status }) {
  const tone = {
    completed: "good",
    running: "neutral",
    pending_approval: "warn",
    declined: "warn",
    failed: "bad",
    error: "bad",
  }[status] || "neutral";
  return <Badge tone={tone}>{statusLabel(status)}</Badge>;
}

function statusLabel(status) {
  if (status === "pending_approval") return "pending";
  return status || "unknown";
}

function oneLine(value) {
  return String(value || "").replace(/\s+/g, " ").trim();
}

function formatShortTime(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatDateTime(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}
