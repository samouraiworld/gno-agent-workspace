# Review: PR [#5892](https://github.com/gnolang/gno/pull/5892)
Event: APPROVE

## Body
Looks good. Verified on d2f3d1337: commenting out the AddPackage [`chargePreprocessGas`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/keeper.go#L590) [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L590) call drops usea in [`addpkg_import_testdep_gas.txtar`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/integration/testdata/addpkg_import_testdep_gas.txtar) [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/integration/testdata/addpkg_import_testdep_gas.txtar) from 3218401 back to 3113401, its pre-charge baseline, so the charge equals the per-byte formula.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5892-meter-preprocess-gas/1-d2f3d1337/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
