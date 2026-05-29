import { Badge } from "@/components/ui/badge";
import { PolicyDisplayName } from "@/components/PolicyDisplayName";
import { PolicyMetaSection } from "@/components/PolicyMetaSection";
import { displayPolicyId } from "@/lib/policy-id";
import { normalizeTags } from "@/lib/policy-tags";
import type { RoutingPolicy } from "@/types/api";

type DeviceRowProps = {
  policy: RoutingPolicy;
  providerName: string;
  onToggleFavorite: () => void;
  onRename: (name: string) => void;
  onSaveDescription: (description: string) => void;
  onSaveTags: (tags: string[]) => void;
  disabled?: boolean;
  compact?: boolean;
};

export function DeviceRow({
  policy,
  providerName,
  onToggleFavorite,
  onRename,
  onSaveDescription,
  onSaveTags,
  disabled,
  compact,
}: DeviceRowProps) {
  const tags = normalizeTags(policy.tags);

  return (
    <div
      className={
        compact
          ? "flex flex-wrap items-start justify-between gap-2 rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2"
          : "flex flex-wrap items-start justify-between gap-3 rounded-lg border border-border px-4 py-3"
      }
    >
      <div className="min-w-0 flex-1 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <PolicyDisplayName
            name={policy.name}
            isFavorite={Boolean(policy.favorite)}
            onToggleFavorite={onToggleFavorite}
            onRename={onRename}
            disabled={disabled}
          />
          {policy.enabled ? (
            <Badge variant="default">Override on</Badge>
          ) : (
            <Badge variant="muted">Off</Badge>
          )}
        </div>
        <p className="font-mono text-xs text-muted-foreground pl-10">
          {displayPolicyId(policy.id)}
        </p>
        <PolicyMetaSection
          description={policy.description}
          tags={tags}
          onSaveDescription={onSaveDescription}
          onSaveTags={onSaveTags}
          disabled={disabled}
        />
      </div>
      <div className="text-sm text-muted-foreground">
        <span className="text-xs uppercase tracking-wide">Route</span>
        <p className="font-medium text-foreground">{providerName}</p>
      </div>
    </div>
  );
}
