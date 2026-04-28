package builtInFunctions

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestC01_LocalMintRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTLocalMintFunc(50, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, drwaEnabledEpochsHandler())
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("caller"),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), big.NewInt(1).Bytes()},
			GasProvided: 5000,
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_LocalBurnRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTLocalBurnFunc(50, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, drwaEnabledEpochsHandler())
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("caller"),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), big.NewInt(1).Bytes()},
			GasProvided: 5000,
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_ESDTBurnRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTBurnFunc(50, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, drwaEnabledEpochsHandler(GlobalMintBurnFlag))
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("caller"),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), big.NewInt(1).Bytes()},
			GasProvided: 5000,
		},
		RecipientAddr: core.ESDTSCAddress,
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_NFTAddQuantityRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTNFTAddQuantityFunc(50, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, drwaEnabledEpochsHandler())
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("caller"),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), big.NewInt(1).Bytes(), big.NewInt(1).Bytes()},
			GasProvided: 5000,
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_NFTCreateRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTNFTCreateFunc(50, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, createNewESDTDataStorageHandler(), &mock.AccountsStub{}, drwaEnabledEpochsHandler())
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("caller"),
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte("CARBON-1"),
				big.NewInt(1).Bytes(),
				[]byte("name"),
				big.NewInt(1).Bytes(),
				[]byte("hash"),
				[]byte("attrs"),
				[]byte("uri"),
			},
			GasProvided: 5000,
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_NFTBurnRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTNFTBurnFunc(50, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, drwaEnabledEpochsHandler())
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("caller"),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), big.NewInt(1).Bytes(), big.NewInt(1).Bytes()},
			GasProvided: 5000,
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_DeleteMetadataRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTDeleteMetadataFunc(ArgsNewESDTDeleteMetadata{
		FuncGasCost:         50,
		Marshalizer:         &mock.MarshalizerMock{},
		Accounts:            &mock.AccountsStub{},
		AllowedAddress:      []byte("caller"),
		Delete:              true,
		EnableEpochsHandler: drwaEnabledEpochsHandler(),
	})
	_, err := f.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("caller"),
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte("CARBON-1"),
				big.NewInt(1).Bytes(),
				big.NewInt(1).Bytes(),
				big.NewInt(1).Bytes(),
			},
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_NFTCreateRoleTransferRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, &mock.AccountsStub{}, mock.NewMultiShardsCoordinatorMock(2), drwaEnabledEpochsHandler())
	_, err := f.ProcessBuiltinFunction(nil, mock.NewUserAccount([]byte("current-owner-32-bytes-addr----")), &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  core.ESDTSCAddress,
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), bytes.Repeat([]byte{2}, 32)},
			GasProvided: 5000,
		},
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_FreezeIsExplicitSystemSCCarveOutWithoutDRWAReader(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	f, _ := NewESDTFreezeWipeFunc(createNewESDTDataStorageHandler(), drwaEnabledEpochsHandler(), marshaller, true, false)

	acnt := mock.NewUserAccount([]byte("dst"))
	tokenKey := []byte("CARBON-1")
	esdtKey := append([]byte(baseESDTKeyPrefix), tokenKey...)
	token := &esdt.ESDigitalToken{Value: big.NewInt(10)}
	tokenBytes, _ := marshaller.Marshal(token)
	_ = acnt.AccountDataHandler().SaveKeyValue(esdtKey, tokenBytes)

	out, err := f.ProcessBuiltinFunction(nil, acnt, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: core.ESDTSCAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{tokenKey},
		},
		RecipientAddr: []byte("dst"),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
}

func TestC01_WipeIsExplicitSystemSCCarveOutWithoutDRWAReader(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	f, _ := NewESDTFreezeWipeFunc(createNewESDTDataStorageHandler(), drwaEnabledEpochsHandler(), marshaller, false, true)

	acnt := mock.NewUserAccount([]byte("dst"))
	tokenKey := []byte("CARBON-2")
	esdtKey := append([]byte(baseESDTKeyPrefix), tokenKey...)
	meta := ESDTUserMetadata{Frozen: true}
	token := &esdt.ESDigitalToken{
		Value:      big.NewInt(7),
		Properties: meta.ToBytes(),
	}
	tokenBytes, _ := marshaller.Marshal(token)
	_ = acnt.AccountDataHandler().SaveKeyValue(esdtKey, tokenBytes)

	out, err := f.ProcessBuiltinFunction(nil, acnt, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: core.ESDTSCAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{tokenKey},
		},
		RecipientAddr: []byte("dst"),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
}

func TestC01_ModifyCreatorRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTModifyCreatorFunc(50, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, drwaEnabledEpochsHandler(DynamicEsdtFlag), &mock.MarshalizerMock{})
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("caller"),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), big.NewInt(1).Bytes()},
			GasProvided: 5000,
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}

func TestC01_ModifyRoyaltiesRequiresDRWAReaderWhenEnforcementEnabled(t *testing.T) {
	t.Parallel()

	f, _ := NewESDTModifyRoyaltiesFunc(50, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, drwaEnabledEpochsHandler(DynamicEsdtFlag), &mock.MarshalizerMock{})
	_, err := f.ProcessBuiltinFunction(mock.NewUserAccount([]byte("caller")), nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("caller"),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("CARBON-1"), big.NewInt(1).Bytes(), big.NewInt(1).Bytes()},
			GasProvided: 5000,
		},
		RecipientAddr: []byte("caller"),
	})
	require.ErrorIs(t, err, errDRWAStateReaderMissing)
}
