# Review: PR #5739
Event: APPROVE

## Body
The flatten path and the direct-declaration path now emit identical FieldTypes for same-package unexported methods. Verified on a3b14e207: built ifaceext, ifacemid, and ifaceext2 in Go and compared the anonymous-interface reflect types. `interface{ ifacemid.Mid }` equals `interface{ ifaceext.Sec }` across the two-hop hoist, and both stay distinct from `interface{ ifaceext2.Sec }`. Matches the VM's iface_embed_xpkg2 and iface_embed_xpkg golden.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/3-a3b14e207/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
