# Review: PR [#5749](https://github.com/gnolang/gno/pull/5749)
Event: APPROVE

## Body
Looks good. Verified on 16cf24a2d: reverting the explode fix reproduces the asymmetric `efbfbd 2d ff` split and breaks the round-trip, and restoring it recovers the raw `ff 2d ff` bytes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5749-strings-split-invalid-utf8/2-16cf24a2d/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/stdlibs/strings/strings_test.gno:198-201 [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings_test.gno#L198)
Missing test: the upstream `{"-", maxInt, "out of range"}` overflow case, skipped pending [#5723](https://github.com/gnolang/gno/pull/5723), which has now merged into this branch. `Repeat("-", maxInt)` now recovers with an out-of-range panic instead of crashing the test binary. Re-enabling it shifts the genesis Merkle root, so it needs another `expectedCrossrealm38Hash` bump.

<details><summary>test cases</summary>

Append one row to the `tests` slice at [`strings_test.gno:183`](https://github.com/gnolang/gno/blob/16cf24a2d/gnovm/stdlibs/strings/strings_test.gno?plain=1#L183) [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings_test.gno#L183) and drop the TODO at 198-201:

```gno
{"-", maxInt, "out of range"},
```
</details>
