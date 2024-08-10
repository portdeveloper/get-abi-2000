![logo](https://github.com/user-attachments/assets/d71778e6-4fe5-48e3-aa5f-9ad17f783f5e)

Get-ABI-2000 is a Go-based API service that fetches and caches Ethereum contract ABIs. It supports multiple chains, proxy contract detection, and fallback to decompiled ABIs using the Heimdall API.

## Features

- Fetch ABIs for Ethereum, Sepolia, Optimism, and BSC
- Detect and handle proxy contracts
- Cache ABIs for faster subsequent requests
- Fallback to decompiled ABIs using Heimdall API
- Dockerized for easy deployment

## Prerequisites

- Go 1.22.5 or later
- Docker (for containerized deployment)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/portdeveloper/get-abi-2000.git
   cd get-abi-2000
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Set up environment variables:
   Create a `.env` file in the project root and add your API keys:
   ```
   ETHEREUM_API_KEY=your_ethereum_api_key
   SEPOLIA_API_KEY=your_sepolia_api_key
   OPTIMISM_API_KEY=your_optimism_api_key
   BSC_API_KEY=your_bsc_api_key
   ```

## Usage

### Running Locally

To run the server locally:
```go run .```

The server will start on `http://localhost:8080`.

### API Endpoints

1. Health Check:
   GET `/`

2. Fetch ABI:
   GET `/abi/:chainId/:address/*rpcUrl`

- `:chainId`: The chain ID (1 for Ethereum, 11155111 for Sepolia, 10 for Optimism, 56 for BSC)
- `:address`: The contract address
- `:rpcUrl`: The RPC URL for the blockchain (without 'https://')

Examples:

1. Mainnet (non-proxy, not decompiled):
   ```
   curl http://localhost:8080/abi/1/0x6B175474E89094C44Da98b954EedeAC495271d0F/rpc.ankr.com/eth
   ```

2. Mainnet (proxy, not decompiled):
   ```
   curl http://localhost:8080/abi/1/0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48/rpc.ankr.com/eth
   ```

3. Sepolia (non-proxy, decompiled):
   ```
   curl http://localhost:8080/abi/11155111/0x759c0e9d7858566df8ab751026bedce462ff42df/rpc.ankr.com/eth_sepolia
   ```

### Response

The API returns a JSON object with the following fields:

- `abi`: The contract ABI
- `implementation`: The implementation address if it's a proxy contract
- `isProxy`: Boolean indicating if the contract is a proxy
- `isDecompiled`: Boolean indicating if the ABI was decompiled using Heimdall

## Deployment

The project is configured for deployment on Fly.io.

1. Install the Fly CLI and authenticate:
   ```
   flyctl auth login
   ```

2. Deploy the application:
   ```
   flyctl deploy
   ```

## Testing

Run the tests with:
```go test ./...```  

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the MIT License.