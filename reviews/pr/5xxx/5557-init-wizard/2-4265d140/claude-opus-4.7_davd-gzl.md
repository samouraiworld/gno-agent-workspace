# PR #5557: feat(gnovm): add interactive gno init wizard with template scaffolding

**URL:** https://github.com/gnolang/gno/pull/5557
**Author:** davd-gzl | **Base:** master | **Files:** 12 | **+1661 -81**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.7

## Summary

Round-2 review of PR #5557 at HEAD `4265d140`. Since the previous review (`ee9d35b`), commit `4265d140` addresses the round-1 findings: non-interactive-with-path now scaffolds templates, `validateGnoPath` guards `.gno` arguments against `..`/absolute/empty-name inputs, `execInitRun`/`execInitRunScript` are consolidated into `writeRunScript`, the `terrors` alias is gone, `created` slices are pre-allocated, prompt capitalization is normalized, the ADR no longer references the removed `PromptConfirm`, and three regression tests were added (`TestValidateGnoPath`, `TestModInitBareNoPath`, `TestModInitUnknownTemplateNoPartialWrite`).

Scope of this round: verify those fixes landed as claimed, look for new issues introduced by the refactor, and investigate the CI failure reported on GitHub.

## Test Results

- **Existing tests:** PASS — `go test ./gnovm/cmd/gno/...` and `go test ./tm2/pkg/commands/...` all green.
- **CI:** `main / build` was **FAILING** prior to this round because `make generate` diff was non-empty: `gnovm/cmd/gno/README.md` was missing the `init` subcommand. Fixed in this round (see below). Lint no longer shows the three prealloc warnings flagged in round 1.
- **Edge-case tests:** 1 new regression test added in this round (`TestModInitTemplateFileConflictNoPartialWrite`) for a critical bug discovered during review.

## Critical (must fix)

- [x] `gnovm/cmd/gno/mod.go:294-352` — **Partial-write regression: orphan `gnomod.toml` when a template output file already exists.** All three non-bare code paths (`execModInitNonInteractiveWithArg`, `execModInitInteractiveWithArg`, `execModInitFullWizard`) call `writeGnomod(...)` *before* `writeModule(...)`. `writeModule` resolves the template, renders it, and only then checks file existence via `os.Stat`. If any rendered target (e.g. `<pkg>.gno`) already exists, the function returns an error — but `gnomod.toml` has already been written to disk. Result: broken module dir with a stale gnomod.toml and no sources, which the user must manually clean up before retry. Fixed in this round by extracting `renderModuleFiles()` (render + pre-existence check, no disk writes) and calling it **before** `writeGnomod` in all three paths. `writeModule` now consumes the pre-rendered files and only writes.
- [x] `gnovm/cmd/gno/README.md` — **CI `build` failing on `make generate` drift.** The auto-generated subcommand list did not include `init`. Added the line; `make generate` now clean.

## Warnings (should fix)

- [ ] `gnovm/cmd/gno/mod.go` — **`gno mod init` subcommand is removed outright**, not deprecated. Any external script, Makefile, or CI pipeline calling `gno mod init <path>` breaks silently (cobra emits `unknown command`). Consider either (a) keeping `gno mod init` as a hidden alias that prints a deprecation notice and delegates to `execModInit`, or (b) calling this out explicitly in the PR description and CHANGELOG so downstream users know to migrate. The ADR mentions the move but not the removal.
- [ ] `gnovm/cmd/gno/mod.go` — **Behavior change for non-interactive `gno init <path>`**: previously (`gno mod init`) wrote only `gnomod.toml`; now it also scaffolds template files unless `--bare`. This is intentional and fixes a round-1 finding, but it is a user-visible behavior change for anyone who scripted against `gno mod init`. Worth a CHANGELOG entry.
- [ ] `gnovm/cmd/gno/mod.go` (`validateGnoPath`) — **`strings.HasPrefix(cleaned, "..")` false-positives on legitimate names starting with `..`** such as `..bar/foo.gno` or `..hidden.gno`. After `filepath.Clean`, a real traversal is either `".."` exactly or begins with `"../"` / `"..\\"`. Tighten the check: `cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator))`.
- [ ] `gnovm/cmd/gno/mod.go` (`promptModulePath`) — **Prompt label says "Address or namespace"** but `validateName` only accepts `[a-z0-9_]+`. A real bech32 address (`g1…`, mixed case, 39 chars) is rejected. Either relax validation to accept bech32 addresses when the prompt advertises them, or reword the label to "Namespace".

## Nits

- [ ] `gnovm/cmd/gno/mod.go` (`execInitRun`) — **Hardcoded `run/` directory prefix.** Minor UX: the user passes `gno init foo.gno` and the file lands at `run/foo.gno` with no opt-out. Consider `--output` or honoring an explicit relative path in the argument. Low priority.
- [ ] `gnovm/cmd/gno/mod_init_templates.go` (`renderTemplateDir`) — **Skips subdirectories entirely.** Fine for v1 but limits future templates that want nested layouts (e.g. `cmd/`, `testdata/`). Worth a TODO comment.
- [ ] `gnovm/cmd/gno/mod.go` (`insertPathLetter`) — Not idempotent; if a caller ever passes a path that already contains `/r/` or `/p/`, the letter is inserted again. Currently safe because it is only invoked on wizard-built paths that are guaranteed not to have the letter, but a guard (`if kindFromPath(p) != ""`) would make it robust.

## Missing Tests

- [x] **Template-file conflict must not leave an orphan `gnomod.toml`.** Added `TestModInitTemplateFileConflictNoPartialWrite` in `gnovm/cmd/gno/mod_test.go` covering the exact partial-write bug: pre-creates `<pkg>.gno`, invokes `gno init --template basic <path>`, asserts error + no `gnomod.toml` written.
- [ ] **`validateGnoPath` edge cases** — names starting with `..` but not a traversal (`..bar.gno`). The current implementation would reject them; a test would lock behavior either way.
- [ ] **`gno mod init` removal** — a test asserting that either (a) `gno mod init` still runs, or (b) it errors with a clear "use `gno init`" message, would lock in whichever behavior the author chooses.

## Suggestions

- Keep `gno mod init` registered as a hidden cobra alias that prints `"gno mod init is deprecated; use gno init"` and forwards to `execModInit`. Removes the breaking-change footgun at trivial cost.
- In `renderModuleFiles` (new helper), sort the file list once at render time rather than re-sorting in `writeModule`; the caller already receives a stable slice.
- Consider moving all template/init code out of `mod.go` (still ~900 lines) into `init.go` now that `gno init` is a top-level command. `mod.go` would go back to being purely about `gno mod`.

## Questions for Author

- Intentional full removal of `gno mod init` with no alias?
- Is the `run/` prefix on script output permanent, or will `--output` be added in a follow-up?
- Should the prompt accept bech32 addresses, or is "Address or namespace" wording a bug?

## Verdict

APPROVE (with follow-ups) — Round-1 findings are all addressed; one critical partial-write regression and a CI-blocking `make generate` drift were found and fixed in this round. Remaining items (deprecation alias, `..` prefix check, prompt wording) are non-blocking and can land in a follow-up PR.

---

**Fixes applied on the PR branch in this round** (commit to follow on `feat/mod-init-template`):

- `gnovm/cmd/gno/README.md` — added `init` subcommand line (unblocks CI `build`).
- `gnovm/cmd/gno/mod.go` — extracted `renderModuleFiles()`; pre-check runs before `writeGnomod` in all three non-bare paths; `writeModule` refactored to consume pre-rendered files.
- `gnovm/cmd/gno/mod_test.go` — added `TestModInitTemplateFileConflictNoPartialWrite`.

All `./gnovm/cmd/gno/...` and `./tm2/pkg/commands/...` tests pass; `go vet` clean.
