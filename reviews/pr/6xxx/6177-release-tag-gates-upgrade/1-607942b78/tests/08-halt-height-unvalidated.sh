#!/usr/bin/env bash
# 08-halt-height-unvalidated.sh — cut-release.sh interpolates --halt-height raw
# into the .gno it generates, so a typo emits a body that does not parse.
# Asserts the post-fix state: the script rejects a non-integer height and writes
# nothing. Measured with `gno fmt` on the emitted body. Fails at head 607942b78.
#
# Run: from a local clone of gnolang/gno, at the clone root:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   curl -fsSL -o /tmp/08-halt-height-unvalidated.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/08-halt-height-unvalidated.sh
#   bash /tmp/08-halt-height-unvalidated.sh
#
# Optional: $1 overrides the clone root, $GNO overrides the gno command.

set -uo pipefail

ROOT="${1:-$(git rev-parse --show-toplevel)}"
GNO="${GNO:-go run ./gnovm/cmd/gno}"
LIB="${ROOT}/misc/release/.cut-release-lib.sh"
DIR="${ROOT}/misc/deployments/mainnet.gno.land/transactions/migration/halt-v1.3.0"
TMP="$(mktemp -d)"
cleanup() { rm -f "${LIB}"; rm -rf "${DIR}" "${TMP}"; }
trap cleanup EXIT

# cut-release.sh ends on `main "$@"`; dropping that one line leaves the real
# parse_args and emit_halt_proposal callable. Skipping the preflight changes
# nothing here: no function between parse_args and emit_halt_proposal reads
# HALT_HEIGHT — `grep -n HALT_HEIGHT misc/release/cut-release.sh` lists only the
# default, the flag, the generated body, the tag message and the call guard.
sed '$d' "${ROOT}/misc/release/cut-release.sh" >"${LIB}"

# One cut-release.sh run with the given --halt-height. Echoes the exit status.
emit() {
	rm -rf "${DIR}"
	bash -c '
		set -euo pipefail
		source "$1"
		parse_args v1.3.0 --halt-height "$2" --allow-dirty --commit HEAD
		emit_halt_proposal
	' _ "${LIB}" "$1" >/dev/null 2>&1
	echo $?
}

# gno fmt parses a whole directory, so each body gets its own.
parses() {
	local body="$1" sub="${TMP}/$2"
	mkdir -p "${sub}"
	cp "${body}" "${sub}/halt_v1_3_0.gno"
	(cd "${ROOT}" && ${GNO} fmt "${sub}/halt_v1_3_0.gno" >/dev/null 2>&1)
	echo $?
}

typo_exit="$(emit 120k)"
typo_body_parses=1
[ -f "${DIR}/halt_v1_3_0.gno" ] && {
	typo_line="$(grep -n 'NewSetHaltRequest' "${DIR}/halt_v1_3_0.gno")"
	typo_body_parses="$(parses "${DIR}/halt_v1_3_0.gno" typo)"
}

valid_exit="$(emit 120000)"
valid_body_parses="$(parses "${DIR}/halt_v1_3_0.gno" valid)"

printf '\n--halt-height 120k    cut-release.sh exit=%s  emitted body gno fmt exit=%s\n' \
	"${typo_exit}" "${typo_body_parses}"
printf '    emitted: %s\n' "${typo_line:-<no file written>}"
printf -- '--halt-height 120000  cut-release.sh exit=%s  emitted body gno fmt exit=%s\n\n' \
	"${valid_exit}" "${valid_body_parses}"

rc=0
# SHOULD: a non-integer height is rejected before anything is written, the way
# check_version_shape rejects a malformed VERSION.
[ "${typo_exit}" != 0 ] || {
	echo "FAIL: cut-release.sh accepted --halt-height 120k and wrote an unparseable proposal"
	rc=1
}
# IS: the script exits 0 and the body it wrote does not parse.
# [ "${typo_body_parses}" != 0 ] || echo "unexpected: the 120k body parses"

# Baseline that must keep working: a real height still emits a body that parses.
[ "${valid_exit}" = 0 ] && [ "${valid_body_parses}" = 0 ] || {
	echo "FAIL: --halt-height 120000 no longer emits a parseable proposal"
	rc=1
}

[ "${rc}" = 0 ] && echo PASS || echo FAIL
exit "${rc}"
