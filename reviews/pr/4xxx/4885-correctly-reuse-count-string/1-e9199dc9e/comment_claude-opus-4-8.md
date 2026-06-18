# Review: PR #4885
Event: COMMENT

## Body
Design is sound and both review threads are closed. Checked on e9199dc9e:

- A slice keeps the source backing counted even after the source dies.
- A forked allocator's cleanup no longer touches the parent's ranges.
- Each loaded string is charged once; the +32 / +485 gas deltas are that single charge.

Pre-existing, outside this diff: `StringValue`s built directly (not via `NewString`/`TrackString`) aren't charged to the allocator, so GC undercounts them. See [uverse.go:171](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/uverse.go#L171) and [values.go:2720](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/values.go#L2720).

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4885-correctly-reuse-count-string/1-e9199dc9e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
