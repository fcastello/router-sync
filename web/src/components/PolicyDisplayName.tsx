import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Pencil, Star, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type PolicyDisplayNameProps = {
  name: string;
  isFavorite: boolean;
  onToggleFavorite: () => void;
  onRename: (name: string) => void;
  disabled?: boolean;
};

export function PolicyDisplayName({
  name,
  isFavorite,
  onToggleFavorite,
  onRename,
  disabled,
}: PolicyDisplayNameProps) {
  const [editingName, setEditingName] = useState(false);
  const [draftName, setDraftName] = useState(name);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editingName) {
      setDraftName(name);
    }
  }, [name, editingName]);

  useEffect(() => {
    if (editingName) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editingName]);

  const startEditing = () => {
    if (disabled) return;
    setDraftName(name);
    setEditingName(true);
  };

  const cancelEditing = () => {
    setDraftName(name);
    setEditingName(false);
  };

  const commitRename = () => {
    const trimmed = draftName.trim();
    setEditingName(false);
    if (!trimmed) {
      setDraftName(name);
      return;
    }
    if (trimmed !== name) {
      onRename(trimmed);
    }
  };

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2">
      <Button
        type="button"
        variant="ghost"
        className="h-8 w-8 shrink-0 p-0"
        onClick={onToggleFavorite}
        title={isFavorite ? "Remove from favorites" : "Add to favorites"}
        aria-label={isFavorite ? "Remove from favorites" : "Add to favorites"}
        aria-pressed={isFavorite}
        disabled={disabled}
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
            disabled={disabled}
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
          <span className="truncate font-medium">{name}</span>
          <Button
            type="button"
            variant="ghost"
            className="h-8 w-8 shrink-0 p-0 text-muted-foreground"
            onClick={startEditing}
            title="Edit display name"
            aria-label="Edit display name"
            disabled={disabled}
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
        </div>
      )}
    </div>
  );
}
