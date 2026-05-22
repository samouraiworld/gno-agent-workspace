# PR #5565: feat(gnoweb): consolidate built-in playground UI

**URL:** https://github.com/gnolang/gno/pull/5565
**Author:** jeronimoalbi | **Base:** playground2 | **Files:** 13 | **+301 -165**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR implements milestone 0 (M0) from #5549, consolidating the built-in playground, eval, and run views in gnoweb to be visually and structurally consistent with the gnoweb design system. Key changes:

- **HTML templates:** Unifies header structure across playground, eval, and run views using `b-content-header` / `b-content-h1`. Playground toolbar becomes a `<header>`. Run view collapses the old two-template pattern (`renderRun` + `renderRunContent`) into a single `renderRun` template and reuses playground editor CSS classes (`b-playground-editor-area`, `b-playground-code`). Eval view restructures Quick Call from `<div>` grid to `<ul>/<li>` list, adds `<header>` with page title.

- **CSS (`06-blocks.css`):** Adds `.b-run` block styles (settings, commands, command-header, command-pre). Refactors `.b-playground` (removes gap, adds padding-bottom, rounded code corners, output panel background). Refactors `.b-eval` (min-height, gap, result panel styles). Increases tab font size, output/history min-heights. Applies `s-color-bg-surface-secondary` backgrounds consistently.

- **TypeScript controllers:** `controller-playground.ts` adds `gnomod.toml` file support (creation + default content), duplicate filename detection via `switchToFile` return value, `setOutput()` helper. `controller-run.ts` reformatted for biome compliance. `controller-eval.ts` reformatted. Share URL encoding changes from raw `encodeURIComponent` to `TextEncoder→btoa→encodeURIComponent` (base64).

- **Go (`handler_playground.go`):** Removes redundant `min()` function (now using built-in `min` from Go 1.21+).

- **Generated assets:** `public/js/controller-*.js` and `public/main.css` regenerated.

The PR targets the `playground2` feature branch (not master), as part of a staged playground overhaul.

## Test Results

- **Existing tests:** gnoweb-specific CI checks PASS (`gnoweb_front_lint`, `gnoweb_generate`). Other failures (`gnogenesis/test`, `gnohealth/test`, `main/lint`, `main/test`) are pre-existing and unrelated to this PR.
- **Edge-case tests:** Skipped (UI/frontend PR with no Go logic changes)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gno.land/pkg/gnoweb/components/view_run.go:27` — `NewTemplateComponent("renderRunContent", data)` references a template that no longer exists in any `.html` file (removed by this PR's `run.html` restructure). The component is created but never rendered because the new `renderRun` template doesn't use `.Article`. This is dead code that will panic if someone later adds `{{ template "layout/article" .Article }}` back to the template without realizing `renderRunContent` was deleted. The `runViewParams.Article` field and the entire `content` variable on line 27 should be removed, and `viewData` should be simplified to just pass `data` directly to the `renderRun` template. This was flagged by @alexiscolin in review; author plans to defer, but since the breakage is caused by this PR's template removal, it should be cleaned up here.

- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-playground.ts:13` — `let gnomodFile = -1` is assigned on line 88 (`gnomodFile = files.length`) when a `gnomod.toml` file is added, but the variable is never read anywhere. If the intent was to track the gnomod file index for future use (e.g., preventing deletion, special rendering), add that logic or remove the variable. Currently it's misleading dead code.

- [ ] `gno.land/pkg/gnoweb/components/views/run.html:29-39` — The Run view now uses playground CSS classes (`b-playground-editor-area`, `b-playground-editor`, `b-playground-code`) instead of its own `b-run-editor-*` classes. This creates a hidden coupling: if the playground CSS is refactored or removed, the Run view silently breaks with no compile-time or lint-time signal. Consider either sharing via a common `b-editor` component class or documenting the dependency.

## Nits

- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-playground.ts:81` — Typo: "metatadat" should be "metadata".

## Missing Tests

- [ ] No test for `gnomod.toml` file creation path in `controller-playground.ts` (the `isGnomod` branch at line 82-88). The default content template, the `gnomodFile` index tracking, and the validation that only `.gno` and `gnomod.toml` are allowed should be covered.
- [ ] No test for duplicate filename detection: `addFile()` now calls `switchToFile(name)` before creating a file, switching to the existing tab if the name matches. This behavioral change should have a test.
- [ ] No test for the `setOutput()` helper in `controller-playground.ts` (new abstraction replacing direct `outputEl.textContent` + `classList` manipulation across all actions).
- [ ] The Run view template restructure (`renderRunContent` removal) has no test verifying the Run page still renders correctly after the template collapse. `handler_http_test.go` tests the playground page but not the run page.

## Suggestions

- Remove the `gnomodFile` tracking variable from `controller-playground.ts` if it's not used, or add a TODO comment explaining planned future use. It adds cognitive overhead with no current benefit.
- The `b-playground-output-header` and `b-eval-result-header` CSS rules still set `background-color: var(--s-color-bg-muted)` and `border-bottom`, but these headers are now inside panels that already have `background-color: var(--s-color-bg-surface-secondary)` and borders. The nested muted background may look inconsistent — consider removing the header background to let it inherit the panel surface color.

## Questions for Author

- The `view_run.go` dead code (`renderRunContent` reference, `Article` field) was flagged by @alexiscolin and you noted it would be addressed in a follow-up PR. Since the breakage is directly caused by this PR's template removal, wouldn't it be cleaner to fix it here rather than leave a latent crash trap on the `playground2` branch?
- What is the intended use of the `gnomodFile` index variable? Is there a planned feature (e.g., read-only gnomod tab, special execution handling) that will consume it?

## Verdict

REQUEST CHANGES — The `view_run.go:27` dead reference to the deleted `renderRunContent` template is a latent crash trap that should be cleaned up in this PR rather than deferred, since the breakage is directly caused by this PR's template restructure. The `gnomodFile` dead variable and the "metatadat" typo are minor but should also be addressed. Otherwise the visual consolidation is clean and well-aligned with the gnoweb design system.
