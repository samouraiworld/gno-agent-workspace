# PR [#5531](https://github.com/gnolang/gno/pull/5531): chore(ci): fix layer + mount caching in release build pipeline

URL: https://github.com/gnolang/gno/pull/5531
Author: sw360cab | Base: master | Files: 4 | +80 -46
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 7199a6789 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5531 7199a6789`

Round 2. Head advanced past the round-1 sha (3c8f3dbab, GC'd) via a master merge by moul; reviewed the full PR diff against `merge-base(origin/master, HEAD)` = d1489dc5c. PR content unchanged since round 1: no round-1 finding was fixed, all carry forward. moul (maintainer) APPROVED on 2026-07-06. One new finding added (BUILD_VERSION default leaks into non-release-docker builders).

**TL;DR:** The `release / docker` workflow recompiled every image from scratch on every run, even with no code change. This PR restructures the Dockerfile and the workflow so unchanged source hits the Docker layer cache and Go's build cache, and only modified files force a recompile.

**Verdict: APPROVE** — five real caching failures fixed correctly; only minor items remain: a published staging image now embeds version `dev`, one overstated comment, a stray permission, and cosmetic inconsistencies.

## Summary
Five independent causes broke caching in the 7-image release matrix: `.git` in the build context invalidated `COPY . ./` on every commit; `go env -w GOMODCACHE=/root/.cache/go-build` aimed the module cache at the build-cache dir, making the `/go/pkg/mod` mount a no-op; per-subproject cache IDs (`faucet-modcache`, `gnodev-modcache`, …) prevented sharing the large `tm2/`+`gnovm/` compile across binaries; unscoped `type=gha` let the 7 concurrent matrix jobs race on GHA's global blob namespace; and `git describe` ran inside the build, forcing `.git` into context. The fix excludes `.git` via `.dockerignore`, computes `BUILD_VERSION` in the workflow and passes it as a build arg (default `dev`), removes the `GOMODCACHE` line, unifies to `id=gomodcache` + `id=gobuildcache-${TARGETPLATFORM}`, and switches the layer cache to per-image `type=registry` OCI manifests at `ghcr.io/<repo>/cache:<image>`.

## Fix
Version computation moves out of the image: `setup-gnocore` now does `ARG BUILD_VERSION=dev` then `echo "${BUILD_VERSION}" > /gnoroot/build_version`, declared after `COPY . ./` so a version-only change leaves the source-copy and mod-download layers cached (`Dockerfile:13-18`). The workflow adds `fetch-depth: 0` and a `Compute build version` step running `git describe --tags --exact-match` with the branch-count fallback, forwarded to bake via `env: BUILD_VERSION` and the new `common` arg (`.github/workflows/release-docker.yml:34-41`, `.github/workflows/release-docker.yml:81-82`, `misc/deployments/bake/docker-bake.hcl:9-12`, `misc/deployments/bake/docker-bake.hcl:60-62`). The build arg propagates to child stages through the `/gnoroot/build_version` file, not a re-declared ARG, so no child stage needs `ARG BUILD_VERSION`.

## Verification
- The Dockerfile builds green in PR CI: `ci-val-scenarios.yml` builds `target: all` and `target: gnocontribs` from this Dockerfile on head 7199a6789 (check `build docker images`, 3m19s). It passes no `BUILD_VERSION`, so this also exercises the `ARG BUILD_VERSION=dev` default path.
- Ran the workflow's build-version shell against a detached, tag-less HEAD in the worktree: emits `HEAD.3220+7199a6789`. Confirms the `git rev-parse --abbrev-ref HEAD` fallback prints the literal `HEAD` when the ref is not a branch (see Open questions).
- The caching improvements themselves cannot be verified in this PR's CI: `release-docker.yml` only runs on `chain/*` push/tag and `workflow_dispatch`, never on pull requests.

## Critical (must fix)
None.

## Warnings (should fix)
- **[published staging image loses its version string]** `.github/workflows/release-staging.yml:44-50` — After this PR, the pushed `portalloopd` staging image embeds version `dev`.
  <details><summary>details</summary>

  `release-docker.yml` is the only builder that computes and passes `BUILD_VERSION`. Every other consumer of this Dockerfile now builds with the `ARG BUILD_VERSION=dev` default: `release-staging.yml` builds and pushes `ghcr.io/<repo>/portalloopd` on master push ([`.github/workflows/release-staging.yml:44-50`](https://github.com/gnolang/gno/blob/7199a6789/.github/workflows/release-staging.yml#L44-L50) · [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-staging.yml#L44)), and rebuilds `gnoland:master` for the compose test ([`.github/workflows/release-staging.yml:85`](https://github.com/gnolang/gno/blob/7199a6789/.github/workflows/release-staging.yml#L85) · [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-staging.yml#L85)); `ci-val-scenarios.yml` builds `all`/`gnocontribs` for scenario tests. Before this PR, the in-Dockerfile `git describe` gave these a git-derived string; the removed `RUN … > build_version` plus `ARG BUILD_VERSION=dev` ([`Dockerfile:17-18`](https://github.com/gnolang/gno/blob/7199a6789/Dockerfile#L17-L18) · [↗](../../../../../.worktrees/gno-review-5531/Dockerfile#L17)) now yields `dev`. Test images (`gnoland:master`, scenario builds) don't care, but the published `portalloopd` does. Note the prior release-staging value was already weak (`release-staging.yml` checks out shallow, no `fetch-depth: 0`, so `git describe` failed and the fallback produced `master.1+<sha>`), so this is degradation of an already-degenerate string, not loss of a semver. Fix: pass `BUILD_VERSION` in `release-staging.yml` too, or consciously accept `dev` for staging.
  </details>

## Nits
- `Dockerfile:14-16` — Typo and overstated benefit in one comment. "referes" → "refers", "use in build stages" → "used in build stages". The claim that declaring the ARG late leaves "expensive layers above" cached is true only for `setup-gnocore`'s own layers; every child stage (`FROM setup-gnocore`) still rebuilds fully on a version-only change because its base image digest changed, so the real saving is the `--mount=type=cache` Go build cache, not the Docker layer ordering. [`Dockerfile:14-16`](https://github.com/gnolang/gno/blob/7199a6789/Dockerfile#L14-L16) · [↗](../../../../../.worktrees/gno-review-5531/Dockerfile#L14)
- `.github/workflows/release-docker.yml:27` — `actions: write` is unused by `type=registry` (comment says it's kept for a possible `type=gha` return). Least-privilege would drop it; re-adding is trivial. [`.github/workflows/release-docker.yml:27`](https://github.com/gnolang/gno/blob/7199a6789/.github/workflows/release-docker.yml#L27) · [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-docker.yml#L27)
- `Dockerfile:11` — `setup-gnocore`'s `go mod download` build-cache mount is `id=gobuildcache`, missing the `-${TARGETPLATFORM}` suffix every other stage uses; harmless (`go mod download` writes no build-cache entries) but inconsistent. `setup-gnocore` would need its own `ARG TARGETPLATFORM` to match. [`Dockerfile:11`](https://github.com/gnolang/gno/blob/7199a6789/Dockerfile#L11) · [↗](../../../../../.worktrees/gno-review-5531/Dockerfile#L11)

## Missing Tests
None. CI-pipeline and Dockerfile changes have no unit-testable surface; the caching effect only shows on a live `chain/*` build, which the PR body documents as a manual test plan. Appropriate for the change type.

## Suggestions
- `Dockerfile:64-72` — `build-gnobro` has no explicit `go mod download -x` step, unlike the other contrib stages (`build-gnofaucet`, `build-gnodev`, `build-contribs`, `build-misc`). `go build` downloads modules inline into the shared mount, so it works, but the pattern is inconsistent and the layer boundary is less clear. Pre-existing; this PR only re-touched the mount IDs. [`Dockerfile:64-72`](https://github.com/gnolang/gno/blob/7199a6789/Dockerfile#L64-L72) · [↗](../../../../../.worktrees/gno-review-5531/Dockerfile#L64)
- `.github/workflows/release-docker.yml:91-98` — The per-image `type=registry` cache manifests at `ghcr.io/<repo>/cache:<image>` have no garbage collection. The tag is mutable (overwritten each run), so the manifest doesn't multiply, but `mode=max` exports many blob layers and GHCR does not auto-prune untagged versions, so orphaned blobs accumulate over time. Consider a scheduled prune or document the retention expectation. [`.github/workflows/release-docker.yml:91-98`](https://github.com/gnolang/gno/blob/7199a6789/.github/workflows/release-docker.yml#L91-L98) · [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-docker.yml#L91)

## Open questions
- BUILD_VERSION fallback on a non-tag `workflow_dispatch` ref. The step at [`.github/workflows/release-docker.yml:36-41`](https://github.com/gnolang/gno/blob/7199a6789/.github/workflows/release-docker.yml#L36-L41) · [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-docker.yml#L36) falls to `git rev-parse --abbrev-ref HEAD` when `git describe --exact-match` fails. On a detached checkout that prints the literal `HEAD`, yielding `HEAD.<count>+<sha>` (verified locally: `HEAD.3220+7199a6789`). The `tag` dispatch input is free-form; a non-tag value (branch, SHA) produces this degenerate string. Same fallback the old in-Dockerfile logic had, so not a regression; not posted (edge case, doesn't change the verdict).
- `type=gha,scope=${{ matrix.image }}` was a lighter alternative to `type=registry` for the matrix race (scoping per image sidesteps the shared blob namespace); the PR body already justifies `type=registry` by the GHA 10 GB repo cap. Not posted (design choice, author reasoned it).
</content>
</invoke>
