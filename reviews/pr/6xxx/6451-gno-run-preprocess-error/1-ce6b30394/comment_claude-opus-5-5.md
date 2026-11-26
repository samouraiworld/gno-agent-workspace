# Review: [#6451](https://github.com/gnolang/gno/pull/6451)
Posted: https://github.com/gnolang/gno/pull/6451#pullrequestreview-6080529214
Event: COMMENT
Verdict: NEEDS DISCUSSION, the change improves strictly on its base, and the two open Warnings, initialization cycles aborting with a Go panic and duplicate declarations accepted under the default package path, predate it, so they shape the scope rather than block.
Model: claude-opus-5-5, standard review
Commit: ce6b30394 (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6451 ce6b30394`
Round: 1. Two finders and one judge, 13 candidates: 7 settled by the finders' own read, 6 run or read again by the judge, 6 kept.

## Body

This can go in once two escapes from [`runFiles`](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/run.go#L379) are settled: an initialization cycle aborts with a Go panic, and a plain directory run accepts duplicate declarations. The cycle comment and the duplicate bullet are the open points; the test and location comments are optional.

- Related: `gno run <dir>` with no `-pkgpath` runs the later of two duplicate declarations and exits 0. The default package path `main` makes [`IsStdlib(pv.PkgPath)`](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/pkg/gnolang/machine.go#L646) true, so the duplicate check never runs.

  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6451 -R gnolang/gno
  (cd gnovm && go build -o /tmp/gno-6451 ./cmd/gno)
  d=$(mktemp -d)
  printf 'package main\n\nfunc helper() int { return 1 }\n\nfunc main() { println(helper()) }\n' > "$d/a.gno"
  printf 'package main\n\nfunc helper() int { return 2 }\n' > "$d/b.gno"
  /tmp/gno-6451 run "$d/"; echo "rc=$?"
  /tmp/gno-6451 run -pkgpath example.com/main "$d/"; echo "rc=$?"
  rm -rf "$d"
  ```

  The first run prints the second helper's result and exits 0; the merge base prints the same.

  ```
  2
  rc=0
  running package "example.com/main": redeclarations for identifiers: "helper" (code=gnoPreprocessError)
  rc=1
  ```

  </details>

## gnovm/cmd/gno/run.go:404 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/run.go#L404) · [↗](../../../../../.worktrees/gno-review-6451/gnovm/cmd/gno/run.go#L404) · Warning [posted](https://github.com/gnolang/gno/pull/6451#discussion_r4713408124)
`panic(r)` re-throws the plain-string cycle panic from `resolveEffectiveDeps`, so a cycle through a method, a closure or two functions aborts with a Go panic and exit code 2.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6451 -R gnolang/gno
(cd gnovm && go build -o /tmp/gno-6451 ./cmd/gno)
d=$(mktemp -d) && cd "$d"
mk() { mkdir -p "$1"; cat > "$1/main.gno"; }
mk cycle_method <<'EOF'
package main

type T struct{}

func (T) M() int { return a }

var a = T{}.M()

func main() { println(a) }
EOF
mk cycle_closure <<'EOF'
package main

var a = func() int { return a }()

func main() { println(a) }
EOF
mk cycle_twohop <<'EOF'
package main

var a = f()

func f() int { return g() }

func g() int { return a }

func main() { println(a) }
EOF
for c in cycle_method cycle_closure cycle_twohop; do
	/tmp/gno-6451 run "./$c/" >/dev/null 2>err; echo "$c: exit $?"; head -1 err
done
cd - >/dev/null && rm -rf "$d"
```

Each case exits 2 with a Go panic where a one-line `(code=gnoPreprocessError)` error and exit 1 are expected; the merge base exits 2 the same way.

```
cycle_method: exit 2
panic: main/cycle_method/main.gno:7:5: circular dependency: a -> T.M -> a [recovered, repanicked]
cycle_closure: exit 2
panic: main/cycle_closure/main.gno:3:5: circular dependency: a -> a [recovered, repanicked]
cycle_twohop: exit 2
panic: main/cycle_twohop/main.gno:3:5: circular dependency: a -> f -> g -> a [recovered, repanicked]
```

`const a = b; const b = a`, `var x = x` and `type A = B; type B = A` print one line and exit 1, since they raise inside `Preprocess` and reach the `*gno.PreprocessError` branch. The function shape, `var a = f(); func f() int { return a }`, behaves like the three above.

</details>

## gnovm/cmd/gno/common.go:236 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/common.go#L236) · [↗](../../../../../.worktrees/gno-review-6451/gnovm/cmd/gno/common.go#L236) · Missing test [posted](https://github.com/gnolang/gno/pull/6451#discussion_r4713408130)
Missing test: lint and test errors take their location from this code too, and no txtar asserts the shorter `l2.gno:3:1-21:` form.

<details><summary>test cases</summary>

```
# a preprocess error in a /p/ package is located by its file alone
! gno lint .
stderr '^l2.gno:3:1-21: crossing function \(realm first argument\) declared in non-realm package'
! stderr 'gno.land/p/f1/l2:'

-- gnomod.toml --
module = "gno.land/p/f1/l2"
gno = "0.9"

-- l2.gno --
package l2

func F(cur realm) {}
```

The merge base prints `gno.land/p/f1/l2:gno.land/p/f1/l2/l2.gno:3:1-21: crossing function ...` for the same package.

</details>

## gnovm/cmd/gno/run.go:394 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/run.go#L394) · [↗](../../../../../.worktrees/gno-review-6451/gnovm/cmd/gno/run.go#L394) · Nit [posted](https://github.com/gnolang/gno/pull/6451#discussion_r4713408134)
Nit: this call writes the location `.:0` for a message that has none, so a duplicate inside an imported package prints `.:0: running package ...` where the doc comment promises no location.

## SKIP gnovm/cmd/gno/run.go:387 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/run.go#L387) · [↗](../../../../../.worktrees/gno-review-6451/gnovm/cmd/gno/run.go#L387) · Suggestion
Suggestion: an internal VM assertion reaches this branch too, so `func (int) M() {}` prints `should not happen` as a source error with no stack.

**Fix:** keep the Go stack for a recovered value that is not a source error, for instance behind the existing `DEBUG_PANIC=1`.

Not posted: `gno lint` and `gno test` print the same line through `printError`, and `DEBUG_PANIC=1` already restores the stack, so this aligns `gno run` with them.
