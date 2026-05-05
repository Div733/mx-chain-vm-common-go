package builtInFunctions

import (
	"math/big"
	"testing"

	vm "github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shard topology tests for DRWA cross-shard enforcement.
//
// In MultiversX cross-shard transfers:
//   - Source shard: acntSnd is present, acntDst is nil
//   - Destination shard: acntSnd is nil, acntDst is present
//
// The DRWA gate must validate the local side and skip the remote side.
// These tests exercise both sender-side and receiver-side checks in isolation,
// simulating the split that occurs during cross-shard transfers.
// ---------------------------------------------------------------------------

const (
	shardTopologyTokenID = "SHARD-TOPO1"
)

// setupShardTopologyState creates a standard DRWA-regulated token with
// configurable holder states for sender and receiver.
func setupShardTopologyState(
	t *testing.T,
	state map[string]vmcommon.UserAccountHandler,
	senderHolder *drwaHolderMirrorView,
	receiverHolder *drwaHolderMirrorView,
) {
	t.Helper()

	systemAcc := state[string(vmcommon.SystemAccountAddress)]
	mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
		DRWAEnabled: true,
	})

	sender := state["sender"]
	mustSaveESDTBalance(t, sender, shardTopologyTokenID, 100)
	mustSaveDRWAHolder(t, sender, shardTopologyTokenID, "sender", senderHolder)

	receiver := state["receiver"]
	mustSaveDRWAHolder(t, receiver, shardTopologyTokenID, "receiver", receiverHolder)
}

func makeShardTopologyVMInput() *vmcommon.ContractCallInput {
	return &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("sender"),
			Arguments: [][]byte{
				[]byte(shardTopologyTokenID),
				big.NewInt(1).Bytes(),
			},
			CallValue:   big.NewInt(0),
			GasProvided: 100000,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}
}

// ---------------------------------------------------------------------------
// Test 1: Cross-shard transfer triggers compliance checks on BOTH sides
// (independently). Same-shard transfer validates both sender and receiver.
// ---------------------------------------------------------------------------

func TestShardTopology_SameShardTransfer_ValidatesBothSides(t *testing.T) {
	t.Parallel()

	accounts, state := createDRWATestAccounts()
	transferFunc, _ := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

	// Both sides compliant — transfer succeeds
	setupShardTopologyState(t, state, &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	}, &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	})

	output, err := transferFunc.ProcessBuiltinFunction(state["sender"], state["receiver"], makeShardTopologyVMInput())
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)
}

// ---------------------------------------------------------------------------
// Test 2: Sender shard checks (codes 1-5) and receiver shard checks (codes 6-11)
// are independently evaluated.
// ---------------------------------------------------------------------------

// Sender-side denial codes exercised with acntDst=nil (source shard perspective).
func TestShardTopology_SourceShard_SenderDenialCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		senderHolder   *drwaHolderMirrorView
		expectedErr    error
		description    string
		policyOverride *drwaTokenPolicyView
	}{
		{
			name:         "Code1_TokenPaused",
			senderHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved"},
			expectedErr:  errDRWATokenPaused,
			description:  "Global pause denies at sender side",
			policyOverride: &drwaTokenPolicyView{
				DRWAEnabled: true,
				GlobalPause: true,
			},
		},
		{
			name:         "Code2_KYCRequiredSender",
			senderHolder: &drwaHolderMirrorView{KYCStatus: "pending", AMLStatus: "approved"},
			expectedErr:  errDRWAKYCRequiredSender,
			description:  "Sender without KYC is denied",
		},
		{
			name:         "Code3_AMLBlockedSender",
			senderHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "flagged"},
			expectedErr:  errDRWAAMLBlockedSender,
			description:  "Sender with AML flag is denied",
		},
		{
			name:         "Code4_AssetExpired",
			senderHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved", ExpiryRound: 5},
			expectedErr:  errDRWAAssetExpired,
			description:  "Expired asset denied at sender side (round 0 with expiry set)",
		},
		{
			name:         "Code5_TransferLocked",
			senderHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved", TransferLocked: true},
			expectedErr:  errDRWATransferLocked,
			description:  "Transfer-locked sender is denied",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			accounts, state := createDRWATestAccounts()
			transferFunc, _ := NewESDTTransferFunc(
				10,
				&mock.MarshalizerMock{},
				&mock.GlobalSettingsHandlerStub{},
				&mock.ShardCoordinatorStub{},
				&mock.ESDTRoleHandlerStub{},
				drwaEnabledEpochsHandler(),
			)
			require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
			transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

			systemAcc := state[string(vmcommon.SystemAccountAddress)]
			policy := tc.policyOverride
			if policy == nil {
				policy = &drwaTokenPolicyView{DRWAEnabled: true}
			}
			mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, policy)

			sender := state["sender"]
			mustSaveESDTBalance(t, sender, shardTopologyTokenID, 100)
			mustSaveDRWAHolder(t, sender, shardTopologyTokenID, "sender", tc.senderHolder)

			// Cross-shard: acntDst is nil (receiver is on a different shard).
			// Source shard only validates sender; receiver validation happens
			// on the destination shard.
			vmInput := makeShardTopologyVMInput()
			_, err := transferFunc.ProcessBuiltinFunction(sender, nil, vmInput)
			require.ErrorIs(t, err, tc.expectedErr, tc.description)
		})
	}
}

// Receiver-side denial codes exercised with acntSnd=nil (destination shard perspective).
func TestShardTopology_DestinationShard_ReceiverDenialCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		receiverHolder  *drwaHolderMirrorView
		expectedErr     error
		description     string
		policyOverride  *drwaTokenPolicyView
	}{
		{
			name:           "Code1_TokenPaused",
			receiverHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved"},
			expectedErr:    errDRWATokenPaused,
			description:    "Global pause denies at receiver side too",
			policyOverride: &drwaTokenPolicyView{
				DRWAEnabled: true,
				GlobalPause: true,
			},
		},
		{
			name:           "Code6_KYCRequiredReceiver",
			receiverHolder: &drwaHolderMirrorView{KYCStatus: "pending", AMLStatus: "approved"},
			expectedErr:    errDRWAKYCRequiredReceiver,
			description:    "Receiver without KYC is denied",
		},
		{
			name:           "Code7_AMLBlockedReceiver",
			receiverHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "flagged"},
			expectedErr:    errDRWAAMLBlockedReceiver,
			description:    "Receiver with AML flag is denied",
		},
		{
			name:           "Code8_ReceiveLocked",
			receiverHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved", ReceiveLocked: true},
			expectedErr:    errDRWAReceiveLocked,
			description:    "Receive-locked receiver is denied",
		},
		{
			name:           "Code9_InvestorClassBlocked",
			receiverHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved", InvestorClass: "RETAIL"},
			expectedErr:    errDRWAInvestorClass,
			description:    "Receiver with wrong investor class is denied",
			policyOverride: &drwaTokenPolicyView{
				DRWAEnabled:            true,
				AllowedInvestorClasses: map[string]bool{"QIB": true},
			},
		},
		{
			name:           "Code10_JurisdictionBlocked",
			receiverHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved", JurisdictionCode: "CN"},
			expectedErr:    errDRWAJurisdiction,
			description:    "Receiver in blocked jurisdiction is denied",
			policyOverride: &drwaTokenPolicyView{
				DRWAEnabled:          true,
				AllowedJurisdictions: map[string]bool{"US": true},
			},
		},
		{
			name:           "Code11_AuditorRequired",
			receiverHolder: &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "approved", AuditorAuthorized: false},
			expectedErr:    errDRWAAuditorRequired,
			description:    "Receiver without auditor authorization is denied in strict mode",
			policyOverride: &drwaTokenPolicyView{
				DRWAEnabled:       true,
				StrictAuditorMode: true,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			accounts, state := createDRWATestAccounts()
			transferFunc, _ := NewESDTTransferFunc(
				10,
				&mock.MarshalizerMock{},
				&mock.GlobalSettingsHandlerStub{},
				&mock.ShardCoordinatorStub{},
				&mock.ESDTRoleHandlerStub{},
				drwaEnabledEpochsHandler(),
			)
			require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
			transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

			systemAcc := state[string(vmcommon.SystemAccountAddress)]
			policy := tc.policyOverride
			if policy == nil {
				policy = &drwaTokenPolicyView{DRWAEnabled: true}
			}
			mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, policy)

			receiver := state["receiver"]
			mustSaveDRWAHolder(t, receiver, shardTopologyTokenID, "receiver", tc.receiverHolder)

			// Cross-shard: acntSnd is nil (sender is on a different shard).
			// Destination shard only validates receiver; sender validation
			// already happened on the source shard.
			vmInput := makeShardTopologyVMInput()
			_, err := transferFunc.ProcessBuiltinFunction(nil, receiver, vmInput)
			require.ErrorIs(t, err, tc.expectedErr, tc.description)
		})
	}
}

// ---------------------------------------------------------------------------
// Test 3: Cross-shard transfer where sender passes but receiver fails
// is correctly denied.
// ---------------------------------------------------------------------------

func TestShardTopology_CrossShard_SenderPassesReceiverFails(t *testing.T) {
	t.Parallel()

	// Step 1: Simulate source shard — sender is compliant, acntDst nil.
	// This must succeed (source shard approves sender).
	t.Run("source_shard_approves_sender", func(t *testing.T) {
		t.Parallel()

		accounts, state := createDRWATestAccounts()
		transferFunc, _ := NewESDTTransferFunc(
			10,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.ShardCoordinatorStub{},
			&mock.ESDTRoleHandlerStub{},
			drwaEnabledEpochsHandler(),
		)
		require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
		transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

		systemAcc := state[string(vmcommon.SystemAccountAddress)]
		mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
			DRWAEnabled: true,
		})

		sender := state["sender"]
		mustSaveESDTBalance(t, sender, shardTopologyTokenID, 100)
		mustSaveDRWAHolder(t, sender, shardTopologyTokenID, "sender", &drwaHolderMirrorView{
			KYCStatus: "approved",
			AMLStatus: "approved",
		})

		vmInput := makeShardTopologyVMInput()
		output, err := transferFunc.ProcessBuiltinFunction(sender, nil, vmInput)
		require.NoError(t, err)
		require.NotNil(t, output)
	})

	// Step 2: Simulate destination shard — receiver is non-compliant, acntSnd nil.
	// This must fail (destination shard denies receiver).
	t.Run("destination_shard_denies_receiver", func(t *testing.T) {
		t.Parallel()

		accounts, state := createDRWATestAccounts()
		transferFunc, _ := NewESDTTransferFunc(
			10,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.ShardCoordinatorStub{},
			&mock.ESDTRoleHandlerStub{},
			drwaEnabledEpochsHandler(),
		)
		require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
		transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

		systemAcc := state[string(vmcommon.SystemAccountAddress)]
		mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
			DRWAEnabled: true,
		})

		receiver := state["receiver"]
		// Receiver has AML block — will be denied
		mustSaveDRWAHolder(t, receiver, shardTopologyTokenID, "receiver", &drwaHolderMirrorView{
			KYCStatus: "approved",
			AMLStatus: "flagged",
		})

		vmInput := makeShardTopologyVMInput()
		_, err := transferFunc.ProcessBuiltinFunction(nil, receiver, vmInput)
		require.ErrorIs(t, err, errDRWAAMLBlockedReceiver,
			"destination shard must independently deny receiver even though source shard approved sender")
	})
}

// ---------------------------------------------------------------------------
// Test 4: Sync envelope processing is shard-local (no cross-shard state reads).
// Validates that validateDRWASender/Receiver use only the provided account
// handler and do not require cross-shard data.
// ---------------------------------------------------------------------------

func TestShardTopology_SyncEnvelopeProcessing_ShardLocal(t *testing.T) {
	t.Parallel()

	// Create isolated accounts — each "shard" has its own AccountsStub.
	// Sender shard cannot see receiver's compliance state and vice versa.

	t.Run("sender_shard_validates_without_receiver_state", func(t *testing.T) {
		t.Parallel()

		// Sender shard has only sender's compliance data.
		// Receiver's compliance data is NOT in this shard's accounts.
		senderAccounts, senderState := createDRWATestAccounts()
		reader := mustCreateDRWAReader(t, senderAccounts)

		systemAcc := senderState[string(vmcommon.SystemAccountAddress)]
		mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
			DRWAEnabled: true,
		})

		sender := senderState["sender"]
		mustSaveDRWAHolder(t, sender, shardTopologyTokenID, "sender", &drwaHolderMirrorView{
			KYCStatus: "approved",
			AMLStatus: "approved",
		})
		// receiver's compliance state intentionally NOT saved in this shard

		// Sender validation must succeed using only shard-local state
		err := checkDRWASenderTransfer(reader, []byte(shardTopologyTokenID), []byte("sender"), sender, 100)
		require.NoError(t, err, "sender validation must succeed with shard-local state only")
	})

	t.Run("receiver_shard_validates_without_sender_state", func(t *testing.T) {
		t.Parallel()

		// Receiver shard has only receiver's compliance data.
		// Sender's compliance data is NOT in this shard's accounts.
		receiverAccounts, receiverState := createDRWATestAccounts()
		reader := mustCreateDRWAReader(t, receiverAccounts)

		systemAcc := receiverState[string(vmcommon.SystemAccountAddress)]
		mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
			DRWAEnabled: true,
		})

		receiver := receiverState["receiver"]
		mustSaveDRWAHolder(t, receiver, shardTopologyTokenID, "receiver", &drwaHolderMirrorView{
			KYCStatus: "approved",
			AMLStatus: "approved",
		})
		// sender's compliance state intentionally NOT saved in this shard

		// Receiver validation must succeed using only shard-local state
		err := checkDRWAReceiverTransfer(reader, []byte(shardTopologyTokenID), []byte("receiver"), receiver, 100)
		require.NoError(t, err, "receiver validation must succeed with shard-local state only")
	})

	t.Run("sender_validation_fails_independently_of_receiver", func(t *testing.T) {
		t.Parallel()

		senderAccounts, senderState := createDRWATestAccounts()
		reader := mustCreateDRWAReader(t, senderAccounts)

		systemAcc := senderState[string(vmcommon.SystemAccountAddress)]
		mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
			DRWAEnabled: true,
		})

		sender := senderState["sender"]
		mustSaveDRWAHolder(t, sender, shardTopologyTokenID, "sender", &drwaHolderMirrorView{
			KYCStatus: "pending", // non-compliant
			AMLStatus: "approved",
		})

		// Sender validation fails — no receiver state needed
		err := checkDRWASenderTransfer(reader, []byte(shardTopologyTokenID), []byte("sender"), sender, 100)
		require.ErrorIs(t, err, errDRWAKYCRequiredSender,
			"sender denial must be determined from shard-local state alone")
	})

	t.Run("receiver_validation_fails_independently_of_sender", func(t *testing.T) {
		t.Parallel()

		receiverAccounts, receiverState := createDRWATestAccounts()
		reader := mustCreateDRWAReader(t, receiverAccounts)

		systemAcc := receiverState[string(vmcommon.SystemAccountAddress)]
		mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
			DRWAEnabled: true,
		})

		receiver := receiverState["receiver"]
		mustSaveDRWAHolder(t, receiver, shardTopologyTokenID, "receiver", &drwaHolderMirrorView{
			KYCStatus: "approved",
			AMLStatus: "flagged", // non-compliant
		})

		// Receiver validation fails — no sender state needed
		err := checkDRWAReceiverTransfer(reader, []byte(shardTopologyTokenID), []byte("receiver"), receiver, 100)
		require.ErrorIs(t, err, errDRWAAMLBlockedReceiver,
			"receiver denial must be determined from shard-local state alone")
	})
}

// ---------------------------------------------------------------------------
// Test 5: Both-nil guard — if both acntSnd and acntDst are nil for a
// regulated token, the transfer must be rejected.
// ---------------------------------------------------------------------------

func TestShardTopology_BothAccountsNil_RegulatedTokenDenied(t *testing.T) {
	t.Parallel()

	accounts, state := createDRWATestAccounts()
	transferFunc, _ := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

	systemAcc := state[string(vmcommon.SystemAccountAddress)]
	mustSaveDRWATokenPolicy(t, systemAcc, shardTopologyTokenID, &drwaTokenPolicyView{
		DRWAEnabled: true,
	})

	// Both accounts nil — should never happen in valid protocol.
	// The gate must reject this as a safety invariant.
	vmInput := makeShardTopologyVMInput()
	_, err := transferFunc.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Error(t, err, "both-nil accounts for regulated token must be rejected")
	require.Contains(t, err.Error(), "DRWA enforcement",
		"error message must indicate DRWA enforcement rejection")
}

// ---------------------------------------------------------------------------
// Test 6: Unregulated token is not affected by shard topology checks.
// ---------------------------------------------------------------------------

func TestShardTopology_UnregulatedToken_PassesCrossShard(t *testing.T) {
	t.Parallel()

	accounts, state := createDRWATestAccounts()
	transferFunc, _ := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

	sender := state["sender"]
	mustSaveESDTBalance(t, sender, "UNREG-TOKEN", 100)

	// No DRWA policy set for this token — it is unregulated.
	// Cross-shard transfer (acntDst nil) must succeed without DRWA checks.
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("sender"),
			Arguments: [][]byte{
				[]byte("UNREG-TOKEN"),
				big.NewInt(1).Bytes(),
			},
			CallValue:   big.NewInt(0),
			GasProvided: 100000,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender, nil, vmInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)
}
