export interface InternetProvider {
  id: string;
  name: string;
  /** Map of router hostname -> interface name. Preferred over `interface`. */
  interfaces?: Record<string, string>;
  /** Deprecated legacy single-interface field, still accepted for backward compatibility. */
  interface?: string;
  table_id: number;
  gateway: string;
  description?: string;
  generation?: number;
  writer_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface RoutingPolicy {
  id: string;
  name: string;
  provider_id: string;
  description?: string;
  tags?: string[];
  enabled: boolean;
  favorite?: boolean;
  generation?: number;
  writer_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface HealthResponse {
  status: string;
  timestamp: string;
  service: string;
}

export interface RouterInfo {
  hostname: string;
  agent_version: string;
  log_level: string;
  last_seen: string;
  age_seconds: number;
}

export interface StatsResponse {
  sync: {
    providers_count: number;
    policies_count: number;
    sync_interval?: string;
    policies_per_provider?: Record<string, number>;
  };
  routers?: RouterInfo[];
  log_level?: string;
  timestamp: string;
  version?: string;
  build_time?: string;
  git_commit?: string;
}

export interface LogLevelResponse {
  service_id?: string;
  level: string;
  levels: string[];
}

export interface ServiceLevel {
  level: string;
  source?: string;
  online?: boolean;
}

export interface LogLevelsResponse {
  services: Record<string, ServiceLevel>;
  levels: string[];
}

export interface SetLogLevelRequest {
  level: string;
}

export interface CreateProviderRequest {
  name: string;
  /** Preferred: map of hostname -> interface. */
  interfaces?: Record<string, string>;
  /** Legacy: shared interface name; only used when `interfaces` is empty. */
  interface?: string;
  table_id: number;
  gateway: string;
  description?: string;
}

export interface CreatePolicyRequest {
  name: string;
  source_ip: string;
  provider_id: string;
  description?: string;
  tags?: string[];
  enabled: boolean;
  favorite?: boolean;
}

export interface NetworkInterface {
  name: string;
  mac?: string;
  mtu: number;
  up: boolean;
  addresses: string[];
}

export interface RouteEntry {
  dst: string;
  gateway?: string;
  interface?: string;
  protocol?: string;
  scope?: string;
  metric?: number;
}

export interface RoutingTable {
  id: number;
  name?: string;
  routes: RouteEntry[];
}

export interface IPRule {
  priority: number;
  from: string;
  table: number;
  table_name?: string;
}

export interface RouterState {
  hostname: string;
  agent_version: string;
  log_level: string;
  last_seen: string;
  age_seconds: number;
  online: boolean;
  interfaces: NetworkInterface[];
  tables: RoutingTable[];
  rules: IPRule[];
}
