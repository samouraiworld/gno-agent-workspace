# Review: [#6401](https://github.com/gnolang/gno/pull/6401)
Event: COMMENT
Verdict: APPROVE. Nothing blocks: the open items are a missing test for the settled re-check after verification, two Nits on the restart detector and the import seed's error path, and two Suggestions on what the cache and the pre-check reach.
Model: claude-opus-5-5, standard review, solo shape: two finders, then one judge and writer
Commit: 5454f8165cee97056ae64ef955c541e2c37d9af2
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6401 5454f8165`
Round: 1. 2 finders, no separate reflector (the judge answered the completeness question), 9 candidates, the Missing test and the Nits re-run by a judge that was not their finder, the Suggestions judged by read; 0 refuted; the one PLAUSIBLE Suggestion, raised by both finders, shipped SKIP. Posted as an AI review, so the event goes out as COMMENT.

## Body
> AI review, claude-opus-5-5, standard review, reviewed at [5454f81](https://github.com/gnolang/gno/pull/6401/commits/5454f8165cee97056ae64ef955c541e2c37d9af2), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6401-gpao-import-cache/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6401-gpao-import-cache/1-5454f8165/claims.md) · Status: APPROVE

## contribs/gpao/oracle.go:570 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/oracle.go#L570) · [↗](../../../../../.worktrees/gno-review-6401/contribs/gpao/oracle.go#L570) · Missing test
Missing test: no test enables the package between the two settled checks, so deleting this `recordIfSettled` leaves the gpao suite green.

<details><summary>mutation</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6401 -R gnolang/gno
cd contribs/gpao
sed -i '570,572d' oracle.go
go test -count=1 .
git checkout oracle.go
```

The suite passes with the re-check gone:

```text
ok  	github.com/gnolang/gno/contribs/gpao	25.354s
```

Every settled case in `issettled_test.go` is answered by the pre-check before verification. A case that needs it enables the path between the two checks, for instance from a stub whose `vm/qpkgmeta_json` answer turns live after the first query, and asserts that no `MsgEnablePackage` is broadcast.
</details>

## contribs/gpao/oracle.go:345 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/oracle.go#L345) · [↗](../../../../../.worktrees/gno-review-6401/contribs/gpao/oracle.go#L345) · Nit
Nit: `latest < height-1` treats a tip one block behind the last read height as a restart, and empties the genesis cache and logs a reset.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6401 -R gnolang/gno
cat > contribs/gpao/judge_tip_lag_test.go <<'EOF'
// The genesis import cache should survive a tip one block behind the last
// read height, and a -start-height above the tip where nothing was read.
// Both fail at the reviewed head: the cache is emptied (kept 0).
package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJudgeTipLagKeepsGenesisImports(t *testing.T) {
	for name, tc := range map[string]struct {
		start int64
		tips  []int64
	}{
		"one-block lag after reading 7": {start: 1, tips: []int64{6, 7, 6, 8}},
		"start height above the tip":    {start: 1000, tips: []int64{500}},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			o := newStubOracle(&resetRPC{tips: tc.tips, cancel: cancel})
			o.cfg.pollInterval = time.Millisecond
			o.cfg.startHeight = tc.start
			o.imports = importCache{"gno.land/p/test/lib": chainPackage("gno.land/p/test/lib", "package lib\n")}

			require.NoError(t, o.run(ctx))
			assert.Len(t, o.genesisImports(), 1)
		})
	}
}
EOF
(cd contribs/gpao && go test -count=1 -run TestJudgeTipLag -v .)
rm contribs/gpao/judge_tip_lag_test.go
```

Both cases fail because the seeded cache is emptied, though neither is a restart:

```text
=== RUN   TestJudgeTipLagKeepsGenesisImports/one-block_lag_after_reading_7
        	Error:      	"map[]" should have 1 item(s), but has 0
=== RUN   TestJudgeTipLagKeepsGenesisImports/start_height_above_the_tip
        	Error:      	"map[]" should have 1 item(s), but has 0
--- FAIL: TestJudgeTipLagKeepsGenesisImports (0.01s)
```
</details>

## SKIP contribs/gpao/oracle.go:345-350 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/oracle.go#L345) · [↗](../../../../../.worktrees/gno-review-6401/contribs/gpao/oracle.go#L345) · Suggestion
Suggestion: the restart check reads tip heights alone, so a reset chain already past the last read height at the next poll keeps the old genesis cached.

Not posted: the stale verdict needs a relaunched chain with a changed genesis behind the same remote, which this round did not build.

## contribs/gpao/verifyone.go:120 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/verifyone.go#L120) · [↗](../../../../../.worktrees/gno-review-6401/contribs/gpao/verifyone.go#L120) · Nit
Nit: a seed error exits 1, which `verify` files as a rejection of the candidate; the child's other non-verdict failures call [`exitNoVerdict`](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/verifyone.go#L147).

## contribs/gpao/importcache.go:55 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/importcache.go#L55) · [↗](../../../../../.worktrees/gno-review-6401/contribs/gpao/importcache.go#L55) · Suggestion
Suggestion: `gm.AddPkg.Height == 0` skips packages a hard-forked genesis replays, which [`app.go`](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/gno.land/pkg/gnoland/app.go#L903-L905) runs at their original height, so each verification fetches them again.

## contribs/gpao/oracle.go:513 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/oracle.go#L513) · [↗](../../../../../.worktrees/gno-review-6401/contribs/gpao/oracle.go#L513) · Suggestion
Suggestion: `recordIfSettled` runs here under a context with no deadline, so a stalled node holds each candidate for the RPC client's [`defaultTimeout`](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/tm2/pkg/bft/rpc/client/client.go#L19) before verification starts.
