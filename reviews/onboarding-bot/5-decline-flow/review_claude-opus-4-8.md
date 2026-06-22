# PR #5: Sync templates with Shared.md; rename retry flow to Decline

Repo: samouraiworld/gno-onboarding-bot
URL: https://github.com/samouraiworld/gno-onboarding-bot/pull/5
Author: D4ryl00 (Rémi BARBERO) | Base: master | Head: update-templates-decline-flow | Files: 13 | +93 -73
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: e3aced3
Local checkout: `~/Projects/gno-onboarding-bot` (`git fetch upstream pull/5/head`)

**TL;DR:** Syncs the six candidate/reviewer DM templates to the latest `Shared.md` wording, drops the now-purposeless `review_sla` config end-to-end, and reworks the old "Ask to retry" reviewer action into a "Decline" flow that removes the candidate role and sets a new `Declined` status.

**Verdict: APPROVE** — code correct, build and all tests pass. One operational caveat: the renamed Discord command leaves a stale "Ask to retry" entry registered in the guild that must be pruned manually at deploy.

## Summary

The PR has two parts. First, all six template messages in `templates.yaml` are rewritten to match the source-of-truth `Shared.md` wording, including the operator-address phrasing and removal of the `[DATE OR TIMEFRAME]` placeholder. That placeholder removal makes `review_sla` dead, so it is dropped from `config.go` (field plus required-validation), `config.example.yaml`, `config_test.go`, `Acknowledge()`'s signature, and the docs. A leftover `review_sla` line in an existing `config.yaml` is silently ignored by the YAML unmarshal, so the removal is backward-compatible.

Second, the reviewer "Ask to retry" action becomes "Decline". The new flow removes the `Testnet Validator Candidate` role and sets status `Declined` (the old flow kept the role and set `Needs retry`). The unused `{{.Actions}}` template field and its modal input are dropped. `Declined` is wired into `StatusColors` and `AllStatuses`. A new `sheet.IsReopenable` helper lets both `Needs retry` and `Declined` rows be resubmitted, and `submit.go`'s duplicate guard now calls it instead of comparing against `Needs retry` alone.

## Two reviewer outcomes now

| Command | Status | Candidate role | Next step |
|---|---|---|---|
| Request missing information | `Needs retry` | kept | correct items, `/submit-request` again |
| Decline | `Declined` | removed | restart from `/candidate-testnet` |

## Warnings (should fix)

- **[stale Discord command lingers after deploy]** `internal/handlers/review_decline.go:26` — the rename leaves a dead "Ask to retry" command registered in the guild.
  <details><summary>details</summary>

  `RegisterDecline` registers via `s.ApplicationCommandCreate` with the new name `"Decline"`. Nothing in the codebase prunes old commands: every handler only calls `ApplicationCommandCreate`, never `ApplicationCommandDelete` or a bulk overwrite (`grep -rn "ApplicationCommandDelete\|BulkOverwrite" .` returns nothing). So after deploy the old "Ask to retry" message-context command stays registered in the guild. A reviewer who clicks it gets no response: the only handler now early-returns on `i.ApplicationCommandData().Name != "Decline"`. Fix: delete the old command at deploy (manually, or add a one-time prune on startup), and add a line to `MANUAL_TESTING.md` covering it.
  </details>

## Nits

- `internal/handlers/review_decline.go` (finalizeDecline) — sheet status is set to `Declined` before `GuildMemberRoleRemove`. If role removal fails, the row already reads `Declined` while the role is still present; the reviewer is told to remove it manually and relay the message. Acceptable since it is messaged, but the row and the actual role state diverge until the manual fix.
- `templates.yaml` (decline message) — hardcodes the literal channel name `` `┋💬ㆍgeneral-chat` `` (emoji plus name). Brittle if the channel is renamed; other templates reference `#testnet-onboarding` plainly.

## Good

- `IsReopenable` is case- and whitespace-tolerant (`internal/sheet/sheet.go:457`) and `TestIsReopenable` exercises both directions, including `" needs retry "` and `"DECLINED"`.
- `review_sla` removal is genuinely backward-compatible: unknown YAML keys are ignored on unmarshal, so a leftover line in a live `config.yaml` does not break startup.
- `Declined` is added to both `StatusColors` and `AllStatuses`, so the sheet dropdown and coloring stay consistent.
- `submit.go`'s rollback comment was updated to match the new reopen set ("Needs retry" / "Declined").

## Open questions / dependency

- The matching `Shared.md` wording change is in still-open PR D4ryl00/gno-validators-onboarding#3 and must land alongside this one or the bot text drifts from its source of truth. No runtime risk: templates are verbatim-tested in `templates_test.go`. Already flagged by the author in the PR body.

## Verification

- `go build ./...`, `go vet ./...`, `go test ./internal/...` all pass on commit e3aced3.
- Per `CLAUDE.md`, the `internal/handlers` changes (renamed command, single modal field, role removal, new status) still need a manual pass through `MANUAL_TESTING.md` before deploy. The stale-command warning above is the concrete item that pass should catch.
