# Review: PR [#5944](https://github.com/gnolang/gno/pull/5944)
Event: APPROVE

## Body
Verified on 859dcb88c: dropping the `signer.PubKey == nil` guard brings back the nil-pointer panic on placeholder-signature txs, which [`gnogenesis fork valoper-seed`](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/fork/valoper_seed.go#L372) and [`addpkg`](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/fork/addpkg.go#L124) emit.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5944-gnogenesis-skip-sig-check/1-859dcb88c/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnogenesis/internal/verify/verify.go:96-98 [↗](../../../../../.worktrees/gno-review-5944/contribs/gnogenesis/internal/verify/verify.go#L96)
With `-skip-signature-check`, only the signature loop should be skipped. The tests prove that, but none check that the later validator-coverage check still runs with the flag set. So if the skip is ever widened to bypass more, CI stays green.

<details><summary>test cases</summary>

Add inside `TestGenesis_Verify`. Passes as shipped, fails if the `continue` ever widens into a short-circuit that skips the post-loop coverage check.

```go
t.Run("skip signature check still runs later checks", func(t *testing.T) {
	// -skip-signature-check skips only the per-tx signature loop. Post-loop
	// checks still run, so an uncovered validator is rejected even with the flag.
	t.Parallel()

	tempFile, cleanup := testutils.NewTestFile(t)
	t.Cleanup(cleanup)

	g := getValidTestGenesis()

	sender := ed25519.GenPrivKey()
	tx := std.Tx{
		Msgs: []std.Msg{bank.MsgSend{
			FromAddress: sender.PubKey().Address(),
			ToAddress:   sender.PubKey().Address(),
			Amount:      std.NewCoins(std.NewCoin("ugnot", 10)),
		}},
		Fee: std.Fee{GasWanted: 1000000, GasFee: std.NewCoin("ugnot", 20)},
	}
	tx.Signatures = make([]std.Signature, len(tx.GetSigners()))

	state := g.AppState.(gnoland.GnoGenesisState)
	state.PastChainIDs = []string{"old-chain"}
	state.Txs = []gnoland.TxWithMetadata{{Tx: tx}}
	g.AppState = state

	require.NoError(t, g.SaveAs(tempFile.Name()))

	cmd := NewVerifyCmd(commands.NewTestIO())
	args := []string{
		"--genesis-path", tempFile.Name(),
		"--skip-signature-check",
	}

	cmdErr := cmd.ParseAndRun(context.Background(), args)
	assert.ErrorIs(t, cmdErr, errUncoveredGenesisValidator)
})
```
</details>
