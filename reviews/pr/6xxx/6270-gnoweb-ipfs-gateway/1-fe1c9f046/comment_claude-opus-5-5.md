# Review: [#6270](https://github.com/gnolang/gno/pull/6270)
Posted: https://github.com/gnolang/gno/pull/6270#pullrequestreview-5547698095

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Two Warnings the branch introduces block it: a subdomain gateway such as stock Kubo on localhost gets every IPFS image blocked by the strict CSP, and a later switch of `-ipfs-gateway` breaks every image stored on the default gateway's URL.
Model: claude-opus-5-5, standard review, pipeline: 10 finders at xhigh, reflector and judges at medium, writer and text pass at low
Commit: fe1c9f046d52faaeafe13f9bd48ce44c77f39e20
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6270 fe1c9f046`
Round: 1. 10 finders, one reflector, 11 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted, none of them above Nit.

## Body

- Related suggestion: [`web25.gno:29`](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/examples/quarantined/gno.land/p/moul/web25/web25.gno#L29) returns an `https://ipfs.io/ipfs/<cid>` link, which renders right only where gnoweb rewrites [`ipfs.io`](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L19); returning `"ipfs://" + c.CID` would follow whatever gateway the operator sets.

## gno.land/cmd/gnoweb/main.go:470 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/cmd/gnoweb/main.go#L470) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/cmd/gnoweb/main.go#L470) · Warning [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4294596535)

A subdomain gateway redirects `/ipfs/<cid>` to `<cid>.ipfs.<host>`, which this bare-origin `img-src` entry blocks, and [Kubo redirects so by default on `localhost`](https://github.com/ipfs/kubo/blob/master/docs/config.md#implicit-defaults-of-gatewaypublicgateways).

<details><summary>repro</summary>

The strict case fails because the redirected image request never reaches the gateway, while the control without the CSP fetches it. Needs `chromium` on PATH.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6270 -R gnolang/gno
cat > gno.land/cmd/gnoweb/zz_subdomain_csp_test.go <<'GO'
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/require"
)

const b2CID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

// 1x1 transparent PNG.
var b2PNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=")

func TestB2SubdomainGatewayRedirectCSP(t *testing.T) {
	browser, err := exec.LookPath("chromium")
	if err != nil {
		t.Skip("chromium not on PATH")
	}

	for _, strict := range []bool{false, true} {
		name := map[bool]string{false: "no-strict control", true: "strict (CSP)"}[strict]
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			var seen []string
			gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				seen = append(seen, r.Host+r.URL.Path)
				mu.Unlock()
				host, port, _ := net.SplitHostPort(r.Host)
				if host == "localhost" && strings.HasPrefix(r.URL.Path, "/ipfs/") {
					cid := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/ipfs/"), "/", 2)[0]
					http.Redirect(w, r, "http://"+cid+".ipfs.localhost:"+port+"/", http.StatusMovedPermanently)
					return
				}
				w.Header().Set("Content-Type", "image/png")
				w.Write(b2PNG)
			}))
			defer gw.Close()
			_, gwPort, _ := net.SplitHostPort(gw.Listener.Addr().String())

			appcfg := gnoweb.NewDefaultAppConfig()
			appcfg.ChainID = "dev"
			appcfg.RenderConfig.IPFSGateway = "http://localhost:" + gwPort
			appcfg.Aliases = map[string]gnoweb.AliasTarget{
				"/ipfs-page": {Value: "![x](ipfs://" + b2CID + ")", Kind: gnoweb.StaticMarkdown},
			}
			app, err := gnoweb.NewRouter(log.NewTestingLogger(t), appcfg)
			require.NoError(t, err)
			web := httptest.NewServer(SecureHeadersMiddleware(app, strict, "http://127.0.0.1:26657", appcfg.RenderConfig.IPFSGateway))
			defer web.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			var stderr bytes.Buffer
			cmd := exec.CommandContext(ctx, browser, "--headless=new", "--no-sandbox", "--disable-gpu",
				"--user-data-dir="+t.TempDir(), "--virtual-time-budget=8000", "--enable-logging=stderr", "--v=0",
				"--dump-dom", web.URL+"/ipfs-page")
			cmd.Stderr = &stderr
			dom, err := cmd.Output()
			require.NoError(t, err, stderr.String())
			require.Contains(t, string(dom), `src="http://localhost:`+gwPort+`/ipfs/`+b2CID+`"`, "page carries the rewritten image")

			mu.Lock()
			defer mu.Unlock()
			t.Logf("gateway saw: %q", seen)
			for _, line := range strings.Split(stderr.String(), "\n") {
				if strings.Contains(line, "Content Security Policy") {
					t.Logf("chromium: %s", line)
				}
			}
			want := b2CID + ".ipfs.localhost:" + gwPort + "/"
			found := false
			for _, s := range seen {
				found = found || s == want
			}
			require.True(t, found, "the redirected image request %q never reached the gateway", want)
		})
	}
}
GO
go test ./gno.land/cmd/gnoweb -run TestB2SubdomainGatewayRedirectCSP -count=1 -v
rm gno.land/cmd/gnoweb/zz_subdomain_csp_test.go
```

```
# no-strict control: PASS
gateway saw: ["localhost:40189/ipfs/bafy..." "bafy....ipfs.localhost:40189/"]
# strict (CSP): FAIL
gateway saw: ["localhost:39239/ipfs/bafy..."]
chromium: Loading the image 'http://bafy....ipfs.localhost:39239/' violates the following Content Security Policy directive: "img-src 'self' data: ...
```

</details>

## gno.land/pkg/gnoweb/ipfs.go:17 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/ipfs.go#L17) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/ipfs.go#L17) · Warning [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4294596545)

Images stored as `https://ipfs.filebase.io/ipfs/<cid>` break once an operator moves `-ipfs-gateway`: [`ipfsRetiredGateways`](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L18-L26) lacks this default host, and [`img-src`](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/cmd/gnoweb/main.go#L470-L471) drops it.

<details><summary>repro</summary>

Step 3 fails because the stored filebase URL is not rewritten and filebase leaves `img-src` after the switch.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6270 -R gnolang/gno
cat > gno.land/cmd/gnoweb/zz_gateway_switch_test.go <<'GO'
package main

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
	body, _ = serve("https://gw.example", "![x](ipfs://"+cid+") ![y]("+"https://ipfs.io/ipfs/"+cid+")")
	require.Equal(t, 2, strings.Count(body, `src="https://gw.example/ipfs/`+cid+`"`), "ipfs:// and ipfs.io follow the switch")

	// 3. The operator moves -ipfs-gateway to a dedicated gateway. The stored
	// URL must either follow it or stay allowed.
	body, csp = serve("https://gw.example", stored)
	rewritten := strings.Contains(body, `src="https://gw.example/ipfs/`+cid+`"`)
	allowed := strings.Contains(imgSrc(csp), gnoweb.DefaultIPFSGateway)
	t.Logf("step 3: gateway https://gw.example: stored src rewritten=%v, filebase in img-src=%v; %s", rewritten, allowed, imgSrc(csp))
	assert.True(t, rewritten || allowed, "an image stored as %s/ipfs/<cid> is blocked by the CSP after the gateway switch", gnoweb.DefaultIPFSGateway)
}
GO
go test ./gno.land/cmd/gnoweb -run TestRolloutGatewaySwitch -count=1 -v
rm gno.land/cmd/gnoweb/zz_gateway_switch_test.go
```

```
step 1: autolink rendered with label "https://ipfs.filebase.io/ipfs/bafy..."
step 2: default gateway: stored image allowed (https://ipfs.filebase.io in img-src)
step 3: gateway https://gw.example: stored src rewritten=false, filebase in img-src=false; img-src ... https://*.githubusercontent.com https://gw.example
--- FAIL: TestRolloutGatewaySwitch
```

</details>

## SKIP contribs/gnodev/setup_web.go:42 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/contribs/gnodev/setup_web.go#L42) · [↗](../../../../../.worktrees/gno-review-6270/contribs/gnodev/setup_web.go#L42) · Missing test

Missing test: no gnodev test asserts that `-web-ipfs-gateway` reaches `RenderConfig.IPFSGateway`, so deleting this assignment leaves every gnodev test green.

Not posted: PLAUSIBLE, no judge ran the deletion.

## SKIP gno.land/cmd/gnoweb/main.go:355 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/cmd/gnoweb/main.go#L355) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/cmd/gnoweb/main.go#L355) · Missing test

Missing test: no test pins the `-ipfs-gateway` flag's wiring into `RenderConfig` and the CSP, so deleting this line leaves every test green.

Not posted: PLAUSIBLE, no judge ran the deletion.

## SKIP gno.land/pkg/gnoweb/markdown/ext_ipfs.go:55 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L55) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L55) · Missing test

Missing test: no golden file covers the rewrite on a `<gno-button href="ipfs://...">` or inside columns, alerts, tables and footnotes.

Not posted: PLAUSIBLE, no judge added the fixture.

## gno.land/pkg/gnoweb/app.go:166 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/app.go#L166) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/app.go#L166) · Nit [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4294596555)

Nit: this error prints the raw gateway value, so a value [rejected for carrying credentials](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/ipfs.go#L56-L57) puts the secret in the startup log, unlike [`indexer.Redact`](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/app.go#L245).

## SKIP gno.land/pkg/gnoweb/ipfs.go:21 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/ipfs.go#L21) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/ipfs.go#L21) · Nit

Nit: `gatewayHost` accepts IPv4 shorthands such as `127.1` and `0x7f000001`, which then enter the CSP as a host the browser may normalise differently.

Not posted: PLAUSIBLE, the browser block was not observed.

## gno.land/pkg/gnoweb/ipfs_test.go:86 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/ipfs_test.go#L86) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/ipfs_test.go#L86) · Nit [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4294596562)

Test: this clone is redundant, since [`NewDefaultAppConfig`](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/app.go#L140) already clones `DefaultAliases`, and its comment claims the opposite.

## gno.land/pkg/gnoweb/markdown/ext_ipfs.go:143 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L143) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L143) · Nit [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4294596567)

Nit: `u.EscapedPath()` falls back to the decoded path when the raw path holds a space, so `https://ipfs.io/ipfs/%41<cid>/a.png` stays untouched while `.../a b.png` is rewritten with the name `A<cid>`.

## SKIP gno.land/pkg/gnoweb/markdown/ext_ipfs.go:225 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L225) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L225) · Nit

Nit: `validIPFSName` admits only alphanumerics under `/ipfs/`, so a base64url CID, prefix `u` with `-` and `_`, is never rewritten.

Not posted: PLAUSIBLE, whether gateways serve such CIDs was not checked.

## gno.land/pkg/gnoweb/markdown/ext_links.go:172 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/markdown/ext_links.go#L172) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/markdown/ext_links.go#L172) · Nit [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4294596580)

Nit: `newLinkFromAutoLink` builds the same raw-label link that [`newButtonLink`](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L77-L84) builds by hand; renamed to something not tied to autolinks, it lets `newButtonLink` drop 8 lines to 3.

## SKIP gno.land/cmd/gnoweb/main.go:471 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/gno.land/cmd/gnoweb/main.go#L471) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/cmd/gnoweb/main.go#L471) · Suggestion

Suggestion: with a default Kubo at `http://localhost:8080`, images may redirect to `<cid>.ipfs.localhost`, which the CSP does not list.

Not posted: same defect as the section on main.go:470, which carries the run.

## misc/loop/traefik/gno.yml:7 [gh](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/misc/loop/traefik/gno.yml#L7) · [↗](../../../../../.worktrees/gno-review-6270/misc/loop/traefik/gno.yml#L7) · Suggestion [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4294596585)

Suggestion: staging can drift here, since the [staging gnoweb command](https://github.com/gnolang/gno/blob/fe1c9f046d52faaeafe13f9bd48ce44c77f39e20/misc/loop/docker-compose.yml#L55-L61) passes no `--ipfs-gateway` and so follows whatever `DefaultIPFSGateway` the pulled image carries.
