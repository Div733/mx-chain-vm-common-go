package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-vm-common-go"
)

type esdtLocalBurn struct {
	baseAlwaysActiveHandler
	vmcommon.BlockchainDataProvider
	keyPrefix             []byte
	marshaller            vmcommon.Marshalizer
	globalSettingsHandler vmcommon.ExtendedESDTGlobalSettingsHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	drwaReader            drwaStateReader
	gasConfig             vmcommon.BaseOperationCost
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewESDTLocalBurnFunc returns the esdt local burn built-in function component
func NewESDTLocalBurnFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedESDTGlobalSettingsHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtLocalBurn, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &esdtLocalBurn{
		BlockchainDataProvider: NewBlockchainDataProvider(),
		keyPrefix:              []byte(baseESDTKeyPrefix),
		marshaller:             marshaller,
		globalSettingsHandler:  globalSettingsHandler,
		rolesHandler:           rolesHandler,
		funcGasCost:            funcGasCost,
		enableEpochsHandler:    enableEpochsHandler,
		mutExecution:           sync.RWMutex{},
	}

	return e, nil
}

func (e *esdtLocalBurn) SetDRWAReader(reader drwaStateReader) {
	e.mutExecution.Lock()
	e.drwaReader = reader
	e.mutExecution.Unlock()
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtLocalBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTLocalBurn
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT local burn function call
func (e *esdtLocalBurn) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkInputArgumentsForLocalAction(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	tokenID := vmInput.Arguments[0]
	drwaGasCost := uint64(0)
	if isDRWAEnforcementEnabled(e.enableEpochsHandler) {
		if e.drwaReader == nil {
			recordDRWAGateMetric(drwaGateMetricReaderMissing)
			return nil, errDRWAStateReaderMissing
		}

		regulated, drwaErr := evaluateDRWASenderTransfer(e.drwaReader, tokenID, vmInput.CallerAddr, acntSnd, e.CurrentRound())
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

	err = e.isAllowedToBurn(acntSnd, tokenID)
	if err != nil {
		return nil, err
	}

	if e.enableEpochsHandler.IsFlagEnabled(ConsistentTokensValuesLengthCheckFlag) {
		// TODO: core.MaxLenForESDTIssueMint should be renamed to something more general, such as MaxLenForESDTValues
		if len(vmInput.Arguments[1]) > core.MaxLenForESDTIssueMint {
			return nil, fmt.Errorf("%w: max length for esdt local burn value is %d", ErrInvalidArguments, core.MaxLenForESDTIssueMint)
		}
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	esdtTokenKey := append(e.keyPrefix, tokenID...)
	err = addToESDTBalance(acntSnd, esdtTokenKey, big.NewInt(0).Neg(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - e.funcGasCost - drwaGasCost}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTLocalBurn), vmInput.Arguments[0], 0, value, vmInput.CallerAddr)

	return vmOutput, nil
}

func (e *esdtLocalBurn) isAllowedToBurn(acntSnd vmcommon.UserAccountHandler, tokenID []byte) error {
	esdtTokenKey := append(e.keyPrefix, tokenID...)
	isBurnForAll := e.globalSettingsHandler.IsBurnForAll(esdtTokenKey)
	if isBurnForAll {
		return nil
	}

	return e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.ESDTRoleLocalBurn))
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtLocalBurn) IsInterfaceNil() bool {
	return e == nil
}

func checkBasicESDTArguments(vmInput *vmcommon.ContractCallInput) error {
	if vmInput == nil {
		return ErrNilVmInput
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) < core.MinLenArgumentsESDTTransfer {
		return ErrInvalidArguments
	}
	return nil
}

func checkInputArgumentsForLocalAction(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	funcGasCost uint64,
) error {
	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return err
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return ErrInvalidRcvAddr
	}
	if check.IfNil(acntSnd) {
		return ErrNilUserAccount
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return ErrNegativeValue
	}
	if vmInput.GasProvided < funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}
