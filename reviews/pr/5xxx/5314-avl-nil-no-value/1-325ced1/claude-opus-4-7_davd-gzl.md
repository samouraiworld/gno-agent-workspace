# PR #5314: fix(example/avl): simplify `Get` to return `nil` as "no value"

URL: https://github.com/gnolang/gno/pull/5314
Author: davd-gzl | Base: master | Files: 94 | +543 -533
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5314 325ced1` (then `gh -R gnolang/gno pr checkout 5314` inside it)

**Verdict: APPROVE** — clean breaking-API change across all in-tree `avl.Tree` callers (94 files, 2 approvals already), correctly skips deletion semantics, all targeted tests pass; only concerns are documentation of the breaking nature in the PR body, a contract subtlety in `record.Get` worth a code comment, and one stale test-error message in `p/moul/cow`. Self-review: I'm the author of this PR — bias disclosed.

## Summary

`avl.Tree.Get` (and `rotree.ReadOnlyTree.Get`, `cow.Tree.Get`, `datasource.Fields.Get`, the benchops twin in `gnovm/pkg/benchops/gno/avl`) switches from `(value any, exists bool)` to a single `value any` return, treating a nil interface as "no value". This is breaking but unblocks the [`gno.land/p/nt/avl/v0/README.md:24-43`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/avl/v0/README.md#L24-L43) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/avl/v0/README.md#L24-L43) one-liner pattern `if v, ok := tree.Get(k).(*T); ok { ... }`. The PR explicitly does **not** make `Set(k, nil)` an alias for `Delete` (good — that would be a silent footgun for callers using `nil` as a value-presence sentinel); `Delete` and `Has` remain canonical. Every in-tree caller is migrated (94 files; I cross-checked with a regex sweep — the remaining 2-return `Get(...)` matches in `examples/` are all `bptree.BPTree` receivers, addressed separately in PR #5644).

## Glossary

- **`ITree` / `IReadOnlyTree`** — the public interfaces in `p/nt/avl/v0/{tree.gno,rotree/rotree.gno}` whose `Get` signature this PR rewrites.
- **`p/moul/cow.Tree`** — copy-on-write avl variant; its `Get` is rewritten in parallel.
- **`p/jeronimoalbi/datasource.Fields`** — application-level interface that mirrored the old `Get(name) (any, bool)`; also rewritten to single return.
- **`bptree`** — unrelated B+ tree package with the same anti-pattern; handled in a separate PR (#5644) per author.

## Fix

The core change is small: [`p/nt/avl/v0/tree.gno:56-59`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/avl/v0/tree.gno#L56-L59) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/avl/v0/tree.gno#L56-L59) drops the `exists` return and the `ITree` interface at [`p/nt/avl/v0/tree.gno:8`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/avl/v0/tree.gno#L8) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/avl/v0/tree.gno#L8) follows. `Node.Get` (the internal three-return helper at [`p/nt/avl/v0/node.gno:83`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/avl/v0/node.gno#L83) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/avl/v0/node.gno#L83)) is unchanged — it still returns `(index, value, exists)` so iterator/index code keeps working; the tree wrapper just discards `exists`. The implicit contract is "stored value cannot be untyped nil" — true for every realm in the tree because they all store typed pointers/structs.

Migration shape across the 94 files:
- `v, ok := t.Get(k); if !ok { ... }` -> `v := t.Get(k); if v == nil { ... }`
- `_, ok := t.Get(k); return ok` -> `return t.Has(k)` (the bigger win — replaces O(log n) traversals that throw away the value with a leaner equivalent, and is now the documented idiom in [`docs/resources/effective-gno.md:702-704`](https://github.com/gnolang/gno/blob/325ced1/docs/resources/effective-gno.md#L702-L704) · [↗](../../../../../.worktrees/gno-review-5314/docs/resources/effective-gno.md#L702-L704) and [`docs/resources/gno-data-structures.md:53-56`](https://github.com/gnolang/gno/blob/325ced1/docs/resources/gno-data-structures.md#L53-L56) · [↗](../../../../../.worktrees/gno-review-5314/docs/resources/gno-data-structures.md#L53-L56))

## Critical (must fix)

None.

## Warnings (should fix)

- **[PR body omits the BREAKING-CHANGE disclosure already requested by @jefft0]** [@jefft0](https://github.com/gnolang/gno/pull/5314#pullrequestreview-2841250000) PR body — the change breaks every external realm that uses `avl.Tree.Get`, `rotree.ReadOnlyTree.Get`, `cow.Tree.Get`, or `datasource.Fields.Get`
  <details><summary>details</summary>

  jefft0's approval at 2026-05-11 explicitly says "The PR description should document that this is BREAKING CHANGE." The PR body still doesn't, and the commit messages don't carry a `BREAKING CHANGE:` footer either. With the v0 path (`gno.land/p/nt/avl/v0`) implying API stability, downstream realms that pinned to this path will fail compilation on their next deploy after this lands. Fix: add a `BREAKING CHANGE:` paragraph to the PR body listing the four affected exported types/methods and the one-line migration recipe (`v, ok := t.Get(k)` -> `v := t.Get(k); ok := v != nil`, plus the `Has` upgrade hint).
  </details>

- **[`record.Get` silently collapses "stored nil" with "missing"]** [`p/jeronimoalbi/datastore/record.gno:131-140`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L131-L140) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L131-L140) — `found = v != nil` changes semantics, not just the implementation
  <details><summary>details</summary>

  Before this PR, `record.Get(field)` returned `(nil, true)` if the field index existed and had been `Set(...)` to a nil interface; the schema-known field with explicit nil-value would not panic in `MustGet`. After, that same path returns `(nil, false)` and `MustGet` panics. The new contract is consistent with the rest of the PR (nil-stored == not-stored), and `Set` requires a non-nil `value` at [`p/jeronimoalbi/datastore/record.gno:126`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L126) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L126) anyway, so the divergence is theoretical for in-tree usage — but the `record.Record` interface still exposes the `(value, found bool)` shape at [`p/jeronimoalbi/datastore/record.gno:51`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L51) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L51) and external callers will read `found` as "field exists in schema". Fix: either drop the `found` return on `record.Record.Get` (mirror the avl change) or add a one-line comment above [`p/jeronimoalbi/datastore/record.gno:139`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L139) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/jeronimoalbi/datastore/record.gno#L139) noting the semantic now means "field has a non-nil value" not "schema index exists". The same pattern recurs in [`p/jeronimoalbi/datastore/schema.gno:117-128`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/jeronimoalbi/datastore/schema.gno#L117-L128) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/jeronimoalbi/datastore/schema.gno#L117-L128) (`GetDefaultByIndex`) — same fix applies.
  </details>

## Nits

- [`p/moul/cow/tree_test.gno:45-50`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/moul/cow/tree_test.gno#L45-L50) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/moul/cow/tree_test.gno#L45-L50) — stale assertion strings: "Expected Get to return value1 and true" / "Expected Get to return false for non-existent key" now refer to a `bool` that no longer exists. Trivial copy-edit to match the new single-return semantics ("Expected Get to return value1" / "Expected Has to return false for non-existent key").
- [`p/moul/cow/tree.gno:73-79`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/moul/cow/tree.gno#L73-L79) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/moul/cow/tree.gno#L73-L79) — `cow.Tree.Get` keeps a defensive `if !exists { return nil }` branch while `avl.Tree.Get` at [`p/nt/avl/v0/tree.gno:56-59`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/avl/v0/tree.gno#L56-L59) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/avl/v0/tree.gno#L56-L59) drops the check (since `Node.Get` already returns nil when not found). The two should match — either both keep the redundant check for readability or both elide it. No behavioral impact.
- [`gnovm/pkg/benchops/gno/storage/boards.gno:4-8`](https://github.com/gnolang/gno/blob/325ced1/gnovm/pkg/benchops/gno/storage/boards.gno#L4-L8) · [↗](../../../../../.worktrees/gno-review-5314/gnovm/pkg/benchops/gno/storage/boards.gno#L4-L8) and [`gnovm/pkg/benchops/gno/storage/forum.gno:4-8`](https://github.com/gnolang/gno/blob/325ced1/gnovm/pkg/benchops/gno/storage/forum.gno#L4-L8) · [↗](../../../../../.worktrees/gno-review-5314/gnovm/pkg/benchops/gno/storage/forum.gno#L4-L8) — the import-grouping reshuffle (`stdlib` then `gno.land/...`) is incidental to this PR. Not wrong, but flagging in case you want to keep the diff focused.
- [`p/nt/bptree/v0/tree_test.gno:2331-2340`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/bptree/v0/tree_test.gno#L2331-L2340) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/bptree/v0/tree_test.gno#L2331-L2340) — `TestAVLCrossValidation` now does an extra `at.Has(key)` call to recover the `aOk` parity check. Fine, but it's an extra O(log n) traversal per Get-case iteration of the fuzz loop — when bptree's own API converges with the avl one in #5644 the helper can drop the `Has` call again. Worth a follow-up TODO.

## Missing Tests

- **[no test exercises `Set(k, nil)` post-change]** [`p/nt/avl/v0/tree_test.gno`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/avl/v0/tree_test.gno) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/avl/v0/tree_test.gno) — the PR body promises `Set(k, nil)` is not aliased to `Delete`, but there's no regression test asserting that invariant
  <details><summary>details</summary>

  Concretely: a test that does `tree.Set("a", nil); assert tree.Has("a") == true; assert tree.Get("a") == nil; assert tree.Size() == 1` would lock in the documented "no deletion side effect" semantic and stop a future contributor from "simplifying" `Set` into a delete-on-nil. Cheap to add (3 lines in `TestTreeGet`), high coverage value because the contract is non-obvious without it.
  </details>

## Suggestions

- [`p/nt/avl/v0/README.md:36-43`](https://github.com/gnolang/gno/blob/325ced1/examples/gno.land/p/nt/avl/v0/README.md#L36-L43) · [↗](../../../../../.worktrees/gno-review-5314/examples/gno.land/p/nt/avl/v0/README.md#L36-L43) — the new `Exists` helper example is fine, but the doc would be stronger if it also showed the one-liner pattern that motivated the change (the `if g, ok := tree.Get(k).(*Game); ok` form quoted in the original issue #1071). That's the actual readability win; the bare `tree.Has(k)` is a smaller delta.
- The PR mixes three concurrent API changes (`avl.Tree`, `cow.Tree`, `datasource.Fields`). They share rationale but `cow.Tree` and `datasource.Fields` have no v0/stability commitment, so they could ship in a follow-up if reviewers want to minimize blast-radius for the core change. Not blocking — given the existing approvals and the maintenance cost of carrying conflicts, shipping together is reasonable.

## Questions for Author

- Confirm that no external realm (off this monorepo) is known to depend on the `(any, bool)` signature — if any community realms pin `gno.land/p/nt/avl/v0`, this lands as a silent breakage on their next redeploy. If yes, is a `v1` path warranted (mirroring how `bptree` lives at `v0/`)?
- Is there a follow-up planned to drop the now-vestigial `Iterator`-style 2-return `Fields.Get` exposure in `record.Record` (see warning above), or is preserving that interface intentional to keep dependents compiling?
