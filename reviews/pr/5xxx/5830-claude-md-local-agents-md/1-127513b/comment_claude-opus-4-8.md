# Review: PR #5830
Event: APPROVE

## Body
Looks good. Verified on 127513b that every rule from the deleted `CLAUDE.md` is present in `AGENTS.md` (verification, before/after-metric, PR-description, interrealm, and payment-guard groups), and both docs the security rules point to resolve: `docs/resources/gno-interrealm.md` and `docs/resources/effective-gno.md § Verifying inbound Coin payments`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5830-claude-md-local-agents-md/1-127513b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## AGENTS.md:100 [↗](../../../../../.worktrees/gno-review-5830/AGENTS.md#L100)
The migration dropped this rule's cross-check command: the old `CLAUDE.md` had `grep -rn "IsUser()" examples/` to locate existing `IsUser()`+`OriginSend()` realms, and the new wording says what to look for but not how. Optional given `AGENTS.md` is the terser public file: restore the grep so the rule stays runnable.
