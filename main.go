package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	storage       *ABIStorage
	etherscanAPIs map[int]ChainAPI
)

var ErrABINotFound = errors.New("ABI not found")

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

	if item, ok := storage.Get(chainIdStr + "-" + address); ok {
		c.JSON(http.StatusOK, gin.H{
			"abi":            item.ABI,
			"implementation": item.Implementation,
			"isProxy":        item.IsProxy,
			"isDecompiled":   item.IsDecompiled,
		})
		return
	}

	api, ok := etherscanAPIs[chainId]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported chain ID"})
		return
	}

	nodeURL, err := getRPCURL(chainId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get RPC URL: %v", err)})
		return
	}

	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to Ethereum node"})
		return
	}

	// Check if the address is a contract
	code, err := client.CodeAt(c.Request.Context(), common.HexToAddress(address), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to check contract code: %v", err)})
		return
	}

	if len(code) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The provided address is not a contract"})
		return
	}

	proxyInfo, err := DetectProxyTarget(c.Request.Context(), client, common.HexToAddress(address))
	if err != nil {
		proxyInfo = nil
	}

	targetAddress := address
	var implementation interface{} = nil
	if proxyInfo != nil && proxyInfo.Target != (common.Address{}) {
		targetAddress = proxyInfo.Target.Hex()
		implementation = targetAddress
	}

	abi, err := api.GetABI(targetAddress)
	isDecompiled := false
	if err != nil {
		// Fallback to Heimdall API
		rpcURL := strings.TrimPrefix(nodeURL, "https://")
		abi, err = getABIFromHeimdall(targetAddress, rpcURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ABI from both Etherscan and Heimdall"})
			return
		}
		isDecompiled = true
	}

	storage.Set(chainIdStr+"-"+address, StorageItem{
		ABI:            abi,
		Implementation: implementation,
		IsProxy:        proxyInfo != nil,
		IsDecompiled:   isDecompiled,
	})

	c.JSON(http.StatusOK, gin.H{
		"abi":            abi,
		"implementation": implementation,
		"isProxy":        proxyInfo != nil,
		"isDecompiled":   isDecompiled,
	})
}

func getABIFromHeimdall(address string, rpcURL string) (string, error) {
	url := fmt.Sprintf("https://heimdall-api.fly.dev/%s?rpc_url=%s", address, rpcURL)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("heimdall API error: %s", string(body))
	}

	return string(body), nil
}

func getRPCURL(chainId int) (string, error) {
	switch chainId {
	case 1:
		return "https://rpc.ankr.com/eth", nil
	case 11155111:
		return "https://rpc.ankr.com/eth_sepolia", nil
	case 10:
		return "https://rpc.ankr.com/optimism", nil
	case 56:
		return "https://rpc.ankr.com/bsc", nil
	default:
		return "", fmt.Errorf("unsupported chain ID: %d", chainId)
	}
}
