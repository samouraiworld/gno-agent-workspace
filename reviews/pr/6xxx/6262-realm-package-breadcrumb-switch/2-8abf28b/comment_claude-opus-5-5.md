# Review: [#6262](https://github.com/gnolang/gno/pull/6262)
Posted: https://github.com/gnolang/gno/pull/6262#pullrequestreview-5458892490
Event: COMMENT
Verdict: APPROVE. No Warning remains: the "N matching" link now opens a page listing what it counts, and what is left is one Suggestion on the twinless walk and one Nit on the menu's path line.
Model: claude-opus-5-5, solo review
Commit: 8abf28b6ad75f68641f9acd7f9dbe3edd098d394
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6262 8abf28b6a`
Round: 2, over the fix commits alone (d631c4496..8abf28b6a, master's merge left out). One finder, the judge answering the completeness question in place of a reflector, 3 candidates, each run at the head and the base by an agent that was not its finder. Round 1, each from a run at the head: counterpart.go:97 Warning resolved (`TestCounterpart_LinkOpensWhatItCounts` passes and `/r/tests/vm$source` lists both counted realms); 06-blocks.css:3991 Nit resolved (the `@supports not` block follows the `.b-kind-switch` rule in `public/main.css`, and the finder's Chromium run puts the menu at 56px/64px without `anchor()`); counterpart.go:90 Nit resolved (the subtree case asserts "3 matching packages" and passes); counterpart.go:213 and counterpart.go:247 Suggestions carried, open, since the diff's only `counterpart.go` hunk ends at `counterpartLink` and the author defers both.

## Body

> AI review, claude-opus-5-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/2-8abf28b/claims.md) · Status: APPROVE

## gno.land/pkg/gnoweb/counterpart.go:121-122 [gh](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/counterpart.go#L121-L122) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L121) · Suggestion [posted](https://github.com/gnolang/gno/pull/6262#discussion_r4220790553)
Suggestion: `return dir, 1` offers one realm on a twinless `/p/tests/vm/foo` while `/r/tests/vm$source#subpackages` lists the two that match below it.
Linking that section when two or more direct children match, as the twin branch at [`counterpart.go:103-104`](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/counterpart.go#L103-L104) does, keeps the `/r/gov/dao` case at "Matching realm":

```suggestion
		case n > 1 && slices.Contains(members, dir):
			children := 0
			for _, m := range members {
				if gopath.Dir(m) == dir {
					children++
				}
			}
			if children > 1 {
				return dir + "$source#subpackages", children
			}
			return dir, 1
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6262 -R gnolang/gno
cat > gno.land/pkg/gnoweb/zz_twinless_test.go <<'EOF'
package gnoweb

import (
	"fmt"
	"testing"
)

func TestTwinlessPackageDir(t *testing.T) {
	tgt, n := counterpartTarget("/r/tests/vm/foo", "/r/tests/vm",
		[]string{"/r/tests/vm", "/r/tests/vm/crossrealm", "/r/tests/vm/subtests"})
	fmt.Printf("target=%q n=%d label=%q\n", tgt, n, counterpartLink(tgt, "/r/tests/vm", n).Label)
}
EOF
(cd gno.land && go test ./pkg/gnoweb -run TestTwinlessPackageDir -count=1 -v | grep target=)
rm gno.land/pkg/gnoweb/zz_twinless_test.go
```

The walk stops at `/r/tests/vm` and names it alone, though two of its direct children match:

```text
target="/r/tests/vm" n=1 label="Matching realm"
```

With the suggestion applied the same input gives `target="/r/tests/vm$source#subpackages" n=2 label="2 matching realms"`, and `go test ./pkg/gnoweb -run Counterpart` stays green, the `/r/gov/dao` case of `TestCounterpart_LinkOpensWhatItCounts` included.

</details>

## gno.land/pkg/gnoweb/counterpart.go:104 [gh](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/counterpart.go#L104) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L104) · Nit [posted](https://github.com/gnolang/gno/pull/6262#discussion_r4220790594)
Nit: the menu's path line prints this target whole, so the item under "2 matching realms" reads `/r/tests/vm$source#subpackages` where every other item shows a plain path.

<details><summary>rendered menu</summary>

[`header.html:255`](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/components/layouts/header.html#L255) prints the path line from `.URL`. `/p/tests/vm/crossrealm` served from a `MockClient` holding `/r/tests/vm`, `/r/tests/vm/crossrealm`, `/r/tests/vm/subtests` and `/p/tests/vm/crossrealm`, primary item read from the HTML:

```text
href="/r/tests/vm$source#subpackages" label="2 matching realms" item-path="/r/tests/vm$source#subpackages"
```

</details>

## SKIP gno.land/pkg/gnoweb/counterpart.go:122 [gh](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/counterpart.go#L122) · [↗](../../../../../.worktrees/gno-review-6262/gno.land/pkg/gnoweb/counterpart.go#L122) · Nit
Nit: on a listing cut at 100 paths, `return dir, 1` labels the link "Matching package" while the twin sits past the cut.

Not posted: the link opens the same single package either way, the "100+" label it replaces promised a listing that page never showed, and the twin lost past the 100-path cut is the cap's behaviour, which the fix commits do not touch.
