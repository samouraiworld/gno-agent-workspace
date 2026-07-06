# Review: PR [#5880](https://github.com/gnolang/gno/pull/5880)
Event: REQUEST_CHANGES

## Body
These checklist findings share one root cause: the Review Checklist restates the Quick Checks it summarizes, and each restatement drifts. Some lines are narrower than the check, so they pass code the check calls WRONG; one is broader than the source, so it flags patterns the guide blesses. Verified on 26ca914e2: a `/p/`-typed-pointer callback writes victim state where an `/r/`-declared parameter type is blocked by readonly taint, and a bare `var savedRealm realm` runs clean while assigning a live realm panics only at transaction finalize.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5880-ai-contract-review-guide/1-26ca914e2/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/resources/gno-ai-contract-review.md:48-56 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L48)
The pair implies adding a `*MyState` pointer is the fix, when the fix is declaring the callback's parameter type in your `/r/`. As written the WRONG `func()` gets no pointer so it can't reach your state, and the comment's "inherits the caller's `m.Realm`" is backwards since `m.Realm` stays your realm's. Swap the WRONG example to a `/p/`-typed-pointer callback.

## docs/resources/gno-ai-contract-review.md:97 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L97)
This line flags only a callback with a `/p/`-typed parameter, but Check 4's WRONG example is `func()` with no parameter, so walking the checklist passes the code Check 4 calls WRONG. The danger is the supplied value being a top-level `/p/` function, not the parameter type. Widen it to any caller-supplied func or interface value `/p/` code can satisfy.

## docs/resources/gno-ai-contract-review.md:101 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L101)
Stated unconditionally, this flags every grc20 token realm: a token realm holds its balances in a `/p/`-declared `grc20.Token`, which the security guide blesses as long as the state stays in unexported vars. It is also the one checklist line with no matching Quick Check, so no example tempers the false positive. Add the unexported-storage qualifier.

## docs/resources/gno-ai-contract-review.md:96 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L96)
Check 3 and this line say only "pointer", so a returned internal slice or map header passes. The security guide's headline example for this class is a slice return, `func Users() []*User`. Cover pointer, slice, and map.

## docs/resources/gno-ai-contract-review.md:111 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L111)
The row credits `gno-interrealm.md` with `IsCurrent()`, but that file never mentions it and its header marks it outdated. `IsCurrent()` lives in `gno-interrealm-v2.md`. Point the row there.

## docs/resources/gno-ai-contract-review.md:76-77 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L76)
A bare `var savedRealm realm` with nothing assigned runs clean, and the panic fires on assignment of a live realm at transaction finalize, not at attach. Move the comment onto an assignment line and say finalize.

## docs/resources/gno-ai-contract-review.md:94 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L94)
This says "call `cur.IsCurrent()`", which a no-op `_ = cur.IsCurrent()` satisfies while giving no protection. The result has to gate the body, as Check 1 shows. Say "panic unless `cur.IsCurrent()`".

## docs/resources/gno-ai-contract-review.md:30 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L30)
`IsUser` is a method on a realm value, so the bare `if !IsUser()` here is not valid gno and breaks the parallel with the RIGHT line's `cur.Previous().IsUserCall()`. Write `cur.Previous().IsUser()`.

## docs/resources/gno-ai-contract-review.md:61 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L61)
Embedding passes seal/marker checks, not the canonical-type assert, which is nominal and is the RIGHT fix on the next lines. As written the comment reads as if embedding defeats the assert you recommend. Say "passes seal/marker checks".

## docs/resources/gno-ai-contract-review.md:99 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L99)
The forbidden-storage list here omits map values, which the guide includes, so a `map[string]realm` field slips past. Add map values and slice elements.

## docs/resources/gno-ai-contract-review.md:80-82 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L80)
The `Save` example reads `cur.Previous().Address()` without the `IsCurrent()` guard Check 1 makes standard. Harmless for a true crossing function, but copied into a non-crossing context it becomes an unguarded caller read. Add the guard, or note in Check 1 that it is needed only when the realm value is caller-passed.

## docs/resources/gno-ai-contract-review.md:100 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L100)
This keeps only "unexported" from Check 7, but the body also forbids returning aliased pointers; a `/p/`-embedded value whose method is promoted and reached through an exported getter needs no exported field. Add "and not reachable via a returned or promoted method".
