import type { CreatePolicyRequest, RoutingPolicy } from "@/types/api";
import { normalizeTags } from "@/lib/policy-tags";

export function policyBody(
  policy: RoutingPolicy,
  patch: Partial<CreatePolicyRequest> = {},
): CreatePolicyRequest {
  const body: CreatePolicyRequest = {
    name: policy.name,
    source_ip: policy.id,
    provider_id: policy.provider_id,
    description: policy.description,
    tags: normalizeTags(policy.tags),
    enabled: policy.enabled,
    favorite: policy.favorite ?? false,
    ...patch,
  };
  if (body.tags !== undefined) {
    body.tags = normalizeTags(body.tags);
  }
  return body;
}
