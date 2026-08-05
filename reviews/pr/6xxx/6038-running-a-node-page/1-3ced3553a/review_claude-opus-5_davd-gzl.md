# PR [#6038](https://github.com/gnolang/gno/pull/6038): docs: add a "Running a node" routing page

URL: https://github.com/gnolang/gno/pull/6038
Author: moul | Base: master | Files: 5 | +126 -6
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 3ced3553a (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6038 3ced3553a`

**TL;DR:** Adds one short page that tells a reader who was about to run a Gno node whether they need one at all, and sends each answer to the doc that covers it. Also gives the networks page a column and a section pointing node operators at the per-network deployment directories in the monorepo.

**Verdict: REQUEST CHANGES** — the routing itself is sound, but two of the new sentences describe a convention that Betanet, the row the table marks current, does not follow (3 Warnings, 6 Nits, 1 Suggestion).

## Verify first

- [`docs/resources/gnoland-networks.md:24`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L24) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L24) — this is the link an operator clicks, and it points at `master`. Run `git diff origin/master origin/chain/gnoland1 -- misc/deployments/gnoland1/config.toml` and confirm the two copies carry different `persistent_peers`.
- [`docs/resources/gnoland-networks.md:35-37`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L35-L37) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L35-L37) — confirm the release-artifact promise against Betanet's own tags. `gh api repos/gnolang/gno/releases/tags/chain/gnoland1.1 --jq '[.assets[].name]'` prints `[]`.
- [`docs/builders/running-a-node.md:81-82`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/running-a-node.md?plain=1#L81-L82) · [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L81-L82) — the page's only executable instruction. Run `sh misc/install.sh --full --dir /tmp/x`; it exits 1 before downloading anything.

## Summary

The page answers a real support question and every one of its 15 links resolves. Its weight rests on the new *Deployment files* section it adds to [`gnoland-networks.md`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L19) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L19), which is where a full-node or validator reader is sent for everything operational. That section states two conventions. Betanet follows neither: its `misc/deployments/` copy on `master` is 17 days behind the `chain/gnoland1` copy and carries a one-host peer list against the chain branch's three, and its release tags carry no binaries and no checksum file. Topaz and Test13 do follow both conventions, which is presumably where the generalisation came from.

The docs CI job passes at this head. `make generate -B` in `docs/` leaves the tree clean, so the hand-edited [`misc/docs/sidebar.json`](https://github.com/gnolang/gno/blob/3ced3553a/misc/docs/sidebar.json#L29) · [↗](../../../../../.worktrees/gno-review-6038/misc/docs/sidebar.json#L29) matches what [`indexparser`](https://github.com/gnolang/gno/blob/3ced3553a/misc/docs/tools/indexparser/main.go#L74) · [↗](../../../../../.worktrees/gno-review-6038/misc/docs/tools/indexparser/main.go#L74) generates from [`docs/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/docs/README.md?plain=1#L35) · [↗](../../../../../.worktrees/gno-review-6038/docs/README.md#L35), and `make lint` reports no unreachable local link and no 404 URL.

## Diagram

Where each arm of the router lands, and which target is wrong.

```
docs/builders/running-a-node.md
  ├─ Develop contracts ────► resources/gnodev.md, play.gno.land          ok
  ├─ Read a chain ────────► builders/rpc-clients.md, tx-indexer          ok
  ├─ Full node ───────────► gnoland-networks.md#deployment-files ──┐
  ├─ Validator ───────────► gnoland-networks.md#deployment-files ──┤
  │                         validators/tmkms.md, gnops.io            │  ok
  └─ Local chain ─────────► gno.land/cmd/gnoland (tree link)         │  ok
                            builders/install.md --full           ◄── installer
                                                                     exits 1

              gnoland-networks.md § Deployment files
                ├─ prose link  ──► misc/deployments @ master   ◄── stale copy
                ├─ infra table ──► misc/deployments @ master   ◄── stale copy
                ├─ Betanet row ──► misc/deployments/gnoland1 @ chain/gnoland1  ok
                └─ "artifacts on the release tag" ◄── chain/gnoland1.1 has none
```

## Benchmarks / Numbers

| Release tag | Assets |
|---|---|
| `chain/gnoland1.0` | `genesis.json`, `gno.wasm`, `root.zip` |
| `chain/gnoland1.1` | none |
| `chain/topaz` | `CHECKSUMS.txt`, `genesis.json`, 16 binaries |
| `chain/test13` | `CHECKSUMS.txt`, `genesis.json`, 16 binaries |

## Critical (must fix)

None.

> [PR 6039](https://github.com/gnolang/gno/pull/6039) is stacked on this branch and its body carries `Closes #6038`, so this pull request may close without merging under its own number. The two findings below that survive unchanged at 6039's head, the release-artifacts bullet and the one-line installer, are carried into [the 6039 review](../../6039-move-node-operator-docs/1-2c817cec4/review_claude-opus-5_davd-gzl.md) as well, re-verified at `2c817cec4`. Whichever pull request the author acts on, both reach them.

## Warnings (should fix)

- **[operator copies the wrong peer set]** [`docs/resources/gnoland-networks.md:24`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L24) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L24) — the section's own link to `misc/deployments/` points at `master`, the copy the bullet below it tells the reader not to use.
  <details><summary>details</summary>

  The Betanet table row is pinned to `chain/gnoland1`, but this link and the one in the infrastructure table at [`gnoland-networks.md:43`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L43) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L43) both resolve to `master`, and they sit above and below the prose that says the real directory is on `chain/<name>`. `master`'s copy of the config was last touched 2026-03-27 and names [one persistent peer](https://github.com/gnolang/gno/blob/master/misc/deployments/gnoland1/config.toml#L150) · [↗](../../../../../.worktrees/gno-review-6038/misc/deployments/gnoland1/config.toml#L150); the `chain/gnoland1` copy was last touched 2026-04-13 and names [three](https://github.com/gnolang/gno/blob/chain/gnoland1/misc/deployments/gnoland1/config.toml#L150). `master` also has no `govdao-scripts/`, and its `gen-genesis.sh` still points at the removed `./govdao` path. An operator who follows this link and runs the documented `cp config.toml gnoland-data/config/config.toml` from [`misc/deployments/gnoland1/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/misc/deployments/gnoland1/README.md?plain=1#L50) · [↗](../../../../../.worktrees/gno-review-6038/misc/deployments/gnoland1/README.md#L50) starts a node against the superseded peer set. Fix: point both `misc/deployments` links at the branch the surrounding prose names, or say in the link text that `master` is the archive view.
  </details>

- **[promised artifacts that do not exist]** [`docs/resources/gnoland-networks.md:35-37`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L35-L37) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L35-L37) — Betanet's release tags carry no binaries, no checksum file, and no container image.
  <details><summary>details</summary>

  The bullet promises binaries, container images, `genesis.json` and its checksum on "the matching release tag". `chain/gnoland1.1` has zero assets. `chain/gnoland1.0` has `genesis.json`, `gno.wasm` and `root.zip`, so no binary and no checksum. `chain/topaz` and `chain/test13` do carry the full set, which is where the generalisation holds. Container images are on no release tag at all; they are published to `ghcr.io/gnolang/gno`, as [`install.md:80`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/install.md?plain=1#L80) · [↗](../../../../../.worktrees/gno-review-6038/docs/builders/install.md#L80) already says. This matters because the branch a Betanet operator is sent to tells them to build from source instead, at [`misc/deployments/gnoland1/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/misc/deployments/gnoland1/README.md?plain=1#L33) · [↗](../../../../../.worktrees/gno-review-6038/misc/deployments/gnoland1/README.md#L33). Fix: scope the claim to the tags that carry the assets, and move container images to their registry. [repro](comment_claude-opus-5.md)
  </details>

- **[the page's one command fails]** [`docs/builders/running-a-node.md:81-82`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/running-a-node.md?plain=1#L81-L82) · [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L81-L82) — the one-line installer's `--full` flag cannot resolve a version today, so the local-chain arm ends in an error.
  <details><summary>details</summary>

  [`misc/install.sh`](https://github.com/gnolang/gno/blob/3ced3553a/misc/install.sh#L218-L220) · [↗](../../../../../.worktrees/gno-review-6038/misc/install.sh#L218-L220) resolves `latest` by taking the newest non-prerelease release whose tag starts with `v`. `gnolang/gno` publishes eight releases and all of them are `chain/*`, so the walk finds nothing and the run [dies](https://github.com/gnolang/gno/blob/3ced3553a/misc/install.sh#L298) · [↗](../../../../../.worktrees/gno-review-6038/misc/install.sh#L298) with `no v* release found`. The `v1.1.0` and `v1.0.0` git tags exist but have no GitHub release behind them, so the suggested `--version <tag>` escape does not work either. The defect is in `misc/install.sh` and predates this diff; what the diff changes is the exposure, since this line is the only executable instruction on a page written for people who want a node. The alternative path already documented for the same binary is `make install.gnoland`, at [`gno.land/cmd/gnoland/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/gno.land/cmd/gnoland/README.md?plain=1#L18-L22) · [↗](../../../../../.worktrees/gno-review-6038/gno.land/cmd/gnoland/README.md#L18-L22). Fix: route the reader to a path that works today, or fix the resolver first. [repro](comment_claude-opus-5.md)
  </details>

## Nits

- **[reader hunts for a file that is not there]** [`docs/builders/running-a-node.md:50`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/running-a-node.md?plain=1#L50) · [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L50) — says each network pins a "genesis file", but the Betanet directory ships `gen-genesis.sh` and a `make generate` step, not a genesis.
  <details><summary>details</summary>

  [`gnoland-networks.md:26`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L26) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L26), the page this line links to, is careful about exactly this and writes "its genesis (or the script that regenerates it)". Fix: carry the same hedge here.
  </details>

- **[convention stated wider than it holds]** [`docs/resources/gnoland-networks.md:25`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L25) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L25) — "one directory per network, holding its `config.toml`" does not hold for two of the directories the section indexes.
  <details><summary>details</summary>

  `misc/deployments/topaz.gno.land` on `chain/topaz` holds `README.md`, `VALIDATOR.md`, `gen-genesis.sh`, `govdao-exec.sh` and `transactions`, with no `config.toml`. On `master`, `misc/deployments/test12.gno.land` holds only `govdao/`. Fix: say what is usually there rather than what every directory holds.
  </details>

- **[column promises something the target is not]** [`docs/resources/gnoland-networks.md:8`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L8) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L8) — Staging's Deployment files cell points at `misc/loop`, which holds none of the three things the section below defines as deployment files.
  <details><summary>details</summary>

  `misc/loop` runs the portal loop: a docker container swapped behind a [Traefik proxy](https://github.com/gnolang/gno/blob/3ced3553a/misc/loop/README.md?plain=1#L20-L30) · [↗](../../../../../.worktrees/gno-review-6038/misc/loop/README.md#L20-L30) on every master change. There is no `config.toml`, no genesis and no join instructions, because Staging is not a network anyone joins. The same link already sits in the infrastructure table at [`gnoland-networks.md:44`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L44) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L44) with an accurate description. Fix: leave the cell empty for Staging.
  </details>

- **[placeholder matches nothing]** [`docs/resources/gnoland-networks.md:31`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L31) · [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L31) — `chain/<name>` reads as if `<name>` were something on the same row, and for Topaz it is none of the three.
  <details><summary>details</summary>

  Topaz's branch is `chain/topaz`, its directory is `topaz.gno.land` and its chain ID is `topaz-1`. Betanet happens to line up, so a reader who checks only the first row will read the rule as literal. Fix: name the branch per row, which the Deployment files column already does.
  </details>

- **[candidate gets two channels and no current one]** [`docs/builders/running-a-node.md:72-73`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/running-a-node.md?plain=1#L72-L73) · [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L72-L73) — sends a would-be validator to Discord, while the deployment README it routes through sends them to a Signal group.
  <details><summary>details</summary>

  [`misc/deployments/gnoland1/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/misc/deployments/gnoland1/README.md?plain=1#L89) · [↗](../../../../../.worktrees/gno-review-6038/misc/deployments/gnoland1/README.md#L89) step 5 reads "Ping the team on the validators Signal group so we can add you via a GovDAO proposal". A candidate reading both does not know which is live. Fix: name one.
  </details>

- **[bullet breaks the list's pattern]** [`README.md:34`](https://github.com/gnolang/gno/blob/3ced3553a/README.md?plain=1#L34) · [↗](../../../../../.worktrees/gno-review-6038/README.md#L34) — the other three bullets in that list link `./docs#<section>`; this one links a page file. Cosmetic, no enabled linter covers it, and the link resolves. Not posted, no change needed.

## Missing Tests

None. The `docs` job of [`ci-codegen-verify.yml`](https://github.com/gnolang/gno/blob/3ced3553a/.github/workflows/ci-codegen-verify.yml#L69-L87) · [↗](../../../../../.worktrees/gno-review-6038/.github/workflows/ci-codegen-verify.yml#L69-L87) already regenerates the sidebar and link-checks every added link.

## Suggestions

- **[cross-page anchors decay silently]** [`misc/docs/tools/linter/links.go:60-62`](https://github.com/gnolang/gno/blob/3ced3553a/misc/docs/tools/linter/links.go#L60-L62) · [↗](../../../../../.worktrees/gno-review-6038/misc/docs/tools/linter/links.go#L60-L62) — the docs linter truncates a link at `#` before checking it, so nothing verifies that `#deployment-files` names a heading.
  <details><summary>details</summary>

  This PR introduces the first two cross-page anchor links in `docs/`, at [`running-a-node.md:52`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/running-a-node.md?plain=1#L52) · [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L52) and [`running-a-node.md:66`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/running-a-node.md?plain=1#L66) · [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L66). Both are correct today. Renaming the `### Deployment files` heading later would leave the linter green and both links landing at the top of the page. Out of scope for this PR; noted for whoever touches the linter next.
  </details>

## Verified

- Ran the installer the page routes to. `sh misc/install.sh --full --dir /tmp/gno-install-check` exits 1 with `no v* release found` before contacting the download endpoint. The releases list confirms why: all eight releases on `gnolang/gno` are `chain/*` tags.
- Queried every Betanet release tag. `chain/gnoland1.1` returns an empty asset list and `chain/gnoland1.0` returns `genesis.json gno.wasm root.zip`, against `chain/topaz` and `chain/test13` which each return `CHECKSUMS.txt`, `genesis.json` and 16 binaries.
- Confirmed the two copies of the Betanet deployment directory diverge. `git diff origin/master chain/gnoland1 -- misc/deployments/gnoland1` shows a different `persistent_peers` and `seeds` value, a `govdao-scripts/` directory present only on the chain branch, and a `gen-genesis.sh` comment on `master` still naming the removed `./govdao` path.
- All 15 links the new page adds resolve, and `https://docs.gno.land/validators/tmkms` returns 200, so the tmkms target is routable on the live site even though it is in no sidebar.
- `make generate -B` and `make lint` in `docs/` both run clean at this head, reproducing the `docs` job of `ci-codegen-verify.yml`.

## Open questions

- PR [#6039](https://github.com/gnolang/gno/pull/6039) is stacked on this one: its branch carries 3ced3553a as its first commit, then rewrites the *Join a network* and *Become a validator* sections of this same page and renames `docs/validators/tmkms.md` to `gno.land/cmd/gnoland/TMKMS.md`. The tmkms link at `running-a-node.md:68` is the tree's only inbound reference to that file, and 6039's own commit removes it, so no merge order breaks it. The two Warnings on `gnoland-networks.md` survive 6039 unchanged, and one gets worse: 6039 rewrites the `misc/deployments` link from a `master` blob URL to the relative `../../misc/deployments`, which still lands on the stale copy when read on GitHub and escapes the docs root on docs.gno.land. Not posted here because it is 6039's diff, not this one.
- 6040 gates heavy CI on documentation-only paths. Constraint worth carrying: this PR touches `misc/docs/sidebar.json`, which is outside `docs/**`, and the only job that would have caught a hand-edited sidebar is the `docs` job of `ci-codegen-verify.yml`, whose path filter is `docs/**` plus `**/*.go`. A sidebar-only change already runs no verification today and triggers no docs deploy, since `deploy-docs.yml` filters on `docs/**` alone. 6040 does not modify `ci-codegen-verify.yml`, so it does not make this worse. Not posted, it is a 6040 question.
