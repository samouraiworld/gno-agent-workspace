# Review: PR [#5892](https://github.com/gnolang/gno/pull/5892)
Event: APPROVE

## Body
Verified on 03ab3eea2: each re-pinned fixture's gas delta is exactly its source-byte count times 1250, re-derived against the post-#5891 baseline, so the charge matches the formula it documents.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5892-meter-preprocess-gas/2-03ab3eea2/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/machine.go:330-340 [↗](../../../../../.worktrees/gno-review-5892/gnovm/pkg/gnolang/machine.go#L330)
This guard is unreachable: [`IterMemPackage`](https://github.com/gnolang/gno/blob/03ab3eea2/gnovm/pkg/gnolang/store.go#L1243) · [↗](../../../../../.worktrees/gno-review-5892/gnovm/pkg/gnolang/store.go#L1243) already skips nil at [`store.go:1263-1269`](https://github.com/gnolang/gno/blob/03ab3eea2/gnovm/pkg/gnolang/store.go#L1263-L1269) · [↗](../../../../../.worktrees/gno-review-5892/gnovm/pkg/gnolang/store.go#L1263) before its only channel send, and `defaultStore` is the only type implementing [the interface method](https://github.com/gnolang/gno/blob/03ab3eea2/gnovm/pkg/gnolang/store.go#L92) · [↗](../../../../../.worktrees/gno-review-5892/gnovm/pkg/gnolang/store.go#L92). It looks like a merge artifact: the hunk comes from #5891's un-squashed branch commit, while the squash on master put the skip in `store.go` and left `machine.go` untouched, so the merge kept both. The comment's premise holds for `GetMemPackage` but not for the channel the guard sits on.

## gno.land/pkg/sdk/vm/keeper.go:582-598 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L582)
The charge sums `mpkg.Files` only, so imported dependency source stays priced at store-read rates while the comment reads as though the vector is closed. Dependencies are re-type-checked every transaction: [`keeper.go:403`](https://github.com/gnolang/gno/blob/03ab3eea2/gno.land/pkg/sdk/vm/keeper.go#L403) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L403) clones the type-check cache per tx and discards it at tx end, and the backing cache is only ever filled at init and stdlib load, so a small package importing 1 MB of dependencies imposes roughly 1.25 s of validator type-check CPU for about 20M gas. The PR body names this as deferred, so the ask is narrow: say in the comment that the charge prices the submitted package only.

## gno.land/pkg/sdk/vm/keeper.go:1081 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L1081)
Missing test: the `Run` charge is asserted nowhere. Deleting this line leaves `go test ./gno.land/pkg/sdk/vm/` green along with every gas fixture, because [`maketx_run.txtar:13`](https://github.com/gnolang/gno/blob/03ab3eea2/gno.land/pkg/integration/testdata/maketx_run.txtar#L13) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/integration/testdata/maketx_run.txtar#L13) pins `GAS USED: ` with no value.

<details><summary>test cases</summary>

Full file: [`tests/preprocess_gas_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5892-meter-preprocess-gas/2-03ab3eea2/tests/preprocess_gas_test.go) · [↗](tests/preprocess_gas_test.go). `TestRunPreprocessGasCharged` passes at 03ab3eea2 and fails with the charge removed:

```
--- FAIL: TestRunPreprocessGasCharged
    expected RunPreprocess gas 68750, got 0
```
</details>

## contribs/gnogenesis/internal/fork/generate.go:484-486 [↗](../../../../../.worktrees/gno-review-5892/contribs/gnogenesis/internal/fork/generate.go#L484)
Missing test: nothing pins this fill, so re-nesting it inside the fingerprint branch reintroduces the bug it just fixed and the whole `internal/fork` package stays green. The two tests that call `Validate()` take the fingerprint-match path, and both fingerprint-miss tests ([`generate_test.go:128`](https://github.com/gnolang/gno/blob/03ab3eea2/contribs/gnogenesis/internal/fork/generate_test.go#L128) · [↗](../../../../../.worktrees/gno-review-5892/contribs/gnogenesis/internal/fork/generate_test.go#L128) and [`:276`](https://github.com/gnolang/gno/blob/03ab3eea2/contribs/gnogenesis/internal/fork/generate_test.go#L276) · [↗](../../../../../.worktrees/gno-review-5892/contribs/gnogenesis/internal/fork/generate_test.go#L276)) never call `Validate()` nor read the field.

<details><summary>test cases</summary>

Full file: [`tests/fork_preprocess_fill_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5892-meter-preprocess-gas/2-03ab3eea2/tests/fork_preprocess_fill_test.go) · [↗](tests/fork_preprocess_fill_test.go). `TestBuildHardforkGenesisFillsPreprocessGasPerByteWhenTuned` passes at 03ab3eea2 and fails with the fill re-nested:

```
--- FAIL: TestBuildHardforkGenesisFillsPreprocessGasPerByteWhenTuned
    expected PreprocessGasPerByte 1250, got 0
```
</details>

## gno.land/pkg/sdk/vm/keeper.go:590-598 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L590)
Missing test: that `_test`/`_filetest` bytes are charged and non-`.gno` files are not is a consensus rule no test constrains, so narrowing the byte base to prod-only leaves every gas pin green. The one fixture carrying a `_test.gno` does not pin the gas of the deploy containing it, and the pinned deploys have no test files.

<details><summary>test cases</summary>

`TestChargePreprocessGasByteBase` in [`tests/preprocess_gas_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5892-meter-preprocess-gas/2-03ab3eea2/tests/preprocess_gas_test.go) · [↗](tests/preprocess_gas_test.go), passing at 03ab3eea2.
</details>

## gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:93-95 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L93)
Nit: the bump is not caused by the default being non-zero. Params are stored one JSON KV per struct field with no zero-skipping, since [`encodeStructFields`](https://github.com/gnolang/gno/blob/03ab3eea2/tm2/pkg/sdk/params/amino_helper.go#L13-L27) · [↗](../../../../../.worktrees/gno-review-5892/tm2/pkg/sdk/params/amino_helper.go#L13) writes every field unconditionally, so a zero default would add the key and move the root too. Forcing the default to 0 gives `e1169b5ed6d0…`, not the pre-PR pin `d576406059fa…`; the pinned value itself is correct.

## gno.land/pkg/sdk/vm/genesis.go:52 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/genesis.go#L52)
Nit: defaulting before `ValidateGenesis` means its [empty-state check](https://github.com/gnolang/gno/blob/03ab3eea2/gno.land/pkg/sdk/vm/genesis.go#L33-L35) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/genesis.go#L33) can no longer fire from `InitGenesis`, because the defaulted struct is never `DeepEqual` to the zero value. `InitGenesis(ctx, GenesisState{})` now panics "invalid default storage deposit" instead of "vm genesis state cannot be empty". The node refuses to boot either way, so only the diagnostic degrades.

## gno.land/pkg/integration/testdata/addpkg_import_testdep_gas.txtar:15-18 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/integration/testdata/addpkg_import_testdep_gas.txtar#L15)
Nit: "On master (pre-split, pre-charge) usea == 3172364" names a baseline that no longer exists. With #5891 squashed, master yields 2691401, so a reader computing branch-vs-master sees a 375,963 gas decrease despite a +105,000 charge. The equality guard below is unaffected.

## gno.land/pkg/integration/testdata/interrealm_final.txtar:22 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/integration/testdata/interrealm_final.txtar#L22)
Nit: this deploy now uses 17,074,820 of 18,000,000 gas-wanted, 5.1% headroom, down from about 29%. It is the tightest unintentional margin in the suite, so any later change adding more than 5% to it reddens the fixture.

## gno.land/pkg/sdk/vm/params_test.go:339-348 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/params_test.go#L339)
Nit: this pins `IterNextCostFlat` and the three depth defaults but not `PreprocessGasPerByte`. The new default is pinned only indirectly by the app-hash test, which reports it as an opaque hash mismatch.

## gno.land/pkg/sdk/vm/params.go:76-78 [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/params.go#L76)
Suggestion: the stated reason for rejecting zero does not hold on its own, since governance can set 1 and disable the protection just as effectively while passing `Validate`. What the bound really guarantees is that the legacy zero-sentinel stays unambiguous, which [`params.go:238-245`](https://github.com/gnolang/gno/blob/03ab3eea2/gno.land/pkg/sdk/vm/params.go#L238-L245) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/params.go#L238) already says.
