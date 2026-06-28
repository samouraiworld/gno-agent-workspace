# PR #5728: refactor(grc721): token, private ledger split with token teller

URL: https://github.com/gnolang/gno/pull/5728
Author: jinoosss | Base: master | Files: 37 | +2797 -1380
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `eac94f444` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5728 eac94f444`

Round 2. Head advanced `c463023cb` → `eac94f444`; PR content changed (three new commits land jeronimoalbi's review comments). The round-1 Critical and all three Warnings survive verbatim. New since round 1: a `TokenUriUpdate` event on `SetTokenURI`, a `Teller`-interface tidy-up, and a Burn/Transfer write-order fix. A maintainer hold (moul: split the PR, ship the library under `p/onbloc/` v0) is open and unaddressed since the new commits.

**TL;DR:** This PR reorganizes the `grc721` NFT library so token reads, admin writes, and caller-resolved writes each live behind their own type, and adds opt-in metadata and royalty extension packages. The three commits since the last review add an event when a token's URI changes, simplify two interface declarations, and move approval-clearing after the error checks in burn and transfer.

**Verdict: NEEDS DISCUSSION** — maintainer (moul) requested a smaller PR scoped to the event emission and asked that the library land under `p/onbloc/` as version 0, not in `p/demo/`; that direction is unresolved and decides the PR's shape. Independent of that, the round-1 Critical still holds: `metadata.TokenURI` and `royalty.RoyaltyInfo` read storage with no `OwnerOf` gate, so a host that forgets the extension `Burn` leaks stale URI/royalty across a burn-and-remint.

## Summary

Follow-up to #5603. Applies inter-realm v2 (#5669) across the Teller layer, renames `*NFT` → `*Token` for GRC20 parity, freezes `origRealm` at construction, and exposes `IsCanonicalTeller`. Metadata and royalty writes live on `*PrivateLedger` only; extension Tellers are read-only. The host realm owns extension cleanup: after `coreLedger.Burn(tid)` it must call each extension's `Burn(tid)`, and nothing surfaces the bug when it forgets. Three commits since round 1: `SetTokenURI` now emits a `TokenUriUpdate` event ([metadata/token.gno:74-78](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L74-L78) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L74-L78)); the extension `Teller` interfaces now embed `IMetadata`/`IGRC2981` instead of restating their methods; and `Burn`/`Transfer` move `tokenApprovals.Remove` after the `balanceOf` reads so no approval is cleared on a path that then returns an error ([token.gno:217-224](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/token.gno#L217-L224) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L217-L224)). The reorder is correct and removes a partial-write window. All three new commits answer jeronimoalbi's inline comments.

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
- `accountFn` — closure inside `fnTeller` that supplies the principal per write call (nil for read-only).
- `origRealm` — `rlm.PkgPath()` captured at `NewToken` under `rlm.IsCurrent()`; unforgeable, used in `Token.ID()`.
- `MetadataUpdate` / `TokenUriUpdate` — events emitted from the metadata extension's `SetTokenMetadata` / `SetTokenURI`.

## Critical (must fix)

- **[host-forget-extension-burn leaks stale URI/royalty]** [`metadata/token.gno:37-44`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44), [`royalty/token.gno:49-58`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) — `TokenURI` and `RoyaltyInfo` read storage directly with no `OwnerOf` gate, so if the host realm forgets the extension `Burn(tid)` after `coreLedger.Burn(tid)`, stale entries silently leak across the burn, and re-mint inherits them.
  <details><summary>details</summary>

  `TokenMetadata` does gate on `OwnerOf` ([`metadata/token.gno:50-61`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L50-L61) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L50-L61)), so the three reads are inconsistent, but the gate only narrows the window. Verified on eac94f444 with a side-by-side probe (println output below): after burn, `OwnerOf` errors, so the gate would reject — `TokenURI`/`RoyaltyInfo` leak there only because they skip it. After re-mint to a new owner, `OwnerOf` succeeds, so the gate passes and `TokenMetadata` ITSELF serves the previous owner's `alice-meta` to bob. The `OwnerOf` gate therefore closes the burn-window leak but not the re-mint inheritance; only the host's extension `Burn` (or an `OnBurn` hook) clears the stale entry.

  ```
  after-burn   OwnerOf err: true   -> gate rejects (burn window closed by gate)
  after-remint OwnerOf err: false  -> gate passes; TokenMetadata returns name="alice-meta" to bob
  ```

  Confirmed behaviorally on eac94f444: `TestHostForget_TokenURI_LeaksAfterCoreBurn` and `_PersistsAcrossRemint` pass (URI `ipfs://alice-art` leaks to bob after burn+remint), and the royalty equivalents pass (alice's payee keeps collecting on bob's re-minted tid). Repro in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md). For an EIP-2981 marketplace honoring `RoyaltyInfo`, the failure mode is routing royalty to the previous owner's payee on a burned-and-reminted tid.

  The host-burn contract is documented ([metadata/token.gno:82-88](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L82-L88) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L82-L88)) and pinned with `TestLedger_Burn_Clears*` for the happy path where the host does call extension `Burn`, but nothing surfaces the failure when it forgets. None of the four migrated downstream realms (`foo721`, `btree_dao`, `eventix`, `tokenhub`) use the extension subpackages, so the field has no example of the right pattern. The first realm shipping metadata/royalty alongside a `Burn` admin entry point is one missed extension `Burn` away from this bug.

  Fix: two parts. (1) Gate `metadata.TokenURI` and `royalty.RoyaltyInfo` on `OwnerOf(tid)` the way `TokenMetadata` already is, so a forgotten `Burn` can't surface stale data while the tid is unminted. (2) The re-mint inheritance survives the gate on all three reads, so the durable fix is an `OnBurn` callback list on `grc721.PrivateLedger` that extensions register and that fires inside core `Burn`; that removes the host bookkeeping entirely. See Suggestions.
  </details>

## Warnings (should fix)

- **[extension Tellers ship dead `accountFn` and 3 redundant factories]** [`metadata/tellers.gno:9-70`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno#L9-L70) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno#L9-L70), [`royalty/tellers.gno:9-70`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/royalty/tellers.gno#L9-L70) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/tellers.gno#L9-L70) — `Teller` in both extensions exposes only read methods, but each subpackage still ships `CallerTeller`, `RealmTeller`, and `ImpersonateTeller` alongside `ReadonlyTeller`. The `accountFn` field is set in every factory and never read; `grep -rn '.accountFn' metadata/ royalty/` returns nothing.
  <details><summary>details</summary>

  The result is four factory names producing structurally identical Tellers. A reader of [`metadata/types.gno:82-88`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L82-L88) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L82-L88) ("`accountFn` supplies the acting principal for write methods; a nil `accountFn` marks the Teller as read-only") will assume a write surface exists; it does not. `ReadonlyTeller`'s godoc says its "write methods all return `grc721.ErrReadonly`" ([`metadata/tellers.gno:22-23`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno#L22-L23) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno#L22-L23)) but `fnTeller` has no write methods at all. `CallerTeller`'s doc ("Safe to expose publicly") and `ImpersonateTeller`'s ("acting as addr") describe behavior the type cannot exhibit.

  This is leftover from the earlier draft the PR body describes (writes once routed through caller-resolved Tellers). The factory shape was kept "for parity" with core while the write methods were stripped; the supporting field stayed. The new `Teller`-interface embed of `IMetadata`/`IGRC2981` sharpened the read-only surface but left the factory and field redundancy untouched.

  Fix: collapse to a single read-only constructor (rename `ReadonlyTeller` to `NewTeller`, drop the other three and the `accountFn` field), or, if symmetric API space is being reserved for future writes, mark `accountFn` reserved and document that the four factories are aliases today. The test file proves the collapse is honest: [`metadata/tellers_test.gno:28-39`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/tellers_test.gno#L28-L39) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/tellers_test.gno#L28-L39) uses `CallerTeller` only because it needs *a* Teller, not because the call resolves any caller.
  </details>

- **[stale doc comment references removed types]** [`foo721.gno:66-74`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/r/demo/foo721/foo721.gno#L66-L74) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/demo/foo721/foo721.gno#L66-L74) — the comment block above the `Approve`/`SetApprovalForAll`/`TransferFrom` wrappers still describes the pre-#5603 world: "the unexported `foo *basicNFT`", "concrete writer methods", "see `p/demo/tokens/grc721/igrc721.gno` for the Reader/Writer split doctrine".
  <details><summary>details</summary>

  None of these references resolve under the new layout: `basicNFT` was removed in commit `c980a0b7a1`, `igrc721.gno` in `b95b4459e0`, and the Reader/Writer split is replaced by the `Token`/`PrivateLedger`/`Teller` axis. The comment also misdescribes the runtime: the wrappers dispatch through `userTeller.Approve(0, cur, ...)`, which re-resolves `cur.Previous().Address()` per call, so the trust boundary is one level down inside `fnTeller.Approve`, not in these wrappers.

  Fix: replace with a two-line note pointing at the new doctrine ("userTeller is `foo.CallerTeller()`; keep the wrappers as the public surface so cross-realm access resolves the caller as the user, not the calling realm"), or delete it since the doctrine now lives in `types.gno` Teller godoc.
  </details>

- **[tokenhub render example is wrong]** [`render.gno:68`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L68) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L68), [`render.gno:79`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L79) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L79) — the home page render shows `tokenhub.RegisterToken(myToken, "my_token")` and `tokenhub.RegisterMultiToken(myMultiToken.Getter(), "123")` with no `cross(cur)` first argument, but both signatures at [`tokenhub.gno:35,63`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L35-L63) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L35-L63) take `cur realm` as the first parameter.
  <details><summary>details</summary>

  Line 74's `RegisterNFT(cross(cur), myNFT, ...)` is correct, making the inconsistency more confusing: a reader copying the snippet hits a compile error on the `RegisterToken` line, fixes it, and may or may not catch the `RegisterMultiToken` one. The snippet also passes string literals at `render.gno:73` (`myLedger.Mint("g1your_address_here", "1")`) where `Mint(to address, tid TokenID)` expects an address-typed value.

  Fix: rewrite the snippet to compile as-is, mirroring an actual `init(cur realm) { ... }`. The realms in the migration commits (`foo721`, `eventix`) are the canonical references.
  </details>

## Nits

- [`metadata/types.gno:79`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L79) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L79) — the constant `TokenURIUpdateEvent` carries the wire string `"TokenUriUpdate"` (`Uri`), while its symbol and the surrounding `TokenURI`/`SetTokenURI` API spell the acronym `URI`. Every other event in the package matches name to value (`MintEvent`="Mint", `MetadataUpdateEvent`="MetadataUpdate"). An author emitting from the symbol name would guess `"TokenURIUpdate"`; pick one casing.
- [`token.gno:159-164`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/token.gno#L159-L164) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L159-L164) — `RenderHome` is `func (tok Token)` (value receiver) but `Token.ID()` at [token.gno:107](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/token.gno#L107) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L107) is `func (tok *Token)` (pointer). Pick one receiver shape for the file.
- [`types.gno:115-121`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/types.gno#L115-L121) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/types.gno#L115-L121) — core `PrivateLedger` declares its BPTree fields as values (`bptree.BPTree`, relying on the lazy fanout fill), while the extension ledgers declare them as `*bptree.BPTree` and call `bptree.NewBPTree32()` ([metadata/types.gno:70-72](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L70-L72) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L70-L72)). Both work; the split shape is a smell, pick one.
- [`types.gno:120-121`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/types.gno#L120-L121) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/types.gno#L120-L121) — `operatorApprovals` key is still `owner + ":" + op`. Carry-over nit from #5603; addresses currently can't contain `:`, but a struct key or an explicit comment locks the invariant.
- [`token.gno:201-205`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/token.gno#L201-L205) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L201-L205) — `SafeMint` is still a permanent `Mint` alias pending #5546. The comment is honest, but a `Deprecated:` tag or a greppable TODO marker would surface it later.
- [`token.gno:193,236`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/token.gno#L193) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/token.gno#L193) — `Mint`/`Burn` emit `MintEvent`/`BurnEvent` rather than EIP-721's `Transfer` with `from`/`to == address(0)`. Carry-over from #5603; either also emit `Transfer` with the zero address or document the divergence in the package godoc so indexers watching `Transfer` know not to expect mint/burn there.
- [`tellers.gno:208-218`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L208-L218) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L208-L218) — `accountSlugAddr` still accepts a slug containing `/`. Not a security issue, but a `/foo/bar` slug looks like a path; document or reject (carry-over from #5603).
- [`btree_dao.gno:47-56`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/r/demo/btree_dao/btree_dao.gno#L47-L56) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/demo/btree_dao/btree_dao.gno#L47-L56) — `PlantTree(_ realm, ...)` and `PlantSeed(_ realm, ...)` keep `_ realm` as a crossing marker but never use it; the caller comes from `unsafe.OriginCaller()`. Pre-existing pattern, but a `// XXX: use cur.Previous() instead of unsafe.OriginCaller` would catch the next migration.

## Missing Tests

- **[host-forget-extension-burn]** [`metadata/token.gno:37-44`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44), [`royalty/token.gno:49-58`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) — `TestLedger_Burn_Clears*` covers only the happy path (host calls both `Burn`s); no test exercises "host calls core `Burn` but forgets extension `Burn`". Repros carried in [tests/metadata_burn_leak_test.gno](../1-c463023cb/tests/metadata_burn_leak_test.gno) and [tests/royalty_burn_leak_test.gno](../1-c463023cb/tests/royalty_burn_leak_test.gno) (still pass on eac94f444). The `OwnerOf` gate alone fixes only the `_LeaksAfterCoreBurn` cases (flip to `ErrInvalidTokenId`); the `_PersistsAcrossRemint` cases survive the gate and need the `OnBurn` hook before they flip. Ship both as regressions once the fix lands.
- **[new event unverified]** [`metadata/token.gno:74-78`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L74-L78) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L74-L78) — the new `TokenUriUpdate` emission has no test asserting type, key set, or `token`/`tokenId` values; `grep -rn 'TokenUriUpdate' *_test.gno` in the package returns nothing. `MetadataUpdate` is likewise unasserted in unit tests. The `grc721_emit.txtar` integration test exercises only core events through `foo721`, which never attaches the metadata extension. Add a unit test reading emitted events after `SetTokenURI`/`SetTokenMetadata`.
- **[`RealmTeller` real-write smoke on core]** [`tellers_test.gno:199-217`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L199-L217) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L199-L217) — `TestRealmTeller_Smoke` and `TestRealmSubTeller_Smoke` only check `teller != nil`. Carry-over from #5603; the captured-address invariant for `RealmTeller`/`RealmSubTeller` stays unexercised. Add a Mint-then-Transfer-via-teller flow with `testing.SetRealm(testing.NewCodeRealm("..."))` to lock the principal capture.

## Suggestions

- [`metadata/token.gno:37-44`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37-L44), [`royalty/token.gno:49-58`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49-L58) — an `OnBurn` callback hook on `grc721.PrivateLedger` is the durable fix the `OwnerOf` gate cannot deliver: the gate closes only the burn window, while the hook clears the extension entry at burn time so re-mint inheritance can't happen, and it removes the host-bookkeeping burden.
  <details><summary>details</summary>

  Sketch: `func (l *PrivateLedger) RegisterOnBurn(fn func(TokenID))`, invoked at the end of `Burn` before the event emit; extensions register at `NewPrivateLedger` time. Adding a third extension later then can't risk a forgotten `Burn` line. A simpler variant: `func (l *PrivateLedger) BurnAll(tid TokenID, extras ...func(TokenID))` accepting extension burn funcs at the call site.
  </details>

- [`tellers.gno:163-167`](https://github.com/gnolang/gno/blob/eac94f444/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L163-L167) · [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L163-L167) — `fnTeller.Approve` checks `caller != owner && !IsApprovedForAll(owner, caller)`. The condition is correct, but a comment naming the EIP-721 semantics ("only owner or approved-for-all may approve") saves the next reader from re-deriving it.

## Open questions

- moul's hold ties this PR to PR #5726 ("extract non-test-13 example packages to examples/quarantined/", merged 2026-06-01) and a `p/onbloc/` v0 plan. On current master, `examples/gno.land/p/demo/tokens/grc721` is NOT quarantined and `p/onbloc/` has no tokens package, so the intended landing spot for this library is unresolved. Not posted: it restates a maintainer thread the author already owns; the decision is theirs.
- The three new commits each answer a jeronimoalbi inline comment; the corresponding findings here (Teller embed, event emission, write reorder) are positive observations, not new defects, so they carry no posted comment.
