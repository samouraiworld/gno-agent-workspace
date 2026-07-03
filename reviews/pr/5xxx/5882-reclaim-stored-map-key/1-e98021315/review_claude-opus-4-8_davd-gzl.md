# PR [#5882](https://github.com/gnolang/gno/pull/5882): fix(gnovm): reclaim stored key object on delete() of object-keyed map

URL: https://github.com/gnolang/gno/pull/5882
Author: omarsy | Base: master | Files: 3 | +113 -8
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: e98021315 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5882 e98021315`

**TL;DR:** `delete(m, key)` on a realm map whose keys are arrays or structs dropped the map entry but left the key's stored object behind in realm storage, so storage grew on every add/delete cycle and the key's deposit was never refunded. The fix makes `delete` also reclaim that stored key object.

**Verdict: APPROVE** — minimal, correct fix; red→green verified, generalizes to struct keys, single caller, no new defect.

## Summary
The `delete` builtin removed a map entry, then detached the wrong object: it marked the caller's argument key (`itv`), not the distinct stored key object the map holds (an `iv.Copy` child created at insertion). The stored key kept `RefCount == 1`, was never marked deleted, and was never removed from the store, so a `[1]int`-keyed realm leaked ~213 bytes per add/delete cycle (a 2-int struct key leaks ~252), and the storage deposit was never refunded. The value side already used the stored value and was correct. The fix has `MapValue.DeleteForKey` return the stored key it removed; the builtin marks that key's object in `DidUpdate`. Deterministic. The leak is specific to array and struct keys, which insertion deep-copies into a distinct object the map owns; primitive keys have no separate object, and pointer keys share the pointee with the caller, so `GetFirstObject` on the old argument key already resolved to the right object. Both were and remain unaffected.

## Glossary
- storage deposit: per-realm refundable charge for on-chain storage, released in proportion to the byte delta on removal.

## Fix
`DeleteForKey` now returns the stored key `TypedValue` it removed (`nil` for NaN or absent) at [`values.go:828-840`](https://github.com/gnolang/gno/blob/e98021315/gnovm/pkg/gnolang/values.go#L828-L840) · [↗](../../../../../.worktrees/gno-review-5882/gnovm/pkg/gnolang/values.go#L828). The `delete` builtin passes that stored key's first object to `DidUpdate` at [`uverse.go:1016-1024`](https://github.com/gnolang/gno/blob/e98021315/gnovm/pkg/gnolang/uverse.go#L1016-L1024) · [↗](../../../../../.worktrees/gno-review-5882/gnovm/pkg/gnolang/uverse.go#L1016), so the object's ref count is decremented and it is marked deleted once it reaches zero. The load-bearing constraint is that `DidUpdate` must receive the object the map actually persisted, not the transient argument key; `GetFirstObject` lazily loads the stored key from the store when it is an unfilled `RefValue`, so returning `&mli.Key` without filling it is safe. `DeleteForKey` has one caller, so the signature change is contained.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- `gnovm/tests/files/zrealm_map5.gno` — the regression golden covers only an array (`[1]int`) key; struct and pointer keys are not asserted.
  <details><summary>details</summary>

  The fix is type-agnostic: `GetFirstObject` returns the array/struct value directly, so one code path serves both value-composite key shapes. Verified behaviorally: a `struct{A,B int}` key reclaims identically (`d[...:8](-252)`); deleting an entry whose array-key value is also held through another variable leaves that outside copy readable and reclaims only the map's private copy (`d[...:9](-213)`, `keep[0]` still 1); a pointer key emits no key deletion because the pointee is shared and only decremented. The array golden is representative and the gap is low-risk. A second golden with a struct key would lock the general case. Not posted.
  </details>

## Suggestions
- [design: where the realm bookkeeping lives] [`values.go:828-840`](https://github.com/gnolang/gno/blob/e98021315/gnovm/pkg/gnolang/values.go#L828-L840) · [↗](../../../../../.worktrees/gno-review-5882/gnovm/pkg/gnolang/values.go#L828) — `DeleteForKey` returns the removed key for the builtin to mark, mirroring `GetValueForKey` returning the value; consider whether the key's `DidUpdate` belongs inside `DeleteForKey` instead.
  <details><summary>details</summary>

  The builtin marks both the key object and the value object via `DidUpdate` at [`uverse.go:1021-1028`](https://github.com/gnolang/gno/blob/e98021315/gnovm/pkg/gnolang/uverse.go#L1021-L1028), keeping realm bookkeeping in one place while `MapValue` methods stay pure container ops. `DeleteForKey` already takes `m *Machine` and `m.Realm.DidUpdate` is nil-safe, so it could mark the key itself; that would consolidate the key path but split it from the value-side marking. Either shape is fine. Posted as a question so the author can confirm the layering was deliberate.
  </details>

## Open questions
- Reassignment of an existing object key (`m[k] = v2` with a fresh equal key, `GetPointerForKey` `mli.Key = key` at [`values.go:791`](https://github.com/gnolang/gno/blob/e98021315/gnovm/pkg/gnolang/values.go#L791)) already reclaims the old key object and creates the new one (verified: finalize emits `d[...:8]` + `c[...:10]`), so it does not share this leak. No action; confirms the fix's scope is the delete path only.
- The fix changes the finalize output and storage-deposit accounting of `delete` on object-keyed maps, so it is a state-machine change that all nodes on the new binary agree on. It does not retroactively reclaim keys already leaked on-chain by past deletes. Inherent to a VM correctness fix; handled by the release process, not a code concern.
