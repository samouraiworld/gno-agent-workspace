# gnolang/gno [#6072](https://github.com/gnolang/gno/pull/6072): feat(grc721): rewrite core on the Token/PrivateLedger/Teller axis (1/4)

URL: https://github.com/gnolang/gno/pull/6072
Author: jinoosss | Base: master | Files: 27 | +1886 -1606
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: f151f475d (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6072 f151f475d`
Overview: [overview](../overview.html)
Open the code: [github.dev](https://github.dev/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno) · [vscode.dev](https://vscode.dev/github/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno) · `./scripts/review-worktrees.sh gno 6072`

## Overview

The NFT library under `p/demo/tokens/grc721` is rewritten from the `BasicNFT` monolith onto the same three-type shape `grc20` already uses: a `Token` read view, a `PrivateLedger` write authority the issuer keeps, and a `Teller` capability that writes as some account. Five teller grades cover the cases from "acts as whoever crossed in" to an issuer-only impersonation grade. The old API took a `caller address` argument on every write; the new one derives identity from the crossing frame. Metadata and royalty leave the core package and return in the later three PRs of the series. The four in-tree consumers, `btree_dao`, `eventix`, `foo721` and `tokenhub`, move onto the new API in this same PR so `examples/` never breaks.

**Verdict: COMMENT** — the rewrite maps cleanly onto grc20's shape and the package suite covers the five teller grades and the new revoke path, but `SetApprovalForAll(operator, false)` stores a permanent AVL node for a relationship that does not exist, where this PR's own new `Approve` revoke and `setBalance` both remove such a node (1 Warning, 1 Missing test, 2 Nits).

## Verify first

- [`token.gno:271`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L271) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L271) — the `false` branch writes a node. Run [`grc721_approvalall_storage.txtar`](tests/grc721_approvalall_storage.txtar) and read `bytes_delta` on a `SetApprovalForAll(op, false)` for an operator never approved: 1084 bytes, charged to the issuing realm.

## Summary

`Token` carries the getters, `PrivateLedger` carries the mutators and `RegisterExtension`, and `NewToken` returns the ledger once so only the issuing realm holds write authority ([`token.gno:22-57`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L22-L57) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L22)). Each writing teller re-checks `rlm.IsCurrent()` and rejects a stale frame with `ErrSpoofedRealm`. The package suite is thorough: the five teller grades, the `Approve(zero)` revoke this PR adds, `ErrReadonly` on every readonly write, `ImpersonateTeller` on an invalid address, and the `RegisterExtension` kind-collision branch all have cases. The one state-safety gap is `SetApprovalForAll`, whose `false` path stores instead of removing.

## Warnings (should fix)

- **[a no-op that leaves permanent state]** `examples/gno.land/p/demo/tokens/grc721/token.gno:271` — `SetApprovalForAll(operator, false)` for an operator that was never approved writes an AVL node rather than removing one, so operator entries accumulate unbounded and never release.
  <details><summary>details</summary>

  `SetApprovalForAll` calls [`operatorApprovals.Set(operatorKey(owner, operator), approved)`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L271) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L271) unconditionally, so a `false` value is stored as a live node. This PR's own `Approve` revoke path [removes the node](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L245-L249) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L245) when the target is the zero address, and [`setBalance`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L341-L349) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L341) removes a zero balance, so the store-nothing-on-clear convention is established in the same file and this path misses it. Measured at f151f475d: one `SetApprovalForAll(op, false)` for an operator never approved writes 1084 bytes charged to the issuing realm, and `isApprovedForAll` reads a missing key and a `false` node identically, so the stored node changes no read and the realm has no method that removes it. The call also emits an `ApprovalForAll ... approved=false` event as though a relationship changed. The bloat predates the branch, `basic_nft.gno` stored the same way, but the branch rewrites this function and adds the remove-on-clear pattern beside it. Reproduced on chain; see the [txtar](tests/grc721_approvalall_storage.txtar) and the [regression test](tests/sgd_false_store_test.gno). Fix: on `false`, `Remove` the key instead of setting it, mirroring the revoke path above.
  </details>

## Missing Tests

- **[the false-write bloat ships undetected]** `examples/gno.land/p/demo/tokens/grc721/token_test.gno:401` — the "revoke operator approval" case sets `true` then `false` and asserts `IsApprovedForAll` is false, which a stored `false` node satisfies as readily as a removed one, so no case reaches the never-approved `false` write.
  <details><summary>details</summary>

  `IsApprovedForAll` returns false whether the node is absent or present with value false, so an assertion on it cannot see the difference the Warning turns on. [`sgd_false_store_test.gno`](tests/sgd_false_store_test.gno) asserts `operatorApprovals.Size()` is 0 after a lone `SetApprovalForAll(op, false)`, and is red at f151f475d, green once the `false` path removes.
  </details>

## Nits

- **[assertions that stopped pinning the id]** `gno.land/pkg/integration/testdata/grc721_emit.txtar:9` — every `stdout` regex ends `.*\]` and writes the token id as `FNFT[^"]*`, so each matches only the first event in the array and accepts any id suffix; the id is deterministic and could be pinned. Not posted, test-only. The seqid is a fixed `0000000`, so the value is `gno.land/r/foo721.FNFT.0000000`, confirmed against the emitted event.
- **[a swallowed mint error after payment]** `examples/quarantined/gno.land/r/jjoptimist/eventix/eventix.gno:120` — `BuyTicket` takes payment, then calls `ticketLedger.Mint(buyer, tokenId)` and drops its error, so a mint failure leaves the buyer paid with no ticket while `ticketsSold` still increments. The error is unreachable under the current id scheme, where `event_<id>_ticket_<n>` is unique per sale, and the pattern predates the branch, which only renames the receiver. Not posted.

## Verified

- The comment at [`types.gno:91`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/types.gno#L91) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/types.gno#L91), "Symbol charset matches grc20reg slug", holds: `validSymbol` accepts `[A-Za-z0-9_-]` ([`token.gno:73-85`](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L73-L85) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L73)) and `grc20reg`'s `validateSlug` accepts the same set ([`grc20reg.gno:159-172`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L159-L172)).
- `go run ../gnovm/cmd/gno test ./gno.land/p/demo/tokens/grc721/...` green at f151f475d.
- The `SetApprovalForAll` fix (`Remove` on false) keeps the whole grc721 package suite green, run in a scratch copy and reverted.

## Open questions

- `NewToken` takes a caller-supplied `seqid.ID`, so one realm can build two tokens with the same `Token.ID()`; the `NewToken` event is the documented mitigation and PR 6028 moves id issuance into the registry. Left to that seam, not posted.
