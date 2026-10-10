#!/bin/sh
# Repro for gnolang/gno#6229 at ea1c620f7f5413f9a6db21883476c51b14ba8b9a, from a plain clone:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6229/head && git checkout ea1c620f7f5413f9a6db21883476c51b14ba8b9a
#   sh <this file>
#
# TestCanonicalURLArgs (new, t.Parallel) builds newPagePolicy(nil, DefaultAliases, ...),
# whose aliasTargets ranges over the package-level DefaultAliases map, while the
# parallel TestStaticMarkdownDevLinks writes into that same map through
# NewDefaultAppConfig().Aliases (it is DefaultAliases itself, not a copy).
#
# Observed at the head (2026-10-10):
#   go test -race -count=6 -run 'TestCanonicalURLArgs|TestStaticMarkdownDevLinks'
#     5 WARNING: DATA RACE, FAIL
#   Read at ... aliasTargets() page_kind.go:69 <- newPagePolicy() page_kind.go:59
#     <- TestCanonicalURLArgs() canonical_origin_test.go:73
#   Previous write at ... TestStaticMarkdownDevLinks() app_test.go:181
#   go test -race -count=4 ./gno.land/pkg/gnoweb/   -> 4 races, all this pair, FAIL
set -e
go test -race -count=6 ./gno.land/pkg/gnoweb/ -run 'TestCanonicalURLArgs|TestStaticMarkdownDevLinks' 2>&1 \
  | grep -E 'WARNING: DATA RACE|page_kind.go|canonical_origin_test.go|app_test.go|^(ok|FAIL)' || true
