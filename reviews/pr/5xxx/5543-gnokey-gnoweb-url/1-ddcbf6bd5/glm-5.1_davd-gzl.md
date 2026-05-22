# PR #5543: feat(gnokey): show gnoweb URL after successful addpkg deploy

**URL:** https://github.com/gnolang/gno/pull/5543
**Author:** davd-gzl | **Base:** master | **Files:** 4 | **+175 -0**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

After a successful `gnokey maketx addpkg`, the PR prints two extra lines: `PKG PATH:` (the deployed package path) and `VIEW AT:` (a gnoweb URL to browse it). The gnoweb URL is derived heuristically from the `--remote` RPC address by stripping the `rpc.` hostname prefix and swapping the port to 8888. The URL is only shown if a gnoweb instance is verified reachable at the derived address (HTTP GET + check for `"gno.land"` in first 1KB of response). The feature lives in `GnowebURLFromRemote` in `root.go`, called from the `OnTxSuccess` callback in `addpkg.go`. Includes an ADR and unit tests for the new helper functions.

**Files changed:**
- `gno.land/adr/pr5543_addpkg_post_deploy_url.md` — ADR documenting the decision
- `gno.land/pkg/keyscli/addpkg.go` — Calls `GnowebURLFromRemote` in `OnTxSuccess`
- `gno.land/pkg/keyscli/root.go` — New `GnowebURLFromRemote` and `isGnowebReachable` helpers
- `gno.land/pkg/keyscli/root_test.go` — Unit tests for both helpers

## Test Results

- **Existing tests:** PASS — all unit tests pass (`TestGnowebURLFromRemote`, `TestIsGnowebReachable`)
- **Edge-case tests:** skipped
- **CI:** All checks pass (build, lint, test, e2e, CodeQL, analyze). Merge-requirements bot is pending approval (normal).
- **Codecov:** Patch coverage 74.07% — 7 lines uncovered (URL construction happy path in `root.go`, callback lines in `addpkg.go`)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `root.go:141` — The reachability probe uses whatever scheme the RPC remote has (`u.Scheme`). If `--remote` is `https://rpc.gno.land:443`, the derived gnoweb URL becomes `https://gno.land:8888/...`, which is wrong — production gnoweb is behind a reverse proxy on port 443, not serving HTTPS directly on 8888. The reachability check will fail, so the `VIEW AT:` line is silently omitted in production. This means the feature is effectively dead for the primary production deployment, only working for local dev (port 8888). Consider forcing `http://` for gnoweb regardless of RPC scheme, or handling the HTTPS/443 case by omitting the port (production gnoweb URL would be `https://gno.land/r/...`).
- [ ] `root_test.go` — No positive-path test for `GnowebURLFromRemote`. The only non-empty-remote test case is "unreachable remote returns empty". There should be a test with a `httptest.NewServer` mock that verifies the actual URL construction (e.g., remote=`rpc.gno.land:26657` + pkgPath=`gno.land/r/demo/counter` → `http://gno.land:8888/r/demo/counter`). This is the most important test case and it's missing, which also explains the 74% Codecov patch coverage.

## Nits

- [ ] `addpkg.go:145` — `"PKG PATH:  "` uses 2 trailing spaces + Println's auto-space = 3 spaces before value, while `addpkg.go:147` `"VIEW AT:   "` uses 3 trailing spaces + auto-space = 4 spaces. The alignment is inconsistent with each other and with existing `PrintTxInfo` lines. Consider using consistent padding (e.g., align all values to the same column).
- [ ] `root.go:137` — `u.Host = strings.TrimPrefix(u.Hostname(), "rpc.") + ":8888"` discards any userinfo from the parsed URL (`u.User`). Not a realistic scenario for gno RPC URLs, but the code silently drops parts of the parsed URL. Consider building the URL from scratch instead of mutating the parsed RPC URL, to avoid leaking unrelated URL components (query params, fragment, userinfo).

## Missing Tests

- [ ] Positive-path test for `GnowebURLFromRemote` with a reachable mock gnoweb server — verify correct URL format, scheme, hostname, port, and path stripping (`root_test.go`)
- [ ] Test for remote with existing scheme (e.g., `"http://rpc.gno.land:26657"`, `"https://rpc.gno.land:443"`) — verify scheme handling (`root_test.go`)
- [ ] Test for hostname without `rpc.` prefix (e.g., `"test3.gno.land:26657"`) — verify no stripping occurs (`root_test.go`)
- [ ] Integration/txtar test for the full `addpkg` flow verifying `PKG PATH:` and `VIEW AT:` output lines

## Suggestions

- The `call` subcommand (`call.go:150`) also has a `PkgPath` field and sets `OnTxSuccess`. Consider showing the `VIEW AT:` URL there too, since calling a realm function also benefits from linking to the gnoweb page. This would provide a consistent UX across both deployment-related commands.
- For the reachability check, consider also verifying the response status code (e.g., `resp.StatusCode == http.StatusOK`). A 5xx error response that happens to contain `"gno.land"` in its body would pass the current check.
- The 2-second timeout in `isGnowebReachable` (`root.go:153`) blocks the CLI output. Consider reducing to 1 second — for local dev networks, gnoweb responds in milliseconds. If it doesn't respond in 1 second, it's likely not running.

## Questions for Author

- Is the intent for this feature to work only in local dev setups (port 8888), or should it also handle production (where gnoweb is behind a reverse proxy on 443)? The ADR acknowledges the heuristic limitation, but if the feature is invisible in production, its value is limited.
- Should the `VIEW AT:` URL be shown for `gnokey maketx call` as well? It has the same `PkgPath` available and would benefit from the same UX improvement.

## Verdict

REQUEST CHANGES — Missing positive-path test for `GnowebURLFromRemote` is a significant coverage gap (7 uncovered lines, no test verifying the core URL construction logic). The production-scheme issue should also be addressed or explicitly scoped as "local-dev only" in the ADR. Otherwise the implementation is clean, the reachability check is a smart safeguard, and the ADR is well-written.
