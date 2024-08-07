package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	storage       *ABIStorage
	etherscanAPIs map[int]ChainAPI
)

var ErrABINotFound = errors.New("ABI not found")

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
	if err != nil {
		if err == ErrABINotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "ABI not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ABI"})
		}
		return
	}

	storage.Set(chainIdStr+"-"+address, StorageItem{
		ABI:            abi,
		Implementation: implementation,
		IsProxy:        proxyInfo != nil,
	})

	c.JSON(http.StatusOK, gin.H{
		"abi":            abi,
		"implementation": implementation,
		"isProxy":        proxyInfo != nil,
	})
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
