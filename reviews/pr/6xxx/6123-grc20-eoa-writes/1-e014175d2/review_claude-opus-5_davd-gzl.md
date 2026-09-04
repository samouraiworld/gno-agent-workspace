# PR [#6123](https://github.com/gnolang/gno/pull/6123): feat(grc20): add EOA writes through MsgCall

URL: https://github.com/gnolang/gno/pull/6123
Author: notJoon | Base: master | Files: 16 | +796 -21
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: e014175d2 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6123 e014175d2`
Overview: [overview](../overview.md)

## Overview

A GRC20 token decides whose balance a write touches by picking a *teller*. The branch adds [`UserTeller`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L42-L55), which resolves the account from the calling frame like the existing [`CallerTeller`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L24-L36) but refuses the call unless that frame is a signing account reached through `MsgCall`. On top of it, a token can name other realms allowed to relay its users' writes, through [`TrustHost`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L60-L66) and [`UserTellerTrusted`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L84-L97), and the token registry gains [`UserTransfer`, `UserApprove` and `UserTransferFrom`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L152-L167) built on that. The renamed [`guardWrite`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L223-L238) carries both new conditions. Tests come as gno unit tests, filetests, a new fixture realm under `examples/gno.land/r/tests/`, and scripted integration chains.

**Verdict: REQUEST CHANGES** — the trusted-relay mechanism lets a named realm spend any holder's balance unbidden, and the new fixture realm hands its own realm capability to any caller; both ship to users (3 Warnings, 1 missing test, 1 suggestion, 3 not posted).

## Verify first

- [`tellers.gno:234-236`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L234-L236) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L234) — the branch's only widening of who may spend a user's balance. Run [`grc20_trusted_host_drain.txtar`](tests/grc20_trusted_host_drain.txtar) and read what a signed `relay.Claim()` with no arguments costs the signer.
- [`grc20_user_teller_leak.gno:14-16`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L14-L16) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L14) — a new package under `examples/`, which [`start.go:451-455`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/cmd/gnoland/start.go#L451-L455) loads whole into genesis. Run [`grc20_leak_fixture_capability.txtar`](tests/grc20_leak_fixture_capability.txtar).
- [`token.gno:72`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/token.gno#L72) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/token.gno#L72) — the new per-token tree is what moves the genesis storage deposit, which is why [`storage_deposit_price_change.txtar`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L41) carries two re-derived balances. Confirm they are re-derived rather than hand-edited.

## Summary

Before the branch a token realm already served a direct `MsgCall` from a signing user: `CallerTeller` resolves the actor from the frame that crossed into the token realm, and [`wugnot.Transfer`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L89-L92) is a crossing wrapper over it. The branch's own new [`eoa_surface_filetest.gno`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/gnoland/wugnot/filetests/eoa_surface_filetest.gno#L26-L38) passes unchanged against the merge base d43dc0c5019ec8106df2ceca119776e1b846dfc7, so what `UserTeller` adds over `CallerTeller` is the refusal of a non-user caller, not the restoration of a lost surface.

The second half of the branch is a different thing: `TrustHost` lets a token name realms whose frames `guardWrite` will accept, and `UserTellerTrusted` is reachable from the published `*Token` rather than the private ledger. That combination is a spend capability over every holder, granted by the token and exercised without the holder's consent to any particular write. The three registry helpers are its only consumer.

## Numbers

Same filetest, same inputs, run with the branch's `gno` binary against two `examples/` trees.

| `examples/gno.land/r/gnoland/wugnot` filetest | Gas | Storage |
| --- | ---: | --- |
| merge base d43dc0c5019ec8106df2ceca119776e1b846dfc7 | 9,145,698 | `wugnot:+4147b` |
| head e014175d2 | 9,158,076 | `wugnot:+4147b` |

## Warnings (should fix)

- **[security]** [`tellers.gno:60-66`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L60-L66) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L60) — `TrustHost` lets the named realm spend a holder's whole balance on any call that holder makes into it, bounded by no amount, no recipient and no particular function.
  <details><summary>details</summary>

  Trust is not scoped to a function, an amount or a recipient. A trusted realm builds the teller from the published `*Token` and every one of its crossing functions reached by a direct `MsgCall` moves the caller's balance, so the signing view the description sets out to protect shows a function name unrelated to the token. [`grc20_trusted_host_drain.txtar`](tests/grc20_trusted_host_drain.txtar) signs `relay.Claim()` with no arguments and the caller's 1,000,000 DRAIN lands at the relay's address. The package documents a bounded route for a realm moving a user's funds: `Approve`, then `RealmTeller().TransferFrom`, against an amount the holder set for a spender the holder named. This one is bounded by nothing the holder chose. Fix: document that scope at `TrustHost`, or hold `trustedHosts`, `UserTellerTrusted` and the three registry helpers back and ship `UserTeller` alone.
  </details>

- **[security]** [`grc20_user_teller_leak.gno:10-16`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L10-L16) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L10) — the fixture stores any caller's implementation and invokes it holding its own live realm value, which is spend authority over the fixture's address.
  <details><summary>details</summary>

  `Store` takes the interface from any caller and `Transfer` calls it with the fixture's `cur`. [`NewBanker`](https://github.com/gnolang/gno/blob/e014175d2/gnovm/stdlibs/chain/banker/banker.gno#L117-L129) accepts that value after one `IsCurrent()` check and binds the banker to `rlm.Address()`, which is the shape [`gno-ai-contract-review.md`](https://github.com/gnolang/gno/blob/7ae735fdae3632d94a6f6029dc8438366900cc6d/docs/resources/gno-ai-contract-review.md?plain=1#L73-L86) names as a caller-supplied callback holding realm authority. [`start.go:451-455`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/cmd/gnoland/start.go#L451-L455) loads the whole `examples/` tree into genesis, so the package is live on every chain built from this tree. [`grc20_leak_fixture_capability.txtar`](tests/grc20_leak_fixture_capability.txtar) parks 500,000ugnot on the fixture's address and an ordinary signed call empties it. The confinement the fixture demonstrates is already asserted by the branch's own [`grc20reg_user_helpers.txtar`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/pkg/integration/testdata/grc20reg_user_helpers.txtar#L85-L86), whose `untrusted` package leaks a teller inline and needs no permanent deployment. Fix: drop the fixture package and the filetest hunk importing it, or reject in `Store` and `Transfer` any caller other than the filetest realm.
  </details>

- **[state safety]** [`tellers.gno:70-76`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L70-L76) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L70) — `UntrustHost` returns nothing and drops [`avl.Tree.Remove`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/nt/avl/v0/tree.gno#L80)'s `removed` result, so a revocation naming a host the token never stored is indistinguishable from one that withdrew the capability.
  <details><summary>details</summary>

  The token realm's only way to learn the outcome is to attempt a write through the teller. [`untrust_host_typo_test.gno`](tests/untrust_host_typo_test.gno) revokes a trusted relay with a trailing slash on the path and the relay stays trusted. Withdrawing a spend capability is the one operation here that must not fail quietly. Fix: return `Remove`'s `removed`.
  </details>

## Missing Tests

- **[tests]** [`grc20reg_test.gno:318`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L318) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L318) — the one test added to this file covers the realm wrappers the branch leaves alone, while `UserTransfer`, `UserApprove` and `UserTransferFrom` reach no unit test at all.
  <details><summary>details</summary>

  `TestWrappersProvideRealmTransferApproveAndTransferFrom` asserts the actor binding that [`TestWrappersBindActorToCallingRealm`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L234) and [`TestTransferFromSpendsAllowanceGrantedToCallingRealm`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L288) already hold, on functions this diff does not touch. The three new helpers are covered only by [`grc20reg_user_helpers.txtar`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/pkg/integration/testdata/grc20reg_user_helpers.txtar#L36-L50), which boots a chain, so a change to the actor binding shows up minutes later rather than in the package's own suite. `testing.NewUserRealm` gives `cur.Previous().IsUserCall()` in the unit harness, so both the happy path and the two refusals fit there. Fix: add [`grc20reg_user_helpers_test.gno`](tests/grc20reg_user_helpers_test.gno), which passes at this head.
  </details>

## Suggestions

- **[correctness]** [`tellers.gno:78-97`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L78-L97) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L78) — `UserTellerTrusted` builds the same `fnTeller` as `UserTeller`, so the trusted-host widening its doc reserves for itself admits a `UserTeller` value too.
  <details><summary>details</summary>

  Both constructors set `homeGuard`, set `userOnly` and carry the same `*Token`, and [`guardWrite`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L234-L236) reads `trustedHosts` off the token rather than off the teller, so the two values are indistinguishable to it and to `IsCanonicalTeller`. [`user_teller_identity_test.gno`](tests/user_teller_identity_test.gno) builds one of each against a token that trusts a relay and both pass the relay's frame. The real difference is reachability: `UserTeller` needs the private ledger, `UserTellerTrusted` needs only the published `*Token`. Fix: say that in the doc, or drop `UserTeller` and keep the one constructor.
  </details>

## Verified

- The branch's new wugnot filetest passes against the merge base with the base `examples/` tree, gas 9,145,698 there against 9,158,076 at head. `wugnot.gno` is byte-identical between the two, so the EOA `MsgCall` write surface it exercises predates the branch.
- The `CallerTeller` column of the overview's table: [`grc20_caller_teller_matrix.txtar`](tests/grc20_caller_teller_matrix.txtar) shows a direct `MsgCall` and a `MsgRun` script both debiting the signer, and an intermediate realm debiting itself.
- Green at e014175d2: `gno test ./gno.land/p/demo/tokens/grc20`, `./gno.land/r/demo/defi/grc20reg`, `./gno.land/r/gnoland/wugnot`, and `TestTestdata/{grc20reg_user_helpers,grc20_userteller_msgcall,grc20_token_no_caller_teller,grc20_token_no_trusthost,grc20_token_no_user_teller}`.

## Invariant walk

Every class in `skills/invariant-catalog.md`, against the diff.

| Class | Touched | Result |
| --- | --- | --- |
| Determinism | yes | `trustedHosts` is an `avl.Tree` read by key; no map range, clock, randomness or float on any write path |
| Gas | yes | no new VM op, stdlib call or gas-schedule change; the added field read and branch cost 12,378 gas over the wugnot filetest's four writes |
| Realm state safety | yes | `guardWrite` runs before the ledger write in all three methods, and nothing crosses realms between the check and the mutation |
| Caller & access control | yes | Warnings 1 and 2; the predicate itself is the correct one, `IsUserCall()` rather than `IsUser()` |
| Coin & banker | yes | Warning 2; no coin arithmetic changed |
| Storage deposit | yes | one new tree per token, which is why the genesis balances in `storage_deposit_price_change.txtar` move |
| Global mutable state | yes | Warning 2's `leaked` var; no Go-level global added |
| Error & panic handling | yes | Warning 3, plus the discarded `ok` from `SplitPkgSubPath` in both trust setters |
| VM-fault recoverability | no | no VM change |
| VM semantics vs Go | no | no interpreter change |
| Type-check & preprocess | yes | `gno lint` clean, sanity-checked against a planted undefined symbol |

Realm audit patterns: `callback-param` and `interface-realm-param` both fire on the fixture realm, which is Warning 2. `current-guard` does not: `guardWrite`'s `rlm` is a secondary realm value on a non-crossing helper, and all three write methods check `rlm.IsCurrent()` before calling it. `interface-canonical-assert` does not: the registry helpers take a concrete `*grc20.Token` from `MustGet`. `exported-pointer-leak` and `exported-pointer-field` do not: `trustedHosts` is unexported with no accessor. `realm-only-gate`, `origin-caller-auth`, `payment-user-call`, `render-map-iteration`, `render-markdown` and `getcoins-single-denom` are untouched.

## Not posted

- [`grc20reg_user_helpers.txtar:1-9`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/pkg/integration/testdata/grc20reg_user_helpers.txtar#L1-L9) — the header sentence breaks across a blank comment line at "only for tokens that / trust the registry", and the trailing "because the immediate caller is a realm rather than an EOA" dangles after the two-item list it explains. Comment wording, no behaviour.
- [`grc20reg.gno:108-112`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L108-L112) — "a signing user cannot reach these at all" now sits five lines above three functions a signing user is meant to reach, and the parenthetical pointing users at "the token realm's own entry points" is the route the branch replaces. It still holds for the realm wrappers it is scoped to, and a reader meets it before the new functions. Comment wording, no behaviour.
- [`token.gno:72`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/token.gno#L72) — `trustedHosts: avl.Tree{}` sets the field to its zero value, which the other two `avl.Tree` fields on `PrivateLedger` leave implicit. Cosmetic, no enabled linter covers it.

## Open questions

- `UserTellerTrusted` hangs off `*Token` while `UserTeller` hangs off `*PrivateLedger`, and the file's own SECURITY note calls that placement load-bearing for `CallerTeller`. Nothing exploits the difference, since `guardWrite` rejects an untrusted host either way, so it stays a note rather than a finding.
- A token trusting a `gno.land/e/<address>/run` path would admit that one account's `MsgRun` scripts, since a run script's own frame sees the signer as its previous. Only the token can grant it and only that address benefits, so there is nothing for the author to decide now.
