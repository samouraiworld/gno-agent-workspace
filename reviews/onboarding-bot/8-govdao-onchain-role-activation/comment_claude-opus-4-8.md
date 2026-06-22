# Review: PR #8
Posted: https://github.com/samouraiworld/gno-onboarding-bot/pull/8#pullrequestreview-4547621908
Event: REQUEST_CHANGES

## Body
Requesting changes for one blocking security issue, inline. Verified on 283c572 against the gnolang/gno source: the `validators` RPC returns bech32 `g1` addresses matching the realm's rendered signing address, and the method takes no page params, so one call reads the whole active set.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/onboarding-bot/8-govdao-onchain-role-activation/review_claude-opus-4-8.md

## internal/valoper/valoper.go:70-71 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/283c572/internal/valoper/valoper.go#L70) [posted](https://github.com/samouraiworld/gno-onboarding-bot/pull/8#discussion_r3455024072)
blocking: `ParseRender` returns the first `- Signing Address:` line, but the realm renders the operator's free-text description before the real one, so a candidate can paste `- Signing Address: <any active validator>` into their description and the poller reads that instead. It then finds the spoofed address in the active set and grants the `Testnet Validator` role even though the candidate's own validator never joined. Read the signing address from the canonical block, not the description, the way `submit.go` already does for the operator address.

<details><summary>repro</summary>

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

The returned signing address is the one injected in the description, not the attacker's real `g1attackerrealsigning...`.
</details>

<!-- ===== follow-up: deep multi-lens pass; UNPOSTED, post as a second COMMENT review ===== -->

## FOLLOWUP-BODY
Follow-up from a deeper multi-lens pass on the new poller. The signing-address injection above is the blocker; the items below are additional should-fix and nit findings. One unanchored doc note: `MANUAL_TESTING.md:39` still says Approve grants `Testnet Validator` and removes the candidate role, which this PR moved off Approve onto the poller, so line 39 now contradicts the new activation section the PR added.

## internal/handlers/activation_poller.go:96
The poller writes `GovDAO submitted` here from a status snapshot taken at tick start, without re-checking the row. A reviewer Decline that lands after the snapshot but before this write sets `Declined` and removes the candidate role; this write then reverts the row to `submitted` and grants the validator role to the just-rejected candidate. Re-read and confirm the row is still `GovDAO pending` immediately before this write.

## internal/handlers/activation_poller.go:102
Status is already `GovDAO submitted` by the time this role-add runs, and each tick only reselects `GovDAO pending` rows. A transient failure here, or a crash before this line, strands the candidate at `submitted` with no role and no retry, across restarts. Reconcile `submitted` rows that lack the role, or write the terminal status only after the grant succeeds.

## internal/handlers/activation_poller.go:85
This recovers the Discord ID from the column-B hyperlink, but submit writes that hyperlink best-effort (`submit.go:174`, in the block where "failures here are logged but do not fail the submission"). If that write failed at submit, the row has no link and this path logs "grant manually" every tick forever without ever activating. Fall back to another ID source, or surface these rows as needs-attention instead of looping silently.

## internal/handlers/activation_poller.go:49
Missing test: the poller has no coverage (handlers package at 4.4%). `runActivationTick` is fakeable through `chainClient` and `sheet.API`, but nothing exercises the pending-filter, signing-empty, not-in-set, or promote branches, and `activateCandidate` cannot be tested at all because it takes a concrete `*discordgo.Session`. A small role/DM interface plus a table test would cover the selection logic.

## internal/handlers/activation_poller.go:64
Optional: a deregistered or unparseable valoper on a `GovDAO pending` row hits this `continue` after a log line every tick, indefinitely, with no backoff or terminal state. At the 5m default that is steady log noise with no signal that a specific candidate needs manual help. Consider a needs-attention status or rate-limiting the per-row log after N failures.

## internal/config/config.go:67
Nit: this rejects `<= 0` but sets no floor, so `validator_poll_interval: "1ns"` is accepted and would hammer the RPC node and Sheets API every tick. A small minimum, rejecting `< 5s`, fails closed.

## internal/sheet/sheet.go:433
Nit: this accepts any non-empty trailing text without `/?#`, so a hand-edited column-B cell like `.../users/@everyone` would be passed straight to `GuildMemberRoleAdd`. The bot writes this link itself in the normal flow, so it is low-risk; a numeric `^\d{17,20}$` check fails closed.
