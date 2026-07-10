# Review: PR [#5880](https://github.com/gnolang/gno/pull/5880)
Event: REQUEST_CHANGES

## Body
Most findings share one root cause: the Review Checklist restates the Quick Checks, and the restatements drift. Narrowed lines pass code their own check calls wrong. Broadened lines flag patterns the security guide blesses. All fixes are wording changes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5880-ai-contract-review-guide/2-365f5eb91/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/resources/gno-ai-contract-review.md:48-56 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L48)
The pair implies adding a `*MyState` pointer is the fix, when the fix is declaring the callback's parameter type in your `/r/`. The wrong `func()` gets no pointer and [can name no `/r/` symbol](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/pkg/gnolang/preprocess.go#L5450) · [↗](../../../../../.worktrees/gno-review-5880/gnovm/pkg/gnolang/preprocess.go#L5450), so it can't reach your state, and the comment's "inherits the caller's `m.Realm`" is backwards, since [`m.Realm` stays your realm's](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L130-L137) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L130). Swap the wrong example to a `/p/`-typed-pointer callback.

## docs/resources/gno-ai-contract-review.md:194 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L194)
This line flags only a callback with a `/p/`-typed parameter, but [Check 4's wrong example](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L51) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L51) is `func()` with no parameter, so walking the checklist passes the code Check 4 calls wrong. The danger is the supplied value being a top-level `/p/` function, not the parameter type.

```go
// Check 4's own WRONG example. No /p/-typed parameter, so this line passes it.
func ApplyHook(fn func()) { fn() }
```

Widen it to any caller-supplied func or interface value `/p/` code can satisfy.

## docs/resources/gno-ai-contract-review.md:198 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L198)
Stated unconditionally, this flags every grc20 token realm: [balances live](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L15-L18) · [↗](../../../../../.worktrees/gno-review-5880/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L15) in the `/p/`-declared `grc20.Token` and `grc20.PrivateLedger`, which [§4 of the security guide](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L174-L178) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L174) calls the canonical example of safe `/p/` data.

```go
// sensitive state in /p/-declared types, stored unexported:
// exactly what §8's escape clause permits, and this line flags it
var (
	token *grc20.Token
	admin *grc20.PrivateLedger
)
```

It is also the one checklist line with no matching Quick Check, so no example tempers the false positive. Add the guide's [unexported-storage qualifier](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L495-L497) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L495).

## docs/resources/gno-ai-contract-review.md:191 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L191)
[Check 3](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L36-L44) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L36) and this line say only "pointer", so a returned internal slice or map header passes. The security guide's [headline example for this class](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L208-L209) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L208) is a slice return.

```go
// a slice header is not a pointer, so this line passes it; the caller
// can still call any mutation method on the elements
var users []*User
func Users() []*User { return users }
```

## docs/resources/gno-ai-contract-review.md:209 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L209)
The row credits `gno-interrealm.md` with `IsCurrent()`, but that file never mentions it and its [header marks it outdated](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-interrealm.md?plain=1#L3-L5) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-interrealm.md#L3). `IsCurrent()` lives in [`gno-interrealm-v2.md`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-interrealm-v2.md?plain=1#L378) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-interrealm-v2.md#L378).

## docs/resources/gno-ai-contract-review.md:189 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L189)
This line and [Check 9's rule](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L161) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L161) flag any import of `chain/runtime/unsafe` alongside `cur realm`, but that package also exports [`OriginCaller()`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L51) · [↗](../../../../../.worktrees/gno-review-5880/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L51) and [`OriginSend()`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L64) · [↗](../../../../../.worktrees/gno-review-5880/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L64). Their [godoc](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L8-L11) · [↗](../../../../../.worktrees/gno-review-5880/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L8) reserves them for tx-origin uses that `cur` cannot provide, so a crossing realm can legitimately import the package.

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
Nit: a bare `var savedRealm realm` runs clean, and the panic fires on assignment of a live realm at [transaction finalize](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/tests/files/zrealm_cur_persist_var.gno) · [↗](../../../../../.worktrees/gno-review-5880/gnovm/tests/files/zrealm_cur_persist_var.gno), not at attach. Move the comment onto an assignment line and say finalize.

## docs/resources/gno-ai-contract-review.md:188 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L188)
Nit: this says "call `cur.IsCurrent()`", which a no-op satisfies while giving no protection.

```go
func Set(cur realm, key, value string) {
	_ = cur.IsCurrent() // the line is satisfied; nothing is guarded
	store.Set(key, value)
}
```

Say "panic unless `cur.IsCurrent()`", as [Check 1 shows](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L20) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L20).

## docs/resources/gno-ai-contract-review.md:30 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L30)
Nit: [`IsUser`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/pkg/gnolang/uverse.go#L131) · [↗](../../../../../.worktrees/gno-review-5880/gnovm/pkg/gnolang/uverse.go#L131) is a method on a realm value, so the bare `if !IsUser()` here is not valid gno and breaks the parallel with the right line's `cur.Previous().IsUserCall()`. Write `cur.Previous().IsUser()`.

## docs/resources/gno-ai-contract-review.md:61 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L61)
Nit: "bypasses interface checks" reads as if embedding defeats the canonical-type assert the next lines recommend. Embedding passes [seal/marker checks](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L315-L316) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L315), while the assert is [nominal](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/p/demo/tokens/grc20/types.gno#L171) · [↗](../../../../../.worktrees/gno-review-5880/examples/gno.land/p/demo/tokens/grc20/types.gno#L171) and rejects it. Say "passes seal/marker checks".

## docs/resources/gno-ai-contract-review.md:196 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L196)
Nit: the forbidden-storage list here omits map values and slice elements, which [the guide includes](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L523-L524) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L523).

```go
// the var is map-typed, not realm-typed, so this line passes it
var seen = map[string]realm{}
```

## docs/resources/gno-security-guide.md:230 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L230)
Nit: §5.1a says `avl.Tree` has fields `root` and `size`, but the [real struct](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/p/nt/avl/v0/tree.gno#L27) · [↗](../../../../../.worktrees/gno-review-5880/examples/gno.land/p/nt/avl/v0/tree.gno#L27) is `type Tree struct { node *Node }`. The point still holds: unexported fields with exported [`Set`/`Remove`](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/p/nt/avl/v0/tree.gno#L65-L73) · [↗](../../../../../.worktrees/gno-review-5880/examples/gno.land/p/nt/avl/v0/tree.gno#L65) mutators. Use `node`, or drop the parenthetical.

## docs/resources/gno-ai-contract-review.md:80-82 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L80)
Suggestion: the `Save` example reads `cur.Previous().Address()` without the `IsCurrent()` guard [Check 1 makes standard](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L20) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L20). Harmless for a true crossing function, but copied into a non-crossing context it becomes an unguarded caller read. Add the guard, or note in Check 1 that it is needed only when the realm value is caller-passed.

## docs/resources/gno-ai-contract-review.md:197 [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L197)
Suggestion: this keeps only "unexported" from Check 7, but [the body](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L88) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L88) also forbids returning aliased pointers. An unexported field handed out by an exported getter satisfies the line and leaks the same handle.

```go
type Registry struct {
	tree *avl.Tree // unexported, so this line passes
}

// the alias escapes anyway, and Iterate then runs a caller-supplied
// /p/ callback under this realm's authority
func (r *Registry) Tree() *avl.Tree { return r.tree }
```

Add "and not returned by any exported function or method".
