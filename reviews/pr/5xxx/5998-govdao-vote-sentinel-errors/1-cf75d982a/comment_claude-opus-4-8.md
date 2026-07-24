# Review: PR [#5998](https://github.com/gnolang/gno/pull/5998)
Posted: https://github.com/gnolang/gno/pull/5998#pullrequestreview-4771981362
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Repros run at cf75d982a.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5998-govdao-vote-sentinel-errors/1-cf75d982a/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/gov/dao/v3/impl/govdao.gno:19-30 [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L19-L30) [posted](https://github.com/gnolang/gno/pull/5998#discussion_r3644373069)
These sentinels live in the versioned implementation, but callers reach voting through [`gno.land/r/gov/dao`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/proxy.gno#L101-L106), and [`UpdateImpl`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/proxy.gno#L146-L160) swaps the implementation behind that entry point. The next implementation declares its own [`errors.New`](https://github.com/gnolang/gno/blob/cf75d982a/gnovm/stdlibs/errors/errors.gno#L44-L46) values, so after a swap a caller's `errors.Is` against [`ErrProposalNotFound`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L20) goes false while the message stays identical, and the caller's branch silently stops firing. A sentinel that survives a swap has to be declared where the entry point is.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5998 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/dao5998_implswap.txtar <<'EOF'
adduserfrom member 'success myself purchase tray reject demise scene little legend someone lunar hope media goat regular test area smart save flee surround attack rapid smoke'
stdout 'g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0'

loadpkg gno.land/r/gov/dao
loadpkg gno.land/r/gov/dao/v3/impl
loadpkg gno.land/r/gov/dao/v4/impl $WORK/v4impl
loadpkg gno.land/r/demo/upgrader $WORK/upgrader
loadpkg gno.land/r/gov/dao/v3/loader $WORK/loader

gnoland start

# before the swap: the v3 sentinel matches
gnokey maketx run -gas-fee 4300001ugnot -gas-wanted 50_000_000 -chainid=tendermint_test member $WORK/run/check.gno
stdout 'msg: proposal not found'
stdout 'v3-sentinel-matches: true'

gnokey maketx call -pkgpath gno.land/r/demo/upgrader -func Upgrade -gas-fee 1900001ugnot -gas-wanted 19_000_000 -chainid=tendermint_test member
stdout OK!

# after the swap: identical message, sentinel no longer matches
gnokey maketx run -gas-fee 4300001ugnot -gas-wanted 50_000_000 -chainid=tendermint_test member $WORK/run/check.gno
stdout 'msg: proposal not found'
stdout 'v3-sentinel-matches: false'    # IS:     the caller's branch silently disappears
# stdout 'v3-sentinel-matches: true'   # SHOULD: a version-independent sentinel keeps matching

-- run/check.gno --
package main

import (
	"errors"

	"gno.land/r/gov/dao"
	v3impl "gno.land/r/gov/dao/v3/impl"
)

func main(cur realm) {
	err := dao.VoteOnProposal(cross(cur), dao.NewVoteRequest(dao.YesVote, dao.ProposalID(999)))
	println("msg:", err.Error())
	println("v3-sentinel-matches:", errors.Is(err, v3impl.ErrProposalNotFound))
}
-- v4impl/v4impl.gno --
package impl

import (
	"errors"

	"gno.land/r/gov/dao"
)

var ErrProposalNotFound = errors.New("proposal not found")

type GovDAOv4 struct{}

var _instance *GovDAOv4 = &GovDAOv4{}

func GetInstance(_ int, rlm realm) *GovDAOv4 { return _instance }

func (g *GovDAOv4) PreCreateProposal(_ int, rlm realm, r dao.ProposalRequest) (address, error) {
	return "", errors.New("not implemented")
}

func (g *GovDAOv4) PostCreateProposal(_ int, rlm realm, r dao.ProposalRequest, pid dao.ProposalID) {}

func (g *GovDAOv4) VoteOnProposal(_ int, rlm realm, r dao.VoteRequest) error {
	return ErrProposalNotFound
}

func (g *GovDAOv4) PreExecuteProposal(_ int, rlm realm, pid dao.ProposalID) (bool, error) {
	return false, errors.New("not implemented")
}

func (g *GovDAOv4) ExecuteProposal(_ int, rlm realm, pid dao.ProposalID, e dao.Executor) error {
	return nil
}

func (g *GovDAOv4) Render(cur realm, pkgpath string, path string) string { return "v4" }
-- upgrader/upgrader.gno --
package upgrader

import (
	"gno.land/r/gov/dao"
	v4impl "gno.land/r/gov/dao/v4/impl"
)

func Upgrade(cur realm) {
	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(v4impl.GetInstance(0, cur), []string{"gno.land/r/demo/upgrader"}))
}
-- loader/loader.gno --
package loader

import (
	"gno.land/r/gov/dao"
	"gno.land/r/gov/dao/v3/impl"
	"gno.land/r/gov/dao/v3/memberstore"
)

func init(cur realm) {
	memberstore.Get(0, cur).SetTier(memberstore.T1)
	memberstore.Get(0, cur).SetTier(memberstore.T2)
	memberstore.Get(0, cur).SetTier(memberstore.T3)

	memberstore.Get(0, cur).SetMember(memberstore.T1, address("g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0"), memberstore.NewMember(3))

	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(impl.GetInstance(0, cur), []string{"gno.land/r/gov/dao/v3/impl", "gno.land/r/demo/upgrader"}))
}
EOF
go test -v -run 'TestTestdata/dao5998_implswap' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/dao5998_implswap.txtar
```

```
# before the swap
> gnokey maketx run … $WORK/run/check.gno
msg: proposal not found
v3-sentinel-matches: true
# …
> gnokey maketx call -pkgpath gno.land/r/demo/upgrader -func Upgrade …
OK!
# after the swap
> gnokey maketx run … $WORK/run/check.gno
msg: proposal not found
v3-sentinel-matches: false
# …
--- PASS: TestTestdata/dao5998_implswap (33.57s)
```
</details>

## examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno:338-402 [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L338-L402) [posted](https://github.com/gnolang/gno/pull/5998#discussion_r3644373072)
Missing test: nothing asserts that a sentinel matches only itself. Replacing the body of [`ProposalClosedError.Is`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L45-L47) with `return true` leaves this test green, and so would collapsing all five sentinels onto one value.

<details><summary>test cases</summary>

```go
func TestSentinelsAreDisjoint(t *testing.T) {
	sentinels := []error{
		ErrMemberNotFound,
		ErrProposalNotFound,
		ErrNotAllowedToVote,
		ErrAlreadyVoted,
		ErrInvalidVoteOption,
		ErrProposalClosed,
	}

	for i, a := range sentinels {
		for j, b := range sentinels {
			uassert.Equal(t, i == j, errors.Is(a, b))
		}
	}

	closed := error(&ProposalClosedError{Accepted: true})
	for _, s := range sentinels {
		uassert.Equal(t, s == ErrProposalClosed, errors.Is(closed, s))
	}
	uassert.False(t, errors.Is(ErrProposalClosed, closed))
}
```
</details>

## examples/gno.land/r/gov/dao/v3/impl/govdao.gno:120-133 [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L120-L133) [posted](https://github.com/gnolang/gno/pull/5998#discussion_r3644373077)
Nit: no code realm can reach these returns. [`isValidCall`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L214-L229) admits only a direct user call or a `maketx run` script. A realm calling [`dao.VoteOnProposal`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/proxy.gno#L101-L106) on chain is turned away by [the one branch still returning an ad-hoc `errors.New`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L108-L110), so it gets `proposal voting must be done directly by a user`, which no sentinel matches.
