# Review: PR #5739
Posted: https://github.com/gnolang/gno/pull/5739#pullrequestreview-4604057569
Event: APPROVE

## Body
Verified at a00dde6b3.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/types.go:1062-1077 [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1062) [posted](https://github.com/gnolang/gno/pull/5739#discussion_r3543375014)
Minor: The loop returns on the first entry named `sec`, so the result depends on method order. Unexported identity is package-scoped, so `ifaceext.sec` and `main.sec` are two different methods with the same name. In `interface{ ifaceext.Sec; sec() int }` the embedded `ifaceext.sec` comes first, fails the origin gate, and `x.sec()` from main is rejected; put the own `sec()` first and it compiles. Go accepts both orders. 

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5739 -R gnolang/gno
cat > gnovm/tests/files/iface_embed_sel_order.gno <<'EOF'
package main

import "filetests/extern/ifaceext"

// embed-first: foreign sealed ifaceext.sec listed before main's own sec.
func selEmbedFirst(x interface {
	ifaceext.Sec
	sec() int
}) int {
	return x.sec()
}

// own-first: main's own sec listed before the foreign embed (compiles today).
func selOwnFirst(x interface {
	sec() int
	ifaceext.Sec
}) int {
	return x.sec()
}

func main() {
	_, _ = selEmbedFirst, selOwnFirst
	println("ok")
}

// Output:
// ok
EOF
go test -v -run 'TestFiles/iface_embed_sel_order.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/iface_embed_sel_order.gno
```

```
--- FAIL: TestFiles/iface_embed_sel_order.gno (0.00s)
    files_test.go:129: unexpected panic: main/iface_embed_sel_order.gno:23:9-14: cannot access interface {sec func() int; sec func() int}.sec from main
```

A go/build oracle accepts both orders. With the localized fix the same command prints `ok`, and `iface_embed_xpkg_access` / `iface_embed_xpkg` / `iface_embed_xpkg2` stay green. Test: [`iface_embed_sel_order.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/tests/iface_embed_sel_order.gno) · [↗](tests/iface_embed_sel_order.gno).
</details>
