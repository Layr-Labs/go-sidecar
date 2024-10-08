# Development

## Dependencies

* Go 1.22
* Sqlite3 (version 9.x.x)
* Python3 (version 3.12)
* GCC (for building sqlite3 extensions)
* Homebrew (if on MacOS)

## Supported build environments

* MacOS
* Linux (Ubuntu/Debian)

## Environment setup

If you have basic build tools like `make` already installed, you can run:

```bash
make deps
```

If you are starting from a fresh linux install with nothing, run:
```bash
./scripts/installDeps.sh

make deps
```

## Testing

First run:

```bash
make build
```

This will build everything you need, including the sqlite extensions if they have not yet been built.

### Entire suite

```bash
make test
```

### One off tests

`goTest.sh` is a convenience script that sets up all relevant environment variables and runs the tests.

```bash
./scripts/goTest.sh -v ./internal/types/numbers -v -p 1 -run '^Test_Numbers$' 
```

### Long-running Rewards tests

The rewards tests are time and resource intensive and are not enabled to run by default.

*Download the test data*

```bash
./scripts/downloadTestData.sh testnet-reduced
```
Run the rewards tests

```bash
REWARDS_TEST_CONTEXT=testnet-reduced TEST_REWARDS=true ./scripts/goTest.sh -timeout 0 ./pkg/rewards -v -p 1 -run '^Test_Rewards$'
````

Options:
* `REWARDS_TEST_CONTEXT` determines which test data to use.
* `TEST_REWARDS` enables the rewards tests.

# Build

This will build the go binary and the associated sqlite3 extensions:

```bash
make deps

make build
```

# Running

### Directly using Go

```bash
# Create the directory to hold the sqlite database
mkdir ./sidecar-data || true

./bin/sidecar run \
    --ethereum.rpc-url="http://34.229.43.36:8545" \
    --chain="holesky" \
    --etherscan.api-keys="<your etherscan key>" \
    --statsd.url="localhost:8125" \
    --datadir="./sidecar-data"

```

### Using the public Docker container
```bash
# Create the directory to hold the sqlite database
mkdir ./sqlite || true

docker run -it --rm \
  -e SIDECAR_DEBUG=false \
  -e SIDECAR_ETHEREUM_RPC_BASE_URL="http://34.229.43.36:8545" \
  -e SIDECAR_CHAIN="holesky" \
  -e SIDECAR_ETHERSCAN_API_KEYS="<your etherscan key>" \
  -e SIDECAR_STATSD_URL="localhost:8125" \
  -e SIDECAR_DATADIR="/sidecar" \
  -v "$(pwd)/sqlite:/sidecar" \
  --tty -i \
  public.ecr.aws/z6g0f8n7/go-sidecar:latest run
```

### Build and run a container locally
```bash
# Create the directory to hold the sqlite database
mkdir ./sqlite || true

make docker-buildx-self

docker run \
  -e "SIDECAR_DEBUG=false" \
  -e "SIDECAR_ETHEREUM_RPC_BASE_URL=http://34.229.43.36:8545" \
  -e "SIDECAR_CHAIN=holesky" \
  -e "SIDECAR_ETHERSCAN_API_KEYS='<your etherscan key>'" \
  -e "SIDECAR_STATSD_URL=localhost:8125" \
  -e SIDECAR_DATADIR="/sidecar" \
  -v "$(pwd)/sqlite:/sidecar" \
  --tty -i \
  go-sidecar:latest run
```

## RPC Routes

### Get current block height

```bash
grpcurl -plaintext -d '{}'  localhost:7100 eigenlayer.sidecar.api.v1.Rpc/GetBlockHeight
```

### Get the stateroot at a block height

```bash
grpcurl -plaintext -d '{ "blockNumber": 1140438 }'  localhost:7100 eigenlayer.sidecar.api.v1.Rpc/GetStateRoot
