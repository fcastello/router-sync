import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { usePolicies, useProviders } from "@/hooks/useRouterSync";
import { allTags, loadDeviceMeta, updateDeviceMeta } from "@/lib/device-meta";
import { displayPolicyId } from "@/lib/policy-id";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/hooks/useRouterSync";
import { useMemo, useState } from "react";

export function DevicesPage() {
  const policies = usePolicies();
  const providers = useProviders();
  const qc = useQueryClient();
  const [tagFilter, setTagFilter] = useState("");
  const [editing, setEditing] = useState<string | null>(null);
  const [form, setForm] = useState({ friendlyName: "", mac: "", tags: "" });

  const meta = loadDeviceMeta();
  const tags = allTags(meta);

  const providerMap = useMemo(() => {
    const m = new Map<string, string>();
    (providers.data ?? []).forEach((p) => m.set(p.id, p.name));
    return m;
  }, [providers.data]);

  const rows = (policies.data ?? []).map((p) => {
    const dm = meta[p.id] || { tags: [] };
    return { policy: p, meta: dm };
  });

  const filtered = tagFilter
    ? rows.filter((r) => r.meta.tags?.includes(tagFilter))
    : rows;

  const startEdit = (policyId: string) => {
    const dm = meta[policyId] || { tags: [] };
    setEditing(policyId);
    setForm({
      friendlyName: dm.friendlyName ?? "",
      mac: dm.mac ?? "",
      tags: (dm.tags ?? []).join(", "),
    });
  };

  const save = () => {
    if (!editing) return;
    updateDeviceMeta(editing, {
      friendlyName: form.friendlyName || undefined,
      mac: form.mac || undefined,
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
        <h1 className="text-2xl font-semibold">Device registry</h1>
        <p className="text-sm text-muted-foreground">
          Policies map source IPs/CIDRs to routes. Friendly names and tags are stored in this browser until a backend registry exists.
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
                <th className="pb-2 pr-4">Friendly name</th>
                <th className="pb-2 pr-4">Source IP / CIDR</th>
                <th className="pb-2 pr-4">MAC</th>
                <th className="pb-2 pr-4">Tags</th>
                <th className="pb-2 pr-4">Route</th>
                <th className="pb-2">Status</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(({ policy, meta: dm }) => (
                <tr key={policy.id} className="border-b border-border/60">
                  <td className="py-3 pr-4 font-medium">
                    {dm.friendlyName || policy.name}
                    {!policy.enabled && (
                      <Badge variant="muted" className="ml-2">
                        policy off
                      </Badge>
                    )}
                  </td>
                  <td className="py-3 pr-4 font-mono text-xs">
                    {displayPolicyId(policy.id)}
                  </td>
                  <td className="py-3 pr-4 text-xs">{dm.mac || "—"}</td>
                  <td className="py-3 pr-4">
                    <div className="flex flex-wrap gap-1">
                      {(dm.tags ?? []).map((t) => (
                        <Badge key={t} variant="default">
                          {t}
                        </Badge>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 pr-4">
                    {providerMap.get(policy.provider_id) ?? policy.provider_id}
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
              No policies yet. Create one on the Policies page to register a device.
            </p>
          )}
        </CardContent>
      </Card>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit device metadata</CardTitle>
          </CardHeader>
          <CardContent className="grid max-w-md gap-3">
            <div>
              <Label>Friendly name</Label>
              <Input
                value={form.friendlyName}
                onChange={(e) => setForm((f) => ({ ...f, friendlyName: e.target.value }))}
              />
            </div>
            <div>
              <Label>MAC (local only)</Label>
              <Input
                value={form.mac}
                onChange={(e) => setForm((f) => ({ ...f, mac: e.target.value }))}
                placeholder="aa:bb:cc:dd:ee:ff"
              />
            </div>
            <div>
              <Label>Tags (comma-separated)</Label>
              <Input
                value={form.tags}
                onChange={(e) => setForm((f) => ({ ...f, tags: e.target.value }))}
                placeholder="IoT, Kids, Work"
              />
            </div>
            <div className="flex gap-2">
              <Button onClick={save}>Save</Button>
              <Button variant="outline" onClick={() => setEditing(null)}>
                Cancel
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
