import { DeviceRow } from "@/components/DeviceRow";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import {
  queryKeys,
  useDeviceMeta,
  usePolicies,
  usePolicyMutations,
  useProviders,
} from "@/hooks/useRouterSync";
import { allTags, updateDeviceMeta } from "@/lib/device-meta";
import { fuzzyMatch } from "@/lib/fuzzy";
import { migrateLocalDisplayNames } from "@/lib/migrate-display-names";
import { migrateLocalPolicyFavorites } from "@/lib/migrate-policy-favorites";
import { policyBody } from "@/lib/policy-body";
import { displayPolicyId } from "@/lib/policy-id";
import { sortPolicies, type PolicySortKey } from "@/lib/policy-sort";
import type { RoutingPolicy } from "@/types/api";
import { useQueryClient } from "@tanstack/react-query";
import { Search, Star } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

type DeviceEntry = {
  policy: RoutingPolicy;
  tags: string[];
};

export function DevicesPage() {
  const policies = usePolicies();
  const providers = useProviders();
  const { update } = usePolicyMutations();
  const qc = useQueryClient();
  const { data: meta = {} } = useDeviceMeta();
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<PolicySortKey>("name");
  const [tagFilter, setTagFilter] = useState("");

  const policyList = policies.data ?? [];
  const knownTags = allTags(meta);

  useEffect(() => {
    if (!policyList.length) return;
    Promise.all([
      migrateLocalPolicyFavorites(policyList),
      migrateLocalDisplayNames(policyList),
    ])
      .then(() => {
        qc.invalidateQueries({ queryKey: queryKeys.policies });
        qc.invalidateQueries({ queryKey: queryKeys.deviceMeta });
      })
      .catch(() => {
        /* ignore migration errors */
      });
  }, [policyList, qc]);

  const providerMap = useMemo(() => {
    const m = new Map<string, string>();
    (providers.data ?? []).forEach((p) => m.set(p.id, p.name));
    return m;
  }, [providers.data]);

  const allEntries: DeviceEntry[] = useMemo(
    () =>
      policyList.map((policy) => ({
        policy,
        tags: meta[policy.id]?.tags ?? [],
      })),
    [policyList, meta],
  );

  const displayedEntries = useMemo(() => {
    const q = search.trim();
    let list = allEntries.filter(({ policy, tags: rowTags }) => {
      if (tagFilter && !rowTags.includes(tagFilter)) {
        return false;
      }
      const providerName = providerMap.get(policy.provider_id) ?? policy.provider_id;
      return fuzzyMatch(
        q,
        policy.name,
        displayPolicyId(policy.id),
        policy.id,
        policy.description,
        providerName,
        ...rowTags,
        policy.enabled ? "enabled on active override" : "disabled off default",
        policy.favorite ? "favorite starred" : "",
      );
    });
    const sorted = sortPolicies(
      list.map((e) => e.policy),
      sortBy,
    );
    const order = new Map(sorted.map((p, i) => [p.id, i]));
    return [...list].sort(
      (a, b) => (order.get(a.policy.id) ?? 0) - (order.get(b.policy.id) ?? 0),
    );
  }, [allEntries, search, sortBy, tagFilter, providerMap]);

  const { favoriteEntries, otherEntries } = useMemo(() => {
    const favorites: DeviceEntry[] = [];
    const others: DeviceEntry[] = [];
    for (const entry of displayedEntries) {
      if (entry.policy.favorite) {
        favorites.push(entry);
      } else {
        others.push(entry);
      }
    }
    return { favoriteEntries: favorites, otherEntries: others };
  }, [displayedEntries]);

  const renamePolicy = (policy: RoutingPolicy, name: string) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { name }) });
  };

  const toggleFavorite = (policy: RoutingPolicy) => {
    update.mutate({
      id: policy.id,
      body: policyBody(policy, { favorite: !policy.favorite }),
    });
  };

  const saveTags = (policyId: string, tags: string[]) => {
    updateDeviceMeta(policyId, { tags });
    qc.invalidateQueries({ queryKey: queryKeys.deviceMeta });
  };

  const renderDeviceRow = ({ policy, tags: rowTags }: DeviceEntry, compact?: boolean) => (
    <DeviceRow
      key={policy.id}
      policy={policy}
      tags={rowTags}
      providerName={providerMap.get(policy.provider_id) ?? policy.provider_id}
      onToggleFavorite={() => toggleFavorite(policy)}
      onRename={(name) => renamePolicy(policy, name)}
      onSaveTags={(nextTags) => saveTags(policy.id, nextTags)}
      disabled={update.isPending}
      compact={compact}
    />
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Devices</h1>
        <p className="text-sm text-muted-foreground">
          Same policies as on the Policies page — search, sort, and favorites work the same way.
          Tags are optional browser-only labels.
        </p>
      </div>

      <Card>
        <CardHeader className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle>
              Devices ({displayedEntries.length}
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
              <Select
                value={tagFilter}
                onChange={(e) => setTagFilter(e.target.value)}
              >
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
          {favoriteEntries.length > 0 && (
            <section className="space-y-3">
              <div className="flex items-center gap-2">
                <Star className="h-4 w-4 fill-amber-400 text-amber-400" aria-hidden />
                <h2 className="text-sm font-semibold">Favorites ({favoriteEntries.length})</h2>
              </div>
              <div className="space-y-2">
                {favoriteEntries.map((e) => renderDeviceRow(e, true))}
              </div>
            </section>
          )}

          {(favoriteEntries.length > 0 || otherEntries.length > 0) && (
            <section className="space-y-3">
              {favoriteEntries.length > 0 && otherEntries.length > 0 && (
                <h2 className="text-sm font-semibold text-muted-foreground">All devices</h2>
              )}
              <div className="space-y-3">{otherEntries.map((e) => renderDeviceRow(e, false))}</div>
            </section>
          )}

          {policyList.length === 0 && (
            <p className="text-sm text-muted-foreground">No policies yet. Create one on the Policies page.</p>
          )}
          {policyList.length > 0 && displayedEntries.length === 0 && (
            <p className="text-sm text-muted-foreground">No devices match your search or filters.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
