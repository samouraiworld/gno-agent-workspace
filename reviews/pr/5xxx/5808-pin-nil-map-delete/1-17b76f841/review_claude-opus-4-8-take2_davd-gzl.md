# PR #5808: test(gnovm): pin nil-map delete semantics (follow-up to #5196)

URL: https://github.com/gnolang/gno/pull/5808
Author: omarsy | Base: master | Files: 3 | +107 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8-take2 | Commit: `17b76f841` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5808 17b76f841`
Overview: [visual overview](../overview.html) · [↗](../overview.html)

Round note (2026-06-11): blind validation run. [review_claude-fable-5_davd-gzl.md](review_claude-fable-5_davd-gzl.md) reviewed this same commit and surfaced the recoverability staleness as a posted Warning plus the unpinned readonly-delete panic; its verdict is REQUEST CHANGES. Prefer it for posting.

**TL;DR:** In Go, `delete(m, k)` on a nil map does nothing. GnoVM used to crash instead. The actual one-line fix already merged in #5196; this PR only adds tests that lock in the corrected behavior across more ways of reaching a nil map, plus a design note (ADR) recording two deliberate edge-case choices.

**Verdict: APPROVE** — purely additive tests + ADR for an already-merged guard; both new filetests pass, CI green, no code path changed. One reviewer-only note on the ADR's recoverability framing (Open questions), not a blocker and not posted.

## Summary
#5196 added a nil guard to the `delete` builtin ([`uverse.go:978-980`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/uverse.go#L978-L980) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/uverse.go#L978-L980)): return before the `*MapValue` type assertion, so deleting from a nil map is a no-op instead of an `interface conversion: ... is nil, not *gnolang.MapValue` VM abort. This PR adds `delete1.gno` (nil maps via local/package var, struct field, function return, conversion literal, plus an unhashable-key case) and `zrealm_mapnil.gno` (a nil map obtained from another realm), and an ADR documenting two consequences. No production code changes; +107 lines, all tests/docs.

## Glossary
- **filetest** — `.gno` file under `gnovm/tests/files/` run by `TestFiles`, asserting on an `// Output:` or `// Error:` directive.
- **readonly taint** — cross-realm write guard: a real object owned by an external realm cannot be mutated by the executing realm (`Machine.IsReadonly`).
- **recoverable exception** — a gno panic raised as `panic(&Exception{...})`, catchable by a gno `defer/recover`. A raw Go `panic(string)` inside the VM is not catchable and aborts the whole run.

## Fix
No fix in this PR. The guard it documents lives at [`uverse.go:976-980`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/uverse.go#L976-L980) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/uverse.go#L976-L980): a `*MapType` arrives with `arg0.TV.V == nil` for a nil map, and the early `return` precedes both the `*MapValue` assertion (line 981) and the readonly check (line 983). The key expression is still evaluated at the call site (params are resolved by `GetParams2` before the native body runs), so the Go-spec "key evaluated exactly once" property holds independent of the guard.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. Coverage is the point of this PR and is thorough: the nil map is reached via local var, package var, struct field, function return, conversion literal (`delete1.gno`), and a cross-realm value (`zrealm_mapnil.gno`); the readonly-tainted *non-nil* path stays pinned by the pre-existing `zrealm_map1.gno`/`zrealm_map3.gno` (both verified passing). The one path worth naming is covered indirectly — see Open questions.

## Suggestions
None.

## Open questions
- The ADR's consequence 2 says gno's unhashable-map-key panic "does not go through the recoverable exception path," so hashing the key before the nil return "would today turn the case into an unrecoverable VM abort." Against the code this is true for some unhashable keys and false for others, and `delete1.gno` pins the *false* case. [`values.go:1661-1662`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/values.go#L1661-L1662) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/values.go#L1661-L1662) raises the slice-key panic as `panic(&Exception{...})` — recoverable; I confirmed a non-nil-map `delete(m, []int{1})` is caught by a gno `defer/recover` ([opus2_slicekey_recover_probe.gno](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/opus2_slicekey_recover_probe.gno) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/opus2_slicekey_recover_probe.gno) prints `recovered: ...`). A *func* key instead hits the `default` arm [`values.go:1683-1686`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/values.go#L1683-L1686) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/values.go#L1683-L1686), a raw `panic(fmt.Sprintf(...))` that does abort unrecoverably (opus2_funckey_recover_probe.gno: runner reports `unexpected panic: unexpected map key type func()`). So the "unrecoverable abort" framing fits the func/map key, not the slice key the ADR's example actually exercises. The chosen behavior (no-op) and the test are both correct on the primary spec-compliance ground (Go spec says nil-map delete is a no-op, and gno's nil-map *reads* never hash the key either — [`op_expressions.go:17-18`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/op_expressions.go#L17-L18) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/op_expressions.go#L17-L18)); only the secondary recoverability rationale is imprecise. Not posted: it touches ADR rationale, not code behavior, and changes neither the verdict nor the author's next action.
- ADR consequence 1 names `TypedValue.IsReadonly` and `SetReadonly`; the actual symbols are `TypedValue.IsReadonlyBy` and `Machine.IsReadonly`, and there is no `SetReadonly` in the tree. The behavioral claim (a nil value is never readonly-tainted) is correct regardless: a nil `tv.V` falls to the `default` arm of `IsReadonlyBy` and returns false ([`ownership.go:525-527`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/ownership.go#L525-L527) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/ownership.go#L525-L527)). Not posted: documentation-symbol nit, no code impact.

## Verified live
Not applicable: no server/tool runtime behavior changes (filetests + ADR only). Verification was the filetest suite plus differential oracles, below.

Ran from the worktree (`17b76f841`):

| What | Result |
|------|--------|
| `go test -run 'TestFiles/delete1.gno$' ./gnovm/pkg/gnolang/` | PASS |
| `go test -run 'TestFiles/zrealm_mapnil.gno$' ./gnovm/pkg/gnolang/` | PASS |
| `go test -run 'TestFiles/map48.gno$' ./gnovm/pkg/gnolang/` (#5196's test) | PASS |
| `go test -run 'TestFiles/zrealm_map1.gno$' + zrealm_map3` (readonly-tainted delete) | PASS |
| slice-key recoverability probe (`opus2_slicekey_recover_probe.gno`) | `recovered: ... slice type cannot be used as map key` |
| func-key recoverability probe (`opus2_funckey_recover_probe.gno`) | unrecoverable abort: `unexpected map key type func()` |
| Go gc oracle (`opus2_go_oracle_nilmap_delete.go`) | nil-map slice-key delete: `PANIC: hash of unhashable type: []int` |

Note: a full `go test -run Files -test.short ./gnovm/pkg/gnolang/` run shows one unrelated failure in `or_f0.gno` (a `|`-operator type-check error-message mismatch). That file is not touched by this PR (`origin/master..HEAD` log empty for it) and is unrelated to map/delete; pre-existing on the base, not introduced here.
