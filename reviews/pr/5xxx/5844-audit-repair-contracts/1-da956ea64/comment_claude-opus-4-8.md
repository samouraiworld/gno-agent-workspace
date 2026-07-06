# Review: PR [#5844](https://github.com/gnolang/gno/pull/5844)
Event: COMMENT

## Body
Reproduced on da956ea64. The repair contract gates on a text-scan heuristic (RunRule hit counts) plus a raw-byte regex over top-level function names, so a green run proves the flagged text pattern is gone and the top-level function names are unchanged, not that the fixture was repaired.

This stacks on #5835, so the diff against master still carries the whole harness stack. Only the top two commits are this PR. Rebasing once #5835 lands collapses it to the repair support plus the CI line.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5844-audit-repair-contracts/1-da956ea64/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:339-378 [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L339)
A target that deletes the guarded code path, or one still exploitable but no longer tripping the heuristic scan, passes every gate. `goal` is only checked non-empty. The green fixture then reads as a validated repair an agent can learn from when it may be gutted or still vulnerable.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5844 -R gnolang/gno
cd misc/audit-pattern-harness
f=fixtures/current-guard/fixed/admin.gno
cp "$f" /tmp/admin.bak
# "fixed" target with the authorization deleted entirely (gutted, not guarded)
cat > "$f" <<'EOF'
package admin

var owner = address("g1qz8e0fz3y0pl9y4dq9d7c5dwnyu6qf04hs7z0a")

func TransferOwnership(cur realm, next address) {
	// authorization removed entirely
	owner = next
}

func Owner() address {
	return owner
}
EOF
go test -count=1 -run 'TestRepairContracts/current-guard' ./internal/auditpattern/
cp /tmp/admin.bak "$f"
```

```
ok  	github.com/gnolang/gno/misc/audit-pattern-harness/internal/auditpattern	0.008s
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:487-500 [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L487)
`exportedFuncNames` runs the `^func` regex over raw file bytes, so a flush-left `func` in a block comment or raw string literal counts as an exported function. This both masks a real API removal and rejects valid repairs whose fixture carries a doc example. Extract the names from parsed source, not raw text.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5844 -R gnolang/gno
cd misc/audit-pattern-harness
# false pass: delete the real func Owner(), hide a flush-left "func Owner(" in a comment
f=fixtures/current-guard/fixed/admin.gno
cp "$f" /tmp/admin.bak
cat > "$f" <<'EOF'
package admin

var owner = address("g1qz8e0fz3y0pl9y4dq9d7c5dwnyu6qf04hs7z0a")

func TransferOwnership(cur realm, next address) {
	if !cur.IsCurrent() {
		panic("spoofed realm")
	}
	if cur.Previous().Address() != owner {
		panic("owner only")
	}
	owner = next
}

/*
func Owner() address { return owner }
*/
EOF
go test -count=1 -run 'TestRepairContracts/current-guard' ./internal/auditpattern/
cp /tmp/admin.bak "$f"
# false fail: a valid render-markdown repair carrying a doc example is rejected
g=fixtures/render-markdown/fixed/echo.gno
cp "$g" /tmp/echo.bak
cat > "$g" <<'EOF'
package echo

import "gno.land/p/moul/md"

var usage = `
func Example() string {
	return Render("x")
}
`

func Render(path string) string {
	return "# Echo\n\n" + md.EscapeText(path)
}
EOF
go test -count=1 -run 'TestRepairContracts/render-markdown' ./internal/auditpattern/
cp /tmp/echo.bak "$g"
```

```
ok  	github.com/gnolang/gno/misc/audit-pattern-harness/internal/auditpattern	0.006s
--- FAIL: TestRepairContracts/render-markdown
    run_test.go:377: repair target should preserve exported top-level function names: from=[Render] to=[Example Render]
FAIL
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/record.go:90-98 [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/record.go#L90)
`validate()` rejects any record whose repair block is missing `from_fixture`, `to_fixture`, or `goal`, and it runs in the shared `LoadRecord` path. So a record with no repair block fails the pre-existing pattern contract and the CLI, not just the repair test, though the README calls the block experimental. Make repair optional-when-present, or drop the experimental framing and document it as required.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5844 -R gnolang/gno
cd misc/audit-pattern-harness
y=expected/current-guard.yaml
cp "$y" /tmp/y.bak
# drop the repair block from a record; the pre-existing pattern contract still reds
sed -i '/^repair:/,/^  goal:/d' "$y"
go test -count=1 -run 'TestAgentPatternContract$' ./internal/auditpattern/
cp /tmp/y.bak "$y"
```

```
--- FAIL: TestAgentPatternContract (0.00s)
    run_test.go:261: .../expected/current-guard.yaml: repair: missing from_fixture
FAIL
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:356-362 [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356)
The target is checked only through `RunRule`, a text scan, so a repaired fixture passes even when it does not compile, and the misc CI job installs no gno toolchain to catch it. Build the target inside the contract with `gno lint`, not `gno test`: a package with no `_test.gno` file makes `gno test` exit 0 without type-checking. Fail rather than skip when no gno binary is present, so a green run means the targets compiled.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5844 -R gnolang/gno
cd misc/audit-pattern-harness
f=fixtures/current-guard/fixed/admin.gno
cp "$f" /tmp/admin.bak
# hit-free but references an undeclared identifier: never compiled, still green
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
go test -count=1 -run 'TestRepairContracts/current-guard' ./internal/auditpattern/
cp /tmp/admin.bak "$f"
```

```
ok  	github.com/gnolang/gno/misc/audit-pattern-harness/internal/auditpattern	0.008s
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:373-378 [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L373)
The export-stability gate compares only top-level `func` names, so it misses exported var/const/type removals and same-name signature changes. `exported-pointer-leak` has no `allow_removed_exports` yet its repair drops the exported `var PublicVault` and changes `GetVault`'s return type from `*Vault` to `Vault`, both unseen. Decide whether the gate should cover exported vars and signatures, or scope the README claim to top-level function names.
