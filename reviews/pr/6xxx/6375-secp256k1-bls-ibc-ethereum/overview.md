# crypto/bls12381: `FastAggregateVerify` and `FastAggregateVerifyExcluding`
Written by claude-opus-5-5.
PR: [gnolang/gno#6375](https://github.com/gnolang/gno/pull/6375), BLS natives part only.

## TLDR

An Ethereum light client on gno.land has to check, for every update, one signature made jointly by up to 512 Ethereum validators. Doing that math in interpreted Gno would cost far more gas than a block holds, so this part of the change adds a new standard library, `crypto/bls12381`, that runs the check in Go. Realms get two functions, one taking every signer's key and one taking the group's combined key minus the few members who did not sign.

## BLS signatures and why they aggregate

A BLS public key is a point on the G1 group of the BLS12-381 curve (48 bytes compressed), and a signature is a point on G2 (96 bytes). The message is first hashed to a G2 point, `H(m)`, and a signature `sig` by secret key `sk` is valid when the pairing equation `e(pk, H(m)) == e(G1, sig)` holds. Points add: when every member of a committee signs the same message, the sum of their signatures verifies against the sum of their keys. One pairing check then covers the whole committee, however large it is, and the only per-signer work is decoding and adding keys.

`verifyAggregate` in [`bls12381.go`](https://github.com/gnolang/gno/pull/6375/files#diff-302d77199f3d9fa66beab16da6bf3bb2cce9b8b6f4a9b72be34aafc7bfe4a7a2R115-R132) writes the equation as one product, `e(pk, H(m)) * e(-G1, sig) == 1`, so gnark's `PairingCheck` runs two Miller loops and one final exponentiation. `negG1Gen` holds `-G1`, computed once at `init()`.

## Decoding a point safely

Every key and signature arrives as bytes, and gnark's `SetBytes` turns them into a point. It refuses three kinds of bad input: a coordinate that is not a canonical field element (x >= p), bytes that are not a point on the curve, and a point on the curve that lies outside the prime-order subgroup. That last check matters because points outside the subgroup break the security argument of the pairing. The natives rely on gnark doing all three; they add only the identity check below.

## The identity check

The identity point (infinity) is a valid point that behaves like zero. If the key handed to the pairing were the identity, `e(0, H(m))` would be 1 for every message, and the identity signature would satisfy the equation for anything. The natives refuse an identity key, and `verifyAggregate` also refuses an identity aggregate, because two valid keys `pk` and `-pk` sum to it. The signature is not refused for being the identity: with a non-identity key the equation would then read `e(pk, H(m)) == 1`, which does not hold.

## Proof of possession, the caller's job

Summing keys opens a "rogue key" attack: an attacker publishes a key chosen as their own key minus the honest members' keys, and the sum is then a key only they control. The IETF ciphersuite Ethereum uses (`BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_`) closes that by requiring a proof that each key's holder knows its secret. This package does not check such proofs. The package doc in [`bls12381.gno`](https://github.com/gnolang/gno/pull/6375/files#diff-3a317d926a5baee5f67f9f2cc9bd3a5c179e8de3a893a508efcd7084e9b7db87R1-R18) puts that duty on the caller. Ethereum sync committee keys qualify, since the beacon chain checked each validator's deposit signature before admitting the key.

## Subtracting the non-signers

An Ethereum sync committee commits to its aggregate key, the sum of all 512 member keys, and usually over 95% of members sign. `FastAggregateVerifyExcluding(aggregate, excluded, msg, sig)` takes that aggregate and the keys of the members who did not sign, and subtracts them, so a light client decodes a handful of keys instead of about 500. The catch is that the native cannot tell whether `excluded` really lists committee members: an attacker could pick `excluded` so that the aggregate minus it is the attacker's own key. The caller must prove each excluded key's membership, for example with a Merkle branch to the committee's pubkeys root.

## Hashing the message to the curve

`bls.HashToG2(msg, ethDST)` implements RFC 9380: `expand_message_xmd` stretches `msg` with SHA-256 under the domain separation tag `ethDST`, then two field elements are mapped to the curve and the cofactor is cleared. The tag ties signatures to Ethereum's ciphersuite, so a signature made for another protocol does not verify here. SHA-256 runs over every byte of `msg`, so hashing costs time in proportion to the message length.

## Native gas rows

A native call is charged from its row in [`native_gas.go`](https://github.com/gnolang/gno/pull/6375/files#diff-4149844e757e8efee94a08e2cd7087614005be1ead9d28aa0f03a39c4713a1ef): a fixed `Base`, plus `Slope` per 1024 units of the size of the argument at `SlopeIdx`, with `SizeLenBytes` meaning its length in bytes. A second slope (`Slope2`, `Slope2Idx`) can price another argument, as `crypto/merkle`'s `innerHash` does. The charge lands before the native runs. The two BLS rows charge about 2.02M plus 1962000 per KiB of keys, on `pubkeys` for `fastAggregateVerify` and on `excluded` for `fastAggregateVerifyExcluding`, whose base adds one key. The figures come from an Apple M4 Pro scaled by 2.3x toward the Xeon the neighbouring rows were measured on, and are marked draft until the table is regenerated on the reference machine.

## The wrapper bounds lengths first

Each exported Gno function checks lengths before calling its native: `pubkeys` a non-empty multiple of 48, `aggregate` exactly 48, `excluded` a multiple of 48, `sig` exactly 96. A malformed call then returns `false` without its bytes being copied into Go. `adr/native_input_bounds.md` asks every variable-length native argument to be either bounded this way or priced by a slope.

## Words used here

| Name | What it is |
| --- | --- |
| `FastAggregateVerify` | Gno function: true when `sig` is the aggregate signature of `msg` by every key in `pubkeys` (n x 48 bytes, n >= 1). |
| `FastAggregateVerifyExcluding` | Gno function: true when `sig` verifies against `aggregate` minus every key in `excluded` (zero or more 48-byte keys). |
| `verifyAggregate` | Go helper both natives end in: refuses an identity key, decodes `sig`, hashes `msg`, runs the pairing check. |
| `ethDST` | The domain separation tag `BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_`. |
| `negG1Gen` | The G1 generator negated, set once at package `init()`. |
| `PubKeySize`, `SignatureSize` | 48 and 96, the compressed sizes of a G1 key and a G2 signature. |
| `SizeLenBytes` | Gas slope kind: the argument's length in bytes. |
| `SlopeIdx` | Index of the argument a gas row's slope prices; -1 for none. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file.

### 1. `warm` · The Gno API and its length guards, [`gnovm/stdlibs/crypto/bls12381/bls12381.gno`](https://github.com/gnolang/gno/pull/6375/files#diff-3a317d926a5baee5f67f9f2cc9bd3a5c179e8de3a893a508efcd7084e9b7db87R27-R69)
Decides what a realm may pass and which duties stay with the caller.
**If wrong:** a realm aggregates unproven keys or unchecked excluded keys and accepts a forged update.

### 2. `hot` · Key decoding and aggregation, [`gnovm/stdlibs/crypto/bls12381/bls12381.go`](https://github.com/gnolang/gno/pull/6375/files#diff-302d77199f3d9fa66beab16da6bf3bb2cce9b8b6f4a9b72be34aafc7bfe4a7a2R42-R103)
Decodes every key with gnark's checks, refuses the identity, and sums or subtracts.
**If wrong:** an off-subgroup or identity key reaches the pairing.

### 3. `hot` · The pairing check, [`gnovm/stdlibs/crypto/bls12381/bls12381.go`](https://github.com/gnolang/gno/pull/6375/files#diff-302d77199f3d9fa66beab16da6bf3bb2cce9b8b6f4a9b72be34aafc7bfe4a7a2R115-R132)
Refuses an identity aggregate, hashes the message and checks the equation.
**If wrong:** the identity signature verifies any message.

### 4. `hot` · The gas rows, [`gnovm/stdlibs/native_gas.go`](https://github.com/gnolang/gno/pull/6375/files#diff-4149844e757e8efee94a08e2cd7087614005be1ead9d28aa0f03a39c4713a1ef)
Prices each call from its key bytes.
**If wrong:** validators do work the caller did not pay for.

### 5. `hot` · The decision record, [`gnovm/adr/pr6375_bls12381_native.md`](https://github.com/gnolang/gno/pull/6375/files#diff-c60149df089ac1151cb69d257fcba494e0f39a6b13112ae47e02a036e74bc1df)
States the two natives, the caller's duties and the gas fit.
**If wrong:** a later reader prices or uses the natives from a wrong record.

### 6. `cold` · The tests, [`gnovm/stdlibs/crypto/bls12381/bls12381_test.go`](https://github.com/gnolang/gno/pull/6375/files#diff-f690a634a9579f961a94cc356cc5748132e62ccca6052a66dca34077a174127a)
Signs with gnark and checks the refusals: wrong message, missing signer, identity key and aggregate, malformed input.
**If wrong:** a guard above is removed and nothing turns red.

</details>
