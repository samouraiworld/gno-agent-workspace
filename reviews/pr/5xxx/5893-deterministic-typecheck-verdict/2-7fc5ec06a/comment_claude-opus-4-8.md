# Review: PR [#5893](https://github.com/gnolang/gno/pull/5893)
Event: REQUEST_CHANGES

## Body
The error-coarsening half lands. The `GoVersion` pin does not, for a root cause the comments below can't carry on their own: it constrains the `types.Config`, but `go/types` lets each file declare its own language version. That per-file version overrides the config and is resolved against the toolchain that compiled the binary. So the gate is settable from inside the package it is meant to gate, and clearing `ast.File.GoVersion` at parse time closes every symptom below at once.

Verified on 7fc5ec06a: two Go toolchains go.mod admits give the same submitted package opposite verdicts through `VMKeeper.AddPackage`, so they disagree on state and not only on the results hash. Removing the line that sets `GoVersion: "go1.18"` makes this build accept `for range 10`, so the pin does fix the untagged case.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5893-deterministic-typecheck-verdict/2-7fc5ec06a/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/gotypecheck.go:190 [↗](../../../../../.worktrees/gno-review-5893/gnovm/pkg/gnolang/gotypecheck.go#L190)
A `//go:build go1.N` line in a submitted file overrides this pin, so the verdict stays a function of the toolchain that built the binary, and [go.mod](https://github.com/gnolang/gno/blob/7fc5ec06a/go.mod#L3) · [↗](../../../../../.worktrees/gno-review-5893/go.mod#L3) admits both go1.25.9 and go1.26.5. `go/types` sets the file's version to `max(fileVersion, go1.21)` with no gate on `Config.GoVersion` and rejects any file version above its own toolchain, stdlib `go/parser` fills `ast.File.GoVersion` from the build line, and [`prepareGoGno0p9`](https://github.com/gnolang/gno/blob/7fc5ec06a/gnovm/pkg/gnolang/gotypecheck.go#L367) · [↗](../../../../../.worktrees/gno-review-5893/gnovm/pkg/gnolang/gotypecheck.go#L367) does not strip it. Clearing `ast.File.GoVersion` on every parsed `.gno` file in [`GoParseMemPackage`](https://github.com/gnolang/gno/blob/7fc5ec06a/gnovm/pkg/gnolang/gotypecheck.go#L587) · [↗](../../../../../.worktrees/gno-review-5893/gnovm/pkg/gnolang/gotypecheck.go#L587) closes both halves, since build constraints carry no meaning in Gno.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5893 -R gnolang/gno

cat > gno.land/pkg/sdk/vm/zz_fork_test.go <<'EOF'
package vm

import (
	"runtime"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestZZFork(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	const pkgPath = "gno.land/r/d2"
	body := "//go:build go1.26\n\npackage d2\n\nfunc Add(a, b int) int { return a + b }\n"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "test.gno", Body: body},
	}
	err := env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files))

	verdict := "ACCEPTED - package deployed, state changed"
	if err != nil {
		verdict = "REJECTED - " + err.Error()
	}
	res := bft.ABCIResult{Error: sdk.ABCIError(err)}
	t.Logf("built with %s => %s", runtime.Version(), verdict)
	t.Logf("built with %s => hashed ABCIResult.Bytes()=%x", runtime.Version(), res.Bytes())
}
EOF

GOTOOLCHAIN=go1.26.5 go test -count=1 -v -run 'TestZZFork$' ./gno.land/pkg/sdk/vm/ 2>&1 | grep 'built with'
GOTOOLCHAIN=go1.25.9 go test -count=1 -v -run 'TestZZFork$' ./gno.land/pkg/sdk/vm/ 2>&1 | grep 'built with'

rm gno.land/pkg/sdk/vm/zz_fork_test.go
```

```
    zz_fork_test.go:38: built with go1.26.5 => ACCEPTED - package deployed, state changed
    zz_fork_test.go:39: built with go1.26.5 => hashed ABCIResult.Bytes()=
    zz_fork_test.go:38: built with go1.25.9 => REJECTED - invalid gno package; type check failed
    zz_fork_test.go:39: built with go1.25.9 => hashed ABCIResult.Bytes()=0a140a122f766d2e54797065436865636b4572726f72
```

Same commit, same package, two Go releases go.mod admits: one node deploys it, the other rejects it.
</details>

## gnovm/pkg/gnolang/gotypecheck.go:187-189 [↗](../../../../../.worktrees/gno-review-5893/gnovm/pkg/gnolang/gotypecheck.go#L187)
The "here rather than downstream" claim does not hold for a tagged file: `//go:build go1.22` plus `for range 10` passes the gate and dies in the preprocessor with `range iteration requires map, string, array, slice, or pointer to array; got BigintKind`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5893 -R gnolang/gno

cat > gnovm/pkg/gnolang/zz_tag_test.go <<'EOF'
package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestZZTag(t *testing.T) {
	for _, body := range []string{
		"package z\nfunc F() { for range 10 {} }\n",
		"//go:build go1.22\n\npackage z\nfunc F() { for range 10 {} }\n",
	} {
		mp := &std.MemPackage{
			Type:  MPUserProd,
			Name:  "z",
			Path:  "gno.land/p/demo/z",
			Files: []*std.MemFile{{Name: "z.gno", Body: body}},
		}
		_, err := TypeCheckMemPackage(mp, TypeCheckOptions{Mode: TCLatestRelaxed})
		t.Logf("type-check => %v", err)
	}
}
EOF

go test -count=1 -v -run 'TestZZTag$' ./gnovm/pkg/gnolang/ 2>&1 | grep 'type-check =>'

rm gnovm/pkg/gnolang/zz_tag_test.go
```

```
    zz_tag_test.go:21: type-check => gno.land/p/demo/z/z.gno:2:22: cannot range over 10 (untyped int constant): requires go1.22 or later
    zz_tag_test.go:21: type-check => <nil>
```

One comment line and the pinned gate stops applying.
</details>

## gno.land/pkg/sdk/vm/pb3_gen.go:857-861 [↗](../../../../../.worktrees/gno-review-5893/gno.land/pkg/sdk/vm/pb3_gen.go#L857)
Dropping field 1 means a `TypeCheckError` written by a pre-upgrade node no longer decodes, and [`LoadABCIResponses`](https://github.com/gnolang/gno/blob/7fc5ec06a/tm2/pkg/bft/state/store.go#L188-L195) · [↗](../../../../../.worktrees/gno-review-5893/tm2/pkg/bft/state/store.go#L188) answers that unmarshal failure with `osm.Exit`, so a node upgraded across this change hard-exits when it reads back a height at which a type-check rejection was recorded. Those responses are saved every block and never pruned, and [`/block_results`](https://github.com/gnolang/gno/blob/7fc5ec06a/tm2/pkg/bft/rpc/core/blocks.go#L144) · [↗](../../../../../.worktrees/gno-review-5893/tm2/pkg/bft/rpc/core/blocks.go#L144) takes a caller-chosen height. The field needs skipping rather than erroring if any chain carries such stored results across the coordinated release.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5893 -R gnolang/gno

cat > gno.land/pkg/sdk/vm/zz_wire_test.go <<'EOF'
package vm

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
)

func TestZZWire(t *testing.T) {
	// Field 1 (tag 0x0a), length, then the diagnostic: what a pre-PR node wrote.
	s := "a.gno:1:1: undefined: foo"
	old := append([]byte{0x0a, byte(len(s))}, []byte(s)...)
	var tce TypeCheckError
	t.Logf("decode pre-PR bytes => err=%v", amino.Unmarshal(old, &tce))
}
EOF

go test -count=1 -v -run 'TestZZWire$' ./gno.land/pkg/sdk/vm/ 2>&1 | grep 'decode pre-PR'

rm gno.land/pkg/sdk/vm/zz_wire_test.go
```

```
    zz_wire_test.go:16: decode pre-PR bytes => err=unknown field number 1 for TypeCheckError
```
</details>

## gnovm/pkg/gnolang/gotypecheck_test.go:469-495 [↗](../../../../../.worktrees/gno-review-5893/gnovm/pkg/gnolang/gotypecheck_test.go#L469)
Missing test: CI stays green for any pin in [go1.18, go1.21], and go1.21 admits `min`/`max`/`clear`, which the GnoVM's uverse has no entry for. No case type-checks a package with an import either, so the pin covers dependencies only by virtue of the shared `types.Config` and nothing would catch a refactor that gives the importer its own.

<details><summary>test cases</summary>

```go
// Upper bound: nothing above go1.18 may be accepted.
assert.ErrorContains(t, tc("package z\nfunc F() int { return min(1, 2) }\n"), "go1.21")
assert.ErrorContains(t, tc("package z\nfunc F() int { return max(1, 2) }\n"), "go1.21")
assert.ErrorContains(t, tc("package z\nfunc F() { m := map[int]int{}; clear(m) }\n"), "go1.21")
assert.ErrorContains(t, tc("package z\nfunc F() { for range 10 {} }\n"), "go1.22")

// Lower bound: the .gnobuiltins shim's `func revive[F any](fn F) any` needs go1.18.
assert.NoError(t, tc("package z\nfunc F(x any) any { return x }\n"))
```

Full file, including the imported-package guard: [`gotypecheck_pin_exact_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5893-deterministic-typecheck-verdict/2-7fc5ec06a/tests/gotypecheck_pin_exact_test.go) · [↗](tests/gotypecheck_pin_exact_test.go). Both pass at 7fc5ec06a; the first goes red if the pin moves to go1.21, which the committed test does not catch.
</details>
