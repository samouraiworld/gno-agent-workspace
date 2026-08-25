# Review: PR [#5871](https://github.com/gnolang/gno/pull/5871)
Posted: https://github.com/gnolang/gno/pull/5871#pullrequestreview-5016272715
Event: COMMENT

## Body
[AI review]

- [`r/docs/security_patterns`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L61) guards `assertAdmin` with `cur.IsCurrent()` on the frame's own `cur` and [advertises that check as defence one](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L37-L39), while [the new index](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/home/home.gno#L49) links it as the security page. That realm and the `current-guard` rule arrived together in [#5835](https://github.com/gnolang/gno/pull/5835), so the guard and the rule are one decision: deleting it while the rule stands puts the realm back on the harness's vulnerable side.
- [`p/samcrew/piechart/README.md`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/p/samcrew/piechart/README.md?plain=1#L40) links `/r/docs/charts:piechart`, and `charts` is deleted here rather than revived. Absent packages account for the gauge half, which imports `p/samcrew/gauge`; the piechart half imports [`p/samcrew/piechart`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/p/samcrew/piechart/piechart.gno#L1), which is in the tree.

## AGENTS.md:108 [gh](https://github.com/gnolang/gno/blob/7c41a297d/AGENTS.md?plain=1#L108) · [↗](../../../../../.worktrees/gno-review-5871/AGENTS.md#L108) [posted](https://github.com/gnolang/gno/pull/5871#discussion_r3850704344)
Already raised: https://github.com/gnolang/gno/pull/5871#discussion_r3850382486

The [`current-guard` fixture pair](https://github.com/gnolang/gno/blob/7c41a297d/misc/audit-pattern-harness/fixtures/current-guard/vulnerable/admin.gno#L5-L11) names a crossing function reading its own first `cur` as the vulnerable side, and that fixture is not vulnerable, so the rule this line reverses is the one that should move. Running it over `examples/gno.land/r/docs/` at this head reports 24 hits across 12 files, every one a first `cur`.

<details><summary>repro</summary>

The merged fixture is not vulnerable. Thirteen routes to its first `cur`, deployed verbatim as a realm:

| # | Route | Refused at | Outcome |
|---|---|---|---|
| 1 | `r := unsafe.CurrentRealm(); F(cross(r))` | type-check | `runtime.Realm does not implement gno0p9.realm (missing method IsCurrent)` |
| 2 | `r := unsafe.PreviousRealm(); F(cross(r))` | type-check | same as 1 |
| 3 | call `F` from `Render`, no realm in scope | type-check | `not enough arguments in call to F` |
| 4 | `F(cross(cur.Previous()))` | preprocess | `cross argument must be a bare realm-typed identifier` |
| 5 | `p := cur.Previous(); F(p)` | preprocess | ``only `cur` or `cross(rlm)` are allowed as the first argument`` |
| 6 | `p := cur.Previous(); F(cross(p))` | runtime, in `cross` | `cross: rlm is not the current cur (stale capture or sibling frame)` |
| 7 | `h(0, cur.Previous())` then `F(cross(rlm))` in `h` | runtime, in `cross` | same as 6 |
| 8 | `var saved realm; saved = cur`, reused next tx | finalize | `cannot persist realm value: realm values are ephemeral and tied to a call frame` |
| 9 | closure capturing `cur`, kept in realm state | finalize | same as 8 |
| 10 | `F(cross(cur))` from a user transaction | arrives | `IsCurrent() == true` |
| 11 | `F(cross(cur))` realm to realm | arrives | `IsCurrent() == true` |
| 12 | `r := cur; F(cross(r))` | arrives | `IsCurrent() == true` |
| 13 | `init(cur realm)` then `F(cross(cur))` | arrives | `IsCurrent() == true` |

Rows 1 and 2 never reach the realm type system: `unsafe.CurrentRealm()` and `unsafe.PreviousRealm()` return `runtime.Realm`, a plain struct. Rows 7, 10 and 11 are below.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5871 -R gnolang/gno
mkdir -p examples/gno.land/r/zzprobe/admin examples/gno.land/r/zzprobe/attacker
printf 'module = "gno.land/r/zzprobe/admin"\ngno = "0.9"\n'    > examples/gno.land/r/zzprobe/admin/gnomod.toml
printf 'module = "gno.land/r/zzprobe/attacker"\ngno = "0.9"\n' > examples/gno.land/r/zzprobe/attacker/gnomod.toml
cp misc/audit-pattern-harness/fixtures/current-guard/vulnerable/admin.gno examples/gno.land/r/zzprobe/admin/
cat >> examples/gno.land/r/zzprobe/admin/admin.gno <<'EOF'

func Probe(cur realm) bool { return cur.IsCurrent() }
EOF
cat > examples/gno.land/r/zzprobe/attacker/a.gno <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Attack(cur realm, next address) { launder(0, cur.Previous(), next) }

func launder(_ int, rlm realm, next address) {
	admin.TransferOwnership(cross(rlm), next)
}

func ProbeCrossed(cur realm) bool { return admin.Probe(cross(cur)) }
EOF
cat > examples/gno.land/r/zzprobe/attacker/z_run_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/zzprobe/runx
package runx

import (
	"gno.land/r/zzprobe/admin"
	"gno.land/r/zzprobe/attacker"
)

func main(cur realm) {
	println("guard would read, user tx  ->", admin.Probe(cross(cur)))
	println("guard would read, realm    ->", attacker.ProbeCrossed(cross(cur)))
	attacker.Attack(cross(cur), address("g1evilevilevilevilevilevilevilevilevil0"))
}
EOF
(cd examples && gno test -v ./gno.land/r/zzprobe/attacker/) 2>&1 | grep -m3 -E 'guard would read|unexpected panic'
rm -rf examples/gno.land/r/zzprobe
```

The laundering attempt never reaches the fixture's body, and on the paths that do arrive the guard reads `true`.

```
guard would read, user tx  -> true
guard would read, realm    -> true
unexpected panic: cross: rlm is not the current cur (stale capture or sibling frame)
```
</details>

## examples/gno.land/r/docs/routing/routing.gno:29 [gh](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/routing/routing.gno#L29) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/routing/routing.gno#L29) [posted](https://github.com/gnolang/gno/pull/5871#discussion_r3850704358)
[`mux.Router.Render`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/p/nt/mux/v0/router.gno#L44) reads `reqParts[i]` before it breaks out on the `*` segment, so a one-segment request against this two-segment pattern indexes past the end and `/r/docs/routing:wildcard` answers "Error: internal error". That URL is one segment up from [the one this realm's index links](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/routing/routing.gno#L46), and the realm is the only live one registering a wildcard route.

```suggestion
	router.HandleFunc("wildcard", wildcardHandler)
	router.HandleFunc("wildcard/*", wildcardHandler)
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5871 -R gnolang/gno
python3 - <<'PY'
p = "gno.land/pkg/gnoweb/app_test.go"
s = open(p).read()
old = '{"/r/demo/disperse", http.StatusOK, "gnomod.toml"}'
s = s.replace(old, old + ',\n\t\t\t{"/r/docs/routing:wildcard", http.StatusOK, "Wildcard handler"},\n\t\t\t{"/r/docs/routing:wildcard/foo", http.StatusOK, "Wildcard handler"}', 1)
open(p, "w").write(s)
PY
go test ./gno.land/pkg/gnoweb/ -run TestRoutes -v 2>&1 | grep -E 'unable to fetch realm|routing:wildcard'
git checkout -- gno.land/pkg/gnoweb/app_test.go
```

The bare segment fails while the same route one segment deeper passes, and the page gnoweb returns reads "Error: internal error. Something went wrong."

```
# … level=ERROR source=gno.land/pkg/gnoweb/handler_http.go:469
msg="unable to fetch realm" error="RPC node response error: runtime error: slice index out of bounds: 1 (len=1)" path=/r/docs/routing:wildcard
    --- FAIL: TestRoutes/test_route_/r/docs/routing:wildcard (0.01s)
    --- PASS: TestRoutes/test_route_/r/docs/routing:wildcard/foo (0.01s)
```
</details>

## examples/gno.land/r/docs/complexargs/complexargs.gno:74 [gh](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/complexargs/complexargs.gno#L74) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/complexargs/complexargs.gno#L74) [posted](https://github.com/gnolang/gno/pull/5871#discussion_r3850704373)
`InlineText` escapes a backtick to `` \` `` and a backslash is literal inside a code span, so one backtick in this name closes the span early and the leftover backtick pairs with the next one along the line. The [helper table](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L53) names `InlineCode` for this slot, and the same pairing already sits at [`registry.gno:149`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/registry/registry.gno#L149) and [`userprofile.gno:123`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/userprofile/userprofile.gno#L123), where the value is the URL path itself.

```suggestion
		out += "Value of myObject: " + sanitize.InlineCode("CustomType{Name: "+myObject.Name+", Numbers: "+s+"}")
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5871 -R gnolang/gno
d=gno.land/pkg/gnoweb/markdown/golden/sanitize
cat > $d/zz-codespan-inlinetext.txtar <<'EOF'
// MARKDOWNFUNC: InlineText
// CONTEXT: Value of myObject: `CustomType{Name: %s, Numbers: 1}`
-- input.md --
x`y
EOF
cat > $d/zz-codespan-inlinecode.txtar <<'EOF'
// MARKDOWNFUNC: InlineCode
// CONTEXT: Value of myObject: %s
-- input.md --
CustomType{Name: x`y, Numbers: 1}
EOF
go test ./gno.land/pkg/gnoweb/markdown/ -run TestSanitizeIntegration -update-golden-tests >/dev/null
tail -1 $d/zz-codespan-inlinetext.txtar
tail -1 $d/zz-codespan-inlinecode.txtar
rm $d/zz-codespan-inlinetext.txtar $d/zz-codespan-inlinecode.txtar
```

[`SetMyObject` is unguarded on purpose](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/complexargs/complexargs.gno#L38-L41), so any account sets that name through `MsgRun`. One backtick in it ends the span two words early and leaves the rest of the value as prose, with a stray backtick closing the sentence.

```html
<p>Value of myObject: <code>CustomType{Name: x\</code>y, Numbers: 1}`</p>
<p>Value of myObject: <code>CustomType{Name: x`y, Numbers: 1}</code></p>
```
</details>

## examples/gno.land/r/docs/minisocial/v2/posts.gno:85-86 [gh](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/minisocial/v2/posts.gno#L85-L86) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/minisocial/v2/posts.gno#L85) [posted](https://github.com/gnolang/gno/pull/5871#discussion_r3850704382)
Missing test: restoring master's `post.updatedAt.After(post.createdAt.Add(time.Minute * 10))` here leaves the suite green, because a post that was never edited carries `updatedAt == createdAt` and passes that check at any age.

<details><summary>test cases</summary>

[`SkipHeights`](https://github.com/gnolang/gno/blob/7c41a297d/gnovm/tests/stdlibs/testing/context_testing.gno#L105-L110) advances block time five seconds a height, so 240 is twenty minutes.

```go
func TestUpdateWindowClosesAfterTenMinutes(cur realm, t *testing.T) {
	resetPostsForTest()

	alice := testutils.TestAddress("alice")
	testing.SetRealm(testing.NewUserRealm(alice))
	uassert.NoError(t, CreatePost(cross(cur), "original"))
	id := postID.String()

	testing.SkipHeights(240)

	testing.SetRealm(testing.NewUserRealm(alice))
	err := UpdatePost(cross(cur), id, "late edit")
	uassert.ErrorIs(t, err, ErrUpdateWindowExpired, "edit twenty minutes later must be refused")
}
```

On master's comparison it reports `error mismatch, expected update window expired, got %!s(<nil>)`.
</details>

## examples/gno.land/r/docs/soliditypatterns/banker/banker.gno:23 [gh](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/soliditypatterns/banker/banker.gno#L23) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/soliditypatterns/banker/banker.gno#L23) [posted](https://github.com/gnolang/gno/pull/5871#discussion_r3850704387)
Missing test: nothing asserts that this guard refuses the `maketx run` script `IsUser()` would admit, which is the whole distinction the page teaches. `banker` and [`reentrancy`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy.gno#L1) are the two soliditypatterns realms that move coins and the two still shipping no `_test.gno`, while `counter`, `ownable` and `statelock` each gained one here.

## examples/gno.land/r/docs/minisocial/v1/posts_test.gno:36-45 [gh](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/minisocial/v1/posts_test.gno#L36-L45) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/minisocial/v1/posts_test.gno#L36) [posted](https://github.com/gnolang/gno/pull/5871#discussion_r3850704390)
Nit: [`admin_test.gno`'s `TestResetPostsUnauthorized`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/minisocial/v1/admin_test.gno#L13-L28) asserts the same rejection and adds a survival check on top. [`resetPostsForTest`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/minisocial/v1/posts_test.gno#L12-L16) exists for this one call, and its comment promises an isolation the file's other two tests never take.

## examples/gno.land/r/docs/home/home_test.gno:31 [gh](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/home/home_test.gno#L31) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/home/home_test.gno#L31) [posted](https://github.com/gnolang/gno/pull/5871#discussion_r3850704395)
Nit: the closing paren narrows this entry, so a link written with a sub-path such as `/r/docs/avl_pager:2` walks through the assertion.

```suggestion
		"/r/docs/avl_pager",
```
