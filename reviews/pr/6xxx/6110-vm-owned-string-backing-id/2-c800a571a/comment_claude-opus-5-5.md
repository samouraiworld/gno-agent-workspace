# Review: [#6110](https://github.com/gnolang/gno/pull/6110)
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. The GC recount dedups strings on a backing pointer whose sharing differs between a long-running and a restarted node, a consensus divergence the branch introduces, and a `map[any]` literal with a typed-nil slice key is rejected at preprocess.
Model: claude-opus-5-5, standard review
Commit: c800a571ab179a07530dcf876142271afc64e40a
Overview: [overview.md](../overview.md)
Open the code: [head worktree](../../../../../.worktrees/gno-review-6110)
Round: 2. 13 finders, one reflector, 13 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 4 refuted.

## Body

- Critical: the [GC visitor](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/garbage_collector.go#L335-L340) counts string bytes once per backing pointer, and whether two realms' copies of one imported const [share a backing](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/preprocess.go#L4514) depends on a node restart. The same live values then recount 1000 bytes apart, so a transaction near [`maxAllocTx`](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gno.land/pkg/sdk/vm/keeper.go#L50) can abort on one validator and pass on another.

  <details><summary>repro</summary>

  The recount of the same two realms must not depend on whether the node restarted.

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6110 -R gnolang/gno
  curl -fsSL -o gno.land/pkg/sdk/vm/zz_b2catalog_test.go \
    https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/2-c800a571a/tests/b2-catalog-const-backing-restart_test.go
  go test -count=1 -run TestB2CatalogConstBackingDependsOnRestart -v ./gno.land/pkg/sdk/vm
  rm gno.land/pkg/sdk/vm/zz_b2catalog_test.go
  ```

  The two node histories disagree on the recount:

  ```text
  long-running node: qq.B and rr.C share a backing = false, recount = 2096
  restarted node: qq.B and rr.C share a backing = true, recount = 1096
  expected: 2096 actual: 1096
  --- FAIL
  ```

  </details>

- Suggestion: a `[]byte` call argument is a [`SliceValue` literal](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gno.land/pkg/sdk/vm/convert.go#L218) that no allocator call ever charges, unlike the same bytes [passed as a string](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gno.land/pkg/sdk/vm/convert.go#L56). Building it through [`alloc.NewSliceFromData`](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/alloc.go#L698) charges both alike.

## gnovm/pkg/gnolang/preprocess.go:2510 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/preprocess.go#L2510) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/preprocess.go#L2510) · Warning
`ComputeMapKey` panics with `hash of unhashable type` on a typed-nil slice, map or func key in a map literal. Preprocessing then rejects `map[any]int{[]int(nil): 1}` in an uncalled function, which Go compiles.

<details><summary>repro</summary>

The file must preprocess and print `ok`, as `go run` does.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
curl -fsSL -o gnovm/tests/files/b2lines_maplit_a.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/2-c800a571a/tests/b2-lines-maplit-unhashable-const-key.gno
go test ./gnovm/pkg/gnolang -run 'TestFiles/b2lines_maplit_a' -count=1
rm gnovm/tests/files/b2lines_maplit_a.gno
```

The preprocessor panics before `main` runs:

```text
unexpected panic: main/b2lines_maplit_a.gno:24:7-33: &{("runtime error: hash of unhashable type []int" .uverse..runtimeError) ...}
FAIL
```

</details>

## gnovm/adr/pr6110_string_backing_id.md:85 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/adr/pr6110_string_backing_id.md?plain=1#L85) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/adr/pr6110_string_backing_id.md#L85) · Nit
Nit: the ADR's list of direct-equality sites misses `tv.V != old.V` in [`StaticBlock.Define2`](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/nodes.go#L2404), which compares a string const's backing pointer.

## SKIP gnovm/pkg/gnolang/alloc.go:1030-1031 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/alloc.go#L1030-L1031) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L1030) · Missing test
Missing test: no test checks, per value kind, that `internalStringSize` charges exactly the strings `fillTypesOfValue` mints, as its doc comment requires.

Not posted: no verifier ran the check.

## SKIP gnovm/pkg/gnolang/nodes.go:2404 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/nodes.go#L2404) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/nodes.go#L2404) · Suggestion
Suggestion: `tv.V != old.V` compares string consts by backing pointer rather than by their `Str` content.

Not posted: no path redefining a defined string const was found, so the read could not settle it.

## SKIP gno.land/pkg/integration/testdata/gc_load_evicts_object.txtar:9 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gno.land/pkg/integration/testdata/gc_load_evicts_object.txtar#L9) · [↗](../../../../../.worktrees/gno-review-6110/gno.land/pkg/integration/testdata/gc_load_evicts_object.txtar#L9) · Missing test
Missing test: no run shows the fixture failing once `fillTypesOfValue` charges each string again, so nothing proves it guards the single up-front charge.

Not posted: no verifier ran the revert.

## SKIP gnovm/pkg/gnolang/alloc_test.go:138 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/alloc_test.go#L138) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc_test.go#L138) · Missing test
Missing test: `TestInternalStringSize` leaves the `MapValue`, `BoundMethodValue`, `HeapItemValue` and `PackageValue` cases of `internalStringSize` uncovered.

Not posted: no verifier ran the mutation.

## SKIP gnovm/pkg/gnolang/parity_test.go:38 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gnovm/pkg/gnolang/parity_test.go#L38) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/parity_test.go#L38) · Missing test
Missing test: every amino parity case builds a `StringValue` with no backing, so nothing pins that a minted backing never reaches the wire.

Not posted: no verifier ran the check.

## SKIP gno.land/pkg/sdk/vm/keeper.go:1280 [gh](https://github.com/gnolang/gno/blob/c800a571ab179a07530dcf876142271afc64e40a/gno.land/pkg/sdk/vm/keeper.go#L1280) · [↗](../../../../../.worktrees/gno-review-6110/gno.land/pkg/sdk/vm/keeper.go#L1280) · Nit
Nit: a GC fired by the string argument charge before `SetActivePackage` dereferences a nil `m.Package`. No message reaches that window, since the allocator resets at each message start.

Not posted: PLAUSIBLE Nit, and the reachable case needs a package block loading within one argument of 500 MB.
