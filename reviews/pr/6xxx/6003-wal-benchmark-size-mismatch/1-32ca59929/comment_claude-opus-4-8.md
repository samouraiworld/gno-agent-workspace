# Review: PR [#6003](https://github.com/gnolang/gno/pull/6003)
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Changing the reader back to the fixed `maxTestMsgSize` reproduces the error from [issue #910](https://github.com/gnolang/gno/issues/910) verbatim: `length 102445 exceeded maximum possible value of 65536 bytes`.

Repros run at 32ca59929.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6003-wal-benchmark-size-mismatch/1-32ca59929/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/wal/wal_test.go:291-293 [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L291-L293)
`BenchmarkWalRead1GB` peaks at 12.3 GB resident for a single iteration, so the package's benchmark run goes from 46 MB to 12.8 GB and is OOM-killed under a 12 GiB budget. Consensus opens the WAL at a [1 MB max record size](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/consensus/reactor.go#L29) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/consensus/reactor.go#L29), so a real node's writer rejects a record this size.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6003 -R gnolang/gno
go test -c -o /tmp/wal.test ./tm2/pkg/bft/wal/
python3 - <<'EOF'
import resource, subprocess
subprocess.run(["/tmp/wal.test", "-test.run", "^$", "-test.bench", "BenchmarkWalRead1GB",
                "-test.benchtime", "1x", "-test.benchmem"])
print("MAX_RSS_MB=%.0f" % (resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss / 1024))
EOF
rm /tmp/wal.test
```

```
goos: linux
goarch: amd64
pkg: github.com/gnolang/gno/tm2/pkg/bft/wal
cpu: AMD Ryzen 7 7840HS w/ Radeon 780M Graphics
BenchmarkWalRead1GB-16    	       1	1528271404 ns/op	8631233144 B/op	  349598 allocs/op
PASS
MAX_RSS_MB=12313
```
</details>

## tm2/pkg/bft/wal/wal_test.go:275-277 [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L275-L277)
Missing test: outside the benchmarks nothing in this package reads back a record above 64 KB. The tm2 job runs [`go test ... ./...`](https://github.com/gnolang/gno/blob/32ca59929/.github/workflows/_ci-go.yml#L124) · [↗](../../../../../.worktrees/gno-review-6003/.github/workflows/_ci-go.yml#L124) with no `-bench`, so CI never exercises the path this PR repairs. [`TestWALWriterReader`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L41-L67) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L41-L67) round-trips two messages with empty payloads.

<details><summary>test cases</summary>

```go
func TestWALWriterReaderLargeMessage(t *testing.T) {
	t.Parallel()

	const n = 100 * 1024 // above maxTestMsgSize
	maxSize := int64(n) + 64

	msg := TimedWALMessage{
		Time: tmtime.Now().Round(time.Second).UTC(),
		Msg:  TestMessage{Height: 1, Round: 1, Data: random.RandBytes(n)},
	}

	buf := new(bytes.Buffer)
	require.NoError(t, NewWALWriter(buf, maxSize).Write(msg))

	decoded, meta, err := NewWALReader(buf, maxSize).ReadMessage()
	require.NoError(t, err)
	require.Nil(t, meta)
	assert.Equal(t, msg.Msg, decoded.Msg)
	assert.Equal(t, msg.Time, decoded.Time)
}
```
</details>
