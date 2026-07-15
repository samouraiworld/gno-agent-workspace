# Review: PR [#5907](https://github.com/gnolang/gno/pull/5907)
Posted: https://github.com/gnolang/gno/pull/5907#pullrequestreview-4705969385
Event: APPROVE

## Body
This should be rebased on https://github.com/gnolang/gno/pull/5908.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:43 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) [posted](https://github.com/gnolang/gno/pull/5907#discussion_r3588887917)
Nit: the event emits `slug`, but it is not part of the key anymore.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:24 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L24) [posted](https://github.com/gnolang/gno/pull/5907#discussion_r3588887923)
Missing test: nothing covers the `if token == nil` guard.

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

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:33 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L33) [posted](https://github.com/gnolang/gno/pull/5907#discussion_r3588887925)
Nit: a wrong-realm registration panics `token ID mismatch`, which reads like an internal bug. A clearer message would name the rule: register from the token's own realm.
