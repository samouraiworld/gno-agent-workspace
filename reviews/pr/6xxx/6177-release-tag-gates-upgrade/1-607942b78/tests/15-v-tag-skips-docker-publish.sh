#!/usr/bin/env bash
# Asserts that the release tag shape RELEASING.md now documents, vMAJOR.MINOR.PATCH,
# reaches release / chain-tag but not release / docker, so a release publishes no
# version-named container image. Measured against the live ghcr.io registry:
# chain-gnoland1.1 is published, v1.1.0 and v1.0.0 are not.
# Every row passes at head 607942b78fa4fdf6f378fecce32bc1d1d984ab8e: the active
# assertions pin what the release does today, the commented ones what it should do.
# At merge base 1fc4c140ec4064e8d66cb9828ddea280a723cca6 two rows move: v1.3.0 reached
# neither workflow, and misc/release/ did not exist. Run it there with NO_NET=1.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   curl -fsSL -o /tmp/15-v-tag-skips-docker-publish.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/15-v-tag-skips-docker-publish.sh
#   bash /tmp/15-v-tag-skips-docker-publish.sh
# Needs curl and network for the registry half; run with NO_NET=1 to skip it.

set -u
cd "$(git rev-parse --show-toplevel)"
fail=0

# The tag globs GitHub matches a pushed ref against, read from on.push.tags of one workflow.
tag_globs() {
	awk '
		/^on:/        { in_on=1; next }
		/^[a-z]/      { in_on=0 }
		in_on && /^  push:/  { in_push=1; next }
		in_on && /^  [a-z]/  { in_push=0 }
		in_push && /^    tags:/ { in_tags=1; next }
		in_push && /^    [a-z]/ { in_tags=0 }
		in_tags && /^      - / { gsub(/^      - |"/, ""); print }
	' "$1"
}

# GitHub filter-pattern semantics: * matches any run of characters except /.
triggers() { # <workflow> <tag>
	local g
	for g in $(tag_globs "$1"); do
		# shellcheck disable=SC2254
		case "$2" in $g) return 0 ;; esac
	done
	return 1
}

say() { # <label> <actual> <expected>
	if [ "$2" = "$3" ]; then
		printf 'ok   %-46s %s\n' "$1" "$2"
	else
		printf 'FAIL %-46s got %s, want %s\n' "$1" "$2" "$3"
		fail=1
	fi
}

yn() { if triggers "$1" "$2"; then echo yes; else echo no; fi; }

CT=.github/workflows/release-chain-tag.yml
DK=.github/workflows/release-docker.yml

echo "== which workflow a pushed release tag reaches =="
# Baseline: betanet's chain tag reached both, which is why its image exists.
say "chain/gnoland1.1 -> release / chain-tag" "$(yn $CT chain/gnoland1.1)" yes
say "chain/gnoland1.1 -> release / docker"    "$(yn $DK chain/gnoland1.1)" yes
# The documented release shape reaches the binary job only.
say "v1.3.0 -> release / chain-tag"           "$(yn $CT v1.3.0)"           yes
say "v1.3.0 -> release / docker"              "$(yn $DK v1.3.0)"           no   # IS:     no image job for a release
# say "v1.3.0 -> release / docker"            "$(yn $DK v1.3.0)"           yes  # SHOULD: both publication jobs fire

echo
echo "== what misc/release/cut-release.sh --push actually pushes =="
grep -n 'git -C "${REPO_ROOT}" push origin' misc/release/cut-release.sh
say "pushes a chain/* tag too" \
	"$(grep -c 'push origin "refs/tags/chain' misc/release/cut-release.sh)" 0   # IS:     v tag only
# say "pushes a chain/* tag too" "$(grep -c ...)" 1                              # SHOULD: n/a, the v tag should be enough

echo
echo "== RELEASING.md on the container images =="
say "RELEASING.md mentions docker/ghcr/image" \
	"$(grep -ci -E 'docker|ghcr|image' RELEASING.md)" 0   # IS:     the dropped step is undocumented

if [ "${NO_NET:-0}" = 0 ]; then
	echo
	echo "== ghcr.io/gnolang/gno/gnoland, published tags =="
	tok=$(curl -s --max-time 20 \
		"https://ghcr.io/token?scope=repository:gnolang/gno/gnoland:pull&service=ghcr.io" \
		| sed -E 's/.*"token":"([^"]+)".*/\1/')
	probe() {
		curl -s --max-time 20 -o /dev/null -w '%{http_code}' -I -H "Authorization: Bearer $tok" \
			-H 'Accept: application/vnd.oci.image.index.v1+json,application/vnd.docker.distribution.manifest.list.v2+json' \
			"https://ghcr.io/v2/gnolang/gno/gnoland/manifests/$1"
	}
	# Betanet's upgrade tag chain/gnoland1.1 reached release / docker, so it has an image.
	say "image chain-gnoland1.1 (betanet's upgrade)" "$(probe chain-gnoland1.1)" 200
	# The chain branch keeps publishing, but under a tag that moves with every push.
	say "image chain-mainnet (branch trigger, moving)" "$(probe chain-mainnet)"  200
	say "image v1.1.0 (the same release, v shape)"   "$(probe v1.1.0)"           404  # IS:     no image under the v name
	say "image v1.0.0 (the same release, v shape)"   "$(probe v1.0.0)"           404  # IS:     same
	# say "image v1.1.0" "$(probe v1.1.0)" 200                                        # SHOULD: a release is pullable by its version
fi

echo
[ $fail -eq 0 ] && echo "as measured: a vX.Y.Z release ships binaries and no image of its own" \
	|| echo "a measurement moved; re-read the rows above"
exit $fail
