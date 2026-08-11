package mcpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"router-sync/internal/policies"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options configures the policy MCP HTTP handler.
type Options struct {
	Version   string
	BuildTime string
	GitCommit string
}

// NewHTTPHandler returns a streamable HTTP MCP handler for routing policy management.
func NewHTTPHandler(svc *policies.Service, opts Options) http.Handler {
	server := newPolicyMCPServer(svc, opts)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
}

func newPolicyMCPServer(svc *policies.Service, opts Options) *mcp.Server {
	version := opts.Version
	if version == "" {
		version = "dev"
	}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "router-sync-policies",
		Version: version,
	}, nil)

	registerPolicyTools(server, svc)
	return server
}

func toolJSONResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

func toolError(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("error: %v", err)},
		},
		IsError: true,
	}, nil, nil
}
