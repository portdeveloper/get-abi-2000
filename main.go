package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	storage       *ABIStorage
	etherscanAPIs map[int]ChainAPI
)

var ErrABINotFound = errors.New("ABI not found")

// TODO: Remove allow all origins

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	storage = NewABIStorage()

	etherscanAPIs = make(map[int]ChainAPI)
	etherscanAPIs[1] = &GenericEtherscanAPI{BaseURL: "https://api.etherscan.io/api", EnvKey: "ETHEREUM_API_KEY"}
	etherscanAPIs[11155111] = &GenericEtherscanAPI{BaseURL: "https://api-sepolia.etherscan.io/api", EnvKey: "SEPOLIA_API_KEY"}
	etherscanAPIs[10] = &GenericEtherscanAPI{BaseURL: "https://api-optimistic.etherscan.io/api", EnvKey: "OPTIMISM_API_KEY"}
	etherscanAPIs[56] = &GenericEtherscanAPI{BaseURL: "https://api.bscscan.com/api", EnvKey: "BSC_API_KEY"}
}

func main() {
	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	router.GET("/", healthCheck)
	router.GET("/abi/:chainId/:address/*rpcUrl", getABI)

	log.Fatal(router.Run(":8080"))
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Get-ABI-2000 is up and running",
	})
}

func getABI(c *gin.Context) {
	chainId := c.Param("chainId")
	address := c.Param("address")
	rpcURL := strings.TrimPrefix(c.Param("rpcUrl"), "/")

	if err := validateInput(chainId, address, rpcURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	abiFetcher := NewABIFetcher(storage, etherscanAPIs)
	response, err := abiFetcher.FetchABI(c, chainId, address, rpcURL)
	if err != nil {
		status, errorMessage := handleError(err)
		c.JSON(status, gin.H{"error": errorMessage})
		return
	}

	c.JSON(http.StatusOK, response)
}

func validateInput(chainId, address, rpcURL string) error {
	if _, err := strconv.Atoi(chainId); err != nil {
		return fmt.Errorf("invalid chainId: must be a number")
	}
	if !common.IsHexAddress(address) {
		return fmt.Errorf("invalid address: must be a valid Ethereum address")
	}
	if rpcURL == "" {
		return fmt.Errorf("invalid rpcURL: cannot be empty")
	}
	return nil
}

func handleError(err error) (int, string) {
	switch err.(type) {
	case *InvalidInputError:
		return http.StatusBadRequest, err.Error()
	case *ContractNotFoundError:
		return http.StatusNotFound, err.Error()
	case *EtherscanAPIError:
		return http.StatusServiceUnavailable, "External API error: " + err.Error()
	default:
		return http.StatusInternalServerError, "Internal server error"
	}
}
