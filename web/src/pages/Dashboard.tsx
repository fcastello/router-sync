import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  useHealth,
  usePolicies,
  useProviders,
  useRouters,
  useStats,
  useTriggerSync,
} from "@/hooks/useRouterSync";
import { displayPolicyId } from "@/lib/policy-id";
import type { InternetProvider, RouterState } from "@/types/api";
import { cn } from "@/lib/utils";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import { Cpu, RefreshCw } from "lucide-react";

const CHART_COLORS = ["#2563eb", "#16a34a", "#d97706", "#7c3aed", "#dc2626"];

function relativeAge(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s ago`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  if (seconds < 86_400) return `${Math.round(seconds / 3600)}h ago`;
  return `${Math.round(seconds / 86_400)}d ago`;
}

export function DashboardPage() {
  const health = useHealth(5000);
  const stats = useStats(10000);
  const providers = useProviders();
  const policies = usePolicies();
  const routers = useRouters(10000);
  const sync = useTriggerSync();

  const providerList = providers.data ?? [];
  const policyList = policies.data ?? [];
  const routerList = routers.data ?? [];

  // Allocation counts only enabled policies per provider. We compute this on the
  // client rather than using stats.policies_per_provider, which counts every
  // policy regardless of its enabled state.
  const allocation = providerList.map((p, i) => ({
    name: p.name,
    value: policyList.filter((pol) => pol.provider_id === p.id && pol.enabled)
      .length,
    fill: CHART_COLORS[i % CHART_COLORS.length],
  }));
  const allocationTotal = allocation.reduce((sum, a) => sum + a.value, 0);

  const enabledCount = policyList.filter((p) => p.enabled).length;
  const apiOk = health.data?.status === "ok" || health.data?.status === "healthy";

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Global overview</h1>
          <p className="text-sm text-muted-foreground">
            API health, router state, and policy allocation across uplinks.
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
            <CardTitle>Routers</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-semibold">{routerList.length}</p>
            <p className="text-xs text-muted-foreground">
              {routerList.filter((r) => r.online ?? r.age_seconds < 30).length} online
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Providers</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-semibold">
              {stats.data?.sync.providers_count ?? providerList.length}
            </p>
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
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Router state</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {routerList.length === 0 && (
              <p className="text-sm text-muted-foreground">
                No agents reporting yet. Deploy agents on the routers to see live
                interface mappings.
              </p>
            )}
            {routerList.map((r) => (
              <RouterSummary key={r.hostname} router={r} providers={providerList} />
            ))}
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
              <>
                <div className="h-56">
                  {allocationTotal === 0 ? (
                    <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                      No enabled policies to allocate.
                    </div>
                  ) : (
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={allocation.filter((a) => a.value > 0)}
                          dataKey="value"
                          nameKey="name"
                          cx="50%"
                          cy="50%"
                          innerRadius={50}
                          outerRadius={80}
                          paddingAngle={2}
                        >
                          {allocation
                            .filter((a) => a.value > 0)
                            .map((entry) => (
                              <Cell key={entry.name} fill={entry.fill} />
                            ))}
                        </Pie>
                        <Tooltip />
                      </PieChart>
                    </ResponsiveContainer>
                  )}
                </div>
                <ul className="mt-4 space-y-1 text-sm">
                  {allocation.map((a) => (
                    <li key={a.name} className="flex items-center justify-between">
                      <span className="flex items-center gap-2">
                        <span
                          className="inline-block h-2.5 w-2.5 rounded-full"
                          style={{ backgroundColor: a.fill }}
                        />
                        {a.name}
                      </span>
                      <span className="font-medium">{a.value} device(s)</span>
                    </li>
                  ))}
                </ul>
              </>
            )}
          </CardContent>
        </Card>
      </div>

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

function RouterSummary({
  router,
  providers,
}: {
  router: RouterState;
  providers: InternetProvider[];
}) {
  const online = router.online ?? router.age_seconds < 30;
  const providerMappings = providers
    .map((p) => ({
      name: p.name,
      iface: p.interfaces?.[router.hostname] ?? "",
    }))
    .filter((x) => x.iface);

  return (
    <div className="rounded-md border border-border px-3 py-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2 font-medium">
          <Cpu className="h-4 w-4 text-primary" />
          {router.hostname}
          <span
            className={cn(
              "h-2 w-2 rounded-full",
              online ? "bg-green-500" : "bg-red-500",
            )}
          />
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Badge variant={online ? "success" : "warn"}>
            {online ? "online" : "stale"}
          </Badge>
          <span>{relativeAge(router.age_seconds)}</span>
        </div>
      </div>
      <p className="mt-1 text-xs text-muted-foreground">
        agent {router.agent_version || "unknown"} · log {router.log_level || "?"} ·
        {" "}
        {router.interfaces?.length ?? 0} ifaces ·
        {" "}
        {router.tables?.length ?? 0} tables ·
        {" "}
        {router.rules?.length ?? 0} rules
      </p>
      {providerMappings.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1 text-xs">
          {providerMappings.map((p) => (
            <Badge key={p.name} variant="secondary">
              {p.name} → {p.iface}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}
