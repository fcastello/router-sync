import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { loadDeviceMeta } from "@/lib/device-meta";
import type { CreatePolicyRequest, CreateProviderRequest } from "@/types/api";

export const queryKeys = {
  health: ["health"] as const,
  stats: ["stats"] as const,
  providers: ["providers"] as const,
  policies: ["policies"] as const,
  routers: ["routers"] as const,
  router: (hostname: string) => ["router", hostname] as const,
  deviceMeta: ["deviceMeta"] as const,
  logLevel: ["logLevel"] as const,
  logLevels: ["logLevels"] as const,
};

export function useHealth(pollMs = 5000) {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: () => api.health(),
    refetchInterval: pollMs,
    retry: 1,
  });
}

export function useStats(pollMs = 10000) {
  return useQuery({
    queryKey: queryKeys.stats,
    queryFn: () => api.stats(),
    refetchInterval: pollMs,
  });
}

export function useProviders() {
  return useQuery({
    queryKey: queryKeys.providers,
    queryFn: () => api.listProviders(),
  });
}

export function usePolicies() {
  return useQuery({
    queryKey: queryKeys.policies,
    queryFn: () => api.listPolicies(),
  });
}

export function useRouters(pollMs = 10000) {
  return useQuery({
    queryKey: queryKeys.routers,
    queryFn: () => api.listRouters(),
    refetchInterval: pollMs,
  });
}

export function useRouter(hostname: string, pollMs = 10000) {
  return useQuery({
    queryKey: queryKeys.router(hostname),
    queryFn: () => api.getRouter(hostname),
    enabled: Boolean(hostname),
    refetchInterval: pollMs,
  });
}

export function useDeviceMeta() {
  return useQuery({
    queryKey: queryKeys.deviceMeta,
    queryFn: async () => loadDeviceMeta(),
    initialData: loadDeviceMeta(),
  });
}

export function useProviderMutations() {
  const qc = useQueryClient();
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: queryKeys.providers });
    qc.invalidateQueries({ queryKey: queryKeys.stats });
  };

  return {
    create: useMutation({
      mutationFn: (body: CreateProviderRequest) => api.createProvider(body),
      onSuccess: invalidate,
    }),
    update: useMutation({
      mutationFn: ({ id, body }: { id: string; body: CreateProviderRequest }) =>
        api.updateProvider(id, body),
      onSuccess: invalidate,
    }),
    remove: useMutation({
      mutationFn: (id: string) => api.deleteProvider(id),
      onSuccess: invalidate,
    }),
  };
}

export function usePolicyMutations() {
  const qc = useQueryClient();
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: queryKeys.policies });
    qc.invalidateQueries({ queryKey: queryKeys.stats });
  };

  const update = useMutation({
    mutationFn: ({ id, body }: { id: string; body: CreatePolicyRequest }) =>
      api.updatePolicy(id, body),
    onMutate: async ({ id, body }) => {
      await qc.cancelQueries({ queryKey: queryKeys.policies });
      const prev = qc.getQueryData(queryKeys.policies);
      qc.setQueryData(
        queryKeys.policies,
        (old: Awaited<ReturnType<typeof api.listPolicies>> | undefined) =>
          old?.map((p) =>
            p.id === id
              ? {
                  ...p,
                  enabled: body.enabled,
                  provider_id: body.provider_id,
                  name: body.name,
                  favorite: body.favorite,
                }
              : p,
          ),
      );
      return { prev };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.prev) qc.setQueryData(queryKeys.policies, ctx.prev);
    },
    onSettled: invalidate,
  });

  return {
    create: useMutation({
      mutationFn: (body: CreatePolicyRequest) => api.createPolicy(body),
      onSuccess: invalidate,
    }),
    update,
    remove: useMutation({
      mutationFn: (id: string) => api.deletePolicy(id),
      onSuccess: invalidate,
    }),
  };
}

export function useLogLevel() {
  return useQuery({
    queryKey: queryKeys.logLevel,
    queryFn: () => api.getOwnLogLevel(),
  });
}

export function useLogLevels(pollMs = 10000) {
  return useQuery({
    queryKey: queryKeys.logLevels,
    queryFn: () => api.listLogLevels(),
    refetchInterval: pollMs,
  });
}

export function useLogLevelMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (level: string) => api.setOwnLogLevel({ level }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.logLevel });
      qc.invalidateQueries({ queryKey: queryKeys.logLevels });
      qc.invalidateQueries({ queryKey: queryKeys.stats });
    },
  });
}

export function useServiceLogLevelMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ serviceId, level }: { serviceId: string; level: string }) =>
      api.setServiceLogLevel(serviceId, { level }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.logLevels });
      qc.invalidateQueries({ queryKey: queryKeys.logLevel });
      qc.invalidateQueries({ queryKey: queryKeys.routers });
    },
  });
}

export function useTriggerSync() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.triggerSync(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.stats });
      qc.invalidateQueries({ queryKey: queryKeys.policies });
      qc.invalidateQueries({ queryKey: queryKeys.providers });
    },
  });
}
