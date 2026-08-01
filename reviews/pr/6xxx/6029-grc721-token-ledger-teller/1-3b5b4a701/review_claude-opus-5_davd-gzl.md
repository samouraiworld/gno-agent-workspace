# PR [#6029](https://github.com/gnolang/gno/pull/6029): refactor(grc721): restructure on Token/Ledger/Teller with stackable extensions and a discovery registry

URL: https://github.com/gnolang/gno/pull/6029
Author: jinoosss | Base: master | Files: 46 | +4282 -1700
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 3b5b4a701 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6029 3b5b4a701`

**TL;DR:** Rewrites the NFT library so a collection is split into a public read handle and a private admin handle, adds metadata, royalty and enumerable as detachable add-ons that can be stacked in any combination, and adds a public directory realm where a collection can list itself so other contracts can find it.

**Verdict: REQUEST CHANGES** — the new directory realm renders an attacker-supplied string straight into its shared listing page, and an add-on attached after the first mint silently desynchronises from the ledger (1 Critical, 5 Warnings, 2 Missing tests, 3 Nits, 2 Suggestions).

## Verify first

- [`grc721reg.gno:146`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L146) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L146) — the extension kind reaches `Render` with no escaping and no charset check anywhere on its path. Run the [injection txtar](tests/grc721reg_kind_injection.txtar) and read the rendered listing.
- [`token.gno:114-131`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/token.gno#L114-L131) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token.gno#L114-L131) — confirm nothing rejects a late attach: run [`attach_after_mint_test.gno`](tests/attach_after_mint_test.gno) and compare `Token.TotalSupply` against `Enumerable.TotalSupply`.
- [`grc721reg.gno:32-36`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L32-L36) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L32-L36) — the key is the caller's slug and the prefix check binds only the realm, where [grc20reg binds realm and symbol](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45). Decide whether symbol-keyed lookup is meant to be dropped.

## Summary

`*BasicNFT` carried metadata and royalty by embedding, so a collection could have one or the other, never both, and `Token.ID()` was `realm.symbol` with no sequence id, which grc20reg's keying convention does not accept. This PR splits the core into a `*Token` read view and a `*PrivateLedger` admin surface, adds five `IsCurrent()`-gated `Teller` factories mirroring [grc20's](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L45-L60) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L45-L60), turns metadata and royalty into stackable `Extension` implementations driven by `OnMint`/`OnTransfer`/`OnBurn` hooks, adds an enumerable extension, and adds `r/demo/grc721reg` for discovery. Four quarantined consumers move onto the new API and the `Mint`/`Burn` events become `Transfer` with an empty side, matching EIP-721 and grc20.

The core and the tellers are close to grc20 and hold up. The new surface is where the problems are: the registry accepts an unconstrained `ExtensionView` from any realm and writes the kind it reports into a page shared by every collection, and the hook system has no lifecycle guard, so an extension attached after minting starts publishes an index that never agrees with the ledger.

Reading order: [`types.gno`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/types.gno) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/types.gno), [`token.gno`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/token.gno) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token.gno), [`tellers.gno`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/tellers.gno) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/tellers.gno), the three extension packages, [`collection/`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno), [`grc721reg.gno`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno), then the consumers.

## Diagram

```
issuer realm                                 grc721reg (shared realm)
------------                                 ------------------------
  NewToken(name, symbol, id, cur)
        |                                       registry avl.Tree
        +--> *Token   (read view) --------------------+
        |                                             |
        +--> *PrivateLedger (admin)                   v
                 |                          *collection.Collection
                 | RegisterExtension            token  *Token
                 v                              exts   kind -> ExtensionView
             []Extension                                  ^
             metadata.Ledger                              |
             royalty.Ledger      Register(cur, token, slug, views...)
             enumerable.Ledger   <-- kind string crosses here unchecked
                 ^                                        |
                 | OnMint/OnTransfer/OnBurn                v
             Mint/Burn/transfer                     Render("") lists every
                                                    collection's kinds raw
```

The edge this PR adds and this review flags is the last one: a kind string authored by an arbitrary realm reaches the shared listing.

## Fix

The core writes ledger state first, emits, then fans the hooks out, so a hook cannot observe a half-updated ledger ([`token.gno:148-163`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/token.gno#L148-L163) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token.gno#L148-L163)). Hooks take no `realm` parameter, so an extension cannot capture a crossing frame and re-enter a teller write ([`types.gno:30-36`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/types.gno#L30-L36) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/types.gno#L30-L36)). What the design does not carry is a lifecycle constraint: `RegisterExtension` is the only gate and it checks kind uniqueness only, so ordering is left to a doc comment in each extension package.

## Critical (must fix)

- **[phishing link on a page every collection shares]** `examples/gno.land/r/demo/grc721reg/grc721reg.gno:146` — the extension kind is written into the registry listing with no escaping, and any realm chooses that string.
  <details><summary>details</summary>

  `Register` accepts `grc721.ExtensionView` values from the calling realm ([`grc721reg.gno:22`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L22) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L22)), and `ExtensionView` declares only `ExtensionKind()` and `TokenID()` ([`types.gno:38-41`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/types.gno#L38-L41) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/types.gno#L38-L41)), so an attacker implements it in their own realm and returns whatever kind they like. `attach` stores that string as the tree key with no charset check ([`collection.gno:39-44`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L39-L44) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L39-L44)), and [`extensionBadges`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L146) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L146) concatenates it raw while the collection name and symbol beside it go through `md.EscapeText`. `Render("")` lists every registered collection, so one registration puts an arbitrary heading and link on the page users browse to find collections. `§10` of [`gno-ai-contract-review.md`](https://github.com/gnolang/gno/blob/3b5b4a701/docs/resources/gno-ai-contract-review.md?plain=1#L163) is this exact case. The same string also lands in the `register` event's `extensions` attribute, joined by commas ([`grc721reg.gno:51`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L51) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L51)), so a kind carrying a comma or a newline corrupts the event an indexer parses. Reproduced on chain; see the [repro](comment_claude-opus-5.md) and the [txtar](tests/grc721reg_kind_injection.txtar). The slug on the same path is already charset-checked by [`validateSlug`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L152-L162) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L152-L162). Fix: hold the kind to the same charset the slug already satisfies, at `attach`, so the event is covered too.
  </details>

## Warnings (should fix)

- **[an index that silently stops matching the ledger]** `examples/gno.land/p/demo/tokens/grc721/token.gno:114-131` — an extension attached after the first mint indexes only later tokens, and nothing rejects it.
  <details><summary>details</summary>

  `RegisterExtension` checks the pointer and the kind, never the ledger's state, so it accepts an extension at any point in a token's life. Both extension constructors put the ordering requirement in a doc comment instead ([`enumerable/token.gno:10`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L10) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L10), [`metadata/token.gno:12`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L12) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L12)). Measured at 3b5b4a701: attaching an enumerable after one mint and then minting again leaves the core at two tokens and the enumerable at one, and `TokenByIndex(1)` returns `index out of range`. Transferring a pre-attach token is worse: `OnTransfer` calls `addToOwner` unconditionally ([`enumerable/token.gno:69-72`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L69-L72) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L69-L72)) while the global list only ever grows in `OnMint`, so after the move `TokenOfOwnerByIndex(newOwner, 0)` returns token `1` while the global list holds only `2`. A collection in that state still advertises `enumerable` through the registry. Ready-to-add test: [`attach_after_mint_test.gno`](tests/attach_after_mint_test.gno). Fix: reject a registration once the ledger holds a token.
  </details>

- **[a directory nobody can query by symbol]** `examples/gno.land/r/demo/grc721reg/grc721reg.gno:32-36` — the key is the caller's free-form slug and the prefix check binds only the realm, so the token's own symbol constrains nothing.
  <details><summary>details</summary>

  [grc20reg](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45) builds its key from `token.GetSymbol()` and checks `HasPrefix(token.ID(), key+".")`, which is what lets a caller resolve a token from the realm and symbol it already knows and what stops a realm listing two tokens under one symbol. Here the key is `fqname.Construct(rlmPath, slug)` and the check is `HasPrefix(token.ID(), rlmPath+".")`. Verified at 3b5b4a701: one realm registered the same token under slugs `one` and `two`, and registered a token whose symbol is `REAL` under key `gno.land/r/demo/probe_slug.FAKE`. The overwrite guard therefore guards the slug, not the collection. Fix: state whether symbol-keyed lookup is meant to be dropped, and if it is not, derive the key from the symbol as grc20reg does.
  </details>

- **[an approval the owner cannot take back]** `examples/gno.land/p/demo/tokens/grc721/token.gno:207-210` — `Approve` rejects the empty address, so a per-token approval stands until the token moves.
  <details><summary>details</summary>

  EIP-721 clears a per-token approval by approving the zero address; `Approve` returns `ErrInvalidAddress` for it instead. Verified at 3b5b4a701: after the owner approves an account, `Approve(owner, "", tid)` returns `invalid address`, `GetApproved` still reports the approved account, and that account completes a `TransferFrom`. The owner's only exits are transferring the token or approving some other address, and `SetApprovalForAll` covers a different relation. The behaviour predates the branch, `basic_nft.gno` on the merge base carried the same guard, but this PR rewrites the function and pins the rejection in a test ([`token_test.gno:317-322`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L317-L322) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L317-L322)), which turns a carry-over into a decision. Fix: accept the empty address as a clear, or say in the doc comment that revocation is out of scope.
  </details>

- **[review noise hiding the migration]** `examples/quarantined/gno.land/r/jjoptimist/eventix/eventix.gno:22-24` — the consumer commits delete 110 comment lines that the API change does not touch.
  <details><summary>details</summary>

  The migration off `*BasicNFT` needs the call sites and the variable types. It does not need the doc comments on `PlantTree`, `PlantSeed`, `Render`, `GetUserTokenBalances`, `GetToken`, `MustGetToken`, `RegisterMultiToken` and the rest, all of which go. The eventix counterfeit-token warning is compressed from ten lines naming the attack and the concrete fix to three, and the tokenhub `GetNFT` warning loses the explanation of why an aggregator result is not authority ([`getters.gno:42-43`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/getters.gno#L42-L43) · [↗](../../../../../.worktrees/gno-review-6029/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/getters.gno#L42-L43)). Count from `git diff <merge-base>..3b5b4a701 -- examples/quarantined/ | grep -c '^-\s*//'`. Fix: restore the comments the migration does not require.
  </details>

- **[two extension lists, no check they agree]** `examples/gno.land/p/demo/tokens/grc721/collection/nft.gno:34-36` — `Register` builds its own `Collection` and ignores the one the `NFT` holds.
  <details><summary>details</summary>

  `NFT.Collection()` is documented as the object to register, warning that a later `Attach` mutates it. `Register` never takes a `*Collection`; it takes the token plus views and calls `collection.NewCollection` itself ([`grc721reg.gno:41`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L41) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L41)). The reference consumer therefore writes the list twice, once through `nft.Attach(meta).Attach(roy)` and once as `Register(cross(cur), nft.Token(), "", meta, roy)` ([`foo721.gno:30-37`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/demo/foo721/foo721.gno#L30-L37) · [↗](../../../../../.worktrees/gno-review-6029/examples/quarantined/gno.land/r/demo/foo721/foo721.gno#L30-L37)), and nothing compares them, so a realm that adds an extension to one list and not the other publishes a set that differs from what its own handle reports. `NFT.Collection()` has no non-test caller anywhere in `examples/`. Fix: have `Register` take the `*Collection`, or drop `Attach` and the `Collection()` doc claim.
  </details>

## Missing Tests

- **[the desync ships undetected]** `examples/gno.land/p/demo/tokens/grc721/enumerable/token_test.gno:267` — no case attaches an extension after a mint.
  <details><summary>details</summary>

  `TestDefensiveNoOps` covers `removeFromAll` and `removeFromOwner` on unknown ids and `OnBurn` on a token that was never minted, which is the "extension is ahead of the ledger" direction. The opposite direction, the ledger being ahead of the extension, is the one a realm actually reaches by calling `NewEnumerable` after `Mint`, and it is untested. [`attach_after_mint_test.gno`](tests/attach_after_mint_test.gno) closes it and is red at 3b5b4a701.
  </details>

- **[the registry's headline claim is only tested with its own types]** `examples/gno.land/r/demo/grc721reg/grc721reg_test.gno:28` — no case registers a view the extension packages did not build.
  <details><summary>details</summary>

  Every registry case passes `*metadata.Metadata`, `*royalty.Royalty` or `*enumerable.Enumerable`, all of which report a constant kind. "Open to any extension" is the design statement and the untested half is where the Critical lives. [`grc721reg_kind_injection.txtar`](tests/grc721reg_kind_injection.txtar) registers a foreign implementation and asserts the listing stays clean.
  </details>

## Nits

- **[an assertion that stopped asserting]** `gno.land/pkg/integration/testdata/grc721_emit.txtar:14` — the two metadata assertions, [line 14](https://github.com/gnolang/gno/blob/3b5b4a701/gno.land/pkg/integration/testdata/grc721_emit.txtar#L14) for `TokenURIUpdate` and [line 18](https://github.com/gnolang/gno/blob/3b5b4a701/gno.land/pkg/integration/testdata/grc721_emit.txtar#L18) for `MetadataUpdate`, changed from a closed attribute list to `.*` on both sides of `pkg_path`, so each now matches a `pkg_path` belonging to any event in the array rather than the one carrying its own event type. The token id also became `FNFT[^"]*`, which accepts any suffix; the sequence id is a fixed `0000000` and can be written out.
- **[the reference realm modelling a swallowed error]** `examples/quarantined/gno.land/r/demo/foo721/foo721.gno:33` — `royLedger.SetDefaultRoyalty(admin, 500)` drops its error while every other call in the file panics on one. It cannot fail with these arguments, and the file is what a realm author copies.
- **[an extension nothing exercises]** `examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno:11` — no realm in the tree uses `enumerable`; `foo721` stacks metadata and royalty only. Its swap-and-pop is covered by unit tests but never by a realm, and the desync above is exactly what a first consumer would hit.

## Suggestions

- **[stored metadata that changes with no event]** `examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:76-90` — `SetTokenMetadata` stores the caller's `Data` value, whose `Attributes` slice keeps pointing at the caller's backing array, and `TokenMetadata` hands the same array back.
  <details><summary>details</summary>

  `Data.Attributes` is a `[]Trait` ([`metadata/types.gno:34-44`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L34-L44) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L34-L44)), so the struct copy the tree stores shares its elements with whatever the issuer passed in. Verified at 3b5b4a701 inside the issuing realm: mutating the slice after `SetTokenMetadata` changes what `TokenMetadata` returns, and mutating the returned value's slice changes stored state, both without a `MetadataUpdate` event, which is the signal EIP-4906 consumers follow. A foreign realm reading the published view cannot use this: the on-chain readonly taint rejects the element write, verified in an integration run. Copying the slice on the way in and on the way out closes the in-realm case.
  </details>

- **[the reference entry points skipping the guard the docs show]** `examples/gno.land/r/demo/grc721reg/grc721reg.gno:31` — `Register` reads `cur.Previous().PkgPath()` with no `cur.IsCurrent()` first, and `foo721`'s four admin functions do the same with `cur.Previous().Address()` ([`foo721.gno:111`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/demo/foo721/foo721.gno#L111) · [↗](../../../../../.worktrees/gno-review-6029/examples/quarantined/gno.land/r/demo/foo721/foo721.gno#L111)).
  <details><summary>details</summary>

  The preprocessor rejects any realm value other than `cur` or `cross(rlm)` at a crossing call, so no path is demonstrated that reaches these reads with a forgeable value, and grc20reg omits the guard too. It is still the pattern `§1` of [`gno-ai-contract-review.md`](https://github.com/gnolang/gno/blob/3b5b4a701/docs/resources/gno-ai-contract-review.md?plain=1#L12) shows, and this PR's own `/p/` code applies it at every entry point that takes a realm ([`token.gno:15-17`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/token.gno#L15-L17) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token.gno#L15-L17), [`tellers.gno:39-41`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L39-L41) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L39-L41)), so the two new realm-side entry points read as an omission. Not posted, no change needed if the split is deliberate.
  </details>

## Verified

- A foreign realm registers a collection with its own `ExtensionView` and the injected heading and link appear in `vm/qrender` output for `gno.land/r/demo/grc721reg:`. Run on chain through `gno.land/pkg/integration`, not reachable from any unit test.
- The published `*metadata.Metadata` read view returns an aliased `Attributes` slice, and a second realm writing to it through `grc721reg.GetExtension` is rejected on chain with `cannot directly modify readonly tainted object`. A `testing.SetRealm` unit probe accepts the same write, so the unit harness alone would have reported a leak that does not exist.
- Attaching an enumerable after one mint leaves the core at two tokens and the extension at one; transferring a pre-attach token then yields a `TokenOfOwnerByIndex` hit for a token the global list does not contain.
- One realm registers the same token under two slugs, and a token whose symbol is `REAL` under a key ending `.FAKE`.
- An owner cannot clear a per-token approval: `Approve(owner, "", tid)` returns `invalid address` and the approved account still completes a `TransferFrom`.
- Green at 3b5b4a701: `gno test` over `p/demo/tokens/grc721/...`, `r/demo/grc721reg`, and the four migrated consumers; `gno lint` clean on all six new and changed packages, confirmed to typecheck by feeding it an undefined symbol.

## Existing threads

None. No review comments and no reviews on the PR; the only comment is the `Gno2D2` checks summary. `Merge Requirements` is red because no review team member has approved yet, not for a code reason.

## Open questions

- `Extension` is compared by `==` in the dedup loop ([`token.gno:121`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/token.gno#L121) · [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token.gno#L121)), so an extension implemented on a value type holding a slice would fault on registration. Only the issuer can reach it and all three shipped extensions are pointer types, so there is no trigger; not posted.
- `btree_dao` still authenticates through `unsafe.OriginCaller()` in a function this PR edits. The auth line is unchanged by the diff and predates it, so it is out of scope here, but it is the `origin-caller-auth` shape and worth its own pass.
- `Collection.Extension` tells callers to type-assert to the concrete type without saying the stored view may be a foreign implementation, so a caller using the one-value form aborts on a spoofed entry. Deferred: the Critical's fix does not change it and the concrete extension types cannot be forged, so royalty and metadata data itself stays trustworthy.
