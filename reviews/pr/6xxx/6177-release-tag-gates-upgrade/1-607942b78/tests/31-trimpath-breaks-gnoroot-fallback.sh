#!/usr/bin/env bash
# Asserts: the -trimpath added at .github/workflows/release-chain-tag.yml:83 makes every
# released gno/gnoland binary panic on any invocation when GNOROOT is unset, because it
# kills guessRootDir's runtime.Caller fallback (gnovm/pkg/gnoenv/gnoroot.go:65).
# Measured at head 607942b78: with -trimpath both panic (rc=2); without it both print
# "<name> version: v1.2.0" (rc=0). Fails at the reviewed head.
#
# from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 31-trimpath-breaks-gnoroot-fallback.sh
set -u

LDFLAGS="-w -s -X github.com/gnolang/gno/tm2/pkg/version.Version=v1.2.0"
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

for pkg in ./gnovm/cmd/gno ./gno.land/cmd/gnoland; do
  name="$(basename "$pkg")"
  # exactly the release job's flags (line 83), then the same build without -trimpath
  go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/${name}-trimpath" "$pkg" || exit 1
  go build            -ldflags "$LDFLAGS" -o "$OUT/${name}-plain"    "$pkg" || exit 1

  # a downloaded release binary: no GNOROOT, no `go` on PATH, cwd outside the module
  for variant in trimpath plain; do
    cd "$OUT" || exit 1
    env -i PATH=/usr/bin:/bin HOME="$OUT" "$OUT/${name}-${variant}" version >"$OUT/log" 2>&1
    rc=$?
    printf '%-8s %-8s rc=%s %s\n' "$name" "$variant" "$rc" "$(head -1 "$OUT/log")"
    cd - >/dev/null || exit 1
  done
done

# IS:     released binaries panic before any subcommand runs
#         gno      trimpath rc=2 panic: gno was unable to determine GNOROOT. ...
#         gnoland  trimpath rc=2 panic: gno was unable to determine GNOROOT. ...
# SHOULD: both report the tag, as they do without -trimpath
#         gno      trimpath rc=0 gno version: v1.2.0
#         gnoland  trimpath rc=0 gnoland version: v1.2.0
