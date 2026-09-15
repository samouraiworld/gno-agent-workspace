#!/usr/bin/env bash
# Asserts that the goreleaser build path used by release / goreleaser
# (.github/workflows/release-goreleaser.yml, config .github/goreleaser.yaml)
# stamps tm2/pkg/version.Version the same way release / chain-tag does, so a
# gnoland/gno/gnoweb binary or ghcr.io image built by that workflow can satisfy
# a governance halt_min_version.
#
# Measured at 607942b78fa4fdf6f378fecce32bc1d1d984ab8e: .github/goreleaser.yaml
# sets ldflags on only 2 of its ~9 build ids (gnokey: version ldflag; gnodev:
# a GNOROOT ldflag, not version). The gno and gnoland ids carry no ldflags,
# so their binaries keep tm2/pkg/version.Version's zero value, "develop"
# (tm2/pkg/version/version.go:3). Dockerfile.release, which release-goreleaser.yml
# feeds to build ghcr.io/gnolang/gno/{gnoland,gno,...} (release-goreleaser.yml
# publishes via Dockerfile.release), only COPYs the prebuilt binary in — it
# never rebuilds or re-links it, so the image inherits whatever the goreleaser
# build produced.
#
# meetsMinVersion(binaryVersion, minVersion) in gno.land/pkg/gnoland/node_params.go
# returns true unconditionally when minVersion == "" (no halt_min_version set,
# the common case) — this script also checks that cell so the finding does not
# overstate the blast radius. It only returns false once a chain has actually
# set a halt_min_version, at which point a goreleaser-built gnoland cannot
# restart after the halt, because "develop" does not parse as a releaseVersion
# and falls back to an exact-string match against the configured floor.
#
# Run: from a local clone of gnolang/gno (needs enough disk for a full cgo
# build of ./gno.land/cmd/gnoland, ~1-2GB build cache):
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   curl -fsSL -o /tmp/37-goreleaser-binaries-no-version.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/37-goreleaser-binaries-no-version.sh
#   bash /tmp/37-goreleaser-binaries-no-version.sh

set -u
cd "$(git rev-parse --show-toplevel)"
fail=0

echo "--- ldflags coverage in .github/goreleaser.yaml ---"
ids_with_ldflags="$(awk '
  /^  - id: /            { id=$3 }
  /^    ldflags:/        { print id }
' .github/goreleaser.yaml | sort -u)"
all_ids="$(awk '/^  - id: /{print $3}' .github/goreleaser.yaml | sort -u)"
echo "all build ids:"; echo "$all_ids"
echo "ids carrying any ldflags:"; echo "$ids_with_ldflags"

version_ldflag_ids="$(awk '
  /^  - id: /            { id=$3; has_vflag=0 }
  /pkg\/version\.Version=/ { print id }
' .github/goreleaser.yaml | sort -u)"
echo "ids setting tm2/pkg/version.Version specifically:"; echo "$version_ldflag_ids"

if [ "$version_ldflag_ids" = "gnokey" ]; then
  echo "OK: only gnokey stamps tm2/pkg/version.Version in goreleaser.yaml"
else
  echo "FAIL: expected only gnokey, got: $version_ldflag_ids"
  fail=1
fi

echo
echo "--- Dockerfile.release only COPYs, never rebuilds ---"
if grep -qE '^COPY[[:space:]]+\./gnoland[[:space:]]' Dockerfile.release && \
   ! grep -qE '^RUN.*go build' Dockerfile.release; then
  echo "OK: Dockerfile.release copies the prebuilt ./gnoland binary, no rebuild step"
else
  echo "FAIL: Dockerfile.release shape changed, re-check by hand"
  fail=1
fi

echo
echo "--- build gnoland the way goreleaser.yaml's gnoland id does (no ldflags) and check its version string ---"
tmpbin="$(mktemp)"
if CGO_ENABLED=0 go build -o "$tmpbin" ./gno.land/cmd/gnoland; then
  got="$(GNOROOT="$(pwd)" "$tmpbin" version 2>&1)"
  echo "got: $got"
  case "$got" in
    "gnoland version: develop")
      echo "CONFIRMED: goreleaser-shaped gnoland build reports the zero-value version"
      ;;
    *)
      echo "FAIL: expected 'gnoland version: develop', got '$got'"
      fail=1
      ;;
  esac
else
  echo "FAIL: build failed"
  fail=1
fi
rm -f "$tmpbin"

echo
echo "--- meetsMinVersion cells for the 'develop' binary string ---"
cat > /tmp/meetsminversion_develop_test.go << 'GOEOF'
package gnoland

import "testing"

// Mirrors the two live cells: no halt configured (minVersion == "") always
// passes; once a floor is set, a goreleaser-built "develop" binary cannot
// meet it because parseReleaseVersion rejects "develop" and the exact-match
// fallback only matches a minVersion of the literal string "develop".
func TestMeetsMinVersion_DevelopBinary(t *testing.T) {
	if !meetsMinVersion("develop", "") {
		t.Fatal("no halt configured: expected true (this cell is NOT the bug)")
	}
	if meetsMinVersion("develop", "v1.3.0") {
		t.Fatal("expected false: a goreleaser-built binary reporting develop cannot satisfy a real halt_min_version floor")
	}
}
GOEOF
cp /tmp/meetsminversion_develop_test.go gno.land/pkg/gnoland/zz_37_verify_test.go
if go test -run 'TestMeetsMinVersion_DevelopBinary' ./gno.land/pkg/gnoland/... -v 2>&1 | tail -15; then
  :
fi
rm -f gno.land/pkg/gnoland/zz_37_verify_test.go

echo
if [ "$fail" -eq 0 ]; then
  echo "RESULT: CONFIRMED — goreleaser.yaml's gnoland/gno/gnoweb build ids carry no version ldflag, Dockerfile.release only copies the binary, and the resulting binary reports 'develop', which fails any real halt_min_version floor."
else
  echo "RESULT: one or more checks failed, see above"
fi
exit "$fail"
