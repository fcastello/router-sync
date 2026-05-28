/** True if every whitespace-separated term appears in the combined fields (case-insensitive). */
export function fuzzyMatch(query: string, ...fields: (string | undefined)[]): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  const hay = fields.filter(Boolean).join(" ").toLowerCase();
  return q.split(/\s+/).every((term) => hay.includes(term));
}
