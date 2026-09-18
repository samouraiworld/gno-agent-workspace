#!/usr/bin/env bash
# What this measures: the two halves of candidate #163 on
# .github/workflows/pr-preview-publish.yml, which the finder named but did not run.
#
#   1. reach — the bytes the privileged job commits are chosen by the pull request:
#      pr-preview.yml triggers on `pull_request`, so GitHub runs the FORK's copy of
#      that workflow file, and its line 77 builds the fork's own misc/gnopreview
#      before line 97 runs it into _preview/. The publish job's only filters, at
#      pr-preview-publish.yml:127-132, are a `.git` prune and a 40 MiB `du -sm`,
#      after which line 140 `cp -r ../_preview/. "pr-$PR/"` commits and pushes.
#   2. impact — whether the previews site shares its browser origin with another
#      gnolang Pages site that runs script. It does: both resolve on the single
#      host gnolang.github.io, and the godoc mirror there reads document.cookie.
#
# Both checks are read-only: no page is published, nothing is pushed.
#
# Repro from a plain clone (network access to gnolang.github.io required):
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>
set -u

echo "== 1. the render job builds and runs the pull request's own gnopreview =="
sed -n '14,15p;74,78p;96,102p' .github/workflows/pr-preview.yml

echo
echo "== 2. everything the privileged job does to those bytes before committing them =="
sed -n '127,132p;138,141p' .github/workflows/pr-preview-publish.yml

echo
echo "== 3. one origin, two Pages sites =="
for u in https://gnolang.github.io/gno-previews/ https://gnolang.github.io/gno/; do
	printf '  %-44s HTTP %s\n' "$u" "$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$u")"
done

echo
echo "== 4. the neighbour on that origin runs script against document.cookie =="
curl -s --max-time 15 https://gnolang.github.io/gno/ | grep -n -m1 -B2 'document.cookie'

echo
echo "A preview page served from https://gnolang.github.io/gno-previews/pr-<N>/ is"
echo "same-origin with https://gnolang.github.io/gno/: it can fetch it, frame and read"
echo "it, and read or write that origin's cookies and localStorage. The file's header"
echo "at lines 14-17 settles the shared origin with an argument about hostnames being"
echo "mistaken for production, which is about phishing and not about script isolation."
