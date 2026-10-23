// Repro (from a plain clone of gnolang/gno at 73bc358ebb7fbe473d233dd2fa84fa949aac0ad4):
//   cp tests/solo-finder-1-broken-test-file-prepare.go contribs/gpao/zz_solo1_test.go
//   cd contribs/gpao && go test -count=1 -run TestSoloBrokenTestFileHidesParkedImport -v .
// Observed at head: a _test.gno whose import block does not parse skips the prepare walk
// (0 qfile calls, parked empty), but the ProdOnly typecheck fails on that same file
// ("user_test.gno:6:1: missing import path") even once the import is live, so the
// rejection matches the validator's: settles the candidate as no finding.
package main

import (
	"errors"
	"testing"

	vm "github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

// The candidate's own _test.gno decides whether its production imports are
// prefetched and classified: buildChainImports parses every file of the
// candidate with packages.Imports and skips the whole walk on one parse error.
func TestSoloBrokenTestFileHidesParkedImport(t *testing.T) {
	const parkedPath = "gno.land/p/test/parkeddep"
	const prodBody = "package user\n\nimport \"gno.land/p/test/parkeddep\"\n\nfunc Use() int { return parkeddep.Answer() }\n"
	const goodTest = "package user\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n"
	const brokenTest = "package user\n\nimport (\n\t\"testing\"\n\nfunc TestX(t *testing.T) {}\n"

	candidate := func(testBody string) *std.MemPackage {
		mpkg := chainPackage("gno.land/p/test/user", prodBody)
		if testBody != "" {
			mpkg.Files = append(mpkg.Files, &std.MemFile{Name: "user_test.gno", Body: testBody})
			mpkg.Sort()
		}
		return mpkg
	}
	parkedChain := func(t *testing.T) (*verifier, *int) {
		v := newRPCVerifier(t) // vm/qfile serves nothing: a parked package is invisible to it
		v.rpc.qmeta = func(p string) (vm.PackageMeta, error) {
			st := vm.PackageStatusAbsent
			if p == parkedPath {
				st = vm.PackageStatusInert
			}
			return vm.PackageMeta{Path: p, Status: st}, nil
		}
		calls := 0
		inner := v.rpc.qfile
		v.rpc.qfile = func(fp string) ([]byte, error) { calls++; return inner(fp) }
		return v, &calls
	}

	for _, tc := range []struct{ name, test string }{
		{"no test file", ""},
		{"test file parses", goodTest},
		{"test file import block does not parse", brokenTest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mpkg := candidate(tc.test)
			require.NoError(t, gno.ValidateMemPackageAny(mpkg), "the chain's AddPackage validation accepts these bytes")
			v, calls := parkedChain(t)
			require.NoError(t, v.prepare(mpkg))
			onPrepare := *calls
			err := v.verifyPackage(mpkg)
			t.Logf("qfile calls in prepare=%d, in verifyPackage (on the budget)=%d", onPrepare, *calls-onPrepare)
			t.Logf("parked=%v unserved=%v", v.parked, v.unserved)
			t.Logf("errImportParked=%v err=%v", errors.Is(err, errImportParked), err)
		})
	}

	// The validator's own question once the dependency is live: the same
	// typecheck and preprocess, run on the same bytes.
	t.Run("broken test file, dependency live", func(t *testing.T) {
		dep := chainPackage(parkedPath, "package parkeddep\n\nfunc Answer() int { return 42 }\n")
		mpkg := candidate(brokenTest)
		v := newRPCVerifier(t, dep)
		require.NoError(t, v.prepare(mpkg))
		t.Logf("verifyPackage once live: err=%v", v.verifyPackage(mpkg))
	})
}
