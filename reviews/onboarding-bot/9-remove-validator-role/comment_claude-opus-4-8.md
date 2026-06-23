# Review: PR #9
Event: COMMENT

## Body
No blocking issues in the command itself. Verified on 8084f81 against Discord's docs. The REST List Guild Members call is gated on the **Server Members** Developer Portal toggle. That toggle is independent of the gateway Identify intents. So keeping `IntentsGuildMembers` out of Identify is correct, and a missing toggle fails only this command instead of blocking startup.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/onboarding-bot/9-remove-validator-role/review_claude-opus-4-8.md [↗](review_claude-opus-4-8.md)

## internal/handlers/remove_validator_role.go:85 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/8084f81/internal/handlers/remove_validator_role.go#L85)
After a reset the Sheet row stays `GovDAO approved`. `IsReopenable` rejects that, so a re-submit of the same operator address is refused at [submit.go:128](https://github.com/samouraiworld/gno-onboarding-bot/blob/8084f81/internal/handlers/submit.go#L128) and the harvest skips the row because [`IsValidated`](https://github.com/samouraiworld/gno-onboarding-bot/blob/8084f81/internal/sheet/sheet.go#L496) stays true. The `role_removed` DM sends them to the onboarding flow that then refuses them, so does this command need to move affected rows to a reopenable state, or does the round use a fresh Sheet?

## internal/handlers/remove_validator_role.go:78-83 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/8084f81/internal/handlers/remove_validator_role.go#L78)
A per-member render error returns out of the whole handler after earlier members have already lost the role and been DMed, and the discarded summary leaves the reviewer no record of the partial run. The path is unreachable today, since `render` writes to a `bytes.Buffer` over fixed string fields, so the real concern is the abort placement in a destructive bulk command. Validate the template once before the loop, before any role is removed, so the only abort point mutates nothing.

## internal/handlers/remove_validator_role.go:19 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/8084f81/internal/handlers/remove_validator_role.go#L19)
Nit: `api sheet.API` is unused here. Name it `_ sheet.API` to match [`RegisterHarvest`](https://github.com/samouraiworld/gno-onboarding-bot/blob/8084f81/internal/handlers/harvest.go#L32)'s unused `_ *templates.Templates`.
