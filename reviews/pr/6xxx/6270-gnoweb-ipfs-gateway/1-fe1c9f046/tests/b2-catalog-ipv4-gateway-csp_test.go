// b2-catalog, hypothesis checked and refuted: normalizeIPFSGateway accepts a
// gateway whose host is an IPv4 address ("host must be a DNS name or an IPv4
// address", ipfs.go), NewRouter starts cleanly with it, and
// SecureHeadersMiddleware writes it into img-src verbatim. CSP3 host-part
// matching (6.7.2.10) returns "Does Not Match" for a host that is not a
// domain, and its grammar note says only 127.0.0.1 matches as an IP source, so
// the hypothesis was that every rewritten image on an IPv4 gateway other than
// 127.0.0.1 is refused by the browser.
//
// The fake gateway below listens on 127.0.0.2 (loopback, so http is accepted,
// as it would be on any https IPv4 host) and records every /ipfs/ request it
// receives; a request the CSP refuses never reaches it. 127.0.0.1 is the
// control the spec carves out, and the same 127.0.0.2 gateway with -no-strict
// (no CSP) is the control proving the page does request the image.
//
// Repro, from a plain clone of gnolang/gno with chromium (or
// google-chrome-stable) and optionally firefox on PATH:
//
//	git fetch origin pull/6270/head && git checkout --detach fe1c9f046d52faaeafe13f9bd48ce44c77f39e20
//	cp <this file> gno.land/cmd/gnoweb/b2_catalog_ipv4_gateway_csp_test.go
//	go test ./gno.land/cmd/gnoweb -run TestB2CatIPv4GatewayCSP -v -count=1
//
// Measured at head (chromium and firefox headless, 2026-10-16): every subtest
// PASSES. Both engines load the image from http://127.0.0.2:<port> under an
// img-src naming it, and refuse it when img-src omits it (the negative
// control), so browsers match IPv4 sources despite the CSP3 note: the
// candidate is refuted.
package main

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/require"
)

const b2CatCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

var b2CatPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=")

func b2CatBrowsers(t *testing.T) map[string]func(ctx context.Context, dir, url string) *exec.Cmd {
	out := map[string]func(ctx context.Context, dir, url string) *exec.Cmd{}
	for _, name := range []string{"chromium", "google-chrome-stable", "google-chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			out["chromium"] = func(ctx context.Context, dir, url string) *exec.Cmd {
				return exec.CommandContext(ctx, p, "--headless=new", "--no-sandbox", "--disable-gpu",
					"--no-first-run", "--user-data-dir="+dir, "--virtual-time-budget=8000", "--dump-dom", url)
			}
			break
		}
	}
	if p, err := exec.LookPath("firefox"); err == nil {
		out["firefox"] = func(ctx context.Context, dir, url string) *exec.Cmd {
			return exec.CommandContext(ctx, p, "--headless", "--no-remote", "--profile", dir,
				"--screenshot", filepath.Join(dir, "shot.png"), url)
		}
	}
	if len(out) == 0 {
		t.Skip("no browser on PATH")
	}
	return out
}

func TestB2CatIPv4GatewayCSP(t *testing.T) {
	browsers := b2CatBrowsers(t)
	cases := []struct {
		host   string
		strict bool
		cspGW  string // img-src gateway when set, to prove the CSP is enforced
	}{
		{"127.0.0.1", true, ""},                  // the one IP CSP3 matches
		{"127.0.0.2", false, ""},                 // control: no CSP, the image is requested
		{"127.0.0.2", true, ""},                  // the case under test: IPv4 gateway in img-src
		{"127.0.0.2", true, "http://127.0.0.3:1"}, // negative control: gateway absent from img-src, must get 0 hits
	}
	for bname, browser := range browsers {
		for _, tc := range cases {
			name := bname + "/" + tc.host + "/" + map[bool]string{true: "strict", false: "no-strict"}[tc.strict]
			if tc.cspGW != "" {
				name += "/csp-without-gateway"
			}
			t.Run(name, func(t *testing.T) {
				ln, err := net.Listen("tcp", tc.host+":0")
				require.NoError(t, err)
				var hits atomic.Int64
				gw := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/ipfs/") {
						hits.Add(1)
					}
					w.Header().Set("Content-Type", "image/png")
					w.Write(b2CatPNG)
				})}
				go gw.Serve(ln)
				defer gw.Close()

				appcfg := gnoweb.NewDefaultAppConfig()
				appcfg.ChainID = "dev"
				appcfg.RenderConfig.IPFSGateway = "http://" + ln.Addr().String()
				appcfg.Aliases["/p"] = gnoweb.NewStaticAlias("![x](ipfs://" + b2CatCID + ")")
				router, err := gnoweb.NewRouter(log.NewTestingLogger(t), appcfg)
				require.NoError(t, err, "gnoweb accepts the IPv4 gateway")
				// Same wiring as setupWeb: the normalized gateway feeds the CSP.
				cspGW := appcfg.RenderConfig.IPFSGateway
				if tc.cspGW != "" {
					cspGW = tc.cspGW
				}
				page := httptest.NewServer(SecureHeadersMiddleware(router, tc.strict, appcfg.NodeRemote, cspGW))
				defer page.Close()

				resp, err := http.Get(page.URL + "/p")
				require.NoError(t, err)
				resp.Body.Close()
				t.Logf("CSP: %q", resp.Header.Get("Content-Security-Policy"))

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				dom, _ := browser(ctx, t.TempDir(), page.URL+"/p").Output()
				if bname == "chromium" {
					require.Contains(t, string(dom), `src="http://`+ln.Addr().String()+"/ipfs/"+b2CatCID, "page carries the rewritten image")
				}
				time.Sleep(500 * time.Millisecond)
				t.Logf("gateway /ipfs/ hits: %d", hits.Load())
				if tc.cspGW != "" {
					require.Zero(t, hits.Load(), "CSP not enforced by this harness")
					return
				}
				require.Greater(t, hits.Load(), int64(0), "image request never reached gateway %s", appcfg.RenderConfig.IPFSGateway)
			})
		}
	}
}
