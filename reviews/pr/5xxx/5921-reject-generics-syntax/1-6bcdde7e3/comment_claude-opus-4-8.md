# Review: PR [#5921](https://github.com/gnolang/gno/pull/5921)
Event: APPROVE

## Body
Reproduced on 6bcdde7e3: removing the [`checkNoGenerics`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/pkg/gnolang/nogenerics.go#L23) call, `type Foo[T any] int` and `interface{ int | string }` pass go/types and then panic in the gno preprocessor; with the call each fails add-package with a located type-check error.

The red `build` check is unrelated: [`misc/gendocs`](https://github.com/gnolang/gno/blob/6bcdde7e3/misc/gendocs/Makefile) installs [`golang.org/x/pkgsite`](https://pkg.go.dev/golang.org/x/pkgsite), whose latest now needs go1.26 while CI runs go1.25.9.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5921-reject-generics-syntax/1-6bcdde7e3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/nogenerics.go:17-19 [↗](../../../../../.worktrees/gno-review-5921/gnovm/pkg/gnolang/nogenerics.go#L17)
Suggestion: The comment calls bare non-interface type-set elements undetectable syntactically, but that holds only for the bare-ident case: `interface{ *int }`, `interface{ []byte }`, and `interface{ struct{} }` also slip through, and none can be a legal embedded interface. Reject any embedded element that is not a plain or qualified identifier, or narrow the comment to the bare-ident case.
