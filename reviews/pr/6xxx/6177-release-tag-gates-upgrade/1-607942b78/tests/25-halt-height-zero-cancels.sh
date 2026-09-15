#!/usr/bin/env bash
# 25-halt-height-zero-cancels.sh — cut-release.sh accepts --halt-height 0, the
# sentinel r/sys/params.NewSetHaltRequest reads as "cancel the scheduled halt",
# and stamps it into a proposal and a tag that both read as a halt.
# Asserts the post-fix state: 0 is rejected before anything is written.
# Measured against examples/gno.land/r/sys/params/halt.gno. Fails at 607942b78.
#
# Run: from a local clone of gnolang/gno, at the clone root:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   curl -fsSL -o /tmp/25-halt-height-zero-cancels.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/25-halt-height-zero-cancels.sh
#   bash /tmp/25-halt-height-zero-cancels.sh
#
# Optional: $1 overrides the clone root.

set -uo pipefail

ROOT="${1:-$(git rev-parse --show-toplevel)}"
LIB="${ROOT}/misc/release/.cut-release-lib-25.sh"
DIR="${ROOT}/misc/deployments/mainnet.gno.land/transactions/migration/halt-v1.3.0"
HALT_GNO="${ROOT}/examples/gno.land/r/sys/params/halt.gno"
cleanup() {
	rm -f "${LIB}"
	rm -rf "${DIR}"
	git -C "${ROOT}" tag -d v1.3.0 >/dev/null 2>&1
}
trap cleanup EXIT

# cut-release.sh ends on `main "$@"`; dropping that one line leaves the real
# parse_args, emit_halt_proposal and create_tag callable. The preflight reads no
# HALT_HEIGHT: `grep -n HALT_HEIGHT misc/release/cut-release.sh` lists only the
# default, the flag, the generated body, the tag message and the call guard.
sed '$d' "${ROOT}/misc/release/cut-release.sh" >"${LIB}"

# One cut-release.sh run with the given --halt-height, through the real
# emit_halt_proposal and create_tag. Echoes the exit status.
emit() {
	rm -rf "${DIR}"
	git -C "${ROOT}" tag -d v1.3.0 >/dev/null 2>&1
	bash -c '
		set -euo pipefail
		source "$1"
		parse_args v1.3.0 --halt-height "$2" --allow-dirty --commit HEAD
		COMMIT="$(git -C "${REPO_ROOT}" rev-parse HEAD)"
		[ -z "${HALT_HEIGHT}" ] || emit_halt_proposal
		create_tag
	' _ "${LIB}" "$1" >/dev/null 2>&1
	echo $?
}

body="${DIR}/halt_v1_3_0.gno"

zero_exit="$(emit 0)"
zero_call="$(grep -h 'NewSetHaltRequest' "${body}" 2>/dev/null | sed 's/^[[:space:]]*//')"
zero_header="$(grep -h 'Every node stops' "${body}" 2>/dev/null | sed 's|^// ||')"
zero_tag="$(git -C "${ROOT}" tag -l -n20 v1.3.0 2>/dev/null | grep -o 'Coordinated upgrade at height.*')"

real_exit="$(emit 120000)"
real_call="$(grep -h 'NewSetHaltRequest' "${body}" 2>/dev/null | sed 's/^[[:space:]]*//')"

# What the realm itself says height 0 means, quoted from its own source.
sentinel="$(grep -h 'height=0' "${HALT_GNO}" | sed 's|^// ||')"
desc0="$(grep -h 'Cancel the scheduled chain halt' "${HALT_GNO}" | sed 's/^[[:space:]]*//')"

printf '\nr/sys/params/halt.gno contract:\n  %s\n  height == 0 =>  %s\n\n' \
	"${sentinel}" "${desc0}"
printf -- '--halt-height 0       cut-release.sh exit=%s\n' "${zero_exit}"
printf '    emitted call:   %s\n' "${zero_call:-<no file written>}"
printf '    emitted header: %s\n' "${zero_header:-<no file written>}"
printf '    tag message:    %s\n' "${zero_tag:-<no tag>}"
printf -- '--halt-height 120000  cut-release.sh exit=%s\n' "${real_exit}"
printf '    emitted call:   %s\n\n' "${real_call:-<no file written>}"

rc=0
# SHOULD: 0 is the cancel sentinel, so cut-release.sh rejects it before writing
# anything, the way check_version_shape rejects a malformed VERSION.
[ "${zero_exit}" != 0 ] && [ ! -e "${body}" ] || {
	echo "FAIL: --halt-height 0 emitted a cancel-halt proposal described as a halt"
	rc=1
}
# IS: exit 0, and the file it wrote calls NewSetHaltRequest(cross(cur), 0, ...)
# under a header reading "Every node stops after committing block 0".
# [ "${zero_exit}" = 0 ] || echo "unexpected: --halt-height 0 was rejected"

# Baseline that must keep working: a real height still emits a halt proposal.
[ "${real_exit}" = 0 ] && [ -n "${real_call}" ] || {
	echo "FAIL: --halt-height 120000 no longer emits a proposal"
	rc=1
}

[ "${rc}" = 0 ] && echo PASS || echo FAIL
exit "${rc}"
