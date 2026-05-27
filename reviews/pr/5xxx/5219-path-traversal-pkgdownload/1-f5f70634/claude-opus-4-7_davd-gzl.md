# PR #5219: fix: prevent path traversal in `pkgdownload.Download` and `MemPackage.WriteTo`

URL: https://github.com/gnolang/gno/pull/5219
Author: davd-gzl | Base: master | Files: 4 | +186 -0
Reviewed by: davd-gzl (AI agent: claude-opus-4-7) | Model: claude-opus-4-7 | Commit: `f5f70634` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5219 f5f70634`

Verdict: APPROVE — small, well-targeted defense-in-depth fix for HackenProof NEWTENDG-143; `filepath.IsLocal` is the canonical Go stdlib guard for this exact class, upfront-validation loop prevents partial writes, tests cover both the standalone-malicious and mixed-with-legit cases, CI green, prior approval from @notJoon. Self-review disclosure: PR authored by the same human running this sweep; review performed by an AI agent.

## Summary

Two file-writing entry points — [`pkgdownload.Download`](https://github.com/gnolang/gno/blob/f5f70634/gnovm/pkg/packages/pkgdownload/pkgdownload.go#L12) · [↗](../../../../../.worktrees/gno-review-5219/gnovm/pkg/packages/pkgdownload/pkgdownload.go#L12) (used by `gno mod download` via `modcache.go:65`) and [`MemPackage.WriteTo`](https://github.com/gnolang/gno/blob/f5f70634/tm2/pkg/std/memfile.go#L239) · [↗](../../../../../.worktrees/gno-review-5219/tm2/pkg/std/memfile.go#L239) (used by `gno lint` package materialization) — accepted file names from a remote `PackageFetcher` and joined them straight into a destination with `filepath.Join(dst, file.Name)`. A name like `../ufmt/ufmt.gno` from a compromised RPC would resolve outside `dst`, enabling cross-package poisoning of the modcache or arbitrary writes wherever `WriteTo`'s caller points. The fix prepends a validation pass that rejects any entry where [`filepath.IsLocal`](https://pkg.go.dev/path/filepath#IsLocal) is false; only after every name passes does the loop call `os.WriteFile`. Two-pass shape is what the unresolved [`mvallenet` thread](https://github.com/gnolang/gno/pull/5219#discussion_r2089840728-style) flagged — addressed in commit `4972798c`. Final refactor to `IsLocal` came from [@notJoon's suggestion](https://github.com/gnolang/gno/pull/5219#pullrequestreview), now resolved.

## Fix

Before: each entry was joined-and-written in one loop; a single malicious `..` name landed outside `dst`. After: a separate validation pass over `files` (or `mpkg.Files`) returns early if any `IsLocal` check fails, then the write loop proceeds. The load-bearing guarantee comes from `filepath.IsLocal`'s contract: "If IsLocal(path) returns true, then Join(base, path) will always produce a path contained within base." Two-pass means no legitimate file is written when validation fails on a later entry — see [`pkgdownload.go:22-26`](https://github.com/gnolang/gno/blob/f5f70634/gnovm/pkg/packages/pkgdownload/pkgdownload.go#L22-L26) · [↗](../../../../../.worktrees/gno-review-5219/gnovm/pkg/packages/pkgdownload/pkgdownload.go#L22-L26) and [`memfile.go:240-244`](https://github.com/gnolang/gno/blob/f5f70634/tm2/pkg/std/memfile.go#L240-L244) · [↗](../../../../../.worktrees/gno-review-5219/tm2/pkg/std/memfile.go#L240-L244).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`pkgdownload.go:18-20`](https://github.com/gnolang/gno/blob/f5f70634/gnovm/pkg/packages/pkgdownload/pkgdownload.go#L18-L20) · [↗](../../../../../.worktrees/gno-review-5219/gnovm/pkg/packages/pkgdownload/pkgdownload.go#L18-L20) — `os.MkdirAll(dst, ...)` runs before the validation loop, so a malicious payload still creates the destination directory before the error returns. Idempotent and not a security issue (modcache dirs are precomputed by `modcache.go:64` and would be created on the next legitimate download anyway), but moving the `MkdirAll` below validation costs nothing and tightens the "no side effects on rejected input" contract that the two-pass refactor already commits to.

## Missing Tests

- [`memfile.go:241`](https://github.com/gnolang/gno/blob/f5f70634/tm2/pkg/std/memfile.go#L241) · [↗](../../../../../.worktrees/gno-review-5219/tm2/pkg/std/memfile.go#L241) — no explicit test for an absolute-path name (e.g. `Name: "/etc/passwd"`), only `../` traversal. `filepath.IsLocal` rejects both, so behavior is covered transitively, but a one-line table entry would document the broader threat surface (absolute paths are the other half of zipslip-style attacks). Same for `pkgdownload_test.go`. Optional.

## Suggestions

- [`memfile.go:25`](https://github.com/gnolang/gno/blob/f5f70634/tm2/pkg/std/memfile.go#L25) · [↗](../../../../../.worktrees/gno-review-5219/tm2/pkg/std/memfile.go#L25) — `MemFile.ValidateBasic()`'s `reFileName` already rejects `..` and `/`, but neither `Download` nor `WriteTo` calls `ValidateBasic` (verified by grep — only `MemPackage.ValidateBasic` self-calls it). Worth a short comment on the new `IsLocal` check pointing this out, or a follow-up to wire `ValidateBasic` into these write paths. Without it, the regex and `IsLocal` are two independent gates that need to stay aligned — a future relaxation of one could re-open the hole.

## Questions for Author

- Per AGENTS.md "Every non-trivial AI-assisted PR must include an ADR" — was this PR AI-assisted? If yes, a small ADR under `gnovm/adr/pr5219_path_traversal.md` would document the threat model (compromised RPC `PackageFetcher`) and the choice of `filepath.IsLocal` over `MemFile.ValidateBasic` wiring. If not, ignore.
