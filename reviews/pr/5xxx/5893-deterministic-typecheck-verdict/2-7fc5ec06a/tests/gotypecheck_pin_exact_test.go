/* Run:

	gh pr checkout 5893 -R gnolang/gno && git checkout 7fc5ec06a
	cp <this-file> gnovm/pkg/gnolang/zz_pin_exact_test.go
	go test -count=1 -run 'TestTypeCheckMemPackage_PinIsExactly118|TestTypeCheckMemPackage_PinAppliesToImports' -v ./gnovm/pkg/gnolang/
	rm gnovm/pkg/gnolang/zz_pin_exact_test.go

Both are GREEN at 7fc5ec06a. They are regression guards, not bug repros: each
closes a hole in TestTypeCheckMemPackage_GoVersionPinned that lets a
consensus-affecting edit ship with CI green.

To see PinIsExactly118 earn its keep, change gotypecheck.go's pin from "go1.18"
to "go1.21" and re-run: the committed TestTypeCheckMemPackage_GoVersionPinned
still passes, this one goes red.

To see PinAppliesToImports earn its keep, give the importer its own types.Config
(i.e. drop the shared gimp.cfg) and re-run.
*/

package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The committed TestTypeCheckMemPackage_GoVersionPinned only asserts that a
// go1.22 feature is rejected. That leaves the pin free to move anywhere in
// [go1.18, go1.21] with CI green, and go1.21 is not a safe place for it: it
// unlocks the min/max/clear builtins, which the GnoVM has no uverse entry for.
// The pin's own doc comment claims it "rejects features Gno can't run (min/max,
// range-over-int/func) here rather than downstream" — the min/max half of that
// claim is unguarded. Pin the floor from below as well as from above.
func TestTypeCheckMemPackage_PinIsExactly118(t *testing.T) {
	t.Parallel()

	tc := func(body string) error {
		mp := &std.MemPackage{
			Type:  MPUserProd,
			Name:  "z",
			Path:  "gno.land/p/demo/z",
			Files: []*std.MemFile{{Name: "z.gno", Body: body}},
		}
		_, err := TypeCheckMemPackage(mp, TypeCheckOptions{Mode: TCLatestRelaxed})
		return err
	}

	// Upper bound: nothing above go1.18 may be accepted.
	assert.ErrorContains(t, tc("package z\nfunc F() int { return min(1, 2) }\n"),
		"go1.21", "min is a go1.21 builtin with no GnoVM uverse entry; the pin must reject it")
	assert.ErrorContains(t, tc("package z\nfunc F() int { return max(1, 2) }\n"),
		"go1.21", "max is a go1.21 builtin with no GnoVM uverse entry; the pin must reject it")
	assert.ErrorContains(t, tc("package z\nfunc F() { m := map[int]int{}; clear(m) }\n"),
		"go1.21", "clear is a go1.21 builtin with no GnoVM uverse entry; the pin must reject it")
	assert.ErrorContains(t, tc("package z\nfunc F() { for range 10 {} }\n"),
		"go1.22", "range-over-int is go1.22; the pin must reject it")

	// Lower bound: the pin cannot drop below go1.18, because the injected
	// .gnobuiltins shim itself uses `any` and a type parameter
	// (`func revive[F any](fn F) any`). At go1.17 the shim stops compiling and
	// every package on the chain fails to type-check.
	assert.NoError(t, tc("package z\nfunc F(x any) any { return x }\n"),
		"`any` is go1.18 and the shim depends on it; the pin must not drop below go1.18")
}

// The pin lives on the single types.Config shared by gnoImporter, so it covers
// imported packages as well as the target. Nothing asserts that. A refactor
// that hands the importer its own Config would silently un-pin every dependency
// while the committed test — which only ever type-checks a single standalone
// package — stays green.
func TestTypeCheckMemPackage_PinAppliesToImports(t *testing.T) {
	t.Parallel()

	dep := &std.MemPackage{
		Type: MPUserProd,
		Name: "dep",
		Path: "gno.land/p/demo/dep",
		Files: []*std.MemFile{{
			Name: "dep.gno",
			// go1.22 feature, in the *imported* package.
			Body: "package dep\nfunc G() { for range 10 {} }\n",
		}},
	}
	target := &std.MemPackage{
		Type: MPUserProd,
		Name: "z",
		Path: "gno.land/p/demo/z",
		Files: []*std.MemFile{{
			Name: "z.gno",
			Body: "package z\nimport \"gno.land/p/demo/dep\"\nfunc F() { dep.G() }\n",
		}},
	}
	getter := mockPackageGetter{dep, target}

	_, err := TypeCheckMemPackage(target, TypeCheckOptions{
		Getter:     getter,
		TestGetter: getter,
		Mode:       TCLatestRelaxed,
	})
	require.Error(t, err, "a go1.22 construct in an imported package must be rejected too")
	assert.Contains(t, err.Error(), "go1.22",
		"the pinned GoVersion must apply to imported packages, not just the target")
}
