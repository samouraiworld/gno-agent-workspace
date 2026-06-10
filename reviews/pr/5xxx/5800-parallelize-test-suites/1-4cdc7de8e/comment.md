# Review: PR #5800
Posted: https://github.com/gnolang/gno/pull/5800#pullrequestreview-4471795728
Event: REQUEST_CHANGES

## Body
Solid design and a real CI win. Two blockers, both reproduced on the current head (13064df):

- `go test -race ./gnovm/pkg/gnolang`: 0 races on master, 27 on this branch. Culprits: the global `enabled` debug bool and an `*Allocator` shared across pool stores. Repro below, breakdown in the full review.
- `gno test -jobs -1` hangs forever with no output. Inline comment with fix and repro.

Minor, inline: the `-jobs` parallel path has zero test coverage; in-order output buffering nit. Also `-update-golden-tests` isn't forced sequential with `-jobs >1`, unlike `TestFiles` (`poolSize=1` under `*withSync`); flagging in case unintentional.

<details><summary>race repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5800 -R gnolang/gno
GNOROOT=$PWD go test -race -short -run 'TestFiles/a' ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'DATA RACE'
```
```
27
```
Same invocation on master:
```
0
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5800-parallelize-test-suites/1-4cdc7de8e/claude-opus-4-8_davd-gzl.md

*(AI Agent)*

## gnovm/cmd/gno/test.go:281-285 [↗](../../../../../.worktrees/gno-review-5800/gnovm/cmd/gno/test.go#L281)
This guard only catches `jobs == 0`: a negative value slips through, spawns zero workers, and the collector waits forever. Reject `jobs < 0` here.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5800 -R gnolang/gno
GNOROOT=$PWD timeout 25 go run ./gnovm/cmd/gno test -C examples -jobs -1 \
  ./gno.land/p/moul/once/ ./gno.land/p/moul/fifo/
echo "exit=$?"  # 124 = hung (timed out)
```
```
exit=124
```
</details>

*(AI Agent)*

## gnovm/cmd/gno/test.go:277-285 [↗](../../../../../.worktrees/gno-review-5800/gnovm/cmd/gno/test.go#L277)
The parallel path has no test (existing txtars all run the default `-jobs 1`). Two txtar cases in `testdata/test/` would cover it: run a few fixture packages with `-jobs 4` and assert the same package-ordered output a sequential run prints; run `-jobs -1` and assert an immediate error instead of the current hang.

*(AI Agent)*

## gnovm/cmd/gno/test.go:338-349 [↗](../../../../../.worktrees/gno-review-5800/gnovm/cmd/gno/test.go#L338)
Nit: output drains in index order, so a slow first package buffers everything behind it in memory. Fine tradeoff for determinism.

*(AI Agent)*
