# Review: PR #5739
Event: APPROVE

## Body
Looks good. Verified on a00dde6b3: reverting the selection gate to the enclosing-package form lets `main` select `ifaceext`'s unexported `sec` through a locally-spelled interface, so `iface_embed_xpkg_access` and `iface_embed_xpkg` fail; the gate is load-bearing. The hard-error for pre-flattening state sits at `fillType`, the one decode boundary every stored type reaches through `GetType`, so such state panics on load before `TypeID` can emit a split identity.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/4-a00dde6b3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
