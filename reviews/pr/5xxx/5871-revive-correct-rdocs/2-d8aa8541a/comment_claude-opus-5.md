# Review: PR [#5871](https://github.com/gnolang/gno/pull/5871)
Event: REQUEST_CHANGES

## Body
- [`reentrancy.gno:1`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy.gno#L1) opens with "explains why Gno is not exposed to Solidity-style reentrancy", the claim [line 43](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy.gno#L43) denies, and [`TestRenderDoesNotOverstateSafety`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy_test.gno#L14) reads `Render("")`, which is the one string the package doc is not in.

## misc/audit-pattern-harness/internal/auditpattern/run.go:300 [gh](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/internal/auditpattern/run.go#L300) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/internal/auditpattern/run.go#L300)
`guardedRealmParams` fills `guarded` only from a line whose trimmed prefix is `func `, so a realm parameter on a func literal is never checked: [`tellers.gno:17`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L17) resolves caller identity from `rlm.Previous().Address()` and the rule stays silent. Matching a `func(` literal as well as a `func ` declaration closes it, and `go/parser` closes the one-line-signature limit with it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5871 -R gnolang/gno
cat > misc/audit-pattern-harness/internal/auditpattern/zz_literal_test.go <<'EOF'
package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

func scanOne(t *testing.T, src string) int {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.gno"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err := RunRule("current_guard", dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(hits)
}

func TestZZLiteral(t *testing.T) {
	const decl = "package x\n\n" +
		"func accountFn(_ int, rlm realm) address {\n" +
		"\treturn rlm.Previous().Address()\n" +
		"}\n"
	const lit = "package x\n\n" +
		"func teller() *fnTeller {\n" +
		"\treturn &fnTeller{\n" +
		"\t\taccountFn: func(_ int, rlm realm) address {\n" +
		"\t\t\treturn rlm.Previous().Address()\n" +
		"\t\t},\n" +
		"\t}\n" +
		"}\n"
	t.Logf("declared func: %d hits", scanOne(t, decl))
	t.Logf("func literal:  %d hits", scanOne(t, lit))
}
EOF
(cd misc/audit-pattern-harness && go test ./internal/auditpattern -run TestZZLiteral -v) | grep hits
rm misc/audit-pattern-harness/internal/auditpattern/zz_literal_test.go
```

The same body scores a hit as a declaration and none as a literal, which is the shape a non-crossing callback is forced into: a realm-typed first parameter would make it a crossing function.

```
    zz_literal_test.go:34: declared func: 1 hits
    zz_literal_test.go:35: func literal:  0 hits
```

`expected/current-guard.yaml` calls the family "secondary realm parameter trusted without IsCurrent", and the README's two documented limits are the same-function scope and the one-line signature, neither of which covers a closure.
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:274 [gh](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/internal/auditpattern/run.go#L274) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/internal/auditpattern/run.go#L274)
`realmValueRead` matches three of the eleven methods a realm value answers, so a secondary parameter read through `String()`, `Sub()`, `Subpath()` or the four `Is*` predicates passes unguarded: [`r/sys/users/errors.gno:48`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/sys/users/errors.gno#L48) stores `caller.String()` off one and scores no hit. Treating any `rlm.` selector other than `IsCurrent()` as a read needs no list and does not go stale when the realm type gains a method.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5871 -R gnolang/gno
cat > misc/audit-pattern-harness/internal/auditpattern/zz_accessor_test.go <<'EOF'
package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZZAccessors(t *testing.T) {
	for _, accessor := range []string{
		"Previous()", "Address()", "PkgPath()", "String()",
		"Sub(\"treasury\")", "Subpath()",
		"IsUser()", "IsUserCall()", "IsUserRun()", "IsCode()",
	} {
		dir := t.TempDir()
		src := "package x\n\nfunc f(_ int, rlm realm) {\n\t_ = rlm." + accessor + "\n}\n"
		if err := os.WriteFile(filepath.Join(dir, "a.gno"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		hits, err := RunRule("current_guard", dir)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("rlm.%-18s %d hits", accessor, len(hits))
	}
}
EOF
(cd misc/audit-pattern-harness && go test ./internal/auditpattern -run TestZZAccessors -v) | grep hits
rm misc/audit-pattern-harness/internal/auditpattern/zz_accessor_test.go
```

Seven of the ten reads are silent. `Sub()` mints a sub-realm identity from the value and the `Is*` predicates answer authorization questions, which is what [quick check 2](https://github.com/gnolang/gno/blob/d8aa8541a/docs/resources/gno-ai-contract-review.md?plain=1#L52-L60) is about.

```
    zz_accessor_test.go:24: rlm.Previous()         1 hits
    zz_accessor_test.go:24: rlm.Address()          1 hits
    zz_accessor_test.go:24: rlm.PkgPath()          1 hits
    zz_accessor_test.go:24: rlm.String()           0 hits
    zz_accessor_test.go:24: rlm.Sub("treasury")    0 hits
    zz_accessor_test.go:24: rlm.Subpath()          0 hits
    zz_accessor_test.go:24: rlm.IsUser()           0 hits
    zz_accessor_test.go:24: rlm.IsUserCall()       0 hits
    zz_accessor_test.go:24: rlm.IsUserRun()        0 hits
    zz_accessor_test.go:24: rlm.IsCode()           0 hits
```
</details>

## examples/gno.land/r/docs/complexargs/complexargs.gno:74 [gh](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/complexargs/complexargs.gno#L74) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/complexargs/complexargs.gno#L74)
Missing test: [`z_filetest.gno`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/complexargs/z_filetest.gno#L11) sets the name to `Bob`, whose golden output is byte-identical under the `InlineText` line this replaces, so nothing holds the fence that survives a backtick in a name any account can set.

<details><summary>test cases</summary>

```go
// PKGPATH: gno.land/r/docs/complexargs/z_sanitize_filetest

package z_sanitize_filetest

import (
	"strings"

	"gno.land/r/docs/complexargs"
)

// SetMyObject is unguarded on purpose, so any account sets Name through
// MsgRun. A backtick in the name must not close the code span early.
func main(cur realm) {
	obj := complexargs.NewCustomType("x`y", []int{1})
	complexargs.SetMyObject(cross(cur), obj)
	out := complexargs.Render("")
	println(out[strings.LastIndex(out, "Value of myObject: "):])
}

// Output:
// Value of myObject: ``CustomType{Name: x`y, Numbers: 1}``
```

On the line this replaces, the same filetest reports:

```
-Value of myObject: ``CustomType{Name: x`y, Numbers: 1}``
+Value of myObject: `CustomType{Name: x\`y, Numbers: 1}`
```
</details>
