#!/usr/bin/env bash
# Asserts that a pre-release tag RELEASING.md blesses, v1.3.0-rc.1, reaches
# release-chain-tag.yml at the reviewed head and not at the merge base, and that
# the workflow's own "Ensure release exists" body then runs `gh release create`
# with no --prerelease and no --latest=false, so the rc is published as a full
# release. Measured: head trigger matches, base trigger does not, 0 gating flags.
# Fails at 607942b78; passes once the create command gates the pre-release shape.
#
# from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 49-prerelease-tag-publishes-as-latest.sh
#   # add GH_ONLINE=1 for the read-only github.com corroboration (needs gh auth)

set -u
REPO_ROOT="${1:-$(git rev-parse --show-toplevel)}"
BASE=1fc4c140ec4064e8d66cb9828ddea280a723cca6
WF=.github/workflows/release-chain-tag.yml
TAG=v1.3.0-rc.1   # RELEASING.md: "Pre-release tags (v1.3.0-rc.1) are allowed"
fail=0

# the push-tag globs a workflow file declares
globs() { sed -n '/^    tags:/,/^  [a-z_]/p' | grep -oE '"[^"]+"' | tr -d '"'; }
matches() { case "$1" in $2) return 0 ;; esac; return 1; }

echo "== 1. does $TAG reach the workflow? =="
for state in base head; do
  if [ "$state" = base ]; then body=$(git -C "$REPO_ROOT" show "$BASE:$WF"); else body=$(cat "$REPO_ROOT/$WF"); fi
  hit=no
  for g in $(printf '%s\n' "$body" | globs); do matches "$TAG" "$g" && { hit="yes (glob $g)"; break; }; done
  echo "  $state: triggered=$hit"
  [ "$state" = head ] && head_hit=$hit || base_hit=$hit
done
[ "$base_hit" = no ] || { echo "  UNEXPECTED: the base already admits the tag"; fail=1; }

echo "== 2. shape check in misc/release/cut-release.sh =="
shape_re=$(sed -n 's/.*=~ \(\^v.*\$\) \]\].*/\1/p' "$REPO_ROOT/misc/release/cut-release.sh" | head -1)
if [[ $TAG =~ $shape_re ]]; then echo "  check_version_shape accepts $TAG  (regex $shape_re)"
else echo "  check_version_shape rejects $TAG -- the path is closed"; fail=1; fi

echo "== 3. what the workflow actually asks GitHub for =="
# run the step's own body, unmodified, with a gh that records argv instead of publishing
stub=$(mktemp -d); trap 'rm -rf "$stub"' EXIT
cat > "$stub/gh" <<'SH'
#!/usr/bin/env bash
[ "${1:-}" = release ] && [ "${2:-}" = view ] && exit 1   # no release exists yet for a fresh tag
printf 'gh %s\n' "$*" >> "$GH_CALLS"
SH
chmod +x "$stub/gh"
step=$(awk '/- name: Ensure release exists/{f=1} f&&/run: \|/{r=1;next} r&&/^      - name:/{exit} r{sub(/^          /,"");print}' "$REPO_ROOT/$WF")
[ -n "$step" ] || { echo "  could not extract the step body"; exit 2; }
GH_CALLS="$stub/calls" PATH="$stub:$PATH" TAG="$TAG" GITHUB_REPOSITORY=gnolang/gno bash -c "$step"
sed 's/^/  /' "$stub/calls"

if grep -qE -- '--prerelease|-p( |$)|--latest' "$stub/calls"; then
  echo "  SHOULD: the rc is created gated, GitHub keeps it off /releases/latest"
else
  echo "  IS:     no --prerelease and no --latest=false -> prerelease=false, GitHub marks it Latest"
  fail=1
fi

if [ "${GH_ONLINE:-0}" = 1 ]; then
  echo "== 4. github.com does not infer prerelease from the tag shape =="
  gh api 'repos/traefik/traefik/releases?per_page=100' \
    --jq '.[]|select(.prerelease==false and (.tag_name|test("-rc")))|"  \(.tag_name) prerelease=\(.prerelease) \(.html_url)"'
fi

echo
[ "$fail" -eq 0 ] && echo "PASS" || echo "FAIL: at $head_hit the rc tag builds a release nothing marks as a pre-release"
exit "$fail"
