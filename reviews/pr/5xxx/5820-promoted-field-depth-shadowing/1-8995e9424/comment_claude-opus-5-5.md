# Review: [#5820](https://github.com/gnolang/gno/pull/5820)
Posted: https://github.com/gnolang/gno/pull/5820#pullrequestreview-5547554754
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. The branch introduces two Warnings in `FindEmbeddedFieldType`: a selector that runs a different method than Go picks with no error, and a Go-valid program the merge base ran that now fails preprocess after go/types accepts it; both come from walks whose result is discarded leaving types marked in the shared `m`.
Model: claude-opus-5-5, standard review, solo shape: 2 finders at xhigh, judge at high
Commit: `8995e9424`
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-5820 8995e9424`
Round: 1. Two finders and one reflector pass, 9 candidates, each Warning and the Missing test run from scratch at the head and at the merge base by a judge that was not its finder, with a Go twin; one Nit and one Warning left PLAUSIBLE with their checks named in `claims.md`.

## Body

## gnovm/pkg/gnolang/types.go:902 [gh](https://github.com/gnolang/gno/blob/8995e9424/gnovm/pkg/gnolang/types.go#L902) · [↗](../../../../../.worktrees/gno-review-5820/gnovm/pkg/gnolang/types.go#L902) · Warning [posted](https://github.com/gnolang/gno/pull/5820#discussion_r4294484003)

Inside `S{C; X}` this case keeps `S.C` though the `X.R` walk marked `R`, so `T{S; R}` runs `S.C.M` where Go picks the direct `R`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5820 -R gnolang/gno
cat > gnovm/tests/files/shadow_wrong_method.gno <<'EOF'
package main

type I interface{ M() string }

type C struct{ V string }

func (C) M() string { return "C.M" }

type R struct{ V string }

func (R) M() string { return "R.M" }

type X struct{ R }

type S struct {
	C
	X
}

type T struct {
	S
	R
}

func main() {
	t := T{S{C{"c"}, X{R{"inner"}}}, R{"outer"}}
	var i I = t
	println(i.M(), t.V)
}

// Output:
// R.M outer
EOF
(cd gnovm/pkg/gnolang && go test -count=1 -run 'TestFiles/shadow_wrong_method.gno$' .)
rm gnovm/tests/files/shadow_wrong_method.gno
```

The test fails: the expected line is what Go prints for the same program, and the VM runs `C.M` and reads `C.V` instead, with no `TypeCheckError` diff, so go/types accepts it.

```
    --- FAIL: TestFiles/shadow_wrong_method.gno (0.00s)
        files_test.go:111: Output diff:
            --- Expected
            +++ Actual
            @@ -1 +1 @@
            -R.M outer
            +C.M c
```

A second shape, `T{Sub; C}` with `Sub{A; W}` and `W{C}`, prints `A` for the interface call, the method and the field where Go prints `C`.

</details>

## gnovm/pkg/gnolang/types.go:907 [gh](https://github.com/gnolang/gno/blob/8995e9424/gnovm/pkg/gnolang/types.go#L907) · [↗](../../../../../.worktrees/gno-review-5820/gnovm/pkg/gnolang/types.go#L907) · Warning [posted](https://github.com/gnolang/gno/pull/5820#discussion_r4294484008)

This tie only sets `ambiguous`, so `S{A; B; Z}` walks on into `Z{R}` and marks `R`, and `T{S; R}`, valid Go resolving to `R.M`, fails preprocess.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5820 -R gnolang/gno
cat > gnovm/tests/files/shadow_tie_walk.gno <<'EOF'
package main

type I interface{ M() string }

type A struct{}

func (A) M() string { return "A.M" }

type B struct{}

func (B) M() string { return "B.M" }

type R struct{}

func (R) M() string { return "R.M" }

type Z struct{ R }

type S struct {
	A
	B
	Z
}

type T struct {
	S
	R
}

func main() {
	var i I = T{}
	println(i.M())
}

// Output:
// R.M
EOF
(cd gnovm/pkg/gnolang && go test -count=1 -run 'TestFiles/shadow_tie_walk.gno$' .)
rm gnovm/tests/files/shadow_tie_walk.gno
```

The test fails at preprocess on a program Go runs, printing `R.M`.

```
    --- FAIL: TestFiles/shadow_tie_walk.gno (0.00s)
        files_test.go:111: unexpected panic: main/shadow_tie_walk.gno:31:6-15: main.T does not implement main.I (missing method M)
# …
```

A second shape, `T{Amb; C}` with `Amb{A; B; W}` and `W{C}`, fails with `missing field M in main.T` where Go resolves `C.M`.

</details>

## gnovm/tests/files/embed_method_shadow_internal_ambiguous.gno:24 [gh](https://github.com/gnolang/gno/blob/8995e9424/gnovm/tests/files/embed_method_shadow_internal_ambiguous.gno#L24) · [↗](../../../../../.worktrees/gno-review-5820/gnovm/tests/files/embed_method_shadow_internal_ambiguous.gno#L24) · Missing test [posted](https://github.com/gnolang/gno/pull/5820#discussion_r4294484015)

Missing test: the VM runs `C.M` for `Clean{C}` where Go reports an ambiguous `T.M`, as this fixture rejects only through the `A` already in `m`.

<details><summary>test cases</summary>

This case fails: the VM prints `C`.

```go
package main

type I interface{ M() string }

type A struct{}

func (A) M() string { return "A" }

type B struct{}

func (B) M() string { return "B" }

type C struct{}

func (C) M() string { return "C" }

type Amb struct {
	A
	B
}

type Clean struct{ C }

type T struct {
	Amb
	Clean
}

func main() {
	var i I = T{}
	println(i.M())
}

// Error:
// main/embed_method_shadow_internal_ambiguous2.gno:30:6-15: main.T does not implement main.I (missing method M)

// TypeCheckError:
// main/embed_method_shadow_internal_ambiguous2.gno:30:12: cannot use T{} (value of struct type T) as I value in variable declaration: T does not implement I (ambiguous selector T.M)
```

</details>

## gnovm/pkg/gnolang/types.go:896 [gh](https://github.com/gnolang/gno/blob/8995e9424/gnovm/pkg/gnolang/types.go#L896) · [↗](../../../../../.worktrees/gno-review-5820/gnovm/pkg/gnolang/types.go#L896) · Nit [posted](https://github.com/gnolang/gno/pull/5820#discussion_r4294484027)

Nit: a revisit pruned by `m` returns `nil` like a miss, so `T{A{Base}; B{Base}; C{D{E}}}` runs `Base.M` where Go reports an ambiguous selector.

<details><summary>repro</summary>

The filetest of that shape prints `base` at the head, and the harness reports `TypeCheckError ambiguous selector T{}.M`. A Go-ambiguous `T{A{C; X{D}}; B{D}}` resolves to `C.M` by the same pruning.

</details>

## SKIP gnovm/pkg/gnolang/types.go:900 [gh](https://github.com/gnolang/gno/blob/8995e9424/gnovm/pkg/gnolang/types.go#L900) · [↗](../../../../../.worktrees/gno-review-5820/gnovm/pkg/gnolang/types.go#L900) · Nit

Nit: `cand` is built for every embedded field that matches, the deeper and tied ones the switch then drops included.

Not posted: the extra allocation per lookup is read off the code and its count was not taken, and the function needs reworking for the two Warnings above anyway.
