// cmd/gnoweb with no -trusted-paths and no -index-community, as
// misc/loop/docker-compose.yml runs it for staging.gno.land, on a chain that
// never enables r/sys/names (nothing under misc/loop calls names.Enable, and
// verifier.gno starts `enabled = false`). Anyone funded by the staging faucet
// may deploy under `gnoland`; gnoweb then classifies the page official and
// lends its h1 and paragraph to <title> and description, index, follow.
//
// Repro, from a plain clone of gnolang/gno at ea1c620f7:
//
//	cp b1-lines-removed-claims-reach-catalog-loop-defaults_test.go gno.land/cmd/gnoweb/zz_loop_defaults_test.go
//	go test ./gno.land/cmd/gnoweb -run TestLoopDefaultsLendUnenforcedNamespace -v -count=1
//	rm gno.land/cmd/gnoweb/zz_loop_defaults_test.go
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

type loopClient struct {
	*gnoweb.MockClient
	render map[string]string
}

func (c loopClient) Realm(ctx context.Context, path, args string) ([]byte, error) {
	if s, ok := c.render[path]; ok {
		return []byte(s), nil
	}
	return c.MockClient.Realm(ctx, path, args)
}

func TestLoopDefaultsLendUnenforcedNamespace(t *testing.T) {
	t.Setenv("GNOWEB_REALM_NOTICE_TEXT", "")
	cfg := defaultWebOptions // the flags misc/loop leaves unset keep these
	appcfg := gnoweb.NewDefaultAppConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	require.NoError(t, setupTrust(&cfg, appcfg, logger))
	appcfg.IndexCommunity = cfg.indexCommunity
	t.Logf("TrustedPaths=%v IndexCommunity=%v", appcfg.TrustedPaths, appcfg.IndexCommunity)

	files := map[string]string{"render.gno": `package airdrop; func Render(string) string { return "" }`}
	lure := "# Official GNOT airdrop\n\nClaim your free GNOT airdrop at evil.example before it closes tonight.\n"
	client := loopClient{
		MockClient: gnoweb.NewMockClient(&gnoweb.MockPackage{Domain: "gno.land", Path: "/r/gnoland/airdrop", Files: files}),
		render:     map[string]string{"/r/gnoland/airdrop": lure},
	}
	h, err := gnoweb.NewHTTPHandler(logger, &gnoweb.HTTPHandlerConfig{
		ClientAdapter:  client,
		Renderer:       gnoweb.NewHTMLRenderer(logger, gnoweb.NewDefaultRenderConfig(), nil),
		Aliases:        map[string]gnoweb.AliasTarget{},
		Meta:           gnoweb.StaticMetadata{Domain: "gno.land", RealmNotice: appcfg.RealmNotice},
		TrustedPaths:   appcfg.TrustedPaths,
		IndexCommunity: appcfg.IndexCommunity,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest("GET", "/r/gnoland/airdrop", nil))
	body := rec.Body.String()
	first := func(expr string) string {
		if m := regexp.MustCompile(expr).FindStringSubmatch(body); m != nil {
			return m[1]
		}
		return ""
	}
	title := first(`<title>([^<]*)</title>`)
	desc := first(`<meta name="description" content="([^"]*)"`)
	robots := first(`<meta name="robots" content="([^"]*)"`)
	t.Logf("status=%d title=%q description=%q robots=%q X-Robots-Tag=%q notice=%v",
		rec.Code, title, desc, robots, rec.Header().Get("X-Robots-Tag"), regexp.MustCompile(`Community realm`).MatchString(body))
	assert.Contains(t, title, "Official GNOT airdrop")
	assert.Contains(t, desc, "evil.example")
	assert.Equal(t, "index, follow", robots)
}
