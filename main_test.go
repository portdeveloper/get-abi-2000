package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

type MockChainAPI struct {
	ABIs map[string]string
}

func (m *MockChainAPI) GetABI(address string) (string, error) {
	abi, ok := m.ABIs[address]
	if !ok {
		return "", ErrABINotFound
	}
	return abi, nil
}

func TestGetABI(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/abi/:chainId/:address", getABI)

	// Mock ChainAPI
	mockAPI := &MockChainAPI{
		ABIs: map[string]string{
			"0x123": "mock ABI",
		},
	}
	etherscanAPIs[1] = mockAPI

	// Test cases
	testCases := []struct {
		name           string
		chainID        string
		address        string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "Valid request",
			chainID:        "1",
			address:        "0x123",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"chainId": float64(1),
				"address": "0x123",
				"abi":     "mock ABI",
			},
		},
		{
			name:           "Invalid chain ID",
			chainID:        "999",
			address:        "0x123",
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Unsupported chain ID",
			},
		},
		{
			name:           "Invalid address",
			chainID:        "1",
			address:        "0x456",
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "ABI not found",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/abi/"+tc.chainID+"/"+tc.address, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, response)
		})
	}
}

func TestABIStorage(t *testing.T) {
	storage := NewABIStorage()

	// Test Set and Get
	storage.Set("test-key", "test-abi")
	abi, ok := storage.Get("test-key")
	assert.True(t, ok)
	assert.Equal(t, "test-abi", abi)

	// Test Get with non-existent key
	abi, ok = storage.Get("non-existent")
	assert.False(t, ok)
	assert.Empty(t, abi)
}

func TestRealEtherscanCall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}

	// Ensure the API key is set
	apiKey := os.Getenv("OPTIMISM_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: OPTIMISM_API_KEY not set")
	}

	// Create a GenericEtherscanAPI instance for Optimism
	optimismAPI := &GenericEtherscanAPI{
		BaseURL: "https://api-optimistic.etherscan.io/api",
		EnvKey:  "OPTIMISM_API_KEY",
	}

	// Test address
	address := "0xE575E956757c20b22C5a11eB542F719564c32Fe8"

	// Call GetABI
	abi, err := optimismAPI.GetABI(address)
	if err != nil {
		t.Fatalf("Error getting ABI: %v", err)
	}

	// Basic check to ensure ABI is not empty
	assert.NotEmpty(t, abi, "ABI should not be empty")

	// You could add more specific checks here if you know what the ABI should contain
	t.Logf("Received ABI: %s", abi)
}
