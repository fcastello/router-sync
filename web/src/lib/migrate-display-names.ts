import { api } from "@/lib/api";
import { loadDeviceMeta, saveDeviceMeta } from "@/lib/device-meta";
import { policyBody } from "@/lib/policy-body";
import type { RoutingPolicy } from "@/types/api";

const MIGRATION_KEY = "router-sync-display-names-migrated-v1";

type LegacyMeta = {
  friendlyName?: string;
  mac?: string;
  tags?: string[];
};

/** One-time: copy browser-local friendly names into NATS policy.name, then strip legacy fields. */
export async function migrateLocalDisplayNames(policies: RoutingPolicy[]): Promise<void> {
  if (localStorage.getItem(MIGRATION_KEY)) {
    return;
  }

  const meta = loadDeviceMeta();
  const nextMeta = { ...meta };

  for (const policy of policies) {
    const legacy = meta[policy.id] as LegacyMeta | undefined;
    const displayName = legacy?.friendlyName?.trim();
    if (displayName && displayName !== policy.name) {
      await api.updatePolicy(policy.id, policyBody(policy, { name: displayName }));
    }

    if (legacy) {
      const { friendlyName: _fn, mac: _mac, ...rest } = legacy;
      const hasTags = (rest.tags?.length ?? 0) > 0;
      if (hasTags) {
        nextMeta[policy.id] = { tags: rest.tags ?? [] };
      } else {
        delete nextMeta[policy.id];
      }
    }
  }

  saveDeviceMeta(nextMeta);
  localStorage.setItem(MIGRATION_KEY, "1");
}
