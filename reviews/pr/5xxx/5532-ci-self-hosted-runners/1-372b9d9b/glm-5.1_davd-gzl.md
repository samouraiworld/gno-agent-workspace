# PR #5532: chore(ci): Introducing self-hosted runners leveraging persistent BuildKit

**URL:** https://github.com/gnolang/gno/pull/5532
**Author:** sw360cab | **Base:** master | **Files:** 6 | **+89 -50**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR routes the Docker release and benchmark workflows to self-hosted ARC runners (`gnolang-self-runner-xlarge`) and connects `docker/setup-buildx-action` to a persistent BuildKit daemon via a remote TCP endpoint (`tcp://buildkit:1234`). The goal is to make BuildKit's `--mount=type=cache` (Go module and build caches) survive across CI runs, since on ephemeral GH runners those caches are discarded after each job.

Key changes across 6 files:

1. **`.dockerignore`** — Adds `.git`, `.gitignore`, `.gitattributes` to prevent `.git` from entering the Docker build context, which would invalidate the layer cache on every commit.

2. **`.github/actionlint.yaml`** — Replaces the old `benchmarks` label with `gnolang-self-runner-large` and `gnolang-self-runner-xlarge` so actionlint recognizes the new runner labels.

3. **`.github/workflows/release-bench-history.yml`** — Switches `runs-on` from `[self-hosted, Linux, X64, benchmarks]` to `gnolang-self-runner-xlarge`. Also removes `@zivkovicmilos` from `alert-comment-cc-users`.

4. **`.github/workflows/release-docker.yml`** — Switches `runs-on` from `ubuntu-latest` to `gnolang-self-runner-xlarge`. Adds `fetch-depth: 0` for full git history. Moves version computation from inside the Dockerfile to a workflow step (`build_version`), passing it as `BUILD_VERSION` env/bake-arg. Configures Buildx with `driver: remote` + `endpoint: tcp://buildkit:1234`. Switches cache from `type=gha` to `type=registry` (per-image OCI cache manifests in ghcr.io) to avoid concurrent matrix job cache corruption.

5. **`Dockerfile`** — Removes the incorrect `RUN go env -w GOMODCACHE=/root/.cache/go-build` (was setting GOMODCACHE to the build cache path instead of the module cache path). Replaces inline `RUN git describe` version computation with `ARG BUILD_VERSION=dev` + `RUN echo`. Unifies all cache mount IDs to `gomodcache` (shared) and `gobuildcache-${TARGETPLATFORM}` (per-arch). Adds `ARG TARGETPLATFORM` to all build stages for multi-arch cache isolation. Minor whitespace alignment in COPY directives.

6. **`misc/deployments/bake/docker-bake.hcl`** — Adds `BUILD_VERSION` variable with default `dev`, forwards it as a build arg in the `common` target. Trailing whitespace cleanup on the `platforms` block.

The design is sound: version computation is moved out of Docker (since `.git` is now excluded from context), the ARG is declared late in the Dockerfile to minimize cache invalidation, and per-platform build cache IDs prevent cross-arch cache collisions.

## Test Results

- **Existing tests:** PASS — All CI checks pass (actionlint, build, lint, test, e2e-test, gno-checks, zizmor). The "Merge Requirements" failure is a bot gating check unrelated to this PR's changes.
- **Edge-case tests:** skipped (CI/infra-only PR; no runtime code changed)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `.github/workflows/release-docker.yml:55` — BuildKit remote TCP endpoint (`tcp://buildkit:1234`) uses unencrypted, unauthenticated TCP. BuildKit supports mTLS (`tcp://buildkit:1234` with client/server certs via `DOCKER_TLS_VERIFY`, `DOCKER_CERT_DIR`). On a shared cluster, any pod that can reach `buildkit:1234` can execute arbitrary builds. If the self-hosted runner network is strictly isolated this may be acceptable, but the PR should explicitly document the threat model or add TLS configuration before merging.
- [ ] `Dockerfile:64-72` — The `build-gnobro` stage still lacks `go mod download -x` before `go build`, unlike every other contrib build stage (`build-gnofaucet`, `build-gnodev`, `build-contribs`, `build-misc` all have it). This means gnobro's first build on a cold cache must download all modules during `go build`, which is slower and produces less predictable caching. Not a regression (same in master) but now more visible since all other stages were unified.

## Nits

- [ ] `Dockerfile:14` — Typo: "referes" should be "refers".
- [ ] `.github/actionlint.yaml:4` — Missing trailing newline (file ends without `\n`).
- [ ] `.github/workflows/release-docker.yml:52` — Comment "remove this if GH runner is used" is vague. Consider specifying the condition more precisely, e.g., "Remove `with:` block when switching back to GitHub-hosted runners; `driver: remote` is incompatible with the default docker-container driver."

## Missing Tests

- [ ] No smoke test verifies that the workflow actually succeeds on `gnolang-self-runner-xlarge` with the remote BuildKit endpoint. The PR's test plan acknowledges this (cold run, cached run, incremental run) but these are manual. Consider adding a `workflow_dispatch` dry-run or a separate CI job that validates the BuildKit connectivity before merging.
- [ ] No test for the `BUILD_VERSION` arg defaulting to `dev` when built locally (without bake/CI). Verify `docker build .` still produces a working image with version "dev".

## Suggestions

- Add `go mod download -x` to the `build-gnobro` stage (between `WORKDIR` and `go build`) for consistency with all other contrib stages and to ensure the shared `gomodcache` is warm for gnobro's own `go.mod`.
- If the BuildKit endpoint will remain plain TCP, add a comment in the workflow noting the assumed network isolation (e.g., `# tcp://buildkit:1234 is acceptable only within the isolated runner network; do not expose externally`). This helps future maintainers understand the security trade-off.
- Consider pinning the BuildKit daemon version (or image tag) in the infra config and documenting it alongside this workflow so that BuildKit upgrades are intentional and tracked.

## Questions for Author

- Is the BuildKit daemon deployed with TLS, or is it intentionally plain TCP? If plain TCP, what network isolation guarantees exist between the runner pod and the BuildKit service?
- The `fetch-depth: 0` checkout brings the full git history (~50k+ commits). Was a shallow clone with sufficient depth (e.g., `fetch-depth: 100`) considered to reduce checkout time while still supporting `git describe` and `git rev-list --count`?
- The removed `@zivkovicmilos` from `alert-comment-cc-users` in `release-bench-history.yml:63` — is this intentional or an unrelated change?

## Verdict

REQUEST CHANGES — The plain-TCP BuildKit endpoint needs explicit security justification or TLS configuration before this ships to production CI. Everything else (cache strategy, BUILD_VERSION refactor, .dockerignore, actionlint labels) is well-designed and correct. Fix the BuildKit auth concern and the gnobro `go mod download` gap, and this is ready to merge.
