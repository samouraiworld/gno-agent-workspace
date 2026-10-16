// Round 2, solo-finder-2: both round-1 Warnings hold at 36b475912 (dropped rows, no finding).
// Repro from a plain clone of gnolang/gno (needs chromium on PATH for the CSP case):
//   git fetch origin pull/6270/head && git checkout --detach 36b475912eb31badc961e9b35fa765def60f250f
//   cp <this file> gno.land/cmd/gnoweb/zz_r2f2_test.go
//   go test ./gno.land/cmd/gnoweb -run TestR2 -count=1 -v
//   rm gno.land/cmd/gnoweb/zz_r2f2_test.go
// Observed: --- PASS: TestR2SubdomainRedirectCSP (ipfs, ipns dnslink), --- PASS: TestR2GatewaySwitch (4 stored filebase shapes rewritten).

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const r2CID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

var r2PNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=")

// Kubo-like subdomain gateway on localhost: /ipfs/<cid> -> <cid>.ipfs.localhost,
// /ipns/<dnslink> -> <inlined>.ipns.localhost.
func TestR2SubdomainRedirectCSP(t *testing.T) {
	browser, err := exec.LookPath("chromium")
	if err != nil {
		t.Skip("chromium not on PATH")
	}
	cases := []struct{ name, md, wantSeen string }{
		{"ipfs", "![x](ipfs://" + r2CID + ")", r2CID + ".ipfs.localhost"},
		{"ipns dnslink", "![x](ipns://en.wikipedia-on-ipfs.org/i.png)", "en-wikipedia--on--ipfs-org.ipns.localhost"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var seen []string
			gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				seen = append(seen, r.Host+r.URL.Path)
				mu.Unlock()
				host, port, _ := net.SplitHostPort(r.Host)
				if host == "localhost" {
					parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 3)
					if len(parts) >= 2 && (parts[0] == "ipfs" || parts[0] == "ipns") {
						label := parts[1]
						if parts[0] == "ipns" {
							label = strings.ReplaceAll(strings.ReplaceAll(label, "-", "--"), ".", "-")
						}
						rest := "/"
						if len(parts) == 3 {
							rest += parts[2]
						}
						http.Redirect(w, r, "http://"+label+"."+parts[0]+".localhost:"+port+rest, http.StatusMovedPermanently)
						return
					}
				}
				w.Header().Set("Content-Type", "image/png")
				w.Write(r2PNG)
			}))
			defer gw.Close()
			_, gwPort, _ := net.SplitHostPort(gw.Listener.Addr().String())

			appcfg := gnoweb.NewDefaultAppConfig()
			appcfg.ChainID = "dev"
			appcfg.RenderConfig.IPFSGateway = "http://localhost:" + gwPort
			appcfg.Aliases = map[string]gnoweb.AliasTarget{"/p": {Value: tc.md, Kind: gnoweb.StaticMarkdown}}
			app, err := gnoweb.NewRouter(log.NewTestingLogger(t), appcfg)
			require.NoError(t, err)
			web := httptest.NewServer(SecureHeadersMiddleware(app, true, "http://127.0.0.1:26657", appcfg.RenderConfig.IPFSGateway))
			defer web.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			var stderr bytes.Buffer
			cmd := exec.CommandContext(ctx, browser, "--headless=new", "--no-sandbox", "--disable-gpu",
				"--user-data-dir="+t.TempDir(), "--virtual-time-budget=8000", "--enable-logging=stderr", "--v=0",
				"--dump-dom", web.URL+"/p")
			cmd.Stderr = &stderr
			_, err = cmd.Output()
			require.NoError(t, err, stderr.String())
			mu.Lock()
			defer mu.Unlock()
			t.Logf("gateway saw: %q", seen)
			for _, line := range strings.Split(stderr.String(), "\n") {
				if strings.Contains(line, "Content Security Policy") {
					t.Logf("chromium: %s", line)
				}
			}
			found := false
			for _, s := range seen {
				found = found || strings.HasPrefix(s, tc.wantSeen+":")
			}
			assert.True(t, found, "redirected request to %s never reached the gateway", tc.wantSeen)
		})
	}
}

// Round 1 step 3: a stored filebase URL must follow a gateway switch.
func TestR2GatewaySwitch(t *testing.T) {
	serve := func(gateway, page string) (string, string) {
		cfg := gnoweb.NewDefaultAppConfig()
		cfg.ChainID = "dev"
		cfg.RenderConfig.IPFSGateway = gateway
		cfg.Aliases["/p"] = gnoweb.AliasTarget{Value: page, Kind: gnoweb.StaticMarkdown}
		router, err := gnoweb.NewRouter(log.NewTestingLogger(t), cfg)
		require.NoError(t, err)
		h := SecureHeadersMiddleware(router, true, "http://127.0.0.1:26657", cfg.RenderConfig.IPFSGateway)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/p", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		return rec.Body.String(), rec.Header().Get("Content-Security-Policy")
	}
	for _, stored := range []string{
		gnoweb.DefaultIPFSGateway + "/ipfs/" + r2CID,
		gnoweb.DefaultIPFSGateway + "/ipfs/" + r2CID + "/x.png?filename=x.png",
		"http://ipfs.filebase.io/ipfs/" + r2CID,
		"//ipfs.filebase.io/ipfs/" + r2CID,
	} {
		body, csp := serve("https://gw.example", "![x]("+stored+")")
		t.Logf("%s -> rewritten=%v", stored, strings.Contains(body, `src="https://gw.example/ipfs/`+r2CID))
		assert.Contains(t, body, `src="https://gw.example/ipfs/`+r2CID, stored)
		assert.Contains(t, csp, "https://gw.example")
	}
}
