import { PolicyRow } from "@/components/PolicyRow";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { queryKeys, usePolicies, usePolicyMutations, useProviders } from "@/hooks/useRouterSync";
import { fuzzyMatch } from "@/lib/fuzzy";
import { migrateLocalPolicyFavorites } from "@/lib/migrate-policy-favorites";
import { displayPolicyId } from "@/lib/policy-id";
import { sortPolicies, type PolicySortKey } from "@/lib/policy-sort";
import type { CreatePolicyRequest, RoutingPolicy } from "@/types/api";
import { useQueryClient } from "@tanstack/react-query";
import { Search, Star } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

const emptyForm = {
  name: "",
  source_ip: "",
  provider_id: "",
  description: "",
  enabled: true,
  favorite: false,
};

function policyBody(policy: RoutingPolicy, patch: Partial<CreatePolicyRequest> = {}): CreatePolicyRequest {
  return {
    name: policy.name,
    source_ip: policy.id,
    provider_id: policy.provider_id,
    description: policy.description,
    enabled: policy.enabled,
    favorite: policy.favorite ?? false,
    ...patch,
  };
}

export function PoliciesPage() {
  const policies = usePolicies();
  const providers = useProviders();
  const { create, update, remove } = usePolicyMutations();
  const qc = useQueryClient();
  const [form, setForm] = useState(emptyForm);
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<PolicySortKey>("name");

  const providerList = providers.data ?? [];
  const policyList = policies.data ?? [];

  useEffect(() => {
    if (!policyList.length) return;
    migrateLocalPolicyFavorites(policyList)
      .then(() => qc.invalidateQueries({ queryKey: queryKeys.policies }))
      .catch(() => {
        /* ignore migration errors */
      });
  }, [policyList, qc]);

  const providerNameById = useMemo(() => {
    const m = new Map<string, string>();
    providerList.forEach((p) => m.set(p.id, p.name));
    return m;
  }, [providerList]);

  const displayedPolicies = useMemo(() => {
    const q = search.trim();
    let list = policyList.filter((policy) => {
      const providerName = providerNameById.get(policy.provider_id) ?? policy.provider_id;
      return fuzzyMatch(
        q,
        policy.name,
        displayPolicyId(policy.id),
        policy.id,
        policy.description,
        providerName,
        policy.enabled ? "enabled on active override" : "disabled off default",
        policy.favorite ? "favorite starred" : "",
      );
    });
    list = sortPolicies(list, sortBy);
    return list;
  }, [policyList, search, sortBy, providerNameById]);

  const { favoritePolicies, otherPolicies } = useMemo(() => {
    const favorites: RoutingPolicy[] = [];
    const others: RoutingPolicy[] = [];
    for (const policy of displayedPolicies) {
      if (policy.favorite) {
        favorites.push(policy);
      } else {
        others.push(policy);
      }
    }
    return { favoritePolicies: favorites, otherPolicies: others };
  }, [displayedPolicies]);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const body: CreatePolicyRequest = {
      name: form.name,
      source_ip: form.source_ip.trim(),
      provider_id: form.provider_id,
      description: form.description || undefined,
      enabled: form.enabled,
      favorite: form.favorite,
    };
    create.mutate(body, {
      onSuccess: () => setForm(emptyForm),
    });
  };

  const toggleEnabled = (policy: RoutingPolicy) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { enabled: !policy.enabled }) });
  };

  const changeProvider = (policy: RoutingPolicy, providerId: string) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { provider_id: providerId }) });
  };

  const toggleFavorite = (policy: RoutingPolicy) => {
    update.mutate({
      id: policy.id,
      body: policyBody(policy, { favorite: !policy.favorite }),
    });
  };

  const renamePolicy = (policy: RoutingPolicy, name: string) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { name }) });
  };

  const renderPolicyRow = (policy: RoutingPolicy, compact?: boolean) => (
    <PolicyRow
      key={policy.id}
      policy={policy}
      providers={providerList}
      isFavorite={Boolean(policy.favorite)}
      onToggleFavorite={() => toggleFavorite(policy)}
      onToggleEnabled={() => toggleEnabled(policy)}
      onChangeProvider={(providerId) => changeProvider(policy, providerId)}
      onRename={(name) => renamePolicy(policy, name)}
      onDelete={() => {
        if (confirm(`Delete policy for ${displayPolicyId(policy.id)}?`)) {
          remove.mutate(policy.id);
        }
      }}
      updatePending={update.isPending}
      compact={compact}
    />
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Policy builder</h1>
        <p className="text-sm text-muted-foreground">
          Route traffic by source IP or CIDR through a chosen uplink. Edit display names with the
          pencil icon (saved in NATS). Star policies for the favorites section.
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
              <div className="flex flex-wrap items-center gap-6 pt-6">
                <div className="flex items-center gap-2">
                  <Switch
                    checked={form.enabled}
                    onCheckedChange={(enabled) => setForm((f) => ({ ...f, enabled }))}
                  />
                  <Label>Enabled on create</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch
                    checked={form.favorite}
                    onCheckedChange={(favorite) => setForm((f) => ({ ...f, favorite }))}
                  />
                  <Label>Favorite</Label>
                </div>
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
        <CardContent className="space-y-6">
          {favoritePolicies.length > 0 && (
            <section className="space-y-3">
              <div className="flex items-center gap-2">
                <Star className="h-4 w-4 fill-amber-400 text-amber-400" aria-hidden />
                <h2 className="text-sm font-semibold">Favorites ({favoritePolicies.length})</h2>
              </div>
              <div className="space-y-2">{favoritePolicies.map((p) => renderPolicyRow(p, true))}</div>
            </section>
          )}

          {(favoritePolicies.length > 0 || otherPolicies.length > 0) && (
            <section className="space-y-3">
              {favoritePolicies.length > 0 && otherPolicies.length > 0 && (
                <h2 className="text-sm font-semibold text-muted-foreground">All policies</h2>
              )}
              <div className="space-y-3">{otherPolicies.map((p) => renderPolicyRow(p, false))}</div>
            </section>
          )}

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
