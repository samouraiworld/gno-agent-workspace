# PR #5283: feat(gnoweb): add State Explorer for browsing on-chain realm state

URL: https://github.com/gnolang/gno/pull/5283
Author: jaekwon | Base: master | Files: 63 | +7425 -247
Reviewed by: davd-gzl | Model: claude-opus-4-7 (1m) | Commit: `7c3677c4` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5283 7c3677c4`

**Verdict: REQUEST CHANGES** — feature is well-scoped and tests pass, but `vm/qobject_json` and `vm/qobject_binary` panic on any oversized hex prefix (a public unauthenticated DoS / log-spam vector on every node, reachable from `$state&oid=...` in gnoweb).

## Summary

Adds a State Explorer tab to gnoweb that browses persisted realm state as a lazy-loaded tree. Three new public ABCI endpoints (`vm/qpkg_json`, `vm/qobject_json`, `vm/qobject_binary`, `vm/qtype_json`), a refactor of `qeval`'s JSON output to standard Amino JSON via a new `ExportValues`/`ExportObject` pipeline (`gnovm/pkg/gnolang/values_export.go`, +387 lines), and a fresh `@gnojs/amino` TypeScript decoder library (+~1500 lines). The new endpoints share a single user-supplied entrypoint (`ObjectID.UnmarshalAmino`) whose `hex.Decode(Hashlet[:20], parts[0])` panics on any input where `len(parts[0]) >= 42`; nothing on the query path recovers, so every malformed long OID becomes a runtime stack-trace.

## Glossary

- **ExportValues / ExportObject** — new defensive-copy walkers that replace persisted objects with `RefValue` and break ephemeral cycles with `ExportRefValue{":N"}`; the JSON path for VM queries.
- **ObjectID** — `PkgID:NewTime` string; PkgID is `[20]byte` (40 hex chars max).
- **qobject_json / qobject_binary** — new ABCI endpoints returning Amino JSON / binary for a single persisted object by ObjectID.
- **qpkg_json / qtype_json** — new ABCI endpoints returning a package's named block variables / a type definition.

## Fix

Currently `qobject_json` and `qobject_binary` forward `req.Data` directly to `(*ObjectID).UnmarshalAmino` ([`gno.land/pkg/sdk/vm/keeper.go:1230`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/keeper.go#L1230) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/keeper.go#L1230)). `UnmarshalAmino` splits on `:` and calls `hex.Decode(oid.PkgID.Hashlet[:], []byte(parts[0]))` ([`gnovm/pkg/gnolang/ownership.go:57`](https://github.com/gnolang/gno/blob/7c3677c4/gnovm/pkg/gnolang/ownership.go#L57) · [↗](../../../../../.worktrees/gno-review-5283/gnovm/pkg/gnolang/ownership.go#L57)). `Hashlet` is a fixed `[20]byte` ([`gnovm/pkg/gnolang/hash_image.go:28-30`](https://github.com/gnolang/gno/blob/7c3677c4/gnovm/pkg/gnolang/hash_image.go#L28-L30) · [↗](../../../../../.worktrees/gno-review-5283/gnovm/pkg/gnolang/hash_image.go#L28-L30)), so any `parts[0]` whose decoded length exceeds 20 (i.e. `>= 42` hex chars, even) overflows the destination and stdlib panics with `runtime error: index out of range [20] with length 20`. The handler in `gno.land/pkg/sdk/vm/handler.go` does not `recover` for these paths (only `queryType` does), and `baseapp.handleQueryCustom` ([`tm2/pkg/sdk/baseapp.go:475-515`](https://github.com/gnolang/gno/blob/7c3677c4/tm2/pkg/sdk/baseapp.go#L475-L515) · [↗](../../../../../.worktrees/gno-review-5283/tm2/pkg/sdk/baseapp.go#L475-L515)) has no recovery — only the outermost RPC handler does. Net effect: the node stays alive, but every oversized OID returns a generic 500 plus a stack trace in logs, easily abusable as log-spam / CPU-cost DoS through gnoweb's `$state&oid=...` URL which is unauthenticated.

Fix shape: validate `len(parts[0]) <= 2*HashSize` in `UnmarshalAmino` (or guard at the keeper before calling it). One-liner: `if hex.DecodedLen(len(parts[0])) > HashSize { return errors.New("invalid ObjectID") }`. Add the existing handler-side `defer recover()` (mirroring `queryType`) as defense in depth.

## Benchmarks / Numbers

None.

## Critical (must fix)

- **[unauth panic / DoS]** [`gno.land/pkg/sdk/vm/keeper.go:1230`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/keeper.go#L1230) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/keeper.go#L1230) — `qobject_json`/`qobject_binary` panic on any OID with >40 hex chars before `:`.
  <details><summary>details</summary>

  **Shape:** user supplies `aaaa…aaa:1` (42+ hex), `UnmarshalAmino` runs `hex.Decode(Hashlet[:20], parts[0])`, stdlib panics on out-of-range write, panic propagates through `vmHandler.queryObjectJSON` → `baseapp.handleQueryCustom` (no recover) up to the RPC server.

  **Mechanism:** the new ABCI endpoints inherited a parser written when callers were trusted ABCI internals; now they sit behind a public web endpoint at `gno.land/<realm>$state&oid=<user-supplied>&json` ([`gno.land/pkg/gnoweb/handler_http.go:744`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/gnoweb/handler_http.go#L744) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/gnoweb/handler_http.go#L744)) with zero pre-validation. Reproducible with the adversarial test below — both `qobject_json` and `qobject_binary` fail.

  **Repro:** test file at [`reviews/pr/5xxx/5283-state-explorer-gnoweb/1-7c3677c4/tests/qobject_oid_panic_test.go`](tests/qobject_oid_panic_test.go); copy into `gno.land/pkg/sdk/vm/` and `go test -run TestVmHandlerQuery_Object._LongHexPanic`. Both tests fail on current HEAD with `runtime error: index out of range [20] with length 20`.

  **Fix:** add length pre-check inside `(*ObjectID).UnmarshalAmino` (`if hex.DecodedLen(len(parts[0])) != HashSize { return errors.New("invalid ObjectID hex length") }`) and add `defer recover()` in `queryObjectJSON`/`queryObjectBinary` mirroring the pattern already used in `queryType` ([`gno.land/pkg/sdk/vm/handler.go:322-330`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/handler.go#L322-L330) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/handler.go#L322-L330)).
  </details>

## Warnings (should fix)

- **[stale comment]** [`gno.land/pkg/sdk/vm/convert.go:280`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/convert.go#L280) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/convert.go#L280) — `tryGetError` refers to a `doRecoverQuery` symbol that doesn't exist.
  <details><summary>details</summary>

  The comment in `tryGetError` justifies the OOG re-panic by referencing "doRecoverQuery returns error" but no such symbol exists in the tree. Misleading for future maintainers. Either remove the parenthetical or point at the actual recover location (`baseapp.runTx` only recovers in tx mode, not query mode — so the comment is also factually wrong for the query path).
  </details>

- **[orphan switch case]** [`contribs/gnodev/pkg/proxy/path_interceptor.go:317`](https://github.com/gnolang/gno/blob/7c3677c4/contribs/gnodev/pkg/proxy/path_interceptor.go#L317) · [↗](../../../../../.worktrees/gno-review-5283/contribs/gnodev/pkg/proxy/path_interceptor.go#L317) — `"vm/qobject"` case kept after the endpoint was renamed to `vm/qobject_json`.
  <details><summary>details</summary>

  `handleQuery` switches on `"vm/qobject", "vm/qobject_json", "vm/qtype_json"`. Commits `2dca26d` and `23c1551` renamed the endpoint; the bare `vm/qobject` path no longer exists anywhere else. Harmless but invites confusion — the case should be dropped, or a comment added that this is forward-compat tolerance for older clients (probably unnecessary since these endpoints are brand new in this PR).
  </details>

- **[no allocator cap]** [`gno.land/pkg/sdk/vm/keeper.go:1268-1272`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/keeper.go#L1268-L1272) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/keeper.go#L1268-L1272) — `QueryPkg`/`QueryType`/`exportObject` set a gas meter but no `gno.NewAllocator(maxAllocQuery)`.
  <details><summary>details</summary>

  `queryEvalInternal` does both (gas + alloc, [`keeper.go:1117-1119`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/keeper.go#L1117-L1119) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/keeper.go#L1117-L1119)). The new query methods only set gas. `ExportValues` walks the value tree and defensively copies every field/element ([`values_export.go:165-274`](https://github.com/gnolang/gno/blob/7c3677c4/gnovm/pkg/gnolang/values_export.go#L165-L274) · [↗](../../../../../.worktrees/gno-review-5283/gnovm/pkg/gnolang/values_export.go#L165-L274)) — pure Go allocations, not counted by the gas meter. A package with a deeply nested struct or a slice-of-struct-of-slice could blow memory before gas runs out. Cap with the same allocator the eval path uses, or document why these paths are exempt.
  </details>

- **[trusts client-side OID escape]** [`gno.land/pkg/sdk/vm/keeper.go:1254`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/keeper.go#L1254) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/keeper.go#L1254) — `fmt.Sprintf("{\"objectid\":%q,...}", oidStr, ...)` uses Go `%q`, not JSON-safe escape.
  <details><summary>details</summary>

  Go's `%q` produces a Go-quoted string, not strictly JSON. For ASCII identifiers (which is all valid ObjectIDs after `UnmarshalAmino` passes), the output coincides with JSON, so today this is safe. But the safety depends on the invariant "validated OID contains only `[a-f0-9:]` + digits" — once that invariant is broken (e.g. the panic fix above lets through a different shape, or someone changes `UnmarshalAmino`), the JSON envelope could emit `\xNN` escapes that JSON.parse rejects. Use `json.Marshal(oidStr)` or `amino.MarshalJSON` instead; the cost is zero.
  </details>

- **[file/start/end query unauthenticated]** [`gno.land/pkg/gnoweb/handler_http.go:704-738`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/gnoweb/handler_http.go#L704-L738) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/gnoweb/handler_http.go#L704-L738) — `ServeStateJSON` exposes a new `$state&file=...&start=N&end=N&json` route that returns syntax-highlighted source for any line range.
  <details><summary>details</summary>

  The file lookup goes through `h.Client.File(ctx, gnourl.Path, fileName)`, which itself validates the package path. So directory traversal is bounded. But the route is unauthenticated and unrate-limited; bots can scrape entire source trees through it where they previously had to hit `$source`. Probably acceptable since the same content is reachable via `$source`, but worth confirming that the gnoweb codeowners are OK adding another scraping vector. No fix required if intentional.
  </details>

## Nits

- [`gno.land/pkg/sdk/vm/keeper.go:1355`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/keeper.go#L1355) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/keeper.go#L1355) — `const maxTypeDepth = 8` is below the explorer's lazy-fetch behavior and may silently truncate deeply-nested generic-like declarations. Tune up to 16, or document.
- [`gno.land/pkg/sdk/vm/keeper.go:1413`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/keeper.go#L1413) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/keeper.go#L1413) — `marshalTypeJSON` default case emits `RefType` for "unknown" types; no fallback for `blockType`/`tupleType`/`heapItemType` etc. The TS decoder may receive a RefType where it expects a structural type. Add explicit cases or a TODO.
- [`misc/gnojs/src/decode.ts:378-380`](https://github.com/gnolang/gno/blob/7c3677c4/misc/gnojs/src/decode.ts) · [↗](../../../../../.worktrees/gno-review-5283/misc/gnojs/src/decode.ts) — `if (!typeId.includes("/"))` skips stdlib types by heuristic. Brittle: any user type named `time` in a pkg path that happens to lack `/` (impossible today, but) would also skip. Use an explicit allowlist or `startsWith("gno.land/")`.
- [`gno.land/pkg/gnoweb/components/view_state.go:22`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/gnoweb/components/view_state.go#L22) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/gnoweb/components/view_state.go#L22) — `template.JS(data.NodesJSON)` is safe today (amino → `json.Marshal` HTML-escapes `<` → `<`). Add a unit test that asserts a string-value containing `</script>` in package state cannot break out of the `<script type="application/json">` block.
- [`examples/gno.land/r/demo/closuretest/closuretest.gno`](https://github.com/gnolang/gno/blob/7c3677c4/examples/gno.land/r/demo/closuretest/closuretest.gno) · [↗](../../../../../.worktrees/gno-review-5283/examples/gno.land/r/demo/closuretest/closuretest.gno) — demo realm is fine as a test fixture, but consider moving under `examples/gno.land/r/demo/tests/` or similar to signal it's not a user-facing showcase. Otherwise it pollutes the realm index.

## Missing Tests

- **[panic coverage gap]** [`gno.land/pkg/sdk/vm/handler_test.go:438`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/handler_test.go#L438) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/handler_test.go#L438) — `TestVmHandlerQuery_ObjectJSON` only tests `invalid` and a well-formed-but-missing OID. Add the oversized-hex case (would have caught the Critical above).
- **[binary success path silence]** [`gno.land/pkg/sdk/vm/handler_test.go:545-555`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/handler_test.go#L545-L555) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/handler_test.go#L545-L555) — `TestVmHandlerQuery_ObjectBinary_Success` doesn't decode the returned bytes via `amino.UnmarshalAny` to assert the value shape — it just asserts length > 0. A regression that swaps the encoding (e.g. forgets the type prefix) would pass.
- **[ExportRefValue cycle determinism]** [`gnovm/pkg/gnolang/values_export_test.go`](https://github.com/gnolang/gno/blob/7c3677c4/gnovm/pkg/gnolang/values_export_test.go) · [↗](../../../../../.worktrees/gno-review-5283/gnovm/pkg/gnolang/values_export_test.go) — no test asserts that `:1`, `:2`, … assignment is stable across re-exports (map iteration order in `seen` is fine since it's `map[Object]int` indexed by pointer, but the assignment order depends on tree-walk order — confirm with a struct containing two pointers to the same ephemeral object).

## Suggestions

- [`gno.land/pkg/sdk/vm/handler.go:325-329`](https://github.com/gnolang/gno/blob/7c3677c4/gno.land/pkg/sdk/vm/handler.go#L325-L329) · [↗](../../../../../.worktrees/gno-review-5283/gno.land/pkg/sdk/vm/handler.go#L325-L329) — promote the `defer recover()` pattern into a small `safeQuery(name string, fn func() (string, error))` helper used by all four new endpoints, so the recover is structural rather than per-endpoint vigilance.
- [`gnovm/pkg/gnolang/ownership.go:52-67`](https://github.com/gnolang/gno/blob/7c3677c4/gnovm/pkg/gnolang/ownership.go#L52-L67) · [↗](../../../../../.worktrees/gno-review-5283/gnovm/pkg/gnolang/ownership.go#L52-L67) — once `UnmarshalAmino` validates length, also reject non-hex characters explicitly with a typed `ErrInvalidObjectID` so handler-level callers can branch on it; the current `errors.New` is opaque.
- The proto registration of `ExportRefValue` in [`gnovm/pkg/gnolang/package.go:33`](https://github.com/gnolang/gno/blob/7c3677c4/gnovm/pkg/gnolang/package.go#L33) · [↗](../../../../../.worktrees/gno-review-5283/gnovm/pkg/gnolang/package.go#L33) is mixed in with on-chain persisted types. Add a comment that `ExportRefValue` is export-only and must never reach the store (the `DeepFill` no-op in [`values_fill.go:75-77`](https://github.com/gnolang/gno/blob/7c3677c4/gnovm/pkg/gnolang/values_fill.go#L75-L77) · [↗](../../../../../.worktrees/gno-review-5283/gnovm/pkg/gnolang/values_fill.go#L75-L77) hints at this but isn't loud).
- ADR-002 says "Replace the custom JSON types" — confirm in the body of the PR whether any tooling outside this tree depends on the old `JSONField`/`JSONStructValue` format. If yes, this is a breaking change for them.

## Questions for Author

- Is the gnoweb State Explorer expected to ship in the same release as the new ABCI endpoints, or do you plan to land the VM-side endpoints first (the PR body says "Depends on #5274" — what's the merge order)?
- The PR adds three Co-Authored-By Claude trailers per commit; per project AGENTS.md the convention is `Assisted-By` (not `Co-Authored-By`). Worth a squash-rewrite before merge?
- `maxTypeDepth = 8` in `marshalTypeJSON`: was this chosen empirically (stack overflow on `time.Time` cited as motivator) or arbitrarily? If empirically, consider documenting the test case.
- Should the new endpoints be gated behind a config flag (similar to other dev-oriented endpoints) until the response formats stabilize? Once mainnet clients depend on the Amino JSON wire format, changing it is a chain-wide migration.
