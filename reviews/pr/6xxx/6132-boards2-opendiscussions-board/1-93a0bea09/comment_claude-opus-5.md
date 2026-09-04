# Review: [#6132](https://github.com/gnolang/gno/pull/6132)
Event: REQUEST_CHANGES

## Body
A realm's source is fixed once deployed, so this `init()` reaches a chain only at its genesis. The live `gnoland1` chain already carries an [`OpenDiscussions` board at ID 1](https://gno.land/r/gnoland/boards2/v1) beside `atomone-governance` at ID 2.

## examples/gno.land/r/gnoland/boards2/v1/boards.gno:14 [gh](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L14) · [↗](../../../../../.worktrees/gno-review-6132/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L14)
This swap leaves the old GovDAO T1 address in the four other sources hardcoding it, including the one-shot `Enable()` gate at [`r/sys/names/verifier.gno:59`](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/sys/names/verifier.gno#L59). [PR 6131](https://github.com/gnolang/gno/pull/6131) moves all five and holds until the signer set is settled, so drop ddaf67802 and 93a0bea09 here.

## examples/gno.land/r/gnoland/boards2/v1/boards.gno:69 [gh](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L69) · [↗](../../../../../.worktrees/gno-review-6132/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L69)
Suggestion: this open board refuses a thread from a non-member holding under 3,000 GNOT, so no ordinary account can post until a multisig transaction lowers the threshold. Assigning [`RequiredAccountAmount`](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L25) a lower value in the same `init()` avoids that.

<details><summary>repro</summary>

A throwaway realm asserts what the PR description promises: a funded ordinary account posts on the new board.

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

The account holds 100 GNOT, and `init()`'s board is the only one:

```
boards after init: 1
=== RUN   ./gno.land/r/gnoland/zzgate/z_gate_filetest.gno
--- FAIL: ./gno.land/r/gnoland/zzgate/z_gate_filetest.gno (elapsed: 0.05s, gas: 3452935)
unexpected panic: caller is not allowed to create threads: account amount is lower than 3000 GNOT
output:
boards after init: 1

stacktrace:
```

Change the balance in that file to `3_000_000_000` and nothing else, and the run goes green with `thread 1`. The same threshold is already live under the name `gOpenAccountAmount`, on an [`OpenDiscussions` board](https://gno.land/r/gnoland/boards2/v1:OpenDiscussions) that has shown zero threads since 2026-03-21. [`validateOpenThreadCreate`](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/gnoland/boards2/v1/permissions_validators_open.gno#L111-L127) exempts owners and admins only, and an invited guest is refused a thread while [`validateOpenReplyCreate`](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/gnoland/boards2/v1/permissions_validators_open.gno#L138-L154) lets the same guest reply.
</details>

## examples/gno.land/r/gnoland/boards2/v1/filetests/z_ui_home_02_filetest.gno:17 [gh](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/gnoland/boards2/v1/filetests/z_ui_home_02_filetest.gno#L17) · [↗](../../../../../.worktrees/gno-review-6132/examples/gno.land/r/gnoland/boards2/v1/filetests/z_ui_home_02_filetest.gno#L17)
Nit: rewriting this file's expected output drops the last cover of the empty-state branch at [`render.gno:131-135`](https://github.com/gnolang/gno/blob/93a0bea09/examples/gno.land/r/gnoland/boards2/v1/render.gno#L131-L135), which `init()` makes unreachable by seeding a listed board nothing removes from `gListedBoardsByID`.
