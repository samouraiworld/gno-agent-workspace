# Audit: porting Go 1.21's `min`/`max` builtins to Gno

**Status:** WIP branch exists, blocked on a separate language-rule decision.
**WIP code:** `.worktrees/gno-feat-min-max-builtins/` (branch `feat/min-max-builtins`).
**Reference PR (parent context):** [#5753](https://github.com/gnolang/gno/pull/5753) — the stdlib catch-up PR that motivated this work (regexp/syntax `calcFlags` had to inline `if` clamps instead of `max(minFold, x)` / `min(maxFold, y)`).

---

## TL;DR

The actual `min`/`max` builtin implementation is small (~280 LOC across `uverse.go` + `preprocess.go` + 8 filetests) and passes its 8 new tests. **The blocker is unrelated:** Gno enforces "builtin identifiers cannot be shadowed" *globally* (not at package scope like Go does), so registering any new uverse name breaks every existing local variable / parameter / func of the same name across the entire stdlib + examples corpus.

The shadow rule should be fixed first — it's a Go-spec divergence with no clear motivation, and once fixed `min`/`max` lands with zero stdlib churn.

---

## Why `min`/`max` matter

- Go 1.21 builtins. Among the most-used post-1.17 additions in real Go code.
- Block verbatim upstream ports of any stdlib code that uses them. First concrete case: `regexp/syntax.calcFlags` in PR #5753 had to use explicit `if` clamps:

  ```go
  // upstream Go 1.22:
  lo := max(minFold, re.Rune[i])
  hi := min(maxFold, re.Rune[i+1])

  // Gno today:
  lo := re.Rune[i]
  if lo < minFold { lo = minFold }
  hi := re.Rune[i+1]
  if hi > maxFold { hi = maxFold }
  ```

  This divergence multiplies across every future upstream port.

- Constant folding (`const x = min(1, 2)`) makes them useful even where a generic library function couldn't suffice.

---

## Current state in Gno

- **No open PR.** No issue. Confirmed via `gh pr list -R gnolang/gno --search 'min max builtin'` and `gh issue list -R gnolang/gno --search 'min max builtin'`.
- **Compatibility doc** ([gno/docs/resources/go-gno-compatibility.md:107](../../gno/docs/resources/go-gno-compatibility.md#L107)) lists `builtin` as `full` with footnote noting "all functions up to Go 1.17 exist" — i.e., 1.21+ additions are explicitly excluded.
- **Common misconception:** "generics would replace builtin min/max." False. Even upstream Go implements them as compiler builtins (see [cmd/compile/internal/ssagen/ssa.go](https://github.com/golang/go/blob/master/src/cmd/compile/internal/ssagen/ssa.go)). The generic signature in `src/builtin/builtin.go` is documentation-only. Reasons: constant folding, untyped-constant inference, spec compliance. Generics wouldn't remove the need for builtin-recognition.

---

## Findings

### F1 — Global shadow rule blocks all new builtins (HIGH)

Gno enforces "builtin identifiers cannot be shadowed" at the package level, not just at the file/global declaration level.

Sites:
- [gno/gnovm/pkg/gnolang/preprocess.go:5387](../../gno/gnovm/pkg/gnolang/preprocess.go#L5387) — `predefineRecursively2` rejects redeclaration of any uverse name.
- [gno/gnovm/pkg/gnolang/preprocess.go:5893](../../gno/gnovm/pkg/gnolang/preprocess.go#L5893) — equivalent check on lookups.

Go's behavior: shadowing builtins at function/block scope is legal — `min := 5` inside a function just hides the builtin in that scope. Gno's check is stricter than the spec.

Impact when adding `min`/`max`:
- 70+ files in stdlib + examples currently use `min` or `max` as local var, param, or func names. Each breaks with "builtin identifiers cannot be shadowed: max".
- Concrete sites identified (non-exhaustive): `gno/gnovm/stdlibs/io/io.gno`, `gno/gnovm/stdlibs/chain/markdown/markdown_test.go`, `gno/gnovm/stdlibs/crypto/chacha20/rand/rand.gno`, `gno/gnovm/stdlibs/time/format.gno` (heavy — many local `min` uses), `gno/gnovm/stdlibs/regexp/all_test.gno`, `gno/gnovm/stdlibs/sort/search_test.gno`, plus several `examples/` files.
- Same problem will recur for *every* future builtin addition (`clear`, `min`, `max`, future Go additions). Permanent recurring tax unless the rule is fixed.

**Recommendation:** before any new builtin lands, change the shadow rule to match Go's spec (file/function scope shadowing allowed; package-scope redeclaration still rejected since that's actually invalid Go too). Estimated change: ~10 LOC in `preprocess.go` plus 2-3 filetests pinning the new behavior. Then min/max lands with zero stdlib churn.

**Alternative (worse):** preserve the strict rule, do a one-shot "rename local shadows" sweep across stdlib+examples before adding any builtins. Mechanical but noisy; doesn't solve the problem for the next builtin.

**Question for the team:** is the global shadow rule load-bearing for something I'm not seeing? It's stricter than Go spec; the reason for the divergence isn't documented in the code or commit history. Worth a brief investigation before changing.

---

### F2 — `math.Min`/`math.Max` ≠ builtin `min`/`max` (semantic divergence)

`math.Min(-0.0, +0.0) == -0.0` (IEEE-754 signed-zero handling).
`min(-0.0, +0.0)` returns either zero — the builtin only NaN-propagates, no ±0/±Inf special-casing.

This matches upstream Go's behavior — the builtins are deliberately simpler than the package functions. But:

- The WIP branch had to rename the internal helpers in `math.gno` to `mathMin`/`mathMax` so the package functions keep their IEEE-754 semantics while the builtin coexists.
- **Future risk:** any stdlib port that mechanically replaces `math.Min` calls with builtin `min` will silently lose the signed-zero/±Inf handling. The compiler won't catch this — both compile fine, both type-check fine, behavior diverges only on edge cases.

**Recommendation:** add a note to `audits/stdlib-test-gap/bugs.md` and to the Gno stdlib porting docs. When porting code that uses `math.Min`/`math.Max`, do NOT replace with builtin `min`/`max` without checking whether the caller relies on signed-zero semantics.

---

### F3 — `isLss` is not gas-free; preprocess-time constant folding nil-derefs (MEDIUM)

[`isLss`](../../gno/gnovm/pkg/gnolang/op_binary.go) calls `m.incrCPU` internally for `StringKind` and bigint paths. Any preprocess-time use (constant folding, lint analysis, future static checks) that doesn't have a `*Machine` available will nil-deref.

The WIP branch worked around this by writing a parallel `tvLssNoGas` helper for the `foldMinMaxConst` path. That's local duplication.

**Recommendation:** either
- Extract a `cmpOrderedNoGas` helper at package level so future constant folders don't make the same mistake, OR
- Make `incrCPU` nil-receiver-safe (cheap nil check, no semantic change).

The second is preferable — it's a one-line guard that prevents an entire category of preprocess-time crashes that nobody's hit yet because no constant folding exists today.

---

### F4 — Preprocessor error messages largely unreachable (LOW, scope question)

Gno's `min`/`max` preprocessor errors (no args, non-ordered type, mismatched types) are caught *first* by Go's typechecker (1.21+) before the preprocessor sees them. Why: Gno's `.gnobuiltins.gno` shim runs through `go/types` at Go 1.25, which knows min/max natively.

Result: user code that's typechecked normally hits Go's error messages, not Gno's. Gno's nicer errors only fire for paths that skip TC (e.g., `MPFiletests` without TC mode, or internal stdlib).

**Recommendation:** worth a separate audit pass on whether Gno's preprocessor error messages should match Go's verbatim where Go also catches the same error. If the user-visible error always comes from `go/types`, Gno's strings are dead text from a UX perspective.

---

### F5 — Implementation viable in ~280 LOC (DONE in WIP)

Approach taken in `.worktrees/gno-feat-min-max-builtins/`:
- **Runtime:** `defNative("min", ...)` and `defNative("max", ...)` in `uverse.go`. Use `Vrd(GenT("T", nil))` to declare the variadic-generic signature; the existing `Specify` machinery resolves `T`. Runtime helper `doUverseMinMax` folds via `isLss` with NaN propagation.
- **Type checking + constant folding:** dedicated special case in `preprocess.go` (alongside `make`/`append`/`copy`/`cross`). Validates ≥1 arg, walks args to compute common type with untyped-constant convergence via `shouldSwapOnSpecificity`, enforces `isOrdered`, calls `checkOrConvertType` per arg, specifies the FuncType, constant-folds when all args are `*ConstExpr`.

Tests: 8 filetests covering int/float (incl. NaN/±Inf)/string/const/mixed-constants/no-args-error/non-ordered-error/mismatched-types-error. All pass.

LOC breakdown:
- `uverse.go`: +171
- `preprocess.go`: +109
- 8 filetests: ~150 LOC

Plus 12 stdlib/example renames the agent could touch (~50 LOC mechanical changes) — these become unnecessary if F1 is fixed.

---

### F6 — Side observation: `shouldSwapOnSpecificity` has counterintuitive semantics (NIT)

[gno/gnovm/pkg/gnolang/preprocess.go](../../gno/gnovm/pkg/gnolang/preprocess.go) — the function comment says "higher specificity has lower value, so backwards." `t1s < t2s` returns true when t1 is *more* specific. Existing callers (e.g. preprocess.go:4759) rely on the "swap" semantic to push the more-specific type onto the less-specific operand.

First-time reader of the function inverts the meaning. Trivial rename to `firstIsMoreSpecific` or invert + rename to `shouldSwap` would prevent that.

---

## Recommended sequencing

1. **Investigate F1.** Decide whether the global shadow rule is load-bearing or accidental. Brief code archaeology — check the commit that introduced it and whether any test pins the strict behavior.
2. **If F1 is accidental:** fix it. ~10 LOC change. Add filetests pinning Go-spec shadow behavior.
3. **Land min/max.** With F1 fixed, the WIP branch reduces to `uverse.go` + `preprocess.go` + 8 filetests — clean review story.
4. **Refactor `regexp/syntax.calcFlags`** in PR #5753 (or follow-up) to use `max`/`min` builtins — finally verbatim upstream.
5. **F3 cleanup:** make `incrCPU` nil-safe or extract `cmpOrderedNoGas`.
6. **F4 audit:** document or fix Gno-vs-Go-typechecker error message divergences.
7. **F2 docs note:** add the `math.Min` vs builtin `min` warning to stdlib porting docs.

---

## What this audit does NOT cover

- Other Go 1.21+ language additions (`clear` builtin, generics, range-over-func, `for` loop var scoping). Each likely has its own findings; out of scope here.
- Performance of the chosen implementation (defNative-based dispatch vs emitted SSA-style nodes). Runtime cost is dominated by the comparisons themselves; benchmarking is premature until F1 is resolved.
- Whether Gno should preserve `min`/`max` as builtins even if generics later land. Per upstream Go's own architecture, yes — see "common misconception" above.
