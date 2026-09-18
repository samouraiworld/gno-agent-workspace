// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f2
//	cp <this file> misc/gnopreview/b5_lines_r2_baseline_test.go
//	cd misc/gnopreview && go test ./... -run 'BaselineLoss|NewRealm' -v
//
// Both tests PASS at ecf7af0f2, which is the finding: the comment a reviewer
// reads is byte-identical in shape whether the merge-base render succeeded and
// found nothing to compare, or whether renderBase failed and there is no
// baseline at all. renderBase (main.go:231-280) returns a nil *Crawler on five
// paths -- empty -base-root, startGnodev error, waitReady timeout, base.Run
// error, base.Write error -- each of which reports to stderr only.
package main

import (
	"strings"
	"testing"
)

// The state main.go leaves behind when renderBase returns nil: the realm exists
// on the merge base (so newRealms does not mark it New), but no before shot was
// taken. ScreenshotPairs still appends the pair, with Before == "" and New == false.
func TestB5BaselineLossIsUnannounced(t *testing.T) {
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/x/leaf"},
		Pairs: []ShotPair{
			{Realm: "gno.land/r/x/leaf", After: "_shots/a.png", URL: "r/x/leaf/"},
		},
	}
	got := Comment(p, "https://example.test/pr-1", "1")
	t.Logf("comment with a failed merge-base render:\n%s", got)

	for _, unwanted := range []string{"before", "merge base", "baseline", "compare", "unavailable"} {
		if strings.Contains(strings.ToLower(got), unwanted) {
			t.Errorf("comment mentions %q, so the reader is told: %s", unwanted, got)
		}
	}
	if !strings.Contains(got, "_shots/a.png") {
		t.Fatalf("after shot missing, wrong fixture:\n%s", got)
	}
}

// The genuinely-new realm renders the same markup plus one <sub> note. Strip
// the note and the two bodies are identical, so the note is the only signal a
// reader has, and the failed-baseline case does not carry it.
func TestB5NewRealmAndFailedBaselineAreTheSameMarkup(t *testing.T) {
	mk := func(isNew bool) string {
		p := &Plan{
			ChangedRealms: []string{"gno.land/r/x/leaf"},
			Realms:        []string{"gno.land/r/x/leaf"},
			Pairs: []ShotPair{
				{Realm: "gno.land/r/x/leaf", After: "_shots/a.png", URL: "r/x/leaf/", New: isNew},
			},
		}
		return Comment(p, "https://example.test/pr-1", "1")
	}
	newRealm, failedBase := mk(true), mk(false)
	const note = "\n\n<sub>New in this PR — nothing to compare against.</sub>"
	if strings.ReplaceAll(newRealm, note, "") != failedBase {
		t.Fatalf("markup differs beyond the note:\nnew:\n%s\nfailed:\n%s", newRealm, failedBase)
	}
	t.Logf("identical apart from the New note; failed-baseline body:\n%s", failedBase)
}

// Nit repros, same package, same command (-run 'B5NoBaseURL|B5IndirectHeading'):
//
//   base="" glues the intro into the next block (comment.go:52 writes one \n):
//     Comment(&Plan{Gnoweb: true, ChangedRealms: []string{"gno.land/r/x/leaf"},
//       ChangedPkgs: []string{"gno.land/p/x/b"}, Realms: []string{"gno.land/r/x/leaf"}}, "", "3")
//     -> "...realm sources. \n**Changed realms (1)**\n\n"
//
//   the indirect heading names a package when none changed (comment.go:75):
//     Comment(&Plan{ChangedRealms: []string{"gno.land/r/x/lib"},
//       Realms: []string{"gno.land/r/x/app", "gno.land/r/x/lib"}}, "https://e.test/pr-4", "4")
//     -> "**Realms affected through a changed package (1)**" with ChangedPkgs empty
