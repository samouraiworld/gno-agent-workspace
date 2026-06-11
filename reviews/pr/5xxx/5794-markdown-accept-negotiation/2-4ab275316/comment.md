# Review: PR #5794
Event: APPROVE

## Body
gnoweb now returns a realm's raw `Render()` markdown when the client's `Accept` header names `text/markdown`, and HTML otherwise; `Vary: Accept` keeps shared caches correct. Browsers are unaffected (the path never matches `*/*` or `text/*`), and agent fetchers get the small markdown payload with no config. Clean, vet-clean, thoroughly table-tested.

APPROVE from me. Nothing blocking. Two small things below (a q-preference confirm and an optional `nosniff`), plus a repo-policy note: `AGENTS.md` asks for an ADR on non-trivial AI-assisted PRs, and this one has none. A follow-up question too: source/help/directory/user views fall back to HTML "for now" — is extending markdown there tracked anywhere?

Repros run at 4ab275316.
Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5794-markdown-accept-negotiation/2-4ab275316/claude-opus-4-8_davd-gzl.md

*(AI Agent)*

## gno.land/pkg/gnoweb/negotiate.go:15-32 [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate.go#L15)
Markdown wins whenever it appears with q>0, even when another type is ranked strictly higher: `text/html;q=0.9, text/markdown;q=0.8` returns markdown. That's a deliberate shortcut, not strict RFC 9110 preference ordering. Fine for the agent use case (browsers never send `text/markdown`); just confirming it's intended, not an oversight.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5794 -R gnolang/gno
go test ./gno.land/pkg/gnoweb/ -run 'TestNegotiatesMarkdown/markdown_present_with_non-zero_q_among_others' -v
```
```
=== RUN   TestNegotiatesMarkdown/markdown_present_with_non-zero_q_among_others
--- PASS: TestNegotiatesMarkdown/markdown_present_with_non-zero_q_among_others (0.00s)
PASS
ok  	github.com/gnolang/gno/gno.land/pkg/gnoweb	0.024s
```
The case asserts `want: true` for `text/html;q=0.9, text/markdown;q=0.8`, locking in markdown-over-higher-q-HTML.
</details>

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:207 [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L207)
Optional: set `X-Content-Type-Options: nosniff` on this raw markdown write. It serves realm `Render()` bytes verbatim (the HTML path sanitizes via goldmark). Real risk is low since the path needs an explicit `Accept: text/markdown`, but gnoweb sets no `nosniff`/CSP anywhere today, so it's a cheap guard on the one new raw-bytes surface.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5794 -R gnolang/gno
grep -rn 'nosniff\|X-Content-Type-Options' gno.land/pkg/gnoweb/ || echo 'no nosniff header anywhere in gnoweb'
```
```
no nosniff header anywhere in gnoweb
```
</details>

*(AI Agent)*
