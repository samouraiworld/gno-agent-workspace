# Review: PR #5739
Posted: https://github.com/gnolang/gno/pull/5739#pullrequestreview-4520219411
Event: REQUEST_CHANGES

## Body
Verified on 155f1a7 against a side-by-side Go run and current master:

```
                 struct{ Int }     interface{ SAlias }
                 vs struct{ int }   vs interface{ Stringer }
  Go             distinct  ✓        identical ✓
  master (pre)   identical ✗        identical ✓
  PR 155f1a7     distinct  ✓        distinct  ✗  <- new divergence
```

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/1-155f1a7/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/types.go:2658-2661 [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2658) [posted](https://github.com/gnolang/gno/pull/5739#discussion_r3431959329)
Embedding an aliased interface inside an interface (`type SAlias = Stringer; interface{ SAlias }`) derives its identity from the written name here, so it no longer equals `interface{ Stringer }`, which Go and master treat as one type. For an embedded interface, derive identity from the resolved type, not the source spelling, so structurally identical interfaces keep one TypeID; the struct-embed renaming is correct and stays.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5739 -R gnolang/gno
cat > gnovm/tests/files/iface_alias_id.gno <<'EOF'
package main

type Stringer interface{ Str() string }
type SAlias = Stringer

func main() {
	var x, y interface{} = struct{ A interface{ SAlias } }{}, struct{ A interface{ Stringer } }{}
	println("alias==canonical:", x == y)
}

// Output:
// alias==canonical: true
EOF
go test -run 'TestFiles/iface_alias_id.gno$' ./gnovm/pkg/gnolang/ 2>&1 | grep -E 'Expected|Actual|alias==canonical'
rm gnovm/tests/files/iface_alias_id.gno
```

```
--- Expected
+++ Actual
-alias==canonical: true
+alias==canonical: false
```
Go prints `alias==canonical: true` for the same program; master prints `true` too. Asserts the Go result, so the filetest fails at 155f1a7 and passes once interface-embed identity stops using the written name.
</details>
