# Review: PR [#5900](https://github.com/gnolang/gno/pull/5900)
Event: APPROVE

## Body
Looks good. Reverting the store change on 69be9a6ed re-adds exactly 59000 gas: the addpkg golden goes from GAS USED 2756592 to 2815592, one flat store read per transaction.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5900-skip-synthetic-path-probe/1-69be9a6ed/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
