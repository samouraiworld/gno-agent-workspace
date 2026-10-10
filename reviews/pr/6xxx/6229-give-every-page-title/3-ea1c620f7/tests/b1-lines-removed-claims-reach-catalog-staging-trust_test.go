// gnodev staging serves gnoweb with DefaultTrustedPaths and
// -index-community=registered, on a chain whose r/sys/names enforcement is
// off (verifier.gno: `enabled = false`, and nothing in contribs/gnodev calls
// names.Enable). Any account may then deploy under `gnoland`, and the page it
// deploys is classified official: its own h1 and paragraph become the <title>
// and description, with index, follow.
//
// Repro, from a plain clone of gnolang/gno at ea1c620f7:
//
//	cp b1-lines-removed-claims-reach-catalog-staging-trust_test.go contribs/gnodev/zz_staging_trust_test.go
//	cd contribs/gnodev && go test . -run TestStagingTrustsNamespacesItsChainDoesNotEnforce -v -count=1
//	rm zz_staging_trust_test.go
package main

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stagingClient struct {
	*gnoweb.MockClient
	render map[string]string
}

func (c stagingClient) Realm(ctx context.Context, path, args string) ([]byte, error) {
	if s, ok := c.render[path]; ok {
		return []byte(s), nil
	}
	return c.MockClient.Realm(ctx, path, args)
}

func TestStagingTrustsNamespacesItsChainDoesNotEnforce(t *testing.T) {
	staging := defaultStagingOptions
	appcfg := gnoWebAppConfig(&staging, "")
	t.Logf("staging TrustedPaths=%v IndexCommunity=%v", appcfg.TrustedPaths, appcfg.IndexCommunity)

	files := map[string]string{"render.gno": `package airdrop; func Render(string) string { return "" }`}
	lure := "# Official GNOT airdrop\n\nClaim your free GNOT airdrop at evil.example before it closes tonight.\n"
	client := stagingClient{
		MockClient: gnoweb.NewMockClient(
			&gnoweb.MockPackage{Domain: "gno.land", Path: "/r/gnoland/airdrop", Files: files},
			&gnoweb.MockPackage{Domain: "gno.land", Path: "/r/brandname/airdrop", Files: files},
		),
		render: map[string]string{"/r/gnoland/airdrop": lure, "/r/brandname/airdrop": lure},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h, err := gnoweb.NewHTTPHandler(logger, &gnoweb.HTTPHandlerConfig{
		ClientAdapter:  client,
		Renderer:       gnoweb.NewHTMLRenderer(logger, gnoweb.NewDefaultRenderConfig(), nil),
		Aliases:        map[string]gnoweb.AliasTarget{},
		Meta:           gnoweb.StaticMetadata{Domain: "gno.land"},
		TrustedPaths:   appcfg.TrustedPaths,
		IndexCommunity: appcfg.IndexCommunity,
	})
	require.NoError(t, err)

	title := regexp.MustCompile(`<title>([^<]*)</title>`)
	desc := regexp.MustCompile(`<meta name="description" content="([^"]*)"`)
	robots := regexp.MustCompile(`<meta name="robots" content="([^"]*)"`)
	get := func(path string) (string, string, string, string) {
		rec := httptest.NewRecorder()
		h.Get(rec, httptest.NewRequest("GET", path, nil))
		body := rec.Body.String()
		first := func(re *regexp.Regexp) string {
			if m := re.FindStringSubmatch(body); m != nil {
				return m[1]
			}
			return ""
		}
		t.Logf("%s status=%d title=%q description=%q robots=%q X-Robots-Tag=%q",
			path, rec.Code, first(title), first(desc), first(robots), rec.Header().Get("X-Robots-Tag"))
		return first(title), first(desc), first(robots), rec.Header().Get("X-Robots-Tag")
	}

	// Under a default trusted namespace: the deployer's own h1 titles the page.
	ti, de, ro, xr := get("/r/gnoland/airdrop")
	assert.Contains(t, ti, "Official GNOT airdrop")
	assert.Contains(t, de, "evil.example")
	assert.Equal(t, "index, follow", ro)
	assert.Empty(t, xr)

	// Under any other name: indexed too, since staging keeps "registered" and
	// no name on its chain is registered by anyone in particular.
	_, _, ro, xr = get("/r/brandname/airdrop")
	assert.Equal(t, "index, follow", ro)
	assert.Empty(t, xr)
}
