# PR #5737: fix(gnovm): match Go's call-time dispatch for interface-bound method values

URL: https://github.com/gnolang/gno/pull/5737
Author: ltzmaxwell | Base: master | Files: 31 | +1062 -80
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `ccb6c94ad` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5737 ccb6c94ad`

Round 2 (head advanced 4c57c37e4 → ccb6c94ad, PR content changed: full re-review of the delta). The PR was redesigned: round 1's single `nilReceiverPanic` flag at `VPDerefValMethod` is gone, replaced by lazy interface binding resolved at call time. Both round-1 CHANGES_REQUESTED items are resolved.

**TL;DR:** A method value formed through an interface, like `g := i.M` or `defer i.M()`, used to be wired up the instant you wrote it. Go instead waits until the call to pick the concrete method and copy the receiver. This makes GnoVM wait too, so nil-panic timing, receiver snapshots, embedded promotion, and dynamic re-dispatch all match Go, and two interface-method-value VM crashes are fixed.

**Verdict: APPROVE** — interface-bound method values now match Go on every axis I tested (timing, nil panic, embedded promotion, field re-read, dynamic re-dispatch, across persistence); both round-1 blockers and both of @omarsy's findings are resolved. One open item, author-acknowledged in code: the two consensus gas constants are ratio-scaled placeholders that must be measured on the reference hardware before the hard fork activates.

## Summary

Correct fix. Go materializes an interface-formed method value's concrete method and receiver inside the call, not at the bind; GnoVM resolved it eagerly, diverging on nil-panic timing, receiver snapshot, embedded promotion, field re-read through a boxed pointer, and dynamic re-dispatch, and crashing on `defer i.M()` over a non-nil interface and on a nil embedded pointer-receiver. The redesign binds an interface method value lazily (`BoundMethodValue.Func == nil`, operand saved in `Receiver`, selector in the new persisted `Method` field) and resolves it at the call via `resolveLazyBound`, which walks the operand's current value. A hard fork: `Method` is proto field 4 and `BoundMethodValue` grows 200→216 bytes, changing wire/merkle/gas, so the `stdlib_restart_compare` gas pin moves 2235646→2235788. Round 1's two blockers are gone: the concrete `pt.M` path keeps its eager deref ([values.go:1919-1925](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/values.go#L1919-L1925) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1919-L1925)), and the nil-receiver timing now survives persistence because it lives in the saved nil `*T` operand, not a transient flag.

```
receiver form          Go            master         round 1 (4c57c37e4)   this PR (ccb6c94ad)
---------------------  ------------  -------------  --------------------  -------------------
defer pt.M() concrete  eager(=0)     eager(=0) ✓    call-time(=1) ✗        eager(=0) ✓
defer i.M()  iface      call-time(=1) eager(=0) ✗   call-time(=1) ✓        call-time(=1) ✓
G=i.M; persist; G()    panic         (binds eager)  no panic ✗            panic ✓
```

## Examples

| written form | what it now does | Go |
|---|---|---|
| `c:=&T{1}; g:=c.Get; c.x=2; g()` | `1` (concrete: receiver snapshot at bind) | `1` |
| `var i I=p; h:=i.Get; p.x=2; h()` | `2` (interface: deref at call) | `2` |
| `var i I=pt /*nil*/; g:=i.Get; g()` | panic at call (recoverable) | panic at call |
| `defer pt.M() /*concrete nil*/` | eager panic at the defer stmt | eager panic |
| `s.IG=other; g()` after `g:=o.Get` | re-dispatches to `other` | re-dispatches |
| `s.IG=s` (cyclic embed), `g()` | fatal, uncatchable panic | `fatal error: stack overflow` |

## Glossary

- method value — the expression `x.M` without calling; binds the receiver now, callable later (storable, deferrable).
- `BoundMethodValue` — GnoVM representation of a method value; a first-class, persistable object.
- lazy bind — a `BoundMethodValue` with `Func == nil`: an interface-formed method value whose concrete method and receiver are resolved at the call from the saved operand.
- operand — the boxed value captured when the method value was formed; for a lazy bind it lives in `Receiver` and is re-walked at the call.
- `VPInterface` / `VPDerefValMethod` — `ValuePath` kinds: interface method selection (now binds lazily) vs. dereference-then-value-method (concrete `pt.M`, still eager).
- hard fork — a change to a persisted value's wire format; old state still decodes, but block bytes / IAVL hashes / gas differ, so it ships on fresh genesis or a coordinated upgrade.

## Fix

Bind ([values.go:2087-2106](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/values.go#L2087-L2106) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L2087-L2106)): `VPInterface` now saves the operand into a lazy `BoundMethodValue{Func: nil, Receiver: *dtv, Method: name}` instead of walking the trail eagerly. Call ([values.go:730-763](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/values.go#L730-L763) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L730-L763)): `resolveLazyBound` fills the operand from the store, walks `findEmbeddedFieldType` + `resolveInterfaceTrail` to the concrete method + receiver, and loops once per embedded-interface layer, so the deref, snapshot, nil panic, field re-read and re-dispatch all happen live at the call. Both call sites in [op_call.go:35-37](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/op_call.go#L35-L37) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/op_call.go#L35-L37) (immediate) and [op_call.go:579-584](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/op_call.go#L579-L584) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/op_call.go#L579-L584) (deferred) gate on `IsLazy()`. Defers carry the callable on the unified `Defer.Callable Value` ([frame.go:113-117](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/frame.go#L113-L117) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/frame.go#L113-L117)).

## Benchmarks / Numbers

| constant | before | after | note |
|---|---|---|---|
| `OpCPUSelectorInterface` | 751 | 276 | walk moved off the bind |
| `OpCPULazyBoundResolve` | — | 529 | new, charged per embedded-interface hop |
| `_allocBoundMethodValue` | 200 | 216 | +16 for the `Method Name` field |
| `stdlib_restart_compare` gas | 2235646 | 2235788 | hard-fork pin |
| net per interface call | — | ≈ +54 gas | single-hop |

## Round-1 findings: status

- **Critical — concrete `defer pt.M()` regression** ([round 1](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/1-4c57c37e4/claude-opus-4-8_davd-gzl.md)): **resolved.** Interface dispatch moved out of `VPDerefValMethod` into the `VPInterface` lazy bind, so the concrete path keeps its eager deref. `defer pt.M()` on a nil `*T` again prints `0` (eager), matching Go. Verified: `tests/concrete_defer_eager.gno` passes on this head (was the failing case in round 1).
- **Warning — unexported flag dropped on persist** (round 1): **resolved.** The nil-receiver timing now lives in the persisted nil `*T` operand plus `Method` (proto field 4), not a transient bool, so a reloaded `G := i.M` still derefs and panics at the later call. Covered by `method_nil_value_persist.txtar`.
- Round-1 Nits (alloc cost, stale `defer12.gno` description) and the design-ADR Suggestion are all addressed: the +16 bytes is now load-bearing (carries `Method`), the description is rewritten, and `gnovm/adr/pr5737_nil_value_method_panic_timing.md` documents the semantics.

## Critical (must fix)

None.

## Warnings (should fix)

- **[uncalibrated consensus gas ships in the fork]** [`machine.go:1408`](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/machine.go#L1408) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/machine.go#L1408) — `OpCPULazyBoundResolve` (529) and the re-fit `OpCPUSelectorInterface` (276) are ratio-scaled placeholders, not measured on the gas-table reference hardware; the values set consensus gas for every interface method call.
  <details><summary>details</summary>

  Both constants carry an in-code `TODO(calibration)` ([machine.go:1469](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/machine.go#L1469) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/machine.go#L1469)): they were derived from a lazy-vs-concrete bench ratio against `OpCPUPrecallBoundMethod`'s known value because the reference HW was unavailable. The magnitude is consensus-affecting: it is the per-call gas of every `i.M()` and scales per embedded-interface hop. The fix is a hard fork shipping only on fresh genesis or a coordinated upgrade, so the constants are not yet live, but they must be re-measured before activation; landing the code without the measurement leaves a consensus number unverified. Fix: re-measure both on the reference hardware before the fork is activated.
  </details>

## Nits

- [`values.go:765-775`](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/values.go#L765-L775) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L765-L775) — `(*BoundMethodValue).IsCrossing()` returns `false` for a lazy bind, but `doOpPrecall` resolves the bind before reading `IsCrossing()` on the concrete `fn` ([op_call.go:40](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/op_call.go#L40) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/op_call.go#L40)), so the lazy branch is defensive and not on the call path. No behavior change; confirmed behaviorally: the interface-dispatch filetests pass.
- [`bounded_strings.go:739-741`](https://github.com/gnolang/gno/blob/ccb6c94ad/gnovm/pkg/gnolang/bounded_strings.go#L739-L741) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/bounded_strings.go#L739-L741) — a lazy bind renders as `<nil>` in bounded stacktrace output (its `Func` is nil); the richer `values_string.go` `String()` handles it (`<T>.Method(?)(?)`). Cosmetic, only if an unresolved bind surfaces mid-resolution in a panic stacktrace.

## Missing Tests

None blocking. The PR adds 14 filetests (timing, nil, embedded, operand-capture, dynamic, defer, six cyclic) and three persist txtars; I confirmed coverage of every parity axis against a side-by-side Go run, plus a persisted 3-node cycle (`tests/cyclic_3node_persist.txtar`) and a recursive embedded type, neither of which regress.

## Suggestions

None.

## Open questions

- Caching a value-operand lazy bind's resolution is deferred in the ADR as itself hard-fork-class. Not posted: it is a documented, deliberate scope-out with no risk in this PR.
- #5852 (size bound methods by the wrapper) is stacked on this branch and out of scope here. Not posted: tracked separately, do-not-review-until-merged per its own body.
