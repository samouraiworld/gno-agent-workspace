# PR #5645: fix(examples): `namereg` realm fixes

**URL:** https://github.com/gnolang/gno/pull/5645
**Author:** jeronimoalbi | **Base:** master | **Files:** 10 | **+245 -28**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Bundle of small fixes to `examples/gno.land/r/sys/namereg/v1`. Each addresses a sharp edge surfaced after the realm landed in #5600.

- `admin.gno`
  - `setPaused` now emits `NameRegistrationPaused` / `NameRegistrationUnpaused`. `setRegisterPrice` emits `NameRegistrationPriceChanged` with the new price.
  - `updateUsername` switches from `userData.UpdateName(newName)` to `userData.UpdateNameIgnoreCanonical(newName)` — GovDAO renames now bypass canonical-collision detection.
  - `NewSetPausedExecutor` panics at proposal-creation time if the requested pause value already matches the current state, and proposals are routed through `dao.NewProposalRequestWithFilter` with `govimpl.FilterByTier{Tier: memberstore.T1}` (mirrors the names realm).
  - `ProposeNewName` adds a preflight `if data, _ := susers.ResolveName(newName); data != nil { panic(ErrNameTaken) }`, then formats the title with the user's address (and a Markdown link to the user page) instead of the current username.
  - `ProposeDeleteUser` rewrites the title similarly and prepends a description warning that the address won't be able to register a new name after deletion.
- `errors.gno` — corrects stale error prefixes from `r/gnoland/users:` to `r/sys/namereg:`.
- `preregister.gno` — bootstrap loop now checks the error from `RegisterUserIgnoreCanonical` and panics on anything other than `ErrNotWhitelisted` (the only error expected outside genesis, since unit tests run at block height 123). Replaces the previous "intentionally discarded" comment.
- `users.gno` — `Register` adds a multi-coin guard `if registerPrice > 0 && len(banker.OriginSend()) != 1 { panic(errInvalidPayment()) }` to reject envelopes that carry coins beyond `ugnot`.
- `users_test.gno` / new filetests
  - `TestRegister_MultipleCoins` covers the new multi-coin guard.
  - `z_4_prop1_filetest.gno` — GovDAO renames via `UpdateNameIgnoreCanonical` overwrite a canonical entry already owned by another user.
  - `z_5_prop1_filetest.gno` — preflight `ResolveName` check rejects a proposal when the target name is currently held by another user.
  - `z_6_prop4_filetest.gno` — proposing the current pause value panics at creation time.
  - Existing `z_0` / `z_1` filetests updated to match the new proposal-title shape.

## Test Results

- **Existing tests:** SKIPPED locally — my installed `gno` binary uses stale stdlibs (`chain/banker` missing `IsCanonical`) and the only workable path was to `go install` from this worktree; that aborted on a pre-existing stash conflict in `gnovm/pkg/gnolang/op_binary.go` left by another worktree, which I couldn't resolve without explicit authorization. **CI status:** all checks passing (https://github.com/gnolang/gno/pull/5645#issuecomment chain). Codecov reports all modified lines covered.
- **Edge-case tests:** skipped (described in the Missing Tests section so the author can pick them up).

## Critical (must fix)

- None.

## Warnings (should fix)

- [ ] `examples/gno.land/r/sys/namereg/v1/users.gno:70` — the multi-coin guard is conditional on `registerPrice > 0`, so when registration is free (the default `registerPrice = int64(0)` at `users.gno:19`) a caller who attaches a non-`ugnot` coin still loses it. `banker.OriginSend().AmountOf("ugnot")` returns `0` and matches `registerPrice == 0`, the multi-coin check is skipped, and any `ufoo`-style coin in the envelope lands at the realm address with no recovery path. This is the same loss-of-funds scenario the commit message ("autocomplete used to generate the gnokey command") motivates the fix for — the fix just doesn't cover the default-price configuration. Either drop the `registerPrice > 0` clause, or rewrite the predicate as "reject any coin in OriginSend whose denom isn't `ugnot`, regardless of price".
- [ ] `examples/gno.land/r/sys/namereg/v1/admin.gno:91` — the preflight `susers.ResolveName(newName) != nil` filters soft-deleted entries (see `r/sys/users/users.gno:13-15`), but the execution-time check inside `updateName` at `r/sys/users/store.gno:187` uses `nameStore.Has(newName)`, which returns `true` for deleted entries (entries are never removed from `nameStore`). A `ProposeNewName(addr, X)` where `X` is the name of a previously-deleted user will pass the preflight, be voted on, then `panic(ErrNameTaken)` at execution. Wasted governance cycle. Either expose an `IsNameTaken(name)` helper from `r/sys/users` that mirrors the executor's exact check, or replicate the `nameStore.Has` semantics here. Same gap applies if you ever decide to also preflight `ProposeDeleteUser`'s downstream constraints.
- [ ] `examples/gno.land/r/sys/namereg/v1/admin.gno:71-81` — the doc comment on `ProposeNewName` only mentions the `nym-...\d{3}` format bypass, but with the switch to `UpdateNameIgnoreCanonical` (line 31) governance now also bypasses canonical-collision detection. `z_4_prop1_filetest.gno` exercises the new semantic: bob holds `vital1k`, GovDAO renames alice to `vitalik`, and after execution `IsCanonicalTaken("v1tal1k")` resolves to `vitalik` (alice's name) — i.e. a confusable-namespace flip that mutates bob's canonical entry. This is consistent with decision #14 ("later-wins on bypass") in `r/sys/users/store.gno:191-205`, but a reader of the doc-comment alone would not know it. Extend the comment to call out that canonical collisions are intentionally suppressed and that this is the only privileged path that can shift canonical ownership between addresses.

## Nits

- [ ] `examples/gno.land/r/sys/namereg/v1/admin.gno:105, 141` — proposal titles now embed `[g1...](/r/sys/namereg/v1:g1...)`. The gov/dao render path escapes brackets/parentheses inside titles (visible in `filetests/z_0_prop1_filetest.gno:73` etc. as `\[g1...\]\(/r/sys/namereg/v1:g1...\)`), so the Markdown link renders as literal text — not a clickable link. The longer escaped form just bloats the title compared to plain `g1...`. Two options: (a) drop the link from the title and keep the bare address — the description has the room for a clickable link if you want one; (b) check whether `r/gov/dao` can be taught to relax the title escaping (riskier, more invasive). The `Author:` line in the same renders is unescaped, so this is title-specific.
- [ ] `examples/gno.land/r/sys/namereg/v1/admin.gno:52-54` — panic message is `"paused value is already " + strconv.FormatBool(newPausedValue)`. The names realm uses the same idempotency guard with the message `"paused state already matches requested value; no-op proposal rejected"` (`examples/gno.land/r/sys/names/verifier.gno:168`). Consistency between sibling sys realms is worth more than a few characters; align on the longer message (it actually tells a reader what the constraint is). Also consider exporting a sentinel `var ErrPausedNoOp = errors.New(...)` and `panic(ErrPausedNoOp)` so callers can match it with `errors.Is`.
- [ ] `examples/gno.land/r/sys/namereg/v1/admin.gno:22-26, 41-44` — the pause events emit no actor field, but the equivalent in names emits `"by", runtime.PreviousRealm().PkgPath()` (`r/sys/names/verifier.gno:197-199`). Same for `NameRegistrationPriceChanged`. Adding a `by` attribute makes off-chain audit trails uniform across sys realms and costs nothing.
- [ ] `examples/gno.land/r/sys/namereg/v1/preregister.gno:42-46` — typos in the comment: `undelying` → `underlying`, `effectibly` → `effectively`. Also "the list of curated names is not updated accordingly" reads as a copy-paste from a scenario that doesn't quite fit; the actual contract is "the validation tightens such that a curated name no longer validates."
- [ ] `examples/gno.land/r/sys/namereg/v1/filetests/z_4_prop1_filetest.gno:1` — the file header says "Test updating a name via GovDAO when there's a canonical collision", but the test actually verifies that the canonical collision is *intentionally bypassed* (the rename succeeds and overwrites bob's canonical entry). Rename to something like "Test GovDAO rename bypasses canonical-collision detection (decision #14, later-wins)" so a future reader doesn't read the assertion as expected-to-fail.

## Missing Tests

- [ ] `examples/gno.land/r/sys/namereg/v1/users_test.gno` — add a `TestRegister_MultipleCoins_FreeRegistration` case: with `registerPrice = 0`, send `chain.NewCoins(chain.NewCoin("ufoo", 1))` and assert `Register` either panics (recommended outcome) or — under current code — silently accepts. The current `TestRegister_MultipleCoins` only exercises the `registerPrice = 42` branch.
- [ ] `examples/gno.land/r/sys/namereg/v1/filetests/` — add a `z_*` test for the preflight gap: register a user, delete them (or use the bypass to put a stale entry into `nameStore`), then call `ProposeNewName(otherAddr, deletedName)` and document the resulting `ErrNameTaken` at *execution* time rather than at proposal-creation time. Captures the discrepancy until it's fixed.
- [ ] `examples/gno.land/r/sys/namereg/v1/admin_test.gno` (new) — a unit test for `NewSetPausedExecutor` idempotency mirroring `examples/gno.land/r/sys/names/verifier_test.gno:TestProposeSetPaused_idempotency`. Right now this is only exercised by the new `z_6` filetest, which is more expensive to run than a direct unit test on `paused`.
- [ ] No test asserts that `setRegisterPrice` and `setPaused` actually emit the new events. Worth a small test that exercises the executor path end-to-end and inspects the emitted events.

## Suggestions

- The `users.gno:65-72` block would be cleaner as a single predicate that explicitly enumerates intent: `coins := banker.OriginSend(); if len(coins) != 1 || coins[0].Denom != "ugnot" || coins[0].Amount != registerPrice { panic(errInvalidPayment()) }`. That collapses the two checks into one, removes the `registerPrice > 0` exception, and reads as a single payment-shape assertion. (The current shape leaves the door open to the warning above.)
- `admin.gno:104-109` and `admin.gno:140-144` duplicate the address-link snippet (`[%s](/r/sys/namereg/v1:%s)` with `userData.Addr()` twice). Tiny helper `func userPageLink(u *susers.UserData) string` would keep the two titles in sync and make the eventual fix for the escaping nit a one-liner.
- Per `gno/AGENTS.md` ("Every non-trivial AI-assisted PR must include an ADR"), the `UpdateName` → `UpdateNameIgnoreCanonical` switch is non-trivial and worth a short ADR in `gno.land/adr/` recording the trust posture (DAO is sole path to canonical reshuffling) and the precedent it sets for other controllers that may add `Propose*Name` flows in the future. Empty PR body makes that gap especially visible.

## Questions for Author

- Was the `registerPrice > 0` clause in `users.gno:70` deliberate (e.g. to keep `TestRegister_Free` passing as-written), or an oversight? If deliberate, what's the threat model that makes "single foreign coin attached at free registration" acceptable but "single foreign coin attached at paid registration" not?
- The preflight gap (W2) — was this considered and accepted, or just not surfaced? Soft-deleted names that re-enter the namespace via `ProposeNewName` are at least theoretical recovery flows that this gap silently breaks.
- For the title link rendering (Nit 1): did you confirm in a local gnoweb session that the link is clickable in the final UI? The escaped filetest output suggests it isn't, but I'd rather hear it from the source than guess from the txt fixtures.

## Verdict

NEEDS DISCUSSION — none of the issues are blocking, but the silent foreign-coin loss at the default `registerPrice = 0` (W1) and the preflight/execution skew (W2) are worth resolving before merge; the canonical-bypass expansion (W3) deserves a docstring + ADR even if the implementation stays as-is.
