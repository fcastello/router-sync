import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { displayPolicyId } from "@/lib/policy-id";
import type { InternetProvider, RoutingPolicy } from "@/types/api";
import { Star, Trash2 } from "lucide-react";

type PolicyRowProps = {
  policy: RoutingPolicy;
  providers: InternetProvider[];
  isFavorite: boolean;
  onToggleFavorite: () => void;
  onToggleEnabled: () => void;
  onChangeProvider: (providerId: string) => void;
  onDelete: () => void;
  updatePending?: boolean;
  compact?: boolean;
};

export function PolicyRow({
  policy,
  providers,
  isFavorite,
  onToggleFavorite,
  onToggleEnabled,
  onChangeProvider,
  onDelete,
  updatePending,
  compact,
}: PolicyRowProps) {
  const isOverride = policy.enabled;

  return (
    <div
      className={
        compact
          ? "flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2"
          : "flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border px-4 py-3"
      }
    >
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="ghost"
            className="h-8 w-8 shrink-0 p-0"
            onClick={onToggleFavorite}
            title={isFavorite ? "Remove from favorites" : "Add to favorites"}
            aria-label={isFavorite ? "Remove from favorites" : "Add to favorites"}
            aria-pressed={isFavorite}
          >
            <Star
              className={`h-4 w-4 ${isFavorite ? "fill-amber-400 text-amber-400" : "text-muted-foreground"}`}
            />
          </Button>
          <span className="font-medium">{policy.name}</span>
          {isOverride ? (
            <Badge variant="default">Override</Badge>
          ) : (
            <Badge variant="muted">Default (disabled)</Badge>
          )}
        </div>
        <p className="font-mono text-xs text-muted-foreground pl-10">
          {displayPolicyId(policy.id)}
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <Select
          value={policy.provider_id}
          onChange={(e) => onChangeProvider(e.target.value)}
          className="w-36"
        >
          {providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </Select>
        <div className="flex items-center gap-2">
          <Switch
            checked={policy.enabled}
            onCheckedChange={onToggleEnabled}
            disabled={updatePending}
          />
          <span className="text-xs text-muted-foreground">{policy.enabled ? "On" : "Off"}</span>
        </div>
        <Button variant="ghost" className="text-destructive" onClick={onDelete}>
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
