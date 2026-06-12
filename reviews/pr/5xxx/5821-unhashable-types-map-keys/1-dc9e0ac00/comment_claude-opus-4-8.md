# Review: PR #5821
Posted: https://github.com/gnolang/gno/pull/5821#pullrequestreview-4489266912
Event: APPROVE

## Body
Looks good. Verified on dc9e0ac00: the panic message matches Go for every key shape, and equality and lookup still work for comparable interface-boxed keys.

- Optional: the new filetests only cover the failure path; a small positive-case filetest (a comparable key boxed into `map[interface{}]V`) would guard the happy path too.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5821-unhashable-types-map-keys/1-dc9e0ac00/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*
