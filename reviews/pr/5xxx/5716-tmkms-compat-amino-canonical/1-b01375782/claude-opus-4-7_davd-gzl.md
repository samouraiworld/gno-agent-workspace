# PR #5716: feat(tm2): tmkms-compat (1/3) — amino varint, canonical retag, vendored protos, golden vectors

URL: https://github.com/gnolang/gno/pull/5716
Author: @clockworkgr | Base: master | Files: 35 | +5608 -30
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5716 b01375782` (then `gh -R gnolang/gno pr checkout 5716` inside it)

**Verdict: APPROVE** — clean split of the upstream-Tendermint v0.34 wire-format foundation from #5625; additive (new amino tag, new package, new tests) with one consensus-sensitive retag (`CanonicalProposal.POLRound` fixed64→varint, `CanonicalPartSetHeader` Total int→uint32) that is correct, well-tested, and explicitly framed as pre-mainnet. PR is already merged (jaekwon).

## Summary

This is the first slice (of three) bringing unmodified upstream **tmkms** compatibility to tm2. Nothing in this PR is yet wired into the validator — PR2 adds the secret-conn signer client, PR3 the listener integration. What lands here:

1. **New opt-in amino struct tag `binary:"varint"`** emitting plain protobuf varint (wire-type 0, schema int64/int32) instead of the default zigzag. Threaded through reflect encoder/decoder, genproto, genproto2 codegen (marshal/size/unmarshal), and NList nested-list propagation/disambiguation.
2. **CanonicalProposal / CanonicalVote / CanonicalPartSetHeader retagged** to be byte-identical to upstream Tendermint v0.34's `canonical.proto`. `POLRound` switches from `fixed64` to `varint` (now 10 bytes for −1, up from 8); `CanonicalPartSetHeader.{Total,Hash}` order swaps and `Total` becomes `uint32` (previously platform-int). `CanonicalizePartSetHeader` rejects out-of-range Total at the boundary.
3. **New `tm2/pkg/bft/privval/upstream/` package** holding upstream-shaped `Vote`/`Proposal`/`BlockID`/`PartSetHeader` with byte-identical amino emission, tm2↔upstream translators, and a `WrapMsg`/`UnwrapMsg` envelope over protoc-generated `upstreampb` types.
4. **Three-layer wire-compat test suite plus hex-frozen golden vectors** — protowire schema walks, hand-rolled spec encoders, stdlib protobuf round-trip, and frozen-byte goldens (some captured from upstream protoc, some from tm2 amino).

## Glossary

- `binary:"varint"` — new amino struct-tag emitting plain protobuf varint (proto int64/int32) instead of zigzag (sint64/sint32). Negative values consume 10 bytes.
- `CanonicalProposal` / `CanonicalVote` — the sign-byte projection of `Proposal` / `Vote`; the bytes that `SignBytes()` feeds to the signer. Wire format is what tmkms canonicalizes locally to verify the signature.
- `upstreampb` — protoc-generated Go types from a minimal subset of upstream Tendermint v0.34's `types.proto`/`canonical.proto`/`privval/types.proto`, vendored into `tm2/pkg/bft/privval/upstream/upstreampb/`.
- `upstream` package — tm2-amino-encoded mirrors of upstreampb's operational types. Their wire bytes are asserted byte-identical to upstreampb's `proto.Marshal` output.
- `SignedMsgType` — single-byte enum (1=Prevote, 2=Precommit, 0x20=Proposal). Encoded by amino as varint, byte-equivalent to upstream's `uint32 type` field.

## Fix

Before: tm2 amino had two binary tags (`fixed32`, `fixed64`). Signed-int fields without a tag encoded as zigzag varint (proto sint64), so `CanonicalProposal.POLRound = -1` was 1 byte and `CanonicalPartSetHeader.Total = int(-1)` was unrepresentable on the wire matching upstream. tmkms canonicalizes its received Vote/Proposal locally per upstream's `canonical.proto` and signs that — so any byte-shift breaks signature verification.

After: A third opt-in tag, `binary:"varint"`, emits plain varint (proto int64/int32). Threaded through:
- Reflect path: [`tm2/pkg/amino/binary_encode.go:118`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/binary_encode.go#L118) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/binary_encode.go#L118), [`binary_decode.go:133`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/binary_decode.go#L133) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/binary_decode.go#L133), [`encoder.go:49`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/encoder.go#L49) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/encoder.go#L49), [`decoder.go:57`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/decoder.go#L57) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/decoder.go#L57).
- Codegen path: [`genproto2/gen_marshal.go`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/genproto2/gen_marshal.go) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/genproto2/gen_marshal.go), `gen_size.go`, `gen_unmarshal.go` — all gain a `BinPlainVarint` branch.
- Schema path: [`genproto/genproto.go:341`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/genproto/genproto.go#L341) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/genproto/genproto.go#L341) (Int/Int32/Int64 → `int64`/`int32`), [`genproto/types.go:284`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/genproto/types.go#L284) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/genproto/types.go#L284) nested-list propagation + `NList.Name()` disambiguation prefix `Varint`.
- Registration validation: [`codec.go:168`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/codec.go#L168) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/codec.go#L168) — `set>1` mutual exclusion across all three binary tags, and per-tag kind whitelist (only Int / Int32 / Int64 for varint; rejects Int8 / Int16 / string / bool). Parser is now comma-split, so `binary:"varint,fixed64"` is detectable rather than silently no-op.

Canonical retag at [`canonical.go:26-39`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/types/canonical.go#L26-L39) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/types/canonical.go#L26-L39); the upstream package types at [`upstream/types.go:41-101`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/upstream/types.go#L41-L101) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/upstream/types.go#L41-L101).

The chain's own `Vote` / `Proposal` / p2p-message types in `tm2/pkg/bft/types` are untouched — only the canonical sign-byte projection changes (and only the upstream-shaped mirrors in the new package). PR description confirms: "Nothing in the rest of tm2 is rewired by this PR — the new wire format is dormant until PR2 ships the transport and PR3 wires it into `NewPrivValidatorFromConfig`."

## Benchmarks / Numbers

| Field | Before | After | Why |
|---|---|---|---|
| `CanonicalProposal.POLRound = −1` | 8 bytes (sfixed64) | 10 bytes (plain varint) | Upstream wire format requires int64 plain varint; −1 sign-extends to 10 bytes. |
| `CanonicalPartSetHeader.Total` field 1 type | int → sint64 zigzag | uint32 plain varint | Upstream schema is `uint32 total = 1`. |
| `MaxVoteBytes` | 247 | 242 | 5-byte saving (Total cap MaxUint32 now caps zigzag-int to 5 bytes from 10). |
| `TestEvidenceByteSize` expected | 548 | 538 | 5 bytes × 2 BlockIDs per evidence. |
| `TestProposalSignBytesTestVectors` expected length | 42 | 44 | POLRound = −1 grew from 8 → 10 bytes; field-key 0x20 unchanged. |

## Critical (must fix)

None for this PR as scoped. The `CanonicalPartSetHeader` field-order + `POLRound` encoding change does break sign-byte parity with prior tm2 — pre-PR signed bytes are not verifiable post-PR — but this is intentional and pre-mainnet (called out in commit `26c7e4e406` and the PR description). Same flag I raised on #5625; the answer is unchanged: pre-mainnet timing is the migration plan. If a node carries a pre-PR `priv_validator_state.json` across the upgrade, [`tm2/pkg/bft/privval/state/state.go:115`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/state/state.go#L115) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/state/state.go#L115)'s `amino.UnmarshalSized(fs.SignBytes, &lastVote)` will either fail or decode garbage — operator implication is "wipe state on upgrade", which should be in the PR3 operator doc rather than this PR's scope.

## Warnings (should fix)

- **[panic via amino decode → DoS]** [`tm2/pkg/bft/privval/upstream/translator.go:36-49`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/upstream/translator.go#L36-L49) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/upstream/translator.go#L36-L49) — `ToTM2Vote` / `ToTM2Proposal` panic on malformed wire input via `addressFromBytes`.
  <details><summary>details</summary>

  `addressFromBytes` ([translator.go:131-138](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/upstream/translator.go#L131-L138) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/upstream/translator.go#L131-L138)) panics when the wire address length is anything other than `crypto.AddressSize` (20). The doc comment frames this as "a protocol violation we surface immediately", but a panic in a translator that consumes externally-sourced bytes is a crash, not a surface. The sibling `translator_pb.go::addressFromProtoBytes` ([line 223](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/upstream/translator_pb.go#L223) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/upstream/translator_pb.go#L223)) correctly returns an error. Since the upstream-package amino path is dormant in this PR, the panic isn't reachable from network ingress yet — but PR2/3 wire this in and the API contract should be settled now. Fix: convert `addressFromBytes` and the `int32From` narrowing helper to error-return, mirroring `addressFromProtoBytes` and `narrowInt32` in the sibling file. Same fopt about `FromTM2PartSetHeader`'s panics on Total range — pair these in one cleanup. Raised on #5625 review; restating here because the API survived unchanged into the split.
  </details>

- **[silent uint32 typ3 widening]** [`tm2/pkg/amino/codec.go:189-200`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/codec.go#L189-L200) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/codec.go#L189-L200) — `BinFixed32` is now accepted on `Int`/`Uint` without overflow-checking the body conversion.
  <details><summary>details</summary>

  The pre-PR code panicked at registration time for `Int`+`fixed32` ("not yet supported for int/uint"). This PR drops that and accepts it, with a code comment noting "range overflow at runtime is currently silent (the Go conversion truncates)". `typeToTyp3` now correctly returns `Typ34Byte` for Int/Uint + BinFixed32 ([codec.go:946](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/codec.go#L946) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/codec.go#L946)), and the decoder gains a fixed32 arm for both reflect.Int and reflect.Uint ([binary_decode.go:196](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/binary_decode.go#L196) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/binary_decode.go#L196), [:274](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/binary_decode.go#L274) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/binary_decode.go#L274)), so the round-trip is now consistent — the `TestBinFixed32_Int_RoundTrip` regression test pins this. The remaining concern: if a future registered type has `Int + binary:"fixed32"` and a real-world value > `math.MaxInt32`, the `int32(rv.Int())` cast at [binary_encode.go:143](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/binary_encode.go#L143) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/binary_encode.go#L143) truncates silently, producing a value that round-trips to a different number than was encoded. Fix: either keep the registration-time rejection of `Int+fixed32` (the conservative path — no live consumer needs this combination) or add a runtime range check in the encoder, panicking on overflow with a descriptive message. This change is orthogonal to the tmkms goal; consider extracting to a separate PR for review clarity.
  </details>

- **[register-time TypeURL exposure for canonical types]** [`tm2/pkg/bft/types/package.go:41-46`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/types/package.go#L41-L46) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/types/package.go#L41-L46) — `CanonicalBlockID` / `CanonicalPartSetHeader` / `CanonicalProposal` / `CanonicalVote` are now amino-registered.
  <details><summary>details</summary>

  Registration drives the proto schema export (correctly — this is the stated motivation, "so external implementations [...] can verify wire-byte compatibility"). The side effect: these types now have stable `TypeURL`s and can be encoded as `google.protobuf.Any` if ever referenced through an interface field. I grepped for callers — `bft/privval/state/state.go` uses them as concrete unmarshal targets, not via interface, so the Any path isn't reachable today. Flag if a future caller stuffs a Canonical* into an interface (`SignBytes` doesn't, but an audit-log path could): the type URL is `/tm.CanonicalVote` etc., which would conflict with any external system that expects only the operational `Vote` URL. Suggest a doc comment on the registration block stating "these are exposed for schema purposes only; do not encode through an interface".
  </details>

## Nits

- [`tm2/pkg/amino/decoder.go:57-69`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/decoder.go#L57-L69) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/decoder.go#L57-L69) — `DecodePlainVarint` returns `int64(u)` directly from `binary.Uvarint`; for proto3 `int64` this is correct, but a comment noting that decoding a 10-byte all-ones varint to int64 yields −1 (rather than overflow-erroring) would help future readers. The protobuf spec is clear, but tm2 amino's existing `DecodeVarint` rejects 64-bit overflow via `binary.Varint`'s negative-n return; the asymmetry with `DecodePlainVarint` deserves one line.

- [`tm2/pkg/bft/privval/upstream/types.go:42`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/upstream/types.go#L42) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/upstream/types.go#L42) — `Type types.SignedMsgType` is `byte`, encoded as Typ3Varint, byte-equivalent to upstream's `uint32 type`. A code comment confirming "single-byte enum, amino encodes Uint8 as varint, byte-identical to upstream's `uint32 type` for values in [0, 127]" would head off a future reviewer assuming the type widths matter.

- [`tm2/pkg/bft/privval/upstream/golden_vectors_test.go:13-26`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/upstream/golden_vectors_test.go#L13-L26) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/upstream/golden_vectors_test.go#L13-L26) — Comment correctly flags that `canonical/*` golden vectors are self-derived from tm2 amino, NOT cross-checked against upstream protoc. The other three test layers (schema walks, hand-rolled encoders, protobuf-go round-trips) do that cross-check at runtime, but the goldens themselves don't. Worth restating in the test names (e.g. `TestGolden_Canonical_*_SelfDerived`) so a future reader skimming the test list doesn't assume parity with upstream.

- [`misc/genproto/Makefile:21-43`](https://github.com/gnolang/gno/blob/b01375782/misc/genproto/Makefile#L21-L43) · [↗](../../../../.worktrees/gno-review-5716/misc/genproto/Makefile#L21-L43) — `sed -i.bak` strips the protoc version comments. The pattern matches `// \tprotoc-gen-go v` and `// \tprotoc        v` literally (with the embedded tabs in the regex). Worth a comment that the strip is keyed to the exact `protoc-gen-go` output format; a future `protoc-gen-go` version with a different header layout would silently leave the version comment in place, re-tripping ci-codegen-verify.

## Missing Tests

- **[backward-decode of pre-retag bytes]** None — by design.
  <details><summary>details</summary>

  A test confirming that `CanonicalVote` / `CanonicalProposal` produced by pre-PR amino does NOT round-trip through post-PR amino would document the consensus-byte breakage. Today there's no such test, just inference. Suggest adding a tiny `TestCanonical_NoBackcompat` that hard-codes a sample pre-PR sign-byte sequence (e.g. the old `TestProposalSignBytesTestVectors` 42-byte vector) and asserts `UnmarshalSized` fails or returns garbage. Pins the contract.
  </details>

- **[register-time int/uint+fixed32 round-trip with overflow]** [`tm2/pkg/amino/tests/binary_fixes_test.go:31`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/tests/binary_fixes_test.go#L31) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/tests/binary_fixes_test.go#L31) — `TestBinFixed32_Int_RoundTrip` covers values within int32 range. A negative test asserting that `Int=math.MaxInt64` + `binary:"fixed32"` either (a) panics on encode or (b) round-trips with explicit truncation documented would pin the silent-truncation behavior.

## Suggestions

- [`tm2/pkg/amino/codec.go:209-221`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/amino/codec.go#L209-L221) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/amino/codec.go#L209-L221) — The varint-tag kind whitelist correctly rejects `Int8`/`Int16`. Worth also extending the panic message to explicitly say "use binary:\"fixed32\" or no tag" — current message is correct but terse.

- [`tm2/pkg/bft/privval/upstream/upstreampb/upstream.proto`](https://github.com/gnolang/gno/blob/b01375782/tm2/pkg/bft/privval/upstream/upstreampb/upstream.proto) · [↗](../../../../.worktrees/gno-review-5716/tm2/pkg/bft/privval/upstream/upstreampb/upstream.proto) — File header should pin the upstream Tendermint git SHA the subset was derived from (not just version 0.34). v0.34 had 35 patch releases; future contributors will want to know which one. The `Makefile` exception already documents the regen recipe; pinning the source SHA closes the loop.

## Questions for Author

- Re the canonical-byte change: is there a follow-up PR or doc landing a `priv_validator_state.json` migration / "wipe-on-upgrade" advisory for any pre-mainnet operators carrying state forward? The state.go decode path is silent on schema drift; that's an operator footgun even pre-mainnet.

- Is there a planned home for the orphan `Int/Uint + binary:"fixed32"` widening (the `codec.go:189` change that drops the "not yet supported" panic)? This is a real fix, not just a tmkms-needed change — could land standalone with `TestBinFixed32_Int_RoundTrip` as its own PR for review clarity.
