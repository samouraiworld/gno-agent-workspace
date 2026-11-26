# `crypto/secp256k1.RecoverPubkey`
Written by claude-opus-5-5.
PR: [gnolang/gno#6484](https://github.com/gnolang/gno/pull/6484)
Review files: [reviews/pr/6xxx/6484-secp256k1-recover-pubkey](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6484-secp256k1-recover-pubkey)

## TLDR

Before this change, a realm could check a secp256k1 signature only against a public key it already held. Ethereum-style signers are known by an address instead. This change adds `RecoverPubkey`, which rebuilds the signer's public key from a signature and the signed hash, as Ethereum's `ecrecover` does. A realm can then check a signature from an Ethereum address, a Wormhole guardian's for one.

## Key recovery

An ECDSA signature over a hash is a pair of numbers, r and s. From the hash and the pair, anyone can compute the signer's public key, which is a point on the curve, except that the computation yields two possible points. `RecoverPubkey` does that computation and returns the key as 65 bytes, `0x04 || X || Y`. It proves nothing on its own: a wrong hash or a wrong choice still yields a valid-looking key, only not the signer's, so the caller compares the result with the key or address it expects.

## The recovery ID

The 65th signature byte picks which of the two candidate points to return. Only 0 and 1 are accepted. Ethereum transmits it as `v` = 27 or 28, or under EIP-155 as 35 + 2 × chain ID + the ID, so a realm converts `v` back to 0 or 1 first. decred's `RecoverCompact`, which does the arithmetic, would also take 2 and 3, the rare case where the signing point's x-coordinate is r plus the curve's order n; the Go function behind `RecoverPubkey` refuses those first, as `ecrecover` does.

## The digest must be trusted

The 32-byte digest is not hashed again. Anyone who chooses the digest can build a signature that recovers any key they like, so a realm hashes the message it authorizes itself, Keccak-256 for Ethereum, and never takes the digest from its caller. The doc comment states this and a test pins one forged case.

## High-S signatures

For every signature (r, s) there is a twin (r, n - s) that recovers the same key with the other recovery ID. `RecoverPubkey` accepts both, as `ecrecover` does, so a realm must never use the signature bytes as a replay key.

## The wrapper and the native

The exported Gno function [`RecoverPubkey`](https://github.com/gnolang/gno/pull/6484/files#diff-595268614eb046e5a6b701a30ec27120aa89b4fe1c39c1cffe9ca53aa22fc1bbR29-R42) checks the two lengths, then copies the bytes into fresh `[32]byte` and `[65]byte` arrays and calls the private native `recoverPubkey`. The generated binding copies the arguments from Gno values into Go values before the native body runs, and no gas is charged for that copy; fixed-size arrays keep it the same size on every call. The Go side, [`X_recoverPubkey`](https://github.com/gnolang/gno/pull/6484/files#diff-fa7bd8d062cb1cd8bfb6f7f281c11ccbd020e87063b7fb82a5c6ae6d61d683a4R17-R36), refuses a recovery ID above 1, builds decred's compact form (header 27 + ID, then r and s) and calls `ecdsa.RecoverCompact` directly, so every build runs the same pure-Go code; tm2's own secp256k1 package, which `Verify` uses, switches to the C library libsecp256k1 under the `libsecp256k1` build tag. Any error comes back as a zero key and false; nothing panics on input.

## The gas row

Both inputs have a fixed size, so the cost does not grow with them, and a recovery has one flat row in `calibratedNativeGas`, priced on the slowest input measured, [Base 350,000](https://github.com/gnolang/gno/pull/6484/files#diff-4149844e757e8efee94a08e2cd7087614005be1ead9d28aa0f03a39c4713a1efR130-R130), charged before the native runs. It is derived rather than fitted: the slowest input measured took 1.279 times a 64-byte `verify` when the two were timed in alternating pairs. That input is a signature whose s value makes the curve multiplication do nearly as many point additions as it can. The row is that ratio times the `verify` row at 64 bytes (227,309) times 1.2 headroom, 348,874, rounded up. The `verify` row is itself marked preliminary, to be re-fit on the reference machine, after which this row is re-derived by hand.

| Input | Work in the native | Charge |
| --- | --- | --- |
| wrong length | none, refused by the wrapper | none |
| recovery ID 2 or more | refused in about 0.6 µs | 350,000 |
| valid signature | full recovery | 350,000 |

The table is read off the native's code, and `TestRecoverPubkeyFlatCharge` asserts the same charge for every fixed-size input.

## Determinism

The recovery is pure integer arithmetic in decred's library, with no package state. The change adds recovery cases to `misc/multiarch-determinism/corpus.gno`, so the multi-architecture run compares the outputs across platforms, and `stdlib_secp256k1_recover.txtar`, an integration script that starts a node and sends transactions to it, pins the exact gas of one call.

## What a user notices

A realm author can import `crypto/secp256k1` and call `RecoverPubkey(digest, sig)` to get a signer's key, then derive the Ethereum address from the last 20 bytes of Keccak-256 of `pubkey[1:]`. Each call costs at least 350,000 gas, whatever the input past the length check.

## Not fixed

- `v` values of 27/28 or EIP-155 are not converted; the realm converts them.
- The gas row tracks the `verify` row only by hand: nothing re-derives it when `verify` is re-fit.

## Words used here

| Name | What it is |
| --- | --- |
| `RecoverPubkey` | the exported function in `crypto/secp256k1`: 32-byte digest and 65-byte signature in, 65-byte uncompressed key and ok out |
| `recoverPubkey` / `X_recoverPubkey` | the private native and its Go implementation |
| `ecrecover` | Ethereum's precompile for the same operation |
| recovery ID | byte 65 of the signature, 0 or 1, picking one of two candidate keys |
| `calibratedNativeGas` | the table of native gas rows in `gnovm/stdlibs/native_gas.go` |
| flat row | a gas row with no size term, `SizeFlat` |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file.

### 1. `warm` · The wrapper, [`gnovm/stdlibs/crypto/secp256k1/recover.gno`](https://github.com/gnolang/gno/pull/6484/files#diff-595268614eb046e5a6b701a30ec27120aa89b4fe1c39c1cffe9ca53aa22fc1bbR29-R42)
Refuses wrong lengths and hands the native fixed-size arrays.
**If wrong:** a caller-sized slice reaches the native conversion.

### 2. `warm` · The native, [`gnovm/stdlibs/crypto/secp256k1/recover.go`](https://github.com/gnolang/gno/pull/6484/files#diff-fa7bd8d062cb1cd8bfb6f7f281c11ccbd020e87063b7fb82a5c6ae6d61d683a4R17-R36)
Refuses recovery IDs above 1, then recovers through decred.
**If wrong:** IDs 2 and 3 recover keys `ecrecover` would refuse.

### 3. `hot` · The gas row, [`gnovm/stdlibs/native_gas.go`](https://github.com/gnolang/gno/pull/6484/files#diff-4149844e757e8efee94a08e2cd7087614005be1ead9d28aa0f03a39c4713a1efR130-R130)
Prices every recovery at 350,000.
**If wrong:** validators run recoveries for less gas than their time.

### 4. `cold` · The Machine-level tests, `gnovm/stdlibs/secp256k1_recover_test.go`
Run the wrapper, the binding and the flat charge in a real Machine.
**If wrong:** a binding or gas regression goes unseen.

</details>
