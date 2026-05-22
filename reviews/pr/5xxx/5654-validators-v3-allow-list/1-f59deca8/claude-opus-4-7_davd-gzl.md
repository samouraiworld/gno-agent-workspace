# PR #5654: feat(r/sys/validators/v3): add allow list

**URL:** https://github.com/gnolang/gno/pull/5654
**Author:** tbruyelle | **Base:** master | **Files:** 3 | **+326 -0**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Adds an "allow list" mechanism to `r/sys/validators/v3` letting whitelisted realms
bypass GovDAO and call `AddValidator` / `RemoveValidator` directly, on top of the
existing operator-keyed proposal path. The whitelist is a `bptree.BPTree32` of
realm paths managed via a new `NewPropAllowedRealmUpdateRequest` GovDAO proposal
constructor (insert/remove batched, dedupe enforced at create-time).

New file `allowed.gno` (135 lines) exposes:
- `AddValidator(cur realm, change ValoperChange, signingPubKey string, signingAddress address)`
  — auth-checks `PreviousRealm().PkgPath()` against `allowedRealms`, auto-registers
  the operator in `valoperCache` if missing (commit `74bab7f31`), then reuses
  `newValoperChangeExecutor` to apply and publish.
- `RemoveValidator(cur realm, op address)` — same auth gate; no-ops if the
  operator is not in the cache; otherwise reuses the executor with `Power: 0`.
- `IsAllowedRealm(pkgPath string) bool`, `GetAllowedRealms() []string` —
  public reads.
- `NewPropAllowedRealmUpdateRequest(add, remove []string, title, description string) dao.ProposalRequest`
  — non-crossing builder, rejects empty title / empty add+remove / any path
  appearing more than once across both lists.

`validators.gno` gains a 4-line doc-only paragraph pointing at `allowed.gno`. The
PR also adds a 187-line test file covering allow/deny auth paths, power-zero
rejection in `AddValidator`, the dedupe/empty/overlap panics in the proposal
builder, and description rendering.

Branch history: `44e6ea9c4` introduced the file; `74bab7f31` (different author,
Julien Robert) added the `signingPubKey`/`signingAddress` parameters and the
auto-registration / no-op-on-missing behavior; `f59deca82` fixes tests.

## Test Results

- **Existing tests:** PASS (33/33 in `examples/gno.land/r/sys/validators/v3/`, run
  via in-tree built `gno` binary with `GNOROOT` pointing at the worktree to
  pick up the stdlib that defines `GetSysParamUint64`).
- **CI status:** `generate` and `gno-checks / fmt` are failing on the PR head;
  `Merge Requirements` (fork-edit permission) is also red. Build / docker / check
  pass. `codecov/patch`: all modified lines covered.
- **Edge-case tests:** skipped (findings are reasoned from source; the existing
  test suite already covers the auth/dedupe edges).

## Critical (must fix)

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:30-46` — `AddValidator`
  does not validate that `signingAddress` is the address derived from
  `signingPubKey`. An allow-listed realm can pass `(pubKeyA, addrZ)` and the
  resulting `cacheEntry` is published to the effective valset with a signing
  pubkey that doesn't match the signing address. `RotateValoperSigningKey`
  derives the address from the pubkey via `chain.PubKeyAddress` (cache.gno:82-89);
  this path bypasses that check. Add
  `derived, err := chain.PubKeyAddress(signingPubKey); if err != nil || derived != signingAddress { panic(...) }`
  before the `valoperCache.Set` to keep the invariant uniform with rotation. The
  blast radius depends on how Tendermint interprets `(pubkey, address)` mismatches
  in `valset:proposed`; even if it rejects at boundary check, allowing the bad
  entry to be written into the cache poisons valset publishes downstream.

## Warnings (should fix)

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:35-41` — `AddValidator`
  silently ignores the provided `signingPubKey`/`signingAddress` when the
  operator is already in `valoperCache`. The caller can reasonably expect those
  params to be the authoritative values for the resulting valset entry; instead,
  the executor reads the cache (`proposal.gno:142-145`) and publishes the *old*
  signing key. Either (a) panic if `signingPubKey != cached.SigningPubKey` (force
  rotation through `RotateValoperSigningKey`), or (b) overwrite the cache and
  document that AddValidator is also a key-rotation entry point.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:51-60` — `RemoveValidator`
  silently no-ops when the operator is not in `valoperCache`. A downstream realm
  that calls `RemoveValidator` and treats a non-panic as "validator removed"
  would be wrong. Either panic with a clear "unknown operator" message, or
  return a `(removed bool)` so callers can branch. Cf. `proposal.gno:148-152`
  which panics on the equivalent missing-from-valset condition.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:30-46` — Allow-listed
  realms that auto-register operators here never create a corresponding profile
  in `r/gnops/valopers`. The file-level comment in `cache.gno:20-26` documents
  the invariant "`valoperCache` mirrors the (operator -> current signing key)
  view from `r/gnops/valopers`. Written by valopers via `NotifyValoperChanged`."
  This PR breaks that invariant: the operator can never rotate (no profile to
  rotate), can never opt out via `UpdateKeepRunning`, and shows up in
  `AssertGenesisValopersConsistent` only if it happens to be seeded at genesis.
  Either (a) extend the invariant comment to allow allow-list-side registration
  and document the consequences, or (b) call back into `valopers` to create the
  profile (which the PR description deliberately avoids per the
  "READ-ONLY against valopers" comment on `NotifyValoperChanged`). At minimum
  document the divergence in `allowed.gno`.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:83-135` —
  `NewPropAllowedRealmUpdateRequest` has no cap on `len(add)+len(remove)`. The
  sibling `NewValidatorProposalRequest` caps at 40 (`proposal.gno:70-72`) with
  the exact same rationale ("max number of allowed validators per proposal");
  the same upper bound (or higher) here would prevent a single proposal from
  rendering a multi-MB description and iterating a huge whitelist diff in one
  block during executor run.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:83-135` — No validation
  that entries in `add` are well-formed realm paths (non-empty, start with
  `gno.land/`, no trailing `/`, etc.). Garbage entries are accepted and silently
  bloat the bptree. An empty-string entry is benign at auth time (the empty-path
  check at `assertCallerIsAllowed:18-20` fires first for users) but pollutes
  `GetAllowedRealms()` output. Recommend `strings.TrimSpace`+non-empty check
  matching `proposal.gno:67`.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:83-135` — No invocation
  pattern documented. `NewPropAllowedRealmUpdateRequest` is non-crossing (no
  `cur realm`), matching `NewValidatorProposalRequest`. The latter at least
  documents the situation (`proposal.gno:35-37`: "NON-CROSSING (no `cur realm`).
  Direct MsgCall is unsupported; proposers route through
  r/gnops/valopers/proposal's facade …"). Add an equivalent paragraph here.
  Without it, a user MsgCall to this function will fail with a confusing error
  and the recipient won't know which facade to use.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:121-131` — Removing a
  realm from the allow list does not revoke validators that realm previously
  added (no cleanup hook). Governance can disenfranchise realm X but X's added
  validators stay in the active set. If this is intentional, document it; if
  not, consider a "remove the realm AND remove validators it added" governance
  primitive (which would require tracking provenance).

## Nits

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:13` — `bptree.BPTree32`
  with `true` as the always-stored value is presence-only; storing booleans
  wastes a pointer per entry. Acceptable for the expected small whitelist size;
  not worth changing unless a `bptree.Set` variant exists.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:107-119` — Description
  is built with the user-supplied `description` first (no header), then a
  blank-line gap, then `## Allowed Realm Updates`. If the user passes a Markdown
  document with an `<h1>`/`<h2>` of the same title, voters see two top-level
  headings. Cosmetic; the sibling builder has the same shape.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed.gno:67-75` —
  `GetAllowedRealms` allocates a `[]string` sized at `allowedRealms.Size()`
  before iterating; since `Iterate` already gives keys in sorted order from a
  bptree, this is fine. The `return false` continuation pattern (continue) reads
  awkwardly — bptree's `IterCbFn` convention is "return true to stop". A
  one-line comment would help.

- [ ] `examples/gno.land/r/sys/validators/v3/allowed_test.gno:133` — The empty
  title rejection test uses `"   "` (three spaces). Good for testing the
  `strings.TrimSpace` path. But there's no test for `description` containing
  Markdown-injection content like `\n## Allowed Realm Updates\n- add: evil`; a
  malicious proposer could prepend a fake "Allowed Realm Updates" block to
  mislead voters skimming the description. Not exploitable in itself but worth
  pinning behavior.

## Missing Tests

- [ ] No test for the pubkey/address-mismatch case flagged in the Critical
  section. Add a case calling `AddValidator(cross, {op-X, Power: 5}, pubKeyA, mustAddr(t, pubKeyB))`
  and asserting it aborts.

- [ ] No test that exercises the executor's `cur realm` context when the
  callback mutates `allowedRealms`. The existing tests poke `allowedRealms`
  directly in test setup; an end-to-end test that builds a `ProposalRequest`,
  drives its `Executor.Execute(cross)`, and asserts the whitelist was updated
  would catch a regression where the closure captures the wrong storage realm.

- [ ] No test for `NewPropAllowedRealmUpdateRequest` with whitespace-only entries
  in `add`/`remove`, and no test that the same realm path with different casing
  is treated as distinct (or not — the current code treats them as distinct via
  `seen[r]` map keys).

- [ ] No test for repeated `AddValidator` on the same operator with different
  `signingPubKey`/`signingAddress` (the "silent ignore" Warning above). Whatever
  the chosen semantics, pin it in a test.

- [ ] No test for `RemoveValidator` on an operator that was never in the cache
  (the "silent no-op" Warning). Pin the chosen semantics in a test.

## Suggestions

- Move the `assertCallerIsAllowed` helper to `cache.gno` next to
  `assertValopersCaller` so the two caller-auth gates live in one place; both
  rely on the same "called via cross from a crossing function" precondition
  documented at `cache.gno:34-46`.

- Consider exposing an `IsAllowedRealmList() []string` paginated variant; a
  whitelist that grows to thousands of entries (no cap today) makes
  `GetAllowedRealms` an expensive call.

- Add a one-line `Render` extension that lists the current allow list under the
  valset render — operators of allow-listed realms benefit from being able to
  read the current state without instrumenting `gnoclient`.

- Consider emitting an `AllowedRealmAdded`/`Removed` event with the proposal ID
  (not just the realm path) so off-chain indexers can correlate whitelist
  changes with their governance proposal. The current event payload is only
  `("realm", r)`.

## Questions for Author

- Is the auto-registration in `AddValidator` (commit `74bab7f31`) intentionally
  bypassing the valopers profile lifecycle? If yes, please document the
  trade-off in `cache.gno`'s invariant comment so the next reader doesn't trust
  the "cache mirrors valopers" line.

- What is the intended invocation flow for `NewPropAllowedRealmUpdateRequest`?
  Is there a planned facade in `r/gnops/valopers/proposal` (mirroring how the
  proposal-keyed builder is reached), or should this live somewhere else?

- The Codecov check reports full patch coverage, but I count no test that
  actually drives `NewPropAllowedRealmUpdateRequest`'s callback end-to-end (only
  validation paths). Did the test suite intentionally skip the
  build-proposal-then-execute round-trip?

- Are stale-operator validators left over from a now-revoked allow-listed realm
  acceptable, or do you have a follow-up to clean those up?

- The PR description says "supersedes #5168" — what changed semantically
  between this and #5168? Knowing the diff helps me focus the next review round
  on the deltas.

## Verdict

REQUEST CHANGES — the pubkey/signing-address mismatch is a real attack surface
(allow-listed realm can poison the valset); the silent-no-op semantics on
`AddValidator`/`RemoveValidator` need to be either fixed or pinned in tests
before merge. The structural design (GovDAO-gated whitelist + reuse of the
operator-keyed executor + crossing-function caller-auth via `PreviousRealm`)
is sound and the CI failures appear to be fork-edit and `gnofmt` issues, not
logic.
