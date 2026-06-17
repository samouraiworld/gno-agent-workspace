# PR #5728: refactor(grc721): token, private ledger split with token teller

URL: https://github.com/gnolang/gno/pull/5728
Author: jinoosss | Base: master | Files: 37 | +2789 -1380
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `c463023cb` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5728 c463023cb`

**Verdict: REQUEST CHANGES** — host-forget-burn leaks stale `TokenURI`/`RoyaltyInfo` (same shape as the prior PR #5603 critical, now downgraded to a documented host contract — but `TokenMetadata` is gated while `TokenURI` and `RoyaltyInfo` are not, so the contract fails silently rather than uniformly); extension Tellers ship dead `accountFn` + three functionally-identical factory variants; `foo721.gno` carries stale `basicNFT`/`igrc721.gno` doc comments; `tokenhub/render.gno` example snippet is wrong (missing `cross(cur)`). Inter-realm v2 application, `IsCanonicalTeller` defense, and the host-burn-contract regression tests are all sound — the core split is good, the rough edges are at the seams.

## Summary

Follow-up to #5603. Reorganizes `gno.land/p/demo/tokens/grc721` to apply inter-realm v2 (#5669) throughout the Teller layer: every write takes `(_ int, rlm realm, ...)` and gates on `rlm.IsCurrent()`, returning `ErrSpoofedRealm` outside the live crossing frame. Renames `*NFT` → `*Token` for GRC20 parity, captures `rlm.PkgPath()` as the unforgeable `origRealm` at construction, and exposes `IsCanonicalTeller` (nominal-type guard against embedding-based wrappers — see [`tellers_test.gno:30-40`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L30-L40) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L30-L40)). Drops `tokenURI` from core into the metadata subpackage; collapses the `metadata`/`royalty` Teller surfaces to **read-only** (writes live on `*PrivateLedger` only) so a current token holder can no longer rewrite the URI/royalty after acquisition. Adds extension-level `Burn(tid)` and pins the host-realm-burn contract with `TestLedger_Burn_Clears*` / `_Idempotent` regressions.

```
host realm (admin) ──┐                               extensions:
                     │                                  metadata.PrivateLedger.tokenURIs / entries
                     ▼                                  royalty.PrivateLedger.royalties
              grc721.PrivateLedger                       │
              ├─ Mint/Burn/Transfer                      │  cleanup is HOST'S responsibility:
              ├─ Approve/SetApprovalForAll               │  coreLedger.Burn(tid)
              ├─ ImpersonateTeller(addr)                 │  mLedger.Burn(tid)  ◄── if forgotten,
              └─ exposes Token (read view)               │  rLedger.Burn(tid)  ◄── stale data leaks
                            │                            │
                            ▼                            │
                       Teller (caller-resolved write)    │
                       ├─ CallerTeller   rlm.Previous()  │
                       ├─ RealmTeller    rlm.Address()   │
                       ├─ RealmSubTeller +slug           │
                       └─ ReadonlyTeller                 │
```

## Glossary

- `*Token` — public read view (name, symbol, OwnerOf, balances, approvals). Was `*NFT` pre-#5603.
- `*PrivateLedger` — admin write surface; possessing the pointer IS the authorization.
- `Teller` — caller-resolved write API on the core token; resolves principal via one of five factories.
- `fnTeller` — canonical concrete Teller. `IsCanonicalTeller` checks `*fnTeller` nominally to defeat embedding wrappers.
- `accountFn` — closure inside `fnTeller` that supplies the principal per write call (nil for `ReadonlyTeller`).
- `origRealm` — `rlm.PkgPath()` captured at `NewToken` under `rlm.IsCurrent()` — unforgeable, used in `Token.ID()`.
- `MetadataUpdate` — EIP-4906 event now emitted from `(*PrivateLedger).SetTokenMetadata` in the metadata extension.

## Fix

Three layers move at once. (1) Core: `NewToken` panics on `!rlm.IsCurrent()` and empty `rlm.PkgPath()`, freezing `origRealm` ([token.gno:25-53](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/token.gno#L25-L53) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L25-L53)); name/symbol now validate against length + charset caps with explicit errors. (2) Teller surface: every write method on `*fnTeller` re-checks `rlm.IsCurrent()` and returns `ErrSpoofedRealm` instead of panicking ([tellers.gno:130-182](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L130-L182) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L130-L182)); `ImpersonateTeller` panics on `!addr.IsValid()` ([tellers.gno:102-117](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L102-L117) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L102-L117)) — closes the empty-sentinel hole I flagged on #5603. (3) Extensions: writes (`SetTokenURI`, `SetTokenMetadata`, `SetTokenRoyalty`) move off the Teller onto `*PrivateLedger` only, and each subpackage now exposes `Burn(tid)` so the host can clean up stale per-tid state after core burn ([metadata/token.gno:83-87](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L83-L87) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L83-L87), [royalty/token.gno:67-69](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L67-L69) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L67-L69)).

## Critical (must fix)

- **[host-forget-extension-burn leaks stale URI/royalty]** [`metadata/token.gno:37-44`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44), [`royalty/token.gno:49-58`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) — `TokenURI` and `RoyaltyInfo` read storage directly with no `OwnerOf` gate, so if the host realm forgets the extension `Burn(tid)` after `coreLedger.Burn(tid)`, stale entries silently leak across the burn (and re-mint inherits them).
  <details><summary>details</summary>

  Same shape as the #5603 critical, now downgraded to a documented contract — the contract works for `TokenMetadata` (which does gate on `OwnerOf`, see [`metadata/token.gno:50-61`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L50-L61) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L50-L61)), but the two other reads are inconsistent. The user-visible failure mode for an EIP-2981 marketplace honoring `RoyaltyInfo` is routing royalty payments to the previous owner's payee on a burned-and-reminted tid — verified live by [tests/metadata_burn_leak_test.gno](tests/metadata_burn_leak_test.gno) (TokenURI leaks `ipfs://alice-art` to bob after burn+remint) and [tests/royalty_burn_leak_test.gno](tests/royalty_burn_leak_test.gno) (alice's payee keeps collecting on bob's re-minted tid). Both pass on `c463023cb`.

  The host-burn-contract is documented ([metadata/token.gno:76-82](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L76-L82) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L76-L82)) and pinned with `TestLedger_Burn_Clears*` regressions when the host does call extension Burn — but nothing surfaces the failure when the host forgets. None of the four migrated downstream realms (`foo721`, `btree_dao`, `eventix`, `tokenhub`) use the extension subpackages, so the field has no example of the right pattern; the first realm that does ship metadata/royalty alongside a `Burn` admin entry point is one missed extension Burn away from this bug.

  Fix: cheapest — gate `metadata.TokenURI` and `royalty.RoyaltyInfo` on `OwnerOf(tid)` the same way `TokenMetadata` already is. One `if _, err := led.token.OwnerOf(tid); err != nil { return ..., err }` per function. The contract becomes "even if the host forgets, no stale data surfaces" instead of "host MUST remember or surfaces silently". Heavier — add an `OnBurn` callback list on `grc721.PrivateLedger` and register the extension Burns; this also removes the host-realm bookkeeping. Pick one.
  </details>

## Warnings (should fix)

- **[extension Tellers ship dead `accountFn` and 3 redundant factories]** [`metadata/tellers.gno:9-70`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno#L9-L70) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno#L9-L70), [`royalty/tellers.gno:9-70`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/royalty/tellers.gno#L9-L70) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/tellers.gno#L9-L70) — `Teller` in both extensions exposes only read methods (`TokenURI`/`TokenMetadata`, or `RoyaltyInfo`), but the subpackages still ship `CallerTeller`, `RealmTeller`, and `ImpersonateTeller` alongside `ReadonlyTeller`. The `accountFn` field is set at construction in each factory and never read again — `grep -n "ft.accountFn" metadata/ royalty/` returns nothing.
  <details><summary>details</summary>

  The result is four factory names that produce structurally identical Tellers. A reader of [`metadata/types.gno:80-86`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L80-L86) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L80-L86) ("`accountFn` supplies the acting principal for write methods; a nil `accountFn` marks the Teller as read-only") will assume the write surface exists somewhere; it does not. `CallerTeller`'s doc says "Safe to expose publicly" — true, but meaningless since the only methods callable on it are reads that need no caller resolution. `ImpersonateTeller`'s doc says "acting as addr" — but no method on the returned Teller dispatches against `addr`.

  This is leftover from the earlier draft the PR body describes ("an earlier draft routed metadata/royalty writes through caller-resolved Tellers with an owner check"). The author kept the factory shape "for parity" with core but stripped the write methods; the supporting field stayed.

  Fix: pick one — (a) collapse to a single read-only constructor (e.g. rename `ReadonlyTeller` to `Teller`/`NewTeller`, drop the other three and the `accountFn` field); (b) if the intent is to keep symmetric API surface for future writes, add a `// reserved for forthcoming write methods; currently no-op` comment on `accountFn` and document that the four factories are aliases today. (a) is the honest choice — the test file proves it: [`metadata/tellers_test.gno:28-39`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/tellers_test.gno#L28-L39) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/tellers_test.gno#L28-L39) uses `CallerTeller` only because it needs *a* Teller, not because the call resolves any caller.
  </details>

- **[stale doc comment references removed types]** [`foo721.gno:66-74`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/r/demo/foo721/foo721.gno#L66-L74) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/demo/foo721/foo721.gno#L66-L74) — comment block above the `Approve`/`SetApprovalForAll`/`TransferFrom` wrappers still describes the pre-#5603 world: "the unexported `foo *basicNFT`", "concrete writer methods", "see `p/demo/tokens/grc721/igrc721.gno` for the Reader/Writer split doctrine".
  <details><summary>details</summary>

  None of these references resolve under the new layout: `basicNFT` was removed in commit `c980a0b7a1`, `igrc721.gno` was removed in `b95b4459e0`, and the `Reader/Writer split` doctrine has been replaced by the `Token`/`PrivateLedger`/`Teller` axis. The comment also misdescribes the runtime — the wrappers now dispatch through `userTeller.Approve(0, cur, ...)` which itself re-resolves `cur.Previous().Address()` per call, not "the owning realm's trusted boundary derives `caller` from `cur.Previous().Address()`" as the comment claims (the boundary is one level down, inside `fnTeller.Approve`).

  Fix: replace with a 2-line note pointing at the new doctrine: "userTeller is `foo.CallerTeller()`; cross-realm direct access would resolve the caller as the calling realm, not the original user. Keep the wrappers as the public surface so the assumption holds." Or just delete it — the doctrine is now in `types.gno` Teller godoc.
  </details>

- **[tokenhub render example is wrong]** [`render.gno:67,79`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L67-L79) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L67-L79) — the home page render shows `tokenhub.RegisterToken(myToken, "my_token")` and `tokenhub.RegisterMultiToken(myMultiToken.Getter(), "123")` without `cross(cur),` as the first argument, but the actual signatures at [`tokenhub.gno:35,63`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L35-L63) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L35-L63) require it.
  <details><summary>details</summary>

  Line 74's `RegisterNFT(cross(cur), myNFT, ...)` is correct, making the inconsistency more confusing — a reader copying the snippet will hit a compile error on the `RegisterToken` line, fix that, and may or may not catch the `RegisterMultiToken` one. The example also omits the `_, ledger :=` two-return shape on line 67 for `grc20.NewToken` (which returns `(*Token, *PrivateLedger)` in the new API). Plus `myLedger.Mint("g1your_address_here", "1")` on line 73 passes string literals where `Mint(address, TokenID)` expects typed values.

  Fix: rewrite the snippet to compile as-is, mirroring what `init(cur realm) { ... }` actually looks like. The realms in the migration commits (`foo721`, `eventix`) are the canonical references.
  </details>

## Nits

- [`token.gno:159-164`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/token.gno#L159-L164) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L159-L164) — `RenderHome` is `func (tok Token) RenderHome() string` (value receiver) but reads `tok.ledger.owners.Size()`; works because `ledger` is a pointer, but inconsistent with `Token.ID()` which uses a pointer receiver. Pick one for the whole file.
- [`types.gno:115-121`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/types.gno#L115-L121) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/types.gno#L115-L121) — core `PrivateLedger` declares BPTree fields as values (`bptree.BPTree`, zero-init relies on the lazy fanout=32 fill in `tree.gno:118`), while the extension ledgers declare them as `*bptree.BPTree` and call `bptree.NewBPTree32()` ([metadata/types.gno:71-73](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L71-L73) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L71-L73)). Both work, but the inconsistency is a smell; pick one shape.
- [`types.gno:120`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/types.gno#L120) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/types.gno#L120) — `operatorApprovals` key is still `owner + ":" + op`. Carry-over nit from #5603; addresses currently can't contain `:`, but a struct key or an explicit comment locks the invariant.
- [`token.gno:201-205`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/token.gno#L201-L205) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L201-L205) — `SafeMint` is still a permanent `Mint` alias pending #5546. The `// SafeMint currently aliases Mint` comment is honest but a `Deprecated:` tag or a `_ = "TODO(#5546)"` import would make it greppable.
- [`token.gno:166-198`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/token.gno#L166-L198) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L166-L198) — `Mint`/`Burn` still emit `MintEvent`/`BurnEvent` rather than EIP-721's `Transfer` event with `from`/`to == address(0)`. Carry-over from #5603; either also emit `Transfer` with the zero address or document the divergence in the package godoc so wallets indexing `Transfer` know not to expect mint/burn there.
- [`tellers.gno:209-218`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L209-L218) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L209-L218) — `accountSlugAddr` still accepts a slug with `/` in it. Not a security issue but a `/foo/bar` slug looks like a path — document or reject (carry-over from #5603).
- [`btree_dao.gno:47-56`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/r/demo/btree_dao/btree_dao.gno#L47-L56) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/demo/btree_dao/btree_dao.gno#L47-L56) — `PlantTree(_ realm, ...)` and `PlantSeed(_ realm, ...)` keep `_ realm` as a marker that they're crossing functions but never use it; the actual caller comes from `unsafe.OriginCaller()`. Pre-existing pattern, but worth a `// XXX: use cur.Previous() instead of unsafe.OriginCaller` so the next migration catches it.

## Missing Tests

- **[host-forget-extension-burn]** [`metadata/token.gno:37-44`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44), [`royalty/token.gno:49-58`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) — `TestLedger_Burn_Clears*` covers the happy path (host calls both Burns); no test exercises "host calls core Burn but forgets extension Burn". Repros in [tests/metadata_burn_leak_test.gno](tests/metadata_burn_leak_test.gno) and [tests/royalty_burn_leak_test.gno](tests/royalty_burn_leak_test.gno). Once the OwnerOf gate (or OnBurn hook) lands, flip these to assert `ErrInvalidTokenId` and ship them as regressions.
- **[`RealmTeller` real-write smoke on core]** [`tellers_test.gno:199-217`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L199-L217) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L199-L217) — `TestRealmTeller_Smoke` and `TestRealmSubTeller_Smoke` only check `teller != nil`. Carry-over from #5603 — the captured-address invariant for `RealmTeller` / `RealmSubTeller` remains unexercised. Add a Mint-then-Transfer-via-teller flow with `testing.SetRealm(testing.NewCodeRealm("..."))` to lock the principal capture.

## Suggestions

- [`metadata/token.gno:37-44`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44), [`royalty/token.gno:49-58`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) — beyond the OwnerOf-gate quick fix, the `OnBurn` callback hook on `grc721.PrivateLedger` is the structurally honest fix; it removes the host-bookkeeping burden and makes the extension lifecycle visible at the type level.
  <details><summary>details</summary>

  Sketch: `func (l *PrivateLedger) RegisterOnBurn(fn func(TokenID))`, invoked at the end of `Burn` before the event emit. Extensions register at `NewPrivateLedger` time. The host then doesn't need to remember anything — adding a third extension later doesn't risk a forgotten Burn line. A simpler variant: `func (l *PrivateLedger) BurnAll(tid TokenID, extras ...func(TokenID))` accepting extension burn funcs at call sites.
  </details>

- [`tellers.gno:165-167`](https://github.com/gnolang/gno/blob/c463023cb/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L163-L167) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L163-L167) — `fnTeller.Approve` checks `caller != owner && !IsApprovedForAll(owner, caller)`. The condition is correct but a comment naming the EIP-721 semantics ("only owner or approved-for-all can approve") would save the next reader from re-deriving it.

## Questions for Author

- The metadata/royalty extension Tellers are read-only by design (per PR body) — was the choice to keep `CallerTeller`/`RealmTeller`/`ImpersonateTeller` instead of a single `ReadonlyTeller`/`NewTeller` intentional, or is this leftover from the earlier draft? If intentional, what's the planned write surface they reserve API space for?
- The host-burn contract for extensions is documented but not enforced. Was an `OnBurn` hook considered and rejected (and if so, for what reason — gas? simplicity?), or is this an explicit follow-up?
- Was emitting `Transfer` events with the zero address for Mint/Burn (EIP-721 standard) considered? Wallets indexing by `Transfer` will miss them today.
