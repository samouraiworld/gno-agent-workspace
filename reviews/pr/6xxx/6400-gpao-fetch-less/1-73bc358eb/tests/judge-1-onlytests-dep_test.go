// Asserts nothing; logs what prepare and verifyPackage answer for an importer
// of a live dependency holding gnomod.toml and a _test.gno file only (the shape
// AddPackage refuses since #5892). Run at the reviewed head and at the merge base.
package main

import (
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestJudgeOnlyTestsDep(t *testing.T) {
	const depPath = "gno.land/p/x/onlytests"
	dep := &std.MemPackage{Name: "onlytests", Path: depPath, Type: gno.MPUserAll, Files: []*std.MemFile{
		{Name: "gnomod.toml", Body: gno.GenGnoModLatest(depPath)},
		{Name: "onlytests_test.gno", Body: "package onlytests\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n"},
	}}
	dep.Sort()
	mpkg := chainPackage("gno.land/p/test/user", "package user\n\nimport \"gno.land/p/x/onlytests\"\n\nvar V = onlytests.X\n")
	v := newRPCVerifier(t, dep)
	perr := v.prepare(mpkg)
	t.Logf("prepare: %v", perr)
	if perr == nil {
		t.Logf("verifyPackage: %v", v.verifyPackage(mpkg))
	}
}
