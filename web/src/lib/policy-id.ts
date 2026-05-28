/** API path id: CIDR policies may use `_` instead of `/` in URLs. */
export function policyIdForPath(id: string): string {
  if (id.includes("/")) return id.replace(/\//g, "_");
  return encodeURIComponent(id);
}

export function displayPolicyId(id: string): string {
  if (id.includes("_") && !id.includes("/")) {
    const parts = id.split("_");
    if (parts.length === 2 && parts[1].match(/^\d+$/)) {
      return `${parts[0]}/${parts[1]}`;
    }
  }
  return id;
}

export function normalizeSourceIp(input: string): string {
  return input.trim();
}
