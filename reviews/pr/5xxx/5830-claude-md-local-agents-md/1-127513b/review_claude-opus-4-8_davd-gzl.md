# PR #5830: chore: let CLAUDE.md stay local

URL: https://github.com/gnolang/gno/pull/5830
Author: moul | Base: master | Files: 3 | +22 -29
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `127513b` (stale — +7 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5830 127513b`

**TL;DR:** Stops tracking the repo's `CLAUDE.md`, folds its content into the already-tracked `AGENTS.md`, and gitignores `CLAUDE.md` so each contributor can keep a local Claude-specific file. Pure docs/tooling, no code.

**Verdict: APPROVE** — the migration is content-complete: every rule from the deleted `CLAUDE.md` is present in `AGENTS.md`, and both docs the security rules point to resolve. One optional nit: a cross-check `grep` command was dropped from the payment-guard rule.

## Summary

`AGENTS.md` becomes the single tracked source of truth for agent guidance; `CLAUDE.md` is removed and added to `.gitignore` (under "Editor Leftovers", alongside `.vscode`/`.idea`). The five `CLAUDE.md` topic groups are redistributed into `AGENTS.md`'s existing structure rather than appended verbatim, which reads better: verification/before-after rules become a new `### Verification Rules` under `## Build & Test`, the PR-description rules join `## Conventions`, and the interrealm + payment-guard rules become a new `## Gno Security Semantics`.

## Migration map

| `CLAUDE.md` rule (removed) | Lands in `AGENTS.md` | Status |
|---|---|---|
| Gas/alloc/GC verification commands | `### Verification Rules` ([L52-L62](https://github.com/gnolang/gno/blob/127513b/AGENTS.md#L52-L62) · [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L52)) | kept (dropped the "(the txtar testdata suite)" aside) |
| `/simplify` on non-trivial work | `### Verification Rules` | kept |
| Before/after metric rules (2) | `### Verification Rules` | kept |
| PR-description completeness (2) | `## Conventions` ([L86-L87](https://github.com/gnolang/gno/blob/127513b/AGENTS.md#L86-L87) · [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L86)) | kept |
| interrealm read-first + `PreviousRealm()` bug | `## Gno Security Semantics` ([L97-L98](https://github.com/gnolang/gno/blob/127513b/AGENTS.md#L97-L98) · [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L97)) | kept |
| `IsUserCall()` vs `IsUser()` payment guard | `## Gno Security Semantics` ([L99](https://github.com/gnolang/gno/blob/127513b/AGENTS.md#L99) · [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L99)) | kept (docs link intact) |
| Flag existing `IsUser()`+`OriginSend()` | `## Gno Security Semantics` ([L100](https://github.com/gnolang/gno/blob/127513b/AGENTS.md#L100) · [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L100)) | kept, but the concrete `grep` cross-check command was dropped (see Nit) |

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`AGENTS.md:100`](https://github.com/gnolang/gno/blob/127513b/AGENTS.md#L100) · [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L100) — the payment-guard "flag existing usage" rule lost its actionable cross-check command. `CLAUDE.md` had ``grep -rn "IsUser()" examples/ | xargs ...`` plus "it's the same bug"; the migrated line is "flag it and cross-check nearby `OriginSend` usage", which tells an agent what to look for but not how. Optional: restore the grep so the rule stays runnable. Author's call given `AGENTS.md` is the public, terser file.

- [`AGENTS.md:97`](https://github.com/gnolang/gno/blob/127513b/AGENTS.md#L97) · [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L97) — bullet 1 lists `/e/` as a place you "write or review ... code", but ephemeral `/e/` packages are user-submitted `maketx run` scripts, not stored on-chain and never durable repo code (there is no `examples/gno.land/e/`). `/e/` is the *caller* the model defends against (it returns `IsUser() == true`, the whole basis of the `IsUserCall()` bullet two lines down), not an authoring location. Optional: drop `/e/` from this bullet's list and keep it where it's load-bearing. Pre-existing from `CLAUDE.md`; surfaced because it now lives in the tracked source of truth.

## Missing Tests

None (docs).

## Suggestions

None.

## Open questions

- A fresh clone now ships no `CLAUDE.md`; agents that auto-load `CLAUDE.md` rather than `AGENTS.md` get the project guidance only if the contributor creates a local file (e.g. symlink or `@AGENTS.md` import). This is the PR's explicit intent, not a defect; noting only because it's the one behavioral change behind a docs diff. Not posted.
- `.gitignore` adds `CLAUDE.md` unanchored, so it matches at any depth, not just the repo root. Consistent with the `.vscode`/`.idea` entries it sits beside, so likely intended; `/CLAUDE.md` would scope it to root if only the root file is meant to be local. Not posted.
