// judge-190-help-view-real-origin_test.go — gno.land/pkg/gnoweb/components/view_action.go:73
//
// Asserts the premise both b2 candidates hand-wrote into a fixture: the REAL
// $help template, rendered with the origin gnopreview's crawler dials, puts
// that absolute origin in exactly two attributes, neither of them href or src.
// Measured at ecf7af0f2: 2 attributes, action= and data-copy-text-value=.
// Passes at the reviewed head — it is the input to crawl.go's rewrite(), which
// matches only `\b(href|src)="..."` (misc/gnopreview/crawl.go:40), so both
// survive verbatim into the published preview.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> gno.land/pkg/gnoweb/components/judge_help_origin_test.go
//	go test ./gno.land/pkg/gnoweb/components/ -run TestHelpViewEmitsCrawlOriginOutsideHrefSrc -v

package components

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/doc"
)

// crawlOrigin is what requestOrigin(r) (handler_http.go:933, reading r.Host)
// returns for every page gnopreview fetches: Crawler.Base is the loopback
// gnodev the workflow booted, http://127.0.0.1:<port>.
const crawlOrigin = "http://127.0.0.1:8899"

func TestHelpViewEmitsCrawlOriginOutsideHrefSrc(t *testing.T) {
	v := HelpView(HelpData{
		RealmName: "boards2",
		PkgPath:   "gno.land/r/gnoland/boards2/v0",
		Domain:    "gno.land",
		Origin:    crawlOrigin, // handler_http.go:390 sets this from the request
		ChainId:   "dev",
		Remote:    "127.0.0.1:26657",
		Functions: []HelpFunction{{JSONFunc: &doc.JSONFunc{
			Name:   "CreateBoard",
			Params: []*doc.JSONField{{Name: "name", Type: "string"}},
		}}},
	})

	var buf bytes.Buffer
	if err := v.Render(&buf); err != nil {
		t.Fatal(err)
	}

	// Every attribute whose value carries the crawl-time origin.
	attrRe := regexp.MustCompile(`([\w-]+)="[^"]*` + regexp.QuoteMeta("127.0.0.1:8899") + `[^"]*"`)
	hits := attrRe.FindAllStringSubmatch(buf.String(), -1)

	var names []string
	for _, h := range hits {
		t.Logf("attribute carrying the crawl origin: %s", h[0])
		names = append(names, h[1])
	}
	if len(names) == 0 {
		t.Fatal("no attribute carried the origin — the finding's premise would be refuted")
	}

	// IS: gnoweb puts the absolute origin only in attributes crawl.go ignores.
	want := "action,data-copy-text-value"
	// SHOULD (once crawl.go:40 covers them, or gnoweb emits relative help URLs):
	// want := ""
	got := strings.Join(sortedUnique(names), ",")
	if got != want {
		t.Errorf("origin-carrying attributes = %q, want %q", got, want)
	}
	for _, n := range names {
		if n == "href" || n == "src" {
			t.Errorf("%s= carries the origin, so crawl.go's attrRe would reach it", n)
		}
	}
}

func sortedUnique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for i := range out { // tiny insertion sort, keeps the file import-free
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
