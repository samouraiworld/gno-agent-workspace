# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# Candidate #23: misc/release/cut-release.sh:184 check_protocol_constants calls
# bump-protocol-version.sh --check with no ref, which reads REPO_ROOT's working
# tree (bump-protocol-version.sh:29, `readonly REPO_ROOT=... pwd`), not the
# commit --commit names. check_build_reports_tag (cut-release.sh:191) instead
# builds a worktree AT the commit, so this preflight alone answers about the
# wrong tree.
#
# Rehearsal: commit a broken protocol constant as a child of HEAD (so it never
# touches the working tree), reset the working tree back to the clean parent,
# then run cut-release.sh --commit <broken-child>. The preflight is supposed to
# gate the tagged commit's constants; it prints "protocol-version constants
# agree" anyway because it reads the clean working tree instead.
#
# from a local clone of gnolang/gno:
#   git fetch origin pull/6177/head:pr6177 && git checkout pr6177
set -euo pipefail

echo "== working tree starts clean, at the commit being reviewed =="
git status --porcelain
git log --oneline -1

echo
echo "== create a child commit with a broken protocol-version constant, without checking it out =="
sed -i 's/const Version = "v1.0.0-rc.0"/const Version = "v9.9.9-broken"/' tm2/pkg/crypto/version.go
git add tm2/pkg/crypto/version.go
git -c user.email=t@t -c user.name=t commit -q -m "TEST: break protocol constant"
BROKEN=$(git rev-parse HEAD)
echo "broken commit: $BROKEN"
git reset --hard HEAD~1 -q
echo "back on the clean parent:"
git log --oneline -1

echo
echo "== run cut-release.sh against the BROKEN commit while the working tree is clean/good =="
misc/release/cut-release.sh v9.9.9 --chain test13 --commit "$BROKEN" 2>&1 \
  | grep -E "protocol|error|tagging"

echo
echo "== the target commit's own constants, for comparison: they disagree =="
git show "$BROKEN":tm2/pkg/crypto/version.go | grep Version

# cleanup
git tag -d v9.9.9 >/dev/null 2>&1 || true
git checkout -- . && git clean -fdq
