// b2-lines: a subdomain gateway (Kubo's implicit default on localhost) answers
// a rewritten path URL with a redirect to <cid>.ipfs.<host>, which gnoweb's
// img-src (the bare gateway origin) does not allow, so the image never loads.
//
// Kubo docs/config.md, "Implicit defaults of Gateway.PublicGateways":
// "localhost": {"Paths": ["/ipfs", "/ipns"], "UseSubdomains": true}, and
// UseSubdomains true means "requests for content paths such as
// http://{hostname}/ipfs/{cid} produce redirect to http://{cid}.ipfs.{hostname}".
// The fake gateway below does exactly that redirect, then serves a PNG on the
// subdomain host and records every request it receives.
//
// Repro, from a plain clone of gnolang/gno with chromium on PATH:
//
//	git fetch origin pull/6270/head && git checkout --detach fe1c9f046d52faaeafe13f9bd48ce44c77f39e20
//	cp <this file> gno.land/cmd/gnoweb/b2_lines_subdomain_redirect_csp_test.go
//	go test ./gno.land/cmd/gnoweb -run TestB2SubdomainGatewayRedirectCSP -v -count=1
//
// Expected at head: the strict (CSP) case FAILS, the subdomain request never
// reaches the gateway; the non-strict control passes, so the redirect itself
// works and the CSP is what blocks it.
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
