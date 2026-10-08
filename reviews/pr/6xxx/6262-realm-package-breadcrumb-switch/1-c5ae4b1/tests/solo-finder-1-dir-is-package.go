// solo-finder-1: the "N matching" switch link opens a single package when the
// directory it targets is itself a package or realm.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6262/head && git checkout c5ae4b13a8883303a382d6b1ae1c61b929fd048c
//	cp <this file> gno.land/pkg/gnoweb/solo_finder1_dir_test.go
//	cd gno.land/pkg/gnoweb && go test -count=1 -run TestSoloFinder1_DirIsPackage -v .
//
// Observed at c5ae4b13a (go1.25.9), all three subtests FAIL:
//
//	page /p/tests/vm/crossrealm: switch href="/r/tests/vm" label="2 matching realms"
//	follow /r/tests/vm: status=200 explorer-mode=false
//	  followed page does not mention /r/tests/vm/crossrealm
//	  followed page does not mention /r/tests/vm/subtests
//	page /p/gov/dao/utils: switch href="/r/gov/dao" label="3 matching realms"
//	  followed page does not mention /r/gov/dao/impl/v0, /r/gov/dao/init/v0
//	page /r/alice/golf/v1: switch href="/p/alice/golf" label="2 matching packages"
//	  followed page does not mention /p/alice/golf/v1, /p/alice/golf/v2
//
// Merge base 41841e92f has no counterpart.go: the switch is new in this diff.
// The /r/tests/vm and /r/gov/dao shapes (a realm at the project root with
// sub-realms) exist in examples/gno.land at the reviewed head.
package gnoweb_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
)

var reSwitch = regexp.MustCompile(`<a href="([^"]*)" class="item item--primary">\s*<svg[^>]*><use[^>]*></use></svg>\s*<span class="item-label">([^<]*)</span>`)

func primaryLink(body string) (href, label string) {
	m := reSwitch.FindStringSubmatch(body)
	if m == nil {
		return "", ""
	}
	return m[1], m[2]
}

func TestSoloFinder1_DirIsPackage(t *testing.T) {
	cases := []struct {
		name string
		pkgs []*gnoweb.MockPackage
		page string
		lost []string // paths the followed page should list and does not
	}{
		{
			name: "twin with siblings under a realm dir (/r/tests/vm shape)",
			pkgs: []*gnoweb.MockPackage{
				{Path: "/r/tests/vm", Files: map[string]string{"vm.gno": "package vm"}, Functions: renderFuncs},
				{Path: "/r/tests/vm/crossrealm", Files: map[string]string{"c.gno": "package crossrealm"}, Functions: renderFuncs},
				{Path: "/r/tests/vm/subtests", Files: map[string]string{"s.gno": "package subtests"}, Functions: renderFuncs},
				{Path: "/p/tests/vm/crossrealm", Files: map[string]string{"c.gno": "package crossrealm"}},
			},
			page: "/p/tests/vm/crossrealm",
			lost: []string{"/r/tests/vm/crossrealm", "/r/tests/vm/subtests"},
		},
		{
			name: "no twin, project root is a realm (/r/gov/dao shape)",
			pkgs: []*gnoweb.MockPackage{
				{Path: "/r/gov/dao", Files: map[string]string{"dao.gno": "package dao"}, Functions: renderFuncs},
				{Path: "/r/gov/dao/impl/v0", Files: map[string]string{"i.gno": "package impl"}, Functions: renderFuncs},
				{Path: "/r/gov/dao/init/v0", Files: map[string]string{"i.gno": "package init"}, Functions: renderFuncs},
				{Path: "/p/gov/dao/utils", Files: map[string]string{"u.gno": "package utils"}},
			},
			page: "/p/gov/dao/utils",
			lost: []string{"/r/gov/dao/impl/v0", "/r/gov/dao/init/v0"},
		},
		{
			name: "twin with siblings, project root is a package",
			pkgs: []*gnoweb.MockPackage{
				{Path: "/p/alice/golf", Files: map[string]string{"golf.gno": "package golf"}},
				{Path: "/p/alice/golf/v1", Files: map[string]string{"v1.gno": "package v1"}},
				{Path: "/p/alice/golf/v2", Files: map[string]string{"v2.gno": "package v2"}},
				{Path: "/r/alice/golf/v1", Files: map[string]string{"g.gno": "package v1"}, Functions: renderFuncs},
			},
			page: "/r/alice/golf/v1",
			lost: []string{"/p/alice/golf/v1", "/p/alice/golf/v2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCounterpartHandler(t, gnoweb.NewMockClient(tc.pkgs...))
			rr := serve(h, httptest.NewRequest(http.MethodGet, tc.page, nil))
			href, label := primaryLink(rr.Body.String())
			t.Logf("page %s: switch href=%q label=%q", tc.page, href, label)
			if href == "" {
				t.Fatalf("no switch link")
			}
			rr2 := serve(h, httptest.NewRequest(http.MethodGet, href, nil))
			body := rr2.Body.String()
			listing := strings.Contains(body, "Explorer") || strings.Contains(body, "explorer")
			t.Logf("follow %s: status=%d explorer-mode=%v", href, rr2.Code, listing)
			missing := 0
			for _, p := range tc.lost {
				if !strings.Contains(body, p) {
					missing++
					t.Logf("  followed page does not mention %s", p)
				}
			}
			if missing > 0 {
				t.Errorf("label %q promises a listing; %s shows one package and omits %d of %d", label, href, missing, len(tc.lost))
			}
		})
	}
}
