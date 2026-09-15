#!/usr/bin/env bash
# Asserts that the release tag shape RELEASING.md now prescribes, vMAJOR.MINOR.PATCH,
# reaches both release workflows, and that the container image of a release commit
# reports a version meetsMinVersion accepts.
# Measured at 607942b78: v1.3.0 fires release-chain-tag.yml only; the image the
# chain-branch push builds carries BUILD_VERSION=chain/mainnet.<N>+<sha>, and
# meetsMinVersion("chain/mainnet.2+8d1d4e2", "v1.3.0") is false.
# Both assertions fail at the reviewed head.
#
#   tag                base: binaries/images   head: binaries/images
#   v1.3.0             no  / no                yes / no      <- the new cell
#   chain/gnoland1.2   yes / yes               yes / yes
#   chain/mainnet      yes / yes               yes / yes
#
# cut-release.sh:115 rejects any tag that is not vMAJOR.MINOR.PATCH and :347
# pushes that one ref, so the release the tooling cuts never reaches release-docker.yml.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   curl -fsSL -o /tmp/09-release-docker-skips-v-tags.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/09-release-docker-skips-v-tags.sh
#   bash /tmp/09-release-docker-skips-v-tags.sh

set -u
cd "$(git rev-parse --show-toplevel)"
fail=0

# --- Part 1: which release workflow fires for the tag RELEASING.md prescribes ---
# Extract the on.push.tags globs of one workflow file.
tags_of() {
  awk '
    /^on:/                   {on=1; next}
    /^[^[:space:]#]/         {on=0; push=0; tags=0}
    on && /^  push:/         {push=1; next}
    on && push && /^  [^ ]/  {push=0; tags=0}
    push && /^    tags:/     {tags=1; next}
    push && tags && /^    [^ -]/ {tags=0}
    tags && /^      - /      {gsub(/^      - "?|"$/, ""); print}
  ' "$1"
}

TAG=v1.3.0   # the shape cut-release.sh enforces and RELEASING.md documents
echo "release tag under test: $TAG"

# Does any glob in this workflow's on.push.tags match $TAG?
fires() {
  local h=no g
  while read -r g; do
    [ -n "$g" ] || continue
    # shellcheck disable=SC2053
    [[ $TAG == $g ]] && h=yes
  done < <(tags_of "$1")
  echo "$h"
}

for wf in release-chain-tag.yml release-docker.yml; do
  printf '  %-24s tags=[%s] fires=%s\n' "$wf" \
    "$(tags_of ".github/workflows/$wf" | tr '\n' ' ' | sed 's/ $//')" "$(fires ".github/workflows/$wf")"
done

chain_hit=$(fires .github/workflows/release-chain-tag.yml)
docker_hit=$(fires .github/workflows/release-docker.yml)

# SHOULD: a release tag builds both the binaries and the container images.
if [ "$chain_hit" = yes ] && [ "$docker_hit" = yes ]; then
  echo "PASS part 1: $TAG builds binaries and images"
else
  echo "FAIL part 1: binaries=$chain_hit images=$docker_hit for $TAG"
  echo "         IS:     binaries=yes images=no   — the release ships no container image"
  echo "         SHOULD: binaries=yes images=yes"
  fail=1
fi

# --- Part 2: the version baked into the only image a release commit gets ---
# release-docker.yml also fires on a push to a chain/* branch. Its "Compute build
# version" step runs `git describe --tags --exact-match` there; the v tag is pushed
# afterwards by cut-release.sh, so the fallback branch.count+sha is what is compiled in.
sim=$(mktemp -d)
git -C "$sim" init -q -b chain/mainnet .
git -C "$sim" commit -q --allow-empty -m one
git -C "$sim" commit -q --allow-empty -m two
BUILD_VERSION=$(git -C "$sim" describe --tags --exact-match 2>/dev/null \
  || echo "$(git -C "$sim" rev-parse --abbrev-ref HEAD).$(git -C "$sim" rev-list --count HEAD)+$(git -C "$sim" rev-parse --short HEAD)")
rm -rf "$sim"
echo "image BUILD_VERSION (branch push, no exact tag): $BUILD_VERSION"

# --- Part 3: does that version satisfy the halt gate this PR adds? ---
probe=gno.land/pkg/gnoland/zz_pr6177_docker_version_test.go
cat > "$probe" <<'GO'
package gnoland

import "testing"

// The halt gate compares the running binary's version.Version against the
// proposal's halt_min_version through meetsMinVersion.
func TestPR6177DockerImageMeetsHaltGate(t *testing.T) {
	const imageVersion = "chain/mainnet.2+8d1d4e2" // what release-docker.yml bakes in
	const floor = "v1.3.0"                         // the release the upgrade was cut for

	if _, ok := parseReleaseVersion(imageVersion); ok {
		t.Fatalf("parseReleaseVersion(%q) parsed; the fixture no longer models the image", imageVersion)
	}

	got := meetsMinVersion(imageVersion, floor)
	// IS:     false — a validator on the container image is refused by the halt gate
	// SHOULD: true  — the image published at v1.3.0 reports v1.3.0
	if got != true {
		t.Fatalf("meetsMinVersion(%q, %q) = %v, want true", imageVersion, floor, got)
	}
}
GO
go test -count=1 -run 'TestPR6177DockerImageMeetsHaltGate' ./gno.land/pkg/gnoland/ 2>&1 \
  | grep -E '^(ok|FAIL|---|    zz_pr6177)' 
rc=${PIPESTATUS[0]}
rm -f "$probe"
[ "$rc" -eq 0 ] || fail=1

echo "---"
[ "$fail" -eq 0 ] && echo "ALL PASS" || echo "FAILED at 607942b78 (expected: this is the finding)"
exit "$fail"
