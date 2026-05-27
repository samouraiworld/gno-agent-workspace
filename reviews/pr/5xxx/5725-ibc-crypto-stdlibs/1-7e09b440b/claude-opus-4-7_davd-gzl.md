# PR #5725: feat(gnovm/stdlibs): IBC crypto stdlibs (bn254, cometbls, keccak256, merkle, modexp)

URL: https://github.com/gnolang/gno/pull/5725
Author: moul | Base: master | Files: 60 | +3904 -4
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `7e09b440b` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5725 7e09b440b`

**Verdict: NEEDS DISCUSSION** — six new stdlibs land clean (tests green, determinism txtar pins gas, apphash pinned correctly), but four open items must close before merge: gas table is a Xeon-Silver-4114 draft (not the reference Xeon 8168), modexp's linear gas schema mis-charges a cubic primitive, ship-decision between `cometbls` (monolithic native) vs `cometblszk` (pure-gno on top of bn254/keccak/modexp) is still TBD, and the embedded verifying-key constants are union-devnet-1337 — not mainnet material. EIP-196 padding parity also drifts between `g1Add` (right-pads short input) and `g1Mul` (strict 96-byte reject), which contradicts both the package doc and the precompile spec.

## Summary

Six new packages under `gnovm/stdlibs/crypto/`: `bn254` (EIP-196/197 G1 add/mul + pairing check via gnark-crypto), `cometbls` (CometBLS Groth16 verifier as one native call), `cometblszk` (pure-gno port of the same verifier on top of bn254/keccak256/modexp), `keccak256` (Legacy Keccak-256), `merkle` (Tendermint simple Merkle), `modexp` (EIP-198 MODEXP). The natives are wired through `gnovm/stdlibs/generated.go` (regenerated via `go generate`), gas charged through 10 new entries in `native_gas.go`, and a determinism txtar (`stdlib_ibc_crypto_determinism.txtar`) pins both the digest bytes and `GAS USED == 2_798_422` across a `gnoland restart`. Genesis stdlib set grew, so `TestAppHashCrossrealm38`'s pinned multistore hash was bumped to `059428e5…0262` (verified locally — passes). The whole submission is consensus-relevant because the `.gno` source bytes ship in genesis and the new natives charge gas.

## Glossary

- **G1 / G2**: BN254 curve points in F_p and F_p² respectively. G1 has prime order (cofactor 1, so on-curve ⇒ in-subgroup); G2 has cofactor ≠ 1 (subgroup check required).
- **EIP-196 / EIP-197**: Ethereum precompile specs for BN254 ECADD/ECMUL/ECPAIRING.
- **Groth16**: Pairing-based SNARK; verification is a 4-term pairing product equality to 1 (when α/β/γ/δ are pre-negated).
- **Pedersen POK**: Commitment proof-of-knowledge added to the verifying key; CometBLS bundles it next to the Groth16 proof.
- **Apphash / multistore commit hash**: iavlStore Merkle root; shifts whenever the genesis stdlib set or any `.gno` source byte changes.
- **EXACT_GAS**: txtar matcher that asserts an exact `GAS USED` value, used here to pin natives across restart.

## Fix

The PR is purely additive on the stdlib side: six new package directories under `gnovm/stdlibs/crypto/` plus regenerated `generated.go` bindings and gas-table entries. The native dispatcher wiring is auto-generated, so the only hand-written glue is the `X_*` host functions in each package. The consensus-relevant non-additive change is the bumped `expectedCrossrealm38Hash` in [`gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:55`](https://github.com/gnolang/gno/blob/7e09b440b/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L55) · [↗](../../../../../.worktrees/gno-review-5725/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L55), driven by the genesis stdlib set growing.

## Critical (must fix)

- **[devnet verifying key embedded as a constant]** [`gnovm/stdlibs/crypto/cometbls/constants.go:29-37`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/cometbls/constants.go#L29-L37) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/cometbls/constants.go#L29-L37) — `verifying_key.bin` is the union-devnet-1337 setup; `DeltaNegG2 == PedersenGRootSigmaNeg` byte-for-byte gives that away.
  <details><summary>details</summary>

  Both `cometbls/constants.go` and `cometblszk/constants.gno` carry the same constants extracted from `verifying_key.bin`, and `DeltaNegG2` is byte-identical to `PedersenGRootSigmaNeg` (G2 point `07b8dbef…02aca5d2…2edb19cb…1696ccaf…`). In a real ceremony those two are independent secrets — equality here means the devnet setup reused δ as the Pedersen σ-root, which is fine for `union-devnet-1337` but unacceptable for anything else. The matching test vector (`testProofHex` in [`cometbls_test.go:12-17`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/cometbls/cometbls_test.go#L12-L17) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/cometbls/cometbls_test.go#L12-L17)) is keyed to the same devnet. Fix: before flipping CometBLS on for any non-devnet light client, regenerate constants from the production verifying-key blob (`go run ./cmd/gen -in verifying_key.bin -out constants.go` per the file header), add a determinism txtar pinned to the mainnet vector, and document the upgrade path — currently a chain upgrade is required to swap the key (compiled-in `init()` panics if absent), which is itself worth a Question for Author. The cometblszk package has the same issue and is fed from the same upstream blob.
  </details>

- **[modexp linear gas schema mis-charges cubic primitive]** [`gnovm/stdlibs/native_gas.go:148`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/native_gas.go#L148) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/native_gas.go#L148) — `Base: 58000, Slope: 24647680 (per 1024)` calibrated at N=256B modulus; smaller inputs over-charged, very large inputs **under-charged**.
  <details><summary>details</summary>

  `big.Int.Exp` is O(len(exp) · len(mod)²) (school-multiplication on the modular reduction step) — quadratic in `len(modulus)`, linear in `len(exp)`. The gas row charges `Base + Slope · SizeLenBytes` against `SlopeIdx: 2` (`modulus`), so doubling the modulus length doubles the charge but quadruples the runtime. At N=256 the fit is tight (~6.16ms / ~24M gas), but a 4096-byte modulus would cost ~16× the actual gas budget while consuming ~256× the CPU; a malicious realm could DoS a block producer at a fraction of the gas headroom. The PR body and the comment at [`native_gas.go:143-147`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/native_gas.go#L143-L147) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/native_gas.go#L143-L147) acknowledge this, but no input cap is enforced — `X_modExp` accepts arbitrary `[]byte` and lets `big.Int` do the work. Fix: either (a) reject `len(modulus) > 256` (or whatever the calibration boundary is) at the gno wrapper, returning nil per the EIP-198 spec semantics, or (b) ship a quadratic gas schema before enabling on test13. Option (a) is the safer interim — a hard upper bound is auditable, a model-fit isn't.
  </details>

- **[g1Add vs g1Mul EIP-196 padding parity diverges]** [`gnovm/stdlibs/crypto/bn254/bn254.go:14-32`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254.go#L14-L32) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254.go#L14-L32) vs [`bn254.go:35-38`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254.go#L35-L38) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254.go#L35-L38) — `X_g1Add` right-pads short input to 128B per EIP-196; `X_g1Mul` rejects anything that isn't exactly 96B. Both wrappers document themselves as mirroring the EVM precompile.
  <details><summary>details</summary>

  go-ethereum's `runBn256Add` / `runBn256ScalarMul` both call `getData(input, 0, …)`, which right-pads short input to the precompile's expected length. The bn254 package doc string at [`bn254.go:12`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254.go#L12) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254.go#L12) advertises EIP-196 parity, the gno-side doc at [`bn254.gno:40-45`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254.gno#L40-L45) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254.gno#L40-L45) says "must be exactly G1MulInputSize bytes", and the implementation at [`bn254.go:36-38`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254.go#L36-L38) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254.go#L36-L38) honors the gno-side doc — so the divergence is between `g1Mul` and the EIP-196 claim, not within the code itself. Also note: [`bn254.gno:33-37`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254.gno#L33-L37) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254.gno#L33-L37) says G1Add "input must be exactly G1AddInputSize bytes" but the host actually pads — the gno doc lies about its own host. Pick one and stay consistent: either (a) make `X_g1Mul` right-pad to 96B too (closest to EIP-196 and to `X_g1Add`), or (b) make `X_g1Add` strict and update the package-level claim. Whichever path, regenerate the gno doc comments. Tests under `bn254_test.go` already assert short-input padding for `g1Add` (`cdetrio4_empty`, `cdetrio6`, `cdetrio8`) so any change to that side must keep them green.
  </details>

## Warnings (should fix)

- **[gas table calibrated on wrong reference machine]** [`gnovm/stdlibs/native_gas.go:66-84`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/native_gas.go#L66-L84) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/native_gas.go#L66-L84) — Xeon Silver 4114 draft for IBC rows, Apple M2 for everything else, reference is Xeon 8168.
  <details><summary>details</summary>

  Two different "draft" machines compose the table. The existing 46 rows were calibrated on Apple M2; the 10 new IBC rows on Xeon Silver 4114. The reference machine per `gnovm/cmd/calibrate/README.md` is Xeon 8168. Concretely: `pairingCheck` is the priciest call (~457k base + 1.7M slope per pair) and a single-pair check is currently 2.2M gas — about 22% of a `gas-wanted=10M` tx. If the Xeon 8168 measurement comes in higher, the relative price shifts; if it comes in lower, the chain is over-charging. The PR is explicit about this — the table comment, the per-row "draft" tag, and the PR body all flag it. Fix: re-calibrate on Xeon 8168 before any test13 chain upgrade. Until then, gating on `// draft` means the chain ships with a known-wrong fee schedule. Tracking issue would help (none linked).
  </details>

- **[cometbls vs cometblszk ship-decision deferred]** [`gnovm/stdlibs/crypto/cometbls/`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/cometbls/) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/cometbls/) and [`gnovm/stdlibs/crypto/cometblszk/`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/cometblszk/) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/cometblszk/) — two implementations of the same Groth16 verifier, both gated to genesis. Need a call.
  <details><summary>details</summary>

  `cometbls` is a single native (Go code, ~2.6ms/call, panics on init if constants malformed, depends on `gnark-crypto` + `golang.org/x/crypto/sha3`). `cometblszk` is pure-gno on top of `bn254`/`keccak256`/`modexp`/`sha256` natives (smaller trusted surface, auditable from gno, but pays the gno↔native bridge cost on every primitive call — at least 4 pairing-check + 3 scalar-mul + 1 add + 1 keccak + 1 modexp natives per verification). PR body asks "ship both, or one?" but doesn't settle it. Shipping both bakes both into the genesis stdlib set permanently — they can't be removed without a hard fork. Fix: decide before merging. If both ship, document the chooser (latency budget vs trust surface vs gas) somewhere reachable; otherwise drop one. Worth noting the test vector and the verifying-key constants are duplicated between the two packages — every change requires touching both.
  </details>

- **[gnark-crypto and x/crypto/sha3 enter the trusted code base unchecked]** [`go.mod:additions`](https://github.com/gnolang/gno/blob/7e09b440b/go.mod) · [↗](../../../../../.worktrees/gno-review-5725/go.mod) — the PR body flags a sandbox audit; the diff itself doesn't add one.
  <details><summary>details</summary>

  `github.com/consensys/gnark-crypto v0.14.0` and `golang.org/x/crypto/sha3` are pulled into `gnovm/stdlibs/crypto/bn254` and `gnovm/stdlibs/crypto/cometbls` (Go side). Both are vetted upstream libraries, but they bring in transitive deps (`github.com/bits-and-blooms/bitset`, `github.com/consensys/bavard`, `github.com/mmcloughlin/addchain`, etc. — visible in the `contribs/*/go.sum` diffs) and run in the consensus-critical path of every realm that calls these natives. The PR body asks whether gno has an allow-list gate for native dependencies — that question needs an answer in this thread before merge, not after. Fix: confirm with maintainers what the policy is, link the issue/PR if it exists, otherwise add the audit as a Critical for this PR.
  </details>

- **[`hashFromByteSlices` length decoder permits unbounded item count]** [`gnovm/stdlibs/crypto/merkle/merkle.go:53-73`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/merkle/merkle.go#L53-L73) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/merkle/merkle.go#L53-L73) — `decodeByteSlices` reads a 32-bit count then `make([][]byte, count)`. An adversarial caller can request `count = 0x7fffffff` and OOM the dispatcher before per-byte gas charges fire.
  <details><summary>details</summary>

  The current gas row at [`native_gas.go:141`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/native_gas.go#L141) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/native_gas.go#L141) charges on `SizeLenBytes` of the encoded slice (slope ~184 ns/B) — so a small encoded input claiming a giant count slips past gas before `make` runs. Worst case: 4-byte input `[0xff, 0xff, 0xff, 0xff]` → `make([][]byte, 0x7fffffff)` = 16-24 GB allocation, then the subsequent length-prefix loop bails on the first short read but the slice header is already allocated. Fix: bound count against `len(b)/min_per_item_overhead` before allocating, or cap `count` to a hard limit (tm2's `merkle.maxAunts = 100` is the closest existing knob; `merkle.X_verifySimpleProof` already inherits a sane bound via `tmhash.Size = 32` on aunts). The native is also non-streaming, so the realistic upper bound on tree size is whatever fits in `MaxTxBytes` anyway — pinning that as the cap is cheap insurance.
  </details>

## Nits

- [`gnovm/stdlibs/crypto/cometblszk/cometblszk.gno:125-127`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/cometblszk/cometblszk.gno#L125-L127) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/cometblszk/cometblszk.gno#L125-L127) — `copy(buf[:], make([]byte, 32))` allocates a throwaway slice to zero a fixed-size array. Replace with `var buf [32]byte` (already zero-valued in gno) or `clear(buf[:])` if available.
- [`gnovm/stdlibs/crypto/cometbls/cometbls.go:114-127`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/cometbls/cometbls.go#L114-L127) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/cometbls/cometbls.go#L114-L127) — sentinel errors are unexported as `errors.New(…)` strings; consumers from the gno side get them through `X_verifyZKP` as a string and lose `errors.Is` matching. Fine for the wire shape, just note the asymmetry.
- [`gnovm/stdlibs/crypto/bn254/bn254.gno:9`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254.gno#L9) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254.gno#L9) — `package bn254` under `crypto/bn254` — fine, but worth confirming with maintainers since `crypto/sha256` and friends use shorter aliases.
- [`gnovm/stdlibs/crypto/keccak256/keccak256.gno:9`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/keccak256/keccak256.gno#L9) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/keccak256/keccak256.gno#L9) — `Sum256` returns `[Size]byte` while `merkle.LeafHash` returns `[]byte`; intentional but the asymmetry is the kind of thing that bites realm authors. Document in a top-of-package note.

## Missing Tests

- **[no fuzz/property coverage for the EIP-196/197 host bindings]** [`gnovm/stdlibs/crypto/bn254/bn254_test.go`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/crypto/bn254/bn254_test.go) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/crypto/bn254/bn254_test.go) — KAT-only coverage; no property tests for `g1Add` (commutativity, associativity, identity, inverse) or `pairingCheck` (bilinearity).
  <details><summary>details</summary>

  The KAT vectors lift verbatim from go-ethereum, which is great for spec parity but blind to implementation bugs that survive both libraries (gnark-crypto issues, encoding-side off-by-ones). Add: `Add(P, Q) == Add(Q, P)` and `Add(P, -P) == 0` over random points; for pairing, `e(aP, bQ) == e(P, Q)^(ab)` (probabilistic; small `a`/`b`). One short fuzz harness in `bn254_test.go` covers both. Without these, the bug surface is exactly "what wasn't in bn256Add.json".
  </details>

- **[no gno-side bench]** [`gnovm/cmd/calibrate/ibc_native_bench_test.go`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/cmd/calibrate/ibc_native_bench_test.go) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/cmd/calibrate/ibc_native_bench_test.go) — benches drive `dispatchHarness` end-to-end (good), but `cometblszk` (pure-gno on top of natives) has zero benchmark coverage and ships a gas row implicitly (it just sums the underlying natives).
  <details><summary>details</summary>

  Per the comment in [`native_gas.go:79-84`](https://github.com/gnolang/gno/blob/7e09b440b/gnovm/stdlibs/native_gas.go#L79-L84) · [↗](../../../../../.worktrees/gno-review-5725/gnovm/stdlibs/native_gas.go#L79-L84) the table is exhaustive for `generated.go`, and cometblszk is pure-gno so it shouldn't appear there — that's correct on paper. But the end-to-end cost of a `cometblszk.VerifyZKP` call (4 PairingChecks + 3 G1Muls + multiple G1Adds + 1 Keccak + 1 ModExp + sha256 work) needs an empirical number before anyone can know if it's competitive with the monolithic native. Add a single `BenchmarkCometBLSZK_VerifyZKP` next to the cometbls one. The reader needs the comparison to make the ship-decision flagged above.
  </details>

- **[determinism txtar doesn't exercise the cometbls/cometblszk path]** [`gno.land/pkg/integration/testdata/stdlib_ibc_crypto_determinism.txtar:60-83`](https://github.com/gnolang/gno/blob/7e09b440b/gno.land/pkg/integration/testdata/stdlib_ibc_crypto_determinism.txtar#L60-L83) · [↗](../../../../../.worktrees/gno-review-5725/gno.land/pkg/integration/testdata/stdlib_ibc_crypto_determinism.txtar#L60-L83) — pins keccak256, bn254 G1Add, modexp, merkle LeafHash. Skips the two heaviest paths (cometbls/cometblszk verifyZKP) — the ones with the highest non-determinism foot-gun surface (gnark pairing precomputation tables, etc.).
  <details><summary>details</summary>

  The PR rationale ("cold-vs-warm code paths" — lazy table init, accumulator reuse, big.Int aliasing) applies most strongly to the pairing/Groth16 paths, but those are exactly what the txtar omits. The Go-side `cometbls_test.go` runs the verifier but only inside the unit-test process, not across a `gnoland restart`. Add a second realm that calls `cometbls.VerifyZKP(chainID, tvh, header, proof)` (using the existing devnet vector) and pin its `GAS USED` across restart. Same for cometblszk if it ships. Otherwise the determinism guarantee is partial: the primitive natives are covered, the verifier paths aren't.
  </details>

## Suggestions

- [`examples/gno.land/p/samcrew/keccak256/`](https://github.com/gnolang/gno/blob/7e09b440b/examples/gno.land/p/samcrew/keccak256/) · [↗](../../../../../.worktrees/gno-review-5725/examples/gno.land/p/samcrew/keccak256/) — pure-gno keccak256 lives at the example path, now duplicated by `crypto/keccak256` (native). `Villaquiranm` asked about this in the PR thread.
  <details><summary>details</summary>

  `gno.land/p/samcrew/keccak256` is a 992-line pure-gno port of `x/crypto/sha3`. The new native is faster and consumes less gas (one native dispatch vs a full sponge in interpreted gno), so it'll get adopted by every realm that's currently importing samcrew. Add a deprecation note to samcrew's README pointing at `crypto/keccak256`, or fold the migration into a follow-up tracking issue. Don't remove samcrew in this PR — it stays valid for realms that need streaming `hash.Hash` semantics; the native is one-shot `Sum256` only.
  </details>

- [`docs/`](https://github.com/gnolang/gno/blob/7e09b440b/docs/) · [↗](../../../../../.worktrees/gno-review-5725/docs/) — no docs touched. The six new stdlibs are user-facing and ship in genesis; at minimum a one-line entry in the stdlib reference index.
  <details><summary>details</summary>

  Grepped `docs/` for `keccak256|bn254|cometbls|crypto/merkle|crypto/modexp` — zero hits. The PR body is excellent documentation but lives on GitHub. For realm authors discovering the stdlib via the docs site, the IBC crypto family doesn't exist until docs are updated. Worth a follow-up PR or a stub here.
  </details>

## Questions for Author

- Why ship both `cometbls` (native) and `cometblszk` (pure-gno) instead of picking one — what's the decision criterion?
- What's the production verifying-key story for `cometbls`? The constants are compiled-in (`init()` panics if absent), so changing them post-launch needs a chain upgrade. Is there a chain-param-based override planned, or will every counter-party light-client deployment be a hard fork?
- Is the sandbox/native-dependency allow-list audit (PR body item 4) tracked anywhere — issue link, ADR, separate review thread?
- The PR body marks 4 open items in the "Still draft because" section but the PR isn't marked as draft on GitHub. Intentional?
