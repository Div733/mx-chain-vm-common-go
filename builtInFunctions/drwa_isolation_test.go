package builtInFunctions

// drwa_isolation_test.go — isolation regression tests for vanilla ESDT paths.
//
// These tests explicitly prove that non-regulated token transfers remain
// unchanged when DRWA enforcement is enabled. They complement
// drwa_regression_test.go by focusing on the "DRWA enabled but token not
// regulated" scenario.

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vm "github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

// TestIsolation_ESDTTransfer_NonRegulatedToken ensures a plain ESDT transfer
// succeeds when DRWA is enabled but the token has no policy stored.
func TestIsolation_ESDTTransfer_NonRegulatedToken(t *testing.T) {
	t.Parallel()

	accounts, _ := createDRWATestAccounts()
	transferFunc, err := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, err)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

	senderAddress := bytes.Repeat([]byte{2}, 32)
	receiverAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := accounts.LoadAccount(senderAddress)
	require.NoError(t, err)
	receiver, err := accounts.LoadAccount(receiverAddress)
	require.NoError(t, err)

	mustSaveESDTBalance(t, sender.(vmcommon.UserAccountHandler), "PLAIN-ESDT", 10)
	_ = accounts.SaveAccount(sender)
	_ = accounts.SaveAccount(receiver)
	_, _ = accounts.Commit()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{[]byte("PLAIN-ESDT"), big.NewInt(2).Bytes()},
			CallValue:   big.NewInt(0),
			GasProvided: 10000,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: receiverAddress,
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), receiver.(vmcommon.UserAccountHandler), vmInput)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)
}

// TestIsolation_NFTTransfer_NonRegulatedToken ensures a plain NFT transfer
// succeeds when DRWA is enabled but the token has no policy stored.
func TestIsolation_NFTTransfer_NonRegulatedToken(t *testing.T) {
	t.Parallel()

	nftTransfer, _ := createNFTTransferAndStorageHandler(
		0,
		1,
		&mock.GlobalSettingsHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, nftTransfer.SetPayableChecker(&mock.PayableHandlerStub{}))
	nftTransfer.SetDRWAReader(mustCreateDRWAReader(t, nftTransfer.accounts))

	senderAddress := bytes.Repeat([]byte{2}, 32)
	receiverAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := nftTransfer.accounts.LoadAccount(senderAddress)
	require.NoError(t, err)
	receiver, err := nftTransfer.accounts.LoadAccount(receiverAddress)
	require.NoError(t, err)

	mustSaveESDTBalance(t, sender.(vmcommon.UserAccountHandler), "PLAIN-NFT", 1)
	createESDTNFTToken([]byte("PLAIN-NFT"), core.NonFungible, 1, big.NewInt(1), &mock.MarshalizerMock{}, sender.(vmcommon.UserAccountHandler))
	_ = nftTransfer.accounts.SaveAccount(sender)
	_ = nftTransfer.accounts.SaveAccount(receiver)
	_, _ = nftTransfer.accounts.Commit()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{[]byte("PLAIN-NFT"), big.NewInt(1).Bytes(), big.NewInt(1).Bytes(), receiverAddress},
			CallValue:   big.NewInt(0),
			GasProvided: 10000,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: senderAddress,
	}

	output, err := nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), receiver.(vmcommon.UserAccountHandler), vmInput)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)
}
