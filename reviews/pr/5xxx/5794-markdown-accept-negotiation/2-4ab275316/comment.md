# Review: PR #5794
Event: APPROVE

## Body
Realm pages and static-markdown aliases are now served as raw `Render()` markdown when the `Accept` header names `text/markdown`, with `Vary: Accept` so caches key both representations. The check never matches `*/*` or `text/*`, so browsers keep getting HTML, and agent fetchers get the small markdown payload with no configuration.

APPROVE. Nothing blocking, code is clean and the negotiation rules are thoroughly table-tested. Open items:

- Markdown wins even when the client ranks HTML strictly higher; confirm it's intended (inline comment).
- Optional `nosniff` on the new raw-bytes write (inline comment).
- Repo policy: `AGENTS.md` requires an ADR for non-trivial AI-assisted PRs, and this PR has none.
- Source, help, directory and user views still return HTML "for now". Is the markdown follow-up tracked anywhere?

Repros run at 4ab275316.
Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5794-markdown-accept-negotiation/2-4ab275316/claude-opus-4-8_davd-gzl.md

*(AI Agent)*

## gno.land/pkg/gnoweb/negotiate.go:15-32 [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate.go#L15)
`text/html;q=0.9, text/markdown;q=0.8` returns markdown: any non-zero q wins, regardless of how the client ranks the alternatives. That is a shortcut over RFC 9110 preference ordering, harmless for the agent use case since browsers never send `text/markdown`. Confirming it's a choice, not an oversight.

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
The table asserts `want: true` for this header, locking the behavior in.
</details>

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:207 [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L207)
Optional: add `X-Content-Type-Options: nosniff` here. This write serves realm `Render()` output verbatim, while the HTML path sanitizes through goldmark. Risk is low because the path requires an explicit `Accept: text/markdown`, but gnoweb sets no `nosniff` anywhere today and this is its one raw-bytes surface.

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
