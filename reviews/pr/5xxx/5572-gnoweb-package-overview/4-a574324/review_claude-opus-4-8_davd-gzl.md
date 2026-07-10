# PR [#5572](https://github.com/gnolang/gno/pull/5572): feat(gnoweb): add package overview source doc

URL: https://github.com/gnolang/gno/pull/5572
Author: alexiscolin | Base: master | Files: 33 | +3044 -120
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `a574324` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5572 a574324`

Round 4 (deep). Head advanced `714d2f8` → `a574324` (master merge plus one commit): the type card's `data-name` now folds in its method names. All four round-3 findings are resolved in the merge commit's own changes: `computeStats` counts only exported types/consts/vars, the unreachable `GetSourceView` fallback is gone, the subpackage `ListPaths` cap is 1000, and the discarded README error carries a comment. The round-3 missing test also landed. Three other reviewers' findings are fixed too: doubled BUG notes, signature-before-doc ordering, and the symbol filter that could not match methods.

**TL;DR:** Opening `/r/<pkg>$source` used to drop you into the first source file. This PR makes it a pkg.go.dev-style landing page instead: package doc, README, the exported functions, types, constants and variables (each deep-linking to its exact source line), imports, files, subdirectories, and a sidebar with code stats and a detected license.

**Verdict: APPROVE** — two should-fix display bugs, both in new code and both cheap: the license badge never appears for real GPL, AGPL or BSD license files, and a type alias renders identically to a defined type. Neither blocks merge. Routing, deep-link line accuracy, the method-name fold and the sidebar-versus-section counts were all verified against a live node.

## Summary

`/r/<pkg>$source` dispatches on the `file` query: with a file it is the old source-code view, without one it is the new `OverviewView` ([handler_http.go:434-440](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/handler_http.go#L434-L440) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L434-L440)). The handler fans out `ListFiles` plus a metadata-file fetch, `Doc`, the README render, and `ListPaths` over an `errgroup`, then `BuildOverview` turns the results into a pure view payload. Only `ListFiles` can fail the page; qdoc, README and `ListPaths` failures each degrade in place. `gnovm/pkg/doc` supplies `File`, `Line` and `Imports` on the qdoc payload, so the overview deep-links symbols and lists dependencies without refetching or reparsing any `.gno` source.

The remaining defects are all in the presentation layer, where the page states something the underlying data does not support: a license kind it cannot detect, and a type alias it cannot distinguish from a defined type.

## Glossary

- **qdoc** — the `vm/qdoc` ABCI query returning a package's documentation as JSON; served with unexported symbols included, so the render path filters them.
- **degraded mode** — a failed secondary fetch is swallowed so the page still renders without that part.

## Critical (must fix)

None

## Warnings (should fix)

- **[license badge never appears for real GPL, AGPL or BSD files]** [`overview_license.go:12-16`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/overview_license.go#L12-L16) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_license.go#L12-L16) — The signatures assume the license title and its version sit on one line, but real license files wrap there, so detection returns an empty kind.
  <details><summary>details</summary>

  Go's `.` never matches a newline, and the GPL and AGPL patterns bridge title to version with `.*`. A canonical FSF header puts `GNU GENERAL PUBLIC LICENSE` and `Version 3, 29 June 2007` on separate lines, so `.*` cannot reach across. The BSD patterns fail for a second, distinct reason: their anchor phrase `with or without modification` is a literal containing a space, and a real BSD file wraps precisely between `without` and `modification`, so no dotall flag can rescue it. Apache-2.0 and MPL-2.0 survive only by accident, the first because it joins with `\s*` and the second because its header is one line.

  Existing coverage misses this because [`overview_license_test.go:60-62`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/overview_license_test.go#L60-L62) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_license_test.go#L60-L62) feeds the BSD-3 clause as a single unwrapped line. Fed the verbatim wrapped headers, GPL-3.0, AGPL-3.0, BSD-3-Clause and BSD-2-Clause all report an empty kind while Apache-2.0 resolves. The package still lists its LICENSE file; only the kind is dropped. Repro and the red-to-green test in [comment](comment_claude-opus-4-8.md); the test artifact is [`license_wrapped_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5572-gnoweb-package-overview/4-a574324/tests/license_wrapped_test.go) · [↗](tests/license_wrapped_test.go).

  Fix: make each pattern tolerate a newline wherever a real license file wraps, which for the BSD pair means inside the anchor phrase itself, not only between the anchors.
  </details>

- **[a type alias is rendered as a defined type]** [`overview_symbols.go:27-34`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/overview_symbols.go#L27-L34) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_symbols.go#L27-L34) — qdoc reports whether a type is an alias, the overview drops the flag, so `type A = B` and `type A B` render byte-identically.
  <details><summary>details</summary>

  [`json_doc.go:266`](https://github.com/gnolang/gno/blob/a574324/gnovm/pkg/doc/json_doc.go#L266) · [↗](../../../../../.worktrees/gno-review-5572/gnovm/pkg/doc/json_doc.go#L266) sets `Alias` from `typeSpec.Assign != 0`, and the doc fixture [`hello.gno:27`](https://github.com/gnolang/gno/blob/a574324/gnovm/pkg/doc/testdata/integ/hello/hello.gno#L27) · [↗](../../../../../.worktrees/gno-review-5572/gnovm/pkg/doc/testdata/integ/hello/hello.gno#L27) exercises it. But `buildSymbols` never reads the field, `TypeEntry` has no place to put it, and the template never asks for it. `Alias` appears nowhere under `gno.land/pkg/gnoweb`.

  Rendering `{Name: "Aliased", Type: "int", Kind: "ident", Alias: true}` and `{Name: "Defined", Type: "int", Kind: "ident"}` through `BuildOverview` produces the same card shape for both: the header `type <Name>`, an `ident` tag, and a code block holding `int`. A reader cannot tell an alias from a defined type, and the two differ in assignability and method sets. No package under `examples/` declares an alias today, so nothing on chain renders wrong yet, but gnoweb serves arbitrary user packages.

  Fix: carry the alias flag through to the type card so an alias reads as one.
  </details>

## Nits

- **[unexported names reach the page]** [@Villaquiranm](https://github.com/gnolang/gno/pull/5572#issuecomment-4807460830) [`overview_symbols.go:154-157`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/overview_symbols.go#L154-L157) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_symbols.go#L154-L157) — A value group is kept when any one of its names is exported, then every name in it is joined into the card title. Confirmed: a `const` group holding `Exported` and `unexported` renders as `Exported, unexported`. Struct type cards leak the same way, since `JSONType.Type` is the full declaration text and includes unexported fields. The sidebar counts stay correct, because they count groups rather than names.

- **[type cards show the underlying expression, not the declaration]** [@Villaquiranm](https://github.com/gnolang/gno/pull/5572#issuecomment-4807513337) [`overview.html:348-353`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/views/overview.html#L348-L353) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/views/overview.html#L348-L353) — The card prints `type <Name>` in its header, a kind tag, then a code block holding only the underlying expression, so `type T int` never appears as one declaration. The alias Warning above is the sharp edge of the same split.

- [`overview_build.go:24`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/overview_build.go#L24) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_build.go#L24) — `bugsNotInDoc` drops a BUG note whenever its text appears anywhere in the package doc, so a distinct floating note that happens to be a substring of an inline one is swallowed. Confirmed behaviorally: with notes `"Foo panics on nil"` and `"Foo panics on nil input; use Bar"` and only the longer one inline, the function returns an empty list. Comparing whole notes rather than raw substrings would be exact.

- [`handler_http_test.go:571-572`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/handler_http_test.go#L571-L572) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http_test.go#L571-L572) — `TestHTTPHandler_GetSourceView_Error` and both of its comments claim to cover `GetSourceView`, but its request is `/r/errsrc$source` with no file, which this PR now routes to `GetOverviewView`. The assertion still holds, against a different function. The sibling `_NoFiles` test was updated for the new routing; this one was missed.

- [`overview.html:219`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/views/overview.html#L219) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/views/overview.html#L219) — Import tags emit `b-tag--kind-{{ .Kind }}`, yielding `b-tag--kind-stdlib`, `-package`, `-realm` and `-external`, but only the base [`.b-tag--kind`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3268) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3268) rule exists, with no per-kind rule and no attribute selector. The four kinds render identically. Same class on the disabled branch at line 224.

- [`overview.html:75`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/views/overview.html#L75) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/views/overview.html#L75) — The metadata README tag and the sidebar table-of-contents entry both link `#readme` off `Quality.HasReadme`, which is derived from the file list, while the section itself renders only `{{ with .Readme }}` at line 131. A README that is listed but whose fetch fails leaves both links pointing at a section that was never emitted.

## Missing Tests

- **[the whole subpackage path is unexercised]** [`handler_http.go:1108-1123`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/handler_http.go#L1108-L1123) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L1108-L1123) — Every overview handler test returns nil or an error from `ListPaths`, so neither the Directories section nor the failure degradation is ever rendered.
  <details><summary>details</summary>

  The three overview tests ([handler_http_test.go:1522](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/handler_http_test.go#L1522) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http_test.go#L1522), [:1578](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/handler_http_test.go#L1578) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http_test.go#L1578), [:1611](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/handler_http_test.go#L1611) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http_test.go#L1611)) stub `listPathsFunc` to `nil, nil`, except the 404 case where `ListFiles` also fails. Two consequences. The domain-trim loop at 1118-1121 never runs, so nothing catches that `buildSubpackages` needs domain-relative paths and would silently drop every child if `Static.Domain` were misconfigured. And the `return nil` swallow at 1113-1115 is unpinned: a regression to `return err` would take the whole overview down on a transient `ListPaths` error with CI still green.

  The two cases to add are in [comment](comment_claude-opus-4-8.md): one asserting a direct child reaches the Directories section, one asserting a `ListPaths` error still yields 200.
  </details>

## Suggestions

- [`controller-filter.ts:47`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/frontend/js/controller-filter.ts#L47) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/frontend/js/controller-filter.ts#L47) — Method cards are themselves filter items carrying only their own `data-name`, so a query matching just the receiver keeps the type card visible while hiding every method inside it. Filtering `tree` on `/p/nt/avl/v0$source` leaves the `Tree` card with an empty methods block. `a574324` fixed the method-to-type direction; this is its mirror image.

## Verified

- The overview serves live from this worktree. `gnodev` booted at `a574324` and `GET /p/nt/avl/v0$source` returns 200.
- Symbol deep-links land on the exact declaration line. On the live page, `type Tree` links `tree.gno#L27`, `NewTree` `#L32`, `Tree.Size` `#L39` and `Tree.Get` `#L51`; the same four declarations sit on exactly those lines in `tree.gno`.
- The sidebar counts now agree with the sections rendered. Live, `/p/nt/avl/v0$source` reports `Types 4` against 4 type cards and `Funcs 37 (2 exported)` against 2 function cards, with no Consts or Vars row and no such sections.
- The method-name fold works on real data: the live `Tree` card carries `data-name="Tree Get GetByIndex Has Iterate IterateByOffset Remove ReverseIterate ReverseIterateByOffset Set Size"`, so filtering by a method name matches its type card.
- The license defect reproduces against verbatim license text and the proposed change closes it. Feeding wrapped GPL-3.0, AGPL-3.0, BSD-3-Clause and BSD-2-Clause headers to `deriveLicense` yields an empty kind while Apache-2.0 resolves; adding the dotall flag to the GPL pair and a `\s+` inside the BSD anchor phrase turns all five green with the existing license tests still passing.
- Rendering an alias and a defined type with the same underlying expression through `BuildOverview` produces identical cards.
- `go test ./gno.land/pkg/gnoweb/ ./gno.land/pkg/gnoweb/components/ ./gnovm/pkg/doc/` is green at `a574324`.

## Open questions

- The scroll observer only watches sections that a nav anchor points at. Two nav tabs group two sections each and link only the first ([overview.html:94-95](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/views/overview.html#L94-L95) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/views/overview.html#L94-L95)), so scrolling through the unlinked half clears every tab's highlight rather than lighting the wrong one. Not posted: cosmetic, and it only bites a package holding both a README and a package doc, or both consts and vars.
- jefft0 (PR thread): clicking a subdirectory such as `/r/sys/users` lands on the file listing rather than the realm's Render page. The Directories links point at the bare package path, so their routing is the existing `/r/<pkg>` behavior this PR does not touch, and which view appears depends on whether that path declares `Render()` on the running node. Carried unchanged from round 3. Not posted: needs the author's product call, not a code change here.
- Villaquiranm (PR thread): line numbers in the overview's code blocks hurt readability. Pure product preference, no code claim to check.
- CI `main / test` is red on `TestFiles/alloc_7.gno`, an allocator byte-count golden (`6299` expected, `6907` observed) in `gnovm/pkg/gnolang`. The PR touches neither that package nor that file; the golden drifted on master. Not a code problem for this PR.
