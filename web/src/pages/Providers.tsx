import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useProviderMutations, useProviders } from "@/hooks/useRouterSync";
import type { CreateProviderRequest } from "@/types/api";
import { useState } from "react";
import { Pencil, Trash2 } from "lucide-react";

const empty: CreateProviderRequest = {
  name: "",
  interface: "",
  table_id: 100,
  gateway: "",
  description: "",
};

export function ProvidersPage() {
  const providers = useProviders();
  const { create, update, remove } = useProviderMutations();
  const [form, setForm] = useState(empty);
  const [editId, setEditId] = useState<string | null>(null);

  const list = providers.data ?? [];

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editId) {
      update.mutate(
        { id: editId, body: form },
        { onSuccess: () => { setEditId(null); setForm(empty); } },
      );
    } else {
      create.mutate(form, { onSuccess: () => setForm(empty) });
    }
  };

  const startEdit = (id: string) => {
    const p = list.find((x) => x.id === id);
    if (!p) return;
    setEditId(id);
    setForm({
      name: p.name,
      interface: p.interface,
      table_id: p.table_id,
      gateway: p.gateway,
      description: p.description ?? "",
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Internet providers</h1>
        <p className="text-sm text-muted-foreground">
          Uplinks (WAN, VPN, etc.) that policies can target. Provider ID is set from the name on create.
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
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                required
              />
            </div>
            <div>
              <Label>Interface</Label>
              <Input
                value={form.interface}
                onChange={(e) => setForm((f) => ({ ...f, interface: e.target.value }))}
                placeholder="eth0"
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
                  setForm((f) => ({ ...f, table_id: parseInt(e.target.value, 10) || 1 }))
                }
                required
              />
            </div>
            <div>
              <Label>Gateway</Label>
              <Input
                value={form.gateway}
                onChange={(e) => setForm((f) => ({ ...f, gateway: e.target.value }))}
                placeholder="192.168.1.1"
                required
              />
            </div>
            <div className="md:col-span-2">
              <Label>Description</Label>
              <Input
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              />
            </div>
            <div className="flex gap-2 md:col-span-2">
              <Button type="submit" disabled={create.isPending || update.isPending}>
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
          {list.map((p) => (
            <div
              key={p.id}
              className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border px-3 py-2"
            >
              <div>
                <p className="font-medium">{p.name}</p>
                <p className="text-xs text-muted-foreground">
                  id: {p.id} · {p.interface} · table {p.table_id} · {p.gateway}
                </p>
              </div>
              <div className="flex gap-1">
                <Button variant="ghost" onClick={() => startEdit(p.id)}>
                  <Pencil className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  className="text-destructive"
                  onClick={() => {
                    if (confirm(`Delete provider ${p.name}?`)) remove.mutate(p.id);
                  }}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
          {list.length === 0 && (
            <p className="text-sm text-muted-foreground">No providers yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
