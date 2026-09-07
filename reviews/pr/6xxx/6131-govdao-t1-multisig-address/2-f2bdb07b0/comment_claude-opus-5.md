# Review: [#6131](https://github.com/gnolang/gno/pull/6131)
Event: REQUEST_CHANGES

## Body
Eight `test13` caller fields agreed with the `examples/` constants at the merge base and disagree here. `test13` deploys `r/sys/names` and `boards2/v1` from this repository's [`examples/` tree](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L637), so [`names-enable`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json#L5) now [panics with `caller is not admin`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L117-L119) and the seven [`boards2-cascade`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json#L20) callers are rejected by [`WithPermission`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/public.gno#L146). Each cascade patch names that invariant in its own [`reason`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json#L2), which ships into the genesis stream, so the artifact documents a check it no longer passes. Repointing the callers is not the fix either: [the script picks the multisig](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L1535) because replayed history leaves it the only funded account, so the caller has to be both funded and the realm admin.

## examples/gno.land/r/sys/names/verifier.gno:59 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/sys/names/verifier.gno#L59)
Four genesis builders copy this constant into `NAMES_ADMIN` by hand and none reads it back, so nothing fails when the two drift apart. `pearl` and `sapphire` compare their `caller_override` against their `NAMES_ADMIN` at [`pearl/gen-genesis.sh:799-800`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L799-L800), but both operands sit under `misc/deployments/` and this change moved them together.

<details><summary>the check that closes it</summary>

At `gno.land/pkg/deployments/admin_consistency_test.go`, a path `ci / gnoland` already runs and which already triggers on `examples/**`. It fails on `test13` at this head and passes at the merge base.

```go
package deployments

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const repoRoot = "../../.."

// verifierPath holds the admin constant gating r/sys/names.Enable(), which is
// one-shot and one-way: a genesis cut with a stale value cannot be repaired.
const verifierPath = "examples/gno.land/r/sys/names/verifier.gno"

var (
	realmAdminRe = regexp.MustCompile(`(?m)^\s*admin\s+=\s+address\("(g1[0-9a-z]+)"\)`)
	namesAdminRe = regexp.MustCompile(`(?m)^NAMES_ADMIN=(g1[0-9a-z]+)`)
)

// deploymentExemptions lists deployments whose names-enable caller is
// deliberately not the realm admin, with the reason. A missing entry is a
// failure, so an exception has to be written down.
var deploymentExemptions = map[string]string{}

func realmAdmin(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, verifierPath))
	if err != nil {
		t.Fatalf("read %s: %v", verifierPath, err)
	}
	m := realmAdminRe.FindSubmatch(b)
	if m == nil {
		t.Fatalf("%s: no admin declaration found; the constant moved and this "+
			"check stopped checking anything", verifierPath)
	}
	return string(m[1])
}

func TestNamesAdminMatchesRealmConstant(t *testing.T) {
	want := realmAdmin(t)

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
			if m := namesAdminRe.FindSubmatch(b); m != nil {
				if got := string(m[1]); got != want {
					t.Errorf("NAMES_ADMIN = %s, want %s (%s)", got, want, verifierPath)
				}
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
				t.Errorf("names-enable caller_override = %s, want %s (%s)",
					meta.CallerOverride, want, verifierPath)
			}
		})
	}
}
```

```
--- FAIL: TestNamesAdminMatchesRealmConstant/test13.gno.land
    NAMES_ADMIN = g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh, want g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf (the admin in examples/gno.land/r/sys/names/verifier.gno)
    names-enable caller_override = g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh, want g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf (the admin in examples/gno.land/r/sys/names/verifier.gno)
```

</details>

## examples/gno.land/r/gnoland/blog/admin_test.gno:27 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin_test.gno#L27) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin_test.gno#L27)
`clearState` runs first in every test in the package and assigns a second hardcoded copy of the address over [`adminAddr`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20), so its three goldens that carry an address assert this literal and the realm's suite stays green whatever the source constant says.

```suggestion
	adminAddr = initialAdminAddr
```

<details><summary>repro</summary>

Add `var initialAdminAddr = adminAddr` beside `var cur realm` for the suggestion above to compile.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6131 -R gnolang/gno
export GNOROOT=$PWD
go build -o /tmp/gno ./gnovm/cmd/gno
sed -i '20s/g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf/g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh/' \
  examples/gno.land/r/gnoland/blog/admin.gno
/tmp/gno test -C examples ./gno.land/r/gnoland/blog
git checkout -- examples/gno.land/r/gnoland/blog/admin.gno && rm -f /tmp/gno
```

The realm's admin is wrong and its own suite does not notice:

```
ok      ./gno.land/r/gnoland/blog 	3.16s
```

With the two-line change applied, the same revert fails and names both addresses:

```
invalid render output.
expected "…Published by g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf to Gno.land's blog…"
got      "…Published by g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh to Gno.land's blog…"
failed: "TestPackage"
FAIL    ./gno.land/r/gnoland/blog 	3.17s
```

</details>

## misc/deployments/topaz.gno.land/gen-genesis.sh:137 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L137) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/topaz.gno.land/gen-genesis.sh#L137)
Suggestion: `topaz` reads this variable only into the substep label at [`:722`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L722) while the caller that ships comes from `meta.json`, so a half-applied swap logs one address and cuts genesis with another, and `test13` does the same while writing the address longhand at [`:1209`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L1209) for `fork valoper-seed`'s fee payer. [`pearl/gen-genesis.sh:798-800`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L798-L800) is the comparison that catches it.
