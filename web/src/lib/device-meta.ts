import type { DeviceMeta, DeviceMetaMap } from "@/types/api";

const KEY = "router-sync-device-meta";

export function loadDeviceMeta(): DeviceMetaMap {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return {};
    return JSON.parse(raw) as DeviceMetaMap;
  } catch {
    return {};
  }
}

export function saveDeviceMeta(map: DeviceMetaMap) {
  localStorage.setItem(KEY, JSON.stringify(map));
}

export function getDeviceMeta(policyId: string): DeviceMeta {
  const map = loadDeviceMeta();
  return map[policyId] || { tags: [] };
}

export function updateDeviceMeta(policyId: string, patch: Partial<DeviceMeta>) {
  const map = loadDeviceMeta();
  const current = map[policyId] || { tags: [] };
  map[policyId] = {
    ...current,
    ...patch,
    tags: patch.tags ?? current.tags,
  };
  saveDeviceMeta(map);
  return map[policyId];
}

export function allTags(map: DeviceMetaMap): string[] {
  const tags = new Set<string>();
  Object.values(map).forEach((m) => m.tags?.forEach((t) => tags.add(t)));
  return [...tags].sort();
}
