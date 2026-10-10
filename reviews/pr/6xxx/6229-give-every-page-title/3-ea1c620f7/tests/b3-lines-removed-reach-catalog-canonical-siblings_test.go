// Repro for gnolang/gno#6229 at ea1c620f7f5413f9a6db21883476c51b14ba8b9a, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6229/head && git checkout ea1c620f7f5413f9a6db21883476c51b14ba8b9a
//	cp <this file> gno.land/pkg/gnoweb/zz_b3r3_test.go
//	go test ./gno.land/pkg/gnoweb -run 'TestB3R3_' -v -count=1
//
// Each test asserts what the canonical should be and fails at the head.
//
// TestB3R3_RealmAliasDottedKey: round 2's dotted-alias-key fix routes only
// static aliases through the raw request path; a GnowebPath alias keyed
// /license.md or /Terms still parses as File under Path "/", so its canonical
// and og:url read https://gno.land//license.md, a URL that does not serve.
//
// TestB3R3_ViewAliasTakesRealmCanonical: aliasTargets keys an alias by its
// target's path and args only, so an alias to a $source or $help view becomes
// the canonical of the bare realm page.
//
// TestB3R3_FallbackViewsKeepArgs: a realm with no Render (directory view) and a
// namespace with no realm (paths listing) ignore the args, yet each args
// variant is index, follow and names itself canonical, args included: the
// round-2 args fix clears Args only for views it can see in the URL.
package gnoweb_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
)

var (
	b3r3Title     = regexp.MustCompile(`<title>([^<]*)</title>`)
	b3r3Canonical = regexp.MustCompile(`<link rel="canonical" href="([^"]*)"`)
	b3r3OgURL     = regexp.MustCompile(`<meta property="og:url" content="([^"]*)"`)
	b3r3Robots    = regexp.MustCompile(`<meta name="robots" content="([^"]*)"`)
)

func b3r3First(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// b3r3Client answers ListPaths with domain-qualified paths, as the chain does,
// which MockClient (keyed by bare /r/... paths) does not.
type b3r3Client struct {
	*gnoweb.MockClient
}

func (c b3r3Client) ListPaths(ctx context.Context, prefix string, limit int) ([]string, error) {
	if strings.HasPrefix(prefix, "@") {
		return c.MockClient.ListPaths(ctx, prefix, limit)
	}
	paths, err := c.MockClient.ListPaths(ctx, strings.TrimPrefix(prefix, "gno.land"), limit)
	for i := range paths {
		paths[i] = "gno.land" + paths[i]
	}
	return paths, err
}

func b3r3Handler(t *testing.T, index gnoweb.CommunityIndex, aliases map[string]gnoweb.AliasTarget) *gnoweb.HTTPHandler {
	t.Helper()
	render := map[string]string{"render.gno": `package main; func Render(path string) string { return "body" }`}
	noRender := map[string]string{"x.gno": `package x`}
	client := b3r3Client{gnoweb.NewMockClient(
		&gnoweb.MockPackage{Domain: "gno.land", Path: "/r/gnoland/pages", Files: render},
		&gnoweb.MockPackage{Domain: "gno.land", Path: "/r/gnoland/home", Files: render},
		&gnoweb.MockPackage{Domain: "gno.land", Path: "/r/gnoland/norender", Files: noRender},
		&gnoweb.MockPackage{Domain: "gno.land", Path: "/r/nym/norender", Files: noRender},
	)}
	cfg := &gnoweb.HTTPHandlerConfig{
		ClientAdapter: client,
		Renderer: gnoweb.NewHTMLRenderer(slog.New(slog.NewTextHandler(io.Discard, nil)),
			gnoweb.NewDefaultRenderConfig(), nil),
		Aliases: map[string]gnoweb.AliasTarget{},
	}
	cfg.Meta.Domain = "gno.land"
	cfg.Meta.CanonicalOrigin = "https://gno.land"
	cfg.Meta.AssetsPath = "/public/"
	cfg.TrustedPaths = []string{"gnoland"}
	cfg.IndexCommunity = index
	if aliases != nil {
		cfg.Aliases = aliases
	}
	h, err := gnoweb.NewHTTPHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

type b3r3Head struct {
	status                                int
	robots, xrobots, canonical, ogURL, title string
}

func b3r3Get(t *testing.T, h *gnoweb.HTTPHandler, url string) b3r3Head {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
	b := rr.Body.String()
	hd := b3r3Head{rr.Code, b3r3First(b3r3Robots, b), rr.Header().Get("X-Robots-Tag"),
		b3r3First(b3r3Canonical, b), b3r3First(b3r3OgURL, b), b3r3First(b3r3Title, b)}
	t.Logf("%-40s status=%d robots=%q canonical=%q og:url=%q title=%q", url, hd.status, hd.robots, hd.canonical, hd.ogURL, hd.title)
	return hd
}

func TestB3R3_RealmAliasDottedKey(t *testing.T) {
	h := b3r3Handler(t, gnoweb.IndexRegisteredCommunity, map[string]gnoweb.AliasTarget{
		"/about":      {Value: "/r/gnoland/pages:p/about", Kind: gnoweb.GnowebPath},
		"/license.md": {Value: "/r/gnoland/pages:p/license", Kind: gnoweb.GnowebPath},
		"/Terms":      {Value: "/r/gnoland/pages:p/terms", Kind: gnoweb.GnowebPath},
	})
	for _, u := range []string{"/about", "/license.md", "/Terms"} {
		hd := b3r3Get(t, h, u)
		want := "https://gno.land" + u
		if hd.status != http.StatusOK || hd.canonical != want || hd.ogURL != want {
			t.Errorf("%s: status %d, canonical %q, og:url %q, want %q for both", u, hd.status, hd.canonical, hd.ogURL, want)
		}
	}
	// The target names the alias as canonical, and the alias names a URL that does not serve.
	b3r3Get(t, h, "/r/gnoland/pages:p/license")
	if hd := b3r3Get(t, h, "//license.md"); hd.status == http.StatusOK {
		t.Logf("//license.md serves 200")
	}
}

func TestB3R3_ViewAliasTakesRealmCanonical(t *testing.T) {
	h := b3r3Handler(t, gnoweb.IndexRegisteredCommunity, map[string]gnoweb.AliasTarget{
		"/src":     {Value: "/r/gnoland/pages$source", Kind: gnoweb.GnowebPath},
		"/actions": {Value: "/r/gnoland/home$help", Kind: gnoweb.GnowebPath},
	})
	for _, realm := range []string{"/r/gnoland/pages", "/r/gnoland/home"} {
		hd := b3r3Get(t, h, realm)
		if want := "https://gno.land" + realm; hd.canonical != want {
			t.Errorf("%s renders the realm, yet names %q canonical, want %q", realm, hd.canonical, want)
		}
	}
	b3r3Get(t, h, "/src")
	b3r3Get(t, h, "/actions")
}

func TestB3R3_FallbackViewsKeepArgs(t *testing.T) {
	for _, index := range []gnoweb.CommunityIndex{gnoweb.IndexRegisteredCommunity, gnoweb.IndexAllCommunity} {
		h := b3r3Handler(t, index, nil)
		for _, pair := range [][2]string{
			{"/r/gnoland/norender", "/r/gnoland/norender:anything-at-all"},
			{"/r/gnoland", "/r/gnoland:anything-at-all"},
			{"/r/nym/norender", "/r/nym/norender:anything-at-all"},
		} {
			bare, variant := b3r3Get(t, h, pair[0]), b3r3Get(t, h, pair[1])
			if bare.status == http.StatusOK && variant.status == http.StatusOK &&
				variant.robots == "index, follow" && variant.canonical != bare.canonical {
				t.Errorf("%s: %s renders the view of %s, is index, follow, and names itself canonical (%s) instead of %s",
					index, pair[1], pair[0], variant.canonical, bare.canonical)
			}
		}
	}
}
