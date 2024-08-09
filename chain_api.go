package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type ChainAPI interface {
	GetABIFromEtherscan(address string) (string, error)
}

type GenericEtherscanAPI struct {
	BaseURL string
	EnvKey  string
}

func (e *GenericEtherscanAPI) GetABIFromEtherscan(address string) (string, error) {
	apiKey := os.Getenv(e.EnvKey)
	if apiKey == "" {
		return "", fmt.Errorf("API key not set for chain")
	}
	url := fmt.Sprintf("%s?module=contract&action=getabi&address=%s&apikey=%s", e.BaseURL, address, apiKey)
	return fetchABI(url)
}

func fetchABI(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Status != "1" {
		return "", fmt.Errorf("API error: %s", result.Message)
	}

	return result.Result, nil
}
