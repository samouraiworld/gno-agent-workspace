# Review: [#6132](https://github.com/gnolang/gno/pull/6132)
Event: REQUEST_CHANGES

## Body
This `init()` runs at genesis, so it lands on mainnet at launch, whatever the beta network `gnoland1` already carries.

## examples/gno.land/r/gnoland/boards2/v1/boards.gno:69 [gh](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L69) · [↗](../../../../../.worktrees/gno-review-6132/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L69)
`CreateRepost` writes a caller-chosen title and body into the open board [under a check](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/public.gno#L340) that never reads `RequiredAccountAmount`, the 3,000 GNOT gate that [two](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/permissions.gno#L157-L158) of the board's [three public permissions](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/permissions.gno#L111-L115) carry. Give `PermissionThreadRepost` the same validator, or seed the board without it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6132 -R gnolang/gno
D=examples/gno.land/r/gnoland/zzrepost
mkdir -p $D
cat > $D/gnomod.toml <<'EOF'
module = "gno.land/r/gnoland/zzrepost"
gno = "0.9"
EOF
echo 'package zzrepost' > $D/zzrepost.gno
cat > $D/z_repost_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/gnoland/zzrepost/z_repost_filetest
package z_repost_filetest

import (
	"testing"

	boards2 "gno.land/r/gnoland/boards2/v1"
)

const (
	owner address = "g1skl80cuz8zq3lul9pgz5pc35l2pfzgxgfpsqkx"
	user  address = "g1us8428u2a5satrlxzagqqa5m6vmuze025anjlj"
)

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	boards2.CreateThread(cross(cur), 1, "Seed", "Seed body")
}

func main(cur realm) {
	// user was never issued any ugnot.
	testing.SetRealm(testing.NewUserRealm(user))
	println("threads before:", len(boards2.GetThreads(1, 0, 50)))
	println("repost id:", boards2.CreateRepost(cross(cur), 1, 1, 1, "Spam title", "Spam body"))
	println("threads after:", len(boards2.GetThreads(1, 0, 50)))
}

// Output:
// threads before: 1
// repost id: 2
// threads after: 2
EOF
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gnoland/zzrepost
rm -rf $D
```

The pass is the finding: the balance check never runs and the second thread lands.

```
threads before: 1
repost id: 2
threads after: 2
=== RUN   ./gno.land/r/gnoland/zzrepost/z_repost_filetest.gno
--- PASS: ./gno.land/r/gnoland/zzrepost/z_repost_filetest.gno (elapsed: 0.07s, gas: 3821479, storage: gno.land/r/gnoland/boards2/v1:+15104b)
ok      ./gno.land/r/gnoland/zzrepost 	16.96s
```

Swapping `CreateRepost` for `CreateThread` in that same file gives `caller is not allowed to create threads: account amount is lower than 3000 GNOT`, which the realm's own `z_create_thread_06_filetest.gno` already pins. Removing a thread its author keeps then takes the [GovDAO multisig](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/permissions.gno#L153), the board's only owner.
</details>

## examples/gno.land/r/gnoland/boards2/v1/filetests/z_ui_home_02_filetest.gno:17 [gh](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/filetests/z_ui_home_02_filetest.gno#L17) · [↗](../../../../../.worktrees/gno-review-6132/examples/gno.land/r/gnoland/boards2/v1/filetests/z_ui_home_02_filetest.gno#L17)
Nit: this file's rewritten output was the last cover of the empty-state branch at [`render.gno:131-135`](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/render.gno#L131-L135), which `init()` makes unreachable by seeding a listed board that nothing removes from [`gListedBoardsByID`](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/public.gno#L176). Both go: the branch, and this file's opening line still reading "when there are no boards".

## examples/gno.land/r/gnoland/boards2/v1/boards.gno:68 [gh](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L68) · [↗](../../../../../.worktrees/gno-review-6132/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L68)
Suggestion: an account under 3,000 GNOT still needs a multisig transaction before its first thread here, since [`validateOpenThreadCreate`](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/permissions_validators_open.gno#L111-L127) exempts only owners and admins from [`RequiredAccountAmount`](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L25). Set that variable in the same `init()`; the cost is two goldens, `z_create_thread_06_filetest.gno` and `z_create_reply_15_filetest.gno`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6132 -R gnolang/gno
D=examples/gno.land/r/gnoland/zzgate
mkdir -p $D
cat > $D/gnomod.toml <<'EOF'
module = "gno.land/r/gnoland/zzgate"
gno = "0.9"
EOF
echo 'package zzgate' > $D/zzgate.gno
cat > $D/z_gate_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/gnoland/zzgate/z_gate_filetest
package z_gate_filetest

import (
	"chain"
	"testing"

	boards2 "gno.land/r/gnoland/boards2/v1"
)

const user address = "g1us8428u2a5satrlxzagqqa5m6vmuze025anjlj"

func main(cur realm) {
	println("boards after init:", boards2.BoardCount())
	testing.IssueCoins(user, chain.Coins{{"ugnot", 100_000_000}})
	testing.SetRealm(testing.NewUserRealm(user))
	println("thread", boards2.CreateThread(cross(cur), 1, "hello", "body"))
}

// Output:
// boards after init: 1
// thread 1
EOF
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gnoland/zzgate 2>&1 | head -8
rm -rf $D
```

The account holds 100 GNOT and `init()`'s board is the only one:

```
boards after init: 1
=== RUN   ./gno.land/r/gnoland/zzgate/z_gate_filetest.gno
--- FAIL: ./gno.land/r/gnoland/zzgate/z_gate_filetest.gno (elapsed: 0.14s, gas: 3452935)
unexpected panic: caller is not allowed to create threads: account amount is lower than 3000 GNOT
output:
boards after init: 1
```

Issue the account `3_000_000_000ugnot` instead, and the run goes green with `thread 1`.
</details>

## examples/gno.land/r/gnoland/boards2/v1/public.gno:157 [gh](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/r/gnoland/boards2/v1/public.gno#L157) · [↗](../../../../../.worktrees/gno-review-6132/examples/gno.land/r/gnoland/boards2/v1/public.gno#L157)
Suggestion: the helper overwrites every field but the ID of the board it receives. Take `id boards.ID` and call `boards.New(id)` inside, dropping `boards.New` from both call sites.

<details><summary>why the type matters</summary>

[`storage.Add`](https://github.com/gnolang/gno/blob/02ac71476/examples/gno.land/p/gnoland/boards/storage.gno#L99-L112) is a `Set` whose only error is a nil board, so a board carrying an ID already in the tree replaces that board and its threads. Both call sites draw from `gBoardsSequence.Next()`, so nothing reaches it today; the `*boards.Board` parameter is what would let a third caller get there.
</details>
