package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	storage       *ABIStorage
	etherscanAPIs map[int]ChainAPI
	abiFetcher    *ABIFetcher
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
	etherscanAPIs[8453] = &GenericEtherscanAPI{BaseURL: "https://api.basescan.org/api", EnvKey: "BASE_API_KEY"}
	etherscanAPIs[56] = &GenericEtherscanAPI{BaseURL: "https://api.bscscan.com/api", EnvKey: "BSC_API_KEY"}
	etherscanAPIs[137] = &GenericEtherscanAPI{BaseURL: "https://api.polygonscan.com/api", EnvKey: "POLYGON_API_KEY"}

	abiFetcher = NewABIFetcher(storage, etherscanAPIs)
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
	rpcURL := c.Param("rpcUrl")[1:]

	response, err := abiFetcher.FetchABI(c, chainId, address, rpcURL)
	if err != nil {
		switch e := err.(type) {
		case *InvalidInputError:
			c.JSON(http.StatusBadRequest, gin.H{"error": e.Error()})
		case *ContractNotFoundError:
			c.JSON(http.StatusNotFound, gin.H{"error": e.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}
