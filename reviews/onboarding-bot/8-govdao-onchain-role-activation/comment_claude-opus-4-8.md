# Review: PR #8
Event: REQUEST_CHANGES

## Body
Requesting changes for one blocking security issue, inline. Verified on 283c572 against the gnolang/gno source: the `validators` RPC returns bech32 `g1` addresses that match the realm's rendered signing address. That method takes no page params, so the single-call read covers the whole active set.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/onboarding-bot/8-govdao-onchain-role-activation/review_claude-opus-4-8.md

## internal/valoper/valoper.go:70-71 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/283c572/internal/valoper/valoper.go#L70)
blocking: `ParseRender` keeps the first `- Signing Address:` line, but the realm renders the operator's free-text description before the canonical address block, so a candidate can put `- Signing Address: <any active validator>` in their description and the poller reads that spoofed address instead of their own. The poller then finds it in the active set and grants the `Testnet Validator` role to a candidate whose validator never joined, bypassing the on-chain gate this PR adds. `submit.go` already ignores the parsed operator address for this same reason; read the signing address from the canonical block, not the description region.

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
