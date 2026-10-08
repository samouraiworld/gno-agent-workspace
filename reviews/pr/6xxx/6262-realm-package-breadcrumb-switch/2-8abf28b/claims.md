# Claims: gnolang/gno#6262 round 2, 8abf28b6a, claude-opus-5-5, solo review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/counterpart.go:121 | counterpartTarget("/r/tests/vm/foo", "/r/tests/vm", [vm, vm/crossrealm, vm/subtests]) and the rendered menu on /p/tests/vm/foo, at head and base; then /r/tests/vm$source checked for both children | head: `c1 twinless, dir has direct children: target="/r/tests/vm" n=1 label="Matching realm"`, `/r/tests/vm$source lists /r/tests/vm/crossrealm: true`, `... subtests: true`; base d631c4496: `target="/r/tests/vm" n=3 label="3 matching realms"`; with judge-1-fix.patch: `target="/r/tests/vm$source#subpackages" n=2 label="2 matching realms"`, all Counterpart tests ok | tests/judge-1-3-counterpart-target_test.go | hot |
| 2 | CONFIRMED | Nit | gno.land/pkg/gnoweb/counterpart.go:104 | render /p/tests/vm/crossrealm with a MockClient fixture and read the item-path span of the primary item, at head and base | head: `/p/tests/vm/crossrealm: href="/r/tests/vm$source#subpackages" label="2 matching realms" item-path="/r/tests/vm$source#subpackages"`; base: `item-path="/r/tests/vm"` | tests/judge-1-2-menu_test.go | hot |
| 3 | CONFIRMED | Nit | gno.land/pkg/gnoweb/counterpart.go:122 | counterpartTarget("/p/alice/golf/zz", "/p/alice/golf", [/p/alice/golf + 99 /p/alice/golf/aNN]) at head and base | head: `c3 capped listing (100 paths), twin past the cap: target="/p/alice/golf" n=1 label="Matching package"`; base d631c4496: `target="/p/alice/golf" n=100 label="100+ matching packages"` | tests/judge-1-3-counterpart-target_test.go | hot |

Hit rate per tier, from the rows above: hot 3/3 confirmed over 2 files, warm 0/0 confirmed over 0 files, cold 0/0 confirmed over 2 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | gno.land/pkg/gnoweb/counterpart.go:104 | round-1 Warning (N matching opens one package when the twin's directory is a package): suspected still open | round-1 tests/solo-finder-1-dir-is-package.go at 8abf28b6a, request built from the href with the fragment cut as a browser does: `page /p/tests/vm/crossrealm: switch href="/r/tests/vm$source#subpackages" label="2 matching realms"` and `page /r/alice/golf/v1: switch href="/p/alice/golf$source#subpackages" label="2 matching packages"` both list every counted path; `page /p/gov/dao/utils: switch href="/r/gov/dao" label="Matching realm"` no longer promises a listing. Run verbatim it fails only because httptest keeps #subpackages in the request path. Resolved. |
| lines | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:4096 | round-1 Nit (inset: auto overrides the no-anchor fallback): suspected the source reorder never reached the bundle | public/main.css at 8abf28b6a: `.b-kind-switch{position-anchor` at byte 118083, `@supports not (top:anchor(bottom))` at 119957 (after it; at c5ae4b13a it was 117042, before the rule at 117138). round-1 tests/solo-finder-2-kind-switch-fallback.sh re-run on head's main.css, Chromium: `== noanchor ... computed top/left: 56px/64px` (round 1: 0px/0px). Resolved. |
| lines | gno.land/pkg/gnoweb/counterpart.go:106 | round-1 Nit (count covers direct siblings, listing shows the subtree) | go test ./pkg/gnoweb -run Counterpart -count=1 at head: `--- PASS: TestCounterpart_LinkOpensWhatItCounts`, whose case "twin's listing counts the whole subtree it shows" asserts "3 matching packages" and /p/alice/golf/ui/board listed. Resolved. |
| reach | gno.land/pkg/gnoweb/counterpart.go:236 | round-1 Suggestions (errors never cached; full cache stores no new root) | diff.md: the only counterpart.go hunk is `@@ -44,95 +44,118 @@` ending in counterpartLink; counterpartCache.get and store are untouched; author replies 4219456017 and 4219456318 defer both to a follow-up. Carried, open. |
| claims | gno.land/pkg/gnoweb/counterpart.go:41 | the PR body's Target sentence ("whose listing opens instead", "the deepest directory holding several") no longer describes the package-directory branches | fix is the description's wording alone; the task returns nothing whose only fix is wording |
| catalog | gno.land/pkg/gnoweb/counterpart.go:44 | invariant catalog classes against the diff | diff touches gnoweb Go and CSS only (4 files per risk.md); the catalog's classes and realm audit patterns cover .gno code, none applies |

## Completeness

- Angles: lines and reach returned the three candidates; the finder's claims and catalog reads settled with no candidate (rows above). No test-mutation angle ran; the judge's fix run, `tests/judge-1-fix.patch`, kept every `Counterpart` test green, the `/r/gov/dao` case included.
- Catalog: the diff touches gnoweb Go and CSS only; no class of `invariant-catalog.md` applies.
- Siblings by shape: the package-directory shape has two members, the twin branch's `case slices.Contains(members, dir)` and the twinless walk's `case n > 1 && slices.Contains(members, dir)`; candidate 1 is the second. `buildSubpackages` (`components/overview_build.go:138`) keeps direct-child packages only, matching the twin branch's `siblings` count. `countUnder` has one call site; the twinless loop also needs `last` and does not fold into it.
- CSS: `public/main.css` at the head carries `@supports not (top:anchor(bottom)){.b-kind-switch{...}}` at byte 119958, after `.b-kind-switch{position-anchor...` at 118088, same specificity.
- Reflector: none ran as its own agent; `candidates/reflector.json` holds these answers and no new candidate.
