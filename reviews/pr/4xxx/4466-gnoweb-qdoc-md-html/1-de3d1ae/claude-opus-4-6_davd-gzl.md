# PR #4466: fix(gnoweb): vm/qdoc md to html in action template

**URL:** https://github.com/gnolang/gno/pull/4466
**Author:** alexiscolin | **Base:** master | **Files:** 98 | **+2423 -244**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes issue #4417 where gnoweb help pages (`$help`) displayed raw markdown escaping (e.g., `\_`) because the `vm/qdoc` endpoint returns markdown but gnoweb was injecting it directly into HTML templates without conversion.

The solution introduces a dual-renderer architecture in `HTMLRenderer`: one Goldmark instance for realms (full features: strikethrough, tables, footnotes, task lists, syntax highlighting) and one for documentation (minimal: only `ExtCodeExpand` for collapsible code blocks and `ExtLinks` for Gno URL resolution). This separation ensures documentation rendering doesn't inherit realm-specific markdown extensions.

Key changes:
- **`gnovm/pkg/doc/json_doc.go`**: Adds `normalizedMarkdownPrinter()` and `convertIndentedCodeBlocksToFenced()` to convert Go doc's 4-space/tab-indented code blocks into fenced ` ```go ` blocks so Chroma can syntax-highlight them.
- **`gno.land/pkg/gnoweb/render.go`**: New `RenderDocumentation()` method with dedicated Goldmark instance.
- **`gno.land/pkg/gnoweb/render_config.go`**: Separate `NewRealmGoldmarkOptions()` and `NewDocumentationGoldmarkOptions()` factory functions.
- **`gno.land/pkg/gnoweb/markdown/extensions/doc/ext_codeexpand.go`**: New extension that wraps code blocks in `<details><summary>Example</summary>` elements with Chroma syntax highlighting.
- **`gno.land/pkg/gnoweb/chroma/config.go`**: Centralizes Chroma formatter/style configuration shared across all rendering paths.
- **`gno.land/pkg/gnoweb/markdown/extensions/context_key.go`**: Refactors `gUrlContextKey` from the markdown package to a shared `GnoURLContextKey` with exported `GetUrlFromContext` helper.
- **`gno.land/pkg/gnoweb/components/ui/help_function.html`**: New template for function documentation display, using `noescape_string` to inject pre-rendered HTML.
- **Frontend**: Adds `input.css` (964 lines) and `tx.config.js` (115 lines) — Tailwind CSS configuration files that don't exist on master.

## Test Results
- **Existing tests:** PASS — `gnovm/pkg/doc` and `gno.land/pkg/gnoweb` all pass
- **Edge-case tests:** skipped

## Critical (must fix)

- [ ] `gno.land/pkg/gnoweb/markdown/extensions/doc/ext_codeexpand.go:21` — **Data race on `lexerCache`**: Package-level `map[string]chroma.Lexer` is read/written without synchronization. Multiple concurrent Goldmark renders (e.g., parallel HTTP requests) will race on this map. Either use `sync.Map`, protect with `sync.RWMutex`, or remove the cache entirely (lexer creation is cheap — `lexers.Get()` already uses a registry).

## Warnings (should fix)

- [ ] `gnovm/pkg/doc/json_doc.go:388` — **Hardcoded ` ```go ` language tag for all indented code blocks**: Go doc comments can contain non-Go code (shell commands, JSON, plain text). Tagging all indented blocks as Go causes incorrect Chroma highlighting. This was raised by thehowl in review. Use bare ` ``` ` (no language) to let code render without syntax highlighting, matching the original indented-block appearance.
- [ ] `gnovm/pkg/doc/json_doc.go:385` — **Blank lines inside indented code blocks break them**: The logic `isCode := strings.HasPrefix(line, "\t") || (len(line) >= 4 && line[:4] == "    ")` treats empty/blank lines as non-code. In Go doc convention, a blank line between two indented blocks is still part of the same code block. This implementation will close and reopen fences around each paragraph of code, producing multiple separate fenced blocks instead of one continuous block.
- [ ] `gnovm/pkg/doc/json_doc.go:413` — **Error message in French**: `"lecture failed"` should be `"scan failed"` or `"read failed"`. The codebase uses English throughout.
- [ ] `gno.land/pkg/gnoweb/markdown/extensions/doc/ext_codeexpand.go:69` — **Default language `"go"` for untagged code blocks**: When `language` is empty (line 69-71), the lexer defaults to `"go"`. Combined with the hardcoded ` ```go ` in `json_doc.go`, ALL code blocks get Go highlighting regardless of actual content. Should either default to a plaintext lexer or skip highlighting entirely for untagged blocks.
- [ ] `gno.land/pkg/gnoweb/markdown/extensions/doc/ext_codeexpand.go:87` — **Swallowed error on highlight failure**: The `if err != nil || chromaFormatter.Format(...)` pattern silently falls back to `<pre><code>` on any Chroma error. While the fallback is reasonable, the error should be logged so rendering issues are diagnosable.

## Nits

- [ ] `gno.land/pkg/gnoweb/chroma/config.go:29` — `init()` function for shared state could use `sync.Once` + lazy initialization instead. This is a minor style preference; the current approach works but couples initialization to import time.
- [ ] `gno.land/pkg/gnoweb/markdown/extensions/doc/ext_codeexpand.go:41` — Empty class attribute `class=""` on the inner div serves no purpose. Remove or add the intended class.
- [ ] `gno.land/pkg/gnoweb/markdown/extensions/doc/ext_codeexpand.go:58-62` — String concatenation in a loop (`codeContent += string(...)`) for building code content. Use `strings.Builder` or `bytes.Buffer` for better performance with large code blocks.
- [ ] `gno.land/pkg/gnoweb/frontend/css/input.css` and `gno.land/pkg/gnoweb/frontend/css/tx.config.js` — These 1079 lines of new Tailwind CSS configuration files don't exist on master. Their relationship to the qdoc markdown fix is unclear. If they're needed for the `help_function.html` template styling, the PR description should explain this. If unrelated, they should be in a separate PR.

## Missing Tests

- [ ] **Data race test for `lexerCache`**: No concurrent rendering test exists. Running `go test -race` with parallel `RenderDocumentation` calls would expose the race on the package-level map (`ext_codeexpand.go:21`).
- [ ] **Blank-line-in-code-block test**: `json_doc_test.go` tests basic indented code conversion but doesn't test a code block containing blank lines between indented paragraphs (the split-block bug at `json_doc.go:385`).
- [ ] **Non-Go code in doc comments**: No test for indented code blocks containing shell commands, JSON, or other non-Go content — which would be incorrectly highlighted as Go.
- [ ] **Tab-indented vs space-indented mixed blocks**: `normalizeCodeBlockStream` handles both tab and 4-space indentation separately but doesn't test mixed indentation within a single code block.

## Suggestions

- Replace `lexerCache` with `sync.Map` or a `sync.RWMutex`-protected map to prevent data races under concurrent use (`ext_codeexpand.go:21`).
- Consider using `go/doc/comment` package's built-in parser instead of hand-rolling indented-to-fenced conversion. The `go/doc/comment` package understands Go doc conventions (including blank lines within code blocks) and could produce cleaner markdown output (`json_doc.go:362-421`).
- The `normalizeCodeBlockStream` function should track whether the previous line was code to handle blank lines within code blocks — a blank line between two indented lines should remain inside the fence (`json_doc.go:380-410`).
- The `noescape_string` template function bypasses Go's `html/template` auto-escaping. While Goldmark strips raw HTML by default (confirmed by tests at `handler_http_test.go:893-901`), this relies on the `UnsafeHTML` option never being enabled for the documentation renderer. Consider adding an explicit comment or assertion that `WithUnsafe()` must not be applied to the documentation Goldmark instance.

## Questions for Author

- What is the purpose of the new `input.css` and `tx.config.js` files? Are they required for this PR's functionality or should they be split into a separate PR?
- For the indented-to-fenced code block conversion: have you considered using bare ` ``` ` (no language tag) instead of ` ```go ` to handle the case where Go doc comments contain non-Go code examples?
- Is the `lexerCache` in `ext_codeexpand.go` expected to be accessed concurrently? If so, it needs synchronization.

## Verdict

REQUEST CHANGES — The unsynchronized `lexerCache` map is a data race under concurrent HTTP requests, and the code block conversion has correctness issues with blank lines and hardcoded Go language tags that will produce incorrect output for non-trivial documentation.
