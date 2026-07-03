# Review: PR [#5898](https://github.com/gnolang/gno/pull/5898)
Event: APPROVE

## Body
Looks good. Verified on a78260a07: the pasted `-gas-wanted 1_000_000_000` parses in gnokey, since `maketx` binds the flag with `flag.Int64Var` and base-0 `ParseInt` accepts Go underscore literals, and 1B stays under the 3B default block gas cap. Matches the value the Actions screen already emits.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5898-playground-run-gas-wanted/1-a78260a07/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
