# Review: PR #5739
Event: COMMENT

## Body
Verified at a00dde6b3.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/types.go:1062-1077 [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1062)
Selecting an unexported method resolves by source order, so `interface{ ifaceext.Sec; sec() int }` rejects `x.sec()` from main while listing the own `sec()` first compiles. Go binds `x.sec()` to main's own `sec` either way. Minor: it over-rejects rather than grants access, and no concrete type can satisfy the interface.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5739 -R gnolang/gno
curl -fsSL -o gnovm/tests/files/iface_embed_sel_order.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/tests/iface_embed_sel_order.gno
go test -v -run 'TestFiles/iface_embed_sel_order.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/iface_embed_sel_order.gno
```

```
--- FAIL: TestFiles/iface_embed_sel_order.gno (0.00s)
    files_test.go:129: unexpected panic: main/iface_embed_sel_order.gno:23:9-14: cannot access interface {sec func() int; sec func() int}.sec from main
```

A go/build oracle accepts both orders. With the localized fix the same command prints `ok`, and `iface_embed_xpkg_access` / `iface_embed_xpkg` / `iface_embed_xpkg2` stay green. Test: [`iface_embed_sel_order.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/tests/iface_embed_sel_order.gno) · [↗](tests/iface_embed_sel_order.gno).
</details>
