# Review: [#6074](https://github.com/gnolang/gno/pull/6074)
Target: https://github.com/gnolang/gno/pull/6074
Posted: https://github.com/gnolang/gno/pull/6074#discussion_r3871105106
Event: standalone inline comment on 710cdabae, no review body

## examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:11 [gh](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L11)
[AI review]

Suggestion: `Collection` is now one `*grc721.Token` field. `Token()` returns it, `Extension` renames `token.ExtensionView`, `HasExtension` compares it against nil. Only `Kinds` adds anything, and it is a sort and a nil filter.

Move `Kinds` into the core beside `ExtensionKinds`, and #6075 can store the token itself.
