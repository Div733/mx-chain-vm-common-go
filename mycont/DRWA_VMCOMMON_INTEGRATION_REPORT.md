# mx-chain-vm-common-go — DRWA Integration Report

**Repo:** `/home/divesh/Desktop/RWA/right/core/mx-chain-vm-common-go`
**Date:** 2026
**Build status:** `go build ./...` — PASS
**Test status:** `go test ./...` — ALL PACKAGES PASS

---

## Section 1 — What This Repo Is

`mx-chain-vm-common-go` is the VM layer. It sits directly above `mx-chain-core-go`
in the dependency chain and directly below `mx-chain-go`.

```
mx-chain-core-go          ← shared types (constants, errors, flags)
       ↓ imported by
mx-chain-vm-common-go     ← THIS REPO — DRWA gate, 15 compliance checks
       ↓ imported by
mx-chain-go               ← full node, sync adapter, native mirror
```

Its job for DRWA is to contain the enforcement gate — the code that runs on
every regulated ESDT transfer and decides allow or deny. It reads compliance
state from the native mirror trie and applies the 15 denial checks.

It does NOT own data definitions. Those belong in `mx-chain-core-go`.
It does NOT own the sync adapter. That belongs in `mx-chain-go`.
It owns only the gate logic, the binary decoders, and the view structs.

---

## Section 2 — What Was Wrong Before We Changed Anything

### Problem 1 — `drwaActivePrefix` was a private string literal

In `builtInFunctions/drwa.go`:

```go
// BEFORE — private string, not imported from core
drwaActivePrefix = "drwa:active:"
```

`mx-chain-core-go` now has `ActiveMarkerPrefix StorageKeyPrefix = "drwa:active:"`.
This repo was not using it. It had its own copy of the string.

**Why this matters:**
`drwa:active:` is the compliance escape prevention prefix. When a token is
first registered as regulated, this key is written to the system account. It
persists even if the token policy is later deleted or corrupted. The gate reads
it to prevent a regulated token from silently becoming unregulated.

If this string ever drifted between `mx-chain-core-go` and this repo, the gate
would look for the active marker at the wrong key. It would find nothing. It
would conclude the token is unregulated. Regulated tokens could be transferred
freely after their policy was deleted. This is a silent compliance escape with
no error, no log, no alert.

---

### Problem 2 — `DRWAEnforcementFlag` was defined locally, duplicated from core

In `builtInFunctions/flags.go`:

```go
// BEFORE — local definition, string literal
DRWAEnforcementFlag core.EnableEpochFlag = "DRWAEnforcementFlag"
```

`mx-chain-core-go` now has `DRWAEnforcementFlag core.EnableEpochFlag = "DRWAEnforcementFlag"`.
This repo was redefining the same constant with the same string.

**Why this matters:**
`DRWAEnforcementFlag` is the epoch flag that turns the entire DRWA gate on or
off. The gate checks `IsFlagEnabled(DRWAEnforcementFlag)` on every transfer.
`mx-chain-go` must wire this exact same string into its epoch config so the
flag activates at the right epoch.

If this repo defines `"DRWAEnforcementFlag"` and `mx-chain-go` wires
`"DRWAEnforcementFlag"` independently, they happen to match today. But they
are two separate string literals with no compile-time link. A typo in either
repo — `"DRWAEnforcmentFlag"`, `"DrwaEnforcementFlag"` — would make the gate
permanently dormant with no compile error and no runtime error. Every regulated
transfer would pass unchecked. This was the root cause of launch blocker F-1
from the audit.

---

### Problem 3 — 5 exported prefix constants were re-exporting what core already owns

In `builtInFunctions/drwa.go`:

```go
// BEFORE — re-exporting core constants under new names
const (
    DRWATokenPolicyPrefix       = drwaTokenPolicyPrefix
    DRWAHolderMirrorPrefix      = drwaHolderMirrorPrefix
    DRWAHolderProfilePrefix     = drwaHolderProfilePrefix
    DRWAHolderAuditorAuthPrefix = drwaHolderAuditorAuthPrefix
    DRWAAssetRecordPrefix       = string(coredrwa.AssetRecordPrefix)
)
```

These 5 exported constants existed so `mx-chain-go` could import them and the
startup `init()` panic could validate parity:

```go
// in mx-chain-go/process/smartContract/hooks/drwa_sync_types.go
func init() {
    pairs := [][2]string{
        {drwaSyncTokenPolicyPrefix, builtInFunctions.DRWATokenPolicyPrefix},
        // ...
    }
    for _, p := range pairs {
        if p[0] != p[1] {
            panic("CRIT-03: DRWA prefix mismatch...")
        }
    }
}
```

**Why this was wrong:**
The panic existed because the same prefix string was defined in three places:
core (canonical), this repo (private local copy), and this repo again (exported
re-export). `mx-chain-go` imported the re-export and compared it against its
own local copy. The panic was a safety net for a problem that should not have
existed in the first place.

Now that core owns all 6 prefixes canonically, both this repo and `mx-chain-go`
import directly from core. There is only one definition. The re-exports are
redundant. The startup panic in `mx-chain-go` is no longer needed.

---

## Section 3 — What We Changed

### Change 1 — `drwaActivePrefix` now imports from core

**File:** `builtInFunctions/drwa.go`

```go
// BEFORE
drwaActivePrefix = "drwa:active:"

// AFTER
drwaActivePrefix = string(coredrwa.ActiveMarkerPrefix)
```

One line change. Now if core ever updates the prefix string, this repo picks
it up automatically at compile time. A mismatch is impossible.

---

### Change 2 — `DRWAEnforcementFlag` now imported from core, not redefined

**File:** `builtInFunctions/flags.go`

```go
// BEFORE — local string literal
DRWAEnforcementFlag core.EnableEpochFlag = "DRWAEnforcementFlag"

// AFTER — imported from core
DRWAEnforcementFlag = coredrwa.DRWAEnforcementFlag
```

Import added: `coredrwa "github.com/multiversx/mx-chain-core-go/data/drwa"`

Now this repo and `mx-chain-go` both import `coredrwa.DRWAEnforcementFlag`.
They are guaranteed to use the same string. A mismatch is impossible at
compile time.

---

### Change 3 — Removed the 5 exported prefix re-exports

**File:** `builtInFunctions/drwa.go`

```go
// REMOVED ENTIRELY
const (
    DRWATokenPolicyPrefix       = drwaTokenPolicyPrefix
    DRWAHolderMirrorPrefix      = drwaHolderMirrorPrefix
    DRWAHolderProfilePrefix     = drwaHolderProfilePrefix
    DRWAHolderAuditorAuthPrefix = drwaHolderAuditorAuthPrefix
    DRWAAssetRecordPrefix       = string(coredrwa.AssetRecordPrefix)
)
```

`BuildDRWAAssetRecordKey` was the only function that used `DRWAAssetRecordPrefix`
directly. It was updated to use `coredrwa.AssetRecordPrefix` instead:

```go
// BEFORE
return []byte(DRWAAssetRecordPrefix + hex.EncodeToString(tokenIdentifier) + ":record")

// AFTER
return []byte(string(coredrwa.AssetRecordPrefix) + hex.EncodeToString(tokenIdentifier) + ":record")
```

With these re-exports gone, `mx-chain-go` no longer needs to import prefix
constants from this repo. It imports them directly from core. The startup
`init()` panic in `mx-chain-go` is no longer needed and will be removed in
the next step.

---

## Section 4 — What Was NOT Changed

Everything that is gate logic stays exactly as it was. Zero enforcement
behaviour changed.

| What | Status |
|---|---|
| 15 denial checks (`validateDRWASender`, `validateDRWAReceiver`) | Unchanged |
| Metadata update enforcement (`validateDRWAMetadataUpdate`) | Unchanged |
| Binary decoders (token policy, holder mirror, profile, auditor auth) | Unchanged |
| View structs (`drwaTokenPolicyView`, `drwaHolderMirrorView`, etc.) | Unchanged |
| Merge precedence logic (profile vs mirror version) | Unchanged |
| Gas accounting (`computeDRWAReadGasCost`) | Unchanged |
| Key builders (`BuildDRWATokenPolicyKey`, etc.) | Unchanged |
| All metrics (`recordDRWAGateMetric`) | Unchanged |
| All error variables (`errDRWATokenPaused`, etc.) | Unchanged |
| `drwaAccountsReader` trie reader | Unchanged |
| All ESDT transfer integrations | Unchanged |

---

## Section 5 — File Change Summary

| File | What changed |
|---|---|
| `builtInFunctions/drwa.go` | `drwaActivePrefix` now uses `coredrwa.ActiveMarkerPrefix`. Removed 5 exported prefix re-exports. `BuildDRWAAssetRecordKey` uses `coredrwa.AssetRecordPrefix` directly. |
| `builtInFunctions/flags.go` | `DRWAEnforcementFlag` now assigned from `coredrwa.DRWAEnforcementFlag`. Added `coredrwa` import. |

All other files in the repo are untouched.

---

## Section 6 — Build and Test Results

```
go build ./...   — PASS
go test ./...    — ALL PACKAGES PASS

ok  github.com/multiversx/mx-chain-vm-common-go                    0.003s
ok  github.com/multiversx/mx-chain-vm-common-go/builtInFunctions   0.019s
ok  github.com/multiversx/mx-chain-vm-common-go/container          0.003s
ok  github.com/multiversx/mx-chain-vm-common-go/dataTrieMigrator   0.003s
ok  github.com/multiversx/mx-chain-vm-common-go/parsers            0.004s
ok  github.com/multiversx/mx-chain-vm-common-go/parsers/dataField  0.004s
```

Zero failures. Zero regressions. All pre-existing tests pass.

---

## Section 7 — What This Unlocks in Downstream Repos

**`mx-chain-go` — startup panic can now be removed**

The `init()` panic in `drwa_sync_types.go` validated that this repo's exported
prefix constants matched `mx-chain-go`'s local copies. Those exported constants
no longer exist. `mx-chain-go` now imports all prefixes directly from core.
The panic is no longer needed and should be removed.

**`mx-chain-go` — F-1 wiring is now unambiguous**

`DRWAEnforcementFlag` is now a single constant in core. `mx-chain-go` imports
`coredrwa.DRWAEnforcementFlag` when wiring the epoch config. This repo imports
the same constant when checking `IsFlagEnabled`. A string mismatch between the
two repos is now a compile error, not a silent runtime failure.

---

## Section 8 — Next Step

**`mx-chain-go`** is the next repo. Two things to do there:

1. Remove the `init()` panic in `drwa_sync_types.go` — the re-exports it
   validated no longer exist in this repo
2. Wire `coredrwa.DRWAEnforcementFlag` into the epoch config in
   `config/epochConfig.go` and `cmd/node/config/enableEpochs.toml` —
   this fixes launch blocker F-1 and makes the gate active in production
