// Repro for gnolang/gno#6229 at ea1c620f7f5413f9a6db21883476c51b14ba8b9a, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6229/head && git checkout ea1c620f7f5413f9a6db21883476c51b14ba8b9a
//	cp <this file> gno.land/pkg/gnoweb/zz_b5_test.go
//	go test ./gno.land/pkg/gnoweb -run 'TestB5_' -v -count=1
//
// Each test asserts what the page should say and fails at the head where the
// policy in page_kind.go, community_index.go or canonical_origin.go is wrong.
//
// TestB5_RealmAliasNamedLikeAFile: a GnowebPath alias keyed /license.md or
// /Terms still reads canonical https://gno.land//license.md (round 2's fix
// covered static aliases only).
//
// TestB5_ViewAliasTakesRealmCanonical: aliasTargets keys an alias by its
// target's path and args, dropping the target's $ view, so an alias to a
// $source view becomes the canonical of the bare realm page.
//
// TestB5_RegisteredListings: under -index-community=registered the robots
// godoc says a community page is not indexed as "its listing"; a namespace
// listing (/r/nym, /p/nym) is. File and directory views (round 2's Warning)
// are checked too and expected noindex.
//
// TestB5_CanonicalOriginEmptyHost: normalizeCanonicalOrigin accepts an origin
// with a port and no host, and keeps a zero-padded default port.
//
// TestB5_FileViewTwoCanonicals: a file is served at /r/x/file.gno and at
// /r/x$source&file=file.gno; both are index, follow and each names itself.
package gnoweb

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/doc"
)

var (
	b5Canonical = regexp.MustCompile(`<link rel="canonical" href="([^"]*)"`)
	b5OgURL     = regexp.MustCompile(`<meta property="og:url" content="([^"]*)"`)
	b5Robots    = regexp.MustCompile(`<meta name="robots" content="([^"]*)"`)
	b5Title     = regexp.MustCompile(`<title>([^<]*)</title>`)
)

func b5First(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// b5Client answers ListPaths with domain-qualified paths, as the chain does.
type b5Client struct{ *MockClient }

func (c b5Client) ListPaths(ctx context.Context, prefix string, limit int) ([]string, error) {
	if strings.HasPrefix(prefix, "@") {
		return c.MockClient.ListPaths(ctx, prefix, limit)
	}
	paths, err := c.MockClient.ListPaths(ctx, strings.TrimPrefix(prefix, "gno.land"), limit)
	for i := range paths {
		paths[i] = "gno.land" + paths[i]
	}
	return paths, err
}

func b5Handler(t *testing.T, aliases map[string]AliasTarget, index CommunityIndex) *HTTPHandler {
	t.Helper()
	renderFn := []*doc.JSONFunc{{Name: "Render", Params: []*doc.JSONField{{Name: "path", Type: "string"}}, Results: []*doc.JSONField{{Type: "string"}}}}
	src := map[string]string{"render.gno": `package app; func Render(path string) string { return "# Hello\n\nbody" }`}
	client := NewMockClient(
		&MockPackage{Domain: "gno.land", Path: "/r/gnoland/pages", Files: src, Functions: renderFn},
		&MockPackage{Domain: "gno.land", Path: "/r/nym/app", Files: src, Functions: renderFn},
		&MockPackage{Domain: "gno.land", Path: "/p/nym/lib", Files: map[string]string{"lib.gno": "package lib"}},
	)
	if aliases == nil {
		aliases = map[string]AliasTarget{}
	}
	cfg := &HTTPHandlerConfig{
		ClientAdapter:  b5Client{client},
		Renderer:       NewHTMLRenderer(slog.New(slog.NewTextHandler(io.Discard, nil)), NewDefaultRenderConfig(), nil),
		Aliases:        aliases,
		TrustedPaths:   []string{"gnoland"},
		IndexCommunity: index,
	}
	cfg.Meta.Domain = "gno.land"
	cfg.Meta.CanonicalOrigin = "https://gno.land"
	cfg.Meta.AssetsPath = "/public/"
	h, err := NewHTTPHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

type b5Head struct {
	status                           int
	header, robots, canonical, ogurl string
}

func b5Get(t *testing.T, h *HTTPHandler, url string) b5Head {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
	b := rr.Body.String()
	hd := b5Head{rr.Code, rr.Header().Get("X-Robots-Tag"), b5First(b5Robots, b), b5First(b5Canonical, b), b5First(b5OgURL, b)}
	t.Logf("%-44s status=%d X-Robots-Tag=%q robots=%q canonical=%q og:url=%q title=%q",
		url, hd.status, hd.header, hd.robots, hd.canonical, hd.ogurl, b5First(b5Title, b))
	return hd
}

func b5Main(t *testing.T, h *HTTPHandler, url string) string {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
	_, body, _ := strings.Cut(rr.Body.String(), "</head>")
	_, main, _ := strings.Cut(body, "<main")
	return main
}

func TestB5_RealmAliasNamedLikeAFile(t *testing.T) {
	target := AliasTarget{Value: "/r/gnoland/pages:p/a", Kind: GnowebPath}
	h := b5Handler(t, map[string]AliasTarget{"/license.md": target, "/Terms": target, "/about": target}, IndexNoCommunity)
	for _, u := range []string{"/about", "/license.md", "/Terms"} {
		hd := b5Get(t, h, u)
		if want := "https://gno.land" + u; hd.status == http.StatusOK && hd.ogurl != want {
			t.Errorf("%s: og:url %q, want %q", u, hd.ogurl, want)
		}
	}
	b5Get(t, h, "//license.md")
}

func TestB5_ViewAliasTakesRealmCanonical(t *testing.T) {
	h := b5Handler(t, map[string]AliasTarget{"/s": {Value: "/r/gnoland/pages$source", Kind: GnowebPath}}, IndexNoCommunity)
	alias := b5Get(t, h, "/s")
	bare := b5Get(t, h, "/r/gnoland/pages")
	if b5Main(t, h, "/s") != b5Main(t, h, "/r/gnoland/pages$source") {
		t.Fatalf("/s does not serve the $source view")
	}
	t.Logf("/s serves the same body as /r/gnoland/pages$source")
	if bare.canonical == "https://gno.land/s" {
		t.Errorf("/r/gnoland/pages names %q canonical, the alias serving its $source view (alias status %d, robots %q)",
			bare.canonical, alias.status, alias.robots)
	}
}

func TestB5_RegisteredListings(t *testing.T) {
	h := b5Handler(t, nil, IndexRegisteredCommunity)
	for _, u := range []string{"/r/nym/app/render.gno", "/r/nym/app/", "/r/nym/", "/r/nym", "/p/nym", "/p/nym/lib"} {
		hd := b5Get(t, h, u)
		if hd.status == http.StatusOK && hd.robots == "index, follow" && u != "/p/nym/lib" {
			t.Errorf("%s: a community file view or listing is index, follow under registered", u)
		}
	}
	b5Get(t, h, "/r/nym/app")
}

func TestB5_CanonicalOriginEmptyHost(t *testing.T) {
	for _, in := range []string{"https://:8443", "https://:80", "http://:8080", "https://gno.land:0443"} {
		got, err := normalizeCanonicalOrigin(in)
		t.Logf("%-16q -> %q err=%v", in, got, err)
		if err == nil && got != "https://gno.land" {
			t.Errorf("%q accepted as %q: no host, or a default port left in", in, got)
		}
	}
}

func TestB5_FileViewTwoCanonicals(t *testing.T) {
	h := b5Handler(t, nil, IndexNoCommunity)
	a := b5Get(t, h, "/r/gnoland/pages/render.gno")
	b := b5Get(t, h, "/r/gnoland/pages$source&file=render.gno")
	if a.status == http.StatusOK && b.status == http.StatusOK && a.robots == "index, follow" && b.robots == "index, follow" && a.canonical != b.canonical {
		t.Errorf("one file, two indexed URLs naming themselves: %q and %q", a.canonical, b.canonical)
	}
}
