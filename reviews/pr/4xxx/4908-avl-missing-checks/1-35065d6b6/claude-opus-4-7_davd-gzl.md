# PR #4908: fix(avl): add missing checks in avl package

URL: https://github.com/gnolang/gno/pull/4908
Author: davd-gzl | Base: master | Files: 2 | +17 -3
Reviewed by: davd-gzl (AI agent, claude-opus-4-7) | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-4908 35065d6b6` (then `gh -R gnolang/gno pr checkout 4908` inside it)

**Verdict: NEEDS DISCUSSION** — review by the PR's own author via AI agent; disclosure required, conflict of interest noted. Three bounded-input panics added in `p/nt/avl/v0`, but the closing claim in the PR body ("there shouldn't be any other needed checks in avl packages") is contradicted by `TraverseByOffset` still silently truncating on negative `offset`/`limit`, and the new `pageSize <= 0` panic both breaks a documented API contract (previously returned an empty page) and turns the existing `pageSize < 1` guard at [`pager.gno:71-73`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) into dead code.

## Summary

Triage on issue [#4440](https://github.com/gnolang/gno/issues/4440) ("avl uses signed size"). notJoon's conclusion was: keep `int`, add bounds checks for `GetByIndex` and `TraverseByOffset`. This PR addresses two of the three: `GetByIndex` gets an upfront `index < 0` panic ([`node.gno:111-113`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L111-L113) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L111-L113)), `calcHeightAndSize` gets post-hoc overflow guards on `height`/`size` ([`node.gno:274-279`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L274-L279) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L274-L279)), and `Pager.GetPageWithSize` gets a `pageSize <= 0` panic to prevent the div-by-zero in `math.Ceil(N/0)` at [`pager.gno:56-58`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L56-L58) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L56-L58). `TraverseByOffset` is left untouched despite the issue text and intermediate commits explicitly proposing checks, then reverting them. Only `v0` is touched; the legacy `p/demo/avl` paths the issue title references no longer exist in `examples/`.

## Glossary

- `GetByIndex` — `Node`/`Tree` accessor returning the (key, value) at a 0-based positional index in the in-order traversal.
- `TraverseByOffset` — offset/limit traversal used by `Tree.IterateByOffset` and the pager.
- `calcHeightAndSize` — recomputes `node.height` (`int8`) and `node.size` (`int`) after rotations/inserts/removes; called from `Set`, `Remove`, `rotateLeft`, `rotateRight`.
- `Pager.GetPageWithSize` — public pager entry that pages over an `IReadOnlyTree`; reached via `GetPage`, `GetPageByPath`, `MustGetPageByPath`.

## Fix

Before: `GetByIndex(-1)` on a non-leaf recursed and could panic deep in the tree with `"GetByIndex asked for invalid index"`; on a leaf it returned the leaf's value silently for any `index != 0` after the existing branch. `Pager.GetPageWithSize(_, 0)` reached `int(math.Ceil(N/0))` — `+Inf` cast to `int` is implementation-defined. After: both gain an early panic with a clear message. `calcHeightAndSize` adds two post-condition checks that detect `int8` height overflow and `int` size overflow after the sum has already wrapped — defensive only (a balanced AVL with `height > 127` requires `N > ~2^88` nodes; `size` overflow needs `N > 2^63`). The original `if pageSize < 1` branch at [`pager.gno:71-73`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) is now unreachable.

## Warnings (should fix)

- **[disclosure: author = reviewer via AI]** — This review is written by an AI agent run by the PR author (davd-gzl). It is posted in good-faith as adversarial self-review, but it cannot substitute for an independent human reviewer. Mark it as such on the PR (e.g. `[bot]` prefix or "AI self-review" note) so maintainers can weigh accordingly. Per workspace AGENTS.md: "If posting comments or reviews under your owner's GitHub account, disclose that you are an AI agent."
- **[PR claim contradicted by code]** [`node.gno:390-408`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L390-L408) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L390-L408) — PR body says "there shouldn't be any other needed checks in avl packages", but `TraverseByOffset` still silently truncates on negative `offset` and is the third call-out in the linked issue.
  <details><summary>details</summary>

  Issue #4440 listed three problems; notJoon's accepted resolution explicitly says "add guard for GetByIndex and TraverseByOffset methods". This PR only handles `GetByIndex`. The author added a negative-offset/negative-limit panic in commits `bee655fd`/`1d0bcb02`/`ed94b187`, then reverted both ("remove panic on negative offset", "remove panic on negative limit in TraverseByOffset"). No rationale recorded in PR thread for the reversal, and the closing claim in the body conflicts with the empirical behavior below.

  Observed on this PR's HEAD (`35065d6b6`) against `Tree{a,b,c,d,e}`:

  | call | expected | observed |
  | --- | --- | --- |
  | `IterateByOffset(-3, 3, …)` | panic or 3 items | 2 items `[a b]` (silent truncation) |
  | `IterateByOffset(-1, 2, …)` | panic or 2 items | 2 items `[a b]` |
  | `ReverseIterateByOffset(-2, 3, …)` | panic or 3 items | 2 items `[e d]` |
  | `IterateByOffset(0, -1, …)` | 0 items (already short-circuited at [`node.gno:396`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L396) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L396)) | 0 items |

  The `limit <= 0` short-circuit at [`node.gno:396`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L396) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L396) covers negative `limit`, but `offset >= node.size` does not cover negative `offset`. The caller guarantees on the recursive helper at [`node.gno:413-414`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L413-L414) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L413-L414) say "offset < node.size; limit > 0" — they do not say "offset >= 0", which is why the truncation is internally consistent and silent rather than crashy.

  Fix: either add `if offset < 0 { panic(...) }` next to the `pageSize <= 0` check (consistent with `GetByIndex(-1)` now panicking), or normalise (`if offset < 0 { offset = 0 }`) and document. Whichever, update the PR body so it doesn't claim coverage that isn't there.

  Repro (from a local clone of gnolang/gno):

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 4908 -R gnolang/gno
  cat > examples/gno.land/p/nt/avl/v0/neg_offset_test.gno <<'EOF'
  package avl

  import "testing"

  func TestNegOffsetSilentTruncation(t *testing.T) {
      tree := NewTree()
      for _, k := range []string{"a", "b", "c", "d", "e"} {
          tree.Set(k, nil)
      }
      var keys []string
      tree.IterateByOffset(-3, 3, func(k string, _ any) bool {
          keys = append(keys, k)
          return false
      })
      t.Logf("offset=-3 count=3 => %v (expected 3 items or panic, got %d)", keys, len(keys))
      if len(keys) == 3 {
          t.Fatal("issue #4440 'TraverseByOffset with negative offset' would be fixed")
      }
  }
  EOF
  cd examples/gno.land/p/nt/avl/v0 && gno test -v -run TestNegOffsetSilentTruncation .
  cd - && rm examples/gno.land/p/nt/avl/v0/neg_offset_test.gno
  ```
  </details>

- **[API break + dead code]** [`pager.gno:56-58`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L56-L58) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L56-L58) — `pageSize <= 0` now panics; previously `GetPageWithSize(_, 0)` returned an empty page via the now-dead branch at [`pager.gno:71-73`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73).
  <details><summary>details</summary>

  Direct callers of `GetPageWithSize` outside the package: [`r/demo/defi/grc20factory/grc20factory.gno:138`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L138) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L138). All other callers go through `GetPage` → `DefaultPageSize` (set at `NewPager` time, e.g. 5, 10, 12, 20, 50 in `examples/`) or `GetPageByPath` → `ParseQuery`, which already coerces `size <= 0` to `DefaultPageSize` at [`pager.gno:218-220`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L218-L220) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L218-L220). So the realistic blast radius is: (a) direct `GetPageWithSize(_, 0)` callers (one in tree today, future ones), (b) anyone constructing a `Pager` with `DefaultPageSize <= 0` (would explode on first `GetPage`).

  Two issues with the current shape:
  1. The `if pageSize < 1` branch at [`pager.gno:71-73`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) is now unreachable. Delete it; dead code rots.
  2. The behavior change from "empty page" to "panic" is a public API break in `p/nt/avl/v0`. If `v0` is the stability surface, this needs a CHANGELOG note or — more conservatively — coerce instead of panic ("`if pageSize <= 0 { return emptyPage }`") and align with `ParseQuery`'s existing tolerance.

  Fix: pick one — either panic and delete the dead branch, or coerce and keep the existing semantics. Either way, mention in the PR body that this is a behavioral change to `Pager`, not just a guard.
  </details>

- **[overflow checks fire too late]** [`node.gno:270-280`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L270-L280) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L270-L280) — `height` and `size` are summed first, then checked. By the time `node.height < 0` is true, the wrap has already happened and may have been read by a concurrent rotation/balance call frame.
  <details><summary>details</summary>

  The check is logically a post-condition: signed overflow is undefined in Go for `int8` (well, it wraps two's-complement, but the language spec doesn't promise it), so the panic catches the symptom rather than preventing the operation. Practically irrelevant — `int8` height overflow requires ~`2^88` nodes in a balanced AVL, `int` size overflow requires ~`2^63` — but if the goal is "make corrupt state unreachable", the check should compute the sums into wider types (`int16`/`int64`) and panic before assigning. As written this is documentation-of-impossible rather than a load-bearing guard. Either widen the temporaries, drop the checks, or leave a comment that these are belt-and-braces against future refactors that might lower the bound.
  </details>

## Nits

- [`node.gno:115-119`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L115-L119) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L115-L119) — leaf branch was restructured (early-return form) on Villaquiranm's request. Fine, but unrelated to the bugfix; either factor out into its own commit or call it out in the PR body so future `git blame` makes sense.
- [`pager.gno:57`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L57) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L57) — panic message "invalid page size" doesn't say what was invalid; consider `ufmt.Sprintf("GetPageWithSize: pageSize must be > 0, got %d", pageSize)` for symmetric debugging with `GetByIndex`'s message.
- [`node.gno:111-113`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L111-L113) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L111-L113) — message says "negative index not allowed" but the function also panics on `index >= size` (via [`node.gno:117`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L117) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L117)) with a different wording. Either unify both panics into one upfront `if index < 0 || index >= node.size` guard, or align messages.

## Missing Tests

- **[regression-shield missing]** [`pager_test.gno`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager_test.gno) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager_test.gno) — no test for the new `pageSize <= 0` panic.
  <details><summary>details</summary>

  Codecov reports "modified and coverable lines are covered" but the panic path is asserted only implicitly. Add an explicit `defer recover()` test asserting the panic shape, plus a positive test that `GetPageWithSize(1, 1)` still works (boundary). Without that, a future change that drops the `<= 0` to `< 0` slips through silently.
  </details>
- **[regression-shield missing]** [`node_test.gno`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node_test.gno) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node_test.gno) — `TestGetByIndex` already covers `-1` and `len(input)` via `expectPanic: true`, but does not distinguish which panic fired. Pin the message (`"GetByIndex: negative index not allowed"` vs `"GetByIndex asked for invalid index"`) to lock the contract.
- **[no test for overflow guards]** [`node.gno:274-279`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/node.gno#L274-L279) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L274-L279) — there is no way to exercise the `height < 0` / `size < 0` paths short of crafting a Node with hand-rolled `int8` overflow. Either drop the checks or add a focused unit test that directly constructs a `Node{height: math.MaxInt8, ...}` and triggers `calcHeightAndSize`.

## Suggestions

- Add an ADR. Per workspace AGENTS.md, "Every non-trivial AI-assisted PR must include an ADR." This PR is small but it (a) changes a public API behavior in `Pager`, (b) deliberately departs from notJoon's recommendation by skipping `TraverseByOffset`, (c) makes a defensive choice (panic vs coerce) that has long-term implications. An ADR under `examples/gno.land/adr/` (or wherever realms ADRs live) documenting "we chose panic over coerce for invalid pageSize because X" would help the next person.
- Split the four-commit shape into a clean two-commit PR: (1) `fix(avl): panic on negative index in GetByIndex`, (2) `fix(pager): panic on non-positive pageSize`. The overflow-guard commit feels speculative enough to belong in its own PR with a discussion of the threat model. The current `Merge remote-tracking branch 'origin/master'` commit at HEAD pollutes the history; rebase before landing.

## Questions for Author

- Why was the negative-`offset`/`limit` panic on `TraverseByOffset` removed in commits `1d0bcb02` and `ed94b187`? The PR body now claims "no other checks needed", but the linked issue specifically calls out `TraverseByOffset` and the empirical behavior above shows silent truncation. What changed your mind?
- Is `pageSize <= 0` panicking instead of coercing-to-`DefaultPageSize` (the pattern `ParseQuery` already uses at [`pager.gno:218-220`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L218-L220) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L218-L220)) intentional? The two policies in the same file diverge.
- Should the unreachable `if pageSize < 1` at [`pager.gno:71-73`](https://github.com/gnolang/gno/blob/35065d6b6/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L71-L73) be deleted in this PR, or do you want to keep it as a backstop in case the upfront panic is later relaxed?
