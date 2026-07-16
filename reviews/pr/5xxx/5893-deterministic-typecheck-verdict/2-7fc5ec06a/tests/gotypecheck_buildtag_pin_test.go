/* Run:

	gh pr checkout 5893 -R gnolang/gno && git checkout 7fc5ec06a
	cp <this-file> gnovm/pkg/gnolang/zz_buildtag_pin_test.go
	go test -count=1 -run 'TestTypeCheckMemPackage_BuildTagCannotRaisePin' -v ./gnovm/pkg/gnolang/
	rm gnovm/pkg/gnolang/zz_buildtag_pin_test.go

Expected at 7fc5ec06a: RED (both subtests fail; the //go:build line raises the
per-file language version above the pinned go1.18, and the "too new" gate is
resolved against the *building* toolchain's version).
Expected after a fix that clears ast.File.GoVersion on every parsed .gno file:
GREEN.
*/

package gnolang

import (
	"runtime"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tcBody(t *testing.T, body string) error {
	t.Helper()
	mp := &std.MemPackage{
		Type:  MPUserProd,
		Name:  "z",
		Path:  "gno.land/p/demo/z",
		Files: []*std.MemFile{{Name: "z.gno", Body: body}},
	}
	_, err := TypeCheckMemPackage(mp, TypeCheckOptions{Mode: TCLatestRelaxed})
	return err
}

// The consensus type-check pins types.Config.GoVersion to go1.18 so the
// accept/reject verdict is a function of the submitted package alone, never of
// the Go toolchain a given validator binary was built with.
//
// go/types honours a per-file `//go:build go1.N` line by *upgrading* that
// file's language version above the Config pin (go/types.(*Checker).initFiles;
// the upgrade direction is always allowed, the downgrade direction only when
// the Config pin is >= go1.21). Package bodies are attacker-supplied, so a
// submitter can raise the gate on their own file with one comment line. The pin
// must not be raisable from inside the package.
func TestTypeCheckMemPackage_BuildTagCannotRaisePin(t *testing.T) {
	t.Parallel()

	// Guard: prove the pin is actually in effect in this build, so a failure
	// below cannot be misread as "the pin is simply missing".
	require.ErrorContains(t, tcBody(t, "package z\nfunc F() { for range 10 {} }\n"),
		"go1.22", "precondition: the pin must reject range-over-int without a build tag")

	t.Run("build tag must not raise the pinned version", func(t *testing.T) {
		t.Parallel()

		// Identical package, plus one comment line. Must stay rejected.
		err := tcBody(t, "//go:build go1.22\n\npackage z\nfunc F() { for range 10 {} }\n")
		assert.Error(t, err,
			"a //go:build line must not raise the pinned GoVersion: the verdict "+
				"for a submitted package must not be settable by the submitter")

		err = tcBody(t, "//go:build go1.21\n\npackage z\nfunc F() int { return min(1, 2) }\n")
		assert.Error(t, err,
			"a //go:build line must not unlock go1.21 builtins the VM cannot run")

		err = tcBody(t, "//go:build go1.23\n\npackage z\n"+
			"func F(p func(func(int) bool)) { for range p {} }\n")
		assert.Error(t, err,
			"a //go:build line must not unlock range-over-func")
	})

	t.Run("verdict must not depend on the building toolchain", func(t *testing.T) {
		t.Parallel()

		// go/types rejects a file version newer than the toolchain that built
		// the binary ("file requires newer Go version goX (application built
		// with goY)"). Y is the *builder's* Go version, so the same package
		// gets opposite verdicts on two honest validators running binaries
		// built with different Go releases — precisely the fork this PR closes
		// on the Config axis, still open on the per-file axis.
		//
		// Body is valid go1.18 code; only the build tag varies.
		const body = "//go:build go1.99\n\npackage z\nfunc F() int { return 1 }\n"
		err := tcBody(t, body)
		assert.NoError(t, err,
			"verdict must not reference the building toolchain (runtime %s); "+
				"got: %v", runtime.Version(), err)
	})
}
