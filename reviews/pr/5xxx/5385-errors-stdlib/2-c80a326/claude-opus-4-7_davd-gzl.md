# PR #5385: feat(gnovm): add `errors.Unwrap`, `errors.Is`, and `errors.Join` to stdlib

URL: https://github.com/gnolang/gno/pull/5385
Author: davd-gzl | Base: master | Files: 16 | +486 -73
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `c80a326` (stale — +191 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5385 c80a326`

**Verdict: APPROVE with nits** — Implementation is a faithful port of Go's `errors` package, tests pass, the consumer migrations look right, and the package fills a real hole (closes #486). Open items are doc accuracy (`breadth first` claim vs depth-first impl, stale `Is` doc comment about "Unwrap on either"), the missing `errors.As` row in `docs/resources/go-gno-compatibility.md`, and a couple of `wrap_test.gno` coverage gaps inherited from the Go stdlib test table. None block merge.

## Summary

Ports `errors.Unwrap`, `errors.Is`, and `errors.Join` from Go's stdlib, the last building blocks (`errors.As` aside, which needs `reflect`) for idiomatic error wrapping in Gno. The `Is` traversal is byte-for-byte identical to Go's `is()` function, with the only divergence being how target comparability is determined: since Gno lacks `reflect`, `comparable(v error)` does `_ = v == v` under `defer/recover` to probe at runtime. Consumer migrations (`uassert.ErrorIs`, `encoding/csv`, `fmt/errors_test`, `strconv/atoi_test`) drop hand-rolled `Unwrap`/`isErr` shims and un-gate one previously-skipped test. The PR also reveals (and corrects) several pre-existing test bugs in `examples/` that were silently passing under the old `uassert.ErrorIs` — that function's `err == nil || target == nil` branch returned a bool without calling `fail()`, so any test where the implementation returned `nil` instead of the expected error passed by accident.

## Glossary

- `comparable(v error)` — Gno-specific helper that probes comparability via `defer/recover` around `_ = v == v`, replacing Go's `reflectlite.TypeOf(target).Comparable()`.
- `joinError` — internal type implementing `Unwrap() []error`; what `errors.Join` returns when it has at least one non-nil error.
- Sentinel error — a package-level `var ErrFoo = errors.New("foo")` used as the canonical comparison target with `errors.Is`.

## Fix

Three new files under [`gnovm/stdlibs/errors/`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/): `wrap.gno` (Unwrap, Is, comparable), `join.gno` (Join, joinError), and matching `_test.gno` files copied from Go's stdlib test tables. The `Is` body matches Go's exactly modulo the comparable check ([`gnovm/stdlibs/errors/wrap.gno:41-48`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L41-L48) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L41-L48)). The package-level doc in `errors.gno` is rewritten to drop the `errors.As` example and add a one-line note that `As` is omitted pending `reflect` support ([`gnovm/stdlibs/errors/errors.gno:38-39`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/errors.gno#L38-L39) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/errors.gno#L38-L39)). `uassert.ErrorIs` is rewritten to delegate to `errors.Is` and call `fail()` on mismatch in *all* cases ([`examples/gno.land/p/nt/uassert/v0/uassert.gno:66-76`](https://github.com/gnolang/gno/blob/c80a32609/examples/gno.land/p/nt/uassert/v0/uassert.gno#L66-L76) · [↗](../../../../../.worktrees/gno-review-5385/examples/gno.land/p/nt/uassert/v0/uassert.gno#L66-L76)) — this is what surfaces the prior false-positive tests.

## Critical (must fix)

None.

## Warnings (should fix)

- **[doc says breadth-first, code is depth-first]** [@claude-sonnet-4-6](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5385-errors-stdlib/1-c80a326/claude-sonnet-4-6_davd-gzl.md) [`gnovm/stdlibs/errors/wrap.gno:27`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L27) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L27) — Doc comment claims `breadth first traversal`, implementation is depth-first.
  <details><summary>details</summary>

  The doc says "When err wraps multiple errors, Is examines err followed by a breadth first traversal of its children." But the `Unwrap() []error` branch at [`wrap.gno:64-70`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L64-L70) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L64-L70) recurses on each child immediately — classic depth-first. Go's current stdlib comment at `/usr/lib/go/src/errors/wrap.go:32` says "depth-first traversal." The comment was likely copied from an older Go version that had the same bug; either way it now misleads readers about the visit order of joined errors. Fix: replace `breadth first` with `depth-first` to match the implementation and current Go.
  </details>

- **[stale doc claim — `An Is method should only shallowly compare err and the target and not call Is on either`]** [`gnovm/stdlibs/errors/wrap.gno:39-40`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L39-L40) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L39-L40) — Copy-pasted from old Go docs; Go updated this to `not call [Unwrap] on either`.
  <details><summary>details</summary>

  The advice "An Is method should only shallowly compare err and the target and not call Is on either" is the older Go phrasing. Current Go 1.26 says "not call [Unwrap] on either" — which is the actual rule (calling `Is` recursively is fine and sometimes the whole point; calling `Unwrap` from inside `Is` causes double-traversal). The Gno port should match the corrected guidance. Fix: change to `not call Unwrap on either`.
  </details>

- **[compatibility doc stale]** [@claude-sonnet-4-6](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5385-errors-stdlib/1-c80a326/claude-sonnet-4-6_davd-gzl.md) [`docs/resources/go-gno-compatibility.md:164`](https://github.com/gnolang/gno/blob/c80a32609/docs/resources/go-gno-compatibility.md#L164) · [↗](../../../../../.worktrees/gno-review-5385/docs/resources/go-gno-compatibility.md#L164) — `errors` row still marked `part`; only `As` is missing now.
  <details><summary>details</summary>

  With `Unwrap`, `Is`, and `Join` added, `errors` is functionally complete except for `As`. Either keep `part` and add a footnote (`only errors.As missing, blocked on reflect`), or split the row. Decision is the maintainer's, but the row should not be left as a generic `part` when the gap is one well-defined function. Fix: add a footnote pointing at the `As`/reflect dependency.
  </details>

- **[orphan test cases dropped from Go's `TestIs` table]** [`gnovm/stdlibs/errors/wrap_test.gno:31-58`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap_test.gno#L31-L58) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap_test.gno#L31-L58) — Six rows from Go's upstream table are missing.
  <details><summary>details</summary>

  Comparing Gno's `wrap_test.gno` to `/usr/lib/go/src/errors/wrap_test.go:32-62`, six rows are dropped: `{nil, err1, false}`, `{&errorUncomparable{}, err1, false}`, `{multiErr{poser}, err1, true}`, `{multiErr{poser}, err3, true}`, `{multiErr{nil}, nil, false}`, and `{errorUncomparable{}, &errorUncomparable{}, false}` is present but `{&errorUncomparable{}, &errorUncomparable{}, false}` (different shape) is too. Two rows that don't exist upstream were added: `{errorUncomparable{}, nil, false}`, `{nil, errorUncomparable{}, false}` (these are fine). The missing `{nil, err1, false}` and `{multiErr{nil}, nil, false}` are the most load-bearing — they exercise the asymmetric-nil and nil-inside-multierr paths that have caused real bugs in Go's history. Fix: copy the missing rows verbatim from Go's `wrap_test.go`.
  </details>

## Nits

- [`gnovm/stdlibs/errors/wrap.gno:77-88`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L77-L88) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L77-L88) — `comparable` helper name shadows the Go 1.18+ built-in constraint identifier. Not a Gno conflict today (Gno doesn't have generics), but a future generics rollout might collide. Renaming to `isComparable` (matches the local var name at [`wrap.gno:46`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L46) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L46)) sidesteps it.

- [`gnovm/stdlibs/errors/join.gno:40-49`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/join.gno#L40-L49) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/join.gno#L40-L49) — Missing the `len == 1` fast-path that Go's `joinError.Error()` has at `/usr/lib/go/src/errors/join.go:48-50`. Saves one slice-build for the single-error case. Output is identical either way.

- [`gnovm/stdlibs/errors/wrap.gno:41-48`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L41-L48) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L41-L48) — Go's `Is` short-circuits on `err == nil || target == nil`; Gno only short-circuits on `target == nil`. For `err == nil && target != nil` the Gno path calls `comparable(target)` and enters `is()` for nothing. Behaviorally identical (`default` branch returns false), but adds a needless `defer/recover` per call. One line: `if err == nil { return false }` after the `target == nil` check.

- [`gnovm/stdlibs/errors/join.gno:51-53`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/join.gno#L51-L53) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/join.gno#L51-L53) — `Unwrap() []error` returns the internal slice directly, so a caller can mutate it (`joined.Unwrap()[0] = somethingElse`). Same as Go's behavior, but in a realm-storage context a one-line "treat the returned slice as read-only" doc hint is cheap insurance.

- [`gnovm/stdlibs/errors/README.md:20-21`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/README.md#L20-L21) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/README.md#L20-L21) — "Comparability of error values is determined at runtime via recover" is accurate but doesn't say *why a caller would care*. One sentence on the gas/behavior cost (`every Is call with a non-nil target pays one defer/recover round-trip`) would help library authors decide whether to cache the result.

## Missing Tests

- **[no Is test for pure-non-comparable target without an `Is` method]** [`gnovm/stdlibs/errors/wrap_test.gno`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap_test.gno) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap_test.gno) — The existing `errorUncomparable` defines an `Is` method, so `targetComparable == false → no Is method → default → false` is never exercised.
  <details><summary>details</summary>

  I confirmed by running an adversarial filetest in the worktree (see Repro below): the recover path works correctly under GnoVM and returns `false` without panicking. But the in-tree test suite doesn't cover this branch; if a future change to `comparable()` (e.g. swapping `_ = v == v` for something subtler) breaks the recover, no test would catch it. Fix: add `{nonComparableNoIs{}, nonComparableNoIs{}, false}` and `{nonComparableNoIs{}, err1, false}` where `nonComparableNoIs` is a struct with a slice field and no `Is` method.

  **Repro:**

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5385 -R gnolang/gno
  cat > gnovm/tests/files/errors_uncomparable_noIs.gno <<'EOF'
  package main

  import (
  	"errors"
  	"fmt"
  )

  type noIsErr struct{ tags []string }

  func (noIsErr) Error() string { return "no-is" }

  func main() {
  	var e error = noIsErr{tags: []string{"a"}}
  	var t error = noIsErr{tags: []string{"a"}}
  	fmt.Println("Is(e, t):", errors.Is(e, t))
  }

  // Output:
  // Is(e, t): false
  EOF
  go test -v -run 'TestFiles/errors_uncomparable_noIs.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/errors_uncomparable_noIs.gno
  ```

  Result on c80a32609: `PASS`. The `comparable()` recover path catches the panic from `_ = v == v` and `Is` returns `false`. The point of the suggested test row is to lock this behavior in.
  </details>

- **[no test for deeply nested `Join`]** [`gnovm/stdlibs/errors/join_test.gno:80-94`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/join_test.gno#L80-L94) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/join_test.gno#L80-L94) — `TestJoinIs` covers one layer of `Join`; nested `Join(Join(a, b), c)` traversal isn't tested.
  <details><summary>details</summary>

  Adversarial run in the worktree confirmed correct behavior: `Is(Join(Join(a, b), c), a) == true`. But the test suite doesn't pin it. Fix: add a `TestJoinIsNested` covering `Join(Join(a, b), c)` with `Is` against each of `a`, `b`, `c`, plus a `Join(joined, joined)` self-reference case.
  </details>

- **[`uassert.ErrorIs` doesn't pin its own `nil`/`nil` and `nil`/`non-nil` semantics]** [`examples/gno.land/p/nt/uassert/v0/uassert_test.gno`](https://github.com/gnolang/gno/blob/c80a32609/examples/gno.land/p/nt/uassert/v0/uassert_test.gno) · [↗](../../../../../.worktrees/gno-review-5385/examples/gno.land/p/nt/uassert/v0/uassert_test.gno) — The PR rewires `ErrorIs` through `errors.Is`, which changes the `err == nil, target != nil` semantics from `silent false` to `fail`. No test in `uassert` locks this new contract.
  <details><summary>details</summary>

  This is the load-bearing behavior change in the consumer migration — the reason `events_test.gno`, `grc721_metadata_test.gno`, and `grc721_royalty_test.gno` all needed touching. If a future refactor of `uassert.ErrorIs` accidentally restores the silent-false behavior, none of the rewritten tests would catch it (they'd just go back to silently passing wrong assertions). Fix: add a `TestErrorIs_NilActualNonNilExpected_FailsTest` covering the bug shape directly.
  </details>

## Suggestions

- [`examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno:26-31`](https://github.com/gnolang/gno/blob/c80a32609/examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno#L26-L31) · [↗](../../../../../.worktrees/gno-review-5385/examples/gno.land/p/demo/tokens/grc721/grc721_metadata.gno#L26-L31) — The PR's `grc721_metadata_test.gno` rewrite reveals (correctly, by removing false assertions) that `SetTokenMetadata` validates neither token existence nor ownership.
  <details><summary>details</summary>

  This is not a PR-5385 problem — the bug pre-exists and the previous reviewer flagged it. But the PR is the *first artifact that surfaces it in code review terms*, because before c80a32609 the false assertions hid the issue. Suggestion: open a follow-up issue against `grc721` flagging the missing `_, found := s.extensions.Get(tid.String())` check and the missing ownership gate, and link it from the test comment instead of just `Note: SetTokenMetadata does not validate ...`. As-is, the test comment reads like accepted behavior; it should read like a pending bug.
  </details>

- [`gnovm/stdlibs/errors/wrap.gno:23-40`](https://github.com/gnolang/gno/blob/c80a32609/gnovm/stdlibs/errors/wrap.gno#L23-L40) · [↗](../../../../../.worktrees/gno-review-5385/gnovm/stdlibs/errors/wrap.gno#L23-L40) — Doc could mention that, unlike Go, the Gno `Is` *tolerates* a non-comparable target (returns false instead of panicking).
  <details><summary>details</summary>

  Go's doc says "The target must be comparable" — passing a non-comparable target panics. Gno's implementation defends against this via `comparable()` and returns false. That's an improvement, but only if documented; otherwise users porting Go code will assume the panic semantics still apply and write defensive type checks. One sentence in the doc comment: `Unlike Go, target may be non-comparable; comparability is probed at runtime and a non-comparable target without an Is method matches nothing.`
  </details>

- ADR — per [`AGENTS.md`](https://github.com/gnolang/gno/blob/c80a32609/AGENTS.md) · [↗](../../../../../.worktrees/gno-review-5385/AGENTS.md), non-trivial AI-assisted PRs require an ADR. Even a short one (~30 lines) covering the `comparable()` design choice, the `errors.As` omission rationale, and the `joinError` shape decision would be useful for the next reviewer.

## Questions for Author

- Why are six rows from Go's `TestIs` table missing (`{nil, err1, false}`, `{multiErr{nil}, nil, false}`, etc.)? Deliberate trim or oversight when porting?
- Is the `breadth first` doc comment a copy of older Go docs, or an intentional reword? Code says depth-first; one of the two needs to change.
- Was the gas overhead of `comparable()`'s `defer/recover` measured? For a hot path realm using `errors.Is` heavily, this is a per-call cost Go doesn't pay (Go uses `reflectlite.TypeOf(target).Comparable()` which is essentially free).
- Should `Is` accept a `nil` `err` with non-nil `target` and short-circuit (matches Go for one extra line)? Functionally identical today, but saves a recover round-trip on every nil-error check.
