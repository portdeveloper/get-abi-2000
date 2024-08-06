package main

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

func TestProxyDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	client, err := ethclient.Dial("https://mainnet.infura.io/v3/3ceeb58f319b42daad1861eadb3b232b")
	if err != nil {
		t.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name              string
		proxyAddress      string
		expectedTarget    string
		expectedType      string
		expectedImmutable bool
	}{
		{
			name:              "EIP-1967 direct proxy",
			proxyAddress:      "0xA7AeFeaD2F25972D80516628417ac46b3F2604Af",
			expectedTarget:    "0x4bd844f72a8edd323056130a86fc624d0dbcf5b0",
			expectedType:      "Eip1967Direct",
			expectedImmutable: false,
		},
		{
			name:              "EIP-1967 beacon proxy",
			proxyAddress:      "0xDd4e2eb37268B047f55fC5cAf22837F9EC08A881",
			expectedTarget:    "0xe5c048792dcf2e4a56000c8b6a47f21df22752d1",
			expectedType:      "Eip1967Beacon",
			expectedImmutable: false,
		},
		{
			name:              "EIP-1967 beacon variant proxy",
			proxyAddress:      "0x114f1388fAB456c4bA31B1850b244Eedcd024136",
			expectedTarget:    "0x0fa0fd98727c443dd5275774c44d27cff9d279ed",
			expectedType:      "Eip1967Beacon",
			expectedImmutable: false,
		},
		{
			name:              "OpenZeppelin proxy",
			proxyAddress:      "0xC986c2d326c84752aF4cC842E033B9ae5D54ebbB",
			expectedTarget:    "0x0656368c4934e56071056da375d4a691d22161f8",
			expectedType:      "OpenZeppelin",
			expectedImmutable: false,
		},
		{
			name:              "EIP-1167 minimal proxy",
			proxyAddress:      "0x6d5d9b6ec51c15f45bfa4c460502403351d5b999",
			expectedTarget:    "0x210ff9ced719e9bf2444dbc3670bac99342126fa",
			expectedType:      "Eip1167",
			expectedImmutable: true,
		},
		{
			name:              "Safe proxy",
			proxyAddress:      "0x0DA0C3e52C977Ed3cBc641fF02DD271c3ED55aFe",
			expectedTarget:    "0xd9db270c1b5e3bd161e8c8503c55ceabee709552",
			expectedType:      "InterfaceCall",
			expectedImmutable: false,
		},
		{
			name:              "Compound's custom proxy",
			proxyAddress:      "0x3d9819210A31b4961b30EF54bE2aeD79B9c9Cd3B",
			expectedTarget:    "0xbafe01ff935c7305907c33bf824352ee5979b526",
			expectedType:      "InterfaceCall",
			expectedImmutable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxyAddress := common.HexToAddress(tt.proxyAddress)
			proxyInfo, err := DetectProxyTarget(ctx, client, proxyAddress)
			if err != nil {
				t.Fatalf("Failed to detect proxy target: %v", err)
			}

			assert.NotNil(t, proxyInfo, "ProxyInfo should not be nil")
			expectedTarget := common.HexToAddress(tt.expectedTarget)
			assert.Equal(t, expectedTarget, proxyInfo.Target, "Detected target does not match expected")
			assert.Equal(t, tt.expectedType, proxyInfo.Type, "Detected proxy type does not match expected")
			assert.Equal(t, tt.expectedImmutable, proxyInfo.Immutable, "Detected immutability does not match expected")
		})
	}
}
