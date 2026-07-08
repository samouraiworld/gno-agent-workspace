# Review: PR [#5732](https://github.com/gnolang/gno/pull/5732)
Event: APPROVE

## Body
Looks good. Verified on b6b3e5d42, checks the suite does not run:

- The nil-pointer value-method message matches Go: `go run` of the same program prints `value method main.T.F called using nil *T pointer`, the golden in [`ptr11c.gno`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/tests/files/ptr11c.gno) [↗](../../../../../.worktrees/gno-review-5732/gnovm/tests/files/ptr11c.gno).
- A recovered `.runtimeError` survives a store round-trip: a realm assigns a recovered divide-by-zero to a package-level `error`, and a later transaction reads it back as `runtime error: division by zero`.
- `recover().(string)` now returns ok=false for a VM panic, matching Go, and no in-tree realm asserts `.(string)` on one.

One doc note: point 3 of the [PR description](https://github.com/gnolang/gno/pull/5732) still says only one of about 20 sites is migrated, but the code migrates all of them. Update it so it doesn't read as incomplete.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5732-typedruntimeerror-runtime-errors/2-b6b3e5d42/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/tests/files/recover26.gno:1 [↗](../../../../../.worktrees/gno-review-5732/gnovm/tests/files/recover26.gno#L1)
Missing test: a recovered `.runtimeError` stored in realm state and read back in a later block. recover26.gno covers `recover().(error)` in one VM run, not across a store round-trip.

<details><summary>test cases</summary>

Verified passing on b6b3e5d42. Add under `gno.land/pkg/integration/testdata/`:

```
loadpkg gno.land/r/test/rerr $WORK/rerr

adduserfrom user1 'source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast' 1
stdout 'g18e22n23g462drp4pyszyl6e6mwxkaylthgeeq4'

gnoland start

gnokey maketx call -pkgpath gno.land/r/test/rerr -func TriggerAndStore -gas-fee 1000000ugnot -gas-wanted 4100000 -chainid=tendermint_test user1
stdout OK!

gnokey maketx call -pkgpath gno.land/r/test/rerr -func Read -gas-fee 1000000ugnot -gas-wanted 4100000 -chainid=tendermint_test user1
stdout 'runtime error: division by zero'
stdout OK!

-- rerr/gnomod.toml --
module = "gno.land/r/test/rerr"
gno = "0.9"

-- rerr/rerr.gno --
package rerr

var stored error

func TriggerAndStore(cur realm) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			stored = e
		}
	}()
	a, b := 1, 0
	_ = a / b
}

func Read(cur realm) string {
	if stored == nil {
		return "nil"
	}
	return stored.Error()
}
```
</details>
