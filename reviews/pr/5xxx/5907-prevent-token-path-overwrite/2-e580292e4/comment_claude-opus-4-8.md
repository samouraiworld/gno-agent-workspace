# Review: PR [#5907](https://github.com/gnolang/gno/pull/5907)
Event: APPROVE

## Body
The round-1 lookup blocker is fixed: the integration test resolves foo20 by its symbol key and passes on e580292e4.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5907-prevent-token-path-overwrite/2-e580292e4/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:43 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43)
Nit: the event still emits `slug`, but `slug` is no longer part of the registry key. Callers still pass one, like [`grc20factory`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L51) with the symbol or [`tokenhub`](https://github.com/gnolang/gno/blob/e580292e4/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L34) with a user slug, so an indexer that rebuilds the key from `pkgpath + slug` resolves to no entry. Drop `slug` from the event, or document that it is validated and emitted but never keyed.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:24 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L24)
Missing test: the new `if token == nil` guard has no coverage, and removing it degrades the reject from `grc20reg: nil token` to an opaque `value method ...grc20.Token.GetSymbol called using nil *Token pointer` abort.

<details><summary>test cases</summary>

```go
func TestRegisterRejectsNilToken(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_nil"))
	urequire.AbortsContains(t, cur, "nil token", func() {
		Register(cross(cur), nil, "")
	})
}
```

Passes on the head, aborts with the opaque `GetSymbol` message when the guard is removed.
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:33 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L33)
Nit: registering from the wrong realm, or a direct EOA call with an empty `PkgPath`, panics `token ID mismatch`, which reads like a broken internal invariant rather than a usage error. A message naming the constraint, register from the token's own realm, would orient a realm author faster.
