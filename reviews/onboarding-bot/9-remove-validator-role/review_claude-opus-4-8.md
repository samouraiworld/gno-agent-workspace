# PR #9: feat: add /remove-validator-role bulk role removal command

Repo: samouraiworld/gno-onboarding-bot
URL: https://github.com/samouraiworld/gno-onboarding-bot/pull/9
Author: D4ryl00 (Rémi BARBERO) | Base: master | Head: feat/remove-validator-role | Files: 9 | +211 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 8084f81 (latest)
Local checkout: `git clone https://github.com/samouraiworld/gno-onboarding-bot .worktrees/onboarding-bot-review-9 && cd .worktrees/onboarding-bot-review-9 && gh pr checkout 9`

**TL;DR:** Adds a reviewer-only `/remove-validator-role` slash command. It strips the `Testnet Validator` role from every member who holds it and DMs each of them an onboarding-reset notice. Built for starting a fresh testnet onboarding round (Test13).

**Verdict: NEEDS DISCUSSION** — the "no gateway intent" design is correct and the command is otherwise clean, but two things need attention. It mutates Discord roles without touching the Google Sheet, so on a reused Sheet the reset members are blocked from re-onboarding (submit gate) and skipped by the harvest. And a per-member render error aborts the whole batch mid-flight with no report of what it already changed.

## Summary

The command pages the whole guild over the REST List Guild Members endpoint ([remove_validator_role.go:113](internal/handlers/remove_validator_role.go:113)), skips bots and members without the role, removes `validator_role_id`, then DMs each removed member the verbatim `role_removed` notice with a required `announcement-link` ([remove_validator_role.go:73-96](internal/handlers/remove_validator_role.go:73)). Role-removal and DM failures are collected and reported back to the reviewer with the affected member list and the full message text to relay by hand, per the closed-DM fallback invariant ([remove_validator_role.go:98-109](internal/handlers/remove_validator_role.go:98)).

The mechanics are right (see Good). The one substantive concern is what the command does *not* do: it never writes the Sheet, so after a reset the tracker and Discord disagree, and that divergence blocks the very re-onboarding the DM directs members to.

## Good

- **Remove-then-DM ordering is correct.** The DM claims the role was removed, and it is sent only after a successful `GuildMemberRoleRemove` ([remove_validator_role.go:85-92](internal/handlers/remove_validator_role.go:85)); a member whose removal failed is never told it succeeded, and DM failures fall to manual relay.
- **Reviewer summary disables mention parsing.** The aggregated summary interpolates member-controlled display names, and `editEphemeralNoMentions` zeroes `AllowedMentions` so a nickname like `@everyone` cannot ping ([remove_validator_role.go:107-109](internal/handlers/remove_validator_role.go:107), [discord_helpers.go:66](internal/handlers/discord_helpers.go:66)). The per-member DM does not need the guard: a DM channel cannot ping third parties.
- **"No GuildMembers gateway intent" is correct, verified against Discord's docs.** REST List Guild Members is gated on the **Server Members** Developer Portal toggle, which is "independent of Gateway restrictions, and unaffected by which intents your app passes in the `intents` parameter when Identifying" ([Discord gateway docs](https://docs.discord.com/developers/events/gateway)). So keeping `IntentsGuildMembers` out of `s.Identify.Intents` ([main.go:72](main.go:72)) is right: a missing toggle fails only this command, with a clear error ([remove_validator_role.go:64](internal/handlers/remove_validator_role.go:64)), instead of blocking the whole bot at connect time. The bot is single-guild, so no privileged-intent verification gate applies.
- **Pagination matches the endpoint contract.** `limit=1000` is the documented max, and the `after` cursor uses the highest user id of the previous page ([remove_validator_role.go:117-125](internal/handlers/remove_validator_role.go:117)).

## Warnings (should fix)

- **[reset leaves the Sheet tracker out of sync, blocking re-onboarding]** `internal/handlers/remove_validator_role.go:85` — the role is removed but no Sheet row is touched, so reset members stay `GovDAO approved`/`GovDAO pending`.
  <details><summary>details</summary>

  This handler does role mutation with no Sheet write at all (it even takes `api sheet.API` and never uses it). After a reset, the affected rows keep `GovDAO approved` or `GovDAO pending`. Two downstream consequences:

  - `IsReopenable` is false for both statuses, so a former validator who tries to re-onboard with the same operator address is rejected at [submit.go:128](internal/handlers/submit.go:128): "This operator address is already in the tracker (row N, status \"GovDAO approved\"). You can submit again once that row is marked \"Needs retry\" or \"Declined\"." The `role_removed` DM tells them to "apply for the Test13 validator set through the new onboarding process," and that flow refuses them.
  - `IsValidated` stays true ([sheet.go:496](internal/sheet/sheet.go:496)), so the harvest skips these rows on the next pass.

  This bites only if Test13 reuses the same Sheet. If the round points the bot at a fresh Sheet (new `SheetName`), the old rows live elsewhere and neither gate fires. Fix: confirm the reset swaps to a fresh Sheet and document that as a deploy step, or have this command also move each affected row to a reopenable state so re-submission and harvest work.
  </details>

- **[a render error aborts the whole batch mid-flight with no report]** `internal/handlers/remove_validator_role.go:78-83` — on a per-member template-render error the handler `return`s, discarding the summary of the members it already changed.
  <details><summary>details</summary>

  The loop removes the role and DMs each member in turn. If `tpl.RoleRemoved` returns an error for some member, the handler logs, shows a generic "Could not render the message template" reply, and `return`s ([remove_validator_role.go:78-83](internal/handlers/remove_validator_role.go:78)). Members processed before that point have already lost the role and been DMed, but the accumulated `removed`/`roleErrors`/`dmFailures` summary is thrown away, so the reviewer gets no record of what the partial run did on a destructive bulk action.

  The trigger is unreachable today: `render` writes to a `bytes.Buffer`, which never returns a write error, and the `role_removed` template has fixed `.Name`/`.AnnouncementLink` fields supplied as plain strings, so `Execute` cannot fail once `Load` succeeded ([templates.go:69-75](internal/templates/templates.go:69)). The defect is the control flow, not a live crash: a destructive bulk command should not abort partway on a per-member error without reporting the members it already mutated. Fix: validate the template once before the loop, with a placeholder name, so the only abort happens before any role is touched; if a per-member render error becomes possible later, `continue` and record it instead of `return`.
  </details>

## Nits

- `internal/handlers/remove_validator_role.go:19` — `api sheet.API` is unused; name it `_ sheet.API` to match `RegisterHarvest`'s `_ *templates.Templates` ([harvest.go:32](internal/handlers/harvest.go:32)). The parameter is structurally required by the shared `registrations` signature ([main.go:85](main.go:85)).
- `internal/handlers/remove_validator_role.go:125` — `after = page[len(page)-1].User.ID` does not nil-check `.User`, though the loop body at [:74](internal/handlers/remove_validator_role.go:74) does. Harmless for the REST List Guild Members response, which always includes the user object, but the two lines disagree on whether `User` can be nil.

## Missing Tests

- **[pagination and skip logic have no automated coverage]** `internal/handlers/remove_validator_role.go:113` — `allGuildMembers` cursor advancement and short-page termination, plus the bot/no-role skip filter, are untested.
  <details><summary>details</summary>

  The repo convention is manual testing for session-dependent handler code, and the PR adds a MANUAL_TESTING section, so this is a suggestion, not a blocker. But `activation_poller` already isolates discordgo behind a small `disc` interface with a `fakeDiscord` ([activation_poller_test.go:97](internal/handlers/activation_poller_test.go:97)); the same shape would make the paging loop and skip filter unit-testable without a live session.
  </details>

## Suggestions

- `internal/handlers/remove_validator_role.go:46` — the whole operation runs synchronously inside the deferred interaction with no progress feedback. Per removed member it makes ~3 REST calls (role remove + DM-channel create + send); a very large validator set under discordgo's rate limiter could approach the 15-minute interaction-token window, after which the final summary edit fails silently. Low risk for typical testnet set sizes; consider a followup message or chunked progress if the set can grow large.
- `internal/handlers/remove_validator_role.go:104` — the closed-DM relay text uses a `[their name]` placeholder, so the reviewer personalizes per member by hand. The per-member list sits directly above it, so it is workable; minor.

## Open questions

- Concurrency with the activation poller: a poller tick that grants `validator_role_id` while this loop runs (or just after it passes a row) would re-grant the role a reset just removed. Operationally a reset is a deliberate one-off and the poller acts only on `GovDAO pending` rows, so this is unlikely to matter in practice; not posted.
