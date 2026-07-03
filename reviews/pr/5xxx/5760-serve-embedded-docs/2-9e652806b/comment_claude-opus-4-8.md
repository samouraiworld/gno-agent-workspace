# Review: PR [#5760](https://github.com/gnolang/gno/pull/5760)
Event: REQUEST_CHANGES

## Body
The master merge on 9e652806b is only half-reconciled: two upstream changes never landed in the PR's own files. Every red CI job (`main / test`, `main / lint`, `gnodev`, `gnobro`, `e2e-test`, docker) fails on the same `docs.go` compile error below, nothing else. Reproduced on 9e652806b.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5760-serve-embedded-docs/2-9e652806b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoweb/docs.go:108-112 [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L108)
gnoweb fails to build here: master [#5554](https://github.com/gnolang/gno/pull/5554) made `FooterData.Analytics` an `AnalyticsData` struct and dropped `AssetsPath`/`BuildTime`, but this literal and the one in `renderError` at [200-203](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs.go#L200-L203) still use the old shape. [`handler_http.go:216`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/handler_http.go#L216) shows the current form to copy: `Analytics: components.AnalyticsData{Enabled: h.Static.Analytics}` with the two fields dropped.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5760 -R gnolang/gno
go build ./gno.land/pkg/gnoweb/
```

```
# github.com/gnolang/gno/gno.land/pkg/gnoweb
gno.land/pkg/gnoweb/docs.go:109:16: cannot use h.Static.Analytics (variable of type bool) as components.AnalyticsData value in struct literal
gno.land/pkg/gnoweb/docs.go:110:4: unknown field AssetsPath in struct literal of type components.FooterData
gno.land/pkg/gnoweb/docs.go:111:4: unknown field BuildTime in struct literal of type components.FooterData
gno.land/pkg/gnoweb/docs.go:201:16: cannot use h.Static.Analytics (variable of type bool) as components.AnalyticsData value in struct literal
gno.land/pkg/gnoweb/docs.go:202:4: unknown field AssetsPath in struct literal of type components.FooterData
gno.land/pkg/gnoweb/docs.go:203:4: unknown field BuildTime in struct literal of type components.FooterData
```
</details>

## docs/sidebar_test.go:60 [↗](../../../../../.worktrees/gno-review-5760/docs/sidebar_test.go#L60)
`TestSidebarFromReadme` hardcodes the README section title `Resources`, which master renamed to `References` in [`docs/README.md`](https://github.com/gnolang/gno/blob/9e652806b/docs/README.md?plain=1#L36), so `go test ./docs/` fails. The exact-label assertion is brittle: the item-resolves-to-embedded-`.md` check below it already covers the invariant that matters.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5760 -R gnolang/gno
go test ./docs/ -run TestSidebarFromReadme
```

```
--- FAIL: TestSidebarFromReadme (0.00s)
    sidebar_test.go:67: section[2] = "References", want "Resources"
FAIL
FAIL	github.com/gnolang/gno/docs	0.003s
```
</details>

## gno.land/pkg/gnoweb/docs.go:59 [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L59)
`http.FileServer` serves a bare stdlib HTML index for asset directories: `/docs/_assets/`, `/docs/_assets/minisocial/`, and `/docs/images/` return 200 with an unstyled `<pre>` listing and no site layout. This is the one `/docs` surface not wrapped in `IndexLayout`, breaking the consistent-look goal.

## gno.land/pkg/gnoweb/docs.go:143 [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L143)
No test covers the render-failure to 500 branch. A handler with a stub renderer that returns an error would pin the status code and the error copy.

## gno.land/pkg/gnoweb/docs_test.go:75-78 [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs_test.go#L75)
The HEAD case checks status 200 but not that the body is empty. Assert the HEAD body is empty while GET on the same route is non-empty, so a regression that writes a body on HEAD is caught.

## gno.land/pkg/gnoweb/docs.go:39-41 [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L39)
This comment says the sidebar and admonition syntax are unshipped follow-ups, but this PR ships both.

## gno.land/pkg/gnoweb/docs.go:351 [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L351)
Nested admonitions are not handled: an inner `:::note` inside an open block emits a literal `> :::note` line, and the first `:::` closes the outer block. No docs page nests today, so a note in the function comment marking it a known limitation would suffice.

## gno.land/pkg/gnoweb/app.go:30 [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/app.go#L30)
Removing the `/docs` to `/u/docs` alias leaves no test pinning that `/docs` now renders the README instead of redirecting.

## docs/docs_test.go:104-107 [↗](../../../../../.worktrees/gno-review-5760/docs/docs_test.go#L104)
`TestInternalLinksResolve` skips image and asset link targets, so a broken `![](images/foo.png)` reference ships silently even though `images/` and `_assets/` are embedded wholesale.
