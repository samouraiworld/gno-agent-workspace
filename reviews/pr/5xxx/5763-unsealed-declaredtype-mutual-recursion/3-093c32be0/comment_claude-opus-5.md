# Review: PR [#5763](https://github.com/gnolang/gno/pull/5763)
Posted: https://github.com/gnolang/gno/pull/5763#pullrequestreview-5007018155
Event: REQUEST_CHANGES

## Body
[AI review]

Two-type mutual recursion resolves on all seven base kinds and in both declaration orders; three types do not.

- The "## Changes" list names two files, and the third is where the fix lives: [`fillTypeInPlace`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1553) is +49 of the 82 added lines, and it makes the predefine-time base pointer canonical for [every named type](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3077), 2848 of 3182 finalizations across `gnovm/tests/files`, not only recursive ones.

## gnovm/pkg/gnolang/preprocess.go:3077 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3077) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3077) [posted](https://github.com/gnolang/gno/pull/5763#discussion_r3842771891)
Critical: a three-type cycle reaches this guard with `dstT.Base` nil, which skips the fill and lets the next line store that nil, and `gno lint` then exits 0 on the package.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
mkdir -p examples/gno.land/p/demo/nilbase
printf 'module = "gno.land/p/demo/nilbase"\ngno = "0.9"\n' > examples/gno.land/p/demo/nilbase/gnomod.toml
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
ROOT=$(git rev-parse --show-toplevel)
cd examples/gno.land/p/demo/nilbase
GNOROOT=$ROOT go run $ROOT/gnovm/cmd/gno lint . ; echo "lint exit: $?"
GNOROOT=$ROOT go run $ROOT/gnovm/cmd/gno test . 2>&1 | head -6
cd "$ROOT" && rm -rf examples/gno.land/p/demo/nilbase
```

The exit code is the finding: master rejects this package at `nilbase.gno:3:6-11` and here it passes.

```
lint exit: 0
panic: cannot copy nil types [recovered, repanicked]
github.com/gnolang/gno/gnovm/pkg/gnolang.copyTypeWithRefs({0x0?, 0x0?})
	gnovm/pkg/gnolang/realm.go:1501
github.com/gnolang/gno/gnovm/pkg/gnolang.(*defaultStore).SetType(...)
```

The same three types run as a filetest are valid Go printing `7 8 9`, and abort with `invalid memory address or nil pointer dereference` inside [`elideCompositeElements`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5823), where `baseOf` returns nil and the switch falls to `default`. A census over `gnovm/tests/files` counts 3 nil destinations in 3182 calls, so the path is reachable from the stock suite.
</details>

## gnovm/pkg/gnolang/types.go:1577 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1577) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1577) [posted](https://github.com/gnolang/gno/pull/5763#discussion_r3842771904)
`type E error` gives this arm the same pointer for `dst` and `src`, so it writes the process-global [uverse](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/uverse.go#L483) object that no package owns, and two goroutines preprocessing that declaration race on it.

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

Four races here against two on the merge-base, and the new ones name `fillTypeInPlace` at a fixed static address.

```
4
Write at 0x0000026617d0 by goroutine 29:
  github.com/gnolang/gno/gnovm/pkg/gnolang.fillTypeInPlace()
      gnovm/pkg/gnolang/types.go:1577
  github.com/gnolang/gno/gnovm/pkg/gnolang.preprocess1.func1()
      gnovm/pkg/gnolang/preprocess.go:3077
```

The guard proposed below returns the count to two with no `fillTypeInPlace` frame, and leaves `decltype_mutual.gno` and a seven-kind fixture passing.
</details>

## gnovm/pkg/gnolang/preprocess.go:3077-3079 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3077-L3079) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3077) [posted](https://github.com/gnolang/gno/pull/5763#discussion_r3842771909)
This guard is missing the pointer-inequality test the helper's doc comment already assumes, a no-op in every case except the one that writes a foreign object.

```suggestion
						if dstT.Base != nil && dstT.Base != tmp2.Base && fillTypeInPlace(dstT.Base, tmp2.Base) {
							tmp2.Base = dstT.Base
						}
```

## gnovm/pkg/gnolang/preprocess.go:5504 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5504) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5504) [posted](https://github.com/gnolang/gno/pull/5763#discussion_r3842771915)
Dropping this guard widens what a node accepts with no height gate or language-version switch on the path, so a block carrying a failed `MsgAddPackage` of a mutual-recursion package re-executes as a success and hands a syncing node a different AppHash.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
ROOT=$(git rev-parse --show-toplevel)
cat > /tmp/mutual.gno <<'EOF'
package mutual

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1
EOF
for ref in $(git merge-base origin/master HEAD) HEAD; do
  git checkout -q "$ref"
  printf '%s: ' "$ref"
  mkdir -p examples/gno.land/p/demo/mutual
  cp /tmp/mutual.gno examples/gno.land/p/demo/mutual/mutual.gno
  printf 'module = "gno.land/p/demo/mutual"\ngno = "0.9"\n' > examples/gno.land/p/demo/mutual/gnomod.toml
  (cd examples/gno.land/p/demo/mutual && GNOROOT=$ROOT go run $ROOT/gnovm/cmd/gno lint . >/dev/null 2>&1 \
     && echo ACCEPTED || echo REJECTED)
  rm -rf examples/gno.land/p/demo/mutual
done
git checkout -q -
```

The disagreement is the finding: one binary rejects the package the other writes to state.

```
0397fc87f: REJECTED
HEAD: ACCEPTED
```
</details>

## gnovm/tests/files/decltype_mutual.gno:1-19 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/tests/files/decltype_mutual.gno?plain=1#L1-L19) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L1-L19) [posted](https://github.com/gnolang/gno/pull/5763#discussion_r3842771919)
Missing test: this file pins one of the seven arms of [`fillTypeInPlace`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1560-L1588), and removing the other six leaves the suite at exactly the failures it already has.

<details><summary>mutation</summary>

Deleting the `*ArrayType`, `*SliceType`, `*MapType`, `*InterfaceType`, `*FuncType` and `*PointerType` arms together changes no test result. Deleting the [`*StructType`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1555) arm alone fails this file with `struct type struct{} has no field Val`. A mutual pair per base kind passes here and aborts on master, so each is a behaviour this branch adds that nothing asserts.
</details>

## gnovm/tests/files/decltype_mutual.gno:17-19 [gh](https://github.com/gnolang/gno/blob/093c32be0/gnovm/tests/files/decltype_mutual.gno?plain=1#L17-L19) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L17-L19) [posted](https://github.com/gnolang/gno/pull/5763#discussion_r3842771924)
Missing test: what the cycle writes to state, which is the property that moves now that both halves share one `*StructType`; a `// PKGPATH:` variant with `// Types:` and `// Realm:` pins the two TypeIDs and the persisted diff.
