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
				"abi":            "mock ABI",
				"implementation": nil,
				"isProxy":        false,
				"isDecompiled":   false,
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
				"error": "Failed to fetch ABI from both Etherscan and Heimdall",
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
	testItem := StorageItem{
		ABI:            "test-abi",
		Implementation: "0x123",
		IsProxy:        true,
	}
	storage.Set("test-key", testItem)

	item, ok := storage.Get("test-key")
	assert.True(t, ok)
	assert.Equal(t, testItem, item)

	// Test Get with non-existent key
	item, ok = storage.Get("non-existent")
	assert.False(t, ok)
	assert.Equal(t, StorageItem{}, item)
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

func TestRealUSDCProxyDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}

	// Ensure the API key is set
	apiKey := os.Getenv("ETHEREUM_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: ETHEREUM_API_KEY not set")
	}

	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/abi/:chainId/:address", getABI)

	// Create a real EtherscanAPI instance for Ethereum
	etherscanAPI := &GenericEtherscanAPI{
		BaseURL: "https://api.etherscan.io/api",
		EnvKey:  "ETHEREUM_API_KEY",
	}
	etherscanAPIs[1] = etherscanAPI

	// USDC contract address
	address := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"

	// Make the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/abi/1/"+address, nil)
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check the specific fields
	assert.Equal(t, "0x43506849D7C04F9138D1A2050bbF3A0c054402dd", response["implementation"])
	assert.Equal(t, false, response["isDecompiled"])
	assert.Equal(t, true, response["isProxy"])

	// Check if the ABI contains "isBlacklisted"
	abi, ok := response["abi"].(string)
	assert.True(t, ok)
	assert.Contains(t, abi, "isBlacklisted")

	t.Logf("Received ABI: %s", abi)
}
