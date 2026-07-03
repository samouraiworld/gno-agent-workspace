# Review: PR [#5897](https://github.com/gnolang/gno/pull/5897)
Posted: https://github.com/gnolang/gno/pull/5897#pullrequestreview-4627000703
Event: APPROVE

## Body
Verified on ee9f7e0b0: each cap rejects the oversized input it guards, and an expression at the 64 KiB limit still passes. The suite covers none of these paths.

The sizes you asked about are fine: real pkg paths are dozens of bytes and expressions are short calls, so 1024 and 64 KiB leave plenty of headroom.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5897-serveeval-json-size-cap/1-ee9f7e0b0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoweb/feature/playground/handler.go:225 [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler.go#L225) [posted](https://github.com/gnolang/gno/pull/5897#discussion_r3520928469)
Missing test: the three new caps have no coverage. Add to [`TestHandlerPlaygroundEval`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L100-L141):

<details><summary>test cases</summary>

```go
{
	name:       "oversized body",
	method:     http.MethodPost,
	body:       `{"pkg_path":"r/x","expression":"` + strings.Repeat("a", maxEvalBodyBytes+1) + `"}`,
	wantStatus: http.StatusBadRequest,
	wantError:  "invalid request body",
},
{
	name:       "oversized pkg_path",
	method:     http.MethodPost,
	body:       `{"pkg_path":"` + strings.Repeat("a", maxEvalPkgPathLen+1) + `","expression":"Render(\"\")"}`,
	wantStatus: http.StatusBadRequest,
	wantError:  "too long",
},
{
	name:       "oversized expression",
	method:     http.MethodPost,
	body:       `{"pkg_path":"r/x","expression":"` + strings.Repeat("a", maxEvalExpressionLen+1) + `"}`,
	wantStatus: http.StatusBadRequest,
	wantError:  "too long",
},
```
</details>
