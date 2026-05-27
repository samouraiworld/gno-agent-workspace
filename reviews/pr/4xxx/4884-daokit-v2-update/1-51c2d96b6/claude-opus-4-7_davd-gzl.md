# PR #4884: feat(daokit): update daokit framework with latest version

URL: https://github.com/gnolang/gno/pull/4884
Author: davd-gzl | Base: master | Files: 65 | +3592 -699
Reviewed by: davd-gzl (AI-assisted, see disclosure) | Model: claude-opus-4-7[1m] | Commit: `51c2d96b6` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4884 51c2d96b6`

Note: PR author and reviewer share a GitHub account (`davd-gzl`). This review is run as an unattended adversarial pass and is **not** a substitute for human maintainer review. Land the merge decision elsewhere.

**Verdict: REQUEST CHANGES** — three latent runtime bugs land on the diff: `crossingDAO.Render` recurses on itself (stack overflow as soon as any external realm uses the crossing wrapper to render), `RoleThreshold.RenderWithVotes` panics with `ufmt: not enough arguments` on every proposal-detail page using that condition, and `GnoloveDAOCondThreshold.RenderWithVotes` formats `float64` with `%d` (literal `%!d(float64=...)` shows on screen). Plus filename typo `cond_role_treshold.gno`, missing ADR for an AI-assisted change, and a duplicate "Threshold needed" line.

## Summary

This PR re-syncs the samouraiworld `daokit` framework (DAOkit) into `examples/gno.land/p/samcrew/`: an upgradable DAO with proposals, conditions, role-based membership, an extension system, and a cross-realm wrapper. ~3600 added lines across four packages (`daokit`, `basedao`, `daocond`, `realmid`) plus three demo realms under `r/samcrew/daodemo/`. The headline new pieces are the `crossingDAO` wrapper enabling cross-realm DAO use, the `Extension` registry, and migration via `ChangeDAOImplementation`. Tests pass, but render-path code paths used in production gnoweb routes have not been exercised — three of them either recurse or panic on the first call. None of these reach state-mutating logic; impact is rendering DoS rather than fund/state risk.

## Glossary

- `daokit` — core p-package (proposals, actions, resources, extensions).
- `basedao` — membership + role manager built on top of `daokit`.
- `daocond` — stateless voting-condition engine (`MembersThreshold`, `RoleCount`, `RoleThreshold`, `GnoloveDAOCondThreshold`, `And`, `Or`).
- `crossingDAO` — wrapper around a `DAO` that routes state-mutating calls through a user-supplied `CrossFn` so the DAO can be used across realms.
- `daoPublic` / `DAOPrivate` — pair of basedao types: `daoPublic` is the exported `daokit.DAO` interface, `DAOPrivate` holds internal state.
- `Extension` — pluggable read-side view object registered into the DAO's AVL extension store.
- `SetImplemFn` — closure invoked when governance approves a DAO-implementation swap (upgrade).

## Fix

The PR is purely additive at the source level (deletes the old monolithic `daodemo.gno` and replaces it with three sub-realms), so there is no before/after to summarize in the usual sense. The conceptual model: a realm calls `basedao.New(conf)`, gets `(localDAO, daoPrivate)`. The realm wraps `localDAO` with `daokit.NewCrossing(localDAO, crossFn)` to obtain a cross-realm-safe handle. State-mutating ops (`Propose` / `Vote` / `Execute`) on the crossing wrapper jump through `crossFn(cross, ...)`. Read-side ops (`Extension`, `ExtensionsList`, `Render`) are intended to pass through without crossing. The implementation of that pass-through is where the load-bearing bug lives ([Critical #1](#critical-must-fix)).

## Critical (must fix)

- **[stack overflow on every external Render call]** [`examples/gno.land/p/samcrew/daokit/crossing.gno:59-61`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daokit/crossing.gno#L59-L61) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daokit/crossing.gno#L59-L61) — `crossingDAO.Render(path)` returns `c.Render(path)`, i.e. itself, not `c.dao.Render(path)`.
  <details><summary>details</summary>

  The method reads:

  ```go
  func (c *crossingDAO) Render(path string) string {
      return c.Render(path)
  }
  ```

  This is unconditional infinite recursion. The current realm code (`simple_dao.Render` etc.) sidesteps it by calling `localDAO.Render(path)` — `localDAO` is the bare `daoPublic`, not the crossing wrapper. The bug therefore does not fire in the bundled demo realms today, which is why the test suite passes. The instant any external realm imports a samcrew DAO and calls the *exported* crossing handle (`var DAO daokit.DAO = daokit.NewCrossing(...)`) for rendering, it stack-overflows. The cross-realm story is exactly what this PR advertises ("Upgradable DAO" + "extension system" with cross-realm calls), so shipping a wrapper that crashes the read path defeats half the feature. Fix: `return c.dao.Render(path)`. Add a test that calls the wrapper, not the inner DAO. Same shape applies to verifying `Extension` and `ExtensionsList` are correctly delegated.
  </details>

- **[every proposal-detail page panics when the condition is `RoleThreshold`]** [`examples/gno.land/p/samcrew/daocond/cond_role_treshold.gno:59`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daocond/cond_role_treshold.gno#L59) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daocond/cond_role_treshold.gno#L59) — three `%`-verbs, two args.
  <details><summary>details</summary>

  The format string is `"%g%% of %s members with role %s must vote yes\n\n"` (three verbs: `%g`, `%s`, `%s`) but it's passed only `(c.threshold*100, c.role)` — `%s` for `c.role` is missing its second argument. `ufmt.Sprintf` panics: `ufmt: not enough arguments`. Reproduced inside the worktree with a one-shot test:

  ```
  panic: ufmt: not enough arguments
      gno.land/p/samcrew/daocond/cond_role_treshold.gno:59
  ```

  Any DAO that uses `daocond.RoleThreshold(...)` as a condition will panic the entire proposal-detail render (`ProposalDetailView` calls `Condition.RenderWithVotes` at [`view_proposal_detail_page.gno:61`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/basedao/view_proposal_detail_page.gno#L61) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/basedao/view_proposal_detail_page.gno#L61)). Fix: drop one of the duplicated subjects so the string matches its args, e.g. `"%g%% of members with role %s must vote yes\n\n"` with `(c.threshold*100, c.role)`. The current sentence reads "X% of admin members with role admin" which is redundant prose anyway. Also add a test that calls `RenderWithVotes` for every condition type.
  </details>

- **[GnoloveDAO render shows `%!d(float64=…)` to users + double-printed line]** [`examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno:78-81`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L78-L81) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L78-L81) — three `%d` verbs receive `float64`, plus a duplicate "Threshold needed" sentence.
  <details><summary>details</summary>

  Lines 78-80 format `c.voteRatio(ballot, …)` (returns `float64` in 0..1) with `%d`. `ufmt` emits the placeholder literal in the output:

  ```
  Yes: %!d(float64=1.000000)/%!d(float64=3.000000)
  No: %!d(float64=0.000000)/%!d(float64=3.000000)
  Abstain: %!d(float64=0.000000)/%!d(float64=3.000000)
  ```

  Captured directly from a one-shot test against this branch. Two problems compound this:

  1. `voteRatio` returns the ratio, not the count — even if the verb is fixed to `%g`, the value displayed is `1.0` (= 100%) not `1` (= one vote). The author probably wanted `totalVote(...)` (which doesn't exist in this condition — `voteRatio` is the only counter). The intent — display vote counts weighted by tier — needs a real helper.
  2. Line 77 prints `"Threshold needed: %g%% of total voting power"` and line 81 prints `"Voting power needed: %g%% of total voting power"` with `c.threshold*totalPower` (a raw voting-power number, not a percentage). The `%%` after `c.threshold*totalPower` is wrong; it'll read e.g. "1.5% of total voting power" when 1.5 is the absolute power, not a percent. The line is also redundant with line 77.

  Fix: rewrite `RenderWithVotes` to compute weighted Yes/No/Abstain power explicitly (sum `votingPowersByTier[tier]` per vote type) and format with `%g`. Drop one of the two threshold lines. Cover with a test that asserts no `%!` escape leaks into the output.
  </details>

## Warnings (should fix)

- **[file misnamed: `treshold` missing the `h`]** [`examples/gno.land/p/samcrew/daocond/cond_role_treshold.gno`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daocond/cond_role_treshold.gno) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daocond/cond_role_treshold.gno) — the symbol is `RoleThreshold`, the file is `cond_role_treshold.gno`.
  <details><summary>details</summary>

  The condition type is `roleThresholdCond`, constructor `RoleThreshold`, but the file is `cond_role_treshold.gno`. Mismatch with `cond_members_threshold.gno` next door. Renaming a file inside a sample p-package is cheap; once published to chain it's stuck. Rename pre-merge.
  </details>

- **[AI-assisted PR missing the ADR mandated by `gno/AGENTS.md`]** [`gno/AGENTS.md`](https://github.com/gnolang/gno/blob/51c2d96b6/AGENTS.md) · [↗](../../../../../.worktrees/gno-review-4884/AGENTS.md) §"Architecture Decision Records" — "Every non-trivial AI-assisted PR must include an ADR."
  <details><summary>details</summary>

  The PR adds ~3600 lines of governance code across four packages plus three demo realms — far above "trivial bug fix" by any reading. PR body credits four authors and there is no `gno.land/adr/pr4884_*.md`. The ADR exists to record the design choices (why `crossingDAO` over plain methods? why `Extension` privacy is a string PkgPath equality check? what guarantees the upgrade path?) — those questions are exactly the kind the human maintainer will ask. Author it before requesting maintainer review, otherwise the bot blocks merge.
  </details>

- **[`UpdateStatus` mutates proposals during a list iteration, author already flagged it]** [`examples/gno.land/p/samcrew/daokit/proposals.gno:85-99`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daokit/proposals.gno#L85-L99) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daokit/proposals.gno#L85-L99) — `// XXX: costly and probably insecure`.
  <details><summary>details</summary>

  `GetProposals` iterates the AVL tree and calls `prop.UpdateStatus()` for each — read endpoints (e.g. `ProposalsView` for any visitor of `:proposals`) flip `ProposalStatusOpen → ProposalStatusPassed` as a side effect of *rendering*. Two consequences worth surfacing now:

  1. Any read view triggers a state write, charging gas to the unlucky reader who happens to be the first to hit the threshold. On a public gnoweb crawl this is unpredictable.
  2. Whether a proposal's status is `Passed` or `Open` becomes time-of-first-render-dependent, not condition-dependent. That breaks the invariant that proposals can only transition under explicit voter action.

  The XXX comment carries no plan. Either separate the rendering-only "would-it-pass" computation from the persisted `Status` (introduce a derived `EffectiveStatus(ballot)`), or eagerly flip status inside `Vote` (when the new vote satisfies the condition). Both are simple; the current shape will cause cross-tx flakiness in production.
  </details>

- **[`ExtensionsStore.Set` return value contradicts its docstring]** [`examples/gno.land/p/samcrew/daokit/extensions.gno:44-47`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daokit/extensions.gno#L44-L47) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daokit/extensions.gno#L44-L47) — comment says "Returns true if the extension was successfully registered"; actually returns `avl.Tree.Set` semantics (true = key was already there).
  <details><summary>details</summary>

  `avl.Tree.Set(k, v)` returns `true` when an *existing* entry was replaced and `false` when a new entry was inserted. The comment above advertises the opposite ("Returns true if successfully registered"). Callers that act on the return value will treat replacements as success and fresh inserts as failure. Fix the comment or invert the return (`!es.Tree.Set(...)`). Same docstring also says "If an extension with the same path already exists, it will be replaced" — silently replacing a registered extension is itself worth a discussion (privacy escalation, registry hijack inside a single realm). At minimum, return whether it was a replacement and let callers decide.
  </details>

- **[passing arbitrary `migrate` closure as a proposal payload is the inverse of trust]** [`examples/gno.land/p/samcrew/basedao/migrate.gno:34`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/basedao/migrate.gno#L34) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/basedao/migrate.gno#L34) — `migrated := action.migrate(dao, paramsFn())` — caller-supplied `MigrateFn`.
  <details><summary>details</summary>

  `NewChangeDAOImplementationAction(migrate MigrateFn)` takes a Gno closure and stores it inside the action payload. When the proposal passes, `migrate(dao, params)` is executed with full access to the current `*DAOPrivate` and returns whatever it wants as the new `daokit.DAO`. The closure runs in whatever realm-context the daokit machinery runs in, so it can do anything to the calling realm. This is materially a "delegatecall arbitrary function" feature gated only on the voting condition. `docs/resources/gno-interrealm.md:81-98` calls this exact pattern out as the canonical "do not let arbitrary functions be passed in" footgun. The mitigation has to be at the realm level (the realm must reject `NewChangeDAOImplementationAction` from anyone but a controlled curator), but neither the README nor the demo realm warns about this. The action's `String()` returns "WARNING: thoroughly check the migration code before approving this" — that warning needs to be in the README's upgrade section too, and ideally the demo realm should show how to gate the action behind a trusted-curator condition.
  </details>

- **[CI `main / test` is red on this branch]** [run 24354684875](https://github.com/gnolang/gno/actions/runs/24354684875) — `panic: txDispatcher subscription unexpectedly closed` in `gno.land/pkg/integration`.
  <details><summary>details</summary>

  Failure is in `tm2/pkg/bft/rpc/core/mempool.go:413` (`txDispatcher.listenRoutine`) — unrelated to anything in this diff. Looks like a flake or a master-side regression. Rebase on latest master before declaring CI healthy; if the failure persists, surface it as a question for maintainers.
  </details>

- **[`*ActionNewPost.Type()` is dead code, kind comes from `genericAction`]** [`examples/gno.land/r/samcrew/daodemo/custom_resource/custom_resource.gno:17-23`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/r/samcrew/daodemo/custom_resource/custom_resource.gno#L17-L23) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/r/samcrew/daodemo/custom_resource/custom_resource.gno#L17-L23) — `NewPostAction` wraps payload in `daokit.NewAction(ActionNewPostKind, payload)`, which returns `*genericAction`.
  <details><summary>details</summary>

  The returned `daokit.Action` is a `*genericAction` whose `Type()` returns the kind from `genericAction.kind`. `ActionNewPost.Type()` is therefore never called via the interface — it's a method on a struct that's only stored as `interface{}` payload. The custom-resource demo is supposed to teach readers how to add their own action types; right now it teaches them to write methods that don't get invoked. Either drop `ActionNewPost.String()`/`Type()` and let `genericAction` carry both, or document why both exist (and clarify the contract: do action structs need to implement `daokit.Action`? if not, why is `(*ActionNewPost).String()` defined to satisfy that interface?).
  </details>

## Nits

- [`examples/gno.land/p/samcrew/daokit/extensions.gno:7`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daokit/extensions.gno#L7) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daokit/extensions.gno#L7) — `Tree avl.Tree` (value, not pointer) while every other store uses `*avl.Tree`. Inconsistency; pick one shape.
- [`examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno:92`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L92) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L92) — `if totalPower == 0.0 { return totalPower }` returns the zero value instead of `0.0`; clearer as `return 0`.
- [`examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno:60`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L60) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L60) — comment "ufmt.Sprintf("%.2f", ...) is not working" repeated twice. If the limitation still holds, raise a separate issue on `ufmt`; if it was fixed, remove the workaround.
- [`examples/gno.land/p/samcrew/basedao/render.gno:91`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/basedao/render.gno#L91) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/basedao/render.gno#L91) — `getLinkPath` has `if slashIdx != 1` (literally `!= 1`); likely meant `!= -1`. Read-only render helper, so the failure mode is a stripped-too-short prefix on packages that *don't* contain a slash, but it's still wrong.
- [`examples/gno.land/p/samcrew/basedao/view_proposal_detail_page.gno:58`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/basedao/view_proposal_detail_page.gno#L58) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/basedao/view_proposal_detail_page.gno#L58) — `ufmt.Sprintf("%f", signal) + "\n"` is unlabeled; the rendered page shows a bare float dropped between Status and "proposed by". Wrap with a label or hide it.
- [`examples/gno.land/p/samcrew/basedao/utils.gno:96`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/basedao/utils.gno#L96) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/basedao/utils.gno#L96) — `if canvas != nil && len(canvas) > 0` — `canvas` is a slice; the nil check is redundant.
- [`examples/gno.land/r/samcrew/daodemo/simple_dao/simple_dao.gno:30`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/r/samcrew/daodemo/simple_dao/simple_dao.gno#L30) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/r/samcrew/daodemo/simple_dao/simple_dao.gno#L30) — `g1demo1234567890abcdefghijklmnopqrstuvwxyz` is not a valid bech32 address (checksum). It'll never be a real signer, but the demo could use `testutils.TestAddress("demo")` to be honest about it being fake. Same for the `g1fakemember…` strings on `utils.gno:29`.

## Missing Tests

- **[`crossingDAO` is never exercised through the wrapper]** [`examples/gno.land/p/samcrew/daokit/daokit_test.gno`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daokit/daokit_test.gno) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daokit/daokit_test.gno) — the only test in the file is `TestNewDAO` which asserts `Resources.Tree.Size() == 0`.
  <details><summary>details</summary>

  No test invokes `daokit.NewCrossing(...)`, no test calls `.Render`/`.Extension`/`.ExtensionsList` through the wrapper, no test exercises an upgrade via `ChangeDAOImplementation`. The Render bug above survived submission because the test suite only touches `localDAO`. A `TestCrossingDAO_Render` calling `wrappedDAO.Render(...)` is a five-line test that would catch the recursion. Same for `RenderWithVotes` coverage across all condition types — a smoke test that simply calls `cond.RenderWithVotes(emptyBallot)` for each `daocond` constructor would have caught both rendering panics surfaced above.
  </details>

- **[`Extension` privacy enforcement has no test]** [`examples/gno.land/p/samcrew/basedao/basedao.gno:22-31`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/basedao/basedao.gno#L22-L31) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/basedao/basedao.gno#L22-L31) — privacy gate is `runtime.CurrentRealm().PkgPath() != d.impl.Realm.PkgPath()`.
  <details><summary>details</summary>

  No test sets a `Private: true` extension and asserts that a foreign realm panics on access. Given that the privacy check is a string equality on `PkgPath()`, the only thing that can fail is a wrong-realm context being seen as the right one (e.g. if the call traversed an unexpected `cross`). A two-realm txtar test (realm A registers a private extension, realm B reads it, expect panic) is the right shape.
  </details>

- **[migration / upgrade flow has no test]** [`examples/gno.land/p/samcrew/basedao/migrate.gno`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/basedao/migrate.gno) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/basedao/migrate.gno) — `ChangeDAOImplementation` is a load-bearing new feature with zero coverage.
  <details><summary>details</summary>

  No test creates a DAO, proposes a `ChangeDAOImplementationAction`, votes it through, executes it, and asserts the new implementation actually replaces the old. Given the security concerns flagged above (arbitrary `MigrateFn` closure execution), a positive test that documents the expected happy path is the floor — a hostile test (a `MigrateFn` that returns nil; one that panics; one that exfiltrates state) would round out the surface.
  </details>

## Suggestions

- [`examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno:19`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L19) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daocond/cond_gnolovedao.gno#L19) — `var roleWeights = []float64{3.0, 2.0, 1.0}` is a package-level mutable slice. Make it `const`-like by hiding it behind a getter or use `[3]float64` so it can't be reassigned. Minor, but on a governance package the surface for mutable globals should be zero.
  <details><summary>details</summary>

  Globally-writable from anywhere in the package; any future helper that does `roleWeights[0] = 100` silently rebalances voting power for every DAO using `GnoloveDAOCondThreshold`. Replace with `var roleWeights = [3]float64{3, 2, 1}` (length-pinned, still indexable) or an unexported `func roleWeight(i int) float64 { ... }`.
  </details>

- [`examples/gno.land/p/samcrew/daokit/daokit.gno:88-92`](https://github.com/gnolang/gno/blob/51c2d96b6/examples/gno.land/p/samcrew/daokit/daokit.gno#L88-L92) · [↗](../../../../../.worktrees/gno-review-4884/examples/gno.land/p/samcrew/daokit/daokit.gno#L88-L92) — `Core.Execute` does not record `ExecutorID`. The `Proposal.ExecutorID` field is declared but never assigned.
  <details><summary>details</summary>

  `Proposal.ExecutorID` (`proposals.gno:51`) is `string` but `Core.Execute(proposalID uint64)` takes no executor. `DAOPrivate.Execute` calls `d.assertCallerIsMember()` and discards the result. Either thread the caller ID through to `Core.Execute` and store it on the proposal, or remove the field. As-is it's a dead field that audit-readers will assume is populated.
  </details>

## Questions for Author

- The `crossingDAO.Render` bug indicates the wrapper has never been used for rendering anywhere — what is the intended consumer? Is the plan that downstream realms always render through `localDAO` and never through the crossing handle? If so, document it; the wrapper should not expose `Render` at all in that case.
- For the `ChangeDAOImplementation` action: is there an expected pattern for restricting *who* can submit a migrate-action proposal beyond the voting condition? The current shape lets any DAO member propose an arbitrary `MigrateFn`; with a low quorum that's an upgrade-bomb.
- `MembersViewExtension` is registered with `Private: false` (default) — is exposing `IsMember(id) bool` across realms intentional? It's the only extension shipped by default; making it public-by-default sets the expectation for everyone else.
- Why is `realmid.Previous()` defined as a non-crossing helper that returns either `Address()` or `PkgPath()` of `runtime.PreviousRealm()`? The `assertCallerIsMember` chain depends on this returning a string the membership tree was keyed on. If anyone ever swaps the `CallerID` for one that returns user addresses while the store holds package paths (or vice versa), the membership check silently denies everyone. A short doc-comment pinning the contract ("must match the format used by `Members.AddMember`") would prevent that.
- Will the maintainer team accept the `gno.land/p/samcrew/` prefix indefinitely, or is there a vendoring plan into `nt`/`demo`? The README cross-links use `/p/demo/basedao.MembersView` as an example path — that's neither the current samcrew location nor a published demo. Either fix the README example to point at the real path or settle the upstreaming question first.
