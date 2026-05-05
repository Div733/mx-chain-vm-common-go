package builtInFunctions

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type esdtBurn struct {
	baseActiveHandler
	vmcommon.BlockchainDataProvider
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	drwaReader            drwaStateReader
	gasConfig             vmcommon.BaseOperationCost
	funcGasCost           uint64
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	mutExecution          sync.RWMutex
}

// NewESDTBurnFunc returns the esdt burn built-in function component
func NewESDTBurnFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtBurn, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &esdtBurn{
		BlockchainDataProvider: NewBlockchainDataProvider(),
		enableEpochsHandler:    enableEpochsHandler,
		funcGasCost:            funcGasCost,
		marshaller:             marshaller,
		keyPrefix:              []byte(baseESDTKeyPrefix),
		globalSettingsHandler:  globalSettingsHandler,
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(GlobalMintBurnFlag)
	}

	return e, nil
}

func (e *esdtBurn) SetDRWAReader(reader drwaStateReader) {
	e.mutExecution.Lock()
	e.drwaReader = reader
	e.mutExecution.Unlock()
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTBurn
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT burn function call
func (e *esdtBurn) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) != 2 {
		return nil, ErrInvalidArguments
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, ErrNegativeValue
	}
	if !bytes.Equal(vmInput.RecipientAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntSnd) {
		return nil, ErrNilUserAccount
	}
	drwaGasCost := uint64(0)
	if isDRWAEnforcementEnabled(e.enableEpochsHandler) {
		if e.drwaReader == nil {
			recordDRWAGateMetric(drwaGateMetricReaderMissing)
			return nil, errDRWAStateReaderMissing
		}

		regulated, drwaErr := evaluateDRWASenderTransfer(e.drwaReader, vmInput.Arguments[0], vmInput.CallerAddr, acntSnd, e.CurrentRound())
		if drwaErr != nil {
			return nil, drwaErr
		}
		if regulated {
			drwaGasCost = computeDRWAReadGasCost(e.gasConfig, e.funcGasCost, 4)
			if vmInput.GasProvided < e.funcGasCost+drwaGasCost {
				return nil, ErrNotEnoughGas
			}
		}
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	if vmInput.GasProvided < e.funcGasCost {
		return nil, ErrNotEnoughGas
	}

	err = addToESDTBalance(acntSnd, esdtTokenKey, big.NewInt(0).Neg(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, e.funcGasCost)
	if drwaGasCost > 0 {
		gasRemaining = computeGasRemaining(acntSnd, vmInput.GasProvided, e.funcGasCost+drwaGasCost)
	}
	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			core.BuiltInFunctionESDTBurn,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTBurn), vmInput.Arguments[0], 0, value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtBurn) IsInterfaceNil() bool {
	return e == nil
}
