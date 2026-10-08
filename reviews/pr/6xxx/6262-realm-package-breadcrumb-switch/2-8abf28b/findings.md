# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 0 refuted kept out

## gno.land/pkg/gnoweb/counterpart.go:104 [gh](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/counterpart.go#L104) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the menu's path line prints /r/tests/vm$source#subpackages, gnoweb's webquery and fragment included
Check: render /p/tests/vm/crossrealm with a MockClient fixture and read the item-path span of the primary item, at head and base
Details: counterpart.go:104 returns `dir + "$source#subpackages"` as the target, counterpartLink sets URL to it, and header.html:255 prints `<span class="item-path">{{ .URL }}</span>`, so the path line under "2 matching realms" reads /r/tests/vm$source#subpackages where every other item's path line is a plain path.
Evidence: head: `/p/tests/vm/crossrealm: href="/r/tests/vm$source#subpackages" label="2 matching realms" item-path="/r/tests/vm$source#subpackages"`; base: `item-path="/r/tests/vm"`
Artifact: tests/judge-1-2-menu_test.go

## gno.land/pkg/gnoweb/counterpart.go:122 [gh](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/counterpart.go#L122) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: on a listing capped at 100 paths with the project root a package, the label drops from "100+ matching packages" to "Matching package"
Check: counterpartTarget("/p/alice/golf/zz", "/p/alice/golf", [/p/alice/golf + 99 /p/alice/golf/aNN]) at head and base
Details: Mechanism holds: the twin past the cap is absent, the walk reaches /p/alice/golf with n=100 and :121-122 returns (dir, 1). Both sides open the same single package /p/alice/golf; the base's "100+" label promised a listing that page never showed, so the head's label is the accurate one for what opens. The twin hidden by the 100-path cap is the round-1 cap behaviour, untouched by the fix commits. The sibling and subtree floors without "+" predate the fix (the base counted siblings from the same capped list). Kept as a Nit, shipped SKIP.
Evidence: head: `c3 capped listing (100 paths), twin past the cap: target="/p/alice/golf" n=1 label="Matching package"`; base d631c4496: `target="/p/alice/golf" n=100 label="100+ matching packages"`
Artifact: tests/judge-1-3-counterpart-target_test.go

## gno.land/pkg/gnoweb/counterpart.go:121 [gh](https://github.com/gnolang/gno/blob/8abf28b6ad75f68641f9acd7f9dbe3edd098d394/gno.land/pkg/gnoweb/counterpart.go#L121) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: a twinless walk reaching a package directory names it alone even when its Directories section lists the matches
Check: counterpartTarget("/r/tests/vm/foo", "/r/tests/vm", [vm, vm/crossrealm, vm/subtests]) and the rendered menu on /p/tests/vm/foo, at head and base; then /r/tests/vm$source checked for both children
Details: counterpart.go:121-122 `case n > 1 && slices.Contains(members, dir): return dir, 1`. On /p/tests/vm/foo with /r/tests/vm, /r/tests/vm/crossrealm and /r/tests/vm/subtests live and no /r/tests/vm/foo, the menu reads "Matching realm" and opens /r/tests/vm, while /r/tests/vm$source lists both children; the twin branch at :103-104 links that section in the same shape. Counting direct children and linking dir$source#subpackages when there are two or more turns the label into "2 matching realms" and keeps the /r/gov/dao case (grandchildren only) at "Matching realm": tests/judge-1-fix.patch, go test ./pkg/gnoweb -run 'Counterpart|TestJudge' ok.
Evidence: head: `c1 twinless, dir has direct children: target="/r/tests/vm" n=1 label="Matching realm"`, `/r/tests/vm$source lists /r/tests/vm/crossrealm: true`, `... subtests: true`; base d631c4496: `target="/r/tests/vm" n=3 label="3 matching realms"`; with judge-1-fix.patch: `target="/r/tests/vm$source#subpackages" n=2 label="2 matching realms"`, all Counterpart tests ok
Artifact: tests/judge-1-3-counterpart-target_test.go
