// solo-finder-2: the breadcrumb_click analytics delegate no longer sees the
// first-segment navigation on /r/<ns>/... and /p/<ns>/... pages.
//
// analytics.ts fires breadcrumb_click on clicks matching
//   ol[data-searchbar-target="breadcrumb"] a
// At the merge base the first chip is <a href="/r/"> inside that <ol>. At head
// it is a <button> inside the <ol>, and every link it opens (counterpart,
// All in <ns>, All realms) sits in div#kind-switch-menu, a sibling of the <ol>.
//
// Repro from a plain clone of gnolang/gno:
//   git fetch origin pull/6262/head && git checkout --detach c5ae4b13a8883303a382d6b1ae1c61b929fd048c
//   cp <this file> gno.land/pkg/gnoweb/zz_finder2_analytics_test.go
//   FINDER2_HTML=/tmp/page.html go test ./gno.land/pkg/gnoweb -run TestFinder2_BreadcrumbAnalytics -v -count=1
//   # then the browser probe: solo-finder-2-breadcrumb-analytics.sh /tmp/page.html
//
// Expected at head: every menu anchor reports inOl=false; no anchor inside the
// <ol> points at /r/ (the old first-chip target).

package gnoweb_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"golang.org/x/net/html"
)

func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

func inBreadcrumbOl(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "ol" && attr(p, "data-searchbar-target") == "breadcrumb" {
			return true
		}
	}
	return false
}

func TestFinder2_BreadcrumbAnalytics(t *testing.T) {
	rr := serveCounterpart(t, gnoweb.NewMockClient(golfPackages()...), "/r/alice/golf/game")
	body := rr.Body.String()
	if p := os.Getenv("FINDER2_HTML"); p != "" {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	var menuAnchors, menuInOl, olToListing int
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			cls := attr(n, "class")
			in := inBreadcrumbOl(n)
			if strings.Contains(cls, "item--") {
				menuAnchors++
				if in {
					menuInOl++
				}
				t.Logf("menu anchor href=%q inOl=%v", attr(n, "href"), in)
			}
			if in && attr(n, "href") == "/r/" {
				olToListing++
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	t.Logf("menu anchors=%d, menu anchors matched by breadcrumb_click selector=%d, <ol> anchors to /r/=%d",
		menuAnchors, menuInOl, olToListing)
	if menuAnchors == 0 {
		t.Fatal("no kind-switch menu rendered")
	}
	if menuInOl == 0 && olToListing == 0 {
		t.Errorf("breadcrumb_click selector matches none of the %d first-segment links", menuAnchors)
	}
}
