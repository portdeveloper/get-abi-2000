package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

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
	abi, err := optimismAPI.GetABIFromEtherscan(address)
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
	router.GET("/abi/:chainId/:address/*rpcUrl", getABI)

	// Create a real EtherscanAPI instance for Ethereum
	etherscanAPI := &GenericEtherscanAPI{
		BaseURL: "https://api.etherscan.io/api",
		EnvKey:  "ETHEREUM_API_KEY",
	}
	etherscanAPIs[1] = etherscanAPI

	// USDC contract address
	address := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	rpcURL := "rpc.ankr.com/eth" // Replace with a valid Ethereum RPC URL

	// Make the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/abi/1/"+address+"/"+rpcURL, nil)
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

func TestHeimdallAPIResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/abi/:chainId/:address/*rpcUrl", getABI)

	// Test contract address
	address := "0x759c0e9d7858566df8ab751026bedce462ff42df"
	chainID := "11155111" // Sepolia chain ID
	rpcURL := "rpc.ankr.com/eth_sepolia"

	// Make the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/abi/"+chainID+"/"+address+"/"+rpcURL, nil)
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check the specific fields
	assert.Nil(t, response["implementation"])
	assert.Equal(t, true, response["isDecompiled"])
	assert.Equal(t, false, response["isProxy"])

	// Check if the ABI is correct
	expectedABI := `[
  {
    "type": "function",
    "name": "changeOwner",
    "inputs": [
      {
        "name": "arg0",
        "type": "address"
      }
    ],
    "outputs": [],
    "stateMutability": "payable"
  },
  {
    "type": "function",
    "name": "getOwner",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "payable"
  },
  {
    "type": "event",
    "name": "OwnerSet",
    "inputs": [
      {
        "name": "arg0",
        "type": "address",
        "indexed": false
      },
      {
        "name": "arg1",
        "type": "address",
        "indexed": false
      }
    ],
    "anonymous": false
  }
]`

	actualABI, ok := response["abi"].(string)
	assert.True(t, ok)

	// Normalize the JSON strings for comparison
	var expectedJSON, actualJSON interface{}
	json.Unmarshal([]byte(expectedABI), &expectedJSON)
	json.Unmarshal([]byte(actualABI), &actualJSON)

	assert.Equal(t, expectedJSON, actualJSON)

	t.Logf("Received ABI: %s", actualABI)
}

func TestParexContractABI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/abi/:chainId/:address/*rpcUrl", getABI)

	// Test contract address
	address := "0x6058518142C6AD506530F5A62dCc58050bf6fC28"
	chainID := "322202" // Parex chain ID
	rpcURL := "mainnet-rpc.parex.network"

	// Make the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/abi/"+chainID+"/"+address+"/"+rpcURL, nil)
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check the specific fields
	assert.Equal(t, false, response["isProxy"])
	assert.Equal(t, true, response["isDecompiled"])

	// Check if the ABI contains "sendValidatorReward"
	abi, ok := response["abi"].(string)
	assert.True(t, ok)
	assert.Contains(t, abi, "sendValidatorReward")

	t.Logf("Received ABI: %s", abi)
}

func TestEtherscanFailureHeimdallFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/abi/:chainId/:address/*rpcUrl", getABI)

	// Create a mock EtherscanAPI that always fails
	mockEtherscanAPI := &MockEtherscanAPI{
		ShouldFail: true,
	}
	etherscanAPIs[1] = mockEtherscanAPI

	// Test contract address (use a real contract address that Heimdall can decompile)
	address := "0x6B175474E89094C44Da98b954EedeAC495271d0F" // DAI token
	chainID := "1"                                          // Ethereum mainnet
	rpcURL := "rpc.ankr.com/eth"

	// Make the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/abi/"+chainID+"/"+address+"/"+rpcURL, nil)
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check the specific fields
	assert.Nil(t, response["implementation"])
	assert.Equal(t, true, response["isDecompiled"])
	assert.Equal(t, false, response["isProxy"])

	// Check if the ABI contains some expected function
	abi, ok := response["abi"].(string)
	assert.True(t, ok)
	assert.Contains(t, abi, "transfer")

	t.Logf("Received ABI: %s", abi)
}

// MockEtherscanAPI is a mock implementation of the ChainAPI interface
type MockEtherscanAPI struct {
	ShouldFail bool
}

func (m *MockEtherscanAPI) GetABIFromEtherscan(address string) (string, error) {
	if m.ShouldFail {
		return "", fmt.Errorf("mock Etherscan API error")
	}
	return "", nil
}
