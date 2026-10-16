package main

// Repro, from a plain clone of gnolang/gno at fe1c9f046d52faaeafe13f9bd48ce44c77f39e20:
//
//	cp b2-rollout-gateway-switch_test.go gno.land/cmd/gnoweb/b2_rollout_gateway_switch_test.go
//	go test ./gno.land/cmd/gnoweb -run TestRolloutGatewaySwitch -count=1 -v
//
// Claim: a URL written against the default gateway is stranded by the
// gateway switch the ADR plans for gno.land ("gno.land itself needs a
// dedicated or self-hosted gateway"). Writers keep producing that shape:
// gnoweb itself labels a rewritten autolink with the gateway URL, and
// p/nt/markdown/sanitize/v0 (md.Image, md.Link), deployed and frozen,
// accepts only http(s), so a realm built on it can store an IPFS image only
// as a gateway URL. ipfs.filebase.io is not in ipfsRetiredGateways, so after
// -ipfs-gateway moves, that URL is neither rewritten nor in img-src.
//
// The test asserts the behaviour a stored URL needs to survive the switch
// (rewritten to the new gateway, or still allowed by the CSP); it fails at
// fe1c9f046.

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRolloutGatewaySwitch(t *testing.T) {
	const cid = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

	serve := func(gateway, page string) (body, csp string) {
		cfg := gnoweb.NewDefaultAppConfig()
		cfg.ChainID = "dev" // no node needed
		cfg.RenderConfig.IPFSGateway = gateway
		cfg.Aliases = maps.Clone(cfg.Aliases)
		cfg.Aliases["/p"] = gnoweb.AliasTarget{Value: page, Kind: gnoweb.StaticMarkdown}
		router, err := gnoweb.NewRouter(log.NewTestingLogger(t), cfg)
		require.NoError(t, err)
		h := SecureHeadersMiddleware(router, true, "http://127.0.0.1:26657", cfg.RenderConfig.IPFSGateway)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/p", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		return rec.Body.String(), rec.Header().Get("Content-Security-Policy")
	}
	imgSrc := func(csp string) string {
		for _, d := range strings.Split(csp, ";") {
			if d = strings.TrimSpace(d); strings.HasPrefix(d, "img-src") {
				return d
			}
		}
		return ""
	}

	// 1. Today, default gateway: a retired-gateway autolink is shown to the
	// visitor labelled with the filebase URL, the text a visitor copies.
	body, _ := serve(gnoweb.DefaultIPFSGateway, "<https://ipfs.io/ipfs/"+cid+">")
	shown := `rel="noopener">` + gnoweb.DefaultIPFSGateway + "/ipfs/" + cid + "<span"
	require.Contains(t, body, shown, "autolink label")
	t.Logf("step 1: autolink rendered with label %q", gnoweb.DefaultIPFSGateway+"/ipfs/"+cid)

	// 2. The copied URL is stored in realm state (the only IPFS image form
	// md.Image accepts). Under the default it renders and is allowed.
	stored := "![x](" + gnoweb.DefaultIPFSGateway + "/ipfs/" + cid + ")"
	body, csp := serve(gnoweb.DefaultIPFSGateway, stored)
	require.Contains(t, body, `src="`+gnoweb.DefaultIPFSGateway+"/ipfs/"+cid+`"`)
	require.Contains(t, imgSrc(csp), gnoweb.DefaultIPFSGateway)
	t.Logf("step 2: default gateway: stored image allowed (%s in img-src)", gnoweb.DefaultIPFSGateway)

	// Contrast: the forms the PR rewrites follow any gateway.
	body, _ = serve("https://gw.example", "![x](ipfs://"+cid+") ![y](https://ipfs.io/ipfs/"+cid+")")
	require.Equal(t, 2, strings.Count(body, `src="https://gw.example/ipfs/`+cid+`"`), "ipfs:// and ipfs.io follow the switch")

	// 3. The operator moves -ipfs-gateway to a dedicated gateway. The stored
	// URL must either follow it or stay allowed.
	body, csp = serve("https://gw.example", stored)
	rewritten := strings.Contains(body, `src="https://gw.example/ipfs/`+cid+`"`)
	allowed := strings.Contains(imgSrc(csp), gnoweb.DefaultIPFSGateway)
	t.Logf("step 3: gateway https://gw.example: stored src rewritten=%v, filebase in img-src=%v; %s", rewritten, allowed, imgSrc(csp))
	assert.True(t, rewritten || allowed, "an image stored as %s/ipfs/<cid> is blocked by the CSP after the gateway switch", gnoweb.DefaultIPFSGateway)
}
