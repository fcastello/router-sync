import type { RoutingPolicy } from "@/types/api";

export function normalizeTags(tags: string[] | undefined): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const t of tags ?? []) {
    const trimmed = t.trim();
    if (!trimmed || seen.has(trimmed)) continue;
    seen.add(trimmed);
    out.push(trimmed);
  }
  return out.sort();
}

export function parseTagsInput(value: string): string[] {
  return normalizeTags(value.split(","));
}

export function tagsToInput(tags: string[] | undefined): string {
  return normalizeTags(tags).join(", ");
}

export function allPolicyTags(policies: RoutingPolicy[]): string[] {
  const seen = new Set<string>();
  for (const p of policies) {
    for (const t of normalizeTags(p.tags)) {
      seen.add(t);
    }
  }
  return [...seen].sort();
}
