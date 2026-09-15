# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# Candidate #26: misc/release/cut-release.sh:63 usage() runs
# `sed -n '2,34p'` over its own header, dropping line 35 — the --halt-height
# example, the only documented way to emit the halt proposal.
# Confirms: `--help` output ends before the command line the last example
# introduces, and line 35 of the file carries exactly that command.
#
# from a local clone of gnolang/gno:
#   git fetch origin pull/6177/head:pr6177 && git checkout pr6177
set -euo pipefail

echo "== tail of --help output =="
misc/release/cut-release.sh --help | tail -6

echo
echo "== line 35 of the script (missing from the output above) =="
sed -n '35p' misc/release/cut-release.sh
