#!/usr/bin/env bash
# misc/gnopreview/main.go:199 — gnodev is started in its own process group
# (SysProcAttr{Setpgid: true}) and main installs no signal handler, so a Ctrl-C
# on gnopreview never reaches gnodev: the parent dies without running its
# `defer stop()` and gnodev keeps running, holding -port and -port+10000.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   (cd misc/gnopreview && go build -o /tmp/gnopreview .)
#   ROOT=$PWD GNOPREVIEW=/tmp/gnopreview bash misc/../<this file>
#
# Expected if the process group were handled: "gnodev alive after SIGINT: 0".
# Observed at this head: 1 (see the run below).
set -u
# Job control on: without it a non-interactive shell starts a `&` job with
# SIGINT set to SIG_IGN, Go keeps an inherited SIG_IGN, and gnopreview would
# survive the signal for a reason that has nothing to do with the code.
set -m

ROOT=${ROOT:-$PWD}
GNOPREVIEW=${GNOPREVIEW:-/tmp/gnopreview}
RUN=$(mktemp -d)
cd "$RUN" || exit 1

# A stand-in for gnodev: it only has to stay alive, the way a real node does
# while gnopreview waits for gnoweb to answer.
printf '#!/bin/sh\nexec sleep 297\n' > fakegnodev
chmod +x fakegnodev
echo 'examples/gno.land/r/gnoland/home/home.gno' > changed.txt

# setsid puts gnopreview in its own session, so `kill -INT -<pid>` below is
# exactly what a terminal sends the foreground process group on Ctrl-C.
setsid sh -c "exec '$GNOPREVIEW' render -root '$ROOT' -changed changed.txt \
  -out _preview -gnodev '$RUN/fakegnodev' -timeout 120s -port 18899" \
  > out.log 2>&1 &

for _ in $(seq 1 120); do
  pgrep -f 'sleep 297' > /dev/null && break
  sleep 1
done
PID=$(pgrep -f "$GNOPREVIEW render" | head -1)
echo "gnopreview pid=$PID pgid=$(ps -o pgid= -p "$PID" | tr -d ' ')"
echo "gnodev pgid=$(ps -o pgid= -p "$(pgrep -f 'sleep 297' | head -1)" | tr -d ' ')"

kill -INT -"$PID"   # Ctrl-C on the foreground process group
sleep 2

echo "gnopreview alive after SIGINT: $(pgrep -f "$GNOPREVIEW render" | wc -l)"
echo "gnodev alive after SIGINT:     $(pgrep -f 'sleep 297' | wc -l)"

pkill -f 'sleep 297'
rm -rf "$RUN"
