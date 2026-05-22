# PR #5585: feat(gnoweb): make heading text clickable to set URL hash

**URL:** https://github.com/gnolang/gno/pull/5585
**Author:** davd-gzl | **Base:** master | **Files:** 21 | **+188 -32**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Replaces the existing "empty `<a>` after heading text, `§` via `::after`" anchor-link approach with one that wraps the heading text inside a `<a class="heading-anchor" href="#id">` element, so clicking anywhere on a rendered heading updates `window.location.hash`.

Implementation registers a custom `headingRenderer` (`gno.land/pkg/gnoweb/markdown/ext_heading.go`) at priority 1 in the `GnoExtension` pipeline. On `entering`, writes `<h{level}>` plus `<a class="heading-anchor" href="#id" aria-label="Link to this section">` when the heading has a non-empty `id` attribute. On exit, unconditionally writes `</a></h{level}>`.

Also:
- Adds `parser.WithAutoHeadingID()` to the test setup so golden fixtures match production config. Every pre-existing heading fixture gained `id="…"` attributes plus the new anchor wrapper.
- Adds CSS in `c-realm-view` for `.heading-anchor` (inherit color, no underline, focus-visible outline) and in `c-doc-view` (same, plus `content: " §"` on hover).
- Rebuilds minified `public/main.css`.
- Adds ADR `prxxxx_heading_anchor_links.md` — but the ADR text describes the **previous**, rejected approach, not this PR.

PR is **Draft** ("Still WIP" per body). No reviews yet.

## Test Results
- **Existing tests:** PASS (`go test ./gno.land/pkg/gnoweb/markdown/...`)
- **CI:** FAIL — `gnoweb_generate` and `main / build` both report `public/main.css` differs from what `make gnoweb.generate` / `make generate` produce on CI. Bundled CSS out of sync.
- **Edge-case tests:** 2 written (`reviews/pr/5585-heading-anchor-clickable/1-5c82b44a/tests/heading_edge_test.go`); 2 of 2 failed, confirming both critical bugs below.

## Critical (must fix)

- [ ] `gno.land/pkg/gnoweb/markdown/ext_heading.go:45-49` — **Nested `<a>` tags when heading contains an inline link.** Adversarial test input `## Title with [link](/r/foo)` renders:
  ```
  <h2 id="…"><a class="heading-anchor" …>Title with <a href="/r/foo">link<span …>…</span></a></a></h2>
  ```
  Per WHATWG HTML spec, `<a>` content model is *transparent but must not contain interactive content descendants* — nested `<a>` is invalid. Browsers auto-close the outer `<a>` at the inner `<a>`, splitting the heading-anchor into a short leading fragment plus a lifeless tail. Anchor click then no longer updates the hash for most of the heading, and the user-facing behavior regresses depending on where in the heading they click. The ADR itself lists this exact problem under "Alternatives Considered → 1" as the reason to *not* use this approach. This is the core correctness regression vs. the previous empty-anchor design. Either revert to the empty-anchor `::after §` approach, or skip emitting the heading-anchor wrapper when the heading AST contains `ast.KindLink` / `ast.KindAutoLink` children (walk children on `entering`).

- [ ] `gno.land/pkg/gnoweb/markdown/ext_heading.go:50-54` — **Unbalanced `</a>` when heading has no `id`.** `else` branch writes `</a>` unconditionally. If the heading lacks an `id` attribute (no `parser.WithAutoHeadingID()` in the parser config, or a custom user attribute block that clears it), output is `<h1>Hello</a></h1>`. Production (`render_config.go:44`) currently sets `WithAutoHeadingID`, so this is latent — but the extension now silently requires it. Adversarial test confirms `<h1>Hello</a></h1>` when option is omitted. Fix: track `opened` in local var from `entering` and only close if opened; or wrap the `</a>` behind the same `hasID && len(idBytes) > 0` check.

- [ ] `gno.land/pkg/gnoweb/markdown/ext_heading.go:47` — **Accessibility regression from `aria-label`.** `aria-label="Link to this section"` on an `<a>` that wraps the heading's visible text overrides the element's accessible name. Per ARIA name computation, the heading's accessible name becomes "Link to this section" instead of the actual heading text, because the name is taken from the descendant `<a>` whose name is forced by `aria-label`. Screen-reader heading navigation (rotor / H key) will announce every heading as "Link to this section" — users lose the ability to navigate by heading title. The ADR even flags the original `aria-hidden` as the accessibility-correct choice. Options: drop `aria-label` entirely and let the anchor inherit its accessible name from the text (best); or keep the empty-anchor design. `aria-label` on a link that wraps visible text is an anti-pattern (WCAG 2.5.3 "Label in Name" also flagged).

## Warnings (should fix)

- [ ] `gno.land/adr/prxxxx_heading_anchor_links.md:13-17` — ADR describes the **old** empty-`<a>` + `::after §` approach ("append an empty `<a class="heading-anchor" href="#id" aria-hidden="true"></a>`", "avoids nesting `<a>` tags"). Current code does the opposite — wraps heading text and uses `aria-label`. The "Alternatives Considered → 1. Wrap heading content in `<a>`: Would nest `<a>` tags when headings contain links — invalid HTML per spec" is now the chosen implementation. ADR must be rewritten to match current design, or design reverted to match ADR.
- [ ] `gno.land/adr/prxxxx_heading_anchor_links.md` (filename) — placeholder `prxxxx`. Rename to `pr5585_heading_anchor_links.md` before merge.
- [ ] `gno.land/pkg/gnoweb/frontend/css/05-composition.css:262-265` — `.c-realm-view :is(h1,h2,h3,h4):hover .heading-anchor::after` sets `color` + `font-weight` but omits `content:`. Without `content:`, `::after` never renders — rule is dead. Either add `content: " §"` (consistent with `c-doc-view:659`) or remove the `::after` block. If the hover-§ indicator was intentionally dropped in c-realm-view (since the heading text is now itself clickable), delete the dead rule.
- [ ] CI failing: `public/main.css` bundle diverges from CI-generated output. `make gnoweb.generate` locally before pushing, or investigate toolchain drift (browserslist caniuse-lite in CI log is "8 months old" — may not be the cause, but worth noting).

## Nits

- [ ] `gno.land/pkg/gnoweb/markdown/ext_heading.go:37,53` — `"0123456"[n.Level]` is a fine compact trick but silently panics for `n.Level == 0` or `>6`. Goldmark guarantees 1–6 for atx/setext; leave as-is or add an `if n.Level < 1 || n.Level > 6 { n.Level = 1 }` guard only if you care about robustness to malformed AST.
- [ ] `ext_heading.go:17-25` — `newHeadingRenderer` accepts `...html.Option` but none are passed at registration site (`ext_heading.go:65`). The options plumbing is unused; drop it or wire it up.
- [ ] `ext_heading.go:63-67` — extension registers at priority 1 (lower = earlier in goldmark's priority order). goldmark overwrites prior registrations for the same kind, so this works, but relying on registration-order override is brittle. Worth a one-line comment explaining the intent.

## Missing Tests

- [ ] Heading containing an inline link — currently produces nested `<a>` in `ext_link/inline.md.txtar:22` but no assertion flags this as wrong. Add a dedicated `ext_heading` golden case that explicitly asserts no nested anchors (or asserts the fixed behavior once chosen).
- [ ] Heading containing only images / only inline code — verify the anchor wraps correctly and doesn't produce odd DOM.
- [ ] Heading rendered without `parser.WithAutoHeadingID()` — lock down expected behavior (either "no anchor wrapper, no stray `</a>`" or "requires AutoHeadingID, documented").
- [ ] Heading with duplicate title (e.g. two `## Foo`) — goldmark assigns `foo`, `foo-1`; confirm anchors match.
- [ ] Screen-reader / accessibility snapshot if the aria-label question is resolved; at minimum a unit assertion on the attribute choice.
- [ ] JS-side: there's no test that clicking the rendered anchor actually sets `window.location.hash`. The feature is now pure-HTML so browsers handle it, but a Playwright/integration smoke test would catch the nested-anchor regression at click time.

## Suggestions

- Prefer the **empty-anchor + `::after §`** original design (what the ADR still describes). It's the only approach that sidesteps both the nested-`<a>` problem and the aria-label accessibility problem, and it already worked. The current PR solves a real issue (#5579) — clicking heading text doesn't navigate — but so does the empty anchor: users click the `§` indicator on hover. Alternative: add JS to intercept heading click and update `window.location.hash` manually, no DOM nesting needed.
- If you keep the wrap-text design, gate the wrapper on absence of link descendants:
  ```go
  hasLinkDescendant := false
  ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
      if entering && (n.Kind() == ast.KindLink || n.Kind() == ast.KindAutoLink) {
          hasLinkDescendant = true
          return ast.WalkStop, nil
      }
      return ast.WalkContinue, nil
  })
  ```
  Skip the anchor wrapping in that case, fall back to empty-anchor style. Keeps the common case click-anywhere-to-hash while staying valid HTML when needed.
- Track an `opened bool` local and propagate between enter/exit via a struct field keyed by node pointer, or use `node.SetAttribute("_heading_anchor_opened", true)` on entering. Cleanest: compute `shouldWrap` once on entering, store as a bool on the node's own attributes, read on exit.

## Questions for Author

- Why the switch from the original empty-anchor design to wrap-text? The original ADR text suggests that design was deliberate. Was there user feedback that `§`-on-hover wasn't discoverable?
- Is `aria-label="Link to this section"` intentional despite overriding the heading's accessible name, or was it copied from a source that assumed the anchor wraps a non-text element (e.g. an icon)?
- What's the plan for the failing `gnoweb_generate` / `main / build` CI? Do you run `make gnoweb.generate` locally? If the bundled `main.css` is sensitive to local toolchain versions, maybe worth noting in the PR.

## Verdict

**REQUEST CHANGES** — two independently-critical correctness bugs (nested `<a>` on headings with inline links; unbalanced `</a>` without AutoHeadingID), one accessibility regression (`aria-label` on text-wrapping link), a stale ADR that describes the rejected design, a dead CSS rule, and failing CI. The feature itself is worthwhile; recommend either reverting to the empty-anchor design (preferred — matches ADR, sidesteps all three bugs) or guarding the wrap with a link-descendant check and dropping `aria-label`.
