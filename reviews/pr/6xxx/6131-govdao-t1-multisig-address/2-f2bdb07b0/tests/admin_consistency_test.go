// Package deployments holds cross-tree consistency checks for the genesis
// builders under misc/deployments/.
//
// It lives under gno.land/pkg/ because that is a directory `ci / gnoland`
// actually runs (`_ci-go.yml` with modulepath "gno.land"), and that workflow
// already triggers on `examples/**`. Nothing in CI builds, lints or executes
// misc/deployments/*/gen-genesis.sh, so a check placed beside those scripts
// would never run.
package deployments

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// repoRoot is gno.land/pkg/deployments -> ../../..
const repoRoot = "../../.."

// verifierPath holds the admin constant that gates r/sys/names.Enable(),
// which is one-shot and one-way: a genesis cut with a stale value there
// cannot be repaired by a later transaction.
const verifierPath = "examples/gno.land/r/sys/names/verifier.gno"

var (
	realmAdminRe = regexp.MustCompile(`(?m)^\s*admin\s+=\s+address\("(g1[0-9a-z]+)"\)`)
	namesAdminRe = regexp.MustCompile(`(?m)^NAMES_ADMIN=(g1[0-9a-z]+)`)
)

// deploymentExemptions lists deployments whose names-enable caller is
// deliberately not the realm admin, with the reason. An entry here is a
// decision on the record; a missing entry is a failure.
var deploymentExemptions = map[string]string{}

func realmAdmin(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, verifierPath))
	if err != nil {
		t.Fatalf("read %s: %v", verifierPath, err)
	}
	m := realmAdminRe.FindSubmatch(b)
	if m == nil {
		t.Fatalf("%s: no `admin = address(\"g1...\")` declaration found; "+
			"the constant moved and this check stopped checking anything", verifierPath)
	}
	return string(m[1])
}

// TestNamesAdminMatchesRealmConstant ties every genesis builder's NAMES_ADMIN
// and its names-enable caller_override to the admin constant they mirror.
func TestNamesAdminMatchesRealmConstant(t *testing.T) {
	want := realmAdmin(t)
	t.Logf("%s admin = %s", verifierPath, want)

	scripts, err := filepath.Glob(filepath.Join(repoRoot, "misc/deployments/*/gen-genesis.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) == 0 {
		t.Fatal("no misc/deployments/*/gen-genesis.sh found")
	}

	for _, script := range scripts {
		deployment := filepath.Base(filepath.Dir(script))
		t.Run(deployment, func(t *testing.T) {
			if why, ok := deploymentExemptions[deployment]; ok {
				t.Skipf("exempt: %s", why)
			}

			b, err := os.ReadFile(script)
			if err != nil {
				t.Fatal(err)
			}
			if m := namesAdminRe.FindSubmatch(b); m == nil {
				t.Logf("no NAMES_ADMIN in gen-genesis.sh, skipping variable check")
			} else if got := string(m[1]); got != want {
				t.Errorf("NAMES_ADMIN = %s, want %s (the admin in %s)", got, want, verifierPath)
			}

			metaPath := filepath.Join(filepath.Dir(script), "transactions/migration/names-enable/meta.json")
			raw, err := os.ReadFile(metaPath)
			if os.IsNotExist(err) {
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var meta struct {
				Func           string `json:"func"`
				PkgPath        string `json:"pkgpath"`
				CallerOverride string `json:"caller_override"`
			}
			if err := json.Unmarshal(raw, &meta); err != nil {
				t.Fatal(err)
			}
			if meta.PkgPath != "gno.land/r/sys/names" || meta.Func != "Enable" {
				t.Fatalf("names-enable meta.json calls %s.%s, not gno.land/r/sys/names.Enable",
					meta.PkgPath, meta.Func)
			}
			if meta.CallerOverride != want {
				t.Errorf("names-enable caller_override = %s, want %s (the admin in %s)",
					meta.CallerOverride, want, verifierPath)
			}
		})
	}
}
