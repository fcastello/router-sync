/// <reference types="vite/client" />

interface RouterSyncRuntimeConfig {
  apiBaseUrl: string;
}

interface Window {
  __ROUTER_SYNC_CONFIG__?: RouterSyncRuntimeConfig;
}
