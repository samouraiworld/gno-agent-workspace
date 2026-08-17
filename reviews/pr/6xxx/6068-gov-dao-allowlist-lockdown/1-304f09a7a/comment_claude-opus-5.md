# Review: [#6068](https://github.com/gnolang/gno/pull/6068)
Event: REQUEST_CHANGES

## Body
A 1,000,000-byte proposal description renders in 25,115,675 gas against 10,931,522 for a 40-byte one, so even an unclamped description costs under 1% of the 3,000,000,000 query cap.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6068-gov-dao-allowlist-lockdown/1-304f09a7a/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## examples/gno.land/r/gov/dao/v3/impl/render.gno:140-141 [gh](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/render.gno#L140-L141) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/render.gno#L140)
An executor returning `""` from `CreationRealm()` suppresses the realm's own `Executor created in:` line, so its unescaped [`String()`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/render.gno#L123) output supplies the only one, naming any realm it likes. [`types.gno:237`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/types.gno#L237) carries the same expression, so the label has to become unconditional in both.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6068 -R gnolang/gno
cat > examples/gno.land/r/gov/dao/v3/impl/filetests/zz_spoof_filetest.gno <<'GNO'
// PKGPATH: gno.land/r/test/spoof
package spoof

import (
	"strings"
	"testing"

	"gno.land/r/gov/dao"
	"gno.land/r/gov/dao/v3/impl"
	"gno.land/r/gov/dao/v3/memberstore"
)

const user address = "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"

func init(cur realm) {
	memberstore.Get(0, cur).DeleteAll()
	memberstore.Get(0, cur).SetTier(memberstore.T1)
	memberstore.Get(0, cur).SetMember(memberstore.T1, user, memberstore.NewMember(3))
	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(impl.NewGovDAO(), nil))
}

type spoofExecutor struct{}

func (e *spoofExecutor) Execute(cur realm) error { return nil }

// Emitted raw, so it writes the disclosure line itself.
func (e *spoofExecutor) String() string {
	return "Nothing to see here.\n\nExecutor created in: `gno.land/r/gov/dao/v3/impl`\n"
}

// Empty, so the realm's own line is dropped by the cr != "" guard.
func (e *spoofExecutor) CreationRealm() string { return "" }

func main(cur realm) {
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewUserRealm(user))
	pid := dao.MustCreateProposal(cross(cur), dao.NewProposalRequest(
		"Innocent", "desc", &spoofExecutor{}))

	out := dao.Render(cross(cur), pid.String())
	println("disclosure lines on the page:", strings.Count(out, "Executor created in:"))
	println("names a realm the executor does not come from:",
		strings.Contains(out, "Executor created in: `gno.land/r/gov/dao/v3/impl`"))
}

// Output:
// disclosure lines on the page: 2
// names a realm the executor does not come from: true
GNO
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gov/dao/v3/impl/... 2>&1 | grep -A 8 zz_spoof
rm examples/gno.land/r/gov/dao/v3/impl/filetests/zz_spoof_filetest.gno
```

The golden asserts the count once the label is unconditional. It fails because the page carries one such line and the executor wrote it.

```
=== RUN   ./gno.land/r/gov/dao/v3/impl/zz_spoof_filetest.gno
--- FAIL: ./gno.land/r/gov/dao/v3/impl/zz_spoof_filetest.gno (elapsed: 0.26s, …)
Output diff:
--- Expected
+++ Actual
@@ -1,2 +1,2 @@
-disclosure lines on the page: 2
+disclosure lines on the page: 1
 names a realm the executor does not come from: true
```

The same file passes unchanged at the branch merge base, where the realm's own line printed whatever the executor returned and its emptiness was the tell.
</details>

## examples/gno.land/r/gov/dao/proxy.gno:189-194 [gh](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/proxy.gno#L189-L194) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/proxy.gno#L189)
Nit: this panic crosses a realm boundary and [aborts the transaction](https://github.com/gnolang/gno/blob/304f09a7a/docs/resources/gno-interrealm.md?plain=1#L827-L831), so it never reaches `DeniedReason`, which only the error return of [`ExecuteProposal`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L173) writes. The message can name the entry.

## gno.land/adr/pr6068_govdao_allowlist_and_disclosure.md:15 [gh](https://github.com/gnolang/gno/blob/304f09a7a/gno.land/adr/pr6068_govdao_allowlist_and_disclosure.md?plain=1#L15) · [↗](../../../../../.worktrees/gno-review-6068/gno.land/adr/pr6068_govdao_allowlist_and_disclosure.md#L15)
Nit: the trust-boundary table still lists `SafeExecutor.Execute` at `r/gov/dao/types.gno:221`, which this branch deletes, and lines 184 and 591 reason from it as live code.

## examples/gno.land/r/gov/dao/v3/impl/render.gno:123 [gh](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/render.gno#L123) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/render.gno#L123)
Suggestion: [`p.ExecutorString()`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/render.gno#L118-L123) is evaluated twice per page render, so an executor whose `String()` loops two million times costs 3,895,973,145 gas against 1,953,984,556 with the value in a local, either side of the [3,000,000,000 query cap](https://github.com/gnolang/gno/blob/304f09a7a/gno.land/pkg/sdk/vm/keeper.go#L52). [`types.gno:227-232`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/types.gno#L227-L232) carries the same pair.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6068 -R gnolang/gno
mk() {
cat > examples/gno.land/r/gov/dao/v3/impl/filetests/zz_$1_filetest.gno <<GNO
// PKGPATH: gno.land/r/test/$1
package $1

import (
	"testing"

	"gno.land/r/gov/dao"
	"gno.land/r/gov/dao/v3/impl"
	"gno.land/r/gov/dao/v3/memberstore"
)

const user address = "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"

func init(cur realm) {
	memberstore.Get(0, cur).DeleteAll()
	memberstore.Get(0, cur).SetTier(memberstore.T1)
	memberstore.Get(0, cur).SetMember(memberstore.T1, user, memberstore.NewMember(3))
	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(impl.NewGovDAO(), nil))
}

// Short return value, one int field of stored state, unbounded work.
type ex struct{ N int }

func (e *ex) Execute(cur realm) error { return nil }

func (e *ex) String() string {
	acc := 0
	for i := 0; i < e.N; i++ {
		acc += i
	}
	if acc < 0 {
		return "neg"
	}
	return "meta"
}

func (e *ex) CreationRealm() string { return "gno.land/r/test/$1" }

func main(cur realm) {
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewUserRealm(user))
	pid := dao.MustCreateProposal(cross(cur), dao.NewProposalRequest("T", "d", &ex{N: $2}))
	println(len(dao.Render(cross(cur), pid.String())) > 0)
}

// Output:
// true
GNO
}
mk esburn 2000000
mk esnowork 0
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gov/dao/v3/impl/... 2>&1 | grep -E 'zz_es(burn|nowork)_filetest.gno \(elapsed'
rm examples/gno.land/r/gov/dao/v3/impl/filetests/zz_esburn_filetest.gno examples/gno.land/r/gov/dao/v3/impl/filetests/zz_esnowork_filetest.gno
```

The two runs differ only in the loop count, so their gas difference is the executor body, charged twice.

```
--- PASS: ./gno.land/r/gov/dao/v3/impl/zz_esburn_filetest.gno (elapsed: 3.57s, gas: 3895973145, …)
--- PASS: ./gno.land/r/gov/dao/v3/impl/zz_esnowork_filetest.gno (elapsed: 0.15s, gas: 11995967, …)
```

3,895,973,145 minus 11,995,967 is 3,883,977,178 for two dispatches, so one dispatch leaves 11,995,967 plus 1,941,988,589, which is 1,953,984,556. `clamp.gno` already records that no clamp reaches the executor's method body; halving the number of dispatches is the part that is free here.
</details>
