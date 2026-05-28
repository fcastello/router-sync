import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  useProviderMutations,
  useProviders,
  useRouters,
} from "@/hooks/useRouterSync";
import type { CreateProviderRequest, InternetProvider } from "@/types/api";
import { useMemo, useState } from "react";
import { AlertTriangle, Pencil, Plus, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";

interface FormState {
  name: string;
  interfaces: Record<string, string>;
  table_id: number;
  gateway: string;
  description: string;
}

const empty: FormState = {
  name: "",
  interfaces: {},
  table_id: 100,
  gateway: "",
  description: "",
};

function providerInterfaces(p: InternetProvider): Record<string, string> {
  if (p.interfaces && Object.keys(p.interfaces).length > 0) return p.interfaces;
  if (p.interface) return { _legacy: p.interface };
  return {};
}

export function ProvidersPage() {
  const providers = useProviders();
  const routers = useRouters();
  const { create, update, remove } = useProviderMutations();
  const [form, setForm] = useState<FormState>(empty);
  const [editId, setEditId] = useState<string | null>(null);
  const [extraHost, setExtraHost] = useState("");

  const list = providers.data ?? [];
  const routerList = routers.data ?? [];

  const knownHosts = useMemo(() => {
    const set = new Set<string>();
    routerList.forEach((r) => set.add(r.hostname));
    list.forEach((p) => Object.keys(p.interfaces ?? {}).forEach((h) => set.add(h)));
    Object.keys(form.interfaces).forEach((h) => set.add(h));
    return Array.from(set).sort();
  }, [routerList, list, form.interfaces]);

  const buildPayload = (state: FormState): CreateProviderRequest => {
    const cleaned: Record<string, string> = {};
    Object.entries(state.interfaces).forEach(([h, name]) => {
      const trimmed = (name ?? "").trim();
      if (h && trimmed) cleaned[h] = trimmed;
    });
    return {
      name: state.name,
      interfaces: cleaned,
      table_id: state.table_id,
      gateway: state.gateway,
      description: state.description,
    };
  };

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const body = buildPayload(form);
    if (editId) {
      update.mutate(
        { id: editId, body },
        {
          onSuccess: () => {
            setEditId(null);
            setForm(empty);
          },
        },
      );
    } else {
      create.mutate(body, { onSuccess: () => setForm(empty) });
    }
  };

  const startEdit = (id: string) => {
    const p = list.find((x) => x.id === id);
    if (!p) return;
    setEditId(id);
    setForm({
      name: p.name,
      interfaces: { ...(p.interfaces ?? {}) },
      table_id: p.table_id,
      gateway: p.gateway,
      description: p.description ?? "",
    });
  };

  const setIface = (host: string, value: string) => {
    setForm((f) => ({ ...f, interfaces: { ...f.interfaces, [host]: value } }));
  };

  const removeIface = (host: string) => {
    setForm((f) => {
      const next = { ...f.interfaces };
      delete next[host];
      return { ...f, interfaces: next };
    });
  };

  const addCustomHost = () => {
    const h = extraHost.trim();
    if (!h) return;
    setForm((f) => ({ ...f, interfaces: { ...f.interfaces, [h]: "" } }));
    setExtraHost("");
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Internet providers</h1>
        <p className="text-sm text-muted-foreground">
          Uplinks (WAN, VPN, etc.) that policies can target. Each provider can
          map to a different interface name per router.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{editId ? "Edit provider" : "Add provider"}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="grid gap-3 md:grid-cols-2">
            <div>
              <Label>Name (becomes ID)</Label>
              <Input
                value={form.name}
                onChange={(e) =>
                  setForm((f) => ({ ...f, name: e.target.value }))
                }
                required
              />
            </div>
            <div>
              <Label>Routing table ID</Label>
              <Input
                type="number"
                min={1}
                value={form.table_id}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    table_id: parseInt(e.target.value, 10) || 1,
                  }))
                }
                required
              />
            </div>
            <div>
              <Label>Gateway</Label>
              <Input
                value={form.gateway}
                onChange={(e) =>
                  setForm((f) => ({ ...f, gateway: e.target.value }))
                }
                placeholder="192.168.1.1"
                required
              />
            </div>
            <div>
              <Label>Description</Label>
              <Input
                value={form.description}
                onChange={(e) =>
                  setForm((f) => ({ ...f, description: e.target.value }))
                }
              />
            </div>

            <div className="md:col-span-2 space-y-2 rounded-md border border-border p-3">
              <div className="flex items-center justify-between gap-2">
                <Label className="m-0">Interfaces per router</Label>
                <span className="text-xs text-muted-foreground">
                  Different routers can use different interface names for the
                  same logical uplink.
                </span>
              </div>

              {knownHosts.length === 0 && (
                <p className="text-xs text-muted-foreground">
                  No routers have reported state yet. Add a custom host below to
                  pre-populate this provider.
                </p>
              )}

              <div className="grid gap-2 md:grid-cols-2">
                {knownHosts.map((host) => (
                  <div key={host} className="flex items-end gap-2">
                    <div className="flex-1">
                      <Label className="text-xs">{host}</Label>
                      <Input
                        value={form.interfaces[host] ?? ""}
                        onChange={(e) => setIface(host, e.target.value)}
                        placeholder="eth0"
                      />
                    </div>
                    {host in form.interfaces && (
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={() => removeIface(host)}
                        title="Remove this host mapping"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>

              <div className="flex items-end gap-2 pt-2">
                <div className="flex-1">
                  <Label className="text-xs">Add custom host</Label>
                  <Input
                    value={extraHost}
                    onChange={(e) => setExtraHost(e.target.value)}
                    placeholder="r3"
                  />
                </div>
                <Button type="button" variant="outline" onClick={addCustomHost}>
                  <Plus className="mr-1 h-4 w-4" />
                  Add host
                </Button>
              </div>
            </div>

            <div className="flex gap-2 md:col-span-2">
              <Button
                type="submit"
                disabled={create.isPending || update.isPending}
              >
                {editId ? "Save changes" : "Add provider"}
              </Button>
              {editId && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setEditId(null);
                    setForm(empty);
                  }}
                >
                  Cancel
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Configured uplinks</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {list.map((p) => {
            const ifaces = providerInterfaces(p);
            const ifEntries = Object.entries(ifaces);
            const reportingHosts = routerList.map((r) => r.hostname);
            const missing = reportingHosts.filter(
              (h) => !p.interfaces || !p.interfaces[h],
            );
            return (
              <div
                key={p.id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border px-3 py-2"
              >
                <div className="space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-medium">{p.name}</p>
                    {missing.length > 0 && (
                      <Badge variant="warn" className="gap-1">
                        <AlertTriangle className="h-3 w-3" />
                        missing {missing.join(", ")}
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    id: {p.id} · table {p.table_id} · gw {p.gateway}
                  </p>
                  <div className="flex flex-wrap gap-1 text-xs">
                    {ifEntries.length === 0 && (
                      <span className="text-muted-foreground">
                        no interface assigned
                      </span>
                    )}
                    {ifEntries.map(([host, name]) => (
                      <Badge key={host} variant="secondary">
                        {host === "_legacy" ? "legacy" : host}: {name}
                      </Badge>
                    ))}
                  </div>
                </div>
                <div className="flex gap-1">
                  <Button variant="ghost" onClick={() => startEdit(p.id)}>
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    className="text-destructive"
                    onClick={() => {
                      if (confirm(`Delete provider ${p.name}?`))
                        remove.mutate(p.id);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            );
          })}
          {list.length === 0 && (
            <p className="text-sm text-muted-foreground">No providers yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
