# Contribution: PR [#5835](https://github.com/gnolang/gno/pull/5835) — `realm_only_gate` rule
Event: COMMENT

## Body
None of the rules catch a realms-only gate written `if x.IsUserCall() { panic }`. `IsUserCall()` is false inside a `maketx run` realm, so a user script walks straight through it; [#5976](https://github.com/gnolang/gno/pull/5976) does exactly this at `reputation.gno:17`. I have a `realm_only_gate` rule with vulnerable/fixed fixtures in your line-map idiom, harness green. Want it as a PR against `codex/audit-guide-examples`?
