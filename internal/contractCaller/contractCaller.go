package contractCaller

import (
	"context"
	"errors"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/pkg/multicall"
	"github.com/Layr-Labs/go-sidecar/pkg/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
	"strings"
)

type OperatorRestakedStrategy struct {
	Operator string
	Avs      string
	Results  []common.Address
}

type IContractCaller interface {
	GetOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64) ([]common.Address, error)
	GetOperatorRestakedStrategiesMulticall(ctx context.Context, operatorRestakedStrategies []*OperatorRestakedStrategy, blockNumber uint64) ([]*OperatorRestakedStrategy, error)
}

type ContractCaller struct {
	EthereumClient *ethereum.Client
	Logger         *zap.Logger
}

func NewContractCaller(ec *ethereum.Client, l *zap.Logger) *ContractCaller {
	return &ContractCaller{
		EthereumClient: ec,
		Logger:         l,
	}
}

func getOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64, client *ethereum.Client, l *zap.Logger) ([]common.Address, error) {
	a, err := abi.JSON(strings.NewReader(serviceManagerAbi))
	if err != nil {
		l.Sugar().Errorw("GetOperatorRestakedStrategies - failed to parse abi", zap.Error(err))
		return nil, err
	}

	callerClient, err := client.GetEthereumContractCaller()
	if err != nil {
		l.Sugar().Errorw("GetOperatorRestakedStrategies - failed to get contract caller", zap.Error(err))
		return nil, err
	}

	contract := bind.NewBoundContract(common.HexToAddress(avs), a, callerClient, nil, nil)

	bigBlockNumber := big.NewInt(int64(blockNumber))

	results := make([]interface{}, 0)

	err = contract.Call(&bind.CallOpts{BlockNumber: bigBlockNumber, Context: ctx}, &results, "getOperatorRestakedStrategies", common.HexToAddress(operator))
	if err != nil {
		l.Sugar().Errorw("GetOperatorRestakedStrategies - failed to call contract method", zap.Error(err))
		return nil, err
	}

	return results[0].([]common.Address), nil
}

func (cc *ContractCaller) GetOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64) ([]common.Address, error) {
	return getOperatorRestakedStrategies(ctx, avs, operator, blockNumber, cc.EthereumClient, cc.Logger)
}

func (cc *ContractCaller) GetOperatorRestakedStrategiesMulticall(
	ctx context.Context,
	operatorRestakedStrategies []*OperatorRestakedStrategy,
	blockNumber uint64,
) ([]*OperatorRestakedStrategy, error) {
	a, err := abi.JSON(strings.NewReader(serviceManagerAbi))
	if err != nil {
		cc.Logger.Sugar().Errorw("GetOperatorRestakedStrategies - failed to parse abi", zap.Error(err))
		return nil, err
	}

	type MulticallAndError struct {
		Multicall *multicall.MultiCallMetaData[[]common.Address]
		Error     error
	}

	requests := utils.Map(operatorRestakedStrategies, func(ors *OperatorRestakedStrategy, index uint64) MulticallAndError {
		mc, err := multicall.MultiCall(common.HexToAddress(ors.Avs), a, func(data []byte) ([]common.Address, error) {
			res, err := a.Unpack("getOperatorRestakedStrategies", data)
			if err != nil {
				return nil, err
			}
			return res[0].([]common.Address), nil
		}, "getOperatorRestakedStrategies", common.HexToAddress(ors.Operator))
		return MulticallAndError{
			Multicall: mc,
			Error:     err,
		}
	})

	errs := []error{}
	for _, mc := range requests {
		if mc.Error != nil {
			errs = append(errs, mc.Error)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("Failed to create multicalls: %v", errors.Join(errs...))
	}

	allMultiCalls := utils.Map(requests, func(mc MulticallAndError, index uint64) *multicall.MultiCallMetaData[[]common.Address] {
		return mc.Multicall
	})

	client, err := cc.EthereumClient.GetEthereumContractCaller()
	if err != nil {
		cc.Logger.Sugar().Errorw("GetOperatorRestakedStrategiesMulticall - failed to get contract caller", zap.Error(err))
		return nil, err
	}

	multicallInstance, err := multicall.NewMulticallClient(ctx, client, &multicall.TMulticallClientOptions{
		MaxBatchSizeBytes: 4096,
	})
	if err != nil {
		cc.Logger.Sugar().Errorw("GetOperatorRestakedStrategiesMulticall - failed to create multicall client", zap.Error(err))
		return nil, err
	}

	results, err := multicall.DoMultiCallMany(*multicallInstance, allMultiCalls...)
	if err != nil {
		cc.Logger.Sugar().Errorw("GetOperatorRestakedStrategiesMulticall - failed to execute multicalls", zap.Error(err))
		return nil, err
	}

	if results == nil {
		return nil, fmt.Errorf("Results are nil")
	}

	return utils.Map(*results, func(result []common.Address, i uint64) *OperatorRestakedStrategy {
		oas := operatorRestakedStrategies[i]
		oas.Results = result
		return oas
	}), nil
}

type ReconciledContractCaller struct {
	EthereumClients []*ethereum.Client
	Logger          *zap.Logger
}

func NewRecociledContractCaller(ec []*ethereum.Client, l *zap.Logger) (*ReconciledContractCaller, error) {
	if len(ec) == 0 {
		return nil, fmt.Errorf("No ethereum clients provided")
	}
	return &ReconciledContractCaller{
		EthereumClients: ec,
		Logger:          l,
	}, nil
}

func (rcc *ReconciledContractCaller) GetOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64) ([]common.Address, error) {
	allResults := make([][]common.Address, 0)
	for i, ec := range rcc.EthereumClients {
		ec = ec
		results, err := getOperatorRestakedStrategies(ctx, avs, operator, blockNumber, ec, rcc.Logger)
		if err != nil {
			rcc.Logger.Sugar().Errorw("Error fetching results for client", zap.Error(err), zap.Int("clientIndex", i))
		} else {
			allResults = append(allResults, results)
		}
	}

	// make sure the number of total results is equal to the number of clients
	if len(allResults) != len(rcc.EthereumClients) {
		return nil, fmt.Errorf("Failed to fetch results for all clients")
	}

	if len(allResults) == 1 {
		return allResults[0], nil
	}

	// make sure that the results from each client are all the same length
	expectedLength := len(allResults[0])
	for i := 1; i < len(allResults); i++ {
		if len(allResults[i]) != expectedLength {
			return nil, fmt.Errorf("Client %d returned unexpected number of results", i)
		}
	}

	// check each item in each result to make sure they are all the same
	for _, clientResult := range allResults[1:] {
		for i, item := range clientResult {
			if allResults[0][i] != item {
				return nil, fmt.Errorf("Client results do not match")
			}
		}
	}

	return allResults[0], nil
}
