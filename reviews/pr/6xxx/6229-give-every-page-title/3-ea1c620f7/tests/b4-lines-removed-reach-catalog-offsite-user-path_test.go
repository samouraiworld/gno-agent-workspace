// Repro: on a trusted page (LinkPolicy FollowInternalLinks, every package
// under -trusted-paths such as /r/gnoland/*), a link to another site whose
// path starts with /u/ renders with no rel, so search engines follow it from
// an official page, though the LinkPolicy godoc says FollowInternalLinks
// "follows links within the site and marks the others". detectLinkType
// (markdown/ext_links.go) returns GnoLinkTypeUser on target.IsUser() before it
// compares target.Domain with the page's, and replaceWithGnoLink then sets
// Followed from links.follows(GnoLinkTypeUser), which is true. The same link
// without the /u/ prefix is typed external and marked.
//
// From a plain clone of gnolang/gno:
//
//	git fetch origin pull/6229/head:pr6229 && git checkout ea1c620f7f5413f9a6db21883476c51b14ba8b9a
//	cp <this file> gno.land/pkg/gnoweb/zz_offsite_user_path_test.go
//	go test ./gno.land/pkg/gnoweb -run TestZZOffsiteUserPath -v -count=1
//	rm gno.land/pkg/gnoweb/zz_offsite_user_path_test.go
//
// The test asserts the policy the godoc states, so it fails at ea1c620f7 on
// the three /u/ rows of /r/gnoland/forum; /r/nym/app (community) and the
// control row pass.
package gnoweb_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func zzUserPathHandler(t *testing.T, render map[string]func(string) string) *gnoweb.HTTPHandler {
	t.Helper()
	files := map[string]string{"render.gno": `package main; func Render(path string) string { return "" }`}
	pkg := func(path string) *gnoweb.MockPackage {
		return &gnoweb.MockPackage{Domain: "gno.land", Path: path, Files: files}
	}
	client := documentClient{
		MockClient: gnoweb.NewMockClient(pkg("/r/gnoland/forum"), pkg("/r/nym/app")),
		render:     render,
	}
	config := newTestHandlerConfig(t, client)
	withGnoLandMeta(config)
	config.TrustedPaths = []string{"gnoland"}
	logger := slog.New(slog.NewTextHandler(&testingLogger{t}, &slog.HandlerOptions{}))
	config.Renderer = gnoweb.NewHTMLRenderer(logger, gnoweb.NewDefaultRenderConfig(), nil)
	h, err := gnoweb.NewHTTPHandler(logger, config)
	require.NoError(t, err)
	return h
}

func TestZZOffsiteUserPath(t *testing.T) {
	const post = "# Forum\n\nUser post: [a](https://casino.example/u/promo) [b](//casino.example/u/promo) [c](http://casino.example/u/promo:ref) [control](https://casino.example/promo)\n"
	h := zzUserPathHandler(t, map[string]func(string) string{
		"/r/gnoland/forum": func(string) string { return post },
		"/r/nym/app":       func(string) string { return post },
	})
	for _, page := range []string{"/r/gnoland/forum", "/r/nym/app"} {
		rr := httptest.NewRecorder()
		h.Get(rr, httptest.NewRequest(http.MethodGet, page, nil))
		require.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()
		for _, href := range []string{
			"https://casino.example/u/promo",
			"//casino.example/u/promo",
			"http://casino.example/u/promo:ref",
			"https://casino.example/promo",
		} {
			a := regexp.MustCompile(`<a href="` + regexp.QuoteMeta(href) + `"[^>]*>`).FindString(body)
			t.Logf("%-17s %-34s -> %s", page, href, a)
			require.NotEmpty(t, a, "anchor for %s", href)
			// A community page marks every link nofollow; a trusted page must
			// still mark this one, which leaves the site.
			assert.Contains(t, a, "nofollow", "%s on %s: an off-site link must be marked", href, page)
		}
	}
}
