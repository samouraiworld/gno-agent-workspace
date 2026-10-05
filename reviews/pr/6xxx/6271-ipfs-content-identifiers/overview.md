# `gno.land/p/omarsy/cid/v0`: IPFS content identifiers
Written by claude-opus-5-5.
PR: [gnolang/gno#6271](https://github.com/gnolang/gno/pull/6271) · [Review files](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6271-ipfs-content-identifiers)

## TLDR

No package in `examples/` parsed or checked IPFS CIDs. The new `gno.land/p/omarsy/cid/v0` parses the string and binary forms of CIDv0 and CIDv1 (`Parse`, `Decode`, `DecodeFirst`), builds them (`NewV1`, `Sum`), prints them (`String`, `Encode`) and checks a block against one (`Verify`). A realm can then keep a CID in its state and check, on chain, that bytes someone submits are the content it names. Parsing is strict, so each CID has one accepted binary form, and every input is length-capped before it is decoded, so a rejected string costs bounded gas.

## What the package is

A CID names content by its hash: a version, a codec saying what the content is, and a multihash saying which hash function produced which digest. `ipfs add` prints one per file. The package holds a CID as its binary form in an unexported string, so a `CID` compares with `==` and works as a map key through `KeyString`.

## How it works, in 4 steps

The package as added, on the CID of the bytes `hello`, the example its own filetest runs:

1. `Parse("bafkreibm6jg3ux5qumhcn2b3flc3tyu6dmlb4xa7u5bf44yegnrjhc4yeq")`: the leading `b` picks base32, and `decodeMultibase` checks the 58-character body against `maxEncodedLen('b')`, 239, before decoding it to 36 bytes, `01 55 12 20` and a 32-byte digest.
2. `Decode` hands them to `DecodeFirst`, which reads the version varint (1), the codec varint (`0x55`, `Raw`), then `multihashLen` reads the hash code (`0x12`, `SHA256`) and the digest length (32), and requires all 36 bytes to be used.
3. The result is `CID{b}` over those 36 bytes: `Version()` is 1, `Codec()` is `Raw`, and `String()` gives back the same base32 text.
4. `c.Verify([]byte("hello"))` hashes the block with sha2-256 and compares the digests: `nil`. `c.Verify([]byte("hellp"))` returns `ErrDigestMismatch`.

## The parts, at a glance

| File | Job |
| --- | --- |
| [`cid.gno`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/cid.gno) | the `CID` type: `Parse`, `Decode`, `DecodeFirst`, `NewV1`, `Sum`, `V1`, `String`, `Encode`, `Verify` |
| [`multihash.gno`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/multihash.gno) | the `Multihash` type, digest-length rules, and the hashing `Verify` delegates to |
| [`multibase.gno`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/multibase.gno) | strict base32, base58btc, base36 and base16 codecs, and the per-base length caps |
| [`varint.gno`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/varint.gno) | `uvarint`, which rejects encodings over 9 bytes and non-minimal ones |
| [`pr6271_p_omarsy_cid_v0.md`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/gno.land/adr/pr6271_p_omarsy_cid_v0.md) | the ADR: scope, strictness rules and the gas table |

## Read the code in this order

1. [`Parse`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/cid.gno#L31-L48) decides between the two string forms: a 46-character `Qm` string is a CIDv0 in bare base58btc, anything else is multibase. A CIDv0 written in multibase is rejected, so each CID has one string form per base.
   ```go
   if len(s) == 46 && s[:2] == "Qm" {
   	b, err := decodeBaseN(s, 58, base58Digit)
   ```
2. [`DecodeFirst`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/cid.gno#L65-L91) reads one binary CID off the front of a buffer, for CAR files and dag-pb links. It truncates the buffer to `maxCIDLen`, 149 bytes, before copying it, and tells a CIDv0 from a CIDv1 through `isV0`, a 34-byte multihash starting `0x12 0x20`.
3. [`decodeMultibase`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/multibase.gno#L40-L61) caps the body by base before any decoding: base58btc's quadratic decoder is the costly one, so its cap of 204 characters is what bounds the gas of a rejected string.
4. [`Multihash.verify`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/multihash.gno#L97-L121) hashes with sha2-256 or keccak-256, or compares the bytes for identity, and returns `ErrUnsupportedHash` for any other code. It ignores the codec, so a caller wanting the file's own bytes checks `Codec() == Raw`.

## What a user notices

Nothing until a realm imports the package. A realm that does gets strict input: mixed case, padding, whitespace, a non-minimal varint or a trailing byte each make `Parse` return an error rather than a second spelling of the same CID.

## Words used here

| Name | What it is |
| --- | --- |
| CIDv0 | a bare sha2-256 multihash, 34 bytes, written in base58btc as 46 characters starting `Qm`; it always names dag-pb content |
| CIDv1 | version byte 1, a codec varint and a multihash, written with a one-character multibase prefix |
| multibase | the prefix naming the base of a CIDv1 string: `b`/`B` base32, `z` base58btc, `k` base36, `f`/`F` base16 |
| multihash | a hash code varint, a digest-length varint and the digest; `NewMultihash` builds one |
| `maxCIDLen` | 149, the longest binary CID accepted: version byte, two 9-byte varints, a 2-byte length and a 128-byte digest |
| `maxEncodedLen` | the longest body each multibase accepts for a 149-byte CID: 239 base32, 204 base58btc, 231 base36, 298 base16 |
| `Verify` | `c.Verify(block)`: `nil` when `block` hashes to the digest `c` carries |
