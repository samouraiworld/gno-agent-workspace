# Review: PR [#5531](https://github.com/gnolang/gno/pull/5531)
Event: APPROVE

## Body
`release-docker.yml` is the only workflow that computes and passes `BUILD_VERSION`. Every other builder of this Dockerfile now falls to the `ARG BUILD_VERSION=dev` default ([`Dockerfile:17`](https://github.com/gnolang/gno/blob/7199a6789/Dockerfile#L17) · [↗](../../../../../.worktrees/gno-review-5531/Dockerfile#L17)), including [`release-staging.yml`](https://github.com/gnolang/gno/blob/7199a6789/.github/workflows/release-staging.yml#L44-L50) · [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-staging.yml#L44), which pushes the `portalloopd` staging image to ghcr. That image's embedded version drops to `dev`. Pass `BUILD_VERSION` there too, or accept `dev` for staging.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5531-ci-release-build-cache/2-7199a6789/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## Dockerfile:14-16 [↗](../../../../../.worktrees/gno-review-5531/Dockerfile#L14)
Two typos: "referes" should be "refers", and "use in build stages" should be "used". The comment also overstates the benefit. On a version-only change every child stage still rebuilds, because its base image digest changed; the real saving is the `--mount=type=cache` Go build cache, not the layer ordering.

## .github/workflows/release-docker.yml:27 [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-docker.yml#L27)
`actions: write` is not used by `type=registry`; that path needs only `packages: write`. Drop it for least privilege, and re-add it if `type=gha` ever returns.

## .github/workflows/release-docker.yml:91-98 [↗](../../../../../.worktrees/gno-review-5531/.github/workflows/release-docker.yml#L91)
The `type=registry` cache manifests at `ghcr.io/<repo>/cache:<image>` have no cleanup. The tag is overwritten each run so it does not multiply, but `mode=max` exports many blob layers and GHCR does not prune untagged versions, so orphaned blobs accumulate over time. Consider a scheduled prune or a documented retention.
</content>
