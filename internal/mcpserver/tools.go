package mcpserver

import (
	"context"
	"strings"

	"router-sync/internal/models"
	"router-sync/internal/policies"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPolicyTools(server *mcp.Server, svc *policies.Service) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_policies",
		Description: "List all routing policies (source IP/CIDR → uplink provider). " +
			"Optional filters: tag, enabled, provider_id. " +
			"Policy id is the source IP or CIDR; in URLs use underscore for slash (192.168.1.0_24).",
	}, listPoliciesTool(svc))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_policy",
		Description: "Get one routing policy by policy_id (source IP/CIDR; use underscore for / in CIDR).",
	}, getPolicyTool(svc))

	mcp.AddTool(server, &mcp.Tool{
		Name: "create_policy",
		Description: "Create a routing policy: route traffic from source_ip (stored as policy id) through provider_id. " +
			"Set enabled=true to apply ip rules on agents. provider_id must match an existing uplink (e.g. fiber).",
	}, createPolicyTool(svc))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_policy",
		Description: "Replace an existing policy. policy_id is the current key; source_ip may change the key (CIDR).",
	}, updatePolicyTool(svc))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_policy",
		Description: "Delete a routing policy by policy_id.",
	}, deletePolicyTool(svc))

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_policy_routing",
		Description: "Change which uplink a policy uses and whether it is active. " +
			"Use this to move a device/subnet to another ISP or enable/disable routing without editing other fields.",
	}, setPolicyRoutingTool(svc))
}

type listPoliciesParams struct {
	Tag        string `json:"tag,omitempty" jsonschema:"Filter policies that include this tag"`
	Enabled    *bool  `json:"enabled,omitempty" jsonschema:"Filter by enabled (active routing override)"`
	ProviderID string `json:"provider_id,omitempty" jsonschema:"Filter by uplink provider id"`
}

func listPoliciesTool(svc *policies.Service) func(context.Context, *mcp.CallToolRequest, listPoliciesParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, params listPoliciesParams) (*mcp.CallToolResult, any, error) {
		all, err := svc.List()
		if err != nil {
			return toolError(err)
		}
		tag := strings.TrimSpace(params.Tag)
		providerID := strings.TrimSpace(params.ProviderID)
		out := make([]*models.RoutingPolicy, 0, len(all))
		for _, p := range all {
			if tag != "" && !containsTag(p.Tags, tag) {
				continue
			}
			if params.Enabled != nil && p.Enabled != *params.Enabled {
				continue
			}
			if providerID != "" && p.ProviderID != providerID {
				continue
			}
			out = append(out, p)
		}
		return toolJSONResult(map[string]any{
			"count":    len(out),
			"policies": out,
		})
	}
}

type policyIDParams struct {
	PolicyID string `json:"policy_id" jsonschema:"Policy id (source IP or CIDR; use 192.168.1.0_24 for /24)"`
}

func getPolicyTool(svc *policies.Service) func(context.Context, *mcp.CallToolRequest, policyIDParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, params policyIDParams) (*mcp.CallToolResult, any, error) {
		p, err := svc.Get(params.PolicyID)
		if err != nil {
			return toolError(err)
		}
		return toolJSONResult(p)
	}
}

type createPolicyParams struct {
	Name        string   `json:"name" jsonschema:"Display name"`
	SourceIP    string   `json:"source_ip" jsonschema:"Source IP or CIDR (becomes policy id)"`
	ProviderID  string   `json:"provider_id" jsonschema:"Uplink provider id"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Enabled     bool     `json:"enabled" jsonschema:"When true, agents install ip rules for this source"`
	Favorite    bool     `json:"favorite,omitempty"`
}

func createPolicyTool(svc *policies.Service) func(context.Context, *mcp.CallToolRequest, createPolicyParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, params createPolicyParams) (*mcp.CallToolResult, any, error) {
		p, err := svc.Create(policies.CreateRequest{
			Name:        params.Name,
			SourceIP:    params.SourceIP,
			ProviderID:  params.ProviderID,
			Description: params.Description,
			Tags:        params.Tags,
			Enabled:     params.Enabled,
			Favorite:    params.Favorite,
		})
		if err != nil {
			return toolError(err)
		}
		return toolJSONResult(p)
	}
}

type updatePolicyParams struct {
	PolicyID    string   `json:"policy_id" jsonschema:"Current policy id in NATS"`
	Name        string   `json:"name"`
	SourceIP    string   `json:"source_ip"`
	ProviderID  string   `json:"provider_id"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Enabled     bool     `json:"enabled"`
	Favorite    bool     `json:"favorite,omitempty"`
}

func updatePolicyTool(svc *policies.Service) func(context.Context, *mcp.CallToolRequest, updatePolicyParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, params updatePolicyParams) (*mcp.CallToolResult, any, error) {
		p, err := svc.Update(params.PolicyID, policies.UpdateRequest{
			Name:        params.Name,
			SourceIP:    params.SourceIP,
			ProviderID:  params.ProviderID,
			Description: params.Description,
			Tags:        params.Tags,
			Enabled:     params.Enabled,
			Favorite:    params.Favorite,
		})
		if err != nil {
			return toolError(err)
		}
		return toolJSONResult(p)
	}
}

func deletePolicyTool(svc *policies.Service) func(context.Context, *mcp.CallToolRequest, policyIDParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, params policyIDParams) (*mcp.CallToolResult, any, error) {
		if err := svc.Delete(params.PolicyID); err != nil {
			return toolError(err)
		}
		return toolJSONResult(map[string]string{
			"status":    "deleted",
			"policy_id": params.PolicyID,
		})
	}
}

type setPolicyRoutingParams struct {
	PolicyID   string `json:"policy_id"`
	ProviderID string `json:"provider_id" jsonschema:"Uplink provider id to route through"`
	Enabled    bool   `json:"enabled" jsonschema:"Enable or disable routing for this source"`
}

func setPolicyRoutingTool(svc *policies.Service) func(context.Context, *mcp.CallToolRequest, setPolicyRoutingParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, params setPolicyRoutingParams) (*mcp.CallToolResult, any, error) {
		existing, err := svc.Get(params.PolicyID)
		if err != nil {
			return toolError(err)
		}
		p, err := svc.Update(params.PolicyID, policies.UpdateRequest{
			Name:        existing.Name,
			SourceIP:    existing.ID,
			ProviderID:  params.ProviderID,
			Description: existing.Description,
			Tags:        existing.Tags,
			Enabled:     params.Enabled,
			Favorite:    existing.Favorite,
		})
		if err != nil {
			return toolError(err)
		}
		return toolJSONResult(p)
	}
}

func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
