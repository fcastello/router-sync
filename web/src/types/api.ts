export interface InternetProvider {
  id: string;
  name: string;
  interface: string;
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
  enabled: boolean;
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

export interface StatsResponse {
  sync: {
    providers_count: number;
    policies_count: number;
    sync_interval: string;
    policies_per_provider?: Record<string, number>;
  };
  router: Record<string, unknown>;
  log_level?: string;
  timestamp: string;
  version?: string;
  build_time?: string;
  git_commit?: string;
}

export interface LogLevelResponse {
  level: string;
  levels: string[];
}

export interface SetLogLevelRequest {
  level: string;
}

export interface CreateProviderRequest {
  name: string;
  interface: string;
  table_id: number;
  gateway: string;
  description?: string;
}

export interface CreatePolicyRequest {
  name: string;
  source_ip: string;
  provider_id: string;
  description?: string;
  enabled: boolean;
}

export interface DeviceMeta {
  friendlyName?: string;
  tags: string[];
  mac?: string;
}

export type DeviceMetaMap = Record<string, DeviceMeta>;
