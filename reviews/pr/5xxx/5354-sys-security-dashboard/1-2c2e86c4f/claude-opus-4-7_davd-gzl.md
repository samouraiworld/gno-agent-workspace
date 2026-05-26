# PR #5354: feat(example): add `r/sys/security` dashboard realm

URL: https://github.com/gnolang/gno/pull/5354
Author: davd-gzl | Base: master | Files: 4 | +732 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7

Verdict: NEEDS DISCUSSION — design holds for the pre-seeded set, but the user-extensible Add proposal path depends on persisting closures created in an ephemeral MsgRun package; that hasn't been verified end-to-end, and an Add/Add race silently upserts. Self-review by the PR author; second reviewer should validate the closure-persistence claim before merge.

## Summary

New read-only `r/sys/security` realm aggregating five pre-seeded security flags (`boards2.IsRealmLocked`, `boards2.AreRealmMembersLocked`, `validators.GetValidators`, `names.IsEnabled`, `cla.HasValidSignature`) into a single rendered markdown dashboard, plus a static "Not Queryable On-Chain" section for VM/param flags. Three GovDAO proposal constructors (Add/Update/Remove) let governance extend or amend the registry, with `description` text shown to voters. All 15 tests pass; full CI green; gas for the seeded render is ~26.7M (high for a `sys/` realm).

## Glossary

- `Check` — registry entry: label, `func() string` closure, expected value, risk string.
- `safeCall` — `defer recover` wrapper that turns closure panics into `ERROR: <reason>` strings.
- `matchExpected` — equality compare, with `"> 0"` parsed as integer threshold.
- MsgRun — one-shot tx that runs an ephemeral `main` package at `<chainDomain>/e/<caller>/run`; the package is **not persisted** (`gno.land/pkg/sdk/vm/keeper.go:927` calls `RunMemPackage(memPkg, false)`).

## Fix

Adds a single-tree registry seeded at `init()` with five production checks ([`security.gno:36-84`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L36-L84)). `Render` iterates the AVL tree, calls each closure under `safeCall`, bolds mismatches, and appends a help link. GovDAO operations build executors that call `setCheck` / `checks.Remove` on the package-level tree ([`security.gno:99-148`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L99-L148)). Help subpage at `/r/sys/security:help` documents MsgRun templates.

## Critical (must fix)

None confirmed. The closure-persistence concern below is a Warning until verified; if reproduced, promote to Critical.

## Warnings (should fix)

- **[user-supplied closures may not survive the MsgRun that created them]** [`security.gno:93-109`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L93-L109) — `valueFn` is captured from an ephemeral MsgRun package and persisted into `r/sys/security` state; the source package is discarded at end of tx, so later invocation may fail.
  <details><summary>details</summary>

  The help docs at [`security.gno:284-295`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L284-L295) instruct users to submit `NewAddCheckProposalRequest(..., func() string { return myrealm.GetFlag() })` from a MsgRun. Under MsgRun, the runtime creates a package at `chainDomain + "/e/" + caller + "/run"` and calls `RunMemPackage(memPkg, save=false)` ([`gno.land/pkg/sdk/vm/keeper.go:842-929`](../../../../../.worktrees/gno-review-5354/gno.land/pkg/sdk/vm/keeper.go#L842-L929)). The user's closure is a `FuncValue` with `PkgPath` and `Source` set to that ephemeral package ([`gnovm/pkg/gnolang/values.go:482-499`](../../../../../.worktrees/gno-review-5354/gnovm/pkg/gnolang/values.go#L482-L499)). When the GovDAO proposal later executes, `setCheck` stores the closure into `r/sys/security`'s `checks` tree. After the tx, the ephemeral package is gone. On the next render, `safeCall(c.ValueFn)` will try to resolve the closure's source via `GetSource → store.GetBlockNode(refnode.GetLocation())` ([`values.go:564-571`](../../../../../.worktrees/gno-review-5354/gnovm/pkg/gnolang/values.go#L564-L571)) — that lookup likely panics or returns nil, which `safeCall` masks as an `ERROR:` cell. The pre-seeded checks don't hit this because they're declared in the security realm's own permanent package. Fix: either (a) write an integration test (txtar) that submits an Add proposal via MsgRun, executes it, restarts/replays, and renders, to confirm the closure survives; or (b) restrict user-added checks to references that live in permanent realms (e.g. accept a `realmPath, funcName string` pair and resolve via `runtime` calls), removing the closure path entirely. Without (a) or (b), the public API of this realm is partially broken in a way the unit tests can't detect.
  </details>

- **[Add proposals silently upsert under a race]** [`security.gno:93-109`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L93-L109) — `checks.Has` is checked at proposal *creation*, not at *execution*; two concurrent Add proposals for the same id both pass creation, then the second execution overwrites the first via `checks.Set` instead of failing.
  <details><summary>details</summary>

  `NewAddCheckProposalRequest` panics if the id already exists at the time the proposal is built. But the executor's `cb` calls `setCheck` ([`security.gno:165-172`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L165-L172)) which is an unconditional `checks.Set`. Sequence: proposal A and proposal B both created (neither exists yet), A executes (id now present), B executes (silently replaces A's values). Voters who approved A see B's label/expected/risk after execution with no warning. Fix: have the executor re-check existence: `if checks.Has(id) { return errors.New("check already exists: " + id) }`. Same shape applies to Remove (Remove of an already-removed id is currently a no-op via `checks.Remove`; if Remove A and Remove B both target the same id, the second execution returns success but did nothing — minor compared to Add).
  </details>

- **[CLA enforcement check uses a brittle proxy]** [`security.gno:74-83`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L74-L83) — derives enforcement state from `!cla.HasValidSignature(address(""))`; correct today only because the zero address can't realistically sign, and inverts wrongly the moment `cla` exposes a direct getter or the zero-address invariant breaks.
  <details><summary>details</summary>

  `cla.HasValidSignature(addr)` returns `true` when `requiredHash == ""` (enforcement disabled) OR when `addr` has signed ([`cla.gno:38-43`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/cla/cla.gno#L38-L43)). The check exploits that `address("")` is unsignable to convert "has signed" → "is enforced". The semantic carried by the dashboard column (`Expected: "true"`) and the `Render` ([`cla/render.gno:6-7`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/cla/render.gno#L6-L7)) both diverge from this inverted reading. The comment at [`security.gno:77-79`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L77-L79) acknowledges the trick. Fix: add `func IsEnabled() bool { return requiredHash != "" }` to `r/sys/cla` (one-line patch in the same PR or follow-up), then use it directly. Names did exactly this ([`names/verifier.gno:28`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/names/verifier.gno#L28)).
  </details>

- **[render gas is ~5x typical sys realms]** [`security.gno:217-247`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L217-L247) — `TestRender_PreSeededChecks` consumes 26.7M gas for 5 checks; `r/sys/` README says checks must be "minimal, future-proof, and gas-efficient".
  <details><summary>details</summary>

  Render does per-check string concatenation (`out += ...`) which is quadratic in output size in gno (each `+=` allocates a new string). With 5 checks and ~50-char rows, this is tolerable; if the registry grows to 20+ checks the gas curve becomes unfriendly. Fix: build with a `strings.Builder` equivalent or pre-sized slice + `strings.Join`. Verify against [`r/sys/README.md`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/README.md). Not a blocker today but a decay risk as the registry expands.
  </details>

- **[realm dependency expansion]** [`security.gno:13-21`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L13-L21) — pulls in five realms (`boards2/v1`, `validators/v2`, `names`, `cla`, `gov/dao`) into `r/sys/security` deployment graph; any of them failing to deploy breaks security deploy too.
  <details><summary>details</summary>

  Genesis ordering matters: `r/sys/security` must deploy after all five. The ADR notes this under Consequences but doesn't propose a deployment-order test. Fix: add a txtar test that loads only `r/sys/security` and its transitive deps and verifies the render comes up clean — protects against a future PR that breaks one of the upstream APIs.
  </details>

## Nits

- [`security.gno:198-215`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L198-L215) — `Render(path)` only branches on `"help"`; any other non-empty path falls through to the main dashboard. Existing realms typically render a "not found" message. Minor consistency point.
- [`security.gno:150-163`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L150-L163) — `validateCheckArgs` only checks `strings.Contains(id, ":")`. An id like `:flag` passes validation and yields an empty realm path in the rendered link. One extra line: `if strings.HasPrefix(id, ":") { panic("id must have a realm before ':'") }`.
- [`security.gno:99`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L99), [`security.gno:120`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L120), [`security.gno:138`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L138) — `cur realm` parameter is unused in all three closures. Idiomatic in the codebase (`r/sys/cla/admin.gno:17`, `r/sys/users/admin.gno`) so this is fine, but worth a `_ = cur` or a comment if linting ever objects.
- [`security.gno:64`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L64) — `// TODO: update when #5080 is merged` carries a hard dependency on another PR. Either resolve before merging, or convert to a tracked issue.
- [`pr5354_security_dashboard.md:9-10`](../../../../../.worktrees/gno-review-5354/gno.land/adr/pr5354_security_dashboard.md#L9-L10) — ADR references `sysnames_pkgpath` but the dashboard's Not-Queryable table uses `vm:p:syscla_pkgpath`. Verify the names enforcement param key in the table.

## Missing Tests

- **[no end-to-end test for the user-Add path]** [`security_test.gno`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security_test.gno) — every existing test calls `setCheck` directly. None exercises the full Add-proposal → GovDAO-execute → Render lifecycle, which is exactly where the closure-persistence concern lives.
  <details><summary>details</summary>

  Add a txtar in `gno.land/pkg/integration/testdata/` that: (1) submits `NewAddCheckProposalRequest` via MsgRun referencing a function in a permanent example realm, (2) votes + executes via GovDAO, (3) calls `vm/qrender` on `gno.land/r/sys/security:` and asserts the new row is present with the correct current value. This validates the cross-tx closure survival empirically.
  </details>

- **[no test for register-overwrite via Update]** [`security_test.gno:118-137`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security_test.gno#L118-L137) — `TestUpdateCheck` updates via direct `setCheck`. No test verifies that `NewUpdateCheckProposalRequest(...)` on a pre-seeded id actually replaces the closure (i.e. that the new `valueFn` is what `Render` calls).
  <details><summary>details</summary>

  `TestNewUpdateCheckProposalRequest_Validation` only checks the panic/no-panic edge ([`security_test.gno:169-177`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security_test.gno#L169-L177)). Add a test that builds the proposal, invokes the executor's callback directly (`executor.Execute(cross)`), then asserts the rendered value changed. Cheap insurance.
  </details>

## Suggestions

- [`security.gno:355-364`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L355-L364) — `matchExpected` only special-cases `"> 0"`. If the dashboard grows, `>= N`, `< N`, `!= X` will follow. Either commit to the constrained grammar in a doc comment ("only `> 0` is supported as a threshold form; for richer matches encode them in the closure") or factor into a small parser. Doc comment is enough for v0.
- [`security.gno:36-84`](../../../../../.worktrees/gno-review-5354/examples/gno.land/r/sys/security/security.gno#L36-L84) — `init()` does five `checks.Set` calls with similar shape. A small helper `seed(id, label, expected, risk, fn)` halves the line count and makes the seed list easier to read top-to-bottom.
- [`pr5354_security_dashboard.md`](../../../../../.worktrees/gno-review-5354/gno.land/adr/pr5354_security_dashboard.md) — Consequences section should add a row: "Closures stored by Add/Update are FuncValues whose source lives in the proposer's MsgRun package. We rely on [whatever the actual gno persistence behavior is] for them to remain invocable across blocks. If this guarantee weakens, all user-added checks degrade to `ERROR:` rows under `safeCall`."

## Questions for Author

- Did you confirm empirically (gnodev or a chain replay) that a closure created in a MsgRun and stored into `r/sys/security`'s state still runs after the originating tx is finalized and the ephemeral package is discarded? If yes, please link the test. If no, see Warning #1.
- Why is `cla.HasValidSignature(address(""))` preferred over adding a one-line `cla.IsEnabled()` getter? The latter removes the inversion and the zero-address footgun.
- Is the Not-Queryable section intended to remain static forever, or do you anticipate moving it under GovDAO control later? If the latter, an issue link to track that future scope keeps the deferral honest.
- Should the dashboard surface the proposer of each non-pre-seeded check (e.g. via `executor.CreationRealm()`)? Helps voters audit who added a check.
