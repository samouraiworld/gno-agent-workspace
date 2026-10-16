#!/bin/sh
# gnoalias.sh: one interval, now 3600 s, paces every round, so an edit to
# home-override.md, a retry after a failed blog query, and the first-run page
# written without the posts all wait an hour, where the base waited a minute.
#
# Repro, from a plain clone of gnolang/gno, this file copied to /tmp:
#   git fetch origin pull/6327/head
#   git checkout --detach 4ee1b62ff5fa327ac9817b4b0d6f08251765a354
#   sh /tmp/solo-finder-2-hourly-rounds.sh misc/deployments/home-alias/gnoalias.sh
#   git checkout --detach f915442bfd68e0c4d29ab4385dd2a15decb95e23
#   sh /tmp/solo-finder-2-hourly-rounds.sh misc/deployments/home-alias/gnoalias.sh
# Under busybox, as the gnokey image runs it:
#   docker run --rm -v "$PWD:/w" -v /tmp:/t -w /w alpine:3 \
#     sh /t/solo-finder-2-hourly-rounds.sh misc/deployments/home-alias/gnoalias.sh
#
# Stubs: gnokey prints a canned blog row; sleep records its argument, makes
# an operator edit to home-override.md, and stops the loop after two rounds.
# Round 1 fails the blog query (first run), round 2 succeeds.
#
# Measured 2026-10-17:
#   head 4ee1b62ff: round 1 ... sleep 3600, round 2 ... sleep 3600
#   base f915442bf: round 1 ... sleep 60,   round 2 ... sleep 60
set -eu
script=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
d=$(mktemp -d)
mkdir -p "$d/bin" "$d/home"
cat > "$d/bin/gnokey" <<EOF
#!/bin/sh
[ -f "$d/fail" ] && { rm "$d/fail"; echo "rpc unreachable" >&2; exit 1; }
printf 'height: 0\ndata: # blog\n\n<gno-columns>\n### [Post A](/r/gnoland/blog:p/a)\n01 Oct 2026\n<gno-columns-sep>\n### [Post B](/r/gnoland/blog:p/b)\n02 Oct 2026\n<gno-columns-sep>\n### [Post C](/r/gnoland/blog:p/c)\n03 Oct 2026\n</gno-columns>\n'
EOF
cat > "$d/bin/sleep" <<EOF
#!/bin/sh
n=\$(cat "$d/rounds" 2>/dev/null || echo 0); n=\$((n+1)); echo \$n > "$d/rounds"
echo "round \$n: home.md first line '\$(head -n 1 "$d/home/home.md")', blog posts in it: \$(grep -c 'Post A' "$d/home/home.md" || true); next round after sleep \$1"
if [ \$n -ge 2 ]; then kill -TERM \$PPID; exit 0; fi
printf '# Edited by the operator\n\n<!-- latest-blogposts -->\n' > "$d/home/home-override.md"
EOF
chmod +x "$d/bin/gnokey" "$d/bin/sleep"
printf '# Home v1\n\n<!-- latest-blogposts -->\n' > "$d/home/home-override.md"
: > "$d/fail"
echo "script: $1 ($(grep -m1 '^SLEEPING_TIME=' "$script"))"
PATH="$d/bin:$PATH" sh "$script" "$d/home" true 2>/dev/null | grep '^round' || true
rm -rf "$d"
