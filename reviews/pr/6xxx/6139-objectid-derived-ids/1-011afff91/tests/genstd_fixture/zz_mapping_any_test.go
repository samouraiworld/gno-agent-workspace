// Repro, from a plain clone of github.com/gnolang/gno:
//
//	# this file goes to misc/genstd/zz_mapping_any_test.go, and the two
//	# fixture directories to misc/genstd/testdata/
//	go test ./misc/genstd/ -run 'Test_linkFunctions_anyParam|Test_linkFunctions_namedIface' -count=1
//
//	# to see the positive case fail against the pre-widening mapping.go, copy
//	# this directory to reviews-genstd-fixture/ at the repo root, then:
//	go test -overlay=reviews-genstd-fixture/overlay.json ./misc/genstd/ \
//	    -run Test_linkFunctions_anyParam -count=1
//	# expect: doesn't match signature of go function
//
// Covers the isTypedValue widening the reviewed head introduces. genstd used
// to skip the gno-to-Go parameter type check only for gno.TypedValue; it now
// skips it for the empty interface as well, which is what lets
// chain/runtime.objectID declare `v interface{}` on both sides. No fixture in
// the suite exercises that branch.
package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_linkFunctions_anyParam pins the widened rule: an empty-interface Go
// parameter links against any gno parameter type, unchecked, and arrives as
// the TypedValue. `any` and `interface{}` are one type and both must qualify.
//
// It fails before the widening, where linkFunctions panics with "doesn't match
// signature of go function" on AnyParam.
func Test_linkFunctions_anyParam(t *testing.T) {
	chdir(t, "testdata/linkFunctions_anyParam")

	pkgs, err := walkStdlibs(".")
	require.NoError(t, err)

	mappings := linkFunctions(pkgs)
	require.Len(t, mappings, 2)

	for _, m := range mappings {
		require.Len(t, m.Params, 1, "%s", m.GoFunc)
		assert.True(t, m.Params[0].IsTypedValue,
			"%s: an empty-interface Go parameter must receive the TypedValue", m.GoFunc)
	}
}

// Test_linkFunctions_namedIface pins the edge of that widening. Only the empty
// interface is exempt; a named interface is a Go type like any other, and a
// widening that swallowed it would leave every native's Go signature
// unchecked against its gno declaration.
func Test_linkFunctions_namedIface(t *testing.T) {
	chdir(t, "testdata/linkFunctions_namedIface")

	pkgs, err := walkStdlibs(".")
	require.NoError(t, err)

	defer func() {
		r := recover()
		require.NotNil(t, r, "a named interface parameter linked without a type check")
		assert.Contains(t, fmt.Sprint(r), "doesn't match signature of go function")
	}()

	linkFunctions(pkgs)
}
