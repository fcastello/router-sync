import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { usePolicies, usePolicyMutations, useProviders } from "@/hooks/useRouterSync";
import { loadDeviceMeta } from "@/lib/device-meta";
import { fuzzyMatch } from "@/lib/fuzzy";
import { displayPolicyId } from "@/lib/policy-id";
import { sortPolicies, type PolicySortKey } from "@/lib/policy-sort";
import type { CreatePolicyRequest, RoutingPolicy } from "@/types/api";
import { useMemo, useState } from "react";
import { Search, Trash2 } from "lucide-react";

const emptyForm = {
  name: "",
  source_ip: "",
  provider_id: "",
  description: "",
  enabled: true,
};

export function PoliciesPage() {
  const policies = usePolicies();
  const providers = useProviders();
  const { create, update, remove } = usePolicyMutations();
  const [form, setForm] = useState(emptyForm);
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<PolicySortKey>("name");
  const meta = loadDeviceMeta();

  const providerList = providers.data ?? [];
  const policyList = policies.data ?? [];

  const providerNameById = useMemo(() => {
    const m = new Map<string, string>();
    providerList.forEach((p) => m.set(p.id, p.name));
    return m;
  }, [providerList]);

  const displayedPolicies = useMemo(() => {
    const q = search.trim();
    let list = policyList.filter((policy) => {
      const friendly = meta[policy.id]?.friendlyName;
      const providerName = providerNameById.get(policy.provider_id) ?? policy.provider_id;
      return fuzzyMatch(
        q,
        policy.name,
        friendly,
        displayPolicyId(policy.id),
        policy.id,
        policy.description,
        providerName,
        policy.enabled ? "enabled on active override" : "disabled off default",
      );
    });
    list = sortPolicies(list, sortBy);
    return list;
  }, [policyList, search, sortBy, meta, providerNameById]);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const body: CreatePolicyRequest = {
      name: form.name,
      source_ip: form.source_ip.trim(),
      provider_id: form.provider_id,
      description: form.description || undefined,
      enabled: form.enabled,
    };
    create.mutate(body, {
      onSuccess: () => setForm(emptyForm),
    });
  };

  const toggleEnabled = (policy: RoutingPolicy) => {
    const body: CreatePolicyRequest = {
      name: policy.name,
      source_ip: policy.id,
      provider_id: policy.provider_id,
      description: policy.description,
      enabled: !policy.enabled,
    };
    update.mutate({ id: policy.id, body });
  };

  const changeProvider = (policy: RoutingPolicy, providerId: string) => {
    update.mutate({
      id: policy.id,
      body: {
        name: policy.name,
        source_ip: policy.id,
        provider_id: providerId,
        description: policy.description,
        enabled: policy.enabled,
      },
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Policy builder</h1>
        <p className="text-sm text-muted-foreground">
          Route traffic by source IP or CIDR through a chosen uplink.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>New policy</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="space-y-4">
            <div className="flex flex-wrap items-end gap-2 rounded-lg border border-dashed border-border bg-muted/40 p-4 text-sm">
              <span className="text-muted-foreground">Route</span>
              <div className="min-w-[140px] flex-1">
                <Label className="sr-only">Device / source</Label>
                <Input
                  placeholder="192.168.1.50 or 192.168.1.0/24"
                  value={form.source_ip}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      source_ip: e.target.value,
                      name: f.name || e.target.value,
                    }))
                  }
                  required
                />
              </div>
              <span className="text-muted-foreground">named</span>
              <div className="min-w-[120px] flex-1">
                <Input
                  placeholder="Living Room TV"
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  required
                />
              </div>
              <span className="text-muted-foreground">via</span>
              <div className="min-w-[140px]">
                <Select
                  value={form.provider_id}
                  onChange={(e) => setForm((f) => ({ ...f, provider_id: e.target.value }))}
                  required
                >
                  <option value="">Select uplink…</option>
                  {providerList.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </Select>
              </div>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <div>
                <Label>Description (optional)</Label>
                <Input
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                />
              </div>
              <div className="flex items-center gap-2 pt-6">
                <Switch
                  checked={form.enabled}
                  onCheckedChange={(enabled) => setForm((f) => ({ ...f, enabled }))}
                />
                <Label>Enabled on create</Label>
              </div>
            </div>
            <Button type="submit" disabled={create.isPending || !form.provider_id}>
              Add policy
            </Button>
            {create.isError && (
              <p className="text-sm text-destructive">{(create.error as Error).message}</p>
            )}
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle>
              Policies ({displayedPolicies.length}
              {search.trim() ? ` of ${policyList.length}` : ""})
            </CardTitle>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <div className="relative min-w-[200px] flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder="Search name, IP, provider…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <div className="w-40">
              <Label>Sort by</Label>
              <Select value={sortBy} onChange={(e) => setSortBy(e.target.value as PolicySortKey)}>
                <option value="name">Name</option>
                <option value="subnet">Subnet (specific first)</option>
                <option value="status">Status (active first)</option>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {displayedPolicies.map((policy) => {
            const friendly = meta[policy.id]?.friendlyName;
            const isOverride = policy.enabled;
            return (
              <div
                key={policy.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border px-4 py-3"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{friendly || policy.name}</span>
                    {isOverride ? (
                      <Badge variant="default">Override</Badge>
                    ) : (
                      <Badge variant="muted">Default (disabled)</Badge>
                    )}
                  </div>
                  <p className="font-mono text-xs text-muted-foreground">
                    {displayPolicyId(policy.id)}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-3">
                  <Select
                    value={policy.provider_id}
                    onChange={(e) => changeProvider(policy, e.target.value)}
                    className="w-36"
                  >
                    {providerList.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name}
                      </option>
                    ))}
                  </Select>
                  <div className="flex items-center gap-2">
                    <Switch
                      checked={policy.enabled}
                      onCheckedChange={() => toggleEnabled(policy)}
                      disabled={update.isPending}
                    />
                    <span className="text-xs text-muted-foreground">
                      {policy.enabled ? "On" : "Off"}
                    </span>
                  </div>
                  <Button
                    variant="ghost"
                    className="text-destructive"
                    onClick={() => {
                      if (confirm(`Delete policy for ${displayPolicyId(policy.id)}?`)) {
                        remove.mutate(policy.id);
                      }
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            );
          })}
          {policyList.length === 0 && (
            <p className="text-sm text-muted-foreground">No policies configured.</p>
          )}
          {policyList.length > 0 && displayedPolicies.length === 0 && (
            <p className="text-sm text-muted-foreground">No policies match your search.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
