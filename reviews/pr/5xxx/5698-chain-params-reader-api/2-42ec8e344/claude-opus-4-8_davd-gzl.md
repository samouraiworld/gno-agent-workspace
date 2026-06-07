# PR #5698: feat(gnovm): add chain params reader API

URL: https://github.com/gnolang/gno/pull/5698
Author: notJoon | Base: master | Files: 8 | +454 -6
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `42ec8e344` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5698 42ec8e344`

Round 2 of 2. Round 1 reviewed `0b5487b`; this round covers the +16 commits since (mostly `master` merges plus one PR-specific repin commit). Related to #5699 (raw byte-key `chain/params` API, still open).

**Verdict: REQUEST CHANGES.** Round-1 CI failure is fixed and benchmarks now exist, but the cross-type read divergence is unresolved and now actively mis-documented: `SetBytes` then `GetString` returns `(", false)` under `gno test` yet panics on chain, and the new test-stub comment falsely claims parity. Gas table is still hand-copied from `Set*`/`sys/params`, not fit from the new benchmarks (defensible but undocumented).

## Summary
Exposes the six `Get*` readers (`GetString`, `GetBool`, `GetInt64`, `GetUint64`, `GetBytes`, `GetStrings`) of `ParamsInterface` to realms via `chain/params`, completing the round-trip with the existing `Set*` writers. Six 1:1 Go wrappers zero-init a typed pointer, delegate to `ParamsInterface.GetXxx(pkey, &out)`, and return `(value, ok)`. The headline risk is unchanged from round 1: the production keeper's read path (`getIfExists` → `amino.MustUnmarshalJSON`) panics on a type/format mismatch, while the in-memory test stub returns `false`, so a realm that reads a key under the wrong type passes `gno test` and panics on chain.

## What changed since round 1 (`0b5487b` → `42ec8e344`)
- **Round-1 Warning 1 (CI red): RESOLVED.** `expectedCrossrealm38Hash` repinned to `3b2fdccd…85110` in [`apphash_crossrealm38_test.go:52`](https://github.com/gnolang/gno/blob/42ec8e344/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52) · [↗](../../../../../.worktrees/gno-review-5698/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52). `TestAppHashCrossrealm38` now passes locally. CI is green except the bot "Merge Requirements" gate, which only wants reviewer approval, not a real failure.
- **Round-1 Warning 2 (gas not fit): PARTIALLY ADDRESSED.** The 13 `Get*` benchmarks now exist ([`native_machine_bench_test.go:395-496`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/cmd/calibrate/native_machine_bench_test.go#L395-L496) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/cmd/calibrate/native_machine_bench_test.go#L395)) and the calibrator gained `Get*` spec rows ([`gen_native_table.py:94-186`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/cmd/calibrate/gen_native_table.py#L94) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/cmd/calibrate/gen_native_table.py#L94)). But [`native_gas.go:101-129`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/stdlibs/native_gas.go#L101-L129) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/stdlibs/native_gas.go#L101) values are still verbatim copies. Every comment reads "mirrors … cost", none reads `fit base=… R²=…`. See Warning below.
- **Round-1 Missing Test (cross-type read): NOT ADDRESSED, and now mis-documented.** `std12.gno` expanded with happy-path + missing-key cases ([`std12.gno:10-56`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/tests/files/std12.gno#L10-L56) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/tests/files/std12.gno#L10)) but still no cross-type case. The test stub gained a comment claiming the wrong-type path mirrors production, but it does not. See Critical below.
- Unrelated `master` churn folded in: `chain/runtime/unsafe` gas entries and the removal of `BenchmarkNative_Banker_AssertCallerIsRealm` are from `master` (`git show origin/master:gnovm/stdlibs/native_gas.go` already has the unsafe rows), not this PR.

## Glossary
- `pkey`: `"vm:" + rlmPath + ":" + key`; rejects empty keys and any `:` in `key`.
- `getIfExists`: keeper read helper: `stor.Get` then `amino.MustUnmarshalJSON(bz, ptr)`; panics on malformed/wrong-type payload.
- `testParams` / mock stub: in-memory `ParamsInterface` used by `gno test`; stores typed Go values in a `map[string]any`, type-asserts on read.

## Critical (must fix)

- **[passes `gno test`, panics on chain, and the stub comment says otherwise]** [`gnovm/pkg/test/test.go:160-162`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/pkg/test/test.go#L160-L162) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/pkg/test/test.go#L160) : cross-type read (`SetBytes` then `GetString`) returns `(", false)` in tests but panics in `amino.MustUnmarshalJSON` on chain; the new comment claims parity that does not exist.
  <details><summary>details</summary>

  **Shape:** `SetBytes("k", {0,1,2,255})` then `GetString("k")`.
  **What you see:** Production `ParamsKeeper.SetBytes` stores raw bytes with no JSON encoding ([`keeper.go:171-198`](https://github.com/gnolang/gno/blob/42ec8e344/tm2/pkg/sdk/params/keeper.go#L171-L198) · [↗](../../../../../.worktrees/gno-review-5698/tm2/pkg/sdk/params/keeper.go#L171)). `GetString` → `getIfExists` → `amino.MustUnmarshalJSON(bz, *string)` ([`keeper.go:269-278`](https://github.com/gnolang/gno/blob/42ec8e344/tm2/pkg/sdk/params/keeper.go#L269-L278) · [↗](../../../../../.worktrees/gno-review-5698/tm2/pkg/sdk/params/keeper.go#L269)) on the raw, non-JSON `{0,1,2,255}` payload → panic. The in-memory stub instead does a typed map lookup and returns `false` ([`test.go:200-208`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/pkg/test/test.go#L200-L208) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/pkg/test/test.go#L200)). The reverse asymmetry exists too: `SetString` JSON-encodes (`set` → `amino.MustMarshalJSON`, [`keeper.go:302-315`](https://github.com/gnolang/gno/blob/42ec8e344/tm2/pkg/sdk/params/keeper.go#L302-L315) · [↗](../../../../../.worktrees/gno-review-5698/tm2/pkg/sdk/params/keeper.go#L302)), so `SetString("k","hi")` then `GetBytes("k")` returns the JSON-quoted bytes `"hi"` silently (wrong value, no error), while `GetInt64("k")` panics on the unmarshal.
  **Why it matters:** The comment added at [`test.go:160-162`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/pkg/test/test.go#L160-L162) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/pkg/test/test.go#L160) states the stub returns false on wrong-type "same fail-safe shape as the production keeper's amino-unmarshal would surface". The production keeper does not fail safe; it panics. A realm author who reads back under the wrong type (easy to do, since the key-to-type binding is implicit and untyped) ships green and aborts the tx on chain. This is the same gap round 1 flagged as Missing Tests; the PR added a comment that makes it worse by asserting the divergence away.
  **Fix:** Pick one and make stub + keeper + doc agree. Either (a) make the stub mirror reality: panic on a present-but-wrong-type value so tests catch it; or (b) make the keeper fail safe: `getIfExists` returns `false` on unmarshal error instead of `MustUnmarshalJSON` (changes consensus surface, needs gating); or (c) at minimum, correct the comment and document in [`params.gno:14-16`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/stdlibs/chain/params/params.gno#L14-L16) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/stdlibs/chain/params/params.gno#L14) that reading under a mismatched type panics on chain, and add a cross-type filetest. Adversarial repro: [`tests/std12_crosstype.gno`](tests/std12_crosstype.gno) (passes under `gno test`, demonstrating the false-negative).
  </details>

## Warnings (should fix)

- **[gas values copied, not fit, comments hide it]** [`gnovm/stdlibs/native_gas.go:101-129`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/stdlibs/native_gas.go#L101-L129) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/stdlibs/native_gas.go#L101) : `Get*` entries reuse `Set*` bases and `sys/params` return slopes; the new benchmarks were added but never regressed into the table.
  <details><summary>details</summary>

  The benchmarks now exist and the calibrator has `Get*` specs, so the infrastructure is in place, but the table was not regenerated from them. Every `Get*` base equals the matching `Set*` base (`GetString`=1772=`SetString`, `GetBool`=1643, `GetInt64`=1201, `GetUint64`=1219, `GetBytes`=1912), and the post-slopes on `GetBytes` (10584) / `GetStrings` (23215) are copied from `sys/params` `getSysParamBytes`/`getSysParamStrings`. The comments say "mirrors … cost" rather than the `fit base=… slope=… R²=…` shape on every other row, which is the tell. Running the new benchmarks (`go test -bench='BenchmarkNative_Params_Get' ./gnovm/cmd/calibrate`) shows the shapes are roughly right (`GetString` is flat across 1…1000 with no post-slope, correct; `GetBytes` scales with return size with post-slope present, correct), so the mirroring is defensible and likely conservative, not a correctness bug. But this is the consensus-bound price list; copying without a fit means the numbers drift from the benchmarks the PR itself ships, and a future calibrator run will silently overwrite them. Fix: run `gen_native_table.py` over the full bench output and paste the fits (with `R²`), or add one line to the PR description stating the copy is intentional and conservative and why a fit was skipped.
  </details>

- **[GetBytes returns aliased storage slice]** [`tm2/pkg/sdk/params/keeper.go:140-149`](https://github.com/gnolang/gno/blob/42ec8e344/tm2/pkg/sdk/params/keeper.go#L140-L149) · [↗](../../../../../.worktrees/gno-review-5698/tm2/pkg/sdk/params/keeper.go#L140) : `GetBytes` does `*ptr = bz` with no copy, then the wrapper hands that slice to the realm; `SetBytes` copies defensively but `GetBytes` does not.
  <details><summary>details</summary>

  Production `GetBytes` assigns `stor.Get(...)`'s `bz` straight into the caller pointer, and [`params.go:80-85`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/stdlibs/chain/params/params.go#L80-L85) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/stdlibs/chain/params/params.go#L80) returns it to the realm. `SetBytes` copies on write ([`keeper.go:192-194`](https://github.com/gnolang/gno/blob/42ec8e344/tm2/pkg/sdk/params/keeper.go#L192-L194) · [↗](../../../../../.worktrees/gno-review-5698/tm2/pkg/sdk/params/keeper.go#L192)) precisely to avoid aliasing the input; the read side has no symmetric defense. Whether the realm can mutate backing store memory depends on the iavl/store layer's `Get` contract (it likely returns a fresh slice today), but relying on backend-defined semantics for a value that crosses the VM boundary is fragile. Other `GetXxx` go through `amino.MustUnmarshalJSON`, which always allocates fresh, so only `GetBytes` is exposed. This was a round-1 suggestion; promoting to a warning because the new readers make it reachable from realm code. Fix: copy in `GetBytes` (keeper or wrapper), or confirm in the PR description that the store `Get` contract guarantees a fresh slice.
  </details>

## Nits

- [`gnovm/stdlibs/chain/params/params.go:50-92`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/stdlibs/chain/params/params.go#L50-L92) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/stdlibs/chain/params/params.go#L50) : only `GetString` carries a doc comment; the other five wrappers are bare. The `params.gno` doc comment ([`params.gno:14-16`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/stdlibs/chain/params/params.gno#L14-L16) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/stdlibs/chain/params/params.gno#L14)) is the realm-facing contract and is where the cross-type panic caveat belongs (see Critical).

## Missing Tests

- **[type-mismatch read]** [`gnovm/tests/files/std12.gno`](https://github.com/gnolang/gno/blob/42ec8e344/gnovm/tests/files/std12.gno) · [↗](../../../../../.worktrees/gno-review-5698/gnovm/tests/files/std12.gno) : no case exercises `Set`-then-`Get` under a mismatched type, the one path where test and chain diverge.
  <details><summary>details</summary>

  The expanded `std12.gno` is thorough on happy-path and missing-key, but every read uses the same type it was written with, so it can never surface the panic-vs-false divergence. A cross-type case ([`tests/std12_crosstype.gno`](tests/std12_crosstype.gno)) passes under `gno test` today precisely because the stub returns `false` instead of panicking, and that green is the bug. Add the case once the stub/keeper behavior is reconciled (Critical), so the test asserts whatever the agreed-upon observable is.
  </details>

## Suggestions

None beyond the above.

## Questions for Author

- Cross-type reads: is the intended on-chain contract "panic" or "return (zero,false)"? The test stub and keeper currently disagree, and #5699 (raw byte-key API) will widen this surface; worth settling the semantics across both PRs before either merges.
- Was the `Get*` gas table copy from `Set*`/`sys/params` intentional and conservative, or are the fits pending? If intentional, please note it in the PR description so a future calibrator run doesn't silently overwrite hand-tuned values.
