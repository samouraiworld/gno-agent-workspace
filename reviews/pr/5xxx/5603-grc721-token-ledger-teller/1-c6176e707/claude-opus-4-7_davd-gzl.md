# PR #5603: refactor(grc721)!: Token/Ledger split with Teller pattern

URL: https://github.com/gnolang/gno/pull/5603
Author: notJoon | Base: master | Files: 33 | +2027 -1349
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `c6176e707` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5603 c6176e707`

**Verdict: REQUEST CHANGES** — `Burn` leaks stale `tokenURIs` and royalty entries so reads on burned tokens silently return previous values (correctness, not just storage). Otherwise the Token/Ledger/Teller split is sound, all existing tests pass, security boundaries hold.

## Summary

Rewrites `gno.land/p/demo/tokens/grc721` along the GRC20 axis: `*NFT` (read-only identity), `*PrivateLedger` (admin writes, pointer-as-auth), and a `Teller` layer that resolves the acting principal via one of five factory flavors (`CallerTeller`, `ReadonlyTeller`, `RealmTeller`, `RealmSubTeller`, `ImpersonateTeller`). Royalty and on-chain metadata move to opt-in subpackages with their own ledgers. AVL is replaced by BPTrees throughout. Four downstream realms (`foo721`, `btree_dao`, `eventix`, `tokenhub`) are migrated. The big architectural property is clean: `*NFT` and `*PrivateLedger` never call `runtime.PreviousRealm`/`CurrentRealm`, so realm-context confusion is confined to the Teller layer.

```
caller realm (or impersonation) ──┐
                                  ▼
                              Teller.accountFn() ──── owner/approval gate
                                  │
                                  ▼
                              PrivateLedger.Transfer/Approve/...
                                  │
                                  ▼
                              NFT (read-only view)
```

## Glossary

- `*NFT` — public read view of a collection (name, symbol, OwnerOf, TokenURI, balances).
- `*PrivateLedger` — admin write surface; possessing the pointer IS the authorization. Methods take explicit `from`/`owner` args.
- `Teller` — caller-resolved write API. `fnTeller.accountFn()` supplies the principal.
- `CallerTeller` — `accountFn` returns `runtime.PreviousRealm().Address()` per call.
- `RealmTeller` / `RealmSubTeller` — captures the calling realm address at construction; the latter derives a sub-account via `chain.PackageAddress(addr+"/"+slug)`.
- `ImpersonateTeller` — `accountFn` returns a fixed address; reserved for admin/migration paths gated by possession of the `*PrivateLedger`.

## Fix

Three pieces: (1) split the monolithic `basicNFT` into the read-only `*NFT` ([nft.gno:17-39](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/nft.gno#L17-L39) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/nft.gno#L17-L39)) and a hidden `*PrivateLedger` ([ledger.gno:14-211](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L14-L211) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L14-L211)), so admin powers are gated by capability (pointer) instead of by inspecting realm context inside every method. (2) Push caller resolution to a separate `fnTeller` layer ([tellers.gno:42-107](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L42-L107) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L42-L107)); read methods come from the embedded `*NFT`, write methods source their principal from `accountFn`. (3) Extract royalty + on-chain metadata into separate packages with the same shape ([royalty/royalty.gno:48-92](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/royalty/royalty.gno#L48-L92) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/royalty/royalty.gno#L48-L92), [metadata/metadata.gno:53-84](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/metadata/metadata.gno#L53-L84) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/metadata/metadata.gno#L53-L84)), so the keep/remove decision for on-chain metadata (issue #3502) can land independently.

## Critical (must fix)

- **[burned tokens still report their old URI]** [`ledger.gno:52-82`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) — `Burn` clears `owners`, `balances`, `tokenApprovals` but never touches `tokenURIs`, and `NFT.TokenURI` has no OwnerOf guard, so reads on a burned token silently return the previous URI.
  <details><summary>details</summary>

  Already flagged in the prior sonnet-4-6 review on the storage angle, but the user-visible failure mode is correctness, not just bloat. Two consequences confirmed by [tests/burn_leak_test.gno](tests/burn_leak_test.gno):

  - After `Burn(tid)`, `OwnerOf(tid)` correctly errors with `ErrInvalidTokenId`, but `TokenURI(tid)` returns the dead URI with `nil` error (see [`nft.gno:97-103`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/nft.gno#L97-L103) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/nft.gno#L97-L103)).
  - If the same `TokenID` is later re-minted to a different owner, the new token inherits the previous owner's URI unless `SetTokenURI` is called again.

  Compare with [`metadata/metadata.gno:98-107`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/metadata/metadata.gno#L98-L107) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/metadata/metadata.gno#L98-L107), which guards `TokenMetadata` reads with an `OwnerOf` check — the inconsistency between the two extensions is itself a smell.

  Fix: in `Burn`, add `l.tokenURIs.Remove(tidStr)` next to the `tokenApprovals.Remove`. Also gate `NFT.TokenURI` on `exists(tid)` so a stale entry (e.g. from a future bug or migration) can't surface.
  </details>

- **[burned tokens still pay royalties]** [`royalty/royalty.gno:119-127`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/royalty/royalty.gno#L119-L127) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/royalty/royalty.gno#L119-L127) — same shape as the URI leak: `RoyaltyInfo` reads `l.royalties` directly with no OwnerOf gate, and `Burn` never clears the per-token entry, so a marketplace that queries royalties after burn will route funds to the previous owner's payee.
  <details><summary>details</summary>

  Verified in [tests/royalty_burn_test.gno](tests/royalty_burn_test.gno): after burning a token whose royalty is set to 5%, `nft.RoyaltyInfo(tid, salePrice)` returns the original payee and amount with no error. On re-mint of the same `TokenID` to a different account, the new owner inherits the old royalty config.

  The `Burn` cleanup gap is structural — the core ledger exposes no `OnBurn` hook, so the royalty subpackage cannot synchronously clear its own state when a token is burned through `coreLedger.Burn(tid)`.

  Fix: pick one — (a) add an `OnBurn` callback hook on `*PrivateLedger` that subpackages register against, or (b) gate `RoyaltyInfo` (and `TokenMetadata`, defensively) on `OwnerOf(tid)` so any state attached to a non-existent token surfaces as `ErrInvalidTokenId`. (b) is the minimal one-line fix; (a) is more honest about the lifecycle. The same fix applies to `metadata.PrivateLedger.entries` on burn even though `TokenMetadata` already gates reads — the entry leaks invisibly until re-mint flips it back into the data path.
  </details>

## Warnings (should fix)

- **[exporting `UserTeller` is a naming trap]** [`foo721.gno:13`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/r/demo/foo721/foo721.gno#L13) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/r/demo/foo721/foo721.gno#L13) — the variable is exported as `UserTeller` but is a `CallerTeller`: it resolves the acting principal to whoever called into foo721, not "the user".
  <details><summary>details</summary>

  In normal usage (user → blockchain tx → foo721) this is fine. But the exported name suggests "this Teller represents users," whereas a cross-realm caller can directly grab `foo721.UserTeller.Approve(...)` and the teller will resolve the principal as the calling realm, not as the original user behind it. The PR comment on `CallerTeller` ([`tellers.gno:49-52`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L49-L52) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L49-L52)) says "safe to expose publicly" — and structurally it is — but the choice to capitalize the variable in the foo721 host invites callers to import and use it directly under a misleading mental model.

  Fix: lowercase the variable to `userTeller`, keep the wrappers (`Approve`/`SetApprovalForAll`/`TransferFrom`) as the public surface. Same call sites still work; cross-realm direct access is removed.
  </details>

- **[`RealmTeller` / `RealmSubTeller` capture realm at construction]** [`tellers.gno:71-97`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L71-L97) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L71-L97) — both capture `runtime.PreviousRealm().Address()` at construction, with the constraint left to a `WARN:` doc comment.
  <details><summary>details</summary>

  If a realm builds one of these outside a crossing function (e.g. in an `init` whose previous-realm context is not what the author expects), the captured principal is wrong for the lifetime of the teller. There is no test that exercises real writes through `RealmTeller`/`RealmSubTeller` ([`tellers_test.gno:255-269`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L255-L269) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L255-L269) only checks `teller != nil`), so a mis-construction would not show up in tests.

  Fix: add a smoke test that constructs the teller from a `NewCodeRealm`, then performs a write and asserts the captured principal matches expectation. Even one passing test is enough to anchor the invariant; the comment alone is at risk of decay.
  </details>

- **[no `OnBurn` hook for extensions]** [`ledger.gno:52-82`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) — extensions (`royalty`, `metadata`) hold per-token state that the core ledger can invalidate via `Burn` without notification.
  <details><summary>details</summary>

  Today this manifests as the two Critical findings above. Even if those are fixed defensively (gate reads on OwnerOf), the state itself is dead weight, and any future extension repeats the problem. A simple option is `(l *PrivateLedger) RegisterBurnHook(func(TokenID))` invoked at the end of `Burn`, before the event emit. Extensions register at construction; cleanup becomes a one-liner per extension.

  Fix: introduce the hook in this PR or open a follow-up issue. Either way, document that extensions MUST cope with the absence of the hook (i.e. read-gate on `OwnerOf`) until it lands.
  </details>

- **[`IGRC721Enumerable` promises more than `TotalSupply`]** [`interfaces.gno:42-44`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/interfaces.gno#L42-L44) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/interfaces.gno#L42-L44) — the interface name lifts from EIP-721's enumerable extension, but only `TotalSupply()` is on the surface.
  <details><summary>details</summary>

  EIP-721 enumerable also requires `TokenByIndex(uint256)` and `TokenOfOwnerByIndex(address, uint256)`. The comment on the interface says they're "not implemented yet" but exposing the name suggests parity. A downstream consumer typing to `IGRC721Enumerable` and expecting full enumeration support has no compile-time clue they'll be missing two methods.

  Fix: either add the stubs (return `ErrNotImplemented` until the BPTree iterator path is wired up), or rename to `IGRC721Supply` / drop the interface and just have `TotalSupply` on `*NFT`. The name matters because realms will type-cast against it.
  </details>

## Nits

- [`util.gno:3-8`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/util.gno#L3-L8) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/util.gno#L3-L8) — `isValidAddress` could just be `addr.IsValid()` inlined; the function adds nothing beyond the error mapping. Personal taste; leave if uniform across grc20.
- [`ledger.gno:196`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L196) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L196) — `operatorApprovals` key is `owner + ":" + op`. If addresses ever contain `:` (they won't, but a future address format change could), collisions are silent. Worth a `// addresses have no ':'` justification, or use a struct key.
- [`tellers.gno:200-206`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L200-L206) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L200-L206) — `accountSlugAddr` accepts any slug including `"/"`. Not a security issue (hash-based), but a `/foo/bar` slug looks like a path and may confuse callers expecting hierarchy. Document or reject.
- [`tellers.gno:138-140`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L138-L140) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L138-L140) — `SafeTransferFrom` alias is a permanent no-op. If the receiver-check follow-up (issue #5546) stalls, the alias becomes a confusing footgun. Consider gating with a `Deprecated:` doc tag (the ledger's `SafeMint` already follows that convention).
- [`tellers.gno:102-107`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L102-L107) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L102-L107) — `ImpersonateTeller` accepts any address with no validity check. Writes will fail downstream in the ledger's `isValidAddress`, so this only ever produces a silent-broken teller. Reject at construction with a panic, like `NewNFT` does for empty name/symbol.
- [`ledger.gno:217-223`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L217-L223) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L217-L223) — no event emitted for `SetTokenURI`. EIP-4906 specifies a `MetadataUpdate` event; emit one or document the omission.
- [`ledger.gno:14-40`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L14-L40) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L14-L40) — `Mint`/`Burn` emit dedicated `MintEvent`/`BurnEvent` rather than the EIP-721-standard `Transfer` event with `from`/`to == address(0)`. Two consumers indexing by `Transfer` won't see these. Either also emit `Transfer` with zero address, or document the divergence in package docs.
- [`tokenhub_test.gno:151-194`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/r/matijamarjanovic/tokenhub/tokenhub_test.gno#L151-L194) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/r/matijamarjanovic/tokenhub/tokenhub_test.gno#L151-L194) — `TestMustGetFunctions` has three `defer recover()` blocks followed by panic-on-not-found calls; after the first panic the rest of the test body is unreachable. Pre-existing (not introduced by this PR), but the migration touches the file — worth cleaning up while it's on the editor.

## Missing Tests

- **[burn cleanup invariants]** [`ledger.gno:52-82`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) — no test asserts that `TokenURI` errors after burn, that `tokenURIs` is removed, or that a re-minted tid gets a clean URI. Repro covered by [tests/burn_leak_test.gno](tests/burn_leak_test.gno).
- **[royalty burn cleanup]** [`royalty/royalty.gno:103-127`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/royalty/royalty.gno#L103-L127) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/royalty/royalty.gno#L103-L127) — same gap for `RoyaltyInfo`; once fixed, lock with a regression test. Repro in [tests/royalty_burn_test.gno](tests/royalty_burn_test.gno).
- **[real-write smoke for `RealmTeller`/`RealmSubTeller`]** [`tellers_test.gno:255-269`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L255-L269) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/tellers_test.gno#L255-L269) — current smoke only asserts non-nil. Add a Mint-then-Transfer-via-teller flow with `testing.SetRealm(NewCodeRealm("..."))` to lock the captured-address invariant.

## Suggestions

- [`ledger.gno:52-82`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/ledger.gno#L52-L82) — add `OnBurn` hook (see Warning above). It would let `royalty`/`metadata` clean up without the OwnerOf-gate workaround.
- [`interfaces.gno:42-44`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/p/demo/tokens/grc721/interfaces.gno#L42-L44) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/p/demo/tokens/grc721/interfaces.gno#L42-L44) — rename to `IGRC721Supply` or stub the missing enumerable methods.
- [`render.gno:67-71`](https://github.com/gnolang/gno/blob/c6176e707/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L67-L71) · [↗](../../../../../.worktrees/gno-review-5603/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L67-L71) — the example code in the rendered home page uses the new `grc721.NewNFT(...)` two-return signature but the `RegisterNFT` example still passes the NFT (good); however the snippet omits `cross` in the `RegisterToken` / `RegisterNFT` calls compared to the actual signature `func RegisterNFT(cur realm, ...)`. Update or remove the snippet.

## Questions for Author

- Is `OnBurn` cleanup desired in this PR, or split to a follow-up? If split, what's the migration story for tokens minted under the leaking version?
- The migration cheat-sheet in the PR body is excellent. Will it be mirrored into the package godoc, or stay PR-only?
- Was the choice not to emit `Transfer` for Mint/Burn intentional (gas? clarity?) or a carry-over from the old implementation?
