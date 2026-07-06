# Review: PR [#5844](https://github.com/gnolang/gno/pull/5844)
Event: COMMENT

## Body
Reproduced on da956ea64: reverting the repaired `current-guard/fixed/admin.gno` to a hit-free but non-compiling body keeps TestRepairContracts green. The "target removes the pattern" check passes without the target compiling. Inline is a 32-line fix for that deferred compile-check: build the target with `gno lint` inside the contract, failing loudly when no gno binary is present.

This stacks on #5835, so the diff against master still carries the whole harness stack. Only the top two commits are this PR. Rebasing once #5835 lands collapses it to the repair support plus the CI line.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5844-audit-repair-contracts/1-da956ea64/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:356-362 [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356)
This is the "verify those outputs compile" step the PR body defers. The repair target is checked only through RunRule, a text scan, so a repaired fixture passes the contract even when it does not compile, and the misc CI job installs no gno toolchain so nothing there catches it. A 32-line fix that closes it: build the target inside TestRepairContracts with `gno lint`, not `gno test` (repair targets have no _test.gno file, so `gno test` prints `[no test files]` and exits 0 without type-checking), and fail the test rather than skip when no gno binary is present, so a green run means the targets actually compiled. TestAgentPatternContractWithGNO keeps its opt-in skip. Verified on da956ea64: the assertion passes on all 8 fixtures; the repro below turns current-guard red once the fix is in and green again on restore.

<details><summary>repro (gap on da956ea64)</summary>

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
( cd misc/audit-pattern-harness && go test -run 'TestRepairContracts/current-guard' ./internal/auditpattern/ )
cp /tmp/admin.gno.bak "$f"
```

```
ok  	github.com/gnolang/gno/misc/audit-pattern-harness/internal/auditpattern	0.008s
```
</details>
