# Review: PR [#5944](https://github.com/gnolang/gno/pull/5944)
Event: APPROVE

## Body
Looks good. Verified on 859dcb88c: removing the `signer.PubKey == nil` guard reproduces the original nil-pointer panic on placeholder-signature txs, which [`gnogenesis fork valoper-seed`](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/fork/valoper_seed.go#L372) and [`addpkg`](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/fork/addpkg.go#L124) both emit.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5944-gnogenesis-skip-sig-check/1-859dcb88c/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnogenesis/internal/verify/verify.go:96-98 [↗](../../../../../.worktrees/gno-review-5944/contribs/gnogenesis/internal/verify/verify.go#L96)
Missing test: no case asserts a non-signature check still fails with `-skip-signature-check` set. The two skip subtests only prove the signature loop is bypassed, so a later change widening the skip past signatures would still pass CI. A subtest that runs the flag against a hardfork genesis with an uncovered validator should still fail with [`errUncoveredGenesisValidator`](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/verify/verify.go#L20).

<details><summary>test cases</summary>

Add inside `TestGenesis_Verify`. Green as shipped; red if the `continue` ever becomes a broader short-circuit, since the post-loop coverage check would be skipped.

```go
t.Run("skip signature check still runs later checks", func(t *testing.T) {
	// -skip-signature-check skips only the per-tx signature loop.
	// Post-loop checks still run: a hardfork genesis with an uncovered
	// validator and a placeholder-signature tx is rejected for the
	// coverage gap even with the flag set.
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
