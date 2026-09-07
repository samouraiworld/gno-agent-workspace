# PR [#6131](https://github.com/gnolang/gno/pull/6131): chore: update GovDAO T1 multisig address

URL: https://github.com/gnolang/gno/pull/6131
Author: moul | Base: master | Files: 253 | +333 -333
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: `f2bdb07b0` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6131 f2bdb07b0`
Overview: [overview](../overview.md)

Round 2, deep mode over the commit round 1 reviewed. The head did not move. Four
lenses ran on f2bdb07b0: red team, blue team, correctness and deployments. The
round confirms round 1's REQUEST CHANGES and its `test13` Warning, and adds four
Warnings round 1 did not reach. Round 1's topaz Suggestion is widened here: the
check it asked topaz to copy cannot catch this change's failure at all.

## Overview

The GovDAO T1 multisig changed one signer, so the address derived from its signer
set changed with it. Five gno.land realms hold that address as a source constant,
and several hundred test fixtures and genesis files hold copies that have to agree
with those constants. This change rewrites 393 of the 407 copies in the tree and
leaves 14 under `misc/deployments/gnoland1/` and `misc/deployments/test13.gno.land/`,
which record chains that already ran. The realm source is only half the surface:
four launched chains have a build script that pins a sha256 of the genesis it
produced, and those scripts read the same `examples/` tree the change rewrites.

**Verdict: REQUEST CHANGES** — the substitution is complete and provably pure, and
it stops [`pearl`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L213) from
rebuilding its own genesis, leaves eight `test13` callers pointing at an address
neither realm accepts, and lands in a tree where nothing ties any of it back to the
constant it mirrors (6 Warnings, 2 Suggestions, 1 Nit).

## Verify first

- [`examples/gno.land/r/sys/names/verifier.gno:59`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L59) — `admin` appears exactly twice in the realm, here and at the comparison on [`:117`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L117) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L117). There is no assignment, so there is no rotation. Confirm the signer set is final before any genesis is cut from this branch.
- Run [`tests/checksum-replay.sh`](tests/checksum-replay.sh) for `pearl` at the merge base and at this head, and compare both values against the heredoc at [`gen-genesis.sh:213`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L213) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L213). The base value is the locked one and this head's is not.
- Run [`tests/admin_consistency_test.go`](tests/admin_consistency_test.go) from a checkout of this branch: it passes at the merge base and fails on `test13` here.

## Summary

Every added line equals its removed line once the two addresses normalise to one
token, apart from three comment lines in the `pearl`, `sapphire` and `topaz`
builders that drop the word `gnoland1` from a phrase naming the multisig. The new
address is byte-identical to the file the description cites as its source, decodes
through the repository's own bech32 reader to twenty bytes, and is rejected on a
one-character corruption. So nothing unrelated rode along and the value is right.

What the sweep did not reach is everything downstream of `examples/`. Four launched
chains rebuild their genesis from that tree and check the result against a locked
hash, and **`pearl`** is the one whose lock was still live: it reproduces at the
merge base and not here, so its build now aborts. Underneath both that and round 1's
`test13` finding sits one gap: no test, lint or CI job ties a deployment's admin to
the realm constant it mirrors, and the one check that exists compares two copies the
same substitution rewrote together.

Reading order: [`verifier.gno:59`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L59),
[`boards.gno:47`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47),
then [`pearl/gen-genesis.sh:213`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L213) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L213),
then [`test13.gno.land/README.md:102`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/README.md?plain=1#L102) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/README.md#L102),
then the 237 fixtures and genesis data files, which carry no logic.

## Benchmarks / Numbers

Each builder replayed through its own `verify_checksum "$GENESIS_TXS_JSONL"` gate
with [`tests/checksum-replay.sh`](tests/checksum-replay.sh), and the package list
regenerated for all four. `work/valoper-seed.jsonl` is the fidelity control: it
does not derive from `examples/`, so a run reproducing its locked value has
replayed the pipeline faithfully.

| Builder | First gate that fails | Locked | Merge base | f2bdb07b0 | Caused here |
| --- | --- | --- | --- | --- | --- |
| `pearl` | `genesis_txs.jsonl`, step 7 | `499d9fba` | `499d9fba` | `79cea215` | **yes** |
| `sapphire` | `genesis_txs.jsonl`, step 7 | `ed781241` | `6f583f6d` | `014cafdb` | no |
| `topaz` | `packages.gen.txt`, step 4 | `dac29a0c` | `2f686094` | `2f686094` | no |
| `test13` | `packages.gen.txt`, phase 1 step 4 | `397c3c90` | `2f686094` | `2f686094` | no |

`valoper-seed.jsonl` reproduces its locked value at both revisions, `7717a8fc` for
`pearl` and `bf1c81dd` for `sapphire`. The four builders resolve the same twelve
`FILTERED_PACKAGES` entries against the same tree, so the three different
`packages.gen.txt` hashes cannot all be reachable.

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
none fires when both are wrong together, so the green suite proves the tree is
self-consistent rather than that the address is right.

## Warnings (should fix)

- **[pearl can no longer rebuild its own genesis]** [`pearl.gno.land/gen-genesis.sh:213`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L213) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L213) — the locked `work/genesis_txs.jsonl` hash reproduces at the merge base and not at this head, so the build aborts at step 7 of 9, and re-locking it would contradict the hash validators are told to check.
  <details><summary>details</summary>

  The change rewrites 224 files inside the 85-package closure `pearl` deploys, and
  four of the addpkg transactions in its genesis stream carry the address. The
  replay in [`tests/checksum-replay.sh`](tests/checksum-replay.sh) gives
  `499d9fbaaea8822d873a8e6693e329c9347bc69223daf056c4f31fe28aa437dc` at the merge
  base, equal to the locked value, and
  `79cea215fe95bc8a4548e31f44e59f041f4578575970a05d73b6af63888f4393` here.
  [`verify_checksum`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L347) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L347)
  ends in `exit 1` on a mismatch, reached from the call at
  [`:862`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L862) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L862).

  The description reads this as all four locks going stale together, which the
  table above measures otherwise: `sapphire` already misses its lock at the merge
  base, and `topaz` and `test13` already miss theirs one gate earlier. `pearl` is
  the only builder this change breaks.

  That matters because the remedy differs. Re-running and pasting the new hashes is
  right for a builder whose lock was already meaningless. For `pearl` it discards
  the one thing the lock records: [`VALIDATOR.md:45`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/VALIDATOR.md?plain=1#L45) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/VALIDATOR.md#L45)
  publishes `c45fe60c` as the hash a validator checks the released genesis against,
  and [`gen-genesis.sh:216`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L216) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L216)
  pins the same value, so a re-lock points the script at a genesis that is not the
  chain now running. Fix: pin `pearl`'s inputs to the tree that produced its
  genesis, or record in the script that it no longer reproduces the shipped
  artifact, rather than re-locking it.
  </details>

- **[eight callers stopped matching the constants they mirror]** [`test13.gno.land/transactions/migration/names-enable/meta.json:5`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json#L5) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json#L5) — eight `test13` caller fields agreed with the `examples/` constants at the merge base and disagree here, so a re-derived `test13` genesis loses namespace enforcement and seven boards2 replays.
  <details><summary>details</summary>

  Carried from round 1 and re-verified. `test13` does not run gnoland1's deployed
  code: its [`FILTERED_PACKAGES`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L113-L126) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/gen-genesis.sh#L113-L126)
  lists `./gno.land/r/sys/...` and `./gno.land/r/gnoland/boards2/...`, resolved
  against [`EXAMPLES_DIR`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L637) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/gen-genesis.sh#L637),
  which is the repository's own `examples/` directory.

  Round 2 adds the audit text. Each of the seven cascade patches states the
  invariant as its own reason for existing:
  [`h126810/meta.json:2`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json#L2) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json#L2)
  says it rewrites the caller to the multisig because that is "the only address
  still listed as boards2/v1 admin". After this change the only address listed at
  [`boards.gno:47`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47)
  is the new one, while every patch still writes the old one. That `reason` field is
  carried into the genesis stream, so the shipped artifact documents a permission
  check it no longer passes. The same claim sits in
  [`README.md:102`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/README.md?plain=1#L102) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/README.md#L102).

  Swapping the `test13` files is not the fix. The
  [script picks the multisig](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L1535) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/gen-genesis.sh#L1535)
  because replayed gnoland1 history leaves it the only funded account at migration
  time, and the new address inherits no balance. The caller now has to be funded and
  be the realm admin, and no single address is both. Fix: record that constraint in
  the description and in the `test13` README, in place of the immutability claim.
  </details>

- **[nothing ties a deployment admin to the constant it mirrors]** [`examples/gno.land/r/sys/names/verifier.gno:59`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L59) — four builders document this constant as the value their `NAMES_ADMIN` mirrors, none reads it, and the one check that exists compares two copies the same substitution rewrote together.
  <details><summary>details</summary>

  `pearl` and `sapphire` `die` when `caller_override` and `NAMES_ADMIN` disagree, at
  [`pearl/gen-genesis.sh:799-800`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L799-L800) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L799-L800).
  Both operands live under `misc/deployments/`, so one pass over the tree satisfies
  the check by construction, which is what happened here. Nothing in CI builds,
  lints or executes any `gen-genesis.sh`, and `ci / genesis-verify` runs on a
  matrix of one chain that is none of these four.

  Fix: [`tests/admin_consistency_test.go`](tests/admin_consistency_test.go), 108
  lines at `gno.land/pkg/deployments/`, which is a path `ci / gnoland` already runs
  and which already triggers on `examples/**`. It reads `admin` out of
  `verifier.gno` and asserts every builder's `NAMES_ADMIN` and every `names-enable`
  `caller_override` equals it, failing the run if the declaration stops matching so
  the check cannot decay into a no-op. It passes at the merge base and fails here:

  ```
  --- FAIL: TestNamesAdminMatchesRealmConstant/test13.gno.land
      NAMES_ADMIN = g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh, want g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf (the admin in examples/gno.land/r/sys/names/verifier.gno)
      names-enable caller_override = g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh, want g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf (the admin in examples/gno.land/r/sys/names/verifier.gno)
  ```
  </details>

- **[the blog suite is green for any admin address]** [`examples/gno.land/r/gnoland/blog/admin_test.gno:27`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin_test.gno#L27) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin_test.gno#L27) — `clearState` assigns a second hardcoded copy of the address over the source constant, so the realm's own tests assert the test's literal rather than [`admin.gno:20`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L20).
  <details><summary>details</summary>

  Every test in the package calls `clearState` first, so the three goldens in
  [`gnoblog_test.gno`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/gnoblog_test.gno#L8) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/gnoblog_test.gno#L8) that carry an address render the assigned literal. Reverting `admin.gno:20` alone to
  the old address leaves `gno test ./gno.land/r/gnoland/blog` green. The failure
  that does catch it comes from `gno.land/pkg/gnoweb`, a different module, which
  boots a node and replays the two genesis `ModAddPost` calls.

  Fix: capture the constant once before any test mutates it, five lines in
  [`tests/blog-admin-test-fix.patch`](tests/blog-admin-test-fix.patch). With the
  patch applied the suite stays green on an untouched constant and fails on a
  reverted one, naming both addresses in the diff of the rendered post.
  </details>

- **[the description asks for a merge hold the pull request does not enforce]** — the body opens "Draft on purpose" and asks that this not merge until the signer set settles, while the pull request is marked ready for review and `MERGEABLE`.
  <details><summary>details</summary>

  `gh pr view 6131 -R gnolang/gno --json isDraft,state` reads
  `{"isDraft":false,"state":"OPEN"}`. The description's own argument is what makes
  this worth raising: `Enable` is one-shot, `enabled` has no setter back to false,
  and `admin` has no assignment anywhere in the realm, so a genesis cut from a
  stale constant needs a new chain. Round 1 left this in Open questions on the
  grounds that the sentence reaches whoever merges, which it does: it is the 34th
  word of a 941-word description, directly under `## Summary`. What it does not
  reach is the merge button. Fix: convert the pull request to a draft, or delete
  the sentence.
  </details>

- **[the balance move the description schedules strands the balance]** — `ugnot` transfers are restricted chain-wide on `gnoland1` and only a whitelisted sender may move them, so the transfer the description plans succeeds and leaves 119 million GNOT at an address that cannot spend it.
  <details><summary>details</summary>

  Queried against `https://rpc.gno.land`, network `gnoland1`, for this round.
  `params/bank:p:restricted_denoms` reads `["ugnot"]` and
  `params/auth:p:unrestricted_addrs` lists the old multisig and two other
  addresses. The old account carries `"attributes": "1"`, which is
  [`flagTokenLockWhitelisted`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/gnoland/types.go#L27) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/gnoland/types.go#L27), and the
  new address has no account at all, so it is created on receipt with no flags.

  [`SendCoins`](https://github.com/gnolang/gno/blob/f2bdb07b0/tm2/pkg/sdk/bank/keeper.go#L153) · [↗](../../../../../.worktrees/gno-review-6131/tm2/pkg/sdk/bank/keeper.go#L153) gates on
  [`canSendCoins`](https://github.com/gnolang/gno/blob/f2bdb07b0/tm2/pkg/sdk/bank/keeper.go#L135-L150) · [↗](../../../../../.worktrees/gno-review-6131/tm2/pkg/sdk/bank/keeper.go#L135-L150) for the sender
  alone and returns `RestrictedTransferError` otherwise. So the description's step
  works in the direction it describes, the old multisig signing while its signer
  set still exists, and every send out of the new address afterwards fails.

  The new address cannot repair its own state. The whitelist has two write paths,
  [`applyUnrestrictedAddrs`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/gnoland/app.go#L606) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/gnoland/app.go#L606) at
  InitChainer, which `gnoland1` is long past, and a GovDAO proposal through
  [`ProposeAddUnrestrictedAcctsRequest`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/params/unlock.gno#L22) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/params/unlock.gno#L22).
  Live T1 membership is one address, `g1manfred47kzduec920z88wfr64ylksmdcedlf5`,
  and the multisig is not in it. Fix: land the unrestricted-accounts proposal for
  the new address before the transfer, and say so in the paragraph that schedules
  the transfer.

  The same paragraph is where two realm ownerships belong. `pearl`, `sapphire` and
  `topaz` were each cut from a genesis carrying the old address, so
  [`blog.adminAddr`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L20) and boards2
  [`gPerms`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) are owned on
  those chains by the old multisig and this change does not move them. Each rotation
  needs a transaction signed by the same signer set. `r/gnoland/home` and
  `r/demo/defi/foo20` hold the constant too and carry no such deadline: the twelve
  [`FILTERED_PACKAGES`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L68) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L68)
  entries the four builders share resolve to 85 packages and neither is among them.
  </details>

## Suggestions

- **[two builders never check the caller they actually ship]** [`topaz.gno.land/gen-genesis.sh:137`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L137) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/topaz.gno.land/gen-genesis.sh#L137) — `topaz` and `test13` read `NAMES_ADMIN` only into a log line while the real caller comes from `meta.json`, so a half-applied swap prints one address and cuts genesis with another.
  <details><summary>details</summary>

  Widened from round 1, which named `topaz` alone. `topaz` uses the variable once,
  in the substep label at [`:722`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L722) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/topaz.gno.land/gen-genesis.sh#L722).
  `test13` uses it in a comment and two labels, and then writes the address out
  longhand at [`:1209`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L1209) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/gen-genesis.sh#L1209)
  where `fork valoper-seed` takes its fee payer, a thousand lines below the
  variable holding the same value.

  Fix: copy the three-line comparison from `pearl` into both, and pass
  `--caller "\$NAMES_ADMIN"` at `test13:1209`. This is worth doing beside the check
  above rather than instead of it: the comparison catches a half-applied swap, and
  only the test catches a fully applied one.
  </details>

- **[three fixtures that were never the multisig moved with it]** [`examples/gno.land/p/gnoland/boards/exts/hub/comment_test.gno:14`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/p/gnoland/boards/exts/hub/comment_test.gno#L14) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/p/gnoland/boards/exts/hub/comment_test.gno#L14) — this address is bound to `bob`, and two more sites bind it to `sink` and `chain_addr`, so three fixtures now read as governance references and will move again on every rotation.
  <details><summary>details</summary>

  The other two are
  [`grc20_registry_wrappers.txtar:115`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/integration/testdata/grc20_registry_wrappers.txtar#L115) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/integration/testdata/grc20_registry_wrappers.txtar#L115)
  and [`grc721_callerteller_home.txtar:93`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/integration/testdata/grc721_callerteller_home.txtar#L93) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/integration/testdata/grc721_callerteller_home.txtar#L93).
  None was ever tied to the multisig: `git log -S` on the old address puts each in a
  separate unrelated pull request that dropped the value in as an anonymous
  participant, and no comment at either revision connects them.

  Reverting all three to the old address leaves `gno test` on the `hub` package and
  `go test` on both txtar files green, while the same edit on a genuinely tied
  fixture reddens `r/gnoland/blog` with both addresses in the diff. So the null
  result discriminates. Fix: leave the three as they were, or point them at a
  distinct fixture address so the coupling is broken rather than recreated.
  </details>


## Nits

- **[one comment now names two addresses]** [`test13.gno.land/transactions/patched/boards2-permissions/h227223/pkg/permissions.gno:13`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/patched/boards2-permissions/h227223/pkg/permissions.gno#L13) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/transactions/patched/boards2-permissions/h227223/pkg/permissions.gno#L13) — the bare phrase `GovDAO T1 multisig` sits beside the old address here and beside the new one under `examples/`, and the sibling at [`govdao_prop1_test13.gno:60`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/base/bootstrap/govdao_prop1_test13.gno#L60) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/transactions/base/bootstrap/govdao_prop1_test13.gno#L60) shows the fix already in use, `inherited from gnoland1`.

## Verified

Every claim below is a run made for this round, in the reviewed worktree at
f2bdb07b0, restored clean afterwards.

- The new address decodes to twenty bytes that round-trip to the same string, and
  bumping its last character is rejected with `invalid checksum`, so the decode
  discriminates rather than accepting anything. 38 of its 40 characters differ from
  the old address, which rules out a typo. It appears nowhere in the tree at the
  merge base, so it was not already some other actor's address.
- The live `gnoland1` chain answers `118999987767600ugnot` for the old address and
  has no account for the new one, which is the description's figure.
- Neither address's twenty bytes appear anywhere in the tree as hex or base64, and
  searching the bech32 data part alone, which is prefix independent and would catch
  a `cosmos1` or `gno1` rendering, returns the same 13 files as the full-address
  search. No markdown file carries either address, so no doc needed moving.
- The `genesis_txs.jsonl` rewrite was necessary. Restoring only the merge base's
  copy of that file under this head's realms, which is the state forgetting it would
  have shipped, panics the node during genesis:
  `VM panic: access restricted: not moderator`, raised at
  [`admin.gno:145`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L145) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L145), which the genesis
  `ModAddPost` path reaches from [`:65`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L65) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L65), and taking [`TestRoutes`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/gnoweb/app_test.go#L65) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/gnoweb/app_test.go#L65) down with it.
- Regenerating the goldens over `boards2/v1`, `blog`, `home`, `sys/names`, `foo20`
  and `p/gnoland/boards/exts/hub` leaves the tree unchanged, so the description's
  byte-exact claim holds.
- Reverting [`blog/admin.gno:20`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin.gno#L20)
  alone leaves `gno test` on that realm green. With
  [`tests/blog-admin-test-fix.patch`](tests/blog-admin-test-fix.patch) applied the
  same revert fails, printing both addresses in the rendered post, and an untouched
  constant still passes.
- Reverting all three untied fixtures at once leaves the `hub` package and both
  txtar tests green, while the same edit on a tied fixture reddens `blog`. So the
  null result discriminates.
- `gnodev` loads the five changed quarantined realms unless
  [`without-quarantined-examples`](https://github.com/gnolang/gno/blob/f2bdb07b0/contribs/gnodev/app_config.go#L148) · [↗](../../../../../.worktrees/gno-review-6131/contribs/gnodev/app_config.go#L148)
  is passed.

## Open questions

- Historical gnoland1 transactions sent by the old multisig against `r/gnoland/blog`
  and `r/gnoland/home` would meet the same admin mismatch on a re-derived `test13`,
  and only `boards2` has a patch group today. Measuring that needs the gnoland1
  transaction stream, which the description puts at one to two hours to fetch. Not
  posted: the Warning already asks for the check that would surface it.
- Several statements in the description do not survive a read of the files they
  cite: `gnoland1/govdao_prop1.gno` is a live generator input rather than a record,
  the `boards2-permissions` file's own `meta.json` gives API drift as its reason
  rather than agreement with the cascade patches, and the funding rationale the
  description attaches to `names-enable` belongs to `rotate-call`, and [`misc/loop/scripts/start.sh:46`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/loop/scripts/start.sh#L46) feeds
  `genesis_txs.jsonl` to a live `gnoland start` beside the one test consumer the
  description names. Not posted
  individually: the `test13` Warning asks for the paragraph to be rewritten, which
  covers all three.
- The description counts 228 boards2 filetests where 218 carry the address and 228
  is the directory total. Not posted: it changes nothing the author would do.

## Not posted, no change needed

- `sapphire`, `topaz` and `test13` all miss a locked hash, and each already missed
  it at the merge base, so this change is not the cause. `topaz` and `test13` miss
  `packages.gen.txt`, which this change cannot move: the package list regenerates
  to the same value at both revisions.
- [`verifier_test.gno:208`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier_test.gno#L208) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier_test.gno#L208), inside `TestEnable`,
  makes the test pass for any value of `admin`, since it builds its caller realm from the
  constant under test. Folded into the coverage table above rather than posted
  separately, because the check being asked for is the same one.
- [`gnoland1/Makefile:2`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/gnoland1/Makefile#L2) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/gnoland1/Makefile#L2)
  pins a genesis hash too, so five deployments pin one rather than the four the
  description names. Already stale on master rather than here: `git log -S` on that
  variable returns one commit, 3a60bd305 from 2026-03-16, and the value has not
  moved since.
