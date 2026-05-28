import { getApiBaseUrl } from "@/lib/config";
import { policyIdForPath } from "@/lib/policy-id";
import type {
  CreatePolicyRequest,
  CreateProviderRequest,
  HealthResponse,
  InternetProvider,
  LogLevelResponse,
  LogLevelsResponse,
  RouterState,
  RoutingPolicy,
  SetLogLevelRequest,
  StatsResponse,
} from "@/types/api";

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public details?: string,
  ) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const base = getApiBaseUrl();
  const url = `${base}${path}`;
  const res = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
  });

  if (!res.ok) {
    let details = "";
    try {
      const body = await res.json();
      details = body.details || body.error || JSON.stringify(body);
    } catch {
      details = await res.text();
    }
    throw new ApiError(`Request failed: ${path}`, res.status, details);
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const api = {
  health: () => request<HealthResponse>("/health"),
  stats: () => request<StatsResponse>("/api/v1/stats"),
  triggerSync: () =>
    request<{ message: string }>("/api/v1/sync", { method: "POST" }),

  listProviders: () => request<InternetProvider[]>("/api/v1/providers"),
  getProvider: (id: string) =>
    request<InternetProvider>(`/api/v1/providers/${encodeURIComponent(id)}`),
  createProvider: (body: CreateProviderRequest) =>
    request<InternetProvider>("/api/v1/providers", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateProvider: (id: string, body: CreateProviderRequest) =>
    request<InternetProvider>(`/api/v1/providers/${encodeURIComponent(id)}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deleteProvider: (id: string) =>
    request<void>(`/api/v1/providers/${encodeURIComponent(id)}`, {
      method: "DELETE",
    }),

  listPolicies: () => request<RoutingPolicy[]>("/api/v1/policies"),
  getPolicy: (id: string) =>
    request<RoutingPolicy>(`/api/v1/policies/${policyIdForPath(id)}`),
  createPolicy: (body: CreatePolicyRequest) =>
    request<RoutingPolicy>("/api/v1/policies", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updatePolicy: (id: string, body: CreatePolicyRequest) =>
    request<RoutingPolicy>(`/api/v1/policies/${policyIdForPath(id)}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deletePolicy: (id: string) =>
    request<void>(`/api/v1/policies/${policyIdForPath(id)}`, {
      method: "DELETE",
    }),

  listRouters: () => request<RouterState[]>("/api/v1/routers"),
  getRouter: (hostname: string) =>
    request<RouterState>(`/api/v1/routers/${encodeURIComponent(hostname)}`),

  getOwnLogLevel: () => request<LogLevelResponse>("/api/v1/logging/level"),
  setOwnLogLevel: (body: SetLogLevelRequest) =>
    request<LogLevelResponse>("/api/v1/logging/level", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  listLogLevels: () => request<LogLevelsResponse>("/api/v1/logging/levels"),
  setServiceLogLevel: (serviceId: string, body: SetLogLevelRequest) =>
    request<LogLevelResponse>(
      `/api/v1/logging/level/${encodeURIComponent(serviceId)}`,
      { method: "PUT", body: JSON.stringify(body) },
    ),

  // Legacy aliases (kept so older Settings code keeps working).
  getLogLevel: () => request<LogLevelResponse>("/api/v1/logging/level"),
  setLogLevel: (body: SetLogLevelRequest) =>
    request<LogLevelResponse>("/api/v1/logging/level", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
};
