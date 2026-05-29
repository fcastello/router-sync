import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { displayPolicyId } from "@/lib/policy-id";
import type { InternetProvider, RoutingPolicy } from "@/types/api";
import { Pencil, Star, Trash2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type PolicyRowProps = {
  policy: RoutingPolicy;
  providers: InternetProvider[];
  isFavorite: boolean;
  onToggleFavorite: () => void;
  onToggleEnabled: () => void;
  onChangeProvider: (providerId: string) => void;
  onRename: (name: string) => void;
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
  onRename,
  onDelete,
  updatePending,
  compact,
}: PolicyRowProps) {
  const isOverride = policy.enabled;
  const [editingName, setEditingName] = useState(false);
  const [draftName, setDraftName] = useState(policy.name);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editingName) {
      setDraftName(policy.name);
    }
  }, [policy.name, editingName]);

  useEffect(() => {
    if (editingName) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editingName]);

  const startEditing = () => {
    if (updatePending) return;
    setDraftName(policy.name);
    setEditingName(true);
  };

  const cancelEditing = () => {
    setDraftName(policy.name);
    setEditingName(false);
  };

  const commitRename = () => {
    const trimmed = draftName.trim();
    setEditingName(false);
    if (!trimmed) {
      setDraftName(policy.name);
      return;
    }
    if (trimmed !== policy.name) {
      onRename(trimmed);
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
          {editingName ? (
            <div className="flex min-w-0 flex-1 items-center gap-1">
              <Input
                ref={inputRef}
                className="h-8 max-w-xs text-sm"
                value={draftName}
                onChange={(e) => setDraftName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    commitRename();
                  }
                  if (e.key === "Escape") {
                    e.preventDefault();
                    cancelEditing();
                  }
                }}
                onBlur={commitRename}
                disabled={updatePending}
                aria-label="Display name"
              />
              <Button
                type="button"
                variant="ghost"
                className="h-8 w-8 shrink-0 p-0"
                onMouseDown={(e) => e.preventDefault()}
                onClick={cancelEditing}
                title="Cancel"
                aria-label="Cancel rename"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <div className="flex min-w-0 items-center gap-1">
              <span className="truncate font-medium">{policy.name}</span>
              <Button
                type="button"
                variant="ghost"
                className="h-8 w-8 shrink-0 p-0 text-muted-foreground"
                onClick={startEditing}
                title="Edit display name"
                aria-label="Edit display name"
                disabled={updatePending}
              >
                <Pencil className="h-3.5 w-3.5" />
              </Button>
            </div>
          )}
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
          disabled={updatePending}
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
        <Button variant="ghost" className="text-destructive" onClick={onDelete} disabled={updatePending}>
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
