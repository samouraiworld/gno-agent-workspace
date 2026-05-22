# PR #5557: feat(gnovm): add interactive gno init wizard with template scaffolding

**URL:** https://github.com/gnolang/gno/pull/5557
**Author:** davd-gzl | **Base:** master | **Files:** 12 | **+1603 -53**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR promotes `gno mod init` to a top-level `gno init` command with an interactive wizard that scaffolds realm, package, or run script projects from embedded templates. Key changes:

- **`gno init` as top-level command** — moved from `gno mod init` to `gno init`, with `--bare` flag for gnomod.toml-only creation and `--template` flag for non-interactive template selection.
- **Three module kinds**: realm (`/r/`), package (`/p/`), run script (`.gno`). Kind is auto-detected from the module path or prompted interactively.
- **Short-form paths**: `r/demo/foo` and `p/demo/foo` auto-expand to `gno.land/r/demo/foo` and `gno.land/p/demo/foo`.
- **`.gno` argument shorthand**: `gno init run/hello.gno` creates a run script without gnomod.toml.
- **Directory-based template system** in `mod_init_templates.go` with `go:embed`, `text/template`, and templated filenames (e.g. `{{.PkgName}}.gno.tmpl` → `myrealm.gno`). Adding a new template requires only `.tmpl` files + one registry entry.
- **Shared prompt primitives** in `tm2/pkg/commands/prompt.go`: `IsInteractive()`, `PromptString()`, `PromptChoice()`, `PromptSelect`. Designed for reuse by `gnokey maketx` wizard.
- **ADR** at `gnovm/adr/pr5557_mod_init_template.md` documenting decisions and alternatives.
- **File conflict detection** before writing any output file.

Files changed: `main.go` (register init), `mod.go` (init logic + wizard), `mod_init_templates.go` (template registry + rendering), `mod_test.go` (396 lines of new tests), `prompt.go` + `prompt_test.go` (shared prompts), 5 `.tmpl` template files, 1 ADR.

## Test Results

- **Existing tests:** PASS — all `TestModInit*`, `TestPromptModuleKind`, `TestSelectTemplate`, `TestKindFromPath`, `TestRenderTemplateDir`, `TestPromptModulePath`, `TestResolveTemplate`, `TestSanitizeModuleName`, `TestNormalizeModulePath`, `TestPromptString`, `TestPromptChoice`, `TestPromptSelect` pass locally.
- **CI:** `main / lint` FAIL — 3 prealloc warnings on `var created []string` in `mod.go:299,339,382`. `main / build` FAIL — appears to be a CI cache issue (tar "Cannot open: File exists" errors), not PR-related.
- **E2E tests (manual):** 15 scenarios tested. See findings below for issues discovered.

## Critical (must fix)

- [ ] `mod.go:186-191` — **Non-interactive mode skips template scaffolding entirely.** When `IsInteractive()` returns false (piped input, CI, scripts), `execModInit` only calls `writeGnomod()` even with an explicit module path. The `--bare` flag exists precisely for this use case, so non-interactive + path should still scaffold templates (auto-detect kind, use default template). Current behavior: `gno init gno.land/r/demo/myrealm` in a Makefile gives you just a gnomod.toml with no source files — barely useful. The `--template` flag is also ignored in this path.
- [ ] `mod.go:170-177` — **Path traversal in `.gno` argument.** `gno init '../escape.gno'` writes a file outside the CWD. `gno init '/tmp/hack.gno'` creates `tmp/hack.gno` inside CWD (misleading). The `.gno` path argument needs validation: reject absolute paths, reject `..` components, ensure the path stays within CWD.
- [ ] `mod.go:324` — **Empty script name from `.gno` edge case.** `gno init '.gno'` produces a file named `.gno` with `ScriptName=""` — a hidden file with an empty Go identifier. Should validate that the basename before `.gno` is non-empty and a valid identifier.

## Warnings (should fix)

- [ ] `mod.go:318-353` — **`execInitRunScript` and `execInitRun` duplicate logic.** Both do: conflict checking, sorted file writing, output printing. They should be consolidated — the only difference is how the script name and output directory are determined. Suggest a single `writeRunScript(rootDir, relPath string, tmpl initTemplate, io commands.IO)` that both callers use.
- [ ] `mod.go:299,339,382` — **`var created []string` should pre-allocate** (`created = make([]string, 0, len(files))`). This is the cause of the CI lint failure.
- [ ] `mod.go:475-493` — **`insertPathLetter` uses `strings.Index` (finds first `/`).** For a path like `gno.land/myname/sub/deep`, this correctly inserts after the domain. But if the domain itself contains no `/` (e.g. `gno.land` with no namespace), it returns an error. The function should also handle paths that already have `/r/` or `/p/` (idempotent), or at least document that it's only called on wizard-generated paths that are guaranteed to need insertion.
- [ ] `mod.go:194-206` — **Interactive-with-argument path writes gnomod.toml before checking template validity.** If `resolveTemplate` fails (e.g. unknown `--template` name), `gnomod.toml` is already on disk. The gnomod write and template resolution should be ordered so that validation errors surface before any files are written, or a partial write should be cleaned up.
- [ ] `mod.go:457-462` — **`kindFromPath` only recognizes `r/` as realm, everything else is `package`.** Paths like `gno.land/x/something` are treated as packages, but `IsUserlib` accepts them. If `x/` or other prefixes exist, `kindFromPath` should handle them or explicitly reject them.
- [ ] `prompt.go:32-34` — **`IsInteractive` hardcodes `os.Stdin`**, making it untestable and inflexible. It should accept an `os.File` or `fd` parameter so callers (and tests) can inject a custom input source. Currently the non-interactive fallback is tested by relying on the test environment not being a TTY, which is fragile.

## Nits

- [ ] `mod.go:22` — `terrors` alias for `tm2/pkg/errors` is only used once (line 689). Consider using the stdlib `fmt.Errorf` instead, consistent with all other error creation in this file, and drop the alias.
- [ ] `mod.go:406` — Prompt text `Module kind — [r]ealm, [P]ackage, or [m]ain:` uses em-dash and inconsistent bracket capitalization (`[P]` uppercase but `[r]` and `[m]` lowercase). Should be consistent.
- [ ] `mod_init_templates.go:30` — `path/filepath` is imported but `filepath.Join` is used only once. `path.Join` would work for embedded FS paths (always forward-slash), and the import could be `path` instead. Minor.
- [ ] `ADR pr5557_mod_init_template.md:96` — ADR mentions `PromptConfirm` in key functions list, but it was removed in commit `6e24a5c` (YAGNI). The ADR should be updated to reflect this.

## Missing Tests

- [ ] No test for **non-interactive mode with `--template`** creating template files. Currently `TestModInitWithTemplateFlag` has a comment admitting it can't verify template creation because `IsInteractive()` is false in tests — this is the exact bug flagged in Critical.
- [ ] No test for **path traversal** in `.gno` argument (`../`, absolute paths).
- [ ] No test for **empty/invalid basename** in `.gno` argument (`.gno`, `..gno`).
- [ ] No test for **`gno.land/x/`** path behavior with `kindFromPath`.
- [ ] No test for **`insertPathLetter`** with paths that already contain `/r/` or `/p/`.
- [ ] No test for **gnomod.toml partial write** when template resolution fails after gnomod is written.

## Suggestions

- Extract `execInitRunScript` and `execInitRun` into a single `writeRunScript(rootDir, relOutPath string, tmpl initTemplate, io commands.IO) error` function. The callers just compute the relative output path differently.
- Move `resolveTemplate`, `initTemplate`, `templateData`, `renderTemplateDir`, and the template registries into `mod_init_templates.go` (or a new `mod_init.go`) to keep `mod.go` focused on `gno mod` subcommands. Currently `mod.go` mixes 400 lines of init logic with unrelated mod commands.
- Validate the `.gno` argument early: reject `..`, reject absolute paths, require a non-empty valid identifier before the `.gno` suffix. Add `validateGnoPath(path string) error`.
- Make `IsInteractive` accept a parameter or use the `IO` interface so it can be tested properly.

## Questions for Author

- Should `gno mod init` still work as a subcommand for backward compatibility, or is the removal intentional? The old `newModInitCmd()` was removed from `newModCmd`'s subcommands with no deprecation path.
- Is `gno.land/x/` a valid module kind? `kindFromPath` maps it to `package`, but should the init wizard reject unknown prefixes?
- Should `--template` in non-interactive mode force template creation even without a TTY? The current design treats "no TTY" as "only gnomod.toml", but `--template` semantically means "I want templates" — the two flags seem to conflict in intent.

## Verdict

REQUEST CHANGES — Non-interactive mode incorrectly skips all template scaffolding (the core feature), path traversal in `.gno` arguments is a security issue, and `execInitRunScript`/`execInitRun` should be consolidated. CI lint also fails on prealloc. The design and ADR are solid, tests are good, but the critical paths need fixing before merge.

---

**Update:** All issues from this review have been fixed in commit `4265d14` pushed to the PR branch. Key fixes:

- Non-interactive mode now scaffolds template files when a module path is provided (only `--bare` skips templates)
- Added `validateGnoPath()` — rejects absolute paths, `..` traversal, and empty/invalid script names
- Consolidated `execInitRunScript` + `execInitRun` into single `writeRunScript` function
- Template resolution happens before `gnomod.toml` write (no partial writes on template errors)
- Pre-allocated `created` slices (lint fix)
- Removed `terrors` alias, fixed prompt capitalization `[p]ackage`, used `path.Join` for embed FS, removed `PromptConfirm` from ADR
- Added tests: `TestValidateGnoPath`, `TestModInitBareNoPath`, `TestModInitUnknownTemplateNoPartialWrite`
