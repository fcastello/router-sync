import { displayPolicyId } from "@/lib/policy-id";
import type { RoutingPolicy } from "@/types/api";

export type PolicySortKey = "name" | "subnet" | "status";

function parseSubnetKey(id: string): { prefixLen: number; ip: string } {
  const display = displayPolicyId(id);
  if (display.includes("/")) {
    const [ip, mask] = display.split("/");
    const bits = parseInt(mask, 10);
    return { prefixLen: Number.isNaN(bits) ? 32 : bits, ip };
  }
  return { prefixLen: 32, ip: display };
}

function compareIp(a: string, b: string): number {
  const pa = a.split(".").map((x) => parseInt(x, 10) || 0);
  const pb = b.split(".").map((x) => parseInt(x, 10) || 0);
  for (let i = 0; i < 4; i++) {
    const d = (pa[i] ?? 0) - (pb[i] ?? 0);
    if (d !== 0) return d;
  }
  return 0;
}

export function sortPolicies(
  policies: RoutingPolicy[],
  sortBy: PolicySortKey,
  statusDesc = true,
): RoutingPolicy[] {
  const copy = [...policies];
  copy.sort((a, b) => {
    switch (sortBy) {
      case "name":
        return a.name.localeCompare(b.name, undefined, { sensitivity: "base" });
      case "status": {
        if (a.enabled !== b.enabled) {
          const av = a.enabled ? 1 : 0;
          const bv = b.enabled ? 1 : 0;
          return statusDesc ? bv - av : av - bv;
        }
        return a.name.localeCompare(b.name);
      }
      case "subnet": {
        const sa = parseSubnetKey(a.id);
        const sb = parseSubnetKey(b.id);
        if (sa.prefixLen !== sb.prefixLen) return sb.prefixLen - sa.prefixLen;
        const ipCmp = compareIp(sa.ip, sb.ip);
        if (ipCmp !== 0) return ipCmp;
        return a.name.localeCompare(b.name);
      }
      default:
        return 0;
    }
  });
  return copy;
}
