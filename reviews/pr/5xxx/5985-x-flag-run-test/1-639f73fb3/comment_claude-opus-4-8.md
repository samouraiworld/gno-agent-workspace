# Review: PR [#5985](https://github.com/gnolang/gno/pull/5985)
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Repros run at 639f73fb3.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5985-x-flag-run-test/1-639f73fb3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/cmd/gno/test.go:449-453 [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/test.go#L449-L453)
An override name carries no package qualifier, so one flag rewrites the same-named var in every package under test, `_test.gno` and `_filetest.gno` files included. A single `-X Version=1.2.3 ./...` set `Version` in two unrelated packages. The [flag help](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/test.go#L196-L197) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/test.go#L196-L197) cites `go build -ldflags "-X ..."`, whose [documented form](https://pkg.go.dev/cmd/link) is `importpath.name=value` and which rejects the unqualified spelling.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5985 -R gnolang/gno
cat > gnovm/cmd/gno/testdata/test/xscope.txtar <<'EOF'
gno test -v -X Version=1.2.3 ./...

stdout 'pkga=1.2.3'
stdout 'pkgb=1.2.3'

-- gnowork.toml --
gno = "0.9"

-- pkga/gnomod.toml --
module = 'gno.test/r/integ/pkga'

-- pkga/pkga.gno --
package pkga

var Version = "dev"

-- pkga/pkga_test.gno --
package pkga

import "testing"

func TestA(t *testing.T) { println("pkga=" + Version) }

-- pkgb/gnomod.toml --
module = 'gno.test/r/integ/pkgb'

-- pkgb/pkgb.gno --
package pkgb

var Version = "dev"

-- pkgb/pkgb_test.gno --
package pkgb

import "testing"

func TestB(t *testing.T) { println("pkgb=" + Version) }
EOF
go test -v -run 'Test_Scripts/test/xscope$' ./gnovm/cmd/gno/
rm gnovm/cmd/gno/testdata/test/xscope.txtar
```

```
> gno test -v -X Version=1.2.3 ./...
[stdout]
pkga=1.2.3
pkgb=1.2.3
# …
```
</details>

## gnovm/cmd/gno/xflag.go:46-75 [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L46-L75)
A name that matches nothing is silently ignored, so the run stays green on the default value. That includes the qualified `main.myVar=...` form the [Go linker](https://pkg.go.dev/cmd/link) requires. `-X Count=7` on `var Count = 3` and `-X Konst=z` on a `const` are equally quiet, where the linker errors with `main.Count: cannot set with -X: not a var of type string (type:int)`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5985 -R gnolang/gno
cat > gnovm/cmd/gno/testdata/test/xsilent.txtar <<'EOF'
gno run -X main.myVar=ovr .

stdout 'myVar=default'               # IS:     the go-style qualified name is dropped
! stderr 'myVar'                     # IS:     nothing is reported
# stderr 'myVar'                     # SHOULD: unused -X name reported

gno run -X Count=7 -X Konst=z .

stdout 'Count=3'                     # IS:     non-string initializer silently skipped
stdout 'Konst=k'                     # IS:     const silently skipped

-- main.gno --
package main

import "strconv"

var myVar = "default"
var Count = 3

const Konst = "k"

func main() {
	println("myVar=" + myVar)
	println("Count=" + strconv.Itoa(Count))
	println("Konst=" + Konst)
}
EOF
go test -v -run 'Test_Scripts/test/xsilent$' ./gnovm/cmd/gno/
rm gnovm/cmd/gno/testdata/test/xsilent.txtar
```

```
> gno run -X main.myVar=ovr .
[stdout]
myVar=default
Count=3
Konst=k

> gno run -X Count=7 -X Konst=z .
[stdout]
myVar=default
Count=3
Konst=k
```
</details>

## gnovm/cmd/gno/xflag.go:83-88 [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L83-L88)
Reported line numbers stop matching the file on disk, because re-quoting collapses a multi-line raw string initializer to one line and the VM parses that re-render. A panic under a three-line `var Banner` reports `main.gno:7` with `-X` and `main.gno:10` without, and `gno test` puts the same file's type error at line 6 instead of line 9. Splicing the quoted value into the original bytes at the literal's own offsets would leave the rest of the file untouched.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5985 -R gnolang/gno
cat > gnovm/cmd/gno/testdata/test/xlines.txtar <<'EOF'
! gno run .

stderr 'main\.gno:10'

! gno run -X Banner=hi .

stderr 'main\.gno:7'                 # IS:     position shifted by three lines
# stderr 'main\.gno:10'              # SHOULD: position matches the file on disk

-- main.gno --
package main

var Banner = `
  welcome
  to gno
`

func main() {
	println(Banner)
	panic("boom")
}
EOF
go test -v -run 'Test_Scripts/test/xlines$' ./gnovm/cmd/gno/
rm gnovm/cmd/gno/testdata/test/xlines.txtar
```

```
> ! gno run .
[stderr]
panic: boom
main<VPBlock(1,1)>()
    main/main.gno:10

> ! gno run -X Banner=hi .
[stderr]
panic: boom
main<VPBlock(1,1)>()
    main/main.gno:7
```
</details>

## gnovm/cmd/gno/run_test.go:22-33 [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/run_test.go#L22-L33)
Missing test: `gno test -X`. Coverage stops at `gno run`, so deleting the [MemPackage patch loop](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/test.go#L449-L453) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/test.go#L449-L453) keeps the suite green.

<details><summary>test cases</summary>

`gnovm/cmd/gno/testdata/test/flag_x.txtar`:

```
# gno test -X overrides a package-level string var in the package under test.

gno test -v -X Version=1.2.3 .

stdout 'version=1.2.3'
stderr '--- PASS: TestVersion'

# without -X the declared default is used.

gno test -v .

stdout 'version=dev'

-- flag_x.gno --
package flag_x

var Version = "dev"

func V() string { return Version }

-- flag_x_test.gno --
package flag_x

import "testing"

func TestVersion(t *testing.T) {
	println("version=" + V())
}

-- gnomod.toml --
module = 'gno.test/r/integ/flag_x'
```
</details>
