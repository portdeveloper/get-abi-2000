package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	storage   *ABIStorage
	chainAPIs map[int]ChainAPI
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	storage = NewABIStorage()

	// Initialize chain APIs
	chainAPIs = make(map[int]ChainAPI)
	chainAPIs[1] = &GenericEtherscanAPI{BaseURL: "https://api.etherscan.io/api", EnvKey: "ETHEREUM_API_KEY"}
	chainAPIs[11155111] = &GenericEtherscanAPI{BaseURL: "https://api-sepolia.etherscan.io/api", EnvKey: "SEPOLIA_API_KEY"}
	chainAPIs[10] = &GenericEtherscanAPI{BaseURL: "https://api-optimistic.etherscan.io/api", EnvKey: "OPTIMISM_API_KEY"}
	chainAPIs[56] = &GenericEtherscanAPI{BaseURL: "https://api.bscscan.com/api", EnvKey: "BSC_API_KEY"}
	// Add more chains here as needed
}

func main() {
	router := gin.Default()

	router.GET("/abi/:chainId/:address", getABI)

	log.Fatal(router.Run(":8080"))
}

func getABI(c *gin.Context) {
	chainIdStr := c.Param("chainId")
	address := c.Param("address")

	chainId, err := strconv.Atoi(chainIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	// Check if ABI is in storage
	if abi, ok := storage.Get(chainIdStr + "-" + address); ok {
		c.JSON(http.StatusOK, gin.H{"chainId": chainId, "address": address, "abi": abi})
		return
	}

	// Get the appropriate chain API
	api, ok := chainAPIs[chainId]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported chain ID"})
		return
	}

	// Fetch ABI from the chain-specific API
	abi, err := api.GetABI(address)
	if err != nil {
		// Check if the error is due to missing API key
		if err.Error() == "API key not set for chain" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "API key not configured for this chain"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Store the fetched ABI
	storage.Set(chainIdStr+"-"+address, abi)

	c.JSON(http.StatusOK, gin.H{"chainId": chainId, "address": address, "abi": abi})
}
