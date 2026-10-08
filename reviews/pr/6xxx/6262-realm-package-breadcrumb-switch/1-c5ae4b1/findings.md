# Findings in posting order, from round assemble: 8 to post, 1 SKIP, 2 refuted kept out

## gno.land/pkg/gnoweb/counterpart.go:97 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L97) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: an 'N matching' link whose directory is itself a realm or package opens that one package, which names none of the N
Check: go test -count=1 -run TestSoloFinder1_DirIsPackage -v ./gno.land/pkg/gnoweb with tests/solo-finder-1-dir-is-package.go copied in
Details: counterpartTarget returns gopath.Dir(twin) at line 97 and the walked dir at line 112 whenever n > 1, without checking that the directory is not a package. GetPackageView sends a realm path to GetRealmView (Render output) and a pure one to GetDirectoryView (its file list); GetPathsListView is reached only on ErrClientPackageNotFound (handler_http.go:546, 1149). Trigger: /r/alice/golf is a realm with sub-realms /r/alice/golf/v1 and /r/alice/golf/v2 (the /r/tests/vm and /r/gov/dao shape), and /p/alice/golf/v1 is viewed. The PR body promises the listing opens. Mirroring counterpartTarget over examples/gno.land finds 1 page with an N-matching link and 0 whose target is a package, so the case is reached by deployments, not by the examples tree.
Evidence: rerun at c5ae4b13a in a scratch worktree: 'page /p/tests/vm/crossrealm: switch href="/r/tests/vm" label="2 matching realms"', 'follow /r/tests/vm: status=200 explorer-mode=false', 'followed page does not mention /r/tests/vm/crossrealm'; same for /p/gov/dao/utils -> /r/gov/dao (3) and /r/alice/golf/v1 -> /p/alice/golf (2); 3 subtests FAIL. judge-1-examples-mirror.py: 'pages with an N-matching link: 1  of which target is itself a package: 0'. Merge base has no counterpart.go.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/tests/solo-finder-1-dir-is-package.go

## gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3991 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3991) · Warning
State: CONFIRMED, band: Warning, angle: lines (hot file, found in passing)
TL;DR: in a browser without anchor positioning the menu opens at the viewport's top-left corner, not under the header as the fallback intends
Check: tests/solo-finder-2-kind-switch-fallback.sh on the rendered page with the committed public/main.css, shipped and with anchor( renamed xanchor(
Details: public/main.css carries '@supports not (top:anchor(bottom)){.b-kind-switch{left:var(--g-space-16);top:var(--g-space-14)}}' at byte 117046 and the main '.b-kind-switch{...left:auto;left:anchor(left);...top:auto;top:anchor(bottom)}' after it at byte 117142, same specificity. Where anchor() does not parse, left:auto and top:auto survive and win by order. The CSS comment names Firefox as the fallback's browser, and the PR body says Firefox was not checked by hand.
Evidence: rerun: shipped 'fallback @supports active: false', 'menu top/left px: 45/91'; noanchor 'fallback @supports active: true', 'menu top/left px: 4/0', 'computed top/left: 0px/0px'. Simulated by renaming anchor( in Chromium, not run in Firefox. Merge base has no kind switch.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/tests/solo-finder-2-kind-switch-fallback.sh

## gno.land/pkg/gnoweb/components/layouts/header.html:249 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/components/layouts/header.html#L249) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the first breadcrumb segment's clicks no longer reach breadcrumb_click and the menu links are not tracked
Check: go test TestFinder2_BreadcrumbAnalytics plus tests/solo-finder-2-breadcrumb-analytics.sh in Chromium with public/js/analytics.js
Details: analytics.ts:76 delegates on 'ol[data-searchbar-target="breadcrumb"] a'; at base the chip is <a href="{{ $part.URL }}/"> inside that <ol> (base header.html:209-213), at head a <button> and the menu div is outside the <ol>. The PR body states this choice: the event no longer fires on the chip and the menu links are not tracked.
Evidence: rerun: 'menu anchors=3, menu anchors matched by breadcrumb_click selector=0'; Chromium 'control <ol> anchor /r/alice=breadcrumb_click', 'menu /p/alice/golf=none', 'menu /u/alice=none', 'menu /r/=none'
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/tests/solo-finder-2-breadcrumb-analytics.sh

## gno.land/pkg/gnoweb/counterpart.go:28 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L28) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: the counterpartGrace comment says a slow lookup never slows the page, and the first view of a slow root is served 300 ms later
Check: go test -count=1 -run TestSoloFinder1_GraceAndErrors -v ./gno.land/pkg/gnoweb, slow empty answer row
Details: comment at counterpart.go:26-28; the code waits up to counterpartGrace by design, as the PR body says.
Evidence: rerun: slow empty answer 'request 0: 300ms'
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/tests/solo-finder-1-grace-errors.go

## gno.land/pkg/gnoweb/counterpart.go:90 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L90) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the count in the twin-with-siblings branch covers only direct children of the directory while the listing it opens shows every package below it
Check: read counterpart.go:88-97 against GetPathsListView's prefix
Details: line 90 counts members with gopath.Dir(m) == dir; GetPathsListView lists prefix path.Join(domain, path)+"/" (handler_http.go:1106), every depth. Input {/p/alice/golf/v0, /p/alice/golf/v2, /p/alice/golf/ui/board} viewing /r/alice/golf/v0: label '2 matching packages', listing shows 3.
Evidence: read at head: counterpart.go:88-97, handler_http.go:1103-1107

## SKIP gno.land/pkg/gnoweb/counterpart.go:154 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L154) · Suggestion
State: PLAUSIBLE, band: Suggestion, angle: reach
TL;DR: a detached lookup holds one of the 32 RPC slots for up to 5 s after its page is served
Check: run an rpcClient against an httptest node answering vm/qpaths in 2 s and vm/qrender at once, 64 distinct roots, and time qrender against the merge base
Details: client.go:326 acquireRPCSlot(ctx, c.rpcSlots) with the lookup's WithoutCancel ctx; app.go:104 MaxConcurrentRPC: 32. Whether page queries measurably queue behind lookups was not run.
Evidence: read at head: client.go:326, app.go:104; the latency run is still to do

## gno.land/pkg/gnoweb/counterpart.go:182 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L182) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: the 300 ms grace starts when the page is ready, not when the lookup starts
Check: read counterpart.go:182 against handler_http.go:472-474 and 319-334
Details: time.After(counterpartGrace) is evaluated inside the returned closure, which prepareIndexBodyView calls after GetPackageView (handler_http.go:472-474) and the state path after State.Handle (319-334). The PR body describes exactly this ('waits at most 300 ms more'), so this is a design preference.
Evidence: read at head; not timed

## gno.land/pkg/gnoweb/counterpart.go:213 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L213) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: a lookup that fails slower than the grace is retried on every view, so each page of that root waits 300 ms and issues one more ListPaths
Check: go test -count=1 -run TestSoloFinder1_GraceAndErrors -v ./gno.land/pkg/gnoweb with tests/solo-finder-1-grace-errors.go copied in
Details: get stores only when err == nil (line 213), so a failing node gets no cache entry; the PR body states errors are not cached by design, the cost of that choice on a slow failure is the finding.
Evidence: rerun: slow empty answer 'request 0: 300ms', 'request 1: 0s', 'ListPaths calls over 4 requests: 1'; slow error 'request 0..3: 300ms' each, 'ListPaths calls over 4 requests: 4'. PASS (the test logs, it does not assert).
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1/tests/solo-finder-1-grace-errors.go

## gno.land/pkg/gnoweb/counterpart.go:247 [gh](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L247) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: once 4096 live entries fill the cache, every new root, real or junk, goes uncached until the earliest expiry
Check: read counterpart.go:247-262
Details: store returns at line 249 'if now.Before(c.sweep) { return }' and at 259-260 when no entry expired; empty answers are stored with a full TTL (line 263), so distinct nonexistent roots fill it. Effect on real pages (one ListPaths each and up to 300 ms) follows from the lines; the request rate needed was not measured.
Evidence: read at head: lines 247-262 as quoted; TestCounterpartCacheFull pins that a full cache keeps no more.
