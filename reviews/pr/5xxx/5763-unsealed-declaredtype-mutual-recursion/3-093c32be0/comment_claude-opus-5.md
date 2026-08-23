# Review: PR [#5763](https://github.com/gnolang/gno/pull/5763)
Event: REQUEST_CHANGES

## Body
Mutual type-decl recursion resolves correctly for two types across all seven base kinds, in both declaration orders, byte-identical to Go on 51 program pairs. It stops working at three.

- The description's "## Changes" lists two files, and the third is where the fix lives. Dropping the panic alone leaves the dependent observing an empty base; [`types.go`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1545-L1592)'s `fillTypeInPlace` is what resolves the cycle, and it is +49 of the 82 added lines. Someone reading the body approves a one-line revert of a stale assertion and merges a change that makes the predefine-time base pointer canonical for every named type in every package: 2,848 of 3,182 finalizations over `gnovm/tests/files` keep it instead of the one built at finalize. The body's `realm.go:1788` citation is also off by a section, the sealed gate being at [`realm.go:1810`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/realm.go#L1810).

## gnovm/pkg/gnolang/preprocess.go:3077 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3077) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3077)
Critical: this guard skips the fill whenever `dstT.Base` is nil, and the next line writes that nil into the slot. Three types in a cycle reach it, where the dependent finalizes before its source has a base. Nothing downstream asserts otherwise: `gno lint` exits 0 on the package, and the failure resurfaces in `copyTypeWithRefs` with no file or line. Fix: reject a nil `dstT.Base` here with a positioned error.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
mkdir -p examples/gno.land/p/demo/nilbase
cat > examples/gno.land/p/demo/nilbase/gnomod.toml <<'EOF'
module = "gno.land/p/demo/nilbase"
gno = "0.9"
EOF
cat > examples/gno.land/p/demo/nilbase/nilbase.gno <<'EOF'
package nilbase

type T2 T1

type T1 struct {
	Next *T3
	Val  int
}

type T3 T2

func Hello() string { return "hello" }
EOF
cat > examples/gno.land/p/demo/nilbase/nilbase_test.gno <<'EOF'
package nilbase

import "testing"

func TestHello(t *testing.T) {
	if Hello() != "hello" {
		t.Fatal("bad")
	}
}
EOF
cd examples/gno.land/p/demo/nilbase
GNOROOT=$(git rev-parse --show-toplevel) go run $(git rev-parse --show-toplevel)/gnovm/cmd/gno lint . ; echo "lint exit: $?"
GNOROOT=$(git rev-parse --show-toplevel) go run $(git rev-parse --show-toplevel)/gnovm/cmd/gno test . 2>&1 | head -8
cd - >/dev/null && rm -rf examples/gno.land/p/demo/nilbase
```

The lint exit code is the finding: master rejects this package with a positioned error, and here it passes.

```
lint exit: 0
panic: cannot copy nil types [recovered, repanicked]
github.com/gnolang/gno/gnovm/pkg/gnolang.copyTypeWithRefs({0x0?, 0x0?})
	gnovm/pkg/gnolang/realm.go:1501
github.com/gnolang/gno/gnovm/pkg/gnolang.(*defaultStore).SetType(...)
```

The same three types as a filetest are valid Go printing `7 8 9`, and abort here with `runtime error: invalid memory address or nil pointer dereference` out of `elideCompositeElements`, where `baseOf` returns nil and the type switch falls to `default`.
</details>

## gnovm/pkg/gnolang/types.go:1577 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1577) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1577)
`type E error` makes the destination and the source one pointer, so this arm runs `*dst = *dst` on the process-global uverse object that no package owns. Two goroutines preprocessing that declaration race on it. Fix: add the pointer-inequality test the doc comment already assumes, `dstT.Base != nil && dstT.Base != tmp2.Base && fillTypeInPlace(...)`, a no-op in exactly the case that touches a foreign object.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_uverserace_test.go <<'EOF'
package gnolang

import (
	"sync"
	"testing"
)

func TestZZUverseBaseRace(t *testing.T) {
	_ = UverseNode()
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { recover() }()
			m := NewMachine("testdata", nil)
			defer m.Release()
			nn := m.MustParseFile("testdata.gno", "package testdata\ntype E error\n")
			m.RunFiles(nn)
		}()
	}
	wg.Wait()
}
EOF
go test -race -count=1 -run TestZZUverseBaseRace ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'WARNING: DATA RACE'
rm gnovm/pkg/gnolang/zz_uverserace_test.go
```

Four races here against two on the merge-base, and the two new ones name `fillTypeInPlace`. Applying the guard above returns the count to two with no `fillTypeInPlace` frame, and leaves every mutual-recursion filetest passing.

```
4
Write at 0x0000026617d0 by goroutine 29:
  github.com/gnolang/gno/gnovm/pkg/gnolang.fillTypeInPlace()
      gnovm/pkg/gnolang/types.go:1577
  github.com/gnolang/gno/gnovm/pkg/gnolang.preprocess1.func1()
      gnovm/pkg/gnolang/preprocess.go:3077
```
</details>

## gnovm/pkg/gnolang/preprocess.go:5504 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5504) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5504)
Removing this guard widens what a node accepts, so an unupgraded node rejects a `MsgAddPackage` that an upgraded one writes, and there is no height gate or language-version switch on the path. Fix: name the coordinated upgrade in the description. Then confirm no failed `addpkg` of a mutual-recursion package sits in replayed history, since such a block re-executes as a success and hands a syncing node a different AppHash.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
cat > /tmp/mutual.gno <<'EOF'
package mutual

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1
EOF
base=$(git merge-base origin/master HEAD)
for ref in "$base" HEAD; do
  git stash -q -u 2>/dev/null || true
  git checkout -q "$ref"
  printf '%s: ' "$ref"
  mkdir -p examples/gno.land/p/demo/mutual
  cp /tmp/mutual.gno examples/gno.land/p/demo/mutual/mutual.gno
  printf 'module = "gno.land/p/demo/mutual"\ngno = "0.9"\n' > examples/gno.land/p/demo/mutual/gnomod.toml
  (cd examples/gno.land/p/demo/mutual && GNOROOT=$(git rev-parse --show-toplevel) \
     go run $(git rev-parse --show-toplevel)/gnovm/cmd/gno lint . >/dev/null 2>&1 \
     && echo ACCEPTED || echo REJECTED)
  rm -rf examples/gno.land/p/demo/mutual
done
git checkout -q -
```

The two lines disagree, which is the finding: the same package is rejected by one binary and accepted by the other.

```
0397fc87f…: REJECTED
HEAD: ACCEPTED
```
</details>

## gnovm/tests/files/decltype_mutual.gno:1-19 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/tests/files/decltype_mutual.gno?plain=1#L1-L19) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L1-L19)
Missing test: the six non-struct `fillTypeInPlace` arms. Removing all six leaves the filetest suite unchanged. Removing the `*StructType` arm fails this file, with `struct type struct{} has no field Val`. One arm of seven is pinned, and the other six can regress unnoticed. Fix: add a mutual pair per base kind, slice, map, array, func, pointer and interface.

## gnovm/tests/files/decltype_mutual.gno:17-19 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/tests/files/decltype_mutual.gno?plain=1#L17-L19) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L17-L19)
Missing test: what the cycle writes to state, which is the property the change turns on now that both halves share one `*StructType`. Fix: add a `// PKGPATH:` variant with `// Types:` and `// Realm:` so the two halves' distinct TypeIDs and the persisted object diff are pinned rather than only stdout.

## gnovm/pkg/gnolang/preprocess.go:5508 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5508) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5508)
Suggestion: a forward alias still fails preprocessing here, and the message changes. Go accepts `type A = B` above `type B struct{...}`. Inside a mutual pair, the abort lands on `StaticBlock.Define2(A) cannot change .V` instead of the guard removed here. That user reads a static-block invariant's name, not their own type's. Fix: call the forward alias out of scope in the description, or land a filetest pinning the ordering.
