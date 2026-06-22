# PR #8: Feat/govdao onchain role activation

Repo: samouraiworld/gno-onboarding-bot
URL: https://github.com/samouraiworld/gno-onboarding-bot/pull/8
Author: louis14448 (Lours) | Base: master | Head: feat/govdao-onchain-role-activation | Files: 21 | +1483 -34
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep: red/blue/correctness lenses) | Commit: 283c572
Local checkout: `~/Projects/gno-onboarding-bot` (`git fetch upstream pull/8/head`)

**TL;DR:** Moves the `Testnet Validator` role grant off the reviewer's Approve click and onto an on-chain check. Approve now only forwards the candidate to the GovDAO and sets status `GovDAO pending`; a background poller grants the role once the candidate's validator signing address shows up in the node's active validator set.

**Verdict: REQUEST CHANGES** — the on-chain gate can be bypassed (signing address read from operator-controlled free text), and the new sole-grantor poller has concurrency, recovery, and dependency gaps: a concurrent Decline can be overwritten, a failed role grant strands the candidate forever, and the poller hard-depends on a best-effort submit-time write.

## Summary

Before this PR, a reviewer clicking Approve granted the `Testnet Validator` role directly. The PR replaces that with a two-step flow. Approve sets status `GovDAO pending`, DMs the candidate, and pings the GovDAO contact ([review_approve.go:55](internal/handlers/review_approve.go:55)). A new poller ([activation_poller.go](internal/handlers/activation_poller.go)) ticks every `validator_poll_interval` (default 5m): it reads the active validator set once via the node's `validators` RPC, then for each `GovDAO pending` row derives the candidate's signing address from their `r/gnops/valopers` render and, when that address is in the active set, writes `GovDAO submitted`, grants the role (removing the candidate role), and DMs the candidate.

The activation happy path is wired correctly (see Good). The findings cluster in two places: the signing address is parsed out of an attacker-influenced render (Critical), and the poller is now the only thing that grants the role, so its concurrency and failure handling matter more than before (Warnings).

## How the bypass works

```
realm render of r/gnops/valopers:<operatorAddr>:

  ## <moniker>
  <Description>            <- operator free text, verbatim, 0-2048 chars, newlines allowed
                              attacker puts "- Signing Address: <active validator>" here
  - Operator Address: <real>
  - Signing Address: <real>   <- the canonical line the parser SHOULD read
  - Signing PubKey: <pubkey>
  - Server Type: <type>
```

`ParseRender` scans top-to-bottom and keeps the **first** `- Signing Address:` match ([valoper.go:70](internal/valoper/valoper.go:70)). The operator's Description is rendered before the canonical block, so a line injected there wins. The poller then checks that spoofed address against the active set ([activation_poller.go:77](internal/handlers/activation_poller.go:77)) and grants the role.

## Critical (must fix)

- **[candidate can spoof an active validator and get the role without joining the set]** `internal/valoper/valoper.go:70-71` (consumed at `internal/handlers/activation_poller.go:69,77`) — the signing address comes from a first-match scan over a render that contains operator-controlled free text.
  <details><summary>details</summary>

  The `r/gnops/valopers` realm renders a valoper's `Description` verbatim, before the `- Operator Address:` / `- Signing Address:` block (`Valoper.Render`, `valopers.gno:462-476` in gnolang/gno). `Description` is operator-set free text with no charset or newline restriction, only a non-empty + 2048-char cap (`validateDescription`). So an operator can register their own valoper with a description containing a line `- Signing Address: g1<an already-active validator>`. Because `ParseRender` takes the first `- Signing Address:` it sees and the description is rendered first, that injected line is what the poller reads.

  Impact: a candidate registers their own valoper (operator address = theirs, which is the sheet dedup/query key), embeds a known active validator's signing address in the description, passes the bot challenge, and gets a reviewer to Approve (status → `GovDAO pending`). On the next tick the poller derives the spoofed signing address, finds it in the active set, and grants the `Testnet Validator` role plus the activation DM, even though the candidate's own validator was never admitted. This nullifies the exact gate the PR exists to add. Granted artifact is the testnet Discord role, and a reviewer Approve is still required upstream, but the PR's design demotes Approve to a low-trust "forward to GovDAO" step and makes the on-chain check the real gate.

  The PR already guards the operator address against this exact attack: `submit.go:101-104` ignores the parsed operator address and uses the address it queried with, "so it cannot be spoofed by free text in the description." The signing address needs the same treatment. Fix: read the signing address from the realm's canonical trailing block rather than the first prefix match. Taking the last `- Signing Address:` occurrence is robust here: every field after the real signing line (`Signing PubKey`, `Server Type`, the profile link) is non-free-text, so nothing the operator controls can appear after the canonical line.

  **Repro:** ParseRender returns the injected address. Run at 283c572:

  ```bash
  # from a fresh clone of samouraiworld/gno-onboarding-bot:
  gh pr checkout 8 -R samouraiworld/gno-onboarding-bot
  cat > internal/valoper/zz_injection_test.go <<'EOF'
  package valoper

  import "testing"

  func TestSigningInjection(t *testing.T) {
  	render := "Valoper's details:\n## Attacker\n" +
  		"welcome to my node\n" +
  		"- Signing Address: g1victimactivevalidatorxxxxxxxxxxxxxxxx\n\n" +
  		"- Operator Address: g1attackeroperatorxxxxxxxxxxxxxxxxxxxxxxx\n" +
  		"- Signing Address: g1attackerrealsigningxxxxxxxxxxxxxxxxxxxx\n" +
  		"- Signing PubKey: gpub1xyz\n- Server Type: cloud\n"
  	_, _, signing, _, _ := ParseRender(render)
  	t.Logf("signing = %q", signing)
  }
  EOF
  go test ./internal/valoper/ -run TestSigningInjection -v
  rm internal/valoper/zz_injection_test.go
  ```

  ```
  === RUN   TestSigningInjection
      zz_injection_test.go:14: signing = "g1victimactivevalidatorxxxxxxxxxxxxxxxx"
  --- PASS: TestSigningInjection (0.00s)
  ```

  The returned address is the injected one from the description, not the attacker's real `g1attackerrealsigning...`.
  </details>

## Warnings (should fix)

- **[concurrent reviewer decision silently overwritten]** `internal/handlers/activation_poller.go:60-97` — the poller acts on a status snapshot from tick start and writes `GovDAO submitted` with no compare-and-set, so a reviewer action that races the tick is clobbered.
  <details><summary>details</summary>

  The tick reads every row once, filters on `r.Status == "GovDAO pending"` ([activation_poller.go:61](internal/handlers/activation_poller.go:61)), then for each match renders the valoper (a per-row network call) before `activateCandidate` blind-writes `GovDAO submitted` ([activation_poller.go:96-97](internal/handlers/activation_poller.go:96)) and grants the role. Nothing re-reads the status between the snapshot and the write, and sheet writes share no lock (`appendMu` is private to `AppendCandidateRow`). If a reviewer runs Decline in that window, which sets `Declined` and removes the candidate role ([review_decline.go:95](internal/handlers/review_decline.go:95)), the poller overwrites `Declined` back to `GovDAO submitted` and grants the `Testnet Validator` role to the just-rejected candidate. Approve no longer grants the role, so the poller is the sole grantor and this defeats the human decline. The window is narrow (a Decline concurrent with the tick, while the candidate's signing address is already in the active set), hence Warning. Fix: re-read and confirm the row is still `GovDAO pending` immediately before the status write, or make the write conditional.
  </details>

- **[failed role grant strands the candidate with no retry]** `internal/handlers/activation_poller.go:96-102` — status flips to `GovDAO submitted` before the role is granted, and the pending-only filter means a transient failure or crash is never retried.
  <details><summary>details</summary>

  `UpdateFields` writes `GovDAO submitted` ([activation_poller.go:96](internal/handlers/activation_poller.go:96)) before `GuildMemberRoleAdd` ([activation_poller.go:102](internal/handlers/activation_poller.go:102)). If the role-add returns a transient error (Discord 5xx) or the process dies between the two, the row reads `submitted` with no role, and the next tick skips it because line 61 only matches `GovDAO pending`. Nothing reconciles `submitted` rows, so recovery is the logged "grant manually" line only, permanently, across restarts. The write-before-mutate ordering is deliberate (avoids a double DM on retry) but trades away all automatic recovery. Fix: a reconcile pass over `submitted` rows missing the role, or write the terminal status only after the role mutation succeeds.
  </details>

- **[poller hard-depends on a best-effort write]** `internal/handlers/activation_poller.go:85-93` (depends on `internal/handlers/submit.go:174`) — the Discord ID is recovered from the column-B hyperlink, which submit writes best-effort, so a submit-time blip silently makes the row un-activatable.
  <details><summary>details</summary>

  `activateCandidate` reads the column-B link via `CellLink` and parses the ID ([activation_poller.go:85-90](internal/handlers/activation_poller.go:85)); if it is missing it logs "grant the role manually" and returns ([activation_poller.go:91-93](internal/handlers/activation_poller.go:91)). That link is written at submit time inside an explicitly best-effort block: "failures here are logged but do not fail the submission" ([submit.go:166-174](internal/handlers/submit.go:166)). So a transient `SetLinkedText` failure at submit commits the row without the hyperlink, and the poller can then never resolve the ID: it logs "grant manually" every tick forever and never activates. The design doc treats the persisted ID as a guaranteed precondition. Fix: surface missing-link rows as needs-attention instead of silently looping, or recover the ID from the review-notification embed (which already carries it) as a fallback.
  </details>

- **[stale manual-test step contradicts the new flow]** `MANUAL_TESTING.md:39` — still tells testers Approve grants the validator role, which this PR removed.
  <details><summary>details</summary>

  Line 39 says Approve "grants `Testnet Validator`, removes `Testnet Validator Candidate`". This PR deletes the role grant from Approve ([review_approve.go](internal/handlers/review_approve.go)) and appends a new "GovDAO on-chain role activation" section ([MANUAL_TESTING.md:60-66](MANUAL_TESTING.md:60)) stating Approve grants no role, but leaves line 39 in place. The file now contradicts itself and instructs testers to verify behavior the PR deleted. The same bullet's "exact 'Approve a candidate' wording" is also stale after the `templates.yaml` rewording. Fix: update line 39 to match the new Approve behavior.
  </details>

## Nits

- `internal/config/config.go:67` — `validator_poll_interval` rejects `<= 0` but has no floor, so `"1ns"` is accepted and would hammer the RPC node and Sheets API every tick. A small minimum (reject `< 5s`) fails closed.
- `internal/sheet/sheet.go:433` — `DiscordIDFromUserURL` accepts any non-empty trailing text without `/?#`, so a hand-edited column-B cell like `.../users/@everyone` is passed straight to `GuildMemberRoleAdd`. Bot-written in the normal flow, so low-risk; a `^\d{17,20}$` check fails closed.
- `internal/handlers/activation_poller.go:28-30` — the doc comment says callers "can wait for any in-flight tick to finish", but the tick shares `ctx`; on shutdown the in-flight RPC/Sheet calls are cancelled, not finished. Behavior is safe (main waits the done channel and a cancelled write leaves the row pending), but the comment overstates it.
- `internal/handlers/activation_poller.go:106-117` — a `GuildMemberRoleRemove` failure is logged and swallowed, then `activation: OK row %d` logs success while the candidate still carries the Candidate role; the "OK" overstates the outcome.

## Missing Tests

- **[the whole poller is untested]** `internal/handlers/activation_poller.go:49,84` — `runActivationTick` and `activateCandidate` have zero coverage (handlers package at 4.4%; no test references the poller).
  <details><summary>details</summary>

  `runActivationTick` is fakeable today (`chainClient` and `sheet.API` are interfaces) but no test drives its selection branches: the pending-status filter ([activation_poller.go:61](internal/handlers/activation_poller.go:61)), the signing-empty skip ([activation_poller.go:74](internal/handlers/activation_poller.go:74)), the not-in-set skip ([activation_poller.go:77](internal/handlers/activation_poller.go:77)), and the promote call ([activation_poller.go:80](internal/handlers/activation_poller.go:80)). `activateCandidate` is structurally untestable because it takes a concrete `*discordgo.Session`, so the role-add-fail, role-remove-swallow, sheet-write-fail, and ID-unresolvable branches cannot be exercised; extracting a small role/DM interface would unblock them. `CellLink` plus the textFormatRuns/hyperlink fallback also has no round-trip test (`fakeAPI.CellLink` returns "", [sheet_test.go:155](internal/sheet/sheet_test.go:155)). Highest-value add: a table test with a fake `chainClient` (canned active set + per-realm render) and a `sheet.API` fake returning mixed-status rows, asserting exactly which rows reach activation.
  </details>

- **[no coverage for description-injected fields]** `internal/valoper/valoper_test.go` — `ParseRender` is tested only on well-formed renders. A test with a description containing `- Signing Address:` / `- Operator Address:` lines would have caught the Critical and would lock in the fix.

## Suggestions

- `internal/handlers/activation_poller.go:64-71` — a deregistered (`ErrNotRegistered`) or reformatted (`ErrUnparseable`) valoper on a `GovDAO pending` row logs one error per tick forever, with no max-attempt or backoff. At the default 5m tick this is steady log spam that buries real failures and gives no signal a specific candidate needs manual help. Consider a terminal needs-attention status or per-row log rate-limiting after N failures.

## Good

- `validators` RPC membership is format-correct. The RPC returns `validators[].address` as a bech32 `g1...` string (`tm2/pkg/bft/types/validator.go` `Address ... json:"address"`, `Address.String()` -> bech32), and the realm renders `v.SigningAddress.String()` in the same bech32 form, so the `set[signingAddr]` comparison ([activation_poller.go:77](internal/handlers/activation_poller.go:77)) compares like with like.
- The no-pagination assumption holds for gno. gno's `validators` RPC method is registered with only a `height` arg and takes no `page`/`per_page` (`tm2/pkg/bft/rpc/core/routes.go`, `consensus.go` `Environment.Validators`), unlike upstream Tendermint which caps a page at 100. The single-call read in [client.go:120-124](internal/valoper/client.go:120) is correct, and the comment's caveat about a future paging node is the right hedge.
- Activation path columns line up end to end: submit writes the Discord profile hyperlink to column B via `SetLinkedText` (a `textFormatRuns` link), and `CellLink` reads `textFormatRuns.format.link.uri` back from the same column ([activation_poller.go:85](internal/handlers/activation_poller.go:85)); `DiscordIDFromUserURL` parses the id. Approve sets `ColumnStatus` = `GovDAO pending`, which is exactly what the poller filters on.
- PR-body invariants 1-3 hold: sheet write precedes every role mutation (poller and approve), no Sheet schema change (signing address derived each tick, never stored), and `GovDAO submitted` is reused with no new status value and no conflation (the poller is its only new writer; the `-approved` view and `IsValidated` already covered it). Invariant 4 "drains gracefully" is only partially true (see the shutdown Nit).
- `ParseRender` is null-safe on absent signing addresses: older valopers without a `- Signing Address:` line yield `signingAddr == ""`, which the poller skips ([activation_poller.go:74](internal/handlers/activation_poller.go:74)).
- Graceful shutdown wiring is correct in `main.go`: `StartActivationPoller` returns a done channel closed after the goroutine observes `ctx.Done()`, and `main` waits on it before the deferred session close, so no in-flight tick races the teardown (even if a cancelled tick aborts rather than finishing).

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` all pass on 283c572.
- The Critical is demonstrated by the repro above. The Decline race, the best-effort-write dependency, and the stale `MANUAL_TESTING.md:39` line were each verified against the actual files (decline writes `Declined` with no CAS; the column-B hyperlink sits in submit's logged-non-fatal block; the PR diff appends a section but does not touch line 39).
- The format-match and no-pagination claims in Good were verified by reading the gnolang/gno source in the workspace submodule, not assumed from the PR's own comments.
- Per the bot's testing convention, the Discord/Sheet-dependent activation path still needs a live pass through `MANUAL_TESTING.md` before deploy.

## Open questions

- `UpdateFields` ([internal/sheet/sheet.go:445](internal/sheet/sheet.go:445)) is a non-atomic per-cell loop over a Go map that aborts on the first error, so the multi-field reviewer writes (approve/decline/missing-info/escalate) can land partially in a non-deterministic order. Pre-existing, not introduced here, but #8 adds a concurrent writer that interleaves with them. Not posted.
- `ApprovedTabName`'s doc comment ([internal/sheet/sheet.go:257](internal/sheet/sheet.go:257)) says the `-approved` tab mirrors `GovDAO pending` rows, but the formula selects `submitted` OR `pending`; pre-existing, but #8 makes `submitted` an actively-produced status. Not posted.
- TOCTOU and row identity: a validator in the set at the tick then jailed/rotated out keeps the role (one-shot promotion, no demotion, by design); `r.Row` is a sheet index, so a concurrent row insert/delete retargets the write. Inherent to the sheet-as-store design. Not posted.
- The `validators` RPC returns every active validator including core/genesis ones, so the spoofable target set for the Critical is large and public. Not posted.
