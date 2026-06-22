# PR #8: Feat/govdao onchain role activation

Repo: samouraiworld/gno-onboarding-bot
URL: https://github.com/samouraiworld/gno-onboarding-bot/pull/8
Author: louis14448 (Lours) | Base: master | Head: feat/govdao-onchain-role-activation | Files: 21 | +1483 -34
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 283c572
Local checkout: `~/Projects/gno-onboarding-bot` (`git fetch upstream pull/8/head`)

**TL;DR:** Moves the `Testnet Validator` role grant off the reviewer's Approve click and onto an on-chain check. Approve now only forwards the candidate to the GovDAO and sets status `GovDAO pending`; a background poller grants the role once the candidate's validator signing address shows up in the node's active validator set.

**Verdict: REQUEST CHANGES** — the on-chain gate the PR adds can be bypassed: the poller reads the candidate's signing address from operator-controlled free text, so a candidate can spoof an already-active validator's address and get the role without ever joining the active set.

## Summary

Before this PR, a reviewer clicking Approve granted the `Testnet Validator` role directly. The PR replaces that with a two-step flow. Approve sets status `GovDAO pending`, DMs the candidate, and pings the GovDAO contact ([review_approve.go:55](internal/handlers/review_approve.go:55)). A new poller ([activation_poller.go](internal/handlers/activation_poller.go)) ticks every `validator_poll_interval` (default 5m): it reads the active validator set once via the node's `validators` RPC, then for each `GovDAO pending` row derives the candidate's signing address from their `r/gnops/valopers` render and, when that address is in the active set, writes `GovDAO submitted`, grants the role (removing the candidate role), and DMs the candidate.

The activation path is internally consistent and the happy path checks out (see Good). The defect is in how the signing address is derived: it is parsed out of the realm render, and that render embeds operator-controlled free text ahead of the field the parser keys on.

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

  Impact: a candidate registers their own valoper (operator address = theirs, which is the sheet dedup/query key), embeds a known active validator's signing address in the description, passes the bot challenge, and gets a reviewer to Approve (status → `GovDAO pending`). On the next tick the poller derives the spoofed signing address, finds it in the active set, and grants the `Testnet Validator` role plus the activation DM, even though the candidate's own validator was never admitted to the active set. This nullifies the exact gate the PR exists to add. Granted artifact is the testnet Discord role, and a reviewer Approve is still required upstream, but the PR's design demotes Approve to a low-trust "forward to GovDAO" step and makes the on-chain check the real gate.

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

None.

## Nits

None.

## Missing Tests

- **[no coverage for description-injected fields]** `internal/valoper/valoper_test.go` — `ParseRender` is tested only on well-formed renders. A test with a description containing `- Signing Address:` / `- Operator Address:` lines would have caught the Critical and would lock in the fix.

## Suggestions

- `internal/handlers/activation_poller.go:96-102` — status is set to `GovDAO submitted` before `GuildMemberRoleAdd`. If the role grant fails transiently (Discord 5xx), the row is already `submitted`, so the next tick skips it and the role is never retried; recovery is the logged "grant manually" line only. The sheet state stays truthful (the validator is in the set) and the failure is logged, so this is acceptable graceful degradation, but a transient API blip means permanent manual intervention. Flagging for awareness, not blocking.

## Good

- `validators` RPC membership is format-correct. The RPC returns `validators[].address` as a bech32 `g1...` string (`tm2/pkg/bft/types/validator.go` `Address ... json:"address"`, `Address.String()` -> bech32), and the realm renders `v.SigningAddress.String()` in the same bech32 form, so the `set[signingAddr]` comparison ([activation_poller.go:77](internal/handlers/activation_poller.go:77)) compares like with like.
- The no-pagination assumption holds for gno. gno's `validators` RPC method is registered with only a `height` arg and takes no `page`/`per_page` (`tm2/pkg/bft/rpc/core/routes.go`, `consensus.go` `Environment.Validators`), unlike upstream Tendermint which caps a page at 100. The single-call read in [client.go:120-124](internal/valoper/client.go:120) is correct, and the comment's caveat about a future paging node is the right hedge.
- Activation path columns line up end to end: submit writes the Discord profile hyperlink to column B via `SetLinkedText` (a `textFormatRuns` link), and `CellLink` reads `textFormatRuns.format.link.uri` back from the same column ([activation_poller.go:85](internal/handlers/activation_poller.go:85)); `DiscordIDFromUserURL` parses the id. Approve sets `ColumnStatus` = `GovDAO pending`, which is exactly what the poller filters on.
- Write-before-mutate and idempotency hold: the sheet write to `GovDAO submitted` precedes any role change, and the poller only ever acts on `GovDAO pending` rows, so a row is promoted at most once and a crash mid-tick cannot double-grant.
- `ParseRender` is null-safe on absent signing addresses: older valopers without a `- Signing Address:` line yield `signingAddr == ""`, which the poller skips ([activation_poller.go:74](internal/handlers/activation_poller.go:74)) rather than misbehaving.
- Graceful shutdown is correct: `StartActivationPoller` returns a done channel closed after the goroutine observes `ctx.Done()`, and `main.go` waits on it before exit, so an in-flight tick finishes before the Discord session tears down.

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` all pass on 283c572.
- The Critical is demonstrated by the repro above (ParseRender returns the description-injected address).
- The format-match and no-pagination claims in Good were verified by reading the gnolang/gno source in the workspace submodule, not assumed from the PR's own comments.
- Per the bot's testing convention, the Discord/Sheet-dependent activation path still needs a live pass through `MANUAL_TESTING.md` before deploy.

## Open questions

- The `validators` RPC returns every active validator including core/genesis ones, so the spoofable target set is large and public. Not a separate finding, just confirms the Critical is cheap to exploit. Not posted.
