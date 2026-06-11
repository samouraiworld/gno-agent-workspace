#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# build-indexes.sh — single entry point for all review indexes.
#
# Runs build-reviews-readme.sh (reviews/README.md, live GitHub state) then
# build-reports-index.py (root index.html for GitHub Pages, local walk).
# Args are passed through to build-reviews-readme.sh.
#
# Usage:
#   ./scripts/build-indexes.sh [-o OUTPUT]

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

./scripts/build-reviews-readme.sh "$@"
./scripts/build-reports-index.py
