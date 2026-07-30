# Review: PR [#6012](https://github.com/gnolang/gno/pull/6012)
Event: REQUEST_CHANGES

## Body
One shape recurs and is worth a single decision rather than two: the code writes the immediate structural neighbour where it already derives the correct principal or destination a few lines away. Reproduced on bb9f82f69.

The PR description does not match this head. It documents an `Options` type, an `UpdateOptions` entry point and an `AllowExecution` gate, none of which exist here. It gives `New(name, description, members)` where the actual signature is [`New(cur, name, purpose, description, members)`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/public.gno#L69). It counts 158 files, 21 commits and 92 filetests against 195 files, 51 commits and 107 realm filetests. On a breaking change this size the description is the reviewer's map.

The red `gno-checks / fmt` and `generate` jobs are not a code problem. 17 modified filetests carry stray double blank lines; `gno fmt -w` clears both jobs.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6012-commondao-council-tally-treasuries/1-bb9f82f69/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno:181-189 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L181-L189)
Critical: the sweep sends to `parent.Address()` without checking whether that parent is dissolved, so a DAO's funds can land in an address no council can ever spend from. A dissolved DAO rejects [`Propose`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/commondao.gno#L305-L307), and the clawback that would have rescued the balance dies with the last live ancestor, a state [`render.gno:82-86`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L82-L86) already labels unrecoverable. Sending to the nearest live proper ancestor keeps the funds spendable, and [`hasLiveProperAncestor`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_treasury.gno#L16-L23) already walks to it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6012 -R gnolang/gno
cat > examples/quarantined/gno.land/r/nt/commondao/v0/filetests/zz_sweep_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/nt/commondao/v0/filetests/zz_sweep_filetest

package zz_sweep_filetest

import (
	"chain"
	"chain/banker"
	"testing"

	pdao "gno.land/p/nt/commondao/v0"
	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq")
	user  = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
)

const (
	rootID = uint64(2)
	midID  = uint64(3)
	leafID = uint64(4)
)

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), user)

	testing.SetRealm(testing.NewUserRealm(user))
	commondao.New(cross(cur), "Root", "Purpose", "", "")

	// Root -> Mid -> Leaf
	pID := commondao.CreateSubDAOProposal(cross(cur), rootID, "Mid", "Purpose", "", string(user))
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)

	pID = commondao.CreateSubDAOProposal(cross(cur), midID, "Leaf", "Purpose", "", string(user))
	commondao.Vote(cross(cur), midID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), midID, pID)

	testing.IssueCoins(commondao.GetView(leafID).Address(), chain.NewCoins(chain.NewCoin("ugnot", 500)))

	// Dissolve Mid first. Mid holds nothing, so nothing is swept.
	pID = commondao.CreateDissolutionProposal(cross(cur), midID, "")
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)
}

func main(cur realm) {
	testing.SetRealm(testing.NewUserRealm(user))

	// Leaf's dissolution is hosted in Root, the nearest live ancestor.
	pID := commondao.CreateDissolutionProposal(cross(cur), leafID, "")
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)

	b := banker.NewReadonlyBanker()
	println("mid treasury:", b.GetCoins(commondao.GetView(midID).Address()).String())
	println("root treasury:", b.GetCoins(commondao.GetView(rootID).Address()).String())

	// Root dissolves itself and names a destination. Whatever sits in Mid is
	// now beyond every governance path.
	pID = commondao.CreateDissolutionProposal(cross(cur), rootID, owner)
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)

	println("after root dissolution, mid treasury:", b.GetCoins(commondao.GetView(midID).Address()).String())
	println("destination:", b.GetCoins(owner).String())
}

// Output:
// mid treasury:
// root treasury: 500ugnot
// after root dissolution, mid treasury:
// destination: 500ugnot
EOF
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'zz_sweep_filetest.gno'
rm filetests/zz_sweep_filetest.gno
```

```
=== RUN   ./zz_sweep_filetest.gno
--- FAIL: ./zz_sweep_filetest.gno
Output diff:
--- Expected
+++ Actual
@@ -1,4 +1,4 @@
-mid treasury:
-root treasury: 500ugnot
-after root dissolution, mid treasury:
-destination: 500ugnot
+mid treasury: 500ugnot
+root treasury:
+after root dissolution, mid treasury: 500ugnot
+destination:
```
</details>

## examples/quarantined/gno.land/r/nt/commondao/v0/public.gno:84-91 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/public.gno#L84-L91)
Critical: the invite is checked against `unsafe.OriginCaller()` on line 84 and the council seat is written from `cur.Previous().Address()` on line 91, so a realm the invited user calls consumes their invite and takes the only council seat. It is not one-shot either: the origin is recorded permanently in `creators` and read back by [`isCreator`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/commondao.gno#L55-L57), so every later call that user makes through that realm mints another DAO under the realm's address. [`Invite`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/public.gno#L49) already authenticates the immediate caller.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6012 -R gnolang/gno
cat > examples/quarantined/gno.land/r/nt/commondao/v0/filetests/zz_invite_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/nt/commondao/v0/filetests/zz_invite_filetest

package zz_invite_filetest

import (
	"chain"
	"testing"

	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq")
	user  = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
)

const evilPath = "gno.land/r/x/evil"

var daoID uint64

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), user)

	// The invited account is the transaction origin, but an intermediate
	// realm is the immediate caller.
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewCodeRealm(evilPath))
	daoID = commondao.New(cross(cur), "Hijacked", "Purpose", "", "")
}

func main() {
	view := commondao.GetView(daoID)
	println("invite still held by user:", commondao.IsInvited(user))
	println("user is council member:", view.Council().Has(user))
	println("intermediate realm is council member:", view.Council().Has(chain.PackageAddress(evilPath)))
}

// Output:
// invite still held by user: true
// user is council member: true
// intermediate realm is council member: false
EOF
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'zz_invite_filetest.gno'
rm filetests/zz_invite_filetest.gno
```

```
=== RUN   ./zz_invite_filetest.gno
--- FAIL: ./zz_invite_filetest.gno
Output diff:
--- Expected
+++ Actual
@@ -1,3 +1,3 @@
-invite still held by user: true
-user is council member: true
-intermediate realm is council member: false
+invite still held by user: false
+user is council member: false
+intermediate realm is council member: true
```
</details>

## examples/gno.land/p/nt/commondao/v0/commondao.gno:432-440 [↗](../../../../../.worktrees/gno-review-6012/examples/gno.land/p/nt/commondao/v0/commondao.gno#L432-L440)
The tally finalizes on a denominator that abstainers just shrank, and finalizing then [refuses any further vote](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/commondao.gno#L410-L412), so an abstainer who would have moved to NO is silenced by a state their own abstain created. In a three member council one YES plus two ABSTAIN passes, while the same three members reaching the same final intentions in a different order dismiss the proposal. The comment on line 432 calls the outcome mathematically settled, which holds only for a denominator abstains cannot move, and the [property test](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/record_test.gno#L220-L277) checks final vote sets so it cannot see this.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6012 -R gnolang/gno
cat > examples/quarantined/gno.land/r/nt/commondao/v0/filetests/zz_order_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/nt/commondao/v0/filetests/zz_order_filetest

package zz_order_filetest

import (
	"testing"

	pdao "gno.land/p/nt/commondao/v0"
	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq")
	a     = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
	b     = address("g1us8428u2a5satrlxzagqqa5m6vmuze025anjlj")
	c     = address("g1manfred47kzduec920z88wfr64ylksmdcedlf5")
)

const (
	daoOne = uint64(2)
	daoTwo = uint64(3)
)

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), a)

	// Two DAOs with the same three member council.
	testing.SetRealm(testing.NewUserRealm(a))
	commondao.New(cross(cur), "One", "Purpose", "", string(b)+"\n"+string(c))
	commondao.New(cross(cur), "Two", "Purpose", "", string(b)+"\n"+string(c))
}

func vote(cur realm, daoID, pID uint64, m address, choice pdao.VoteChoice) {
	testing.SetRealm(testing.NewUserRealm(m))
	commondao.Vote(cross(cur), daoID, pID, choice, "")
}

func status(daoID, pID uint64) string {
	p, _ := commondao.GetView(daoID).GetProposal(pID)
	return string(p.Status())
}

func main(cur realm) {
	testing.SetRealm(testing.NewUserRealm(a))
	p1 := commondao.CreateTextProposal(cross(cur), daoOne, "T", "B", 7)
	testing.SetRealm(testing.NewUserRealm(a))
	p2 := commondao.CreateTextProposal(cross(cur), daoTwo, "T", "B", 7)

	// Same council, same supermajority proposal, and b ends on NO in both.
	// Only the cast order differs.

	// Order 1: c abstains before b settles. The two abstains cut the
	// denominator to 1, so a's single YES clears two thirds.
	vote(cur, daoOne, p1, a, pdao.ChoiceYes)
	vote(cur, daoOne, p1, b, pdao.ChoiceAbstain)
	vote(cur, daoOne, p1, c, pdao.ChoiceAbstain)
	println("order 1, b has not settled yet:", status(daoOne, p1))

	// Order 2: c votes NO, the denominator stays at 3, and b's revision
	// to NO still lands.
	vote(cur, daoTwo, p2, a, pdao.ChoiceYes)
	vote(cur, daoTwo, p2, b, pdao.ChoiceAbstain)
	vote(cur, daoTwo, p2, c, pdao.ChoiceNo)
	vote(cur, daoTwo, p2, b, pdao.ChoiceNo)
	println("order 2, b settled on NO:", status(daoTwo, p2))
}

// Output:
// order 1, b has not settled yet: active
// order 2, b settled on NO: dismissed
EOF
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'zz_order_filetest.gno'
rm filetests/zz_order_filetest.gno
```

```
=== RUN   ./zz_order_filetest.gno
--- FAIL: ./zz_order_filetest.gno
Output diff:
--- Expected
+++ Actual
@@ -1,2 +1,2 @@
-order 1, b has not settled yet: active
+order 1, b has not settled yet: passed
 order 2, b settled on NO: dismissed
```
</details>

## examples/quarantined/gno.land/r/nt/commondao/v0/genesis.gno:3 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/genesis.gno#L3)
`cur` is never referenced in this body, and the transpiler lowers it to a local variable rather than a Go parameter, so the [`gno2go` job](https://github.com/gnolang/gno/actions/runs/30430322979/job/90505877027) fails with `genesis.gno:6:2: declared and not used: cur`. `gno test` and `gno lint` both pass on the file, which is why it reached CI. The signature appears 360 times across `examples/` and every other occurrence references `cur`.

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_council.gno:157-172 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_council.gno#L157-L172)
`Validate` checks ancestry and the add/remove overlap but not whether the target is dissolved, so an ancestor can spend a council vote seating members on a DAO that rejects every mutator. Its siblings do check the flag, at [`proposal_treasury.gno:105-107`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_treasury.gno#L105-L107) and [`proposal_subdao.gno:154-159`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L154-L159).

## examples/quarantined/gno.land/r/nt/commondao/v0/render.gno:49-66 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L49-L66)
The pager counts `daos.Size()` while line 58 keeps only listed DAOs and line 64 returns before the page picker is written, so a listed DAO behind ten unlisted ones is on no reachable page and there is no link onward. The proposals pager in the same file counts the collection it actually renders at [`:365`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L365). The mismatch predates this branch: the old gate was an option with the same default, and the rewritten predicate kept the count.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6012 -R gnolang/gno
cat > examples/quarantined/gno.land/r/nt/commondao/v0/filetests/zz_pager_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/nt/commondao/v0/filetests/zz_pager_filetest

package zz_pager_filetest

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq")
	user  = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
)

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), user)
	testing.SetRealm(testing.NewUserRealm(user))
	// 12 DAOs, none listed. Only the genesis DAO is listed.
	for i := 0; i < 12; i++ {
		commondao.New(cross(cur), "D"+strconv.Itoa(i), "Purpose", "", "")
	}
}

func main() {
	println("page 1 shows the listed DAO:", strings.Contains(commondao.Render(""), "/r/nt/commondao/v0:1)"))
	println("page 2 shows the listed DAO:", strings.Contains(commondao.Render("?page=2"), "/r/nt/commondao/v0:1)"))
}

// Output:
// page 1 shows the listed DAO: true
// page 2 shows the listed DAO: false
EOF
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'zz_pager_filetest.gno'
rm filetests/zz_pager_filetest.gno
```

```
=== RUN   ./zz_pager_filetest.gno
--- FAIL: ./zz_pager_filetest.gno
Output diff:
--- Expected
+++ Actual
@@ -1,2 +1,2 @@
-page 1 shows the listed DAO: true
-page 2 shows the listed DAO: false
+page 1 shows the listed DAO: false
+page 2 shows the listed DAO: true
```
</details>

## examples/quarantined/gno.land/r/nt/commondao/v0/render.gno:503 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L503)
`md.EscapeText` is [`sanitize.InlineText`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/moul/md/md.gno#L414-L416), which folds every newline to a space, so a multi-paragraph proposal body renders as one run-on line. The slot it guards accepts [15,000 characters](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_text.gno#L11), and `sanitize` names [proposal descriptions](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L375) as the block helper's use case and says [multi-paragraph prose does not belong in `InlineText`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L329-L331). The golden at [`z_10_b_filetest.gno`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/filetests/z_10_b_filetest.gno) only uses a one-line body.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6012 -R gnolang/gno
cat > examples/quarantined/gno.land/r/nt/commondao/v0/filetests/zz_body_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/nt/commondao/v0/filetests/zz_body_filetest

package zz_body_filetest

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq")
	user  = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
)

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), user)
	testing.SetRealm(testing.NewUserRealm(user))
	commondao.New(cross(cur), "D", "Purpose", "", "")
	commondao.CreateTextProposal(cross(cur), 2, "T", "Para one.\n\nPara two.", 7)
}

func main() {
	out := commondao.Render("2/proposals/1")
	desc := out[strings.Index(out, "## Description"):]
	println("newlines in description:", strconv.Itoa(strings.Count(desc, "\n")))
	println(desc)
}

// Output:
// newlines in description: 4
EOF
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'zz_body_filetest.gno'
rm filetests/zz_body_filetest.gno
```

```
=== RUN   ./zz_body_filetest.gno
--- FAIL: ./zz_body_filetest.gno
Output diff:
--- Expected
+++ Actual
@@ -1 +1,3 @@
-newlines in description: 4
+newlines in description: 3
+## Description
+Para one\.  Para two\.
```
</details>

## examples/gno.land/p/nt/commondao/v0/readonly.gno:133-144 [↗](../../../../../.worktrees/gno-review-6012/examples/gno.land/p/nt/commondao/v0/readonly.gno#L133-L144)
Missing test: nothing pins that `ReadonlyProposal` does not expose the definition, the property the flattened [`Title`/`Body`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/readonly.gno#L176-L184) accessors exist for. Adding a `Definition()` accessor in a later refactor hands `Executor` to any cross-realm holder and every test still passes.

<details><summary>test cases</summary>

```go
func TestReadonlyProposalHidesDefinition(t *testing.T) {
	dao := commondao.New(commondao.WithID(1), commondao.WithCouncilMember(member))
	uassert.NoError(t, dao.RegisterKind(testKind{}))
	p, err := dao.Propose(member, testKind{}.Name(), nil)
	uassert.NoError(t, err)

	view := p.Readonly()
	_, ok := any(view).(interface {
		Definition() commondao.ProposalDefinition
	})
	uassert.False(t, ok, "ReadonlyProposal must not expose the definition")
}

func TestReadonlyCommonDAOHidesMutators(t *testing.T) {
	dao := commondao.New(commondao.WithID(1), commondao.WithCouncilMember(member))
	view := dao.Readonly()

	_, ok := any(view).(interface{ UpdateCouncil(add, remove []address) error })
	uassert.False(t, ok, "ReadonlyCommonDAO must not expose UpdateCouncil")
	_, ok = any(view).(interface{ Dissolve(reason string) })
	uassert.False(t, ok, "ReadonlyCommonDAO must not expose Dissolve")
	_, ok = any(view).(interface{ SetTreasuryFrozen(frozen bool) })
	uassert.False(t, ok, "ReadonlyCommonDAO must not expose SetTreasuryFrozen")
}
```
</details>

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_kinds.gno:358-362 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_kinds.gno#L358-L362)
Missing test: no test drives two concurrently passed kind toggles, the case the comment above `execute` says fails cleanly rather than aborting the transaction. `New` rejects no-ops at [creation time](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_kinds.gno#L290-L295), so the branch is reachable only by creating both proposals while the kind is in the same state. Neither registry error is asserted on an execution path anywhere in the suite.

<details><summary>test cases</summary>

```go
// filetests/z_kind_toggle_race_filetest.gno, after creating a DAO whose
// council is a single member:

// Two proposals disabling the same kind, both created while it is enabled.
p1 := commondao.CreateSetProposalKindProposal(cross(cur), daoID, "text", false)
p2 := commondao.CreateSetProposalKindProposal(cross(cur), daoID, "text", false)

commondao.Vote(cross(cur), daoID, p1, pdao.ChoiceYes, "")
commondao.Execute(cross(cur), daoID, p1)
commondao.Vote(cross(cur), daoID, p2, pdao.ChoiceYes, "")
commondao.Execute(cross(cur), daoID, p2)

view := commondao.GetView(daoID)
a, _ := view.GetProposal(p1)
b, _ := view.GetProposal(p2)
println("first:", string(a.Status()))
println("second:", string(b.Status()), "-", b.StatusReason())

// Output:
// first: executed
// second: failed - proposal kind not found
```
</details>

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno:154-159 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L154-L159)
Missing test: the already-dissolved check is only ever reached by calling `Validate` directly, at [`proposal_subdao_test.gno:116`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao_test.gno#L116), never through `Execute`. The end-to-end path is reachable because a sub-DAO's dissolution proposal is hosted in the live parent and so survives the first dissolution. Every spend precondition is covered in both positions, but dissolution is not.

<details><summary>test cases</summary>

```go
// filetests/z_double_dissolve_filetest.gno: parent hosts two dissolution
// proposals for the same sub-DAO, both pass, both execute.

p1 := commondao.CreateDissolutionProposal(cross(cur), subID, "")
p2 := commondao.CreateDissolutionProposal(cross(cur), subID, "")

commondao.Vote(cross(cur), rootID, p1, pdao.ChoiceYes, "")
commondao.Execute(cross(cur), rootID, p1)
commondao.Vote(cross(cur), rootID, p2, pdao.ChoiceYes, "")
commondao.Execute(cross(cur), rootID, p2)

view := commondao.GetView(rootID)
a, _ := view.GetProposal(p1)
b, _ := view.GetProposal(p2)
println("first:", string(a.Status()))
println("second:", string(b.Status()), "-", b.StatusReason())

// Output:
// first: executed
// second: failed - DAO has already been dissolved
```
</details>

## gno.land/pkg/integration/testdata/commondao_election.txtar:53 [↗](../../../../../.worktrees/gno-review-6012/gno.land/pkg/integration/testdata/commondao_election.txtar#L53)
Missing test: the bare `!` passes on any non-zero exit, including a gas or argument error, so this does not pin that a dismissed proposal cannot be executed. The balance check below it is real; this line is not, because nothing pins that the rejection is `proposal not found`.

## examples/gno.land/p/nt/commondao/v0/commondao.gno:472-473 [↗](../../../../../.worktrees/gno-review-6012/examples/gno.land/p/nt/commondao/v0/commondao.gno#L472-L473)
Nit: `ErrExecutionNotAllowed` is unreachable. `Execute` reads from [`activeProposals` only](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/commondao.gno#L460), where the only statuses ever stored are `StatusActive` and `StatusPassed`, and every other status assignment removes the proposal first.

## examples/quarantined/gno.land/r/nt/commondao/v0/commondao.gno:79-82 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/commondao.gno#L79-L82)
Nit: this says executors mint `cur.Sub(subpathOf(daoID))`. The host mints it at [`public.gno:186`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/public.gno#L186) and passes the result in, which is how [`public.gno:16`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/public.gno#L16) describes it.

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno:115 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L115)
Nit: the panic reads `SubDAO is required` and the type comment at [`:130`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L130) says "dissolving a SubDAO", but this definition also dissolves root DAOs, which is what the whole [`destination` branch](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L117-L125) is for.

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_text.gno:58 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_text.gno#L58)
Nit: this says the marker makes the renderer escape Title and Body. It [gates Body only](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L502-L504), and titles are escaped unconditionally at [`:419`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L419) and [`:462`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L462).

## examples/quarantined/gno.land/r/nt/commondao/v0/render.gno:592-595 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L592-L595)
Nit: the vote reason is rendered whole and nothing bounds its length, neither [`public.gno:135`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/public.gno#L135) nor [`commondao.gno:400`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/commondao.gno#L400), unlike the proposal [title](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_text.gno#L23) and [body](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_text.gno#L32). Escaping is correct, so this is page weight rather than injection.

## examples/gno.land/p/nt/commondao/v0/doc.gno:15-19 [↗](../../../../../.worktrees/gno-review-6012/examples/gno.land/p/nt/commondao/v0/doc.gno#L15-L19)
Suggestion: this gives the supermajority arm and the dismissal rule but omits the [simple-majority arm](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/record.gno#L175-L177) and the [`D > 0` guard](https://github.com/gnolang/gno/blob/bb9f82f69/examples/gno.land/p/nt/commondao/v0/record.gno#L169-L172), so it reads as if supermajority were the only threshold. The README and `TallyDefault`'s own comment both carry the full rule.

## examples/gno.land/p/nt/commondao/v0/proposal.gno:82 [↗](../../../../../.worktrees/gno-review-6012/examples/gno.land/p/nt/commondao/v0/proposal.gno#L82)
Suggestion: `ExecFunc` puts a realm value in the hands of whoever implements the interface, which is safe here only because [`proposalKindCatalog`](https://github.com/gnolang/gno/blob/bb9f82f69/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_kinds.gno#L38-L48) is a fixed in-package list. A second host on this package that registers user-supplied kinds hands each one a realm capability, and passing the caller address instead keeps the executor inert.

## examples/quarantined/gno.land/r/nt/commondao/v0/render.gno:22-28 [↗](../../../../../.worktrees/gno-review-6012/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L22-L28)
Suggestion: a definition is rendered as trusted markdown unless it implements the marker, so a definition that forgets it leaks raw user text at document level. All nine catalog definitions escape their user fields today. Inverting the marker so the chrome builders opt in makes the failure mode safe.
