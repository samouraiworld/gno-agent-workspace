// Repro: `gno run <dir>` with no -pkgpath and no gnomod.toml silently accepts a
// duplicate top-level declaration across files and runs the later one.
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout ce6b303940399f5fc07e1d2eea72ae03eb83ed4d
//	cp <this file> gnovm/cmd/gno/zz_default_pkgpath_dup_test.go
//	go test ./gnovm/cmd/gno -run TestRunDefaultPkgPathDuplicateDecl -count=1 -v
//
// Same fixture by hand, at the head and at the merge base 398e5960d (both print
// "2" and exit 0):
//
//	go run ./gnovm/cmd/gno run gnovm/tests/integ/run_duplicate_decl/; echo "rc=$?"
//
// Why: with no -pkgpath the package path defaults to the package name "main",
// gno.IsStdlib("main") is true, so Machine.RunFiles calls runFileDecls with
// withOverrides=true and checkDuplicates never runs; b.gno's helper replaces
// a.gno's. The PR's own duplicate-declaration case only reaches the new
// RedeclarationErrors branch because it passes -pkgpath example.com/main.
package main

import "testing"

func TestRunDefaultPkgPathDuplicateDecl(t *testing.T) {
	tc := []testMainCase{
		{
			// Same fixture the PR adds, run the way a user runs a directory:
			// no -pkgpath, no gnomod.toml.
			args: []string{"run", "../../tests/integ/run_duplicate_decl/"},

			// CURRENT (ce6b30394 and 398e5960d): runs b.gno's helper, exit 0.
			stdoutShouldBe: "2\n",

			// FIXED: the duplicate is reported like the -pkgpath case.
			// stderrShouldContain: `redeclarations for identifiers: "helper" (code=gnoPreprocessError)`,
			// errShouldBe:         "exit code: 1",
		},
	}
	testMainCaseRun(t, tc)
}
