package main

// Asserts that inScope's trailing-slash guard (crawl.go:191) changes no answer:
// no realm path the planner produces yields a urlOf ending in "/", so the realm
// match below already refuses every directory URL. Measured at ecf7af0f: passes
// at head, and the guard's deletion leaves the whole suite `ok`.
//
/* Run: from a gno checkout:
gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f
curl -fsSL -o misc/gnopreview/judge-160-trailing-slash-guard_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-160-trailing-slash-guard_test.go
cd misc/gnopreview && go test -vet=off -count=1 -run TestJudge160 .
rm judge-160-trailing-slash-guard_test.go
*/

import (
	"strings"
	"testing"
)

func TestJudge160TrailingSlashGuardDecidesNothing(t *testing.T) {
	realms := append([]string{"gno.land/r/gnoland/home", "gno.land/r/docs/security_patterns"}, gnowebSeedRealms...)
	for _, r := range realms {
		if strings.HasSuffix(urlOf(r), "/") {
			t.Fatalf("urlOf(%q) = %q ends in a slash; the guard is load-bearing", r, urlOf(r))
		}
	}
	c := &Crawler{Realms: realms, FileBudget: GnowebFileBudget}
	for _, u := range []string{"/r/gnoland/home/", "/r/gnoland/blog/", "/r/", "/r/gnoland/home/$source"} {
		base, _, _ := splitURL(u)
		matched := false
		for _, r := range realms {
			if base == urlOf(r) {
				matched = true
			}
		}
		// The realm match alone already refuses every one of these.
		if matched {
			t.Errorf("%q matches a realm; the guard is what refuses it", u)
		}
		if c.inScope(u) {
			t.Errorf("inScope(%q) = true", u)
		}
	}
}
