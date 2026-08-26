# gnolang/gno [#6072](https://github.com/gnolang/gno/pull/6072): feat(grc721): rewrite core on the Token/PrivateLedger/Teller axis (1/4)

URL: https://github.com/gnolang/gno/pull/6072
Author: jinoosss | Base: master | Files: 27 | +1886 -1606
Reviewed by: davd-gzl | Model: claude-opus-5 (xhigh, deep) | Commit: f151f475d (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6072 f151f475d`
Overview: [overview](../overview.html)

Round 2, deep, over the same commit round 1 reviewed. Round 1 was a single pass whose lens agents all died on a session limit, so its verdict rested on one Warning. This round ran the lost-test-coverage comparison, the EIP-721 conformance walk and the invariant-catalog walk that round 1 never did. It confirms round 1's COMMENT verdict and adds one Warning: `RegisterExtension` screens duplicates with an interface comparison that is a runtime error for an uncomparable extension type. Round 1's `SetApprovalForAll` Warning is [posted](https://github.com/gnolang/gno/pull/6072#pullrequestreview-5026751967) and carries unchanged.

## Overview

The NFT library under `p/demo/tokens/grc721` is rewritten from the `BasicNFT` monolith onto the same three-type shape `grc20` uses: a `Token` read view, a `PrivateLedger` write authority the issuer keeps, and a `Teller` capability that writes as some account. The old API took a `caller address` argument on every write; the new one derives identity from the crossing frame. Metadata and royalty leave the core package and return in the later PRs of the series. The four in-tree consumers move onto the new API in the same PR.

**Verdict: COMMENT** — confirms round 1. The rewrite maps cleanly onto grc20's shape and the deleted tests lose only two core assertions, but `RegisterExtension` aborts with a VM error rather than its documented panic for a legal extension type, and `SetApprovalForAll(op, false)` still stores a node the file's own neighbours remove (2 Warnings, 4 Nits, 3 Suggestions).

## Verify first

- [`token.gno:139`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L139) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L139) — drop [`extension_uncomparable_test.gno`](tests/extension_uncomparable_test.gno) into the package and run it: the issuer gets `comparing uncomparable type` where the doc promises a kind collision.

## Warnings (should fix)

- **[a duplicate screen that aborts instead of reporting]** `examples/gno.land/p/demo/tokens/grc721/token.gno:139` — `RegisterExtension` compares two `Extension` interface values with `==`, which is a VM runtime error when the dynamic type is uncomparable, so a second instance of a slice-holding extension type aborts the transaction instead of panicking with the documented kind collision.
  <details><summary>details</summary>

  The loop screens duplicates with [`if e == ext`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L139) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L139) before it reaches the [kind check](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L143-L145) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L143) one line below. Comparing interface values compares the dynamic values once the dynamic types match, so an extension written on a value receiver whose type holds a slice or a map survives the first `RegisterExtension` and blows up on the second. Measured at f151f475d: the issuer sees `runtime error: comparing uncomparable type gno.land/p/demo/tokens/grc721.journalExtension` instead of `grc721: extension kind already registered: journal`. The doc above it, "Same pointer twice is a no-op", also overstates what the line does: the comparison is on interface equality, not pointer identity, so two distinct but equal comparable extensions dedupe silently too. The set is extensions implemented on a value receiver over an uncomparable type: no in-tree extension is one, and the series' own `Enumerable` and its `Ledger` are pointer types, so nothing in PRs 2 through 4 triggers it. The kind check alone already rejects every duplicate kind, and the fault is recoverable, so a gno `recover()` catches it. Ready-to-add test: [`extension_uncomparable_test.gno`](tests/extension_uncomparable_test.gno), red at f151f475d. Fix: compare kinds only and drop the identity fast-path, or say in the doc that an extension type must be comparable.
- **[a no-op that leaves permanent state]** `examples/gno.land/p/demo/tokens/grc721/token.gno:271` — carried from round 1 and posted: `SetApprovalForAll(operator, false)` for an operator never approved writes an AVL node rather than removing one, measured at 1084 bytes charged to the issuing realm. Re-verified this round on a worktree confirmed pristine.

## Nits

- **[a provenance claim the type does not support]** `examples/gno.land/p/demo/tokens/grc721/token.gno:16-17` — the doc says "Because Token's fields are unexported, NewToken is the only way a Token can come into existence", and `&grc721.Token{}` compiles from any package. Verified from a foreign realm: the zero value constructs, and every ledger-backed read on it nil-derefs, recoverably. The conclusion the comment draws still holds, since `PrivateLedger.token` is unexported so no externally built `Token` can ever emit, but the stated reason is wrong. Not posted, comment wording.
- **[a nil guard that misses its neighbour]** `examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno:42` — `RegisterNFT` rejects a nil `*grc721.Token` and not a zero-valued one, so `RegisterNFT(cross(cur), &grc721.Token{}, ...)` nil-derefs where the old `*BasicNFT` path returned `ErrInvalidTokenId`. The caller panics its own transaction only.
- **[assertions that stopped pinning the id]** `gno.land/pkg/integration/testdata/grc721_emit.txtar:9` — the five `stdout` regexes write the token id as `FNFT[^"]*`, which accepts any suffix including none, so `Token.ID()` dropping the seqid segment this PR exists to add would pass unnoticed end to end. The value is deterministic, `gno.land/r/foo721.FNFT.0000000`. The exact identity is still pinned by `filetests/newtoken_event_filetest.gno`, so the loss is on the end-to-end path only.
- **[an error with no producer]** `examples/gno.land/p/demo/tokens/grc721/types.gno:79` — `ErrCallerIsNotOwner` is exported and unreachable: its only call sites left with the metadata and royalty files.

## Suggestions

- **[one function should own every balance write]** `examples/gno.land/p/demo/tokens/grc721/token.gno:166` — `Mint` writes through `balances.Set` while `Burn` and `transfer` go through `setBalance`, whose zero-removal is what keeps `KnownAccounts` honest.
- **[an unreachable error branch]** `examples/gno.land/p/demo/tokens/grc721/token.gno:293-296` — `transfer`'s `ownerOf` error cannot fire: its only caller gates on `isApprovedOrOwner`, which is false for a missing id.
- **[silent behaviour changes the description does not carry]** `examples/gno.land/p/demo/tokens/grc721/token.gno:101` — `BalanceOf` dropped its `(int64, error)` signature and reads a malformed address as 0; the `Mint` and `Burn` event types are replaced by `Transfer` with an empty endpoint, which is the EIP-721 shape and also the change most likely to break an indexer. Neither is in the PR body's breaking-change section.

## Verified

- The deleted tests lose exactly two core assertions with no replacement: a `MaxSymbolLen`-length symbol being accepted, and `BalanceOf` on an invalid address. Everything else in `basic_nft_test.gno` maps onto a new case, and the metadata and royalty tests leave with their implementations.
- EIP-721 walk: the `Approve` guard order, `ErrApprovalToCurrentOwner`, self-transfer rejection and `GetApproved` on a valid-but-unapproved token all diverge from the spec, and every one predates this branch in `basic_nft.gno`. No doc comment claims conformance on any of them.
- `&grc721.Token{}` nil-derefs recoverably, so the VM-fault recoverability class in the invariant catalog holds.
- The series' own extensions are pointer types, so the `RegisterExtension` fault is not reachable from PRs 6073 through 6075.
