import { DeviceRow } from "@/components/DeviceRow";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { queryKeys, usePolicies, usePolicyMutations, useProviders } from "@/hooks/useRouterSync";
import { fuzzyMatch } from "@/lib/fuzzy";
import { migrateLocalDisplayNames } from "@/lib/migrate-display-names";
import { migrateLocalPolicyFavorites } from "@/lib/migrate-policy-favorites";
import { migrateLocalPolicyTags } from "@/lib/migrate-policy-tags";
import { policyBody } from "@/lib/policy-body";
import { displayPolicyId } from "@/lib/policy-id";
import { allPolicyTags, normalizeTags } from "@/lib/policy-tags";
import { sortPolicies, type PolicySortKey } from "@/lib/policy-sort";
import type { RoutingPolicy } from "@/types/api";
import { useQueryClient } from "@tanstack/react-query";
import { Search, Star } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

export function DevicesPage() {
  const policies = usePolicies();
  const providers = useProviders();
  const { update } = usePolicyMutations();
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<PolicySortKey>("name");
  const [tagFilter, setTagFilter] = useState("");

  const policyList = policies.data ?? [];
  const knownTags = allPolicyTags(policyList);

  useEffect(() => {
    if (!policyList.length) return;
    Promise.all([
      migrateLocalDisplayNames(policyList),
      migrateLocalPolicyFavorites(policyList),
      migrateLocalPolicyTags(policyList),
    ])
      .then(() => qc.invalidateQueries({ queryKey: queryKeys.policies }))
      .catch(() => {
        /* ignore migration errors */
      });
  }, [policyList, qc]);

  const providerMap = useMemo(() => {
    const m = new Map<string, string>();
    (providers.data ?? []).forEach((p) => m.set(p.id, p.name));
    return m;
  }, [providers.data]);

  const displayedPolicies = useMemo(() => {
    const q = search.trim();
    let list = policyList.filter((policy) => {
      const tags = normalizeTags(policy.tags);
      if (tagFilter && !tags.includes(tagFilter)) {
        return false;
      }
      const providerName = providerMap.get(policy.provider_id) ?? policy.provider_id;
      return fuzzyMatch(
        q,
        policy.name,
        displayPolicyId(policy.id),
        policy.id,
        policy.description,
        ...tags,
        providerName,
        policy.enabled ? "enabled on active override" : "disabled off default",
        policy.favorite ? "favorite starred" : "",
      );
    });
    return sortPolicies(list, sortBy);
  }, [policyList, search, sortBy, tagFilter, providerMap]);

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

  const renamePolicy = (policy: RoutingPolicy, name: string) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { name }) });
  };

  const toggleFavorite = (policy: RoutingPolicy) => {
    update.mutate({
      id: policy.id,
      body: policyBody(policy, { favorite: !policy.favorite }),
    });
  };

  const saveDescription = (policy: RoutingPolicy, description: string) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { description }) });
  };

  const saveTags = (policy: RoutingPolicy, tags: string[]) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { tags }) });
  };

  const renderDeviceRow = (policy: RoutingPolicy, compact?: boolean) => (
    <DeviceRow
      key={policy.id}
      policy={policy}
      providerName={providerMap.get(policy.provider_id) ?? policy.provider_id}
      onToggleFavorite={() => toggleFavorite(policy)}
      onRename={(name) => renamePolicy(policy, name)}
      onSaveDescription={(description) => saveDescription(policy, description)}
      onSaveTags={(tags) => saveTags(policy, tags)}
      disabled={update.isPending}
      compact={compact}
    />
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Devices</h1>
        <p className="text-sm text-muted-foreground">
          Same policies as on the Policies page — search, sort, tags, and favorites are stored in
          NATS.
        </p>
      </div>

      <Card>
        <CardHeader className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle>
              Devices ({displayedPolicies.length}
              {search.trim() || tagFilter ? ` of ${policyList.length}` : ""})
            </CardTitle>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <div className="relative min-w-[200px] flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder="Search name, IP, provider, tags…"
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
            <div className="w-40">
              <Label>Filter tag</Label>
              <Select value={tagFilter} onChange={(e) => setTagFilter(e.target.value)}>
                <option value="">All</option>
                {knownTags.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
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
              <div className="space-y-2">
                {favoritePolicies.map((p) => renderDeviceRow(p, true))}
              </div>
            </section>
          )}

          {(favoritePolicies.length > 0 || otherPolicies.length > 0) && (
            <section className="space-y-3">
              {favoritePolicies.length > 0 && otherPolicies.length > 0 && (
                <h2 className="text-sm font-semibold text-muted-foreground">All devices</h2>
              )}
              <div className="space-y-3">{otherPolicies.map((p) => renderDeviceRow(p, false))}</div>
            </section>
          )}

          {policyList.length === 0 && (
            <p className="text-sm text-muted-foreground">No policies yet. Create one on the Policies page.</p>
          )}
          {policyList.length > 0 && displayedPolicies.length === 0 && (
            <p className="text-sm text-muted-foreground">No devices match your search or filters.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
