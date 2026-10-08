// Repro, gnolang/gno#6231 at dd40ef3dc9b92a3df888bb9e70dc21871dcc192a:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6231/head && git checkout dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp <this file> gno.land/pkg/gnoweb/zz_alias_search_scope_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestAliasHeaderSearchScope -v
//
// EnrichHeaderData builds the no-JS form action from HeaderData.RealmURL.Path,
// which for a GnowebPath alias is the alias TARGET (handler_http.go rewrites
// r.URL.Path before parsing). The JS controller builds the same search from
// window.location.pathname (controller-searchbar.ts scopeBase), i.e. the alias
// itself. So on `/` (-> /r/gnoland/home) and `/about` (-> /r/gnoland/pages:p/about)
// the no-JS search is scoped to the target realm and the JS one is chain-wide.
// Expected: header action equals what the controller sends (`/$search`, `/about$search`).
package gnoweb_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
)

var headerAction = regexp.MustCompile(`id="header-searchbar"[^>]*?\saction="([^"]*)"`)

func TestAliasHeaderSearchScope(t *testing.T) {
	cfg := newTestHandlerConfig(t, gnoweb.NewMockClient(&gnoweb.MockPackage{
		Domain: "example.com",
		Path:   "/r/mock/path",
		Files: map[string]string{
			"render.gno": `package main; func Render(path string) string { return "body" }`,
		},
	}))
	cfg.Aliases = map[string]gnoweb.AliasTarget{
		"/":      {Value: "/r/mock/path", Kind: gnoweb.GnowebPath},
		"/about": {Value: "/r/mock/path:p/about", Kind: gnoweb.GnowebPath},
	}
	logger := slog.New(slog.NewTextHandler(&testingLogger{t}, &slog.HandlerOptions{}))
	h, err := gnoweb.NewHTTPHandler(logger, cfg)
	if err != nil {
		t.Fatal(err)
	}
	get := func(target string) string {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Body.String()
	}
	for _, tc := range []struct{ page, jsBase string }{
		{"/", "/"},
		{"/about", "/about"},
	} {
		m := headerAction.FindStringSubmatch(get(tc.page))
		if m == nil {
			t.Fatalf("%s: no header form action", tc.page)
		}
		noJS := m[1]
		js := tc.jsBase + "$search"
		noJSScoped := strings.Contains(get(noJS+"?q=blog"), "Scoped to")
		jsScoped := strings.Contains(get(js+"&q=blog"), "Scoped to")
		t.Logf("page %-7s no-JS action %-22q scoped=%v | JS href %-16q scoped=%v", tc.page, noJS, noJSScoped, js, jsScoped)
		if noJS != js || noJSScoped != jsScoped {
			t.Errorf("page %s: no-JS form searches %q (scoped=%v), JS searches %q (scoped=%v)", tc.page, noJS, noJSScoped, js, jsScoped)
		}
	}
}
