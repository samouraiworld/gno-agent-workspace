# PR [#6131](https://github.com/gnolang/gno/pull/6131): chore: update GovDAO T1 multisig address

URL: https://github.com/gnolang/gno/pull/6131
Author: moul | Base: master | Files: 253 | +333 -333
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: `f2bdb07b0` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6131 f2bdb07b0`
Overview: [overview](../overview.md)

Round 2, deep mode over the commit round 1 reviewed. The head did not move. Four
lenses ran, then three critics and a claim gate. The round overturns round 1's
REQUEST CHANGES: round 1's `test13` Warning argues against a decision the
description states outright, on a chain that is no longer re-derived, and most of
what the lenses surfaced is already in the description. What survives are two pre-existing
coverage gaps it does not name, neither of which blocks the change, so the verdict
is APPROVE.

## Overview

The GovDAO T1 multisig changed one signer, so the address derived from its signer
set changed with it. Five gno.land realms hold that address as a source constant,
and several hundred test fixtures and genesis files hold copies that have to agree
with those constants. This change rewrites 393 of the 407 copies in the tree and
leaves 14 under `misc/deployments/gnoland1/` and `misc/deployments/test13.gno.land/`,
which record chains that already ran.

**Verdict: APPROVE** — the substitution is complete and provably pure, and what it
leaves are two pre-existing coverage gaps on constants it touches
(1 Suggestion, 1 Nit).

## Verify first

- [`examples/gno.land/r/gnoland/blog/admin.gno:20`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L20) — revert this constant alone and run `gno test -C examples ./gno.land/r/gnoland/blog`. It stays green, so the realm's own suite is not what would catch a wrong admin here.

## Summary

Every added line equals its removed line once the two addresses normalise to one
token, apart from three comment lines in the `pearl`, `sapphire` and `topaz`
builders that drop the word `gnoland1` from a phrase naming the multisig. The new
address decodes through the repository's own bech32 reader to twenty bytes, is
rejected on a one-character corruption, and appears nowhere in the tree at the
merge base. So nothing unrelated rode along and the value is right.

The description already carries the rest of what a reader needs: the one-shot gate
and the param that should replace it, both left-alone deployment trees with a
reason per file, the balance that has to move while the old signers exist, and the
locked checksums going stale on all four builders. What it does not carry is that three of
the constants it edits are pinned by no test that would fail on a wrong value.

## Benchmarks / Numbers

Which constants a wrong value would break, measured by reverting each to the old
address, running, and restoring.

| Constant | Own realm suite | Some other suite |
| --- | --- | --- |
| [`verifier.gno:59`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) `admin` | no | yes, four txtar files |
| [`boards.gno:47`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) `gPerms` | yes | yes |
| [`blog/admin.gno:20`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) `adminAddr` | no | yes, `gnoweb` |
| [`home.gno:16`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/home/home.gno#L16) `Admin` | yes | yes |
| [`foo20.gno:18`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/demo/defi/foo20/foo20.gno#L18) `Ownable` | yes | yes |
| [`foo1155.gno:12`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/demo/foo1155/foo1155.gno#L12) `admin` | yes | no |
| [`foo721.gno:9`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/demo/foo721/foo721.gno#L9) `admin` | yes | no |
| [`faucet.gno:14`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/gnoland/faucet/faucet.gno#L14) `gAdminAddr` | yes | no |
| [`pages/admin.gno:18`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/gnoland/pages/admin.gno#L18) `adminAddr` | no | no |
| [`releases_example/example.gno:12`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/demo/releases_example/example.gno#L12) `admin` | no | no |

Every one of these fires when a source constant and a fixture copy disagree, and
none fires when both are wrong together.

## Suggestions

- **[two constants are asserted nowhere]** [`quarantined/gno.land/r/gnoland/pages/admin.gno:18`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/gnoland/pages/admin.gno#L18) · [↗](../../../../../.worktrees/gno-review-6131/examples/quarantined/gno.land/r/gnoland/pages/admin.gno#L18) — this and [`releases_example/example.gno:12`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/demo/releases_example/example.gno#L12) · [↗](../../../../../.worktrees/gno-review-6131/examples/quarantined/gno.land/r/demo/releases_example/example.gno#L12) each hold the only copy of their address, so a partial sweep past either is silent in every suite.
  <details><summary>details</summary>

  `git grep -c` returns one occurrence in each package, and reverting both to the
  old address leaves `gno test` on both realms green. The description says the
  quarantined realms are covered by `gno test ./...`, which is true of the packages
  and not of these two constants. Fix: assert the constant in each realm's test, as
  the three other quarantined realms already do.
  </details>

## Nits

- **[the blog suite is green for any admin address]** [`examples/gno.land/r/gnoland/blog/admin_test.gno:27`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin_test.gno#L27) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin_test.gno#L27) — `clearState` assigns a second hardcoded copy of the address over the source constant, so the realm's own tests assert the test's literal rather than [`admin.gno:20`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L20).
  <details><summary>details</summary>

  Every test in the package calls `clearState` first, so the three goldens in
  [`gnoblog_test.gno`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/gnoblog_test.gno#L8) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/gnoblog_test.gno#L8)
  that carry an address render the assigned literal. Reverting `admin.gno:20` alone
  to the old address leaves `gno test -C examples ./gno.land/r/gnoland/blog` green.
  The failure that does catch it comes from `gno.land/pkg/gnoweb`, a different
  module, which boots a node and replays the two genesis `ModAddPost` calls.

  The description groups this file under fixtures that must stay consistent with the
  constants, which is what makes it worth saying: it was updated in step with the
  constant, and it would have stayed green had it not been. Fix: capture the
  constant once before any test mutates it, five lines in
  [`tests/blog-admin-test-fix.patch`](tests/blog-admin-test-fix.patch). With the
  patch applied the suite stays green on an untouched constant and fails on a
  reverted one, naming both addresses in the diff of the rendered post.
  </details>

## Verified

Every claim below is a run made for this round, in the reviewed worktree at
f2bdb07b0, restored clean afterwards.

- The substitution damaged no formatting. All 333 changed line pairs are
  byte-identical once each address maps to one token, apart from three comment
  lines dropping `gnoland1`, so no line lost or gained a space. `git diff --check`
  reports nothing, `gno fmt -diff` is clean over all 14 changed packages, and both
  addresses are 40 characters, so no aligned block needed re-padding. The tree
  holds 393 occurrences of the exact new string and 14 of the old, with no
  truncated or malformed variant of either.
- The new address decodes to twenty bytes that round-trip to the same string, and
  bumping its last character is rejected with `invalid checksum`. 38 of its 40
  characters differ from the old address. It appears nowhere in the tree at the
  merge base, so it was not already some other actor's address.
- Neither address's twenty bytes appear anywhere in the tree as hex or base64, and
  searching the bech32 data part alone, which is prefix independent and would catch
  a `cosmos1` or `gno1` rendering, returns the same 13 files as the full-address
  search. No markdown file carries either address.
- The `genesis_txs.jsonl` rewrite was necessary. Restoring only the merge base's
  copy of that file under this head's realms, which is the state forgetting it would
  have shipped, panics the node during genesis:
  `VM panic: access restricted: not moderator`, raised at
  [`admin.gno:145`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L145) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L145), which the genesis
  `ModAddPost` path reaches from [`:65`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L65) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L65), and taking [`TestRoutes`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/gnoweb/app_test.go#L65) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/gnoweb/app_test.go#L65) down with it.
- Regenerating the goldens over `boards2/v1`, `blog`, `home`, `sys/names`, `foo20`
  and `p/gnoland/boards/exts/hub` leaves the tree unchanged, so no golden needed
  regenerating.
- Reverting [`blog/admin.gno:20`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L20)
  alone leaves `gno test` on that realm green. With
  [`tests/blog-admin-test-fix.patch`](tests/blog-admin-test-fix.patch) applied the
  same revert fails, printing both addresses in the rendered post, and an untouched
  constant still passes.

## Not posted, no change needed

- `topaz` carries no check tying `NAMES_ADMIN` to the `caller_override` that ships,
  where the description says `pearl` and `sapphire` assert it and `die`. Not posted:
  the check is that builder's own discipline and nothing in this change turns on it.
- The locked build checksums go stale, which the description names, along with this
  change being the cause and no chain needing a re-cut. Two refinements from
  [`tests/checksum-replay.sh`](tests/checksum-replay.sh): only `pearl` had a live
  lock, since `sapphire` already missed its own at the merge base and `topaz` and
  `test13` already missed `packages.gen.txt` one gate earlier; and re-locking `pearl`
  would contradict [`VALIDATOR.md:45`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/VALIDATOR.md?plain=1#L45) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/VALIDATOR.md#L45),
  which publishes the hash a validator checks the released genesis against. Neither
  changes what the author does: the change is aimed at the next genesis, and nobody
  re-cuts a launched chain.
- `test13` deploys `r/sys/names` and `boards2/v1` from the current `examples/` tree
  rather than from immutable historical code, so its eight retained callers no
  longer match the constants they mirror. Round 1 raised this as a Warning. Not
  posted: the description states the decision to leave the whole tree alone, the
  chain is no longer re-derived, and a re-derivation would have to redo those
  patches deliberately, which the description also says.
- No test, lint or CI job ties a builder's `NAMES_ADMIN` to the `verifier.gno`
  constant it mirrors. [`tests/admin_consistency_test.go`](tests/admin_consistency_test.go)
  closes it and passes at this head once `test13` is out of scope, so it would guard
  a future rotation rather than report anything today.
