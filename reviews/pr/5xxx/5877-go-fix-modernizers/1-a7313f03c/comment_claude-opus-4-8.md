# Review: PR [#5877](https://github.com/gnolang/gno/pull/5877)
Event: APPROVE

## Body
Looks good. Verified on a7313f03c: ran `go fix` with the `omitzero` modernizer and it strips `json:",omitempty"` from Amino struct fields including [RefValue](https://github.com/gnolang/gno/blob/a7313f03c/gnovm/pkg/gnolang/values.go) realm state, so the `-omitzero=false` carve-out prevents a real JSON-format change.

The red `docs` check is the docs URL linter flagging remote links in files this PR does not touch.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5877-go-fix-modernizers/1-a7313f03c/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
