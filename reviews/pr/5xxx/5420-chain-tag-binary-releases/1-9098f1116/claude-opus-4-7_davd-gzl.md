# PR #5420: ci: add binary releases for chain/* tags

URL: https://github.com/gnolang/gno/pull/5420
Author: moul | Base: master | Files: 1 | +127 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m] | Commit: `9098f1116` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5420 9098f1116`

**Verdict: APPROVE** — single new workflow, CI green, action SHAs pinned, `persist-credentials: false`, scoped per-job permissions; only soft notes are an absent `CGO_ENABLED=0` (silently relies on default deps being CGO-free), a `workflow_dispatch` input that accepts un-prefixed tags without validation, and a stray unused `actions/checkout` in the release job.

## Summary

Adds [`release-chain-tag.yml`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml) — a tag-triggered (`chain/*`) workflow that cross-compiles `gnoland`, `gnokey`, `gnoweb`, `gno` for four GOOS/GOARCH combos in a matrix, version-stamps via `-X .../tm2/pkg/version.Version=<tag>`, uploads per-platform artifacts, then merges them in a `release` job that ensures a GitHub release exists (`gh release view ... || gh release create --verify-tag`) and uploads binaries plus a combined `CHECKSUMS.txt`. Follows the `release-*` naming convention from [#5423](https://github.com/gnolang/gno/pull/5423). Companion to docker images built by [`release-docker.yml`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-docker.yml) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-docker.yml). Jeronimo's earlier review suggestion (idempotent `gh release view || gh release create`) was integrated at [`release-chain-tag.yml:110-120`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L110-L120) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L110-L120).

## Fix

The repo previously had no chain-tag release pipeline — `release-goreleaser.yml` only handles master push / nightly. This adds a dedicated workflow gated on the `chain/*` tag pattern (or manual dispatch). The matrix job builds with `actions/setup-go@<sha>` + `go build -ldflags ...`, uploads per-platform `dist/` directories as artifacts, and the dependent `release` job downloads them with `merge-multiple: true`, regenerates `CHECKSUMS.txt`, then runs `gh release create --verify-tag` (only if the release doesn't yet exist) and `gh release upload ... --clobber`. The `--verify-tag` flag ensures the workflow can't manufacture a release for a tag that doesn't exist server-side.

## Critical (must fix)

None.

## Warnings (should fix)

- **[CGO_ENABLED not pinned]** [`release-chain-tag.yml:56-76`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L56-L76) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L56-L76) — build relies on the runner's default `CGO_ENABLED` (`1` on Ubuntu and macOS); a future CGO-importing dependency or a runner image change would silently break darwin/amd64 (cross from arm64 macos-latest) and linux/arm64 (cross from amd64 ubuntu-latest) builds.
  <details><summary>details</summary>

  The companion [`.github/goreleaser.yaml:30`](https://github.com/gnolang/gno/blob/9098f1116/.github/goreleaser.yaml#L30) · [↗](../../../../../.worktrees/gno-review-5420/.github/goreleaser.yaml#L30) sets `CGO_ENABLED=0` for every binary precisely because pure-Go cross-compile is the contract. Today `tm2/pkg/crypto/secp256k1` is gated behind a `libsecp256k1` build tag ([`tm2/pkg/crypto/secp256k1/secp256k1_cgo.go:1`](https://github.com/gnolang/gno/blob/9098f1116/tm2/pkg/crypto/secp256k1/secp256k1_cgo.go#L1) · [↗](../../../../../.worktrees/gno-review-5420/tm2/pkg/crypto/secp256k1/secp256k1_cgo.go#L1)), so default builds are CGO-free and CI is green — but that invariant is one merged dependency away from breaking. Fix: add `CGO_ENABLED: 0` to the `env:` block at [`release-chain-tag.yml:57`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L57) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L57) so the workflow declares its contract instead of inheriting it.
  </details>

- **[unvalidated workflow_dispatch input]** [`release-chain-tag.yml:9-12`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L9-L12) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L9-L12) — `inputs.tag` is a free-form string with `required: true` but no pattern; an operator who types `gnoland1.1` (no `chain/` prefix) lands in a broken half-state.
  <details><summary>details</summary>

  `${RAW_TAG#chain/}` at [line 54](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L54) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L54) is a no-op when the prefix is missing, so `TAG=gnoland1.1` flows through to `gh release view "chain/gnoland1.1"` (looks for a release matching the correctly-prefixed name) but the earlier `actions/checkout` at [line 44](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L44) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L44) used `ref: gnoland1.1` (no prefix) — a ref that doesn't exist — and the job fails at checkout. Inverse problem if the user types `chain/chain/gnoland1.1`: the strip removes one prefix, leaving `chain/gnoland1.1` as TAG and `chain/chain/gnoland1.1` as upload target. Fix: at [`release-chain-tag.yml:53`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L53) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L53) add a guard step that errors out unless `RAW_TAG` matches `^chain/[A-Za-z0-9._-]+$`, e.g. `[[ "$RAW_TAG" =~ ^chain/[A-Za-z0-9._-]+$ ]] || { echo "::error::tag must match chain/<version>"; exit 1; }`.
  </details>

## Nits

- [`release-chain-tag.yml:91-93`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L91-L93) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L91-L93) — the `release` job's `actions/checkout` is unused: subsequent steps only call `gh release` (no local git context) and operate on `dist/` populated by `download-artifact`. Removing it shaves a few seconds and cuts the surface area. `gh release create --verify-tag` verifies tag existence server-side, not via local repo.

- [`release-chain-tag.yml:14-15`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L14-L15) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L14-L15) — `concurrency` block is missing `cancel-in-progress: false` for explicitness. Default is `false`, so behavior is correct, but [`deploy-pages.yml:17-19`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/deploy-pages.yml#L17-L19) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/deploy-pages.yml#L17-L19) sets it explicitly — consistency would help future readers know it's intentional (you don't want to cancel a half-uploaded release).

- [`release-chain-tag.yml:7`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L7) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L7) — `chain/*` tag glob does not match `/` per GitHub filter semantics, so `chain/foo/bar` would silently not trigger. Probably intentional, but worth noting in a comment next to the pattern so the next contributor doesn't add a multi-segment tag thinking it'll work.

- [`release-chain-tag.yml:76`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L76) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L76) — `shasum -a 256 ./*` includes any pre-existing files in `dist/`. The `mkdir -p dist` is fresh per job here (no caching, runner is ephemeral), so safe today, but the line would silently include unrelated content if a future step adds files before the loop.

## Missing Tests

- **[no end-to-end dry run]** [`release-chain-tag.yml:1-127`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L1-L127) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L1-L127) — there's no way to validate the workflow without pushing a real `chain/*` tag, which creates a real release. PR body acknowledges this ("I think the way is to merge on a fork and run on a fork"). A `--snapshot`-style mode (analogous to `release-goreleaser.yml`'s `inputs.snapshot`) that skips `gh release upload` and only stashes artifacts would let an operator validate the pipeline end-to-end without burning a production release.
  <details><summary>details</summary>

  Compare with [`release-goreleaser.yml:11-17`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-goreleaser.yml#L11-L17) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-goreleaser.yml#L11-L17) which exposes `snapshot: boolean` and switches `goreleaser` args accordingly. A similar `dry_run: boolean` input here, branching the final two steps (`Ensure release exists` and `Upload release assets`), would make the workflow testable without manufacturing a tag. Defer to follow-up — not a merge blocker.
  </details>

## Suggestions

- [`release-chain-tag.yml:62`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L62) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L62) — `LDFLAGS="-w -s ..."` matches goreleaser, but goreleaser also stamps `gnodev` with `-X github.com/gnolang/gno/gnovm/pkg/gnoenv._GNOROOT=/gnoroot` ([`.github/goreleaser.yaml:77`](https://github.com/gnolang/gno/blob/9098f1116/.github/goreleaser.yaml#L77) · [↗](../../../../../.worktrees/gno-review-5420/.github/goreleaser.yaml#L77)). `gnodev` isn't in this workflow's build list so it's moot today, but if `gnodev` is later added, the docker-image vs. binary `gnodev` will diverge in behavior.

- [`release-chain-tag.yml:116-120`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L116-L120) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L116-L120) — `--generate-notes` is invoked only on first creation; subsequent runs (re-dispatch on the same tag) skip the `gh release create` branch entirely, leaving notes untouched. That's fine for the create-once contract but worth a one-line comment so the operator knows they need to edit notes by hand after the workflow finishes.

- [`release-chain-tag.yml:1-127`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-chain-tag.yml#L1-L127) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-chain-tag.yml#L1-L127) — no signature step. `release-goreleaser.yml` integrates `sigstore/cosign-installer` and `anchore/sbom-action` ([`release-goreleaser.yml:47-48`](https://github.com/gnolang/gno/blob/9098f1116/.github/workflows/release-goreleaser.yml#L47-L48) · [↗](../../../../../.worktrees/gno-review-5420/.github/workflows/release-goreleaser.yml#L47-L48)). Chain-tag releases ship the binaries the network actually runs — they arguably warrant the same supply-chain hygiene (sigstore keyless signing of binaries + checksums). Defer to follow-up.

## Questions for Author

- Should this workflow also produce a `.tar.gz` / `.zip` archive per platform (the goreleaser pipeline does), so consumers get a single download per OS/arch rather than four bare binaries?

- Is the omission of `gnodev` / `gnofaucet` / `gnocontribs` from the binary set intentional? Chain operators typically only need `gnoland` + `gnokey`, but `gnoweb` is included which suggests "tooling people need to interact with the chain" — `gnokms`/`gnobro` might fit the same bucket.
