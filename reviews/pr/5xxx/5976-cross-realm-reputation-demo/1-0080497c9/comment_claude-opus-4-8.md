# Review: PR [#5976](https://github.com/gnolang/gno/pull/5976)
Posted: https://github.com/gnolang/gno/pull/5976#pullrequestreview-4733409026
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Repros run at 0080497c9.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5976-cross-realm-reputation-demo/1-0080497c9/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/reputation/reputation.gno:16-19 [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L16-L19) [posted](https://github.com/gnolang/gno/pull/5976#discussion_r3613026706)
Critical: a `gnokey maketx run` script gets past this guard, so any account credits any address any amount. The script executes in an ephemeral realm at `gno.land/e/<addr>/run`, and [`IsUserCall` is true only for an empty pkgPath](https://github.com/gnolang/gno/blob/0080497c9/gnovm/stdlibs/chain/runtime/frame.gno#L105-L107) · [↗](../../../../../.worktrees/gno-review-5976/gnovm/stdlibs/chain/runtime/frame.gno#L105-L107). [`IsUser`](https://github.com/gnolang/gno/blob/0080497c9/gnovm/stdlibs/chain/runtime/frame.gno#L83-L85) · [↗](../../../../../.worktrees/gno-review-5976/gnovm/stdlibs/chain/runtime/frame.gno#L83-L85) covers both entries, as in [boards2](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L42-L44) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L42-L44).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5976 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/reputation_msgrun_bypass.txtar <<'EOF'
adduserfrom runner 'apart roast chief monitor bundle auto fade double valid budget able average onion slam rice flame despair wage uphold nominee proud alien spider useful'
stdout 'g1advja7j0c7p8f3xp2yf42qnhuv5tdes7ngqp80'

loadpkg gno.land/r/demo/reputation

gnoland start

gnokey maketx run runner $WORK/run/bypass.gno -gas-fee 1000000ugnot -gas-wanted 20000000 -chainid=tendermint_test

stdout 'credited: 1000000'                              # IS:     run script is accepted as an issuer
# ! stdout 'credited: 1000000'                          # SHOULD: guard rejects every user-driven call
# stderr 'can only be called by another realm'          # SHOULD: abort message surfaces

-- run/bypass.gno --
package main

import (
	"gno.land/r/demo/reputation"
)

func main(cur realm) {
	target := address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
	reputation.AddPoints(cross(cur), target, "selfawarded", 1000000)
	println("credited:", reputation.GetTotalScore(target))
}
EOF
go test -v -run 'TestTestdata/reputation_msgrun_bypass' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/reputation_msgrun_bypass.txtar
```

```
> gnokey maketx run runner $WORK/run/bypass.gno -gas-fee 1000000ugnot -gas-wanted 20000000 -chainid=tendermint_test
[stdout]
credited: 1000000

OK!
GAS USED:   4421552
EVENTS:     [{"bytes_delta":2149,"pkg_path":"gno.land/r/demo/reputation"}]
# …
--- PASS: TestTestdata/reputation_msgrun_bypass (4.11s)
```
</details>

## examples/gno.land/r/demo/reputation/reputation.gno:27-37 [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L27-L37) [posted](https://github.com/gnolang/gno/pull/5976#discussion_r3613026710)
Neither running total is bound-checked, and the [per-award positivity check](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L20-L22) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L20-L22) does not constrain the sum. Two `MaxInt64` awards from one issuer silently leave the score at `-2`, with no way to unwind it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5976 -R gnolang/gno
cat > examples/gno.land/r/demo/reputation/overflow_test.gno <<'EOF'
package reputation

import "testing"

func TestOverflow(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/ovfcaller"))
	target := address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
	const maxInt64 = int64(9223372036854775807)
	AddPoints(cross(cur), target, "ovf", maxInt64)
	AddPoints(cross(cur), target, "ovf", maxInt64)
	println("score:", GetScore(target, "ovf", "gno.land/r/demo/ovfcaller"))
}
EOF
cd examples && go run ../gnovm/cmd/gno test -v -run TestOverflow ./gno.land/r/demo/reputation
rm gno.land/r/demo/reputation/overflow_test.gno
```

```
score: -2
=== RUN   TestOverflow
--- PASS: TestOverflow (0.00s)
```
</details>

## examples/gno.land/r/demo/reputation/reputation_test.gno:9-31 [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation_test.gno#L9-L31) [posted](https://github.com/gnolang/gno/pull/5976#discussion_r3613026715)
Missing test: two issuer realms writing the same category for the same target. The issuer segment in [`scoreKey`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L56-L58) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L56-L58) is what stops one realm overwriting another's award, and every test uses a single issuer, so dropping it would keep the suite green.

<details><summary>test cases</summary>

```go
func TestScoresAreIsolatedPerIssuer(cur realm, t *testing.T) {
	target := address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")

	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/issuera"))
	AddPoints(cross(cur), target, "shared", 10)

	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/issuerb"))
	AddPoints(cross(cur), target, "shared", 5)

	uassert.Equal(t, int64(10), GetScore(target, "shared", "gno.land/r/demo/issuera"))
	uassert.Equal(t, int64(5), GetScore(target, "shared", "gno.land/r/demo/issuerb"))
	uassert.Equal(t, int64(15), GetTotalScore(target))
}
```
</details>

## examples/gno.land/r/demo/reputation/reputation.gno:65-68 [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L65-L68) [posted](https://github.com/gnolang/gno/pull/5976#discussion_r3613026719)
Nit: this walks every address in the ledger with no bound, and the ledger only grows. Measured output reaches 98,206 bytes at 2000 addresses, so `vm/qrender` eventually returns an unusable page. [`IterateByOffset`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/p/nt/avl/v0/tree.gno#L111) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/p/nt/avl/v0/tree.gno#L111) pages it, and the unused `path` argument can carry the page.

## examples/gno.land/r/demo/reputation/reputation.gno:15 [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L15) [posted](https://github.com/gnolang/gno/pull/5976#discussion_r3613026723)
Nit: none of the four exported functions carry a doc comment, and a demo realm is read as a pattern. The realm-only contract on [`AddPoints`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L15-L19) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L15-L19) lives only in the panic string, out of sight of anyone scanning the signature.

## examples/gno.land/r/demo/reputation/reputation.gno:56-58 [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L56-L58) [posted](https://github.com/gnolang/gno/pull/5976#discussion_r3613026729)
Suggestion: on the key shape you asked about, writes are safe but reads are not. The issuer segment comes from `cur.Previous().PkgPath()` and never contains `|`, but [`GetScore`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L40-L46) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L40-L46) takes `issuerPkgPath` verbatim, so an award stored under category `a|b` also reads back as category `a` from issuer `b|<that path>`. Nesting a per-issuer [`avl.Tree`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/p/nt/avl/v0/tree.gno#L27-L29) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/p/nt/avl/v0/tree.gno#L27-L29) under each address drops the concatenation and gives [`Render`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L60-L69) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L60-L69) cheap per-issuer enumeration.

## examples/gno.land/r/demo/reputation/reputation.gno:63 [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L63) [posted](https://github.com/gnolang/gno/pull/5976#discussion_r3613026733)
Suggestion: this line reads as a restriction, but deploying a one-line realm clears it. Anyone can `addpkg` a realm that calls [`AddPoints`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L15-L19) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L15-L19) and mint without limit, and [`GetTotalScore`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L48-L54) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L48-L54) sums across issuers, so the headline number on the page is attacker-controlled. Describe the guarantee as per-issuer attribution instead.
