## Running

### Directly using Go

*Dependencies*

* Go 1.22
* gRPCurl (for testing)

```bash
# Create the directory to hold the sqlite database
mkdir ./sqlite || true

SIDECAR_DEBUG=false \
SIDECAR_ETHEREUM_RPC_BASE_URL="http://34.229.43.36:8545" \
SIDECAR_ENVIRONMENT="testnet" \
SIDECAR_NETWORK="holesky" \
SIDECAR_ETHERSCAN_API_KEYS="<your etherscan key>" \
SIDECAR_STATSD_URL="localhost:8125" \
SIDECAR_SQLITE_DB_FILE_PATH="./sqlite/sidecar.db" \
go run cmd/sidecar/main.go
```

### Using the public Docker container
```bash
# Create the directory to hold the sqlite database
mkdir ./sqlite || true

docker run -it --rm \
  -e SIDECAR_DEBUG=false \
  -e SIDECAR_ETHEREUM_RPC_BASE_URL="http://34.229.43.36:8545" \
  -e SIDECAR_ENVIRONMENT="testnet" \
  -e SIDECAR_NETWORK="holesky" \
  -e SIDECAR_ETHERSCAN_API_KEYS="<your etherscan key>" \
  -e SIDECAR_STATSD_URL="localhost:8125" \
  -e SIDECAR_SQLITE_DB_FILE_PATH="/sqlite/sidecar.db" \
  -v "$(pwd)/sqlite:/sqlite" \
  --tty -i \
  public.ecr.aws/z6g0f8n7/go-sidecar:latest
```

### Build and run a container locally
```bash
# Create the directory to hold the sqlite database
mkdir ./sqlite || true

make docker-buildx-self

docker run \
  -e "SIDECAR_DEBUG=false" \
  -e "SIDECAR_ETHEREUM_RPC_BASE_URL=http://34.229.43.36:8545" \
  -e "SIDECAR_ENVIRONMENT=testnet" \
  -e "SIDECAR_NETWORK=holesky" \
  -e "SIDECAR_ETHERSCAN_API_KEYS='<your etherscan key>'" \
  -e "SIDECAR_STATSD_URL=localhost:8125" \
  -e "SIDECAR_SQLITE_DB_FILE_PATH=/sqlite/sidecar.db" \
  -v "$(pwd)/sqlite:/sqlite" \
  --tty -i \
  go-sidecar:latest
```

## RPC Routes

### Get current block height

```bash
grpcurl -plaintext -d '{}'  localhost:7100 eigenlayer.sidecar.api.v1.Rpc/GetBlockHeight
```

### Get the stateroot at a block height

```bash
grpcurl -plaintext -d '{ "blockNumber": 1140438 }'  localhost:7100 eigenlayer.sidecar.api.v1.Rpc/GetStateRoot
