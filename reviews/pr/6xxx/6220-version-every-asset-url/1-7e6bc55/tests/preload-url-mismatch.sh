#!/usr/bin/env bash
# Asserts that the font URL the page head preloads is the same URL the served
# stylesheet requests, so the preload matches and the font is fetched once.
# Boots the gnoweb binary given as $1 and reads both URLs off it.
# Fails at 7e6bc5514; passes at 877379432.
#
#   go build -o /tmp/gnoweb ./gno.land/cmd/gnoweb && ./preload-url-mismatch.sh /tmp/gnoweb
set -u

bin=${1:?usage: preload-url-mismatch.sh <path to gnoweb binary> [port]}
port=${2:-18899}

"$bin" --bind "127.0.0.1:$port" --remote https://rpc.gno.land:443 > /tmp/gnoweb-probe.log 2>&1 &
pid=$!
trap 'kill $pid 2>/dev/null' EXIT

for _ in 1 2 3 4 5 6 7 8; do
  curl -sS -o /dev/null --max-time 5 "http://127.0.0.1:$port/" 2>/dev/null && break
  sleep 2
done

preload=$(curl -sS --max-time 10 "http://127.0.0.1:$port/" | grep -o 'href="[^"]*Intervar[^"]*"')
css=$(curl -sS --max-time 10 "http://127.0.0.1:$port/public/main.css" | grep -o 'url([^)]*Intervar[^)]*)')

echo "preload:    $preload"
echo "stylesheet: $css"

preload_q=${preload#*Intervar.woff2}
css_q=${css#*Intervar.woff2}
preload_q=${preload_q%\"}
css_q=${css_q%)}

if [ "$preload_q" = "$css_q" ]; then
  echo "PASS: both name the same URL, so the preload answers the stylesheet's request"
  exit 0
fi

echo "FAIL: the preload asks for Intervar.woff2${preload_q:-<no query>} and the stylesheet for Intervar.woff2${css_q:-<no query>}, so the preload matches nothing and the font is fetched twice"
exit 1
