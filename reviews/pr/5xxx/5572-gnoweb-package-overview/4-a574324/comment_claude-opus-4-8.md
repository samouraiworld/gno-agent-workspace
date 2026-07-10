# Review: PR [#5572](https://github.com/gnolang/gno/pull/5572)
Event: APPROVE

## Body
Reproduced on a574324. Verified against a live node: the overview renders, its symbol links land on the exact declaration lines in the source view, and the sidebar counts match the sections shown.

The red `main / test` is [`TestFiles/alloc_7.gno`](https://github.com/gnolang/gno/blob/a574324/gnovm/tests/files/alloc_7.gno), an allocator byte-count golden this PR does not touch. Not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5572-gnoweb-package-overview/4-a574324/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoweb/components/overview_license.go:12-16 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_license.go#L12)
Real LICENSE files wrap between the license title and its version, and between `without` and `modification`, so the GPL, AGPL and BSD signatures never match and the sidebar shows no license kind. The existing case at [`overview_license_test.go:60-62`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/components/overview_license_test.go#L60-L62) passes only because it feeds the BSD clause as one unwrapped line. A dotall flag closes the GPL pair but not the BSD pair, where the wrap falls inside the literal anchor phrase.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5572 -R gnolang/gno

cat > gno.land/pkg/gnoweb/components/zz_license_repro_test.go <<'EOF'
package components

import (
	"fmt"
	"testing"
)

func TestLicenseWrapped(t *testing.T) {
	// Headers copied verbatim from real LICENSE files, line wrapping included.
	cases := []struct{ want, body string }{
		{"GPL-3.0", "                    GNU GENERAL PUBLIC LICENSE\n                       Version 3, 29 June 2007\n"},
		{"AGPL-3.0", "                    GNU AFFERO GENERAL PUBLIC LICENSE\n                       Version 3, 19 November 2007\n"},
		{"BSD-2-Clause", "Redistribution and use in source and binary forms, with or without\nmodification, are permitted provided that the following conditions are met:\n"},
		{"Apache-2.0", "                                 Apache License\n                           Version 2.0, January 2004\n"},
	}
	for _, tc := range cases {
		got := deriveLicense([]string{"LICENSE"}, fileContentFn(map[string][]byte{"LICENSE": []byte(tc.body)}))
		fmt.Printf("want %-13s got Kind=%q\n", tc.want, got.Kind)
	}
}
EOF

go test -run TestLicenseWrapped -v ./gno.land/pkg/gnoweb/components/ | grep 'want '
rm gno.land/pkg/gnoweb/components/zz_license_repro_test.go
```

```
want GPL-3.0       got Kind=""
want AGPL-3.0      got Kind=""
want BSD-2-Clause  got Kind=""
want Apache-2.0    got Kind="Apache-2.0"
```
</details>

## gno.land/pkg/gnoweb/components/overview_symbols.go:27-34 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_symbols.go#L27)
[`json_doc.go:266`](https://github.com/gnolang/gno/blob/a574324/gnovm/pkg/doc/json_doc.go#L266) reports whether a type is an alias, but the overview never reads the flag, so `type A = B` and `type A B` render as identical cards. The two differ in assignability and method sets, so the page tells the reader the wrong thing. No package under `examples/` declares an alias today, so nothing on chain renders wrong yet.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5572 -R gnolang/gno

cat > gno.land/pkg/gnoweb/components/zz_alias_repro_test.go <<'EOF'
package components

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/gnolang/gno/gnovm/pkg/doc"
)

func TestAliasIndistinguishable(t *testing.T) {
	u, _ := weburl.Parse("/r/demo/foo")
	data := BuildOverview(OverviewInput{
		URL:   u,
		Files: []string{"foo.gno"},
		Doc: &doc.JSONDocumentation{Types: []*doc.JSONType{
			{Name: "Defined", Type: "int", Kind: "ident", Alias: false}, // type Defined int
			{Name: "Aliased", Type: "int", Kind: "ident", Alias: true},  // type Aliased = int
		}},
		DocRenderer: noopRenderer{},
		Domain:      "gno.land",
	})
	var buf bytes.Buffer
	if err := OverviewView(data).Render(&buf); err != nil {
		t.Fatal(err)
	}
	card := regexp.MustCompile(`(?s)<article class="b-pkg-symbol" id="type-(?:Defined|Aliased)".*?</article>`)
	strip := regexp.MustCompile(`<[^>]+>`)
	ws := regexp.MustCompile(`\s+`)
	for _, m := range card.FindAllString(buf.String(), -1) {
		name := regexp.MustCompile(`id="type-(\w+)"`).FindStringSubmatch(m)[1]
		text := ws.ReplaceAllString(strip.ReplaceAllString(m, " "), " ")
		text, _, _ = strings.Cut(strings.TrimSpace(text), " copy")
		fmt.Printf("%-8s card renders: %q\n", name, text)
	}
}
EOF

go test -run TestAliasIndistinguishable -v ./gno.land/pkg/gnoweb/components/ | grep 'card renders'
rm gno.land/pkg/gnoweb/components/zz_alias_repro_test.go
```

```
Defined  card renders: "type Defined ident int"
Aliased  card renders: "type Aliased ident int"
```
</details>

## gno.land/pkg/gnoweb/handler_http.go:1108-1123 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L1108)
Missing test: the subpackage goroutine is never exercised. Every overview test stubs `ListPaths` to return nil, so the Directories section is never rendered, and the error swallow at 1113 is unpinned, meaning a regression to `return err` would take the page down on a transient failure with CI still green. The first case below also pins the untested precondition that `buildSubpackages` needs domain-relative paths: pass it the domain-qualified ones and every child is silently dropped.

<details><summary>test cases</summary>

```go
func newOverviewStub(listPaths func(context.Context, string, int) ([]string, error)) *stubClient {
	return &stubClient{
		listFilesFunc: func(context.Context, string) ([]string, error) { return []string{"foo.gno"}, nil },
		docFunc:       func(context.Context, string) (*doc.JSONDocumentation, error) { return &doc.JSONDocumentation{}, nil },
		fileFunc: func(context.Context, string, string) ([]byte, gnoweb.FileMeta, error) {
			return nil, gnoweb.FileMeta{}, gnoweb.ErrClientFileNotFound
		},
		listPathsFunc: listPaths,
	}
}

func TestHTTPHandler_GetOverviewView_RendersSubpackages(t *testing.T) {
	t.Parallel()
	client := newOverviewStub(func(context.Context, string, int) ([]string, error) {
		return []string{"gno.land/r/demo/foo", "gno.land/r/demo/foo/child"}, nil
	})
	cfg := newTestHandlerConfig(t, client)
	cfg.Meta.Domain = "gno.land" // without this the paths stay domain-qualified and every child is dropped
	h, err := gnoweb.NewHTTPHandler(slog.New(slog.NewTextHandler(&testingLogger{t}, nil)), cfg)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/r/demo/foo$source", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "child", "a direct child must reach the Directories section")
}

func TestHTTPHandler_GetOverviewView_DegradedOnListPathsFailure(t *testing.T) {
	t.Parallel()
	client := newOverviewStub(func(context.Context, string, int) ([]string, error) {
		return nil, errors.New("node unavailable")
	})
	cfg := newTestHandlerConfig(t, client)
	h, err := gnoweb.NewHTTPHandler(slog.New(slog.NewTextHandler(&testingLogger{t}, nil)), cfg)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/r/demo/foo$source", nil))
	require.Equal(t, http.StatusOK, rr.Code, "a ListPaths failure must not fail the overview")
	assert.Contains(t, rr.Body.String(), "foo.gno")
}
```
</details>

## gno.land/pkg/gnoweb/components/overview_build.go:24 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_build.go#L24)
A BUG note is dropped whenever its text appears anywhere in the package doc, so a distinct floating note that happens to be a substring of an inline one disappears from the Bugs section.

## gno.land/pkg/gnoweb/handler_http_test.go:571-572 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http_test.go#L571)
The request here is `/r/errsrc$source` with no file, which this PR now routes to `GetOverviewView`, so the test name and both of its comments name a function the test no longer reaches. The sibling `TestHTTPHandler_GetSourceView_NoFiles` was updated for the new routing.

## gno.land/pkg/gnoweb/components/views/overview.html:219 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/views/overview.html#L219)
Import tags emit `b-tag--kind-stdlib`, `-package`, `-realm` and `-external`, but only the base [`.b-tag--kind`](https://github.com/gnolang/gno/blob/a574324/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3268) rule exists and no attribute selector matches the modifier, so the four kinds render identically. The disabled branch at line 224 carries the same class.

## gno.land/pkg/gnoweb/components/views/overview.html:75 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/views/overview.html#L75)
A README that is listed but whose fetch fails leaves this tag and the table-of-contents entry pointing at a `#readme` section that was never emitted, since it renders only on fetch success at line 131.

## gno.land/pkg/gnoweb/frontend/js/controller-filter.ts:47 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/frontend/js/controller-filter.ts#L47)
Method cards are filter items carrying only their own name, so a query that matches just the receiver keeps the type card visible while hiding every method inside it. Filtering `tree` on `/p/nt/avl/v0$source` leaves the `Tree` card with an empty methods block. `a574324` fixed the method-to-type direction; this is its mirror image.

## SKIP gno.land/pkg/gnoweb/components/overview_symbols.go:154-157 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_symbols.go#L154)
Already raised: https://github.com/gnolang/gno/pull/5572#issuecomment-4807460830

A value group is kept when any one of its names is exported, then every name in it is joined into the card title, so a `const` group holding `Exported` and `unexported` renders as `Exported, unexported`. Struct type cards leak the same way, since `JSONType.Type` carries the full declaration text including unexported fields.

## SKIP gno.land/pkg/gnoweb/components/views/overview.html:348-353 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/views/overview.html#L348)
Already raised: https://github.com/gnolang/gno/pull/5572#issuecomment-4807513337

The card prints `type <Name>` in its header, then a kind tag, then a code block holding only the underlying expression, so `type T int` never appears as a single declaration.
