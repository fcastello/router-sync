import type { CreatePolicyRequest, RoutingPolicy } from "@/types/api";

export function policyBody(
  policy: RoutingPolicy,
  patch: Partial<CreatePolicyRequest> = {},
): CreatePolicyRequest {
  return {
    name: policy.name,
    source_ip: policy.id,
    provider_id: policy.provider_id,
    description: policy.description,
    enabled: policy.enabled,
    favorite: policy.favorite ?? false,
    ...patch,
  };
}
