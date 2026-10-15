# Findings in posting order, from round assemble: 14 to post, 7 SKIP, 0 refuted kept out

## Anchors that miss at the head

- #18 gnolang/gno#6075 PR description:10: the file is not at the head: fatal: path 'gnolang/gno#6075 PR description' does not exist in '74ec1b3e1ad9406d645443fffbca8ae277b8edf7'

## Body

- SKIP Nit: gnolang/gno#6075 PR description:10 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/gnolang/gno#6075 PR description#L10), outside the diff, so no anchor
  State: UNVERIFIED, band: Nit, angle: removed, on the finder's read only
  TL;DR: The description still describes the closed #6074 design: it registers `*collection.Collection`, lists a `GetToken` lookup, and says the PR stays a draft until #6074 merges. A test comment still names collection.KindsUpTo.
  Check: gh pr view 6075 -R gnolang/gno --json body,isDraft; gh pr view 6074 -R gnolang/gno --json state; grep -n '^func [A-Z]' examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno; grep -rn 'KindsUpTo' examples --include='*.gno' | grep -v ExtensionKindsUpTo
  Evidence: not run: on the finder's read

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:246 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L246) · Warning
State: CONFIRMED, band: Warning, angle: catalog
TL;DR: the collection page shows 8 of 9 published kinds with no elision mark, and past the window the badge subset follows registration order despite the godoc
Check: same artifact, gno test -run 'TestCatalogBadgesHideKindsBehindHook|TestCatalogBadgeSetDependsOnOrder': expect 8 badges plus '…' and kind0 shown, observe no '…' and kind0 missing
Details: extensionBadges calls publishedKinds(v, maxBadges+1) and marks elision only when i == maxBadges; a view-less hook in the 9-slot window leaves 8 published kinds, so the page shows 8 badges and no ellipsis while GetExtensionKinds returns 9. Separately, sorting happens after the window, so 10 views registered kind9..kind0 render kind1..kind8 plus the ellipsis and never kind0, contradicting the publishedKinds godoc that sorting makes output independent of registration order. Readers of Render lose published kinds with no sign the list is cut; it shares its root cause and fix with candidate 1. Fix: filter view-less kinds before applying the limit, and either sort over the full set or drop the order-independence claim from the godoc.
Evidence: head 74ec1b3e1, gno test -v -run 'TestCatalogBadgesHideKindsBehindHook|TestCatalogBadgeSetDependsOnOrder' ./gno.land/r/demo/defi/grc721reg/: 'badges: published 9 shown 8 elided false', FAIL 'Render shows 8 of 9 published kinds with no elision mark'; 'order: kind0 shown false', FAIL with '- extensions: `kind1` ... `kind8` …'. Base: grc721reg absent.
Artifact: tests/b3-catalog-viewless-window_test.gno

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:278 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L278) · Warning
State: CONFIRMED, band: Warning, angle: removed + catalog + lines + reach
TL;DR: one view-less hook registered first lets a 17-view collection past Register's 16-kind cap, drops a published kind from the register event, and hides the badge elision mark
Check: cp tests/b2-lines-viewless-cap_test.gno examples/gno.land/r/demo/defi/grc721reg/b2lines_cap_test.gno; cd examples && gno test -v -run TestB2Viewless ./gno.land/r/demo/defi/grc721reg/ : expect 'event kinds=16 published kinds=17' and 'badges=8 elided=false published=10'
Details: publishedKinds takes token.ExtensionKindsUpTo(limit) in registration order, then drops kinds whose ExtensionView is nil, so a view-less hook inside the window spends one slot of the bound. Register calls publishedKinds(token, maxEventKinds+1) and only aborts when the filtered set exceeds 16; with 1 hook + 17 views the window keeps 16, the abort never fires and the event lists 16 of the 17 kinds GetExtensionKinds reports. extensionBadges does the same with maxBadges+1, so 1 hook + 10 views renders 8 badges and no ellipsis. Indexers reading the register event and readers of the page lose kinds with nothing marking the list partial. The realm is new in this diff (absent at the merge base). Fix: filter view-less kinds while walking and stop once limit published kinds are collected, bounding the walk separately if a hard cap on iterations is wanted.
Evidence: head 74ec1b3e1, gno test -v -run TestB2Viewless ./gno.land/r/demo/defi/grc721reg/: 'event kinds=16 published kinds=17 maxEventKinds=16' and 'badges=8 elided=false published=10', both PASS asserting the under-report. Base: examples/gno.land/r/demo/defi/grc721reg does not exist ('No such file or directory'), so the diff introduces it.
Artifact: tests/b2-lines-viewless-cap_test.gno

## SKIP examples/gno.land/p/demo/tokens/grc721/token.gno:146 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L146) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines + reach + removed, on the finder's read only
TL;DR: ExtensionKindsUpTo's limit is pinned by no test: removing the break leaves every test green | No test pins ExtensionKindsUpTo's limit: deleting the break leaves every test green, so the bound that keeps Render and Register cost fixed is unguarded | No test pins the limit in ExtensionKindsUpTo. Deleting the break leaves every grc721 and grc721reg test green.
Check: Delete the break at examples/gno.land/p/demo/tokens/grc721/token.gno:147, run cd examples && gno test ./gno.land/p/demo/tokens/grc721/ ./gno.land/r/demo/defi/grc721reg/ : expected red, predicted green. Pin with a token_test case: three kinds registered, len(ExtensionKindsUpTo(2)) == 2 and len(ExtensionKindsUpTo(-1)) == 3 | delete lines 146-148 of token.gno (the limit check and break), run gno test ./gno.land/p/demo/tokens/grc721/... ./gno.land/r/demo/defi/grc721reg: expect a failure, observe none; grep -rn ExtensionKindsUpTo --include='*_test.gno' examples returns no hit | grep -rn 'ExtensionKindsUpTo' examples --include='*_test.gno' (no hit). Then delete token.gno:146-148 and run gno test ./examples/gno.land/p/demo/tokens/grc721/ ./examples/gno.land/r/demo/defi/grc721reg/: a pinning test should go red, but every test passes. A pinning test: register three extensions, then assert len(tok.ExtensionKindsUpTo(2)) == 2 and len(tok.ExtensionKindsUpTo(0)) == 3.
Evidence: not run: no verifier reached it; the check is still to run

## SKIP examples/gno.land/p/demo/tokens/grc721/token.gno:208 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L208) · Missing test
State: PLAUSIBLE, band: Missing test, angle: removed, on the finder's read only
TL;DR: No test pins that a rejected view leaves no entry. Moving the append above the view checks keeps TestRegisterExtensionRejectsAForeignView green.
Check: Move token.gno:208 above line 198 and run gno test ./examples/gno.land/p/demo/tokens/grc721/ -run TestRegisterExtensionRejectsAForeignView. The test should go red; it stays green. Pinning assertion: uassert.Equal(t, 0, len(tokA.ExtensionKinds())) after both panics.
Evidence: not run: no verifier reached it; the check is still to run

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:252 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L252) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: the persistent-realm gate is pinned only against NewUserRealm (pkgPath ""), never against a maketx run realm, the round-2 Critical it was added for
Check: mutate line 51 to prev.IsUserCall(); gno test ./gno.land/r/demo/defi/grc721reg stays green; go test -run TestTestdata/zz_b3_run_gate ./gno.land/pkg/integration/ turns red
Evidence: not run: no verifier reached it; the check is still to run

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:351 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L351) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: No test registers through a sub-identity (cur.Sub), although Register accepts one because IsUser() is false for a sub and keys the entry on its synthesized 'host#subpath' path
Check: grep -n 'Sub(' examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno returns nothing; the sub-identity filetest tests/b3-reach-subrealm-key.gno registers through cross(sub) and passes
Evidence: not run: no verifier reached it; the check is still to run

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:555 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L555) · Missing test
State: PLAUSIBLE, band: Missing test, angle: shape nobody ran (cross-transaction persistence), on the finder's read only
TL;DR: Every grc721reg test runs inside one test function under testing.SetRealm; no txtar or filetest has an issuer realm package register its token in init and then mint or attach an extension in a later transaction while a separate consumer realm reads GetExtension, the shape grc20reg pins in grc20_registry_wrappers.txtar
Check: Write gno.land/pkg/integration/testdata/grc721_registry.txtar modeled on grc20_registry_wrappers.txtar: addpkg an issuer realm whose init does NewToken, NewRoyalty and grc721reg.Register; gnokey maketx call a Mint wrapper in a later tx; addpkg a consumer realm calling grc721reg.GetExtension(key, "royalty").(*royalty.Royalty).RoyaltyInfo; run go test ./gno.land/pkg/integration -run TestTestdata/grc721_registry and expect every step OK.
Evidence: not run: no verifier reached it; the check is still to run

## examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno:20 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L20) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: an enumerable attached after minting is now published through the registry and reports TotalSupply 0 against a core supply of 2
Check: cp tests/b1-lines-removed-reach-catalog-late-enumerable_test.gno examples/gno.land/r/demo/defi/grc721reg/zz_late_enumerable_test.gno; cd examples && gno test ./gno.land/r/demo/defi/grc721reg -run TestLateEnumerable -v: expect NewEnumerable to panic or the view to agree with the core; observe TotalSupply 0 vs 2
Details: NewEnumerable now passes its view to coreLedger.RegisterExtension(led, view), so grc721reg.GetExtension hands it to any realm; attached after two mints it reports TotalSupply 0, TokenByIndex(0) errors, and after a transfer indexes only the moved token. The missing guard predates the diff (base NewEnumerable has the same body and the same 'Attach it before the first mint' comment); the diff makes the stale view reachable by third parties. It breaks a documented precondition, and consumers already trust issuer-supplied views wholesale, since any issuer can register a view of kind 'enumerable' returning anything. TestRegisteredCollectionStaysLive attaches royalty before any mint, not a late enumerable, so it does not present this case as supported. Fix: panic in NewEnumerable when core.TotalSupply() > 0.
Evidence: head 74ec1b3e1, gno test -v -run TestLateEnumerable ./gno.land/r/demo/defi/grc721reg/: 'core TotalSupply: 2 | published enumerable TotalSupply: 0', 'TokenByIndex(0) err: true | TokenOfOwnerByIndex(carol,0): 1 true | TokenOfOwnerByIndex(bob,0) err: true', FAIL. Base enumerable/token.gno: 'coreLedger.RegisterExtension(led)' with the view returned only to the caller; grc721reg absent at base.
Artifact: tests/b1-lines-removed-reach-catalog-late-enumerable_test.gno

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:122 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L122) · Nit
State: UNVERIFIED, band: Nit, angle: removed, on the finder's read only
TL;DR: PR body still describes the removed collection package and a GetToken that head does not define
Check: grep -n 'func Get' examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno; gh pr view 6075 -R gnolang/gno --json body
Evidence: not run: on the finder's read

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:204 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L204) · Nit
State: CONFIRMED, band: Nit, angle: lines + reach
TL;DR: A collection registered by a sub-realm gets an index 'info' link that opens 'Unknown collection' instead of its page.
Check: Same filetest: the golden Output carries the info link with a raw '#' and the line 'Unknown collection: gno\.land/r/demo/regsub' for the cut path. | grep -n 'r/demo/defi/grc721reg' examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno: one literal at line 204, no use of runtime.CurrentRealm().PkgPath()
Details: Register only rejects prev.IsUser(), so a sub-realm registers, and its PkgPath is 'host#subpath' (uverse.go:251 subRealmSep = "#"). Line 204 builds infoLink := "/r/demo/defi/grc721reg:" + item.Key with the raw key, so the browser treats '#dao.SUB' as a fragment and Render receives only the host path, which Get misses. The hardcoded realm path half is weak: the realm lives at that fixed path, so only the '#' half costs a visitor anything. Fix: escape the key in the link (url.PathEscape or replace '#' with '%23') and add a sub-realm row to the render filetest.
Evidence: Read: grc721reg.gno:51 'if prev.IsUser() {' is the only identity gate; :204 'infoLink := "/r/demo/defi/grc721reg:" + item.Key'; uverse.go:251 'const subRealmSep = "#"'. Finder's filetest golden Output: '[info](/r/demo/defi/grc721reg:gno.land/r/demo/regsub#dao.SUB)' and 'Unknown collection: gno\.land/r/demo/regsub' for the fragment-cut path. Not re-run: read-only judge, the filetest has to be copied into the package to run.
Artifact: tests/b3-reach-subrealm-key.gno

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:286 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L286) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: A collection with more than nine viewed kinds shows a badge set chosen by registration order, so two collections with the same kinds can show different badges.
Check: TestLinesBadgesIgnoreRegistrationOrder in tests/b3-lines-viewless-window.gno
Details: extensionBadges calls publishedKinds(v, maxBadges+1) with maxBadges = 8; publishedKinds takes token.ExtensionKindsUpTo(9), which walks tok.ledger.extensions in registration order and breaks at 9, and only then sorts. The sort orders the window but cannot choose it, so the comment's claim that output 'does not depend on the order the issuer registered them' holds only up to nine kinds. Fix: narrow the comment to the window, or sort the full (bounded) kind list before cutting it.
Evidence: Read: grc721reg.gno:246 'kinds := publishedKinds(v, maxBadges+1)'; :278 'for _, kind := range token.ExtensionKindsUpTo(limit) {' then :286 'sort.Strings(kinds)'; token.gno:144-147 'for _, e := range tok.ledger.extensions { ... if limit > 0 && len(kinds) == limit { break'; consts.gno:14 'maxBadges  = 8'. Finder's TestLinesBadgesIgnoreRegistrationOrder logged 'a0 b0..b6 …' vs 'b0..b7 …', FAIL; not re-run (read-only judge).
Artifact: tests/b3-lines-viewless-window.gno

## examples/gno.land/p/demo/tokens/grc721/token.gno:142 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L142) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: 
Check: Grep the callers: grc721reg.gno:158 calls publishedKinds(v, 0) and grc721reg.gno:279 calls token.ExtensionView(kind) inside the loop. Time GetExtensionKinds at n=100 and n=1000.

## examples/gno.land/p/demo/tokens/grc721/token.gno:155 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L155) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: 
Check: Filetest: realm A declares a type whose OnBurn increments a counter, calls RegisterExtension(v, v) and exports the *Token. Realm B calls tok.ExtensionView(kind).(grc721.Extension).OnBurn("1") and reads the counter.

## examples/gno.land/p/demo/tokens/grc721/token.gno:156 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L156) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines + reach
TL;DR: GetExtensionKinds costs quadratic gas in the issuer's own extension count; only that issuer's key is affected
Check: cp tests/b2-lines-kinds-quadratic_test.gno examples/gno.land/r/demo/defi/grc721reg/b2lines_cost_test.gno; cd examples && gno test -v -print-runtime-metrics -run TestB2Cost ./gno.land/r/demo/defi/grc721reg/ : Read minus Setup gas grows ~4x from 200 to 400 hooks
Details: publishedKinds(v, 0) walks every kind and calls ExtensionView per kind, which re-scans led.extensions via lookupExtension, so the read is n(n+1)/2. Measured read cost 28.0M gas at 200 hooks and 107.3M at 400 (x3.83). The issuer pays the same quadratic shape to register them (setup 33.6M -> 118.4M), and the cost lands only on calls for that issuer's own key; no caller in the tree loops GetExtensionKinds over the registry, so it is a latent risk rather than a defect a maintainer would block. Fix: have ExtensionKindsUpTo return (kind, view) pairs, or add a single-pass helper, so publishedKinds is linear.
Evidence: head 74ec1b3e1, gno test -v -print-runtime-metrics -run TestB2Cost ./gno.land/r/demo/defi/grc721reg/: Setup200 GAS 33585552, Read200 61596876, Setup400 118351897, Read400 225629763; read delta 28.0M vs 107.3M. Base has no Token.ExtensionView and no grc721reg.
Artifact: tests/b2-lines-kinds-quadratic_test.gno

## examples/gno.land/p/demo/tokens/grc721/token.gno:193 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L193) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: 
Check: Register an extension whose ExtensionKind returns strings.Repeat("k", 4096) after grc721reg.Register, then len(GetExtensionKinds(key)[0]) == 4096. Fix: a validKind check beside validSymbol in RegisterExtension

## examples/gno.land/p/demo/tokens/grc721/token.gno:198 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L198) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog + lines + reach + removed (class sweep)
TL;DR: 
Check: Filetest: realm A registers a combined hook+view that counts OnTransfer calls and registers with grc721reg; realm B calls GetExtension, asserts grc721.Extension, calls OnTransfer; expect A's counter to move. Fix: in RegisterExtension, panic when view.(Extension) asserts ok

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:33 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L33) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines (dead parameter)
TL;DR: 
Check: Run grep -n slug on grc721reg.gno.

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:60 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L60) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: 
Check: cp tests/b3-reach-subrealm-key.gno examples/gno.land/r/demo/defi/grc721reg/z_reachsub_filetest.gno; cd examples && gno test -v ./gno.land/r/demo/defi/grc721reg

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:79 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L79) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed + catalog
TL;DR: 
Check: Add a grc721reg_test case: 17 view-less RegisterExtension calls, then royalty.NewRoyalty, then Register; assert GetExtensionKinds contains royalty while Render shows _core only_

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:158 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L158) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: 
Check: read token.gno:155-162 and 211-219 against grc721reg.gno:158; time GetExtensionKinds at N=100 and N=1000 and compare GAS lines
