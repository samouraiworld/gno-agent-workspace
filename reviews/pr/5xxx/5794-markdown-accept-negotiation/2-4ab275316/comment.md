# Review: PR #5794
Posted: https://github.com/gnolang/gno/pull/5794#pullrequestreview-4477498171
Event: APPROVE

## Body
Clean and well-tested; verified on 4ab275316. Nothing blocking.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5794-markdown-accept-negotiation/2-4ab275316/claude-opus-4-8_davd-gzl.md · [↗](claude-opus-4-8_davd-gzl.md)

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
[`negotiate_test.go:31`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/negotiate_test.go#L31) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate_test.go#L31) asserts `want: true` for this header.
</details>

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:207 [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L207)
This write serves realm `Render()` bytes verbatim (the HTML path sanitizes through goldmark) and gnoweb sets no `nosniff` anywhere. Add `X-Content-Type-Options: nosniff` here.

*(AI Agent)*
