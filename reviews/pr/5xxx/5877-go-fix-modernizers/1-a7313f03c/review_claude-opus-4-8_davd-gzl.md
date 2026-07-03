# PR [#5877](https://github.com/gnolang/gno/pull/5877): chore: apply `go fix` modernizers and enforce them in CI

URL: https://github.com/gnolang/gno/pull/5877
Author: thehowl | Base: master | Files: 98 | +1366 -1618
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: a7313f03c (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5877 a7313f03c`

**TL;DR:** Runs Go 1.26's `go fix` modernizers across the whole repo (range-over-int, `reflect.TypeFor`, `reflect.Pointer`, `any`, `strings.Builder`, `min`/`max`, `slices`/`maps` helpers, `wg.Go`), then adds a CI step and a `make fix` target so the tree stays modernized. Every rewrite is meant to be behavior-preserving; the review's job is to confirm none slipped a real semantics change past the mechanical sweep.

**Verdict: APPROVE** — behavior-preserving throughout; the one real risk (the `omitzero` fixer stripping Amino `,omitempty`) is caught, reverted, and disabled with a documented carve-out. No blocking findings.

## Summary
A pure modernization sweep: ~880 automated rewrites gated by each module's `go.mod` version, plus one hand-completed fix where `reflect.TypeFor` orphaned a variable. The load-bearing risk is the `omitzero` modernizer, which strips `json:",omitempty"` from struct-typed fields assuming `encoding/json` semantics; [Amino](https://github.com/gnolang/gno/blob/a7313f03c/tm2/pkg/amino/README.md) honors that tag on struct fields, so the strip silently changes JSON output for [RefValue](https://github.com/gnolang/gno/blob/a7313f03c/gnovm/pkg/gnolang/values.go) (realm state), `GenesisDoc`, and `ResultTx.Proof` (RPC). The PR restores the tags and disables that one modernizer (`-omitzero=false`) in both the CI check and `make fix`, with an ADR. Enforcement runs `go fix -omitzero=false -diff ./...` per module pinned to Go 1.26.1; `make fix` applies the same across the same module set so the failure message ("run `make fix`") is actionable.

## Glossary
- Amino: gno's deterministic serialization codec; honors `,omitempty` on struct-typed fields, unlike `encoding/json`.
- TypeID: a gno type's canonical string identity, persisted in on-chain state; changing it for a persisted type is consensus-breaking.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. The change is behavior-preserving; the CI `go fix -diff` check is itself the regression guard, and the reverted `omitempty` tags are covered by the existing `qeval_json.txtar` (ObjectInfo) and Amino JSON tests.

## Suggestions
None.

## Open questions
- CI `go fix` enforcement covers a subset of the root module. The check runs per working-directory (`gno.land`, `gnovm`, `tm2`, and the fixed misc matrix `misc/{autocounterd,genproto,genstd,goscan,loop}`), while `make fix` runs root `./...` which also modernizes `misc/genproto2`, `misc/multiarch-determinism`, and `misc/val-scenarios/*`. Those dirs are modernized by `make fix` but never gated by CI, so future hand-written non-modern code there won't be flagged. Consistent with the pre-existing intentional misc CI matrix ("fixed list because we have some non go programs"), so not posted; a design call for the author, not a defect. `Makefile:94` [↗](../../../../../.worktrees/gno-review-5877/Makefile#L94).
- jefft0's thread on `config_set_test.go:49` asks whether removing `testCase := testCase` changes behavior. It does not: under Go 1.22+ per-iteration loop-variable semantics the shadow is a dead no-op, and the `t.Parallel()` subtests capture `testCase` correctly without it. Answer belongs to the author on that thread, not a new comment.

## Verification
Verified on a7313f03c (Go 1.26.4 local):
- Ran the default `go fix` (omitzero enabled) against the tree: it strips `json:",omitempty"` from 12 Amino struct fields including [RefValue](https://github.com/gnolang/gno/blob/a7313f03c/gnovm/pkg/gnolang/values.go) realm-state fields (`Hash`, `OwnerID`, `ObjectID`) and `AddPkg`. Confirms the `-omitzero=false` carve-out is load-bearing, not defensive: without it the sweep changes on-chain and RPC JSON.
- `go fix -omitzero=false -diff ./...` converges to an empty diff on the root module: the tree is fully modernized and `make fix` is idempotent.
- VM-core rewrites read line by line and confirmed behavior-preserving: `uverse.go` `realmIsUserRun` (`strings.Index` → `strings.Cut`, `before == pkgPath[:idx]` and `ok == found`, identical), `native_gas.go` `sumSliceInnerLen` (`min` clamp), `op_call.go`/`machine.go` copy loops, `preprocess.go` `maps.Copy` and `reflect.TypeFor[*TypedValue]()` (static type == dynamic type, panic string unchanged).
- Three independent lens passes (range/min-max, reflect/slices/maps, Builder/SplitSeq/wg.Go) over the full diff each returned no semantic break; notably `types.go` TypeID hash inputs unchanged, `slices.Sort` used only where the original comparator was ascending-natural (the one descending `bytes.Compare` comparator left as `sort.Slice`), and `reflect.TypeFor` fired only on concrete static types (`transcribe.go` interface arg correctly left as `reflect.TypeOf`).
- `go build ./...` clean, `go vet ./gnovm/pkg/gnolang/` clean, touched-package tests pass (`gnoweb/feature/state`, `gnoweb/components`, `gnolang` short, `amino/genproto` TestBasic). The `pb3_gen.go` "passes lock by value" vet notes are pre-existing generated-file output, not in this diff.

## CI
- `docs` red: the docs URL linter (`misc/docs/tools/linter` run with `-treat-urls-as-err=true`) flags remote links in docs files this PR does not touch (`gno.land`, `rpc.gno.land:443`, etc.). Not caused by this Go-only change.
- Merge Requirements red: the github-bot requires gnoweb codeowner review (this PR touches `gno.land/pkg/gnoweb/`). Process gate, not a code issue.
