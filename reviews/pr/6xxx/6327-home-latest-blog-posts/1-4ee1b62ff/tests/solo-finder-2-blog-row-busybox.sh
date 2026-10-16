#!/bin/sh
# gnoalias.sh render_override on the live mainnet blog, under busybox sh and
# awk (the gnokey image is alpine:3): the marker line of home.mainnet.md and
# home.testnet.md becomes the first <gno-columns> row, titles in bold, links
# prefixed by BLOG_LINK_BASE.
#
# Repro, from a plain clone of gnolang/gno at the head, this file copied to /tmp:
#   git fetch origin pull/6327/head
#   git checkout --detach 4ee1b62ff5fa327ac9817b4b0d6f08251765a354
#   go build -o /tmp/gnokey ./gno.land/cmd/gnokey
#   /tmp/gnokey query vm/qrender -remote https://rpc.gno.land \
#     -data "gno.land/r/gnoland/blog:" > /tmp/mainnet-blog.raw
#   docker run --rm -v "$PWD:/w" -v /tmp:/t -w /w alpine:3 \
#     sh /t/solo-finder-2-blog-row-busybox.sh /t/mainnet-blog.raw
set -eu
raw=$1
d=$(mktemp -d)
mkdir -p "$d/bin"
printf '#!/bin/sh\ncat %s\n' "$raw" > "$d/bin/gnokey"
chmod +x "$d/bin/gnokey"
sed -n '1,94p' misc/deployments/home-alias/gnoalias.sh > "$d/lib.sh"
for page in home.mainnet.md home.testnet.md; do
  for base in "" https://gno.land; do
    cp "misc/deployments/home-alias/$page" "$d/home-override.md"
    echo "== $page BLOG_LINK_BASE='$base'"
    (cd "$d" && PATH="$d/bin:$PATH" BLOG_LINK_BASE=$base sh -c '. ./lib.sh; render_override; echo "exit $?"') |
      sed -n '/Latest from the blog/,/Read the blog/p;/^exit /p'
  done
done
rm -rf "$d"
