# Review: PR #5
Event: COMMENT

## Body
Looks good. Build, vet, and tests pass on e3aced3; `review_sla` removal is backward-compatible since unknown YAML keys are ignored on unmarshal, and `IsReopenable` is tested both directions. One deploy-time caveat inline.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/onboarding-bot/5-decline-flow/review_claude-opus-4-8.md

## internal/handlers/review_decline.go:26 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/e3aced3/internal/handlers/review_decline.go#L26)
Renaming the command to "Decline" registers a new command but does not remove the old "Ask to retry" one. Nothing in the codebase deletes or bulk-overwrites commands, so after deploy the old "Ask to retry" entry stays in the guild and silently does nothing when clicked (its handler now early-returns on any name but "Decline"). Prune it at deploy and add a line to MANUAL_TESTING.md.

## templates.yaml:30 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/e3aced3/templates.yaml#L30)
The decline message hardcodes the channel name `┋💬ㆍgeneral-chat`. If that channel is renamed the text goes stale. Other templates reference `#testnet-onboarding` plainly.
