# PR #5625: feat(tm2/privval): tmkms compatibility for the privval socket

**URL:** https://github.com/gnolang/gno/pull/5625
**Author:** @jaekwon | **Base:** master | **Files:** 65 | **+9615 -94**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR implements upstream Tendermint v0.34 privval socket protocol compatibility so that unmodified **tmkms** (Iqlusion's production-grade KMS, used across major Cosmos chains) can sign for gno.land validators without any tmkms code changes.

### Problem

tm2's two existing privval modes have material production gaps:
- **Local file**: consensus key lives next to a networked process — unsuitable for production stake.
- **gnokms**: alpha-tier, no HSM backends, no threshold signing, fail-open allowlist defaults, dial direction is wrong (gnokms should listen, not dial), and HRS double-sign authority lives on the validator host rather than the signer.

### Solution

A new `tm2/pkg/bft/privval/upstream/` package that speaks the upstream Tendermint v0.34 privval socket wire protocol. Operators add a `[priv_validator.tmkms_listener]` block to `config.toml`; gnoland then listens for tmkms to dial in.

### Architecture

```
 gnoland (validator)                tmkms (signer host)
   TCPListener (allowlist) ◄──dials──  tmkms validator block
   SignerListenerEndpoint                (chain_id, secret_key)
   SignerClient                          consensus.json (HRS gate)
   RetrySignerClient
   consensus engine
```

### Changed areas

**1. Wire format (amino changes)**

- `tm2/pkg/amino/encoder.go` / `binary_encode.go` / `binary_decode.go`: new `binary:"varint"` struct tag and corresponding `EncodePlainVarint`/`DecodePlainVarint` functions to encode signed integers as plain protobuf varint (proto int64/int32) instead of the default zigzag encoding (proto sint64/sint32).
- `tm2/pkg/bft/types/canonical.go`: **Breaking change to canonical sign-bytes** — `CanonicalPartSetHeader` fields reordered from `{Hash, Total int}` to `{Total uint32, Hash}` to match upstream Tendermint v0.34's field order; `CanonicalProposal.POLRound` changed from `binary:"fixed64"` to `binary:"varint"`.

**2. New upstream package** (`tm2/pkg/bft/privval/upstream/`)

- `secret_connection.go`: port of cometbft v0.34.34's Merlin-bound STS SecretConnection (different from tm2's chain p2p SecretConnection, kept unchanged).
- `socket_listener.go`: `TCPListener` (SecretConnection + allowlist) and `UnixListener` (kernel isolation, chmod 0600).
- `signer_endpoint.go`: shared base conn state, nonce tracking, read/write deadline management.
- `signer_listener_endpoint.go`: accept loop, ping keepalive, reconnect machinery.
- `signer_client.go`: `PrivValidator` implementation over the socket — PubKey caching, connection-generation-aware identity re-verification, echo-field validation.
- `retry_signer_client.go`: wraps `SignerClient` with retry-except-on-RemoteSignerError semantics.
- `translator.go` / `translator_pb.go`: tm2 ↔ upstream type conversions.
- `msgs.go`: `WrapMsg`/`UnwrapMsg` for the privval message envelope.
- `types.go`: upstream-shaped `Vote`, `Proposal`, `BlockID`, `PartSetHeader` with varint amino tags.
- `upstreampb/`: protoc-generated Go types from a subset of upstream Tendermint v0.34's proto files.

**3. Config wiring** (`tm2/pkg/bft/privval/upstream_config.go`, `config.go`)

New `TmkmsListenerConfig` struct; `NewPrivValidatorFromConfig` branches to `newTmkmsListenerPrivValidator` when `listen_addr` is set.

**4. gnokms HRS guard** (`contribs/gnokms/internal/common/state_signer.go`)

`HRSGuardedSigner` wraps any `types.Signer` with a persistent FileState gate that enforces strict HRS monotonicity. Mirrors tmkms's `consensus.json` gate for gnokms's signer path.

**5. Tests**

- `bug_fixes_test.go`: regression suite for 10 identified bugs (Stop-blocking-Init, Sign-without-Init, TOCTOU signer-swap, drop-on-EOF, etc.).
- `upstreamwire_test.go`: hand-rolled protobuf encoder checks byte-identity against amino.
- `stdlibproto_test.go`: stdlib protobuf decode round-trips.
- `secret_connection_compat_test.go`: pin the Merlin/HKDF constants that must match upstream.
- `tmkms_integration_test.go`: end-to-end against real tmkms binary (gated behind `tmkms_integration` build tag).

**6. ADR and docs**

`tm2/adr/adr-003-tmkms-compat.md` and `docs/validators/tmkms.md` operator guide.

**Other changes**

- `tm2/pkg/bft/blockchain/reactor.go`: minor nil-guard fix for `switchToConsensusFn` (surfaced during testing).
- `tm2/pkg/internal/p2p/p2p.go`: `MakeConnectedPeer` helper for extending existing test clusters.

---

## Test Results

- **Existing tests:** PASS — `tm2/pkg/bft/privval/upstream/...`, `tm2/pkg/bft/types/...`, `tm2/pkg/amino/...`, `tm2/pkg/bft/blockchain/...` all pass. Three test packages (`privval`, `privval/signer/local`, `privval/state`) have pre-existing failures due to permission-check tests running as root (chmod 0444 directories are bypassed by root); these failures exist identically on master.
- **Edge-case tests:** Extensive — 10 bug-fix regression tests, 3-layer wire compat suite, byte-level test vectors for canonical types.

---

## Critical (must fix)

- [ ] `tm2/pkg/bft/types/canonical.go:27` — **Canonical sign-bytes breaking change for existing validators.** `CanonicalPartSetHeader` field order changed from `{Hash []byte (field 1), Total int (field 2)}` to `{Total uint32 (field 1), Hash []byte (field 2)}`. This changes the amino wire encoding for every vote and proposal that contains a non-empty BlockID (i.e., every real precommit on a block). Any validator running an old binary against a new binary's sign-bytes will fail to verify those signatures, and validators that signed historically with the old encoding cannot have their precommits verified by nodes running the new code. The POLRound change from `fixed64` to `varint` has the same impact on proposals. **This is a consensus-breaking hard fork change.** It needs to be called out explicitly in the PR description with a migration plan, or alternatively it needs a separate feature-flag or version-coordinated deployment. The PR description does not discuss the existing-validator migration path.

  Note: The wire tests all pass because they compare the new amino output against the new protobuf spec — they confirm compatibility with tmkms, but do not check backward compat with historical tm2 signatures.

- [ ] `tm2/pkg/bft/privval/config.go:160–233` — **No FileState gate on the tmkms path.** When `TmkmsListener` is enabled, `NewPrivValidatorFromConfig` returns a `RetrySignerClient` directly, skipping the local `priv_validator_state.json` gate. This is intentional (the PR comment says it would be a "misconfiguration footgun"), but it means that if the operator misconfigures their tmkms `consensus.json` path (e.g., forgets to copy it to a new host), or if the tmkms `consensus.json` is lost/corrupted, double-signing protection evaporates silently. The validator node itself will sign anything tmkms approves. The doc (`tmkms.md`) acknowledges this but the code has no safeguard against a operator's tmkms instance being misconfigured. At minimum, this should be a hard-to-miss warning in the startup log.

---

## Warnings (should fix)

- [ ] `tm2/pkg/bft/privval/upstream/secret_connection.go:300–309` — **Unclamped X25519 private key.** The comment acknowledges that `box.GenerateKey` does not clamp the scalar (upstream Rust uses x25519-dalek which does clamp). The cometbft docs call this "harmless for years" but this is an assertion about the DH computation, not a proof. The divergence should be more prominently documented (not just a code comment) since it's a security-relevant implementation difference from the spec.

- [ ] `tm2/pkg/bft/privval/upstream/signer_client.go:70–75` — **`Init` panics on second call.** The rationale is sound (a double Init is a programmer error), but in production node code, `panic` propagates to the operator as a crash with a stack trace. Consider returning an error with a descriptive message instead, so the operator sees a clear log line rather than a panic dump. Panics in server code are difficult to attribute to their root cause without post-mortem analysis.

- [ ] `tm2/pkg/bft/privval/config.go:201–207` — **UDS chmod is best-effort and silently fails.** The code logs a warning on `os.Chmod` failure but continues. On a filesystem that doesn't honor socket permissions, any local user can connect to the unix socket as the signer. Given that UDS-mode intentionally skips SecretConnection and allowlist checking ("kernel isolation suffices"), a chmod failure effectively gives local users signing capability. Consider failing hard (or at minimum, requiring explicit operator acknowledgment) instead of continuing with a warning.

- [ ] `tm2/pkg/bft/privval/upstream/socket_listener.go:222–232` — **Empty allowlist is fail-open.** `checkAuthorizedKey` returns `nil` (accept) when `authorizedKeys` is empty. The `ValidateBasic` on `TmkmsListenerConfig` enforces a non-empty allowlist when the listener is enabled, but the `TCPListener` itself still accepts any key if constructed directly with an empty slice. If a caller ever bypasses config validation, this is a security gap. Defensive: panic or error on empty allowlist in `NewTCPListener`, since the ValidateBasic guard is the only protection.

- [ ] `tm2/pkg/bft/privval/upstream/translator.go:33-49` — **Panics in `ToTM2Vote` and `ToTM2Proposal` on malformed wire input.** `addressFromBytes` panics if the wire address length is not exactly 20. While the PR comment says "protocol violation we surface immediately", a panic in a response-parsing path can be triggered by a malicious signer and will crash the validator. The `translator_pb.go` version correctly returns an error (`addressFromProtoBytes`). Consider unifying on the error-return pattern.

- [ ] `tm2/pkg/bft/privval/upstream/signer_listener_endpoint.go:304–316` — **Ping loop runs independently of the instance lock; reconnect can race with in-flight SendRequest.** `pingLoop` calls `SendRequest` (which takes `instanceMtx`) but between the ping detection ("endpoint not connected") and `triggerReconnect`, a concurrent `SignVote` could be in flight. The existing code path handles this via the serialization guarantee of `instanceMtx`, but it's subtle; a comment in `pingLoop` would clarify the invariant.

---

## Nits

- [ ] `tm2/pkg/bft/privval/upstream/signer_client.go:94` — `sc.verifiedGen.Store(sc.endpoint.ConnectionGeneration())` runs under `endpoint.Lock()` which is correct, but the comment at line 85 says "Take the instance lock for the whole 'fetch pubkey + record gen' transaction" — could be slightly clearer that this prevents a reconnect from bumping `connGen` between the two operations.

- [ ] `tm2/pkg/bft/privval/upstream/types.go:43` — The `binary:"varint"` tag on `Height int64` makes it a plain varint. Fine for the wire protocol, but the package comment should explicitly note that amino's default for `int64` is zigzag varint, and this tag overrides that. The current comment says "plain varint" but doesn't mention the amino default for newcomers.

- [ ] `tm2/pkg/bft/privval/upstream/protoio.go:58–61` — The `varintLen` re-derivation at the end of `ReadMsg` is unnecessary overhead on the hot path; the `bufio.Reader` already advanced past the varint. Either remove the return of byte counts (return only `error`) or use the actual bytes consumed by the `binary.ReadUvarint` call.

- [ ] `docs/validators/tmkms.md` — Good operator doc. Should explicitly state the consensus-breaking nature of upgrading from a pre-PR to a post-PR binary (CanonicalPartSetHeader reordering). Operators need to know this is not a rolling upgrade.

- [ ] `.github/workflows/ci-tmkms-integration.yml:30` — Pinned actions use full SHA hashes — good security practice. The `actions/checkout` SHA (`93cb6efe18...`) does not match the standard `v4` (or `v3`) tag — worth verifying it corresponds to the intended release.

---

## Missing Tests

- [ ] **CanonicalPartSetHeader field order regression test.** The vote test vectors in `vote_test.go` use zero-value BlockIDs (no parts), so the field-reorder change is not covered by explicit byte-vector tests for `CanonicalPartSetHeader`. A test vector that includes a non-empty `PartSetHeader` in the canonical vote sign-bytes would pin the new field order and catch any future reversion.

- [ ] **UDS chmod-failure path.** The best-effort `os.Chmod` code path is not tested (the only test coverage for UDS would require constructing a filesystem that ignores chmod). At minimum, the behavior should be documented in the test suite with a comment explaining why there is no test.

- [ ] **POLRound varint encoding regression.** The `TestProposalSignBytesTestVectors` test only covers `POLRound=-1` (which encodes as 10 bytes in plain varint vs 8 bytes in fixed64). Add a test case with `POLRound=0` and `POLRound>0` to lock down the full varint range.

- [ ] **`HRSGuardedSigner.classifySignBytes` ambiguity.** The function tries to decode as a vote first, then as a proposal. It's possible for bytes that are semantically a proposal to decode successfully as a vote (if all amino fields happen to match). A property test over random sign-bytes would validate the classification logic is deterministic.

---

## Suggestions

- **Consider a dual-gate option.** Even in tmkms mode, keeping a read-only copy of the HRS state locally (updated but never used as a gate) would let operators audit whether the tmkms `consensus.json` state and the validator's local observation diverge, without introducing a gate that could block signing. This provides defense-in-depth without the "misconfiguration footgun" concern.

- **The amino `binary:"varint"` tag change is a cross-cutting concern.** Adding it to the amino codec is the right approach but it changes the semantic of the existing codec in a subtle way. A dedicated migration guide or changelog entry for amino consumers would prevent surprises for anyone relying on amino's encoding semantics for `int64` fields tagged with `binary:"varint"`.

- **`NewTCPListener` allowlist documentation.** The comment says "An empty allowlist accepts ANY peer" — this is dangerous enough to deserve a panic or at minimum a dedicated `ErrEmptyAllowlist` error type rather than silent fail-open behavior, to prevent future callers from accidentally creating an unauthenticated listener.

---

## Questions for Author

1. **Breaking change acknowledgment**: The `CanonicalPartSetHeader` field reorder and `POLRound` varint change affect every canonical sign-byte for votes and proposals in gno.land's history. Is there a genesis-height or a version flag that gates the old vs. new encoding? Or is this intended as a coordinated hard-fork change requiring all validators to upgrade simultaneously? The PR description doesn't mention this migration path.

2. **tmkms `consensus.json` recovery**: If an operator loses their tmkms `consensus.json` (disk failure), the operator documentation advises against restoring from backup. What is the supported recovery procedure? And how does gnoland communicate to the operator that the tmkms-side HRS gate is what they must protect?

3. **v0.34 vs v0.38**: tmkms 0.15.0 defaults to v0.38 but accepts v0.34. Is there a plan to add v0.38 support, or is v0.34 intended as the stable target? tmkms deprecated v0.34 and some backends may eventually drop it.

4. **Horcrux compatibility**: The PR and docs mention Horcrux as a user of the same upstream protocol. Has Horcrux been tested against this listener, or is that a claim based on protocol compatibility with upstream Tendermint?

5. **`HRSGuardedSigner` and gnokms integration**: The `HRSGuardedSigner` is added to `contribs/gnokms` but is not wired into the gnokms signer path in this PR. Is this deliberate (intended to be added in a follow-up)?

---

## Verdict

NEEDS DISCUSSION — The implementation is technically strong, well-structured, and addresses real security gaps in validator key management. The bug-fix test suite in particular shows careful analysis of race conditions and protocol edge cases. However, the `CanonicalPartSetHeader` field order change and `POLRound` encoding change are **consensus-breaking** for any existing gno.land deployment: they alter the bytes that get signed for every vote and proposal with a non-empty BlockID. This needs an explicit migration plan, a genesis/version gate, or a clear statement that gno.land is pre-mainnet and this is acceptable. The code itself is merge-ready from a correctness standpoint pending that discussion and the minor issues above.
