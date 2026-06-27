# PR #5504: fix(grc721): migrate royalty to basis points per EIP-2981

URL: https://github.com/gnolang/gno/pull/5504
Author: notJoon | Base: master | Files: 4 | +104 -29
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 6d6ab81a5 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5504 6d6ab81a5`

**TL;DR:** The royalty math in the example GRC721 token now uses basis points (a denominator of 10000) instead of whole percents (denominator 100), so a creator can set a 2.5% or 0.5% cut, which whole percents could not express. The change also stops a possible overflow crash and validates the rate is not negative.

**Verdict: APPROVE** — every round-1 finding is fixed in this head: the overflow crash now has a guarded fallback, the discarded-error test bug is corrected, and negative/zero/truncation/overflow cases are tested. Only minor undocumented-behavior nits remain.

## Summary

`p/demo/tokens/grc721` switches its royalty rate from integer percent (denominator 100) to basis points (denominator 10000), matching [EIP-2981](https://eips.ethereum.org/EIPS/eip-2981). The struct field `RoyaltyInfo.Percentage` becomes `RoyaltyInfo.Bps`, `maxRoyaltyPercentage` becomes `maxRoyaltyBps` (now 10000 = 100%), and the rate validation gains a lower bound rejecting negative values. The royalty formula is `salePrice * Bps / 10000`, computed with a checked multiply: on the precise path when `salePrice * Bps` fits in int64, and via a divide-first fallback `(salePrice / 10000) * Bps` when it would overflow, so the call no longer panics on huge sale prices. Blast radius is zero: no realm or package outside this directory references the renamed symbols.

## Examples

| `Bps` | `salePrice` | royalty | meaning |
|------|------------|---------|---------|
| 250 | 1000 | 25 | 2.5% (impossible under old percent model) |
| 50 | 1000 | 5 | 0.5% |
| 1000 | 1000 | 100 | 10% |
| 10000 | 1000 | 1000 | 100% (max) |
| 1000 | 1 | 0 | truncates to 0 (int division) |
| 1000 | MaxInt64 | 922337203685477000 | fallback path; loses 580 vs exact |

## Fix

Round 1 (reviewed at b8329f0) left two warnings: `RoyaltyInfo` panicked through `overflow.Mul64p` on a large `salePrice * Bps` product, and a test discarded the error it then asserted. Both are resolved at this head. The multiply now branches on the `ok` flag from [`overflow.Mul64`](https://github.com/gnolang/gno/blob/6d6ab81a5/examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno#L74-L78) · [↗](../../../../../.worktrees/gno-review-5504/examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno#L74) and falls back to divide-first arithmetic that cannot overflow, and the invalid-token-ID test now captures and asserts its own return value at [`grc721_royalty_test.gno:35`](https://github.com/gnolang/gno/blob/6d6ab81a5/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L35) · [↗](../../../../../.worktrees/gno-review-5504/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L35).

## Critical (must fix)

None

## Warnings (should fix)

None

## Nits

- [`grc721_royalty.gno:77`](https://github.com/gnolang/gno/blob/6d6ab81a5/examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno#L77) · [↗](../../../../../.worktrees/gno-review-5504/examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno#L77) — the fallback path silently returns a less precise royalty than the exact formula, with no comment in the production code. The loss only appears for sale prices above `MaxInt64 / 10000` (≈9.2e15), so no realistic caller hits it, but a one-line comment that the fallback trades precision for range would tell the next reader the divergence is intentional. Confirmed behaviorally: at `salePrice = MaxInt64, Bps = 1000` the fallback returns 922337203685477000 against an exact 922337203685477580, losing 580.

## Missing Tests

None — round 1's four gaps (negative bps, zero bps, truncation, overflow) are all covered now: negative bps at [`grc721_royalty_test.gno:64-68`](https://github.com/gnolang/gno/blob/6d6ab81a5/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L64-L68) · [↗](../../../../../.worktrees/gno-review-5504/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L64), zero and truncation in `TestRoyaltyBpsCompliance` at [`grc721_royalty_test.gno:98-100`](https://github.com/gnolang/gno/blob/6d6ab81a5/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L98-L100) · [↗](../../../../../.worktrees/gno-review-5504/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L98), and the overflow fallback in [`TestRoyaltyInfoOverflow`](https://github.com/gnolang/gno/blob/6d6ab81a5/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L120-L140) · [↗](../../../../../.worktrees/gno-review-5504/examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno#L120).

## Suggestions

None

## Open questions

- `RoyaltyInfo` does not guard a negative `salePrice`; passing `-1000` with `Bps = 1000` returns `-100`. Pre-existing on master, not introduced here, and EIP-2981 is silent on negative prices, so not posted. A caller would have to feed a negative sale price for it to matter.
- The `RoyaltyInfo.Percentage` → `Bps` rename breaks an exported type, which `gno/AGENTS.md` flags as needing discussion for `p/demo/`. Zero consumers exist today and the PR body says the rename was deliberate, so not posted.

## Round note

Round 2, head advanced b8329f0 → 6d6ab81a5 (PR content changed: overflow-guard fallback, the four new test cases, and the discarded-error fix all landed since round 1). Verified on 6d6ab81a5: the Gno royalty output matches a side-by-side Go mirror across the full Examples table including the overflow fallback (922337203685477000 both sides); reverting the `derr =` capture makes the invalid-token-ID assertion compare nil and the test no longer guards anything; bps validation runs before the ownership and token-existence checks, so the over-max and negative-bps cases reach `ErrInvalidRoyaltyBps` regardless of caller. Verdict moves from round 1's APPROVE-with-two-warnings to a clean APPROVE.
