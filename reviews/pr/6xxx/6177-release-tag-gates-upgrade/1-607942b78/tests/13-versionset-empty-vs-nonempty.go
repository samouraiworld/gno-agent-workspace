//go:build ignore

// Candidate #13: tm2/pkg/bft/version/version_test.go:42
//
// Claim: TestVersionSetIsComplete's header says an empty Version on one side
// makes VersionSet.CompatibleWith "compare empty majors, which matches
// anything"; measure what an empty-vs-non-empty pair actually returns,
// against the both-empty case for contrast.
//
// from a local clone of gnolang/gno:
//   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//   cp 13-versionset-empty-vs-nonempty.go tm2/pkg/versionset/scratch_test.go
//   sed -i '1,2d' tm2/pkg/versionset/scratch_test.go   # drop the go:build ignore + blank line
//   sed -i '1i package versionset' tm2/pkg/versionset/scratch_test.go
//   go test -run 'TestEmptyVsNonEmptyScratch|TestEmptyVsEmptyScratch' -v -count=1 ./tm2/pkg/versionset/
//   rm tm2/pkg/versionset/scratch_test.go

package versionset

import (
	"fmt"
	"testing"
)

func TestEmptyVsNonEmptyScratch(t *testing.T) {
	a := VersionSet{{Name: "bft", Version: ""}}
	b := VersionSet{{Name: "bft", Version: "v1.0.0-rc.0"}}
	_, err := a.CompatibleWith(b)
	fmt.Println("empty vs non-empty:", err)
}

func TestEmptyVsEmptyScratch(t *testing.T) {
	a := VersionSet{{Name: "bft", Version: ""}}
	b := VersionSet{{Name: "bft", Version: ""}}
	_, err := a.CompatibleWith(b)
	fmt.Println("empty vs empty:", err)
}
