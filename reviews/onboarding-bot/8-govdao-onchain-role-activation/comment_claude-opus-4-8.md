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
