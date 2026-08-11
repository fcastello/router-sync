package mcpserver_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"router-sync/internal/mcpserver"
	"router-sync/internal/policies"
	"router-sync/internal/models"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/mock"
)

type mockPolicyNATS struct {
	mock.Mock
}

func (m *mockPolicyNATS) ListPolicies() ([]*models.RoutingPolicy, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RoutingPolicy), args.Error(1)
}

func (m *mockPolicyNATS) GetPolicy(id string) (*models.RoutingPolicy, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RoutingPolicy), args.Error(1)
}

func (m *mockPolicyNATS) StorePolicy(policy *models.RoutingPolicy) error {
	return m.Called(policy).Error(0)
}

func (m *mockPolicyNATS) DeletePolicy(id string) error {
	return m.Called(id).Error(0)
}

func (m *mockPolicyNATS) GetProvider(id string) (*models.InternetProvider, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InternetProvider), args.Error(1)
}

func TestMCPListPolicies(t *testing.T) {
	n := &mockPolicyNATS{}
	now := time.Now()
	n.On("ListPolicies").Return([]*models.RoutingPolicy{
		{ID: "192.168.1.10", Name: "Phone", ProviderID: "fiber", Enabled: true, CreatedAt: now, UpdatedAt: now},
	}, nil)

	handler := mcpserver.NewHTTPHandler(policies.NewService(n), mcpserver.Options{Version: "test"})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) < 5 {
		t.Fatalf("expected policy tools, got %d", len(tools.Tools))
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_policies",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool error: %v", result.Content)
	}
	n.AssertExpectations(t)
}

func TestMCPBearerAuth(t *testing.T) {
	n := &mockPolicyNATS{}
	handler := mcpserver.BearerAuthMiddleware("secret")(
		mcpserver.NewHTTPHandler(policies.NewService(n), mcpserver.Options{Version: "test"}),
	)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	_, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL,
	}, nil)
	if err == nil {
		t.Fatal("expected unauthorized without bearer token")
	}
}
