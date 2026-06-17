# PR #5678: WIP feat(gnovm): add math/big stdlib (Int subset)

URL: https://github.com/gnolang/gno/pull/5678
Author: davd-gzl | Base: master | Files: 8 | +1732 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `e46de6fc4` (stale — +1 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5678 e46de6fc4`

**Verdict: NEEDS DISCUSSION** — correct, well-tested, cleanly bridged; the load-bearing open items are (1) gas calibration, which the author already flags as a hard pre-mainnet prerequisite, and (2) confirming that stdlib `_test.gno` files belong in the consensus-relevant persisted package set, since adding test cases shifts the `crossrealm38` apphash. No correctness defects found.

> Parallel/red-team review. Three reviewer agents were dispatched; the red-team agent completed and its gas-DoS findings are folded in below. The blue-team and correctness agents died on API connection errors mid-run, so those two lenses were done directly against source by the parent. notjoon has **no comments** on this PR (no issue comments, no inline review comments, no submitted reviews).

## Summary
Adds `math/big` as a stdlib with the `Int` subset. The `.gno` API mirrors Go's `math/big.Int` (NewInt/Set/Sign/Cmp/Add/Sub/Mul/Quo/Rem/QuoRem/Div/Mod/DivMod/SetString/Text/…) so Go code ports with minimal change. Wire format is `(neg bool, abs []byte)` — big-endian magnitude, no leading zeros, empty == canonical zero. Trivial methods are pure Gno; the seven heavy ops (`add/sub/mul/quoRem/divMod/setString/text`) bridge to a Go-side `*big.Int` through the `genstd` native path, so arithmetic correctness and determinism are inherited from the Go runtime. The seven new `native_gas.go` rows are explicit, uncalibrated placeholders.

## Fix
Methods decode `(neg, abs)` to `*big.Int`, compute, and re-encode via `toBig`/`fromBig` ([`int.go:57-80`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/stdlibs/math/big/int.go#L57-L80) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/stdlibs/math/big/int.go#L57-L80)); `fromBig` returns `(false, nil)` for sign 0 and every Gno setter enforces `neg = neg && len(abs) > 0`, so the single-canonical-zero invariant holds end-to-end. Division is guarded by `divCheck` so divide-by-zero is a recoverable Gno panic rather than an opaque native panic ([`int.gno:33-37`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/stdlibs/math/big/int.gno#L33-L37) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/stdlibs/math/big/int.gno#L33-L37)).

## Glossary
- `genstd` — codegen bridge that wraps Go `X_`-prefixed functions as Gno natives (`gnovm/stdlibs/generated.go`).
- canonical zero — the wire format's single representation of 0: `abs` empty, `neg` false.
- apphash pin — `expectedCrossrealm38Hash`, a hard-coded iavl Merkle root the `crossrealm38` test asserts against to catch unintended consensus-state shifts.
- `SlopeIdx` — index into a native's parameter block telling the gas metering which argument's length drives the per-byte slope.

## Changes applied on the branch (this review)
Fixes committed to the working tree at `e46de6fc4` (not pushed):

1. **Gas: charge on both operands.** `native_gas.go` binary rows now carry `Slope2` on operand `b` (`Slope2Idx: 3`) in addition to `Slope` on `a`, so an asymmetric call no longer escapes the slope (see Warning 1).
2. **Tests: cover `SetUint64`/`Uint64` + the import path.** Added [`gnovm/tests/files/math_big0.gno`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/tests/files/math_big0.gno) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/tests/files/math_big0.gno) — a filetest exercising `math/big` from user code (see Missing Tests). Deliberately a filetest, not an in-package `_test.gno` addition, to avoid shifting the apphash (see Warning 2).
3. **Comment accuracy.** Reworded the "conservative (over-charge)" table note that only held for small balanced operands.

## Critical (must fix)
None.

## Warnings (should fix)
- **[gas slope only counted operand `a` — asymmetric-operand undercharge]** [`native_gas.go:139-143`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/stdlibs/native_gas.go#L139-L143) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/stdlibs/native_gas.go#L139-L143) — every binary row sloped on `a` (`SlopeIdx:1`) only; a caller passing tiny `a`, huge `b` paid the flat base while real CPU scaled with `b`. **Partially fixed on the branch.**
  <details><summary>details</summary>

  The pre-call charge reads exactly one parameter's length ([`gnovm/pkg/gnolang/native_gas.go:136`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/pkg/gnolang/native_gas.go#L136) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/pkg/gnolang/native_gas.go#L136) via `nativeSizeFromBlock`). Red-team measured `add(1, b)` with a 5 MB `b`: actual ≈ 2.37 ms, metered ≈ 2019 ns → ~1174× undercharge; `mul(255, b)` ≈ 983×. Operand size is bounded only by alloc gas (memory, not the per-call CPU). The existing in-VM bigint metering already charges on `max(lv,rv)` (`machine.go:incrCPUBigInt`), so the new rows had regressed from that.

  Fix applied: added `Slope2` on operand `b` (`Slope2Idx:3, SizeLenBytes`) so the charge is `Base + Slope·|a| + Slope2·|b|` — the larger operand is now metered. Residual: the slopes are still **linear**, so `mul` (O(n·m)) and `setString`/`text` (decimal conversion O(n²)) still under-charge for large inputs. The schema has no quadratic term; the real follow-up is a length cap or a quadratic component, which belongs in the calibration pass the ADR already commits to. All values remain explicit placeholders gated behind the "NOT mainnet-safe" disclaimer ([`native_gas.go:81-86`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/stdlibs/native_gas.go#L81-L86) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/stdlibs/native_gas.go#L81-L86)).
  </details>

- **[stdlib `_test.gno` is in the persisted package set — test-only edits shift the consensus apphash]** [`apphash_crossrealm38_test.go:53`](https://github.com/gnolang/gno/blob/e46de6fc4/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L53) · [↗](../../../../../.worktrees/gno-review-5678/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L53) — adding test cases to `int_test.gno` changed `TestAppHashCrossrealm38`'s computed root even though no production source changed.
  <details><summary>details</summary>

  Empirically confirmed: appending two test functions to [`int_test.gno`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/stdlibs/math/big/int_test.gno) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/stdlibs/math/big/int_test.gno) flipped the apphash to a non-pinned value; reverting restored it; the Go-side `native_gas.go` change alone did not. So the `crossrealm38` scenario's store includes stdlib `_test.gno` source bytes. Two consequences worth confirming: (a) any future stdlib test addition is "consensus-breaking" and forces a re-pin, which is surprising friction; (b) if real genesis loading excludes `_test.gno`, this pin is computed against test-store content that production never sees. This is not introduced by this PR — the existing `int_test.gno` is already baked into the pin the author updated — but the math/big package is the first place it bites visibly. Confirm whether `_test.gno` inclusion in the persisted set is intended; if not, the test harness (not this PR) should filter them.
  </details>

## Nits
- [`native_gas.go:141`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/stdlibs/native_gas.go#L141) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/stdlibs/native_gas.go#L141) — `text` gas ignores the output base; `Text(2)` produces ~8× the digit count of `Text(16)`. Fold into the calibration follow-up.

## Missing Tests
- **[`SetUint64`/`Uint64` had zero direct coverage; no user-code import test]** [`int.gno:92-151`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/stdlibs/math/big/int.gno#L92-L151) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/stdlibs/math/big/int.gno#L92-L151) — **fixed on the branch.**
  <details><summary>details</summary>

  `SetUint64` was never called in `int_test.gno`; `Uint64` only indirectly. The boundary that matters — values above `MaxInt64` must stay positive — was untested. Separately, every test lived inside the package (`big_test`); nothing exercised `import "math/big"` from real `.gno` user code through the VM. Both gaps are closed by the new filetest [`math_big0.gno`](https://github.com/gnolang/gno/blob/e46de6fc4/gnovm/tests/files/math_big0.gno) · [↗](../../../../../.worktrees/gno-review-5678/gnovm/tests/files/math_big0.gno), which checks `SetUint64(1<<63)` stays positive, `MaxUint64` round-trips through `Uint64`, and a native-bridge `Mul` runs end-to-end.
  </details>

## Verified correct (no finding)
- Division routing: `Quo/Rem/QuoRem` → `quoRem` (truncated), `Div/Mod/DivMod` → `divMod` (Euclidean); negative-operand outputs match Go (`TestQuoRem`/`TestDivMod`/`TestQuo`/`TestRem`/`TestDiv`/`TestMod` pass).
- Canonical-zero invariant holds on Add/Sub/Mul/Quo/Rem/Div/Mod/Neg/Abs/Set/SetBytes/SetString/SetInt64 (`TestZeroCanonical`, `TestSetBytesNormalization`).
- Aliasing: `z.Add(z,z)`, `z.Mul(z,z)`, `q.QuoRem(q,y,r)`, `d.DivMod(d,m,m)` all correct — native params are copied to Go locals before the body runs (`TestAliasing`).
- Divide-by-zero is a recoverable Gno panic on all six entry points; `QuoRem`/`DivMod` panic on aliased output receivers (`TestDivByZero`, `TestQuoRemDistinctOutputs`, `TestDivModDistinctOutputs`).
- `IsInt64`/`IsUint64`/`Int64`/`Uint64` boundaries (MinInt64, MaxInt64, MaxUint64) correct; `MinInt64` round-trips.
- `SetString` base-0 prefix handling and `Text` invalid-base panic match Go (`TestSetStringEdgeCases`, `TestTextInvalidBase`).
- `gnomod.toml` `gno = "0.9"` matches sibling stdlibs; ADR method list matches the implemented/exported surface.

## Test runs
- `go run ../cmd/gno test ./math/big` (from `gnovm/stdlibs`): 16/16 pass.
- `go test ./gno.land/pkg/sdk/vm/ -run Gas`: ok. `-run TestAppHashCrossrealm38`: ok (pin intact after fixes).
- `go test ./gnovm/pkg/gnolang/ -run TestFiles/math_big0 -test.short`: ok.
- The broader `TestFiles -test.short` failures (`redeclaration3`, `switch13`, `type41`, `types/*_f0`) reproduce identically on clean `e46de6fc4` — pre-existing `go/types` error-message drift, unrelated to this PR.

## Questions for Author
- Is including stdlib `_test.gno` files in the persisted/consensus package set intended (Warning 2)? If real genesis excludes them, the `crossrealm38` pin tests test-store content, not production.
- The gas follow-up needs a quadratic term (or input cap) for `mul`/`setString`/`text` — the linear `Slope2` I added closes the asymmetric escape but not the super-linear growth. Confirm that belongs in the calibration PR.
