// judge 1 and 2: the switch link as rendered, label, href and the item-path
// line under it, on a twin whose directory is a realm (2) and on a twinless
// page under a realm with direct children (1), then what the target lists.
package gnoweb_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
)

var reJudgeItem = regexp.MustCompile(`(?s)<a href="([^"]*)" class="item item--primary">.*?<span class="item-label">([^<]*)</span>\s*<span class="item-path">([^<]*)</span>`)

func TestJudgeMenu(t *testing.T) {
	realm := func(p string) *gnoweb.MockPackage {
		return &gnoweb.MockPackage{Path: p, Files: map[string]string{"r.gno": "package r"}, Functions: renderFuncs}
	}
	pure := func(p string) *gnoweb.MockPackage {
		return &gnoweb.MockPackage{Path: p, Files: map[string]string{"p.gno": "package p"}}
	}
	h := newCounterpartHandler(t, gnoweb.NewMockClient(realm("/r/tests/vm"), realm("/r/tests/vm/crossrealm"), realm("/r/tests/vm/subtests"),
		pure("/p/tests/vm/crossrealm"), pure("/p/tests/vm/foo")))
	for _, page := range []string{"/p/tests/vm/crossrealm", "/p/tests/vm/foo"} {
		body := serve(h, httptest.NewRequest(http.MethodGet, page, nil)).Body.String()
		m := reJudgeItem.FindStringSubmatch(body)
		if m == nil {
			fmt.Printf("%s: no switch link\n", page)
			continue
		}
		fmt.Printf("%s: href=%q label=%q item-path=%q\n", page, m[1], m[2], m[3])
	}
	src := serve(h, httptest.NewRequest(http.MethodGet, "/r/tests/vm$source", nil)).Body.String()
	for _, p := range []string{"/r/tests/vm/crossrealm", "/r/tests/vm/subtests"} {
		fmt.Printf("/r/tests/vm$source lists %s: %v\n", p, strings.Contains(src, `href="`+p+`"`))
	}
}
