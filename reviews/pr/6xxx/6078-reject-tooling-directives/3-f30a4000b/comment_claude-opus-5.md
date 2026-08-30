# Review: [#6078](https://github.com/gnolang/gno/pull/6078)
Posted: https://github.com/gnolang/gno/pull/6078#pullrequestreview-5061296055
Event: COMMENT
Status: posted as an AI on 2026-08-30, forced to COMMENT with the verdict on the Body Status line. Round re-anchored to f30a4000b; all four findings re-tested against that head and still open. On `post as an AI` the Body leads with `[AI review, opus 5] (not manually verified)`, then `Status: APPROVE`.

## Body
[AI review, opus 5] (not manually verified)
Status: APPROVE

`gno run` and `gno test` on a local package whose file opens with `//go:build ignore` both still work, so the refusal stays on the lint and chain paths and does not reach a scratch file.

## gnovm/cmd/gno/lint.go:82 [gh](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/cmd/gno/lint.go#L82) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/cmd/gno/lint.go#L82) [posted](https://github.com/gnolang/gno/pull/6078#discussion_r3889878270)
Nit: [`filepath.Abs`](https://github.com/golang/go/blob/go1.25.9/src/path/filepath/path.go#L161) leaves a symlink in the path intact, so a stdlib directory reached through one compares unequal to the root and `gno lint` reports Go's own `//go:generate` lines as errors; resolving both sides with `filepath.EvalSymlinks` before the comparison makes the two spellings equal.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6078 -R gnolang/gno
root=$(pwd)
go build -o "$root/gnobin" ./gnovm/cmd/gno
ln -sfn "$root" /tmp/gnolink
cd /tmp/gnolink/gnovm/stdlibs
GNOROOT=$root "$root/gnobin" lint ./math/bits || echo "exit: $?"
rm -f /tmp/gnolink "$root/gnobin"
```

The lint call fails on standard library source that no chain will ever see:

```
/tmp/gnolink/gnovm/stdlibs/math/bits/bits.gno: directives are not supported: "//go:generate" (code=gnoDirectiveError)
exit: 1
```

`gnoenv.RootDir` derives the root from `go list` or the caller stack, both of which give the resolved path, while the directory argument comes from the shell's own view of the working directory, so the two disagree whenever the checkout is reached through a link.
</details>

## gnovm/pkg/gnolang/mempackage.go:780 [gh](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L780) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/pkg/gnolang/mempackage.go#L780) [posted](https://github.com/gnolang/gno/pull/6078#discussion_r3889878273)
Nit: the extraction left `ReadMemPackage`'s fourteen-line doc comment sitting above `MemPackageFilePaths`, so godoc for the new function opens with "ReadMemPackage initializes a new MemPackage" and [`ReadMemPackage`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L852) keeps a one-line stub in its place.

## gnovm/pkg/gnolang/mempackage.go:1290 [gh](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L1290) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/pkg/gnolang/mempackage.go#L1284) [posted](https://github.com/gnolang/gno/pull/6078#discussion_r3889878274)
Nit: the `continue` skips `PackageNameFromFileBody`, so `pkgNameFound` stays false and the rejection carries a second clause saying the package name is not in the files, which it is.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6078 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_directive_msg_test.go <<'EOF'
package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestDirectiveErrorMessage(t *testing.T) {
	mpkg := &std.MemPackage{
		Name: "tagged",
		Path: "gno.land/r/demo/tagged",
		Type: MPUserAll,
		Files: []*std.MemFile{
			{Name: "gnomod.toml", Body: "module = \"gno.land/r/demo/tagged\"\ngno = \"0.9\"\n"},
			{Name: "tagged.gno", Body: "//go:build ignore\n\npackage tagged\n\nfunc F() int { return 1 }\n"},
		},
	}
	t.Log(ValidateMemPackageAny(mpkg))
}
EOF
go test -run TestDirectiveErrorMessage -count=1 -v ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_directive_msg_test.go
```

The package declares `tagged` on the line after the constraint, and the second clause denies it:

```
    zz_directive_msg_test.go:19: invalid file "tagged.gno": directives are not supported: "//go:build"; package name "tagged" not found in files
--- PASS: TestDirectiveErrorMessage (0.00s)
```

Rejecting on the first directive found, before the per-file loop starts, leaves the directive error alone.
</details>

## gnovm/pkg/gnolang/mempackage.go:1347 [gh](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L1347) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/pkg/gnolang/mempackage.go#L1341) [posted](https://github.com/gnolang/gno/pull/6078#discussion_r3889878278)
Refactor: this walk over line starts is `strings.SplitSeq(body, "\n")`, already used in 12 places in the tree and allocating nothing, which takes the function from 15 lines to 9.

```go
func hasRawGoGenerate(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "//go:generate ") ||
			strings.HasPrefix(line, "//go:generate\t") {
			return true
		}
	}
	return false
}
```

<details><summary>equivalence over 200k bodies</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6078 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_equiv_test.go <<'EOF'
package gnolang

import (
	"math/rand"
	"strings"
	"testing"
)

func splitSeqGoGenerate(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "//go:generate ") ||
			strings.HasPrefix(line, "//go:generate\t") {
			return true
		}
	}
	return false
}

func TestHasRawGoGenerateEquivalence(t *testing.T) {
	frags := []string{
		"//go:generate ls", "//go:generate\tls", "//go:generate", "//go:generatex ls",
		" //go:generate ls", "\t//go:generate ls", "//go:generate ls\r",
		"package p", "", "\r", "/*", "*/", "var s = `", "`", "// ordinary",
	}
	seps := []string{"\n", "\r\n", ""}
	rng := rand.New(rand.NewSource(1))
	for i := range 200_000 {
		var b strings.Builder
		for range rng.Intn(6) {
			b.WriteString(frags[rng.Intn(len(frags))])
			b.WriteString(seps[rng.Intn(len(seps))])
		}
		body := b.String()
		if got, want := splitSeqGoGenerate(body), hasRawGoGenerate(body); got != want {
			t.Fatalf("case %d disagree on %q: splitSeq=%v offsetLoop=%v", i, body, got, want)
		}
	}
}
EOF
go test -run TestHasRawGoGenerateEquivalence -count=1 ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_equiv_test.go
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.159s
```

The fragments cover both separators, the near-miss prefixes, indentation, CRLF endings and a final line with no newline.
</details>
