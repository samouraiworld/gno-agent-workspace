# PR #5792: feat(grc721): require token owner for SetTokenMetadata

URL: https://github.com/gnolang/gno/pull/5792
Author: davd-gzl | Base: master | Files: 17 | +533 -89
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: e68b124f2 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5792 e68b124f2`

**Verdict: APPROVE** — adds the missing owner/existence guard to `SetTokenMetadata`, byte-for-byte consistent with `SetTokenURI` and `SetTokenRoyalty`; tests pass; the red CI is Codecov-uploader infra noise, not test failures. Stacked on #5385, so most of the diff is the dependency.

## Summary
`SetTokenMetadata` was the only mutating grc721 method with no authorization: any caller could overwrite any token's metadata, including metadata for tokens that were never minted. The fix takes `caller` as an explicit first arg (like the other setters), resolves the owner via `OwnerOf` (which returns `ErrInvalidTokenId` for unknown tokens), and rejects `caller != owner` with `ErrCallerIsNotOwner`. The owning realm derives `caller` from `rlm.Previous().Address()`, the standard pattern for these `p/` token libraries where the package itself is not crossing-aware.

## Scope note
This PR is stacked on #5385 (the `errors.Is`/`Unwrap`/`Join` stdlib addition). Only three files are this PR's own content:

| File | What |
|------|------|
| `grc721_metadata.gno` | owner + existence guard, new `caller` arg |
| `grc721_metadata_test.gno` | tests the two new error paths |
| `grc721_emit.txtar` | integration realm updated for the new signature |

Everything else (`errors/*`, `uassert.gno`, `grc20/token_test.gno`, `ulist_test.gno`, `events_test.gno`, `csv`/`strconv`/`fmt` test tweaks, `errors_is_filetest.gno`) belongs to #5385 and is reviewed there. Once #5385 merges this diff reduces to the three rows above.

## Fix
Before: [`SetTokenMetadata(tid, metadata)`](https://github.com/gnolang/gno/blob/e68b124f2/examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno#L29) wrote straight into the `extensions` AVL tree and emitted, no checks. After: [`grc721_metadata.gno:29-37`](https://github.com/gnolang/gno/blob/e68b124f2/examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno#L29-L37) · [↗](../../../../../.worktrees/gno-review-5792/examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno#L29-L37) calls `OwnerOf` (propagating its `ErrInvalidTokenId` on a miss) then enforces `caller == owner`. Identical shape to [`SetTokenURI` at `basic_nft.gno:97-110`](https://github.com/gnolang/gno/blob/e68b124f2/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L97-L110) · [↗](../../../../../.worktrees/gno-review-5792/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L97-L110) and [`SetTokenRoyalty` at `grc721_royalty.gno:31`](https://github.com/gnolang/gno/blob/e68b124f2/examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno#L31) · [↗](../../../../../.worktrees/gno-review-5792/examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno#L31). The setter is a concrete method on the unexported `*metadataNFT`, not part of the `IGRC721MetadataOnchain` interface (which exposes only the read-side `TokenMetadata`), so no interface signature changed.

The `grc721_emit.txtar` realm needed two adjustments for the new contract: it now holds a persistent `NewNFTWithMetadata` instance (bound through a local `metadataNFT` interface, since the concrete type is unexported) instead of constructing a throwaway NFT inside `SetMetadata`, and it mints into that instance so the `SetMetadata` call targets a token the caller actually owns. The previous code called the 2-arg form on a never-minted token, which both fails to compile under the new signature and would fail the ownership check.

## Verification
```
gno test ./examples/gno.land/p/demo/tokens/grc721/   → ok (TestSetMetadata, TestSetTokenRoyalty pass)
go test ./gno.land/pkg/integration -run grc721_emit  → PASS (4.8s)
```
Both require the worktree's own gnovm (built from the PR) since the tests now use `errors.Is` from #5385; with `GNOROOT` pointed at a stale tree the type checker reports the old 2-arg signature, which is an environment artifact, not a PR defect.

## CI
The dozen red `<contrib> / test` jobs (`gnobr`, `gnobro`, `gnodev`, `gnofaucet`, …) all fail at the same step: the Codecov uploader's GPG check (`gpg: Can't check signature: No public key` → `Could not verify signature`), an infra failure hitting every PR right now, not these tests. The load-bearing job `gno-checks / test` (which builds the PR's gnovm and runs the examples suite) passes, as do `build`, `e2e-test`, `gno2go`, and all `lint`/`fmt` jobs.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`grc721_metadata.gno:30`](https://github.com/gnolang/gno/blob/e68b124f2/examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno#L30) · [↗](../../../../../.worktrees/gno-review-5792/examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno#L30) — comment says "the caller is its owner" but the check actually rejects approved operators too; `SetTokenURI`/`SetTokenRoyalty` share this owner-only model, so it's consistent, just worth knowing the metadata setter is stricter than ERC-721's typical owner-or-approved.

## Missing Tests
None blocking. The unit test covers the happy path, unknown-token (`ErrInvalidTokenId`), and non-owner (`ErrCallerIsNotOwner`); the txtar exercises the owner happy path end-to-end. A negative txtar case (non-owner key calling `SetMetadata` and asserting the panic) would mirror the unit coverage at the integration layer, but the unit test already pins that path.

## Questions for Author
- Owner-only vs owner-or-approved: intentional that an approved operator cannot set metadata/royalty, only the owner? Matches the existing setters, so likely yes by design, but flagging since ERC-721 metadata extensions usually allow approved operators.

## Suggestions
None.
