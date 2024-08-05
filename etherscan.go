// etherscan.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const etherscanAPIURL = "https://api.etherscan.io/api"

func getABIFromEtherscan(address string, apiKey string) (string, error) {
	url := fmt.Sprintf("%s?module=contract&action=getabi&address=%s&apikey=%s", etherscanAPIURL, address, apiKey)

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
		return "", fmt.Errorf("Etherscan API error: %s", result.Message)
	}

	return result.Result, nil
}
