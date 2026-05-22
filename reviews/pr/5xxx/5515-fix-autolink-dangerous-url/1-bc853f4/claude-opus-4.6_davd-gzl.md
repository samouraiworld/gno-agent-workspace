# PR #5515: fix(gnoweb): apply IsDangerousURL to angle-bracket autolinks

**URL:** https://github.com/gnolang/gno/pull/5515
**Author:** thehowl | **Base:** master | **Files:** 2 | **+47 -9**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a security vulnerability in gnoweb's markdown renderer where angle-bracket autolinks (e.g. `<javascript:alert(1)>`) bypassed `IsDangerousURL` sanitization, enabling XSS.

**Background:** Goldmark's default `renderAutoLink` (in `renderer/html/html.go:502-524`) does NOT call `IsDangerousURL` — it writes autolink URLs directly into `href` attributes with only URL-escaping and HTML-escaping. The existing `linkTransformer` only handled `*ast.Link` nodes (from `[text](url)` syntax), so `*ast.AutoLink` nodes (from `<scheme:...>` syntax) fell through to goldmark's default renderer. This meant `<javascript:alert(1)>` in realm markdown would render as a live `<a href="javascript:alert(1)">` — a direct XSS vector.

Goldmark's autolink parser recognizes any scheme matching `[A-Za-z][A-Za-z0-9.+-]{1,31}:`, so `javascript:`, `vbscript:`, `file:`, and `data:` autolinks are all parsed and were all vulnerable.

**The fix:** Extends `linkTransformer.Transform` to also intercept `*ast.AutoLink` nodes via a type switch. For each autolink, it:
1. Extracts the URL via `n.URL(source)` (and prepends `mailto:` for email autolinks)
2. Builds a synthetic `ast.Link` with the URL as destination
3. Creates an `ast.String` child with the label text (from `n.Label(source)`)
4. Wraps the synthetic link in a `GnoLink` and replaces the original `AutoLink` node in the AST

This routes all autolinks through the existing `renderGnoLink` code path, which calls `html.IsDangerousURL` before writing `href`, adds `rel="noopener nofollow ugc"` for external links, and appends link-type icons/tooltips.

**Files changed:**
- `gno.land/pkg/gnoweb/markdown/ext_links.go` — Core fix: refactors the walker from a simple `node.(*ast.Link)` type assertion to a `switch n := node.(type)` with `case *ast.Link` and `case *ast.AutoLink` branches. Adds `source := reader.Source()` to extract autolink URL/label from the source buffer.
- `gno.land/pkg/gnoweb/markdown/golden/ext_link/autolink.md.txtar` — New golden test covering `https:`, `mailto:` (email), `javascript:`, and `data:` autolinks. Dangerous schemes produce `href=""`, safe ones render normally with full link metadata.

**Blast radius:** Contained entirely within the `markdown` package. `ExtLinks` is only consumed by `ext.go:75` where it's wired into `NewGnoExtension()`. No external callers of any changed symbols exist outside the package.

## Test Results
- **Existing tests:** PASS — all 37 golden tests in `gno.land/pkg/gnoweb/markdown/` pass (0.023s), including the new `autolink.md.txtar`
- **CI:** All checks green (build, lint, test, codecov). Only "Merge Requirements" pending codeowner approval (expected — needs alexiscolin or gfanton).
- **Codecov:** All modified and coverable lines are covered by tests
- **Edge-case tests:** skipped

## Critical (must fix)

None

## Warnings (should fix)

None

## Nits

- [ ] `gno.land/pkg/gnoweb/markdown/ext_links.go:97` — `source := reader.Source()` is fetched unconditionally at the top of `Transform`, even when there are no `AutoLink` nodes in the document. This is harmless (it just returns a slice header), but moving it inside the `case *ast.AutoLink:` branch would make it clearer that only autolinks need the source buffer. Very minor.

## Missing Tests

- [ ] `vbscript:` and `file:` autolinks — `IsDangerousURL` blocks four scheme families (`javascript:`, `vbscript:`, `file:`, `data:`) but only two are tested in the golden file. Adding `<vbscript:MsgBox(1)>` and `<file:///etc/passwd>` would document the full set. — `gno.land/pkg/gnoweb/markdown/golden/ext_link/autolink.md.txtar`
- [ ] Safe `data:image/png;base64,...` autolink — `IsDangerousURL` explicitly allows `data:image/{png,gif,jpeg,webp,svg+xml};` URLs. A test confirming a safe data-image autolink renders with a real `href` (not empty) would document this exception.
- [ ] Autolinks inside other block elements (blockquote, list item, heading) to confirm the `ast.Walk` handles nested structures. The walker should handle these correctly, but a test would prevent regressions.

## Suggestions

- Consider adding a brief comment at `ext_links.go:124-128` explaining why a synthetic `ast.Link` is needed: `AutoLink` is a leaf node in goldmark's AST with no children, so we must construct the link and label child manually. This would help future maintainers understand the design.
- A comment at `ext_links.go:127` explaining `labelNode.SetRaw(true)` prevents double-escaping of the label text would improve readability.
- The `mailto:` question below is worth considering for a follow-up.

## Questions for Author

- Is it intentional that email autolinks now get `rel="noopener nofollow ugc"`? The `mailto:` scheme is classified as `GnoLinkTypeExternal` by `detectLinkType` (because `weburl.ParseFromURL` fails for `mailto:` and the scheme is non-empty, hitting the `return nil, GnoLinkTypeExternal` path). The original goldmark email autolink renderer would not add these attributes. This is arguably a defensible security posture but changes the rendering of email autolinks beyond just the XSS fix.

## Verdict

**APPROVE** — Clean, minimal, well-targeted security fix that closes a real XSS vector. The approach of converting autolinks to synthetic `ast.Link` nodes and reusing the existing `renderGnoLink` pipeline is the right design — it avoids duplicating the sanitization/rendering logic and ensures all link types get consistent treatment. The implementation is correct: buffer aliasing is safe (no mutations), email autolinks correctly get a `mailto:` prefix, and the golden test covers the key dangerous schemes. The missing test cases for `vbscript:` and `file:` are nice-to-haves, not blockers.
