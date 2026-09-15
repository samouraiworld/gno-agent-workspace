#!/usr/bin/env bash
# Asserts misc/release/cut-release.sh refuses a --halt-height that is not a
# positive block height. Measured at 607942b78: `--halt-height --push` swallows
# the next flag, so the emitted proposal reads
# `params.NewSetHaltRequest(cross(cur), --push, "v9.8.7")` (gofmt: "expected
# operand, found '--'") and PUSH stays 0; `--halt-height 0` emits the documented
# cancel form. Both assertions below FAIL at the reviewed head and pass once
# parse_args validates the value.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   curl -fsSL -o /tmp/45-cut-release-halt-height-unvalidated.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/45-cut-release-halt-height-unvalidated.sh
#   bash /tmp/45-cut-release-halt-height-unvalidated.sh; echo "exit=$?"

set -uo pipefail

root="$(git rev-parse --show-toplevel)"
cd "${root}"
mig="misc/deployments/mainnet.gno.land/transactions/migration"
fail=0

cleanup() {
	git tag -d v9.8.7 v9.8.6 >/dev/null 2>&1 || true
	rm -rf "${mig}/halt-v9.8.7" "${mig}/halt-v9.8.6"
}
trap cleanup EXIT
cleanup

# --- case 1: --halt-height immediately followed by another flag -------------
# parse_args takes "${2-}" unvalidated and shifts past both, so the flag lands
# in HALT_HEIGHT and never reaches the --push case.
out1="$(misc/release/cut-release.sh v9.8.7 --halt-height --push --allow-dirty --commit HEAD 2>&1)"
rc1=$?
gen1="${mig}/halt-v9.8.7/halt_v9_8_7.gno"

echo "== case 1: cut-release.sh v9.8.7 --halt-height --push"
echo "   exit code            : ${rc1}"
echo "   emitted call         : $(grep -o 'NewSetHaltRequest(.*)' "${gen1}" 2>/dev/null || echo '<no file>')"
echo "   parses as Gno source : $(gofmt -e "${gen1}" >/dev/null 2>&1 && echo yes || echo no)"
echo "   final line           : $(printf '%s' "${out1}" | tail -1 | sed 's/\x1b\[[0-9;]*m//g')"

# SHOULD: the script dies on a --halt-height value that is not a positive integer.
# IS (607942b78): rc=0, file written, gofmt rejects it, and --push was consumed
#                 as the height so the run ends on "to undo the local tag".
if [[ ${rc1} -eq 0 ]]; then
	echo "   FAIL: exited 0 on '--halt-height --push'"
	fail=1
fi
if [[ -f ${gen1} ]] && ! gofmt -e "${gen1}" >/dev/null 2>&1; then
	echo "   FAIL: emitted a proposal that is not valid Gno source"
	fail=1
fi

# --- case 2: a height the halt API reads as "cancel" ------------------------
# examples/gno.land/r/sys/params/halt.gno: "Use height=0 to cancel a previously
# scheduled halt." Nothing between parse_args and emit_halt_proposal rejects it.
out2="$(misc/release/cut-release.sh v9.8.6 --halt-height 0 --allow-dirty --commit HEAD 2>&1)"
rc2=$?
gen2="${mig}/halt-v9.8.6/halt_v9_8_6.gno"

echo "== case 2: cut-release.sh v9.8.6 --halt-height 0"
echo "   exit code            : ${rc2}"
echo "   emitted call         : $(grep -o 'NewSetHaltRequest(.*)' "${gen2}" 2>/dev/null || echo '<no file>')"
echo "   tag message          : $(git tag -n9 -l v9.8.6 | tail -1 | sed 's/^ *//')"

# SHOULD: rejected, because 0 cancels a halt rather than scheduling one.
# IS (607942b78): a compiling proposal whose on-chain description reads
#                 "Cancel the scheduled chain halt", under a tag whose message
#                 reads "Coordinated upgrade at height 0".
if [[ ${rc2} -eq 0 ]]; then
	echo "   FAIL: exited 0 on '--halt-height 0'"
	fail=1
fi

echo
[[ ${fail} -eq 0 ]] && echo "PASS: --halt-height is validated" || echo "FAIL: --halt-height is taken unvalidated"
exit "${fail}"
