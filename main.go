package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	storage *ABIStorage
	apiKey  string
)

func init() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	storage = NewABIStorage()
	apiKey = os.Getenv("ETHERSCAN_API_KEY")
	if apiKey == "" {
		log.Fatal("ETHERSCAN_API_KEY environment variable is not set in .env file")
	}
}

func main() {
	router := gin.Default()

	router.GET("/abi/:address", getABI)

	log.Fatal(router.Run(":8080"))
}

func getABI(c *gin.Context) {
	address := c.Param("address")

	// Check if ABI is in storage
	if abi, ok := storage.Get(address); ok {
		c.JSON(http.StatusOK, gin.H{"address": address, "abi": abi})
		return
	}

	// If not in storage, fetch from Etherscan
	abi, err := getABIFromEtherscan(address, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Store the fetched ABI
	storage.Set(address, abi)

	c.JSON(http.StatusOK, gin.H{"address": address, "abi": abi})
}
