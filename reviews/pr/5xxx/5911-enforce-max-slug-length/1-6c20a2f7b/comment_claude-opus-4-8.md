# Review: PR [#5911](https://github.com/gnolang/gno/pull/5911)
Event: APPROVE

## Body
Looks good. Verified on 6c20a2f7b: removing the length-check panic makes [`TestValidateSlugPanicsOnTooLong`](https://github.com/gnolang/gno/blob/6c20a2f7b/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L64-L68) [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L64) fail with `should have panicked`, so the new test guards the check.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5911-enforce-max-slug-length/1-6c20a2f7b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:88 [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L88)
The 128 cap is measured in bytes via `len(slug)`, but the constant has no unit comment. The issue's proposal included `// bytes`.
</content>
