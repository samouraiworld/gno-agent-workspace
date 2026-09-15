#!/usr/bin/env bash
# Asserts that gno.land/cmd/gnoland/UPGRADES.md:208 ("`make build.gnoland` on a
# release tag | the tag, e.g. `v1.2.0`") is false whenever the release commit
# also carries a chain/ tag, which RELEASING.md:58 says mainnet's launch commit
# does. gno.land/Makefile:21 derives VERSION from an unfiltered
# `git describe --tags --exact-match`, which returns the chain/ tag there.
# Measured: both v tags in the repo that are dual-tagged report the chain/ tag,
# and a synthetic mainnet-shaped commit reports `chain/mainnet`, which
# gno.land/pkg/gnoland.meetsMinVersion does not accept for a v1.2.0 floor.
# Fails at head 607942b78.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   git fetch --tags origin
#   curl -fsSL -o /tmp/10-make-build-reports-chain-tag.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/10-make-build-reports-chain-tag.sh
#   bash /tmp/10-make-build-reports-chain-tag.sh
set -u
ROOT="$(git rev-parse --show-toplevel)"
fail=0

# The exact expression from gno.land/Makefile:21.
makefile_version() {
  git -C "$1" describe --tags --exact-match 2>/dev/null \
    || echo "$(git -C "$1" rev-parse --abbrev-ref HEAD).$(git -C "$1" rev-list --count HEAD)+$(git -C "$1" rev-parse --short HEAD)"
}

echo "== 1. every v tag in gnolang/gno, and what make build.gnoland would stamp =="
printf '%-12s %-34s %s\n' 'v TAG' 'TAGS ON THAT COMMIT' 'VERSION the Makefile derives'
for t in $(git -C "$ROOT" tag -l 'v*' | sort -V); do
  c=$(git -C "$ROOT" rev-list -n1 "$t")
  got=$(git -C "$ROOT" describe --tags --exact-match "$c" 2>/dev/null)
  printf '%-12s %-34s %s\n' "$t" "$(git -C "$ROOT" tag --points-at "$c" | paste -sd, -)" "$got"
  # IS:     a dual-tagged release tag stamps the chain/ tag, not the v tag
  # SHOULD: [ "$got" = "$t" ]   # the row at UPGRADES.md:208, once the Makefile filters on 'v*'
  [ "$got" = "$t" ] || fail=1
done

echo
echo "== 2. mainnet's launch commit, built to RELEASING.md:58 and :76 =="
# RELEASING.md:58 — chain/mainnet carries "v1.x.x, plus the chain/mainnet launch tag".
# RELEASING.md:76 — "Tags are annotated (git tag -a), so git describe prefers them."
S=$(mktemp -d); git init -q "$S"
git -C "$S" -c user.email=r@e -c user.name=r commit -q --allow-empty -m "mainnet launch"
git -C "$S" -c user.email=r@e -c user.name=r tag -a -m 'release' v1.2.0
git -C "$S" -c user.email=r@e -c user.name=r tag -a -m 'launch'  chain/mainnet
STAMPED="$(makefile_version "$S")"
echo "tags on the launch commit : $(git -C "$S" tag --points-at HEAD | paste -sd, -)"
echo "VERSION make build.gnoland stamps: ${STAMPED}"
# IS:     chain/mainnet — a bare chain tag RELEASING.md:70 calls "not a version"
# SHOULD: [ "$STAMPED" = "v1.2.0" ]
[ "$STAMPED" = "v1.2.0" ] || fail=1

echo
echo "== 3. does that binary satisfy halt_min_version=v1.2.0? (head's own meetsMinVersion) =="
cat > "$ROOT/gno.land/pkg/gnoland/zz_upgrades_claim_test.go" <<'GO'
package gnoland

import "testing"

// The version string `make build.gnoland` stamps at mainnet's launch commit,
// checked against the floor that same release would be gated on.
func TestUPGRADESClaim_StampedVersionMeetsItsOwnFloor(t *testing.T) {
	const stamped = "chain/mainnet"
	if _, ok := parseReleaseVersion(stamped); !ok {
		t.Logf("parseReleaseVersion(%q) does not parse; the comparison degrades to byte equality", stamped)
	}
	got := meetsMinVersion(stamped, "v1.2.0")
	t.Logf("meetsMinVersion(%q, %q) = %v", stamped, "v1.2.0", got)
	if !got {
		t.Errorf("a node built with `make build.gnoland` at the v1.2.0 launch commit is refused by halt_min_version=v1.2.0")
	}
}
GO
LOG=$(mktemp)
( cd "$ROOT" && go test -count=1 -run 'TestUPGRADESClaim_StampedVersionMeetsItsOwnFloor' -v ./gno.land/pkg/gnoland/ ) >"$LOG" 2>&1
rc=$?
grep -E 'meetsMinVersion|does not parse|refused|^(--- |ok|FAIL|PASS)' "$LOG"
rm -f "$LOG"
[ "$rc" = 0 ] || fail=1
rm -f "$ROOT/gno.land/pkg/gnoland/zz_upgrades_claim_test.go"
rm -rf "$S"

echo
[ "$fail" = 0 ] && echo "RESULT: PASS — UPGRADES.md:208 holds" || echo "RESULT: FAIL — UPGRADES.md:208 is false for a dual-tagged release commit"
exit "$fail"
