# Review: PR [#5880](https://github.com/gnolang/gno/pull/5880)
Event: REQUEST_CHANGES

## Body
Seven of the fourteen findings sit on the Review Checklist, where each item restates a Quick Check in one sentence. Five restate their check too narrowly, so code the check catches passes the item; one of them passes Check 4's own wrong example. Two restate it too broadly: one flags every grc20 token realm, the other flags any import of `chain/runtime/unsafe`, though that package also exports `OriginCaller()` and `OriginSend()`. All fixes are wording changes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5880-ai-contract-review-guide/2-365f5eb91/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/resources/gno-ai-contract-review.md:48-56 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L48)
The pair implies adding a `*MyState` pointer is the fix, but the fix is declaring the callback's parameter type in your `/r/`. The wrong `func()` gets no pointer and cannot name any `/r/` symbol, so it can't reach your state. The comment's "inherits the caller's `m.Realm`" is also backwards, since `m.Realm` stays your realm's. Swap the wrong example to a `/p/`-typed-pointer callback.

## docs/resources/gno-ai-contract-review.md:194 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L194)
This line flags only a callback with a `/p/`-typed parameter, but Check 4's wrong example is `func()` with no parameter, so walking the checklist passes the code Check 4 calls wrong. The danger is the supplied value being a top-level `/p/` function, not the parameter type.

```go
// Check 4's own WRONG example. No /p/-typed parameter, so this line passes it.
func ApplyHook(fn func()) { fn() }
```

Widen it to any caller-supplied func or interface value `/p/` code can satisfy.

## docs/resources/gno-ai-contract-review.md:198 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L198)
Stated unconditionally, this flags every grc20 token realm: a token realm keeps its balances in the `/p/`-declared `grc20.Token` and `grc20.PrivateLedger`, which §4 of the security guide calls the canonical example of safe `/p/` data.

```go
// sensitive state in /p/-declared types, stored unexported:
// exactly what §8's escape clause permits, and this line flags it
var (
	token *grc20.Token
	admin *grc20.PrivateLedger
)
```

It is also the one checklist line with no matching Quick Check, so no example tempers the false positive. Add the unexported-storage qualifier.

## docs/resources/gno-ai-contract-review.md:191 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L191)
Check 3 and this line say only "pointer", so a returned internal slice or map header passes. The security guide's headline example for this class is a slice return.

```go
// a slice header is not a pointer, so this line passes it; the caller
// can still call any mutation method on the elements
var users []*User
func Users() []*User { return users }
```

Cover pointer, slice, and map.

## docs/resources/gno-ai-contract-review.md:209 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L209)
The row credits `gno-interrealm.md` with `IsCurrent()`, but that file never mentions it and its header marks it outdated. `IsCurrent()` lives in `gno-interrealm-v2.md`. Point the row there.

## docs/resources/gno-ai-contract-review.md:189 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L189)
This line and Check 9's rule flag any import of `chain/runtime/unsafe` alongside `cur realm`, but that package also exports `OriginCaller()` and `OriginSend()`. Their godoc keeps them for tx-origin uses, event emission and fee attribution for the first, the payment envelope for the second, both paired with `runtime.AssertOriginCall()`. `cur` gives the immediate caller, not the tx origin, so neither has a `cur` substitute and a crossing realm can legitimately import them.

```go
// r/gnoland/wugnot: crossing, imports unsafe, and correct as written
func Deposit(cur realm) {
	runtime.AssertOriginCall()
	caller := cur.Previous().Address()
	sent := unsafe.OriginSend() // the tx envelope; cur cannot reach it
	...
}
```

Scope the rule to `PreviousRealm()` and `CurrentRealm()` used for caller identity.

## docs/resources/gno-ai-contract-review.md:76-77 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L76)
A bare `var savedRealm realm` runs clean, and the panic fires on assignment of a live realm at transaction finalize, not at attach. Move the comment onto an assignment line and say finalize.

## docs/resources/gno-ai-contract-review.md:188 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L188)
This says "call `cur.IsCurrent()`", which a no-op satisfies while giving no protection.

```go
func Set(cur realm, key, value string) {
	_ = cur.IsCurrent() // the line is satisfied; nothing is guarded
	store.Set(key, value)
}
```

The result has to gate the body, as Check 1 shows. Say "panic unless `cur.IsCurrent()`".

## docs/resources/gno-ai-contract-review.md:30 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L30)
`IsUser` is a method on a realm value, so the bare `if !IsUser()` here is not valid gno and breaks the parallel with the right line's `cur.Previous().IsUserCall()`. Write `cur.Previous().IsUser()`.

## docs/resources/gno-ai-contract-review.md:61 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L61)
Embedding passes seal/marker checks, not the canonical-type assert, which is nominal and is the right fix on the next lines. As written the comment reads as if embedding defeats the assert you recommend. Say "passes seal/marker checks".

## docs/resources/gno-ai-contract-review.md:196 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L196)
The forbidden-storage list here omits map values and slice elements, which the guide includes.

```go
// the var is map-typed, not realm-typed, so this line passes it
var seen = map[string]realm{}
```

Add both.

## docs/resources/gno-security-guide.md:230 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L230)
§5.1a says `avl.Tree` has fields `root` and `size`, but the real struct is `type Tree struct { node *Node }` with a single unexported `node` field. The point still holds: unexported fields with exported `Set`/`Remove` mutators. Only the field names are wrong. Use `node`, or drop the parenthetical.

## docs/resources/gno-ai-contract-review.md:80-82 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L80)
The `Save` example reads `cur.Previous().Address()` without the `IsCurrent()` guard Check 1 makes standard. Harmless for a true crossing function, but copied into a non-crossing context it becomes an unguarded caller read. Add the guard, or note in Check 1 that it is needed only when the realm value is caller-passed.

## docs/resources/gno-ai-contract-review.md:197 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L197)
This keeps only "unexported" from Check 7, but the body also forbids returning aliased pointers. An unexported field handed out by an exported getter satisfies the line and leaks the same handle.

```go
type Registry struct {
	tree *avl.Tree // unexported, so this line passes
}

// the alias escapes anyway, and Iterate then runs a caller-supplied
// /p/ callback under this realm's authority
func (r *Registry) Tree() *avl.Tree { return r.tree }
```

Add "and not returned by any exported function or method".
