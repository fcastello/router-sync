import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import type { InternetProvider } from "@/types/api";
import { X } from "lucide-react";

export type NewPolicyFormState = {
  name: string;
  source_ip: string;
  provider_id: string;
  description: string;
  enabled: boolean;
  favorite: boolean;
};

type NewPolicyFormProps = {
  form: NewPolicyFormState;
  providers: InternetProvider[];
  onChange: (patch: Partial<NewPolicyFormState>) => void;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
  pending?: boolean;
  error?: Error | null;
};

export function NewPolicyForm({
  form,
  providers,
  onChange,
  onSubmit,
  onCancel,
  pending,
  error,
}: NewPolicyFormProps) {
  return (
    <form
      onSubmit={onSubmit}
      className="space-y-3 rounded-lg border border-dashed border-primary/40 bg-muted/30 p-3"
    >
      <div className="flex flex-wrap items-end gap-2 text-sm">
        <span className="text-muted-foreground">Route</span>
        <div className="min-w-[120px] flex-1">
          <Label className="sr-only">Source IP / CIDR</Label>
          <Input
            className="h-8"
            placeholder="192.168.1.50"
            value={form.source_ip}
            onChange={(e) =>
              onChange({
                source_ip: e.target.value,
                name: form.name || e.target.value,
              })
            }
            required
            autoFocus
          />
        </div>
        <span className="text-muted-foreground">named</span>
        <div className="min-w-[100px] flex-1">
          <Input
            className="h-8"
            placeholder="Display name"
            value={form.name}
            onChange={(e) => onChange({ name: e.target.value })}
            required
          />
        </div>
        <span className="text-muted-foreground">via</span>
        <div className="min-w-[120px]">
          <Select
            className="h-8"
            value={form.provider_id}
            onChange={(e) => onChange({ provider_id: e.target.value })}
            required
          >
            <option value="">Uplink…</option>
            {providers.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </Select>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-4">
        <div className="min-w-[160px] flex-1">
          <Input
            className="h-8"
            placeholder="Description (optional)"
            value={form.description}
            onChange={(e) => onChange({ description: e.target.value })}
          />
        </div>
        <label className="flex items-center gap-2 text-xs">
          <Switch
            checked={form.enabled}
            onCheckedChange={(enabled) => onChange({ enabled })}
          />
          Enabled
        </label>
        <label className="flex items-center gap-2 text-xs">
          <Switch
            checked={form.favorite}
            onCheckedChange={(favorite) => onChange({ favorite })}
          />
          Favorite
        </label>
        <div className="ml-auto flex gap-2">
          <Button type="button" variant="ghost" className="h-8 px-2" onClick={onCancel}>
            <X className="mr-1 h-3.5 w-3.5" />
            Cancel
          </Button>
          <Button type="submit" className="h-8 px-3" disabled={pending || !form.provider_id}>
            Add
          </Button>
        </div>
      </div>
      {error && <p className="text-sm text-destructive">{error.message}</p>}
    </form>
  );
}
