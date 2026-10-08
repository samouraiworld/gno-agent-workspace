# Review: [#6262](https://github.com/gnolang/gno/pull/6262)
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. One Warning the branch ships: an "N matching" link whose directory is itself a realm or package opens that one package instead of a listing.
Model: claude-opus-5-5, standard review (finders xhigh, the other stages high)
Commit: c5ae4b13a8883303a382d6b1ae1c61b929fd048c
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6262 c5ae4b13a`
Round: 1. Two finders and one judge that also answered the completeness question, 11 candidates, each judged from scratch by an agent that was not its finder: both Warnings and two Suggestions rerun from their artifacts, the rest read.

## Body

## gno.land/pkg/gnoweb/counterpart.go:97 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L97) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L97) · Warning
When the twin's directory is itself a realm or package, `return dir, n` makes the "N matching" link open that one package instead of a listing.

> Headless Chromium 153 via `capture-web.mjs`, this branch's gnoweb on a stub client: the menu on `/p/tests/vm/crossrealm`, then the `/r/tests/vm` realm its "2 matching realms" link opens.

![The p menu on /p/tests/vm/crossrealm offering "2 matching realms /r/tests/vm", and the /r/tests/vm realm page that link opens, which lists no realm](https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/media/n-matching-opens-realm-chromium.png)

<details><summary>repro</summary>

On `/p/tests/vm/crossrealm` the "2 matching realms" link opens the `/r/tests/vm` realm, and the walk's [`return dir, n`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L112) links `/r/gov/dao` from `/p/gov/dao/utils` the same way.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6262 -R gnolang/gno
cat > gno.land/pkg/gnoweb/zz_dir_is_package_test.go <<'EOF'
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
EOF
go test -count=1 -run TestSoloFinder1_DirIsPackage -v ./gno.land/pkg/gnoweb
rm gno.land/pkg/gnoweb/zz_dir_is_package_test.go
```

Each subtest fails: the page the "N matching" link opens names none of the N paths.

```text
page /p/tests/vm/crossrealm: switch href="/r/tests/vm" label="2 matching realms"
follow /r/tests/vm: status=200 explorer-mode=false
  followed page does not mention /r/tests/vm/crossrealm
  followed page does not mention /r/tests/vm/subtests
# …
page /p/gov/dao/utils: switch href="/r/gov/dao" label="3 matching realms"
follow /r/gov/dao: status=200 explorer-mode=false
# …
page /r/alice/golf/v1: switch href="/p/alice/golf" label="2 matching packages"
follow /p/alice/golf: status=200 explorer-mode=false
# …
--- FAIL: TestSoloFinder1_DirIsPackage (0.00s)
```

[`GetPackageView`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/handler_http.go#L504-L532) sends a realm path to `GetRealmView` and a pure one to `GetDirectoryView`; [`GetPathsListView`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/handler_http.go#L1103) is reached only when no package lives at the path.

</details>

## gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3991 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3991) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3991) · Nit
Nit: `inset: auto` overrides the `@supports not (top: anchor(bottom))` fallback, so a browser without anchor positioning opens the menu away from the header.

> Headless Chromium 153, `capture-web.mjs` clicking the `r` button on this branch's gnoweb: above, `main.css` as committed; below, with `anchor(` renamed, as without anchor positioning.

![The menu under the r button with the stylesheet as shipped, and pinned to the window's top-left corner over the breadcrumb once anchor() is unavailable](https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/media/kind-switch-fallback-chromium.png)

<details><summary>layout probe</summary>

The committed `public/main.css` holds the fallback before the main rule:

```css
@supports not (top:anchor(bottom)){.b-kind-switch{left:var(--g-space-16);top:var(--g-space-14)}}
.b-kind-switch{position-anchor:--kind-switch;…left:auto;left:anchor(left);…top:auto;top:anchor(bottom)}
```

The menu's position in Chromium after `showPopover()` on a rendered `/r/alice/golf/game` page, once with the bundle as shipped and once with every `anchor(` renamed so the `@supports not` branch holds. The second run lands at 0/0, where the fallback asks for `--g-space-14`/`--g-space-16`:

```text
== shipped
fallback @supports active: false
menu top/left px: 45/91
== noanchor
fallback @supports active: true
menu top/left px: 4/0
computed top/left: 0px/0px
```

Neither browser here takes that branch on its own: Firefox 156 answers `CSS.supports("top: anchor(bottom)")` with `true`, so the `(Firefox today)` comment above the block no longer holds either. Forced, the fallback lands elsewhere in each engine, the window's corner in Chromium and the button's own spot over the breadcrumb in Firefox.

> Headless Firefox 156, `firefox --headless --screenshot` on `/r/alice/golf/game` served by this branch's gnoweb, the menu opened by a `showPopover()` call the fixture adds to the page: above, the committed `main.css`; below, the same file with every `anchor(` renamed, the menu's box at top/left 19/91 px against Chromium's 4/0.

![Firefox 156: the menu under the r button with the stylesheet as shipped, and laid over the breadcrumb at the button's position once anchor() is unavailable](https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/media/kind-switch-fallback-firefox-156.png)

</details>

## gno.land/pkg/gnoweb/counterpart.go:90 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L90) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L90) · Nit
Nit: `gopath.Dir(m) == dir` counts only the twin's direct siblings, while the listing the link opens shows every package below that directory.

## gno.land/pkg/gnoweb/counterpart.go:213 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L213) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L213) · Suggestion
Suggestion: errors are never cached, so a slowly failing `ListPaths` costs every view of that root the full 300 ms and one more query.
A short failure entry would cost one lookup per root.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6262 -R gnolang/gno
cat > gno.land/pkg/gnoweb/zz_grace_errors_test.go <<'EOF'
// solo-finder-1: the 300 ms grace opens when the page collects, after its own
// queries, and a failed lookup is never cached, so a lookup that fails slower
// than the grace adds 300 ms and one ListPaths to every page view.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6262/head && git checkout c5ae4b13a8883303a382d6b1ae1c61b929fd048c
//	cp <this file> gno.land/pkg/gnoweb/solo_finder1_grace_test.go
//	cd gno.land/pkg/gnoweb && go test -count=1 -run TestSoloFinder1_GraceAndErrors -v .
//
// Observed at c5ae4b13a (go1.25.9), Realm answering at once, ListPaths 400 ms:
//
//	slow empty answer: request 0: 300ms, requests 1-3: 0s; ListPaths calls over 4 requests: 1
//	slow error:        requests 0-3: 300ms each;        ListPaths calls over 4 requests: 4
package gnoweb_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSoloFinder1_GraceAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{{"slow empty answer", nil}, {"slow error", errors.New("node busy")}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			client := &stubClient{
				realmFunc: func(context.Context, string, string) ([]byte, error) { return []byte("hello"), nil },
				listPathsFunc: func(ctx context.Context, _ string, _ int) ([]string, error) {
					calls.Add(1)
					select {
					case <-time.After(400 * time.Millisecond):
						return nil, tc.err
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				},
			}
			h := newCounterpartHandler(t, client)
			for i := range 4 {
				start := time.Now()
				serve(h, httptest.NewRequest(http.MethodGet, "/r/alice/golf/game", nil))
				t.Logf("request %d: %v", i, time.Since(start).Round(10*time.Millisecond))
				time.Sleep(500 * time.Millisecond) // let the lookup land
			}
			t.Logf("ListPaths calls over 4 requests: %d", calls.Load())
		})
	}
}
EOF
go test -count=1 -run TestSoloFinder1_GraceAndErrors -v ./gno.land/pkg/gnoweb
rm gno.land/pkg/gnoweb/zz_grace_errors_test.go
```

The slow error pays the grace and a `ListPaths` on every request; the slow empty answer pays once.

```text
=== RUN   TestSoloFinder1_GraceAndErrors/slow_empty_answer
    request 0: 300ms
    request 1: 0s
# …
    ListPaths calls over 4 requests: 1
=== RUN   TestSoloFinder1_GraceAndErrors/slow_error
    request 0: 300ms
    request 1: 300ms
    request 2: 300ms
    request 3: 300ms
    ListPaths calls over 4 requests: 4
```

</details>

## gno.land/pkg/gnoweb/counterpart.go:247 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L247) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L247) · Suggestion
Suggestion: a full cache stores no new root until its earliest entry expires, and empty answers for made-up paths fill it as fast as real ones.
Oldest-first eviction would keep new roots cached.

## SKIP gno.land/pkg/gnoweb/components/layouts/header.html:249 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/components/layouts/header.html#L249) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/components/layouts/header.html#L249) · Nit
Nit: the `breadcrumb_click` delegate watches only the breadcrumb `<ol>`, and the menu's links sit outside it, so first-segment navigation goes uncounted.

Not posted: the description states the chip no longer fires `breadcrumb_click` and the menu links are not tracked, a choice the author made.

## SKIP gno.land/pkg/gnoweb/counterpart.go:28 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L28) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L28) · Nit
Nit: the comment promises a slow lookup never slows the page, yet a lookup still running holds the first view of its root for 300 ms.

Not posted: a finding about a code comment's wording.

## SKIP gno.land/pkg/gnoweb/counterpart.go:154 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L154) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L154) · Suggestion
Suggestion: a detached lookup holds an RPC slot up to 5 s past its page, and every page query waits on the same 32 slots.

Not posted: the latency run against an `httptest` node answering `vm/qpaths` in 2 s is still to do.

## SKIP gno.land/pkg/gnoweb/counterpart.go:182 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L182) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L182) · Suggestion
Suggestion: the page waits up to 300 ms after its own queries finish, since `time.After(counterpartGrace)` runs only once the page is built.
Arming the deadline in `startCounterpart` would cap the whole wait at 300 ms.

Not posted: the description states the page waits at most 300 ms once it is ready, which is this behaviour, and no run timed it.
