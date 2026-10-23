// Asserts nothing; logs what prepare and verifyPackage answer for an importer
// of a live dependency whose module file is the legacy gno.mod. Run at the
// reviewed head with and without "gno.mod" in readByVerify's case.
package main

import "testing"

func TestJudgeLegacyGnoModDep(t *testing.T) {
	const depPath = "gno.land/p/test/legacy"
	dep := chainPackage(depPath, "package legacy\n\nfunc Add(a, b int) int { return a + b }\n")
	for _, f := range dep.Files {
		if f.Name == "gnomod.toml" {
			f.Name = "gno.mod"
			f.Body = "module " + depPath + "\n"
		}
	}
	dep.Sort()
	mpkg := chainPackage("gno.land/p/test/user", "package user\n\nimport \"gno.land/p/test/legacy\"\n\nfunc Use() int { return legacy.Add(1, 2) }\n")
	v := newRPCVerifier(t, dep)
	perr := v.prepare(mpkg)
	t.Logf("prepare: %v", perr)
	if perr == nil {
		t.Logf("verifyPackage: %v", v.verifyPackage(mpkg))
	}
}
