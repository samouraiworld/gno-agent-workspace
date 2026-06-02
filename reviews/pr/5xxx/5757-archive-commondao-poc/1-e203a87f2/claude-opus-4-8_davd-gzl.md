# PR #5757: chore(examples): archive PoC of a `commondao` realm

URL: https://github.com/gnolang/gno/pull/5757
Author: jeronimoalbi | Base: master | Files: 77 | +0 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: e203a87f2 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5757 e203a87f2`

**Verdict: REQUEST CHANGES** — the directory moved from `r/nt/commondao/v0` to `r/archive/commondao/v0` but the realm's declared path did not: `gnomod.toml` still says `module = "gno.land/r/nt/commondao/v0"`, so the realm keeps its old on-chain identity and the archive is filesystem-cosmetic only. Sibling archived packages (`p/archive/{acl,bank,rat}`) moved in the same effort all have `module` matching their new dir; this one is the outlier.

## Summary
Pure rename PR: 77 files move `examples/gno.land/r/{nt => archive}/commondao/v0/` with zero content changes (every hunk is R100, +0 -0). The intent is to archive a PoC realm. The defect: a realm's deploy/genesis path comes from the `gnomod.toml` `module` field, not its directory (`gnovm/pkg/packages/readpkglist.go:51` passes `mod.Module` to `ReadMemPackage`). The PR moved the files but left `module = "gno.land/r/nt/commondao/v0"` plus ~60 filetests still pinned to the old path. Result: the realm sits in `r/archive/` on disk yet still deploys at `gno.land/r/nt/commondao/v0`. Tests pass precisely because every self-reference is consistently stale, so nothing inside the package notices.

```
on disk:           examples/gno.land/r/archive/commondao/v0/   <- moved
gnomod.toml module: gno.land/r/nt/commondao/v0                  <- NOT moved -> deploy path
filetests PKGPATH:  gno.land/r/nt/commondao/v0/filetests/...    <- NOT moved
rendered links:     [Foo](/r/nt/commondao/v0:2)                 <- NOT moved (derives from module)
```

## Glossary
- module field — the `module = "..."` line in `gnomod.toml`; the canonical package/realm path used at genesis deploy time, independent of directory location.
- PKGPATH directive — `// PKGPATH: ...` header in a `_filetest.gno`; sets the pkgpath the filetest VM runs the file under.
- `p/nt/commondao/v0` vs `r/nt/commondao/v0` — the realm (`r/...`) imports a stateless package (`p/...`) of the same name; only the `r/...` realm is in scope for this PR.

## Fix
A directory move alone does not re-home a Gno realm. To actually archive at `gno.land/r/archive/commondao/v0`, the PR must also rewrite the self-path in `gnomod.toml` (`r/nt` -> `r/archive`) and in every filetest `// PKGPATH:` header and `import "gno.land/r/nt/commondao/v0"` line. Leave the `gno.land/p/nt/commondao/v0` dependency imports (28 occurrences) alone — that package is not moved by this PR. The sibling archive moves on `master` (`p/archive/{acl,bank,rat}/gnomod.toml` all carry matching `module = "gno.land/p/archive/..."`) are the pattern to follow.

## Benchmarks / Numbers
| Reference | Path | Count in moved dir | Correct? |
|-----------|------|--------------------|----------|
| Self (realm) | `gno.land/r/nt/commondao/v0` | 124 | stale — should be `r/archive/...` |
| Dependency (pkg) | `gno.land/p/nt/commondao/v0` | 28 | correct — `p/...` not moved |

## Critical (must fix)
- **[move is cosmetic — realm still deploys at the old path]** [`examples/gno.land/r/archive/commondao/v0/gnomod.toml:1`](https://github.com/gnolang/gno/blob/e203a87f2/examples/gno.land/r/archive/commondao/v0/gnomod.toml#L1) · [↗](../../../../../.worktrees/gno-review-5757/examples/gno.land/r/archive/commondao/v0/gnomod.toml#L1) — `module` field left as `gno.land/r/nt/commondao/v0` after the directory moved to `r/archive/`.
  <details><summary>details</summary>

  The genesis/examples loader derives a package's path from the `gnomod.toml` `module` field, not its directory: [`gnovm/pkg/packages/readpkglist.go:51`](https://github.com/gnolang/gno/blob/e203a87f2/gnovm/pkg/packages/readpkglist.go#L51) · [↗](../../../../../.worktrees/gno-review-5757/gnovm/pkg/packages/readpkglist.go#L51) calls `gnolang.ReadMemPackage(path, mod.Module, mptype)`, and [`gnovm/pkg/packages/load.go:298`](https://github.com/gnolang/gno/blob/e203a87f2/gnovm/pkg/packages/load.go#L298) · [↗](../../../../../.worktrees/gno-review-5757/gnovm/pkg/packages/load.go#L298) sets `pkgPath := gm.Module`. So this realm, despite living in `r/archive/`, still registers on-chain as `gno.land/r/nt/commondao/v0`. The "archive" is purely a filesystem reshuffle; the realm's identity, its render links (`[Foo](/r/nt/commondao/v0:2)` in test output), and any future importer's path are unchanged. Every sibling archived in the same `master` effort (`p/archive/acl`, `p/archive/bank`, `p/archive/rat`) has `module` matching its new directory, so this is an inconsistency, not a new convention. Fix: change line 1 to `module = "gno.land/r/archive/commondao/v0"` (assuming the goal is to re-home the realm; if the goal is genuinely "keep the path, just relocate the source," say so in the PR body, because nothing else in the tree expects that).
  </details>

- **[~60 filetests pin the old self-path]** [`examples/gno.land/r/archive/commondao/v0/filetests/z_1_a_filetest.gno:1`](https://github.com/gnolang/gno/blob/e203a87f2/examples/gno.land/r/archive/commondao/v0/filetests/z_1_a_filetest.gno#L1) · [↗](../../../../../.worktrees/gno-review-5757/examples/gno.land/r/archive/commondao/v0/filetests/z_1_a_filetest.gno#L1) — `// PKGPATH: gno.land/r/nt/commondao/v0/...` and `import "gno.land/r/nt/commondao/v0"` throughout the `filetests/` dir.
  <details><summary>details</summary>

  Same root cause as the `gnomod.toml` finding. The filetest `// PKGPATH:` headers and the `import "gno.land/r/nt/commondao/v0"` lines all still name the old realm path. They pass today only because they are consistent with the (also-stale) `module` field — the package imports itself under the old name and resolves fine. Once `module` is corrected, these must be updated in lockstep or the filetests break. Scope: 124 occurrences of `gno.land/r/nt/commondao/v0` across the moved directory (gnomod + filetests + commented render links). Leave the 28 `gno.land/p/nt/commondao/v0` dependency imports untouched. Fix: sed the realm self-path `r/nt/commondao/v0` -> `r/archive/commondao/v0` across the directory, excluding the `p/nt/...` dependency import.
  </details>

## Warnings (should fix)
- **[dependency package not archived alongside the realm]** [`examples/gno.land/r/archive/commondao/v0/genesis.gno:4`](https://github.com/gnolang/gno/blob/e203a87f2/examples/gno.land/r/archive/commondao/v0/genesis.gno#L4) · [↗](../../../../../.worktrees/gno-review-5757/examples/gno.land/r/archive/commondao/v0/genesis.gno#L4) — the realm depends on `gno.land/p/nt/commondao/v0`, which stays live under `p/nt/`.
  <details><summary>details</summary>

  This PR archives only the `r/` realm; the `p/nt/commondao/v0` package it depends on remains in the active tree (`examples/gno.land/p/nt/commondao/v0/` still present, 28 imports from the moved realm). That may be intentional (the package is reusable; only the realm PoC is being shelved), but the PR body is empty so a reviewer cannot tell. If the package is also meant to be deprecated, it needs its own move; if not, confirm the split is deliberate. Not blocking on its own, but it compounds the path confusion above. Fix: state the intent in the PR description.
  </details>

## Nits
- PR body is empty. An archive move with a stale module path is exactly the kind of change where one sentence of intent ("relocate source, keep on-chain path" vs "fully re-home to `r/archive`") would prevent this whole ambiguity.

## Missing Tests
- None. No behavior changes; existing filetests/unit tests cover the package and pass unchanged (`gno test ./gno.land/r/archive/commondao/v0/` => ok, 12.00s). The gap here is correctness of the move, not test coverage.

## Suggestions
- Run `gno mod tidy --recursive` and `gno fmt` from `examples/` after correcting the path so any path-derived metadata stays consistent. (`examples/Makefile:72-74` defines `tidy`; `:67-70` defines `fmt`.)

## Questions for Author
- Is the goal to keep the realm's on-chain path at `gno.land/r/nt/commondao/v0` (relocate source only), or to re-home it to `gno.land/r/archive/commondao/v0`? The `gnomod.toml` says the former, the directory implies the latter, and the sibling `p/archive/*` moves did the latter.
- Is leaving the `gno.land/p/nt/commondao/v0` dependency package in the active tree intentional, or should it be archived too?
