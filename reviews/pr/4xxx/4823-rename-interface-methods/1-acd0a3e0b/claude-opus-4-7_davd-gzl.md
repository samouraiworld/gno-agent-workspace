# PR #4823: refactor(gnovm): rename gnovm InterfaceType property Method to FieldTypes and allocator Allocate to Account

URL: https://github.com/gnolang/gno/pull/4823
Author: audrenbdb | Base: master | Files: 5 | +23 -23
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-4823 acd0a3e0b` (then `gh -R gnolang/gno pr checkout 4823` inside it)

**Verdict: NEEDS DISCUSSION** — title and body promise two renames, but the `Allocate -> Account` half was reverted in commit `acd0a3e0b`; what remains (`InterfaceType.Methods -> FieldTypes`) is mechanically correct but inconsistent with the still-unrenamed sibling `InterfaceTypeExpr.Methods` ([`nodes.go:706`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/nodes.go#L706) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/nodes.go#L706)) and method `GetMethodFieldType` ([`types.go:943`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/types.go#L943) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/types.go#L943)), and maintainer @mvertes contested the rename direction itself ([review comment](https://github.com/gnolang/gno/pull/4823#discussion_r2387627316)). Decide scope (drop, expand, or land partial) before merging.

## Summary

Addresses issue [#4794](https://github.com/gnolang/gno/issues/4794) (Informational severity audit finding). The original plan was two renames: `Allocator.Allocate -> Account` (only accounts memory, doesn't allocate) and `InterfaceType.Methods -> FieldTypes` (slice holds both methods and embedded interfaces). The author pushed the first rename in `55e90fa`, then reverted it in `acd0a3e` after pushback from @ltzmaxwell that "allocation happens alongside it afterward". Net effect today: a single 5-file mechanical rename of one struct field, leaving the rest of the surface area inconsistent.

## Fix

The `InterfaceType.Methods []FieldType` field becomes `InterfaceType.FieldTypes []FieldType` ([`types.go:906-912`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/types.go#L906-L912) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/types.go#L906-L912)). All 5 internal call sites are updated: [`op_types.go:145`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/op_types.go#L145) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/op_types.go#L145), [`realm.go:1099`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/realm.go#L1099) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/realm.go#L1099) and [`realm.go:1352-1353`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/realm.go#L1352-L1353) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/realm.go#L1352-L1353), [`types.go` body](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/types.go#L917) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/types.go#L917), [`uverse.go`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/uverse.go#L22) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/uverse.go#L22) struct literals for `error`/`stringer`/`realm`, and [`fmt/print.go:79`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/tests/stdlibs/fmt/print.go#L79) · [↗](../../../../.worktrees/gno-review-4823/gnovm/tests/stdlibs/fmt/print.go#L79). No behavior change. Build is clean; pre-existing unrelated TestFiles failures (`types/and_f0.gno`, `eql_0b4`, `eql_0f0`, `or_f0`) also reproduce on the merge-base `47935dae8`, so not caused by this PR.

## Warnings (should fix)

- **[scope inconsistent with title]** [`types.go:943`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/types.go#L943) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/types.go#L943), [`nodes.go:704-708`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/nodes.go#L704-L708) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/nodes.go#L704-L708) — `GetMethodFieldType` and `InterfaceTypeExpr.Methods` still use the old vocabulary the PR claims to abandon.
  <details><summary>details</summary>

  The PR title still references "allocator Allocate to Account" which was reverted in `acd0a3e0b`; update title and body to reflect actual scope before landing. More importantly, the rename premise (slice mixes methods and embedded interfaces) applies symmetrically to:
  - The accessor [`func (it *InterfaceType) GetMethodFieldType(mname Name) *FieldType`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/types.go#L943) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/types.go#L943) — name still says "Method", and the body iterates `it.FieldTypes`. @ltzmaxwell and the author already agreed in-thread on `GetFieldType` ([discussion_r2386617891](https://github.com/gnolang/gno/pull/4823#discussion_r2386617891)).
  - The AST node [`InterfaceTypeExpr.Methods FieldTypeExprs`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/nodes.go#L706) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/nodes.go#L706) — same shape, same misnomer. Author asked, ltzmaxwell said yes; never landed.
  - Local var `methods` populated in [`op_types.go:136-146`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/op_types.go#L136-L146) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/op_types.go#L136-L146) and the comment "list of methods" on [`nodes.go:706`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/nodes.go#L706) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/nodes.go#L706) carry the same drift.

  Fix: either expand the PR to cover `GetMethodFieldType -> GetFieldType` and `InterfaceTypeExpr.Methods -> FieldTypes` (touch [`nodes_copy.go:143`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/nodes_copy.go#L143) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/nodes_copy.go#L143), [`op_eval.go:356-357`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/op_eval.go#L356-L357) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/op_eval.go#L356-L357), [`go2gno.go:360`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/go2gno.go#L360) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/go2gno.go#L360), [`preprocess.go:4469-4475`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/preprocess.go#L4469-L4475) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/preprocess.go#L4469-L4475) and `5317-5318`, [`transcribe.go:312-313`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/transcribe.go#L312-L313) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/transcribe.go#L312-L313), [`transpile_gno0p9.go:444,451`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/transpile_gno0p9.go#L444-L451) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/transpile_gno0p9.go#L444-L451), [`nodes_string.go:227`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/nodes_string.go#L227) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/nodes_string.go#L227)), or pull title/body back to "rename `InterfaceType.Methods` to `FieldTypes`" only. Half-renaming makes the codebase harder to grep, not easier.
  </details>

- **[unresolved direction disagreement]** [PR thread](https://github.com/gnolang/gno/pull/4823#discussion_r2387627316) — @mvertes pushed back on the rename direction; never resolved.
  <details><summary>details</summary>

  Quote: "I'm not convinced that `InterfaceType.FieldTypes` is better than `InterfaceType.Methods`. Even if objects implementing an interface have embedded structs to reach a particular method, interface types are still about methods only." @ltzmaxwell echoed it: "Overall, these are minor differences. I don't have a strong opinion about changing the name due to the reason mentioned in the initial issue." The original audit finding from `#4794` is Informational severity. The PR has been stale since 2025-10-06 with no resolution. Fix: get an explicit ack or NACK from a maintainer (cc: @thehowl @mvertes per @ltzmaxwell's request) before this can merge; alternative naming `Fields` was raised by @ltzmaxwell as potentially better than `FieldTypes`.
  </details>

## Nits

- [`gnovm/pkg/gnolang/op_types.go:136-146`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/op_types.go#L136-L146) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/op_types.go#L136-L146) — local variable `methods` is now a misnomer; building `[]FieldType` to assign to `FieldTypes`, rename to `fieldTypes` (or `fts`) for consistency.
- [`gnovm/pkg/gnolang/realm.go:1352`](https://github.com/gnolang/gno/blob/acd0a3e0b/gnovm/pkg/gnolang/realm.go#L1352) · [↗](../../../../.worktrees/gno-review-4823/gnovm/pkg/gnolang/realm.go#L1352) — `for i, mthd := range ct.FieldTypes` keeps the `mthd` shorthand; minor stale-naming carry-over.
- PR body still claims a `Allocate -> Account` rename that no longer exists in the diff (commit `acd0a3e0b` reverted `55e90fa0`). Update description if landing as-is.
- No ADR included. `AGENTS.md` requires one for "non-trivial AI-assisted PRs"; a pure rename across a stable VM field arguably qualifies as trivial under "formatting, simple tests, docs-only" carve-out, but maintainer call.

## Missing Tests

- None. Pure mechanical rename, no behavior change, no new code paths.

## Questions for Author

- Are you planning to expand this PR to also cover `GetMethodFieldType` and `InterfaceTypeExpr.Methods` (per [discussion_r2386617891](https://github.com/gnolang/gno/pull/4823#discussion_r2386617891)), or land the partial rename and follow up?
- @ltzmaxwell suggested `Fields` over `FieldTypes` ([same thread](https://github.com/gnolang/gno/pull/4823#discussion_r2386617891)) as a third option — was that considered and rejected, or just dropped? `Fields` matches `StructType.Fields` precedent.
- Given @mvertes's objection that "interface types are still about methods only", do you have a one-line counter-argument worth pinning in the PR body so reviewers don't relitigate?
