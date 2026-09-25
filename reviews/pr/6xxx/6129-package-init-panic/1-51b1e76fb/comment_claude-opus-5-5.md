# PR [#6129](https://github.com/gnolang/gno/pull/6129): fix(gnovm): handle panics during package initialization

Verdict: REQUEST CHANGES, because the new frameless path leaves `m.Exception` set, so a `gno repl` session fails its later deferred calls with the old panic, a regression this branch introduces; node paths now fail with the Gno error at unchanged gas.
Event: COMMENT
Model: claude-opus-5-5 at high effort, quick review, solo shape
Commit: 51b1e76fb
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/51b1e76fb3c69b6ff52e27d950fdd7392d20da2b) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/51b1e76fb3c69b6ff52e27d950fdd7392d20da2b)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6129 51b1e76fb`
Round: 1. One finder, then one agent as reflector, judge and writer, 4 candidates: the finder's 2 rerun from scratch by an agent that was not their finder, and 2 added by the reflector, both refuted by run.


## Body

> AI review, claude-opus-5-5, quick review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6129-package-init-panic/overview.md) · Status: REQUEST CHANGES · not manually verified, posted to help reviewers

## gnovm/pkg/gnolang/machine.go:3275 [gh](https://github.com/gnolang/gno/blob/51b1e76fb3c69b6ff52e27d950fdd7392d20da2b/gnovm/pkg/gnolang/machine.go#L3275) · Warning [posted](https://github.com/gnolang/gno/pull/6129#discussion_r4106572617)

`m.Exception = ex` is never cleared, so after `b := a[0]` panics in `gno repl`, which [reuses its Machine](https://github.com/gnolang/gno/blob/51b1e76fb3c69b6ff52e27d950fdd7392d20da2b/gnovm/pkg/repl/repl.go#L134-L138), every later function with a `defer` fails with that index error.

<details><summary>Repro</summary>

From a checkout of this branch, as `gnovm/pkg/repl/stale_test.go`:

```go
package repl

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestSoloFinderStale(t *testing.T) {
	seqs := map[string][]string{
		"direct": {`a := ""`, `b := a[0]`, `z := 0`, `d := 1/z`,
			`func f() (r any) { defer func() { r = recover() }(); return 1 }`, `println(f())`},
		"viafunc": {`func g() { panic("first") }`, `g()`, `z := 0`, `d := 1/z`,
			`func f() (r any) { defer func() { r = recover() }(); return 1 }`, `println(f())`},
		"plain": {`a := ""`, `b := a[0]`, `func h() int { return 2 }`, `println(h())`, `println(3)`,
			`func k() int { defer println("d"); return 4 }`, `println(k())`},
	}
	for _, name := range []string{"direct", "viafunc", "plain"} {
		out, errb := new(bytes.Buffer), new(bytes.Buffer)
		r := NewRepl(WithIO(os.Stdin, out, errb))
		for i, s := range seqs[name] {
			r.RunStatements(s)
			r.rw.Flush()
			fmt.Printf("STEP %s %d %-12q out=%q err=%q\n", name, i, s, out.String(), errb.String())
			out.Reset()
			errb.Reset()
		}
	}
}
```

```bash
go test ./gnovm/pkg/repl -run TestSoloFinderStale -count=1 -v | grep 'STEP plain'
```

The last step prints `d` and then the index error from step 1, instead of `4`:

```
STEP plain 0 "a := \"\""  out="" err=""
STEP plain 1 "b := a[0]"  out="" err="runtime error: index out of range [0] with length 0\n"
STEP plain 2 "func h() int { return 2 }" out="" err=""
STEP plain 3 "println(h())" out="2\n" err=""
STEP plain 4 "println(3)" out="3\n" err=""
STEP plain 5 "func k() int { defer println(\"d\"); return 4 }" out="" err=""
STEP plain 6 "println(k())" out="d\n" err="runtime error: index out of range [0] with length 0\n"
```

With `machine.go` from the merge base, step 6 prints `out="d\n4\n" err=""`: the old nil dereference on `fr.LastException` fired before the assignment, so `m.Exception` stayed nil. In the `direct` sequence at head, `d := 1/z` reports the index error chained ahead of the division by zero, and `x := recover()` at the prompt hits a host nil dereference in [`Recover`](https://github.com/gnolang/gno/blob/51b1e76fb3c69b6ff52e27d950fdd7392d20da2b/gnovm/pkg/gnolang/machine.go#L3309). With `r.m.Exception = nil` added after `r.rec = rec`, step 6 prints `d` and `4`, the division reports alone, and `go test ./gnovm/pkg/repl` passes. The same reset also clears the leftover an unhandled panic inside a function call leaves through `doOpPanic2`, which predates this change. Node paths release their Machine after each call, so none of them sees the leftover.

</details>

## gnovm/adr/pr6129_package_init_panic.md:41 [gh](https://github.com/gnolang/gno/blob/51b1e76fb3c69b6ff52e27d950fdd7392d20da2b/gnovm/adr/pr6129_package_init_panic.md#L41) · Nit [posted](https://github.com/gnolang/gno/pull/6129#discussion_r4106572638)

Nit: this list leaves out the consensus change, a failed transaction recording the runtime message where it recorded `<error: runtime.errorString>`. The new text moves `LastResultsHash` for every block carrying such a transaction, so old and new binaries disagree on that block.

<details><summary>Repro</summary>

From a checkout of this branch, with [`solo-finder-keeper-init-panic-results.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6129-package-init-panic/1-51b1e76fb/tests/solo-finder-keeper-init-panic-results.go) copied to `gno.land/pkg/sdk/vm/solo_finder_initpanic_test.go`:

```bash
go test ./gno.land/pkg/sdk/vm -run TestSoloFinderInitPanic -count=1 -v | grep addpkg_index_direct
git checkout 1fc4c140e -- gnovm/pkg/gnolang/machine.go
go test ./gno.land/pkg/sdk/vm -run TestSoloFinderInitPanic -count=1 -v | grep addpkg_index_direct
```

The same failed `AddPackage` hashes to a different result at each version, with the same gas:

```
head:        gas=98864 resultsHash=24ed25b35878 abciErr="runtime error: index out of range [0] with length 0"
merge base:  gas=98864 resultsHash=46342025520d abciErr="<error: runtime.errorString>"
```

</details>

## SKIP gnovm/cmd/gno/run.go:299 [gh](https://github.com/gnolang/gno/blob/51b1e76fb3c69b6ff52e27d950fdd7392d20da2b/gnovm/cmd/gno/run.go#L299) · Suggestion

Related suggestion: `m.RunFiles(files...)` has no `recover`, so `gno run` on the #6051 repro still exits 2 with a goroutine dump, now carrying the right message. A `func init()` panic exits the same way at the merge base, so this predates the change and stays outside this review.

<details><summary>Repro</summary>

```bash
go build -o /tmp/gno-head ./gnovm/cmd/gno
d=$(mktemp -d)
printf 'package main\n\nvar (\n\ta = ""\n\tA = a[0]\n)\n\nfunc main() {}\n' > $d/main.gno
/tmp/gno-head run $d/main.gno 2>&1 | head -4; echo "exit=${PIPESTATUS[0]}"
```

```
panic: runtime error: index out of range [0] with length 0

goroutine 1 [running]:
github.com/gnolang/gno/gnovm/pkg/gnolang.(*Machine).pushPanic(0xc000495508, {{0x1651500, 0x1fce3a0}, {0x164d1c0, 0xc00041db30}, {0x0, 0x0, 0x0, 0x0, 0x0, ...}})
exit=2
```

At the merge base the process also exits 2, with the host nil dereference as the first line.

</details>
