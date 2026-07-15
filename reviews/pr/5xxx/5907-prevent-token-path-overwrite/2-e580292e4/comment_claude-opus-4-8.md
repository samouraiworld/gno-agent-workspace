# Review: PR [#5907](https://github.com/gnolang/gno/pull/5907)
Event: APPROVE

## Body
Looks good. The nil, cross-realm, and duplicate guards each reject before the registry write.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5907-prevent-token-path-overwrite/2-e580292e4/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:43 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43)
Nit: the event emits `slug`, but the key is built from the symbol, not `slug`. An indexer that rebuilds the key as `pkgpath + slug` gets a nonexistent entry. Drop `slug`, or document that it is emitted but not part of the key.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:24 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L24)
Missing test: the `if token == nil` guard has no coverage. Without it a nil token aborts inside `GetSymbol` on a nil pointer instead of with `grc20reg: nil token`.

<details><summary>test cases</summary>

```go
func TestRegisterRejectsNilToken(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_nil"))
	urequire.AbortsContains(t, cur, "nil token", func() {
		Register(cross(cur), nil, "")
	})
}
```

Passes with the guard, fails without it.
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:33 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L33)
Nit: registering from a realm other than the token's own panics `token ID mismatch`, which reads like an internal bug, not a usage error. A message like "register from the token's own realm" would be clearer.
