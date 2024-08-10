package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

type ABIFetcher struct {
	storage       *ABIStorage
	etherscanAPIs map[int]ChainAPI
}

func NewABIFetcher(storage *ABIStorage, etherscanAPIs map[int]ChainAPI) *ABIFetcher {
	return &ABIFetcher{
		storage:       storage,
		etherscanAPIs: etherscanAPIs,
	}
}

func (af *ABIFetcher) FetchABI(c *gin.Context, chainId string, address string, rpcURL string) (gin.H, error) {
	if item, ok := af.storage.Get(chainId + "-" + address); ok {
		return af.createResponse(item), nil
	}

	client, err := ethclient.Dial("https://" + rpcURL)
	if err != nil {
		return nil, &InvalidInputError{message: "Failed to connect to Ethereum node: " + err.Error()}
	}

	if err := af.validateContract(c.Request.Context(), client, address); err != nil {
		return nil, err
	}

	proxyInfo, err := DetectProxyTarget(c.Request.Context(), client, common.HexToAddress(address))
	if err != nil {
		proxyInfo = nil
	}

	targetAddress, implementation := af.getTargetAddress(address, proxyInfo)
	abi, isDecompiled, err := af.getABI(chainId, targetAddress, rpcURL)
	if err != nil {
		return nil, err
	}

	item := StorageItem{
		ABI:            abi,
		Implementation: implementation,
		IsProxy:        proxyInfo != nil,
		IsDecompiled:   isDecompiled,
	}
	af.storage.Set(chainId+"-"+address, item)

	return af.createResponse(item), nil
}

func (af *ABIFetcher) validateContract(ctx context.Context, client *ethclient.Client, address string) error {
	code, err := client.CodeAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return fmt.Errorf("failed to check contract code: %v", err)
	}
	if len(code) == 0 {
		return fmt.Errorf("the provided address is not a contract")
	}
	return nil
}

func (af *ABIFetcher) getTargetAddress(address string, proxyInfo *ProxyInfo) (string, interface{}) {
	targetAddress := address
	var implementation interface{} = nil
	if proxyInfo != nil && proxyInfo.Target != (common.Address{}) {
		targetAddress = proxyInfo.Target.Hex()
		implementation = targetAddress
	}
	return targetAddress, implementation
}

func (af *ABIFetcher) getABI(chainId string, targetAddress string, rpcURL string) (string, bool, error) {
	chainIdInt, _ := strconv.Atoi(chainId)
	api, ok := af.etherscanAPIs[chainIdInt]

	if ok {
		abi, err := api.GetABIFromEtherscan(targetAddress)
		if err == nil {
			return abi, false, nil
		}
	}

	abi, err := getABIFromHeimdall(targetAddress, rpcURL)
	if err != nil {
		return "", false, err
	}
	return abi, true, nil
}

func (af *ABIFetcher) createResponse(item StorageItem) gin.H {
	return gin.H{
		"abi":            item.ABI,
		"implementation": item.Implementation,
		"isProxy":        item.IsProxy,
		"isDecompiled":   item.IsDecompiled,
	}
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
