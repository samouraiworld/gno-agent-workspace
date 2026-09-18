// Crawler.rewrite only touches href= and src=, so the absolute URLs gnoweb
// builds from the request origin survive into the published preview pointing at
// the crawler's own loopback gnodev.
//
// gnoweb side, at ecf7af0f29abe4737a52803d672bc5a33c17cc60:
//
//	gno.land/pkg/gnoweb/handler_http.go:390   gnourl.Origin = requestOrigin(r)
//	gno.land/pkg/gnoweb/handler_http.go:933   host := r.Host
//	gno.land/pkg/gnoweb/components/view_action.go:73
//	        url.WriteString(data.Origin + pkgPath + "$help&func=" + fn.Name)
//	gno.land/pkg/gnoweb/components/views/action.html:84
//	        data-copy-text-value="{{ buildHelpURL $data . }}"
//	gno.land/pkg/gnoweb/components/views/action.html:104
//	        <form class="params" ... action="{{ buildHelpURL $data . }}"
//
// The crawler fetches every $help page over http://127.0.0.1:8899
// (misc/gnopreview/main.go:124, misc/gnopreview/crawl.go:92), so r.Host is
// 127.0.0.1:8899 and every buildHelpURL on the captured page carries it.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/b2_reach_r2_help_origin_test.go
//	cd misc/gnopreview && go test ./... -run TestHelpPageKeepsCrawlerOrigin -v
//
// Expected at this head: FAIL — both attributes still name 127.0.0.1:8899.

package main

import (
	"strings"
	"testing"
)

// crawlOrigin is what requestOrigin(r) returns for every page this crawler
// fetches: Crawler.Base is fmt.Sprintf("http://127.0.0.1:%d", cfg.port).
const crawlOrigin = "http://127.0.0.1:8899"

// helpBody is the markup gnoweb emits for one exported function on a $help
// page, with buildHelpURL already expanded against the crawl origin.
const helpBody = `<html><head></head><body>
<a href="/r/gnoland/boards2/v0$source">source</a>
<button data-copy-text-value="` + crawlOrigin + `/r/gnoland/boards2/v0$help&amp;func=CreateBoard">copy</button>
<form class="params" id="form-CreateBoard" method="GET" action="` + crawlOrigin + `/r/gnoland/boards2/v0$help&amp;func=CreateBoard"></form>
</body></html>`

func TestHelpPageKeepsCrawlerOrigin(t *testing.T) {
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	p := &page{
		URL:  "/r/gnoland/boards2/v0$help",
		File: urlToFile("/r/gnoland/boards2/v0$help"),
		Body: helpBody,
	}
	c.pages[p.URL] = p

	got := c.rewrite(p)
	t.Logf("rewritten page:\n%s", got)

	if n := strings.Count(got, crawlOrigin); n != 0 {
		t.Errorf("rewrite() left %d reference(s) to the crawler's own gnodev (%s) in the published page; "+
			"the reader's browser dials loopback for the form action and the copied anchor link", n, crawlOrigin)
	}
}

// TestHrefIsRewrittenSoTheHarnessIsHonest pins that the same body's href= is
// rewritten, so a failure above is the attribute set and not a broken fixture.
func TestHrefIsRewrittenSoTheHarnessIsHonest(t *testing.T) {
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	p := &page{URL: "/r/gnoland/boards2/v0$help", File: urlToFile("/r/gnoland/boards2/v0$help"), Body: helpBody}
	c.pages[p.URL] = p
	if got := c.rewrite(p); strings.Contains(got, `href="/r/gnoland/boards2/v0$source"`) {
		t.Fatalf("href= was not rewritten either — fixture is wrong, not the finding:\n%s", got)
	}
}
