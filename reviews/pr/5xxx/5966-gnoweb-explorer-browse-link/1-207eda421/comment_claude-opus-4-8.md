# Review: PR [#5966](https://github.com/gnolang/gno/pull/5966)
Event: APPROVE

## Body
Verified on 207eda421: booted gnodev from this branch against a real node, and a package-name click opens the realm render while Browse opens the file listing, targets the mock-backed unit tests can't exercise. Restoring the old `{{ .Link }}/` main link reproduces the directory-listing link and fails the new test.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5966-gnoweb-explorer-browse-link/1-207eda421/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoweb/handler_http_test.go:483 [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/handler_http_test.go#L483)
Missing test: pure-package (`/p/`) explorer listings are never exercised, so the gate that hides Open and Action while keeping the always-shown Browse for `/p/` has no coverage. A regression that dropped the `/p/` gate or moved Browse inside it would leave CI green.

<details><summary>test cases</summary>

Green as written; paste beside the new test:

```go
// TestHTTPHandler_ExplorerPathsListBrowsePurePackage verifies the explorer paths-list
// view for a pure /p/ package: the name links to the package (no trailing slash), Browse
// and Source are shown, and Open and Action are hidden.
func TestHTTPHandler_ExplorerPathsListBrowsePurePackage(t *testing.T) {
	t.Parallel()

	subPackage := &gnoweb.MockPackage{
		Domain: "example.com",
		Path:   "/p/mock/sub",
		Files:  map[string]string{"sub.gno": `package sub`},
	}

	handler, err := gnoweb.NewHTTPHandler(
		slog.New(slog.NewTextHandler(&testingLogger{t}, nil)),
		newTestHandlerConfig(t, gnoweb.NewMockClient(subPackage)),
	)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/p/mock/", nil))

	body := rr.Body.String()
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, body, "Packages")
	assert.Contains(t, body, `href="/p/mock/sub">`)
	assert.Contains(t, body, `href="/p/mock/sub/" class="b-inline-btn">Browse</a>`)
	assert.Contains(t, body, `href="/p/mock/sub$source" class="b-inline-btn">Source</a>`)
	assert.NotContains(t, body, `>Open</a>`)
	assert.NotContains(t, body, `>Action</a>`)
}
```
</details>
</content>
