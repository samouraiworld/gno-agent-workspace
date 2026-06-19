# Review: PR #5641
Event: APPROVE

## Body
Verified the output is byte-identical to master, not just to the PR's own goldens: ran the fixture corpus against master's pre-refactor `ProtectedString` (all 52 match), and the wide-print abort fires at `location: stream output`, so the print itself is what's metered. No code blockers. The one item left is a maintainer's conscious sign-off on the consensus gas-schedule change you flagged; it reads as deterministic and net-positive to me.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5641-stream-print-metered-writer/1-8e90f07e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

Repros run at 8e90f07e.

## gno.land/pkg/integration/testdata/print_wide_value_gas_metering.txtar:14 [↗](../../../../../.worktrees/gno-review-5641/gno.land/pkg/integration/testdata/print_wide_value_gas_metering.txtar#L14)
The assertion `(out of gas|allocation limit exceeded)` doesn't pin what the test documents: it would pass even if the slice make tripped a limit before any printing happened. Assert `location: stream output` so the test actually proves the print path is what's metered.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5641 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/oog_loc.txtar <<'EOF'
gnoland start
! gnokey maketx run -max-deposit 2000000000ugnot -gas-fee 10000000ugnot -gas-wanted 9000000 -chainid=tendermint_test test1 $WORK/script/script.gno
stderr '.'
-- script/script.gno --
package main
func main() {
	xs := make([]int, 1_000_000)
	println(xs)
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/oog_loc' -v 2>&1 | grep -iE "location|gasUsed"
rm gno.land/pkg/integration/testdata/oog_loc.txtar
```

```
deliver transaction failed: log:out of gas, gasWanted: 9000000, gasUsed: 9000164 location: stream output
```
</details>

## gnovm/pkg/gnolang/uverse.go:1592-1593 [↗](../../../../../.worktrees/gno-review-5641/gnovm/pkg/gnolang/uverse.go#L1592)
The comment points at lines ~920 and ~942, but `formatUverseOutput`'s only callers are at 1205 and 1227.
