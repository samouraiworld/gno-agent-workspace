# Review: [#6131](https://github.com/gnolang/gno/pull/6131)
Event: REQUEST_CHANGES

## Body
The address moved. Four things that depend on it did not.

- [`pearl`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L213) can no longer rebuild its own genesis. Its steps 4.1 to 7.3 give `499d9fba…` at the merge base, matching the locked `work/genesis_txs.jsonl`, and `79cea215…` here, so [`verify_checksum`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L347) exits 1 at step 7 of 9. It is the only one of the four whose lock was still live: `sapphire` already misses its own at the merge base, and `topaz` and `test13` already miss `packages.gen.txt` one gate earlier. Re-locking is the wrong repair: [`VALIDATOR.md:45`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/VALIDATOR.md?plain=1#L45) publishes `c45fe60c…` as the hash a validator checks the released genesis against and [`:216`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L216) pins it, so a fresh lock would point the script at a genesis that is not the running chain.
- Eight `test13` caller fields agreed with the `examples/` constants at the merge base and disagree here. `test13` deploys `r/sys/names` and `boards2/v1` from this repository's [`examples/` tree](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L637), so [`names-enable`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json#L5) now [panics with `caller is not admin`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L117-L119) and the seven [`boards2-cascade`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json#L20) callers are rejected by [`WithPermission`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/public.gno#L146). Each cascade patch names that invariant in its own [`reason`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json#L2), which ships into the genesis stream, so the artifact documents a check it no longer passes. Repointing the callers is not the fix either: [the script picks the multisig](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L1535) because replayed history leaves it the only funded account, so the caller has to be both funded and the realm admin.
- The description opens `Draft on purpose` and asks that this not merge until the signer set settles, while the pull request is marked ready for review and mergeable. `Enable` is one-shot, `enabled` has no setter back to false, and `admin` has no assignment anywhere in the realm, so a genesis cut from a stale constant needs a new chain.
- The balance move in the notes needs a step before it. `ugnot` is a restricted denom on `gnoland1` and [`canSendCoins`](https://github.com/gnolang/gno/blob/f2bdb07b0/tm2/pkg/sdk/bank/keeper.go#L135-L150) consults the sender's whitelist bit alone, so the transfer succeeds and the new address then holds 119 million GNOT it cannot send: it has no account on that chain and no entry in `params/auth:p:unrestricted_addrs`, so receipt creates the account unwhitelisted. Getting it whitelisted afterwards needs a GovDAO proposal through [`ProposeAddUnrestrictedAcctsRequest`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/params/unlock.gno#L22), and the multisig is not a T1 member. That paragraph also misses two realm ownerships: `pearl`, `sapphire` and `topaz` were each cut from a genesis carrying the old address, so [`blog.adminAddr`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20) and boards2 [`gPerms`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47) are owned on those chains by the old multisig, and every rotation transaction needs the signer set that is about to be dissolved.

<details><summary>replaying a builder's checksum gate</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6131 -R gnolang/gno
mkdir -p /tmp/gnobin
# gnogenesis is its own module, so each binary is built with `go build -C`,
# the same way gen-genesis.sh builds them.
go build -C gnovm/cmd/gno -o /tmp/gnobin/gno .
go build -C gno.land/cmd/gnokey -o /tmp/gnobin/gnokey .
go build -C contribs/gnogenesis -o /tmp/gnobin/gnogenesis .
export PATH=/tmp/gnobin:$PATH

cat > /tmp/replay.sh <<'SH'
set -eu
E="$1"; P="$2"; W="$3"
rm -rf "$W"; mkdir -p "$W"
CHAIN_ID=pearl-1; GENESIS_TIME=1787817600
DEPLOYER_KEY=GenesisDeployer
MN="anchor hurt name seed oak spread anchor filter lesson shaft wasp home improve text behind toe segment lamp turn marriage female royal twice wealth"
GH="$W/gnokey-home"; G="$W/genesis.json"; TXS="$W/genesis_txs.jsonl"
pkg_dirs=$(cd "$E" && gno tool deplist -test-dep ./gno.land/r/sys/... ./gno.land/r/gov/... ./gno.land/r/gnoland/blog/... ./gno.land/r/gnoland/wugnot/... ./gno.land/r/gnoland/coins/... ./gno.land/r/gnoland/boards2/... ./gno.land/r/gnops/valopers/... ./gno.land/p/onbloc/uint256 ./gno.land/p/onbloc/int256 ./gno.land/p/onbloc/json ./gno.land/r/sys/validators/v3 ./gno.land/r/demo/defi/grc20reg)
S="$W/examples"; mkdir -p "$S"
while IFS= read -r dir; do
  [ -z "$dir" ] && continue
  rel="${dir#"$E"/}"; dest="$S/$rel"; mkdir -p "$dest"
  find "$dir" -maxdepth 1 -type f -exec cp {} "$dest/" \;
  [ -d "$dir/filetests" ] && cp -r "$dir/filetests" "$dest/filetests"
done <<<"$pkg_dirs"
printf '%s\n\n' "$MN" | gnokey add --recover "$DEPLOYER_KEY" --home "$GH" --insecure-password-stdin >/dev/null 2>&1
gnogenesis generate -chain-id "$CHAIN_ID" -genesis-time "$GENESIS_TIME" --output-path "$G" >/dev/null
echo "" | gnogenesis txs add packages "$S" -gno-home "$GH" -key-name "$DEPLOYER_KEY" --genesis-path "$G" --insecure-password-stdin >/dev/null 2>&1
gnogenesis txs export "$TXS" --genesis-path "$G" >/dev/null
echo "  addpkg-stage genesis_txs.jsonl sha256: $(sha256sum "$TXS" | cut -d' ' -f1)"
SH
chmod +x /tmp/replay.sh

base=$(git merge-base origin/master HEAD)
for rev in "$base" HEAD; do
  rm -rf /tmp/t && mkdir -p /tmp/t
  git archive "$rev" examples misc/deployments/pearl.gno.land | tar -x -C /tmp/t
  echo "$rev"
  /tmp/replay.sh /tmp/t/examples /tmp/t/misc/deployments/pearl.gno.land /tmp/out
done
rm -rf /tmp/t /tmp/out /tmp/replay.sh /tmp/gnobin
```

The block above stops at the addpkg stage, which is where the four transactions carrying the address enter the stream and where the two revisions already diverge, `b71d574a…` against `5012e2f4…`. Adding the bootstrap, `names-enable` and valoper-seed transactions on top reaches the locked value itself: `499d9fba…` at the base and `79cea215…` at this head, against `499d9fbaaea8822d873a8e6693e329c9347bc69223daf056c4f31fe28aa437dc` in the heredoc. `work/valoper-seed.jsonl` reproduces its own locked `7717a8fc…` at both revisions, which is what shows the replay is faithful rather than merely different.

The same replay against `sapphire` gives `6f583f6d…` at the merge base against a locked `ed781241…`, and the package list regenerates to `2f686094…` at both revisions against `topaz`'s locked `dac29a0c…` and `test13`'s `397c3c90…`, which is why those three are not attributed here.

</details>

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

## examples/gno.land/p/gnoland/boards/exts/hub/comment_test.gno:14 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/p/gnoland/boards/exts/hub/comment_test.gno#L14) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/p/gnoland/boards/exts/hub/comment_test.gno#L14)
Suggestion: this address is bound to `bob` here, to `sink` at [`grc20_registry_wrappers.txtar:115`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/integration/testdata/grc20_registry_wrappers.txtar#L115) and to `chain_addr` at [`grc721_callerteller_home.txtar:93`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/integration/testdata/grc721_callerteller_home.txtar#L93), none of them ever the multisig, so three fixtures now read as governance references and will move again on the next rotation. Give them a fixture address of their own.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6131 -R gnolang/gno
export GNOROOT=$PWD
go build -o /tmp/gno ./gnovm/cmd/gno
NEW=g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf
OLD=g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
sed -i "14s/$NEW/$OLD/" examples/gno.land/p/gnoland/boards/exts/hub/comment_test.gno
sed -i "115s/$NEW/$OLD/" gno.land/pkg/integration/testdata/grc20_registry_wrappers.txtar
sed -i "93s/$NEW/$OLD/" gno.land/pkg/integration/testdata/grc721_callerteller_home.txtar
/tmp/gno test -C examples ./gno.land/p/gnoland/boards/exts/hub
go test ./gno.land/pkg/integration/ -run 'TestTestdata/(grc20_registry_wrappers|grc721_callerteller_home)'
git checkout -- examples gno.land && rm -f /tmp/gno
```

All three revert to the old address and nothing notices, which is what shows they carry no meaning:

```
ok      ./gno.land/p/gnoland/boards/exts/hub 	2.80s
ok  	github.com/gnolang/gno/gno.land/pkg/integration	6.485s
```

</details>
