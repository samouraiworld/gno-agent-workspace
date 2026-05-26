# PR #5380: feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg

URL: https://github.com/gnolang/gno/pull/5380
Author: davd-gzl | Base: master | Files: 10 | +638 -7
Reviewed by: davd-gzl | Model: claude-opus-4-7
Note: self-authored PR — flagged for transparency; findings stand on their own.

**Verdict: REQUEST CHANGES** — two real correctness issues: `QueryLatestVersion` over-counts versions from nested sibling packages under `/vN/.../vM`, and the CLI conflates "no versions" with "unknown ABCI endpoint", silently blocking deploys of `vN` (N>5) against any pre-PR node unless `--force`.

## Summary

The PR adds a tooling-only soft-warning system for non-sequential `/vN` package deployments (chain stays permissive, per ADR's IBC-ordering argument). Three pieces: `ParseVersionSuffix` helper, new `vm/qlatestversion` ABCI query returning `{latest, first_missing, missing}`, and `gnokey maketx addpkg --force` that warns on missing predecessor and blocks when the gap from latest exceeds 5. Two bugs surfaced under empirical testing — the keeper iterates a prefix that pulls in deeper-nested packages without re-anchoring the base, and the CLI treats `qres.Response.Error != nil` identically whether the keeper said "no versions" or the node said "unknown endpoint".

## Glossary

- `ParseVersionSuffix(p)` — extracts `(basePath, N, ok)` from path ending in `/vN`; new helper in `gnovm/pkg/gnolang/mempackage.go`.
- `FindPathsByPrefix(prefix)` — store iterator that yields any registered package path beginning with `prefix` (no segment-boundary check).
- `qres.Response.Error` — ABCI response error field; set both by keeper-returned errors (e.g. "no versions found") AND by `handler.Query` for unknown endpoints. Single channel for two semantic categories.

## Fix

Before: chain accepted any `/vN` deployment with no feedback about gaps. After: chain still accepts any deployment, but `addpkg` queries `vm/qlatestversion` and emits a stderr warning when `version - latestOnChain > 1`, hard-erroring when `gap > 5` unless `--force`. Implementation at [`gno.land/pkg/keyscli/addpkg.go:179-257`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L179-L257), keeper logic at [`gno.land/pkg/sdk/vm/keeper.go:1233-1273`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/keeper.go#L1233-L1273), parser at [`gnovm/pkg/gnolang/mempackage.go:1184-1210`](../../../../../.worktrees/gno-review-5380/gnovm/pkg/gnolang/mempackage.go#L1184-L1210).

## Critical (must fix)

- **[nested path collision]** [`gno.land/pkg/sdk/vm/keeper.go:1240`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/keeper.go#L1240) — `QueryLatestVersion` counts any deeper `/vN` segment as a version of the queried base.
  <details><summary>details</summary>

  The loop iterates every path with prefix `basePath + "/v"` and feeds each into `ParseVersionSuffix`, which extracts the last `/vN` segment regardless of how deep it sits. Any sibling package whose own path happens to end in `/vN` is recorded against the queried base.

  **Repro:** deploy `gno.land/p/demo/avl/v1` and `gno.land/p/demo/avl/v1/sub/v3`. Query `gno.land/p/demo/avl`. Expected `latest:v1, missing:0`. Observed `latest:v3, first_missing:v0, missing:2`. Confirmed via temporary test in `gno.land/pkg/sdk/vm/`:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5380 -R gnolang/gno
  cat > gno.land/pkg/sdk/vm/_collision_test.go <<'EOF'
  package vm

  import (
      "fmt"
      "testing"

      "github.com/gnolang/gno/gnovm/pkg/gnolang"
      abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
      "github.com/gnolang/gno/tm2/pkg/crypto"
      "github.com/gnolang/gno/tm2/pkg/std"
      "github.com/stretchr/testify/assert"
  )

  func TestQLatestVersion_NestedCollision(t *testing.T) {
      env := setupTestEnv()
      ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
      addr := crypto.AddressFromPreimage([]byte("addr1"))
      acc := env.acck.NewAccountWithAddress(ctx, addr)
      env.acck.SetAccount(ctx, acc)
      env.bankk.SetCoins(ctx, addr, std.MustParseCoins("10000000ugnot"))
      for _, p := range []string{"gno.land/p/demo/avl/v1", "gno.land/p/demo/avl/v1/sub/v3"} {
          pname := "avl"
          if p != "gno.land/p/demo/avl/v1" { pname = "sub" }
          files := []*std.MemFile{
              {Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(p)},
              {Name: "x.gno", Body: fmt.Sprintf("package %s\nfunc F() string { return %q }\n", pname, p)},
          }
          assert.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, p, files)))
      }
      env.vmk.CommitGnoTransactionStore(ctx)
      res := env.vmh.Query(env.ctx, abci.RequestQuery{Path: "vm/qlatestversion", Data: []byte("gno.land/p/demo/avl")})
      t.Logf("response: %s", string(res.Data))
      // Bug: prints {"latest":"v3","first_missing":"v0","missing":2}; should print {"latest":"v1","missing":0}.
  }
  EOF
  mv gno.land/pkg/sdk/vm/_collision_test.go gno.land/pkg/sdk/vm/collision_test.go
  go test -v -run TestQLatestVersion_NestedCollision ./gno.land/pkg/sdk/vm/
  rm gno.land/pkg/sdk/vm/collision_test.go
  ```

  Impact: every consumer of the query (the CLI added in this same PR, and the planned gnoweb banner) reads phantom versions. A user deploying `avl/v2` after a sub-deployment ends in `/v3` would be told "small_gap_warns" or blocked. The `r/gov/dao/v3` example in #5365 motivates this PR exactly — the moment that realm's tree has any sub-package ending in `/vN`, the query becomes unreliable.

  Fix: skip paths whose extracted base doesn't equal the queried base. One line:
  ```go
  for p := range store.FindPathsByPrefix(prefix) {
      base, v, ok := gno.ParseVersionSuffix(p)
      if !ok || base != basePath {
          continue
      }
      ...
  }
  ```
  Add a regression test mirroring the repro above.
  </details>

- **[old-node breaks]** [`gno.land/pkg/keyscli/addpkg.go:200-202`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L200-L202) — any ABCI response error is treated as "no versions on chain", blocking `vN` (N>5) deploys against pre-PR nodes.
  <details><summary>details</summary>

  `qres.Response.Error != nil` is set both when the keeper returns "no versions found for X" (legitimate empty state) and when the handler returns `std.ErrUnknownRequest("unknown vm query endpoint qlatestversion ...")` ([handler.go:110-113](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/handler.go#L110-L113)) — i.e. on any node built before this PR. Both go through the same `evalVersionGap(basePath, version, nil, ...)` path. With `result == nil`, `latestVersion = -1`, so `gap = version + 1`. Deploying `v6` against an old node yields `gap = 7 > maxVersionGap(5)` → hard error "version gap too large", refusing to broadcast.

  **Repro:** the unit-testable half:
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5380 -R gnolang/gno
  cat > gno.land/pkg/keyscli/_oldnode_test.go <<'EOF'
  package keyscli

  import (
      "bytes"
      "io"
      "strings"
      "testing"

      "github.com/gnolang/gno/tm2/pkg/commands"
  )

  type nopCloserT struct{ io.Writer }
  func (nopCloserT) Close() error { return nil }

  func TestEvalVersionGap_OldNodeBlocks(t *testing.T) {
      var errBuf bytes.Buffer
      cio := commands.NewDefaultIO()
      cio.SetErr(nopCloserT{&errBuf})
      cio.SetOut(nopCloserT{io.Discard})
      err := evalVersionGap("gno.land/p/foo/bar", 6, nil, false, cio)
      if err == nil || !strings.Contains(err.Error(), "version gap too large") {
          t.Fatalf("expected version-gap-too-large error, got: %v", err)
      }
      t.Logf("blocked: %v", err)
  }
  EOF
  mv gno.land/pkg/keyscli/_oldnode_test.go gno.land/pkg/keyscli/oldnode_test.go
  go test -v -run TestEvalVersionGap_OldNodeBlocks ./gno.land/pkg/keyscli/
  rm gno.land/pkg/keyscli/oldnode_test.go
  ```
  Confirms the block path. The bridge from `checkVersionGap` to this state is `qres.Response.Error != nil → evalVersionGap(..., nil, ...)`, currently uncovered by tests.

  Impact: a user pointing `gnokey addpkg -pkgpath .../v6 -remote <pre-PR-mainnet-or-testnet>` would be told to use `--force`, even though there's no possible "gap" — the node simply doesn't speak this query. The whole "silently ignore" design of `checkVersionGap` (network errors → skip) is undermined by this one branch that promotes a response error to actionable blocking.

  Fix: when `qres.Response.Error != nil`, distinguish by error code or text. Conservative options, pick one:
  1. On any response error, return nil (mirror the `err != nil` network case). Loses the "no versions, fresh chain" warning but is safe.
  2. Treat `ErrUnknownRequest` (handler-side) differently from keeper errors. Cleanest fix is in the keeper: instead of returning an error on no-versions, return `LatestVersionResult{Latest: ""}` and let the CLI key off empty `Latest`. Then `Response.Error != nil` unambiguously means "real failure, skip".

  Option 2 also removes the awkward `if result == nil` branching in `evalVersionGap`.
  </details>

## Warnings (should fix)

- **[silent strconv failure]** [`gno.land/pkg/keyscli/addpkg.go:228-231`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L228-L231) — bad `result.Latest` swallows all gap logic with no warning.
  <details><summary>details</summary>

  If `result.Latest` ever fails to parse (today only via a broken/forged node response, but tomorrow could be a contract-format change), `evalVersionGap` returns nil — no warning, no block, no error message. The user has no signal that the gap check failed. Either log a debug line on stderr, or treat parse failure as a soft-warning skip with a note.

  Same shape repeats at the JSON unmarshal site ([addpkg.go:205-207](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L205-L207)). The cumulative effect is that a corrupted response path is invisible.

  Fix: minimum, `io.ErrPrintfln("Warning: could not parse version info, skipping gap check (%v)", err)` so the user knows the check ran but produced nothing.
  </details>

- **[query swallows context]** [`gno.land/pkg/keyscli/addpkg.go:195`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L195) — `context.Background()` with no timeout.
  <details><summary>details</summary>

  `cli.ABCIQuery(context.Background(), ...)` has no deadline. If the remote RPC is slow or unresponsive, the CLI hangs before `signAndBroadcast` even starts. A 2-3s timeout (`context.WithTimeout`) keeps the soft-check soft. Especially relevant given the "silently ignore network errors" design — without a timeout, a stalled connection doesn't fall through to the silent path, it just blocks indefinitely.
  </details>

- **[regression to older version]** [`gno.land/pkg/keyscli/addpkg.go:236-239`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L236-L239) — deploying a version older than latest emits no warning.
  <details><summary>details</summary>

  When `gap <= 1` (`version <= latestVersion + 1`), the function returns nil. Sequential bumps (gap=1) are correctly silent, but so are back-fills (gap<=0). A user typo `v3` instead of `v6` against a chain at `v5` would silently produce the chain-side `ErrPkgAlreadyExists` failure with no CLI hint that the path is going backwards.

  The chain catches the duplicate; nothing catches "intent to deploy newer than latest but actually deployed older". Low impact since the chain blocks the duplicate, but the explicit warning is essentially free: `if gap < 0 { io.ErrPrintfln("Warning: deploying v%d but latest on-chain is %s — going backwards", version, result.Latest) }`.
  </details>

- **[zero-padded versions]** [`gnovm/pkg/gnolang/mempackage.go:1199-1208`](../../../../../.worktrees/gno-review-5380/gnovm/pkg/gnolang/mempackage.go#L1199-L1208) — `v01` and `v1` parse to the same integer.
  <details><summary>details</summary>

  `ParseVersionSuffix("/v01")` returns version=1, same as `/v1`. The chain's regex `(?:^|/)([^/]+)(?:/v\d+)?$` would accept both `gno.land/p/x/v1` and `gno.land/p/x/v01` as deployable packages, and they'd both be reported under `deployed[1]`. The query would say "no missing" while v01 is essentially a typo.

  Not a critical issue — duplicate paths fail `ErrPkgAlreadyExists`, and zero-padded versions are an anti-pattern. But worth either rejecting via a leading-zero check (`if len(digits) > 1 && digits[0] == '0' { return "", 0, false }`) or documenting that v01 ≡ v1 for this helper. Same constraint Go's module path uses.
  </details>

- **[no integration test for end-to-end CLI]** [`gno.land/pkg/keyscli/addpkg.go:179`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L179) — `checkVersionGap`'s network branches are untested.
  <details><summary>details</summary>

  `evalVersionGap` is well-tested in isolation, but the path from `checkVersionGap` through `cli.ABCIQuery` to `evalVersionGap` is not exercised. Both critical bugs above live in `checkVersionGap`, not `evalVersionGap`. A `.txtar` test under `gno.land/pkg/integration/testdata/` driving `gnokey maketx addpkg` against a live node would catch both — the nested-collision (deploy v1 + v1/sub/v3, then attempt deploy v2 and verify the warning text), and the response-error-vs-network-error distinction.
  </details>

## Nits

- [`gno.land/pkg/keyscli/addpkg.go:83`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L83) — help text says "> 5" but the constant is named `maxVersionGap` and the check is `gap > maxVersionGap`. Help text should reference the constant or read "max 5" for less drift risk.
- [`docs/users/interact-with-gnokey.md:1441-1467`](../../../../../.worktrees/gno-review-5380/docs/users/interact-with-gnokey.md#L1441-L1467) — docs describe `vm/qlatestversion` but never mention the new `--force` flag or the soft-warning behavior of `addpkg`. Either move the warning behavior into the `gnokey maketx addpkg` section or add a cross-reference.
- [`gno.land/adr/pr5380_soft_version_warnings.md:30`](../../../../../.worktrees/gno-review-5380/gno.land/adr/pr5380_soft_version_warnings.md#L30) — example JSON shows `"first_missing": "v1"` but the actual JSON output uses snake_case keys (correct in the types: `first_missing,omitempty`). Spell out that `first_missing` is omitted when `missing == 0` — currently only the `omitempty` tag conveys that.
- [`gno.land/pkg/sdk/vm/keeper.go:1255`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/keeper.go#L1255) — comment claims `(latest + 1) - deployedCount` is O(deployed). True, but the loop above is O(matches-from-prefix-scan) which after the nested-collision fix should be O(actual versions). Worth a one-line update.
- [`gno.land/pkg/keyscli/addpkg.go:130`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L130) — the early `MustReadMemPackage` panics with the user's pkgpath inside `Sprintf("found an empty package %q", cfg.PkgPath)` — existing behavior, but worth noting the version-gap check happens AFTER this, so a typo'd `/v100` on a missing directory would still print the empty-package panic first.

## Missing Tests

- **[nested collision]** [`gno.land/pkg/sdk/vm/handler_test.go:478`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/handler_test.go#L478) — `TestVmHandlerQuery_LatestVersion` covers sequential, single-gap, large-gap, and no-version cases, but not the nested case where another package's path ends with `/vN` under the queried base. See critical finding #1 for the repro.
- **[unknown endpoint]** [`gno.land/pkg/keyscli/addpkg_test.go:26`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg_test.go#L26) — `TestEvalVersionGap` covers the result-is-nil path but doesn't pin down what produces that nil. A focused test that the "old node" / "real failure" cases don't silently block is missing. See critical finding #2.
- **[regression deployment]** `gno.land/pkg/keyscli/addpkg_test.go:26` — no test for `version < latestVersion` (back-fill). Today returns nil, may or may not be intentional; pin the behavior either way.

## Suggestions

- [`gno.land/pkg/sdk/vm/keeper.go:1233`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/keeper.go#L1233) — consider returning a non-error empty result for the "no versions" case.
  <details><summary>details</summary>

  Currently `if latest < 0 { return nil, fmt.Errorf("no versions found for %s", basePath) }`. Returning `&LatestVersionResult{Latest: "", Missing: 0}, nil` instead would:
  1. Make the "no versions" state distinguishable from "endpoint unknown" on the CLI side (fixes critical #2 cleanly).
  2. Let consumers query "any versions of X" without try/catch semantics.
  3. Match the pattern of other vm queries that return empty data for empty state rather than 404.

  Downside: callers must check for empty `Latest` instead of err. Small price.
  </details>

- [`gno.land/pkg/keyscli/addpkg.go:174`](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L174) — make `maxVersionGap` configurable via a flag.
  <details><summary>details</summary>

  The "5" cutoff is arbitrary. A `--max-version-gap` flag (defaulting to 5) lets power users dial it down (strict CI) or up (intentionally sparse versioning) without needing `--force` every time. Optional; the ADR's "1-5 are intentional gaps" framing suggests 5 is the right default but doesn't argue against configurability.
  </details>

## Questions for Author

- The ADR argues IBC-ordering blocks on-chain enforcement; how does that interact with the planned gnoweb banner — does it pull the same `vm/qlatestversion` and inherit the nested-collision bug visually until #1 above is fixed?
- Was the "blocks on old-node" behavior of critical #2 intentional as a forward-incompatibility signal, or should it fall back to silent skip?
- Any thought to also expose `std.LatestVersion()` as the issue's stretch goal mentions, or is that explicitly out of scope for this PR?
