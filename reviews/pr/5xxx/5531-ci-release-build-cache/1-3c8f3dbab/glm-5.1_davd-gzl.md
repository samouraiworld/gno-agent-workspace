# PR #5531: chore(ci): fix layer + mount caching in release build pipeline

**URL:** https://github.com/gnolang/gno/pull/5531
**Author:** sw360cab | **Base:** master | **Files:** 4 | **+80 -46**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR fixes five root causes that prevented Docker layer and mount caching from working in the release build pipeline (`release / docker` workflow):

1. **`.git` in build context** — `.git/` contents change every commit, invalidating `COPY . ./` and all downstream layers. Fix: exclude `.git`/`.gitignore`/`.gitattributes` via `.dockerignore`, compute `BUILD_VERSION` externally in the workflow, and pass it as `ARG BUILD_VERSION=dev` after `COPY . ./` so version-only changes don't invalidate expensive earlier layers.

2. **`GOMODCACHE` misconfiguration** — `go env -w GOMODCACHE=/root/.cache/go-build` pointed the module cache at the build cache directory, making the `--mount=type=cache,target=/go/pkg/mod` mount a no-op. Fix: removed the line; Go defaults restore `GOMODCACHE=/go/pkg/mod`.

3. **Per-subproject cache IDs** — Each build stage used isolated IDs (`faucet-modcache`, `gnodev-modcache`, etc.), forcing redundant downloads and compilations. Fix: unified to shared `id=gomodcache` (mod cache) and `id=gobuildcache-${TARGETPLATFORM}` (build cache with arch suffix).

4. **`type=gha` cache races** — GHA cache blob entries are globally scoped; 7 concurrent matrix jobs corrupt each other's cache. Fix: switched to `type=registry` with per-image OCI cache manifests at `ghcr.io/<repo>/cache:<image>`.

5. **BUILD_VERSION moved out of Dockerfile** — `git describe` no longer runs inside the build (requires `.git/`). Instead, the workflow computes it externally and passes it as a build arg. Default `dev` makes non-CI builds obviously distinguishable.

Files changed: `.dockerignore`, `.github/workflows/release-docker.yml`, `Dockerfile`, `misc/deployments/bake/docker-bake.hcl`.

## Test Results

- **Existing tests:** PASS — All CI checks pass (build, lint, test, actionlint, zizmor). The only "fail" is from the merge-requirements bot (manual checklist), not a real test failure.
- **Edge-case tests:** skipped — CI pipeline changes cannot be unit-tested locally; validation requires triggering the workflow itself.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `Dockerfile:15-16` — The comment claims "it is declared late so that a version change (new tag / commit) only invalidates this layer and below excluding expensive layers above." This is true **within the `setup-gnocore` stage**, but misleading about the broader impact: when `setup-gnocore`'s image hash changes (even just the `build_version` file), **every child stage** (`FROM setup-gnocore`) loses its layer cache too, because the base image changed. The real speedup for child-stage rebuilds comes from the `--mount=type=cache` Go build cache, not from Docker layer ordering. The comment should clarify this so future editors don't overestimate the layer-cache benefit.

- [ ] `.github/workflows/release-docker.yml:96-97` — `type=registry` cache manifests are pushed to `ghcr.io/<repo>/cache:<image>` with no cleanup or garbage-collection mechanism. Over time, old cache manifests accumulate in GHCR. Consider adding a scheduled workflow or `docker buildx imagetools` pruning step to GC stale cache manifests, or document the expected retention.

## Nits

- [ ] `Dockerfile:14` — Typo: "referes" should be "refers".

- [ ] `Dockerfile:11` — The `--mount=type=cache,target=/root/.cache/go-build,id=gobuildcache` in `setup-gnocore`'s `go mod download` step uses a different cache ID than the child stages (`id=gobuildcache-${TARGETPLATFORM}`). This is harmless — `go mod download` doesn't produce meaningful build-cache entries — but the mismatch is confusing. Consider either removing the build-cache mount from this step entirely, or matching the `${TARGETPLATFORM}` suffix convention for consistency.

- [ ] `.github/workflows/release-docker.yml:27` — The `actions: write` permission is now unused (comment says "kept in case type=gha is re-enabled"). Fine for forward-compat, but consider removing it to follow least-privilege. Re-adding it is trivial if type=gha is ever restored.

## Missing Tests

- CI pipeline changes are inherently untestable locally. The PR body includes a test plan that requires manual workflow triggering. This is appropriate for the change type. No automated tests expected.

## Suggestions

- `Dockerfile:67-72` — The `build-gnobro` stage lacks a `go mod download -x` step before `go build`, unlike `build-gnofaucet`, `build-gnodev`, `build-contribs`, and `build-misc`. While `go build` will download missing modules inline (into the shared mount cache), an explicit `go mod download -x` step would make the pattern consistent across all contrib stages and produce clearer layer cache boundaries. This is a pre-existing issue, not introduced by this PR.

## Questions for Author

- The `BUILD_VERSION` fallback when no tag matches produces strings like `master.12345+3c8f3db`. On `workflow_dispatch` with a tag input, the checkout uses `ref: ${{ inputs.tag }}`, and `git describe --tags --exact-match` should match. But on `push` triggers for `chain/*` branches (non-tag), the fallback path always fires. Is this the intended version string for branch-pushed images, or should those also embed a semver-like string?

## Verdict

APPROVE — The changes are well-reasoned and correctly address five distinct caching failures. The Dockerfile structure is cleaner, cache sharing is safe by construction (Go caches are content-addressed), and the `type=registry` switch correctly sidesteps GHA cache blob races. The two warnings (misleading comment about ARG layer-cache scope, and lack of cache manifest GC) are worth addressing but not blocking.
