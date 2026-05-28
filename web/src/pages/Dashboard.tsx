import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  useHealth,
  usePolicies,
  useProviders,
  useStats,
  useTriggerSync,
} from "@/hooks/useRouterSync";
import { displayPolicyId } from "@/lib/policy-id";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import { RefreshCw } from "lucide-react";

const CHART_COLORS = ["#2563eb", "#16a34a", "#d97706", "#7c3aed", "#dc2626"];

export function DashboardPage() {
  const health = useHealth(5000);
  const stats = useStats(10000);
  const providers = useProviders();
  const policies = usePolicies();
  const sync = useTriggerSync();

  const providerList = providers.data ?? [];
  const policyList = policies.data ?? [];
  const perProvider = stats.data?.sync?.policies_per_provider ?? {};

  const allocation = providerList.map((p, i) => ({
    name: p.name,
    value: perProvider[p.id] ?? policyList.filter((pol) => pol.provider_id === p.id && pol.enabled).length,
    fill: CHART_COLORS[i % CHART_COLORS.length],
  }));

  const enabledCount = policyList.filter((p) => p.enabled).length;
  const apiOk = health.data?.status === "ok" || health.data?.status === "healthy";

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Global overview</h1>
          <p className="text-sm text-muted-foreground">
            Uplink status, traffic allocation, and sync health
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => sync.mutate()}
          disabled={sync.isPending}
        >
          <RefreshCw className={`mr-2 h-4 w-4 ${sync.isPending ? "animate-spin" : ""}`} />
          Trigger sync
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card>
          <CardHeader>
            <CardTitle>API</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className={`h-3 w-3 rounded-full ${apiOk ? "bg-green-500" : "bg-red-500"}`} />
              <span className="text-lg font-medium">{apiOk ? "Online" : "Offline"}</span>
            </div>
            {health.data?.timestamp && (
              <p className="mt-1 text-xs text-muted-foreground">
                Last check: {new Date(health.data.timestamp).toLocaleTimeString()}
              </p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Providers</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-semibold">{stats.data?.sync.providers_count ?? providerList.length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Active policies</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-semibold">{enabledCount}</p>
            <p className="text-xs text-muted-foreground">
              of {policyList.length} total
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Sync interval</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-lg font-medium">{stats.data?.sync.sync_interval ?? "—"}</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Uplink status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {providerList.length === 0 && (
              <p className="text-sm text-muted-foreground">No providers configured yet.</p>
            )}
            {providerList.map((p) => (
              <div
                key={p.id}
                className="flex items-center justify-between rounded-md border border-border px-3 py-2"
              >
                <div>
                  <p className="font-medium">{p.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {p.interface} · gw {p.gateway} · table {p.table_id}
                  </p>
                </div>
                <div className="text-right">
                  <Badge variant="success">Configured</Badge>
                  <p className="mt-1 text-xs text-muted-foreground">Latency N/A</p>
                </div>
              </div>
            ))}
            <p className="text-xs text-muted-foreground">
              Live latency requires future health probes; gateway reachability is inferred from sync stats.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Traffic allocation</CardTitle>
          </CardHeader>
          <CardContent>
            {allocation.length === 0 ? (
              <p className="text-sm text-muted-foreground">Add policies to see allocation.</p>
            ) : (
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={allocation}
                      dataKey="value"
                      nameKey="name"
                      cx="50%"
                      cy="50%"
                      innerRadius={50}
                      outerRadius={80}
                      paddingAngle={2}
                    >
                      {allocation.map((entry) => (
                        <Cell key={entry.name} fill={entry.fill} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
                <ul className="mt-2 space-y-1 text-sm">
                  {allocation.map((a) => (
                    <li key={a.name} className="flex justify-between">
                      <span>{a.name}</span>
                      <span className="font-medium">{a.value} device(s)</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {stats.data?.router && (
        <Card>
          <CardHeader>
            <CardTitle>Router state</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="overflow-auto rounded-md bg-muted p-3 text-xs">
              {JSON.stringify(stats.data.router, null, 2)}
            </pre>
          </CardContent>
        </Card>
      )}

      {policyList.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Recent overrides</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="divide-y divide-border text-sm">
              {policyList.slice(0, 5).map((p) => (
                <li key={p.id} className="flex justify-between py-2">
                  <span>{p.name}</span>
                  <span className="font-mono text-xs text-muted-foreground">
                    {displayPolicyId(p.id)}
                  </span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
