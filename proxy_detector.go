package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	EIP1967LogicSlot               = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"
	EIP1967BeaconSlot              = "0xa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6cb3582b35133d50"
	EIP1822LogicSlot               = "0xc5f16f0fcc639fa48a6947836d9850f504798523bf8c9a3a87d5876cf622bcf7"
	OpenZeppelinImplementationSlot = "0x7050c9e0f4ca769c69bd3a8ef740bc37934f8e2c036e5a723fd8ee048ed3f8c3"
)

var (
	EIP1167BeaconMethods = []string{
		"0x5c60da1b00000000000000000000000000000000000000000000000000000000",
		"0xda52571600000000000000000000000000000000000000000000000000000000",
	}
	EIP897Interface           = []string{"0x5c60da1b00000000000000000000000000000000000000000000000000000000"}
	GnosisSafeProxyInterface  = []string{"0xa619486e00000000000000000000000000000000000000000000000000000000"}
	ComptrollerProxyInterface = []string{"0xbb82aa5e00000000000000000000000000000000000000000000000000000000"}
	EIP1167BytecodePrefix     = "0x363d3d373d3d3d363d"
	EIP1167BytecodeSuffix     = "57fd5bf3"
)

type ProxyInfo struct {
	Target    common.Address
	Immutable bool
	Type      string
}

func DetectProxyTarget(ctx context.Context, client *ethclient.Client, proxyAddress common.Address) (*ProxyInfo, error) {
	detectUsingBytecode := func() (*ProxyInfo, error) {
		bytecode, err := client.CodeAt(ctx, proxyAddress, nil)
		if err != nil {
			return nil, err
		}
		return parse1167Bytecode(bytecode)
	}

	detectUsingEIP1967LogicSlot := func() (*ProxyInfo, error) {
		logicAddress, err := client.StorageAt(ctx, proxyAddress, common.HexToHash(EIP1967LogicSlot), nil)
		if err != nil {
			return nil, err
		}
		if isZeroAddress(logicAddress) {
			return nil, fmt.Errorf("zero address in EIP1967 logic slot")
		}
		return &ProxyInfo{
			Target:    common.BytesToAddress(logicAddress),
			Immutable: false,
			Type:      "Eip1967Direct",
		}, nil
	}

	detectUsingEIP1967BeaconSlot := func() (*ProxyInfo, error) {
		beaconAddress, err := client.StorageAt(ctx, proxyAddress, common.HexToHash(EIP1967BeaconSlot), nil)
		if err != nil {
			return nil, err
		}
		if isZeroAddress(beaconAddress) {
			return nil, fmt.Errorf("zero address in EIP1967 beacon slot")
		}
		resolvedBeaconAddress := common.BytesToAddress(beaconAddress)
		for _, method := range EIP1167BeaconMethods {
			data, err := client.CallContract(ctx, ethereum.CallMsg{To: &resolvedBeaconAddress, Data: common.FromHex(method)}, nil)
			if err == nil && !isZeroAddress(data) {
				return &ProxyInfo{
					Target:    common.BytesToAddress(data[12:]),
					Immutable: false,
					Type:      "Eip1967Beacon",
				}, nil
			}
		}
		return nil, fmt.Errorf("beacon method calls failed")
	}

	detectUsingEIP1822LogicSlot := func() (*ProxyInfo, error) {
		logicAddress, err := client.StorageAt(ctx, proxyAddress, common.HexToHash(EIP1822LogicSlot), nil)
		if err != nil {
			return nil, err
		}
		if isZeroAddress(logicAddress) {
			return nil, fmt.Errorf("zero address in EIP1822 logic slot")
		}
		return &ProxyInfo{
			Target:    common.BytesToAddress(logicAddress),
			Immutable: false,
			Type:      "Eip1822",
		}, nil
	}

	detectUsingInterfaceCalls := func(data string) (*ProxyInfo, error) {
		result, err := client.CallContract(ctx, ethereum.CallMsg{To: &proxyAddress, Data: common.FromHex(data)}, nil)
		if err != nil {
			return nil, err
		}
		if len(result) < 32 {
			return nil, fmt.Errorf("invalid result length")
		}
		return &ProxyInfo{
			Target:    common.BytesToAddress(result[12:]),
			Immutable: false,
			Type:      "InterfaceCall",
		}, nil
	}

	detectUsingOpenZeppelinSlot := func() (*ProxyInfo, error) {
		implementationAddr, err := client.StorageAt(ctx, proxyAddress, common.HexToHash(OpenZeppelinImplementationSlot), nil)
		if err != nil {
			return nil, err
		}
		if isZeroAddress(implementationAddr) {
			return nil, fmt.Errorf("zero address in OpenZeppelin implementation slot")
		}
		return &ProxyInfo{
			Target:    common.BytesToAddress(implementationAddr),
			Immutable: false,
			Type:      "OpenZeppelin",
		}, nil
	}
	detectionMethods := []func() (*ProxyInfo, error){
		detectUsingBytecode,
		detectUsingEIP1967LogicSlot,
		detectUsingEIP1967BeaconSlot,
		detectUsingOpenZeppelinSlot,
		detectUsingEIP1822LogicSlot,
		func() (*ProxyInfo, error) { return detectUsingInterfaceCalls(EIP897Interface[0]) },
		func() (*ProxyInfo, error) { return detectUsingInterfaceCalls(GnosisSafeProxyInterface[0]) },
		func() (*ProxyInfo, error) { return detectUsingInterfaceCalls(ComptrollerProxyInterface[0]) },
	}

	results := make(chan *ProxyInfo, len(detectionMethods))
	errors := make(chan error, len(detectionMethods))

	for _, method := range detectionMethods {
		go func(m func() (*ProxyInfo, error)) {
			result, err := m()
			if err != nil {
				errors <- err
			} else if result != nil {
				results <- result
			}
		}(method)
	}

	for i := 0; i < len(detectionMethods); i++ {
		select {
		case result := <-results:
			return result, nil
		case <-errors:
			// Just ignore individual errors and continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("unable to detect proxy target")
}

func isZeroAddress(addr []byte) bool {
	return new(big.Int).SetBytes(addr).Cmp(big.NewInt(0)) == 0
}

func parse1167Bytecode(bytecode []byte) (*ProxyInfo, error) {
	bytecodeHex := common.Bytes2Hex(bytecode)
	if !strings.HasPrefix(bytecodeHex, strings.TrimPrefix(EIP1167BytecodePrefix, "0x")) {
		return nil, fmt.Errorf("not an EIP-1167 bytecode")
	}

	pushNHex := bytecodeHex[len(EIP1167BytecodePrefix)-2 : len(EIP1167BytecodePrefix)]
	addressLength := int(new(big.Int).SetBytes(common.FromHex(pushNHex)).Int64()) - 0x5f

	if addressLength < 1 || addressLength > 20 {
		return nil, fmt.Errorf("invalid address length in EIP-1167 bytecode")
	}

	addressFromBytecode := bytecodeHex[len(EIP1167BytecodePrefix) : len(EIP1167BytecodePrefix)+addressLength*2]
	suffixStartIndex := len(EIP1167BytecodePrefix) + addressLength*2 + 22

	if len(bytecodeHex) < suffixStartIndex+len(EIP1167BytecodeSuffix) ||
		!strings.HasSuffix(bytecodeHex[suffixStartIndex:], EIP1167BytecodeSuffix) {
		return nil, fmt.Errorf("invalid EIP-1167 bytecode suffix")
	}

	return &ProxyInfo{
		Target:    common.HexToAddress(addressFromBytecode),
		Immutable: true,
		Type:      "Eip1167",
	}, nil
}
