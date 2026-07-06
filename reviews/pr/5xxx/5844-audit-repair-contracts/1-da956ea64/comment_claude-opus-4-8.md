# Review: PR [#5844](https://github.com/gnolang/gno/pull/5844)
Event: COMMENT

## Body
Reproduced on da956ea64: reverting the repaired `current-guard/fixed/admin.gno` to a hit-free but non-compiling body keeps TestRepairContracts green. The "target removes the pattern" check passes without the target compiling. Flagged inline where to close that.

This stacks on #5835, so the diff against master still carries the whole harness stack. Only the top two commits are this PR. Rebasing once #5835 lands collapses it to the repair support plus the CI line.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5844-audit-repair-contracts/1-da956ea64/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:356-362 [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356)
The repair target is checked only through RunRule, a text scan, so a repaired fixture passes the contract even when it does not compile. The gno-compile check in TestAgentPatternContractWithGNO self-skips without a gno binary, and the misc CI job installs none, so nothing in CI compiles the repair targets. Run gno test on the target when a gno binary is available, or note the contract is text-only.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5844 -R gnolang/gno
f=misc/audit-pattern-harness/fixtures/current-guard/fixed/admin.gno
cp "$f" /tmp/admin.gno.bak
cat > "$f" <<'EOF'
package admin

var owner = address("g1qz8e0fz3y0pl9y4dq9d7c5dwnyu6qf04hs7z0a")

func TransferOwnership(cur realm, next address) {
	owner = thisDoesNotExist
}

func Owner() address {
	return owner
}
EOF
go test -run 'TestRepairContracts/current-guard' ./misc/audit-pattern-harness/internal/auditpattern/
cp /tmp/admin.gno.bak "$f"
```

```
ok  	github.com/gnolang/gno/misc/audit-pattern-harness/internal/auditpattern	0.008s
```
</details>
