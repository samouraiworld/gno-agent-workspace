# PR [#5737](https://github.com/gnolang/gno/pull/5737): fix(gnovm): match Go's call-time dispatch for interface-bound method values

URL: https://github.com/gnolang/gno/pull/5737
Author: ltzmaxwell | Base: master | Files: 45 | +1562 -85
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `60dca7f36` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5737 60dca7f36`
Overview: [visual overview](https://samouraiworld.github.io/gno-agent-workspace/reviews/pr/5xxx/5737-defer-nil-receiver-panic/overview.html) · [↗](../overview.html)

Round 4. Head advanced `c26e69ed9` → `60dca7f36`, +3 commits, no rebase. The delta is exactly @omarsy's three 2026-07-14 findings, one commit each: the gnoweb state walker [nil-derefs a lazy bind](https://github.com/gnolang/gno/pull/5737#discussion_r3578551168), [`pb3_gen.go` was hand-edited](https://github.com/gnolang/gno/pull/5737#discussion_r3578551297) rather than generated, and the new `VPSubrefField` nil guard [panicked with a plain string](https://github.com/gnolang/gno/pull/5737#discussion_r3578551393). All three are fixed and verified here; no VM logic changed. Round-3 findings carry unchanged: the local-type persistence regression still reproduces, and the struct-carried-cycle persist test is still missing.

**TL;DR:** A method value formed through an interface, like `g := i.M` or `defer i.M()`, used to be wired up the instant you wrote it. Go instead waits until the call to pick the concrete method and copy the receiver. This makes GnoVM wait too, so nil-panic timing, receiver snapshots, embedded promotion, and dynamic re-dispatch all match Go, and two interface-method-value VM crashes are fixed.

**Verdict: APPROVE** — the three findings in this round's delta each close the gap they were raised for, confirmed by running the code rather than reading it, and every row of round 1's posted `CHANGES_REQUESTED` table now matches Go. Posting this round also clears that stale blocker, which is the last thing holding the PR after @omarsy's 2026-07-16 approval. The carried local-type persistence regression stays non-blocking: it rides a pre-existing limitation that master's eager bind happened to dodge, and it is [@omarsy](https://github.com/gnolang/gno/pull/5737#discussion_r3513857885)'s open question to the author, not a new one.

## Summary

Correct fix, carried forward from round 2. Go materializes an interface-formed method value's concrete method and receiver inside the call, not at the bind; GnoVM resolved it eagerly, diverging on nil-panic timing, receiver snapshot, embedded promotion, field re-read, and dynamic re-dispatch, and crashing on `defer i.M()` over a non-nil interface and on a nil embedded pointer-receiver. The redesign binds lazily and resolves at the call via `resolveLazyBound`.

This round adds no VM logic. It closes three consumer-side gaps that the eager→lazy shape change opened: the gnoweb state walker now renders an unresolved bind instead of passing its nil `Func` into `decodeFuncInline` ([walker.go:495-504](https://github.com/gnolang/gno/blob/60dca7f36/gno.land/pkg/gnoweb/feature/state/walker.go#L495-L504) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/gnoweb/feature/state/walker.go#L495-L504)); `pb3_gen.go` is regenerated so the committed bytes match the generator ([pb3_gen.go:1654](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/pb3_gen.go#L1654) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/pb3_gen.go#L1654)); and the `VPSubrefField` nil-pointer guard raises `typedRuntimeError` instead of `typedString` ([values.go:1923](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/values.go#L1923) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1923)), so the one nil-deref this PR newly makes recoverable now recovers as an `error`, like Go's `runtime.Error`.

## Glossary

- lazy bind — an interface-formed method value held with `Func == nil`, resolved at call time from the saved operand.
- function-local type — a type declared inside a function body; its TypeID is never written to the type store, so a persisted value referencing one dangles on reload.
- gnoweb — the web frontend serving chain content; its state explorer walks persisted realm objects into a tree.

## Fix

`typedRuntimeError` builds a value of the VM's [`.runtimeError` type](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/uverse.go#L35-L41) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/uverse.go#L35-L41), which carries an [`Error()` method](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/uverse.go#L569) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/uverse.go#L569), where `typedString` builds a bare `StringType` ([values.go:3015-3019](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/values.go#L3015-L3019) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L3015-L3019)). The `VPSubrefField` guard was the only nil-deref site in `getPointerToFromTV` still on the string form; its two siblings at [values.go:1957](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/values.go#L1957) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1957) and [values.go:1986](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/values.go#L1986) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1986) already used the error form. In the walker, the load-bearing constraint is that no signature exists to render: the bind holds a selector name and an operand, and the concrete `*FuncValue` only materializes inside the call.

## Round-1 posted verdict: status

The only review davd-gzl has posted on this PR is round 1's `CHANGES_REQUESTED` (2026-06-17, at `4c57c37e4`); rounds 2-4 stayed local. That verdict is still the posted state, and all three rows of its regression table now hold at `60dca7f36`:

| receiver form | Go | round 1 (`4c57c37e4`) | head (`60dca7f36`) |
|---|---|---|---|
| `defer pt.M()` concrete | eager, `f()` = 0 | call-time, = 1 | eager, = 0 |
| `defer i.M()` interface | call-time, = 1 | call-time, = 1 | call-time, = 1 |
| `G=i.M`; persist; `G()` | panic | no panic | panic |

The round-1 blocker was that `nilReceiverPanic` was read at the single `VPDerefValMethod` site, too late to tell a concrete `*T` from an unwrapped interface. That flag no longer exists: the decision moved to the bind site, where `VPInterface` binds lazily and `VPDerefValMethod` derefs eagerly, which is where round 1 asked for it. The ADR round 1 asked for is at [`gnovm/adr/pr5737_nil_value_method_panic_timing.md`](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/adr/pr5737_nil_value_method_panic_timing.md). Rows 1 and 2 verified by running the same program under gno and go1.26.5; row 3 is pinned by [`method_nil_value_persist.txtar`](https://github.com/gnolang/gno/blob/60dca7f36/gno.land/pkg/integration/testdata/method_nil_value_persist.txtar) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/integration/testdata/method_nil_value_persist.txtar), green at head.

## Round-3 findings: status

- **Warning — stored method value over a local type bricks on reload**: **carried, unchanged.** Still reproduces at `60dca7f36`: the reload tx fails with `unexpected type with id ...S` where master returns `(7 int)`. Reattributed this round to [@omarsy](https://github.com/gnolang/gno/pull/5737#discussion_r3513857885), who raised it on 2026-07-02, before round 3; round 3 recorded it as a new finding, which was wrong.
- **Missing test — struct-carried cycle has only in-memory coverage**: **carried.** [`method_iface_cyclic_persist.txtar`](https://github.com/gnolang/gno/blob/60dca7f36/gno.land/pkg/integration/testdata/method_iface_cyclic_persist.txtar) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/integration/testdata/method_iface_cyclic_persist.txtar) pins the *pointer* cycle (`s.IG = s`) on the persisted path; the *struct-carried* cycle (`s.IG = W{s}`, the `dd77e415f` fix) is still pinned only by the in-memory filetest.
- **Nit — gas constants ratio-scaled, not measured**: **carried.** Unchanged this round.
- **Nit — `IsCrossing()` lazy branch is defensive**: **carried.** Unchanged this round.

## Critical (must fix)

None.

## Warnings (should fix)

- **[stored method value over a local type bricks on reload]** [@omarsy](https://github.com/gnolang/gno/pull/5737#discussion_r3513857885) [`values.go:784`](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/values.go#L784) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L784) — a method value bound over a function-local type, persisted then reloaded, panics `unexpected type with id ...S` where master returned the value.
  <details><summary>details</summary>

  The lazy bind saves the original operand and its type into `Receiver`; at the call `resolveLazyBound` re-derives the dispatch trail on that type, which reaches `Store.GetType`. A function-local type's TypeID is never written to the type store, so the reload panics. Master eager-bound the concrete method and stored the promoted package-level receiver `T{7}`, so its reload returned `(7 int)`.

  This rides a pre-existing limitation: a raw interface value over a local type (`var G I = S{}; ...; G.Get()` across reload) already panics on both master and this PR, so local types are non-persistable in general. Master's eager method-value bind dodged it by resolving to a package-level receiver; the lazy bind no longer dodges it. The trigger is narrow, but it is a real behavior regression, and the failure is a raw internal panic rather than a clean error. Re-verified at `60dca7f36`, unchanged from round 3. Fix: either resolve eagerly when the operand type is a local (non-persistable) type, restoring master's result, or reject the bind with a clear message instead of an internal `unexpected type with id`. Regression pinned in [`tests/method_iface_local_type_persist.txtar`](tests/method_iface_local_type_persist.txtar) (master `(7 int)`, this PR panics).
  </details>

## Nits

- **[stale comment describes the rejected design]** [`method_nil_value_bind.gno:1-5`](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/tests/files/method_nil_value_bind.gno#L1-L5) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/tests/files/method_nil_value_bind.gno#L1-L5) — the header says a concrete `defer pt.M()` on a nil `pt` defers its panic to call time and returns 1; it panics eagerly and returns 0.
  <details><summary>details</summary>

  The comment is verbatim from `73d25b560`, the first commit, when the `nilReceiverPanic` flag deferred both the concrete and the interface case. The redesign split them: `VPDerefValMethod` now derefs eagerly for a concrete bind ([values.go:1971-1977](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/values.go#L1971-L1977) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1971-L1977)), which is the whole point of the round-1 objection. Confirmed behaviorally: the same program returns `concrete 0` on both go1.26.5 and gno at `60dca7f36`, against the 1 the comment claims. The file only exercises the interface case, so no test contradicts the comment and it will outlive anyone who reads it. Fix: drop the `defer pt.M()` / `pt itself is nil` parentheticals, or state that the concrete case panics at the bind.
  </details>

- [`walker.go:500`](https://github.com/gnolang/gno/blob/60dca7f36/gno.land/pkg/gnoweb/feature/state/walker.go#L500) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/gnoweb/feature/state/walker.go#L500) — the lazy branch renders `func Get()` for a method whose real signature is `func() int`, so the displayed result type is dropped. Confirmed behaviorally: the live state page for a persisted `G = i.Get` over `func (T) Get() int` shows `func Get()`. The resolved-bind path shows the true signature via `funcSignature(fv.Type)`, so the two disagree on the same value. No cheap fix exists in the walker (the signature lives behind a store lookup the walker doesn't do), and this mirrors `decodeFuncInline`'s own name-only fallback at [walker.go:710-711](https://github.com/gnolang/gno/blob/60dca7f36/gno.land/pkg/gnoweb/feature/state/walker.go#L710-L711) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/gnoweb/feature/state/walker.go#L710-L711); flagging for whoever touches the walker next.
- [`machine.go:1410`](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/machine.go#L1410) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/machine.go#L1410) — `OpCPULazyBoundResolve` (529) and [`OpCPUSelectorInterface`](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/machine.go#L1471) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/machine.go#L1471) (276) are ratio-scaled, not measured on the gas-table reference hardware; the in-code TODO documents the methodology and defers direct measurement to the next HW refresh. Consensus-affecting per interface call, but on the same footing as other ratio-scaled table entries; flagging for whoever refreshes the gas table.
- [`values.go:801-811`](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/pkg/gnolang/values.go#L801-L811) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L801-L811) — `(*BoundMethodValue).IsCrossing()` returns `false` for a lazy bind; `doOpPrecall` resolves the bind before reading `IsCrossing()` on the concrete `fn`, so the lazy branch is defensive and off the call path. No behavior change.

## Missing Tests

- **[cyclic-value fix has only in-memory coverage]** [`gnovm/tests/files/method_iface_cyclic_value.gno`](https://github.com/gnolang/gno/blob/60dca7f36/gnovm/tests/files/method_iface_cyclic_value.gno) · [↗](../../../../../.worktrees/gno-review-5737/gnovm/tests/files/method_iface_cyclic_value.gno) — the shipped filetest exercises the struct-carried cycle in a single in-memory run; the persisted and unbounded query and `Render()` paths [@omarsy](https://github.com/gnolang/gno/pull/5737#discussion_r3543718293) flagged are pinned for the pointer cycle only.
  <details><summary>details</summary>

  The fix keys the seen-set on `*StructValue` identity and relies on that identity staying cache-stable when the operand is reloaded from the store. I verified this holds on both the persisted tx path and the unbounded qrender path (each terminates in ~3.5s with the fatal cyclic panic, no hang). [`method_iface_cyclic_persist.txtar`](https://github.com/gnolang/gno/blob/60dca7f36/gno.land/pkg/integration/testdata/method_iface_cyclic_persist.txtar) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/integration/testdata/method_iface_cyclic_persist.txtar) covers the reload path for `s.IG = s`, whose operand stays a `PointerValue`; it never exercises the `*StructValue` key that `dd77e415f` added. A regression test on the reload path for the struct-carried shape would guard the identity-stability assumption against a future store-cache change; the in-memory filetest would not catch such a break. Ready to add: [`tests/method_iface_cyclic_value_persist.txtar`](tests/method_iface_cyclic_value_persist.txtar), which stores the cycle and hits it through qrender.
  </details>

## Verified

- The walker fix closes a real break on the live gnoweb path, not a synthetic one. Booted gnodev from this worktree over a realm holding `var G = i.Get` (a package-level lazy bind), and the object query returns exactly the guarded shape: `"Func":null,"Method":"Get","MethodPkg":"gno.land/r/lazy"`. The state page for that object returns 200 and renders `func Get()`. Rebuilding gnodev with the `IsLazy()` branch deleted turns the same URL into HTTP 500 `failed to decode state object`, logging `panic recovered: runtime error: invalid memory address or nil pointer dereference`. `recoverToErr` contains it, so the blast radius is a dead state view for that object, never a node crash — matching [@omarsy](https://github.com/gnolang/gno/pull/5737#discussion_r3578551168)'s own impact read.
- Only one consumer outside the VM touches `BoundMethodValue`: `walker.go:495`, now guarded. A lazy bind nested inside a struct field never reaches it — `decodeTypedValueAt` has no `BoundMethodValue` case and falls through to its `<%T>` fallback, which is what master does for an eager bind too, so no regression hides there. `decodeValueChildrenTyped` delegates to the guarded `decodeValueChildren`.
- The `typedRuntimeError` switch is what makes `recover().(error)` true, and it matches Go. Reverting the one word at `values.go:1923` back to `typedString` flips `method_iface_dynamic.gno` from `c5 2` to `c5 1`. The equivalent Go program recovers a `runtime.errorString`, satisfies `error`, and prints `c5 2` on go1.26.5, so the filetest's new assertion pins real Go parity rather than a GnoVM convention.
- Green at `60dca7f36`: all 19 `method_iface_*` / `method_nil_value_*` filetests plus `zrealm_method_iface.gno`, the full `gno.land/pkg/gnoweb/feature/state` package, and the three consensus gas txtars (`stdlib_restart_compare`, `restart_gas`, `simulate_gas`). The `pb3_gen.go` regeneration is covered by the green `genproto2` job, which byte-compares the committed file against a fresh `make -C misc/genproto2`. The `docs` check is red on a dead external link in `docs/MANIFESTO.md`, a file this PR does not touch.

## Open questions

- The walker renders a lazy bind's `Method` name but an eager bind's signature, so the same conceptual value reads differently depending on whether it happens to be resolved. Closing that would mean teaching the walker to resolve a method signature from `Receiver.T`, which needs a store lookup it deliberately avoids. Not posted: no defect, and the fix is a walker redesign well outside this PR.
- Round 3 recorded the local-type persistence regression as its own discovery when @omarsy had raised it ten days earlier. Round 3's draft was never posted, so nothing wrong reached the PR; the round-4 draft SKIPs the section as a duplicate and thumbs-ups his thread instead. Noting it here because the miss was a process gap, not a code one.
