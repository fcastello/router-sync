import { api } from "@/lib/api";
import { clearDeviceMeta, loadDeviceMeta } from "@/lib/device-meta";
import { policyBody } from "@/lib/policy-body";
import { normalizeTags } from "@/lib/policy-tags";
import type { RoutingPolicy } from "@/types/api";

const MIGRATION_KEY = "router-sync-tags-migrated-v1";

/** One-time: push browser-local tags from device-meta into NATS-backed policies. */
export async function migrateLocalPolicyTags(policies: RoutingPolicy[]): Promise<void> {
  if (localStorage.getItem(MIGRATION_KEY)) {
    return;
  }

  const meta = loadDeviceMeta();
  let migrated = false;

  for (const policy of policies) {
    const localTags = normalizeTags(meta[policy.id]?.tags);
    if (localTags.length === 0) {
      continue;
    }
    const serverTags = normalizeTags(policy.tags);
    if (serverTags.length > 0) {
      continue;
    }
    await api.updatePolicy(policy.id, policyBody(policy, { tags: localTags }));
    migrated = true;
  }

  if (migrated || Object.keys(meta).length > 0) {
    clearDeviceMeta();
  }
  localStorage.setItem(MIGRATION_KEY, "1");
}
