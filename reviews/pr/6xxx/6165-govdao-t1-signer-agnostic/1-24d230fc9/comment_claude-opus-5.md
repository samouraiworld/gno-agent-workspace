# Review: [#6165](https://github.com/gnolang/gno/pull/6165)
Event: COMMENT

## Body
- [`misc/govdao-scripts/README.md:17`](https://github.com/gnolang/gno/blob/24d230fc9/misc/govdao-scripts/README.md?plain=1#L17) describes the command as `add 6 T1 members to govDAO (one-time bootstrap)`, and the roster holds seven.

## misc/govdao-scripts/extend-govdao-t1.sh:50 [gh](https://github.com/gnolang/gno/blob/24d230fc9/misc/govdao-scripts/extend-govdao-t1.sh#L50) · [↗](../../../../../.worktrees/gno-review-6165/misc/govdao-scripts/extend-govdao-t1.sh#L50)
Related: this [`memberstore.Get`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L186-L188) call sees the caller as [`gno.land/e/<signer>/run`](https://github.com/gnolang/gno/blob/24d230fc9/gno.land/pkg/sdk/vm/keeper.go#L1398), a path missing from [`AllowedDAOs`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/proxy.gno#L231-L240) on every network the script names, so the script seats nobody. Seating a T1 member on those networks goes through [`NewAddMemberRequest`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L80) as a govDAO proposal.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6165 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/roster_locked.txtar <<'TXTAR'
loadpkg gno.land/r/gov/dao
loadpkg gno.land/r/gov/dao/v3/impl
loadpkg gno.land/r/gov/dao/v3/loader $WORK/loader

gnoland start

# test1 is the sole T1 member and the signer, exactly as aeddi is on
# pearl/sapphire/topaz.
! gnokey maketx run -gas-fee 5000000ugnot -gas-wanted 100000000 -chainid=tendermint_test test1 $WORK/run/extend_govdao.gno
stderr 'this Realm is not allowed to get the Members data: gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run'

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

	// test1, the signer below, is seated as the sole T1 member.
	memberstore.Get(0, cur).SetMember(memberstore.T1, address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"), memberstore.NewMember(3))

	// The production lockdown, copied from
	// misc/deployments/pearl.gno.land/transactions/base/bootstrap/govdao_prop1_pearl.gno
	// and misc/deployments/gnoland1/govdao_prop1.gno.
	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(impl.GetInstance(0, cur), []string{"gno.land/r/gov/dao/v3/impl"}))
}
-- run/extend_govdao.gno --
package main

import (
	"gno.land/r/gov/dao/v3/memberstore"
)

type rosterEntry struct {
	name string
	addr address
}

// t1Roster is the full target T1 membership, deliberately signer-agnostic:
// the signer is necessarily already a T1 member (that is what authorizes this
// MsgRun), so their own entry is filtered out at runtime instead of being
// hardcoded out of the list.
var t1Roster = []rosterEntry{
	{"Jae", "g1ecsuj0q572jr0dhu29q9njtnmw03hyu7tyyvv6"},
	{"Morgan", "g1m0rgan0rla00ygmdmp55f5m0unvsvknluyg2a4"},
	{"Aeddi", "g1aeddlftlfk27ret5rf750d7w5dume3kcsm8r8m"},
	{"Dongwon", "g1gzhj234kpajz963z5vf42j4ylddscnkez2wvly"},
	{"Maxwell", "g127l4gkhk0emwsx5tmxe96sp86c05h8vg5tufzq"},
	{"Milos", "g1e6gxg5tvc55mwsn7t7dymmlasratv7mkv0rap2"},
	{"Manfred", "g1manfred47kzduec920z88wfr64ylksmdcedlf5"},
}

func main(cur realm) {
	ms := memberstore.Get(0, cur)
	for _, r := range t1Roster {
		// SetMember errors out if the address already sits in any tier, and a
		// single error would abort the whole transaction. Skip instead: this is
		// what drops the signer's own entry, and it makes reruns idempotent.
		if _, tier := ms.GetMember(r.addr); tier != "" {
			println("skip " + r.name + " -- already " + tier)
			continue
		}
		if err := ms.SetMember(memberstore.T1, r.addr, &memberstore.Member{InvitationPoints: 3}); err != nil {
			panic(err.Error())
		}
		println("seat " + r.name + " as T1")
	}
}
TXTAR

go test -v ./gno.land/pkg/integration/ -run 'TestTestdata/roster_locked'
rm gno.land/pkg/integration/testdata/roster_locked.txtar
```

The signed run is refused before the roster loop starts:

```
> ! gnokey maketx run -gas-fee 5000000ugnot -gas-wanted 100000000 -chainid=tendermint_test test1 $WORK/run/extend_govdao.gno
"gnokey" error: --= Error =--
Data: this Realm is not allowed to get the Members data: gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run
# ...
panic: this Realm is not allowed to get the Members data: gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run
Get at gno.land/r/gov/dao/v3/memberstore/memberstore.gno:188
main at gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run/extend_govdao.gno:27
--- PASS: TestTestdata/roster_locked (2.40s)
```

| Network | Line that closes the genesis window |
| --- | --- |
| gnoland1 | [`govdao_prop1.gno:118-120`](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/gnoland1/govdao_prop1.gno#L118-L120) |
| test13 | [`govdao_prop1_test13.gno:109`](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/test13.gno.land/transactions/base/bootstrap/govdao_prop1_test13.gno#L109) |
| pearl | [`govdao_prop1_pearl.gno:49`](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/pearl.gno.land/transactions/base/bootstrap/govdao_prop1_pearl.gno#L49) |
| sapphire | [`govdao_prop1_sapphire.gno:49`](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/sapphire.gno.land/transactions/base/bootstrap/govdao_prop1_sapphire.gno#L49) |
| topaz | [`govdao_prop1_topaz.gno:49`](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/topaz.gno.land/transactions/base/bootstrap/govdao_prop1_topaz.gno#L49) |
</details>

## misc/govdao-scripts/extend-govdao-t1.sh:59 [gh](https://github.com/gnolang/gno/blob/24d230fc9/misc/govdao-scripts/extend-govdao-t1.sh#L59) · [↗](../../../../../.worktrees/gno-review-6165/misc/govdao-scripts/extend-govdao-t1.sh#L59)
`&memberstore.Member{InvitationPoints: 3}` allocates [`memberstore`'s own type](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L17-L19), which the run realm may not do, so the transaction aborts before seating anyone.

```suggestion
		if err := ms.SetMember(memberstore.T1, r.addr, memberstore.NewMember(3)); err != nil {
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6165 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/roster_open.txtar <<'TXTAR'
loadpkg gno.land/r/gov/dao
loadpkg gno.land/r/gov/dao/v3/impl
loadpkg gno.land/r/gov/dao/v3/loader $WORK/loader

gnoland start

# Current behaviour at 24d230fc9: the whole MsgRun aborts, nothing is seated.
! gnokey maketx run -gas-fee 5000000ugnot -gas-wanted 100000000 -chainid=tendermint_test test1 $WORK/run/extend_govdao.gno
stderr 'cannot allocate gno.land/r/gov/dao/v3/memberstore.Member in realm gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run'

# Post-fix behaviour, measured with memberstore.NewMember(3) in place of the
# composite literal. Uncomment these four and drop the two lines above once the
# fix lands.
# gnokey maketx run -gas-fee 5000000ugnot -gas-wanted 100000000 -chainid=tendermint_test test1 $WORK/run/extend_govdao.gno
# stdout 'seat Jae as T1'
# stdout 'skip Manfred -- already T1'
# stdout OK!

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

	// Manfred is on the roster the script seats, so his entry is what
	// exercises the skip branch.
	memberstore.Get(0, cur).SetMember(memberstore.T1, address("g1manfred47kzduec920z88wfr64ylksmdcedlf5"), memberstore.NewMember(3))

	// AllowedDAOs left empty: the genesis bootstrap window, where
	// InAllowedDAOs returns true for any caller.
	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(impl.GetInstance(0, cur), nil))
}
-- run/extend_govdao.gno --
package main

import (
	"gno.land/r/gov/dao/v3/memberstore"
)

type rosterEntry struct {
	name string
	addr address
}

// t1Roster is the full target T1 membership, deliberately signer-agnostic:
// the signer is necessarily already a T1 member (that is what authorizes this
// MsgRun), so their own entry is filtered out at runtime instead of being
// hardcoded out of the list.
var t1Roster = []rosterEntry{
	{"Jae", "g1ecsuj0q572jr0dhu29q9njtnmw03hyu7tyyvv6"},
	{"Morgan", "g1m0rgan0rla00ygmdmp55f5m0unvsvknluyg2a4"},
	{"Aeddi", "g1aeddlftlfk27ret5rf750d7w5dume3kcsm8r8m"},
	{"Dongwon", "g1gzhj234kpajz963z5vf42j4ylddscnkez2wvly"},
	{"Maxwell", "g127l4gkhk0emwsx5tmxe96sp86c05h8vg5tufzq"},
	{"Milos", "g1e6gxg5tvc55mwsn7t7dymmlasratv7mkv0rap2"},
	{"Manfred", "g1manfred47kzduec920z88wfr64ylksmdcedlf5"},
}

func main(cur realm) {
	ms := memberstore.Get(0, cur)
	for _, r := range t1Roster {
		// SetMember errors out if the address already sits in any tier, and a
		// single error would abort the whole transaction. Skip instead: this is
		// what drops the signer's own entry, and it makes reruns idempotent.
		if _, tier := ms.GetMember(r.addr); tier != "" {
			println("skip " + r.name + " -- already " + tier)
			continue
		}
		if err := ms.SetMember(memberstore.T1, r.addr, &memberstore.Member{InvitationPoints: 3}); err != nil {
			panic(err.Error())
		}
		println("seat " + r.name + " as T1")
	}
}
TXTAR

go test -v ./gno.land/pkg/integration/ -run 'TestTestdata/roster_open'
rm gno.land/pkg/integration/testdata/roster_open.txtar
```

Even inside the genesis window, with `AllowedDAOs` still empty, the run aborts:

```
> ! gnokey maketx run -gas-fee 5000000ugnot -gas-wanted 100000000 -chainid=tendermint_test test1 $WORK/run/extend_govdao.gno
"gnokey" error: --= Error =--
Data: cannot allocate gno.land/r/gov/dao/v3/memberstore.Member in realm gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run
# ...
main at gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run/extend_govdao.gno:0
--- PASS: TestTestdata/roster_open (2.41s)
```

[`NewMember`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L21) allocates inside `memberstore`, and it is what the genesis scripts on [pearl](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/pearl.gno.land/transactions/base/bootstrap/govdao_prop1_pearl.gno#L44) and [test13](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/test13.gno.land/transactions/base/bootstrap/govdao_prop1_test13.gno#L97) write in the same slot. With it in place the same run prints six `seat` lines, `skip Manfred -- already T1`, and `OK!`.
</details>

## SKIP misc/govdao-scripts/extend-govdao-t1.sh:36-37 [gh](https://github.com/gnolang/gno/blob/24d230fc9/misc/govdao-scripts/extend-govdao-t1.sh#L36-L37) · [↗](../../../../../.worktrees/gno-review-6165/misc/govdao-scripts/extend-govdao-t1.sh#L36)
Nit: nothing reads the signer's tier, so a run inside the genesis window seats the whole roster for a key holding no membership.
Skipped: a finding about a code comment's own wording changes no behaviour.
