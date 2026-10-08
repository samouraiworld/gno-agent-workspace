# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 0 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_buttons.go:121 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L121) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: isVisibleRune treats a lone U+2800, U+034F, U+FE0F, U+E0100, U+17B4 or U+180B as visible, so each renders a button with no readable text
Check: go test ./gno.land/pkg/gnoweb/markdown -run TestFinder1BlankButton -v (from the repo root; the finder's `cd gno.land` fails, there is no gno.land/go.mod)
Details: buttonLabel drops Cf and control runes and isVisibleRune rejects spaces and four Hangul fillers only. U+034F, U+17B4 (Other_Default_Ignorable_Code_Point), U+FE0F, U+E0100, U+180B (Variation_Selector) and U+2800 (So) pass both. Replacing the switch with `r == 0x2800` plus `!unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector)` rejects all six and keeps TestGnoExtension, TestButton*, TestSanitizeIntegration and TestParseButton* green (tests/judge-1-default-ignorable.patch, 2 insertions, 3 deletions).
Evidence: head: 6 label subtests FAIL, e.g. `blank button rendered: <p><a href="/r/test" class="gno-button">⠀</a></p>`; with the patch only the two href subtests fail
Artifact: tests/solo-finder-1-blank-button.go
State: CONFIRMED, band: Nit, angle: reach
TL;DR: same defect as candidate 1, one edit to isVisibleRune closes both
Check: same run as candidate 1, label subtests U+2800, U+034F, U+FE0F
Details: Re-anchored from 124 to 121, the switch the edit replaces. The three runes the candidate names are three of candidate 1's six; the same run reddens them.
Evidence: label_U+2800_braille_blank, label_U+034F_grapheme_joiner, label_U+FE0F_variation_selector FAIL at head and at ab27ce5c0
Artifact: tests/solo-finder-1-blank-button.go

## gno.land/pkg/gnoweb/markdown/ext_buttons.go:136 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L136) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: an href decoding to U+00A0 or U+3000 renders a button to %C2%A0 or %E3%80%80, since trimLeadingControlAndSpace strips bytes <= ' ' only
Check: TestFinder1BlankButton href subtests
Details: Reproduces. Ships SKIP: a URL parser strips only ASCII whitespace from an href, so U+00A0 is a path character and the button leads to a missing page, as any unresolvable href does; the `&#32;` case the godoc names is rejected (invalid_href_blank_after_decode golden green).
Evidence: href_U+00A0_no-break_space: `<p><a href="%C2%A0" class="gno-button">Go</a></p>`; href_U+3000: `href="%E3%80%80"`
Artifact: tests/solo-finder-1-blank-button.go
State: CONFIRMED, band: Nit, angle: reach
TL;DR: same defect as candidate 2
Check: TestFinder1BlankButton href_U+00A0_no-break_space
Details: Re-anchored from 135 to 136, the emptiness check. U+2060 in an href was not run.
Evidence: href_U+00A0_no-break_space FAIL at head and at ab27ce5c0
Artifact: tests/solo-finder-1-blank-button.go

## gnovm/stdlibs/chain/markdown/markdown.go:389 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gnovm/stdlibs/chain/markdown/markdown.go#L389) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: `[evil]: <gno-button x>` comes out as `x>` at head and empty at the base; no reference binds
Check: tests/judge-2001-lrd-leftover.go at head and ab27ce5c0
Details: Ships SKIP: the leftover is two characters of the user's own text and the brackets are removed, so nothing binds to realm chrome.
Evidence: head: "[evil]: <gno-button x>\n" -> B="x>\n"; ab27ce5c0: B=""
Artifact: tests/judge-2001-lrd-leftover.go

## gno.land/pkg/gnoweb/markdown/ext_buttons.go:98 [gh](https://github.com/gnolang/gno/blob/a4be1f67b066a1f0a28ed57d854c8d96c775965c/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L98) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: StripBidiAndZeroWidth removes 15 runes, every one Cf, which the strings.Map Cf case drops anyway
Check: enumerate U+0000..U+10FFFF through StripBidiAndZeroWidth; drop line 98's call and run the package's button and golden tests
Details: With `label := string(decoded)` (1 insertion, 2 deletions, the chainmd import stays for ext_foreign.go) TestGnoExtension, TestButton*, TestSanitizeIntegration and TestParseButton* stay green. Ships SKIP: the call ties the label to the set sanitize strips, which the godoc names as its intent, and nothing renders differently.
Evidence: TestJudgeEnumStrip: `stripped=15 notCf=0`; rewrite run: only the finder's own 8 subtests fail, as at head
