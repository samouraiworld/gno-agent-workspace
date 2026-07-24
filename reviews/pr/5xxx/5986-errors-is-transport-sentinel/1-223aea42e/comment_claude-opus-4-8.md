# Review: PR [#5986](https://github.com/gnolang/gno/pull/5986)
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Restoring `errors.As(err, &errTransportClosed)` on [`switch.go:637`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch.go#L637) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch.go#L637) reproduces the data race on the first `-race` run of [`TestFlappyBroadcastTxForPeerStopsWhenPeerStops`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/bft/mempool/reactor_test.go#L208) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/bft/mempool/reactor_test.go#L208). Without that edit the same command is clean over 50 runs.

Repros run at 223aea42e.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5986-errors-is-transport-sentinel/1-223aea42e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/p2p/switch.go:637 [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch.go#L637)
Missing test: the accept loop receiving another sentinel from the [same var block](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L24-L30) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L24-L30), the one case `errors.Is` and `errors.As` disagree on. [`TestSwitchAcceptLoopTransportClosed`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch_test.go#L896-L923) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch_test.go#L896-L923) feeds it the sentinel itself, which both comparisons match, so it stays green with `errors.As` back on this line.

<details><summary>test cases</summary>

```go
func TestSwitchAcceptLoopUnrelatedSentinel(t *testing.T) {
	// No t.Parallel: the assertions read the package-level sentinel.
	original := errTransportClosed
	t.Cleanup(func() { errTransportClosed = original })

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	var accepts atomic.Int64

	mockTransport := &mockTransport{
		acceptFn: func(ctx context.Context, _ PeerBehavior) (PeerConn, error) {
			// Block once the loop has proven it survives the unrelated sentinel.
			if accepts.Add(1) > 3 {
				<-ctx.Done()

				return nil, ctx.Err()
			}

			return nil, errDuplicateConnection
		},
	}

	sw := NewMultiplexSwitch(mockTransport)

	done := make(chan struct{})

	go func() {
		sw.runAcceptLoop(ctx)
		close(done)
	}()

	require.Eventually(
		t,
		func() bool { return accepts.Load() > 3 },
		2*time.Second,
		10*time.Millisecond,
		"accept loop exited on an error that is not errTransportClosed",
	)

	cancelFn()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "accept loop did not exit on context cancellation")
	}

	assert.Equal(t, "transport is closed", errTransportClosed.Error())
}
```
</details>

## tm2/pkg/bft/mempool/reactor_test.go:201-207 [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/bft/mempool/reactor_test.go#L201-L207)
Nit: the remaining failure is neither `-race`-only nor intermittent. `STABILITY_FILTER=flappy go test ./tm2/pkg/bft/mempool/` failed 5 of 6 test instances over 3 runs, and both tests fail the same way at the merge base. The pointer to the accompanying fix also stops resolving once this branch is squashed onto master.
