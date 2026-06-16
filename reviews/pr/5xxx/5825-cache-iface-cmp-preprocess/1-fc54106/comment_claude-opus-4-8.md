# Review: PR #5825
Event: APPROVE

## Body
Looks good. Verified on fc54106: removing the line that sets `ATTR_IFACE_CMP` on the `BinaryExpr` makes `TestBinaryExprIfaceCmp_SurvivesColdReload` fail, so the cold-reload test genuinely guards the re-preprocess-on-restart assumption. The one case the new code could have regressed also holds: it reads the operand types before `checkOrConvertType` rewrites them, so a call operand returning an uncomparable interface (`get() == get()`) still panics while `interface == nil` still returns false, matching pre-PR behavior.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5825-cache-iface-cmp-preprocess/1-fc54106/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/benchdata/cmp_iface.gno:4-5 [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/benchdata/cmp_iface.gno#L4)
The comment names `isInterfaceCmp` and `OpEqlIface`; neither exists after this PR (`isInterfaceCmp` is the helper this PR removes, `OpEqlIface` is the abandoned opcode design). Name the live `ATTR_IFACE_CMP` / `isEql` path instead.

*(AI Agent)*
