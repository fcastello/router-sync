import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PolicyDisplayName } from "@/components/PolicyDisplayName";
import { displayPolicyId } from "@/lib/policy-id";
import type { RoutingPolicy } from "@/types/api";
import { Pencil, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type DeviceRowProps = {
  policy: RoutingPolicy;
  tags: string[];
  providerName: string;
  onToggleFavorite: () => void;
  onRename: (name: string) => void;
  onSaveTags: (tags: string[]) => void;
  disabled?: boolean;
  compact?: boolean;
};

export function DeviceRow({
  policy,
  tags,
  providerName,
  onToggleFavorite,
  onRename,
  onSaveTags,
  disabled,
  compact,
}: DeviceRowProps) {
  const [editingTags, setEditingTags] = useState(false);
  const [draftTags, setDraftTags] = useState(tags.join(", "));
  const tagsInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editingTags) {
      setDraftTags(tags.join(", "));
    }
  }, [tags, editingTags]);

  useEffect(() => {
    if (editingTags) {
      tagsInputRef.current?.focus();
      tagsInputRef.current?.select();
    }
  }, [editingTags]);

  const startEditingTags = () => {
    if (disabled) return;
    setDraftTags(tags.join(", "));
    setEditingTags(true);
  };

  const cancelEditingTags = () => {
    setDraftTags(tags.join(", "));
    setEditingTags(false);
  };

  const commitTags = () => {
    setEditingTags(false);
    const next = draftTags
      .split(",")
      .map((t) => t.trim())
      .filter(Boolean);
    const same =
      next.length === tags.length && next.every((t, i) => t === tags[i]);
    if (!same) {
      onSaveTags(next);
    }
  };

  return (
    <div
      className={
        compact
          ? "flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2"
          : "flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border px-4 py-3"
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
        <div className="flex flex-wrap items-center gap-2 pl-10">
          {editingTags ? (
            <div className="flex min-w-0 flex-1 items-center gap-1">
              <Input
                ref={tagsInputRef}
                className="h-8 max-w-md text-sm"
                value={draftTags}
                onChange={(e) => setDraftTags(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    commitTags();
                  }
                  if (e.key === "Escape") {
                    e.preventDefault();
                    cancelEditingTags();
                  }
                }}
                onBlur={commitTags}
                disabled={disabled}
                placeholder="IoT, Kids, Work"
                aria-label="Tags"
              />
              <Button
                type="button"
                variant="ghost"
                className="h-8 w-8 shrink-0 p-0"
                onMouseDown={(e) => e.preventDefault()}
                onClick={cancelEditingTags}
                title="Cancel"
                aria-label="Cancel tag edit"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <div className="flex min-w-0 flex-wrap items-center gap-1">
              <div className="flex flex-wrap gap-1">
                {tags.length > 0 ? (
                  tags.map((t) => (
                    <Badge key={t} variant="default">
                      {t}
                    </Badge>
                  ))
                ) : (
                  <span className="text-xs text-muted-foreground">No tags</span>
                )}
              </div>
              <Button
                type="button"
                variant="ghost"
                className="h-8 w-8 shrink-0 p-0 text-muted-foreground"
                onClick={startEditingTags}
                title="Edit tags"
                aria-label="Edit tags"
                disabled={disabled}
              >
                <Pencil className="h-3.5 w-3.5" />
              </Button>
            </div>
          )}
        </div>
      </div>
      <div className="text-sm text-muted-foreground">
        <span className="text-xs uppercase tracking-wide">Route</span>
        <p className="font-medium text-foreground">{providerName}</p>
      </div>
    </div>
  );
}
