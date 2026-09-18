#!/usr/bin/env bash
# Does --virtual-time-budget=4000 make `--headless=new` wait, as
# misc/gnopreview/shots.go:186-189 claims ("Let the controller modules load and
# the webfonts settle before the frame is grabbed; without it the shot is
# unstyled text")?
#
# Repro from a plain clone (no repo files needed; chromium or google-chrome on PATH):
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
#   bash misc/../<this file>          # or: bash b4-claims-r2-virtual-time.sh
#
# Serves a page over http:// (the scheme shots.go's serve() exists to provide),
# whose body only becomes red after a 2s timer AND after a dynamically imported
# ES module runs -- the two things the comment says the flag waits for. The
# screenshot is taken with the exact flag list of chromeShot().
set -eu
BIN="${CHROME:-$(command -v chromium || command -v google-chrome-stable || command -v google-chrome)}"
W="$(mktemp -d)"; trap 'rm -rf "$W"' EXIT
mkdir -p "$W/site"
cat > "$W/site/mod.js" <<'JS'
export function paint() { document.body.style.background = '#ff0000'; document.title = 'LATE'; }
JS
cat > "$W/site/index.html" <<'HTML'
<!doctype html><meta charset=utf-8><title>EARLY</title>
<body style="background:#ffffff;margin:0">
<script type="module">
  setTimeout(async () => { const m = await import('./mod.js'); m.paint(); }, 2000);
</script>
HTML
( cd "$W/site" && python3 -m http.server 8731 >/dev/null 2>&1 & echo $! > "$W/pid" )
sleep 1
shot() { # $1 = output png, rest = extra flags
  local out="$1"; shift
  "$BIN" --headless=new --disable-gpu --no-sandbox --hide-scrollbars \
    --force-device-scale-factor=1 --window-size=1280,860 "$@" \
    --user-data-dir="$W/profile-$(basename "$out")" \
    --screenshot="$out" http://127.0.0.1:8731/ >/dev/null 2>&1 || true
}
shot "$W/with.png"    --virtual-time-budget=4000
shot "$W/without.png"
kill "$(cat "$W/pid")" 2>/dev/null || true
cat > "$W/pix.go" <<'GO'
package main
import ("fmt";"image/png";"os")
func main() {
  for _, p := range os.Args[1:] {
    f, err := os.Open(p); if err != nil { fmt.Println(p, "MISSING"); continue }
    im, err := png.Decode(f); if err != nil { fmt.Println(p, "UNDECODABLE"); continue }
    r, g, b, _ := im.At(640, 430).RGBA()
    st, _ := os.Stat(p)
    fmt.Printf("%s center=#%02x%02x%02x size=%d\n", p, r>>8, g>>8, b>>8, st.Size())
  }
}
GO
( cd "$W" && go mod init pix >/dev/null 2>&1; go run pix.go "$W/with.png" "$W/without.png" )
echo "expected if the flag works: with.png center=#ff0000 (timer+module ran), without.png center=#ffffff"
