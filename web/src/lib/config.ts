const STORAGE_KEY = "router-sync-api-base";

export function getApiBaseUrl(): string {
  const runtime = window.__ROUTER_SYNC_CONFIG__?.apiBaseUrl?.trim();
  if (runtime) return runtime.replace(/\/$/, "");

  const env = import.meta.env.VITE_API_BASE_URL?.trim();
  if (env) return env.replace(/\/$/, "");

  // In dev, prefer the Vite proxy (same origin). A saved Settings URL bypasses the proxy
  // and triggers cross-origin requests (needs CORS on the API).
  if (import.meta.env.DEV) {
    return "";
  }

  const stored = localStorage.getItem(STORAGE_KEY)?.trim();
  if (stored) return stored.replace(/\/$/, "");

  return "";
}

export function clearApiBaseUrl() {
  localStorage.removeItem(STORAGE_KEY);
  if (window.__ROUTER_SYNC_CONFIG__) {
    window.__ROUTER_SYNC_CONFIG__.apiBaseUrl = "";
  }
}

export function setApiBaseUrl(url: string) {
  const normalized = url.trim().replace(/\/$/, "");
  localStorage.setItem(STORAGE_KEY, normalized);
  window.__ROUTER_SYNC_CONFIG__ = {
    ...(window.__ROUTER_SYNC_CONFIG__ || { apiBaseUrl: "" }),
    apiBaseUrl: normalized,
  };
}
