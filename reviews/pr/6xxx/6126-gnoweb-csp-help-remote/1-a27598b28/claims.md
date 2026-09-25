# Claims: gnolang/gno#6126 round 1, a27598b28, claude-opus-5-5, solo review

Round shape: solo round, one agent, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/cmd/gnoweb/main_test.go:134 | tests/solo-test-misses-wiring.sh: revert main.go:283 to NodeRemote, go test ./cmd/gnoweb -run 'TestSecureHeadersMiddleware\|TestSetupWeb': expect FAIL, observe ok | bash tests/solo-test-misses-wiring.sh in a scratch worktree at a27598b28: main.go 1 insertion 1 deletion; --- PASS: TestSecureHeadersMiddlewareUsesRemoteHelp (0.00s); ok github.com/gnolang/gno/gno.land/cmd/gnoweb 0.022s | tests/solo-test-misses-wiring.sh |  |
| 2 | REFUTED | Warning | gno.land/cmd/gnoweb/main.go:389 | grep every fetch in gno.land/pkg/gnoweb/frontend/js and every .Remote in components | handler_http.go:688 Remote: h.Static.RemoteHelp; controller-copy.ts:110 if (target.origin !== location.origin) |  |  |
| 3 | REFUTED | Warning | gno.land/cmd/gnoweb/main.go:236 | read defaultWebOptions and setupWeb's fallback | main.go:236 if appcfg.RemoteHelp == "" { appcfg.RemoteHelp = appcfg.NodeRemote } |  |  |
| 4 | REFUTED | Warning | gno.land/cmd/gnoweb/main.go:341 | grep -rn SecureHeadersMiddleware over the whole tree | main.go:389 return SecureHeadersMiddleware(next, strict, cfg.RemoteHelp) is the only non-test call |  |  |

## Completeness

- Angles: lines, removed, reach and catalog asked in one pass; claims did not run, the diff carrying no doc file. Nothing came back thin: the diff is one changed call, one wrapper and one test.
- Extremes through the new call: `-help-remote` unset, blank (`normalizeRemoteURL` trims to empty), equal to `-remote`, and different from it. The first three fall back to `NodeRemote` at `main.go:236` and give the base policy byte for byte; only the fourth changes `connect-src`, which is the fix. A trailing slash or a `;` in the flag value reach the policy exactly as they did through `-remote` before, operator input on both sides of the diff.
- Browser consumers of `NodeRemote`: none. The one cross-origin fetch, `controller-action-function.ts:233`, reads the `RemoteHelp` rendered at `views/action.html:163`; `controller-copy.ts` refuses cross-origin URLs and the search bar fetches a same-origin path.
- Catalog: every class is a realm audit pattern (caller identity, the `current-guard` rule); the diff touches no Gno realm code, so no class applies.
- Siblings of the edited test: `TestSecureHeadersMiddlewareStrict` and `TestSecureHeadersMiddlewareNonStrict` call `SecureHeadersMiddleware` with a literal remote, the same shape as the new test, and none of the three reaches `setupWeb`, which is candidate 1. `TestSetupWeb` reaches `setupWeb` and discards the handler. The `-help-remote` fallback has no CSP-level test on either side of the diff.
- Pre-existing defects: none found, so no issue draft.
