# PR [#6177](https://github.com/gnolang/gno/pull/6177): fix: make a release tag able to gate a chain upgrade, and add the tooling that checks it
Event: REQUEST_CHANGES
Model: claude-opus-5, standard review
Commit: 607942b78 (latest)
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/607942b78fa4fdf6f378fecce32bc1d1d984ab8e) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/607942b78fa4fdf6f378fecce32bc1d1d984ab8e)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6177 607942b78`
Round: 1. 6 finders, one critic, 55 candidates, each run from scratch by an agent that was not its finder.

## Body
- [`WillSetParam`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L69-L73) type-checks `p:halt_min_version` and never reads its value, unlike [`p:halt_height`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L60-L67) beside it, so a GovDAO proposal naming a floor the node cannot parse executes on chain and the operators find out at the restart.
- [`gnoland`](https://github.com/gnolang/gno/blob/607942b78/.github/goreleaser.yaml#L26-L36), [`gno`](https://github.com/gnolang/gno/blob/607942b78/.github/goreleaser.yaml#L15-L25) and [`gnoweb`](https://github.com/gnolang/gno/blob/607942b78/.github/goreleaser.yaml#L50-L60) are built in `.github/goreleaser.yaml` with no version ldflag, [`gnokey`](https://github.com/gnolang/gno/blob/607942b78/.github/goreleaser.yaml#L40-L41) being the one binary that carries it, and [`Dockerfile.release`](https://github.com/gnolang/gno/blob/607942b78/Dockerfile.release#L14) copies the artifact rather than rebuilding it, so that path publishes a `gnoland` reporting `develop`.
- [`VersionSet.CompatibleWith`](https://github.com/gnolang/gno/blob/607942b78/tm2/pkg/versionset/versionset.go#L59) carries no test anywhere in the tree and its godoc still reads [`// TODO: test`](https://github.com/gnolang/gno/blob/607942b78/tm2/pkg/versionset/versionset.go#L58), while [`RELEASING.md:163-165`](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L163-L165) and [`bump-protocol-version.sh:130-137`](https://github.com/gnolang/gno/blob/607942b78/misc/release/bump-protocol-version.sh#L130-L137) now rest on its major-versus-minor split.
- [`versionset.go:92`](https://github.com/gnolang/gno/blob/607942b78/tm2/pkg/versionset/versionset.go#L92) asks whether one major is greater than the other inside the branch that has already proved the two equal, so the arm keeping the peer's `MajorMinor` never runs and the negotiated minor is always the receiver's.

## gno.land/pkg/gnoland/node_params.go:259 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L259) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/node_params.go#L259)
`v.pre < o.pre` compares pre-releases as bytes, so a halt floor of `v1.3.0-rc.10` admits rc.2 through rc.9 and refuses the rc.10 binary the upgrade was cut for. [The comment above it](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L249-L251) claims the opposite for the `rc.N` shape [`RELEASING.md:77`](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L77) allows, and [`golang.org/x/mod/semver`](https://github.com/gnolang/gno/blob/607942b78/go.mod#L55) is already a direct dependency.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno

cat > gno.land/pkg/gnoland/zz_rc_order_test.go <<'EOF'
package gnoland

import (
	"fmt"
	"testing"

	"golang.org/x/mod/semver"
)

func TestRCOrdering(t *testing.T) {
	for _, p := range [][2]string{
		{"v1.3.0-rc.2", "v1.3.0-rc.1"},
		{"v1.3.0-rc.1", "v1.3.0-rc.2"},
		{"v1.3.0-rc.10", "v1.3.0-rc.2"},
		{"v1.3.0-rc.2", "v1.3.0-rc.10"},
		{"v1.3.0-rc.11", "v1.3.0-rc.9"},
	} {
		gate := meetsMinVersion(p[0], p[1])
		want := semver.Compare(p[0], p[1]) >= 0
		fmt.Printf("  %-13s floor %-13s gate=%-5v semver=%v\n", p[0], p[1], gate, want)
		if gate != want {
			t.Errorf("meetsMinVersion(%q, %q) = %v, semver says %v", p[0], p[1], gate, want)
		}
	}
}
EOF
go test -count=1 -v -run TestRCOrdering ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/zz_rc_order_test.go
```

Every expectation is `semver.Compare` rather than a hand-written value, and the two rows the gate answers backwards are the two with a two-digit release candidate:

```
=== RUN   TestRCOrdering
  v1.3.0-rc.2   floor v1.3.0-rc.1   gate=true  semver=true
  v1.3.0-rc.1   floor v1.3.0-rc.2   gate=false semver=false
  v1.3.0-rc.10  floor v1.3.0-rc.2   gate=false semver=true
  v1.3.0-rc.2   floor v1.3.0-rc.10  gate=true  semver=false
  v1.3.0-rc.11  floor v1.3.0-rc.9   gate=false semver=true
--- FAIL: TestRCOrdering (0.00s)
```

The table at [`node_params_version_test.go:96-97`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params_version_test.go#L96-L97) pairs `rc.2` against `rc.1`, where byte order and numeric order agree, so no row reaches the case.
</details>

## gno.land/pkg/gnoland/node_params.go:279-281 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L279-L281) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/node_params.go#L279)
`chain/` tags reach `strconv.Atoi` without the guards `v` tags get at [`:302-310`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L302-L310), so a floor of `chain/gnoland-1.0` parses as major -1 and every binary whose own version parses clears it. That shape is one hyphen from the example [`halt.gno:25`](https://github.com/gnolang/gno/blob/607942b78/examples/gno.land/r/sys/params/halt.gno#L25) gives governance, and every other typo there fails closed.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno

cat > gno.land/pkg/gnoland/zz_legacy_guard_test.go <<'EOF'
package gnoland

import (
	"fmt"
	"testing"
)

func TestLegacyGuards(t *testing.T) {
	for _, s := range []string{"chain/gnoland-1.0", "chain/gnoland1.01", "chain/gnoland1.+1", "chain/gnoland01.0", "v-1.2.0", "v1.02.0", "v1.+2.0"} {
		got, ok := parseReleaseVersion(s)
		fmt.Printf("  %-20q ok=%-6v %+v\n", s, ok, got)
	}
	for _, p := range [][2]string{
		{"v1.2.0", "chain/gnoland-1.0"},
		{"chain/gnoland1.0", "chain/gnoland-1.0"},
		{"develop", "chain/gnoland-1.0"},
	} {
		fmt.Printf("  binary=%-18q floor=%-20q -> %v\n", p[0], p[1], meetsMinVersion(p[0], p[1]))
	}
}
EOF
go test -count=1 -v -run TestLegacyGuards ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/zz_legacy_guard_test.go
```

Four shapes that were never a tag parse, and the negative major is a floor nothing sits below:

```
  "chain/gnoland-1.0"  ok=true   {major:-1 minor:0 patch:0 pre:}
  "chain/gnoland1.01"  ok=true   {major:1 minor:1 patch:0 pre:}
  "chain/gnoland1.+1"  ok=true   {major:1 minor:1 patch:0 pre:}
  "chain/gnoland01.0"  ok=true   {major:1 minor:0 patch:0 pre:}
  "v-1.2.0"            ok=false  {major:0 minor:0 patch:0 pre:}
  "v1.02.0"            ok=false  {major:0 minor:0 patch:0 pre:}
  "v1.+2.0"            ok=false  {major:0 minor:0 patch:0 pre:}
  binary="v1.2.0"           floor="chain/gnoland-1.0"  -> true
  binary="chain/gnoland1.0" floor="chain/gnoland-1.0"  -> true
  binary="develop"          floor="chain/gnoland-1.0"  -> false
```
</details>

<details><summary>test cases</summary>

Paste into the malformed-input block of [`TestParseReleaseVersion`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params_version_test.go#L51-L52), whose only legacy negatives today are the missing-minor and non-numeric ones:

```go
		{"legacy signed component", "chain/gnoland1.+1", releaseVersion{}, false},
		{"legacy negative component", "chain/gnoland-1.0", releaseVersion{}, false},
		{"legacy leading zero", "chain/gnoland1.01", releaseVersion{}, false},
```
</details>

## .github/workflows/release-chain-tag.yml:19 [gh](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L19) · [↗](../../../../../.worktrees/gno-review-6177/.github/workflows/release-chain-tag.yml#L19)
A `v*` tag reaches this workflow and no other: [`release / docker`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-docker.yml#L8-L9) still keys on `chain/*` alone, so a release ships the four binaries and no container image. [`cut-release.sh:347`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L347) pushes that one ref, and every network's [`VALIDATOR.md`](https://github.com/gnolang/gno/blob/607942b78/misc/deployments/mainnet.gno.land/VALIDATOR.md?plain=1#L33) sends validators to `ghcr.io/gnolang/gno/gnoland`.

## .github/workflows/release-chain-tag.yml:83 [gh](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L83) · [↗](../../../../../.worktrees/gno-review-6177/.github/workflows/release-chain-tag.yml#L83)
`-trimpath` strips the absolute source path [`guessRootDir`](https://github.com/gnolang/gno/blob/607942b78/gnovm/pkg/gnoenv/gnoroot.go#L63-L65) needs, and no `_GNOROOT` ldflag replaces it, so the published `gnoland` and `gno` panic on every subcommand unless `GNOROOT` is set. [`.github/goreleaser.yaml:75-77`](https://github.com/gnolang/gno/blob/607942b78/.github/goreleaser.yaml#L75-L77) and [`Dockerfile:61`](https://github.com/gnolang/gno/blob/607942b78/Dockerfile#L61) both pass that ldflag.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno

L="-w -s -X github.com/gnolang/gno/tm2/pkg/version.Version=v1.2.0"
go build -trimpath -ldflags "$L" -o /tmp/gno-trim  ./gnovm/cmd/gno
go build            -ldflags "$L" -o /tmp/gno-plain ./gnovm/cmd/gno
for b in gno-trim gno-plain; do
  printf '%-10s ' "$b"
  env -i PATH=/usr/bin:/bin HOME=/tmp "/tmp/$b" version 2>&1 | head -1
done
rm -f /tmp/gno-trim /tmp/gno-plain
```

The flag is the only difference between the two builds, and the first is the one the release publishes:

```
gno-trim   panic: gno was unable to determine GNOROOT. Please set the GNOROOT environment variable
gno-plain  gno version: v1.2.0
```

`GNOROOT` pointing at a path that does not exist is enough to make the trimmed binary answer `gno version: v1.2.0`, so the absolute-path test is the whole of it. `gnokey` is unaffected, never reaching `gnoenv.RootDir`, and the assert step at [`:90-104`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L90-L104) sets `GNOROOT` for itself, which is why it answers `v1.2.0` for a binary a downloader cannot run.
</details>

## .github/workflows/release-chain-tag.yml:138 [gh](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L138) · [↗](../../../../../.worktrees/gno-review-6177/.github/workflows/release-chain-tag.yml#L138)
`gh release create` runs with neither `--prerelease` nor `--latest=false`, both [opt-in flags](https://cli.github.com/manual/gh_release_create), so a `v1.3.0-rc.1` tag publishes as a full release and takes the repository's Latest badge. [`RELEASING.md:77`](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L77) allows that shape, [`check_version_shape`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L115) accepts it, and GitHub gives Latest to [every newly published release](https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28#create-a-release).

## gno.land/cmd/gnoland/UPGRADES.md:208 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/cmd/gnoland/UPGRADES.md?plain=1#L208) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/cmd/gnoland/UPGRADES.md#L208)
[`gno.land/Makefile:21`](https://github.com/gnolang/gno/blob/607942b78/gno.land/Makefile#L21) derives the version from an unfiltered `git describe --tags --exact-match`, which answers the `chain/` tag on a commit carrying both shapes. The launch commit carries `chain/mainnet` beside its `v` tag, so the binary stamps a string [`parseReleaseVersion`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L272) refuses.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno
git fetch --tags origin

for t in v1.0.0 v1.1.0; do
  c="$(git rev-parse "$t^{commit}")"
  printf '%-8s tags=%-28s describe=%s\n' "$t" \
    "$(git tag --points-at "$c" | paste -sd,)" \
    "$(git describe --tags --exact-match "$c")"
done
```

Both commits carry a `v` tag and a `chain/` tag, and `describe` returns the `chain/` one on each:

```
v1.0.0   tags=chain/gnoland1.0,v1.0.0    describe=chain/gnoland1.0
v1.1.0   tags=chain/gnoland1.1,v1.1.0    describe=chain/gnoland1.1
```

`chain/mainnet` and `v1.2.0` sit on one commit the same way, and `git describe --match 'v*'` is what picks the release tag there.
</details>

## misc/release/cut-release.sh:84-86 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L84-L86) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L84)
`--halt-height` takes `${2-}` and nothing between here and [`emit_halt_proposal`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L279-L310) reads the value again, while every other input is preflighted:

- `0` emits `NewSetHaltRequest(cross(cur), 0, "v1.3.0")`, which [`halt.gno:40-41`](https://github.com/gnolang/gno/blob/607942b78/examples/gno.land/r/sys/params/halt.gno#L40-L41) renders on chain as `Cancel the scheduled chain halt`, under a generated header reading `Every node stops after committing block 0`.
- `abc` emits a body no Gno parser accepts, and the run still prints `wrote ...` and creates the tag.
- a missing value swallows the next argument, so `--halt-height --push` emits `NewSetHaltRequest(cross(cur), --push, "v1.3.0")` and pushes nothing.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno

# the script's own emit_halt_proposal, with only the trailing `main "$@"` dropped,
# rooted at a scratch tree so nothing is written into the checkout.
S=$(mktemp -d); mkdir -p "$S/misc/release" "$S/misc/deployments/mainnet.gno.land"
sed '$d' misc/release/cut-release.sh > "$S/misc/release/lib.sh"
cd "$S" && . misc/release/lib.sh
for h in 0 abc --push 120000; do
  HALT_HEIGHT="$h"; VERSION="v1.3.0"; CHAIN="mainnet"
  emit_halt_proposal >/dev/null 2>&1
  f=misc/deployments/mainnet.gno.land/transactions/migration/halt-v1.3.0/halt_v1_3_0.gno
  printf -- '--halt-height %-8s call: %s | header: %s\n' "$h" \
    "$(grep NewSetHaltRequest $f | sed 's/^\t*//')" "$(sed -n 3p $f | sed 's|^// ||')"
done
cd - >/dev/null && rm -rf "$S"
```

The first three rows are values no check rejects, and the last is the one the documentation uses:

```
--halt-height 0        call: req := params.NewSetHaltRequest(cross(cur), 0, "v1.3.0")      | header: Every node stops after committing block 0, and refuses to
--halt-height abc      call: req := params.NewSetHaltRequest(cross(cur), abc, "v1.3.0")    | header: Every node stops after committing block abc, and refuses to
--halt-height --push   call: req := params.NewSetHaltRequest(cross(cur), --push, "v1.3.0") | header: Every node stops after committing block --push, and refuses to
--halt-height 120000   call: req := params.NewSetHaltRequest(cross(cur), 120000, "v1.3.0") | header: Every node stops after committing block 120000, and refuses to
```
</details>

## misc/release/cut-release.sh:115 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L115) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L115)
The regex accepts the leading zeros in `v1.02.0`, `v01.2.0` and `v1.2.00`, and [`parseReleaseVersion`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L304-L306) rejects all three, so this preflight blesses a tag whose halt proposal orders nothing. [The comment above it](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L112-L113) calls the regex the shape the node parses, and the regex also refuses `v1.2.0+deadbeef`, which the node takes.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno

cat > gno.land/pkg/gnoland/zz_shape_test.go <<'EOF'
package gnoland

import (
	"fmt"
	"regexp"
	"testing"
)

func TestShapeMatchesParser(t *testing.T) {
	// The regex check_version_shape applies, from misc/release/cut-release.sh:115.
	shape := regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
	for _, v := range []string{"v1.2.0", "v1.3.0-rc.1", "v1.02.0", "v01.2.0", "v1.2.00", "v1.2.0+deadbeef", "v1.2", "1.2.0"} {
		sh := shape.MatchString(v)
		_, parsed := parseReleaseVersion(v)
		fmt.Printf("  %-16s cut-release.sh=%-6v parseReleaseVersion=%v\n", v, sh, parsed)
		if sh != parsed {
			t.Errorf("%q: the two validators disagree", v)
		}
	}
}
EOF
go test -count=1 -v -run TestShapeMatchesParser ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/zz_shape_test.go
```

Four of the eight shapes disagree, the three leading-zero tags being the live cell, since those are the ones a release engineer can type and tag:

```
  v1.2.0           cut-release.sh=true   parseReleaseVersion=true
  v1.3.0-rc.1      cut-release.sh=true   parseReleaseVersion=true
  v1.02.0          cut-release.sh=true   parseReleaseVersion=false
    zz_shape_test.go:17: "v1.02.0": the two validators disagree
  v01.2.0          cut-release.sh=true   parseReleaseVersion=false
  v1.2.00          cut-release.sh=true   parseReleaseVersion=false
  v1.2.0+deadbeef  cut-release.sh=false  parseReleaseVersion=true
  v1.2             cut-release.sh=false  parseReleaseVersion=false
  1.2.0            cut-release.sh=false  parseReleaseVersion=false
```

</details>

## misc/release/cut-release.sh:184 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L184) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L184)
[`bump-protocol-version.sh`](https://github.com/gnolang/gno/blob/607942b78/misc/release/bump-protocol-version.sh#L29) takes no ref and resolves `REPO_ROOT` from its own path, so this check reads the working tree rather than the commit being tagged. [`check_build_reports_tag`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L197-L198) two lines below builds a detached worktree at `${COMMIT}`, and a clean worktree closes nothing here: the wrong commit is not a dirty file.

## misc/release/cut-release.sh:224-225 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L224-L225) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L224)
`--sort=-v:refname` over every `v*` tag in the repository ranks a pre-release above the release it leads to and filters nothing by reachability:

- cutting `v1.3.0` after `v1.3.0-rc.1` picks the candidate as `PREVIOUS`, and [`classify`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L242-L246) compares major and minor only, so the run prints `PATCH: no validator coordination needed` and never names `--halt-height`.
- a `v2.0.0` tag anywhere in the repository becomes `PREVIOUS` for a `v1.2.1` cut on the older line.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno

S=$(mktemp -d); cd "$S"; git init -q .; git commit -q --allow-empty -m init
git tag -a v1.2.0 -m v1.2.0; git tag -a v1.3.0-rc.1 -m v1.3.0-rc.1; git tag -a v1.3.0 -m v1.3.0
echo "newest_release_tag: $(git tag --list 'v*' --sort=-v:refname | awk 'NR==1')"
echo "sort order:         $(git tag --list 'v*' --sort=-v:refname | paste -sd' ')"
prev=1.3.0-rc.1; new=1.3.0
echo "classify: prev_minor=$(printf %s "$prev" | cut -d. -f2) new_minor=$(printf %s "$new" | cut -d. -f2)"
cd - >/dev/null && rm -rf "$S"
```

Equal majors and equal minors are what `classify` reads, so the MINOR branch is not reached:

```
newest_release_tag: v1.3.0-rc.1
sort order:         v1.3.0-rc.1 v1.3.0 v1.2.0
classify: prev_minor=3 new_minor=3
```

Neither documented invocation, [`RELEASING.md:96`](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L96) and [`:99`](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L99), passes `--previous`.
</details>

## gno.land/pkg/gnoland/node_params_version_test.go:68 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params_version_test.go#L68) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/node_params_version_test.go#L68)
Missing test: no test drives a release tag through [`checkNodeStartupParams`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L134), the caller both halt checks live in, so neither outcome the release flow exists for is pinned. [`tm2/pkg/version.Version`](https://github.com/gnolang/gno/blob/607942b78/tm2/pkg/version/version.go#L3) is `develop` under `go test` and [`TestCheckNodeStartupParams`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/app_test.go#L3398) names only `develop` and `chain/gnoland9.9`, so every case takes the byte-equality fallback.

<details><summary>test cases</summary>

Both rows fail against the merge base's parser and pass here, so they discriminate the change this file otherwise tests one level down:

```go
func TestStartupGateWithAReleaseTag(t *testing.T) {
	t.Parallel()

	orig := tmver.Version
	tmver.Version = "v1.2.0"
	t.Cleanup(func() { tmver.Version = orig })

	t.Run("upgraded binary readmitted after halt", func(t *testing.T) {
		prmk, ms := newTestParamsKeeper(t, 100, "v1.1.0")
		require.NoError(t, checkNodeStartupParams(ms.MultiCacheWrap(), prmk, 100, 0))
	})

	t.Run("upgraded binary refused before halt", func(t *testing.T) {
		prmk, ms := newTestParamsKeeper(t, 100, "v1.1.0")
		require.Error(t, checkNodeStartupParams(ms.MultiCacheWrap(), prmk, 50, 0))
	})
}
```
</details>

## misc/release/cut-release.sh:37 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L37) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L37)
Missing test: no pull-request job reads `misc/release`, and [`ci-dir-misc.yml`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/ci-dir-misc.yml#L37-L43)'s fixed `program:` matrix has no `release` row, so nothing holds these 519 lines to the parser they mirror. One Go test holds [`check_version_shape`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L115)'s regex to `parseReleaseVersion`, runs in a package the pull-request jobs already build, and reddens today.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno
wc -l misc/release/*.sh
grep -rnE 'shellcheck|shfmt|bash -n' .github/
grep -rn 'misc/release' .github/ Makefile*
```

The only match under `.github/` is a suppression inside an unrelated job, and no workflow names the directory:

```
  163 misc/release/bump-protocol-version.sh
  356 misc/release/cut-release.sh
  519 total
.github/workflows/_ci-go.yml:130:          # shellcheck disable=SC2086
```
</details>

## docs/resources/gnoland-networks.md:8 [gh](https://github.com/gnolang/gno/blob/607942b78/docs/resources/gnoland-networks.md?plain=1#L8) · [↗](../../../../../.worktrees/gno-review-6177/docs/resources/gnoland-networks.md#L8)
Nit: the Betanet cell's `../../misc/deployments/betanet` resolves same-origin on the published site and answers 404 there, as the Staging cell's `../../misc/loop` already does. The Mainnet and Pearl cells carry absolute GitHub URLs for the same kind of target.

<details><summary>repro</summary>

```bash
for u in https://docs.gno.land/resources/gnoland-networks \
         https://docs.gno.land/misc/loop \
         https://docs.gno.land/misc/deployments/betanet; do
  printf '%-52s %s\n' "$u" "$(curl -s -o /dev/null -w '%{http_code}' -L "$u")"
done
```

The page renders and both relative targets do not:

```
https://docs.gno.land/resources/gnoland-networks     200
https://docs.gno.land/misc/loop                      404
https://docs.gno.land/misc/deployments/betanet       404
```
</details>

## .github/workflows/release-chain-tag.yml:68 [gh](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L68) · [↗](../../../../../.worktrees/gno-review-6177/.github/workflows/release-chain-tag.yml#L68)
Refactor: `${{ inputs.tag || github.ref_name }}` is spelled out four times, here and at [`:94`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L94), [`:135`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L135) and [`:147`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L147), where two job-level `env:` lines carry it once per job.

## gno.land/pkg/gnoland/app_test.go:2954-2957 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/app_test.go#L2954-L2957) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/app_test.go#L2954)
Refactor: `TestParseGnolandVersion` is named nowhere else in the tree, so this comment is the only trace of a symbol a reader greps for and does not find.

```suggestion
```

## gno.land/pkg/gnoland/node_params.go:293-294 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L293-L294) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/node_params.go#L293)
Nit: both `strings.Cut` calls discard the `found` result, so a floor of `v1.2.0-` or `v1.2.0+` parses as the final release `v1.2.0`.

<details><summary>test cases</summary>

Paste into the malformed-input block of [`TestParseReleaseVersion`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params_version_test.go#L42-L52); both fail today:

```go
		{"trailing dash", "v1.2.0-", releaseVersion{}, false},
		{"trailing plus", "v1.2.0+", releaseVersion{}, false},
```
</details>

## misc/release/bump-protocol-version.sh:148 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/bump-protocol-version.sh#L148) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/bump-protocol-version.sh#L148)
Nit: a `die` inside the patch loop leaves the already-patched constants on disk unnamed, and the next run stops at [`constants already disagree`](https://github.com/gnolang/gno/blob/607942b78/misc/release/bump-protocol-version.sh#L90) rather than resuming.

## misc/release/cut-release.sh:63 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L63) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L63)
Nit: `sed -n '2,34p'` stops one line short of the header, so `--help` prints the caption and drops [line 35](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L35) under it, the only place `--halt-height` is shown in use.

```suggestion
usage() { sed -n '2,35p' "${BASH_SOURCE[0]}" | sed 's|^# \{0,1\}||'; }
```

## misc/release/cut-release.sh:170-171 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L170-L171) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L170)
Nit: the `git log origin/master..` substitution carries no `2>/dev/null`, unlike the checks at [`:158`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L158) and [`:164`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L164), so a checkout missing `origin/master` aborts the preflight with `fatal: bad revision`.

## misc/release/README.md:30-31 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/README.md?plain=1#L30-L31) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/README.md#L30)
Nit: [`emit_halt_proposal`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L284-L308) writes its file before the tag, so `git tag -d` leaves an untracked `transactions/migration/halt-<version>/` behind and the next run stops at [`check_worktree`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L124-L126).

## RELEASING.md:20 [gh](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L20) · [↗](../../../../../.worktrees/gno-review-6177/RELEASING.md#L20)
Nit: nothing under `misc/release/` names `AppVersion`, and the third surface is the literal [`baseApp.SetAppVersion("dev")`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/app.go#L107), so the tooling moves two of the three rows above.

## RELEASING.md:59 [gh](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L59) · [↗](../../../../../.worktrees/gno-review-6177/RELEASING.md#L59)
Nit: origin carries `chain/gnoland1` and no `chain/betanet`, so [`resolve_commit`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L143-L144) dies with `no origin/chain/betanet` for the branch this row names.

## RELEASING.md:76 [gh](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L76) · [↗](../../../../../.worktrees/gno-review-6177/RELEASING.md#L76)
Nit: `v1.0.0` on origin is a lightweight tag, so plain `git describe` skips it at its own commit and `git describe --tags` answers `chain/gnoland1.0` there.

## RELEASING.md:109 [gh](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L109) · [↗](../../../../../.worktrees/gno-review-6177/RELEASING.md#L109)
Nit: [`check_on_master`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L157-L181) ends in `ok` or `warn` on every path and [never `die`](https://github.com/gnolang/gno/blob/607942b78/misc/release/README.md?plain=1#L47), so item 3 is the one entry here that does not stop the tag.

## tm2/pkg/bft/version/version_test.go:22 [gh](https://github.com/gnolang/gno/blob/607942b78/tm2/pkg/bft/version/version_test.go#L22) · [↗](../../../../../.worktrees/gno-review-6177/tm2/pkg/bft/version/version_test.go#L22)
Nit: a drift in any of the six constants panics in an `init()` before `go test` reaches this function, so its `assert.Equal` message never prints. [`RELEASING.md:160`](https://github.com/gnolang/gno/blob/607942b78/RELEASING.md?plain=1#L160) names this test as what reports drift.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno

sed -i 's/const Version = "v1.0.0-rc.0"/const Version = "v1.0.0-rc.1"/' tm2/pkg/crypto/version.go
go test -count=1 -run TestProtocolVersionsAgree ./tm2/pkg/bft/version/ 2>&1 | head -4
git checkout -- tm2/pkg/crypto/version.go
```

The failure arrives from package initialisation, with no test name and none of the guidance the assertion carries:

```
panic: protocol version mismatch: crypto.Version is v1.0.0-rc.1 but BlockVersion is v1.0.0-rc.0; ...
	github.com/gnolang/gno/tm2/pkg/bft/types/version.init.0()
FAIL	github.com/gnolang/gno/tm2/pkg/bft/version	0.011s
```
</details>

## misc/release/bump-protocol-version.sh:33-40 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/bump-protocol-version.sh#L33-L40) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/bump-protocol-version.sh#L33)
Suggestion: one constant in a dependency-free leaf package, aliased by the six declarations, makes a bump a one-line edit. It removes this list, the three `init()` guards and [`TestProtocolVersionsAgree`](https://github.com/gnolang/gno/blob/607942b78/tm2/pkg/bft/version/version_test.go#L22).

<details><summary>what it removes</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno
wc -l misc/release/bump-protocol-version.sh tm2/pkg/bft/version/version_test.go
```

```
  163 misc/release/bump-protocol-version.sh
   52 tm2/pkg/bft/version/version_test.go
  215 total
```

The six declarations become aliases of one constant, so the equality the script writes and the test asserts holds by construction.
</details>

## misc/release/cut-release.sh:201-213 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L201-L213) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L201)
Suggestion: this ldflags string and version-extracting `awk` are copied at [`release-chain-tag.yml:71`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L71) and [`:98`](https://github.com/gnolang/gno/blob/607942b78/.github/workflows/release-chain-tag.yml#L98), so the check that proves the workflow's flags apply shares no line with the workflow. A rename of [`tm2/pkg/version.Version`](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L40) applied to one leaves the other checking a symbol nothing declares.

## misc/release/cut-release.sh:280 [gh](https://github.com/gnolang/gno/blob/607942b78/misc/release/cut-release.sh#L280) · [↗](../../../../../.worktrees/gno-review-6177/misc/release/cut-release.sh#L280)
Suggestion: every entry under `transactions/migration/` is a `meta.json`, and [`gen-genesis.sh`](https://github.com/gnolang/gno/blob/607942b78/misc/deployments/mainnet.gno.land/gen-genesis.sh#L1663-L1677) reads one directory it names rather than globbing, so the bare `.gno` written here has no reader. test13's hand-written proposals reach their `.gno` through a [`body_file`](https://github.com/gnolang/gno/blob/607942b78/misc/deployments/test13.gno.land/transactions/patched/set-minfee/h274926/meta.json#L16) key instead.

## SKIP gno.land/pkg/gnoland/node_params.go:236-247 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params.go#L236-L247) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/node_params.go#L236)
Refactor: the loop over the `[][2]int` literal is three `cmp.Compare` guards, 9 lines against 11 including the import, with `TestParseReleaseVersion` and `TestMeetsMinVersion` green both ways.

Skipped: adopting `semver.Compare` for the pre-release ordering removes this loop outright, so the two edits collide.

## SKIP gno.land/pkg/gnoland/node_params_version_test.go:80-83 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params_version_test.go#L80-L83) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/node_params_version_test.go#L80)
Nit: a `chain/mainnet` floor still falls through to byte equality, which the rows at [`:120-121`](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params_version_test.go#L120-L121) assert, so it is not a direction this table orders.

Skipped: a finding on a code comment's own wording changes no behaviour; the measurement is in `claims.md`.

## SKIP gno.land/pkg/gnoland/node_params_version_test.go:109 [gh](https://github.com/gnolang/gno/blob/607942b78/gno.land/pkg/gnoland/node_params_version_test.go#L109) · [↗](../../../../../.worktrees/gno-review-6177/gno.land/pkg/gnoland/node_params_version_test.go#L109)
Test: this row passes on arithmetic, not on a separation of the two tag shapes: `1.1` is below `1.2`, and `{"chain/gnoland1.1", "v1.1.0", false}` fails on the same number line.

Skipped: the behaviour is what the doc comment describes, so the finding is about the case's name.

## SKIP tm2/pkg/bft/version/version_test.go:42-50 [gh](https://github.com/gnolang/gno/blob/607942b78/tm2/pkg/bft/version/version_test.go#L42-L50) · [↗](../../../../../.worktrees/gno-review-6177/tm2/pkg/bft/version/version_test.go#L42)
Test: an empty version on one side is refused by every non-empty peer rather than matching it, so the case the comment describes is both sides empty at once. The assertion is true by construction: all four `VersionSet.Set` arguments are package constants the `init()` guard has already compared.

Skipped: a finding on a code comment's own wording, and an assertion that is true by construction changes no behaviour.

## SKIP run
Cost and shape of the round that produced this draft. The upload script drops every `SKIP` section before it validates anchors, so nothing here posts.

| Stage | Agents | Turns | Output | Cache read | Cache write |
| --- | ---: | ---: | ---: | ---: | ---: |
| parent | 1 | 257 | 239,750 | 86,964,570 | 1,498,864 |
| workflow agents | 55 | 2,463 | 970,482 | 154,991,343 | 9,481,595 |
| total | 56 | 2,720 | 1,210,232 | 241,955,913 | 10,980,459 |

| | |
| --- | --- |
| Preset | standard, `finder.cap` 12 from +1193 added lines |
| Stages | 6 finders, one verifier per candidate, critic, writer, text pass |
| Candidates | 55 raised, 51 kept, 4 refuted |
| Agents failed | 0 |
| Wall clock | 130 min |
| Tool calls | 1,352 |
| Cost at Opus 5 list rates | $219.89 — $30 output, $121 cache read, $69 cache write |

Cache is 86% of the round. Every agent opens near 16k of context and closes between 70k and 136k, so what each tool result leaves behind is re-read on every later call of that agent. `finder.cap` has since been pinned flat at 6, which projects this diff at 41 agents rather than 72.
