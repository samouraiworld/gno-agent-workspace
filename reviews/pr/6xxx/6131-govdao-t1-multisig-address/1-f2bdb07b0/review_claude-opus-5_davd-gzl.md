# PR [#6131](https://github.com/gnolang/gno/pull/6131): chore: update GovDAO T1 multisig address

URL: https://github.com/gnolang/gno/pull/6131
Author: moul | Base: master | Files: 253 | +333 -333
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: `f2bdb07b0` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6131 f2bdb07b0`
Overview: [overview](../overview.md)

## Overview

The GovDAO T1 multisig changed one signer, which changes the address derived
from its signer set. Five gno.land realms hold that address as a source
constant, and several hundred test fixtures and genesis files hold copies that
have to agree with those constants. This change rewrites 393 of the 407 copies
in the tree and leaves 14 under `misc/deployments/gnoland1/` and
`misc/deployments/test13.gno.land/`, which record chains that already ran.

**Verdict: REQUEST CHANGES** — the swap is complete and provably a pure
substitution, and it leaves eight `test13` caller fields pointing at an address
neither realm accepts, for a reason `test13`'s own README contradicts
(1 Warning, 1 Suggestion).

## Verify first

- [`examples/gno.land/r/sys/names/verifier.gno:59`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L59) — this constant is compared at [`verifier.gno:117`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L117) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L117) inside a one-shot call. Confirm the signer set is final before any genesis is cut from this branch, because a chain cut with a stale value there needs a new chain.
- Run [`tests/deployment-admin-consistency.sh`](tests/deployment-admin-consistency.sh) from a checkout of this branch: it prints one row per deployment admin and exits non-zero on `test13`.

## Summary

Every added line equals its removed line once the two addresses are normalised
to the same token, apart from three comment lines in the `pearl`, `sapphire` and
`topaz` `gen-genesis.sh` files that drop the word `gnoland1` from a description
of the multisig. So nothing unrelated rode along, and no line carries a partial
substitution. The new address decodes as a valid twenty-byte gno address, which
a consistent typo would not, and no hex or base64 encoding of either address's
bytes appears anywhere in the tree, so no derived copy escaped the textual
sweep.

What survives is a consistency problem the sweep created rather than missed.
`misc/deployments/test13.gno.land/` deploys `r/sys/names` and `boards2/v1` from
the current `examples/` tree, so its retained old-address callers now point at
an address neither realm accepts.

Reading order: [`verifier.gno:59`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L59),
[`boards.gno:47`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47),
then [`test13.gno.land/README.md:102`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/README.md?plain=1#L102) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/README.md#L102),
then the 237 test fixtures and genesis data files, which carry no logic.

## Benchmarks / Numbers

Occurrences of each address, counted with `git grep -o` at the merge base
afffd8782 and at f2bdb07b0.

| Area | Old at base | New at head | Old at head |
| --- | ---: | ---: | ---: |
| `examples/gno.land/r/gnoland/boards2/v1/filetests/` | 348 | 348 | 0 |
| `examples/quarantined/` | 14 | 14 | 0 |
| `examples/gno.land/r/gnoland/blog/` | 9 | 9 | 0 |
| `gno.land/pkg/integration/testdata/` | 6 | 6 | 0 |
| `examples/gno.land/r/demo/defi/foo20/` | 3 | 3 | 0 |
| `examples/gno.land/r/gnoland/home/` | 2 | 2 | 0 |
| `gno.land/genesis/` | 2 | 2 | 0 |
| `examples/gno.land/r/gnoland/boards2/v1/` | 1 | 1 | 0 |
| `examples/gno.land/r/sys/names/` | 1 | 1 | 0 |
| `examples/gno.land/p/gnoland/boards/` | 1 | 1 | 0 |
| `misc/deployments/pearl.gno.land/` | 2 | 2 | 0 |
| `misc/deployments/sapphire.gno.land/` | 2 | 2 | 0 |
| `misc/deployments/topaz.gno.land/` | 2 | 2 | 0 |
| `misc/deployments/test13.gno.land/` | 13 | 0 | 13 |
| `misc/deployments/gnoland1/` | 1 | 0 | 1 |
| **Total** | **407** | **393** | **14** |

266 files held the old address at the merge base and 13 still hold it, which
accounts for all 253 changed files. Sixteen of those are realm source or
deployment scripts and 237 are test fixtures and genesis data. No file holds
both addresses, and no file holds a truncated form of either.

Admin agreement between `examples/` and each deployment, from
[`tests/deployment-admin-consistency.sh`](tests/deployment-admin-consistency.sh).

| Check | Merge base | f2bdb07b0 |
| --- | --- | --- |
| `pearl` `NAMES_ADMIN` and `caller_override` | agree | agree |
| `sapphire` `NAMES_ADMIN` and `caller_override` | agree | agree |
| `topaz` `NAMES_ADMIN` and `caller_override` | agree | agree |
| `test13` `NAMES_ADMIN` | agrees | differs |
| `test13` `names-enable` `caller_override` | agrees | differs |
| `test13` 7 `boards2-cascade` callers | agree | differ |

## Warnings (should fix)

- **[eight callers stopped matching the constants they mirror]** [`test13.gno.land/transactions/migration/names-enable/meta.json:5`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json#L5) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json#L5) — eight `test13` caller fields agreed with the `examples/` constants at the merge base and disagree at this head, so a re-derived `test13` genesis loses namespace enforcement and seven boards2 replays.
  <details><summary>details</summary>

  `test13` does not run gnoland1's deployed code. Its
  [`FILTERED_PACKAGES`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L113-L126) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/gen-genesis.sh#L113-L126)
  lists `./gno.land/r/sys/...` and `./gno.land/r/gnoland/boards2/...`, resolved
  against [`EXAMPLES_DIR`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L637) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/gen-genesis.sh#L637),
  which is the repository's own `examples/` directory. The README states the
  consequence directly: the
  [`boards2-cascade` group](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/README.md?plain=1#L102) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/README.md#L102)
  exists because [PR 5280](https://github.com/gnolang/gno/pull/5280) edited
  `initRealmPermissions` in the repository and broke historical operations on
  `test13`. That group could not exist if `test13` ran immutable historical
  code, so the description's reason for leaving these files alone does not hold.

  After this change [`gPerms`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47)
  lists only the new address, while each cascade patch still rewrites its
  caller to the old one, so [`WithPermission`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/public.gno#L146) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/boards2/v1/public.gno#L146) rejects the seven transactions the
  patches exist to admit. The same split hits
  [`admin`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L59)
  and the `names-enable` caller.

  Swapping the `test13` files is not the fix. The
  [script picks the multisig](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L1535) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/test13.gno.land/gen-genesis.sh#L1535)
  because replayed gnoland1 history leaves it the only funded account at
  migration time, and the new address inherits no balance. The caller must now
  be funded and be the realm admin, and no single address is both. Fix: record
  that constraint in the description in place of the immutability claim, and add
  a check tying each `NAMES_ADMIN` to the `admin` constant in `verifier.gno` so
  the next divergence fails the script instead of the chain.
  </details>

## Suggestions

- **[NAMES_ADMIN reaches only a log line]** [`topaz.gno.land/gen-genesis.sh:137`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L137) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/topaz.gno.land/gen-genesis.sh#L137) — `topaz` has no check that `NAMES_ADMIN` and the `names-enable` `caller_override` agree, so its `NAMES_ADMIN` only ever reaches a log line.
  <details><summary>details</summary>

  `pearl` and `sapphire` compare the two and call `die` on a mismatch at
  [`pearl/gen-genesis.sh:798-800`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L798-L800) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/pearl.gno.land/gen-genesis.sh#L798-L800).
  `topaz` reads `NAMES_ADMIN` once, in the
  [substep label](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L722) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/topaz.gno.land/gen-genesis.sh#L722),
  and takes the real caller from `meta.json`, so a half-applied swap would print
  one address and cut genesis with another. Fix: copy the three-line comparison
  from `pearl`.
  </details>

## Verified

- The diff is a pure substitution. Added and removed lines are equal as
  multisets after mapping each address to one token, apart from three
  `gen-genesis.sh` comment lines that drop `gnoland1` from a phrase naming the
  multisig. Command in [`tests/normalised-diff.sh`](tests/normalised-diff.sh).
- The new address decodes through [`crypto.AddressFromBech32`](https://github.com/gnolang/gno/blob/f2bdb07b0/tm2/pkg/crypto/bech32.go#L19) · [↗](../../../../../.worktrees/gno-review-6131/tm2/pkg/crypto/bech32.go#L19) to twenty bytes and
  round-trips to the same string. Corrupting its last character is rejected with
  `invalid checksum`, so the decode is discriminating rather than permissive.
- No hex or base64 encoding of either address's twenty bytes appears in the
  tree, so no fixture holds a derived copy the textual sweep could not see.
- A stale `patchpkg` address fails loudly rather than silently. Reverting only
  [`sys_namereg_v1_controller.txtar:24`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/integration/testdata/sys_namereg_v1_controller.txtar#L24) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/integration/testdata/sys_namereg_v1_controller.txtar#L24)
  to the old address makes the test fail with `VM panic: caller is not admin`,
  raised at [`verifier.gno:118`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L118) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L118).
- The consistency harness runs nine checks over `test13`, the eight caller
  fields plus the `NAMES_ADMIN` variable that mirrors them. All nine agree at
  the merge base and all nine disagree at f2bdb07b0, so the branch causes the
  divergence rather than exposing an older one.
- Green at f2bdb07b0: `gno test` over `r/gnoland/blog`, `r/gnoland/home`,
  `r/sys/names`, `r/demo/defi/foo20`, `p/gnoland/boards/exts/hub`,
  `r/gnoland/boards2/v1` covering 228 filetests, and the five quarantined
  realms; `go test ./gno.land/pkg/integration/` over the six named txtar files.

## Open questions

- Historical gnoland1 transactions sent by the old multisig against
  `r/gnoland/blog`, `r/gnoland/home` and `boards2/v1` would meet the same admin
  mismatch on a re-derived `test13`, and only `boards2` has a patch group today.
  Measuring that needs the gnoland1 transaction stream, which the description
  puts at one to two hours to fetch. Not posted: the Warning already asks for
  the check that would surface it.
- The description says the change touches 228 boards2 filetests where 218 carry
  the address and 228 is the directory's total. Not posted: it changes nothing
  the author would do.
- The description opens `Draft on purpose` and asks that the change not be
  merged until the signer set settles, while the pull request was marked ready
  for review on 2026-09-03. Not posted: the sentence asking not to merge is
  still in the description, so the request reaches whoever merges.

## Not posted, no change needed

- `gno.land/genesis/genesis_txs.jsonl` keeps a signature and public key on both
  rewritten lines that belong to neither address. The public key decodes to
  `g1u7y667z64x2h7vc6fmpcprgey4ck233jaww9zq`, so caller and signer already
  disagreed at the merge base, and the only consumer is
  [`gnoweb/app_test.go:47`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/gnoweb/app_test.go#L47) · [↗](../../../../../.worktrees/gno-review-6131/gno.land/pkg/gnoweb/app_test.go#L47),
  which loads the file with verification off.
- The realm quick checks in [`gno-ai-contract-review.md` quick checks](https://github.com/gnolang/gno/blob/f2bdb07b0/docs/resources/gno-ai-contract-review.md?plain=1#L8), read
  from the [PR 5871](https://github.com/gnolang/gno/pull/5871) branch, and every
  class in the invariant catalog: the diff adds no statement, no caller check
  and no control flow, so each check reads the same code before and after.
