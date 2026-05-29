import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { migrateLocalDisplayNames } from "@/lib/migrate-display-names";
import { policyBody } from "@/lib/policy-body";
import { displayPolicyId } from "@/lib/policy-id";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";

export function DevicesPage() {
  const policies = usePolicies();
  const providers = useProviders();
  const { update } = usePolicyMutations();
  const qc = useQueryClient();
  const { data: meta = {} } = useDeviceMeta();
  const [tagFilter, setTagFilter] = useState("");
  const [editing, setEditing] = useState<string | null>(null);
  const [form, setForm] = useState({ name: "", tags: "" });

  const policyList = policies.data ?? [];
  const tags = allTags(meta);

  useEffect(() => {
    if (!policyList.length) return;
    migrateLocalDisplayNames(policyList)
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

  const rows = policyList.map((policy) => ({
    policy,
    tags: meta[policy.id]?.tags ?? [],
  }));

  const filtered = tagFilter ? rows.filter((r) => r.tags.includes(tagFilter)) : rows;

  const startEdit = (policyId: string) => {
    const policy = policyList.find((p) => p.id === policyId);
    if (!policy) return;
    setEditing(policyId);
    setForm({
      name: policy.name,
      tags: (meta[policyId]?.tags ?? []).join(", "),
    });
  };

  const save = () => {
    if (!editing) return;
    const policy = policyList.find((p) => p.id === editing);
    if (!policy) return;

    const name = form.name.trim();
    if (name && name !== policy.name) {
      update.mutate({ id: editing, body: policyBody(policy, { name }) });
    }

    updateDeviceMeta(editing, {
      tags: form.tags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean),
    });
    qc.invalidateQueries({ queryKey: queryKeys.deviceMeta });
    setEditing(null);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Devices</h1>
        <p className="text-sm text-muted-foreground">
          Same policies as on the Policies page. Display names are stored in NATS; tags are
          optional labels kept in this browser only.
        </p>
      </div>

      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
          <CardTitle>Devices ({filtered.length})</CardTitle>
          <div className="flex items-center gap-2">
            <Label>Filter tag</Label>
            <Select
              value={tagFilter}
              onChange={(e) => setTagFilter(e.target.value)}
              className="w-40"
            >
              <option value="">All</option>
              {tags.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </Select>
          </div>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-muted-foreground">
                <th className="pb-2 pr-4">Display name</th>
                <th className="pb-2 pr-4">Source IP / CIDR</th>
                <th className="pb-2 pr-4">Tags</th>
                <th className="pb-2 pr-4">Route</th>
                <th className="pb-2 pr-4">Status</th>
                <th className="pb-2" />
              </tr>
            </thead>
            <tbody>
              {filtered.map(({ policy, tags: rowTags }) => (
                <tr key={policy.id} className="border-b border-border/60">
                  <td className="py-3 pr-4 font-medium">
                    {policy.name}
                    {policy.favorite && (
                      <Badge variant="default" className="ml-2">
                        favorite
                      </Badge>
                    )}
                  </td>
                  <td className="py-3 pr-4 font-mono text-xs">{displayPolicyId(policy.id)}</td>
                  <td className="py-3 pr-4">
                    <div className="flex flex-wrap gap-1">
                      {rowTags.length > 0 ? (
                        rowTags.map((t) => (
                          <Badge key={t} variant="default">
                            {t}
                          </Badge>
                        ))
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </div>
                  </td>
                  <td className="py-3 pr-4">
                    {providerMap.get(policy.provider_id) ?? policy.provider_id}
                  </td>
                  <td className="py-3 pr-4">
                    {policy.enabled ? (
                      <Badge variant="default">Override on</Badge>
                    ) : (
                      <Badge variant="muted">Off</Badge>
                    )}
                  </td>
                  <td className="py-3">
                    <Button variant="ghost" className="h-8 px-2" onClick={() => startEdit(policy.id)}>
                      Edit
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {filtered.length === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No policies yet. Create one on the Policies page.
            </p>
          )}
        </CardContent>
      </Card>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit device</CardTitle>
          </CardHeader>
          <CardContent className="grid max-w-md gap-3">
            <div>
              <Label>Display name</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                required
              />
              <p className="mt-1 text-xs text-muted-foreground">
                Saved to NATS — also shown on the Policies page.
              </p>
            </div>
            <div>
              <Label>Tags (comma-separated, browser only)</Label>
              <Input
                value={form.tags}
                onChange={(e) => setForm((f) => ({ ...f, tags: e.target.value }))}
                placeholder="IoT, Kids, Work"
              />
            </div>
            <div className="flex gap-2">
              <Button onClick={save} disabled={update.isPending || !form.name.trim()}>
                Save
              </Button>
              <Button variant="outline" onClick={() => setEditing(null)}>
                Cancel
              </Button>
            </div>
            {update.isError && (
              <p className="text-sm text-destructive">{(update.error as Error).message}</p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
