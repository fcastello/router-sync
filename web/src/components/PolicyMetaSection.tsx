import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { parseTagsInput, tagsToInput } from "@/lib/policy-tags";
import { Pencil, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type PolicyMetaSectionProps = {
  description?: string;
  tags?: string[];
  onSaveDescription: (description: string) => void;
  onSaveTags: (tags: string[]) => void;
  disabled?: boolean;
  className?: string;
};

export function PolicyMetaSection({
  description = "",
  tags = [],
  onSaveDescription,
  onSaveTags,
  disabled,
  className = "pl-10",
}: PolicyMetaSectionProps) {
  const [editingDescription, setEditingDescription] = useState(false);
  const [draftDescription, setDraftDescription] = useState(description);
  const descriptionRef = useRef<HTMLInputElement>(null);

  const [editingTags, setEditingTags] = useState(false);
  const [draftTags, setDraftTags] = useState(tagsToInput(tags));
  const tagsInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editingDescription) {
      setDraftDescription(description);
    }
  }, [description, editingDescription]);

  useEffect(() => {
    if (!editingTags) {
      setDraftTags(tagsToInput(tags));
    }
  }, [tags, editingTags]);

  useEffect(() => {
    if (editingDescription) {
      descriptionRef.current?.focus();
      descriptionRef.current?.select();
    }
  }, [editingDescription]);

  useEffect(() => {
    if (editingTags) {
      tagsInputRef.current?.focus();
      tagsInputRef.current?.select();
    }
  }, [editingTags]);

  const commitDescription = () => {
    setEditingDescription(false);
    const next = draftDescription.trim();
    if (next !== description.trim()) {
      onSaveDescription(next);
    }
  };

  const commitTags = () => {
    setEditingTags(false);
    const next = parseTagsInput(draftTags);
    const current = parseTagsInput(tagsToInput(tags));
    const same =
      next.length === current.length && next.every((t, i) => t === current[i]);
    if (!same) {
      onSaveTags(next);
    }
  };

  return (
    <div className={`space-y-2 ${className}`}>
      <div className="flex flex-wrap items-center gap-1">
        {editingDescription ? (
          <div className="flex min-w-0 flex-1 items-center gap-1">
            <Input
              ref={descriptionRef}
              className="h-8 max-w-md text-sm"
              value={draftDescription}
              onChange={(e) => setDraftDescription(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  commitDescription();
                }
                if (e.key === "Escape") {
                  e.preventDefault();
                  setDraftDescription(description);
                  setEditingDescription(false);
                }
              }}
              onBlur={commitDescription}
              disabled={disabled}
              placeholder="Description (optional)"
              aria-label="Description"
            />
            <Button
              type="button"
              variant="ghost"
              className="h-8 w-8 shrink-0 p-0"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                setDraftDescription(description);
                setEditingDescription(false);
              }}
              title="Cancel"
              aria-label="Cancel description edit"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        ) : (
          <div className="flex min-w-0 flex-1 items-center gap-1">
            {description.trim() ? (
              <p className="text-sm text-muted-foreground">{description}</p>
            ) : (
              <span className="text-xs text-muted-foreground">No description</span>
            )}
            <Button
              type="button"
              variant="ghost"
              className="h-8 w-8 shrink-0 p-0 text-muted-foreground"
              onClick={() => {
                if (disabled) return;
                setDraftDescription(description);
                setEditingDescription(true);
              }}
              title="Edit description"
              aria-label="Edit description"
              disabled={disabled}
            >
              <Pencil className="h-3.5 w-3.5" />
            </Button>
          </div>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-1">
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
                  setDraftTags(tagsToInput(tags));
                  setEditingTags(false);
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
              onClick={() => {
                setDraftTags(tagsToInput(tags));
                setEditingTags(false);
              }}
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
              onClick={() => {
                if (disabled) return;
                setDraftTags(tagsToInput(tags));
                setEditingTags(true);
              }}
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
  );
}
