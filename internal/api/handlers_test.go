package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"router-sync/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNATSClient is a mock implementation of the NATS client
type MockNATSClient struct {
	mock.Mock
}

func (m *MockNATSClient) StoreProvider(provider *models.InternetProvider) error {
	args := m.Called(provider)
	return args.Error(0)
}

func (m *MockNATSClient) GetProvider(id string) (*models.InternetProvider, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InternetProvider), args.Error(1)
}

func (m *MockNATSClient) ListProviders() ([]*models.InternetProvider, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.InternetProvider), args.Error(1)
}

func (m *MockNATSClient) DeleteProvider(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockNATSClient) StorePolicy(policy *models.RoutingPolicy) error {
	args := m.Called(policy)
	return args.Error(0)
}

func (m *MockNATSClient) GetPolicy(id string) (*models.RoutingPolicy, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RoutingPolicy), args.Error(1)
}

func (m *MockNATSClient) ListPolicies() ([]*models.RoutingPolicy, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RoutingPolicy), args.Error(1)
}

func (m *MockNATSClient) DeletePolicy(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockNATSClient) Close() {
	m.Called()
}

func TestCreateProvider_WithNameAsID(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a mock NATS client
	mockNATS := &MockNATSClient{}

	// Create a test server
	server := &Server{
		natsClient: mockNATS,
	}

	// Test data
	providerName := "TestProvider"
	createRequest := CreateProviderRequest{
		Name:        providerName,
		Interface:   "eth0",
		TableID:     100,
		Gateway:     "192.168.1.1",
		Description: "Test provider",
	}

	// Set up mock expectations
	mockNATS.On("GetProvider", providerName).Return(nil, assert.AnError) // Provider doesn't exist
	mockNATS.On("StoreProvider", mock.AnythingOfType("*models.InternetProvider")).Return(nil)

	// Create request body
	requestBody, _ := json.Marshal(createRequest)

	// Create HTTP request
	req, _ := http.NewRequest("POST", "/api/v1/providers", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the function
	server.createProvider(c)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse response
	var response models.InternetProvider
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify that the ID is set to the name
	assert.Equal(t, providerName, response.ID)
	assert.Equal(t, providerName, response.Name)

	// Verify that the mock was called correctly
	mockNATS.AssertExpectations(t)
}

func TestCreateProvider_DuplicateName(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a mock NATS client
	mockNATS := &MockNATSClient{}

	// Create a test server
	server := &Server{
		natsClient: mockNATS,
	}

	// Test data
	providerName := "ExistingProvider"
	createRequest := CreateProviderRequest{
		Name:        providerName,
		Interface:   "eth0",
		TableID:     100,
		Gateway:     "192.168.1.1",
		Description: "Test provider",
	}

	// Create an existing provider
	existingProvider := &models.InternetProvider{
		ID:          providerName,
		Name:        providerName,
		Interface:   "eth1",
		TableID:     200,
		Gateway:     "192.168.2.1",
		Description: "Existing provider",
	}

	// Set up mock expectations
	mockNATS.On("GetProvider", providerName).Return(existingProvider, nil) // Provider exists

	// Create request body
	requestBody, _ := json.Marshal(createRequest)

	// Create HTTP request
	req, _ := http.NewRequest("POST", "/api/v1/providers", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the function
	server.createProvider(c)

	// Assertions
	assert.Equal(t, http.StatusConflict, w.Code)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify error message
	assert.Equal(t, "Provider already exists", response["error"])
	assert.Contains(t, response["details"], providerName)

	// Verify that the mock was called correctly
	mockNATS.AssertExpectations(t)
}
