# Review: PR [#5880](https://github.com/gnolang/gno/pull/5880)
Event: APPROVE

## Body
The seven checks distill `gno-security-guide.md` §5; I checked each WRONG/RIGHT pair against that guide and the VM, and every cited symbol resolves. Repro ran on 26ca914e289cdd917c0c19909bd40d6edd35ce49.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5880-ai-contract-review-guide/1-26ca914e2/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/resources/gno-ai-contract-review.md:74-77 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L74)
The `panics at attach time` comment sits on a bare `var savedRealm realm`, which never panics on its own. The panic fires only when a live realm value is assigned, and at transaction finalize, not attach. Move the comment onto an assignment line and say finalize.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5880 -R gnolang/gno
# A: the doc's exact bare declaration — expect a clean run, no panic
cat > gnovm/tests/files/zz_bare.gno <<'EOF'
// PKGPATH: gno.land/r/zzbare
package zzbare
var savedRealm realm
func main(cur realm) { println("ran clean, savedRealm is zero value") }
// Output:
// ran clean, savedRealm is zero value
EOF
# B: store a live realm — expect finalize panic after the assignment runs
cat > gnovm/tests/files/zz_store.gno <<'EOF'
// PKGPATH: gno.land/r/zzstore
package zzstore
var savedRealm realm
func main(cur realm) { savedRealm = cur; println("assignment ok, awaiting finalize panic") }
// Error:
// cannot persist realm value: realm values are ephemeral and tied to a call frame
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_bare.gno$|TestFiles/zz_store.gno$' -test.short
rm gnovm/tests/files/zz_bare.gno gnovm/tests/files/zz_store.gno
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang
```
</details>

## docs/resources/gno-ai-contract-review.md:113 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L113)
The relationship table lists `misc/audit-pattern-harness/`, which is not on master. It ships in the still-open #5835. If this merges first, the row points at a path that does not exist yet.

## docs/resources/gno-ai-contract-review.md:80-82 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L80)
Check 1 makes `if !cur.IsCurrent()` the standard before reading caller identity, but this `Save` example reads `cur.Previous().Address()` without it. A reader copying it gets an unguarded caller read that contradicts Check 1. Either add the guard here or note in Check 1 that it applies only when the realm is caller-passed, not the live `cur`.
</content>
