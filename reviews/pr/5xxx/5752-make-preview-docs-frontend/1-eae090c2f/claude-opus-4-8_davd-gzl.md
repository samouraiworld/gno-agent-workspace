# PR #5752: docs: add `make preview` target for the docs.gno.land frontend

URL: https://github.com/gnolang/gno/pull/5752
Author: davd-gzl | Base: master | Files: 3 | +118 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: eae090c2f (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5752 eae090c2f`

**Verdict: APPROVE** — dev-only tooling, additive, no production/CI/runtime surface; the rsync `--delete` blast radius is correctly fenced and the path assumptions match the frontend's own wiring. Only nits remain (named-`.nvmrc` brittleness, no corepack preflight).

## Summary
Adds a `preview` Make target in `docs/Makefile` backed by a new `misc/docs/preview.sh` (109 lines) that boots the [docs.gno.land](https://github.com/gnolang/docs.gno.land) Docusaurus frontend against the local working-tree `docs/`. The frontend lives in a separate repo; this repo holds only Markdown. The script clones the frontend into `docs/.preview/`, rsyncs local `docs/` into it, regenerates `sidebar.json` from the local README via `indexparser`, and runs the dev server on the Node version pinned in the frontend's `.nvmrc` (via `fnm` when present). `docs/.preview/` is gitignored. Nothing here runs in CI or production: it is a local authoring convenience.

## Glossary
- `indexparser` — `misc/docs/tools/indexparser`, builds the Docusaurus sidebar JSON from the docs `README.md` index. Writes to stdout.
- `sync_docs` — the script's rsync helper that mirrors `docs/` into the clone with `--delete`.
- `DEST` — `docs/.preview/docs.gno.land/docs`, the copy of the docs the frontend actually renders.

## Fix
Before: previewing a docs change locally meant manually cloning the frontend, copying docs in, regenerating the sidebar, and matching the Node version by hand. After: `make -C docs preview` does all of it and hot-reloads edits on a 1s rsync poll. The load-bearing safety gate is a two-part guard before any `rsync --delete`: `go.mod` must contain `module github.com/gnolang/gno` ([`misc/docs/preview.sh:42`](https://github.com/gnolang/gno/blob/eae090c2f/misc/docs/preview.sh#L42) · [↗](../../../../../.worktrees/gno-review-5752/misc/docs/preview.sh#L42)) and `$DOCS_DIR` must exist and be named `docs/` ([`misc/docs/preview.sh:53`](https://github.com/gnolang/gno/blob/eae090c2f/misc/docs/preview.sh#L53) · [↗](../../../../../.worktrees/gno-review-5752/misc/docs/preview.sh#L53)). `--delete` only ever prunes the throwaway clone under `$DEST`, never the source `docs/`.

## Verification

Structural assumptions checked against the live frontend repo `gnolang/docs.gno.land` at `main`:

- `.nvmrc` exists at frontend root and is `v20.9.0` — clean numeric, so `tr -dc '0-9.'` yields `20.9.0` correctly.
- `docusaurus/` dir exists; `cd "$CLONE_DIR/docusaurus"` then `yarn start` matches the frontend's own `make dev`.
- `docusaurus/sidebars.js` reads `../docs/sidebar.json` (i.e. `$CLONE_DIR/docs/sidebar.json`), and the script writes the regenerated sidebar to exactly `$DEST/sidebar.json` = `$CLONE_DIR/docs/sidebar.json`. Match.
- Frontend's own `scripts/download-docs.sh` lands docs at `<root>/docs` and sidebar at `<root>/docs/sidebar.json` — identical layout to what this script produces, so the preview faithfully mirrors prod assembly. The one intentional divergence: prod copies the committed `misc/docs/sidebar.json`, the preview regenerates it from the local README so unsaved index edits show up.
- `indexparser/main.go` writes the sidebar to stdout (`fmt.Println(output)` at `main.go:128`), so the `>` redirect is correct.
- `bash -n misc/docs/preview.sh` passes; file mode is `100755` (executable, so the bare `../misc/docs/preview.sh` invocation in the Makefile works).
- `git check-ignore docs/.preview/docs.gno.land/docs/README.md` → ignored. Clone is fully excluded from git.
- `preview` is not referenced by any `.github/` workflow; `make generate -B` in `ci-codegen-verify.yml` is unaffected (preview is a separate target).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`misc/docs/preview.sh:88`](https://github.com/gnolang/gno/blob/eae090c2f/misc/docs/preview.sh#L88) · [↗](../../../../../.worktrees/gno-review-5752/misc/docs/preview.sh#L88) — `NODE_VERSION="$(tr -dc '0-9.' < "$CLONE_DIR/.nvmrc")"` assumes a numeric `.nvmrc`. The current frontend pins `v20.9.0` so this is fine today, but if the frontend ever switches to a named alias (`lts/iron`, `lts/*`) the parse yields garbage and both the fnm `--using` and the non-fnm major check break silently/opaquely.
  <details><summary>details</summary>

  `tr -dc '0-9.'` on `lts/iron` produces an empty string; `fnm install ""` fails (swallowed by `|| true`), then `fnm exec --using=""` errors with a confusing message. Low likelihood since the pin has been numeric, but the failure mode is non-obvious. Fix: let fnm resolve `.nvmrc` directly where possible (`fnm use` / `fnm exec --using-file`-style resolution reads `.nvmrc` natively), or guard with a clear error if `NODE_VERSION` comes out empty.
  </details>
- [`misc/docs/preview.sh:107-109`](https://github.com/gnolang/gno/blob/eae090c2f/misc/docs/preview.sh#L107-L109) · [↗](../../../../../.worktrees/gno-review-5752/misc/docs/preview.sh#L107-L109) — `run_node corepack yarn install` has no preflight for `corepack`. Node gets a clear "needs Node vX" check at `:98-103`; corepack/yarn do not, so a Node install without corepack enabled fails with a rawer error than the rest of the script's friendly diagnostics. Minor: corepack ships with modern Node.
- [`misc/docs/preview.sh:79`](https://github.com/gnolang/gno/blob/eae090c2f/misc/docs/preview.sh#L79) · [↗](../../../../../.worktrees/gno-review-5752/misc/docs/preview.sh#L79) — the watcher subshell is killed via an `EXIT` trap on the main shell, but `yarn start` (the foreground process) is what the user Ctrl-C's; the SIGINT propagates to the whole process group, so the background poller dies too. Works, but worth a one-line comment that the trap is the belt-and-suspenders path for non-signal exits.

## Missing Tests
None — shell tooling for local docs preview; no unit-testable surface, and the structural assumptions are validated against the live frontend repo above. CI does not and should not exercise this target (it clones an external repo and starts a long-lived dev server).

## Suggestions
- [`misc/docs/preview.sh:90-96`](https://github.com/gnolang/gno/blob/eae090c2f/misc/docs/preview.sh#L90-L96) · [↗](../../../../../.worktrees/gno-review-5752/misc/docs/preview.sh#L90-L96) — consider having `run_node` echo the Node version actually selected once at startup (e.g. `node -v` under the chosen path), so a user debugging a render mismatch can confirm fnm picked the pinned version rather than a stale global.

## Questions for Author
- `UPDATE=1` does `git -C "$CLONE_DIR" pull --ff-only` on a `--depth 1` shallow clone — on a force-push or rebased frontend `main` this can fail with "not possible to fast-forward". Intended (fail loud, user re-clones) or should it fall back to a fresh clone?

## CI
All functional checks green (build, lint, test, docs, e2e). The only red is the `Merge Requirements` bot gate ("Pending initial approval by a review team member"), which is the standard non-tech-staff approval requirement, not a code failure.
