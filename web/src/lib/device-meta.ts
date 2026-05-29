/** Legacy browser-local metadata (tags/favorites/display names). Used only by one-time migrations. */

const KEY = "router-sync-device-meta";

export type LegacyDeviceMeta = {
  tags?: string[];
  favorite?: boolean;
  friendlyName?: string;
  mac?: string;
};

export type LegacyDeviceMetaMap = Record<string, LegacyDeviceMeta>;

export function loadDeviceMeta(): LegacyDeviceMetaMap {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return {};
    return JSON.parse(raw) as LegacyDeviceMetaMap;
  } catch {
    return {};
  }
}

export function saveDeviceMeta(map: LegacyDeviceMetaMap) {
  localStorage.setItem(KEY, JSON.stringify(map));
}

export function clearDeviceMeta() {
  localStorage.removeItem(KEY);
}
