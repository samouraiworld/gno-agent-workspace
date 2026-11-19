package test

// Prints the gas a make([]byte, 10) site adds over a []byte literal, once in a
// main-package filetest and once in a realm filetest, after a stdlib load
// warms the shared store as in TestFiles. Run from gnovm/pkg/test:
// go test -run TestJudge2RealmGas -v .
import (
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJudge2RealmGas(t *testing.T) {
	rootDir, err := filepath.Abs("../../../")
	require.NoError(t, err)
	opts := NewTestOptions(rootDir, io.Discard, io.Discard, nil)
	run := func(name, src string) int64 {
		_, gas, _, err := opts.RunFiletest(name+".gno", []byte(src), opts.TestStore)
		require.NoError(t, err)
		return gas
	}
	run("warm", "package main\n\nimport \"strings\"\n\nfunc main() { _ = strings.Repeat(\"a\", 3) }\n")
	lit := "\n\nfunc main() {\n\tbuf := []byte(\"0123456789\")\n\t_ = string(buf)\n}\n"
	mk := "\n\nfunc main() {\n\tbuf := make([]byte, 10)\n\t_ = string(buf)\n}\n"
	for _, hdr := range []string{"package main", "// PKGPATH: gno.land/r/test/judge\npackage judge"} {
		a, b := run("lit", hdr+lit), run("mk", hdr+mk)
		t.Logf("GAS %q literal=%d make=%d delta=%d", hdr, a, b, b-a)
	}
}
