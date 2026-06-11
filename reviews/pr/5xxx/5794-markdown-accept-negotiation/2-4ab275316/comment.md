# Review: PR #5794
Event: APPROVE

## Body
Clean and well-tested; verified on the current head (4ab275316). Nothing blocking:

- Markdown wins even when the client ranks HTML strictly higher. Inline comment; confirm intended.
- The raw markdown write skips sanitization; worth a `nosniff`. Inline comment.
- Is the markdown follow-up for source/help/directory/user views tracked anywhere?

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5794-markdown-accept-negotiation/2-4ab275316/claude-opus-4-8_davd-gzl.md

*(AI Agent)*

## gno.land/pkg/gnoweb/negotiate.go:15-32 [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate.go#L15)
`text/html;q=0.9, text/markdown;q=0.8` returns markdown: any non-zero q wins regardless of client ranking, a shortcut over RFC 9110 preference ordering. Harmless since browsers never send `text/markdown`; confirming it's intentional.

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
```
The table asserts `want: true` for this header.
</details>

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:207 [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L207)
This write serves realm `Render()` bytes verbatim (the HTML path sanitizes through goldmark) and gnoweb sets no `nosniff` anywhere. Add `X-Content-Type-Options: nosniff` here.

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
