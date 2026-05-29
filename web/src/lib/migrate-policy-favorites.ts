import { api } from "@/lib/api";
import { loadDeviceMeta, saveDeviceMeta } from "@/lib/device-meta";
import type { RoutingPolicy } from "@/types/api";

const MIGRATION_KEY = "router-sync-favorites-migrated-v2";

/** One-time: push browser-local favorites from device-meta into NATS-backed policies. */
export async function migrateLocalPolicyFavorites(policies: RoutingPolicy[]): Promise<void> {
  if (localStorage.getItem(MIGRATION_KEY)) {
    return;
  }

  const meta = loadDeviceMeta();
  const toMigrate = policies.filter((p) => {
    const legacy = meta[p.id] as { favorite?: boolean } | undefined;
    return legacy?.favorite && !p.favorite;
  });
  if (toMigrate.length === 0) {
    localStorage.setItem(MIGRATION_KEY, "1");
    return;
  }

  const nextMeta = { ...meta };
  for (const policy of toMigrate) {
    await api.updatePolicy(policy.id, {
      name: policy.name,
      source_ip: policy.id,
      provider_id: policy.provider_id,
      description: policy.description,
      enabled: policy.enabled,
      favorite: true,
    });
    const entry = nextMeta[policy.id];
    if (entry) {
      const { favorite: _removed, ...rest } = entry;
      if (Object.keys(rest).length > 0) {
        nextMeta[policy.id] = rest;
      } else {
        delete nextMeta[policy.id];
      }
    }
  }
  saveDeviceMeta(nextMeta);
  localStorage.setItem(MIGRATION_KEY, "1");
}
