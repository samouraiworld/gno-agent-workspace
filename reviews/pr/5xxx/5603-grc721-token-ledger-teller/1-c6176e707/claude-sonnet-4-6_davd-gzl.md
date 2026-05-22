# PR #5603: refactor(grc721)!: Token/Ledger split with Teller pattern

**URL:** https://github.com/gnolang/gno/pull/5603
**Author:** notJoon | **Base:** master | **Files:** 33 | **+2027 -1349**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

Rewrites `grc721` with three-layer architecture: `*NFT` (read-only), `*PrivateLedger` (mutable state, pointer-as-auth), `Teller` (caller-resolved writes, five factory flavors). Old `basicNFT`/`IGRC721`/`NFTGetter` deleted. Royalty/metadata as independent subpackages. AVL replaced with BPTrees. Four realms migrated. Security: `*NFT` and `*PrivateLedger` never call `PreviousRealm`; `TransferFrom` correctly double-checks approved-or-owner then verifies `owner==from`.

## Test Results
- **Existing tests:** ALL PASS (80 tests across 7 packages)
- **Edge-case tests:** skipped

## Critical (must fix)
- [ ] `ledger.gno:52-82` — `Burn()` does not remove `tokenURIs` entry for burned tokens. `tokenApprovals` correctly removed but `l.tokenURIs` retains dead entries permanently. Royalty and metadata subpackages also lack burn hooks. Permanent on-chain storage leak.

## Warnings (should fix)
- [ ] `tellers.gno:76-97` — `RealmTeller`/`RealmSubTeller` capture `PreviousRealm()` at construction with only a comment warning; wrong address if called outside crossing function.
- [ ] `foo721.gno:13` — `UserTeller` exported: cross-realm callers get calling realm's address, not end user's. Should be unexported.
- [ ] `tokenhub_test.gno:161-194` — `TestMustGetFunctions` silently dead after first panic; NFT and multi-token sub-tests never execute.
- [ ] `interfaces.gno:8-17` — Breaking API changes (signature changes) with no migration guide.

## Nits
- [ ] `tellers.gno:200-206` — Slug not validated for `/` separator.
- [ ] `ledger.gno:195-210` — `"owner:operator"` key separator assumption undocumented.
- [ ] `nft.gno:61` — `origRealm` behavior in test context undocumented.
- [ ] `royalty/royalty.gno:145-148` — Two independent `accountFn` closures without explanation.

## Missing Tests
- [ ] `Burn` then `TokenURI` state (would catch storage leak).
- [ ] Real write test for `RealmTeller`/`RealmSubTeller`.
- [ ] `ImpersonateTeller` with zero/invalid address.
- [ ] Slug-based `RealmSubTeller` round-trip transfer.

## Suggestions
- Add `OnBurn` hook to `PrivateLedger` for extensions, or document host-realm cleanup responsibility.
- Rename exported `UserTeller` to `userTeller` in `foo721`.
- Add `TokenByIndex`/`TokenOfOwnerByIndex` stubs to `IGRC721Enumerable` now.
- Fix stale API in `render.gno` doc strings (missing `cross` arg).

## Questions for Author
- Is `SetTokenURI` owner-only intentional?
- Is `SafeTransferFrom` kept as permanent no-op alias?
- Any on-chain realms using old `basicNFT`/`NFTGetter` API?
- Is `origRealm` immutable by design?

## Verdict
REQUEST CHANGES — `Burn()` storage leak, exported `UserTeller` cross-realm identity trap, and dead test code are blocking. Core architecture and security properties are sound.
