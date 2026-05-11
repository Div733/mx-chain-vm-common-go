package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createESDTNFTMultiTransferWithStubArguments() *esdtNFTMultiTransfer {
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag || flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	}

	multiTransfer, _ := NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		enableEpochsHandler,
		&mock.ESDTRoleHandlerStub{},
		createNewESDTDataStorageHandler(),
	)

	return multiTransfer
}

func createAccountsAdapterWithMap() vmcommon.AccountsAdapter {
	mapAccounts := make(map[string]vmcommon.UserAccountHandler)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
		GetExistingAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
		SaveAccountCalled: func(account vmcommon.AccountHandler) error {
			mapAccounts[string(account.AddressBytes())] = account.(vmcommon.UserAccountHandler)
			return nil
		},
	}
	return accounts
}

func createESDTNFTMultiTransferWithMockArguments(selfShard uint32, numShards uint32, globalSettingsHandler vmcommon.GlobalMetadataHandler) *esdtNFTMultiTransfer {
	return createESDTNFTMultiTransferWithMockArgumentsWithLogEventFlag(selfShard, numShards, globalSettingsHandler, false)
}

func createESDTNFTMultiTransferWithMockArgumentsWithLogEventFlag(selfShard uint32, numShards uint32, globalSettingsHandler vmcommon.GlobalMetadataHandler, isScToScEventLogEnabled bool) *esdtNFTMultiTransfer {
	marshaller := &mock.MarshalizerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(numShards)
	shardCoordinator.CurrentShard = selfShard
	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		lastByte := uint32(address[len(address)-1])
		return lastByte
	}
	accounts := createAccountsAdapterWithMap()

	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag ||
				flag == CheckCorrectTokenIDForTransferRoleFlag ||
				(flag == ScToScLogEventFlag && isScToScEventLogEnabled)
		},
	}
	multiTransfer, _ := NewESDTNFTMultiTransferFunc(
		1,
		marshaller,
		globalSettingsHandler,
		accounts,
		shardCoordinator,
		vmcommon.BaseOperationCost{},
		enableEpochsHandler,
		&mock.ESDTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
				if bytes.Equal(action, []byte(core.ESDTRoleTransfer)) {
					return ErrActionNotAllowed
				}
				return nil
			},
		},
		createNewESDTDataStorageHandlerWithArgs(globalSettingsHandler, accounts, enableEpochsHandler),
	)

	return multiTransfer
}

func TestNewESDTNFTMultiTransferFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			nil,
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.EnableEpochsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
		)
		assert.True(t, check.IfNil(multiTransfer))
		assert.Equal(t, ErrNilMarshalizer, err)
	})
	t.Run("nil global settings should error", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			&mock.MarshalizerMock{},
			nil,
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.EnableEpochsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
		)
		assert.True(t, check.IfNil(multiTransfer))
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil accounts adapter should error", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			nil,
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.EnableEpochsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
		)
		assert.True(t, check.IfNil(multiTransfer))
		assert.Equal(t, ErrNilAccountsAdapter, err)
	})
	t.Run("nil shard coordinator should error", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			nil,
			vmcommon.BaseOperationCost{},
			&mock.EnableEpochsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
		)
		assert.True(t, check.IfNil(multiTransfer))
		assert.Equal(t, ErrNilShardCoordinator, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			nil,
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
		)
		assert.True(t, check.IfNil(multiTransfer))
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.EnableEpochsHandlerStub{},
			nil,
			createNewESDTDataStorageHandler(),
		)
		assert.True(t, check.IfNil(multiTransfer))
		assert.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil storage handler should error", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.EnableEpochsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			nil,
		)
		assert.True(t, check.IfNil(multiTransfer))
		assert.Equal(t, ErrNilESDTNFTStorageHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		multiTransfer, err := NewESDTNFTMultiTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.EnableEpochsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
		)
		assert.False(t, check.IfNil(multiTransfer))
		assert.Nil(t, err)
	})
}

func TestESDTNFTMultiTransfer_SetPayable(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithStubArguments()
	err := multiTransfer.SetPayableChecker(nil)
	assert.Equal(t, ErrNilPayableHandler, err)

	handler := &mock.PayableHandlerStub{}
	err = multiTransfer.SetPayableChecker(handler)
	assert.Nil(t, err)
	assert.True(t, handler == multiTransfer.payableHandler) // pointer testing
}

func TestESDTNFTMultiTransfer_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithStubArguments()
	multiTransfer.SetNewGasConfig(nil)
	assert.Equal(t, uint64(0), multiTransfer.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, multiTransfer.gasConfig)

	gasCost := createMockGasCost()
	multiTransfer.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.ESDTNFTMultiTransfer, multiTransfer.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, multiTransfer.gasConfig)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionInvalidArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithStubArguments()
	vmOutput, err := multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	}
	vmOutput, err = multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidArguments, err)

	multiTransfer.shardCoordinator = &mock.ShardCoordinatorStub{ComputeIdCalled: func(address []byte) uint32 {
		return core.MetachainShardId
	}}

	token1 := []byte("token")
	senderAddress := bytes.Repeat([]byte{2}, 32)
	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{core.ESDTSCAddress, big.NewInt(1).Bytes(), token1, big.NewInt(1).Bytes(), big.NewInt(1).Bytes()},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmOutput, err = multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnSameShardWithScCall(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	payableChecker, _ := NewPayableCheckFunc(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == FixAsyncCallbackCheckFlag || flag == CheckFunctionArgumentFlag
			},
		})

	_ = multiTransfer.SetPayableChecker(payableChecker)
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransfer.marshaller, sender.(vmcommon.UserAccountHandler))

	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshaller, destination.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransfer.marshaller, destination.(vmcommon.UserAccountHandler))

	_ = multiTransfer.accounts.SaveAccount(sender)
	_ = multiTransfer.accounts.SaveAccount(destination)
	_, _ = multiTransfer.accounts.Commit()

	// reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransfer.accounts.SaveAccount(sender)
	_, _ = multiTransfer.accounts.Commit()

	// reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransfer.marshaller, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransfer.marshaller, sender, token2, 0, big.NewInt(2))
	testNFTTokenShouldExist(t, multiTransfer.marshaller, destination, token1, tokenNonce, big.NewInt(4))
	testNFTTokenShouldExist(t, multiTransfer.marshaller, destination, token2, 0, big.NewInt(4))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionRejectsEmptyArguments(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})

	vmOutput, err := multiTransfer.ProcessBuiltinFunction(
		&mock.UserAccountStub{},
		&mock.UserAccountStub{},
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr:  bytes.Repeat([]byte{2}, 32),
				Arguments:   [][]byte{},
				CallValue:   big.NewInt(0),
				GasProvided: 1_000,
			},
			RecipientAddr: bytes.Repeat([]byte{2}, 32),
		},
	)

	require.Nil(t, vmOutput)
	require.ErrorIs(t, err, ErrInvalidArguments)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnSameShardShouldCheckTokenValueLength(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	payableChecker, _ := NewPayableCheckFunc(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == FixAsyncCallbackCheckFlag || flag == CheckFunctionArgumentFlag
			},
		})

	_ = multiTransfer.SetPayableChecker(payableChecker)
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransfer.marshaller, sender.(vmcommon.UserAccountHandler))

	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshaller, destination.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransfer.marshaller, destination.(vmcommon.UserAccountHandler))

	_ = multiTransfer.accounts.SaveAccount(sender)
	_ = multiTransfer.accounts.SaveAccount(destination)
	_, _ = multiTransfer.accounts.Commit()

	// reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantity, _ := big.NewInt(0).SetString("1"+strings.Repeat("0", 250), 10)
	smallQuantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, smallQuantityBytes, token2, big.NewInt(0).Bytes(), quantity.Bytes()},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	// before flag activation
	vmOutput, err := multiTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Contains(t, err.Error(), "insufficient quantity")
	require.Empty(t, vmOutput)

	// after flag activation
	multiTransfer.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ConsistentTokensValuesLengthCheckFlag
		},
	}
	vmOutput, err = multiTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Contains(t, err.Error(), "max length for a transfer value is")
	require.Empty(t, vmOutput)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationDoesNotHoldingNFTWithSCCall(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableChecker(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1000000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred

	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshaller, destination, token1, tokenNonce, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshaller, destination, token2, 0, big.NewInt(1))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationAddToEsdtBalanceShouldErr(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableChecker(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1000000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred

	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	multiTransferDestinationShard.globalSettingsHandler = &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(tokenKey []byte) bool {
			esdtTokenKey := []byte(baseESDTKeyPrefix)
			esdtTokenKey = append(esdtTokenKey, token2...)
			return bytes.Equal(tokenKey, esdtTokenKey)
		},
	}
	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Error(t, err)
	require.Equal(t, "esdt token is paused for token token2", err.Error())
	require.Nil(t, vmOutput)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsOneTransfer(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableChecker(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(1).Bytes(), token1, nonceBytes, quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens1 := big.NewInt(1000)
	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, destinationNumTokens1, multiTransferDestinationShard.marshaller, destination.(vmcommon.UserAccountHandler))
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	expectedTokens1 := big.NewInt(0).Add(destinationNumTokens1, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshaller, destination, token1, tokenNonce, expectedTokens1)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationHoldsNFT(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableChecker(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token2, 0, big.NewInt(2))          // 3 initial - 1 transferred
	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens1 := big.NewInt(1000)
	destinationNumTokens2 := big.NewInt(1000)
	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, destinationNumTokens1, multiTransferDestinationShard.marshaller, destination.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, destinationNumTokens2, multiTransferDestinationShard.marshaller, destination.(vmcommon.UserAccountHandler))
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	expectedTokens1 := big.NewInt(0).Add(destinationNumTokens1, big.NewInt(1))
	expectedTokens2 := big.NewInt(0).Add(destinationNumTokens2, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshaller, destination, token1, tokenNonce, expectedTokens1)
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshaller, destination, token2, 0, expectedTokens2)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsShouldErr(t *testing.T) {
	t.Parallel()

	payableChecker, _ := NewPayableCheckFunc(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == FixAsyncCallbackCheckFlag || flag == CheckFunctionArgumentFlag
			},
		})

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableChecker(payableChecker)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableChecker(payableChecker)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token2, 0, big.NewInt(2))
	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens := big.NewInt(1000)
	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, destinationNumTokens, multiTransferDestinationShard.marshaller, destination.(vmcommon.UserAccountHandler))
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	payableChecker, _ = NewPayableCheckFunc(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return false, nil
			},
		}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == FixAsyncCallbackCheckFlag || flag == CheckFunctionArgumentFlag
			},
		})

	_ = multiTransferDestinationShard.SetPayableChecker(payableChecker)
	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Error(t, err)
	require.Equal(t, "sending value to non payable contract", err.Error())
	require.Nil(t, vmOutput)

	// check the multi transfer for fungible ESDT transfers as well
	vmInput.Arguments = [][]byte{big.NewInt(2).Bytes(), token1, big.NewInt(0).Bytes(), quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes}
	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Error(t, err)
	require.Equal(t, "sending value to non payable contract", err.Error())
	require.Nil(t, vmOutput)
}

func TestESDTNFTMultiTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	globalSettings := &mock.GlobalSettingsHandlerStub{}
	transferFunc := createESDTNFTMultiTransferWithMockArguments(0, 1, globalSettings)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress[31] = 0
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	esdtFrozen := ESDTUserMetadata{Frozen: true}

	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	destination, _ := transferFunc.accounts.LoadAccount(destinationAddress)
	tokenId := append(keyPrefix, token1...)
	esdtKey := computeESDTNFTTokenKey(tokenId, tokenNonce)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(0), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ := transferFunc.marshaller.Marshal(esdtToken)
	_ = destination.(vmcommon.UserAccountHandler).AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)
	_ = transferFunc.accounts.SaveAccount(destination)
	_, _ = transferFunc.accounts.Commit()

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("%s for token %s", ErrESDTIsFrozenForAccount, string(token1)), err.Error())

	globalSettings.IsLimiterTransferCalled = func(token []byte) bool {
		return true
	}
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("%s for token %s", ErrActionNotAllowed, string(token1)), err.Error())

	globalSettings.IsLimiterTransferCalled = func(token []byte) bool {
		return false
	}
	vmInput.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Nil(t, err)
}

func TestESDTNFTMultiTransfer_NotEnoughGas(t *testing.T) {
	t.Parallel()

	transferFunc := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, err, ErrNotEnoughGas)
}

func TestESDTNFTMultiTransfer_WithEgldValue(t *testing.T) {
	t.Parallel()

	transferFunc := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()

	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(1),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, output)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
}

func TestComputeInsufficientQuantityESDTError(t *testing.T) {
	t.Parallel()

	resErr := computeInsufficientQuantityESDTError([]byte("my-token"), 0)
	require.NotNil(t, resErr)
	require.Equal(t, errors.New("insufficient quantity for token: my-token").Error(), resErr.Error())

	resErr = computeInsufficientQuantityESDTError([]byte("my-token-2"), 5)
	require.NotNil(t, resErr)
	require.Equal(t, errors.New("insufficient quantity for token: my-token-2 nonce 5").Error(), resErr.Error())
}

func TestESDTNFTMultiTransfer_LogEventsEpochActivationTest(t *testing.T) {
	t.Parallel()

	vmOutput, err := runMultiTransfer(t, false)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	require.Equal(t, 2, len(vmOutput.Logs))
	require.Equal(t, []byte("MultiESDTNFTTransfer"), vmOutput.Logs[0].Identifier)
	require.Equal(t, 4, len(vmOutput.Logs[0].Topics))
	require.Equal(t, []byte("token1"), vmOutput.Logs[0].Topics[0])
	require.Equal(t, []byte("MultiESDTNFTTransfer"), vmOutput.Logs[1].Identifier)
	require.Equal(t, 4, len(vmOutput.Logs[1].Topics))
	require.Equal(t, []byte("token2"), vmOutput.Logs[1].Topics[0])

	vmOutput, err = runMultiTransfer(t, true)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	require.Equal(t, 1, len(vmOutput.Logs))
	require.Equal(t, []byte("MultiESDTNFTTransfer"), vmOutput.Logs[0].Identifier)
	require.Equal(t, 7, len(vmOutput.Logs[0].Topics))
	require.Equal(t, []byte("token1"), vmOutput.Logs[0].Topics[0])
	require.Equal(t, []byte("token2"), vmOutput.Logs[0].Topics[3])
}

func runMultiTransfer(t *testing.T, isScToScEventLogEnabled bool) (*vmcommon.VMOutput, error) {
	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArgumentsWithLogEventFlag(0, 2, &mock.GlobalSettingsHandlerStub{}, isScToScEventLogEnabled)
	_ = multiTransferSenderShard.SetPayableChecker(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArgumentsWithLogEventFlag(1, 2, &mock.GlobalSettingsHandlerStub{}, isScToScEventLogEnabled)
	_ = multiTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)
	token2Nonce := uint64(2)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.NonFungible, token2Nonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	nonce2Bytes := big.NewInt(int64(token2Nonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments: [][]byte{destinationAddress, big.NewInt(2).Bytes(),
				token1, nonceBytes, quantityBytes,
				token2, nonce2Bytes, quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	return multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
}

// creates accounts with token1 and eGLD balances.
func createSetupForMultiTransferWithEGLD(t *testing.T) (*vmcommon.ContractCallInput, *esdtNFTMultiTransfer) {
	multiTransfer := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	payableChecker, _ := NewPayableCheckFunc(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == FixAsyncCallbackCheckFlag || flag == CheckFunctionArgumentFlag
			},
		})

	_ = multiTransfer.SetPayableChecker(payableChecker)
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte(vmcommon.EGLDIdentifier)
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshaller, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshaller, destination.(vmcommon.UserAccountHandler))

	_ = sender.(vmcommon.UserAccountHandler).AddToBalance(initialTokens)

	_ = multiTransfer.accounts.SaveAccount(sender)
	_ = multiTransfer.accounts.SaveAccount(destination)
	_, _ = multiTransfer.accounts.Commit()

	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	return vmInput, multiTransfer
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnSameShardWithScCallWithEGLDNotEnabled(t *testing.T) {
	t.Parallel()

	vmInput, multiTransfer := createSetupForMultiTransferWithEGLD(t)
	multiTransfer.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag != EGLDInESDTMultiTransferFlag
		},
	}
	multiTransfer.SetDRWAReader(newNoopDRWAReader())

	sender, err := multiTransfer.accounts.LoadAccount(vmInput.CallerAddr)
	require.Nil(t, err)
	destination, err := multiTransfer.accounts.LoadAccount(vmInput.Arguments[0])
	require.Nil(t, err)

	_, err = multiTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.NotNil(t, err)
	require.Equal(t, err.Error(), "insufficient quantity for token: EGLD-000000 for token EGLD-000000")
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunction_DRWACachesPolicyPerTokenOnSenderShard(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	multiTransfer.enableEpochsHandler = drwaEnabledEpochsHandler(ESDTNFTImprovementV1Flag, CheckCorrectTokenIDForTransferRoleFlag)

	payableChecker, err := NewPayableCheckFunc(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		},
		&mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == FixAsyncCallbackCheckFlag || flag == CheckFunctionArgumentFlag
			},
		},
	)
	require.NoError(t, err)
	require.NoError(t, multiTransfer.SetPayableChecker(payableChecker))

	tokenPolicyCalls := 0
	multiTransfer.SetDRWAReader(&drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			tokenPolicyCalls++
			return &drwaTokenPolicyView{DRWAEnabled: true}, nil
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			return &drwaHolderMirrorView{
				KYCStatus: "approved",
				AMLStatus: "approved",
			}, nil
		},
	})

	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1

	senderAccount, err := multiTransfer.accounts.LoadAccount(senderAddress)
	require.NoError(t, err)
	destinationAccount, err := multiTransfer.accounts.LoadAccount(destinationAddress)
	require.NoError(t, err)

	createESDTNFTToken([]byte("CARBON-MULTI"), core.Fungible, 0, big.NewInt(3), multiTransfer.marshaller, senderAccount.(vmcommon.UserAccountHandler))
	require.NoError(t, multiTransfer.accounts.SaveAccount(senderAccount))
	require.NoError(t, multiTransfer.accounts.SaveAccount(destinationAccount))
	_, _ = multiTransfer.accounts.Commit()

	senderAccount, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.NoError(t, err)
	destinationAccount, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.NoError(t, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  senderAddress,
			CallValue:   big.NewInt(0),
			GasProvided: 100000,
			Arguments: [][]byte{
				destinationAddress,
				big.NewInt(1).Bytes(),
				[]byte("CARBON-MULTI"),
				big.NewInt(0).Bytes(),
				big.NewInt(1).Bytes(),
			},
		},
		RecipientAddr: senderAddress,
	}

	output, err := multiTransfer.ProcessBuiltinFunction(senderAccount.(vmcommon.UserAccountHandler), destinationAccount.(vmcommon.UserAccountHandler), vmInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, 1, tokenPolicyCalls, "multi transfer should read DRWA policy once per token and reuse it across validation passes")
}

func TestESDTNFTMultiTransfer_TransferOneTokenOnSenderShard_DoesNotMutateBeforeLimitedTransferAuthorization(t *testing.T) {
	t.Parallel()

	tokenID := []byte("limited-token")
	nonce := uint64(7)
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{3}, 32)
	sender := mock.NewUserAccount(senderAddress)
	destination := mock.NewUserAccount(destinationAddress)

	saveCalls := 0
	storageHandler := &mock.ESDTNFTStorageHandlerStub{
		GetESDTNFTTokenOnSenderCalled: func(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, providedNonce uint64) (*esdt.ESDigitalToken, error) {
			require.Equal(t, append([]byte(baseESDTKeyPrefix), tokenID...), esdtTokenKey)
			require.Equal(t, nonce, providedNonce)
			return &esdt.ESDigitalToken{
				Type:  uint32(core.NonFungible),
				Value: big.NewInt(2),
			}, nil
		},
		SaveESDTNFTTokenCalled: func(senderAddress []byte, acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, providedNonce uint64, esdtData *esdt.ESDigitalToken, saveArgs vmcommon.NftSaveArgs) ([]byte, error) {
			saveCalls++
			return nil, nil
		},
	}

	globalSettings := &mock.GlobalSettingsHandlerStub{
		IsLimiterTransferCalled: func(token []byte) bool {
			return true
		},
		IsSenderOrDestinationWithTransferRoleCalled: func(sender, destination, token []byte) bool {
			return false
		},
	}

	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag || flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	}

	multiTransfer, err := NewESDTNFTMultiTransferFunc(
		1,
		&mock.MarshalizerMock{},
		globalSettings,
		&mock.AccountsStub{},
		mock.NewMultiShardsCoordinatorMock(4),
		vmcommon.BaseOperationCost{},
		enableEpochsHandler,
		&mock.ESDTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, token []byte, action []byte) error {
				return ErrActionNotAllowed
			},
		},
		storageHandler,
	)
	require.NoError(t, err)

	_, err = multiTransfer.transferOneTokenOnSenderShard(
		sender,
		destination,
		destinationAddress,
		&vmcommon.ESDTTransfer{
			ESDTTokenName:  tokenID,
			ESDTTokenNonce: nonce,
			ESDTValue:      big.NewInt(1),
		},
		false,
	)
	require.ErrorIs(t, err, ErrActionNotAllowed)
	require.Zero(t, saveCalls, "sender-side NFT state must not be persisted before limited-transfer authorization succeeds")
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnSameShardWithScCallWithEGLD(t *testing.T) {
	t.Parallel()

	vmInput, multiTransfer := createSetupForMultiTransferWithEGLD(t)
	multiTransfer.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return true
		},
	}
	multiTransfer.SetDRWAReader(newNoopDRWAReader())

	sender, err := multiTransfer.accounts.LoadAccount(vmInput.CallerAddr)
	require.Nil(t, err)
	destination, err := multiTransfer.accounts.LoadAccount(vmInput.RecipientAddr)
	require.Nil(t, err)

	vmOutput, err := multiTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransfer.accounts.SaveAccount(sender)
	_, _ = multiTransfer.accounts.Commit()

	// reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(vmInput.CallerAddr)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(vmInput.Arguments[0])
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransfer.marshaller, sender, []byte("token1"), 1, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransfer.marshaller, destination, []byte("token1"), 1, big.NewInt(4))

	require.Equal(t, sender.(vmcommon.UserAccountHandler).GetBalance(), big.NewInt(2))
	require.Equal(t, destination.(vmcommon.UserAccountHandler).GetBalance(), big.NewInt(1))

	funcName, args := extractScResultsFromVmOutput(t, vmOutput)

	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestESDTNFTMultiTransfer_TransferOneTokenOnSenderShardPropagatesMigratedTokenTypeToTransferPayload(t *testing.T) {
	t.Parallel()

	tokenID := []byte("NFT-123456")
	nonce := uint64(1)
	senderAddress := []byte("sender")
	destinationAddress := []byte("destination")
	sender := mock.NewUserAccount(senderAddress)

	storageHandler := &mock.ESDTNFTStorageHandlerStub{
		GetESDTNFTTokenOnSenderCalled: func(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, providedNonce uint64) (*esdt.ESDigitalToken, error) {
			require.Equal(t, append([]byte(baseESDTKeyPrefix), tokenID...), esdtTokenKey)
			require.Equal(t, nonce, providedNonce)

			return &esdt.ESDigitalToken{
				Type:  uint32(core.NonFungible),
				Value: big.NewInt(2),
			}, nil
		},
		SaveESDTNFTTokenCalled: func(senderAddr []byte, acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, providedNonce uint64, esdtData *esdt.ESDigitalToken, _ vmcommon.NftSaveArgs) ([]byte, error) {
			require.Equal(t, senderAddress, senderAddr)
			require.Equal(t, append([]byte(baseESDTKeyPrefix), tokenID...), esdtTokenKey)
			require.Equal(t, nonce, providedNonce)

			// Mirrors the real SaveESDTNFTToken() migration after updateTokenID:
			// the sender-side persisted record is upgraded to NonFungibleV2.
			esdtData.Type = uint32(core.NonFungibleV2)
			return nil, nil
		},
	}

	multiTransfer, err := NewESDTNFTMultiTransferFunc(
		1,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		mock.NewMultiShardsCoordinatorMock(4),
		vmcommon.BaseOperationCost{},
		&mock.EnableEpochsHandlerStub{},
		&mock.ESDTRoleHandlerStub{},
		storageHandler,
	)
	require.NoError(t, err)

	esdtData, err := multiTransfer.transferOneTokenOnSenderShard(
		sender,
		nil,
		destinationAddress,
		&vmcommon.ESDTTransfer{
			ESDTTokenName:  tokenID,
			ESDTTokenNonce: nonce,
			ESDTValue:      big.NewInt(1),
		},
		false,
	)
	require.NoError(t, err)
	require.Equal(t, uint32(core.NonFungibleV2), esdtData.Type)
	require.Equal(t, int64(1), esdtData.Value.Int64())
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsWithEGLD(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableChecker(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte(vmcommon.EGLDIdentifier)
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	_ = sender.(vmcommon.UserAccountHandler).AddToBalance(initialTokens)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1000000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	multiTransferSenderShard.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return true
		},
	}
	multiTransferSenderShard.SetDRWAReader(newNoopDRWAReader())
	multiTransferDestinationShard.SetDRWAReader(newNoopDRWAReader())

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshaller, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	require.Equal(t, sender.(vmcommon.UserAccountHandler).GetBalance(), big.NewInt(2))

	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshaller, destination, token1, tokenNonce, big.NewInt(1))
	require.Equal(t, destination.(vmcommon.UserAccountHandler).GetBalance(), big.NewInt(1))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}
