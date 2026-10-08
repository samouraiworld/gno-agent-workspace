# Review: [#6298](https://github.com/gnolang/gno/pull/6298)

Event: COMMENT
Verdict: NEEDS DISCUSSION. The open Warning is carried from round 1: gnoweb renders `<gno-button>` as soon as it deploys, and only the ADR's upgrade note, no code, keeps sanitized user text from rendering as a button until the chain runs the new escape; the one new posted finding is a Nit.
Model: claude-opus-5-5, solo review
Commit: a4be1f67b066a1f0a28ed57d854c8d96c775965c
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6298 a4be1f67b`
Round: 2, scoped to the fix commits, ab27ce5c0..a4be1f67b. 2 finders, a reflector pass by the judge, 6 candidates (5 from the finders, 1 from the reflector), each run again by an agent that was not its finder; 0 refuted, 1 posted. Round 1's posted sections: `ext.go:96` Warning open, carried (ext.go is not in the diff; the ADR's Upgrade ordering note names the dependency and no code gates it); `markdown.go:427` Warning resolved (the four pointy-link cases of TestEscapeBlockHazards pass at the head and fail at ab27ce5c0, sanitize goldens green); ADR `227-230` Nit resolved (the Upgrade ordering note names the order and the on-chain output change); `06-blocks.css:3270` Nit resolved (headless Chromium computes margin 0px for the icon inside a button, `.c-realm-view .gno-button.gno-button > .tooltip` outranking `a > .tooltip:last-of-type`); `ext_buttons.go:72` Nit resolved for `&#32;` (invalid_href_blank_after_decode golden green), Unicode blanks drafted SKIP below; `ext_buttons.go:96` Nit resolved (valid_label_entities_attribute_rules golden green, `&not=` kept); `ext_buttons.go:97` Nit partly (the four format characters are rejected, six other blank runes posted at `ext_buttons.go:117-124`); `ext_buttons_test.go:113` Nit resolved (the error is checked, the package's tests build and pass); `markdown.go:438` Nit resolved (the comment describes the visible backslash the four-spaces golden records); `utils.go:105` Suggestion open, carried, declined by the author for a joint change with #6297 and #6299.

## Body

## gno.land/pkg/gnoweb/markdown/ext_buttons.go:117-124 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L117-L124) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L117) · Nit

Nit: `isVisibleRune` counts six blank runes as visible, U+2800, U+034F, U+FE0F, U+E0100, U+17B4 and U+180B, so `label="&#x2800;"` renders an empty button.

```suggestion
// isVisibleRune reports whether r draws something: not a space, not a
// default-ignorable code point (Hangul fillers and variation selectors
// included) and not U+2800 BRAILLE PATTERN BLANK.
func isVisibleRune(r rune) bool {
	if r == 0x2800 {
		return false
	}
	return !unicode.IsSpace(r) && !unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector)
}
```

## SKIP gno.land/pkg/gnoweb/markdown/ext_buttons.go:136 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L136) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L136) · Nit

Nit: `trimLeadingControlAndSpace` strips only bytes up to `' '`, so `href="&#xA0;"` or `href="&#x3000;"` passes `len(dest) == 0` and renders a button linking to `%C2%A0` or `%E3%80%80`.

Not posted: a browser's URL parser strips only ASCII whitespace too, so this href is a path that resolves to no page like any other, and the `&#32;` case the godoc names is rejected.

## SKIP gnovm/stdlibs/chain/markdown/markdown.go:389 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gnovm/stdlibs/chain/markdown/markdown.go#L389) · [↗](../../../../../.worktrees/gno-review-6298/gnovm/stdlibs/chain/markdown/markdown.go#L389) · Nit

Nit: `escapeGnoButtonTags` running ahead of the bracket walker turns `[evil]: <gno-button x>` into a reference definition that ends at the space, so `Block` strips it and leaves `x>` as text.

Not posted: the brackets are gone, so nothing binds to the realm's references, and the leftover is two characters of the user's own text.

## SKIP gno.land/pkg/gnoweb/markdown/ext_buttons.go:98 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L98) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L98) · Suggestion

Suggestion: `chainmd.StripBidiAndZeroWidth` removes 15 runes, all Cf, and the `strings.Map` below already drops every Cf rune.

Not posted: the call ties the label to the set sanitize strips, which the godoc names as its intent, and the label renders the same either way.
