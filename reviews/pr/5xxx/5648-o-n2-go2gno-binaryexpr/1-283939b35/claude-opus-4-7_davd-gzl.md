# PR #5648: fix(gnolang): O(N²) in Go2Gno Span for BinaryExpr chains

URL: https://github.com/gnolang/gno/pull/5648
Author: omarsy | Base: master | Files: 4 | +382 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7

> **Verdict — APPROVE (with caveats).** BinaryExpr fix correct, benchmarks convincing.
>
> Two open concerns: Y-side `ParenExpr` column drift, and SelectorExpr / IndexExpr / CallExpr / StarExpr also measured O(N²) at parse time ([@thehowl](https://github.com/gnolang/gno/pull/5648#pullrequestreview-4320920070)) — same DoS surface as BinaryExpr, left untouched.

---

## Summary

Long `+` chain `1 + 1 + ... + 1` parses left-leaning. Recording each node's Span calls `gon.Pos()`; for `BinaryExpr` stdlib defines `Pos() = X.Pos()` → walks all the way left. N nodes × N walk = O(N²). At N=54 000 (~216 KB source, ~20% of one `MaxTxBytes=1 MB` tx): **~13 s validator CPU on M1 Pro**, multiple block-production budgets. Fires at parse time, before any gas meter.

Fix: outer node reads already-computed Span from translated children instead of re-walking raw Go AST. O(1) per node, O(N) total.

```
Chain: a + b + c + d + e  ==  (((a+b)+c)+d)+e
                          outer
                        /      \
                     L4          e
                    /  \
                  L3    d
                 /  \
               L2    c
              /  \
             a    b

Before: outer.Pos() walks L4→L3→L2→a  (N hops) × N nodes = O(N²)
After:  outer reads L4.Span.Pos        (1 hop)  × N nodes = O(N)
```

---

## Glossary

- **Span** = source range of a node (start + end position).
- **`setSpan`** = records the Span on a node.
- **`SpanFromGo`** = builds a Span by asking the Go AST node its `Pos()`/`End()`. `BinaryExpr.Pos()` walks left — source of the blow-up.

---

## Fix

In [`gnovm/pkg/gnolang/go2gno.go:276-298`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno.go#L276-L298), the `*ast.BinaryExpr` case used to just build the Gno node and return it — leaving the deferred `setSpan` to compute the Span by calling the costly `gon.Pos()`. Now: after building the node, if `X` is itself a `BinaryExpr` (chain link), the code sets the Span directly from the children's already-computed Spans (`Pos` from the left child, `End` from the right) and skips the costly call. Restricted to chained X so non-chain shapes (`ParenExpr` / `Ident` / etc.) keep the original column semantics. Recursion bottoms out at the deepest BinaryExpr whose X is a leaf, where the default path runs once at O(1).

## Benchmarks

M1 Pro, median ns/op:

| N      | before    | after    | speedup | growth (4× N) |
|--------|----------|---------|--------|---------------|
| 1 000  | 3.97 ms  | 0.55 ms | 7.3×   | —             |
| 4 000  | 62.97 ms | 2.57 ms | 24.5×  | 15.9× / 4.7×  |
| 16 000 | 991.6 ms | 11.6 ms | 85.6×  | 15.7× / 4.5×  |

Reproduced on Ryzen 7 7840HS: 0.89 / 5.34 / 35.68 ms at N=1k/4k/16k.

## Critical (must fix)

None.

## Warnings (should fix)

- **[parse-time O(N²) on 4+ other AST shapes]** [@thehowl](https://github.com/gnolang/gno/pull/5648#pullrequestreview-4320920070) [`pr5648_spanfromgo_quadratic.md:195`](../../../../../.worktrees/gno-review-5648/gnovm/adr/pr5648_spanfromgo_quadratic.md#L195) — same DoS on SelectorExpr / IndexExpr / CallExpr / StarExpr; type-check argument doesn't apply pre-preprocess.
  <details><summary>details</summary>

  **Shape:** `a.x.x.x...` (Selector), `a[0][0]...` (Index), `f()()()...` (Call), `***...int` (Star). All recurse via `Pos()`/`End()` in `go/ast`.

  **Measured by @thehowl** (Ryzen 7 7840U, `-benchtime=10x`):

  | shape                     | N=1k    | N=4k    | N=16k  | 4×N ratio     | verdict |
  |---------------------------|--------:|--------:|-------:|--------------:|---------|
  | `a.x.x.x…` Selector       | 1.5 ms  | 17.7 ms | 250 ms | 11.4× / 14.1× | O(N²)   |
  | `a[0][0][0]…` Index       | 2.0 ms  | 23.4 ms | 370 ms | 11.9× / 15.8× | O(N²)   |
  | `f()()()…` Call           | 1.4 ms  | 20.5 ms | 281 ms | 15.1× / 13.7× | O(N²)   |
  | `***…int` Star (End)      | 1.2 ms  | 16.7 ms | 219 ms | 13.5× / 13.1× | O(N²)   |

  Same pattern by inspection: `IndexListExpr`, `SliceExpr`, `TypeAssertExpr`, `UnaryExpr`, `ArrayType`, `MapType`, `ChanType` (`go/ast/ast.go:495-569`).

  **Why dismissal fails:** the audit cites `MaxTypeDepth=8`, "CallExpr.Fun typically Ident", "SelectorExpr needs nested embedding to type-check". But O(N²) lands at parse/Go2Gno, **before any type validation**. Parser only caps `maxNestLev=1e5` ([`parser.go:1718-1719`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/parser/parser.go#L1718-L1719), [`parser.go:1874-1875`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/parser/parser.go#L1874-L1875)) — same cap that lets BinaryExpr reach DoS depth. `validateTypeDepth` runs in preprocess, after Go2Gno.

  **Why it matters:** at N=1e5, measured slopes extrapolate to ~10-30 s validator CPU — same order as the BinaryExpr surface this PR closes. Closing one door, leaving 4+ open.

  **Fix:** apply children-span pattern to each recursive `Pos`/`End` case. ~10 LOC per case, same ParenExpr gating considerations. Pin each with a span test. This PR or follow-up; writeup needs correcting either way.
  </details>

- **[Span.End drops 1 col when Y is ParenExpr]** [@thehowl](https://github.com/gnolang/gno/pull/5648#pullrequestreview-4320920070) [`go2gno.go:295`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno.go#L295) — gate only addresses X side; same drift on Y.
  <details><summary>details</summary>

  **Shape:** `const x = 1 + 2 + (3 + 4)` — `gon.X = (1+2)` is BinaryExpr (gate fires), `gon.Y = (3+4)` is `ParenExpr`.

  **Mechanism:** `Go2Gno` unwraps `ParenExpr` at [`go2gno.go:264-265`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno.go#L264-L265) (`return toExpr(fs, gon.X)`), returning inner BinaryExpr. Inner's Span.End sits at column-after-`4`, not after `)`. Fast path reads `bx.Right.GetSpan().End` → loses `)`.

  **Result:**
  - Pre-fix: outer `Span.End` = col 26 (after `)`).
  - Post-fix: outer `Span.End` = col 25 (after `4`).

  `TestPR5648_RightmostParen` confirms: outer Span `3:11-25` post-fix, was `3:11-26`. Same shape `1 + 2 + (3 + 4 + 5)` → `3:11-29`.

  **Impact:** `Span.End` feeds [`Span.Compare`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/nodes_location.go#L210) (sort) and [`Span.Num`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/nodes_location.go#L99) (collision). 1-col-shorter End can flip ordering vs sibling whose End is after `)`. Error location strings shift too.

  **Why silent:** no column-sensitive fixture covers rightmost-paren shape.

  **Fix:**
  1. Widen gate: also fall back when `gon.Y` is `*ast.ParenExpr` — one extra type assertion, keeps O(N) for pure-leaf-Y (the adversarial shape).
  2. Use `SpanFromGo(fs, gon.Y).End` for `rspan.End` when Y unwrappable — keeps fast path, pays one O(1) call.
  </details>

- **[bug can come back invisibly]** [`go2gno.go:294`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno.go#L294) — defensive `if` check protects against an impossible case, but if it ever does happen the code silently runs the O(N²) path again.
  <details><summary>details</summary>

  The fast path is wrapped in `if !lspan.IsZero() && !rspan.IsZero() { ... }`. Today the `if` is always true: every translated child already gets a non-zero Span from `setSpan` at [`go2gno.go:237-239`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno.go#L237-L239), which runs deferred at the top of every `Go2Gno` call. But that's an assumption — if a future change to `setSpan` makes it false, the code silently falls back to the slow path and the original O(N²) bug returns. Nothing tests, nothing warns.

  Fix: drop the `if`. If the assumption breaks one day, let `SetSpan` panic loudly rather than degrade silently. Same applies to the `bx.Left != nil && bx.Right != nil` clause — also impossible for valid Go ASTs.
  </details>

## Nits

- [`bench_parse_test.go:43`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/bench_parse_test.go#L43) — `m.Release()` inside timed loop. Probably intended; one-line comment confirming.
- [`go2gno_span_test.go:34`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno_span_test.go#L34) — "End is exclusive" reads cleaner as "one past the last column".
- [`pr5648_spanfromgo_quadratic.md:158`](../../../../../.worktrees/gno-review-5648/gnovm/adr/pr5648_spanfromgo_quadratic.md#L158) — PR body link uses branch ref → 404 after merge. Use path-relative.

## Missing Tests

- **[covers W#2]** [`go2gno_span_test.go`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno_span_test.go) — `a + b + (c + d)`.
  <details><summary>details</summary>

  No fixture covers rightmost `ParenExpr`. Lock in new semantics or surface regression. Shape in [`tests/zz_review_adversarial_test.go`](tests/zz_review_adversarial_test.go).
  </details>

- **[depth-coverage gap]** [`go2gno_span_test.go`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno_span_test.go) — intermediate node Spans in a deep chain.
  <details><summary>details</summary>

  Existing tests assert only the outermost Span. N-1 intermediate BinaryExpr nodes go unchecked. A Left/Right ordering off-by-one would slip past. Walk the tree on `1+2+3+4+5`, assert each subtree's Span matches `SpanFromGo` on the original Go AST.
  </details>

- **[mixed-precedence shape]** [`go2gno_span_test.go`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno_span_test.go) — `a + b * c + d`.
  <details><summary>details</summary>

  Parses `((a + (b*c)) + d)` — outer `gon.X` is BinaryExpr, but `b*c` is a non-chain subtree. Assert Span on outer and middle.
  </details>

## Suggestions

- [`go2gno.go:295`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/go2gno.go#L295) — use `Span4` constructor.
  <details><summary>details</summary>

  `Span4(lspan.Pos.Line, lspan.Pos.Column, rspan.End.Line, rspan.End.Column)` matches helper at [`nodes_location.go:103`](../../../../../.worktrees/gno-review-5648/gnovm/pkg/gnolang/nodes_location.go#L103).
  </details>

- [`pr5648_spanfromgo_quadratic.md`](../../../../../.worktrees/gno-review-5648/gnovm/adr/pr5648_spanfromgo_quadratic.md) — flag rightmost-paren shape in future audit.
  <details><summary>details</summary>

  Future child-span optimizations on `CallExpr`/`IndexExpr`/etc. need both-ends thinking. W#2's drift generalizes to them.
  </details>

## Questions for Author

- Y-side ParenExpr column shift: intentional, or unexercised by audited fixtures? Rationale only covers `gon.X` shapes. If intentional, document; if not, widen gate.
