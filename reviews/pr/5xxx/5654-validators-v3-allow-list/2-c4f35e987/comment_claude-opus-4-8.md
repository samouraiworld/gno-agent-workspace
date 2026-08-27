# Review: PR [#5654](https://github.com/gnolang/gno/pull/5654)
Posted: https://github.com/gnolang/gno/pull/5654#pullrequestreview-5042800816
Event: COMMENT
Status: posted as an AI, marker left in the long form: this is the first review of a PR by this author. Original verdict REQUEST_CHANGES, forced to COMMENT.

## Body
[AI review, not manually checked, opus 4.8]
Status: REQUEST_CHANGES

The 20+ red checks share one root cause. [`allowed.gno`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno) [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno) no longer preprocesses against master, and [`r/gnops/valopers`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/gnops/valopers/valopers.gno) [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/gnops/valopers/valopers.gno) imports v3, so lint, test, e2e and every validator scenario fail downstream of that one package.

## examples/gno.land/r/sys/validators/v3/allowed.gno:16 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L16) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L16) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391345)
`assertCallerIsAllowed` calls `runtime.PreviousRealm()`, which master moved to [`chain/runtime/unsafe`](https://github.com/gnolang/gno/blob/f3d5a5d13/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno?plain=1#L26). Route caller-auth through `cur.Previous().PkgPath()` instead; both `AddValidator` and `RemoveValidator` already thread `cur realm`. Three more calls use pre-drift signatures: `newValoperChangeExecutor` now takes `(cur, changes)`, `Execute` takes `cross(cur)`, and `NewSimpleExecutor` takes `(0, cur, callback, "")`.

## examples/gno.land/r/sys/validators/v3/allowed.gno:30-46 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L30-L46) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L30) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391354)
`AddValidator` never checks that `signingAddress` derives from `signingPubKey`. A mismatched pair publishes the validator under the pubkey-derived address while the cache records the passed address, and `RemoveValidator` then looks up the cached address in a set keyed by the derived one and panics, so the validator can never be removed. Derive `chain.PubKeyAddress(signingPubKey)` and reject a mismatch before the cache write.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5654 -R gnolang/gno
cat > examples/gno.land/r/sys/validators/v3/zz_repro_test.gno <<'EOF'
package validators

import (
	"testing"

	"gno.land/p/nt/bptree/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestReproMismatchStuckValidator(cur realm, t *testing.T) {
	allowedRealms = bptree.NewBPTree32()
	resetValset(t)
	resetCache()
	testing.SetSysParamStrings(module, submodule, currKey, []string{pubKeyA + ":10"})
	allowedRealms.Set("gno.land/r/demo/ics", true)
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/ics"))

	op := testutils.TestAddress("op-mismatch")
	addrB := mustAddr(t, pubKeyB) // real derived address of pubKeyB
	addrC := mustAddr(t, pubKeyC) // wrong address we pass in

	// pubKeyB paired with addrC (addrC != addr(pubKeyB)): accepted, no panic.
	AddValidator(cross, ValoperChange{OperatorAddress: op, Power: 7}, pubKeyB, addrC)

	uassert.True(t, IsValidator(addrB))  // live validator is addrB
	uassert.False(t, IsValidator(addrC)) // addrC never became a validator

	// Operator-keyed remove looks up the cached addrC in a set keyed by addrB.
	urequire.AbortsWithMessage(t, "validator does not exist: "+addrC.String(), func() {
		RemoveValidator(cross, op)
	})
}
EOF
cd examples && go run ../gnovm/cmd/gno test -v -run TestReproMismatchStuckValidator ./gno.land/r/sys/validators/v3
cd .. && rm examples/gno.land/r/sys/validators/v3/zz_repro_test.gno
```

```
=== RUN   TestReproMismatchStuckValidator
--- PASS: TestReproMismatchStuckValidator (0.00s)
ok      ./gno.land/r/sys/validators/v3 	1.00s
```
The green run confirms the mismatched pair is accepted, the validator is live under `addrB`, and the operator-keyed remove panics on `addrC`, so it is stuck.
</details>

## examples/gno.land/r/sys/validators/v3/allowed.gno:35-41 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L35-L41) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L35) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391363)
Re-adding an operator already in `valoperCache` silently drops the passed `signingPubKey`/`signingAddress` and keeps the cached pair. The executor then publishes the stale signing key, so a caller rotating the key gets the old one, not the new one. Panic when the passed pubkey differs from the cached value, or overwrite and document `AddValidator` as a rotation entry point.

## examples/gno.land/r/sys/validators/v3/allowed.gno:53-54 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L53-L54) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L53) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391375)
`RemoveValidator` returns without error when the operator is not in `valoperCache`, so a caller treating a non-panic as "removed" is wrong. The operator-keyed proposal path panics on the same missing condition, so the two disagree. Panic with an "unknown operator" message, or return `(removed bool)`, and pin it in a test.

## examples/gno.land/r/sys/validators/v3/allowed.gno:36 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L36) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L36) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391384)
Operators auto-registered here have no `r/gnops/valopers` profile, but the [`valoperCache`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/cache.gno#L19-L26) comment states the cache mirrors valopers and is written only by `NotifyValoperChanged`. An operator added this way can never rotate its key or opt out via `UpdateKeepRunning`. Extend the invariant comment to cover allow-list-side registration and its consequences, or create the profile.

## examples/gno.land/r/sys/validators/v3/allowed.gno:83-135 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L83-L135) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L83) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391392)
`NewPropAllowedRealmUpdateRequest` caps neither `len(add)+len(remove)` nor the total whitelist size. The sibling [`NewValidatorProposalRequest`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L70-L72) caps at 40, so one proposal here can render a multi-MB description and iterate a huge whitelist diff in a single block. Cap add+remove.

## examples/gno.land/r/sys/validators/v3/allowed.gno:93-105 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L93-L105) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L93) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391401)
The add/remove loops validate uniqueness only, so an empty or malformed path enters the bptree and pollutes `GetAllowedRealms`. Apply per entry the same [`strings.TrimSpace` plus non-empty guard](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L84-L87) this builder already runs on the title.

## examples/gno.land/r/sys/validators/v3/allowed.gno:77-82 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L77-L82) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L77) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391411)
`NewPropAllowedRealmUpdateRequest` is non-crossing, so a direct user MsgCall fails with a confusing error and no pointer to the right facade. The sibling documents this exact case at [`proposal.gno:34-37`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L34-L37). Add the same paragraph naming the intended facade.

## examples/gno.land/r/sys/validators/v3/allowed.gno:121-131 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L121-L131) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L121) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391417)
Removing a realm from the allow list leaves the validators that realm added in the active set, with no cleanup hook or provenance tracking. Governance can disenfranchise a realm while its validators stay live. Document the intent, or add a remove-realm-and-its-validators primitive.

## examples/gno.land/r/sys/validators/v3/allowed_test.gno:79 [gh](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed_test.gno#L79) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed_test.gno#L79) [posted](https://github.com/gnolang/gno/pull/5654#discussion_r3873391423)
Missing test: `AddValidator` with a signing pubkey and address that don't match.

<details><summary>test cases</summary>

```go
func TestAddValidator_RejectsMismatchedSigningPair(cur realm, t *testing.T) {
	allowedRealms = bptree.NewBPTree32()
	resetValset(t)
	resetCache()
	testing.SetSysParamStrings(module, submodule, currKey, []string{pubKeyA + ":10"})

	const allowedPath = "gno.land/r/demo/ics"
	allowedRealms.Set(allowedPath, true)
	testing.SetRealm(testing.NewCodeRealm(allowedPath))

	op := testutils.TestAddress("op-mismatch")
	// pubKeyB paired with addr(pubKeyC): the address does not derive from the pubkey.
	urequire.AbortsWithMessage(t,
		"signingAddress does not match signingPubKey",
		func() {
			AddValidator(cross, ValoperChange{OperatorAddress: op, Power: 7}, pubKeyB, mustAddr(t, pubKeyC))
		},
	)
}
```
Asserts the post-fix state. It fails today, since the pair is accepted and the validator becomes unremovable, and passes once `AddValidator` derives the address from the pubkey and rejects a mismatch. Adjust the message to whatever the fix panics with.
</details>
