package main

type InvalidInputError struct {
	message string
}

func (e *InvalidInputError) Error() string {
	return e.message
}

type ContractNotFoundError struct {
	address string
}

func (e *ContractNotFoundError) Error() string {
	return "Contract not found at address: " + e.address
}

type EtherscanAPIError struct {
	message string
}

func (e *EtherscanAPIError) Error() string {
	return "Etherscan API error: " + e.message
}
