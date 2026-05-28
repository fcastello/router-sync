import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useProviders, useRouters } from "@/hooks/useRouterSync";
import { cn } from "@/lib/utils";
import type {
  IPRule,
  InternetProvider,
  NetworkInterface,
  RouterState,
  RoutingTable,
} from "@/types/api";
import { ChevronDown, ChevronRight, Cpu, Network, Search } from "lucide-react";
import { useMemo, useState } from "react";

function isOnline(state: RouterState): boolean {
  if (typeof state.online === "boolean") return state.online;
  return state.age_seconds < 30;
}

function relativeAge(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s ago`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  if (seconds < 86_400) return `${Math.round(seconds / 3600)}h ago`;
  return `${Math.round(seconds / 86_400)}d ago`;
}

function tableLabel(t: RoutingTable, providers: InternetProvider[]): string {
  if (t.name) return `${t.name} (#${t.id})`;
  if (t.id === 254) return "main (#254)";
  if (t.id === 253) return "default (#253)";
  if (t.id === 255) return "local (#255)";
  const match = providers.find((p) => p.table_id === t.id);
  if (match) return `${match.name} (#${t.id})`;
  return `table #${t.id}`;
}

export function RoutersPage() {
  const routers = useRouters();
  const providers = useProviders();
  const [query, setQuery] = useState("");

  const list = routers.data ?? [];
  const providerList = providers.data ?? [];

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return list;
    return list.filter((r) => r.hostname.toLowerCase().includes(q));
  }, [list, query]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-2xl font-semibold">Routers</h1>
          <p className="text-sm text-muted-foreground">
            Live state reported every 5s by each agent: interfaces, routing
            tables, and ip rules.
          </p>
        </div>
        <div className="relative w-72 max-w-full">
          <Search className="pointer-events-none absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-8"
            placeholder="Filter routers by hostname…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      {filtered.length === 0 && (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            {routers.isLoading
              ? "Loading router state…"
              : "No routers have reported state yet."}
          </CardContent>
        </Card>
      )}

      {filtered.map((r) => (
        <RouterCard key={r.hostname} router={r} providers={providerList} />
      ))}
    </div>
  );
}

interface RouterCardProps {
  router: RouterState;
  providers: InternetProvider[];
}

function RouterCard({ router, providers }: RouterCardProps) {
  const [openTables, setOpenTables] = useState<Record<number, boolean>>(() => ({
    [254]: true,
  }));
  const [openSections, setOpenSections] = useState({
    interfaces: true,
    tables: true,
    rules: true,
  });

  const online = isOnline(router);

  return (
    <Card>
      <CardHeader className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="flex items-center gap-2">
            <Cpu className="h-5 w-5 text-primary" />
            {router.hostname}
            <span
              className={cn(
                "ml-1 h-2 w-2 rounded-full",
                online ? "bg-green-500" : "bg-red-500",
              )}
              title={online ? "online" : "offline"}
            />
          </CardTitle>
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <Badge variant={online ? "success" : "warn"}>
              {online ? "online" : "stale"}
            </Badge>
            <span>agent {router.agent_version || "unknown"}</span>
            <span>log {router.log_level || "?"}</span>
            <span>last seen {relativeAge(router.age_seconds)}</span>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <Section
          title={`Interfaces (${router.interfaces?.length ?? 0})`}
          open={openSections.interfaces}
          onToggle={() =>
            setOpenSections((s) => ({ ...s, interfaces: !s.interfaces }))
          }
        >
          <InterfaceTable interfaces={router.interfaces ?? []} />
        </Section>

        <Section
          title={`Routing tables (${router.tables?.length ?? 0})`}
          open={openSections.tables}
          onToggle={() =>
            setOpenSections((s) => ({ ...s, tables: !s.tables }))
          }
        >
          {(router.tables ?? []).map((t) => (
            <TableRow
              key={t.id}
              table={t}
              open={openTables[t.id] ?? false}
              onToggle={() =>
                setOpenTables((s) => ({ ...s, [t.id]: !s[t.id] }))
              }
              providers={providers}
            />
          ))}
          {router.tables?.length === 0 && (
            <p className="text-xs text-muted-foreground">No routes reported.</p>
          )}
        </Section>

        <Section
          title={`IP rules (${router.rules?.length ?? 0})`}
          open={openSections.rules}
          onToggle={() =>
            setOpenSections((s) => ({ ...s, rules: !s.rules }))
          }
        >
          <RulesTable rules={router.rules ?? []} />
        </Section>
      </CardContent>
    </Card>
  );
}

function Section({
  title,
  open,
  onToggle,
  children,
}: {
  title: string;
  open: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-md border border-border">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center justify-between px-3 py-2 text-left text-sm font-medium"
      >
        <span className="flex items-center gap-2">
          {open ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          {title}
        </span>
      </button>
      {open && <div className="border-t border-border p-3">{children}</div>}
    </div>
  );
}

function InterfaceTable({ interfaces }: { interfaces: NetworkInterface[] }) {
  if (interfaces.length === 0) {
    return <p className="text-xs text-muted-foreground">No interfaces.</p>;
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead className="text-left text-muted-foreground">
          <tr>
            <th className="pb-1 pr-3">Name</th>
            <th className="pb-1 pr-3">State</th>
            <th className="pb-1 pr-3">MAC</th>
            <th className="pb-1 pr-3">MTU</th>
            <th className="pb-1">Addresses</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {interfaces.map((iface) => (
            <tr key={iface.name}>
              <td className="py-1 pr-3 font-mono">
                <span className="flex items-center gap-1">
                  <Network className="h-3 w-3" />
                  {iface.name}
                </span>
              </td>
              <td className="py-1 pr-3">
                <span
                  className={cn(
                    "inline-flex h-2 w-2 rounded-full",
                    iface.up ? "bg-green-500" : "bg-red-500",
                  )}
                />{" "}
                {iface.up ? "up" : "down"}
              </td>
              <td className="py-1 pr-3 font-mono">{iface.mac || "—"}</td>
              <td className="py-1 pr-3">{iface.mtu}</td>
              <td className="py-1 font-mono text-[11px]">
                {(iface.addresses ?? []).join(", ") || "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TableRow({
  table,
  open,
  onToggle,
  providers,
}: {
  table: RoutingTable;
  open: boolean;
  onToggle: () => void;
  providers: InternetProvider[];
}) {
  return (
    <div className="mb-2 rounded border border-border last:mb-0">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center justify-between px-2 py-1 text-left text-xs font-medium"
      >
        <span className="flex items-center gap-2">
          {open ? (
            <ChevronDown className="h-3 w-3" />
          ) : (
            <ChevronRight className="h-3 w-3" />
          )}
          {tableLabel(table, providers)}
        </span>
        <span className="text-[11px] text-muted-foreground">
          {table.routes?.length ?? 0} routes
        </span>
      </button>
      {open && (
        <div className="border-t border-border p-2">
          {table.routes.length === 0 ? (
            <p className="text-xs text-muted-foreground">No routes.</p>
          ) : (
            <table className="w-full text-[11px]">
              <thead className="text-left text-muted-foreground">
                <tr>
                  <th className="pb-1 pr-3">Destination</th>
                  <th className="pb-1 pr-3">Via</th>
                  <th className="pb-1 pr-3">Dev</th>
                  <th className="pb-1 pr-3">Proto</th>
                  <th className="pb-1 pr-3">Scope</th>
                  <th className="pb-1">Metric</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border font-mono">
                {table.routes.map((r, i) => (
                  <tr key={`${r.dst}-${i}`}>
                    <td className="py-1 pr-3">{r.dst}</td>
                    <td className="py-1 pr-3">{r.gateway || "—"}</td>
                    <td className="py-1 pr-3">{r.interface || "—"}</td>
                    <td className="py-1 pr-3">{r.protocol || "—"}</td>
                    <td className="py-1 pr-3">{r.scope || "—"}</td>
                    <td className="py-1">{r.metric ?? 0}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}

function RulesTable({ rules }: { rules: IPRule[] }) {
  if (rules.length === 0) {
    return <p className="text-xs text-muted-foreground">No rules reported.</p>;
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead className="text-left text-muted-foreground">
          <tr>
            <th className="pb-1 pr-3">Priority</th>
            <th className="pb-1 pr-3">From</th>
            <th className="pb-1 pr-3">Table</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border font-mono">
          {[...rules]
            .sort((a, b) => a.priority - b.priority)
            .map((r) => (
              <tr key={`${r.priority}-${r.from}`}>
                <td className="py-1 pr-3">{r.priority}</td>
                <td className="py-1 pr-3">{r.from}</td>
                <td className="py-1 pr-3">
                  {r.table_name ? `${r.table_name} (#${r.table})` : `#${r.table}`}
                </td>
              </tr>
            ))}
        </tbody>
      </table>
    </div>
  );
}
