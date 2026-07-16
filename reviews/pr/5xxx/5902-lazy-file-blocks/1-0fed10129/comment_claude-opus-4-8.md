# Review: PR [#5902](https://github.com/gnolang/gno/pull/5902)
Event: APPROVE

## Body
Verified on 0fed10129: reverting the line that routes [`GetParent`](https://github.com/gnolang/gno/blob/0fed10129/gnovm/pkg/gnolang/values.go#L640) through `GetFileBlock` reproduces the #4527 `file block missing` panic in [`TestLazyFileBlocksSkipUnusedStoreReads`](https://github.com/gnolang/gno/blob/0fed10129/gnovm/pkg/gnolang/store_test.go#L180), and a direct call into a 3-file realm touching one file costs 1943076 gas against master's 2074880.

The field [`fBlocksMap`](https://github.com/gnolang/gno/blob/0fed10129/gnovm/pkg/gnolang/values.go#L864) now carries an invariant with nothing recording it: on a store-loaded multi-file package it is empty or partial, so reads must go through `GetFileBlock`. The codebase already has one direct `fBlocksMap[...]` read at [`nodes.go:1441`](https://github.com/gnolang/gno/blob/0fed10129/gnovm/pkg/gnolang/nodes.go#L1441), safe only because it runs on the creation path; the same line added on a load path would reintroduce the #4527 panic class.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5902-lazy-file-blocks/1-0fed10129/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
