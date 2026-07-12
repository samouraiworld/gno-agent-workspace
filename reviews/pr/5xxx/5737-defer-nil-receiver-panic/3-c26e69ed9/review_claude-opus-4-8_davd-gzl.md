# PR #5737: fix(gnovm): match Go's call-time dispatch for interface-bound method values

URL: https://github.com/gnolang/gno/pull/5737
Author: ltzmaxwell | Base: master | Files: 43 | +1536 -85
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `c26e69ed9` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5737 c26e69ed9`

Round 3 (head advanced ccb6c94ad → c26e69ed9 over a master merge; full re-review of the delta). The core lazy-bind design is unchanged from round 2; the delta is hardening plus one new field. @omarsy's cyclic-loop thread is fully closed: the struct-value-carried cycle (`dd77e415f`) now terminates, verified on the persisted and unbounded render paths @omarsy flagged, not just in-memory. A new `MethodPkg` field package-qualifies the call-time lookup (`BoundMethodValue` 216→232 bytes). Round-2 findings: the `<nil>` bounded-render nit is fixed (`f5171d539`); the gas-calibration Warning is reworded to a ratio-anchored rationale and drops to a Nit. One new regression surfaced: a method value bound over a function-local type no longer survives persistence.

**TL;DR:** A method value formed through an interface, like `g := i.M` or `defer i.M()`, used to be wired up the instant you wrote it. Go instead waits until the call to pick the concrete method and copy the receiver. This makes GnoVM wait too, so nil-panic timing, receiver snapshots, embedded promotion, and dynamic re-dispatch all match Go, and two interface-method-value VM crashes are fixed.

**Verdict: APPROVE** — interface-bound method values match Go on every axis tested, and @omarsy's cyclic hang is closed on the tx, persisted, and unbounded render paths. One non-blocking regression: a stored method value over a function-local type panics on reload where master returned a value; it rides a pre-existing local-type persistence limitation that master's eager bind happened to dodge. The consensus gas constants remain ratio-scaled placeholders, now documented as such.

## Summary

Correct fix, carried forward from round 2. Go materializes an interface-formed method value's concrete method and receiver inside the call, not at the bind; GnoVM resolved it eagerly, diverging on nil-panic timing, receiver snapshot, embedded promotion, field re-read, and dynamic re-dispatch, and crashing on `defer i.M()` over a non-nil interface and on a nil embedded pointer-receiver. The redesign binds lazily (`BoundMethodValue.Func == nil`, operand in `Receiver`, selector in `Method`, bind-site package in the new `MethodPkg`) and resolves at the call via `resolveLazyBound`. Round 3 hardens the edges: struct-carried cycles are now detected ([values.go:747-763](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/values.go#L747-L763) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L747-L763)), `exportCopyValue` no longer nil-derefs a lazy bind's `Func` and carries `Method` ([values_export.go:244-256](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/values_export.go#L244-L256) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values_export.go#L244-L256)), a bounded stacktrace renders `<bound-method ?.M>` instead of `<nil>` ([bounded_strings.go:714-722](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/bounded_strings.go#L714-L722) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/bounded_strings.go#L714-L722)), and `MethodPkg` qualifies the lookup so an unexported method isn't matched against a same-spelled member in the dynamic type's own package. The `MethodPkg` field is proto field 5 and grows `BoundMethodValue` 216→232 bytes, extending the round-2 hard fork.

## Glossary

- lazy bind — a `BoundMethodValue` with `Func == nil`: an interface-formed method value whose concrete method and receiver are resolved at the call from the saved operand.
- operand — the boxed value captured when the method value was formed; for a lazy bind it lives in `Receiver` and is re-walked at the call.
- `MethodPkg` — new field: the package that formed the method value; qualifies the call-time lookup so an unexported method keeps its bind-site identity.
- seen-set — the visited-identity map in `resolveLazyBound` that breaks a cyclic embedded-interface walk; keys on pointer and struct identity.
- hard fork — a change to a persisted value's wire format; old state still decodes, but block bytes / IAVL hashes / gas differ, so it ships on fresh genesis or a coordinated upgrade.

## Round-2 findings: status

- **Warning — uncalibrated consensus gas** (round 2): **downgraded to Nit.** `b792eb335` reworks the TODO: `OpCPULazyBoundResolve` = 529 and the re-fit `OpCPUSelectorInterface` = 276 are now documented as ratio-scaled against `OpCPUPrecallBoundMethod`'s known reference value so the machine-speed factor cancels, with direct re-measurement deferred to the next reference-HW refresh rather than gated on the fork. That is the same footing as other ratio-scaled entries already in the table; no longer a should-block item.
- **Nit — lazy bind renders `<nil>` in bounded stacktrace** (round 2): **resolved.** `f5171d539` renders `<bound-method ?.<Method>>` for an unresolved bind.
- **Nit — `IsCrossing()` lazy branch is defensive** (round 2): **carried.** Still defensive (the bind is resolved before `IsCrossing()` is read); no behavior change.

## Critical (must fix)

None.

## Warnings (should fix)

- **[stored method value over a local type bricks on reload]** [`values.go:739`](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/values.go#L739) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L739) — a method value bound over a function-local type, persisted then reloaded, panics `unexpected type with id ...S` where master returned the value.
  <details><summary>details</summary>

  The lazy bind saves the original operand and its type into `Receiver`; at the call `resolveLazyBound` re-derives the dispatch trail on that type, which calls `Store.GetType` ([store.go:780](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/store.go#L780) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/store.go#L780)). A function-local type's TypeID is never written to the type store, so the reload panics. Master eager-bound the concrete method and stored the promoted package-level receiver `T{7}`, so its reload returned `(7 int)`.

  This rides a pre-existing limitation: a raw interface value over a local type (`var G I = S{}; ...; G.Get()` across reload) already panics on both master and this PR, so local types are non-persistable in general. Master's eager method-value bind dodged it by resolving to a package-level receiver; the lazy bind no longer dodges it. The trigger is narrow, but it is a real behavior regression, and the failure is a raw internal panic rather than a clean error. Fix: either resolve eagerly when the operand type is a local (non-persistable) type, restoring master's result, or reject the bind with a clear message instead of an internal `unexpected type with id`. Regression pinned in [`tests/method_iface_local_type_persist.txtar`](tests/method_iface_local_type_persist.txtar) (master `(7 int)`, this PR panics).
  </details>

## Nits

- [`machine.go:1408`](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/machine.go#L1408) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/machine.go#L1408) — `OpCPULazyBoundResolve` (529) and `OpCPUSelectorInterface` (276) are ratio-scaled, not measured on the gas-table reference hardware; the in-code TODO now documents the methodology and defers direct measurement to the next HW refresh. Consensus-affecting per interface call, but on the same footing as other ratio-scaled table entries; flagging for whoever refreshes the gas table.
- [`values.go:801-811`](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/values.go#L801-L811) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L801-L811) — `(*BoundMethodValue).IsCrossing()` returns `false` for a lazy bind; `doOpPrecall` resolves the bind before reading `IsCrossing()` on the concrete `fn`, so the lazy branch is defensive and off the call path. No behavior change.

## Missing Tests

- **[cyclic-value fix has only in-memory coverage]** [`gnovm/tests/files/method_iface_cyclic_value.gno`](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/tests/files/method_iface_cyclic_value.gno) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/tests/files/method_iface_cyclic_value.gno) — the shipped filetest exercises the struct-carried cycle in a single in-memory run; @omarsy's concern was the persisted and unbounded query and `Render()` paths, which no test pins.
  <details><summary>details</summary>

  The fix keys the seen-set on `*StructValue` identity and relies on that identity staying cache-stable when the operand is reloaded from the store. I verified this holds on both the persisted tx path and the unbounded qrender path (each terminates in ~3.5s with the fatal cyclic panic, no hang). A regression test on the reload path would guard the identity-stability assumption against a future store-cache change; the in-memory filetest would not catch such a break. Ready to add: [`tests/method_iface_cyclic_value_persist.txtar`](tests/method_iface_cyclic_value_persist.txtar), which stores the cycle and hits it through qrender.
  </details>

## Verified

- @omarsy's struct-value cycle is closed on every path he flagged, not just in-memory. Storing `s.IG = W{s}` and reaching it through a later tx and through the unbounded `vm/qrender` query both terminate with `cyclic embedded interface in method-value dispatch` in ~3.5s rather than hanging; the reload keeps `*StructValue` identity cache-stable, so the seen-set fires. Pinned in [`tests/method_iface_cyclic_value_persist.txtar`](tests/method_iface_cyclic_value_persist.txtar).
- The exhaustiveness argument holds: another lazy hop requires a method promoted through an embedded interface, and only structs embed, so every recurring operand is a struct (`*StructValue`) or pointer-to-struct (`PointerValue`), both keyed; non-struct receivers resolve a directly-declared method and terminate. All 18 `method_iface_*` filetests pass, including the negative guards `method_iface_shadow_cyclic` and `method_iface_deep_converge` that must not fire on legitimate deep dispatch.
- The local-type persistence regression is a genuine behavior change, not a test artifact: the identical program returns `(7 int)` on the merge-base `a19f13f90` and panics on `c26e69ed9`. A package-level type of the same shape persists and returns `(7 int)` on this PR, scoping the break to function-local types.
- The hard-fork wire change is consistent: `MethodPkg` is proto field 5 ([gnolang.proto:87](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/gnolang.proto#L87) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/gnolang.proto#L87)) with marshal/size/unmarshal wired in `pb3_gen.go`; the `init()` size self-check ([alloc.go:158](https://github.com/gnolang/gno/blob/c26e69ed9/gnovm/pkg/gnolang/alloc.go#L158) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/alloc.go#L158)) confirms `_allocBoundMethodValue` = 232 equals `unsafe.Sizeof(BoundMethodValue{})`; and the consensus gas pins `stdlib_restart_compare`, `restart_gas`, and `simulate_gas` pass.
- `exportCopyValue` on a lazy bind no longer nil-derefs and preserves `Method`, pinned by the new [`qeval_json_lazy_method.txtar`](https://github.com/gnolang/gno/blob/c26e69ed9/gno.land/pkg/integration/testdata/qeval_json_lazy_method.txtar) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/integration/testdata/qeval_json_lazy_method.txtar).
- Green at c26e69ed9: the 18 `method_iface_*` filetests, the two new regression/verification txtars, and the three consensus gas txtars. The two `redeclaration3/4.gno` TypeCheckError diffs are pre-existing local-Go-version drift (go1.26.4 emits `foo (local variable) is not a type`); they fail identically on the merge-base `a19f13f90` and are unrelated to this PR.

## Open questions

- Whether the local-type persistence break should be fixed or documented is the author's call: the pattern is exotic, the underlying local-type limitation is pre-existing, and the PR's behavior is arguably more consistent (a local type doesn't silently persist just because a method value was taken on it). Posted as a Warning so the choice is explicit, not because it must block.
