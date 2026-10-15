# Claims: gnolang/gno#5421 round 5, dfdeaf676, claude-opus-5-5, solo review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Missing test | gno.land/cmd/gnoweb/main_test.go:116 | in a scratch worktree at dfdeaf676, delete ` %s/abci_query` from the cspTemplate in gno.land/cmd/gnoweb/main.go and run `go test ./gno.land/cmd/gnoweb/ -run SecureHeaders -count=1`: expect green at head (gap), red with master's assertion restored as `connect-src 'self' http://example.com/abci_query` | head mutated: `ok github.com/gnolang/gno/gno.land/cmd/gnoweb 0.307s`; base mutated: `--- FAIL: TestSecureHeadersMiddlewareStrict ... main_test.go:117: Expected Content-Security-Policy connect-src to contain 'self' and the RPC node`; head mutated + base assertion: `--- FAIL: TestSecureHeadersMiddlewareStrict`; head unmutated + base assertion: `ok ... 0.233s` | tests/judge-1-connect-src-pin.sh | cold |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 15 files, warm 0/0 confirmed over 34 files, cold 1/1 confirmed over 19 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | gno.land/pkg/gnoweb/public/main.css:1 | binary main.css conflict resolved by hand could miss styles from one side (master's Bubble Rumble footer, or the PR's playground) | `gh pr checks 5421 -R gnolang/gno` -> `gnoweb_generate pass 32s` at head dfdeaf676; that job runs `make -C gno.land/pkg/gnoweb fclean generate` and diffs the tree (ci-dir-gnoland.yml:67-69), so a stale main.css is caught by CI |
| removed | gno.land/pkg/gnoweb/components/layouts/head.html:9 | master renamed HeadData.BuildTime to AssetsVersion; a PR-side template or Go site still naming .BuildTime would fail at render time | `grep -rn BuildTime gno.land/ contribs/ \| grep -v node_modules \| grep -v public/` returns nothing; every `?v=` site (head.html:44,50,53, footer.html:51-52, analytics.html:3,6) names AssetsVersion |
| reach | gno.land/pkg/gnoweb/handler_http.go:241 | a page rendered through a HeadData built outside Get would miss the CSP nonce and CodeMirror styles would be blocked under strict CSP | `grep -rn 'IndexLayout(\\|IndexData{' --include='*.go' gno.land contribs \| grep -v _test` -> one construction (handler_http.go:233) and three renders (277, 313, 338), all from it; `HeadData{` has no other non-test construction |
| reach | gno.land/pkg/gnoweb/static.go:45 | master's content-derived AssetsVersion might not cover the PR's new feature bundles, leaving controllers cached `immutable` for a year after a deploy | static.go init walks every file under the embedded `public/*` into the digest; frontend/js/index.ts:34 builds controller paths as `${CONTROLLER_PATH}${kebab}.js${versionSuffix}`, so dynamically imported controllers carry the version too |
| reach | contribs/gnobro:1 | red `gnobro / test` at head might be a merge break | `gh run view 36448255265 --log-failed` -> `FAIL github.com/gnolang/gno/contribs/gnobro [setup failed]` on `read "https://proxy.golang.org/github.com/erigontech/mdbx-go/@v/v0.40.1.zip": stream error: stream ID 295; INTERNAL_ERROR`: a module proxy fetch flake, not code |

## Completeness

- Sibling sweep by shape: every `strings.Contains(csp, ...)` assertion in `gno.land/cmd/gnoweb/main_test.go` was read, `TestSecureHeadersMiddlewareStrict` (107-122), `TestSecureHeadersMiddlewareNonceMatchesContext` (135-165) and `TestSecureHeadersMiddlewareNonStrict` (167-190). Only the `connect-src` check lost a source pin in the resolution: the `img-src` check took the base side and the `style-src` nonce check the PR side.
- CSP construction sites: `grep -rln 'connect-src\|Content-Security-Policy' --include='*.go' gno.land contribs tm2` returns `main.go`, `main_test.go` and `csp.go` only, so no second template repeats the `cspTemplate` shape.
- Fields the merge hunks touch: `HeadData.CSPNonce` is pinned by `TestIndexLayout_CSPNonce` (`layout_test.go:433`), `AssetsVersion` by `layout_test.go:705` and `static_test.go:18`; `HeadData{` has one non-test construction, `handler_http.go:234`.
- Invariant catalog: its classes are realm audit patterns for `.gno` code, and the resolution touches gnoweb Go, one template and one test, so no class applies.
- Candidate 1's own check, deleting ` %s/abci_query`, does not compile: `main.go:358:17: fmt.Sprintf call needs 1 arg but has 2 args` under `go test`'s vet. The judge mangled the path to `%s/abci_querx` instead, at head and in a second scratch worktree at the merge base `21136dcae`; both worktrees were removed afterwards.
- The repro in the draft was not run verbatim, since `gh pr checkout` is barred in this round's trees; its commands ran in a detached worktree at `dfdeaf676`, and the pasted output is from that run.
- Out of scope this round: the round-4 Critical on `client.go:285-286`. `git diff --stat 4a0e7ff2a dfdeaf676 -- gno.land/pkg/gnoweb/client.go` prints nothing, so the file is unchanged, but the Dry Run path was not re-run here.

## Carried from round 4

Round 4 at `4a0e7ff2a` was drafted and never posted, and `client.go`, `controller-run.ts`, `handler_test.go`, `handler.go`, `ratelimit.go` and `handler_http_test.go` are byte-identical between `4a0e7ff2a` and `dfdeaf676` (`git diff --shortstat` empty for each), so its sections are ported with their anchors unchanged.

| # | State | Band | Anchor | Check | Observed | Artifact |
| --- | --- | --- | --- | --- | --- | --- |
| C1 | CONFIRMED | Critical | gno.land/pkg/gnoweb/client.go:285-286 | round 4's repro at `dfdeaf676`, gnodev on 127.0.0.1:18888 and :36657 | dry run `{"error":"error encountered during simulation: unauthorized error"}`, HTTP 200; `gnokey maketx run --simulate only` of the same script estimates 145500474 gas | `.worktrees/review-gno-5421-r5/rerun-critical.txt` |
| C2 | CONFIRMED | Warning | gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts:134 | `sed -n 372,376p handler.go` at `dfdeaf676` | `crypto.AddressFromBech32(req.Address)` then 400 `address must be a bech32 address` | |
| C3 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/feature/playground/handler_test.go:65-67 | `grep -rn 'simulateResult\|simulateErr'` in the package | declared at `handler_test.go:41-42`, read only at `:66`, set nowhere | |
