# PR #5543: feat(gnokey): show gnoweb URL after successful addpkg deploy

**URL:** https://github.com/gnolang/gno/pull/5543
**Author:** davd-gzl | **Base:** master | **Files:** 17 | **+794 -2**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Round-2 review at commit `3a6019978`. PR was substantially refactored since round 1
(`ddcbf6bd5`): the heuristic gnoweb URL derivation (strip `rpc.` prefix + swap port to
`8888` + HTTP probe) is gone, replaced with a registry-backed lookup. The PR now
also bundles the dependency #5596 (canonical networks registry + `/api/networks`
endpoint + ADR + nightly live test + CI workflow), so the surface area is much
larger than round 1.

**Behavior:** after a successful `gnokey maketx addpkg`, two extra lines are printed:

```
PKG PATH:   gno.land/r/demo/counter
VIEW AT:    https://gno.land/r/demo/counter
```

The base URL for `VIEW AT` is resolved in priority order:
1. `GNO_GNOWEB_URL` env var (private/custom networks).
2. `networks.json` entry matched by `--chainid` (must have non-empty `gnoweb_url`).
3. Special chain ID `"dev"` → `http://127.0.0.1:8888` (gnodev default).

If none match, the `VIEW AT` line is silently omitted.

**New artifacts in this PR:**
- `gno.land/pkg/networks/{networks.go,networks.json,networks_test.go,networks_live_test.go}` — registry pkg (Load/Raw, embed JSON, opt-in live probe gated by `GNO_NETWORKS_LIVE=1`).
- `gno.land/pkg/gnoweb/{networks.go,networks_test.go}` — `/api/networks` HTTP handler with ETag + `Cache-Control: public, max-age=3600` + `Access-Control-Allow-Origin: *`.
- `gno.land/pkg/gnoweb/app.go` — wires `/api/networks` into `NewRouter`.
- `gno.land/pkg/gnoweb/app_test.go` — adds `/api/networks` route assertion.
- `gno.land/pkg/keyscli/{root.go,root_test.go}` — `GnowebURLForPkg` + helpers + 12 table-driven cases.
- `gno.land/pkg/keyscli/addpkg.go` — calls `GnowebURLForPkg` in `OnTxSuccess`.
- `.github/workflows/ci-networks-live.yml` — nightly cron + path-scoped PR trigger; auto-files an issue on scheduled failure.
- `gno.land/Makefile` — `_test.networks.live` target, opt-in via `GNO_NETWORKS_LIVE=1`.
- `gno.land/adr/pr5543_addpkg_post_deploy_url.md`, `gno.land/adr/pr5596_networks_registry.md` — two ADRs.
- `docs/resources/gnoland-networks.md`, `docs/users/interact-with-gnokey.md` — table refresh + `VIEW AT` doc + resolution-order block.

## Test Results

- **Existing tests:** PASS
  - `go test ./gno.land/pkg/keyscli/ ./gno.land/pkg/networks/ ./gno.land/pkg/gnoweb/` (all relevant subtests pass).
  - `TestRoutes/test_route_/api/networks` PASS — `/api/networks` returns `gnoland1` substring.
  - `TestGnowebURLForPkg` PASS — 12 cases.
  - `TestHandlerNetworksJSON` PASS — content type, cache header, ETag, 304 conditional GET.
- **Live test:** `GNO_NETWORKS_LIVE=1 go test ./gno.land/pkg/networks/ -run TestActiveNetworksReachable` PASS — all 4 active networks (`gnoland1`, `staging`, `test11`, `test12`) reachable, chain IDs match.
- **CI:** Most non-gating jobs still `pending` at this commit; merge-bot fails on the gnoweb codeowner rule (alexiscolin/gfanton review required) which is not a code issue.

## Round-1 follow-up

Findings from `1-ddcbf6bd5/glm-5.1_davd-gzl.md`:
- Warning "HTTPS scheme issue" (heuristic produced `https://gno.land:8888/...` for prod) — **resolved.** Registry now provides authoritative `gnoweb_url`s.
- Warning "no positive-path test for `GnowebURLFromRemote`" — **resolved.** `TestGnowebURLForPkg` covers env override, registry hit, dev chain, unknown chain, anchored-prefix safety, etc.
- Nit "PKG PATH/VIEW AT alignment" — **non-issue on re-check.** Both prefixes are 11 chars wide, matching the `PrintTxInfo` column convention; alignment is consistent.
- Nit "userinfo dropped by `u.Host = ...`" — **resolved.** No URL parsing/mutation; we now build the URL from a pre-validated base.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gno.land/pkg/keyscli/root.go:168-172` — `gnowebBaseFor` does not consider `Network.Status`. A network entry with `status == "offline"` still resolves to its `gnoweb_url`, so a deploy against an offline-flagged network would print a `VIEW AT:` URL pointing at a dead host. Either filter by `Status == StatusActive` (and arguably also accept `StatusDeprecated`, since deprecated networks may still be browsable), or document that the registry should remove `gnoweb_url` for offline entries. Right now the `Status` field has no behavioral effect on URL resolution.
- [ ] `gno.land/pkg/keyscli/root.go:168-172` — the `for _, n := range reg.Networks` loop continues iterating after a `chainID` match if `n.GnowebURL == ""`. Since `TestLoad` enforces unique chain IDs, this is effectively dead — but the semantics are ambiguous. Replace with an explicit lookup that breaks on the first chain-ID match (and only returns the URL if non-empty); makes intent clear and avoids the impression we'd fall through to a "next" entry with the same chain ID.

## Nits

- [ ] `gno.land/pkg/keyscli/root_test.go:78-82` — case `pkgPath equal to gno.land` expects `https://gno.land/` (trailing slash). After stripping, `rel = ""`, and the `!HasPrefix("/")` branch prepends `/`. That trailing slash is a side effect of the prefix logic. Arguably the result for "the domain itself" should be `https://gno.land` (no slash) since gnoweb's `/` is the home alias. Behavior is harmless, but the test enshrines a slightly odd shape.
- [ ] `gno.land/pkg/networks/networks.go:48` — `Load()` re-parses `networks.json` on every invocation. Each `gnokey maketx addpkg` call only triggers it once, but the package is intended for broader reuse (CLI wizards, gnodev). Wrapping it in `sync.Once` (or computing the typed `Registry` once at init alongside `rawJSON`) avoids redundant unmarshals.
- [ ] `gno.land/pkg/gnoweb/networks.go:17` — `body := networks.Raw()` shares the embedded slice's backing array. The doc on `Raw()` says callers must not mutate, and this handler only writes — but if a future `WithExtraHeader`-style middleware ever mutated the body, it would corrupt the registry for the rest of the process. A defensive `bytes.Clone` once at handler-init cost is essentially zero.
- [ ] `gno.land/pkg/networks/networks_live_test.go:32-47` — the live test only verifies `chain_id` from `/status`. A typo'd `gnoweb_url` (e.g. `https://stating.gno.land`) would not be caught. Adding a HEAD/GET against each `gnoweb_url` would close the loop.
- [ ] `gno.land/pkg/networks/networks.json:1-33` — no schema version. Future incompatible additions can't be signaled to old consumers. Adding `"version": 1` (and bumping on breaking changes) is a cheap forward-compat hedge. Out of scope; mention only.
- [ ] `gno.land/pkg/gnoweb/app.go:25` — `Cache-Control: public, max-age=3600` means downstream consumers see stale registry data for up to an hour after a testnet rotation. ETag enables revalidation but only when the client opts in with `If-None-Match`. ADR notes the trade-off; acceptable, but worth flagging given testnet rotations are exactly the events you'd want propagated quickly.
- [ ] `gno.land/adr/pr5543_addpkg_post_deploy_url.md:39` — references "logic lives in `GnowebURLForPkg` … in `keyscli/root.go`" — accurate, but it could also mention `gnowebBaseFor` and `joinPkgPath` as the unexported helpers, so future readers don't grep for the old `GnowebURLFromRemote`.

## Missing Tests

- [ ] Integration/txtar test for the full `gnokey maketx addpkg` flow asserting `PKG PATH:` and `VIEW AT:` output lines. Unit coverage is good, but no end-to-end test exercises the wired-up call site (`addpkg.go:146-148`).
- [ ] `TestGnowebURLForPkg` does not cover a registered chain whose `gnoweb_url` is empty. Adding a synthetic registry case (or asserting that "Staging" with `gnoweb_url` populated is the only positive registry path today) would lock in the empty-URL fall-through behavior.
- [ ] No test asserts that `Network.Status` does **not** affect URL resolution (or, after the warning above, that it does). Pick a contract and pin it.
- [ ] No test for `/api/networks` `Vary` / 304 response with a different (mismatched) ETag value — confirms 200 + body is returned. Existing 304 path is tested; the negative path is not.

## Suggestions

- Filter `gnowebBaseFor` by `Status` (at minimum reject `StatusOffline`) so offline-flagged registry entries don't yield dead URLs. Document the chosen contract in the ADR.
- Cache `Load()` once via `sync.Once` (or compute a typed registry at init time alongside `rawJSON`). Keeps Raw() zero-copy while making typed access cheap.
- Same UX in `gnokey maketx call`: it has `PkgPath` available and would benefit from the same `VIEW AT:` line. (Carried over from round 1; still applies.)
- `dev` chain ID is hardcoded inside `gnowebBaseFor`. Consider expressing it as an entry in `networks.json` with `status: "local"` (new status) or omitting it from the file but documenting it as a reserved sentinel. The hardcoded constant is fine for now, just a follow-up consideration.
- `ci-networks-live.yml` triggers on `gno.land/Makefile` changes; that's broader than needed (any Makefile edit triggers the full live probe). Tightening to `gno.land/pkg/networks/**` only (since the make target wraps the same `go test`) would avoid spurious runs.

## Questions for Author

- Should deprecated or offline networks still print a `VIEW AT:` URL? Right now they would, because `Status` has no effect on resolution. Is that intentional?
- Why is `dev` hardcoded as a const rather than living in `networks.json`? A `"local"`/`"dev"` status would let us keep one source of truth.
- The ADR mentions the env var is "highest precedence" — is there a use case where a chainID match should override env (e.g. when the user explicitly wants the canonical URL)? Today env always wins.

## Verdict

APPROVE — round-1 critical and warning items are all addressed. The registry-backed
approach is the right design, tests cover the substantive paths, the live probe +
nightly CI catch staleness, and ADRs document the decisions. Remaining items are
warnings/nits/follow-ups (status-aware resolution, `Load()` caching, missing
integration test, CI trigger scope), none of which block merge. Note that the bot
still gates on gnoweb codeowner approval (alexiscolin / gfanton).
