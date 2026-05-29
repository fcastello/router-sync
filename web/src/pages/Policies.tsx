import { NewPolicyForm, type NewPolicyFormState } from "@/components/NewPolicyForm";
import { PolicyRow } from "@/components/PolicyRow";
import { Button } from "@/components/ui/button";
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
import { allPolicyTags, normalizeTags, parseTagsInput } from "@/lib/policy-tags";
import { sortPolicies, type PolicySortKey } from "@/lib/policy-sort";
import type { CreatePolicyRequest, RoutingPolicy } from "@/types/api";
import { useQueryClient } from "@tanstack/react-query";
import { Plus, Search, Star } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

const emptyForm: NewPolicyFormState = {
  name: "",
  source_ip: "",
  provider_id: "",
  description: "",
  tags: "",
  enabled: true,
  favorite: false,
};

export function PoliciesPage() {
  const policies = usePolicies();
  const providers = useProviders();
  const { create, update, remove } = usePolicyMutations();
  const qc = useQueryClient();
  const [form, setForm] = useState(emptyForm);
  const [showNewPolicy, setShowNewPolicy] = useState(false);
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<PolicySortKey>("name");
  const [tagFilter, setTagFilter] = useState("");

  const providerList = providers.data ?? [];
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

  const providerNameById = useMemo(() => {
    const m = new Map<string, string>();
    providerList.forEach((p) => m.set(p.id, p.name));
    return m;
  }, [providerList]);

  const displayedPolicies = useMemo(() => {
    const q = search.trim();
    let list = policyList.filter((policy) => {
      const tags = normalizeTags(policy.tags);
      if (tagFilter && !tags.includes(tagFilter)) {
        return false;
      }
      const providerName = providerNameById.get(policy.provider_id) ?? policy.provider_id;
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
    list = sortPolicies(list, sortBy);
    return list;
  }, [policyList, search, sortBy, tagFilter, providerNameById]);

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
    const tags = parseTagsInput(form.tags);
    const body: CreatePolicyRequest = {
      name: form.name,
      source_ip: form.source_ip.trim(),
      provider_id: form.provider_id,
      description: form.description || undefined,
      tags: tags.length > 0 ? tags : undefined,
      enabled: form.enabled,
      favorite: form.favorite,
    };
    create.mutate(body, {
      onSuccess: () => {
        setForm(emptyForm);
        setShowNewPolicy(false);
      },
    });
  };

  const openNewPolicy = () => {
    setForm(emptyForm);
    setShowNewPolicy(true);
  };

  const cancelNewPolicy = () => {
    setForm(emptyForm);
    setShowNewPolicy(false);
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

  const saveDescription = (policy: RoutingPolicy, description: string) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { description }) });
  };

  const saveTags = (policy: RoutingPolicy, tags: string[]) => {
    update.mutate({ id: policy.id, body: policyBody(policy, { tags }) });
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
      onSaveDescription={(description) => saveDescription(policy, description)}
      onSaveTags={(tags) => saveTags(policy, tags)}
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
          Route traffic by source IP or CIDR through a chosen uplink. Use + to add a policy; edit
          names, descriptions, and tags with the pencil (saved in NATS). Star policies for favorites.
        </p>
      </div>

      <Card>
        <CardHeader className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle>
              Policies ({displayedPolicies.length}
              {search.trim() || tagFilter ? ` of ${policyList.length}` : ""})
            </CardTitle>
            <Button
              type="button"
              variant={showNewPolicy ? "secondary" : "default"}
              className="h-9 w-9 shrink-0 p-0"
              onClick={() => (showNewPolicy ? cancelNewPolicy() : openNewPolicy())}
              title={showNewPolicy ? "Cancel new policy" : "Add policy"}
              aria-label={showNewPolicy ? "Cancel new policy" : "Add policy"}
              aria-expanded={showNewPolicy}
            >
              {showNewPolicy ? <span className="text-lg leading-none">×</span> : <Plus className="h-5 w-5" />}
            </Button>
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
          {showNewPolicy && (
            <NewPolicyForm
              form={form}
              providers={providerList}
              onChange={(patch) => setForm((f) => ({ ...f, ...patch }))}
              onSubmit={submit}
              onCancel={cancelNewPolicy}
              pending={create.isPending}
              error={create.isError ? (create.error as Error) : null}
            />
          )}

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

          {policyList.length === 0 && !showNewPolicy && (
            <p className="text-sm text-muted-foreground">
              No policies yet. Click + to add one.
            </p>
          )}
          {policyList.length > 0 && displayedPolicies.length === 0 && (
            <p className="text-sm text-muted-foreground">No policies match your search or filters.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
