# PR #4699: feat(examples): straw poll realm

URL: https://github.com/gnolang/gno/pull/4699
Author: audrenbdb | Base: master | Files: 6 | +426 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — won't compile against current master (imports `std`, `gno.land/p/demo/avl`, `gno.land/p/demo/ufmt` — all renamed by the std-split #4040 and `p/demo` → `p/nt` reorg merged after this PR's last commit), and `Vote` has an unbounded lower-index check that panics on `pollNumber <= 0`. Rebase + lower-bound guard, then this is a clean small example.

## Summary

Adds a minimal straw-poll example: a `p/demo/strawpoll` package holding the `Poll` type (creator, question, options, two `avl.Tree`s for `votes` and `voters`) and a `r/demo/strawpoll` realm wrapping it with `CreatePoll`/`Vote` entry points and a piechart-rendered `Render`. Total 426 lines including tests. The PR has been stale since Sep 2025 and gathered two maintainer requests to rebase ([@leohhhn](https://github.com/gnolang/gno/pull/4699#issuecomment-3360225891), [@jefft0](https://github.com/gnolang/gno/pull/4699#issuecomment-2772116562)); the rebase is the actual blocker, not the design.

## Glossary

- std split (#4040) — merged on master after this PR; `std` is no longer a single stdlib. `PreviousRealm`/`Address`/etc. moved to `chain/runtime`, `chain` (for `Address`), `chain/banker`, etc.
- `p/demo` → `p/nt` rename — `avl`, `ufmt`, and several other "primitive" packages were moved from `gno.land/p/demo/` to `gno.land/p/nt/` (see `examples/gno.land/p/nt/avl`, `examples/gno.land/p/nt/ufmt`).
- crossing function — `func fn(cur realm, ...)`; called via `fn(cross, ...)` to perform an explicit realm boundary cross. The realm's `CreatePoll`/`Vote` here use the `_ realm` form (anonymous receiver).

## Fix

Splits a straw-poll example across `p` and `r` per the refactor suggestion in [the second commit thread](https://github.com/gnolang/gno/pull/4699#issuecomment-3232149562): `p/demo/strawpoll.Poll` holds state + validation (min 2 / max 10 options, duplicate-voter guard), `r/demo/strawpoll` holds a package-global `polls []Poll`, two `txlink.NewLink` URLs for the Render action menu, and the `CreatePoll`/`Vote`/`Render` entrypoints. Render uses the existing `p/samcrew/piechart` package to produce an SVG per poll.

## Critical (must fix)

- **[doesn't compile on master]** [`examples/gno.land/p/demo/strawpoll/strawpoll.gno:5-9`](../../../../../.worktrees/gno-review-4699/examples/gno.land/p/demo/strawpoll/strawpoll.gno#L5-L9) — imports `std`, `gno.land/p/demo/avl`, `gno.land/p/demo/ufmt`; none of those resolve on current master.
  <details><summary>details</summary>

  `std` was split by [#4040 `feat(stdlibs)!: std split`](https://github.com/gnolang/gno/pull/4040) — `gnovm/stdlibs/std` no longer exists. `std.PreviousRealm()` and `std.Address` live under `chain/runtime` and `chain` respectively now (see `examples/gno.land/p/demo/microblog/microblog.gno:3-10` which uses `"chain/runtime"` and `"gno.land/p/nt/avl"` instead). Both `gno.land/p/demo/avl` and `gno.land/p/demo/ufmt` were moved to `gno.land/p/nt/avl` and `gno.land/p/nt/ufmt` — only the `nt/` paths exist under `examples/gno.land/p` on master. CI confirms: `Run gno test`, `Run gno lint`, `gno2go`, `mod-tidy` and the GnoVM/gno.land Go suites all fail. Repro below. Fix: rebase on master, switch imports to `chain/runtime` (for `PreviousRealm`/`Address`), `gno.land/p/nt/avl`, `gno.land/p/nt/ufmt`. Same three import lines in [`r/demo/strawpoll/strawpoll.gno:4-12`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll.gno#L4-L12) need the same treatment (no `std` there, but `p/demo/strawpoll` itself transitively breaks).

  Repro:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 4699 -R gnolang/gno
  go install ./gnovm/cmd/gno
  cd examples/gno.land/p/demo/strawpoll
  gno test -v .
  # expect: package std is not in std (...)
  ```
  </details>

- **[panics on pollNumber <= 0]** [`examples/gno.land/r/demo/strawpoll/strawpoll.gno:37-47`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll.gno#L37-L47) — `Vote(cross, 0, ...)` or any negative `pollNumber` panics with index-out-of-range instead of returning `ErrPollNotFound`.
  <details><summary>details</summary>

  `pollIdx := pollNumber - 1` followed only by `if pollIdx >= int64(len(polls))` — there is no lower-bound check. `Vote(cross, 0, "x")` gives `pollIdx = -1`, passes the upper bound (since `-1 < len(polls)`), then hits `&polls[-1]` which panics. A panic across the realm boundary aborts the program in Gno (see `docs/resources/gno-interrealm.md:233-239`), so this is an uncaught panic any external caller can trigger by passing `0`. Fix: change the guard to `if pollIdx < 0 || pollIdx >= int64(len(polls))`. The test [`r/demo/strawpoll/strawpoll_test.gno:56-61`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll_test.gno#L56-L61) only covers `999` (too large), not `0`.
  </details>

## Warnings (should fix)

- **[options containing commas are silently split]** [`examples/gno.land/r/demo/strawpoll/strawpoll.gno:25-28`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll.gno#L25-L28) — `strings.Split(choices, ",")` means an option like `"Apple, Banana or Cherry"` becomes three options.
  <details><summary>details</summary>

  The realm signature `CreatePoll(_ realm, question string, choices string)` is constrained by gnoweb's `$help&func=CreatePoll&question=...&choices=...` URL form, which is single-string-per-arg. The comma delimiter is convenient but means polls can't have options that contain commas — and there is no way for the caller to escape, so the result is silent option-list mangling, not an error. Two options: (1) document the limitation in the doc-comment on `CreatePoll`, and trim each option with `strings.TrimSpace` so leading whitespace doesn't make `"Foo, Bar"` produce `"Foo"` + `" Bar"` (a distinct option from `"Bar"`); (2) switch to a less collision-prone delimiter (e.g. `|` or `;`) — still imperfect but rarer in natural text. Either way the function should reject empty options after splitting (`strings.Split(",foo", ",")` yields `["", "foo"]` which currently becomes a valid empty-string option). Today, two empty options also pass validation since min/max only counts length, not content — and a voter could then pass `""` to `Vote`.
  </details>

- **[no max-polls cap → unbounded realm state]** [`examples/gno.land/r/demo/strawpoll/strawpoll.gno:16,31`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll.gno#L16) — `polls` grows without bound; anyone can create polls forever, and each poll carries two `avl.Tree`s that grow per voter.
  <details><summary>details</summary>

  For an example realm this is borderline acceptable (demos don't have to be production-hardened), but the `maximumOptions = 10` cap in `p/demo/strawpoll` shows the author is conscious of bounding state. Adding a `maxPolls` constant (e.g. 1000) with `ErrTooManyPolls` keeps the example consistent and is the kind of thing that gets copy-pasted into real realms. Note also that polls are never deleted: there's no `ClosePoll`/`DeletePoll`/`Cleanup`. At minimum, document the unbounded growth or add a cap. Render already only shows the last 3, so capping at e.g. 100 polls would not change the UX.
  </details>

- **[realm test pollutes package-global state]** [`examples/gno.land/r/demo/strawpoll/strawpoll_test.gno:10-83`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll_test.gno#L10-L83) — `polls` is package-level and shared across all subtests; `TestCreatePoll/Success` leaves one poll behind, `TestVote/Success` adds another. Order-dependent.
  <details><summary>details</summary>

  Today the tests happen to pass because `TestVote/PollNotFound` uses `999` (not "first + 1") and `TestVote/Success` uses `int64(len(polls))` (relative). If anyone reorders or adds tests assuming a clean slate, this silently breaks. Fix: either reset `polls = polls[:0]` (or `make([]strawpoll.Poll, 0)`) in a `t.Cleanup` or at the top of each subtest, or factor a `t.Run` helper that creates a fresh local poll list and operates on it. The `p` package tests sidestep this by creating a fresh `Poll` per subtest, which is the right pattern.
  </details>

## Nits

- [`examples/gno.land/p/demo/strawpoll/strawpoll.gno:36-37`](../../../../../.worktrees/gno-review-4699/examples/gno.land/p/demo/strawpoll/strawpoll.gno#L36-L37) — field doc-comments say "A map to store..." but the fields are `*avl.Tree` (renamed in commit `6506bfb` but comments not updated for `votes`/`voters`).
- [`examples/gno.land/p/demo/strawpoll/strawpoll.gno:20`](../../../../../.worktrees/gno-review-4699/examples/gno.land/p/demo/strawpoll/strawpoll.gno#L20) — `ErrMaximumOptionsExceeded` is built with `ufmt.Errorf` at package init for a static string that interpolates a const; a plain `errors.New("a maximum of 10 options...")` is one less import and one less init-time call. Cosmetic.
- [`examples/gno.land/r/demo/strawpoll/strawpoll.gno:17-20`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll.gno#L17-L20) — `createPollLink`/`voteLink` are stored as `string` (`.URL()` is called at init). They're computed once, fine, but the example pre-fills `pollNumber=1` and `choice=Orange` which assume context that may not exist by the time someone clicks. Consider linking to the latest existing poll (`len(polls)`) in `voteLink`, or dropping the example value.
- [`examples/gno.land/p/demo/strawpoll/strawpoll.gno:24-28`](../../../../../.worktrees/gno-review-4699/examples/gno.land/p/demo/strawpoll/strawpoll.gno#L24-L28) — `pieColors` is a package-level slice of 10 hex strings — works, but a `var pieColors = [...]string{...}` (array) signals "fixed palette" more clearly.
- `examples/gno.land/r/demo/strawpoll/strawpoll.gno:1` — doc-comment says "package strawpoll **eases**…"; "is a small example realm for creating polls" reads less marketing-y. Tiny.

## Missing Tests

- **[Vote lower-bound]** [`examples/gno.land/r/demo/strawpoll/strawpoll_test.gno:56-61`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/strawpoll/strawpoll_test.gno#L56-L61) — no test for `Vote(cross, 0, ...)` or negative `pollNumber`. Would have caught the Critical above.
- **[invalid option name]** `examples/gno.land/r/demo/strawpoll/strawpoll_test.gno:55-83` — no test asserting `ErrOptionNotAvailable` from the realm side (only from the `p` package side).
- **[duplicate-voter across polls]** `examples/gno.land/p/demo/strawpoll/strawpoll_test.gno:74-97` — `voters` is per-poll, so voting in poll A then poll B should succeed. No explicit test pins that invariant; if someone later refactors to a global voter set, nothing catches the regression.
- **[empty-string option after split]** `examples/gno.land/r/demo/strawpoll/strawpoll_test.gno:10-53` — no test for `CreatePoll(cross, "q", ",foo")` or `"foo,"`; both currently produce a poll with an empty-string option.

## Suggestions

- `examples/gno.land/r/demo/strawpoll/strawpoll.gno:36-47` — once the bound bug is fixed, consider returning the new pollID from `CreatePoll` (currently returns only `error`). Without it the caller has to call `Render` and parse to find out which number their poll got; with it, they can deep-link into Vote immediately.
  <details><summary>details</summary>

  Signature would become `func CreatePoll(_ realm, question string, choices string) (int64, error)` returning `int64(len(polls))` on success. Matches the convention in [`examples/gno.land/r/demo/microblog/microblog.gno`](../../../../../.worktrees/gno-review-4699/examples/gno.land/r/demo/microblog/microblog.gno) and is what most "I want to share my poll" flows need.
  </details>

- `examples/gno.land/p/demo/strawpoll/strawpoll.gno:50-77` — `New()` returns `Poll` by value; storing it in a slice (`polls = append(polls, poll)`) then taking `&polls[pollIdx]` works but is fragile if a future refactor passes the value around. Returning `*Poll` from `New()` and storing `[]*Poll` would be more idiomatic.
  <details><summary>details</summary>

  Today this works because `votes`/`voters` are pointer-typed (`*avl.Tree`), so the value copy still shares the underlying tree. But anyone who copies `Poll` after a few votes and then writes to one copy's `creator`/`question`/`options` will silently desync against the canonical one in `polls`. `*Poll` makes "this is a single mutable entity" explicit and removes the gotcha. Nothing in the diff strictly forces this, so it's a Suggestion rather than a Warning.
  </details>

## Questions for Author

- Is there a reason to keep `creator` private with no accessor? `Render` doesn't surface it today, but for an example realm showing "your address" near results is a common pattern.
- Did you consider supporting poll closing / expiry (block-height or timestamp gated)? Not necessary for a demo, but pattern-shapes other realms will copy.
