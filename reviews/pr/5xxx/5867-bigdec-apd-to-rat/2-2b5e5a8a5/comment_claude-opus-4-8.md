# Review: PR [#5867](https://github.com/gnolang/gno/pull/5867)
Event: APPROVE

## Body
The numerator guard closes the preprocess OOM from the last round. Verified on 2b5e5a8a5: the package I flagged before now rejects in about 0.2 s at preprocess, where before the fix it took 72 s and 481 MB. The denominator and numerator bounds each have a filetest, the conversion error renders decimals again, and const63 is a real filetest now.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/2-2b5e5a8a5/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
