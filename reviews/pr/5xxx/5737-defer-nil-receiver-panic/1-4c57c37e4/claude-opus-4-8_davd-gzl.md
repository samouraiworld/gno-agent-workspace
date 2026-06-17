# PR #5737: fix(gnovm): defer nil-pointer panic for value method bound to nil receiver

URL: https://github.com/gnolang/gno/pull/5737
Author: ltzmaxwell | Base: master | Files: 6 | +89 -13
Reviewed by: davd-gzl | Model: claude-opus-4.8 | Commit: `4c57c37e4` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5737 4c57c37e4`

**Verdict: REQUEST CHANGES** — the fix is correct for an interface-boxed nil receiver but regresses the *concrete* nil-pointer case that master already matched to Go (now panics at call time instead of eagerly at bind), and the transient unexported `nilReceiverPanic` flag is silently dropped on persistence, so a stored nil-receiver method value runs on a zero receiver after reload instead of panicking.

## Summary

In Go, a value-receiver method bound to a nil pointer panics at different times depending on the receiver: a **concrete** `*T(nil)` derefs the value copy `*pt` when the method value is *formed* (eager — `defer pt.M()` panics at the defer statement), while an **interface** holding a typed-nil `*T` defers the deref to *call time* (`defer i.M()` panics when the deferred call runs). Master panicked eagerly for both, so the interface case diverged from Go. This PR replaces the eager panic in the `VPDerefValMethod`-nil path with a `BoundMethodValue` carrying a `nilReceiverPanic` flag, deferring the panic to `doOpPrecall`/`doOpReturnCallDefers`. That fixes the interface case but applies to both — the path can't distinguish them — so it newly breaks the concrete case (1 of 2 cases fixed, the other regressed). Separately, the flag is unexported and dropped by amino + `exportValue`, so a persisted nil-receiver method value loses its "panic at call" property across a store round-trip.

```
receiver form        Go            master         this PR
-------------------  ------------  -------------  --------------
defer pt.M() concrete eager(=0)    eager(=0) ✓    call-time(=1) ✗   <- regressed
defer i.M()  iface    call-time(=1) eager(=0) ✗   call-time(=1) ✓   <- fixed
G=i.M; persist; G()   panic         (binds eager)  no panic ✗        <- new hole
```

## Glossary

- value receiver — method declared `func (T) M()`, not `func (*T) M()`; the call needs a copy of `T`.
- method value — the expression `x.M` *without* calling; binds the receiver now, callable later (storable, deferrable).
- `BoundMethodValue` — GnoVM representation of a method value (receiver + func); a first-class, persistable object.
- `VPDerefValMethod` — `ValuePath` kind for "dereference a pointer, then select a value-receiver method" — the path reached for both `pt.M` (concrete `*T`) and `i.M` (interface unwrapped to `*T`).
- `doOpDefer` / `doOpPrecall` / `doOpReturnCallDefers` — VM op handlers for registering a defer, calling a value, and running deferred calls.
- filetest — `gnovm/tests/files/*.gno` checked against its `// Output:` block.

## Fix

Before: [`values.go:1836`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/values.go#L1836) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1836) panicked immediately when the `VPDerefValMethod` receiver `tv.V == nil`. After: it binds a `BoundMethodValue{nilReceiverPanic:true}` with a zero receiver ([`values.go:1835-1859`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/values.go#L1835-L1859) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1835-L1859)) and fires the panic later in [`doOpPrecall`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/op_call.go#L30-L33) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/op_call.go#L30-L33) (direct/stored call) or [`doOpReturnCallDefers`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/op_call.go#L568-L571) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/op_call.go#L568-L571) (deferred call), with `doOpDefer` copying the flag into the `Defer` ([`op_call.go:713`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/op_call.go#L713) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/op_call.go#L713)). The `BoundMethodValue` grows 8 bytes (200→208), tracked in [`alloc.go:88`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/alloc.go#L88) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/alloc.go#L88) and self-checked at [`alloc.go:162`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/alloc.go#L162) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/alloc.go#L162) (verified: `unsafe.Sizeof(BoundMethodValue{}) == 208`), driving the +4 gas bump in the restart txtar.

## Critical (must fix)

- **[regresses Go parity that master already had]** [`values.go:1835`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/values.go#L1835) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1835) — concrete `defer pt.M()` / `g := pt.M` on a nil `*T` value method now panic at call time; Go (and master) panic eagerly at bind.
  <details><summary>details</summary>

  For a concrete pointer, Go forms the value-receiver method value by copying `*pt`, so the nil-deref is **eager**: `defer pt.M()` panics at the defer statement (the later `r=1` never runs → `f()==0`), and `g := pt.M` panics at the assignment. Only an interface-boxed receiver defers the deref to call time. By the time `GetPointerToFromTV` reaches `VPDerefValMethod`, both cases look identical (`tv.T` is `*T`, `tv.V == nil`) — the interface has already been unwrapped to its dynamic `*T` — so the handler cannot tell them apart and defers unconditionally. Result: the concrete case, which master matched to Go, now diverges.

  Verified: Go `f()==0`, master filetest prints `0`, this PR prints `1` (see Repro). The PR's only filetest covers the interface case, so this regression is uncaught. Fix: only the interface path may defer to call time; the concrete `VPDerefValMethod`-nil path must keep the eager panic. That distinction isn't available here — it needs to be made where the interface is unwrapped (interface method dispatch) or marked upstream in the preprocessor — so the current single-site approach can't be correct for both.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5737 -R gnolang/gno
  # `// Output: 0` is Go's behaviour (eager panic at the defer stmt; r=1 never runs).
  # master prints 0; this PR prints 1, so the filetest FAILS on the PR.
  cat > gnovm/tests/files/zz_concrete_defer.gno <<'EOF'
  package main

  type T struct{ x int }

  func (T) M() {}

  var pt *T

  func f() (r int) {
  	defer func() { recover() }()
  	defer pt.M()
  	r = 1
  	return
  }

  func main() {
  	println(f())
  }

  // Output:
  // 0
  EOF
  go test -run 'TestFiles/zz_concrete_defer.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/zz_concrete_defer.gno
  ```
  ```
  --- FAIL: TestFiles/zz_concrete_defer.gno (0.00s)
      files_test.go:111: Output diff:
          --- Expected
          +++ Actual
          @@ -1 +1 @@
          -0
          +1
  FAIL
  ```
  </details>

## Warnings (should fix)

- **[unexported flag dropped on persist → silent zero-receiver call]** [`values.go:655`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/values.go#L655) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L655) — a stored value method bound to a nil pointer panics in the binding tx but runs on a zero receiver in any later tx.
  <details><summary>details</summary>

  The field comment claims the value "is consumed immediately by doOpPrecall / doOpDefer and never escapes to persistent storage." That is false for a method value that is bound but not immediately called — `G := i.M` (interface holding nil `*T`) binds successfully and is storable. Bound methods *do* persist: [`realm.go:1184`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/realm.go#L1184) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/realm.go#L1184) only rejects them from a *private* realm, and both the amino codegen ([`pb3_gen.go:1649`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/pb3_gen.go#L1649) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/pb3_gen.go#L1649)) and `exportValue` ([`values_export.go:237-253`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/values_export.go#L237-L253) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values_export.go#L237-L253)) reconstruct `BoundMethodValue` without the unexported flag. So after a store round-trip the flag is `false` and the receiver is a zero `T`; calling it executes `M` on `T{}` and returns normally instead of panicking.

  This is deterministic across nodes (every node reloads identically), so it is **not** a consensus split — but it is a Go-semantics divergence (a nil-deref silently becomes a real call) and a surprising state-dependent behavior: the same `G` panics when called in the tx that bound it and succeeds when called later. Master never had this hole because it panicked eagerly at bind, so the value was never stored. Trigger is exotic, but the fix introduces it. If the deferred-panic model is kept, the "panic at call" property must survive serialization (e.g. a persisted marker, or reconstruct deterministically on load) rather than living in a transient unexported bool.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5737 -R gnolang/gno
  cat > gno.land/pkg/integration/testdata/zz_nil_persist.txtar <<'EOF'
  gnoland start

  gnokey maketx addpkg -pkgdir $WORK/myrealm -pkgpath gno.land/r/test/myrealm -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
  stdout OK!

  # same tx as the bind: in-memory flag intact -> nil deref panic (matches Go)
  ! gnokey maketx call -pkgpath gno.land/r/test/myrealm -func BindAndCall -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
  stderr 'nil pointer dereference'

  # persist G = i.M (amino drops the unexported flag)
  gnokey maketx call -pkgpath gno.land/r/test/myrealm -func Bind -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
  stdout OK!

  # later tx: G reloaded, flag gone -> no panic, method runs on a zero receiver
  gnokey maketx call -pkgpath gno.land/r/test/myrealm -func CallLater -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
  stdout ZERO-RECEIVER
  stdout OK!

  -- myrealm/gnomod.toml --
  module = "gno.land/r/test/myrealm"
  gno = "0.9"

  -- myrealm/myrealm.gno --
  package myrealm

  type I interface{ M() string }

  type T struct{ x int }

  func (T) M() string { return "ZERO-RECEIVER" }

  var pt *T
  var G func() string

  func BindAndCall(cur realm) string { var i I = pt; G = i.M; return G() }
  func Bind(cur realm)               { var i I = pt; G = i.M }
  func CallLater(cur realm) string   { return G() }
  EOF
  go test -run 'TestTestdata/zz_nil_persist$' ./gno.land/pkg/integration/
  rm gno.land/pkg/integration/testdata/zz_nil_persist.txtar
  ```
  ```
  ok  	github.com/gnolang/gno/gno.land/pkg/integration	3.0s
  # PASS == bug present: BindAndCall (same tx) panics with nil deref,
  # but CallLater (later tx) returns ("ZERO-RECEIVER" string) + OK! instead of panicking.
  ```
  </details>

## Nits

- [`values.go:655`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/pkg/gnolang/values.go#L655) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L655) — every `BoundMethodValue` in every program now costs +8 alloc bytes (200→208) to carry a bool that is meaningful only for the nil-receiver edge case and is dropped on persist. Minor, but it is a global gas cost for a rare flag; worth confirming the trade-off is intended.
- PR body and branch name say `defer12.gno`, but the added filetest is [`nil_value_method_bind.gno`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/tests/files/nil_value_method_bind.gno) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/tests/files/nil_value_method_bind.gno). Stale description.

## Missing Tests

- **[regression uncaught]** [`gnovm/tests/files/nil_value_method_bind.gno`](https://github.com/gnolang/gno/blob/4c57c37e4/gnovm/tests/files/nil_value_method_bind.gno) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/tests/files/nil_value_method_bind.gno) — only the interface receiver is tested. Add the concrete `defer pt.M()` and `g := pt.M` cases (Critical above), which currently diverge from Go.
  <details><summary>details</summary>

  Without a concrete-receiver filetest, the Critical regression slips through green CI. See `tests/concrete_defer_regression.gno` and `tests/concrete_methodval_regression.gno` in this review directory — both currently FAIL on the PR with the Go-correct `// Output:`.
  </details>
- **[persistence path unguarded]** [`gno.land/pkg/integration/testdata/`](https://github.com/gnolang/gno/tree/4c57c37e4/gno.land/pkg/integration/testdata) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/integration/testdata) — no test exercises a nil-receiver method value across a store round-trip. See `tests/nilmethod_persist_nondeterminism.txtar`.

## Suggestions

- Given the subtlety (eager-vs-call-time deref differs by receiver kind, plus the persistence interaction), a short ADR under `gnovm/adr/` documenting the intended semantics and why the chosen layer is correct would help reviewers and future contributors. The current single-site `VPDerefValMethod` fix can't satisfy both the concrete and interface cases at once, so the design rationale matters.

## Questions for Author

- Is deferring the concrete-receiver panic intentional, or was only the interface case in scope? It regresses Go parity that master had.
- How should a value method bound to a nil pointer behave after being persisted and reloaded — panic (match Go / same-tx behavior) or run on a zero receiver (current)? The unexported flag makes the latter the de-facto answer.
