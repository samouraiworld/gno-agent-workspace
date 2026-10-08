// Repro for gnolang/gno#6231 at dd40ef3dc9b92a3df888bb9e70dc21871dcc192a:
// what the server-rendered omnisearch results page (templates/page.html,
// templates/_results.html) tells the reader on seven edge paths.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6231/head && git checkout dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp <this file> gno.land/pkg/gnoweb/feature/omnisearch/zz_b8_results_page_test.go
//	go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB8 -v
//
// Every subtest logs the rendered verdict lines and fails when the page says
// something the backend state contradicts. Uses the package's own mocks
// (search_test.go) and indexer.New for the real client.
package omnisearch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/indexer"
)

var b8Tags = regexp.MustCompile(`<[^>]+>`)

func b8Render(t *testing.T, data SearchData) string {
	t.Helper()
	var b strings.Builder
	if err := NewPageView(data).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return strings.Join(strings.Fields(b8Tags.ReplaceAllString(b.String(), " ")), " ")
}

// 1. A group the backend could not answer renders "Could not answer: ..." and
// the page also prints "Nothing matched." under it.
func TestB8ErrorGroupAlsoSaysNothingMatched(t *testing.T) {
	dir := newDiscoveryDir()
	dir.err = errors.New("dial tcp: connection refused")
	h := newHandlerWithDir(t, newDiscoveryClient(), dir, nil)
	out := b8Render(t, h.build(context.Background(), mustQuery(t, h, "blog", "")))
	could := strings.Contains(out, "Could not answer")
	nothing := strings.Contains(out, "Nothing matched.")
	t.Logf("could-not-answer=%v nothing-matched=%v", could, nothing)
	if could && nothing {
		t.Errorf("page says both %q and %q for one failed backend", "Could not answer", "Nothing matched.")
	}
}

// 2. A truncated listing whose visible part holds no match renders "Nothing
// matched." and no truncation notice: the notice lives inside a group, and
// discover drops every empty group.
func TestB8TruncatedListingWithNoMatchSaysNothingMatched(t *testing.T) {
	dir := newDiscoveryDir()
	dir.truncated = true
	h := newHandlerWithDir(t, newDiscoveryClient(), dir, nil)
	data := h.build(context.Background(), mustQuery(t, h, "zzzz", ""))
	out := b8Render(t, data)
	nothing := strings.Contains(out, "Nothing matched.")
	notice := strings.Contains(out, "Showing part of the chain")
	t.Logf("groups=%d nothing-matched=%v truncation-notice=%v", len(data.Groups), nothing, notice)
	if nothing && !notice {
		t.Errorf("listing truncated, yet the page answers %q with no truncation notice", "Nothing matched.")
	}
}

// 3. On a realm page a bare word runs discovery over the whole chain while the
// header says "Scoped to <realm>".
func TestB8ScopedHeaderOverWholeChainResults(t *testing.T) {
	h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), nil)
	u := parseURL(t, "/r/alice/blog$search&q=blog")
	r := httptest.NewRequest(http.MethodGet, "/r/alice/blog", nil)
	_, view := h.Handle(context.Background(), httptest.NewRecorder(), r, u)
	var b strings.Builder
	if err := view.Render(&b); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(strings.Fields(b8Tags.ReplaceAllString(b.String(), " ")), " ")
	scoped := strings.Contains(out, "Scoped to /r/alice/blog")
	foreign := strings.Contains(out, "/r/bob/blog")
	t.Logf("scoped-header=%v foreign-realm-listed=%v", scoped, foreign)
	if scoped && foreign {
		t.Errorf("header says %q but the results list %q", "Scoped to /r/alice/blog", "/r/bob/blog")
	}
}

// 4. The provenance footer prints the configured indexer URL verbatim,
// userinfo included, through the real client's URL().
func TestB8ProvenanceFooterPrintsIndexerURLVerbatim(t *testing.T) {
	idx := indexer.New("http://ops:s3cret@127.0.0.1:1/graphql/query", "")
	h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), idx)
	data := h.build(context.Background(), mustQuery(t, h, "account:g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5", ""))
	out := b8Render(t, data)
	leaked := strings.Contains(out, "ops:s3cret@")
	someAbove := strings.Contains(out, "Some results above come from an indexer")
	t.Logf("footer-has-password=%v footer-claims-results-above=%v results=%d", leaked, someAbove, len(data.Groups[0].Results))
	i := strings.Index(out, "Some results above")
	if i >= 0 {
		t.Logf("footer: %s", out[i:min(len(out), i+260)])
	}
	if leaked {
		t.Errorf("indexer password rendered into the public page footer")
	}
	if someAbove && !data.HasResults() {
		t.Errorf("footer claims indexer results above, page has none")
	}
}

// 5. A one-character bare word never reaches discovery (MinTermLen 2), and the
// page answers "Nothing matched." rather than naming the floor.
func TestB8ShortBareWordSaysNothingMatched(t *testing.T) {
	h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), nil)
	out := b8Render(t, h.build(context.Background(), mustQuery(t, h, "a", "")))
	t.Logf("nothing-matched=%v (every listed realm contains %q)", strings.Contains(out, "Nothing matched."), "a")
	if strings.Contains(out, "Nothing matched.") {
		t.Errorf("q=a answers %q although /r/alice/blog contains it", "Nothing matched.")
	}
}

// 6. An is: value that is neither realm nor package skips both kinds and the
// page answers "Nothing matched." instead of flagging the value.
func TestB8UnknownIsKindSaysNothingMatched(t *testing.T) {
	h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), nil)
	out := b8Render(t, h.build(context.Background(), mustQuery(t, h, "is:pkg util", "")))
	t.Logf("nothing-matched=%v (/p/alice/util exists)", strings.Contains(out, "Nothing matched."))
	if strings.Contains(out, "Nothing matched.") {
		t.Errorf("is:pkg util answers %q although /p/alice/util exists", "Nothing matched.")
	}
}

// 7. Discovery keeps at most maxDiscoverResults rows per group and the group
// header count reads the kept rows, with nothing saying more matched.
func TestB8DiscoveryCountReadsTheCapNotTheMatches(t *testing.T) {
	dir := &mockDirectory{}
	for i := 0; i < 25; i++ {
		dir.realms = append(dir.realms, "gno.land/r/u"+string(rune('a'+i))+"/blog")
	}
	h := newHandlerWithDir(t, newDiscoveryClient(), dir, nil)
	data := h.build(context.Background(), mustQuery(t, h, "blog", ""))
	out := b8Render(t, data)
	i := strings.Index(out, "Realms")
	t.Logf("matching realms=25 group=%q shown=%d header=%q", data.Groups[0].Label, len(data.Groups[0].Results), out[i:min(len(out), i+12)])
	if len(data.Groups[0].Results) < 25 && !strings.Contains(out, "Showing part") && !strings.Contains(out, "more") {
		t.Errorf("25 realms match, header counts %d, nothing says more matched", len(data.Groups[0].Results))
	}
}
