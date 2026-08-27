# Review: PR [#5576](https://github.com/gnolang/gno/pull/5576)
Event: REQUEST_CHANGES
Status: not posted. On `post as an AI` the Body leads with `[AI review, not manually checked, opus 4.8 xhigh]`, then `Status: REQUEST_CHANGES`.

## Body
Verified on 79c02d050: two runs at a fixed `-benchcount` produce byte-identical result lines, so the determinism the design promises holds. The same loop reports 27159 cycles/op at N=1 and 1231 at N=1000 on one machine, so the default N=1 run measures setup, not steady state.

## gnovm/cmd/gno/test.go:159-164 [gh](https://github.com/gnolang/gno/blob/79c02d050/gnovm/cmd/gno/test.go#L159-L164) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/cmd/gno/test.go#L159)
The PR body promises `b.N` doubling to a `-benchcycles` target, but no such flag exists and `-benchcount` is used directly as a fixed `b.N`. With the default `b.N` of 1, `gno test -bench .` reports one-time setup as the per-op cost. Implement the scaling, or change the body to describe the fixed-count behavior.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5576 -R gnolang/gno
export GNOROOT=$(git rev-parse --show-toplevel)
mkdir -p /tmp/bb && cd /tmp/bb
cat > gnomod.toml <<'EOF'
module = 'gno.test/p/bb'
EOF
cat > bb.gno <<'EOF'
package bb

var Sink int
EOF
cat > bb_test.gno <<'EOF'
package bb

import "testing"

func BenchmarkLoop(b *testing.B) {
	total := 0
	for i := 0; i < b.N; i++ {
		total += i * 3
	}
	Sink = total
}
EOF
go run "$GNOROOT/gnovm/cmd/gno" test -bench . -benchcount 1 .
go run "$GNOROOT/gnovm/cmd/gno" test -bench . -benchcount 1000 .
cd - && rm -rf /tmp/bb
```

```
BenchmarkLoop	       1	       27159 cycles/op	       27502 gas/op
ok      . 	0.97s
BenchmarkLoop	    1000	        1231 cycles/op	        1232 gas/op
ok      . 	0.93s
```
</details>

## gnovm/pkg/test/test_test.go:160-168 [gh](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test_test.go#L160-L168) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test_test.go#L160)
The subtests run with `t.Parallel()` and share the package-global regex cache in `matchString`, which has no synchronization. The race makes `anchored_end_matches_exact_suffix` fail intermittently. Drop `t.Parallel()` from the subtests.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5576 -R gnolang/gno
go test ./gnovm/pkg/test/ -run TestShouldRun -race -count=1
for i in $(seq 1 40); do
  go test ./gnovm/pkg/test/ -run TestShouldRun -count=1 2>&1 | grep -q FAIL && { echo "flaked on run $i"; break; }
done
```

```
WARNING: DATA RACE
Write at ... by goroutine 32:
  ...test.matchString()  gnovm/pkg/test/util_match.go:176
  ...test.shouldRun()    gnovm/pkg/test/test.go:847
  ...test.TestShouldRun.func1()  gnovm/pkg/test/test_test.go:167
--- FAIL: TestShouldRun/anchored_end_matches_exact_suffix
    expected: true
    actual  : false
flaked on run 4
```
</details>

## gnovm/pkg/test/test.go:794-795 [gh](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test.go#L794-L795) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test.go#L794)
`rep.Allocs/n` truncates to 0 allocs/op when total allocations are fewer than N, while `B/op` stays nonzero, so the output reports bytes allocated with zero objects. At the default N=1 this is the common case.

## gnovm/pkg/test/test.go:391 [gh](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test.go#L391) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test.go#L391)
"materialised" is British spelling; the rest of the codebase uses American English.

## gnovm/pkg/gnolang/alloc.go:268-273 [gh](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/gnolang/alloc.go#L268-L273) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/gnolang/alloc.go#L268)
`resetLiveBytesForGC` has no doc comment. Say it zeroes only live bytes so the cumulative benchmark counters survive a GC re-walk, which is what distinguishes it from `Reset()`.

Repros run at 79c02d050.
